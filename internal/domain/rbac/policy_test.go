package rbac

import (
	"testing"

	"github.com/tinboxw/skoll/internal/domain/shared"
)

func TestAllows(t *testing.T) {
	binding := Binding{
		RoleID: shared.ID("r-1"),
		Permissions: []Permission{
			{Resource: "user", Action: "read"},
		},
	}

	if !Allows(binding, "user", "read") {
		t.Fatalf("expected access to be allowed")
	}
	if Allows(binding, "user", "delete") {
		t.Fatalf("expected access to be denied")
	}
}
