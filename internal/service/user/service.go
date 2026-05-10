package user

import (
	"context"

	domainuser "github.com/tinboxw/skoll/internal/domain/user"
)

type Service interface {
	Create(ctx context.Context, in CreateUserInput) (*domainuser.User, error)
	CreateBatch(ctx context.Context, in BatchCreateInput) ([]BatchCreateResult, error)
	Get(ctx context.Context, id string) (*domainuser.User, error)
	List(ctx context.Context, in ListInput) ([]*domainuser.User, error)
	Update(ctx context.Context, in UpdateUserInput) (*domainuser.User, error)
	UpdateEmail(ctx context.Context, in UpdateEmailInput) (*domainuser.User, error)
	Delete(ctx context.Context, id string) error
	Disable(ctx context.Context, id, actorID string) error
}
