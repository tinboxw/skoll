package permission

import "testing"

func TestResourceTypes(t *testing.T) {
	cases := map[ResourceType]string{
		ResourceTypeAPI:       "api",
		ResourceTypeMenu:      "menu",
		ResourceTypeButton:    "button",
		ResourceTypeDataScope: "data_scope",
		ResourceTypePlugin:    "plugin",
	}

	for got, want := range cases {
		if string(got) != want {
			t.Fatalf("resource type = %q, want %q", got, want)
		}
	}
}

func TestNewResourceDefaultsEnabled(t *testing.T) {
	resource := NewResource(ResourceIdentity{
		Key:    "user.read",
		Type:   ResourceTypeAPI,
		Module: "user",
		Source: "system",
	}, "Read users")

	if !resource.Enabled {
		t.Fatal("new resource should be enabled by default")
	}
	if resource.Key() != "user.read" {
		t.Fatalf("key = %q", resource.Key())
	}
	if resource.Type() != ResourceTypeAPI {
		t.Fatalf("type = %q", resource.Type())
	}
	if resource.Module() != "user" {
		t.Fatalf("module = %q", resource.Module())
	}
	if resource.Source() != "system" {
		t.Fatalf("source = %q", resource.Source())
	}
}
