package pharmaoa

import (
	"context"
	"fmt"
	"math"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	domainpharma "github.com/tinboxw/skoll/internal/domain/pharmaoa"
	"github.com/tinboxw/skoll/internal/domain/shared"
	pharmaoarepo "github.com/tinboxw/skoll/internal/repository/pharmaoa"
	auditsvc "github.com/tinboxw/skoll/internal/service/audit"
	notificationsvc "github.com/tinboxw/skoll/internal/service/notification"
)

type PaymentInvoiceSalesReader interface {
	GetOrder(context.Context, string) (*domainpharma.SalesOrder, error)
}

type PaymentInvoiceNotifier interface {
	Create(context.Context, notificationsvc.CreateInput) (notificationsvc.Item, error)
	Complete(context.Context, string, string) (notificationsvc.Item, error)
}

type PaymentInvoiceService interface {
	CreatePaymentPlan(context.Context, PaymentPlanCreateInput) (*domainpharma.PaymentPlan, error)
	ListPaymentPlans(context.Context, PaymentPlanListInput) ([]*domainpharma.PaymentPlan, error)
	RecordPayment(context.Context, string, PaymentRecordInput) (*domainpharma.PaymentPlan, error)
	CreateInvoice(context.Context, InvoiceCreateInput) (*domainpharma.InvoiceRecord, error)
	ListInvoices(context.Context, InvoiceListInput) ([]*domainpharma.InvoiceRecord, error)
	VoidInvoice(context.Context, string, InvoiceVoidInput) (*domainpharma.InvoiceRecord, error)
	RunOverdueScan(context.Context, PaymentOverdueScanInput) (*domainpharma.PaymentReminderJob, error)
	RetryOverdueScan(context.Context, string, string) (*domainpharma.PaymentReminderJob, error)
	ListReminderJobs(context.Context, PaymentFinanceAccessScope, string) ([]*domainpharma.PaymentReminderJob, error)
}

type PaymentFinanceAccessScope struct {
	OwnerID    string
	IncludeAll bool
}

type PaymentPlanCreateInput struct {
	SalesOrderID string
	AmountCents  int64
	DueAt        time.Time
	Note         string
	Attachments  []domainpharma.FinancialAttachment
	ActorID      string
	Scope        PaymentFinanceAccessScope
}

type PaymentPlanListInput struct {
	Keyword      string
	SalesOrderID string
	Status       domainpharma.PaymentPlanStatus
	ActorID      string
	Scope        PaymentFinanceAccessScope
}

type PaymentRecordInput struct {
	AmountCents int64
	PaidAt      time.Time
	Reference   string
	Attachments []domainpharma.FinancialAttachment
	ActorID     string
	Scope       PaymentFinanceAccessScope
}

type InvoiceCreateInput struct {
	Number       string
	SalesOrderID string
	AmountCents  int64
	IssuedAt     time.Time
	Note         string
	Attachments  []domainpharma.FinancialAttachment
	ActorID      string
	Scope        PaymentFinanceAccessScope
}

type InvoiceListInput struct {
	Keyword      string
	SalesOrderID string
	Status       domainpharma.InvoiceRecordStatus
	ActorID      string
	Scope        PaymentFinanceAccessScope
}

type InvoiceVoidInput struct {
	Reason  string
	ActorID string
	Scope   PaymentFinanceAccessScope
}

type PaymentOverdueScanInput struct {
	RecipientID string
	ActorID     string
	Scope       PaymentFinanceAccessScope
}

type PaymentInvoiceRepositories struct {
	Plans    pharmaoarepo.PaymentPlanRepository
	Invoices pharmaoarepo.InvoiceRecordRepository
	Jobs     pharmaoarepo.PaymentReminderJobRepository
}

type paymentInvoiceService struct {
	mu             sync.RWMutex
	opMu           sync.Mutex
	runMu          sync.Mutex
	sales          PaymentInvoiceSalesReader
	notifications  PaymentInvoiceNotifier
	audit          auditsvc.Service
	plans          map[string]*domainpharma.PaymentPlan
	invoices       map[string]*domainpharma.InvoiceRecord
	jobs           map[string]*domainpharma.PaymentReminderJob
	nowFn          func() time.Time
	planCounter    int64
	receiptCounter int64
	invoiceCounter int64
	jobCounter     int64
	planRepo       pharmaoarepo.PaymentPlanRepository
	invoiceRepo    pharmaoarepo.InvoiceRecordRepository
	jobRepo        pharmaoarepo.PaymentReminderJobRepository
}

