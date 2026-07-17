package pharmaoa

import (
	"context"
	"fmt"
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

type InventoryAlertStockReader interface {
	ListBalances(ctx context.Context) ([]domainpharma.StockBalance, error)
	ListBatches(ctx context.Context) ([]domainpharma.StockBatch, error)
}

type InventoryAlertService interface {
	Run(ctx context.Context, policy domainpharma.InventoryAlertPolicy) (*domainpharma.InventoryAlertJob, error)
	Retry(ctx context.Context, jobID string) (*domainpharma.InventoryAlertJob, error)
	ListJobs(ctx context.Context) ([]*domainpharma.InventoryAlertJob, error)
	ListAlerts(ctx context.Context, activeOnly bool) ([]*domainpharma.InventoryAlert, error)
}

type inventoryAlertService struct {
	mu            sync.RWMutex
	runMu         sync.Mutex
	stock         InventoryAlertStockReader
	notifications *notificationsvc.Service
	audit         auditsvc.Service
	jobs          map[string]*domainpharma.InventoryAlertJob
	alerts        map[string]*domainpharma.InventoryAlert
	nowFn         func() time.Time
	jobCounter    int64
	repo          pharmaoarepo.InventoryAlertRepository
}

func NewInventoryAlertService(stock InventoryAlertStockReader, notifications *notificationsvc.Service, audit auditsvc.Service, repositories ...pharmaoarepo.InventoryAlertRepository) InventoryAlertService {
	repo := pharmaoarepo.InventoryAlertRepository(pharmaoarepo.NewMemoryInventoryAlertRepository())
	if len(repositories) > 0 && repositories[0] != nil {
		repo = repositories[0]
	}
	return &inventoryAlertService{stock: stock, notifications: notifications, audit: audit, jobs: map[string]*domainpharma.InventoryAlertJob{}, alerts: map[string]*domainpharma.InventoryAlert{}, repo: repo, nowFn: func() time.Time { return time.Now().UTC() }}
}

func (s *inventoryAlertService) Run(ctx context.Context, policy domainpharma.InventoryAlertPolicy) (*domainpharma.InventoryAlertJob, error) {
	if s == nil || s.stock == nil || s.notifications == nil {
		return nil, fmt.Errorf("inventory alert dependencies are required")
	}
	if err := policy.Validate(); err != nil {
		return nil, err
	}
	s.runMu.Lock()
	defer s.runMu.Unlock()
	if err := s.syncFromRepository(ctx); err != nil {
		return nil, err
	}
	s.mu.Lock()
	s.jobCounter++
	id := shared.ID("inventory-alert-job-" + strconv.FormatInt(s.jobCounter, 10))
	job, err := domainpharma.NewInventoryAlertJob(id, policy, s.nowFn())
	s.mu.Unlock()
	if err != nil {
		return nil, err
	}
	job.IdempotencyKey = "inventory-alert:" + id.String()
	if err = s.repo.CreateJob(ctx, job); err != nil {
		return nil, err
	}
	s.storeJobCached(job)
	return s.execute(ctx, job, false)
}

func (s *inventoryAlertService) Retry(ctx context.Context, jobID string) (*domainpharma.InventoryAlertJob, error) {
	if s == nil || s.stock == nil || s.notifications == nil {
		return nil, fmt.Errorf("inventory alert dependencies are required")
	}
	s.runMu.Lock()
	defer s.runMu.Unlock()
	if err := s.syncFromRepository(ctx); err != nil {
		return nil, err
	}
	s.mu.RLock()
	job := cloneInventoryAlertJob(s.jobs[strings.TrimSpace(jobID)])
	s.mu.RUnlock()
	if job == nil {
		return nil, fmt.Errorf("inventory alert job not found")
	}
	if job.Status != domainpharma.InventoryAlertJobFailed {
		return nil, fmt.Errorf("only failed inventory alert jobs can be retried")
	}
	job.RetryCount++
	return s.execute(ctx, job, true)
}

func (s *inventoryAlertService) execute(ctx context.Context, job *domainpharma.InventoryAlertJob, retry bool) (*domainpharma.InventoryAlertJob, error) {
	now := s.nowFn()
	if err := job.Start(now); err != nil {
		return nil, err
	}
	if err := s.saveJob(ctx, job); err != nil {
		return nil, err
	}
	balances, err := s.stock.ListBalances(ctx)
	if err != nil {
		return s.failJob(ctx, job, err, retry)
	}
	batches, err := s.stock.ListBatches(ctx)
	if err != nil {
		return s.failJob(ctx, job, err, retry)
	}
	batchByID := make(map[string]domainpharma.StockBatch, len(batches))
	for _, batch := range batches {
		batchByID[batch.ID.String()] = batch
	}
	observed := map[string]struct{}{}
	created := 0
	for _, balance := range balances {
		batch, ok := batchByID[balance.BatchID]
		if !ok {
			return s.failJob(ctx, job, fmt.Errorf("stock batch %s not found", balance.BatchID), retry)
		}
		conditions := evaluateInventoryAlertConditions(balance, batch, job.Policy, now)
		for _, condition := range conditions {
			key := inventoryAlertKey(condition.kind, balance.ID.String())
			observed[key] = struct{}{}
			wasCreated, upsertErr := s.upsertAlert(ctx, balance, batch, condition, job.Policy.RecipientID, now)
			if upsertErr != nil {
				return s.failJob(ctx, job, upsertErr, retry)
			}
			if wasCreated {
				created++
			}
		}
	}
	resolved, err := s.resolveMissingAlerts(ctx, observed, now)
	if err != nil {
		return s.failJob(ctx, job, err, retry)
	}
	if err = job.Succeed(len(observed), created, resolved, s.nowFn()); err != nil {
		return s.failJob(ctx, job, err, retry)
	}
	if err := s.saveJob(ctx, job); err != nil {
		return nil, err
	}
	s.appendAudit(ctx, "system", "pharma_oa.alert.run", job.ID.String(), map[string]any{"status": job.Status, "matched": job.MatchedCount, "created": job.CreatedCount, "resolved": job.ResolvedCount, "retryCount": job.RetryCount})
	return cloneInventoryAlertJob(job), nil
}

type inventoryAlertCondition struct {
	kind      domainpharma.InventoryAlertType
	threshold int
}

func evaluateInventoryAlertConditions(balance domainpharma.StockBalance, batch domainpharma.StockBatch, policy domainpharma.InventoryAlertPolicy, now time.Time) []inventoryAlertCondition {
	out := make([]inventoryAlertCondition, 0, 2)
	if !batch.ExpiresAt.IsZero() && !batch.ExpiresAt.After(now.AddDate(0, 0, policy.NearExpiryDays)) {
		out = append(out, inventoryAlertCondition{kind: domainpharma.InventoryAlertNearExpiry, threshold: policy.NearExpiryDays})
	}
	if balance.AvailableQuantity() <= policy.LowStockThreshold {
		out = append(out, inventoryAlertCondition{kind: domainpharma.InventoryAlertLowStock, threshold: policy.LowStockThreshold})
	} else if balance.Quantity >= policy.OverStockThreshold {
		out = append(out, inventoryAlertCondition{kind: domainpharma.InventoryAlertOverStock, threshold: policy.OverStockThreshold})
	}
	return out
}

func (s *inventoryAlertService) upsertAlert(ctx context.Context, balance domainpharma.StockBalance, batch domainpharma.StockBatch, condition inventoryAlertCondition, recipient string, now time.Time) (bool, error) {
	key := inventoryAlertKey(condition.kind, balance.ID.String())
	s.mu.RLock()
	existing := cloneInventoryAlert(s.alerts[key])
	s.mu.RUnlock()
	if existing != nil && existing.Status == domainpharma.InventoryAlertActive {
		existing.Quantity = balance.Quantity
		existing.Threshold = condition.threshold
		existing.ExpiresAt = batch.ExpiresAt
		existing.LastSeenAt = now.UTC()
		s.mu.Lock()
		s.alerts[key] = existing
		s.mu.Unlock()
		if err := s.repo.UpsertAlert(ctx, existing); err != nil {
			return false, err
		}
		return false, nil
	}
	id := shared.ID("inventory-alert-" + string(condition.kind) + "-" + balance.ID.String())
	targetPath := inventoryAlertTargetPath(balance)
	title, body, dueAt := inventoryAlertMessage(condition, balance, batch, now)
	item, err := s.notifications.Create(ctx, notificationsvc.CreateInput{ID: id.String(), Category: notificationsvc.CategoryReminder, Title: title, Body: body, ActorID: recipient, Target: notificationsvc.Target{Type: "inventory_stock", ID: balance.ID.String(), Path: targetPath}, DueAt: dueAt})
	if err != nil {
		return false, err
	}
	alert := &domainpharma.InventoryAlert{ID: id, Type: condition.kind, Status: domainpharma.InventoryAlertActive, BalanceID: balance.ID.String(), ProductID: balance.ProductID, WarehouseID: balance.WarehouseID, AreaID: balance.AreaID, LocationID: balance.LocationID, BatchID: balance.BatchID, Quantity: balance.Quantity, Threshold: condition.threshold, ExpiresAt: batch.ExpiresAt, RecipientID: strings.TrimSpace(recipient), NotificationID: item.ID, TargetPath: targetPath, FirstSeenAt: now.UTC(), LastSeenAt: now.UTC()}
	s.mu.Lock()
	s.alerts[key] = alert
	s.mu.Unlock()
	if err = s.repo.UpsertAlert(ctx, alert); err != nil {
		return false, err
	}
	s.appendAudit(ctx, "system", "pharma_oa.alert.create", alert.ID.String(), map[string]any{"type": alert.Type, "balanceId": alert.BalanceID, "recipientId": alert.RecipientID})
	return true, nil
}

func (s *inventoryAlertService) resolveMissingAlerts(ctx context.Context, observed map[string]struct{}, now time.Time) (int, error) {
	s.mu.RLock()
	active := make(map[string]*domainpharma.InventoryAlert)
	for key, alert := range s.alerts {
		if alert.Status == domainpharma.InventoryAlertActive {
			active[key] = cloneInventoryAlert(alert)
		}
	}
	s.mu.RUnlock()
	resolved := 0
	for key, alert := range active {
		if _, ok := observed[key]; ok {
			continue
		}
		if _, err := s.notifications.Complete(ctx, alert.NotificationID, alert.RecipientID); err != nil {
			return resolved, err
		}
		resolvedAt := now.UTC()
		alert.Status = domainpharma.InventoryAlertResolved
		alert.ResolvedAt = &resolvedAt
		alert.LastSeenAt = resolvedAt
		s.mu.Lock()
		s.alerts[key] = alert
		s.mu.Unlock()
		if err := s.repo.UpsertAlert(ctx, alert); err != nil {
			return resolved, err
		}
		resolved++
		s.appendAudit(ctx, "system", "pharma_oa.alert.resolve", alert.ID.String(), map[string]any{"type": alert.Type, "balanceId": alert.BalanceID})
	}
	return resolved, nil
}

func inventoryAlertMessage(condition inventoryAlertCondition, balance domainpharma.StockBalance, batch domainpharma.StockBatch, now time.Time) (string, string, time.Time) {
	switch condition.kind {
	case domainpharma.InventoryAlertNearExpiry:
		return "Inventory batch near expiry", fmt.Sprintf("Batch %s for product %s expires on %s.", batch.BatchNo, balance.ProductID, batch.ExpiresAt.UTC().Format("2006-01-02")), batch.ExpiresAt.UTC()
	case domainpharma.InventoryAlertLowStock:
		return "Low inventory", fmt.Sprintf("Product %s has %d available units; threshold is %d.", balance.ProductID, balance.AvailableQuantity(), condition.threshold), now.UTC()
	default:
		return "Inventory above limit", fmt.Sprintf("Product %s has %d units; threshold is %d.", balance.ProductID, balance.Quantity, condition.threshold), now.UTC()
	}
}

func inventoryAlertTargetPath(balance domainpharma.StockBalance) string {
	query := url.Values{}
	query.Set("warehouseId", balance.WarehouseID)
	query.Set("balanceId", balance.ID.String())
	query.Set("batchId", balance.BatchID)
	return "/skoll/pharma-oa/warehouses?" + query.Encode()
}

func inventoryAlertKey(kind domainpharma.InventoryAlertType, balanceID string) string {
	return string(kind) + "|" + strings.TrimSpace(balanceID)
}

func (s *inventoryAlertService) failJob(ctx context.Context, job *domainpharma.InventoryAlertJob, cause error, retry bool) (*domainpharma.InventoryAlertJob, error) {
	job.Fail(cause, s.nowFn())
	if err := s.saveJob(ctx, job); err != nil {
		return cloneInventoryAlertJob(job), err
	}
	s.appendAudit(ctx, "system", "pharma_oa.alert.fail", job.ID.String(), map[string]any{"error": job.Error, "retry": retry, "retryCount": job.RetryCount})
	return cloneInventoryAlertJob(job), cause
}

func (s *inventoryAlertService) saveJob(ctx context.Context, job *domainpharma.InventoryAlertJob) error {
	if err := s.repo.UpsertJob(ctx, job); err != nil {
		return err
	}
	s.storeJobCached(job)
	return nil
}
func (s *inventoryAlertService) storeJobCached(job *domainpharma.InventoryAlertJob) {
	s.mu.Lock()
	s.jobs[job.ID.String()] = cloneInventoryAlertJob(job)
	s.mu.Unlock()
}

func (s *inventoryAlertService) ListJobs(ctx context.Context) ([]*domainpharma.InventoryAlertJob, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := s.syncFromRepository(ctx); err != nil {
		return nil, err
	}
	s.mu.RLock()
	out := make([]*domainpharma.InventoryAlertJob, 0, len(s.jobs))
	for _, job := range s.jobs {
		out = append(out, cloneInventoryAlertJob(job))
	}
	s.mu.RUnlock()
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out, nil
}

func (s *inventoryAlertService) ListAlerts(ctx context.Context, activeOnly bool) ([]*domainpharma.InventoryAlert, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := s.syncFromRepository(ctx); err != nil {
		return nil, err
	}
	s.mu.RLock()
	out := make([]*domainpharma.InventoryAlert, 0, len(s.alerts))
	for _, alert := range s.alerts {
		if activeOnly && alert.Status != domainpharma.InventoryAlertActive {
			continue
		}
		out = append(out, cloneInventoryAlert(alert))
	}
	s.mu.RUnlock()
	sort.Slice(out, func(i, j int) bool { return out[i].ID.String() < out[j].ID.String() })
	return out, nil
}

func (s *inventoryAlertService) syncFromRepository(ctx context.Context) error {
	jobs, err := s.repo.ListJobs(ctx, pharmaoarepo.ListFilter{})
	if err != nil {
		return err
	}
	alerts, err := s.repo.ListAlerts(ctx, pharmaoarepo.ListFilter{})
	if err != nil {
		return err
	}
	jobMap := make(map[string]*domainpharma.InventoryAlertJob, len(jobs))
	alertMap := make(map[string]*domainpharma.InventoryAlert, len(alerts))
	var counter int64
	for index := range jobs {
		item := cloneInventoryAlertJob(&jobs[index])
		jobMap[item.ID.String()] = item
		if value := sequenceFromID(item.ID.String(), "inventory-alert-job-"); value > counter {
			counter = value
		}
	}
	for index := range alerts {
		item := cloneInventoryAlert(&alerts[index])
		alertMap[inventoryAlertKey(item.Type, item.BalanceID)] = item
	}
	s.mu.Lock()
	s.jobs, s.alerts, s.jobCounter = jobMap, alertMap, counter
	s.mu.Unlock()
	return nil
}

func cloneInventoryAlert(alert *domainpharma.InventoryAlert) *domainpharma.InventoryAlert {
	if alert == nil {
		return nil
	}
	out := *alert
	if alert.ResolvedAt != nil {
		resolved := *alert.ResolvedAt
		out.ResolvedAt = &resolved
	}
	return &out
}

func cloneInventoryAlertJob(job *domainpharma.InventoryAlertJob) *domainpharma.InventoryAlertJob {
	if job == nil {
		return nil
	}
	out := *job
	out.Logs = append([]domainpharma.InventoryAlertJobLog(nil), job.Logs...)
	if job.StartedAt != nil {
		started := *job.StartedAt
		out.StartedAt = &started
	}
	if job.CompletedAt != nil {
		completed := *job.CompletedAt
		out.CompletedAt = &completed
	}
	return &out
}

func (s *inventoryAlertService) appendAudit(ctx context.Context, actorID, action, resourceID string, detail map[string]any) {
	if s.audit == nil {
		return
	}
	_, _ = s.audit.Append(ctx, actorID, action, "pharma_oa_inventory_alert", resourceID, detail)
}
