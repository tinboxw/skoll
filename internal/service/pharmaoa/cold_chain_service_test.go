package pharmaoa

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	domainpharma "github.com/tinboxw/skoll/internal/domain/pharmaoa"
	"github.com/tinboxw/skoll/internal/domain/shared"
	auditsvc "github.com/tinboxw/skoll/internal/service/audit"
	notificationsvc "github.com/tinboxw/skoll/internal/service/notification"
	"github.com/tinboxw/skoll/internal/store/clickhouse"
)

func TestColdChainScanCreatesAndResolvesBatchAnomaly(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 7, 13, 10, 0, 0, 0, time.UTC)
	inventory, warehouses, balanceID, batchID := seedColdChainContext(t, ctx, now)
	notifications := notificationsvc.NewService(func() time.Time { return now }, nil)
	auditService := auditsvc.NewService(clickhouse.NewAuditStore())
	service := NewColdChainService(inventory, warehouses, notifications, auditService)
	impl := service.(*coldChainService)
	impl.nowFn = func() time.Time { return now }

	contexts, err := service.ListContexts(ctx)
	if err != nil || len(contexts) != 1 || contexts[0].BatchID != batchID || contexts[0].MinCelsius != 2 || contexts[0].MaxCelsius != 8 {
		t.Fatalf("cold-chain context mismatch: %+v %v", contexts, err)
	}
	abnormal, err := service.CreateRecord(ctx, ColdChainRecordCreateInput{BalanceID: balanceID, TemperatureCelsius: 10, HumidityPercent: 80, Source: "sensor-1", RecordedAt: now, ActorID: "quality-user"})
	if err != nil || abnormal.BatchID != batchID {
		t.Fatalf("create abnormal record: %+v %v", abnormal, err)
	}
	policy := domainpharma.ColdChainScanPolicy{MinHumidityPercent: 30, MaxHumidityPercent: 70, RecipientID: "quality-manager"}
	job, err := service.Run(ctx, policy, "quality-user")
	if err != nil || job.Status != domainpharma.ColdChainJobSucceeded || job.MatchedCount != 1 || job.CreatedCount != 1 {
		t.Fatalf("run cold-chain scan: %+v %v", job, err)
	}
	anomalies, _ := service.ListAnomalies(ctx, true)
	if len(anomalies) != 1 || anomalies[0].Risk != domainpharma.ColdChainRiskHigh || anomalies[0].BatchID != batchID || len(anomalies[0].Reasons) != 2 || !strings.Contains(anomalies[0].TargetPath, "batchId=") {
		t.Fatalf("risk-dashboard anomaly mismatch: %+v", anomalies)
	}
	items, _ := notifications.List(ctx, notificationsvc.Filter{ActorID: "quality-manager", Status: notificationsvc.StatusPending})
	if len(items) != 1 {
		t.Fatalf("anomaly did not enter notification center: %+v", items)
	}
	again, err := service.Run(ctx, policy, "quality-user")
	if err != nil || again.CreatedCount != 0 || again.MatchedCount != 1 {
		t.Fatalf("duplicate cold-chain scan is not idempotent: %+v %v", again, err)
	}
	items, _ = notifications.List(ctx, notificationsvc.Filter{ActorID: "quality-manager"})
	if len(items) != 1 {
		t.Fatalf("duplicate scan created notification: %+v", items)
	}

	now = now.Add(time.Hour)
	if _, err = service.CreateRecord(ctx, ColdChainRecordCreateInput{BalanceID: balanceID, TemperatureCelsius: 5, HumidityPercent: 50, Source: "sensor-1", RecordedAt: now, ActorID: "quality-user"}); err != nil {
		t.Fatalf("create normal record: %v", err)
	}
	resolvedJob, err := service.Run(ctx, policy, "quality-user")
	if err != nil || resolvedJob.ResolvedCount != 1 || resolvedJob.MatchedCount != 0 {
		t.Fatalf("resolve cold-chain anomaly: %+v %v", resolvedJob, err)
	}
	active, _ := service.ListAnomalies(ctx, true)
	all, _ := service.ListAnomalies(ctx, false)
	if len(active) != 0 || len(all) != 1 || all[0].Status != domainpharma.ColdChainAnomalyResolved || all[0].ResolvedAt == nil {
		t.Fatalf("cold-chain anomaly was not resolved: active=%+v all=%+v", active, all)
	}
	items, _ = notifications.List(ctx, notificationsvc.Filter{ActorID: "quality-manager", Status: notificationsvc.StatusDone})
	if len(items) != 1 {
		t.Fatalf("resolved cold-chain reminder not completed: %+v", items)
	}
	records, _ := auditService.ListByTimeRange(ctx, time.Now().Add(-time.Hour), time.Now().Add(time.Hour), 50)
	actions := map[string]int{}
	for _, record := range records {
		actions[record.Action]++
	}
	if actions["pharma_oa.cold_chain.anomaly_create"] != 1 || actions["pharma_oa.cold_chain.anomaly_resolve"] != 1 || actions["pharma_oa.cold_chain.run"] != 3 {
		t.Fatalf("cold-chain audit is incomplete: %+v", actions)
	}
}

