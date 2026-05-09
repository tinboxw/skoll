package bootstrap

import (
	"fmt"
	"net"

	"github.com/tinboxw/skoll/pkg/config"
)

func validateRuntimeConfig(cfg RuntimeConfig) error {
	if err := config.Validate(cfg.AppConfig); err != nil {
		return err
	}

	if _, err := net.ResolveTCPAddr("tcp", cfg.AppConfig.Server.Address); err != nil {
		return fmt.Errorf("invalid server address %q: %w", cfg.AppConfig.Server.Address, err)
	}

	return nil
}
