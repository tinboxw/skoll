package persistent_test

import (
	"errors"
	"testing"
	"time"

	"github.com/tinboxw/skoll/internal/module/storageadapter/persistent"
	"github.com/tinboxw/skoll/internal/module/storageadapter/persistent/db"
	"github.com/tinboxw/skoll/internal/module/user"
)

func newUserRepo(t *testing.T) *persistent.UserRepository {
	t.Helper()
	gdb := newTestDB(t)
	if err := db.Migrate(gdb, (&persistent.UserRepository{}).Models()...); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	repo, err := persistent.NewUserRepository(gdb)
	if err != nil {
		t.Fatalf("new user repo: %v", err)
	}
	return repo
}

func TestUserRepository_CreateGetList(t *testing.T) {
	repo := newUserRepo(t)
	a := repo.Create("alice", "alice@example.com")
	b := repo.Create("bob", "bob@example.com")
	if a.ID == 0 || b.ID == 0 || a.ID == b.ID {
		t.Fatalf("ids: %+v %+v", a, b)
	}
	got, err := repo.Get(a.ID)
	if err != nil || got.Name != "alice" {
		t.Fatalf("get: %+v err=%v", got, err)
	}
	all := repo.List()
	if len(all) != 2 {
		t.Fatalf("list: %+v", all)
	}
}

func TestUserRepository_SecurityFlow(t *testing.T) {
	repo := newUserRepo(t)
	u := repo.Create("alice", "alice@example.com")
	now := time.Unix(1_700_000_000, 0).UTC()

	st, err := repo.RotatePassword(u.ID, 0, now)
	if err != nil || st.PasswordRotatedAt.IsZero() {
		t.Fatalf("rotate: %+v err=%v", st, err)
	}

	if _, err := repo.RotatePassword(u.ID, time.Hour, now.Add(time.Minute)); !errors.Is(err, user.ErrPasswordRotationTooFrequent) {
		t.Fatalf("expected rotation-too-frequent, got %v", err)
	}

	st, _ = repo.RegisterLoginFailure(u.ID, 3, time.Minute, now)
	st, _ = repo.RegisterLoginFailure(u.ID, 3, time.Minute, now)
	st, _ = repo.RegisterLoginFailure(u.ID, 3, time.Minute, now)
	if st.LockedUntilUnixSec == 0 {
		t.Fatalf("expected lock: %+v", st)
	}
	st, _ = repo.ResetUserLock(u.ID, now)
	if st.LockedUntilUnixSec != 0 || st.FailedLoginCount != 0 {
		t.Fatalf("reset: %+v", st)
	}

	st, _ = repo.SetMFA(u.ID, true, "totp", now)
	if !st.MFAEnabled || st.MFAProvider != "totp" {
		t.Fatalf("mfa: %+v", st)
	}
}

func TestUserRepository_AuthSessionLifecycle(t *testing.T) {
	repo := newUserRepo(t)
	u := repo.Create("alice", "alice@example.com")
	now := time.Unix(1_700_000_000, 0).UTC()

	pair, err := repo.CreateAuthSession(u.ID, 5, "v1", now)
	if err != nil || pair.SessionID == "" || pair.RefreshToken == "" {
		t.Fatalf("create auth: %+v err=%v", pair, err)
	}

	got, err := repo.GetAuthSession(pair.SessionID)
	if err != nil || got.UserID != u.ID {
		t.Fatalf("get auth: %+v err=%v", got, err)
	}

	pair2, err := repo.RefreshAuthSession(pair.RefreshToken, now.Add(time.Minute))
	if err != nil || pair2.SessionID != pair.SessionID || pair2.RefreshToken == pair.RefreshToken {
		t.Fatalf("refresh: %+v err=%v", pair2, err)
	}

	revoked, err := repo.RevokeAuthSession(pair.SessionID, "logout", now.Add(time.Hour))
	if err != nil || !revoked.Revoked || revoked.RevokeReason == "" {
		t.Fatalf("revoke: %+v err=%v", revoked, err)
	}

	if _, err := repo.RefreshAuthSession(pair2.RefreshToken, now.Add(2*time.Hour)); err == nil {
		t.Fatalf("refresh after revoke must fail")
	}
}

func TestUserRepository_HydrateAcrossInstances(t *testing.T) {
	gdb := newTestDB(t)
	if err := db.Migrate(gdb, (&persistent.UserRepository{}).Models()...); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	repo, err := persistent.NewUserRepository(gdb)
	if err != nil {
		t.Fatalf("first new: %v", err)
	}

	u := repo.Create("alice", "alice@example.com")
	now := time.Unix(1_700_000_000, 0).UTC()
	if _, err := repo.RotatePassword(u.ID, 0, now); err != nil {
		t.Fatalf("rotate: %v", err)
	}
	if _, err := repo.SetMFA(u.ID, true, "totp", now); err != nil {
		t.Fatalf("mfa: %v", err)
	}
	pair, err := repo.CreateAuthSession(u.ID, 9, "v1", now)
	if err != nil {
		t.Fatalf("create auth: %v", err)
	}

	// Reconstruct over the same database.
	repo2, err := persistent.NewUserRepository(gdb)
	if err != nil {
		t.Fatalf("hydrate: %v", err)
	}

	got, err := repo2.Get(u.ID)
	if err != nil || got.Email != "alice@example.com" {
		t.Fatalf("hydrated user: %+v err=%v", got, err)
	}

	gotSess, err := repo2.GetAuthSession(pair.SessionID)
	if err != nil || gotSess.UserID != u.ID || gotSess.ClaimsVersion != "v1" {
		t.Fatalf("hydrated session: %+v err=%v", gotSess, err)
	}

	// Refresh after hydration must succeed (refresh-token index restored).
	pair2, err := repo2.RefreshAuthSession(pair.RefreshToken, now.Add(time.Minute))
	if err != nil || pair2.SessionID != pair.SessionID {
		t.Fatalf("refresh after hydrate: %+v err=%v", pair2, err)
	}

	// Next-id counter continues monotonically.
	next := repo2.Create("bob", "bob@example.com")
	if next.ID <= u.ID {
		t.Fatalf("next id did not advance: %d vs %d", next.ID, u.ID)
	}
}
