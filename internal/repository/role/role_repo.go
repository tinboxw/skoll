package role

import (
	"context"

	"github.com/tinboxw/skoll/internal/domain/role"
	"github.com/tinboxw/skoll/internal/domain/shared"
)

type RoleRepository interface {
	GetByID(ctx context.Context, id shared.ID) (*role.Role, error)
	GetByKey(ctx context.Context, key string) (*role.Role, error)
	List(ctx context.Context, offset, limit int) ([]*role.Role, error)
	Save(ctx context.Context, entity *role.Role) error
	Delete(ctx context.Context, id shared.ID) error
}
