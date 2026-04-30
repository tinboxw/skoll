package user

import "time"

// SecurityImport is a normalized, persistence-friendly view of per-user
// security state used when restoring from a SQL adapter.
type SecurityImport struct {
	UserID            int64
	PasswordRotatedAt time.Time
	FailedLoginCount  int
	LockedUntil       time.Time
	MFAEnabled        bool
	MFAProvider       string
	LastActionAt      time.Time
}

// AuthSessionImport mirrors the persisted auth-session row needed to
// rehydrate the service's auth-session map and refresh-token index.
type AuthSessionImport struct {
	SessionID        string
	UserID           int64
	RoleID           int64
	Subject          string
	ClaimsVersion    string
	IssuedAt         time.Time
	AccessExpiresAt  time.Time
	RefreshExpiresAt time.Time
	RefreshTokenHash string
	Revoked          bool
	RevokedAt        time.Time
	RevokeReason     string
}

// UserStateSnapshot bundles the persistent slices of user.Service. Ephemeral
// session anomaly/consistency state is intentionally absent: those are reset
// on restart by design (similar to JWT-only auth).
type UserStateSnapshot struct {
	NextID       int64
	Users        []User
	Security     []SecurityImport
	AuthSessions []AuthSessionImport
}

// ImportSnapshot replaces the persistent slices of in-memory state from a
// previously-persisted snapshot. Used by SQL-backed adapters during
// hydration on boot.
func (s *Service) ImportSnapshot(snap UserStateSnapshot) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.items = make(map[int64]User, len(snap.Users))
	for _, u := range snap.Users {
		s.items[u.ID] = u
	}

	s.security = make(map[int64]userSecurity, len(snap.Security))
	for _, sec := range snap.Security {
		s.security[sec.UserID] = userSecurity{
			passwordRotatedAt: sec.PasswordRotatedAt,
			failedLoginCount:  sec.FailedLoginCount,
			lockedUntil:       sec.LockedUntil,
			mfaEnabled:        sec.MFAEnabled,
			mfaProvider:       sec.MFAProvider,
			lastActionAt:      sec.LastActionAt,
		}
	}

	s.authSessions = make(map[string]authSessionRecord, len(snap.AuthSessions))
	s.refreshIndex = make(map[string]string, len(snap.AuthSessions))
	for _, a := range snap.AuthSessions {
		rec := authSessionRecord{
			sessionID:        a.SessionID,
			userID:           a.UserID,
			roleID:           a.RoleID,
			subject:          a.Subject,
			claimsVersion:    a.ClaimsVersion,
			issuedAt:         a.IssuedAt,
			accessExpiresAt:  a.AccessExpiresAt,
			refreshExpiresAt: a.RefreshExpiresAt,
			refreshTokenHash: a.RefreshTokenHash,
			revoked:          a.Revoked,
			revokedAt:        a.RevokedAt,
			revokeReason:     a.RevokeReason,
		}
		s.authSessions[a.SessionID] = rec
		if !a.Revoked && a.RefreshTokenHash != "" {
			s.refreshIndex[a.RefreshTokenHash] = a.SessionID
		}
	}

	if snap.NextID > s.nextID {
		s.nextID = snap.NextID
	}
}

// ExportSecurity returns a copy of the security state for the given user, or
// false when not present. Used by SQL-backed adapters to capture the
// post-mutation state for persistence.
func (s *Service) ExportSecurity(userID int64) (SecurityImport, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	st, ok := s.security[userID]
	if !ok {
		return SecurityImport{}, false
	}
	return SecurityImport{
		UserID:            userID,
		PasswordRotatedAt: st.passwordRotatedAt,
		FailedLoginCount:  st.failedLoginCount,
		LockedUntil:       st.lockedUntil,
		MFAEnabled:        st.mfaEnabled,
		MFAProvider:       st.mfaProvider,
		LastActionAt:      st.lastActionAt,
	}, true
}

// ExportAuthSession returns the persisted-shape auth-session for the given id.
func (s *Service) ExportAuthSession(sessionID string) (AuthSessionImport, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rec, ok := s.authSessions[sessionID]
	if !ok {
		return AuthSessionImport{}, false
	}
	return AuthSessionImport{
		SessionID:        rec.sessionID,
		UserID:           rec.userID,
		RoleID:           rec.roleID,
		Subject:          rec.subject,
		ClaimsVersion:    rec.claimsVersion,
		IssuedAt:         rec.issuedAt,
		AccessExpiresAt:  rec.accessExpiresAt,
		RefreshExpiresAt: rec.refreshExpiresAt,
		RefreshTokenHash: rec.refreshTokenHash,
		Revoked:          rec.revoked,
		RevokedAt:        rec.revokedAt,
		RevokeReason:     rec.revokeReason,
	}, true
}

// ExportNextID returns the current next-id counter.
func (s *Service) ExportNextID() int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.nextID
}
