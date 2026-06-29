package user

import (
	"context"
	"strings"

	"github.com/tinboxw/skoll/internal/domain/shared"
	"github.com/tinboxw/skoll/internal/domain/user"
)

type ListFilter struct {
	UserIDs       []shared.ID
	DepartmentIDs []string
}

func (f ListFilter) Empty() bool {
	return len(f.NormalizedUserIDs()) == 0 && len(f.NormalizedDepartmentIDs()) == 0
}

func (f ListFilter) NormalizedUserIDs() []shared.ID {
	seen := make(map[shared.ID]struct{}, len(f.UserIDs))
	out := make([]shared.ID, 0, len(f.UserIDs))
	for _, id := range f.UserIDs {
		trimmed := shared.ID(strings.TrimSpace(id.String()))
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	return out
}

func (f ListFilter) NormalizedDepartmentIDs() []string {
	seen := make(map[string]struct{}, len(f.DepartmentIDs))
	out := make([]string, 0, len(f.DepartmentIDs))
	for _, id := range f.DepartmentIDs {
		trimmed := strings.TrimSpace(id)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	return out
}

type UserRepository interface {
	GetByID(ctx context.Context, id shared.ID) (*user.User, error)
	GetByAccount(ctx context.Context, account string) (*user.User, error)
	GetByEmail(ctx context.Context, email user.Email) (*user.User, error)
	List(ctx context.Context, offset, limit int) ([]*user.User, error)
	ListFiltered(ctx context.Context, filter ListFilter, offset, limit int) ([]*user.User, error)
	Save(ctx context.Context, entity *user.User) error
	Delete(ctx context.Context, id shared.ID) error
}
