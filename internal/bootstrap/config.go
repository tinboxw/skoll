package bootstrap

import "github.com/tinboxw/skoll/pkg/config"

// RuntimeConfig groups bootstrap-time runtime settings.
type RuntimeConfig struct {
	AppConfig  config.AppConfig
	AuthPolicy AuthPolicy
}

func loadRuntimeConfigFromEnv() (RuntimeConfig, error) {
	cfg, err := config.Load()
	if err != nil {
		return RuntimeConfig{}, err
	}

	runtimeCfg := RuntimeConfig{
		AppConfig:  cfg,
		AuthPolicy: loadAuthPolicyFromEnv(),
	}

	if err := validateRuntimeConfig(runtimeCfg); err != nil {
		return RuntimeConfig{}, err
	}

	return runtimeCfg, nil
}
