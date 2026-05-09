package repository

import (
	"context"

	"github.com/tinboxw/skoll/internal/domain/role"
	"github.com/tinboxw/skoll/internal/domain/shared"
)

type RoleRepository interface {
	GetByID(ctx context.Context, id shared.ID) (role.Role, error)
	List(ctx context.Context, pager shared.Pager) ([]role.Role, error)
	Save(ctx context.Context, r role.Role) error
	Delete(ctx context.Context, id shared.ID) error
}
