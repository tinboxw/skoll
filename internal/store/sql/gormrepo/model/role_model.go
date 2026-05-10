package model

import (
	"encoding/json"
	"strings"
	"time"

	domainrole "github.com/tinboxw/skoll/internal/domain/role"
	"github.com/tinboxw/skoll/internal/domain/shared"
)

type RoleModel struct {
	ID          uint64 `gorm:"primaryKey;autoIncrement"`
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
		ID:          parseUintID(entity.ID.String()),
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
		ID:          shared.ID(formatUintID(m.ID)),
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
