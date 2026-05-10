package mysql

import (
	"context"
	"sort"

	"github.com/tinboxw/skoll/internal/domain/role"
	"github.com/tinboxw/skoll/internal/domain/shared"
	"gorm.io/gorm"
)

type RoleStore struct {
	db *gorm.DB
}

func NewRoleStore(db *gorm.DB) *RoleStore {
	return &RoleStore{db: db}
}

func (s *RoleStore) GetByID(ctx context.Context, id shared.ID) (*role.Role, error) {
	var model roleModel
	err := s.db.WithContext(ctx).Where("id = ?", id.String()).First(&model).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return model.toDomain(), nil
}

func (s *RoleStore) GetByKey(ctx context.Context, key string) (*role.Role, error) {
	var model roleModel
	err := s.db.WithContext(ctx).Where("`key` = ?", normalizeRoleKey(key)).First(&model).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return model.toDomain(), nil
}

func (s *RoleStore) List(ctx context.Context, offset, limit int) ([]*role.Role, error) {
	q := s.db.WithContext(ctx).Order("id asc")
	if offset > 0 {
		q = q.Offset(offset)
	}
	if limit > 0 {
		q = q.Limit(limit)
	}
	var models []roleModel
	if err := q.Find(&models).Error; err != nil {
		return nil, err
	}
	out := make([]*role.Role, 0, len(models))
	for i := range models {
		out = append(out, models[i].toDomain())
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID.String() < out[j].ID.String() })
	return out, nil
}

func (s *RoleStore) Save(ctx context.Context, entity *role.Role) error {
	model := roleModelFromDomain(entity)
	return s.db.WithContext(ctx).Save(&model).Error
}

func (s *RoleStore) Delete(ctx context.Context, id shared.ID) error {
	return s.db.WithContext(ctx).Delete(&roleModel{}, "id = ?", id.String()).Error
}
