package store

import (
	"context"

	"github.com/tinboxw/skoll/internal/domain/shared"
	domainsystem "github.com/tinboxw/skoll/internal/domain/system"
	storesql "github.com/tinboxw/skoll/internal/store/sql"
	"github.com/tinboxw/skoll/internal/store/sql/gormrepo/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type SystemStore struct {
	db           *gorm.DB
	normalizeKey model.Normalizer
}

func NewSystemStore(db *gorm.DB, normalizeKey model.Normalizer) *SystemStore {
	if normalizeKey == nil {
		normalizeKey = defaultNormalize
	}
	return &SystemStore{db: db, normalizeKey: normalizeKey}
}

func (s *SystemStore) GetSettingByID(ctx context.Context, id shared.ID) (*domainsystem.Setting, error) {
	row, err := storesql.FirstWhere[model.SystemSettingModel](ctx, s.db, "id = ?", id.String())
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, nil
	}
	return row.ToDomain(s.normalizeKey), nil
}

func (s *SystemStore) GetSettingByKey(ctx context.Context, key string) (*domainsystem.Setting, error) {
	row, err := storesql.FirstWhere[model.SystemSettingModel](ctx, s.db, "key = ?", s.normalizeKey(key))
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, nil
	}
	return row.ToDomain(s.normalizeKey), nil
}

func (s *SystemStore) ListSettings(ctx context.Context, offset, limit int) ([]*domainsystem.Setting, error) {
	var rows []model.SystemSettingModel
	err := s.db.WithContext(ctx).
		Order(clause.OrderByColumn{Column: clause.Column{Name: "key"}, Desc: false}).
		Offset(offset).
		Limit(limit).
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]*domainsystem.Setting, 0, len(rows))
	for i := range rows {
		out = append(out, rows[i].ToDomain(s.normalizeKey))
	}
	return out, nil
}

func (s *SystemStore) SaveSetting(ctx context.Context, setting *domainsystem.Setting) error {
	row := model.SystemSettingModelFromDomain(setting, s.normalizeKey)
	return storesql.SaveModel(ctx, s.db, &row)
}

func (s *SystemStore) DeleteSetting(ctx context.Context, id shared.ID) error {
	return storesql.DeleteByID[model.SystemSettingModel](ctx, s.db, id.String())
}
