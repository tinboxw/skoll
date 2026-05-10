package postgres

import (
	"fmt"
	"strings"

	"github.com/tinboxw/skoll/internal/repository"
	storesql "github.com/tinboxw/skoll/internal/store/sql"
	"github.com/tinboxw/skoll/internal/store/sql/gormrepo"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type Adapter struct {
	dsn  string
	db   *gorm.DB
	user repository.UserRepository
	role repository.RoleRepository
	rbac repository.RBACRepository
	sys  repository.SystemRepository
}

func NewAdapter(dsn string) (*Adapter, error) {
	if storesql.NormalizeDSN(dsn) == "" {
		return nil, fmt.Errorf("postgres dsn is required")
	}
	resolvedDSN := strings.TrimSpace(dsn)
	db, err := gorm.Open(postgres.Open(resolvedDSN), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("open postgres connection: %w", err)
	}
	if err := db.AutoMigrate(gormrepo.AllModels()...); err != nil {
		return nil, fmt.Errorf("auto migrate postgres schema: %w", err)
	}
	return &Adapter{
		dsn:  resolvedDSN,
		db:   db,
		user: gormrepo.NewUserStore(db),
		role: gormrepo.NewRoleStore(db, normalizeRoleKey),
		rbac: gormrepo.NewRBACStore(db),
		sys:  gormrepo.NewSystemStore(db, normalizeSettingKey),
	}, nil
}

func (a *Adapter) DSN() string { return a.dsn }
func (a *Adapter) UserRepository() repository.UserRepository {
	return a.user
}
func (a *Adapter) RoleRepository() repository.RoleRepository {
	return a.role
}

func (a *Adapter) RBACRepository() repository.RBACRepository {
	return a.rbac
}

func (a *Adapter) SystemRepository() repository.SystemRepository {
	return a.sys
}

func (a *Adapter) DB() *gorm.DB {
	return a.db
}
