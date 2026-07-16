package pharmaoa

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"
	"time"

	domainpharma "github.com/tinboxw/skoll/internal/domain/pharmaoa"
	auditsvc "github.com/tinboxw/skoll/internal/service/audit"
)

type BusinessMetricsBucket string

const (
	BusinessMetricsBucketDay   BusinessMetricsBucket = "day"
	BusinessMetricsBucketWeek  BusinessMetricsBucket = "week"
	BusinessMetricsBucketMonth BusinessMetricsBucket = "month"
)

type BusinessMetricsInventoryReader interface {
	ListAlerts(context.Context, bool) ([]*domainpharma.InventoryAlert, error)
}

type BusinessMetricsQualificationReader interface {
	List(context.Context, QualificationListInput) ([]QualificationRecord, error)
}

type BusinessMetricsApprovalReader interface {
	ListRequests(context.Context) ([]*domainpharma.PurchaseRequest, error)
}

type BusinessMetricsFollowUpReader interface {
	List(context.Context, CustomerFollowUpListInput) ([]*domainpharma.CustomerFollowUp, error)
}

type BusinessMetricsSalesReader interface {
	ListOrders(context.Context) ([]*domainpharma.SalesOrder, error)
}

type BusinessMetricsService interface {
	Get(context.Context, BusinessMetricsInput) (*BusinessMetricsSnapshot, error)
}

type BusinessMetricsInput struct {
	From              time.Time
	To                time.Time
	Bucket            BusinessMetricsBucket
	QualificationDays int
	ActorID           string
}

type BusinessMetricsWindow struct {
	From   time.Time             `json:"from"`
	To     time.Time             `json:"to"`
	Bucket BusinessMetricsBucket `json:"bucket"`
}

type BusinessMetricsStockAlerts struct {
	Active     int `json:"active"`
	LowStock   int `json:"lowStock"`
	OverStock  int `json:"overStock"`
	NearExpiry int `json:"nearExpiry"`
}

type BusinessMetricsQualifications struct {
	Expired  int `json:"expired"`
	Expiring int `json:"expiring"`
	Employee int `json:"employee"`
	Supplier int `json:"supplier"`
	Customer int `json:"customer"`
}

type BusinessMetricsApprovalEfficiency struct {
	Total                  int     `json:"total"`
	Pending                int     `json:"pending"`
	Approved               int     `json:"approved"`
	Rejected               int     `json:"rejected"`
	CompletionRate         float64 `json:"completionRate"`
	AverageCompletionHours float64 `json:"averageCompletionHours"`
}

type BusinessMetricsCustomerFollowUps struct {
	Total          int     `json:"total"`
	Planned        int     `json:"planned"`
	Completed      int     `json:"completed"`
	Cancelled      int     `json:"cancelled"`
	Overdue        int     `json:"overdue"`
	CompletionRate float64 `json:"completionRate"`
}

type BusinessMetricsSalesPoint struct {
	StartedAt   time.Time `json:"startedAt"`
	OrderCount  int       `json:"orderCount"`
	AmountCents int64     `json:"amountCents"`
}

type BusinessMetricsSalesTrend struct {
	OrderCount  int                         `json:"orderCount"`
	AmountCents int64                       `json:"amountCents"`
	Series      []BusinessMetricsSalesPoint `json:"series"`
}

type BusinessMetricsSnapshot struct {
	Window             BusinessMetricsWindow             `json:"window"`
	StockAlerts        BusinessMetricsStockAlerts        `json:"stockAlerts"`
	Qualifications     BusinessMetricsQualifications     `json:"qualifications"`
	ApprovalEfficiency BusinessMetricsApprovalEfficiency `json:"approvalEfficiency"`
	CustomerFollowUps  BusinessMetricsCustomerFollowUps  `json:"customerFollowUps"`
	SalesTrend         BusinessMetricsSalesTrend         `json:"salesTrend"`
	GeneratedAt        time.Time                         `json:"generatedAt"`
}

type businessMetricsService struct {
	inventory      BusinessMetricsInventoryReader
	qualifications BusinessMetricsQualificationReader
	approvals      BusinessMetricsApprovalReader
	followUps      BusinessMetricsFollowUpReader
	sales          BusinessMetricsSalesReader
	audit          auditsvc.Service
	nowFn          func() time.Time
}

func NewBusinessMetricsService(inventory BusinessMetricsInventoryReader, qualifications BusinessMetricsQualificationReader, approvals BusinessMetricsApprovalReader, followUps BusinessMetricsFollowUpReader, sales BusinessMetricsSalesReader, audit auditsvc.Service) BusinessMetricsService {
	return &businessMetricsService{inventory: inventory, qualifications: qualifications, approvals: approvals, followUps: followUps, sales: sales, audit: audit, nowFn: func() time.Time { return time.Now().UTC() }}
}

