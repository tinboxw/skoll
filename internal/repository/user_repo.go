package repository

import (
	"context"

	"github.com/tinboxw/skoll/internal/domain/shared"
	"github.com/tinboxw/skoll/internal/domain/user"
)

type UserRepository interface {
	GetByID(ctx context.Context, id shared.ID) (user.User, error)
	GetByUsername(ctx context.Context, username string) (user.User, error)
	Save(ctx context.Context, u user.User) error
	Delete(ctx context.Context, id shared.ID) error
}
