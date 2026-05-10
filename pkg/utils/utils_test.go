package utils

import (
	"testing"
	"time"
)

func TestStringHelpers(t *testing.T) {
	if !IsBlank(" \t\n ") {
		t.Fatalf("expected blank string")
	}
	if got := NormalizeSpace("  a   b\n c "); got != "a b c" {
		t.Fatalf("unexpected NormalizeSpace result: %q", got)
	}
	if got := Coalesce(" ", "", "x", "y"); got != "x" {
		t.Fatalf("unexpected Coalesce result: %q", got)
	}
	if got := TruncateByRune("你好world", 3); got != "你好w" {
		t.Fatalf("unexpected truncate result: %q", got)
	}
	if !ContainsFold("HelloWorld", "world") {
		t.Fatalf("expected ContainsFold match")
	}
}

func TestConvertHelpers(t *testing.T) {
	if n, err := ToInt64("42"); err != nil || n != 42 {
		t.Fatalf("unexpected ToInt64 result: n=%d err=%v", n, err)
	}
	if b, err := ToBool("true"); err != nil || !b {
		t.Fatalf("unexpected ToBool result: b=%v err=%v", b, err)
	}
	if _, err := ToInt("not-int"); err == nil {
		t.Fatalf("expected ToInt parse error")
	}
}

func TestTimeHelpers(t *testing.T) {
	now := NowUTC()
	if now.Location() != time.UTC {
		t.Fatalf("NowUTC should return UTC")
	}
	raw := "2026-05-10T09:00:00+08:00"
	parsed, err := ParseRFC3339(raw)
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}
	if parsed.Location() != time.UTC {
		t.Fatalf("ParseRFC3339 should normalize to UTC")
	}
	if got := FormatRFC3339(parsed); got == "" {
		t.Fatalf("expected non-empty formatted time")
	}
	if ms := UnixMilli(parsed); ms <= 0 {
		t.Fatalf("expected positive unix milli")
	}
}
