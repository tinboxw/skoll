package store

import (
	"fmt"
	"strings"

	"github.com/tinboxw/skoll/internal/repository"
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
	Users            userrepo.UserRepository
	Roles            rolerepo.RoleRepository
	RBAC             rbacrepo.RBACRepository
	Audit            auditrepo.AuditRepository
	System           systemrepo.SystemRepository
	Plugins          pluginrepo.PluginRepository
	Permissions      permissionrepo.PermissionRepository
	Menus            menurepo.MenuRepository
	UnitOfWork       repository.UnitOfWork
	AuditEvents      auditrepo.EventRepository
	Files            filerepo.FileRepository
	Organization     organizationrepo.OrganizationRepository
	PharmaEmployees  pharmaoarepo.EmployeeRepository
	PharmaProducts   pharmaoarepo.ProductRepository
	PharmaSuppliers  pharmaoarepo.SupplierRepository
	PharmaCustomers  pharmaoarepo.CustomerRepository
	PharmaWarehouses pharmaoarepo.WarehouseRepository
	PharmaInventory  pharmaoarepo.InventoryRepository
	PharmaPurchases  pharmaoarepo.PurchaseRepository
	PharmaInbounds   pharmaoarepo.PurchaseInboundRepository
	PharmaSales      pharmaoarepo.SalesRepository
	PharmaStocktakes pharmaoarepo.StocktakeRepository
	PharmaTransfers  pharmaoarepo.TransferRepository
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
		organization := memory.NewOrganizationStore()
		return &Bundle{
			Users: users, Roles: roles, RBAC: rbac, Audit: audit, System: system, Plugins: plugins, Permissions: permissions,
			Menus: menus, UnitOfWork: sql.NewUnitOfWork(), AuditEvents: auditEvents, Files: files, Organization: organization,
			PharmaEmployees: pharmaoarepo.NewMemoryEmployeeRepository(), PharmaProducts: pharmaoarepo.NewMemoryProductRepository(),
			PharmaSuppliers: pharmaoarepo.NewMemorySupplierRepository(), PharmaCustomers: pharmaoarepo.NewMemoryCustomerRepository(),
			PharmaWarehouses: pharmaoarepo.NewMemoryWarehouseRepository(),
			PharmaInventory:  pharmaoarepo.NewMemoryInventoryRepository(),
			PharmaPurchases:  pharmaoarepo.NewMemoryPurchaseRepository(),
			PharmaInbounds:   pharmaoarepo.NewMemoryPurchaseInboundRepository(),
			PharmaSales:      pharmaoarepo.NewMemorySalesRepository(),
			PharmaStocktakes: pharmaoarepo.NewMemoryStocktakeRepository(),
			PharmaTransfers:  pharmaoarepo.NewMemoryTransferRepository(),
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
			Permissions:      primary.PermissionRepository(),
			Menus:            primary.MenuRepository(),
			UnitOfWork:       sql.NewUnitOfWorkWithDB(primary.DB()),
			AuditEvents:      auditEvents,
			Files:            primary.FileRepository(),
			Organization:     primary.OrganizationRepository(),
			PharmaEmployees:  primary.PharmaEmployeeRepository(),
			PharmaProducts:   primary.PharmaProductRepository(),
			PharmaSuppliers:  primary.PharmaSupplierRepository(),
			PharmaCustomers:  primary.PharmaCustomerRepository(),
			PharmaWarehouses: primary.PharmaWarehouseRepository(),
			PharmaInventory:  primary.PharmaInventoryRepository(),
			PharmaPurchases:  primary.PharmaPurchaseRepository(),
			PharmaInbounds:   primary.PharmaPurchaseInboundRepository(),
			PharmaSales:      primary.PharmaSalesRepository(),
			PharmaStocktakes: primary.PharmaStocktakeRepository(),
			PharmaTransfers:  primary.PharmaTransferRepository(),
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
			Permissions:      primary.PermissionRepository(),
			Menus:            primary.MenuRepository(),
			UnitOfWork:       sql.NewUnitOfWorkWithDB(primary.DB()),
			AuditEvents:      gormrepo.NewAuditStore(primary.DB()),
			Files:            primary.FileRepository(),
			Organization:     primary.OrganizationRepository(),
			PharmaEmployees:  primary.PharmaEmployeeRepository(),
			PharmaProducts:   primary.PharmaProductRepository(),
			PharmaSuppliers:  primary.PharmaSupplierRepository(),
			PharmaCustomers:  primary.PharmaCustomerRepository(),
			PharmaWarehouses: primary.PharmaWarehouseRepository(),
			PharmaInventory:  primary.PharmaInventoryRepository(),
			PharmaPurchases:  primary.PharmaPurchaseRepository(),
			PharmaInbounds:   primary.PharmaPurchaseInboundRepository(),
			PharmaSales:      primary.PharmaSalesRepository(),
			PharmaStocktakes: primary.PharmaStocktakeRepository(),
			PharmaTransfers:  primary.PharmaTransferRepository(),
		}, nil
	default:
		return nil, fmt.Errorf("unsupported store mode: %q", opts.Mode)
	}
}
