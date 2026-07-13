package pharmaoa

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	domainpharma "github.com/tinboxw/skoll/internal/domain/pharmaoa"
	auditsvc "github.com/tinboxw/skoll/internal/service/audit"
	notificationsvc "github.com/tinboxw/skoll/internal/service/notification"
	"github.com/tinboxw/skoll/internal/store/clickhouse"
)

func TestInventoryAlertJobEmitsDeduplicatedNotifications(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 7, 13, 8, 0, 0, 0, time.UTC)
	inventory := NewInventoryService(nil)
	seedAlertStock(t, ctx, inventory, "near", 50, now.AddDate(0, 0, 10))
	seedAlertStock(t, ctx, inventory, "low", 2, now.AddDate(1, 0, 0))
	seedAlertStock(t, ctx, inventory, "over", 200, now.AddDate(1, 0, 0))
	notifications := notificationsvc.NewService(func() time.Time { return now }, nil)
	auditService := auditsvc.NewService(clickhouse.NewAuditStore())
	service := NewInventoryAlertService(inventory, notifications, auditService)
	impl := service.(*inventoryAlertService)
	impl.nowFn = func() time.Time { return now }
	policy := domainpharma.InventoryAlertPolicy{NearExpiryDays: 30, LowStockThreshold: 5, OverStockThreshold: 100, RecipientID: "inventory-manager"}

	job, err := service.Run(ctx, policy)
	if err != nil {
		t.Fatalf("run inventory alerts: %v", err)
	}
	if job.Status != domainpharma.InventoryAlertJobSucceeded || job.MatchedCount != 3 || job.CreatedCount != 3 || len(job.Logs) < 3 {
		t.Fatalf("unexpected alert job: %+v", job)
	}
	alerts, _ := service.ListAlerts(ctx, true)
	if len(alerts) != 3 {
		t.Fatalf("expected three active alerts, got %+v", alerts)
	}
	kinds := map[domainpharma.InventoryAlertType]bool{}
	for _, alert := range alerts {
		kinds[alert.Type] = true
		if alert.NotificationID == "" || !strings.HasPrefix(alert.TargetPath, "/skoll/pharma-oa/warehouses?") || !strings.Contains(alert.TargetPath, "balanceId=") {
			t.Fatalf("alert is not actionable: %+v", alert)
		}
	}
	if !kinds[domainpharma.InventoryAlertNearExpiry] || !kinds[domainpharma.InventoryAlertLowStock] || !kinds[domainpharma.InventoryAlertOverStock] {
		t.Fatalf("missing alert types: %+v", kinds)
	}
	items, _ := notifications.List(ctx, notificationsvc.Filter{ActorID: "inventory-manager", Category: notificationsvc.CategoryReminder, Status: notificationsvc.StatusPending})
	if len(items) != 3 {
		t.Fatalf("alerts did not enter notification center: %+v", items)
	}
	records, err := auditService.ListByActor(ctx, "system", 20)
	if err != nil {
		t.Fatalf("list inventory alert audit: %v", err)
	}
	actions := map[string]int{}
	for _, record := range records {
		actions[record.Action]++
	}
	if actions["pharma_oa.alert.create"] != 3 || actions["pharma_oa.alert.run"] != 1 {
		t.Fatalf("inventory alert audit is incomplete: %+v", actions)
	}

	again, err := service.Run(ctx, policy)
	if err != nil || again.CreatedCount != 0 || again.MatchedCount != 3 {
		t.Fatalf("duplicate scan must be idempotent: %+v %v", again, err)
	}
	items, _ = notifications.List(ctx, notificationsvc.Filter{ActorID: "inventory-manager"})
	if len(items) != 3 {
		t.Fatalf("duplicate scan created notifications: %+v", items)
	}
}

