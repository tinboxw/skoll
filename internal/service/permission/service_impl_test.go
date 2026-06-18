package permission

import (
	"context"
	"errors"
	"strings"
	"testing"

	domainpermission "github.com/tinboxw/skoll/internal/domain/permission"
	permissionrepo "github.com/tinboxw/skoll/internal/repository/permission"
	"github.com/tinboxw/skoll/internal/store/memory"
)

func TestRegisterResourceIsIdempotent(t *testing.T) {
	ctx := context.Background()
	store := memory.NewPermissionStore()
	svc := NewService(store)

	in := RegisterResourceInput{
		Key:    "system:user:list",
		Type:   domainpermission.ResourceTypeAPI,
		Module: "system",
		Source: "system",
		Name:   "List users",
	}
	if _, err := svc.RegisterResource(ctx, in); err != nil {
		t.Fatalf("RegisterResource(first) error = %v", err)
	}
	in.Name = "List users updated"
	got, err := svc.RegisterResource(ctx, in)
	if err != nil {
		t.Fatalf("RegisterResource(second) error = %v", err)
	}
	if got.Name != "List users updated" || got.Risk != domainpermission.RiskLevelLow {
		t.Fatalf("resource = %+v", got)
	}

	items, err := store.List(ctx, permissionrepo.ListFilter{}, 0, 0)
	if err != nil {
		t.Fatalf("store.List() error = %v", err)
	}
	if len(items) != 1 || items[0].Name != "List users updated" {
		t.Fatalf("items = %+v", items)
	}
}

func TestRegisterResourceRejectsInvalidInput(t *testing.T) {
	svc := NewService(memory.NewPermissionStore())

	_, err := svc.RegisterResource(context.Background(), RegisterResourceInput{
		Key:    "bad key",
		Type:   domainpermission.ResourceTypeAPI,
		Module: "system",
		Source: "system",
		Name:   "Bad",
	})
	if err == nil || !strings.Contains(strings.ToLower(err.Error()), "permission key") {
		t.Fatalf("expected validation error, got %v", err)
	}
}

