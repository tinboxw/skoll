package plugin

import "testing"

func TestTopologicalResolverDetectsCycle(t *testing.T) {
	resolver := NewTopologicalResolver()
	infos := map[string]Info{
		"a": {ID: "a", Name: "A", Version: "1.0.0", Dependencies: []Dependency{{ID: "b"}}},
		"b": {ID: "b", Name: "B", Version: "1.0.0", Dependencies: []Dependency{{ID: "a"}}},
	}

	_, err := resolver.ResolveEnableOrder("a", infos)
	if err == nil {
		t.Fatal("expected cycle error")
	}
}
