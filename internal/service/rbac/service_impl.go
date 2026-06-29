package rbac

import (
	"context"
	"fmt"
	"strings"
	"time"

	domainrbac "github.com/tinboxw/skoll/internal/domain/rbac"
	"github.com/tinboxw/skoll/internal/domain/shared"
	rbacrepo "github.com/tinboxw/skoll/internal/repository/rbac"
)

type serviceImpl struct {
	repo  rbacrepo.RBACRepository
	nowFn func() time.Time
	idFn  func(prefix string) shared.ID
}

func NewService(repo rbacrepo.RBACRepository) Service {
	return &serviceImpl{
		repo:  repo,
		nowFn: func() time.Time { return time.Now().UTC() },
		idFn: func(prefix string) shared.ID {
			return shared.ID("new")
		},
	}
}

func (s *serviceImpl) BindRole(ctx context.Context, in BindRoleInput) (*domainrbac.Binding, error) {
	b, err := domainrbac.NewBinding(
		s.idFn("bind"),
		in.SubjectType,
		shared.ID(in.SubjectID),
		shared.ID(in.RoleID),
		in.Scope,
		s.nowFn(),
	)
	if err != nil {
		return nil, err
	}
	if err := s.repo.CreateBinding(ctx, b); err != nil {
		return nil, err
	}
	return b, nil
}

func (s *serviceImpl) UnbindBinding(ctx context.Context, bindingID string) error {
	target := strings.TrimSpace(bindingID)
	if target == "" {
		return nil
	}
	return s.repo.DeleteBinding(ctx, shared.ID(target))
}

func (s *serviceImpl) ListBindings(ctx context.Context, subjectType domainrbac.SubjectType, subjectID string) ([]*domainrbac.Binding, error) {
	targetID := strings.TrimSpace(subjectID)
	if targetID == "" {
		return []*domainrbac.Binding{}, nil
	}
	if subjectType != domainrbac.SubjectUser && subjectType != domainrbac.SubjectRole {
		subjectType = domainrbac.SubjectUser
	}
	return s.repo.ListBindingsBySubject(ctx, subjectType, shared.ID(targetID))
}

func (s *serviceImpl) SetRolePolicies(ctx context.Context, in SetRolePoliciesInput) error {
	rules := make([]domainrbac.PolicyRule, 0, len(in.Rules))
	for _, rule := range in.Rules {
		if err := rule.Validate(); err != nil {
			return err
		}
		rules = append(rules, rule.Normalized())
	}
	return s.repo.ReplacePolicyRules(ctx, shared.ID(in.RoleID), rules)
}

func (s *serviceImpl) CheckPermission(ctx context.Context, in CheckPermissionInput) (bool, error) {
	decision, err := s.ResolvePermission(ctx, in)
	if err != nil {
		return false, err
	}
	return decision.Allowed, nil
}

func (s *serviceImpl) ResolvePermission(ctx context.Context, in CheckPermissionInput) (PermissionDecision, error) {
	bindings, err := s.repo.ListBindingsBySubject(ctx, in.SubjectType, shared.ID(in.SubjectID))
	if err != nil {
		return PermissionDecision{}, err
	}
	if len(bindings) == 0 {
		return PermissionDecision{}, nil
	}

	allowed := false
	resolvedScope := domainrbac.DataScopeAll
	for _, b := range bindings {
		rules, err := s.repo.ListPolicyRulesByRoleID(ctx, b.RoleID)
		if err != nil {
			return PermissionDecision{}, err
		}
		for _, rule := range rules {
			if rule.Matches(in.Resource, in.Action) {
				if rule.Effect == domainrbac.EffectDeny {
					return PermissionDecision{}, nil
				}
				if rule.Effect == domainrbac.EffectAllow {
					allowed = true
					resolvedScope = moreRestrictiveScope(resolvedScope, moreRestrictiveScope(b.Scope, rule.Scope))
				}
			}
		}
	}

	if !allowed {
		return PermissionDecision{}, nil
	}
	return PermissionDecision{Allowed: true, Scope: resolvedScope}, nil
}

func (s *serviceImpl) ListBindingsByUser(ctx context.Context, userID string) ([]*domainrbac.Binding, error) {
	target := strings.TrimSpace(userID)
	if target == "" {
		return []*domainrbac.Binding{}, nil
	}
	return s.ListBindings(ctx, domainrbac.SubjectUser, target)
}

func (s *serviceImpl) ResolveDataScope(_ context.Context, in ResolveDataScopeInput) (DataScopeDecision, error) {
	scope := domainrbac.NormalizeDataScope(in.Scope)
	if err := scope.Validate(); err != nil {
		return DataScopeDecision{}, err
	}
	decision := DataScopeDecision{Scope: scope}
	switch scope {
	case domainrbac.DataScopeAll:
		decision.All = true
	case domainrbac.DataScopeSelf:
		userID := strings.TrimSpace(in.ActorUserID)
		if userID == "" {
			return DataScopeDecision{}, fmt.Errorf("actor user id is required for self data scope")
		}
		decision.UserIDs = []string{userID}
	case domainrbac.DataScopeDepartment:
		departmentID := strings.TrimSpace(in.ActorDepartmentID)
		if departmentID == "" {
			return DataScopeDecision{}, fmt.Errorf("actor department id is required for department data scope")
		}
		decision.DepartmentIDs = []string{departmentID}
	case domainrbac.DataScopeDepartmentTree:
		departmentIDs := compactUniqueStrings(append([]string{in.ActorDepartmentID}, in.DepartmentTreeIDs...))
		if len(departmentIDs) == 0 {
			return DataScopeDecision{}, fmt.Errorf("department tree ids are required for department_tree data scope")
		}
		decision.DepartmentIDs = departmentIDs
	case domainrbac.DataScopeCustom:
		departmentIDs := compactUniqueStrings(in.CustomDepartmentIDs)
		if len(departmentIDs) == 0 {
			return DataScopeDecision{}, fmt.Errorf("custom department ids are required for custom data scope")
		}
		decision.DepartmentIDs = departmentIDs
	}
	return decision, nil
}

func moreRestrictiveScope(left, right domainrbac.DataScope) domainrbac.DataScope {
	left = domainrbac.NormalizeDataScope(left)
	right = domainrbac.NormalizeDataScope(right)
	if scopeRank(left) <= scopeRank(right) {
		return left
	}
	return right
}

func scopeRank(scope domainrbac.DataScope) int {
	switch domainrbac.NormalizeDataScope(scope) {
	case domainrbac.DataScopeSelf:
		return 0
	case domainrbac.DataScopeDepartment:
		return 1
	case domainrbac.DataScopeDepartmentTree:
		return 2
	case domainrbac.DataScopeCustom:
		return 3
	case domainrbac.DataScopeAll:
		return 4
	default:
		return 4
	}
}

func compactUniqueStrings(items []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(items))
	for _, item := range items {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	return out
}
