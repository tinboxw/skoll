package bootstrap

import "errors"

func ValidateRuntimeConfig(cfg RuntimeConfig) error {
	if cfg.App.Server.Address == "" {
		return errors.New("server address is required")
	}
	if cfg.App.Server.ShutdownTimeout <= 0 {
		return errors.New("shutdown timeout must be > 0")
	}
	return nil
}
