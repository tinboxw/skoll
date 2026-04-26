package user

import (
	"testing"
	"time"
)

func TestServiceCreateGetList(t *testing.T) {
	svc := NewService()
	u1 := svc.Create("alice", "alice@example.com")
	u2 := svc.Create("bob", "bob@example.com")

	got, err := svc.Get(u1.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got.Email != "alice@example.com" {
		t.Fatalf("unexpected user email: %s", got.Email)
	}

	list := svc.List()
	if len(list) != 2 {
		t.Fatalf("expected 2 users, got %d", len(list))
	}
	if list[0].ID != u1.ID || list[1].ID != u2.ID {
		t.Fatalf("expected users sorted by ID")
	}
}

func BenchmarkServiceList(b *testing.B) {
	svc := NewService()
	for i := 0; i < 1000; i++ {
		svc.Create("name", "mail@example.com")
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = svc.List()
	}
}

func TestServiceSecurityHardening(t *testing.T) {
	svc := NewService()
	u := svc.Create("alice", "alice@example.com")
	now := time.Unix(1710000000, 0).UTC()

	rotated, err := svc.RotatePassword(u.ID, 24*time.Hour, now)
	if err != nil {
		t.Fatalf("rotate password failed: %v", err)
	}
	if rotated.PasswordRotatedAt.IsZero() {
		t.Fatalf("expected password rotated timestamp")
	}

	if _, err := svc.RotatePassword(u.ID, 24*time.Hour, now.Add(1*time.Hour)); err != ErrPasswordRotationTooFrequent {
		t.Fatalf("expected ErrPasswordRotationTooFrequent, got %v", err)
	}

	state, err := svc.RegisterLoginFailure(u.ID, 2, 30*time.Minute, now.Add(2*time.Hour))
	if err != nil {
		t.Fatalf("register login failure failed: %v", err)
	}
	if state.FailedLoginCount != 1 {
		t.Fatalf("expected failed count 1, got %d", state.FailedLoginCount)
	}

	state, err = svc.RegisterLoginFailure(u.ID, 2, 30*time.Minute, now.Add(3*time.Hour))
	if err != nil {
		t.Fatalf("register login failure failed: %v", err)
	}
	if state.LockedUntilUnixSec == 0 {
		t.Fatalf("expected lock to be applied")
	}

	state, err = svc.SetMFA(u.ID, true, "totp", now.Add(4*time.Hour))
	if err != nil {
		t.Fatalf("set mfa failed: %v", err)
	}
	if !state.MFAEnabled || state.MFAProvider != "totp" {
		t.Fatalf("expected mfa enabled with provider totp, got %+v", state)
	}

	state, err = svc.ResetUserLock(u.ID, now.Add(5*time.Hour))
	if err != nil {
		t.Fatalf("reset lock failed: %v", err)
	}
	if state.FailedLoginCount != 0 || state.LockedUntilUnixSec != 0 {
		t.Fatalf("expected lock reset state, got %+v", state)
	}
}

func TestServiceSessionSecurity(t *testing.T) {
	svc := NewService()
	now := time.Unix(1710000000, 0).UTC()

	revoked := svc.RevokeSession("sess-1", "manual revoke", now)
	if !revoked.Revoked || revoked.Reason != "manual revoke" {
		t.Fatalf("expected revoked session status, got %+v", revoked)
	}

	anomaly := svc.ReportSessionAnomaly("sess-1", "geo_jump", "ip region changed", now.Add(10*time.Minute))
	if anomaly.Category != "geo_jump" {
		t.Fatalf("unexpected anomaly: %+v", anomaly)
	}

	status := svc.SessionStatus("sess-1")
	if !status.Revoked || status.AnomalyCount != 1 {
		t.Fatalf("unexpected session status: %+v", status)
	}
}
