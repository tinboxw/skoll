package auth

import (
	"testing"

	"github.com/tinboxw/skoll/internal/plugin"
)

func TestBuiltinAuthPluginRegister(t *testing.T) {
	registry := plugin.NewMemoryRegistry()
	p := New()

	if err := p.Register(registry); err != nil {
		t.Fatalf("register plugin error: %v", err)
	}

	snapshot := registry.Snapshot()
	if len(snapshot.Routes) != 1 {
		t.Fatalf("unexpected route count: %d", len(snapshot.Routes))
	}
	if len(snapshot.Middlewares) != 1 {
		t.Fatalf("unexpected middleware count: %d", len(snapshot.Middlewares))
	}
	if len(snapshot.Events) != 1 {
		t.Fatalf("unexpected event count: %d", len(snapshot.Events))
	}
	if len(snapshot.Menus) != 1 {
		t.Fatalf("unexpected menu count: %d", len(snapshot.Menus))
	}
	if len(snapshot.Widgets) != 1 {
		t.Fatalf("unexpected widget count: %d", len(snapshot.Widgets))
	}
	if len(snapshot.Settings) != 1 {
		t.Fatalf("unexpected setting count: %d", len(snapshot.Settings))
	}
}
