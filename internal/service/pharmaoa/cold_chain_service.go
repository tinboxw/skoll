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
	auditsvc "github.com/tinboxw/skoll/internal/service/audit"
	notificationsvc "github.com/tinboxw/skoll/internal/service/notification"
)

type ColdChainInventoryReader interface {
	ListBalances(ctx context.Context) ([]domainpharma.StockBalance, error)
	ListBatches(ctx context.Context) ([]domainpharma.StockBatch, error)
}

type ColdChainWarehouseReader interface {
	List(ctx context.Context, in WarehouseListInput) ([]*domainpharma.Warehouse, error)
}

type ColdChainNotifier interface {
	Create(ctx context.Context, in notificationsvc.CreateInput) (notificationsvc.Item, error)
	Complete(ctx context.Context, id string, actorID string) (notificationsvc.Item, error)
}

type ColdChainService interface {
	ListContexts(ctx context.Context) ([]ColdChainContext, error)
	CreateRecord(ctx context.Context, in ColdChainRecordCreateInput) (*domainpharma.ColdChainRecord, error)
	ListRecords(ctx context.Context, in ColdChainRecordListInput) ([]*domainpharma.ColdChainRecord, error)
	Run(ctx context.Context, policy domainpharma.ColdChainScanPolicy, actorID string) (*domainpharma.ColdChainJob, error)
	Retry(ctx context.Context, jobID string, actorID string) (*domainpharma.ColdChainJob, error)
	ListJobs(ctx context.Context) ([]*domainpharma.ColdChainJob, error)
	ListAnomalies(ctx context.Context, activeOnly bool) ([]*domainpharma.ColdChainAnomaly, error)
}

type ColdChainContext struct {
	BalanceID   string  `json:"balanceId"`
	ProductID   string  `json:"productId"`
	BatchID     string  `json:"batchId"`
	BatchNo     string  `json:"batchNo"`
	WarehouseID string  `json:"warehouseId"`
	AreaID      string  `json:"areaId"`
	LocationID  string  `json:"locationId"`
	Quantity    int     `json:"quantity"`
	MinCelsius  float64 `json:"minCelsius"`
	MaxCelsius  float64 `json:"maxCelsius"`
}

type ColdChainRecordCreateInput struct {
	BalanceID          string
	TemperatureCelsius float64
	HumidityPercent    float64
	Source             string
	RecordedAt         time.Time
	ActorID            string
}

type ColdChainRecordListInput struct {
	BatchID     string
	WarehouseID string
	Limit       int
}

type coldChainService struct {
	mu            sync.RWMutex
	runMu         sync.Mutex
	inventory     ColdChainInventoryReader
	warehouses    ColdChainWarehouseReader
	notifications ColdChainNotifier
	audit         auditsvc.Service
	records       map[string]*domainpharma.ColdChainRecord
	anomalies     map[string]*domainpharma.ColdChainAnomaly
	jobs          map[string]*domainpharma.ColdChainJob
	nowFn         func() time.Time
	recordCounter int64
	jobCounter    int64
}

func NewColdChainService(inventory ColdChainInventoryReader, warehouses ColdChainWarehouseReader, notifications ColdChainNotifier, audit auditsvc.Service) ColdChainService {
	return &coldChainService{inventory: inventory, warehouses: warehouses, notifications: notifications, audit: audit, records: map[string]*domainpharma.ColdChainRecord{}, anomalies: map[string]*domainpharma.ColdChainAnomaly{}, jobs: map[string]*domainpharma.ColdChainJob{}, nowFn: func() time.Time { return time.Now().UTC() }}
}

