package gormrepo

import (
	"context"
	"strconv"
	"strings"

	"github.com/tinboxw/skoll/internal/domain/shared"
	"github.com/tinboxw/skoll/internal/domain/user"
	userrepo "github.com/tinboxw/skoll/internal/repository/user"
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
	idValue, err := strconv.ParseUint(strings.TrimSpace(id.String()), 10, 64)
	if err != nil {
		return nil, nil
	}
	row, err := storesql.FirstWhere[UserModel](ctx, s.db, "id = ?", idValue)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, nil
	}
	return row.ToDomain(), nil
}

func (s *UserStore) GetByAccount(ctx context.Context, account string) (*user.User, error) {
	target := strings.TrimSpace(strings.ToLower(account))
	if target == "" {
		return nil, nil
	}
	query, args, err := storesql.BuildSelectByLower(s.db.Dialector.Name(), UserModel{}.TableName(), "account", target, 1)
	if err != nil {
		return nil, err
	}
	var row UserModel
	tx := s.db.WithContext(ctx).Raw(query, args...).Scan(&row)
	if tx.Error != nil {
		return nil, tx.Error
	}
	if tx.RowsAffected == 0 {
		return nil, nil
	}
	return row.ToDomain(), nil
}

func (s *UserStore) GetByEmail(ctx context.Context, email user.Email) (*user.User, error) {
	row, err := storesql.FirstWhere[UserModel](ctx, s.db, "email = ?", email.String())
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, nil
	}
	return row.ToDomain(), nil
}

func (s *UserStore) List(ctx context.Context, offset, limit int) ([]*user.User, error) {
	rows, err := storesql.ListOrdered[UserModel](ctx, s.db, "id asc", offset, limit)
	if err != nil {
		return nil, err
	}
	out := make([]*user.User, 0, len(rows))
	for i := range rows {
		out = append(out, rows[i].ToDomain())
	}
	return out, nil
}

func (s *UserStore) ListFiltered(ctx context.Context, filter userrepo.ListFilter, offset, limit int) ([]*user.User, error) {
	if filter.Empty() {
		return s.List(ctx, offset, limit)
	}

	userIDs := make([]uint64, 0, len(filter.UserIDs))
	for _, id := range filter.NormalizedUserIDs() {
		idValue, err := strconv.ParseUint(strings.TrimSpace(id.String()), 10, 64)
		if err != nil || idValue == 0 {
			continue
		}
		userIDs = append(userIDs, idValue)
	}
	departmentIDs := filter.NormalizedDepartmentIDs()
	if len(userIDs) == 0 && len(departmentIDs) == 0 {
		return []*user.User{}, nil
	}

	query := s.db.WithContext(ctx).Model(&UserModel{})
	switch {
	case len(userIDs) > 0 && len(departmentIDs) > 0:
		query = query.Where("id IN ? OR department_id IN ?", userIDs, departmentIDs)
	case len(userIDs) > 0:
		query = query.Where("id IN ?", userIDs)
	default:
		query = query.Where("department_id IN ?", departmentIDs)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}
	if limit > 0 {
		query = query.Limit(limit)
	}

	var rows []UserModel
	if err := query.Order("id asc").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]*user.User, 0, len(rows))
	for i := range rows {
		out = append(out, rows[i].ToDomain())
	}
	return out, nil
}

func (s *UserStore) Save(ctx context.Context, entity *user.User) error {
	row := UserModelFromDomain(entity)
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

func (s *UserStore) Delete(ctx context.Context, id shared.ID) error {
	idValue, err := strconv.ParseUint(strings.TrimSpace(id.String()), 10, 64)
	if err != nil {
		return nil
	}
	return storesql.DeleteByID[UserModel](ctx, s.db, idValue)
}
