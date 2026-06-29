package user

import (
	"context"
	"strings"
	"testing"
	"time"

	domainrbac "github.com/tinboxw/skoll/internal/domain/rbac"
	"github.com/tinboxw/skoll/internal/domain/shared"
	domainuser "github.com/tinboxw/skoll/internal/domain/user"
	rbacservice "github.com/tinboxw/skoll/internal/service/rbac"
	"github.com/tinboxw/skoll/internal/store"
)

func TestUserServiceCreateAndDisable(t *testing.T) {
	bundle, err := store.NewBundle(store.Options{Mode: store.ModeMemory})
	if err != nil {
		t.Fatalf("store.NewBundle error: %v", err)
	}

	svc := NewService(bundle.Users, bundle.Audit, bundle.UnitOfWork)
	created, err := svc.Create(context.Background(), CreateUserInput{
		Account:      "svc_user",
		Name:         "Service User",
		Email:        "svc@example.com",
		PasswordHash: "1234567890abcdef",
		ActorID:      "admin-1",
	})
	if err != nil {
		t.Fatalf("Create error: %v", err)
	}
	if created.ID == "" {
		t.Fatalf("expected generated id")
	}

	if err := svc.Disable(context.Background(), created.ID.String(), "admin-1"); err != nil {
		t.Fatalf("Disable error: %v", err)
	}

	got, err := svc.Get(context.Background(), created.ID.String())
	if err != nil {
		t.Fatalf("Get error: %v", err)
	}
	if got == nil || got.IsActive() {
		t.Fatalf("expected disabled user")
	}
}

func TestUserServiceCreateAcceptsPlainPassword(t *testing.T) {
	bundle, err := store.NewBundle(store.Options{Mode: store.ModeMemory})
	if err != nil {
		t.Fatalf("store.NewBundle error: %v", err)
	}

	svc := NewService(bundle.Users, bundle.Audit, bundle.UnitOfWork)
	created, err := svc.Create(context.Background(), CreateUserInput{
		Account:      "svc_plain_pwd",
		Name:         "Service Plain Password",
		Email:        "svc_plain@example.com",
		PasswordHash: "password123",
		ActorID:      "admin-1",
	})
	if err != nil {
		t.Fatalf("Create error: %v", err)
	}
	if created == nil {
		t.Fatalf("expected created user")
	}
	if !strings.HasPrefix(created.Password.String(), "sha256:") {
		t.Fatalf("expected hashed password, got %q", created.Password.String())
	}
}

func TestUserServiceCreateAndUpdateOrganization(t *testing.T) {
	bundle, err := store.NewBundle(store.Options{Mode: store.ModeMemory})
	if err != nil {
		t.Fatalf("store.NewBundle error: %v", err)
	}

	svc := NewService(bundle.Users, bundle.Audit, bundle.UnitOfWork)
	created, err := svc.Create(context.Background(), CreateUserInput{
		Account:      "svc_org",
		Name:         "Service Org",
		Email:        "svc_org@example.com",
		PasswordHash: "password123",
		DepartmentID: "dept-root",
		PositionID:   "pos-admin",
		ActorID:      "admin-1",
	})
	if err != nil {
		t.Fatalf("Create error: %v", err)
	}
	if created.DepartmentID != "dept-root" || created.PositionID != "pos-admin" {
		t.Fatalf("unexpected organization fields: %+v", created)
	}

	updated, err := svc.Update(context.Background(), UpdateUserInput{
		ID:           created.ID.String(),
		Name:         created.Name,
		Email:        created.Email.String(),
		Status:       string(created.Status),
		DepartmentID: "dept-ops",
		PositionID:   "pos-user",
	})
	if err != nil {
		t.Fatalf("Update error: %v", err)
	}
	if updated.DepartmentID != "dept-ops" || updated.PositionID != "pos-user" {
		t.Fatalf("unexpected updated organization fields: %+v", updated)
	}
}

