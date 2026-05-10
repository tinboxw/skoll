package mysql

import (
	"context"
	"sort"

	"github.com/tinboxw/skoll/internal/domain/shared"
	"github.com/tinboxw/skoll/internal/domain/user"
	"gorm.io/gorm"
)

type UserStore struct {
	db *gorm.DB
}

func NewUserStore(db *gorm.DB) *UserStore {
	return &UserStore{db: db}
}

func (s *UserStore) GetByID(ctx context.Context, id shared.ID) (*user.User, error) {
	var model userModel
	err := s.db.WithContext(ctx).Where("id = ?", id.String()).First(&model).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return model.toDomain(), nil
}

func (s *UserStore) GetByEmail(ctx context.Context, email user.Email) (*user.User, error) {
	var model userModel
	err := s.db.WithContext(ctx).Where("email = ?", email.String()).First(&model).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return model.toDomain(), nil
}

func (s *UserStore) List(ctx context.Context, offset, limit int) ([]*user.User, error) {
	q := s.db.WithContext(ctx).Order("id asc")
	if offset > 0 {
		q = q.Offset(offset)
	}
	if limit > 0 {
		q = q.Limit(limit)
	}
	var models []userModel
	if err := q.Find(&models).Error; err != nil {
		return nil, err
	}
	out := make([]*user.User, 0, len(models))
	for i := range models {
		out = append(out, models[i].toDomain())
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID.String() < out[j].ID.String() })
	return out, nil
}

func (s *UserStore) Save(ctx context.Context, entity *user.User) error {
	model := userModelFromDomain(entity)
	return s.db.WithContext(ctx).Save(&model).Error
}

func (s *UserStore) Delete(ctx context.Context, id shared.ID) error {
	return s.db.WithContext(ctx).Delete(&userModel{}, "id = ?", id.String()).Error
}