func (s *businessMetricsService) Get(ctx context.Context, in BusinessMetricsInput) (*BusinessMetricsSnapshot, error) {
	if s == nil || s.inventory == nil || s.qualifications == nil || s.approvals == nil || s.followUps == nil || s.sales == nil {
		return nil, fmt.Errorf("business metrics dependencies are required")
	}
	now := s.nowFn().UTC()
	if err := normalizeBusinessMetricsInput(&in, now); err != nil {
		return nil, err
	}
	actorID := strings.TrimSpace(in.ActorID)
	if actorID == "" {
		return nil, fmt.Errorf("actorId is required")
	}

	var alerts []*domainpharma.InventoryAlert
	var qualifications []QualificationRecord
	var approvals []*domainpharma.PurchaseRequest
	var followUps []*domainpharma.CustomerFollowUp
	var orders []*domainpharma.SalesOrder
	errCh := make(chan error, 5)
	var wg sync.WaitGroup
	run := func(load func() error) {
		defer wg.Done()
		if err := load(); err != nil {
			errCh <- err
		}
	}
	wg.Add(5)
	go run(func() (err error) { alerts, err = s.inventory.ListAlerts(ctx, true); return err })
	go run(func() (err error) {
		qualifications, err = s.qualifications.List(ctx, QualificationListInput{Days: in.QualificationDays})
		return err
	})
	go run(func() (err error) { approvals, err = s.approvals.ListRequests(ctx); return err })
	go run(func() (err error) {
		followUps, err = s.followUps.List(ctx, CustomerFollowUpListInput{From: in.From, To: in.To, ActorID: actorID, Scope: CustomerFollowUpAccessScope{IncludeAll: true}})
		return err
	})
	go run(func() (err error) { orders, err = s.sales.ListOrders(ctx); return err })
	wg.Wait()
	close(errCh)
	for err := range errCh {
		if err != nil {
			return nil, err
		}
	}

	snapshot := &BusinessMetricsSnapshot{
		Window:      BusinessMetricsWindow{From: in.From, To: in.To, Bucket: in.Bucket},
		GeneratedAt: now,
		SalesTrend:  BusinessMetricsSalesTrend{Series: businessMetricsSeries(in.From, in.To, in.Bucket)},
	}
	snapshot.StockAlerts = aggregateBusinessMetricsAlerts(alerts)
	snapshot.Qualifications = aggregateBusinessMetricsQualifications(qualifications)
	snapshot.ApprovalEfficiency = aggregateBusinessMetricsApprovals(approvals, in.From, in.To)
	snapshot.CustomerFollowUps = aggregateBusinessMetricsFollowUps(followUps, now)
	snapshot.SalesTrend = aggregateBusinessMetricsSales(orders, in.From, in.To, snapshot.SalesTrend.Series, in.Bucket)
	s.appendAudit(ctx, actorID, in, snapshot)
	return snapshot, nil
}

func normalizeBusinessMetricsInput(in *BusinessMetricsInput, now time.Time) error {
	if in.From.IsZero() && in.To.IsZero() {
		in.To = now
		in.From = startOfBusinessMetricsDay(now.AddDate(0, 0, -29))
	} else if in.From.IsZero() || in.To.IsZero() {
		return fmt.Errorf("from and to must be provided together")
	}
	in.From, in.To = in.From.UTC(), in.To.UTC()
	if in.To.Before(in.From) {
		return fmt.Errorf("to must not be before from")
	}
	if in.To.Sub(in.From) > 366*24*time.Hour {
		return fmt.Errorf("business metrics window cannot exceed 366 days")
	}
	if in.Bucket == "" {
		in.Bucket = BusinessMetricsBucketDay
	}
	if in.Bucket != BusinessMetricsBucketDay && in.Bucket != BusinessMetricsBucketWeek && in.Bucket != BusinessMetricsBucketMonth {
		return fmt.Errorf("bucket must be day, week, or month")
	}
	if in.QualificationDays == 0 {
		in.QualificationDays = 30
	}
	if in.QualificationDays < 1 || in.QualificationDays > 365 {
		return fmt.Errorf("qualificationDays must be between 1 and 365")
	}
	return nil
}

func aggregateBusinessMetricsAlerts(items []*domainpharma.InventoryAlert) BusinessMetricsStockAlerts {
	out := BusinessMetricsStockAlerts{}
	for _, item := range items {
		if item == nil || item.Status != domainpharma.InventoryAlertActive {
			continue
		}
		out.Active++
		switch item.Type {
		case domainpharma.InventoryAlertLowStock:
			out.LowStock++
		case domainpharma.InventoryAlertOverStock:
			out.OverStock++
		case domainpharma.InventoryAlertNearExpiry:
			out.NearExpiry++
		}
	}
	return out
}

func aggregateBusinessMetricsQualifications(items []QualificationRecord) BusinessMetricsQualifications {
	out := BusinessMetricsQualifications{}
	for _, item := range items {
		if item.Status != QualificationStatusExpired && item.Status != QualificationStatusExpiring || !qualificationSubjectActive(item) {
			continue
		}
		if item.Status == QualificationStatusExpired {
			out.Expired++
		} else {
			out.Expiring++
		}
		switch item.SubjectType {
		case QualificationSubjectEmployee:
			out.Employee++
		case QualificationSubjectSupplier:
			out.Supplier++
		case QualificationSubjectCustomer:
			out.Customer++
		}
	}
	return out
}