func TestUserServiceCreateBatchAtomicStopsOnFirstError(t *testing.T) {
	bundle, err := store.NewBundle(store.Options{Mode: store.ModeMemory})
	if err != nil {
		t.Fatalf("store.NewBundle error: %v", err)
	}

	svc := NewService(bundle.Users, bundle.Audit, bundle.UnitOfWork)
	results, err := svc.CreateBatch(context.Background(), BatchCreateInput{
		Atomic: true,
		Items: []CreateUserInput{
			{
				Account:      "batch_ok",
				Name:         "Batch OK",
				Email:        "batch_ok@example.com",
				PasswordHash: "1234567890abcdef",
				ActorID:      "admin-1",
			},
			{
				Account:      "batch_bad",
				Name:         "Batch Bad",
				Email:        "batch_bad@example.com",
				PasswordHash: "short",
				ActorID:      "admin-1",
			},
		},
	})
	if err != nil {
		t.Fatalf("CreateBatch returned unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if !results[0].Success {
		t.Fatalf("expected first item success, got %+v", results[0])
	}
	if results[1].Success {
		t.Fatalf("expected second item failed, got %+v", results[1])
	}
	if !strings.Contains(strings.ToLower(results[1].Message), "password") {
		t.Fatalf("expected password validation failure message, got %q", results[1].Message)
	}

	list, err := svc.List(context.Background(), ListInput{Offset: 0, Limit: 20})
	if err != nil {
		t.Fatalf("List error: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected one persisted user before stop, got %d", len(list))
	}
	if list[0].Account != "batch_ok" {
		t.Fatalf("expected first user to persist, got account=%q", list[0].Account)
	}
}

func TestUserServiceListAppliesDataScope(t *testing.T) {
	bundle, err := store.NewBundle(store.Options{Mode: store.ModeMemory})
	if err != nil {
		t.Fatalf("store.NewBundle error: %v", err)
	}
	ctx := context.Background()
	for _, item := range []*domainuser.User{
		newScopedUser("1", "alice", "dept-a"),
		newScopedUser("2", "bob", "dept-b"),
		newScopedUser("3", "carol", "dept-c"),
		newScopedUser("4", "dave", "dept-b-child"),
	} {
		if err := bundle.Users.Save(ctx, item); err != nil {
			t.Fatalf("seed user %s error: %v", item.ID, err)
		}
	}

	svc := NewServiceWithDataScope(bundle.Users, bundle.Audit, bundle.UnitOfWork, rbacservice.NewService(bundle.RBAC))

	cases := []struct {
		name string
		in   ListInput
		want []string
	}{
		{
			name: "self",
			in:   ListInput{DataScope: domainrbac.DataScopeSelf, ActorUserID: "2"},
			want: []string{"bob"},
		},
		{
			name: "department",
			in:   ListInput{DataScope: domainrbac.DataScopeDepartment, ActorDepartmentID: "dept-b"},
			want: []string{"bob"},
		},
		{
			name: "department tree",
			in: ListInput{
				DataScope:         domainrbac.DataScopeDepartmentTree,
				ActorDepartmentID: "dept-b",
				DepartmentTreeIDs: []string{"dept-b-child"},
			},
			want: []string{"bob", "dave"},
		},
		{
			name: "custom",
			in:   ListInput{DataScope: domainrbac.DataScopeCustom, CustomDepartmentIDs: []string{"dept-a", "dept-c"}},
			want: []string{"alice", "carol"},
		},
		{
			name: "super admin bypass",
			in:   ListInput{DataScope: domainrbac.DataScopeSelf, ActorUserID: "2", SuperAdmin: true},
			want: []string{"alice", "bob", "carol", "dave"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := svc.List(ctx, tc.in)
			if err != nil {
				t.Fatalf("List error: %v", err)
			}
			gotAccounts := make([]string, 0, len(got))
			for _, item := range got {
				gotAccounts = append(gotAccounts, item.Account)
			}
			if strings.Join(gotAccounts, ",") != strings.Join(tc.want, ",") {
				t.Fatalf("expected accounts %v, got %v", tc.want, gotAccounts)
			}
		})
	}
}

func TestUserServiceListRejectsMissingDataScopeContext(t *testing.T) {
	bundle, err := store.NewBundle(store.Options{Mode: store.ModeMemory})
	if err != nil {
		t.Fatalf("store.NewBundle error: %v", err)
	}

	svc := NewServiceWithDataScope(bundle.Users, bundle.Audit, bundle.UnitOfWork, rbacservice.NewService(bundle.RBAC))
	_, err = svc.List(context.Background(), ListInput{DataScope: domainrbac.DataScopeSelf})
	if err == nil {
		t.Fatalf("expected missing actor error")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "actor user id") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUserServiceUpdateRejectsUnsupportedStatus(t *testing.T) {
	bundle, err := store.NewBundle(store.Options{Mode: store.ModeMemory})
	if err != nil {
		t.Fatalf("store.NewBundle error: %v", err)
	}

	svc := NewService(bundle.Users, bundle.Audit, bundle.UnitOfWork)
	created, err := svc.Create(context.Background(), CreateUserInput{
		Account:      "update_status_user",
		Name:         "Update Status User",
		Email:        "update_status@example.com",
		PasswordHash: "1234567890abcdef",
		ActorID:      "admin-1",
	})
	if err != nil {
		t.Fatalf("Create error: %v", err)
	}

	_, err = svc.Update(context.Background(), UpdateUserInput{ID: created.ID.String(), Status: "paused"})
	if err == nil {
		t.Fatalf("expected unsupported status error")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "unsupported status") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func newScopedUser(id, account, departmentID string) *domainuser.User {
	now := time.Now().UTC()
	u := &domainuser.User{
		ID:           shared.ID(id),
		Account:      account,
		Name:         account + " user",
		Email:        domainuser.Email(account + "@example.com"),
		Status:       domainuser.StatusActive,
		Password:     domainuser.PasswordHash("sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"),
		DepartmentID: departmentID,
	}
	u.Meta.Touch(now)
	return u
}

func TestUserServiceDeleteRequiresID(t *testing.T) {
	bundle, err := store.NewBundle(store.Options{Mode: store.ModeMemory})
	if err != nil {
		t.Fatalf("store.NewBundle error: %v", err)
	}

	svc := NewService(bundle.Users, bundle.Audit, bundle.UnitOfWork)
	err = svc.Delete(context.Background(), "   ")
	if err == nil {
		t.Fatalf("expected id required error")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "id is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUserServiceUpdateNotFound(t *testing.T) {
	bundle, err := store.NewBundle(store.Options{Mode: store.ModeMemory})
	if err != nil {
		t.Fatalf("store.NewBundle error: %v", err)
	}

	svc := NewService(bundle.Users, bundle.Audit, bundle.UnitOfWork)
	_, err = svc.Update(context.Background(), UpdateUserInput{ID: "missing-id", Name: "N"})
	if err == nil {
		t.Fatalf("expected user not found error")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "user not found") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUserServiceCreateBatchEmptyItems(t *testing.T) {
	bundle, err := store.NewBundle(store.Options{Mode: store.ModeMemory})
	if err != nil {
		t.Fatalf("store.NewBundle error: %v", err)
	}

	svc := NewService(bundle.Users, bundle.Audit, bundle.UnitOfWork)
	results, err := svc.CreateBatch(context.Background(), BatchCreateInput{Atomic: true, Items: nil})
	if err != nil {
		t.Fatalf("CreateBatch error: %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("expected empty results, got %d", len(results))
	}
}

func TestUserServiceGetRequiresID(t *testing.T) {
	bundle, err := store.NewBundle(store.Options{Mode: store.ModeMemory})
	if err != nil {
		t.Fatalf("store.NewBundle error: %v", err)
	}

	svc := NewService(bundle.Users, bundle.Audit, bundle.UnitOfWork)
	_, err = svc.Get(context.Background(), "")
	if err == nil {
		t.Fatalf("expected id required error")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "id is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUserServiceListRejectsNegativePaging(t *testing.T) {
	bundle, err := store.NewBundle(store.Options{Mode: store.ModeMemory})
	if err != nil {
		t.Fatalf("store.NewBundle error: %v", err)
	}

	svc := NewService(bundle.Users, bundle.Audit, bundle.UnitOfWork)
	_, err = svc.List(context.Background(), ListInput{Offset: -1, Limit: 10})
	if err == nil {
		t.Fatalf("expected offset validation error")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "offset") {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = svc.List(context.Background(), ListInput{Offset: 0, Limit: -1})
	if err == nil {
		t.Fatalf("expected limit validation error")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "limit") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUserServiceUpdateEmailRequiresID(t *testing.T) {
	bundle, err := store.NewBundle(store.Options{Mode: store.ModeMemory})
	if err != nil {
		t.Fatalf("store.NewBundle error: %v", err)
	}

	svc := NewService(bundle.Users, bundle.Audit, bundle.UnitOfWork)
	_, err = svc.UpdateEmail(context.Background(), UpdateEmailInput{ID: "", Email: "x@example.com"})
	if err == nil {
		t.Fatalf("expected id required error")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "id is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUserServiceUpdateEmailNotFound(t *testing.T) {
	bundle, err := store.NewBundle(store.Options{Mode: store.ModeMemory})
	if err != nil {
		t.Fatalf("store.NewBundle error: %v", err)
	}

	svc := NewService(bundle.Users, bundle.Audit, bundle.UnitOfWork)
	_, err = svc.UpdateEmail(context.Background(), UpdateEmailInput{ID: "missing", Email: "x@example.com"})
	if err == nil {
		t.Fatalf("expected user not found error")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "user not found") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUserServiceDisableRequiresID(t *testing.T) {
	bundle, err := store.NewBundle(store.Options{Mode: store.ModeMemory})
	if err != nil {
		t.Fatalf("store.NewBundle error: %v", err)
	}

	svc := NewService(bundle.Users, bundle.Audit, bundle.UnitOfWork)
	err = svc.Disable(context.Background(), "", "admin-1")
	if err == nil {
		t.Fatalf("expected id required error")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "id is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUserServiceDisableNotFound(t *testing.T) {
	bundle, err := store.NewBundle(store.Options{Mode: store.ModeMemory})
	if err != nil {
		t.Fatalf("store.NewBundle error: %v", err)
	}

	svc := NewService(bundle.Users, bundle.Audit, bundle.UnitOfWork)
	err = svc.Disable(context.Background(), "missing", "admin-1")
	if err == nil {
		t.Fatalf("expected user not found error")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "user not found") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUserServiceDeleteNotFound(t *testing.T) {
	bundle, err := store.NewBundle(store.Options{Mode: store.ModeMemory})
	if err != nil {
		t.Fatalf("store.NewBundle error: %v", err)
	}

	svc := NewService(bundle.Users, bundle.Audit, bundle.UnitOfWork)
	err = svc.Delete(context.Background(), "missing")
	if err == nil {
		t.Fatalf("expected user not found error")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "user not found") {
		t.Fatalf("unexpected error: %v", err)
	}
}