func NewPaymentInvoiceService(sales PaymentInvoiceSalesReader, notifications PaymentInvoiceNotifier, audit auditsvc.Service, repositories ...PaymentInvoiceRepositories) PaymentInvoiceService {
	repos := PaymentInvoiceRepositories{Plans: pharmaoarepo.NewMemoryPaymentPlanRepository(), Invoices: pharmaoarepo.NewMemoryInvoiceRecordRepository(), Jobs: pharmaoarepo.NewMemoryPaymentReminderJobRepository()}
	if len(repositories) > 0 {
		if repositories[0].Plans != nil {
			repos.Plans = repositories[0].Plans
		}
		if repositories[0].Invoices != nil {
			repos.Invoices = repositories[0].Invoices
		}
		if repositories[0].Jobs != nil {
			repos.Jobs = repositories[0].Jobs
		}
	}
	return &paymentInvoiceService{
		sales: sales, notifications: notifications, audit: audit,
		plans: map[string]*domainpharma.PaymentPlan{}, invoices: map[string]*domainpharma.InvoiceRecord{}, jobs: map[string]*domainpharma.PaymentReminderJob{},
		planRepo: repos.Plans, invoiceRepo: repos.Invoices, jobRepo: repos.Jobs,
		nowFn: func() time.Time { return time.Now().UTC() },
	}
}

func (s *paymentInvoiceService) CreatePaymentPlan(ctx context.Context, in PaymentPlanCreateInput) (*domainpharma.PaymentPlan, error) {
	s.opMu.Lock()
	defer s.opMu.Unlock()
	if err := s.syncFromRepositories(ctx); err != nil {
		return nil, err
	}
	actorID, scope, err := normalizePaymentActorAndScope(in.ActorID, in.Scope)
	if err != nil {
		return nil, err
	}
	order, totalCents, err := s.authorizedOrder(ctx, in.SalesOrderID, scope)
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	planned := int64(0)
	for _, item := range s.plans {
		if item.SalesOrderID == order.ID.String() {
			planned += item.AmountCents
		}
	}
	if in.AmountCents <= 0 || planned+in.AmountCents > totalCents {
		s.mu.Unlock()
		return nil, fmt.Errorf("payment plans exceed the sales order total")
	}
	s.planCounter++
	id := shared.ID("payment-plan-" + strconv.FormatInt(s.planCounter, 10))
	s.mu.Unlock()
	item, err := domainpharma.NewPaymentPlan(id, order.ID.String(), order.Number, order.CustomerID, order.CreatedBy, totalCents, in.AmountCents, in.DueAt, in.Note, in.Attachments, s.nowFn())
	if err != nil {
		return nil, err
	}
	if err = s.planRepo.Create(ctx, item); err != nil {
		return nil, err
	}
	s.storePaymentPlan(item)
	s.appendAudit(ctx, actorID, "pharma_oa.payment_plan.create", "pharma_oa_payment_plan", item.ID.String(), map[string]any{"salesOrderId": item.SalesOrderID, "amountCents": item.AmountCents, "dueAt": item.DueAt})
	return clonePaymentPlan(item), nil
}

func (s *paymentInvoiceService) ListPaymentPlans(ctx context.Context, in PaymentPlanListInput) ([]*domainpharma.PaymentPlan, error) {
	if err := s.syncFromRepositories(ctx); err != nil {
		return nil, err
	}
	actorID, scope, err := normalizePaymentActorAndScope(in.ActorID, in.Scope)
	if err != nil {
		return nil, err
	}
	keyword := strings.ToLower(strings.TrimSpace(in.Keyword))
	orderID := strings.TrimSpace(in.SalesOrderID)
	s.mu.RLock()
	out := make([]*domainpharma.PaymentPlan, 0, len(s.plans))
	for _, item := range s.plans {
		if !paymentOwnerInScope(item.OwnerID, scope) || orderID != "" && item.SalesOrderID != orderID || in.Status != "" && item.Status != in.Status {
			continue
		}
		if keyword != "" && !strings.Contains(strings.ToLower(item.SalesOrderNumber+" "+item.CustomerID+" "+item.Note), keyword) {
			continue
		}
		out = append(out, clonePaymentPlan(item))
	}
	s.mu.RUnlock()
	sort.Slice(out, func(i, j int) bool {
		if out[i].DueAt.Equal(out[j].DueAt) {
			return out[i].ID.String() < out[j].ID.String()
		}
		return out[i].DueAt.Before(out[j].DueAt)
	})
	s.appendAudit(ctx, actorID, "pharma_oa.payment_plan.read_list", "pharma_oa_payment_plan", "payment-plans", map[string]any{"count": len(out), "salesOrderId": orderID, "status": in.Status})
	return out, nil
}