func aggregateBusinessMetricsApprovals(items []*domainpharma.PurchaseRequest, from, to time.Time) BusinessMetricsApprovalEfficiency {
	out := BusinessMetricsApprovalEfficiency{}
	var duration time.Duration
	completed := 0
	for _, item := range items {
		if item == nil || !businessMetricsInWindow(item.Meta.CreatedAt, from, to) {
			continue
		}
		out.Total++
		switch item.Status {
		case domainpharma.PurchaseRequestPending:
			out.Pending++
		case domainpharma.PurchaseRequestApproved:
			out.Approved++
			completed++
			duration += nonNegativeBusinessMetricsDuration(item.Meta.CreatedAt, item.Meta.UpdatedAt)
		case domainpharma.PurchaseRequestRejected:
			out.Rejected++
			completed++
			duration += nonNegativeBusinessMetricsDuration(item.Meta.CreatedAt, item.Meta.UpdatedAt)
		}
	}
	out.CompletionRate = businessMetricsPercent(completed, out.Total)
	if completed > 0 {
		out.AverageCompletionHours = math.Round(duration.Hours()/float64(completed)*100) / 100
	}
	return out
}

func aggregateBusinessMetricsFollowUps(items []*domainpharma.CustomerFollowUp, now time.Time) BusinessMetricsCustomerFollowUps {
	out := BusinessMetricsCustomerFollowUps{}
	for _, item := range items {
		if item == nil {
			continue
		}
		out.Total++
		switch item.Status {
		case domainpharma.CustomerFollowUpPlanned:
			out.Planned++
			if item.ScheduledAt.Before(now) {
				out.Overdue++
			}
		case domainpharma.CustomerFollowUpCompleted:
			out.Completed++
		case domainpharma.CustomerFollowUpCancelled:
			out.Cancelled++
		}
	}
	out.CompletionRate = businessMetricsPercent(out.Completed, out.Total)
	return out
}

func aggregateBusinessMetricsSales(items []*domainpharma.SalesOrder, from, to time.Time, series []BusinessMetricsSalesPoint, bucket BusinessMetricsBucket) BusinessMetricsSalesTrend {
	out := BusinessMetricsSalesTrend{Series: series}
	index := make(map[int64]int, len(series))
	for i := range series {
		index[series[i].StartedAt.Unix()] = i
	}
	for _, item := range items {
		if item == nil || !businessMetricsInWindow(item.CreatedAt, from, to) {
			continue
		}
		amount := int64(math.Round(item.TotalAmount * 100))
		out.OrderCount++
		out.AmountCents += amount
		key := businessMetricsBucketStart(item.CreatedAt, bucket).Unix()
		if i, ok := index[key]; ok {
			out.Series[i].OrderCount++
			out.Series[i].AmountCents += amount
		}
	}
	return out
}

func businessMetricsSeries(from, to time.Time, bucket BusinessMetricsBucket) []BusinessMetricsSalesPoint {
	start := businessMetricsBucketStart(from, bucket)
	out := make([]BusinessMetricsSalesPoint, 0, 367)
	for !start.After(to) {
		out = append(out, BusinessMetricsSalesPoint{StartedAt: start})
		switch bucket {
		case BusinessMetricsBucketWeek:
			start = start.AddDate(0, 0, 7)
		case BusinessMetricsBucketMonth:
			start = start.AddDate(0, 1, 0)
		default:
			start = start.AddDate(0, 0, 1)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].StartedAt.Before(out[j].StartedAt) })
	return out
}

func businessMetricsBucketStart(value time.Time, bucket BusinessMetricsBucket) time.Time {
	value = startOfBusinessMetricsDay(value.UTC())
	switch bucket {
	case BusinessMetricsBucketWeek:
		days := (int(value.Weekday()) + 6) % 7
		return value.AddDate(0, 0, -days)
	case BusinessMetricsBucketMonth:
		return time.Date(value.Year(), value.Month(), 1, 0, 0, 0, 0, time.UTC)
	default:
		return value
	}
}

func startOfBusinessMetricsDay(value time.Time) time.Time {
	return time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, time.UTC)
}
func businessMetricsInWindow(value, from, to time.Time) bool {
	return !value.Before(from) && !value.After(to)
}
func nonNegativeBusinessMetricsDuration(from, to time.Time) time.Duration {
	if to.Before(from) {
		return 0
	}
	return to.Sub(from)
}
func businessMetricsPercent(part, total int) float64 {
	if total == 0 {
		return 0
	}
	return math.Round(float64(part)*10000/float64(total)) / 100
}

func (s *businessMetricsService) appendAudit(ctx context.Context, actorID string, in BusinessMetricsInput, snapshot *BusinessMetricsSnapshot) {
	if s.audit == nil {
		return
	}
	_, _ = s.audit.Append(ctx, actorID, "pharma_oa.business_metrics.read", "pharma_oa_business_metrics", "business-metrics", map[string]any{"from": in.From, "to": in.To, "bucket": in.Bucket, "qualificationDays": in.QualificationDays, "orders": snapshot.SalesTrend.OrderCount, "activeAlerts": snapshot.StockAlerts.Active})
}