func TestInventoryAlertJobResolvesClearedCondition(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 7, 13, 8, 0, 0, 0, time.UTC)
	inventory := NewInventoryService(nil)
	seed := seedAlertStock(t, ctx, inventory, "low-resolve", 2, now.AddDate(1, 0, 0))
	notifications := notificationsvc.NewService(func() time.Time { return now }, nil)
	service := NewInventoryAlertService(inventory, notifications, nil)
	impl := service.(*inventoryAlertService)
	impl.nowFn = func() time.Time { return now }
	policy := domainpharma.InventoryAlertPolicy{NearExpiryDays: 30, LowStockThreshold: 5, OverStockThreshold: 100, RecipientID: "inventory-manager"}
	if _, err := service.Run(ctx, policy); err != nil {
		t.Fatalf("first scan: %v", err)
	}
	_, err := inventory.Inbound(ctx, StockMovementInput{ReferenceID: "replenish", ProductID: seed.Balance.ProductID, WarehouseID: seed.Balance.WarehouseID, AreaID: seed.Balance.AreaID, LocationID: seed.Balance.LocationID, BatchID: seed.Balance.BatchID, Quantity: 8})
	if err != nil {
		t.Fatalf("replenish stock: %v", err)
	}
	now = now.Add(time.Hour)
	job, err := service.Run(ctx, policy)
	if err != nil || job.ResolvedCount != 1 {
		t.Fatalf("resolve scan: %+v %v", job, err)
	}
	active, _ := service.ListAlerts(ctx, true)
	all, _ := service.ListAlerts(ctx, false)
	if len(active) != 0 || len(all) != 1 || all[0].Status != domainpharma.InventoryAlertResolved || all[0].ResolvedAt == nil {
		t.Fatalf("alert was not resolved: active=%+v all=%+v", active, all)
	}
	items, _ := notifications.List(ctx, notificationsvc.Filter{ActorID: "inventory-manager", Status: notificationsvc.StatusDone})
	if len(items) != 1 {
		t.Fatalf("resolved alert notification was not completed: %+v", items)
	}
}

func TestInventoryAlertFailedJobCanRetry(t *testing.T) {
	ctx := context.Background()
	reader := &failingInventoryAlertReader{err: errors.New("inventory unavailable")}
	notifications := notificationsvc.NewService(nil, nil)
	service := NewInventoryAlertService(reader, notifications, nil)
	policy := domainpharma.InventoryAlertPolicy{NearExpiryDays: 30, LowStockThreshold: 5, OverStockThreshold: 100, RecipientID: "inventory-manager"}
	failed, err := service.Run(ctx, policy)
	if err == nil || failed.Status != domainpharma.InventoryAlertJobFailed || !strings.Contains(failed.Error, "inventory unavailable") {
		t.Fatalf("expected observable failed job, got %+v %v", failed, err)
	}
	reader.err = nil
	retried, err := service.Retry(ctx, failed.ID.String())
	if err != nil || retried.Status != domainpharma.InventoryAlertJobSucceeded || retried.RetryCount != 1 {
		t.Fatalf("retry failed: %+v %v", retried, err)
	}
	if _, err = service.Retry(ctx, failed.ID.String()); err == nil {
		t.Fatal("succeeded job must not retry")
	}
}

type failingInventoryAlertReader struct{ err error }

func (r *failingInventoryAlertReader) ListBalances(context.Context) ([]domainpharma.StockBalance, error) {
	if r.err != nil {
		return nil, r.err
	}
	return nil, nil
}

func (r *failingInventoryAlertReader) ListBatches(context.Context) ([]domainpharma.StockBatch, error) {
	return nil, nil
}

func seedAlertStock(t *testing.T, ctx context.Context, inventory InventoryService, suffix string, quantity int, expiresAt time.Time) StockMovementResult {
	t.Helper()
	result, err := inventory.Inbound(ctx, StockMovementInput{ReferenceID: "seed-" + suffix, ProductID: "product-" + suffix, WarehouseID: "warehouse-" + suffix, AreaID: "area-" + suffix, LocationID: "location-" + suffix, BatchNo: "batch-" + suffix, ProductionDate: expiresAt.AddDate(-1, 0, 0), ExpiresAt: expiresAt, Quantity: quantity, ActorID: "inventory-manager"})
	if err != nil {
		t.Fatalf("seed %s stock: %v", suffix, err)
	}
	return result
}
