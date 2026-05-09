package role

import (
	"fmt"
	"strings"
	"time"

	"github.com/tinboxw/skoll/internal/domain/shared"
)

type Role struct {
	ID          shared.ID
	Name        string
	Key         string
	Description string
	Permissions []string
	BuiltIn     bool
	Meta        shared.AuditMeta
}

func New(id shared.ID, name, key, description string, permissions []string, builtIn bool, now time.Time) (*Role, error) {
	if id.IsZero() {
		return nil, fmt.Errorf("id is required")
	}
	if err := ValidateName(name); err != nil {
		return nil, err
	}
	if err := ValidateKey(key); err != nil {
		return nil, err
	}

	r := &Role{
		ID:          id,
		Name:        strings.TrimSpace(name),
		Key:         strings.TrimSpace(strings.ToLower(key)),
		Description: strings.TrimSpace(description),
		Permissions: NormalizePermissions(permissions),
		BuiltIn:     builtIn,
	}
	r.Meta.Touch(now)
	return r, nil
}

func (r *Role) Grant(permission string, now time.Time) {
	r.Permissions = NormalizePermissions(append(r.Permissions, permission))
	r.Meta.Touch(now)
}

func (r *Role) Revoke(permission string, now time.Time) {
	target := strings.TrimSpace(strings.ToLower(permission))
	if target == "" {
		return
	}
	filtered := make([]string, 0, len(r.Permissions))
	for _, p := range r.Permissions {
		if p == target {
			continue
		}
		filtered = append(filtered, p)
	}
	r.Permissions = filtered
	r.Meta.Touch(now)
}
