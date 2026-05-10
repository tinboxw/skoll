package mysql

import (
	"context"
	"sort"

	"github.com/tinboxw/skoll/internal/domain/shared"
	domainsystem "github.com/tinboxw/skoll/internal/domain/system"
	"gorm.io/gorm"
)

type SystemStore struct {
	db *gorm.DB
}

func NewSystemStore(db *gorm.DB) *SystemStore {
	return &SystemStore{db: db}
}

func (s *SystemStore) GetSettingByID(ctx context.Context, id shared.ID) (*domainsystem.Setting, error) {
	var model systemSettingModel
	err := s.db.WithContext(ctx).Where("id = ?", id.String()).First(&model).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return model.toDomain(), nil
}

func (s *SystemStore) GetSettingByKey(ctx context.Context, key string) (*domainsystem.Setting, error) {
	var model systemSettingModel
	err := s.db.WithContext(ctx).Where("`key` = ?", normalizeSettingKey(key)).First(&model).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return model.toDomain(), nil
}

func (s *SystemStore) ListSettings(ctx context.Context, offset, limit int) ([]*domainsystem.Setting, error) {
	q := s.db.WithContext(ctx).Order("`key` asc")
	if offset > 0 {
		q = q.Offset(offset)
	}
	if limit > 0 {
		q = q.Limit(limit)
	}
	var models []systemSettingModel
	if err := q.Find(&models).Error; err != nil {
		return nil, err
	}
	out := make([]*domainsystem.Setting, 0, len(models))
	for i := range models {
		out = append(out, models[i].toDomain())
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Key < out[j].Key })
	return out, nil
}

func (s *SystemStore) SaveSetting(ctx context.Context, setting *domainsystem.Setting) error {
	model := systemSettingModelFromDomain(setting)
	return s.db.WithContext(ctx).Save(&model).Error
}

func (s *SystemStore) DeleteSetting(ctx context.Context, id shared.ID) error {
	return s.db.WithContext(ctx).Delete(&systemSettingModel{}, "id = ?", id.String()).Error
}
