package mysql

import (
	"context"

	"github.com/tinboxw/skoll/internal/domain/rbac"
	"github.com/tinboxw/skoll/internal/domain/shared"
	"gorm.io/gorm"
)

type RBACStore struct {
	db *gorm.DB
}

func NewRBACStore(db *gorm.DB) *RBACStore {
	return &RBACStore{db: db}
}

func (s *RBACStore) CreateBinding(ctx context.Context, binding *rbac.Binding) error {
	model := bindingModelFromDomain(binding)
	return s.db.WithContext(ctx).Save(&model).Error
}

func (s *RBACStore) DeleteBinding(ctx context.Context, id shared.ID) error {
	return s.db.WithContext(ctx).Delete(&bindingModel{}, "id = ?", id.String()).Error
}

func (s *RBACStore) ListBindingsBySubject(ctx context.Context, subjectType rbac.SubjectType, subjectID shared.ID) ([]*rbac.Binding, error) {
	var models []bindingModel
	err := s.db.WithContext(ctx).
		Where("subject_type = ? AND subject_id = ?", string(subjectType), subjectID.String()).
		Find(&models).Error
	if err != nil {
		return nil, err
	}
	out := make([]*rbac.Binding, 0, len(models))
	for i := range models {
		out = append(out, models[i].toDomain())
	}
	return out, nil
}

func (s *RBACStore) ListPolicyRulesByRoleID(ctx context.Context, roleID shared.ID) ([]rbac.PolicyRule, error) {
	var models []policyRuleModel
	err := s.db.WithContext(ctx).
		Where("role_id = ?", roleID.String()).
		Order("id asc").
		Find(&models).Error
	if err != nil {
		return nil, err
	}
	out := make([]rbac.PolicyRule, 0, len(models))
	for i := range models {
		out = append(out, models[i].toDomain())
	}
	return out, nil
}

func (s *RBACStore) ReplacePolicyRules(ctx context.Context, roleID shared.ID, rules []rbac.PolicyRule) error {
	tx := s.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return tx.Error
	}
	if err := tx.Where("role_id = ?", roleID.String()).Delete(&policyRuleModel{}).Error; err != nil {
		_ = tx.Rollback().Error
		return err
	}
	for i := range rules {
		model := policyRuleModelFromDomain(roleID, rules[i])
		if err := tx.Create(&model).Error; err != nil {
			_ = tx.Rollback().Error
			return err
		}
	}
	return tx.Commit().Error
}
