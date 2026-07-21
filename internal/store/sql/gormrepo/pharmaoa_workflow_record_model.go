package gormrepo

import "time"

type PharmaContractModel struct {
	ID                 string    `gorm:"primaryKey;size:64"`
	Number             string    `gorm:"size:128;not null;uniqueIndex:uk_pharma_contracts_number"`
	Title              string    `gorm:"size:255;not null"`
	PartyType          string    `gorm:"size:32;not null;index:idx_pharma_contracts_party_status,priority:1"`
	PartyID            string    `gorm:"size:64;not null;index:idx_pharma_contracts_party_status,priority:2"`
	Status             string    `gorm:"size:32;not null;index:idx_pharma_contracts_party_status,priority:3;index:idx_pharma_contracts_expiry,priority:2"`
	WorkflowInstanceID string    `gorm:"size:64;not null;index"`
	OwnerID            string    `gorm:"size:64;not null;index"`
	ApproverID         string    `gorm:"size:64;not null"`
	Amount             float64   `gorm:"type:decimal(18,2);not null"`
	Currency           string    `gorm:"size:16;not null"`
	EffectiveAt        time.Time `gorm:"not null"`
	ExpiresAt          time.Time `gorm:"not null;index:idx_pharma_contracts_expiry,priority:1"`
	PayloadJSON        string    `gorm:"type:text;not null"`
	CreatedAt          time.Time `gorm:"not null"`
	UpdatedAt          time.Time `gorm:"not null"`
	CreatedBy          string    `gorm:"size:64;not null;default:''"`
	UpdatedBy          string    `gorm:"size:64;not null;default:''"`
}

func (PharmaContractModel) TableName() string { return "pharma_oa_contracts" }

type PharmaQualityComplaintModel struct {
	ID                 string    `gorm:"primaryKey;size:64"`
	Number             string    `gorm:"size:128;not null;uniqueIndex:uk_pharma_complaints_number"`
	Title              string    `gorm:"size:255;not null"`
	CustomerID         string    `gorm:"size:64;not null;index"`
	ProductID          string    `gorm:"size:64;not null;index:idx_pharma_complaints_status_product,priority:2"`
	BatchID            string    `gorm:"size:64;not null;index:idx_pharma_complaints_batch"`
	Status             string    `gorm:"size:32;not null;index:idx_pharma_complaints_status_product,priority:1"`
	WorkflowInstanceID string    `gorm:"size:64;not null;index"`
	ReporterID         string    `gorm:"size:64;not null"`
	HandlerID          string    `gorm:"size:64;not null"`
	PayloadJSON        string    `gorm:"type:text;not null"`
	CreatedAt          time.Time `gorm:"not null"`
	UpdatedAt          time.Time `gorm:"not null"`
	CreatedBy          string    `gorm:"size:64;not null;default:''"`
	UpdatedBy          string    `gorm:"size:64;not null;default:''"`
}

func (PharmaQualityComplaintModel) TableName() string { return "pharma_oa_quality_complaints" }

type PharmaDrugRecallModel struct {
	ID                string    `gorm:"primaryKey;size:64"`
	Number            string    `gorm:"size:128;not null;uniqueIndex:uk_pharma_recalls_number"`
	Title             string    `gorm:"size:255;not null"`
	ProductID         string    `gorm:"size:64;not null"`
	BatchID           string    `gorm:"size:64;not null;index:idx_pharma_recalls_batch_status,priority:1"`
	SourceComplaintID string    `gorm:"size:64;not null;default:'';index"`
	Status            string    `gorm:"size:32;not null;index:idx_pharma_recalls_batch_status,priority:2"`
	InitiatedBy       string    `gorm:"size:64;not null"`
	PayloadJSON       string    `gorm:"type:text;not null"`
	CreatedAt         time.Time `gorm:"not null"`
	UpdatedAt         time.Time `gorm:"not null"`
	CreatedBy         string    `gorm:"size:64;not null;default:''"`
	UpdatedBy         string    `gorm:"size:64;not null;default:''"`
}

func (PharmaDrugRecallModel) TableName() string { return "pharma_oa_drug_recalls" }

type PharmaCustomerFollowUpModel struct {
	ID             string    `gorm:"primaryKey;size:64"`
	CustomerID     string    `gorm:"size:64;not null;index:idx_pharma_followups_customer_time,priority:1"`
	CustomerCode   string    `gorm:"size:128;not null"`
	CustomerName   string    `gorm:"size:255;not null"`
	OwnerID        string    `gorm:"size:64;not null;index:idx_pharma_followups_scope_status,priority:2"`
	OrganizationID string    `gorm:"size:64;not null;index:idx_pharma_followups_scope_status,priority:1"`
	Status         string    `gorm:"size:32;not null;index:idx_pharma_followups_scope_status,priority:3"`
	ScheduledAt    time.Time `gorm:"not null;index:idx_pharma_followups_customer_time,priority:2"`
	PayloadJSON    string    `gorm:"type:text;not null"`
	CreatedAt      time.Time `gorm:"not null"`
	UpdatedAt      time.Time `gorm:"not null"`
	CreatedBy      string    `gorm:"size:64;not null;default:''"`
	UpdatedBy      string    `gorm:"size:64;not null;default:''"`
}

