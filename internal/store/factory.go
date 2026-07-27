package store

import (
	"fmt"
	"strings"

	pluginruntime "github.com/tinboxw/skoll/internal/plugin"
	"github.com/tinboxw/skoll/internal/repository"
	auditrepo "github.com/tinboxw/skoll/internal/repository/audit"
	filerepo "github.com/tinboxw/skoll/internal/repository/file"
	menurepo "github.com/tinboxw/skoll/internal/repository/menu"
	organizationrepo "github.com/tinboxw/skoll/internal/repository/organization"
	permissionrepo "github.com/tinboxw/skoll/internal/repository/permission"
	pluginrepo "github.com/tinboxw/skoll/internal/repository/plugin"
	rbacrepo "github.com/tinboxw/skoll/internal/repository/rbac"
	rolerepo "github.com/tinboxw/skoll/internal/repository/role"
	systemrepo "github.com/tinboxw/skoll/internal/repository/system"
	userrepo "github.com/tinboxw/skoll/internal/repository/user"
	jobsvc "github.com/tinboxw/skoll/internal/service/job"
	notificationsvc "github.com/tinboxw/skoll/internal/service/notification"
	workflowsvc "github.com/tinboxw/skoll/internal/service/workflow"
	"github.com/tinboxw/skoll/internal/store/clickhouse"
	"github.com/tinboxw/skoll/internal/store/memory"
	"github.com/tinboxw/skoll/internal/store/sql"
	"github.com/tinboxw/skoll/internal/store/sql/gormrepo"
	"github.com/tinboxw/skoll/internal/store/sql/mysql"
	"github.com/tinboxw/skoll/internal/store/sql/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
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
	Users             userrepo.UserRepository
	Roles             rolerepo.RoleRepository
	RBAC              rbacrepo.RBACRepository
	Audit             auditrepo.AuditRepository
	System            systemrepo.SystemRepository
	Plugins           pluginrepo.PluginRepository
	PluginMigrations  pluginruntime.MigrationStore
	Permissions       permissionrepo.PermissionRepository
	Menus             menurepo.MenuRepository
	UnitOfWork        repository.UnitOfWork
	AuditEvents       auditrepo.EventRepository
	Files             filerepo.FileRepository
	Organization      organizationrepo.OrganizationRepository
	Workflow          workflowsvc.Repository
	Notifications     notificationsvc.Repository
	Jobs              jobsvc.Repository
	PluginDataDB      *gorm.DB
	PluginDataDialect string
}

func NewBundle(opts Options) (*Bundle, error) {
	switch Mode(strings.ToLower(strings.TrimSpace(string(opts.Mode)))) {
	case ModeMemory:
		pluginDataDB, err := openMemoryPluginDataDB()
		if err != nil {
			return nil, err
		}
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
		organization := memory.NewOrganizationStore()
		return &Bundle{
			Users: users, Roles: roles, RBAC: rbac, Audit: audit, System: system, Plugins: plugins, Permissions: permissions,
			Menus: menus, UnitOfWork: sql.NewUnitOfWorkWithDB(pluginDataDB), AuditEvents: auditEvents, Files: files, Organization: organization,
			Workflow:      workflowsvc.NewMemoryRepository(),
			Notifications: notificationsvc.NewMemoryRepository(),
			Jobs:          jobsvc.NewMemoryRepository(),
			PluginDataDB:  pluginDataDB, PluginDataDialect: "sqlite",
		}, nil
	case ModeMySQL:
		primary, err := mysql.NewAdapter(opts.PrimaryDSN)
		if err != nil {
			return nil, err
		}
		auditRepo := primary.AuditRepository()
		auditEvents, _ := auditRepo.(auditrepo.EventRepository)
		return &Bundle{
			Users:            primary.UserRepository(),
			Roles:            primary.RoleRepository(),
			RBAC:             primary.RBACRepository(),
			Audit:            auditRepo,
			System:           primary.SystemRepository(),
			Plugins:          primary.PluginRepository(),
			PluginMigrations: gormrepo.NewPluginMigrationStore(primary.DB()),
			Permissions:      primary.PermissionRepository(),
			Menus:            primary.MenuRepository(),
			UnitOfWork:       sql.NewUnitOfWorkWithDB(primary.DB()),
			AuditEvents:      auditEvents,
			Files:            primary.FileRepository(),
			Organization:     primary.OrganizationRepository(),
			Workflow:         primary.WorkflowRepository(),
			Notifications:    primary.NotificationRepository(),
			Jobs:             primary.JobRepository(),
			PluginDataDB:     primary.DB(), PluginDataDialect: "mysql",
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
			Users:            primary.UserRepository(),
			Roles:            primary.RoleRepository(),
			RBAC:             primary.RBACRepository(),
			Audit:            audit.AuditRepository(),
			System:           primary.SystemRepository(),
			Plugins:          primary.PluginRepository(),
			PluginMigrations: gormrepo.NewPluginMigrationStore(primary.DB()),
			Permissions:      primary.PermissionRepository(),
			Menus:            primary.MenuRepository(),
			UnitOfWork:       sql.NewUnitOfWorkWithDB(primary.DB()),
			AuditEvents:      gormrepo.NewAuditStore(primary.DB()),
			Files:            primary.FileRepository(),
			Organization:     primary.OrganizationRepository(),
			Workflow:         primary.WorkflowRepository(),
			Notifications:    primary.NotificationRepository(),
			Jobs:             primary.JobRepository(),
			PluginDataDB:     primary.DB(), PluginDataDialect: "postgresql",
		}, nil
	default:
		return nil, fmt.Errorf("unsupported store mode: %q", opts.Mode)
	}
}

func openMemoryPluginDataDB() (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent), TranslateError: true})
	if err != nil {
		return nil, fmt.Errorf("open memory plugin datastore: %w", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("resolve memory plugin datastore: %w", err)
	}
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)
	if err = db.AutoMigrate(
		&gormrepo.PluginDataMutationModel{}, &gormrepo.PluginEventOutboxModel{}, &gormrepo.PluginEventInboxModel{},
		&gormrepo.DocumentNumberSequenceModel{}, &gormrepo.DocumentNumberIssueModel{},
		&gormrepo.DocumentWorkflowBindingModel{}, &gormrepo.DocumentWorkflowActionModel{},
		&gormrepo.DocumentAttachmentModel{}, &gormrepo.DocumentCommentModel{}, &gormrepo.DocumentTimelineEventModel{},
	); err != nil {
		return nil, fmt.Errorf("migrate memory plugin datastore: %w", err)
	}
	return db, nil
}
