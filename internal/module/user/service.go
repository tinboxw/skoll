package user

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

var ErrUserNotFound = errors.New("user not found")
var ErrPasswordRotationTooFrequent = errors.New("password rotation too frequent")
var ErrAuthSessionNotFound = errors.New("auth session not found")
var ErrAuthInvalidRefreshToken = errors.New("invalid refresh token")
var ErrAuthRefreshTokenExpired = errors.New("refresh token expired")
var ErrAuthSessionRevoked = errors.New("auth session revoked")

const (
	defaultJWTIssuer      = "skoll-admin"
	defaultJWTClaimsVer   = "v1"
	defaultJWTAccessTTL   = 15 * time.Minute
	defaultJWTRefreshTTL  = 24 * time.Hour
	defaultJWTSigningSalt = "skoll-dev-jwt-secret"
)

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

type SessionConsistency struct {
	SessionID            string `json:"session_id"`
	Version              int64  `json:"version"`
	WriterInstance       string `json:"writer_instance,omitempty"`
	LastHeartbeatUnixSec int64  `json:"last_heartbeat_unix_sec,omitempty"`
	ConflictCount        int    `json:"conflict_count"`
	Consistent           bool   `json:"consistent"`
	LastConflictReason   string `json:"last_conflict_reason,omitempty"`
}

type AuthTokenPair struct {
	TokenType           string `json:"token_type"`
	SessionID           string `json:"session_id"`
	AccessToken         string `json:"access_token"`
	RefreshToken        string `json:"refresh_token"`
	ExpiresInSec        int64  `json:"expires_in_sec"`
	RefreshExpiresInSec int64  `json:"refresh_expires_in_sec"`
	IssuedAtUnixSec     int64  `json:"issued_at_unix_sec"`
}

