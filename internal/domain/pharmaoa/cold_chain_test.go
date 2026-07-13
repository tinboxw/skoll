package pharmaoa

import (
	"testing"
	"time"

	"github.com/tinboxw/skoll/internal/domain/shared"
)

func TestColdChainRecordValidationAndJobLifecycle(t *testing.T) {
	now := time.Date(2026, 7, 13, 9, 0, 0, 0, time.UTC)
	in := ColdChainRecordInput{BalanceID: "balance-1", ProductID: "product-1", BatchID: "batch-1", BatchNo: "B-001", WarehouseID: "warehouse-1", AreaID: "area-1", LocationID: "location-1", TemperatureCelsius: 4.5, HumidityPercent: 55, MinCelsius: 2, MaxCelsius: 8, Source: "sensor-1", RecordedBy: "quality-user", RecordedAt: now}
	record, err := NewColdChainRecord(shared.ID("record-1"), in, now)
	if err != nil || record.RecordedAt.Location() != time.UTC {
		t.Fatalf("create cold-chain record: %+v %v", record, err)
	}
	in.HumidityPercent = 101
	if _, err = NewColdChainRecord(shared.ID("record-2"), in, now); err == nil {
		t.Fatal("invalid humidity must fail")
	}

	policy := ColdChainScanPolicy{MinHumidityPercent: 30, MaxHumidityPercent: 70, RecipientID: "quality-manager"}
	job, err := NewColdChainJob(shared.ID("job-1"), policy, "quality-user", now)
	if err != nil || job.Status != ColdChainJobPending {
		t.Fatalf("create cold-chain job: %+v %v", job, err)
	}
	if err = job.Start("quality-user", now.Add(time.Minute)); err != nil {
		t.Fatalf("start cold-chain job: %v", err)
	}
	if err = job.Succeed(1, 1, 0, now.Add(2*time.Minute)); err != nil || job.Status != ColdChainJobSucceeded || job.CompletedAt == nil {
		t.Fatalf("complete cold-chain job: %+v %v", job, err)
	}
	if err = job.Start("quality-user", now.Add(3*time.Minute)); err == nil {
		t.Fatal("succeeded cold-chain job must not restart")
	}
}