func (s *paymentInvoiceService) RecordPayment(ctx context.Context, id string, in PaymentRecordInput) (*domainpharma.PaymentPlan, error) {
	s.opMu.Lock()
	defer s.opMu.Unlock()
	if err := s.syncFromRepositories(ctx); err != nil {
		return nil, err
	}
	actorID, scope, err := normalizePaymentActorAndScope(in.ActorID, in.Scope)
	if err != nil {
		return nil, err
	}
	id = strings.TrimSpace(id)
	s.mu.Lock()
	current := s.plans[id]
	if current == nil || !paymentOwnerInScope(current.OwnerID, scope) {
		s.mu.Unlock()
		return nil, fmt.Errorf("payment plan not found or access denied")
	}
	s.receiptCounter++
	receiptID := shared.ID("payment-receipt-" + strconv.FormatInt(s.receiptCounter, 10))
	next := clonePaymentPlan(current)
	if err = next.Receive(receiptID, in.AmountCents, in.PaidAt, in.Reference, actorID, in.Attachments, s.nowFn()); err != nil {
		s.mu.Unlock()
		return nil, err
	}
	if next.Status == domainpharma.PaymentPlanPaid && next.NotificationID != "" && s.notifications != nil {
		if _, err = s.notifications.Complete(ctx, next.NotificationID, next.ReminderRecipientID); err != nil {
			s.mu.Unlock()
			return nil, err
		}
	}
	s.mu.Unlock()
	if err = s.planRepo.Upsert(ctx, next); err != nil {
		return nil, err
	}
	s.storePaymentPlan(next)
	s.appendAudit(ctx, actorID, "pharma_oa.payment_plan.receive", "pharma_oa_payment_plan", id, map[string]any{"receiptId": receiptID.String(), "amountCents": in.AmountCents, "paidAmountCents": next.PaidAmountCents, "status": next.Status})
	return clonePaymentPlan(next), nil
}

func (s *paymentInvoiceService) CreateInvoice(ctx context.Context, in InvoiceCreateInput) (*domainpharma.InvoiceRecord, error) {
	s.opMu.Lock()
	defer s.opMu.Unlock()
	if err := s.syncFromRepositories(ctx); err != nil {
		return nil, err
	}
	actorID, scope, err := normalizePaymentActorAndScope(in.ActorID, in.Scope)
	if err != nil {
		return nil, err
	}
	order, totalCents, err := s.authorizedOrder(ctx, in.SalesOrderID, scope)
	if err != nil {
		return nil, err
	}
	number := strings.TrimSpace(in.Number)
	s.mu.Lock()
	invoiced := int64(0)
	for _, item := range s.invoices {
		if strings.EqualFold(item.Number, number) {
			s.mu.Unlock()
			return nil, fmt.Errorf("invoice number already exists")
		}
		if item.SalesOrderID == order.ID.String() && item.Status == domainpharma.InvoiceRecordIssued {
			invoiced += item.AmountCents
		}
	}
	if in.AmountCents <= 0 || invoiced+in.AmountCents > totalCents {
		s.mu.Unlock()
		return nil, fmt.Errorf("issued invoices exceed the sales order total")
	}
	s.invoiceCounter++
	id := shared.ID("invoice-record-" + strconv.FormatInt(s.invoiceCounter, 10))
	s.mu.Unlock()
	item, err := domainpharma.NewInvoiceRecord(id, number, order.ID.String(), order.Number, order.CustomerID, order.CreatedBy, actorID, totalCents, in.AmountCents, in.IssuedAt, in.Note, in.Attachments, s.nowFn())
	if err != nil {
		return nil, err
	}
	if err = s.invoiceRepo.Create(ctx, item); err != nil {
		return nil, err
	}
	s.storeInvoice(item)
	s.appendAudit(ctx, actorID, "pharma_oa.invoice_record.create", "pharma_oa_invoice_record", item.ID.String(), map[string]any{"salesOrderId": item.SalesOrderID, "number": item.Number, "amountCents": item.AmountCents})
	return cloneInvoiceRecord(item), nil
}

