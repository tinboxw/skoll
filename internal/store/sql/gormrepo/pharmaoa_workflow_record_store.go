package gormrepo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	domainpharma "github.com/tinboxw/skoll/internal/domain/pharmaoa"
	"github.com/tinboxw/skoll/internal/domain/shared"
	pharmaoarepo "github.com/tinboxw/skoll/internal/repository/pharmaoa"
	"gorm.io/gorm"
)

type pharmaJSONAggregateStore[T any] struct {
	db           *gorm.DB
	table        string
	id           func(*T) string
	columns      func(*T) (map[string]any, error)
	statusColumn string
	ownerColumn  bool
	orgColumn    bool
}

func (s *pharmaJSONAggregateStore[T]) Create(ctx context.Context, item *T) error {
	values, err := s.values(item)
	if err != nil {
		return err
	}
	return s.db.WithContext(ctx).Table(s.table).Create(values).Error
}

func (s *pharmaJSONAggregateStore[T]) Upsert(ctx context.Context, item *T) error {
	values, err := s.values(item)
	if err != nil {
		return err
	}
	id := values["id"]
	result := s.db.WithContext(ctx).Table(s.table).Where("id = ?", id).Updates(values)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return s.db.WithContext(ctx).Table(s.table).Create(values).Error
	}
	return nil
}

func (s *pharmaJSONAggregateStore[T]) Get(ctx context.Context, id shared.ID) (*T, error) {
	return s.get(ctx, id.String())
}

func (s *pharmaJSONAggregateStore[T]) get(ctx context.Context, id string) (*T, error) {
	var row struct{ PayloadJSON string }
	err := s.db.WithContext(ctx).Table(s.table).Select("payload_json").Where("id = ?", strings.TrimSpace(id)).Take(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return decodePayload[T](row.PayloadJSON)
}

func (s *pharmaJSONAggregateStore[T]) List(ctx context.Context, filter pharmaoarepo.ListFilter) ([]T, error) {
	query := s.db.WithContext(ctx).Table(s.table).Select("payload_json")
	if s.statusColumn != "" && strings.TrimSpace(filter.Status) != "" {
		query = query.Where(s.statusColumn+" = ?", strings.TrimSpace(filter.Status))
	}
	organizationIDs := filter.NormalizedOrganizationIDs()
	ownerID := filter.OwnerID.String()
	if filter.ScopeAny && (len(organizationIDs) > 0 || ownerID != "") {
		parts := make([]string, 0, 2)
		args := make([]any, 0, 2)
		if s.orgColumn && len(organizationIDs) > 0 {
			parts, args = append(parts, "organization_id IN ?"), append(args, organizationIDs)
		}
		if s.ownerColumn && ownerID != "" {
			parts, args = append(parts, "owner_id = ?"), append(args, ownerID)
		}
		if len(parts) > 0 {
			query = query.Where("("+strings.Join(parts, " OR ")+")", args...)
		}
	} else {
		if s.ownerColumn && ownerID != "" {
			query = query.Where("owner_id = ?", ownerID)
		}
		if s.orgColumn && len(organizationIDs) > 0 {
			query = query.Where("organization_id IN ?", organizationIDs)
		}
	}
	if keyword := strings.TrimSpace(filter.Keyword); keyword != "" {
		query = query.Where("LOWER(payload_json) LIKE ?", "%"+strings.ToLower(keyword)+"%")
	}
	if filter.Offset > 0 {
		query = query.Offset(filter.Offset)
	}
	if filter.Limit > 0 {
		query = query.Limit(filter.Limit)
	}
	var rows []struct{ PayloadJSON string }
	if err := query.Order("created_at DESC").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]T, 0, len(rows))
	for _, row := range rows {
		item, err := decodePayload[T](row.PayloadJSON)
		if err != nil {
			return nil, err
		}
		out = append(out, *item)
	}
	return out, nil
}

func (s *pharmaJSONAggregateStore[T]) values(item *T) (map[string]any, error) {
	if item == nil || strings.TrimSpace(s.id(item)) == "" {
		return nil, fmt.Errorf("aggregate id is required")
	}
	values, err := s.columns(item)
	if err != nil {
		return nil, err
	}
	values["id"] = s.id(item)
	payload, err := json.Marshal(item)
	if err != nil {
		return nil, err
	}
	values["payload_json"] = string(payload)
	return values, nil
}

