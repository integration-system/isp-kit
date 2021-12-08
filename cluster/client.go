package cluster

import (
	"context"
	"time"

	"github.com/integration-system/isp-kit/lb"
	"github.com/integration-system/isp-kit/log"
	"github.com/integration-system/isp-kit/requestid"
	"github.com/pkg/errors"
)

type Client struct {
	moduleInfo   ModuleInfo
	configData   ConfigData
	lb           *lb.RoundRobin
	eventHandler *EventHandler
	logger       log.Logger
}

func NewClient(moduleInfo ModuleInfo, configData ConfigData, hosts []string, eventHandler *EventHandler, logger log.Logger) *Client {
	return &Client{
		moduleInfo:   moduleInfo,
		configData:   configData,
		lb:           lb.NewRoundRobin(hosts),
		eventHandler: eventHandler,
		logger:       logger,
	}
}

func (c *Client) Run(ctx context.Context) error {
	for {
		err := c.runSession(ctx)
		if errors.Is(err, context.Canceled) {
			return nil
		}

		select {
		case <-ctx.Done():
			return nil
		case <-time.After(1 * time.Second):
		}
	}
}

func (c *Client) runSession(ctx context.Context) error {
	host, err := c.lb.Next()
	if err != nil {
		return errors.WithMessage(err, "peek config service host")
	}

	sessionId := requestid.Next()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	ctx = log.ToContext(ctx, log.String("configService", host), log.String("sessionId", sessionId))

	requiredModules := make([]string, 0)
	for moduleName := range c.eventHandler.requiredModules {
		requiredModules = append(requiredModules, moduleName)
	}
	requirements := ModuleRequirements{
		RequiredModules: requiredModules,
		RequireRoutes:   c.eventHandler.routesReceiver != nil,
	}

	handshake := NewHandshake(c.moduleInfo, c.configData, requirements, c, c.logger)
	cli, err := handshake.Do(ctx, host)
	if err != nil {
		return errors.WithMessage(err, "do handshake")
	}
	defer cli.Close()

	for moduleName := range c.eventHandler.requiredModules {
		event := ModuleConnectedEvent(moduleName)
		upgrader := c.eventHandler.requiredModules[moduleName]
		cli.On(event, func(data []byte) {
			hosts, err := readHosts(data)
			if err != nil {
				c.logger.Error(ctx, err)
				return
			}
			upgrader.Upgrade(hosts)
		})
	}
	cli.On(ConfigSendConfigChanged, func(data []byte) {
		err := c.applyRemoteConfig(ctx, data)
		if err != nil {
			c.logger.Error(ctx, err)
		}
	})
	cli.On(ConfigSendRoutesChanged, func(data []byte) {
		routes, err := readRoutes(data)
		if err != nil {
			c.logger.Error(ctx, err)
			return
		}
		err = c.eventHandler.routesReceiver.ReceiveRoutes(routes)
		if err != nil {
			c.logger.Error(ctx, err)
		}
	})

	for {
		err := c.waitAndPing(ctx, cli)
		if err != nil {
			c.logger.Error(ctx, err)
			return err
		}
	}
}

func (c *Client) confirm(ctx context.Context, data HandshakeData) error {
	for module, hosts := range data.initialModulesHosts {
		upgrader := c.eventHandler.requiredModules[module]
		upgrader.Upgrade(hosts)
	}

	if c.eventHandler.remoteConfigReceiver != nil {
		return c.applyRemoteConfig(ctx, data.initialRemoteConfig)
	}

	if c.eventHandler.routesReceiver != nil {
		err := c.eventHandler.routesReceiver.ReceiveRoutes(data.initialRoutes)
		if err != nil {
			return errors.WithMessagef(err, "receive routes")
		}
	}

	return nil
}

func (c *Client) applyRemoteConfig(ctx context.Context, config []byte) (err error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	errChan := make(chan error)

	c.logger.Info(ctx, "remote config applying...")
	defer func() {
		if err != nil {
			c.logger.Error(ctx, errors.WithMessage(err, "remote config applying"))
			return
		}
		c.logger.Info(ctx, "remote config successfully applied")
	}()

	go func() {
		errChan <- c.eventHandler.remoteConfigReceiver.ReceiveConfig(config)
	}()

	select {
	case err := <-errChan:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (c *Client) waitAndPing(ctx context.Context, cli *clientWrapper) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(1 * time.Second):
	}

	ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()

	err := cli.Ping(ctx)
	if err != nil && !errors.Is(err, context.Canceled) {
		return errors.WithMessage(err, "ping config service")
	}

	return err
}
