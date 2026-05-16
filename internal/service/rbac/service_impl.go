package rbac

import (
	"context"
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

func (s *serviceImpl) SetRolePolicies(ctx context.Context, in SetRolePoliciesInput) error {
	for _, rule := range in.Rules {
		if err := rule.Validate(); err != nil {
			return err
		}
	}
	return s.repo.ReplacePolicyRules(ctx, shared.ID(in.RoleID), in.Rules)
}

func (s *serviceImpl) CheckPermission(ctx context.Context, in CheckPermissionInput) (bool, error) {
	bindings, err := s.repo.ListBindingsBySubject(ctx, in.SubjectType, shared.ID(in.SubjectID))
	if err != nil {
		return false, err
	}
	if len(bindings) == 0 {
		return false, nil
	}

	allowed := false
	for _, b := range bindings {
		rules, err := s.repo.ListPolicyRulesByRoleID(ctx, b.RoleID)
		if err != nil {
			return false, err
		}
		for _, rule := range rules {
			if rule.Matches(in.Resource, in.Action) {
				if rule.Effect == domainrbac.EffectDeny {
					return false, nil
				}
				if rule.Effect == domainrbac.EffectAllow {
					allowed = true
				}
			}
		}
	}

	return allowed, nil
}