func (PharmaCustomerFollowUpModel) TableName() string { return "pharma_oa_customer_follow_ups" }

type PharmaSalesOpportunityModel struct {
	ID                  string    `gorm:"primaryKey;size:64"`
	Title               string    `gorm:"size:255;not null"`
	CustomerID          string    `gorm:"size:64;not null;index:idx_pharma_opportunities_customer"`
	CustomerCode        string    `gorm:"size:128;not null"`
	CustomerName        string    `gorm:"size:255;not null"`
	OwnerID             string    `gorm:"size:64;not null;index:idx_pharma_opportunities_scope_stage,priority:2"`
	OrganizationID      string    `gorm:"size:64;not null;index:idx_pharma_opportunities_scope_stage,priority:1"`
	Stage               string    `gorm:"size:32;not null;index:idx_pharma_opportunities_scope_stage,priority:3"`
	ExpectedAmountCents int64     `gorm:"not null"`
	EstimatedCloseDate  time.Time `gorm:"not null;index"`
	PayloadJSON         string    `gorm:"type:text;not null"`
	CreatedAt           time.Time `gorm:"not null"`
	UpdatedAt           time.Time `gorm:"not null"`
	CreatedBy           string    `gorm:"size:64;not null;default:''"`
	UpdatedBy           string    `gorm:"size:64;not null;default:''"`
}

func (PharmaSalesOpportunityModel) TableName() string { return "pharma_oa_sales_opportunities" }

type PharmaPaymentPlanModel struct {
	ID               string    `gorm:"primaryKey;size:64"`
	SalesOrderID     string    `gorm:"size:64;not null;uniqueIndex:uk_pharma_payment_plan_order"`
	SalesOrderNumber string    `gorm:"size:128;not null"`
	CustomerID       string    `gorm:"size:64;not null"`
	OwnerID          string    `gorm:"size:64;not null;index"`
	OrderTotalCents  int64     `gorm:"not null"`
	AmountCents      int64     `gorm:"not null"`
	PaidAmountCents  int64     `gorm:"not null"`
	Status           string    `gorm:"size:32;not null;index:idx_pharma_payment_plan_due_status,priority:1"`
	DueAt            time.Time `gorm:"not null;index:idx_pharma_payment_plan_due_status,priority:2"`
	PayloadJSON      string    `gorm:"type:text;not null"`
	CreatedAt        time.Time `gorm:"not null"`
	UpdatedAt        time.Time `gorm:"not null"`
	CreatedBy        string    `gorm:"size:64;not null;default:''"`
	UpdatedBy        string    `gorm:"size:64;not null;default:''"`
}

func (PharmaPaymentPlanModel) TableName() string { return "pharma_oa_payment_plans" }

type PharmaInvoiceRecordModel struct {
	ID               string    `gorm:"primaryKey;size:64"`
	Number           string    `gorm:"size:128;not null;uniqueIndex:uk_pharma_invoices_number"`
	SalesOrderID     string    `gorm:"size:64;not null;index:idx_pharma_invoices_order_status,priority:1"`
	SalesOrderNumber string    `gorm:"size:128;not null"`
	CustomerID       string    `gorm:"size:64;not null"`
	OwnerID          string    `gorm:"size:64;not null;index"`
	OrderTotalCents  int64     `gorm:"not null"`
	AmountCents      int64     `gorm:"not null"`
	Status           string    `gorm:"size:32;not null;index:idx_pharma_invoices_order_status,priority:2"`
	IssuedAt         time.Time `gorm:"not null"`
	VoidedAt         *time.Time
	PayloadJSON      string    `gorm:"type:text;not null"`
	CreatedAt        time.Time `gorm:"not null"`
	UpdatedAt        time.Time `gorm:"not null"`
	CreatedBy        string    `gorm:"size:64;not null;default:''"`
	UpdatedBy        string    `gorm:"size:64;not null;default:''"`
}

func (PharmaInvoiceRecordModel) TableName() string { return "pharma_oa_invoice_records" }

