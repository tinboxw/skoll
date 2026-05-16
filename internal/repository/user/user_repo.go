package user

import (
	"context"

	"github.com/tinboxw/skoll/internal/domain/shared"
	"github.com/tinboxw/skoll/internal/domain/user"
)

type UserRepository interface {
	GetByID(ctx context.Context, id shared.ID) (*user.User, error)
	GetByAccount(ctx context.Context, account string) (*user.User, error)
	GetByEmail(ctx context.Context, email user.Email) (*user.User, error)
	List(ctx context.Context, offset, limit int) ([]*user.User, error)
	Save(ctx context.Context, entity *user.User) error
	Delete(ctx context.Context, id shared.ID) error
}
