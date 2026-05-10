package postgres

import (
	"fmt"

	"github.com/tinboxw/skoll/internal/repository"
	"github.com/tinboxw/skoll/internal/store/memory"
	storesql "github.com/tinboxw/skoll/internal/store/sql"
)

type Adapter struct {
	dsn  string
	user repository.UserRepository
	role repository.RoleRepository
	sys  repository.SystemRepository
}

func NewAdapter(dsn string) (*Adapter, error) {
	if storesql.NormalizeDSN(dsn) == "" {
		return nil, fmt.Errorf("postgres dsn is required")
	}
	return &Adapter{
		dsn:  dsn,
		user: memory.NewUserStore(),
		role: memory.NewRoleStore(),
		sys:  memory.NewSystemStore(),
	}, nil
}

func (a *Adapter) DSN() string { return a.dsn }
func (a *Adapter) UserRepository() repository.UserRepository {
	return a.user
}
func (a *Adapter) RoleRepository() repository.RoleRepository {
	return a.role
}

func (a *Adapter) SystemRepository() repository.SystemRepository {
	return a.sys
}
