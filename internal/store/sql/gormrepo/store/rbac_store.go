package store

import (
	"context"

	"github.com/tinboxw/skoll/internal/domain/rbac"
	"github.com/tinboxw/skoll/internal/domain/shared"
	storesql "github.com/tinboxw/skoll/internal/store/sql"
	"github.com/tinboxw/skoll/internal/store/sql/gormrepo/model"
	"gorm.io/gorm"
)

type RBACStore struct {
	db *gorm.DB
}

func NewRBACStore(db *gorm.DB) *RBACStore {
	return &RBACStore{db: db}
}

func (s *RBACStore) CreateBinding(ctx context.Context, binding *rbac.Binding) error {
	row := model.BindingModelFromDomain(binding)
	return storesql.SaveModel(ctx, s.db, &row)
}

func (s *RBACStore) DeleteBinding(ctx context.Context, id shared.ID) error {
	return storesql.DeleteByID[model.BindingModel](ctx, s.db, id.String())
}

func (s *RBACStore) ListBindingsBySubject(ctx context.Context, subjectType rbac.SubjectType, subjectID shared.ID) ([]*rbac.Binding, error) {
	var rows []model.BindingModel
	err := s.db.WithContext(ctx).
		Where("subject_type = ? AND subject_id = ?", string(subjectType), subjectID.String()).
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]*rbac.Binding, 0, len(rows))
	for i := range rows {
		out = append(out, rows[i].ToDomain())
	}
	return out, nil
}

func (s *RBACStore) ListPolicyRulesByRoleID(ctx context.Context, roleID shared.ID) ([]rbac.PolicyRule, error) {
	var rows []model.PolicyRuleModel
	err := s.db.WithContext(ctx).
		Where("role_id = ?", roleID.String()).
		Order("id asc").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]rbac.PolicyRule, 0, len(rows))
	for i := range rows {
		out = append(out, rows[i].ToDomain())
	}
	return out, nil
}

func (s *RBACStore) ReplacePolicyRules(ctx context.Context, roleID shared.ID, rules []rbac.PolicyRule) error {
	tx := s.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return tx.Error
	}
	if err := tx.Where("role_id = ?", roleID.String()).Delete(&model.PolicyRuleModel{}).Error; err != nil {
		_ = tx.Rollback().Error
		return err
	}
	for i := range rules {
		row := model.PolicyRuleModelFromDomain(roleID, rules[i])
		if err := tx.Create(&row).Error; err != nil {
			_ = tx.Rollback().Error
			return err
		}
	}
	return tx.Commit().Error
}
