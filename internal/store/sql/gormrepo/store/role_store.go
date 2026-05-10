package store

import (
	"context"

	"github.com/tinboxw/skoll/internal/domain/role"
	"github.com/tinboxw/skoll/internal/domain/shared"
	storesql "github.com/tinboxw/skoll/internal/store/sql"
	"github.com/tinboxw/skoll/internal/store/sql/gormrepo/model"
	"gorm.io/gorm"
)

type RoleStore struct {
	db           *gorm.DB
	normalizeKey model.Normalizer
}

func NewRoleStore(db *gorm.DB, normalizeKey model.Normalizer) *RoleStore {
	if normalizeKey == nil {
		normalizeKey = defaultNormalize
	}
	return &RoleStore{db: db, normalizeKey: normalizeKey}
}

func (s *RoleStore) GetByID(ctx context.Context, id shared.ID) (*role.Role, error) {
	row, err := storesql.FirstWhere[model.RoleModel](ctx, s.db, "id = ?", id.String())
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, nil
	}
	return row.ToDomain(s.normalizeKey), nil
}

func (s *RoleStore) GetByKey(ctx context.Context, key string) (*role.Role, error) {
	row, err := storesql.FirstWhere[model.RoleModel](ctx, s.db, "key = ?", s.normalizeKey(key))
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, nil
	}
	return row.ToDomain(s.normalizeKey), nil
}

func (s *RoleStore) List(ctx context.Context, offset, limit int) ([]*role.Role, error) {
	rows, err := storesql.ListOrdered[model.RoleModel](ctx, s.db, "id asc", offset, limit)
	if err != nil {
		return nil, err
	}
	out := make([]*role.Role, 0, len(rows))
	for i := range rows {
		out = append(out, rows[i].ToDomain(s.normalizeKey))
	}
	return out, nil
}

func (s *RoleStore) Save(ctx context.Context, entity *role.Role) error {
	row := model.RoleModelFromDomain(entity, s.normalizeKey)
	return storesql.SaveModel(ctx, s.db, &row)
}

func (s *RoleStore) Delete(ctx context.Context, id shared.ID) error {
	return storesql.DeleteByID[model.RoleModel](ctx, s.db, id.String())
}
