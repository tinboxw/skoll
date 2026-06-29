package rbac

import (
	"context"

	domainrbac "github.com/tinboxw/skoll/internal/domain/rbac"
)

type Service interface {
	BindRole(ctx context.Context, in BindRoleInput) (*domainrbac.Binding, error)
	UnbindBinding(ctx context.Context, bindingID string) error
	ListBindings(ctx context.Context, subjectType domainrbac.SubjectType, subjectID string) ([]*domainrbac.Binding, error)
	SetRolePolicies(ctx context.Context, in SetRolePoliciesInput) error
	CheckPermission(ctx context.Context, in CheckPermissionInput) (bool, error)
	ResolvePermission(ctx context.Context, in CheckPermissionInput) (PermissionDecision, error)
	ResolveDataScope(ctx context.Context, in ResolveDataScopeInput) (DataScopeDecision, error)
	ListBindingsByUser(ctx context.Context, userID string) ([]*domainrbac.Binding, error)
}
