package pharmaoa

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	domainpharma "github.com/tinboxw/skoll/internal/domain/pharmaoa"
	"github.com/tinboxw/skoll/internal/domain/shared"
	auditsvc "github.com/tinboxw/skoll/internal/service/audit"
	notificationsvc "github.com/tinboxw/skoll/internal/service/notification"
	"github.com/tinboxw/skoll/internal/store/clickhouse"
)

func TestPaymentInvoiceLifecycleAndSalesOrderLimits(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 7, 16, 9, 0, 0, 0, time.UTC)
	reader := paymentOrderFixture("order-1", "SO-001", "customer-1", "sales-1", 1000)
	service := NewPaymentInvoiceService(reader, notificationsvc.NewService(notificationsvc.NewMemoryRepository(), nil, nil), nil)
	impl := service.(*paymentInvoiceService)
	impl.nowFn = func() time.Time { return now }

	first, err := service.CreatePaymentPlan(ctx, PaymentPlanCreateInput{SalesOrderID: "order-1", AmountCents: 60000, DueAt: now.AddDate(0, 0, 10), ActorID: "sales-1"})
	if err != nil {
		t.Fatal(err)
	}
	if first.OrderTotalCents != 100000 || first.Status != domainpharma.PaymentPlanPending || first.Attachments == nil || first.Receipts == nil {
		t.Fatalf("payment plan snapshot mismatch: %+v", first)
	}
	if _, err = service.CreatePaymentPlan(ctx, PaymentPlanCreateInput{SalesOrderID: "order-1", AmountCents: 50000, DueAt: now.AddDate(0, 0, 20), ActorID: "sales-1"}); err == nil {
		t.Fatal("payment plans exceeded the sales order total")
	}
	if _, err = service.CreatePaymentPlan(ctx, PaymentPlanCreateInput{SalesOrderID: "order-1", AmountCents: 40000, DueAt: now, ActorID: "sales-2"}); err == nil {
		t.Fatal("another sales owner created a payment plan")
	}

	partial, err := service.RecordPayment(ctx, first.ID.String(), PaymentRecordInput{AmountCents: 20000, PaidAt: now, Reference: "BANK-1", ActorID: "sales-1"})
	if err != nil || partial.Status != domainpharma.PaymentPlanPartial || partial.PaidAmountCents != 20000 || len(partial.Receipts) != 1 {
		t.Fatalf("partial payment mismatch: %+v %v", partial, err)
	}
	if _, err = service.RecordPayment(ctx, first.ID.String(), PaymentRecordInput{AmountCents: 50000, PaidAt: now, ActorID: "sales-1"}); err == nil {
		t.Fatal("receipt exceeded the remaining payment balance")
	}
	paid, err := service.RecordPayment(ctx, first.ID.String(), PaymentRecordInput{AmountCents: 40000, PaidAt: now, Reference: "BANK-2", ActorID: "sales-1"})
	if err != nil || paid.Status != domainpharma.PaymentPlanPaid || paid.PaidAmountCents != paid.AmountCents {
		t.Fatalf("full payment mismatch: %+v %v", paid, err)
	}

	invoice, err := service.CreateInvoice(ctx, InvoiceCreateInput{Number: "INV-001", SalesOrderID: "order-1", AmountCents: 70000, IssuedAt: now, ActorID: "sales-1", Attachments: []domainpharma.FinancialAttachment{{FileID: "file-1", FileName: "invoice.pdf", Size: 120}}})
	if err != nil || invoice.Status != domainpharma.InvoiceRecordIssued || len(invoice.Attachments) != 1 {
		t.Fatalf("invoice create mismatch: %+v %v", invoice, err)
	}
	if _, err = service.CreateInvoice(ctx, InvoiceCreateInput{Number: "INV-002", SalesOrderID: "order-1", AmountCents: 40000, IssuedAt: now, ActorID: "sales-1"}); err == nil {
		t.Fatal("invoices exceeded the sales order total")
	}
	voided, err := service.VoidInvoice(ctx, invoice.ID.String(), InvoiceVoidInput{Reason: "Incorrect tax code", ActorID: "sales-1"})
	if err != nil || voided.Status != domainpharma.InvoiceRecordVoided || voided.VoidedAt == nil {
		t.Fatalf("invoice void mismatch: %+v %v", voided, err)
	}
	if _, err = service.CreateInvoice(ctx, InvoiceCreateInput{Number: "INV-003", SalesOrderID: "order-1", AmountCents: 100000, IssuedAt: now, ActorID: "sales-1"}); err != nil {
		t.Fatalf("voided invoice must release the order amount: %v", err)
	}
	if _, err = service.CreateInvoice(ctx, InvoiceCreateInput{Number: "INV-UNSAFE", SalesOrderID: "order-1", AmountCents: 1, IssuedAt: now, ActorID: "sales-1", Attachments: []domainpharma.FinancialAttachment{{FileID: "file-x", FileName: "payload.exe"}}}); err == nil {
		t.Fatal("unsafe financial attachment was accepted")
	}
}

