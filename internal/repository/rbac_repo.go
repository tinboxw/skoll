package repository

import (
	"context"

	"github.com/tinboxw/skoll/internal/domain/rbac"
	"github.com/tinboxw/skoll/internal/domain/shared"
)

type RBACRepository interface {
	GetRoleBinding(ctx context.Context, roleID shared.ID) (rbac.Binding, error)
	SaveRoleBinding(ctx context.Context, binding rbac.Binding) error
	DeleteRoleBinding(ctx context.Context, roleID shared.ID) error
}
