package rbac

import (
	"context"

	domainrbac "github.com/tinboxw/skoll/internal/domain/rbac"
)

type Service interface {
	BindRole(ctx context.Context, in BindRoleInput) (*domainrbac.Binding, error)
	SetRolePolicies(ctx context.Context, in SetRolePoliciesInput) error
	CheckPermission(ctx context.Context, in CheckPermissionInput) (bool, error)
}