type AuthSession struct {
	SessionID               string `json:"session_id"`
	UserID                  int64  `json:"user_id"`
	RoleID                  int64  `json:"role_id"`
	Subject                 string `json:"subject"`
	ClaimsVersion           string `json:"claims_version"`
	IssuedAtUnixSec         int64  `json:"issued_at_unix_sec"`
	AccessExpiresAtUnixSec  int64  `json:"access_expires_at_unix_sec"`
	RefreshExpiresAtUnixSec int64  `json:"refresh_expires_at_unix_sec"`
	Revoked                 bool   `json:"revoked"`
	RevokedAtUnixSec        int64  `json:"revoked_at_unix_sec,omitempty"`
	RevokeReason            string `json:"revoke_reason,omitempty"`
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

type sessionConsistencyRecord struct {
	version            int64
	writerInstance     string
	lastHeartbeatAt    time.Time
	conflictCount      int
	lastConflictReason string
}

type authSessionRecord struct {
	sessionID        string
	userID           int64
	roleID           int64
	subject          string
	claimsVersion    string
	issuedAt         time.Time
	accessExpiresAt  time.Time
	refreshExpiresAt time.Time
	refreshTokenHash string
	revoked          bool
	revokedAt        time.Time
	revokeReason     string
}

type Service struct {
	mu                 sync.RWMutex
	nextID             int64
	items              map[int64]User
	security           map[int64]userSecurity
	sessions           map[string]sessionRecord
	sessionConsistency map[string]sessionConsistencyRecord
	authSessions       map[string]authSessionRecord
	refreshIndex       map[string]string
	jwtSigningSecret   []byte
	jwtIssuer          string
	jwtAccessTTL       time.Duration
	jwtRefreshTTL      time.Duration
}

func NewService() *Service {
	return &Service{
		nextID:             1,
		items:              make(map[int64]User),
		security:           make(map[int64]userSecurity),
		sessions:           make(map[string]sessionRecord),
		sessionConsistency: make(map[string]sessionConsistencyRecord),
		authSessions:       make(map[string]authSessionRecord),
		refreshIndex:       make(map[string]string),
		jwtSigningSecret:   []byte(defaultJWTSigningSalt),
		jwtIssuer:          defaultJWTIssuer,
		jwtAccessTTL:       defaultJWTAccessTTL,
		jwtRefreshTTL:      defaultJWTRefreshTTL,
	}
}

func (s *Service) CreateAuthSession(userID, roleID int64, claimsVersion string, now time.Time) (AuthTokenPair, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	u, ok := s.items[userID]
	if !ok {
		return AuthTokenPair{}, ErrUserNotFound
	}

	issuedAt := now.UTC()
	sessionID, err := newToken(18)
	if err != nil {
		return AuthTokenPair{}, fmt.Errorf("create session id: %w", err)
	}
	refreshToken, err := newToken(32)
	if err != nil {
		return AuthTokenPair{}, fmt.Errorf("create refresh token: %w", err)
	}
	refreshHash := tokenHash(refreshToken)
	claimsVersion = strings.ToLower(strings.TrimSpace(claimsVersion))
	if claimsVersion == "" {
		claimsVersion = defaultJWTClaimsVer
	}

	rec := authSessionRecord{
		sessionID:        sessionID,
		userID:           userID,
		roleID:           roleID,
		subject:          u.Email,
		claimsVersion:    claimsVersion,
		issuedAt:         issuedAt,
		accessExpiresAt:  issuedAt.Add(s.jwtAccessTTL),
		refreshExpiresAt: issuedAt.Add(s.jwtRefreshTTL),
		refreshTokenHash: refreshHash,
	}
	s.authSessions[sessionID] = rec
	s.refreshIndex[refreshHash] = sessionID

	accessToken, err := s.signAccessToken(rec)
	if err != nil {
		delete(s.authSessions, sessionID)
		delete(s.refreshIndex, refreshHash)
		return AuthTokenPair{}, fmt.Errorf("sign access token: %w", err)
	}

	return AuthTokenPair{
		TokenType:           "Bearer",
		SessionID:           sessionID,
		AccessToken:         accessToken,
		RefreshToken:        refreshToken,
		ExpiresInSec:        int64(s.jwtAccessTTL.Seconds()),
		RefreshExpiresInSec: int64(s.jwtRefreshTTL.Seconds()),
		IssuedAtUnixSec:     issuedAt.Unix(),
	}, nil
}

func (s *Service) RefreshAuthSession(refreshToken string, now time.Time) (AuthTokenPair, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	refreshToken = strings.TrimSpace(refreshToken)
	if refreshToken == "" {
		return AuthTokenPair{}, ErrAuthInvalidRefreshToken
	}

	hash := tokenHash(refreshToken)
	sessionID, ok := s.refreshIndex[hash]
	if !ok {
		return AuthTokenPair{}, ErrAuthInvalidRefreshToken
	}

	rec, ok := s.authSessions[sessionID]
	if !ok {
		delete(s.refreshIndex, hash)
		return AuthTokenPair{}, ErrAuthSessionNotFound
	}
	if rec.revoked {
		return AuthTokenPair{}, ErrAuthSessionRevoked
	}
	now = now.UTC()
	if now.After(rec.refreshExpiresAt) {
		delete(s.refreshIndex, hash)
		return AuthTokenPair{}, ErrAuthRefreshTokenExpired
	}

	newRefreshToken, err := newToken(32)
	if err != nil {
		return AuthTokenPair{}, fmt.Errorf("create refresh token: %w", err)
	}
	newHash := tokenHash(newRefreshToken)

	delete(s.refreshIndex, rec.refreshTokenHash)
	rec.refreshTokenHash = newHash
	rec.issuedAt = now
	rec.accessExpiresAt = now.Add(s.jwtAccessTTL)
	s.authSessions[sessionID] = rec
	s.refreshIndex[newHash] = sessionID

	accessToken, err := s.signAccessToken(rec)
	if err != nil {
		return AuthTokenPair{}, fmt.Errorf("sign access token: %w", err)
	}

	return AuthTokenPair{
		TokenType:           "Bearer",
		SessionID:           sessionID,
		AccessToken:         accessToken,
		RefreshToken:        newRefreshToken,
		ExpiresInSec:        int64(s.jwtAccessTTL.Seconds()),
		RefreshExpiresInSec: int64(time.Until(rec.refreshExpiresAt).Seconds()),
		IssuedAtUnixSec:     rec.issuedAt.Unix(),
	}, nil
}

func (s *Service) RevokeAuthSession(sessionID, reason string, now time.Time) (AuthSession, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	rec, ok := s.authSessions[sessionID]
	if !ok {
		return AuthSession{}, ErrAuthSessionNotFound
	}
	rec.revoked = true
	rec.revokeReason = strings.TrimSpace(reason)
	if rec.revokeReason == "" {
		rec.revokeReason = "logout"
	}
	rec.revokedAt = now.UTC()
	s.authSessions[sessionID] = rec
	delete(s.refreshIndex, rec.refreshTokenHash)

	return toAuthSession(rec), nil
}

func (s *Service) GetAuthSession(sessionID string) (AuthSession, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rec, ok := s.authSessions[sessionID]
	if !ok {
		return AuthSession{}, ErrAuthSessionNotFound
	}
	return toAuthSession(rec), nil
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

func (s *Service) HeartbeatSessionConsistency(sessionID, instanceID string, version int64, now time.Time) SessionConsistency {
	s.mu.Lock()
	defer s.mu.Unlock()

	if version <= 0 {
		version = 1
	}

	rec := s.sessionConsistency[sessionID]
	conflict := false
	conflictReason := ""

	if rec.version > 0 {
		switch {
		case version < rec.version:
			conflict = true
			conflictReason = "stale_version"
		case version == rec.version && rec.writerInstance != "" && instanceID != "" && rec.writerInstance != instanceID:
			conflict = true
			conflictReason = "writer_conflict"
		}
	}

	if conflict {
		rec.conflictCount++
		rec.lastConflictReason = conflictReason
		s.sessionConsistency[sessionID] = rec
		return toSessionConsistency(sessionID, rec, false)
	}

	rec.version = version
	rec.writerInstance = instanceID
	rec.lastHeartbeatAt = now.UTC()
	s.sessionConsistency[sessionID] = rec
	return toSessionConsistency(sessionID, rec, true)
}

func (s *Service) SessionConsistencyStatus(sessionID string) SessionConsistency {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return toSessionConsistency(sessionID, s.sessionConsistency[sessionID], true)
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

func toSessionConsistency(sessionID string, rec sessionConsistencyRecord, consistent bool) SessionConsistency {
	out := SessionConsistency{
		SessionID:          sessionID,
		Version:            rec.version,
		WriterInstance:     rec.writerInstance,
		ConflictCount:      rec.conflictCount,
		Consistent:         consistent,
		LastConflictReason: rec.lastConflictReason,
	}
	if !rec.lastHeartbeatAt.IsZero() {
		out.LastHeartbeatUnixSec = rec.lastHeartbeatAt.Unix()
	}
	return out
}

func toAuthSession(rec authSessionRecord) AuthSession {
	out := AuthSession{
		SessionID:               rec.sessionID,
		UserID:                  rec.userID,
		RoleID:                  rec.roleID,
		Subject:                 rec.subject,
		ClaimsVersion:           rec.claimsVersion,
		IssuedAtUnixSec:         rec.issuedAt.Unix(),
		AccessExpiresAtUnixSec:  rec.accessExpiresAt.Unix(),
		RefreshExpiresAtUnixSec: rec.refreshExpiresAt.Unix(),
		Revoked:                 rec.revoked,
		RevokeReason:            rec.revokeReason,
	}
	if !rec.revokedAt.IsZero() {
		out.RevokedAtUnixSec = rec.revokedAt.Unix()
	}
	return out
}

func (s *Service) signAccessToken(rec authSessionRecord) (string, error) {
	headerJSON, err := json.Marshal(map[string]string{"alg": "HS256", "typ": "JWT"})
	if err != nil {
		return "", err
	}
	payloadJSON, err := json.Marshal(map[string]any{
		"iss":            s.jwtIssuer,
		"sub":            rec.subject,
		"sid":            rec.sessionID,
		"uid":            rec.userID,
		"role_id":        rec.roleID,
		"claims_version": rec.claimsVersion,
		"iat":            rec.issuedAt.Unix(),
		"exp":            rec.accessExpiresAt.Unix(),
	})
	if err != nil {
		return "", err
	}

	encodedHeader := base64.RawURLEncoding.EncodeToString(headerJSON)
	encodedPayload := base64.RawURLEncoding.EncodeToString(payloadJSON)
	unsigned := encodedHeader + "." + encodedPayload

	mac := hmac.New(sha256.New, s.jwtSigningSecret)
	_, _ = mac.Write([]byte(unsigned))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))

	return unsigned + "." + sig, nil
}

func newToken(size int) (string, error) {
	b := make([]byte, size)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func tokenHash(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
