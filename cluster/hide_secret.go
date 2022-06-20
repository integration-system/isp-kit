package cluster

import (
	"strings"

	"github.com/integration-system/bellows"
	"github.com/integration-system/isp-kit/json"
	"github.com/pkg/errors"
)

var (
	tagConfigSecrets    = []string{"password", "secret", "token"}
	hidingSecretsEvents = map[string]bool{
		ConfigSendConfigWhenConnected: true,
		ConfigSendConfigChanged:       true,
	}
)

func HideSecrets(data []byte) ([]byte, error) {
	config := make(map[string]interface{})
	err := json.Unmarshal(data, &config)
	if err != nil {
		return nil, errors.WithMessage(err, "unmarshal config for replacement secret")
	}

	flattenConf := bellows.Flatten(config)

	for key := range flattenConf {
		if flattenConf[key] == "" {
			continue
		}
		for _, tag := range tagConfigSecrets {
			if strings.Contains(strings.ToLower(key), tag) {
				flattenConf[key] = "***"
			}
		}
	}

	expandConf := bellows.Expand(flattenConf)
	if expandConf == nil {
		expandConf = make(map[string]interface{})
	}

	config, ok := expandConf.(map[string]interface{})
	if !ok {
		return nil, errors.WithMessagef(err, "unexpected type from bellows, expected map, got %T", config)
	}

	data, err = json.Marshal(config)
	if err != nil {
		return nil, errors.WithMessage(err, "marshal config for replacement secret")
	}

	return data, nil
}