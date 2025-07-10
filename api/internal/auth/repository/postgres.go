package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hexabase/hexabase-ai/api/internal/auth/domain"
	"gorm.io/gorm"
)

type postgresRepository struct {
	db                  *gorm.DB
	tokenHashRepository *tokenHashRepository
}

// NewPostgresRepository creates a new PostgreSQL auth repository
func NewPostgresRepository(db *gorm.DB) *postgresRepository {
	return &postgresRepository{
		db: db,
	}
}

// User operations

func (r *postgresRepository) CreateUser(ctx context.Context, user *domain.User) error {
	if err := r.db.WithContext(ctx).Create(user).Error; err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}
	return nil
}

func (r *postgresRepository) GetUser(ctx context.Context, userID string) (*domain.User, error) {
	var user domain.User
	if err := r.db.WithContext(ctx).Where("id = ?", userID).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("failed to get user %s: %w", userID, domain.ErrUserNotFound)
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return &user, nil
}

func (r *postgresRepository) GetUserByExternalID(ctx context.Context, externalID, provider string) (*domain.User, error) {
	var user domain.User
	if err := r.db.WithContext(ctx).
		Where("external_id = ? AND provider = ?", externalID, provider).
		First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("failed to get user by external ID %s (provider: %s): %w", externalID, provider, domain.ErrUserNotFound)
		}
		return nil, fmt.Errorf("failed to get user by external ID: %w", err)
	}
	return &user, nil
}

func (r *postgresRepository) GetUserByEmail(ctx context.Context, email string) (*domain.User, error) {
	var user domain.User
	if err := r.db.WithContext(ctx).Where("email = ?", email).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("failed to get user by email %s: %w", email, domain.ErrUserNotFound)
		}
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}
	return &user, nil
}

func (r *postgresRepository) UpdateUser(ctx context.Context, user *domain.User) error {
	if err := r.db.WithContext(ctx).Save(user).Error; err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}
	return nil
}

func (r *postgresRepository) UpdateLastLogin(ctx context.Context, userID string) error {
	if err := r.db.WithContext(ctx).
		Model(&domain.User{}).
		Where("id = ?", userID).
		Update("last_login_at", time.Now()).Error; err != nil {
		return fmt.Errorf("failed to update last login: %w", err)
	}
	return nil
}

// Session operations

func (r *postgresRepository) CreateSession(ctx context.Context, session *domain.Session) error {
	// Repository should receive already-hashed token from service layer
	// Service layer is responsible for business logic and token hashing
	if err := r.db.WithContext(ctx).Create(session).Error; err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}
	return nil
}

func (r *postgresRepository) GetSession(ctx context.Context, sessionID string) (*domain.Session, error) {
	var session domain.Session
	if err := r.db.WithContext(ctx).Where("id = ?", sessionID).First(&session).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("failed to get session %s: %w", sessionID, domain.ErrSessionNotFound)
		}
		return nil, fmt.Errorf("failed to get session: %w", err)
	}
	return &session, nil
}

func (r *postgresRepository) GetSessionByRefreshTokenSelector(ctx context.Context, selector string) (*domain.Session, error) {
	var session domain.Session
	if err := r.db.WithContext(ctx).Where("refresh_token_selector = ?", selector).First(&session).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("session not found")
		}
		return nil, fmt.Errorf("failed to get session by selector: %w", err)
	}
	return &session, nil
}

func (r *postgresRepository) GetAllActiveSessions(ctx context.Context) ([]*domain.Session, error) {
	var sessions []*domain.Session
	if err := r.db.WithContext(ctx).Where("revoked = ? AND expires_at > ?", false, time.Now()).Find(&sessions).Error; err != nil {
		return nil, fmt.Errorf("failed to get active sessions: %w", err)
	}
	return sessions, nil
}

func (r *postgresRepository) ListUserSessions(ctx context.Context, userID string) ([]*domain.Session, error) {
	var sessions []*domain.Session
	if err := r.db.WithContext(ctx).
		Where("user_id = ? AND revoked = ? AND expires_at > ?", userID, false, time.Now()).
		Order("created_at DESC").
		Find(&sessions).Error; err != nil {
		return nil, fmt.Errorf("failed to list user sessions: %w", err)
	}
	return sessions, nil
}

func (r *postgresRepository) UpdateSession(ctx context.Context, session *domain.Session) error {
	if err := r.db.WithContext(ctx).Save(session).Error; err != nil {
		return fmt.Errorf("failed to update session: %w", err)
	}
	return nil
}

func (r *postgresRepository) DeleteSession(ctx context.Context, sessionID string) error {
	if err := r.db.WithContext(ctx).Where("id = ?", sessionID).Delete(&domain.Session{}).Error; err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}
	return nil
}

func (r *postgresRepository) DeleteUserSessions(ctx context.Context, userID string, exceptSessionID string) error {
	query := r.db.WithContext(ctx).Where("user_id = ?", userID)
	if exceptSessionID != "" {
		query = query.Where("id != ?", exceptSessionID)
	}

	if err := query.Delete(&domain.Session{}).Error; err != nil {
		return fmt.Errorf("failed to delete user sessions: %w", err)
	}
	return nil
}

func (r *postgresRepository) CleanupExpiredSessions(ctx context.Context, before time.Time) error {
	if err := r.db.WithContext(ctx).
		Where("expires_at < ?", before).
		Delete(&domain.Session{}).Error; err != nil {
		return fmt.Errorf("failed to cleanup expired sessions: %w", err)
	}
	return nil
}

// Auth state operations (Redis) - These would typically be in a Redis repository
// For now, using PostgreSQL with TTL

