package gormrepo

import (
	"encoding/json"
	"fmt"
	"time"

	domainpharma "github.com/tinboxw/skoll/internal/domain/pharmaoa"
	"github.com/tinboxw/skoll/internal/domain/shared"
)

type PharmaPurchaseRequestModel struct {
	ID                 string  `gorm:"primaryKey;size:64"`
	Number             string  `gorm:"size:128;not null;uniqueIndex:uk_pharma_purchase_requests_number"`
	SupplierID         string  `gorm:"size:64;not null;index:idx_pharma_purchase_requests_supplier"`
	RequesterID        string  `gorm:"size:64;not null"`
	ApproverID         string  `gorm:"size:64;not null;index:idx_pharma_purchase_requests_status_approver,priority:2"`
	Reason             string  `gorm:"size:1024"`
	LinesJSON          string  `gorm:"type:text;not null"`
	TotalAmount        float64 `gorm:"type:decimal(18,2);not null"`
	Status             string  `gorm:"size:32;not null;index:idx_pharma_purchase_requests_status_approver,priority:1"`
	WorkflowInstanceID string  `gorm:"size:64;not null"`
	PurchaseOrderID    string  `gorm:"size:64"`
	CreatedAt          time.Time
	UpdatedAt          time.Time
	CreatedBy          string `gorm:"size:64"`
	UpdatedBy          string `gorm:"size:64"`
}

func (PharmaPurchaseRequestModel) TableName() string { return "pharma_oa_purchase_requests" }

type PharmaPurchaseOrderModel struct {
	ID                string    `gorm:"primaryKey;size:64"`
	Number            string    `gorm:"size:128;not null;uniqueIndex:uk_pharma_purchase_orders_number"`
	PurchaseRequestID string    `gorm:"size:64;not null;uniqueIndex:uk_pharma_purchase_orders_request"`
	SupplierID        string    `gorm:"size:64;not null;index:idx_pharma_purchase_orders_supplier_status,priority:1"`
	LinesJSON         string    `gorm:"type:text;not null"`
	TotalAmount       float64   `gorm:"type:decimal(18,2);not null"`
	Status            string    `gorm:"size:32;not null;index:idx_pharma_purchase_orders_supplier_status,priority:2"`
	ApprovedBy        string    `gorm:"size:64;not null"`
	ApprovedAt        time.Time `gorm:"not null"`
	CreatedAt         time.Time
	UpdatedAt         time.Time
	CreatedBy         string `gorm:"size:64"`
	UpdatedBy         string `gorm:"size:64"`
}

func (PharmaPurchaseOrderModel) TableName() string { return "pharma_oa_purchase_orders" }

type PharmaPurchaseInboundModel struct {
	ID              string    `gorm:"primaryKey;size:64"`
	Number          string    `gorm:"size:128;not null;uniqueIndex:uk_pharma_inbounds_number"`
	PurchaseOrderID string    `gorm:"size:64;not null;index:idx_pharma_inbounds_order"`
	WarehouseID     string    `gorm:"size:64;not null"`
	AreaID          string    `gorm:"size:64;not null"`
	LocationID      string    `gorm:"size:64;not null"`
	LinesJSON       string    `gorm:"type:text;not null"`
	Status          string    `gorm:"size:32;not null"`
	ReceivedBy      string    `gorm:"size:64;not null"`
	ReceivedAt      time.Time `gorm:"not null"`
	AttachmentsJSON string    `gorm:"type:text;not null"`
	IdempotencyKey  string    `gorm:"size:255;not null;uniqueIndex:uk_pharma_inbounds_idempotency"`
	CreatedAt       time.Time
	UpdatedAt       time.Time
	CreatedBy       string `gorm:"size:64"`
	UpdatedBy       string `gorm:"size:64"`
}

func (PharmaPurchaseInboundModel) TableName() string { return "pharma_oa_purchase_inbounds" }

