package user

import (
	"errors"
	"sort"
	"sync"
	"time"
)

var ErrUserNotFound = errors.New("user not found")
var ErrPasswordRotationTooFrequent = errors.New("password rotation too frequent")

type User struct {
	ID        int64
	Name      string
	Email     string
	Active    bool
	CreatedAt time.Time
}

type SecurityState struct {
	UserID                 int64     `json:"user_id"`
	PasswordRotatedAt      time.Time `json:"password_rotated_at,omitempty"`
	FailedLoginCount       int       `json:"failed_login_count"`
	LockedUntilUnixSec     int64     `json:"locked_until_unix_sec,omitempty"`
	MFAEnabled             bool      `json:"mfa_enabled"`
	MFAProvider            string    `json:"mfa_provider,omitempty"`
	LastSecurityActionUnix int64     `json:"last_security_action_unix_sec"`
}

type SessionStatus struct {
	SessionID          string `json:"session_id"`
	Revoked            bool   `json:"revoked"`
	Reason             string `json:"reason,omitempty"`
	RevokedAtUnixSec   int64  `json:"revoked_at_unix_sec,omitempty"`
	AnomalyCount       int    `json:"anomaly_count"`
	LastAnomalyUnixSec int64  `json:"last_anomaly_unix_sec,omitempty"`
}

type SessionAnomaly struct {
	SessionID    string `json:"session_id"`
	Category     string `json:"category"`
	Detail       string `json:"detail"`
	ReportedUnix int64  `json:"reported_unix_sec"`
}

type userSecurity struct {
	passwordRotatedAt time.Time
	failedLoginCount  int
	lockedUntil       time.Time
	mfaEnabled        bool
	mfaProvider       string
	lastActionAt      time.Time
}

type sessionRecord struct {
	revoked         bool
	reason          string
	revokedAt       time.Time
	anomalyCount    int
	lastAnomalyAt   time.Time
	lastAnomalyType string
}

type Service struct {
	mu       sync.RWMutex
	nextID   int64
	items    map[int64]User
	security map[int64]userSecurity
	sessions map[string]sessionRecord
}

func NewService() *Service {
	return &Service{nextID: 1, items: make(map[int64]User), security: make(map[int64]userSecurity), sessions: make(map[string]sessionRecord)}
}

func (s *Service) Create(name, email string) User {
	s.mu.Lock()
	defer s.mu.Unlock()

	u := User{
		ID:        s.nextID,
		Name:      name,
		Email:     email,
		Active:    true,
		CreatedAt: time.Now().UTC(),
	}
	s.nextID++
	s.items[u.ID] = u
	s.security[u.ID] = userSecurity{lastActionAt: time.Now().UTC()}
	return u
}

func (s *Service) Get(id int64) (User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	u, ok := s.items[id]
	if !ok {
		return User{}, ErrUserNotFound
	}
	return u, nil
}

func (s *Service) List() []User {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]User, 0, len(s.items))
	for _, u := range s.items {
		out = append(out, u)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func (s *Service) RotatePassword(userID int64, minInterval time.Duration, now time.Time) (SecurityState, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.items[userID]; !ok {
		return SecurityState{}, ErrUserNotFound
	}
	state := s.security[userID]
	if !state.passwordRotatedAt.IsZero() && minInterval > 0 && now.Sub(state.passwordRotatedAt) < minInterval {
		return SecurityState{}, ErrPasswordRotationTooFrequent
	}
	state.passwordRotatedAt = now.UTC()
	state.lastActionAt = now.UTC()
	s.security[userID] = state
	return toSecurityState(userID, state), nil
}

func (s *Service) RegisterLoginFailure(userID int64, lockThreshold int, lockDuration time.Duration, now time.Time) (SecurityState, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.items[userID]; !ok {
		return SecurityState{}, ErrUserNotFound
	}
	if lockThreshold <= 0 {
		lockThreshold = 5
	}
	state := s.security[userID]
	state.failedLoginCount++
	if state.failedLoginCount >= lockThreshold && lockDuration > 0 {
		state.lockedUntil = now.UTC().Add(lockDuration)
	}
	state.lastActionAt = now.UTC()
	s.security[userID] = state
	return toSecurityState(userID, state), nil
}

func (s *Service) ResetUserLock(userID int64, now time.Time) (SecurityState, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.items[userID]; !ok {
		return SecurityState{}, ErrUserNotFound
	}
	state := s.security[userID]
	state.failedLoginCount = 0
	state.lockedUntil = time.Time{}
	state.lastActionAt = now.UTC()
	s.security[userID] = state
	return toSecurityState(userID, state), nil
}

func (s *Service) SetMFA(userID int64, enabled bool, provider string, now time.Time) (SecurityState, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.items[userID]; !ok {
		return SecurityState{}, ErrUserNotFound
	}
	state := s.security[userID]
	state.mfaEnabled = enabled
	if enabled {
		state.mfaProvider = provider
	} else {
		state.mfaProvider = ""
	}
	state.lastActionAt = now.UTC()
	s.security[userID] = state
	return toSecurityState(userID, state), nil
}

func (s *Service) RevokeSession(sessionID, reason string, now time.Time) SessionStatus {
	s.mu.Lock()
	defer s.mu.Unlock()
	rec := s.sessions[sessionID]
	rec.revoked = true
	rec.reason = reason
	rec.revokedAt = now.UTC()
	s.sessions[sessionID] = rec
	return toSessionStatus(sessionID, rec)
}

func (s *Service) SessionStatus(sessionID string) SessionStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return toSessionStatus(sessionID, s.sessions[sessionID])
}

func (s *Service) ReportSessionAnomaly(sessionID, category, detail string, now time.Time) SessionAnomaly {
	s.mu.Lock()
	defer s.mu.Unlock()
	rec := s.sessions[sessionID]
	rec.anomalyCount++
	rec.lastAnomalyAt = now.UTC()
	rec.lastAnomalyType = category
	s.sessions[sessionID] = rec
	return SessionAnomaly{SessionID: sessionID, Category: category, Detail: detail, ReportedUnix: now.UTC().Unix()}
}

func toSecurityState(userID int64, state userSecurity) SecurityState {
	out := SecurityState{
		UserID:                 userID,
		FailedLoginCount:       state.failedLoginCount,
		MFAEnabled:             state.mfaEnabled,
		MFAProvider:            state.mfaProvider,
		LastSecurityActionUnix: state.lastActionAt.Unix(),
	}
	if !state.passwordRotatedAt.IsZero() {
		out.PasswordRotatedAt = state.passwordRotatedAt
	}
	if !state.lockedUntil.IsZero() {
		out.LockedUntilUnixSec = state.lockedUntil.Unix()
	}
	return out
}

func toSessionStatus(sessionID string, rec sessionRecord) SessionStatus {
	out := SessionStatus{SessionID: sessionID, Revoked: rec.revoked, Reason: rec.reason, AnomalyCount: rec.anomalyCount}
	if !rec.revokedAt.IsZero() {
		out.RevokedAtUnixSec = rec.revokedAt.Unix()
	}
	if !rec.lastAnomalyAt.IsZero() {
		out.LastAnomalyUnixSec = rec.lastAnomalyAt.Unix()
	}
	return out
}
