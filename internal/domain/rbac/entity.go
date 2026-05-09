package rbac

import (
	"fmt"
	"strings"
	"time"

	"github.com/tinboxw/skoll/internal/domain/shared"
)

type SubjectType string

const (
	SubjectUser SubjectType = "user"
	SubjectRole SubjectType = "role"
)

type Binding struct {
	ID          shared.ID
	SubjectType SubjectType
	SubjectID   shared.ID
	RoleID      shared.ID
	Scope       DataScope
	Meta        shared.AuditMeta
}

func NewBinding(id shared.ID, subjectType SubjectType, subjectID, roleID shared.ID, scope DataScope, now time.Time) (*Binding, error) {
	if id.IsZero() || subjectID.IsZero() || roleID.IsZero() {
		return nil, fmt.Errorf("binding ids are required")
	}
	if subjectType != SubjectUser && subjectType != SubjectRole {
		return nil, fmt.Errorf("unsupported subject type: %s", subjectType)
	}
	if err := scope.Validate(); err != nil {
		return nil, err
	}
	b := &Binding{
		ID:          id,
		SubjectType: subjectType,
		SubjectID:   subjectID,
		RoleID:      roleID,
		Scope:       scope,
	}
	b.Meta.Touch(now)
	return b, nil
}

type Permission struct {
	Resource string
	Action   string
}

func (p Permission) Validate() error {
	if strings.TrimSpace(p.Resource) == "" {
		return fmt.Errorf("permission resource is required")
	}
	if strings.TrimSpace(p.Action) == "" {
		return fmt.Errorf("permission action is required")
	}
	return nil
}

func (p Permission) Key() string {
	return strings.TrimSpace(p.Resource) + ":" + strings.TrimSpace(p.Action)
}
