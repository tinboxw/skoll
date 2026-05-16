package postgres

import (
	"fmt"
	"strings"

	pluginrepo "github.com/tinboxw/skoll/internal/repository/plugin"
	rbacrepo "github.com/tinboxw/skoll/internal/repository/rbac"
	rolerepo "github.com/tinboxw/skoll/internal/repository/role"
	systemrepo "github.com/tinboxw/skoll/internal/repository/system"
	userrepo "github.com/tinboxw/skoll/internal/repository/user"
	storesql "github.com/tinboxw/skoll/internal/store/sql"
	"github.com/tinboxw/skoll/internal/store/sql/gormrepo"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type Adapter struct {
	dsn  string
	db   *gorm.DB
	user userrepo.UserRepository
	role rolerepo.RoleRepository
	rbac rbacrepo.RBACRepository
	sys  systemrepo.SystemRepository
	plg  pluginrepo.PluginRepository
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
		plg:  gormrepo.NewPluginStore(db),
	}, nil
}

func (a *Adapter) DSN() string { return a.dsn }
func (a *Adapter) UserRepository() userrepo.UserRepository {
	return a.user
}
func (a *Adapter) RoleRepository() rolerepo.RoleRepository {
	return a.role
}

func (a *Adapter) RBACRepository() rbacrepo.RBACRepository {
	return a.rbac
}

func (a *Adapter) SystemRepository() systemrepo.SystemRepository {
	return a.sys
}

func (a *Adapter) PluginRepository() pluginrepo.PluginRepository {
	return a.plg
}

func (a *Adapter) DB() *gorm.DB {
	return a.db
}
