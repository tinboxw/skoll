package sql

import (
	"strings"
	"testing"
)

func TestBuildSelectByLower(t *testing.T) {
	query, args, err := BuildSelectByLower("mysql", "sk_users", "account", "Alice", 1)
	if err != nil {
		t.Fatalf("BuildSelectByLower error: %v", err)
	}
	if !strings.Contains(strings.ToLower(query), "from") || !strings.Contains(strings.ToLower(query), "where") {
		t.Fatalf("unexpected query: %s", query)
	}
	if strings.Contains(query, `"sk_users"`) || strings.Contains(query, `"account"`) {
		t.Fatalf("mysql query should not use double-quoted identifiers: %s", query)
	}
	if len(args) < 1 || args[0] != "alice" {
		t.Fatalf("unexpected args: %+v", args)
	}
}

func TestBuildOrderedSelect(t *testing.T) {
	query, args, err := BuildOrderedSelect("postgres", "sk_system_settings", "key", 10, 20)
	if err != nil {
		t.Fatalf("BuildOrderedSelect error: %v", err)
	}
	if !strings.Contains(strings.ToLower(query), "order by") {
		t.Fatalf("missing order by: %s", query)
	}
	if len(args) != 2 {
		t.Fatalf("unexpected args count: %d", len(args))
	}
}

func TestBuildOrderedSelectRejectsInvalidInput(t *testing.T) {
	if _, _, err := BuildOrderedSelect("mysql", "", "key", 0, 10); err == nil {
		t.Fatalf("expected table validation error")
	}
	if _, _, err := BuildOrderedSelect("mysql", "sk_system_settings", "", 0, 10); err == nil {
		t.Fatalf("expected order column validation error")
	}
}
