package gormrepo

import (
	"strings"
	"time"

	"github.com/tinboxw/skoll/internal/domain/rbac"
	"github.com/tinboxw/skoll/internal/domain/shared"
)

type BindingModel struct {
	ID          uint64 `gorm:"primaryKey;autoIncrement"`
	SubjectType string `gorm:"size:32;index:idx_subject"`
	SubjectID   uint64 `gorm:"index:idx_subject"`
	RoleID      uint64 `gorm:"index"`
	Scope       string `gorm:"size:32"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (BindingModel) TableName() string { return "sk_rbac_bindings" }

func BindingModelFromDomain(entity *rbac.Binding) BindingModel {
	return BindingModel{
		ID:          parseUintID(entity.ID.String()),
		SubjectType: string(entity.SubjectType),
		SubjectID:   parseUintID(entity.SubjectID.String()),
		RoleID:      parseUintID(entity.RoleID.String()),
		Scope:       string(rbac.NormalizeDataScope(entity.Scope)),
		CreatedAt:   entity.Meta.CreatedAt,
		UpdatedAt:   entity.Meta.UpdatedAt,
	}
}

func (m BindingModel) ToDomain() *rbac.Binding {
	return &rbac.Binding{
		ID:          shared.ID(formatUintID(m.ID)),
		SubjectType: rbac.SubjectType(m.SubjectType),
		SubjectID:   shared.ID(formatUintID(m.SubjectID)),
		RoleID:      shared.ID(formatUintID(m.RoleID)),
		Scope:       rbac.NormalizeDataScope(rbac.DataScope(m.Scope)),
		Meta: shared.AuditMeta{
			CreatedAt: m.CreatedAt,
			UpdatedAt: m.UpdatedAt,
		},
	}
}

type PolicyRuleModel struct {
	ID       uint64 `gorm:"primaryKey;autoIncrement"`
	RoleID   uint64 `gorm:"index"`
	Resource string `gorm:"size:256"`
	Action   string `gorm:"size:128"`
	Effect   string `gorm:"size:16"`
	Scope    string `gorm:"size:32"`
}

func (PolicyRuleModel) TableName() string { return "sk_rbac_policy_rules" }

func PolicyRuleModelFromDomain(roleID shared.ID, rule rbac.PolicyRule) PolicyRuleModel {
	return PolicyRuleModel{
		RoleID:   parseUintID(roleID.String()),
		Resource: strings.TrimSpace(rule.Resource),
		Action:   strings.TrimSpace(rule.Action),
		Effect:   string(rule.Effect),
		Scope:    string(rbac.NormalizeDataScope(rule.Scope)),
	}
}

func (m PolicyRuleModel) ToDomain() rbac.PolicyRule {
	return rbac.PolicyRule{
		Resource: m.Resource,
		Action:   m.Action,
		Effect:   rbac.Effect(m.Effect),
		Scope:    rbac.NormalizeDataScope(rbac.DataScope(m.Scope)),
	}
}