func TestRegisterResourcePropagatesRepositoryError(t *testing.T) {
	wantErr := errors.New("store failed")
	svc := NewService(&failingPermissionRepository{err: wantErr})

	_, err := svc.RegisterResource(context.Background(), RegisterResourceInput{
		Key:    "system:user:list",
		Type:   domainpermission.ResourceTypeAPI,
		Module: "system",
		Source: "system",
		Name:   "List users",
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("err = %v, want %v", err, wantErr)
	}
}

func TestListResourcesAndGetResource(t *testing.T) {
	ctx := context.Background()
	store := memory.NewPermissionStore()
	svc := NewService(store)

	_, _ = svc.RegisterResource(ctx, RegisterResourceInput{
		Key:    "system:user:list",
		Type:   domainpermission.ResourceTypeAPI,
		Module: "system",
		Source: "system",
		Name:   "List users",
	})
	_, _ = svc.RegisterResource(ctx, RegisterResourceInput{
		Key:    "plugin.demo:report:list",
		Type:   domainpermission.ResourceTypeMenu,
		Module: "report",
		Source: "plugin.demo",
		Name:   "Reports",
	})
	_ = store.SetEnabled(ctx, "plugin.demo:report:list", false)

	enabled := true
	items, err := svc.ListResources(ctx, ListResourcesInput{
		Type:    domainpermission.ResourceTypeAPI,
		Source:  "system",
		Enabled: &enabled,
	})
	if err != nil {
		t.Fatalf("ListResources() error = %v", err)
	}
	if len(items) != 1 || items[0].Key() != "system:user:list" {
		t.Fatalf("items = %+v", items)
	}

	got, err := svc.GetResource(ctx, "system:user:list")
	if err != nil {
		t.Fatalf("GetResource() error = %v", err)
	}
	if got == nil || got.Key() != "system:user:list" {
		t.Fatalf("got = %+v", got)
	}
}

func TestListResourcesRejectsInvalidPagination(t *testing.T) {
	svc := NewService(memory.NewPermissionStore())

	_, err := svc.ListResources(context.Background(), ListResourcesInput{Offset: -1})
	if err == nil || !strings.Contains(strings.ToLower(err.Error()), "invalid pagination") {
		t.Fatalf("expected invalid pagination error, got %v", err)
	}
}

func TestGetResourceRequiresKey(t *testing.T) {
	svc := NewService(memory.NewPermissionStore())

	_, err := svc.GetResource(context.Background(), " ")
	if err == nil || !strings.Contains(strings.ToLower(err.Error()), "permission key") {
		t.Fatalf("expected key required error, got %v", err)
	}
}

func TestEnableDisableResource(t *testing.T) {
	ctx := context.Background()
	store := memory.NewPermissionStore()
	svc := NewService(store)
	_, _ = svc.RegisterResource(ctx, RegisterResourceInput{
		Key:    "system:user:list",
		Type:   domainpermission.ResourceTypeAPI,
		Module: "system",
		Source: "system",
		Name:   "List users",
	})

	if err := svc.DisableResource(ctx, "system:user:list"); err != nil {
		t.Fatalf("DisableResource() error = %v", err)
	}
	got, _ := svc.GetResource(ctx, "system:user:list")
	if got == nil || got.Enabled {
		t.Fatalf("after disable got = %+v", got)
	}

	if err := svc.EnableResource(ctx, "system:user:list"); err != nil {
		t.Fatalf("EnableResource() error = %v", err)
	}
	got, _ = svc.GetResource(ctx, "system:user:list")
	if got == nil || !got.Enabled {
		t.Fatalf("after enable got = %+v", got)
	}
}

func TestEnableDisableResourceValidationAndRepositoryError(t *testing.T) {
	t.Run("requires key", func(t *testing.T) {
		svc := NewService(memory.NewPermissionStore())
		err := svc.EnableResource(context.Background(), " ")
		if err == nil || !strings.Contains(strings.ToLower(err.Error()), "permission key") {
			t.Fatalf("expected key required error, got %v", err)
		}
	})

	t.Run("propagates repository error", func(t *testing.T) {
		wantErr := errors.New("store failed")
		svc := NewService(&failingPermissionRepository{err: wantErr})
		err := svc.DisableResource(context.Background(), "system:user:list")
		if !errors.Is(err, wantErr) {
			t.Fatalf("err = %v, want %v", err, wantErr)
		}
	})
}

func TestEnableDisableResourceAuditHookReserved(t *testing.T) {
	ctx := context.Background()
	store := memory.NewPermissionStore()
	svc := NewService(store).(*serviceImpl)
	_, _ = svc.RegisterResource(ctx, RegisterResourceInput{
		Key:    "system:user:list",
		Type:   domainpermission.ResourceTypeAPI,
		Module: "system",
		Source: "system",
		Name:   "List users",
	})

	called := false
	svc.auditFn = func(_ context.Context, key string, enabled bool) error {
		called = key == "system:user:list" && !enabled
		return nil
	}
	if err := svc.DisableResource(ctx, "system:user:list"); err != nil {
		t.Fatalf("DisableResource() error = %v", err)
	}
	if !called {
		t.Fatal("expected audit hook to be called")
	}
}

type failingPermissionRepository struct {
	err error
}

func (r *failingPermissionRepository) Register(context.Context, domainpermission.PermissionResource) error {
	return r.err
}

func (r *failingPermissionRepository) Get(context.Context, string) (*domainpermission.PermissionResource, error) {
	return nil, r.err
}

func (r *failingPermissionRepository) List(context.Context, permissionrepo.ListFilter, int, int) ([]domainpermission.PermissionResource, error) {
	return nil, r.err
}

func (r *failingPermissionRepository) SetEnabled(context.Context, string, bool) error {
	return r.err
}
