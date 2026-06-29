package memory

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/tinboxw/skoll/internal/domain/shared"
	"github.com/tinboxw/skoll/internal/domain/user"
	userrepo "github.com/tinboxw/skoll/internal/repository/user"
)

func TestUserStoreListFiltered(t *testing.T) {
	store := NewUserStore()
	ctx := context.Background()
	for _, item := range []*user.User{
		newMemoryUser("1", "alice", "dept-a"),
		newMemoryUser("2", "bob", "dept-b"),
		newMemoryUser("3", "carol", "dept-c"),
		newMemoryUser("4", "dave", "dept-b"),
	} {
		if err := store.Save(ctx, item); err != nil {
			t.Fatalf("Save error: %v", err)
		}
	}

	got, err := store.ListFiltered(ctx, userrepo.ListFilter{
		UserIDs:       []shared.ID{"1"},
		DepartmentIDs: []string{"dept-b"},
	}, 0, 10)
	if err != nil {
		t.Fatalf("ListFiltered error: %v", err)
	}
	if accounts := memoryUserAccounts(got); accounts != "alice,bob,dave" {
		t.Fatalf("unexpected filtered accounts: %s", accounts)
	}

	paged, err := store.ListFiltered(ctx, userrepo.ListFilter{DepartmentIDs: []string{"dept-b"}}, 1, 1)
	if err != nil {
		t.Fatalf("ListFiltered paged error: %v", err)
	}
	if accounts := memoryUserAccounts(paged); accounts != "dave" {
		t.Fatalf("unexpected paged accounts: %s", accounts)
	}
}

func newMemoryUser(id, account, departmentID string) *user.User {
	now := time.Now().UTC()
	u := &user.User{
		ID:           shared.ID(id),
		Account:      account,
		Name:         account + " user",
		Email:        user.Email(account + "@example.com"),
		Status:       user.StatusActive,
		DepartmentID: departmentID,
	}
	u.Meta.Touch(now)
	return u
}

func memoryUserAccounts(items []*user.User) string {
	accounts := make([]string, 0, len(items))
	for _, item := range items {
		accounts = append(accounts, item.Account)
	}
	return strings.Join(accounts, ",")
}
