package role

import (
	"context"
	"testing"

	"github.com/tinboxw/skoll/internal/store/memory"
)

func TestRoleServiceGrantRevoke(t *testing.T) {
	svc := NewService(memory.NewRoleStore())

	r, err := svc.Create(context.Background(), CreateRoleInput{
		Name:        "Operator",
		Key:         "operator",
		Description: "ops",
		Permissions: []string{"user:read"},
	})
	if err != nil {
		t.Fatalf("Create error: %v", err)
	}

	r, err = svc.Grant(context.Background(), r.ID.String(), "user:write")
	if err != nil {
		t.Fatalf("Grant error: %v", err)
	}
	if len(r.Permissions) < 2 {
		t.Fatalf("expected permissions to grow")
	}

	r, err = svc.Revoke(context.Background(), r.ID.String(), "user:write")
	if err != nil {
		t.Fatalf("Revoke error: %v", err)
	}
	for _, p := range r.Permissions {
		if p == "user:write" {
			t.Fatalf("permission should be revoked")
		}
	}
}
