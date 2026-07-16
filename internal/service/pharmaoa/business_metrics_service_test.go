package pharmaoa

import (
	"context"
	"fmt"
	"testing"
	"time"

	domainaudit "github.com/tinboxw/skoll/internal/domain/audit"
	domainpharma "github.com/tinboxw/skoll/internal/domain/pharmaoa"
	"github.com/tinboxw/skoll/internal/domain/shared"
)

type metricsInventoryFixture struct {
	items []*domainpharma.InventoryAlert
}

func (f metricsInventoryFixture) ListAlerts(context.Context, bool) ([]*domainpharma.InventoryAlert, error) {
	return f.items, nil
}

type metricsQualificationFixture struct{ items []QualificationRecord }

func (f metricsQualificationFixture) List(context.Context, QualificationListInput) ([]QualificationRecord, error) {
	return f.items, nil
}

type metricsApprovalFixture struct {
	items []*domainpharma.PurchaseRequest
}

func (f metricsApprovalFixture) ListRequests(context.Context) ([]*domainpharma.PurchaseRequest, error) {
	return f.items, nil
}

type metricsFollowUpFixture struct {
	items []*domainpharma.CustomerFollowUp
}

func (f metricsFollowUpFixture) List(context.Context, CustomerFollowUpListInput) ([]*domainpharma.CustomerFollowUp, error) {
	return f.items, nil
}

type metricsSalesFixture struct{ items []*domainpharma.SalesOrder }

func (f metricsSalesFixture) ListOrders(context.Context) ([]*domainpharma.SalesOrder, error) {
	return f.items, nil
}

type metricsAuditFixture struct {
	actor  string
	action string
}

func (f *metricsAuditFixture) Append(_ context.Context, actorID, action, _, _ string, _ map[string]any) (*domainaudit.Record, error) {
	f.actor, f.action = actorID, action
	return &domainaudit.Record{}, nil
}
func (*metricsAuditFixture) GetByID(context.Context, string) (*domainaudit.Record, error) {
	return nil, nil
}
func (*metricsAuditFixture) ListByActor(context.Context, string, int) ([]*domainaudit.Record, error) {
	return nil, nil
}
func (*metricsAuditFixture) ListByTimeRange(context.Context, time.Time, time.Time, int) ([]*domainaudit.Record, error) {
	return nil, nil
}
func (*metricsAuditFixture) ClearByTimeRange(context.Context, time.Time, time.Time) (int, error) {
	return 0, nil
}

func TestBusinessMetricsAggregatesBoundedSnapshot(t *testing.T) {
	now := time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC)
	created := now.Add(-48 * time.Hour)
	updated := created.Add(6 * time.Hour)
	service := NewBusinessMetricsService(
		metricsInventoryFixture{items: []*domainpharma.InventoryAlert{{Type: domainpharma.InventoryAlertLowStock, Status: domainpharma.InventoryAlertActive}, {Type: domainpharma.InventoryAlertNearExpiry, Status: domainpharma.InventoryAlertActive}, {Type: domainpharma.InventoryAlertOverStock, Status: domainpharma.InventoryAlertResolved}}},
		metricsQualificationFixture{items: []QualificationRecord{{Status: QualificationStatusExpired, SubjectType: QualificationSubjectSupplier, SubjectStatus: "active"}, {Status: QualificationStatusExpiring, SubjectType: QualificationSubjectCustomer, SubjectStatus: "active"}, {Status: QualificationStatusValid, SubjectType: QualificationSubjectEmployee, SubjectStatus: string(domainpharma.EmployeeStatusActive)}, {Status: QualificationStatusExpired, SubjectType: QualificationSubjectCustomer, SubjectStatus: "disabled"}}},
		metricsApprovalFixture{items: []*domainpharma.PurchaseRequest{{Status: domainpharma.PurchaseRequestApproved, Meta: shared.AuditMeta{CreatedAt: created, UpdatedAt: updated}}, {Status: domainpharma.PurchaseRequestPending, Meta: shared.AuditMeta{CreatedAt: now.Add(-time.Hour), UpdatedAt: now.Add(-time.Hour)}}}},
		metricsFollowUpFixture{items: []*domainpharma.CustomerFollowUp{{Status: domainpharma.CustomerFollowUpCompleted}, {Status: domainpharma.CustomerFollowUpPlanned, ScheduledAt: now.Add(-time.Hour)}, {Status: domainpharma.CustomerFollowUpCancelled}}},
		metricsSalesFixture{items: []*domainpharma.SalesOrder{{CreatedAt: created, TotalAmount: 12.34}, {CreatedAt: now.Add(-24 * time.Hour), TotalAmount: 20}}}, nil,
	)
	impl := service.(*businessMetricsService)
	impl.nowFn = func() time.Time { return now }
	snapshot, err := service.Get(context.Background(), BusinessMetricsInput{From: now.AddDate(0, 0, -6), To: now, Bucket: BusinessMetricsBucketDay, ActorID: "manager"})
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.StockAlerts.Active != 2 || snapshot.StockAlerts.LowStock != 1 || snapshot.StockAlerts.NearExpiry != 1 {
		t.Fatalf("unexpected alert metrics: %+v", snapshot.StockAlerts)
	}
	if snapshot.Qualifications.Expired != 1 || snapshot.Qualifications.Expiring != 1 || snapshot.Qualifications.Supplier != 1 || snapshot.Qualifications.Customer != 1 {
		t.Fatalf("unexpected qualification metrics: %+v", snapshot.Qualifications)
	}
	if snapshot.ApprovalEfficiency.Total != 2 || snapshot.ApprovalEfficiency.Approved != 1 || snapshot.ApprovalEfficiency.Pending != 1 || snapshot.ApprovalEfficiency.CompletionRate != 50 || snapshot.ApprovalEfficiency.AverageCompletionHours != 6 {
		t.Fatalf("unexpected approval metrics: %+v", snapshot.ApprovalEfficiency)
	}
	if snapshot.CustomerFollowUps.Total != 3 || snapshot.CustomerFollowUps.Completed != 1 || snapshot.CustomerFollowUps.Overdue != 1 || snapshot.CustomerFollowUps.CompletionRate != 33.33 {
		t.Fatalf("unexpected follow-up metrics: %+v", snapshot.CustomerFollowUps)
	}
	if snapshot.SalesTrend.OrderCount != 2 || snapshot.SalesTrend.AmountCents != 3234 || len(snapshot.SalesTrend.Series) != 7 {
		t.Fatalf("unexpected sales metrics: %+v", snapshot.SalesTrend)
	}
}

