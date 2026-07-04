package pharmaoa

import (
	"context"
	"testing"
	"time"

	domainpharma "github.com/tinboxw/skoll/internal/domain/pharmaoa"
)

func TestProductServiceCreateUpdateDisable(t *testing.T) {
	service := NewProductService(nil).(*productService)
	service.nowFn = func() time.Time { return time.Date(2026, 7, 4, 0, 0, 0, 0, time.UTC) }

	created, err := service.Create(context.Background(), productFixture("DRUG001"))
	if err != nil {
		t.Fatalf("create product: %v", err)
	}
	if created.Status != domainpharma.ProductStatusActive || created.Temperature.MinCelsius != 2 || created.Temperature.MaxCelsius != 8 {
		t.Fatalf("unexpected created product: %+v", created)
	}

	updated, err := service.Update(context.Background(), created.ID.String(), ProductWriteInput{
		Code:           "DRUG001",
		Name:           "Amoxicillin Capsules",
		Spec:           "0.25g*24",
		DosageForm:     "capsule",
		Manufacturer:   "Skoll Pharma",
		ApprovalNumber: "NMPA-H20260001",
		Temperature:    domainpharma.ProductTemperature{},
		ActorID:        "admin",
	})
	if err != nil {
		t.Fatalf("update product: %v", err)
	}
	if updated.Spec != "0.25g*24" || updated.Temperature.Required {
		t.Fatalf("unexpected updated product: %+v", updated)
	}

	disabled, err := service.Disable(context.Background(), created.ID.String(), ProductDisableInput{Reason: "recalled", ActorID: "admin"})
	if err != nil {
		t.Fatalf("disable product: %v", err)
	}
	if disabled.Status != domainpharma.ProductStatusDisabled || disabled.DisableReason != "recalled" {
		t.Fatalf("unexpected disabled product: %+v", disabled)
	}
}

func TestProductImportValidatesRows(t *testing.T) {
	service := NewProductService(nil)
	rows := []ProductWriteInput{
		productFixture("DRUG001"),
		productFixture("DRUG002"),
		productFixture("DRUG001"),
		{Code: "DRUG003", Name: "Broken", Spec: "box"},
	}
	result, err := service.Import(context.Background(), rows)
	if err != nil {
		t.Fatalf("import products: %v", err)
	}
	if result.Created != 2 || len(result.Failed) != 2 {
		t.Fatalf("unexpected import result: %+v", result)
	}
	if result.Failed[0].Row != 3 || result.Failed[1].Row != 4 {
		t.Fatalf("unexpected failed rows: %+v", result.Failed)
	}
	items, err := service.List(context.Background(), ProductListInput{})
	if err != nil {
		t.Fatalf("list products after import: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("unexpected imported products: %+v", items)
	}
}

func productFixture(code string) ProductWriteInput {
	return ProductWriteInput{
		Code:           code,
		Name:           "Amoxicillin",
		Spec:           "0.25g*12",
		DosageForm:     "capsule",
		Manufacturer:   "Skoll Pharma",
		ApprovalNumber: "NMPA-H20260001",
		Temperature: domainpharma.ProductTemperature{
			Required:   true,
			MinCelsius: 2,
			MaxCelsius: 8,
		},
		ActorID: "admin",
	}
}