func TestColdChainRejectsUnrelatedOrUncontrolledStock(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 7, 13, 10, 0, 0, 0, time.UTC)
	inventory, warehouses, _, _ := seedColdChainContext(t, ctx, now)
	service := NewColdChainService(inventory, warehouses, notificationsvc.NewService(nil, nil), nil)
	if _, err := service.CreateRecord(ctx, ColdChainRecordCreateInput{BalanceID: "missing", TemperatureCelsius: 5, HumidityPercent: 50, Source: "manual", RecordedAt: now, ActorID: "quality-user"}); err == nil {
		t.Fatal("unrelated stock balance must fail")
	}

	uncontrolled := NewWarehouseService(nil)
	_, err := uncontrolled.Create(ctx, WarehouseWriteInput{Code: "AMBIENT", Name: "Ambient", Region: "East", Areas: []domainpharma.WarehouseArea{{ID: "area-u", Code: "AU", Name: "Ambient", Locations: []domainpharma.WarehouseLocation{{ID: "location-u", Code: "LU", Name: "Ambient"}}}}})
	if err != nil {
		t.Fatalf("create uncontrolled warehouse: %v", err)
	}
	contexts, err := NewColdChainService(inventory, uncontrolled, notificationsvc.NewService(nil, nil), nil).ListContexts(ctx)
	if err != nil || len(contexts) != 0 {
		t.Fatalf("uncontrolled stock must not be eligible: %+v %v", contexts, err)
	}
}

func TestColdChainFailedJobCanRetry(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 7, 13, 10, 0, 0, 0, time.UTC)
	inventory, warehouses, balanceID, _ := seedColdChainContext(t, ctx, now)
	notifier := &toggleColdChainNotifier{err: errors.New("notification unavailable")}
	service := NewColdChainService(inventory, warehouses, notifier, nil)
	impl := service.(*coldChainService)
	impl.nowFn = func() time.Time { return now }
	_, err := service.CreateRecord(ctx, ColdChainRecordCreateInput{BalanceID: balanceID, TemperatureCelsius: 12, HumidityPercent: 50, Source: "sensor-1", RecordedAt: now, ActorID: "quality-user"})
	if err != nil {
		t.Fatalf("create retry fixture: %v", err)
	}
	policy := domainpharma.ColdChainScanPolicy{MinHumidityPercent: 30, MaxHumidityPercent: 70, RecipientID: "quality-manager"}
	failed, err := service.Run(ctx, policy, "quality-user")
	if err == nil || failed.Status != domainpharma.ColdChainJobFailed || !strings.Contains(failed.Error, "notification unavailable") {
		t.Fatalf("expected observable failed job: %+v %v", failed, err)
	}
	notifier.err = nil
	retried, err := service.Retry(ctx, failed.ID.String(), "quality-user")
	if err != nil || retried.Status != domainpharma.ColdChainJobSucceeded || retried.RetryCount != 1 || retried.CreatedCount != 1 {
		t.Fatalf("retry cold-chain job: %+v %v", retried, err)
	}
	if _, err = service.Retry(ctx, failed.ID.String(), "quality-user"); err == nil {
		t.Fatal("succeeded cold-chain job must not retry")
	}
}

