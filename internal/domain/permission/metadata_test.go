package permission

import (
	"encoding/json"
	"testing"
)

func TestValidateRisk(t *testing.T) {
	for _, risk := range []RiskLevel{RiskLevelLow, RiskLevelMedium, RiskLevelHigh, RiskLevelCritical} {
		if err := ValidateRisk(risk); err != nil {
			t.Fatalf("risk %q should be valid: %v", risk, err)
		}
	}
	if err := ValidateRisk(RiskLevel("warn")); err == nil {
		t.Fatal("expected invalid risk error")
	}
}

func TestNormalizeAndValidateMetadata(t *testing.T) {
	metadata := NormalizeMetadata(map[string]string{
		" Owner ": " admin ",
		"":        "ignored",
	})
	if metadata["owner"] != "admin" {
		t.Fatalf("metadata = %#v", metadata)
	}
	if err := ValidateMetadata(metadata); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := ValidateMetadata(map[string]string{"Bad Key": "value"}); err == nil {
		t.Fatal("expected invalid metadata key error")
	}
}

func TestResourceMetadataSerializable(t *testing.T) {
	resource, err := NewResourceWithMetadata(ResourceIdentity{
		Key:    "plugin.install",
		Type:   ResourceTypePlugin,
		Module: "plugin",
		Source: "system",
	}, "Install plugin", RiskLevelHigh, map[string]string{"owner": "platform"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resource.Risk != RiskLevelHigh {
		t.Fatalf("risk = %q", resource.Risk)
	}
	if resource.Metadata["owner"] != "platform" {
		t.Fatalf("metadata = %#v", resource.Metadata)
	}
	if _, err := json.Marshal(resource); err != nil {
		t.Fatalf("metadata should be serializable: %v", err)
	}
}