type PharmaSalesOrderModel struct {
	ID                string    `gorm:"primaryKey;size:64"`
	Number            string    `gorm:"size:128;not null;uniqueIndex:uk_pharma_sales_orders_number"`
	CustomerID        string    `gorm:"size:64;not null;index:idx_pharma_sales_orders_customer_status,priority:1"`
	LinesJSON         string    `gorm:"type:text;not null"`
	TotalAmount       float64   `gorm:"type:decimal(18,2);not null"`
	Status            string    `gorm:"size:32;not null;index:idx_pharma_sales_orders_customer_status,priority:2"`
	OrganizationID    string    `gorm:"size:64;index:idx_pharma_sales_orders_scope,priority:1"`
	ActorID           string    `gorm:"size:64"`
	BusinessCreatedAt time.Time `gorm:"not null"`
	CreatedAt         time.Time `gorm:"index:idx_pharma_sales_orders_scope,priority:2"`
	UpdatedAt         time.Time
	CreatedBy         string `gorm:"size:64"`
	UpdatedBy         string `gorm:"size:64"`
}

func (PharmaSalesOrderModel) TableName() string { return "pharma_oa_sales_orders" }

type PharmaSalesOutboundModel struct {
	ID             string    `gorm:"primaryKey;size:64"`
	Number         string    `gorm:"size:128;not null;uniqueIndex:uk_pharma_outbounds_number"`
	SalesOrderID   string    `gorm:"size:64;not null;index:idx_pharma_outbounds_order"`
	CustomerID     string    `gorm:"size:64;not null"`
	WarehouseID    string    `gorm:"size:64;not null"`
	AreaID         string    `gorm:"size:64;not null"`
	LocationID     string    `gorm:"size:64;not null"`
	LinesJSON      string    `gorm:"type:text;not null"`
	Status         string    `gorm:"size:32;not null"`
	ShippedBy      string    `gorm:"size:64;not null"`
	ShippedAt      time.Time `gorm:"not null"`
	IdempotencyKey string    `gorm:"size:255;not null;uniqueIndex:uk_pharma_outbounds_idempotency"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
	CreatedBy      string `gorm:"size:64"`
	UpdatedBy      string `gorm:"size:64"`
}

func (PharmaSalesOutboundModel) TableName() string { return "pharma_oa_sales_outbounds" }

type PharmaStocktakeModel struct {
	ID                 string    `gorm:"primaryKey;size:64"`
	Number             string    `gorm:"size:128;not null;uniqueIndex:uk_pharma_stocktakes_number"`
	ProductID          string    `gorm:"size:64;not null"`
	WarehouseID        string    `gorm:"size:64;not null;index:idx_pharma_stocktakes_warehouse_status,priority:1"`
	AreaID             string    `gorm:"size:64;not null"`
	LocationID         string    `gorm:"size:64;not null"`
	BatchID            string    `gorm:"size:64;not null"`
	SystemQuantity     int       `gorm:"not null"`
	ActualQuantity     int       `gorm:"not null"`
	Difference         int       `gorm:"not null"`
	Reason             string    `gorm:"size:1024"`
	ApproverID         string    `gorm:"size:64;not null"`
	WorkflowInstanceID string    `gorm:"size:64;not null"`
	LedgerID           string    `gorm:"size:64"`
	Status             string    `gorm:"size:32;not null;index:idx_pharma_stocktakes_warehouse_status,priority:2"`
	ActorID            string    `gorm:"size:64;not null"`
	ApprovedBy         string    `gorm:"size:64"`
	BusinessCreatedAt  time.Time `gorm:"not null"`
	CompletedAt        *time.Time
	CreatedAt          time.Time
	UpdatedAt          time.Time
	CreatedBy          string `gorm:"size:64"`
	UpdatedBy          string `gorm:"size:64"`
}

func (PharmaStocktakeModel) TableName() string { return "pharma_oa_stocktakes" }

type PharmaTransferModel struct {
	ID                string    `gorm:"primaryKey;size:64"`
	Number            string    `gorm:"size:128;not null;uniqueIndex:uk_pharma_transfers_number"`
	ProductID         string    `gorm:"size:64;not null"`
	BatchID           string    `gorm:"size:64;not null"`
	Quantity          int       `gorm:"not null"`
	SourceWarehouseID string    `gorm:"size:64;not null"`
	SourceAreaID      string    `gorm:"size:64;not null"`
	SourceLocationID  string    `gorm:"size:64;not null"`
	TargetWarehouseID string    `gorm:"size:64;not null"`
	TargetAreaID      string    `gorm:"size:64;not null"`
	TargetLocationID  string    `gorm:"size:64;not null"`
	OutLedgerID       string    `gorm:"size:64;not null"`
	InLedgerID        string    `gorm:"size:64;not null"`
	Status            string    `gorm:"size:32;not null;index:idx_pharma_transfers_status"`
	TransferredBy     string    `gorm:"size:64;not null"`
	TransferredAt     time.Time `gorm:"not null"`
	IdempotencyKey    string    `gorm:"size:255;not null;uniqueIndex:uk_pharma_transfers_idempotency"`
	CreatedAt         time.Time
	UpdatedAt         time.Time
	CreatedBy         string `gorm:"size:64"`
	UpdatedBy         string `gorm:"size:64"`
}

func (PharmaTransferModel) TableName() string { return "pharma_oa_transfers" }

func encodeOrderJSON(value any) (string, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("encode pharma order aggregate: %w", err)
	}
	return string(raw), nil
}
func decodeOrderJSON(raw string, target any) error {
	if err := json.Unmarshal([]byte(raw), target); err != nil {
		return fmt.Errorf("decode pharma order aggregate: %w", err)
	}
	return nil
}
func purchaseRequestModelFromDomain(v domainpharma.PurchaseRequest) (PharmaPurchaseRequestModel, error) {
	lines, err := encodeOrderJSON(v.Lines)
	return PharmaPurchaseRequestModel{ID: v.ID.String(), Number: v.Number, SupplierID: v.SupplierID, RequesterID: v.RequesterID, ApproverID: v.ApproverID, Reason: v.Reason, LinesJSON: lines, TotalAmount: v.TotalAmount, Status: string(v.Status), WorkflowInstanceID: v.WorkflowInstanceID, PurchaseOrderID: v.PurchaseOrderID, CreatedAt: v.Meta.CreatedAt, UpdatedAt: v.Meta.UpdatedAt}, err
}
func (m PharmaPurchaseRequestModel) toDomain() (*domainpharma.PurchaseRequest, error) {
	var lines []domainpharma.PurchaseLine
	if err := decodeOrderJSON(m.LinesJSON, &lines); err != nil {
		return nil, err
	}
	return &domainpharma.PurchaseRequest{ID: shared.ID(m.ID), Number: m.Number, SupplierID: m.SupplierID, RequesterID: m.RequesterID, ApproverID: m.ApproverID, Reason: m.Reason, Lines: lines, TotalAmount: m.TotalAmount, Status: domainpharma.PurchaseRequestStatus(m.Status), WorkflowInstanceID: m.WorkflowInstanceID, PurchaseOrderID: m.PurchaseOrderID, Meta: shared.AuditMeta{CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt}}, nil
}
func purchaseOrderModelFromDomain(v domainpharma.PurchaseOrder) (PharmaPurchaseOrderModel, error) {
	lines, err := encodeOrderJSON(v.Lines)
	return PharmaPurchaseOrderModel{ID: v.ID.String(), Number: v.Number, PurchaseRequestID: v.PurchaseRequestID, SupplierID: v.SupplierID, LinesJSON: lines, TotalAmount: v.TotalAmount, Status: string(v.Status), ApprovedBy: v.ApprovedBy, ApprovedAt: v.ApprovedAt, CreatedAt: v.Meta.CreatedAt, UpdatedAt: v.Meta.UpdatedAt}, err
}
func (m PharmaPurchaseOrderModel) toDomain() (*domainpharma.PurchaseOrder, error) {
	var lines []domainpharma.PurchaseLine
	if err := decodeOrderJSON(m.LinesJSON, &lines); err != nil {
		return nil, err
	}
	return &domainpharma.PurchaseOrder{ID: shared.ID(m.ID), Number: m.Number, PurchaseRequestID: m.PurchaseRequestID, SupplierID: m.SupplierID, Lines: lines, TotalAmount: m.TotalAmount, Status: domainpharma.PurchaseOrderStatus(m.Status), ApprovedBy: m.ApprovedBy, ApprovedAt: m.ApprovedAt, Meta: shared.AuditMeta{CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt}}, nil
}
func purchaseInboundModelFromDomain(v domainpharma.PurchaseInbound) (PharmaPurchaseInboundModel, error) {
	lines, err := encodeOrderJSON(v.Lines)
	if err != nil {
		return PharmaPurchaseInboundModel{}, err
	}
	attachments, err := encodeOrderJSON(v.Attachments)
	return PharmaPurchaseInboundModel{ID: v.ID.String(), Number: v.Number, PurchaseOrderID: v.PurchaseOrderID, WarehouseID: v.WarehouseID, AreaID: v.AreaID, LocationID: v.LocationID, LinesJSON: lines, Status: string(v.Status), ReceivedBy: v.ReceivedBy, ReceivedAt: v.ReceivedAt, AttachmentsJSON: attachments, IdempotencyKey: v.ID.String(), CreatedAt: v.Meta.CreatedAt, UpdatedAt: v.Meta.UpdatedAt}, err
}
func (m PharmaPurchaseInboundModel) toDomain() (*domainpharma.PurchaseInbound, error) {
	var lines []domainpharma.PurchaseInboundLine
	var attachments []domainpharma.InboundAttachment
	if err := decodeOrderJSON(m.LinesJSON, &lines); err != nil {
		return nil, err
	}
	if err := decodeOrderJSON(m.AttachmentsJSON, &attachments); err != nil {
		return nil, err
	}
	return &domainpharma.PurchaseInbound{ID: shared.ID(m.ID), Number: m.Number, PurchaseOrderID: m.PurchaseOrderID, WarehouseID: m.WarehouseID, AreaID: m.AreaID, LocationID: m.LocationID, Lines: lines, Attachments: attachments, Status: domainpharma.PurchaseInboundStatus(m.Status), ReceivedBy: m.ReceivedBy, ReceivedAt: m.ReceivedAt, Meta: shared.AuditMeta{CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt}}, nil
}
func salesOrderModelFromDomain(v domainpharma.SalesOrder) (PharmaSalesOrderModel, error) {
	lines, err := encodeOrderJSON(v.Lines)
	return PharmaSalesOrderModel{ID: v.ID.String(), Number: v.Number, CustomerID: v.CustomerID, LinesJSON: lines, TotalAmount: v.TotalAmount, Status: string(v.Status), ActorID: v.CreatedBy, BusinessCreatedAt: v.CreatedAt, CreatedAt: v.Meta.CreatedAt, UpdatedAt: v.Meta.UpdatedAt}, err
}
func (m PharmaSalesOrderModel) toDomain() (*domainpharma.SalesOrder, error) {
	var lines []domainpharma.SalesLine
	if err := decodeOrderJSON(m.LinesJSON, &lines); err != nil {
		return nil, err
	}
	return &domainpharma.SalesOrder{ID: shared.ID(m.ID), Number: m.Number, CustomerID: m.CustomerID, Lines: lines, TotalAmount: m.TotalAmount, Status: domainpharma.SalesOrderStatus(m.Status), CreatedBy: m.ActorID, CreatedAt: m.BusinessCreatedAt, Meta: shared.AuditMeta{CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt}}, nil
}
func salesOutboundModelFromDomain(v domainpharma.SalesOutbound) (PharmaSalesOutboundModel, error) {
	lines, err := encodeOrderJSON(v.Lines)
	return PharmaSalesOutboundModel{ID: v.ID.String(), Number: v.Number, SalesOrderID: v.SalesOrderID, CustomerID: v.CustomerID, WarehouseID: v.WarehouseID, AreaID: v.AreaID, LocationID: v.LocationID, LinesJSON: lines, Status: string(v.Status), ShippedBy: v.ShippedBy, ShippedAt: v.ShippedAt, IdempotencyKey: v.ID.String(), CreatedAt: v.Meta.CreatedAt, UpdatedAt: v.Meta.UpdatedAt}, err
}
func (m PharmaSalesOutboundModel) toDomain() (*domainpharma.SalesOutbound, error) {
	var lines []domainpharma.SalesOutboundLine
	if err := decodeOrderJSON(m.LinesJSON, &lines); err != nil {
		return nil, err
	}
	return &domainpharma.SalesOutbound{ID: shared.ID(m.ID), Number: m.Number, SalesOrderID: m.SalesOrderID, CustomerID: m.CustomerID, WarehouseID: m.WarehouseID, AreaID: m.AreaID, LocationID: m.LocationID, Lines: lines, Status: domainpharma.SalesOutboundStatus(m.Status), ShippedBy: m.ShippedBy, ShippedAt: m.ShippedAt, Meta: shared.AuditMeta{CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt}}, nil
}

func stocktakeModelFromDomain(v domainpharma.StocktakeOrder) PharmaStocktakeModel {
	return PharmaStocktakeModel{ID: v.ID.String(), Number: v.Number, ProductID: v.ProductID, WarehouseID: v.WarehouseID, AreaID: v.AreaID, LocationID: v.LocationID, BatchID: v.BatchID, SystemQuantity: v.SystemQuantity, ActualQuantity: v.ActualQuantity, Difference: v.Difference, Reason: v.Reason, ApproverID: v.ApproverID, WorkflowInstanceID: v.WorkflowInstanceID, LedgerID: v.LedgerID, Status: string(v.Status), ActorID: v.CreatedBy, ApprovedBy: v.ApprovedBy, BusinessCreatedAt: v.CreatedAt, CompletedAt: v.CompletedAt, CreatedAt: v.Meta.CreatedAt, UpdatedAt: v.Meta.UpdatedAt}
}
func (m PharmaStocktakeModel) toDomain() *domainpharma.StocktakeOrder {
	return &domainpharma.StocktakeOrder{ID: shared.ID(m.ID), Number: m.Number, ProductID: m.ProductID, WarehouseID: m.WarehouseID, AreaID: m.AreaID, LocationID: m.LocationID, BatchID: m.BatchID, SystemQuantity: m.SystemQuantity, ActualQuantity: m.ActualQuantity, Difference: m.Difference, Reason: m.Reason, ApproverID: m.ApproverID, WorkflowInstanceID: m.WorkflowInstanceID, LedgerID: m.LedgerID, Status: domainpharma.StocktakeOrderStatus(m.Status), CreatedBy: m.ActorID, ApprovedBy: m.ApprovedBy, CreatedAt: m.BusinessCreatedAt, CompletedAt: m.CompletedAt, Meta: shared.AuditMeta{CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt}}
}
func transferModelFromDomain(v domainpharma.TransferOrder) PharmaTransferModel {
	return PharmaTransferModel{ID: v.ID.String(), Number: v.Number, ProductID: v.ProductID, BatchID: v.BatchID, Quantity: v.Quantity, SourceWarehouseID: v.FromWarehouseID, SourceAreaID: v.FromAreaID, SourceLocationID: v.FromLocationID, TargetWarehouseID: v.ToWarehouseID, TargetAreaID: v.ToAreaID, TargetLocationID: v.ToLocationID, OutLedgerID: v.OutLedgerID, InLedgerID: v.InLedgerID, Status: string(v.Status), TransferredBy: v.TransferredBy, TransferredAt: v.TransferredAt, IdempotencyKey: v.ID.String(), CreatedAt: v.Meta.CreatedAt, UpdatedAt: v.Meta.UpdatedAt}
}
func (m PharmaTransferModel) toDomain() *domainpharma.TransferOrder {
	return &domainpharma.TransferOrder{ID: shared.ID(m.ID), Number: m.Number, ProductID: m.ProductID, BatchID: m.BatchID, Quantity: m.Quantity, FromWarehouseID: m.SourceWarehouseID, FromAreaID: m.SourceAreaID, FromLocationID: m.SourceLocationID, ToWarehouseID: m.TargetWarehouseID, ToAreaID: m.TargetAreaID, ToLocationID: m.TargetLocationID, OutLedgerID: m.OutLedgerID, InLedgerID: m.InLedgerID, Status: domainpharma.TransferOrderStatus(m.Status), TransferredBy: m.TransferredBy, TransferredAt: m.TransferredAt, Meta: shared.AuditMeta{CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt}}
}
