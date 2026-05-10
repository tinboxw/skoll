package gormrepo

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/tinboxw/skoll/internal/domain/rbac"
	domainrole "github.com/tinboxw/skoll/internal/domain/role"
	"github.com/tinboxw/skoll/internal/domain/shared"
	domainsystem "github.com/tinboxw/skoll/internal/domain/system"
	domainuser "github.com/tinboxw/skoll/internal/domain/user"
)

type Normalizer func(string) string

type UserModel struct {
	ID          string `gorm:"primaryKey;size:128"`
	Username    string `gorm:"size:128;index"`
	DisplayName string `gorm:"size:128"`
	Email       string `gorm:"size:191;uniqueIndex"`
	Status      string `gorm:"size:32"`
	Password    string `gorm:"size:256"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (UserModel) TableName() string { return "sk_users" }

func UserModelFromDomain(entity *domainuser.User) UserModel {
	return UserModel{
		ID:          entity.ID.String(),
		Username:    entity.Username,
		DisplayName: entity.DisplayName,
		Email:       entity.Email.String(),
		Status:      string(entity.Status),
		Password:    entity.Password.String(),
		CreatedAt:   entity.Meta.CreatedAt,
		UpdatedAt:   entity.Meta.UpdatedAt,
	}
}

func (m UserModel) ToDomain() *domainuser.User {
	return &domainuser.User{
		ID:          shared.ID(m.ID),
		Username:    m.Username,
		DisplayName: m.DisplayName,
		Email:       domainuser.Email(m.Email),
		Status:      domainuser.Status(m.Status),
		Password:    domainuser.PasswordHash(m.Password),
		Meta: shared.AuditMeta{
			CreatedAt: m.CreatedAt,
			UpdatedAt: m.UpdatedAt,
		},
	}
}

type RoleModel struct {
	ID          string `gorm:"primaryKey;size:128"`
	Name        string `gorm:"size:128"`
	Key         string `gorm:"size:128;uniqueIndex"`
	Description string `gorm:"size:512"`
	Permissions string `gorm:"type:text"`
	BuiltIn     bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (RoleModel) TableName() string { return "sk_roles" }

func RoleModelFromDomain(entity *domainrole.Role, normalizeKey Normalizer) RoleModel {
	payload, _ := json.Marshal(entity.Permissions)
	return RoleModel{
		ID:          entity.ID.String(),
		Name:        entity.Name,
		Key:         normalizeKey(entity.Key),
		Description: entity.Description,
		Permissions: string(payload),
		BuiltIn:     entity.BuiltIn,
		CreatedAt:   entity.Meta.CreatedAt,
		UpdatedAt:   entity.Meta.UpdatedAt,
	}
}

func (m RoleModel) ToDomain(normalizeKey Normalizer) *domainrole.Role {
	perms := make([]string, 0)
	if strings.TrimSpace(m.Permissions) != "" {
		_ = json.Unmarshal([]byte(m.Permissions), &perms)
	}
	return &domainrole.Role{
		ID:          shared.ID(m.ID),
		Name:        m.Name,
		Key:         normalizeKey(m.Key),
		Description: m.Description,
		Permissions: domainrole.NormalizePermissions(perms),
		BuiltIn:     m.BuiltIn,
		Meta: shared.AuditMeta{
			CreatedAt: m.CreatedAt,
			UpdatedAt: m.UpdatedAt,
		},
	}
}

type BindingModel struct {
	ID          string `gorm:"primaryKey;size:128"`
	SubjectType string `gorm:"size:32;index:idx_subject"`
	SubjectID   string `gorm:"size:128;index:idx_subject"`
	RoleID      string `gorm:"size:128;index"`
	Scope       string `gorm:"size:32"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (BindingModel) TableName() string { return "sk_rbac_bindings" }

func BindingModelFromDomain(entity *rbac.Binding) BindingModel {
	return BindingModel{
		ID:          entity.ID.String(),
		SubjectType: string(entity.SubjectType),
		SubjectID:   entity.SubjectID.String(),
		RoleID:      entity.RoleID.String(),
		Scope:       string(entity.Scope),
		CreatedAt:   entity.Meta.CreatedAt,
		UpdatedAt:   entity.Meta.UpdatedAt,
	}
}

func (m BindingModel) ToDomain() *rbac.Binding {
	return &rbac.Binding{
		ID:          shared.ID(m.ID),
		SubjectType: rbac.SubjectType(m.SubjectType),
		SubjectID:   shared.ID(m.SubjectID),
		RoleID:      shared.ID(m.RoleID),
		Scope:       rbac.DataScope(m.Scope),
		Meta: shared.AuditMeta{
			CreatedAt: m.CreatedAt,
			UpdatedAt: m.UpdatedAt,
		},
	}
}

type PolicyRuleModel struct {
	ID       uint64 `gorm:"primaryKey;autoIncrement"`
	RoleID   string `gorm:"size:128;index"`
	Resource string `gorm:"size:256"`
	Action   string `gorm:"size:128"`
	Effect   string `gorm:"size:16"`
	Scope    string `gorm:"size:32"`
}

func (PolicyRuleModel) TableName() string { return "sk_rbac_policy_rules" }

func PolicyRuleModelFromDomain(roleID shared.ID, rule rbac.PolicyRule) PolicyRuleModel {
	return PolicyRuleModel{
		RoleID:   roleID.String(),
		Resource: strings.TrimSpace(rule.Resource),
		Action:   strings.TrimSpace(rule.Action),
		Effect:   string(rule.Effect),
		Scope:    string(rule.Scope),
	}
}

func (m PolicyRuleModel) ToDomain() rbac.PolicyRule {
	return rbac.PolicyRule{
		Resource: m.Resource,
		Action:   m.Action,
		Effect:   rbac.Effect(m.Effect),
		Scope:    rbac.DataScope(m.Scope),
	}
}

type SystemSettingModel struct {
	ID        string `gorm:"primaryKey;size:128"`
	Key       string `gorm:"size:128;uniqueIndex"`
	Value     string `gorm:"type:text"`
	Encrypted bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (SystemSettingModel) TableName() string { return "sk_system_settings" }

func SystemSettingModelFromDomain(setting *domainsystem.Setting, normalizeKey Normalizer) SystemSettingModel {
	return SystemSettingModel{
		ID:        setting.ID.String(),
		Key:       normalizeKey(setting.Key),
		Value:     setting.Value,
		Encrypted: setting.Encrypted,
		CreatedAt: setting.Meta.CreatedAt,
		UpdatedAt: setting.Meta.UpdatedAt,
	}
}

func (m SystemSettingModel) ToDomain(normalizeKey Normalizer) *domainsystem.Setting {
	return &domainsystem.Setting{
		ID:        shared.ID(m.ID),
		Key:       normalizeKey(m.Key),
		Value:     m.Value,
		Encrypted: m.Encrypted,
		Meta: shared.AuditMeta{
			CreatedAt: m.CreatedAt,
			UpdatedAt: m.UpdatedAt,
		},
	}
}

func AllModels() []any {
	return []any{
		&UserModel{},
		&RoleModel{},
		&BindingModel{},
		&PolicyRuleModel{},
		&SystemSettingModel{},
	}
}