type PharmaPaymentReminderJobModel struct {
	ID          string `gorm:"primaryKey;size:64"`
	Status      string `gorm:"size:32;not null;index:idx_pharma_payment_reminder_jobs_status_created,priority:1"`
	RecipientID string `gorm:"size:64;not null"`
	OwnerID     string `gorm:"size:64;not null;default:'';index"`
	InitiatedBy string `gorm:"size:64;not null"`
	LastRunBy   string `gorm:"size:64;not null"`
	RetryCount  int    `gorm:"not null"`
	Error       string `gorm:"size:1024;not null;default:''"`
	StartedAt   *time.Time
	CompletedAt *time.Time
	PayloadJSON string    `gorm:"type:text;not null"`
	CreatedAt   time.Time `gorm:"not null;index:idx_pharma_payment_reminder_jobs_status_created,priority:2"`
	UpdatedAt   time.Time `gorm:"not null"`
	CreatedBy   string    `gorm:"size:64;not null;default:''"`
	UpdatedBy   string    `gorm:"size:64;not null;default:''"`
}

func (PharmaPaymentReminderJobModel) TableName() string { return "pharma_oa_payment_reminder_jobs" }

type PharmaInventoryAlertModel struct {
	ID             string    `gorm:"primaryKey;size:64"`
	Type           string    `gorm:"size:32;not null;uniqueIndex:uk_pharma_inventory_alert_position,priority:1;index:idx_pharma_inventory_alerts_status_type_seen,priority:2"`
	Status         string    `gorm:"size:32;not null;index:idx_pharma_inventory_alerts_status_seen,priority:1;index:idx_pharma_inventory_alerts_status_type_seen,priority:1"`
	BalanceID      string    `gorm:"size:64;not null;uniqueIndex:uk_pharma_inventory_alert_position,priority:2"`
	ProductID      string    `gorm:"size:64;not null"`
	WarehouseID    string    `gorm:"size:64;not null"`
	BatchID        string    `gorm:"size:64;not null;index"`
	RecipientID    string    `gorm:"size:64;not null"`
	NotificationID string    `gorm:"size:128;not null"`
	LastSeenAt     time.Time `gorm:"not null;index:idx_pharma_inventory_alerts_status_seen,priority:2;index:idx_pharma_inventory_alerts_status_type_seen,priority:3"`
	ResolvedAt     *time.Time
	PayloadJSON    string    `gorm:"type:text;not null"`
	CreatedAt      time.Time `gorm:"not null"`
	UpdatedAt      time.Time `gorm:"not null"`
	CreatedBy      string    `gorm:"size:64;not null;default:''"`
	UpdatedBy      string    `gorm:"size:64;not null;default:''"`
}

func (PharmaInventoryAlertModel) TableName() string { return "pharma_oa_inventory_alerts" }

type PharmaInventoryAlertJobModel struct {
	ID             string `gorm:"primaryKey;size:64"`
	Status         string `gorm:"size:32;not null;index:idx_pharma_alert_jobs_status_created,priority:1"`
	IdempotencyKey string `gorm:"size:191;not null;uniqueIndex:uk_pharma_alert_jobs_idempotency"`
	RetryCount     int    `gorm:"not null"`
	Error          string `gorm:"size:1024;not null;default:''"`
	StartedAt      *time.Time
	CompletedAt    *time.Time
	PayloadJSON    string    `gorm:"type:text;not null"`
	CreatedAt      time.Time `gorm:"not null;index:idx_pharma_alert_jobs_status_created,priority:2"`
	UpdatedAt      time.Time `gorm:"not null"`
	CreatedBy      string    `gorm:"size:64;not null;default:''"`
	UpdatedBy      string    `gorm:"size:64;not null;default:''"`
}

func (PharmaInventoryAlertJobModel) TableName() string { return "pharma_oa_inventory_alert_jobs" }

type PharmaReportExportJobModel struct {
	ID             string `gorm:"primaryKey;size:64"`
	ReportType     string `gorm:"size:64;not null"`
	Status         string `gorm:"size:32;not null;index:idx_pharma_export_jobs_status_created,priority:1;index:idx_pharma_export_jobs_owner_status_created,priority:2"`
	OwnerID        string `gorm:"size:64;not null;index;index:idx_pharma_export_jobs_owner_status_created,priority:1"`
	IdempotencyKey string `gorm:"size:191;not null;uniqueIndex:uk_pharma_export_jobs_idempotency"`
	RetryCount     int    `gorm:"not null"`
	FileID         string `gorm:"size:64;not null;default:''"`
	Error          string `gorm:"size:1024;not null;default:''"`
	StartedAt      *time.Time
	CompletedAt    *time.Time
	PayloadJSON    string    `gorm:"type:text;not null"`
	CreatedAt      time.Time `gorm:"not null;index:idx_pharma_export_jobs_status_created,priority:2;index:idx_pharma_export_jobs_owner_status_created,priority:3"`
	UpdatedAt      time.Time `gorm:"not null"`
	CreatedBy      string    `gorm:"size:64;not null;default:''"`
	UpdatedBy      string    `gorm:"size:64;not null;default:''"`
}

func (PharmaReportExportJobModel) TableName() string { return "pharma_oa_report_export_jobs" }
