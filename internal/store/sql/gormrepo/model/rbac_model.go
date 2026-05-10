package model

import (
	"strings"
	"time"

	"github.com/tinboxw/skoll/internal/domain/rbac"
	"github.com/tinboxw/skoll/internal/domain/shared"
)

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
