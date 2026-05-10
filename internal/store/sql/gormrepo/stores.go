package gormrepo

import (
	"context"
	"strings"

	"github.com/tinboxw/skoll/internal/domain/rbac"
	"github.com/tinboxw/skoll/internal/domain/role"
	"github.com/tinboxw/skoll/internal/domain/shared"
	domainsystem "github.com/tinboxw/skoll/internal/domain/system"
	"github.com/tinboxw/skoll/internal/domain/user"
	storesql "github.com/tinboxw/skoll/internal/store/sql"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type UserStore struct {
	db *gorm.DB
}

func NewUserStore(db *gorm.DB) *UserStore {
	return &UserStore{db: db}
}

func (s *UserStore) GetByID(ctx context.Context, id shared.ID) (*user.User, error) {
	model, err := storesql.FirstWhere[UserModel](ctx, s.db, "id = ?", id.String())
	if err != nil {
		return nil, err
	}
	if model == nil {
		return nil, nil
	}
	return model.ToDomain(), nil
}

func (s *UserStore) GetByEmail(ctx context.Context, email user.Email) (*user.User, error) {
	model, err := storesql.FirstWhere[UserModel](ctx, s.db, "email = ?", email.String())
	if err != nil {
		return nil, err
	}
	if model == nil {
		return nil, nil
	}
	return model.ToDomain(), nil
}

func (s *UserStore) List(ctx context.Context, offset, limit int) ([]*user.User, error) {
	models, err := storesql.ListOrdered[UserModel](ctx, s.db, "id asc", offset, limit)
	if err != nil {
		return nil, err
	}
	out := make([]*user.User, 0, len(models))
	for i := range models {
		out = append(out, models[i].ToDomain())
	}
	return out, nil
}

func (s *UserStore) Save(ctx context.Context, entity *user.User) error {
	model := UserModelFromDomain(entity)
	return storesql.SaveModel(ctx, s.db, &model)
}

func (s *UserStore) Delete(ctx context.Context, id shared.ID) error {
	return storesql.DeleteByID[UserModel](ctx, s.db, id.String())
}

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
	model, err := storesql.FirstWhere[RoleModel](ctx, s.db, "id = ?", id.String())
	if err != nil {
		return nil, err
	}
	if model == nil {
		return nil, nil
	}
	return model.ToDomain(s.normalizeKey), nil
}

func (s *RoleStore) GetByKey(ctx context.Context, key string) (*role.Role, error) {
	model, err := storesql.FirstWhere[RoleModel](ctx, s.db, "key = ?", s.normalizeKey(key))
	if err != nil {
		return nil, err
	}
	if model == nil {
		return nil, nil
	}
	return model.ToDomain(s.normalizeKey), nil
}

func (s *RoleStore) List(ctx context.Context, offset, limit int) ([]*role.Role, error) {
	models, err := storesql.ListOrdered[RoleModel](ctx, s.db, "id asc", offset, limit)
	if err != nil {
		return nil, err
	}
	out := make([]*role.Role, 0, len(models))
	for i := range models {
		out = append(out, models[i].ToDomain(s.normalizeKey))
	}
	return out, nil
}

func (s *RoleStore) Save(ctx context.Context, entity *role.Role) error {
	model := RoleModelFromDomain(entity, s.normalizeKey)
	return storesql.SaveModel(ctx, s.db, &model)
}

func (s *RoleStore) Delete(ctx context.Context, id shared.ID) error {
	return storesql.DeleteByID[RoleModel](ctx, s.db, id.String())
}

type SystemStore struct {
	db           *gorm.DB
	normalizeKey Normalizer
}

func NewSystemStore(db *gorm.DB, normalizeKey Normalizer) *SystemStore {
	if normalizeKey == nil {
		normalizeKey = defaultNormalize
	}
	return &SystemStore{db: db, normalizeKey: normalizeKey}
}

