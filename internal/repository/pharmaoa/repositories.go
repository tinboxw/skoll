package pharmaoa

type Repositories struct {
	Employees           EmployeeRepository
	Products            ProductRepository
	Suppliers           SupplierRepository
	Customers           CustomerRepository
	Warehouses          WarehouseRepository
	Inventory           InventoryRepository
	Purchases           PurchaseRepository
	Inbounds            PurchaseInboundRepository
	Sales               SalesRepository
	Stocktakes          StocktakeRepository
	Transfers           TransferRepository
	Contracts           ContractRepository
	Complaints          QualityComplaintRepository
	Recalls             DrugRecallRepository
	FollowUps           CustomerFollowUpRepository
	Opportunities       SalesOpportunityRepository
	PaymentPlans        PaymentPlanRepository
	Invoices            InvoiceRecordRepository
	PaymentReminderJobs PaymentReminderJobRepository
	InventoryAlerts     InventoryAlertRepository
	ReportExports       ReportExportJobRepository
}
