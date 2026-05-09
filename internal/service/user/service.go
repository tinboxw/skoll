package user

import (
	"context"

	domainuser "github.com/tinboxw/skoll/internal/domain/user"
)

type Service interface {
	Create(ctx context.Context, in CreateUserInput) (*domainuser.User, error)
	Get(ctx context.Context, id string) (*domainuser.User, error)
	List(ctx context.Context, in ListInput) ([]*domainuser.User, error)
	UpdateEmail(ctx context.Context, in UpdateEmailInput) (*domainuser.User, error)
	Disable(ctx context.Context, id, actorID string) error
}
