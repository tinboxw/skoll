package store

import (
	"context"

	"github.com/tinboxw/skoll/internal/domain/shared"
	"github.com/tinboxw/skoll/internal/domain/user"
	storesql "github.com/tinboxw/skoll/internal/store/sql"
	"github.com/tinboxw/skoll/internal/store/sql/gormrepo/model"
	"gorm.io/gorm"
)

type UserStore struct {
	db *gorm.DB
}

func NewUserStore(db *gorm.DB) *UserStore {
	return &UserStore{db: db}
}

func (s *UserStore) GetByID(ctx context.Context, id shared.ID) (*user.User, error) {
	row, err := storesql.FirstWhere[model.UserModel](ctx, s.db, "id = ?", id.String())
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, nil
	}
	return row.ToDomain(), nil
}

func (s *UserStore) GetByEmail(ctx context.Context, email user.Email) (*user.User, error) {
	row, err := storesql.FirstWhere[model.UserModel](ctx, s.db, "email = ?", email.String())
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, nil
	}
	return row.ToDomain(), nil
}

func (s *UserStore) List(ctx context.Context, offset, limit int) ([]*user.User, error) {
	rows, err := storesql.ListOrdered[model.UserModel](ctx, s.db, "id asc", offset, limit)
	if err != nil {
		return nil, err
	}
	out := make([]*user.User, 0, len(rows))
	for i := range rows {
		out = append(out, rows[i].ToDomain())
	}
	return out, nil
}

func (s *UserStore) Save(ctx context.Context, entity *user.User) error {
	row := model.UserModelFromDomain(entity)
	return storesql.SaveModel(ctx, s.db, &row)
}

func (s *UserStore) Delete(ctx context.Context, id shared.ID) error {
	return storesql.DeleteByID[model.UserModel](ctx, s.db, id.String())
}
