package bootstrap

import (
	"github.com/tinboxw/skoll/pkg/config"
)

type RuntimeConfig struct {
	App config.AppConfig
}

func LoadRuntimeConfigFromEnv() (RuntimeConfig, error) {
	cfg, err := config.LoadFromEnv()
	if err != nil {
		return RuntimeConfig{}, err
	}
	if err := config.Validate(cfg); err != nil {
		return RuntimeConfig{}, err
	}
	return RuntimeConfig{App: cfg}, nil
}