func (s *paymentInvoiceService) ListInvoices(ctx context.Context, in InvoiceListInput) ([]*domainpharma.InvoiceRecord, error) {
	if err := s.syncFromRepositories(ctx); err != nil {
		return nil, err
	}
	actorID, scope, err := normalizePaymentActorAndScope(in.ActorID, in.Scope)
	if err != nil {
		return nil, err
	}
	keyword := strings.ToLower(strings.TrimSpace(in.Keyword))
	orderID := strings.TrimSpace(in.SalesOrderID)
	s.mu.RLock()
	out := make([]*domainpharma.InvoiceRecord, 0, len(s.invoices))
	for _, item := range s.invoices {
		if !paymentOwnerInScope(item.OwnerID, scope) || orderID != "" && item.SalesOrderID != orderID || in.Status != "" && item.Status != in.Status {
			continue
		}
		if keyword != "" && !strings.Contains(strings.ToLower(item.Number+" "+item.SalesOrderNumber+" "+item.CustomerID+" "+item.Note), keyword) {
			continue
		}
		out = append(out, cloneInvoiceRecord(item))
	}
	s.mu.RUnlock()
	sort.Slice(out, func(i, j int) bool { return out[i].IssuedAt.After(out[j].IssuedAt) })
	s.appendAudit(ctx, actorID, "pharma_oa.invoice_record.read_list", "pharma_oa_invoice_record", "invoice-records", map[string]any{"count": len(out), "salesOrderId": orderID, "status": in.Status})
	return out, nil
}

func (s *paymentInvoiceService) VoidInvoice(ctx context.Context, id string, in InvoiceVoidInput) (*domainpharma.InvoiceRecord, error) {
	s.opMu.Lock()
	defer s.opMu.Unlock()
	if err := s.syncFromRepositories(ctx); err != nil {
		return nil, err
	}
	actorID, scope, err := normalizePaymentActorAndScope(in.ActorID, in.Scope)
	if err != nil {
		return nil, err
	}
	id = strings.TrimSpace(id)
	s.mu.Lock()
	current := s.invoices[id]
	if current == nil || !paymentOwnerInScope(current.OwnerID, scope) {
		s.mu.Unlock()
		return nil, fmt.Errorf("invoice record not found or access denied")
	}
	next := cloneInvoiceRecord(current)
	if err = next.Void(actorID, in.Reason, s.nowFn()); err != nil {
		s.mu.Unlock()
		return nil, err
	}
	s.mu.Unlock()
	if err = s.invoiceRepo.Upsert(ctx, next); err != nil {
		return nil, err
	}
	s.storeInvoice(next)
	s.appendAudit(ctx, actorID, "pharma_oa.invoice_record.void", "pharma_oa_invoice_record", id, map[string]any{"number": next.Number, "reason": next.VoidReason})
	return cloneInvoiceRecord(next), nil
}

func (s *paymentInvoiceService) RunOverdueScan(ctx context.Context, in PaymentOverdueScanInput) (*domainpharma.PaymentReminderJob, error) {
	if err := s.syncFromRepositories(ctx); err != nil {
		return nil, err
	}
	actorID, scope, err := normalizePaymentActorAndScope(in.ActorID, in.Scope)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(in.RecipientID) == "" {
		return nil, fmt.Errorf("recipientId is required")
	}
	s.mu.Lock()
	s.jobCounter++
	id := shared.ID("payment-reminder-job-" + strconv.FormatInt(s.jobCounter, 10))
	s.mu.Unlock()
	job, err := domainpharma.NewPaymentReminderJob(id, in.RecipientID, scope.OwnerID, actorID, scope.IncludeAll, s.nowFn())
	if err != nil {
		return nil, err
	}
	if err = s.jobRepo.Create(ctx, job); err != nil {
		return nil, err
	}
	s.storePaymentJob(job)
	return s.executeOverdueScan(ctx, job, actorID, false)
}

