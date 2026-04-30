package persistent

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/tinboxw/skoll/internal/module/storageadapter/contracts"
	"github.com/tinboxw/skoll/internal/module/user"
)

// UserModel persists the core user record.
type UserModel struct {
	ID        int64     `gorm:"primaryKey;autoIncrement:false"`
	Name      string    `gorm:"size:255;not null"`
	Email     string    `gorm:"size:255;index"`
	Active    bool      `gorm:"not null;default:true"`
	CreatedAt time.Time `gorm:"not null"`
}

// TableName overrides the default plural form.
func (UserModel) TableName() string { return "skoll_users" }

// UserSecurityModel persists the per-user security counters/flags.
type UserSecurityModel struct {
	UserID            int64     `gorm:"primaryKey;autoIncrement:false"`
	PasswordRotatedAt time.Time `gorm:""`
	FailedLoginCount  int       `gorm:"not null;default:0"`
	LockedUntil       time.Time `gorm:""`
	MFAEnabled        bool      `gorm:"not null;default:false"`
	MFAProvider       string    `gorm:"size:64"`
	LastActionAt      time.Time `gorm:""`
}

func (UserSecurityModel) TableName() string { return "skoll_user_security" }

// AuthSessionModel persists the long-lived JWT refresh-session record.
type AuthSessionModel struct {
	SessionID        string    `gorm:"primaryKey;size:64"`
	UserID           int64     `gorm:"index;not null"`
	RoleID           int64     `gorm:"not null;default:0"`
	Subject          string    `gorm:"size:255"`
	ClaimsVersion    string    `gorm:"size:32"`
	IssuedAt         time.Time `gorm:""`
	AccessExpiresAt  time.Time `gorm:""`
	RefreshExpiresAt time.Time `gorm:""`
	RefreshTokenHash string    `gorm:"size:128;index"`
	Revoked          bool      `gorm:"not null;default:false"`
	RevokedAt        time.Time `gorm:""`
	RevokeReason     string    `gorm:"size:255"`
}

func (AuthSessionModel) TableName() string { return "skoll_auth_sessions" }

// UserCounterModel stores the next-id counter so user IDs continue
// monotonically across restarts.
type UserCounterModel struct {
	ID     string `gorm:"primaryKey;size:32"` // always "user"
	NextID int64  `gorm:"not null;default:1"`
}

func (UserCounterModel) TableName() string { return "skoll_user_counter" }

const userCounterRowID = "user"

// UserRepository implements contracts.UserRepository using a write-through
// pattern over user.NewService(). The persistent slices (User, Security,
// AuthSession, NextID) are written to SQL on every mutation. Ephemeral
// session anomaly/consistency state is intentionally kept in memory only,
// matching the contract for those signals.
type UserRepository struct {
	db    *gorm.DB
	mu    sync.Mutex
	inner *user.Service
}

// NewUserRepository constructs a SQL-backed user repository, hydrating the
// in-memory state from the database. The required tables must already have
// been migrated — call repo.Models() (or db.Migrate) beforehand.
func NewUserRepository(database *gorm.DB) (*UserRepository, error) {
	r := &UserRepository{
		db:    database,
		inner: user.NewService(),
	}
	if database == nil {
		return nil, fmt.Errorf("persistent: nil db")
	}
	if err := r.hydrate(); err != nil {
		return nil, err
	}
	return r, nil
}

// Models returns the gorm models managed by this repository for migration.
func (r *UserRepository) Models() []any {
	return []any{
		&UserModel{},
		&UserSecurityModel{},
		&AuthSessionModel{},
		&UserCounterModel{},
	}
}

func (r *UserRepository) ctx() context.Context { return context.Background() }

