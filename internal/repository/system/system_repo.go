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

	GetDictionaryTypeByID(ctx context.Context, id shared.ID) (*system.DictionaryType, error)
	GetDictionaryTypeByCode(ctx context.Context, code string) (*system.DictionaryType, error)
	ListDictionaryTypes(ctx context.Context, offset, limit int) ([]system.DictionaryType, error)
	SaveDictionaryType(ctx context.Context, dictType *system.DictionaryType) error
	DeleteDictionaryType(ctx context.Context, id shared.ID) error

	GetDictionaryItemByID(ctx context.Context, id shared.ID) (*system.DictionaryItem, error)
	GetDictionaryItemByTypeAndValue(ctx context.Context, typeCode, value string) (*system.DictionaryItem, error)
	ListDictionaryItems(ctx context.Context, typeCode string, offset, limit int) ([]system.DictionaryItem, error)
	SaveDictionaryItem(ctx context.Context, item *system.DictionaryItem) error
	DeleteDictionaryItem(ctx context.Context, id shared.ID) error
}
