package bootstrap

import "github.com/tinboxw/skoll/pkg/logging"

type Container struct {
	Config RuntimeConfig
	Logger logging.Logger
}

func NewContainer(cfg RuntimeConfig) *Container {
	return &Container{
		Config: cfg,
		Logger: logging.NewZapCompatibleLogger(cfg.App.Log.Level),
	}
}
