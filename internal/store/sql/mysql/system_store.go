package mysql

import (
	"context"

	"github.com/tinboxw/skoll/internal/domain/shared"
	domainsystem "github.com/tinboxw/skoll/internal/domain/system"
	storesql "github.com/tinboxw/skoll/internal/store/sql"
	"gorm.io/gorm"
)

type SystemStore struct {
	db *gorm.DB
}

func NewSystemStore(db *gorm.DB) *SystemStore {
	return &SystemStore{db: db}
}

func (s *SystemStore) GetSettingByID(ctx context.Context, id shared.ID) (*domainsystem.Setting, error) {
	model, err := storesql.FirstWhere[systemSettingModel](ctx, s.db, "id = ?", id.String())
	if err != nil {
		return nil, err
	}
	if model == nil {
		return nil, nil
	}
	return model.toDomain(), nil
}

func (s *SystemStore) GetSettingByKey(ctx context.Context, key string) (*domainsystem.Setting, error) {
	model, err := storesql.FirstWhere[systemSettingModel](ctx, s.db, "`key` = ?", normalizeSettingKey(key))
	if err != nil {
		return nil, err
	}
	if model == nil {
		return nil, nil
	}
	return model.toDomain(), nil
}

func (s *SystemStore) ListSettings(ctx context.Context, offset, limit int) ([]*domainsystem.Setting, error) {
	models, err := storesql.ListOrdered[systemSettingModel](ctx, s.db, "`key` asc", offset, limit)
	if err != nil {
		return nil, err
	}
	out := make([]*domainsystem.Setting, 0, len(models))
	for i := range models {
		out = append(out, models[i].toDomain())
	}
	return out, nil
}

func (s *SystemStore) SaveSetting(ctx context.Context, setting *domainsystem.Setting) error {
	model := systemSettingModelFromDomain(setting)
	return storesql.SaveModel(ctx, s.db, &model)
}

func (s *SystemStore) DeleteSetting(ctx context.Context, id shared.ID) error {
	return storesql.DeleteByID[systemSettingModel](ctx, s.db, id.String())
}
