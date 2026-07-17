package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"strings"
	"time"

	auditrepo "github.com/tinboxw/skoll/internal/repository/audit"
	filerepo "github.com/tinboxw/skoll/internal/repository/file"
	menurepo "github.com/tinboxw/skoll/internal/repository/menu"
	organizationrepo "github.com/tinboxw/skoll/internal/repository/organization"
	permissionrepo "github.com/tinboxw/skoll/internal/repository/permission"
	pharmaoarepo "github.com/tinboxw/skoll/internal/repository/pharmaoa"
	pluginrepo "github.com/tinboxw/skoll/internal/repository/plugin"
	rbacrepo "github.com/tinboxw/skoll/internal/repository/rbac"
	rolerepo "github.com/tinboxw/skoll/internal/repository/role"
	systemrepo "github.com/tinboxw/skoll/internal/repository/system"
	userrepo "github.com/tinboxw/skoll/internal/repository/user"
	storesql "github.com/tinboxw/skoll/internal/store/sql"
	"github.com/tinboxw/skoll/internal/store/sql/gormrepo"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type Adapter struct {
	dsn       string
	db        *gorm.DB
	aud       auditrepo.AuditRepository
	user      userrepo.UserRepository
	role      rolerepo.RoleRepository
	rbac      rbacrepo.RBACRepository
	sys       systemrepo.SystemRepository
	plg       pluginrepo.PluginRepository
	perm      permissionrepo.PermissionRepository
	menu      menurepo.MenuRepository
	file      filerepo.FileRepository
	org       organizationrepo.OrganizationRepository
	employee  pharmaoarepo.EmployeeRepository
	product   pharmaoarepo.ProductRepository
	supplier  pharmaoarepo.SupplierRepository
	customer  pharmaoarepo.CustomerRepository
	warehouse pharmaoarepo.WarehouseRepository
}

func NewAdapter(dsn string) (*Adapter, error) {
	if storesql.NormalizeDSN(dsn) == "" {
		return nil, fmt.Errorf("mysql dsn is required")
	}

	resolvedDSN, err := resolveMySQLDSN(dsn)
	if err != nil {
		return nil, err
	}

	db, err := gorm.Open(mysql.Open(resolvedDSN), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("open mysql connection: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("resolve mysql sql db: %w", err)
	}
	tuneMySQLPool(sqlDB)
	if err := waitMySQLReady(sqlDB); err != nil {
		return nil, fmt.Errorf("mysql ping failed: %w", err)
	}

	db = db.Set("gorm:table_options", "ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci")

	if err := db.AutoMigrate(gormrepo.AllModels()...); err != nil {
		return nil, fmt.Errorf("auto migrate mysql schema: %w", err)
	}

	return &Adapter{
		dsn:       resolvedDSN,
		db:        db,
		aud:       gormrepo.NewAuditStore(db),
		user:      gormrepo.NewUserStore(db),
		role:      gormrepo.NewRoleStore(db, normalizeRoleKey),
		rbac:      gormrepo.NewRBACStore(db),
		sys:       gormrepo.NewSystemStore(db, normalizeSettingKey),
		plg:       gormrepo.NewPluginStore(db),
		perm:      gormrepo.NewPermissionStore(db),
		menu:      gormrepo.NewMenuStore(db),
		file:      gormrepo.NewFileStore(db),
		org:       gormrepo.NewOrganizationStore(db),
		employee:  gormrepo.NewPharmaEmployeeStore(db),
		product:   gormrepo.NewPharmaProductStore(db),
		supplier:  gormrepo.NewPharmaSupplierStore(db),
		customer:  gormrepo.NewPharmaCustomerStore(db),
		warehouse: gormrepo.NewPharmaWarehouseStore(db),
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

func (a *Adapter) AuditRepository() auditrepo.AuditRepository {
	return a.aud
}

func (a *Adapter) SystemRepository() systemrepo.SystemRepository {
	return a.sys
}

func (a *Adapter) PluginRepository() pluginrepo.PluginRepository {
	return a.plg
}

func (a *Adapter) PermissionRepository() permissionrepo.PermissionRepository {
	return a.perm
}

func (a *Adapter) MenuRepository() menurepo.MenuRepository {
	return a.menu
}

func (a *Adapter) FileRepository() filerepo.FileRepository {
	return a.file
}

func (a *Adapter) OrganizationRepository() organizationrepo.OrganizationRepository {
	return a.org
}

func (a *Adapter) PharmaEmployeeRepository() pharmaoarepo.EmployeeRepository   { return a.employee }
func (a *Adapter) PharmaProductRepository() pharmaoarepo.ProductRepository     { return a.product }
func (a *Adapter) PharmaSupplierRepository() pharmaoarepo.SupplierRepository   { return a.supplier }
func (a *Adapter) PharmaCustomerRepository() pharmaoarepo.CustomerRepository   { return a.customer }
func (a *Adapter) PharmaWarehouseRepository() pharmaoarepo.WarehouseRepository { return a.warehouse }

func (a *Adapter) DB() *gorm.DB {
	return a.db
}

func resolveMySQLDSN(raw string) (string, error) {
	value := strings.TrimSpace(raw)
	if !strings.HasPrefix(value, "mysql://") {
		return value, nil
	}

	u, err := url.Parse(value)
	if err != nil {
		return "", fmt.Errorf("parse mysql dsn: %w", err)
	}
	if u.Host == "" {
		return "", fmt.Errorf("mysql host is required")
	}

	username := ""
	password := ""
	if u.User != nil {
		username = u.User.Username()
		password, _ = u.User.Password()
	}
	if username == "" {
		return "", fmt.Errorf("mysql username is required")
	}
	database := strings.TrimPrefix(u.Path, "/")
	if database == "" {
		return "", fmt.Errorf("mysql database is required")
	}

	query := u.Query()
	if query.Get("parseTime") == "" {
		query.Set("parseTime", "true")
	}
	if query.Get("charset") == "" {
		query.Set("charset", "utf8mb4")
	}
	queryString := query.Encode()

	credential := username
	if password != "" {
		credential += ":" + password
	}
	return fmt.Sprintf("%s@tcp(%s)/%s?%s", credential, u.Host, database, queryString), nil
}

func tuneMySQLPool(db *sql.DB) {
	if db == nil {
		return
	}
	// Conservative defaults for local development stability.
	db.SetMaxOpenConns(30)
	db.SetMaxIdleConns(15)
	db.SetConnMaxLifetime(5 * time.Minute)
	db.SetConnMaxIdleTime(2 * time.Minute)
}

func waitMySQLReady(db *sql.DB) error {
	if db == nil {
		return fmt.Errorf("nil sql db")
	}
	const maxAttempts = 8
	var lastErr error
	for i := 0; i < maxAttempts; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		err := db.PingContext(ctx)
		cancel()
		if err == nil {
			return nil
		}
		lastErr = err
		time.Sleep(300 * time.Millisecond)
	}
	return lastErr
}