func decodePayload[T any](payload string) (*T, error) {
	var item T
	if err := json.Unmarshal([]byte(payload), &item); err != nil {
		return nil, err
	}
	return &item, nil
}

type PharmaContractStore struct {
	*pharmaJSONAggregateStore[domainpharma.Contract]
}
type PharmaQualityComplaintStore struct {
	*pharmaJSONAggregateStore[domainpharma.QualityComplaint]
}
type PharmaDrugRecallStore struct {
	*pharmaJSONAggregateStore[domainpharma.DrugRecall]
}
type PharmaCustomerFollowUpStore struct {
	*pharmaJSONAggregateStore[domainpharma.CustomerFollowUp]
}
type PharmaSalesOpportunityStore struct {
	*pharmaJSONAggregateStore[domainpharma.SalesOpportunity]
}
type PharmaPaymentPlanStore struct {
	*pharmaJSONAggregateStore[domainpharma.PaymentPlan]
}
type PharmaInvoiceRecordStore struct {
	*pharmaJSONAggregateStore[domainpharma.InvoiceRecord]
}
type PharmaPaymentReminderJobStore struct {
	*pharmaJSONAggregateStore[domainpharma.PaymentReminderJob]
}

func NewPharmaContractStore(db *gorm.DB) *PharmaContractStore {
	return &PharmaContractStore{&pharmaJSONAggregateStore[domainpharma.Contract]{db: db, table: "pharma_oa_contracts", id: func(item *domainpharma.Contract) string { return item.ID.String() }, statusColumn: "status", ownerColumn: true, columns: func(item *domainpharma.Contract) (map[string]any, error) {
		return map[string]any{"number": item.Number, "title": item.Title, "party_type": item.PartyType, "party_id": item.PartyID, "status": item.Status, "workflow_instance_id": item.WorkflowInstanceID, "owner_id": item.OwnerID, "approver_id": item.ApproverID, "amount": item.Amount, "currency": item.Currency, "effective_at": item.EffectiveAt, "expires_at": item.ExpiresAt, "created_at": item.Meta.CreatedAt, "updated_at": item.Meta.UpdatedAt, "created_by": item.OwnerID, "updated_by": contractUpdatedBy(item)}, nil
	}}}
}

func NewPharmaQualityComplaintStore(db *gorm.DB) *PharmaQualityComplaintStore {
	return &PharmaQualityComplaintStore{&pharmaJSONAggregateStore[domainpharma.QualityComplaint]{db: db, table: "pharma_oa_quality_complaints", id: func(item *domainpharma.QualityComplaint) string { return item.ID.String() }, statusColumn: "status", columns: func(item *domainpharma.QualityComplaint) (map[string]any, error) {
		return map[string]any{"number": item.Number, "title": item.Title, "customer_id": item.CustomerID, "product_id": item.ProductID, "batch_id": item.BatchID, "status": item.Status, "workflow_instance_id": item.WorkflowInstanceID, "reporter_id": item.ReporterID, "handler_id": item.HandlerID, "created_at": item.Meta.CreatedAt, "updated_at": item.Meta.UpdatedAt, "created_by": item.ReporterID, "updated_by": complaintUpdatedBy(item)}, nil
	}}}
}

func NewPharmaDrugRecallStore(db *gorm.DB) *PharmaDrugRecallStore {
	return &PharmaDrugRecallStore{&pharmaJSONAggregateStore[domainpharma.DrugRecall]{db: db, table: "pharma_oa_drug_recalls", id: func(item *domainpharma.DrugRecall) string { return item.ID.String() }, statusColumn: "status", columns: func(item *domainpharma.DrugRecall) (map[string]any, error) {
		return map[string]any{"number": item.Number, "title": item.Title, "product_id": item.ProductID, "batch_id": item.BatchID, "source_complaint_id": item.SourceComplaintID, "status": item.Status, "initiated_by": item.InitiatedBy, "created_at": item.Meta.CreatedAt, "updated_at": item.Meta.UpdatedAt, "created_by": item.InitiatedBy, "updated_by": recallUpdatedBy(item)}, nil
	}}}
}

