package system

import (
	"context"
	"time"

	"github.com/tinboxw/skoll/internal/cache"
	domainsystem "github.com/tinboxw/skoll/internal/domain/system"
	systemrepo "github.com/tinboxw/skoll/internal/repository/system"
)

type Service interface {
	Upsert(ctx context.Context, in UpsertInput) (*domainsystem.Setting, error)
	GetByKey(ctx context.Context, key string) (*domainsystem.Setting, error)
	List(ctx context.Context, in ListInput) ([]*domainsystem.Setting, error)
	Reset(ctx context.Context) (int, error)

	SaveDictionaryType(ctx context.Context, in DictionaryTypeInput) (*domainsystem.DictionaryType, error)
	GetDictionaryTypeByCode(ctx context.Context, code string) (*domainsystem.DictionaryType, error)
	ListDictionaryTypes(ctx context.Context, in DictionaryTypeListInput) ([]domainsystem.DictionaryType, error)
	DeleteDictionaryType(ctx context.Context, id string) error
	SaveDictionaryItem(ctx context.Context, in DictionaryItemInput) (*domainsystem.DictionaryItem, error)
	GetDictionaryItemByTypeAndValue(ctx context.Context, typeCode, value string) (*domainsystem.DictionaryItem, error)
	ListDictionaryItems(ctx context.Context, in DictionaryItemListInput) ([]domainsystem.DictionaryItem, error)
	DeleteDictionaryItem(ctx context.Context, id string) error

	RegisterConfigSchema(ctx context.Context, in ConfigSchemaInput) (*ConfigSchema, error)
	GetConfigSchema(ctx context.Context, scope ConfigScope, owner string) (*ConfigSchema, error)
	ListConfigSchemas(ctx context.Context, in ConfigSchemaListInput) ([]ConfigSchema, error)
	ValidateConfigValues(ctx context.Context, scope ConfigScope, owner string, values map[string]any) (map[string]any, error)
}

func NewCachedService(repo systemrepo.SystemRepository, dictionaryCache cache.BytesCache, ttl time.Duration) Service {
	svc := newServiceImpl(repo)
	svc.dictionaryCache = dictionaryCache
	if ttl > 0 {
		svc.dictionaryTTL = ttl
	}
	return svc
}
