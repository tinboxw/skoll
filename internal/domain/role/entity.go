package role

import (
	"errors"
	"strings"
	"time"

	"github.com/tinboxw/skoll/internal/domain/shared"
)

type Role struct {
	ID          shared.ID
	Name        string
	Description string
	BuiltIn     bool
	Audit       shared.AuditInfo
}

func New(id shared.ID, name, description string, builtIn bool, now time.Time) (Role, error) {
	name = strings.TrimSpace(name)
	if id == "" {
		return Role{}, errors.New("role id is required")
	}
	if err := ValidateName(name); err != nil {
		return Role{}, err
	}
	return Role{
		ID:          id,
		Name:        name,
		Description: strings.TrimSpace(description),
		BuiltIn:     builtIn,
		Audit: shared.AuditInfo{
			CreatedAt: now,
			UpdatedAt: now,
		},
	}, nil
}
