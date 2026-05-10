package mysql

import (
	"context"

	"github.com/tinboxw/skoll/internal/domain/role"
	"github.com/tinboxw/skoll/internal/domain/shared"
	storesql "github.com/tinboxw/skoll/internal/store/sql"
	"gorm.io/gorm"
)

type RoleStore struct {
	db *gorm.DB
}

func NewRoleStore(db *gorm.DB) *RoleStore {
	return &RoleStore{db: db}
}

func (s *RoleStore) GetByID(ctx context.Context, id shared.ID) (*role.Role, error) {
	model, err := storesql.FirstWhere[roleModel](ctx, s.db, "id = ?", id.String())
	if err != nil {
		return nil, err
	}
	if model == nil {
		return nil, nil
	}
	return model.toDomain(), nil
}

func (s *RoleStore) GetByKey(ctx context.Context, key string) (*role.Role, error) {
	model, err := storesql.FirstWhere[roleModel](ctx, s.db, "`key` = ?", normalizeRoleKey(key))
	if err != nil {
		return nil, err
	}
	if model == nil {
		return nil, nil
	}
	return model.toDomain(), nil
}

func (s *RoleStore) List(ctx context.Context, offset, limit int) ([]*role.Role, error) {
	models, err := storesql.ListOrdered[roleModel](ctx, s.db, "id asc", offset, limit)
	if err != nil {
		return nil, err
	}
	out := make([]*role.Role, 0, len(models))
	for i := range models {
		out = append(out, models[i].toDomain())
	}
	return out, nil
}

func (s *RoleStore) Save(ctx context.Context, entity *role.Role) error {
	model := roleModelFromDomain(entity)
	return storesql.SaveModel(ctx, s.db, &model)
}

func (s *RoleStore) Delete(ctx context.Context, id shared.ID) error {
	return storesql.DeleteByID[roleModel](ctx, s.db, id.String())
}
