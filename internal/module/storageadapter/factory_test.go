package storageadapter

import "testing"

func TestNewByMode(t *testing.T) {
	adapter, err := NewByMode("memory")
	if err != nil {
		t.Fatalf("expected memory mode to succeed, got error: %v", err)
	}
	if adapter == nil {
		t.Fatalf("expected non-nil adapter")
	}

	if _, err := NewByMode("mysql"); err == nil {
		t.Fatalf("expected mysql mode to report planned-not-implemented")
	}

	if _, err := NewByMode("postgres"); err == nil {
		t.Fatalf("expected postgres mode to report planned-not-implemented")
	}

	if _, err := NewByMode("unknown"); err == nil {
		t.Fatalf("expected unknown mode to fail")
	}
}