func (r *UserRepository) hydrate() error {
	var (
		users    []UserModel
		secs     []UserSecurityModel
		sessions []AuthSessionModel
		counter  UserCounterModel
	)

	if err := r.db.WithContext(r.ctx()).Find(&users).Error; err != nil {
		return fmt.Errorf("hydrate users: %w", err)
	}
	if err := r.db.WithContext(r.ctx()).Find(&secs).Error; err != nil {
		return fmt.Errorf("hydrate user security: %w", err)
	}
	if err := r.db.WithContext(r.ctx()).Find(&sessions).Error; err != nil {
		return fmt.Errorf("hydrate auth sessions: %w", err)
	}
	if err := r.db.WithContext(r.ctx()).First(&counter, "id = ?", userCounterRowID).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("hydrate user counter: %w", err)
	}

	snap := user.UserStateSnapshot{NextID: counter.NextID}
	for _, u := range users {
		snap.Users = append(snap.Users, user.User{
			ID:        u.ID,
			Name:      u.Name,
			Email:     u.Email,
			Active:    u.Active,
			CreatedAt: u.CreatedAt,
		})
	}
	for _, s := range secs {
		snap.Security = append(snap.Security, user.SecurityImport{
			UserID:            s.UserID,
			PasswordRotatedAt: s.PasswordRotatedAt,
			FailedLoginCount:  s.FailedLoginCount,
			LockedUntil:       s.LockedUntil,
			MFAEnabled:        s.MFAEnabled,
			MFAProvider:       s.MFAProvider,
			LastActionAt:      s.LastActionAt,
		})
	}
	for _, a := range sessions {
		snap.AuthSessions = append(snap.AuthSessions, user.AuthSessionImport{
			SessionID:        a.SessionID,
			UserID:           a.UserID,
			RoleID:           a.RoleID,
			Subject:          a.Subject,
			ClaimsVersion:    a.ClaimsVersion,
			IssuedAt:         a.IssuedAt,
			AccessExpiresAt:  a.AccessExpiresAt,
			RefreshExpiresAt: a.RefreshExpiresAt,
			RefreshTokenHash: a.RefreshTokenHash,
			Revoked:          a.Revoked,
			RevokedAt:        a.RevokedAt,
			RevokeReason:     a.RevokeReason,
		})
	}
	r.inner.ImportSnapshot(snap)
	return nil
}

// --- write-through helpers --------------------------------------------------

func (r *UserRepository) saveUser(u user.User) {
	row := UserModel{
		ID:        u.ID,
		Name:      u.Name,
		Email:     u.Email,
		Active:    u.Active,
		CreatedAt: u.CreatedAt,
	}
	if err := r.db.WithContext(r.ctx()).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{"name", "email", "active", "created_at"}),
	}).Create(&row).Error; err != nil {
		panic(fmt.Errorf("persist user: %w", err))
	}
}

func (r *UserRepository) saveSecurity(userID int64) {
	imp, ok := r.inner.ExportSecurity(userID)
	if !ok {
		return
	}
	row := UserSecurityModel{
		UserID:            imp.UserID,
		PasswordRotatedAt: imp.PasswordRotatedAt,
		FailedLoginCount:  imp.FailedLoginCount,
		LockedUntil:       imp.LockedUntil,
		MFAEnabled:        imp.MFAEnabled,
		MFAProvider:       imp.MFAProvider,
		LastActionAt:      imp.LastActionAt,
	}
	if err := r.db.WithContext(r.ctx()).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "user_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"password_rotated_at", "failed_login_count", "locked_until",
			"mfa_enabled", "mfa_provider", "last_action_at",
		}),
	}).Create(&row).Error; err != nil {
		panic(fmt.Errorf("persist user security: %w", err))
	}
}

func (r *UserRepository) saveAuthSession(sessionID string) {
	imp, ok := r.inner.ExportAuthSession(sessionID)
	if !ok {
		return
	}
	row := AuthSessionModel{
		SessionID:        imp.SessionID,
		UserID:           imp.UserID,
		RoleID:           imp.RoleID,
		Subject:          imp.Subject,
		ClaimsVersion:    imp.ClaimsVersion,
		IssuedAt:         imp.IssuedAt,
		AccessExpiresAt:  imp.AccessExpiresAt,
		RefreshExpiresAt: imp.RefreshExpiresAt,
		RefreshTokenHash: imp.RefreshTokenHash,
		Revoked:          imp.Revoked,
		RevokedAt:        imp.RevokedAt,
		RevokeReason:     imp.RevokeReason,
	}
	if err := r.db.WithContext(r.ctx()).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "session_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"user_id", "role_id", "subject", "claims_version", "issued_at",
			"access_expires_at", "refresh_expires_at", "refresh_token_hash",
			"revoked", "revoked_at", "revoke_reason",
		}),
	}).Create(&row).Error; err != nil {
		panic(fmt.Errorf("persist auth session: %w", err))
	}
}

