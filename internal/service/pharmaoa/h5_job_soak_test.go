package pharmaoa

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	domainpharma "github.com/tinboxw/skoll/internal/domain/pharmaoa"
	"github.com/tinboxw/skoll/internal/testing/jobsoak"
)

func TestH5PharmaJobSoak(t *testing.T) {
	iterations := jobsoak.IterationsFromEnv("SKOLL_H5_SOAK_ITERATIONS", 25)
	common := jobsoak.Thresholds{
		MaxP95Milliseconds: 500, MaxP99Milliseconds: 1500,
		MaxHeapGrowthBytes: 64 << 20, MaxGoroutineGrowth: 4,
		MaxFailures: 0, MaxDuplicateSideEffects: 0,
	}
	scenarios := []jobsoak.Scenario{
		jobsoak.Measure("主数据导入导出与重复提交", iterations, common, soakMasterDataExchange),
		jobsoak.Measure("逾期提醒失败重试与通知幂等", iterations, common, soakPaymentReminders),
		jobsoak.Measure("报表导出失败重试与文件幂等", iterations, common, soakReportExports),
	}
	report := jobsoak.NewReport(jobsoak.BoolFromEnv("SKOLL_H5_RACE_ENABLED"), scenarios...)
	report.Metadata = map[string]string{
		"scope":           "imports, exports, reminders, reports",
		"defaultLanguage": "zh-CN",
		"storage":         "in-memory acceptance fixtures",
	}
	if err := jobsoak.WriteReport(os.Getenv("SKOLL_H5_PHARMA_SOAK_OUTPUT"), report); err != nil {
		t.Fatal(err)
	}
	if !report.Passed {
		for _, scenario := range report.Scenarios {
			t.Logf("%s: %+v", scenario.Name, scenario.Alerts)
		}
		t.Fatal("pharma job soak thresholds failed")
	}
}

func soakMasterDataExchange(index int) (jobsoak.Observation, error) {
	resource, row := soakMasterDataRow(index)
	code := row["code"]
	service := newMasterDataExchangeFixture()
	input := MasterDataImportInput{Resource: resource, Rows: []map[string]string{row}, ActorID: "soak-importer"}
	result, err := service.Import(context.Background(), input)
	if err != nil || result.Created != 1 || len(result.Failed) != 0 {
		return jobsoak.Observation{}, fmt.Errorf("initial import: result=%+v err=%v", result, err)
	}
	duplicate, err := service.Import(context.Background(), input)
	if err != nil || duplicate.Created != 0 || len(duplicate.Failed) != 1 {
		return jobsoak.Observation{}, fmt.Errorf("duplicate import: result=%+v err=%v", duplicate, err)
	}
	exported, err := service.Export(context.Background(), MasterDataExportInput{Resource: resource, ActorID: "soak-exporter"})
	if err != nil || exported.Status != "completed" || exported.FileBase64 == "" {
		return jobsoak.Observation{}, fmt.Errorf("export: job=%+v err=%v", exported, err)
	}
	occurrences := 0
	for _, outputRow := range exported.Rows {
		for _, cell := range outputRow {
			if cell == code {
				occurrences++
			}
		}
	}
	duplicates := 0
	if occurrences > 1 {
		duplicates = occurrences - 1
	}
	observation := jobsoak.Observation{DuplicateSideEffects: duplicates}
	if occurrences != 1 {
		return observation, fmt.Errorf("exported code occurrences=%d, expected 1", occurrences)
	}
	return observation, nil
}

func soakPaymentReminders(index int) (jobsoak.Observation, error) {
	now := time.Date(2026, 7, 18, 9, 0, 0, 0, time.UTC).Add(time.Duration(index) * time.Minute)
	notifier := &togglePaymentNotifier{err: errors.New("temporary notification failure")}
	service := NewPaymentInvoiceService(paymentOrderFixture("order-1", "SO-SOAK", "customer-1", "sales-1", 500), notifier, nil)
	service.(*paymentInvoiceService).nowFn = func() time.Time { return now }
	_, err := service.CreatePaymentPlan(context.Background(), PaymentPlanCreateInput{
		SalesOrderID: "order-1", AmountCents: 50000, DueAt: now.Add(-time.Hour), ActorID: "sales-1",
	})
	if err != nil {
		return jobsoak.Observation{}, err
	}
	failed, err := service.RunOverdueScan(context.Background(), PaymentOverdueScanInput{RecipientID: "finance-1", ActorID: "sales-1"})
	if err == nil || failed.Status != domainpharma.PaymentReminderJobFailed {
		return jobsoak.Observation{}, fmt.Errorf("failed reminder was not observable: job=%+v err=%v", failed, err)
	}
	notifier.err = nil
	retried, err := service.RetryOverdueScan(context.Background(), failed.ID.String(), "sales-1")
	if err != nil || retried.Status != domainpharma.PaymentReminderJobSucceeded || retried.RetryCount != 1 || retried.CreatedCount != 1 {
		return jobsoak.Observation{Retries: 1}, fmt.Errorf("reminder retry: job=%+v err=%v", retried, err)
	}
	again, err := service.RunOverdueScan(context.Background(), PaymentOverdueScanInput{RecipientID: "finance-1", ActorID: "sales-1"})
	if err != nil || again.CreatedCount != 0 {
		return jobsoak.Observation{Retries: 1}, fmt.Errorf("idempotent reminder rerun: job=%+v err=%v", again, err)
	}
	duplicates := 0
	if len(notifier.items) > 1 {
		duplicates = len(notifier.items) - 1
	}
	observation := jobsoak.Observation{Retries: 1, DuplicateSideEffects: duplicates}
	if len(notifier.items) != 1 {
		return observation, fmt.Errorf("notifications=%d, expected 1", len(notifier.items))
	}
	return observation, nil
}