func (s *paymentInvoiceService) RetryOverdueScan(ctx context.Context, id, actorID string) (*domainpharma.PaymentReminderJob, error) {
	if err := s.syncFromRepositories(ctx); err != nil {
		return nil, err
	}
	actorID = strings.TrimSpace(actorID)
	if actorID == "" {
		return nil, fmt.Errorf("actorId is required")
	}
	s.mu.RLock()
	job := clonePaymentReminderJob(s.jobs[strings.TrimSpace(id)])
	s.mu.RUnlock()
	if job == nil {
		return nil, fmt.Errorf("payment reminder job not found")
	}
	if !job.IncludeAll && job.OwnerID != actorID {
		return nil, fmt.Errorf("payment reminder job access denied")
	}
	return s.executeOverdueScan(ctx, job, actorID, true)
}

func (s *paymentInvoiceService) ListReminderJobs(ctx context.Context, scope PaymentFinanceAccessScope, actorID string) ([]*domainpharma.PaymentReminderJob, error) {
	if err := s.syncFromRepositories(ctx); err != nil {
		return nil, err
	}
	actorID, scope, err := normalizePaymentActorAndScope(actorID, scope)
	if err != nil {
		return nil, err
	}
	s.mu.RLock()
	out := make([]*domainpharma.PaymentReminderJob, 0, len(s.jobs))
	for _, item := range s.jobs {
		if scope.IncludeAll || item.OwnerID == scope.OwnerID {
			out = append(out, clonePaymentReminderJob(item))
		}
	}
	s.mu.RUnlock()
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	s.appendAudit(ctx, actorID, "pharma_oa.payment_reminder.job_read", "pharma_oa_payment_reminder_job", "payment-reminder-jobs", map[string]any{"count": len(out)})
	return out, nil
}

func (s *paymentInvoiceService) executeOverdueScan(ctx context.Context, job *domainpharma.PaymentReminderJob, actorID string, retry bool) (*domainpharma.PaymentReminderJob, error) {
	s.runMu.Lock()
	defer s.runMu.Unlock()
	if err := s.syncFromRepositories(ctx); err != nil {
		return nil, err
	}
	stored := s.jobs[job.ID.String()]
	if stored == nil {
		return nil, fmt.Errorf("payment reminder job not found")
	}
	job = clonePaymentReminderJob(stored)
	if err := job.Start(actorID, retry, s.nowFn()); err != nil {
		return nil, err
	}
	if err := s.savePaymentJob(ctx, job); err != nil {
		return nil, err
	}
	if s.notifications == nil {
		return s.failPaymentJob(ctx, job, fmt.Errorf("notification service is required"), retry)
	}
	now := s.nowFn()
	s.mu.RLock()
	ids := make([]string, 0, len(s.plans))
	for id, item := range s.plans {
		if paymentOwnerInScope(item.OwnerID, PaymentFinanceAccessScope{OwnerID: job.OwnerID, IncludeAll: job.IncludeAll}) && item.Status != domainpharma.PaymentPlanPaid && item.DueAt.Before(now) {
			ids = append(ids, id)
		}
	}
	s.mu.RUnlock()
	sort.Strings(ids)
	created := 0
	for _, id := range ids {
		s.mu.Lock()
		plan := clonePaymentPlan(s.plans[id])
		plan.MarkOverdue(now)
		if plan.NotificationID == "" {
			notificationID := "payment-overdue-" + plan.ID.String()
			_, err := s.notifications.Create(ctx, notificationsvc.CreateInput{
				ID: notificationID, Category: notificationsvc.CategoryReminder, ActorID: job.RecipientID,
				Title:  "Payment overdue: " + plan.SalesOrderNumber,
				Body:   fmt.Sprintf("%s is overdue with %d cents outstanding.", plan.SalesOrderNumber, plan.AmountCents-plan.PaidAmountCents),
				Target: notificationsvc.Target{Type: "payment_plan", ID: plan.ID.String(), Path: paymentPlanTargetPath(plan.ID.String())}, DueAt: plan.DueAt,
			})
			if err != nil {
				s.mu.Unlock()
				return s.failPaymentJob(ctx, job, err, retry)
			}
			plan.NotificationID = notificationID
			plan.ReminderRecipientID = job.RecipientID
			value := now.UTC()
			plan.RemindedAt = &value
			created++
			s.appendAudit(ctx, actorID, "pharma_oa.payment_reminder.create", "pharma_oa_payment_plan", plan.ID.String(), map[string]any{"recipientId": job.RecipientID, "notificationId": notificationID})
		}
		s.mu.Unlock()
		if err := s.planRepo.Upsert(ctx, plan); err != nil {
			return s.failPaymentJob(ctx, job, err, retry)
		}
		s.storePaymentPlan(plan)
	}
	job.Succeed(len(ids), created, s.nowFn())
	if err := s.savePaymentJob(ctx, job); err != nil {
		return nil, err
	}
	s.appendAudit(ctx, actorID, "pharma_oa.payment_reminder.run", "pharma_oa_payment_reminder_job", job.ID.String(), map[string]any{"matchedCount": len(ids), "createdCount": created, "retry": retry})
	return clonePaymentReminderJob(job), nil
}

