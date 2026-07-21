package hostservice

import (
	"context"
	"strings"
	"testing"

	auditsvc "github.com/tinboxw/skoll/internal/service/audit"
	systemsvc "github.com/tinboxw/skoll/internal/service/system"
	"github.com/tinboxw/skoll/internal/store/clickhouse"
	"github.com/tinboxw/skoll/internal/store/memory"
)

func TestSecretServiceEncryptsAndIsolatesPluginValues(t *testing.T) {
	settings := systemsvc.NewService(memory.NewSystemStore())
	auditBackend := auditsvc.NewService(clickhouse.NewAuditStore())
	auditPort, err := NewAuditService("pharma_oa", auditBackend)
	if err != nil {
		t.Fatalf("NewAuditService error: %v", err)
	}
	service, err := NewSecretService("pharma_oa", settings, auditPort, "host-secret-at-least-sixteen")
	if err != nil {
		t.Fatalf("NewSecretService error: %v", err)
	}
	if err := service.Set(context.Background(), "erp.api-key", "plain-secret-value"); err != nil {
		t.Fatalf("Set error: %v", err)
	}
	got, err := service.Get(context.Background(), "erp.api-key")
	if err != nil || got != "plain-secret-value" {
		t.Fatalf("Get value=%q err=%v", got, err)
	}
	impl := service.(*secretService)
	stored, err := settings.GetByKey(context.Background(), impl.storageKey("erp.api-key"))
	if err != nil || stored == nil || !stored.Encrypted || strings.Contains(stored.Value, "plain-secret-value") {
		t.Fatalf("secret was not encrypted at rest: %+v err=%v", stored, err)
	}
	otherAudit, _ := NewAuditService("other_plugin", auditBackend)
	other, err := NewSecretService("other_plugin", settings, otherAudit, "host-secret-at-least-sixteen")
	if err != nil {
		t.Fatalf("NewSecretService other error: %v", err)
	}
	if _, err := other.Get(context.Background(), "erp.api-key"); err == nil {
		t.Fatal("another plugin read a foreign secret")
	}
	records, err := auditBackend.ListByActor(context.Background(), "plugin:pharma_oa", 10)
	if err != nil || len(records) != 1 {
		t.Fatalf("secret audit records=%d err=%v", len(records), err)
	}
	if strings.Contains(string(mustJSON(t, records[0].Detail)), "plain-secret-value") {
		t.Fatal("secret plaintext leaked into audit evidence")
	}
}
