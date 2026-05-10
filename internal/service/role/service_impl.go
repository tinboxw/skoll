package role

import (
	"context"
	"fmt"
	"strings"
	"time"

	domainrole "github.com/tinboxw/skoll/internal/domain/role"
	"github.com/tinboxw/skoll/internal/domain/shared"
	"github.com/tinboxw/skoll/internal/repository"
)

type serviceImpl struct {
	repo  repository.RoleRepository
	nowFn func() time.Time
	idFn  func(prefix string) shared.ID
}

func NewService(repo repository.RoleRepository) Service {
	return &serviceImpl{
		repo:  repo,
		nowFn: func() time.Time { return time.Now().UTC() },
		idFn: func(prefix string) shared.ID {
			return shared.ID("new")
		},
	}
}

func (s *serviceImpl) Create(ctx context.Context, in CreateRoleInput) (*domainrole.Role, error) {
	now := s.nowFn()
	r, err := domainrole.New(s.idFn("role"), in.Name, in.Key, in.Description, in.Permissions, in.BuiltIn, now)
	if err != nil {
		return nil, err
	}
	if err := s.repo.Save(ctx, r); err != nil {
		return nil, err
	}
	return r, nil
}

func (s *serviceImpl) Get(ctx context.Context, id string) (*domainrole.Role, error) {
	if id == "" {
		return nil, fmt.Errorf("id is required")
	}
	return s.repo.GetByID(ctx, shared.ID(id))
}

func (s *serviceImpl) List(ctx context.Context, in ListInput) ([]*domainrole.Role, error) {
	if in.Offset < 0 || in.Limit < 0 {
		return nil, fmt.Errorf("invalid pagination")
	}
	return s.repo.List(ctx, in.Offset, in.Limit)
}

func (s *serviceImpl) Update(ctx context.Context, in UpdateRoleInput) (*domainrole.Role, error) {
	id := strings.TrimSpace(in.ID)
	if id == "" {
		return nil, fmt.Errorf("id is required")
	}
	entity, err := s.repo.GetByID(ctx, shared.ID(id))
	if err != nil {
		return nil, err
	}
	if entity == nil {
		return nil, fmt.Errorf("role not found")
	}

	if name := strings.TrimSpace(in.Name); name != "" {
		if err := domainrole.ValidateName(name); err != nil {
			return nil, err
		}
		entity.Name = name
	}
	if key := strings.TrimSpace(strings.ToLower(in.Key)); key != "" {
		if err := domainrole.ValidateKey(key); err != nil {
			return nil, err
		}
		entity.Key = key
	}
	if in.Permissions != nil {
		entity.Permissions = domainrole.NormalizePermissions(in.Permissions)
	}
	entity.Description = strings.TrimSpace(in.Description)
	entity.Meta.Touch(s.nowFn())

	if err := s.repo.Save(ctx, entity); err != nil {
		return nil, err
	}
	return entity, nil
}

func (s *serviceImpl) Grant(ctx context.Context, id, permission string) (*domainrole.Role, error) {
	r, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, fmt.Errorf("role not found")
	}
	r.Grant(permission, s.nowFn())
	if err := s.repo.Save(ctx, r); err != nil {
		return nil, err
	}
	return r, nil
}

func (s *serviceImpl) Revoke(ctx context.Context, id, permission string) (*domainrole.Role, error) {
	r, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, fmt.Errorf("role not found")
	}
	r.Revoke(permission, s.nowFn())
	if err := s.repo.Save(ctx, r); err != nil {
		return nil, err
	}
	return r, nil
}

func (s *serviceImpl) Delete(ctx context.Context, id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("id is required")
	}
	entity, err := s.repo.GetByID(ctx, shared.ID(id))
	if err != nil {
		return err
	}
	if entity == nil {
		return fmt.Errorf("role not found")
	}
	if entity.BuiltIn {
		return fmt.Errorf("built-in role cannot be deleted")
	}
	return s.repo.Delete(ctx, shared.ID(id))
}
