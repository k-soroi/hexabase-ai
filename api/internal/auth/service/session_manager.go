//nolint:ireturn
package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hexabase/hexabase-ai/api/internal/auth/domain"
)

const (
	// maxConcurrentSessions is the maximum number of concurrent sessions per user
	maxConcurrentSessions = 3
	// sessionTimeoutSeconds is the session timeout duration in seconds (30 minutes)
	sessionTimeoutSeconds = 30 * 60
	// lockTimeoutSeconds is the timeout for distributed lock
	lockTimeoutSeconds = 5
)

// sessionManager implements session limit management
type sessionManager struct {
	repo        domain.Repository
	limiterRepo domain.SessionLimiterRepository
	config      *domain.SessionManagerConfig
}

// NewSessionManager creates a new SessionManager
func NewSessionManager(
	repo domain.Repository,
	limiterRepo domain.SessionLimiterRepository,
) domain.SessionManager {
	config := &domain.SessionManagerConfig{
		MaxConcurrentSessions: maxConcurrentSessions,
		SessionTimeout:        sessionTimeoutSeconds,
	}

	return &sessionManager{
		repo:        repo,
		limiterRepo: limiterRepo,
		config:      config,
	}
}

// CreateSession creates a new session and checks concurrent session limit
func (sm *sessionManager) CreateSession(ctx context.Context, userID, sessionID string) error {
	// Acquire distributed lock for this user
	lockKey := "session_create_lock:" + userID
	lockValue := uuid.New().String()
	lockTTL := lockTimeoutSeconds * time.Second

	// Try to acquire lock
	acquired, err := sm.limiterRepo.AcquireLock(ctx, lockKey, lockValue, lockTTL)
	if err != nil {
		return fmt.Errorf("failed to acquire lock: %w", err)
	}

	if !acquired {
		return domain.ErrTooManySessions
	}

	// Ensure lock is released
	defer func() {
		_ = sm.limiterRepo.ReleaseLock(ctx, lockKey, lockValue)
	}()

	// Get current session count
	count, err := sm.limiterRepo.GetUserSessionCount(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get session count: %w", err)
	}

	// If limit reached, return error
	if count >= int64(sm.config.MaxConcurrentSessions) {
		return domain.ErrTooManySessions
	}

	// Add new session to Redis
	ttl := time.Duration(sm.config.SessionTimeout) * time.Second
	if err := sm.limiterRepo.AddSessionToUser(ctx, userID, sessionID, ttl); err != nil {
		return fmt.Errorf("failed to add session to Redis: %w", err)
	}

	return nil
}

// DeleteSession removes the specified session
func (sm *sessionManager) DeleteSession(ctx context.Context, userID, sessionID string) error {
	if err := sm.limiterRepo.RemoveSessionFromUser(ctx, userID, sessionID); err != nil {
		return fmt.Errorf("failed to remove session: %w", err)
	}

	return nil
}

// GetActiveSessionCount gets the count of active sessions for a user
func (sm *sessionManager) GetActiveSessionCount(ctx context.Context, userID string) (int, error) {
	count, err := sm.limiterRepo.GetUserSessionCount(ctx, userID)
	if err != nil {
		return 0, fmt.Errorf("failed to get active session count: %w", err)
	}

	return int(count), nil
}
