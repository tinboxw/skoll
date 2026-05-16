package rbac

import (
	"context"

	"github.com/tinboxw/skoll/internal/domain/rbac"
	"github.com/tinboxw/skoll/internal/domain/shared"
)

type RBACRepository interface {
	CreateBinding(ctx context.Context, binding *rbac.Binding) error
	DeleteBinding(ctx context.Context, id shared.ID) error
	ListBindingsBySubject(ctx context.Context, subjectType rbac.SubjectType, subjectID shared.ID) ([]*rbac.Binding, error)
	ListPolicyRulesByRoleID(ctx context.Context, roleID shared.ID) ([]rbac.PolicyRule, error)
	ReplacePolicyRules(ctx context.Context, roleID shared.ID, rules []rbac.PolicyRule) error
}
