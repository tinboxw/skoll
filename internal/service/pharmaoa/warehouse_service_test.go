package pharmaoa

import (
	"context"
	"strings"
	"testing"
	"time"

	domainpharma "github.com/tinboxw/skoll/internal/domain/pharmaoa"
)

func TestWarehouseServiceMovementLocationEligibility(t *testing.T) {
	service := NewWarehouseService(nil).(*warehouseService)
	service.nowFn = func() time.Time { return time.Date(2026, 7, 4, 0, 0, 0, 0, time.UTC) }
	ctx := context.Background()

	created, err := service.Create(ctx, WarehouseWriteInput{
		Code:        "WH-001",
		Name:        "East Cold Warehouse",
		Region:      "East",
		Temperature: domainpharma.WarehouseTemperature{Controlled: true, MinCelsius: 2, MaxCelsius: 8},
		Areas: []domainpharma.WarehouseArea{{
			ID:     "area-cold",
			Code:   "A-COLD",
			Name:   "Cold Area",
			Status: domainpharma.WarehouseStatusEnabled,
			Locations: []domainpharma.WarehouseLocation{{
				ID:     "loc-001",
				Code:   "L-001",
				Name:   "Shelf 001",
				Status: domainpharma.WarehouseStatusEnabled,
			}},
		}},
		ActorID: "qa-admin",
	})
	if err != nil {
		t.Fatalf("create warehouse: %v", err)
	}
	if created.Temperature.MinCelsius != 2 || created.Areas[0].Locations[0].Code != "L-001" {
		t.Fatalf("unexpected warehouse payload: %+v", created)
	}

	eligible, err := service.ValidateMovementLocation(ctx, WarehouseMovementLocationInput{
		WarehouseID: created.ID.String(),
		AreaID:      "area-cold",
		LocationID:  "loc-001",
	})
	if err != nil {
		t.Fatalf("validate movement location: %v", err)
	}
	if !eligible.Allowed {
		t.Fatalf("expected movement location allowed, got %+v", eligible)
	}

	_, err = service.Update(ctx, created.ID.String(), WarehouseWriteInput{
		Code:        "WH-001",
		Name:        "East Cold Warehouse",
		Region:      "East",
		Temperature: domainpharma.WarehouseTemperature{Controlled: true, MinCelsius: 2, MaxCelsius: 8},
		Areas: []domainpharma.WarehouseArea{{
			ID:     "area-cold",
			Code:   "A-COLD",
			Name:   "Cold Area",
			Status: domainpharma.WarehouseStatusEnabled,
			Locations: []domainpharma.WarehouseLocation{{
				ID:     "loc-001",
				Code:   "L-001",
				Name:   "Shelf 001",
				Status: domainpharma.WarehouseStatusDisabled,
			}},
		}},
		ActorID: "qa-admin",
	})
	if err != nil {
		t.Fatalf("disable location through update: %v", err)
	}
	blocked, err := service.ValidateMovementLocation(ctx, WarehouseMovementLocationInput{
		WarehouseID: created.ID.String(),
		AreaID:      "A-COLD",
		LocationID:  "L-001",
	})
	if err != nil {
		t.Fatalf("validate disabled location: %v", err)
	}
	if blocked.Allowed || !strings.Contains(blocked.Reason, "location is disabled") {
		t.Fatalf("expected disabled location blocked, got %+v", blocked)
	}

	_, err = service.Disable(ctx, created.ID.String(), WarehouseDisableInput{Reason: "maintenance", ActorID: "qa-admin"})
	if err != nil {
		t.Fatalf("disable warehouse: %v", err)
	}
	blocked, err = service.ValidateMovementLocation(ctx, WarehouseMovementLocationInput{
		WarehouseID: created.ID.String(),
		AreaID:      "area-cold",
		LocationID:  "loc-001",
	})
	if err != nil {
		t.Fatalf("validate disabled warehouse: %v", err)
	}
	if blocked.Allowed || !strings.Contains(blocked.Reason, "warehouse is disabled") {
		t.Fatalf("expected disabled warehouse blocked, got %+v", blocked)
	}
}

func TestWarehouseServiceValidation(t *testing.T) {
	service := NewWarehouseService(nil)
	ctx := context.Background()

	_, err := service.Create(ctx, WarehouseWriteInput{
		Code:        "WH-INVALID-TEMP",
		Name:        "Invalid Temp",
		Region:      "North",
		Temperature: domainpharma.WarehouseTemperature{Controlled: true, MinCelsius: 10, MaxCelsius: 2},
		Areas:       []domainpharma.WarehouseArea{{Code: "A1", Name: "Area 1", Locations: []domainpharma.WarehouseLocation{{Code: "L1", Name: "Loc 1"}}}},
	})
	if err == nil || !strings.Contains(err.Error(), "minCelsius") {
		t.Fatalf("expected invalid temperature rejection, got %v", err)
	}

	_, err = service.Create(ctx, WarehouseWriteInput{
		Code:   "WH-DUP",
		Name:   "Duplicate Area",
		Region: "North",
		Areas: []domainpharma.WarehouseArea{
			{ID: "area-1", Code: "A1", Name: "Area 1", Locations: []domainpharma.WarehouseLocation{{Code: "L1", Name: "Loc 1"}}},
			{ID: "area-2", Code: "A1", Name: "Area 2", Locations: []domainpharma.WarehouseLocation{{Code: "L2", Name: "Loc 2"}}},
		},
	})
	if err == nil || !strings.Contains(err.Error(), "duplicate area code") {
		t.Fatalf("expected duplicate area rejection, got %v", err)
	}

	_, err = service.Create(ctx, WarehouseWriteInput{
		Code:   "WH-DUP-LOC",
		Name:   "Duplicate Location",
		Region: "North",
		Areas: []domainpharma.WarehouseArea{{
			Code: "A1",
			Name: "Area 1",
			Locations: []domainpharma.WarehouseLocation{
				{ID: "loc-1", Code: "L1", Name: "Loc 1"},
				{ID: "loc-2", Code: "L1", Name: "Loc 2"},
			},
		}},
	})
	if err == nil || !strings.Contains(err.Error(), "duplicate location code") {
		t.Fatalf("expected duplicate location rejection, got %v", err)
	}
}