func NewPharmaCustomerFollowUpStore(db *gorm.DB) *PharmaCustomerFollowUpStore {
	return &PharmaCustomerFollowUpStore{&pharmaJSONAggregateStore[domainpharma.CustomerFollowUp]{db: db, table: "pharma_oa_customer_follow_ups", id: func(item *domainpharma.CustomerFollowUp) string { return item.ID.String() }, statusColumn: "status", ownerColumn: true, orgColumn: true, columns: func(item *domainpharma.CustomerFollowUp) (map[string]any, error) {
		return map[string]any{"customer_id": item.CustomerID, "customer_code": item.CustomerCode, "customer_name": item.CustomerName, "owner_id": item.OwnerID, "organization_id": item.OrganizationID, "status": item.Status, "scheduled_at": item.ScheduledAt, "created_at": item.CreatedAt, "updated_at": item.UpdatedAt, "created_by": item.OwnerID, "updated_by": item.OwnerID}, nil
	}}}
}

func NewPharmaSalesOpportunityStore(db *gorm.DB) *PharmaSalesOpportunityStore {
	return &PharmaSalesOpportunityStore{&pharmaJSONAggregateStore[domainpharma.SalesOpportunity]{db: db, table: "pharma_oa_sales_opportunities", id: func(item *domainpharma.SalesOpportunity) string { return item.ID.String() }, statusColumn: "stage", ownerColumn: true, orgColumn: true, columns: func(item *domainpharma.SalesOpportunity) (map[string]any, error) {
		return map[string]any{"title": item.Title, "customer_id": item.CustomerID, "customer_code": item.CustomerCode, "customer_name": item.CustomerName, "owner_id": item.OwnerID, "organization_id": item.OrganizationID, "stage": item.Stage, "expected_amount_cents": item.ExpectedAmountCents, "estimated_close_date": item.EstimatedCloseDate, "created_at": item.Meta.CreatedAt, "updated_at": item.Meta.UpdatedAt, "created_by": item.OwnerID, "updated_by": item.OwnerID}, nil
	}}}
}

func NewPharmaPaymentPlanStore(db *gorm.DB) *PharmaPaymentPlanStore {
	return &PharmaPaymentPlanStore{&pharmaJSONAggregateStore[domainpharma.PaymentPlan]{db: db, table: "pharma_oa_payment_plans", id: func(item *domainpharma.PaymentPlan) string { return item.ID.String() }, statusColumn: "status", ownerColumn: true, columns: func(item *domainpharma.PaymentPlan) (map[string]any, error) {
		return map[string]any{"sales_order_id": item.SalesOrderID, "sales_order_number": item.SalesOrderNumber, "customer_id": item.CustomerID, "owner_id": item.OwnerID, "order_total_cents": item.OrderTotalCents, "amount_cents": item.AmountCents, "paid_amount_cents": item.PaidAmountCents, "status": item.Status, "due_at": item.DueAt, "created_at": item.CreatedAt, "updated_at": item.UpdatedAt, "created_by": item.OwnerID, "updated_by": item.OwnerID}, nil
	}}}
}

func NewPharmaInvoiceRecordStore(db *gorm.DB) *PharmaInvoiceRecordStore {
	return &PharmaInvoiceRecordStore{&pharmaJSONAggregateStore[domainpharma.InvoiceRecord]{db: db, table: "pharma_oa_invoice_records", id: func(item *domainpharma.InvoiceRecord) string { return item.ID.String() }, statusColumn: "status", ownerColumn: true, columns: func(item *domainpharma.InvoiceRecord) (map[string]any, error) {
		updatedAt := item.CreatedAt
		if item.VoidedAt != nil {
			updatedAt = *item.VoidedAt
		}
		return map[string]any{"number": item.Number, "sales_order_id": item.SalesOrderID, "sales_order_number": item.SalesOrderNumber, "customer_id": item.CustomerID, "owner_id": item.OwnerID, "order_total_cents": item.OrderTotalCents, "amount_cents": item.AmountCents, "status": item.Status, "issued_at": item.IssuedAt, "voided_at": item.VoidedAt, "created_at": item.CreatedAt, "updated_at": updatedAt, "created_by": item.CreatedBy, "updated_by": invoiceUpdatedBy(item)}, nil
	}}}
}

