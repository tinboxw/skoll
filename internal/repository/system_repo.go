package repository

import (
	"context"

	"github.com/tinboxw/skoll/internal/domain/system"
)

type SystemRepository interface {
	GetConfig(ctx context.Context, key string) (system.ConfigItem, error)
	SaveConfig(ctx context.Context, item system.ConfigItem) error
}