func TestBusinessMetricsRejectsUnboundedAndInvalidQueries(t *testing.T) {
	now := time.Now().UTC()
	cases := []BusinessMetricsInput{
		{From: now.AddDate(-2, 0, 0), To: now, ActorID: "manager"},
		{From: now, To: now.Add(-time.Hour), ActorID: "manager"},
		{From: now.Add(-time.Hour), To: now, Bucket: "quarter", ActorID: "manager"},
		{From: now.Add(-time.Hour), To: now, QualificationDays: 366, ActorID: "manager"},
		{From: now.Add(-time.Hour), To: now},
	}
	service := NewBusinessMetricsService(metricsInventoryFixture{}, metricsQualificationFixture{}, metricsApprovalFixture{}, metricsFollowUpFixture{}, metricsSalesFixture{}, nil)
	for i, in := range cases {
		if _, err := service.Get(context.Background(), in); err == nil {
			t.Fatalf("case %d expected validation error", i)
		}
	}
}

func TestBusinessMetricsAuditsAuthenticatedActor(t *testing.T) {
	audit := &metricsAuditFixture{}
	service := NewBusinessMetricsService(metricsInventoryFixture{}, metricsQualificationFixture{}, metricsApprovalFixture{}, metricsFollowUpFixture{}, metricsSalesFixture{}, audit)
	if _, err := service.Get(context.Background(), BusinessMetricsInput{ActorID: "manager-1"}); err != nil {
		t.Fatal(err)
	}
	if audit.actor != "manager-1" || audit.action != "pharma_oa.business_metrics.read" {
		t.Fatalf("unexpected audit evidence: %+v", audit)
	}
}

func TestBusinessMetricsPerformanceSample(t *testing.T) {
	if testing.Short() {
		t.Skip("performance sample")
	}
	now := time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC)
	orders := make([]*domainpharma.SalesOrder, 10000)
	for i := range orders {
		orders[i] = &domainpharma.SalesOrder{ID: shared.ID(fmt.Sprintf("order-%d", i)), CreatedAt: now.Add(-time.Duration(i%720) * time.Hour), TotalAmount: 19.99}
	}
	service := NewBusinessMetricsService(metricsInventoryFixture{}, metricsQualificationFixture{}, metricsApprovalFixture{}, metricsFollowUpFixture{}, metricsSalesFixture{items: orders}, nil)
	impl := service.(*businessMetricsService)
	impl.nowFn = func() time.Time { return now }
	started := time.Now()
	snapshot, err := service.Get(context.Background(), BusinessMetricsInput{From: now.AddDate(0, 0, -30), To: now, Bucket: BusinessMetricsBucketDay, ActorID: "manager"})
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.SalesTrend.OrderCount != len(orders) {
		t.Fatalf("expected %d orders, got %d", len(orders), snapshot.SalesTrend.OrderCount)
	}
	if elapsed := time.Since(started); elapsed > time.Second {
		t.Fatalf("10k-row aggregation took %s", elapsed)
	}
}
