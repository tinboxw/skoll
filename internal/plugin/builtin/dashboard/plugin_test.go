package dashboard

import (
	"testing"

	"github.com/tinboxw/skoll/internal/plugin"
)

func TestBuiltinDashboardPluginRegister(t *testing.T) {
	registry := plugin.NewMemoryRegistry()
	p := New()

	if err := p.Register(registry); err != nil {
		t.Fatalf("register plugin error: %v", err)
	}

	snapshot := registry.Snapshot()
	if len(snapshot.Routes) != 1 {
		t.Fatalf("unexpected route count: %d", len(snapshot.Routes))
	}
	if len(snapshot.Widgets) != 2 {
		t.Fatalf("unexpected widget count: %d", len(snapshot.Widgets))
	}
	if len(snapshot.Menus) != 1 {
		t.Fatalf("unexpected menu count: %d", len(snapshot.Menus))
	}
}
