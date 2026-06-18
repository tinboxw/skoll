package permission

import "testing"

func TestValidateResource(t *testing.T) {
	valid := ResourceIdentity{
		Key:    "user.read",
		Type:   ResourceTypeAPI,
		Module: "user",
		Source: "system",
	}
	if err := ValidateResource(valid, "Read users"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	cases := []struct {
		name     string
		identity ResourceIdentity
		label    string
	}{
		{name: "missing key", identity: ResourceIdentity{Type: ResourceTypeAPI, Module: "user", Source: "system"}, label: "Read users"},
		{name: "invalid type", identity: ResourceIdentity{Key: "user.read", Type: ResourceType("legacy"), Module: "user", Source: "system"}, label: "Read users"},
		{name: "missing module", identity: ResourceIdentity{Key: "user.read", Type: ResourceTypeAPI, Source: "system"}, label: "Read users"},
		{name: "missing source", identity: ResourceIdentity{Key: "user.read", Type: ResourceTypeAPI, Module: "user"}, label: "Read users"},
		{name: "missing name", identity: ResourceIdentity{Key: "user.read", Type: ResourceTypeAPI, Module: "user", Source: "system"}},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			if err := ValidateResource(tt.identity, tt.label); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestNormalizeIdentity(t *testing.T) {
	got := NormalizeIdentity(ResourceIdentity{
		Key:    " User.Read ",
		Type:   ResourceTypeAPI,
		Module: " User ",
		Source: " System ",
	})
	if got.Key != "user.read" || got.Module != "user" || got.Source != "system" {
		t.Fatalf("NormalizeIdentity() = %#v", got)
	}
}