func TestPaymentOverdueReminderIsIdempotentAndCompletesOnReceipt(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 7, 16, 9, 0, 0, 0, time.UTC)
	notifications := notificationsvc.NewService(notificationsvc.NewMemoryRepository(), func() time.Time { return now }, nil)
	service := NewPaymentInvoiceService(paymentOrderFixture("order-1", "SO-001", "customer-1", "sales-1", 500), notifications, nil)
	impl := service.(*paymentInvoiceService)
	impl.nowFn = func() time.Time { return now }
	plan, err := service.CreatePaymentPlan(ctx, PaymentPlanCreateInput{SalesOrderID: "order-1", AmountCents: 50000, DueAt: now.Add(-time.Hour), ActorID: "sales-1"})
	if err != nil {
		t.Fatal(err)
	}
	job, err := service.RunOverdueScan(ctx, PaymentOverdueScanInput{RecipientID: "finance-1", ActorID: "sales-1"})
	if err != nil || job.Status != domainpharma.PaymentReminderJobSucceeded || job.MatchedCount != 1 || job.CreatedCount != 1 {
		t.Fatalf("overdue scan mismatch: %+v %v", job, err)
	}
	plans, _ := service.ListPaymentPlans(ctx, PaymentPlanListInput{ActorID: "sales-1"})
	if len(plans) != 1 || plans[0].Status != domainpharma.PaymentPlanOverdue || plans[0].NotificationID == "" || plans[0].RemindedAt == nil {
		t.Fatalf("overdue plan mismatch: %+v", plans)
	}
	again, err := service.RunOverdueScan(ctx, PaymentOverdueScanInput{RecipientID: "finance-1", ActorID: "sales-1"})
	if err != nil || again.MatchedCount != 1 || again.CreatedCount != 0 {
		t.Fatalf("duplicate scan was not idempotent: %+v %v", again, err)
	}
	items, _ := notifications.List(ctx, notificationsvc.Filter{ActorID: "finance-1", Status: notificationsvc.StatusPending})
	if len(items) != 1 || !strings.Contains(items[0].Target.Path, "planId=") {
		t.Fatalf("notification trace mismatch: %+v", items)
	}
	if _, err = service.RecordPayment(ctx, plan.ID.String(), PaymentRecordInput{AmountCents: 50000, PaidAt: now, ActorID: "sales-1"}); err != nil {
		t.Fatal(err)
	}
	items, _ = notifications.List(ctx, notificationsvc.Filter{ActorID: "finance-1", Status: notificationsvc.StatusDone})
	if len(items) != 1 {
		t.Fatalf("paid plan did not complete its reminder: %+v", items)
	}
}

func TestPaymentOverdueFailedJobCanRetryAndAuditsActions(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 7, 16, 9, 0, 0, 0, time.UTC)
	notifier := &togglePaymentNotifier{err: errors.New("notification unavailable")}
	audit := auditsvc.NewService(clickhouse.NewAuditStore())
	service := NewPaymentInvoiceService(paymentOrderFixture("order-1", "SO-001", "customer-1", "sales-1", 500), notifier, audit)
	impl := service.(*paymentInvoiceService)
	impl.nowFn = func() time.Time { return now }
	_, err := service.CreatePaymentPlan(ctx, PaymentPlanCreateInput{SalesOrderID: "order-1", AmountCents: 50000, DueAt: now.Add(-time.Hour), ActorID: "sales-1"})
	if err != nil {
		t.Fatal(err)
	}
	failed, err := service.RunOverdueScan(ctx, PaymentOverdueScanInput{RecipientID: "finance-1", ActorID: "sales-1"})
	if err == nil || failed.Status != domainpharma.PaymentReminderJobFailed || !strings.Contains(failed.Error, "notification unavailable") {
		t.Fatalf("failed job is not observable: %+v %v", failed, err)
	}
	notifier.err = nil
	retried, err := service.RetryOverdueScan(ctx, failed.ID.String(), "sales-1")
	if err != nil || retried.Status != domainpharma.PaymentReminderJobSucceeded || retried.RetryCount != 1 || retried.CreatedCount != 1 {
		t.Fatalf("job retry mismatch: %+v %v", retried, err)
	}
	if _, err = service.RetryOverdueScan(ctx, failed.ID.String(), "sales-1"); err == nil {
		t.Fatal("succeeded job retried")
	}
	records, err := audit.ListByActor(ctx, "sales-1", 20)
	if err != nil {
		t.Fatal(err)
	}
	actions := map[string]bool{}
	for _, record := range records {
		actions[record.Action] = true
	}
	for _, action := range []string{"pharma_oa.payment_plan.create", "pharma_oa.payment_reminder.fail", "pharma_oa.payment_reminder.create", "pharma_oa.payment_reminder.run"} {
		if !actions[action] {
			t.Fatalf("audit action %q missing from %+v", action, actions)
		}
	}
}

type paymentOrderReader struct{ item *domainpharma.SalesOrder }

func paymentOrderFixture(id, number, customerID, ownerID string, total float64) paymentOrderReader {
	return paymentOrderReader{item: &domainpharma.SalesOrder{ID: shared.ID(id), Number: number, CustomerID: customerID, TotalAmount: total, CreatedBy: ownerID, Status: domainpharma.SalesOrderOpen}}
}

func (r paymentOrderReader) GetOrder(_ context.Context, id string) (*domainpharma.SalesOrder, error) {
	if r.item == nil || r.item.ID.String() != id {
		return nil, errors.New("sales order not found")
	}
	out := *r.item
	return &out, nil
}

type togglePaymentNotifier struct {
	err   error
	items map[string]notificationsvc.Item
}

func (n *togglePaymentNotifier) Create(_ context.Context, in notificationsvc.CreateInput) (notificationsvc.Item, error) {
	if n.err != nil {
		return notificationsvc.Item{}, n.err
	}
	if n.items == nil {
		n.items = map[string]notificationsvc.Item{}
	}
	item := notificationsvc.Item{ID: in.ID, ActorID: in.ActorID, Status: notificationsvc.StatusPending, Target: in.Target}
	n.items[item.ID] = item
	return item, nil
}

func (n *togglePaymentNotifier) Complete(_ context.Context, id, _ string) (notificationsvc.Item, error) {
	if n.err != nil {
		return notificationsvc.Item{}, n.err
	}
	item := n.items[id]
	item.Status = notificationsvc.StatusDone
	n.items[id] = item
	return item, nil
}