func (s *SystemStore) GetSettingByID(ctx context.Context, id shared.ID) (*domainsystem.Setting, error) {
	model, err := storesql.FirstWhere[SystemSettingModel](ctx, s.db, "id = ?", id.String())
	if err != nil {
		return nil, err
	}
	if model == nil {
		return nil, nil
	}
	return model.ToDomain(s.normalizeKey), nil
}

func (s *SystemStore) GetSettingByKey(ctx context.Context, key string) (*domainsystem.Setting, error) {
	model, err := storesql.FirstWhere[SystemSettingModel](ctx, s.db, "key = ?", s.normalizeKey(key))
	if err != nil {
		return nil, err
	}
	if model == nil {
		return nil, nil
	}
	return model.ToDomain(s.normalizeKey), nil
}

func (s *SystemStore) ListSettings(ctx context.Context, offset, limit int) ([]*domainsystem.Setting, error) {
	var models []SystemSettingModel
	err := s.db.WithContext(ctx).
		Order(clause.OrderByColumn{Column: clause.Column{Name: "key"}, Desc: false}).
		Offset(offset).
		Limit(limit).
		Find(&models).Error
	if err != nil {
		return nil, err
	}
	out := make([]*domainsystem.Setting, 0, len(models))
	for i := range models {
		out = append(out, models[i].ToDomain(s.normalizeKey))
	}
	return out, nil
}

func (s *SystemStore) SaveSetting(ctx context.Context, setting *domainsystem.Setting) error {
	model := SystemSettingModelFromDomain(setting, s.normalizeKey)
	return storesql.SaveModel(ctx, s.db, &model)
}

func (s *SystemStore) DeleteSetting(ctx context.Context, id shared.ID) error {
	return storesql.DeleteByID[SystemSettingModel](ctx, s.db, id.String())
}

type RBACStore struct {
	db *gorm.DB
}

func NewRBACStore(db *gorm.DB) *RBACStore {
	return &RBACStore{db: db}
}

func (s *RBACStore) CreateBinding(ctx context.Context, binding *rbac.Binding) error {
	model := BindingModelFromDomain(binding)
	return storesql.SaveModel(ctx, s.db, &model)
}

func (s *RBACStore) DeleteBinding(ctx context.Context, id shared.ID) error {
	return storesql.DeleteByID[BindingModel](ctx, s.db, id.String())
}

func (s *RBACStore) ListBindingsBySubject(ctx context.Context, subjectType rbac.SubjectType, subjectID shared.ID) ([]*rbac.Binding, error) {
	var models []BindingModel
	err := s.db.WithContext(ctx).
		Where("subject_type = ? AND subject_id = ?", string(subjectType), subjectID.String()).
		Find(&models).Error
	if err != nil {
		return nil, err
	}
	out := make([]*rbac.Binding, 0, len(models))
	for i := range models {
		out = append(out, models[i].ToDomain())
	}
	return out, nil
}

func (s *RBACStore) ListPolicyRulesByRoleID(ctx context.Context, roleID shared.ID) ([]rbac.PolicyRule, error) {
	var models []PolicyRuleModel
	err := s.db.WithContext(ctx).
		Where("role_id = ?", roleID.String()).
		Order("id asc").
		Find(&models).Error
	if err != nil {
		return nil, err
	}
	out := make([]rbac.PolicyRule, 0, len(models))
	for i := range models {
		out = append(out, models[i].ToDomain())
	}
	return out, nil
}

func (s *RBACStore) ReplacePolicyRules(ctx context.Context, roleID shared.ID, rules []rbac.PolicyRule) error {
	tx := s.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return tx.Error
	}
	if err := tx.Where("role_id = ?", roleID.String()).Delete(&PolicyRuleModel{}).Error; err != nil {
		_ = tx.Rollback().Error
		return err
	}
	for i := range rules {
		model := PolicyRuleModelFromDomain(roleID, rules[i])
		if err := tx.Create(&model).Error; err != nil {
			_ = tx.Rollback().Error
			return err
		}
	}
	return tx.Commit().Error
}

func defaultNormalize(v string) string {
	return strings.ToLower(strings.TrimSpace(v))
}