func (s *coldChainService) ListContexts(ctx context.Context) ([]ColdChainContext, error) {
	if s == nil || s.inventory == nil || s.warehouses == nil {
		return nil, fmt.Errorf("cold-chain inventory and warehouse dependencies are required")
	}
	balances, err := s.inventory.ListBalances(ctx)
	if err != nil {
		return nil, err
	}
	batches, err := s.inventory.ListBatches(ctx)
	if err != nil {
		return nil, err
	}
	warehouses, err := s.listWarehouses(ctx)
	if err != nil {
		return nil, err
	}
	batchByID := make(map[string]domainpharma.StockBatch, len(batches))
	for _, batch := range batches {
		batchByID[batch.ID.String()] = batch
	}
	warehouseByID := make(map[string]*domainpharma.Warehouse, len(warehouses))
	for _, warehouse := range warehouses {
		warehouseByID[warehouse.ID.String()] = warehouse
	}
	out := make([]ColdChainContext, 0, len(balances))
	for _, balance := range balances {
		if balance.Quantity <= 0 {
			continue
		}
		batch, ok := batchByID[balance.BatchID]
		if !ok || batch.ProductID != balance.ProductID {
			continue
		}
		temperature, ok := coldChainTemperature(warehouseByID[balance.WarehouseID], balance.AreaID, balance.LocationID)
		if !ok {
			continue
		}
		out = append(out, ColdChainContext{BalanceID: balance.ID.String(), ProductID: balance.ProductID, BatchID: balance.BatchID, BatchNo: batch.BatchNo, WarehouseID: balance.WarehouseID, AreaID: balance.AreaID, LocationID: balance.LocationID, Quantity: balance.Quantity, MinCelsius: temperature.MinCelsius, MaxCelsius: temperature.MaxCelsius})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].BatchNo == out[j].BatchNo {
			return out[i].BalanceID < out[j].BalanceID
		}
		return out[i].BatchNo < out[j].BatchNo
	})
	return out, nil
}

func (s *coldChainService) CreateRecord(ctx context.Context, in ColdChainRecordCreateInput) (*domainpharma.ColdChainRecord, error) {
	contexts, err := s.ListContexts(ctx)
	if err != nil {
		return nil, err
	}
	var selected *ColdChainContext
	for idx := range contexts {
		if contexts[idx].BalanceID == strings.TrimSpace(in.BalanceID) {
			selected = &contexts[idx]
			break
		}
	}
	if selected == nil {
		return nil, fmt.Errorf("cold-chain stock balance is not eligible")
	}
	s.mu.Lock()
	s.recordCounter++
	id := shared.ID("cold-chain-record-" + strconv.FormatInt(s.recordCounter, 10))
	s.mu.Unlock()
	record, err := domainpharma.NewColdChainRecord(id, domainpharma.ColdChainRecordInput{
		BalanceID: selected.BalanceID, ProductID: selected.ProductID, BatchID: selected.BatchID, BatchNo: selected.BatchNo,
		WarehouseID: selected.WarehouseID, AreaID: selected.AreaID, LocationID: selected.LocationID,
		TemperatureCelsius: in.TemperatureCelsius, HumidityPercent: in.HumidityPercent, MinCelsius: selected.MinCelsius,
		MaxCelsius: selected.MaxCelsius, Source: in.Source, RecordedBy: in.ActorID, RecordedAt: in.RecordedAt,
	}, s.nowFn())
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	s.records[record.ID.String()] = cloneColdChainRecord(record)
	s.mu.Unlock()
	s.appendAudit(ctx, in.ActorID, "pharma_oa.cold_chain.record", record.ID.String(), map[string]any{"batchId": record.BatchID, "balanceId": record.BalanceID, "temperatureCelsius": record.TemperatureCelsius, "humidityPercent": record.HumidityPercent, "source": record.Source})
	return cloneColdChainRecord(record), nil
}

