package mysql

import (
	"context"

	"github.com/tinboxw/skoll/internal/domain/shared"
	"github.com/tinboxw/skoll/internal/domain/user"
	storesql "github.com/tinboxw/skoll/internal/store/sql"
	"gorm.io/gorm"
)

type UserStore struct {
	db *gorm.DB
}

func NewUserStore(db *gorm.DB) *UserStore {
	return &UserStore{db: db}
}

func (s *UserStore) GetByID(ctx context.Context, id shared.ID) (*user.User, error) {
	model, err := storesql.FirstWhere[userModel](ctx, s.db, "id = ?", id.String())
	if err != nil {
		return nil, err
	}
	if model == nil {
		return nil, nil
	}
	return model.toDomain(), nil
}

func (s *UserStore) GetByEmail(ctx context.Context, email user.Email) (*user.User, error) {
	model, err := storesql.FirstWhere[userModel](ctx, s.db, "email = ?", email.String())
	if err != nil {
		return nil, err
	}
	if model == nil {
		return nil, nil
	}
	return model.toDomain(), nil
}

func (s *UserStore) List(ctx context.Context, offset, limit int) ([]*user.User, error) {
	models, err := storesql.ListOrdered[userModel](ctx, s.db, "id asc", offset, limit)
	if err != nil {
		return nil, err
	}
	out := make([]*user.User, 0, len(models))
	for i := range models {
		out = append(out, models[i].toDomain())
	}
	return out, nil
}

func (s *UserStore) Save(ctx context.Context, entity *user.User) error {
	model := userModelFromDomain(entity)
	return storesql.SaveModel(ctx, s.db, &model)
}

func (s *UserStore) Delete(ctx context.Context, id shared.ID) error {
	return storesql.DeleteByID[userModel](ctx, s.db, id.String())
}
