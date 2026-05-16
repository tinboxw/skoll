package plugin

import (
	"context"

	"github.com/tinboxw/skoll/internal/plugin"
)

type PluginRepository interface {
	Get(ctx context.Context, pluginID string) (*plugin.Info, error)
	List(ctx context.Context) ([]plugin.Info, error)
	Save(ctx context.Context, info plugin.Info) error
	Delete(ctx context.Context, pluginID string) error
}
