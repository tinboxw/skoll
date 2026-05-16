package gormrepo

import (
	"context"
	"strconv"
	"strings"

	"github.com/tinboxw/skoll/internal/domain/role"
	"github.com/tinboxw/skoll/internal/domain/shared"
	storesql "github.com/tinboxw/skoll/internal/store/sql"
	"gorm.io/gorm"
)

type RoleStore struct {
	db           *gorm.DB
	normalizeKey Normalizer
}

func NewRoleStore(db *gorm.DB, normalizeKey Normalizer) *RoleStore {
	if normalizeKey == nil {
		normalizeKey = defaultNormalize
	}
	return &RoleStore{db: db, normalizeKey: normalizeKey}
}

func (s *RoleStore) GetByID(ctx context.Context, id shared.ID) (*role.Role, error) {
	idValue, err := strconv.ParseUint(strings.TrimSpace(id.String()), 10, 64)
	if err != nil {
		return nil, nil
	}
	row, err := storesql.FirstWhere[RoleModel](ctx, s.db, "id = ?", idValue)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, nil
	}
	return row.ToDomain(s.normalizeKey), nil
}

func (s *RoleStore) GetByKey(ctx context.Context, key string) (*role.Role, error) {
	row, err := storesql.FirstWhere[RoleModel](ctx, s.db, map[string]any{"key": s.normalizeKey(key)})
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, nil
	}
	return row.ToDomain(s.normalizeKey), nil
}

func (s *RoleStore) List(ctx context.Context, offset, limit int) ([]*role.Role, error) {
	rows, err := storesql.ListOrdered[RoleModel](ctx, s.db, "id asc", offset, limit)
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
	row := RoleModelFromDomain(entity, s.normalizeKey)
	if row.ID == 0 {
		if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
			return err
		}
		entity.ID = shared.ID(strconv.FormatUint(row.ID, 10))
		return nil
	}
	if err := storesql.SaveModel(ctx, s.db, &row); err != nil {
		return err
	}
	entity.ID = shared.ID(strconv.FormatUint(row.ID, 10))
	return nil
}

func (s *RoleStore) Delete(ctx context.Context, id shared.ID) error {
	idValue, err := strconv.ParseUint(strings.TrimSpace(id.String()), 10, 64)
	if err != nil {
		return nil
	}
	return storesql.DeleteByID[RoleModel](ctx, s.db, idValue)
}
