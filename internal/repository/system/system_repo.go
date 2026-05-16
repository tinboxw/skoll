package system

import (
	"context"

	"github.com/tinboxw/skoll/internal/domain/shared"
	"github.com/tinboxw/skoll/internal/domain/system"
)

type SystemRepository interface {
	GetSettingByID(ctx context.Context, id shared.ID) (*system.Setting, error)
	GetSettingByKey(ctx context.Context, key string) (*system.Setting, error)
	ListSettings(ctx context.Context, offset, limit int) ([]*system.Setting, error)
	SaveSetting(ctx context.Context, setting *system.Setting) error
	DeleteSetting(ctx context.Context, id shared.ID) error
}
