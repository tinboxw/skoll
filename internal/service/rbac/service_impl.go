package rbac

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	domainrbac "github.com/tinboxw/skoll/internal/domain/rbac"
	"github.com/tinboxw/skoll/internal/domain/shared"
	organizationrepo "github.com/tinboxw/skoll/internal/repository/organization"
	rbacrepo "github.com/tinboxw/skoll/internal/repository/rbac"
	"github.com/tinboxw/skoll/pkg/security"
)

var ErrDataScopeDenied = errors.New("data scope denied")
var rbacIDSequence atomic.Uint64

type serviceImpl struct {
	repo             rbacrepo.RBACRepository
	organizationRepo organizationrepo.OrganizationRepository
	nowFn            func() time.Time
	idFn             func(prefix string) shared.ID
}

func NewService(repo rbacrepo.RBACRepository) Service {
	return NewServiceWithOrganization(repo, nil)
}

func NewServiceWithOrganization(repo rbacrepo.RBACRepository, organizationRepo organizationrepo.OrganizationRepository) Service {
	return &serviceImpl{
		repo:             repo,
		organizationRepo: organizationRepo,
		nowFn:            func() time.Time { return time.Now().UTC() },
		idFn: func(prefix string) shared.ID {
			return shared.ID(prefix + "-" + strconv.FormatInt(time.Now().UTC().UnixNano(), 10) + "-" + strconv.FormatUint(rbacIDSequence.Add(1), 10))
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

func (s *serviceImpl) ResolveDataScope(ctx context.Context, in ResolveDataScopeInput) (DataScopeDecision, error) {
	claims, ok := security.JWTClaimsFromContext(ctx)
	if !ok || strings.TrimSpace(claims.Subject) == "" {
		return DataScopeDecision{}, fmt.Errorf("%w: trusted identity is required", ErrDataScopeDenied)
	}
	if claims.HasRole("super_admin") {
		return DataScopeDecision{Scope: domainrbac.DataScopeAll, All: true}, nil
	}

	resource := strings.TrimSpace(in.Resource)
	action := strings.TrimSpace(in.Action)
	if resource == "" || action == "" {
		return DataScopeDecision{}, fmt.Errorf("%w: resource and action are required", ErrDataScopeDenied)
	}
	permission, err := s.ResolvePermission(ctx, CheckPermissionInput{
		SubjectType: domainrbac.SubjectUser,
		SubjectID:   claims.Subject,
		Resource:    resource,
		Action:      action,
	})
	if err != nil {
		return DataScopeDecision{}, err
	}
	if !permission.Allowed {
		return DataScopeDecision{}, fmt.Errorf("%w: permission is not granted", ErrDataScopeDenied)
	}

	scope := domainrbac.NormalizeDataScope(permission.Scope)
	decision := DataScopeDecision{Scope: scope}
	switch scope {
	case domainrbac.DataScopeAll:
		decision.All = true
	case domainrbac.DataScopeSelf:
		decision.UserIDs = []string{strings.TrimSpace(claims.Subject)}
	case domainrbac.DataScopeDepartment:
		organizationID, err := s.requireTrustedOrganization(ctx, claims.OrganizationID)
		if err != nil {
			return DataScopeDecision{}, err
		}
		decision.DepartmentIDs = []string{organizationID}
	case domainrbac.DataScopeDepartmentTree:
		organizationID, err := s.requireTrustedOrganization(ctx, claims.OrganizationID)
		if err != nil {
			return DataScopeDecision{}, err
		}
		decision.DepartmentIDs, err = s.organizationTreeIDs(ctx, organizationID)
		if err != nil {
			return DataScopeDecision{}, err
		}
	default:
		return DataScopeDecision{}, fmt.Errorf("%w: unsupported granted scope %q", ErrDataScopeDenied, scope)
	}
	return decision, nil
}

func (s *serviceImpl) requireTrustedOrganization(ctx context.Context, raw string) (string, error) {
	organizationID := strings.TrimSpace(raw)
	if organizationID == "" || s.organizationRepo == nil {
		return "", fmt.Errorf("%w: trusted organization is unavailable", ErrDataScopeDenied)
	}
	item, err := s.organizationRepo.GetDepartmentByID(ctx, shared.ID(organizationID))
	if err != nil {
		return "", err
	}
	if item == nil {
		return "", fmt.Errorf("%w: trusted organization no longer exists", ErrDataScopeDenied)
	}
	return organizationID, nil
}

func (s *serviceImpl) organizationTreeIDs(ctx context.Context, rootID string) ([]string, error) {
	items, err := s.organizationRepo.ListDepartments(ctx, 0, 0)
	if err != nil {
		return nil, err
	}
	included := map[string]struct{}{rootID: {}}
	for changed := true; changed; {
		changed = false
		for _, item := range items {
			id := strings.TrimSpace(item.ID.String())
			parentID := strings.TrimSpace(item.ParentID.String())
			if id == "" {
				continue
			}
			if _, parentIncluded := included[parentID]; !parentIncluded {
				continue
			}
			if _, exists := included[id]; exists {
				continue
			}
			included[id] = struct{}{}
			changed = true
		}
	}
	descendants := make([]string, 0, len(included)-1)
	for id := range included {
		if id != rootID {
			descendants = append(descendants, id)
		}
	}
	sort.Strings(descendants)
	return append([]string{rootID}, descendants...), nil
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