func NewPharmaPaymentReminderJobStore(db *gorm.DB) *PharmaPaymentReminderJobStore {
	return &PharmaPaymentReminderJobStore{&pharmaJSONAggregateStore[domainpharma.PaymentReminderJob]{db: db, table: "pharma_oa_payment_reminder_jobs", id: func(item *domainpharma.PaymentReminderJob) string { return item.ID.String() }, statusColumn: "status", ownerColumn: true, columns: func(item *domainpharma.PaymentReminderJob) (map[string]any, error) {
		updatedAt := item.CreatedAt
		if item.CompletedAt != nil {
			updatedAt = *item.CompletedAt
		} else if item.StartedAt != nil {
			updatedAt = *item.StartedAt
		}
		return map[string]any{"status": item.Status, "recipient_id": item.RecipientID, "owner_id": item.OwnerID, "initiated_by": item.InitiatedBy, "last_run_by": item.LastRunBy, "retry_count": item.RetryCount, "error": item.Error, "started_at": item.StartedAt, "completed_at": item.CompletedAt, "created_at": item.CreatedAt, "updated_at": updatedAt, "created_by": item.InitiatedBy, "updated_by": item.LastRunBy}, nil
	}}}
}

type PharmaInventoryAlertStore struct {
	db     *gorm.DB
	jobs   *pharmaJSONAggregateStore[domainpharma.InventoryAlertJob]
	alerts *pharmaJSONAggregateStore[domainpharma.InventoryAlert]
}

func NewPharmaInventoryAlertStore(db *gorm.DB) *PharmaInventoryAlertStore {
	return &PharmaInventoryAlertStore{db: db,
		jobs:   &pharmaJSONAggregateStore[domainpharma.InventoryAlertJob]{db: db, table: "pharma_oa_inventory_alert_jobs", id: func(item *domainpharma.InventoryAlertJob) string { return item.ID.String() }, statusColumn: "status", columns: inventoryAlertJobColumns},
		alerts: &pharmaJSONAggregateStore[domainpharma.InventoryAlert]{db: db, table: "pharma_oa_inventory_alerts", id: func(item *domainpharma.InventoryAlert) string { return item.ID.String() }, statusColumn: "status", columns: inventoryAlertColumns},
	}
}

func (s *PharmaInventoryAlertStore) CreateJob(ctx context.Context, item *domainpharma.InventoryAlertJob) error {
	return s.jobs.Create(ctx, item)
}
func (s *PharmaInventoryAlertStore) UpsertJob(ctx context.Context, item *domainpharma.InventoryAlertJob) error {
	return s.jobs.Upsert(ctx, item)
}
func (s *PharmaInventoryAlertStore) GetJob(ctx context.Context, id shared.ID) (*domainpharma.InventoryAlertJob, error) {
	return s.jobs.Get(ctx, id)
}
func (s *PharmaInventoryAlertStore) ListJobs(ctx context.Context, filter pharmaoarepo.ListFilter) ([]domainpharma.InventoryAlertJob, error) {
	return s.jobs.List(ctx, filter)
}
func (s *PharmaInventoryAlertStore) UpsertAlert(ctx context.Context, item *domainpharma.InventoryAlert) error {
	return s.alerts.Upsert(ctx, item)
}
func (s *PharmaInventoryAlertStore) GetAlert(ctx context.Context, id shared.ID) (*domainpharma.InventoryAlert, error) {
	return s.alerts.Get(ctx, id)
}
func (s *PharmaInventoryAlertStore) ListAlerts(ctx context.Context, filter pharmaoarepo.ListFilter) ([]domainpharma.InventoryAlert, error) {
	return s.alerts.List(ctx, filter)
}

type PharmaReportExportJobStore struct {
	base *pharmaJSONAggregateStore[domainpharma.ReportExportJob]
}

func NewPharmaReportExportJobStore(db *gorm.DB) *PharmaReportExportJobStore {
	return &PharmaReportExportJobStore{base: &pharmaJSONAggregateStore[domainpharma.ReportExportJob]{db: db, table: "pharma_oa_report_export_jobs", id: func(item *domainpharma.ReportExportJob) string { return item.ID }, statusColumn: "status", ownerColumn: true, columns: reportExportJobColumns}}
}

