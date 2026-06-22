package store

import (
	"fmt"
	"strings"

	"github.com/tinboxw/skoll/internal/repository"
	auditrepo "github.com/tinboxw/skoll/internal/repository/audit"
	filerepo "github.com/tinboxw/skoll/internal/repository/file"
	menurepo "github.com/tinboxw/skoll/internal/repository/menu"
	permissionrepo "github.com/tinboxw/skoll/internal/repository/permission"
	pluginrepo "github.com/tinboxw/skoll/internal/repository/plugin"
	rbacrepo "github.com/tinboxw/skoll/internal/repository/rbac"
	rolerepo "github.com/tinboxw/skoll/internal/repository/role"
	systemrepo "github.com/tinboxw/skoll/internal/repository/system"
	userrepo "github.com/tinboxw/skoll/internal/repository/user"
	"github.com/tinboxw/skoll/internal/store/clickhouse"
	"github.com/tinboxw/skoll/internal/store/memory"
	"github.com/tinboxw/skoll/internal/store/sql"
	"github.com/tinboxw/skoll/internal/store/sql/gormrepo"
	"github.com/tinboxw/skoll/internal/store/sql/mysql"
	"github.com/tinboxw/skoll/internal/store/sql/postgres"
)

type Mode string

const (
	ModeMemory   Mode = "memory"
	ModeMySQL    Mode = "mysql"
	ModePostgres Mode = "postgres"
)

type Options struct {
	Mode          Mode
	PrimaryDSN    string
	ClickHouseDSN string
}

type Bundle struct {
	Users       userrepo.UserRepository
	Roles       rolerepo.RoleRepository
	RBAC        rbacrepo.RBACRepository
	Audit       auditrepo.AuditRepository
	System      systemrepo.SystemRepository
	Plugins     pluginrepo.PluginRepository
	Permissions permissionrepo.PermissionRepository
	Menus       menurepo.MenuRepository
	UnitOfWork  repository.UnitOfWork
	AuditEvents auditrepo.EventRepository
	Files       filerepo.FileRepository
}

func NewBundle(opts Options) (*Bundle, error) {
	switch Mode(strings.ToLower(strings.TrimSpace(string(opts.Mode)))) {
	case ModeMemory:
		users := memory.NewUserStore()
		roles := memory.NewRoleStore()
		rbac := memory.NewRBACStore()
		audit := clickhouse.NewAuditStore()
		auditEvents := memory.NewAuditEventStore()
		system := memory.NewSystemStore()
		plugins := memory.NewPluginStore()
		permissions := memory.NewPermissionStore()
		menus := memory.NewMenuStore()
		files := memory.NewFileStore()
		return &Bundle{Users: users, Roles: roles, RBAC: rbac, Audit: audit, System: system, Plugins: plugins, Permissions: permissions, Menus: menus, UnitOfWork: sql.NewUnitOfWork(), AuditEvents: auditEvents, Files: files}, nil
	case ModeMySQL:
		primary, err := mysql.NewAdapter(opts.PrimaryDSN)
		if err != nil {
			return nil, err
		}
		auditRepo := primary.AuditRepository()
		auditEvents, _ := auditRepo.(auditrepo.EventRepository)
		return &Bundle{
			Users:       primary.UserRepository(),
			Roles:       primary.RoleRepository(),
			RBAC:        primary.RBACRepository(),
			Audit:       auditRepo,
			System:      primary.SystemRepository(),
			Plugins:     primary.PluginRepository(),
			Permissions: primary.PermissionRepository(),
			Menus:       primary.MenuRepository(),
			UnitOfWork:  sql.NewUnitOfWorkWithDB(primary.DB()),
			AuditEvents: auditEvents,
			Files:       primary.FileRepository(),
		}, nil
	case ModePostgres:
		primary, err := postgres.NewAdapter(opts.PrimaryDSN)
		if err != nil {
			return nil, err
		}
		audit, err := clickhouse.NewAdapter(opts.ClickHouseDSN)
		if err != nil {
			return nil, err
		}
		return &Bundle{
			Users:       primary.UserRepository(),
			Roles:       primary.RoleRepository(),
			RBAC:        primary.RBACRepository(),
			Audit:       audit.AuditRepository(),
			System:      primary.SystemRepository(),
			Plugins:     primary.PluginRepository(),
			Permissions: primary.PermissionRepository(),
			Menus:       primary.MenuRepository(),
			UnitOfWork:  sql.NewUnitOfWorkWithDB(primary.DB()),
			AuditEvents: gormrepo.NewAuditStore(primary.DB()),
			Files:       primary.FileRepository(),
		}, nil
	default:
		return nil, fmt.Errorf("unsupported store mode: %q", opts.Mode)
	}
}