func soakReportExports(index int) (jobsoak.Observation, error) {
	files := &reportFileFixture{fileID: fmt.Sprintf("file-soak-%d", index), err: errors.New("temporary object storage failure")}
	service := NewReportExportService(&reportMetricsFixture{snapshot: reportSnapshot()}, files, &reportAuditFixture{}).(*reportExportService)
	service.nowFn = func() time.Time {
		return time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC).Add(time.Duration(index) * time.Minute)
	}
	input := ReportExportCreateInput{
		ReportType: ReportExportBusinessMetrics,
		From:       time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC).Add(time.Duration(index) * time.Hour),
		To:         time.Date(2026, 7, 2, 0, 0, 0, 0, time.UTC).Add(time.Duration(index) * time.Hour),
		Bucket:     BusinessMetricsBucketDay,
		ActorID:    "manager-1",
	}
	queued, err := service.Queue(context.Background(), input)
	if err != nil {
		return jobsoak.Observation{}, err
	}
	failed, err := waitForReportSoakStatus(service, queued.ID, input.ActorID, ReportExportFailed)
	if err != nil || failed.Status != ReportExportFailed || failed.Error == "" {
		return jobsoak.Observation{}, fmt.Errorf("failed report was not observable: job=%+v err=%v", failed, err)
	}
	files.mu.Lock()
	files.err = nil
	files.mu.Unlock()
	if _, err = service.Retry(context.Background(), queued.ID, input.ActorID); err != nil {
		return jobsoak.Observation{Retries: 1}, err
	}
	completed, err := waitForReportSoakStatus(service, queued.ID, input.ActorID, ReportExportSucceeded)
	if err != nil || completed.Status != ReportExportSucceeded || completed.RetryCount != 1 {
		return jobsoak.Observation{Retries: 1}, fmt.Errorf("report retry: job=%+v err=%v", completed, err)
	}
	replayed, err := service.Queue(context.Background(), input)
	if err != nil || replayed.ID != queued.ID {
		return jobsoak.Observation{Retries: 1}, fmt.Errorf("report replay: job=%+v err=%v", replayed, err)
	}
	duplicates := 0
	if files.uploads > 2 {
		duplicates = files.uploads - 2
	}
	observation := jobsoak.Observation{Retries: 1, DuplicateSideEffects: duplicates}
	if files.uploads != 2 || !strings.Contains(files.body, "sales,amount,3234,cents") {
		return observation, fmt.Errorf("report uploads=%d body=%q", files.uploads, files.body)
	}
	return observation, nil
}

func waitForReportSoakStatus(service ReportExportService, id, actorID string, expected ReportExportJobStatus) (*ReportExportJob, error) {
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		job, err := service.Get(context.Background(), id, actorID)
		if err != nil {
			return nil, err
		}
		if job.Status == expected {
			return job, nil
		}
		if job.Status == ReportExportFailed || job.Status == ReportExportSucceeded {
			return job, fmt.Errorf("report reached %s, expected %s", job.Status, expected)
		}
		time.Sleep(time.Millisecond)
	}
	return nil, fmt.Errorf("report did not reach %s before timeout", expected)
}

func soakMasterDataRow(index int) (string, map[string]string) {
	code := fmt.Sprintf("SOAK-%04d", index)
	switch index % 4 {
	case 0:
		return MasterDataResourceEmployees, map[string]string{"code": code, "name": "Soak Employee", "departmentId": "quality", "positionId": "qa", "email": code + "@example.test"}
	case 1:
		return MasterDataResourceProducts, map[string]string{"code": code, "name": "Soak Product", "spec": "10ml", "dosageForm": "Injection", "manufacturer": "Skoll Pharma", "approvalNumber": "NMPA-" + code}
	case 2:
		return MasterDataResourceSuppliers, map[string]string{"code": code, "name": "Soak Supplier", "rating": "5", "contactName": "Supplier Contact"}
	default:
		return MasterDataResourceCustomers, map[string]string{"code": code, "name": "Soak Customer", "region": "East", "organizationId": "org-soak", "ownerId": "sales-1", "rating": "4", "contactName": "Customer Contact"}
	}
}
