package cluster

import (
	"context"
	"net"
	"net/http"
	"strings"
	"time"

	etpclient "github.com/integration-system/isp-etp-go/v2/client"
	"github.com/integration-system/isp-kit/json"
	"github.com/integration-system/isp-kit/log"
	"github.com/pkg/errors"
)

type HandshakeConfirmer interface {
	confirm(ctx context.Context, data HandshakeData) error
}

type Handshake struct {
	moduleInfo   ModuleInfo
	configData   ConfigData
	requirements ModuleRequirements
	confirmer    HandshakeConfirmer
	logger       log.Logger
}

func NewHandshake(
	moduleInfo ModuleInfo,
	configData ConfigData,
	requirements ModuleRequirements,
	confirmer HandshakeConfirmer,
	logger log.Logger,
) *Handshake {
	return &Handshake{
		moduleInfo:   moduleInfo,
		configData:   configData,
		requirements: requirements,
		confirmer:    confirmer,
		logger:       logger,
	}
}

type HandshakeData struct {
	cli                 *clientWrapper
	initialRemoteConfig []byte
	initialRoutes       RoutingConfig
	initialModulesHosts map[string][]string
}

func (h Handshake) Do(ctx context.Context, host string) (*clientWrapper, error) {
	etpCli := etpclient.NewClient(etpclient.Config{
		ConnectionReadLimit: 4 * 1024 * 1024,
		HttpClient:          http.DefaultClient,
	})
	cli := newClientWrapper(ctx, etpCli, h.logger)

	remoteConfigChan := cli.EventChan(ConfigSendConfigWhenConnected)
	routesChan := cli.EventChan(ConfigSendRoutesWhenConnected)
	requiredModulesChans := make(map[string]chan []byte)
	for _, module := range h.requirements.RequiredModules {
		event := ModuleConnectedEvent(module)
		requiredModulesChans[module] = cli.EventChan(event)
	}
	errorChan := cli.EventChan(ErrorConnection)
	configErrorChan := cli.EventChan(ConfigError)
	cli.OnDefault(func(event string, data []byte) {
		h.logger.Error(ctx, "unexpected event from config service", log.String("event", event), log.Any("data", json.RawMessage(data)))
	})

	ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()
	err := cli.Dial(ctx, host)
	if err != nil {
		return nil, errors.WithMessagef(err, "connect to config service %s", host)
	}

	configData, err := json.Marshal(h.configData)
	if err != nil {
		return nil, errors.WithMessagef(err, "marshal remote config data")
	}
	_, err = cli.EmitWithAck(ctx, ModuleSendConfigSchema, configData)
	if err != nil {
		return nil, errors.WithMessagef(err, "send remote config data")
	}

	err = readError(errorChan)
	if err != nil {
		return nil, errors.WithMessagef(err, "error from config service (%s)", ErrorConnection)
	}

	requirementsData, err := json.Marshal(h.requirements)
	if err != nil {
		return nil, errors.WithMessagef(err, "marshal module requirements")
	}
	_, err = cli.EmitWithAck(ctx, ModuleSendRequirements, requirementsData)
	if err != nil {
		return nil, errors.WithMessagef(err, "send module requirements")
	}

	err = readError(configErrorChan)
	if err != nil {
		return nil, errors.WithMessagef(err, "error from config service (%s)", ConfigError)
	}

	remoteConfig, err := await(ctx, remoteConfigChan, 1*time.Second)
	if err != nil {
		return nil, errors.WithMessage(err, "await remote config")
	}

	var routes RoutingConfig
	if h.requirements.RequireRoutes {
		data, err := await(ctx, routesChan, 1*time.Second)
		if err != nil {
			return nil, errors.WithMessage(err, "await routes")
		}
		routes, err = readRoutes(data)
		if err != nil {
			return nil, errors.WithMessage(err, "read routes")
		}
	}

	requiredModulesHosts := make(map[string][]string)
	for event, ch := range requiredModulesChans {
		data, err := await(ctx, ch, 1*time.Second)
		if err != nil {
			return nil, errors.WithMessagef(err, "await event %s", event)
		}
		hosts, err := readHosts(data)
		if err != nil {
			return nil, errors.WithMessagef(err, "read hosts %s", event)
		}
		module := strings.ReplaceAll(event, "_"+ModuleConnectionSuffix, "")
		requiredModulesHosts[module] = hosts
	}

	data := HandshakeData{
		cli:                 cli,
		initialRemoteConfig: remoteConfig,
		initialRoutes:       routes,
		initialModulesHosts: requiredModulesHosts,
	}
	err = h.confirmer.confirm(ctx, data)
	if err != nil {
		return nil, errors.WithMessage(err, "handshake confirm")
	}

	readyData, err := json.Marshal(h.moduleInfo)
	if err != nil {
		return nil, errors.WithMessage(err, "marshal module ready data")
	}
	_, err = cli.EmitWithAck(ctx, ModuleReady, readyData)
	if err != nil {
		return nil, errors.WithMessage(err, "send module ready data")
	}

	return cli, nil
}

func readError(ch chan []byte) error {
	select {
	case data := <-ch:
		return errors.New(string(data))
	default:
		return nil
	}
}

func await(ctx context.Context, ch chan []byte, timeout time.Duration) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case data := <-ch:
		return data, nil
	}
}

func readRoutes(data []byte) (RoutingConfig, error) {
	var routes RoutingConfig
	err := json.Unmarshal(data, &routes)
	if err != nil {
		return nil, errors.WithMessage(err, "unmarshal routes")
	}
	return routes, nil
}

func readHosts(data []byte) ([]string, error) {
	addresses := make([]AddressConfiguration, 0)
	err := json.Unmarshal(data, &addresses)
	if err != nil {
		return nil, errors.WithMessagef(err, "unmarshal to address")
	}
	hosts := make([]string, 0)
	for _, addr := range addresses {
		host := net.JoinHostPort(addr.IP, addr.Port)
		hosts = append(hosts, host)
	}
	return hosts, nil
}
