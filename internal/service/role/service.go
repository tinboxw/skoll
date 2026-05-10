package role

import (
	"context"

	domainrole "github.com/tinboxw/skoll/internal/domain/role"
)

type Service interface {
	Create(ctx context.Context, in CreateRoleInput) (*domainrole.Role, error)
	Get(ctx context.Context, id string) (*domainrole.Role, error)
	List(ctx context.Context, in ListInput) ([]*domainrole.Role, error)
	Update(ctx context.Context, in UpdateRoleInput) (*domainrole.Role, error)
	Delete(ctx context.Context, id string) error
	Grant(ctx context.Context, id, permission string) (*domainrole.Role, error)
	Revoke(ctx context.Context, id, permission string) (*domainrole.Role, error)
}
