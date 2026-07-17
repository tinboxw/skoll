package pharmaoa

import (
	"context"

	domainpharma "github.com/tinboxw/skoll/internal/domain/pharmaoa"
	"github.com/tinboxw/skoll/internal/domain/shared"
)

type ListFilter struct {
	Keyword        string
	Status         string
	OrganizationID shared.ID
	Offset         int
	Limit          int
}

type AggregateRepository[T any] interface {
	Upsert(ctx context.Context, item *T) error
	Get(ctx context.Context, id shared.ID) (*T, error)
	List(ctx context.Context, filter ListFilter) ([]T, error)
}

type EmployeeRepository interface {
	AggregateRepository[domainpharma.Employee]
	GetByCode(ctx context.Context, code string) (*domainpharma.Employee, error)
}

type ProductRepository interface {
	AggregateRepository[domainpharma.Product]
	GetByCode(ctx context.Context, code string) (*domainpharma.Product, error)
}

type SupplierRepository interface {
	AggregateRepository[domainpharma.Supplier]
	GetByCode(ctx context.Context, code string) (*domainpharma.Supplier, error)
}

type CustomerRepository interface {
	AggregateRepository[domainpharma.Customer]
	GetByCode(ctx context.Context, code string) (*domainpharma.Customer, error)
}

type WarehouseRepository interface {
	AggregateRepository[domainpharma.Warehouse]
	GetByCode(ctx context.Context, code string) (*domainpharma.Warehouse, error)
}

type InventoryTransaction interface {
	GetBalanceForUpdate(ctx context.Context, position domainpharma.StockPosition) (*domainpharma.StockBalance, error)
	UpsertBatch(ctx context.Context, batch domainpharma.StockBatch) error
	UpsertBalance(ctx context.Context, balance domainpharma.StockBalance) error
	AppendLedger(ctx context.Context, entry domainpharma.StockLedgerEntry) error
	UpsertLock(ctx context.Context, lock domainpharma.StockLock) error
}

type InventoryRepository interface {
	Transact(ctx context.Context, fn func(tx InventoryTransaction) error) error
	ListBalances(ctx context.Context, filter ListFilter) ([]domainpharma.StockBalance, error)
	ListBatches(ctx context.Context, filter ListFilter) ([]domainpharma.StockBatch, error)
	ListLedger(ctx context.Context, filter ListFilter) ([]domainpharma.StockLedgerEntry, error)
}

type PurchaseRepository interface {
	UpsertRequest(ctx context.Context, request *domainpharma.PurchaseRequest) error
	GetRequest(ctx context.Context, id shared.ID) (*domainpharma.PurchaseRequest, error)
	ListRequests(ctx context.Context, filter ListFilter) ([]domainpharma.PurchaseRequest, error)
	UpsertOrder(ctx context.Context, order *domainpharma.PurchaseOrder) error
	GetOrder(ctx context.Context, id shared.ID) (*domainpharma.PurchaseOrder, error)
	ListOrders(ctx context.Context, filter ListFilter) ([]domainpharma.PurchaseOrder, error)
}

type PurchaseInboundRepository interface {
	AggregateRepository[domainpharma.PurchaseInbound]
}

type SalesRepository interface {
	UpsertOrder(ctx context.Context, order *domainpharma.SalesOrder) error
	GetOrder(ctx context.Context, id shared.ID) (*domainpharma.SalesOrder, error)
	ListOrders(ctx context.Context, filter ListFilter) ([]domainpharma.SalesOrder, error)
	UpsertOutbound(ctx context.Context, outbound *domainpharma.SalesOutbound) error
	GetOutbound(ctx context.Context, id shared.ID) (*domainpharma.SalesOutbound, error)
	ListOutbounds(ctx context.Context, filter ListFilter) ([]domainpharma.SalesOutbound, error)
}

type ContractRepository interface {
	AggregateRepository[domainpharma.Contract]
}

type QualityComplaintRepository interface {
	AggregateRepository[domainpharma.QualityComplaint]
}

type DrugRecallRepository interface {
	AggregateRepository[domainpharma.DrugRecall]
}

type CustomerFollowUpRepository interface {
	AggregateRepository[domainpharma.CustomerFollowUp]
}

type SalesOpportunityRepository interface {
	AggregateRepository[domainpharma.SalesOpportunity]
}