func (r *postgresRepository) StoreAuthState(ctx context.Context, state *domain.AuthState) error {
	if err := r.db.WithContext(ctx).Create(state).Error; err != nil {
		return fmt.Errorf("failed to store auth state: %w", err)
	}

	// Schedule cleanup
	go r.scheduleAuthStateCleanup(state.State, state.ExpiresAt)

	return nil
}

func (r *postgresRepository) GetAuthState(ctx context.Context, stateValue string) (*domain.AuthState, error) {
	var state domain.AuthState
	if err := r.db.WithContext(ctx).
		Where("state = ? AND expires_at > ?", stateValue, time.Now()).
		First(&state).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("failed to get auth state %s: %w", stateValue, domain.ErrAuthStateNotFound)
		}
		return nil, fmt.Errorf("failed to get auth state: %w", err)
	}
	return &state, nil
}

func (r *postgresRepository) DeleteAuthState(ctx context.Context, stateValue string) error {
	if err := r.db.WithContext(ctx).Where("state = ?", stateValue).Delete(&domain.AuthState{}).Error; err != nil {
		return fmt.Errorf("failed to delete auth state: %w", err)
	}
	return nil
}

// Refresh token operations (Redis) - These would typically be in a Redis repository

func (r *postgresRepository) BlacklistRefreshToken(ctx context.Context, token string, expiresAt time.Time) error {
	blacklist := &RefreshTokenBlacklist{
		ID:        uuid.New().String(),
		Token:     token,
		ExpiresAt: expiresAt,
	}

	if err := r.db.WithContext(ctx).Create(blacklist).Error; err != nil {
		return fmt.Errorf("failed to blacklist refresh token: %w", err)
	}

	// Schedule cleanup
	go r.scheduleBlacklistCleanup(token, expiresAt)

	return nil
}

func (r *postgresRepository) IsRefreshTokenBlacklisted(ctx context.Context, token string) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&RefreshTokenBlacklist{}).
		Where("token = ? AND expires_at > ?", token, time.Now()).
		Count(&count).Error; err != nil {
		return false, fmt.Errorf("failed to check blacklist: %w", err)
	}
	return count > 0, nil
}

// Security event operations

func (r *postgresRepository) CreateSecurityEvent(ctx context.Context, event *domain.SecurityEvent) error {
	if err := r.db.WithContext(ctx).Create(event).Error; err != nil {
		return fmt.Errorf("failed to create security event: %w", err)
	}
	return nil
}

func (r *postgresRepository) ListSecurityEvents(ctx context.Context, filter domain.SecurityLogFilter) ([]*domain.SecurityEvent, error) {
	var events []*domain.SecurityEvent

	query := r.db.WithContext(ctx).Model(&domain.SecurityEvent{})

	if filter.UserID != "" {
		query = query.Where("user_id = ?", filter.UserID)
	}

	if filter.EventType != "" {
		query = query.Where("event_type = ?", filter.EventType)
	}

	if filter.Level != "" {
		query = query.Where("level = ?", filter.Level)
	}

	if filter.StartDate != nil {
		query = query.Where("created_at >= ?", filter.StartDate)
	}

	if filter.EndDate != nil {
		query = query.Where("created_at <= ?", filter.EndDate)
	}

	if filter.Limit > 0 {
		query = query.Limit(filter.Limit)
	}

	if err := query.Order("created_at DESC").Find(&events).Error; err != nil {
		return nil, fmt.Errorf("failed to list security events: %w", err)
	}

	return events, nil
}

func (r *postgresRepository) CleanupOldSecurityEvents(ctx context.Context, before time.Time) error {
	if err := r.db.WithContext(ctx).
		Where("created_at < ?", before).
		Delete(&domain.SecurityEvent{}).Error; err != nil {
		return fmt.Errorf("failed to cleanup old security events: %w", err)
	}
	return nil
}

// User organization operations

func (r *postgresRepository) GetUserOrganizations(ctx context.Context, userID string) ([]string, error) {
	var orgIDs []string

	if err := r.db.WithContext(ctx).
		Table("organization_users").
		Where("user_id = ?", userID).
		Pluck("organization_id", &orgIDs).Error; err != nil {
		return nil, fmt.Errorf("failed to get user organizations: %w", err)
	}

	return orgIDs, nil
}

func (r *postgresRepository) GetUserWorkspaceGroups(ctx context.Context, userID, workspaceID string) ([]string, error) {
	var groups []string

	// Get workspace member role
	var member struct {
		Role string
	}

	if err := r.db.WithContext(ctx).
		Table("workspace_members").
		Select("role").
		Where("user_id = ? AND workspace_id = ?", userID, workspaceID).
		First(&member).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return []string{}, nil
		}
		return nil, fmt.Errorf("failed to get workspace member: %w", err)
	}

	// Map role to groups
	switch member.Role {
	case "admin":
		groups = []string{"workspace:admin", "workspace:editor", "workspace:viewer"}
	case "editor":
		groups = []string{"workspace:editor", "workspace:viewer"}
	case "viewer":
		groups = []string{"workspace:viewer"}
	}

	return groups, nil
}

// Helper types

// RefreshTokenBlacklist represents a blacklisted refresh token.
type RefreshTokenBlacklist struct {
	ID        string `gorm:"primaryKey"`
	Token     string `gorm:"uniqueIndex"`
	ExpiresAt time.Time `gorm:"index"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
}

// Cleanup functions

func (r *postgresRepository) scheduleAuthStateCleanup(state string, expiresAt time.Time) {
	time.Sleep(time.Until(expiresAt))
	r.db.Where("state = ?", state).Delete(&domain.AuthState{})
}

func (r *postgresRepository) scheduleBlacklistCleanup(token string, expiresAt time.Time) {
	time.Sleep(time.Until(expiresAt))
	r.db.Where("token = ?", token).Delete(&RefreshTokenBlacklist{})
}