func (s *PharmaReportExportJobStore) Create(ctx context.Context, item *domainpharma.ReportExportJob) error {
	return s.base.Create(ctx, item)
}
func (s *PharmaReportExportJobStore) Upsert(ctx context.Context, item *domainpharma.ReportExportJob) error {
	return s.base.Upsert(ctx, item)
}
func (s *PharmaReportExportJobStore) Get(ctx context.Context, id string) (*domainpharma.ReportExportJob, error) {
	return s.base.get(ctx, id)
}
func (s *PharmaReportExportJobStore) List(ctx context.Context, filter pharmaoarepo.ListFilter) ([]domainpharma.ReportExportJob, error) {
	return s.base.List(ctx, filter)
}
func (s *PharmaReportExportJobStore) GetByIdempotencyKey(ctx context.Context, key string) (*domainpharma.ReportExportJob, error) {
	var row struct{ PayloadJSON string }
	err := s.base.db.WithContext(ctx).Table(s.base.table).Select("payload_json").Where("idempotency_key = ?", strings.TrimSpace(key)).Take(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return decodePayload[domainpharma.ReportExportJob](row.PayloadJSON)
}

func inventoryAlertJobColumns(item *domainpharma.InventoryAlertJob) (map[string]any, error) {
	updatedAt := item.CreatedAt
	if item.CompletedAt != nil {
		updatedAt = *item.CompletedAt
	} else if item.StartedAt != nil {
		updatedAt = *item.StartedAt
	}
	return map[string]any{"status": item.Status, "idempotency_key": item.IdempotencyKey, "retry_count": item.RetryCount, "error": item.Error, "started_at": item.StartedAt, "completed_at": item.CompletedAt, "created_at": item.CreatedAt, "updated_at": updatedAt, "created_by": "system", "updated_by": "system"}, nil
}

func inventoryAlertColumns(item *domainpharma.InventoryAlert) (map[string]any, error) {
	return map[string]any{"type": item.Type, "status": item.Status, "balance_id": item.BalanceID, "product_id": item.ProductID, "warehouse_id": item.WarehouseID, "batch_id": item.BatchID, "recipient_id": item.RecipientID, "notification_id": item.NotificationID, "last_seen_at": item.LastSeenAt, "resolved_at": item.ResolvedAt, "created_at": item.FirstSeenAt, "updated_at": item.LastSeenAt, "created_by": "system", "updated_by": "system"}, nil
}

func reportExportJobColumns(item *domainpharma.ReportExportJob) (map[string]any, error) {
	updatedAt := item.CreatedAt
	if item.CompletedAt != nil {
		updatedAt = *item.CompletedAt
	} else if item.StartedAt != nil {
		updatedAt = *item.StartedAt
	}
	return map[string]any{"report_type": item.ReportType, "status": item.Status, "owner_id": item.OwnerID, "idempotency_key": item.IdempotencyKey, "retry_count": item.RetryCount, "file_id": item.FileID, "error": item.Error, "started_at": item.StartedAt, "completed_at": item.CompletedAt, "created_at": item.CreatedAt, "updated_at": updatedAt, "created_by": item.OwnerID, "updated_by": item.OwnerID}, nil
}

func contractUpdatedBy(item *domainpharma.Contract) string {
	if item.RejectedBy != "" {
		return item.RejectedBy
	}
	if item.ApprovedBy != "" {
		return item.ApprovedBy
	}
	return item.OwnerID
}
func complaintUpdatedBy(item *domainpharma.QualityComplaint) string {
	if item.RejectedBy != "" {
		return item.RejectedBy
	}
	if item.ResolvedBy != "" {
		return item.ResolvedBy
	}
	return item.ReporterID
}
func recallUpdatedBy(item *domainpharma.DrugRecall) string {
	if item.CompletedBy != "" {
		return item.CompletedBy
	}
	return item.InitiatedBy
}
func invoiceUpdatedBy(item *domainpharma.InvoiceRecord) string {
	if item.VoidedBy != "" {
		return item.VoidedBy
	}
	return item.CreatedBy
}