func TestColdChainScanDoesNotInheritPublicRecordLimit(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 7, 13, 10, 0, 0, 0, time.UTC)
	service := NewColdChainService(nil, nil, notificationsvc.NewService(nil, nil), nil)
	impl := service.(*coldChainService)
	impl.nowFn = func() time.Time { return now }
	for i := 0; i < 501; i++ {
		id := fmt.Sprintf("record-%d", i)
		balanceID := fmt.Sprintf("balance-%d", i)
		record, err := domainpharma.NewColdChainRecord(shared.ID(id), domainpharma.ColdChainRecordInput{BalanceID: balanceID, ProductID: "product", BatchID: "batch", BatchNo: "BATCH", WarehouseID: "warehouse", AreaID: "area", LocationID: fmt.Sprintf("location-%d", i), TemperatureCelsius: 10, HumidityPercent: 50, MinCelsius: 2, MaxCelsius: 8, Source: "fixture", RecordedBy: "quality-user", RecordedAt: now}, now)
		if err != nil {
			t.Fatalf("create record fixture %d: %v", i, err)
		}
		impl.records[id] = record
	}
	job, err := service.Run(ctx, domainpharma.ColdChainScanPolicy{MinHumidityPercent: 30, MaxHumidityPercent: 70, RecipientID: "quality-manager"}, "quality-user")
	if err != nil || job.MatchedCount != 501 || job.CreatedCount != 501 {
		t.Fatalf("full cold-chain scan was truncated: %+v %v", job, err)
	}
}

func seedColdChainContext(t *testing.T, ctx context.Context, now time.Time) (InventoryService, WarehouseService, string, string) {
	t.Helper()
	warehouses := NewWarehouseService(nil)
	warehouse, err := warehouses.Create(ctx, WarehouseWriteInput{Code: "COLD", Name: "Cold Warehouse", Region: "East", Temperature: domainpharma.WarehouseTemperature{Controlled: true, MinCelsius: 2, MaxCelsius: 8}, Areas: []domainpharma.WarehouseArea{{ID: "area-cold", Code: "COLD-A", Name: "Cold Area", Locations: []domainpharma.WarehouseLocation{{ID: "location-cold", Code: "COLD-L", Name: "Cold Location"}}}}})
	if err != nil {
		t.Fatalf("create cold-chain warehouse: %v", err)
	}
	inventory := NewInventoryService(nil)
	stock, err := inventory.Inbound(ctx, StockMovementInput{ReferenceID: "cold-seed", ProductID: "product-cold", WarehouseID: warehouse.ID.String(), AreaID: "area-cold", LocationID: "location-cold", BatchNo: "COLD-B-001", ProductionDate: now.AddDate(-1, 0, 0), ExpiresAt: now.AddDate(1, 0, 0), Quantity: 20, ActorID: "inventory-user"})
	if err != nil {
		t.Fatalf("seed cold-chain stock: %v", err)
	}
	return inventory, warehouses, stock.Balance.ID.String(), stock.Balance.BatchID
}

type toggleColdChainNotifier struct {
	err   error
	items map[string]notificationsvc.Item
}

func (n *toggleColdChainNotifier) Create(_ context.Context, in notificationsvc.CreateInput) (notificationsvc.Item, error) {
	if n.err != nil {
		return notificationsvc.Item{}, n.err
	}
	if n.items == nil {
		n.items = map[string]notificationsvc.Item{}
	}
	item := notificationsvc.Item{ID: in.ID, ActorID: in.ActorID, Status: notificationsvc.StatusPending}
	n.items[item.ID] = item
	return item, nil
}

func (n *toggleColdChainNotifier) Complete(_ context.Context, id string, _ string) (notificationsvc.Item, error) {
	if n.err != nil {
		return notificationsvc.Item{}, n.err
	}
	item := n.items[id]
	item.Status = notificationsvc.StatusDone
	n.items[id] = item
	return item, nil
}