func (s *paymentInvoiceService) authorizedOrder(ctx context.Context, id string, scope PaymentFinanceAccessScope) (*domainpharma.SalesOrder, int64, error) {
	if s == nil || s.sales == nil {
		return nil, 0, fmt.Errorf("sales service is required")
	}
	order, err := s.sales.GetOrder(ctx, strings.TrimSpace(id))
	if err != nil {
		return nil, 0, err
	}
	if !paymentOwnerInScope(order.CreatedBy, scope) {
		return nil, 0, fmt.Errorf("sales order access denied")
	}
	totalCents := int64(math.Round(order.TotalAmount * 100))
	if totalCents <= 0 {
		return nil, 0, fmt.Errorf("sales order total must be positive")
	}
	return order, totalCents, nil
}

func (s *paymentInvoiceService) failPaymentJob(ctx context.Context, job *domainpharma.PaymentReminderJob, cause error, retry bool) (*domainpharma.PaymentReminderJob, error) {
	job.Fail(cause, s.nowFn())
	if err := s.savePaymentJob(ctx, job); err != nil {
		return clonePaymentReminderJob(job), err
	}
	s.appendAudit(ctx, job.LastRunBy, "pharma_oa.payment_reminder.fail", "pharma_oa_payment_reminder_job", job.ID.String(), map[string]any{"error": job.Error, "retry": retry, "retryCount": job.RetryCount})
	return clonePaymentReminderJob(job), cause
}

func (s *paymentInvoiceService) savePaymentJob(ctx context.Context, job *domainpharma.PaymentReminderJob) error {
	if err := s.jobRepo.Upsert(ctx, job); err != nil {
		return err
	}
	s.storePaymentJob(job)
	return nil
}

func (s *paymentInvoiceService) storePaymentJob(job *domainpharma.PaymentReminderJob) {
	s.mu.Lock()
	s.jobs[job.ID.String()] = clonePaymentReminderJob(job)
	s.mu.Unlock()
}

func (s *paymentInvoiceService) storePaymentPlan(item *domainpharma.PaymentPlan) {
	s.mu.Lock()
	s.plans[item.ID.String()] = clonePaymentPlan(item)
	s.mu.Unlock()
}
func (s *paymentInvoiceService) storeInvoice(item *domainpharma.InvoiceRecord) {
	s.mu.Lock()
	s.invoices[item.ID.String()] = cloneInvoiceRecord(item)
	s.mu.Unlock()
}