func (r *UserRepository) saveCounter() {
	next := r.inner.ExportNextID()
	row := UserCounterModel{ID: userCounterRowID, NextID: next}
	if err := r.db.WithContext(r.ctx()).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{"next_id"}),
	}).Create(&row).Error; err != nil {
		panic(fmt.Errorf("persist user counter: %w", err))
	}
}

// --- contracts.UserRepository ----------------------------------------------

func (r *UserRepository) Create(name, email string) user.User {
	r.mu.Lock()
	defer r.mu.Unlock()
	u := r.inner.Create(name, email)
	r.saveUser(u)
	r.saveSecurity(u.ID)
	r.saveCounter()
	return u
}

func (r *UserRepository) Get(id int64) (user.User, error) { return r.inner.Get(id) }
func (r *UserRepository) List() []user.User               { return r.inner.List() }

func (r *UserRepository) RotatePassword(userID int64, minInterval time.Duration, now time.Time) (user.SecurityState, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	st, err := r.inner.RotatePassword(userID, minInterval, now)
	if err != nil {
		return st, err
	}
	r.saveSecurity(userID)
	return st, nil
}

func (r *UserRepository) RegisterLoginFailure(userID int64, lockThreshold int, lockDuration time.Duration, now time.Time) (user.SecurityState, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	st, err := r.inner.RegisterLoginFailure(userID, lockThreshold, lockDuration, now)
	if err != nil {
		return st, err
	}
	r.saveSecurity(userID)
	return st, nil
}

func (r *UserRepository) ResetUserLock(userID int64, now time.Time) (user.SecurityState, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	st, err := r.inner.ResetUserLock(userID, now)
	if err != nil {
		return st, err
	}
	r.saveSecurity(userID)
	return st, nil
}

func (r *UserRepository) SetMFA(userID int64, enabled bool, provider string, now time.Time) (user.SecurityState, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	st, err := r.inner.SetMFA(userID, enabled, provider, now)
	if err != nil {
		return st, err
	}
	r.saveSecurity(userID)
	return st, nil
}

// Ephemeral session signals: not persisted, by design.

func (r *UserRepository) RevokeSession(sessionID, reason string, now time.Time) user.SessionStatus {
	return r.inner.RevokeSession(sessionID, reason, now)
}

func (r *UserRepository) SessionStatus(sessionID string) user.SessionStatus {
	return r.inner.SessionStatus(sessionID)
}

func (r *UserRepository) ReportSessionAnomaly(sessionID, category, detail string, now time.Time) user.SessionAnomaly {
	return r.inner.ReportSessionAnomaly(sessionID, category, detail, now)
}

func (r *UserRepository) HeartbeatSessionConsistency(sessionID, instanceID string, version int64, now time.Time) user.SessionConsistency {
	return r.inner.HeartbeatSessionConsistency(sessionID, instanceID, version, now)
}

func (r *UserRepository) SessionConsistencyStatus(sessionID string) user.SessionConsistency {
	return r.inner.SessionConsistencyStatus(sessionID)
}

// Auth session methods are write-through.

func (r *UserRepository) CreateAuthSession(userID, roleID int64, claimsVersion string, now time.Time) (user.AuthTokenPair, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	pair, err := r.inner.CreateAuthSession(userID, roleID, claimsVersion, now)
	if err != nil {
		return pair, err
	}
	r.saveAuthSession(pair.SessionID)
	return pair, nil
}

func (r *UserRepository) RefreshAuthSession(refreshToken string, now time.Time) (user.AuthTokenPair, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	pair, err := r.inner.RefreshAuthSession(refreshToken, now)
	if err != nil {
		return pair, err
	}
	r.saveAuthSession(pair.SessionID)
	return pair, nil
}

func (r *UserRepository) RevokeAuthSession(sessionID, reason string, now time.Time) (user.AuthSession, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	sess, err := r.inner.RevokeAuthSession(sessionID, reason, now)
	if err != nil {
		return sess, err
	}
	r.saveAuthSession(sessionID)
	return sess, nil
}

func (r *UserRepository) GetAuthSession(sessionID string) (user.AuthSession, error) {
	return r.inner.GetAuthSession(sessionID)
}

// Compile-time check the contract is satisfied.
var _ contracts.UserRepository = (*UserRepository)(nil)
