package dictionary

import "testing"

func TestServiceCreateGetListByType(t *testing.T) {
	svc := NewService()
	a := svc.Create("status", "Enabled", "1", 20, true)
	_ = svc.Create("status", "Disabled", "0", 10, true)
	_ = svc.Create("region", "CN", "cn", 1, true)

	got, err := svc.Get(a.ID)
	if err != nil {
		t.Fatalf("get dictionary item failed: %v", err)
	}
	if got.Type != "status" || got.Label != "Enabled" {
		t.Fatalf("unexpected dictionary item: %+v", got)
	}

	all := svc.List()
	if len(all) != 3 {
		t.Fatalf("expected 3 dictionary items, got %d", len(all))
	}
	if all[0].Type != "region" {
		t.Fatalf("expected region first by type sort, got %s", all[0].Type)
	}

	statusItems := svc.ListByType("status")
	if len(statusItems) != 2 {
		t.Fatalf("expected 2 status items, got %d", len(statusItems))
	}
	if statusItems[0].Value != "0" || statusItems[1].Value != "1" {
		t.Fatalf("expected type list sorted by sort order, got %+v", statusItems)
	}
}