func (s *paymentInvoiceService) syncFromRepositories(ctx context.Context) error {
	plans, err := s.planRepo.List(ctx, pharmaoarepo.ListFilter{})
	if err != nil {
		return err
	}
	invoices, err := s.invoiceRepo.List(ctx, pharmaoarepo.ListFilter{})
	if err != nil {
		return err
	}
	jobs, err := s.jobRepo.List(ctx, pharmaoarepo.ListFilter{})
	if err != nil {
		return err
	}
	planMap := make(map[string]*domainpharma.PaymentPlan, len(plans))
	invoiceMap := make(map[string]*domainpharma.InvoiceRecord, len(invoices))
	jobMap := make(map[string]*domainpharma.PaymentReminderJob, len(jobs))
	var planCounter, receiptCounter, invoiceCounter, jobCounter int64
	for index := range plans {
		item := clonePaymentPlan(&plans[index])
		planMap[item.ID.String()] = item
		if value := sequenceFromID(item.ID.String(), "payment-plan-"); value > planCounter {
			planCounter = value
		}
		for _, receipt := range item.Receipts {
			if value := sequenceFromID(receipt.ID.String(), "payment-receipt-"); value > receiptCounter {
				receiptCounter = value
			}
		}
	}
	for index := range invoices {
		item := cloneInvoiceRecord(&invoices[index])
		invoiceMap[item.ID.String()] = item
		if value := sequenceFromID(item.ID.String(), "invoice-record-"); value > invoiceCounter {
			invoiceCounter = value
		}
	}
	for index := range jobs {
		item := clonePaymentReminderJob(&jobs[index])
		jobMap[item.ID.String()] = item
		if value := sequenceFromID(item.ID.String(), "payment-reminder-job-"); value > jobCounter {
			jobCounter = value
		}
	}
	s.mu.Lock()
	s.plans, s.invoices, s.jobs = planMap, invoiceMap, jobMap
	s.planCounter, s.receiptCounter, s.invoiceCounter, s.jobCounter = planCounter, receiptCounter, invoiceCounter, jobCounter
	s.mu.Unlock()
	return nil
}

func (s *paymentInvoiceService) appendAudit(ctx context.Context, actorID, action, resource, id string, detail map[string]any) {
	if s.audit == nil {
		return
	}
	_, _ = s.audit.Append(ctx, actorID, action, resource, id, detail)
}

func normalizePaymentActorAndScope(actorID string, scope PaymentFinanceAccessScope) (string, PaymentFinanceAccessScope, error) {
	actorID = strings.TrimSpace(actorID)
	if actorID == "" {
		return "", scope, fmt.Errorf("actorId is required")
	}
	scope.OwnerID = strings.TrimSpace(scope.OwnerID)
	if !scope.IncludeAll && scope.OwnerID == "" {
		scope.OwnerID = actorID
	}
	return actorID, scope, nil
}

func paymentOwnerInScope(ownerID string, scope PaymentFinanceAccessScope) bool {
	return scope.IncludeAll || strings.TrimSpace(ownerID) != "" && strings.TrimSpace(ownerID) == scope.OwnerID
}

func paymentPlanTargetPath(id string) string {
	query := url.Values{}
	query.Set("planId", id)
	return "/skoll/pharma-oa/payment-invoices?" + query.Encode()
}

func clonePaymentPlan(item *domainpharma.PaymentPlan) *domainpharma.PaymentPlan {
	if item == nil {
		return nil
	}
	out := *item
	out.Attachments = append([]domainpharma.FinancialAttachment{}, item.Attachments...)
	out.Receipts = append([]domainpharma.PaymentReceipt{}, item.Receipts...)
	for index := range out.Receipts {
		out.Receipts[index].Attachments = append([]domainpharma.FinancialAttachment{}, item.Receipts[index].Attachments...)
	}
	if item.RemindedAt != nil {
		value := *item.RemindedAt
		out.RemindedAt = &value
	}
	return &out
}

func cloneInvoiceRecord(item *domainpharma.InvoiceRecord) *domainpharma.InvoiceRecord {
	if item == nil {
		return nil
	}
	out := *item
	out.Attachments = append([]domainpharma.FinancialAttachment{}, item.Attachments...)
	if item.VoidedAt != nil {
		value := *item.VoidedAt
		out.VoidedAt = &value
	}
	return &out
}

func clonePaymentReminderJob(item *domainpharma.PaymentReminderJob) *domainpharma.PaymentReminderJob {
	if item == nil {
		return nil
	}
	out := *item
	out.Logs = append([]domainpharma.PaymentReminderJobLog{}, item.Logs...)
	if item.StartedAt != nil {
		value := *item.StartedAt
		out.StartedAt = &value
	}
	if item.CompletedAt != nil {
		value := *item.CompletedAt
		out.CompletedAt = &value
	}
	return &out
}