func (s *coldChainService) ListRecords(ctx context.Context, in ColdChainRecordListInput) ([]*domainpharma.ColdChainRecord, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	limit := in.Limit
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	s.mu.RLock()
	out := make([]*domainpharma.ColdChainRecord, 0, len(s.records))
	for _, record := range s.records {
		if strings.TrimSpace(in.BatchID) != "" && record.BatchID != strings.TrimSpace(in.BatchID) {
			continue
		}
		if strings.TrimSpace(in.WarehouseID) != "" && record.WarehouseID != strings.TrimSpace(in.WarehouseID) {
			continue
		}
		out = append(out, cloneColdChainRecord(record))
	}
	s.mu.RUnlock()
	sort.Slice(out, func(i, j int) bool { return out[i].RecordedAt.After(out[j].RecordedAt) })
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (s *coldChainService) Run(ctx context.Context, policy domainpharma.ColdChainScanPolicy, actorID string) (*domainpharma.ColdChainJob, error) {
	if s == nil || s.notifications == nil {
		return nil, fmt.Errorf("cold-chain notification dependency is required")
	}
	if err := policy.Validate(); err != nil {
		return nil, err
	}
	s.runMu.Lock()
	defer s.runMu.Unlock()
	s.mu.Lock()
	s.jobCounter++
	id := shared.ID("cold-chain-job-" + strconv.FormatInt(s.jobCounter, 10))
	job, err := domainpharma.NewColdChainJob(id, policy, actorID, s.nowFn())
	if err == nil {
		s.jobs[id.String()] = cloneColdChainJob(job)
	}
	s.mu.Unlock()
	if err != nil {
		return nil, err
	}
	return s.execute(ctx, job, actorID, false)
}

func (s *coldChainService) Retry(ctx context.Context, jobID string, actorID string) (*domainpharma.ColdChainJob, error) {
	if s == nil || s.notifications == nil {
		return nil, fmt.Errorf("cold-chain notification dependency is required")
	}
	s.runMu.Lock()
	defer s.runMu.Unlock()
	s.mu.RLock()
	job := cloneColdChainJob(s.jobs[strings.TrimSpace(jobID)])
	s.mu.RUnlock()
	if job == nil {
		return nil, fmt.Errorf("cold-chain job not found")
	}
	if job.Status != domainpharma.ColdChainJobFailed {
		return nil, fmt.Errorf("only failed cold-chain jobs can be retried")
	}
	job.RetryCount++
	return s.execute(ctx, job, actorID, true)
}

func (s *coldChainService) execute(ctx context.Context, job *domainpharma.ColdChainJob, actorID string, retry bool) (*domainpharma.ColdChainJob, error) {
	if err := job.Start(actorID, s.nowFn()); err != nil {
		return nil, err
	}
	s.saveJob(job)
	records, err := s.allRecords(ctx)
	if err != nil {
		return s.failJob(ctx, job, err, retry)
	}
	latest := map[string]*domainpharma.ColdChainRecord{}
	for _, record := range records {
		key := coldChainKey(record)
		if current := latest[key]; current == nil || record.RecordedAt.After(current.RecordedAt) {
			latest[key] = record
		}
	}
	matched, created, resolved := 0, 0, 0
	for key, record := range latest {
		reasons, risk := evaluateColdChain(record, job.Policy)
		if len(reasons) == 0 {
			wasResolved, resolveErr := s.resolveAnomaly(ctx, key, record)
			if resolveErr != nil {
				return s.failJob(ctx, job, resolveErr, retry)
			}
			if wasResolved {
				resolved++
			}
			continue
		}
		matched++
		wasCreated, upsertErr := s.upsertAnomaly(ctx, key, record, job.Policy, reasons, risk)
		if upsertErr != nil {
			return s.failJob(ctx, job, upsertErr, retry)
		}
		if wasCreated {
			created++
		}
	}
	if err = job.Succeed(matched, created, resolved, s.nowFn()); err != nil {
		return s.failJob(ctx, job, err, retry)
	}
	s.saveJob(job)
	s.appendAudit(ctx, actorID, "pharma_oa.cold_chain.run", job.ID.String(), map[string]any{"status": job.Status, "matched": matched, "created": created, "resolved": resolved, "retryCount": job.RetryCount, "recipientId": job.Policy.RecipientID})
	return cloneColdChainJob(job), nil
}

func (s *coldChainService) allRecords(ctx context.Context) ([]*domainpharma.ColdChainRecord, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	out := make([]*domainpharma.ColdChainRecord, 0, len(s.records))
	for _, record := range s.records {
		out = append(out, cloneColdChainRecord(record))
	}
	s.mu.RUnlock()
	return out, nil
}

func evaluateColdChain(record *domainpharma.ColdChainRecord, policy domainpharma.ColdChainScanPolicy) ([]string, domainpharma.ColdChainRisk) {
	reasons := make([]string, 0, 2)
	risk := domainpharma.ColdChainRiskMedium
	if record.TemperatureCelsius < record.MinCelsius {
		reasons = append(reasons, "temperature_below_min")
		risk = domainpharma.ColdChainRiskHigh
	} else if record.TemperatureCelsius > record.MaxCelsius {
		reasons = append(reasons, "temperature_above_max")
		risk = domainpharma.ColdChainRiskHigh
	}
	if record.HumidityPercent < policy.MinHumidityPercent {
		reasons = append(reasons, "humidity_below_min")
	} else if record.HumidityPercent > policy.MaxHumidityPercent {
		reasons = append(reasons, "humidity_above_max")
	}
	return reasons, risk
}

func (s *coldChainService) upsertAnomaly(ctx context.Context, key string, record *domainpharma.ColdChainRecord, policy domainpharma.ColdChainScanPolicy, reasons []string, risk domainpharma.ColdChainRisk) (bool, error) {
	now := s.nowFn().UTC()
	s.mu.RLock()
	existing := cloneColdChainAnomaly(s.anomalies[key])
	s.mu.RUnlock()
	if existing != nil && existing.Status == domainpharma.ColdChainAnomalyActive {
		updateColdChainAnomaly(existing, record, policy, reasons, risk, now)
		s.mu.Lock()
		s.anomalies[key] = existing
		s.mu.Unlock()
		s.appendAudit(ctx, "system", "pharma_oa.cold_chain.anomaly_update", existing.ID.String(), map[string]any{"recordId": record.ID.String(), "batchId": record.BatchID, "risk": risk, "reasons": reasons})
		return false, nil
	}
	id := shared.ID("cold-chain-anomaly-" + record.BalanceID)
	targetPath := coldChainTargetPath(id.String(), record)
	item, err := s.notifications.Create(ctx, notificationsvc.CreateInput{ID: id.String(), Category: notificationsvc.CategoryReminder, Title: "Cold-chain anomaly", Body: fmt.Sprintf("Batch %s has abnormal cold-chain readings at location %s.", record.BatchNo, record.LocationID), ActorID: policy.RecipientID, Target: notificationsvc.Target{Type: "cold_chain_anomaly", ID: id.String(), Path: targetPath}, DueAt: now})
	if err != nil {
		return false, err
	}
	anomaly := &domainpharma.ColdChainAnomaly{ID: id, Status: domainpharma.ColdChainAnomalyActive, NotificationID: item.ID, TargetPath: targetPath, FirstSeenAt: now}
	updateColdChainAnomaly(anomaly, record, policy, reasons, risk, now)
	s.mu.Lock()
	s.anomalies[key] = anomaly
	s.mu.Unlock()
	s.appendAudit(ctx, "system", "pharma_oa.cold_chain.anomaly_create", anomaly.ID.String(), map[string]any{"recordId": record.ID.String(), "batchId": record.BatchID, "risk": risk, "reasons": reasons, "notificationId": item.ID})
	return true, nil
}

func updateColdChainAnomaly(anomaly *domainpharma.ColdChainAnomaly, record *domainpharma.ColdChainRecord, policy domainpharma.ColdChainScanPolicy, reasons []string, risk domainpharma.ColdChainRisk, now time.Time) {
	anomaly.Status = domainpharma.ColdChainAnomalyActive
	anomaly.Risk = risk
	anomaly.RecordID = record.ID.String()
	anomaly.BalanceID = record.BalanceID
	anomaly.ProductID = record.ProductID
	anomaly.BatchID = record.BatchID
	anomaly.BatchNo = record.BatchNo
	anomaly.WarehouseID = record.WarehouseID
	anomaly.AreaID = record.AreaID
	anomaly.LocationID = record.LocationID
	anomaly.TemperatureCelsius = record.TemperatureCelsius
	anomaly.HumidityPercent = record.HumidityPercent
	anomaly.MinCelsius = record.MinCelsius
	anomaly.MaxCelsius = record.MaxCelsius
	anomaly.MinHumidityPercent = policy.MinHumidityPercent
	anomaly.MaxHumidityPercent = policy.MaxHumidityPercent
	anomaly.Reasons = append([]string(nil), reasons...)
	anomaly.RecipientID = strings.TrimSpace(policy.RecipientID)
	anomaly.LastSeenAt = now.UTC()
	anomaly.ResolvedAt = nil
}

func (s *coldChainService) resolveAnomaly(ctx context.Context, key string, record *domainpharma.ColdChainRecord) (bool, error) {
	s.mu.RLock()
	anomaly := cloneColdChainAnomaly(s.anomalies[key])
	s.mu.RUnlock()
	if anomaly == nil || anomaly.Status != domainpharma.ColdChainAnomalyActive {
		return false, nil
	}
	if _, err := s.notifications.Complete(ctx, anomaly.NotificationID, anomaly.RecipientID); err != nil {
		return false, err
	}
	now := s.nowFn().UTC()
	anomaly.Status = domainpharma.ColdChainAnomalyResolved
	anomaly.RecordID = record.ID.String()
	anomaly.LastSeenAt = now
	anomaly.ResolvedAt = &now
	s.mu.Lock()
	s.anomalies[key] = anomaly
	s.mu.Unlock()
	s.appendAudit(ctx, "system", "pharma_oa.cold_chain.anomaly_resolve", anomaly.ID.String(), map[string]any{"recordId": record.ID.String(), "batchId": record.BatchID, "notificationId": anomaly.NotificationID})
	return true, nil
}

func (s *coldChainService) ListJobs(ctx context.Context) ([]*domainpharma.ColdChainJob, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	out := make([]*domainpharma.ColdChainJob, 0, len(s.jobs))
	for _, job := range s.jobs {
		out = append(out, cloneColdChainJob(job))
	}
	s.mu.RUnlock()
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out, nil
}

func (s *coldChainService) ListAnomalies(ctx context.Context, activeOnly bool) ([]*domainpharma.ColdChainAnomaly, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	out := make([]*domainpharma.ColdChainAnomaly, 0, len(s.anomalies))
	for _, anomaly := range s.anomalies {
		if activeOnly && anomaly.Status != domainpharma.ColdChainAnomalyActive {
			continue
		}
		out = append(out, cloneColdChainAnomaly(anomaly))
	}
	s.mu.RUnlock()
	sort.Slice(out, func(i, j int) bool { return out[i].LastSeenAt.After(out[j].LastSeenAt) })
	return out, nil
}

func (s *coldChainService) listWarehouses(ctx context.Context) ([]*domainpharma.Warehouse, error) {
	const pageSize = 200
	out := make([]*domainpharma.Warehouse, 0)
	for offset := 0; ; offset += pageSize {
		items, err := s.warehouses.List(ctx, WarehouseListInput{Offset: offset, Limit: pageSize})
		if err != nil {
			return nil, err
		}
		out = append(out, items...)
		if len(items) < pageSize {
			break
		}
	}
	return out, nil
}

func coldChainTemperature(warehouse *domainpharma.Warehouse, areaID, locationID string) (domainpharma.WarehouseTemperature, bool) {
	if warehouse == nil || warehouse.Status != domainpharma.WarehouseStatusEnabled {
		return domainpharma.WarehouseTemperature{}, false
	}
	var areaTemperature domainpharma.WarehouseTemperature
	for _, area := range warehouse.Areas {
		if area.ID != strings.TrimSpace(areaID) && !strings.EqualFold(area.Code, strings.TrimSpace(areaID)) {
			continue
		}
		if area.Status != domainpharma.WarehouseStatusEnabled {
			return domainpharma.WarehouseTemperature{}, false
		}
		areaTemperature = area.Temperature
		for _, location := range area.Locations {
			if location.ID != strings.TrimSpace(locationID) && !strings.EqualFold(location.Code, strings.TrimSpace(locationID)) {
				continue
			}
			if location.Status != domainpharma.WarehouseStatusEnabled {
				return domainpharma.WarehouseTemperature{}, false
			}
			if location.Temperature.Controlled {
				return location.Temperature, true
			}
			if areaTemperature.Controlled {
				return areaTemperature, true
			}
			if warehouse.Temperature.Controlled {
				return warehouse.Temperature, true
			}
			return domainpharma.WarehouseTemperature{}, false
		}
		return domainpharma.WarehouseTemperature{}, false
	}
	return domainpharma.WarehouseTemperature{}, false
}

func coldChainKey(record *domainpharma.ColdChainRecord) string {
	if record == nil {
		return ""
	}
	return strings.TrimSpace(record.BalanceID)
}

func coldChainTargetPath(anomalyID string, record *domainpharma.ColdChainRecord) string {
	query := url.Values{}
	query.Set("anomalyId", anomalyID)
	query.Set("balanceId", record.BalanceID)
	query.Set("batchId", record.BatchID)
	return "/skoll/pharma-oa/cold-chain?" + query.Encode()
}

func (s *coldChainService) failJob(ctx context.Context, job *domainpharma.ColdChainJob, cause error, retry bool) (*domainpharma.ColdChainJob, error) {
	job.Fail(cause, s.nowFn())
	s.saveJob(job)
	s.appendAudit(ctx, job.LastRunBy, "pharma_oa.cold_chain.fail", job.ID.String(), map[string]any{"error": job.Error, "retry": retry, "retryCount": job.RetryCount})
	return cloneColdChainJob(job), cause
}

func (s *coldChainService) saveJob(job *domainpharma.ColdChainJob) {
	s.mu.Lock()
	s.jobs[job.ID.String()] = cloneColdChainJob(job)
	s.mu.Unlock()
}

func (s *coldChainService) appendAudit(ctx context.Context, actorID, action, resourceID string, detail map[string]any) {
	if s.audit == nil {
		return
	}
	actorID = strings.TrimSpace(actorID)
	if actorID == "" {
		actorID = "system"
	}
	_, _ = s.audit.Append(ctx, actorID, action, "pharma_oa_cold_chain", resourceID, detail)
}

func cloneColdChainRecord(record *domainpharma.ColdChainRecord) *domainpharma.ColdChainRecord {
	if record == nil {
		return nil
	}
	out := *record
	return &out
}

func cloneColdChainAnomaly(anomaly *domainpharma.ColdChainAnomaly) *domainpharma.ColdChainAnomaly {
	if anomaly == nil {
		return nil
	}
	out := *anomaly
	out.Reasons = append([]string(nil), anomaly.Reasons...)
	if anomaly.ResolvedAt != nil {
		resolved := *anomaly.ResolvedAt
		out.ResolvedAt = &resolved
	}
	return &out
}

func cloneColdChainJob(job *domainpharma.ColdChainJob) *domainpharma.ColdChainJob {
	if job == nil {
		return nil
	}
	out := *job
	out.Logs = append([]domainpharma.ColdChainJobLog(nil), job.Logs...)
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
