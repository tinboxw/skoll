package pharmaoa

import (
	"fmt"
	"net/http"

	pharmaoahttp "github.com/tinboxw/skoll/internal/handler/http/v1/pharmaoa"
	pharmaoarepo "github.com/tinboxw/skoll/internal/repository/pharmaoa"
	auditsvc "github.com/tinboxw/skoll/internal/service/audit"
	filesvc "github.com/tinboxw/skoll/internal/service/file"
	notificationsvc "github.com/tinboxw/skoll/internal/service/notification"
	pharmaoasvc "github.com/tinboxw/skoll/internal/service/pharmaoa"
	workflowsvc "github.com/tinboxw/skoll/internal/service/workflow"
	"github.com/tinboxw/skoll/internal/store"
	"github.com/tinboxw/skoll/pkg/pluginsdk"
)

const PluginID = "pharma_oa"

type Dependencies struct {
	Stores       *store.Bundle
	Host         pluginsdk.HostServices
	Audit        auditsvc.Service
	Workflow     workflowsvc.Service
	File         filesvc.Service
	Notification *notificationsvc.Service
}

func NewBackend(deps Dependencies) (http.Handler, error) {
	if err := deps.Host.Validate(); err != nil {
		return nil, err
	}
	stores, err := repositories(deps)
	if err != nil {
		return nil, err
	}

	employees := pharmaoasvc.NewEmployeeService(deps.Audit, stores.Employees)
	products := pharmaoasvc.NewProductService(deps.Audit, stores.Products)
	suppliers := pharmaoasvc.NewSupplierService(deps.Audit, stores.Suppliers)
	customers, err := NewCustomerService(deps)
	if err != nil {
		return nil, err
	}
	followUps := pharmaoasvc.NewCustomerFollowUpService(customers, deps.Audit, stores.FollowUps)
	opportunities := pharmaoasvc.NewSalesOpportunityService(customers, products, deps.Audit, stores.Opportunities)
	warehouses := pharmaoasvc.NewWarehouseService(deps.Audit, stores.Warehouses)
	masterData := pharmaoasvc.NewMasterDataExchangeService(employees, products, suppliers, customers)
	purchases := pharmaoasvc.NewPurchaseService(suppliers, deps.Workflow, deps.Audit, stores.Purchases)
	inventory := pharmaoasvc.NewInventoryService(deps.Audit, stores.Inventory)
	inbounds := pharmaoasvc.NewPurchaseInboundService(purchases, warehouses, inventory, deps.Audit, stores.Inbounds)
	sales := pharmaoasvc.NewSalesService(customers, warehouses, inventory, deps.Audit, stores.Sales)
	inventoryOperations := pharmaoasvc.NewInventoryOperationService(inventory, warehouses, deps.Workflow, deps.Audit, pharmaoasvc.InventoryOperationRepositories{
		Stocktakes: stores.Stocktakes,
		Transfers:  stores.Transfers,
	})
	paymentInvoices := pharmaoasvc.NewPaymentInvoiceService(sales, deps.Notification, deps.Audit, pharmaoasvc.PaymentInvoiceRepositories{
		Plans:    stores.PaymentPlans,
		Invoices: stores.Invoices,
		Jobs:     stores.PaymentReminderJobs,
	})
	inventoryAlerts := pharmaoasvc.NewInventoryAlertService(inventory, deps.Notification, deps.Audit, stores.InventoryAlerts)
	announcements := pharmaoasvc.NewAnnouncementService(deps.Audit)
	contracts := pharmaoasvc.NewContractService(suppliers, customers, deps.Workflow, deps.File, deps.Notification, deps.Audit, stores.Contracts)
	qualifications := pharmaoasvc.NewQualificationService(employees, suppliers, customers, deps.Notification, deps.Audit)
	complaints := pharmaoasvc.NewQualityComplaintService(customers, products, inventory, deps.Workflow, deps.File, deps.Audit, stores.Complaints)
	recalls := pharmaoasvc.NewDrugRecallService(sales, inventory, products, customers, complaints, deps.Audit, stores.Recalls)
	coldChain := pharmaoasvc.NewColdChainService(inventory, warehouses, deps.Notification, deps.Audit)
	compliance := pharmaoasvc.NewComplianceDashboardService(qualifications, complaints, recalls, coldChain, deps.Audit)
	metrics := pharmaoasvc.NewBusinessMetricsService(inventoryAlerts, qualifications, purchases, followUps, sales, deps.Audit)
	reportExports := pharmaoasvc.NewReportExportService(metrics, deps.File, deps.Audit, stores.ReportExports)
	demoSeed := pharmaoasvc.NewDemoSeedService(pharmaoasvc.DemoSeedDependencies{
		Employees: employees, Products: products, Suppliers: suppliers, Customers: customers,
		Warehouses: warehouses, Purchases: purchases, Inbounds: inbounds, Sales: sales,
		Inventory: inventory, FollowUps: followUps, Audit: deps.Audit,
	})

	mux := http.NewServeMux()
	pharmaoahttp.RegisterEmployeeRoutes(mux, employees)
	pharmaoahttp.RegisterProductRoutes(mux, products)
	pharmaoahttp.RegisterSupplierRoutes(mux, suppliers)
	pharmaoahttp.RegisterCustomerRoutes(mux, customers, deps.Host.DataScopes)
	pharmaoahttp.RegisterCustomerFollowUpRoutes(mux, followUps)
	pharmaoahttp.RegisterSalesOpportunityRoutes(mux, opportunities)
	pharmaoahttp.RegisterWarehouseRoutes(mux, warehouses)
	pharmaoahttp.RegisterMasterDataExchangeRoutes(mux, masterData)
	pharmaoahttp.RegisterPurchaseRoutes(mux, purchases)
	pharmaoahttp.RegisterPurchaseInboundRoutes(mux, inbounds)
	pharmaoahttp.RegisterSalesRoutes(mux, sales)
	pharmaoahttp.RegisterPaymentInvoiceRoutes(mux, paymentInvoices)
	pharmaoahttp.RegisterInventoryOperationRoutes(mux, inventoryOperations)
	pharmaoahttp.RegisterInventoryAlertRoutes(mux, inventoryAlerts)
	pharmaoahttp.RegisterAnnouncementRoutes(mux, announcements)
	pharmaoahttp.RegisterContractRoutes(mux, contracts)
	pharmaoahttp.RegisterQualificationRoutes(mux, qualifications)
	pharmaoahttp.RegisterQualityComplaintRoutes(mux, complaints)
	pharmaoahttp.RegisterDrugRecallRoutes(mux, recalls)
	pharmaoahttp.RegisterColdChainRoutes(mux, coldChain)
	pharmaoahttp.RegisterComplianceDashboardRoutes(mux, compliance)
	pharmaoahttp.RegisterBusinessMetricsRoutes(mux, metrics)
	pharmaoahttp.RegisterReportExportRoutes(mux, reportExports)
	pharmaoahttp.RegisterDemoSeedRoutes(mux, demoSeed)
	return mux, nil
}

func NewCustomerService(deps Dependencies) (pharmaoasvc.CustomerService, error) {
	stores, err := repositories(deps)
	if err != nil {
		return nil, err
	}
	return pharmaoasvc.NewCustomerService(deps.Audit, stores.Customers), nil
}

func repositories(deps Dependencies) (*pharmaoarepo.Repositories, error) {
	if deps.Stores == nil || deps.Stores.PharmaOA == nil {
		return nil, fmt.Errorf("pharma OA store factory is required")
	}
	repositories := deps.Stores.PharmaOA()
	if repositories == nil {
		return nil, fmt.Errorf("pharma OA stores are required")
	}
	return repositories, nil
}
