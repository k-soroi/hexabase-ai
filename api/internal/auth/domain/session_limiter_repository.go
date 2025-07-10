package domain

import (
	"context"
	"time"
)

// SessionLimiterRepository is a Redis operations interface for session limit management
type SessionLimiterRepository interface {
	// AddSessionToUser adds a session ID to the user's active session set
	AddSessionToUser(ctx context.Context, userID, sessionID string, ttl time.Duration) error

	// RemoveSessionFromUser removes a session ID from the user's active session set
	RemoveSessionFromUser(ctx context.Context, userID, sessionID string) error

	// GetUserSessionCount gets the count of active sessions for a user
	GetUserSessionCount(ctx context.Context, userID string) (int64, error)

	// GetUserSessionIDs gets all active session IDs for a user
	GetUserSessionIDs(ctx context.Context, userID string) ([]string, error)

	// IsSessionInUserSet checks if a specified session ID exists in the user's set
	IsSessionInUserSet(ctx context.Context, userID, sessionID string) (bool, error)

	// AcquireLock tries to acquire a distributed lock
	AcquireLock(ctx context.Context, key, value string, ttl time.Duration) (bool, error)

	// ReleaseLock releases a distributed lock if the value matches
	ReleaseLock(ctx context.Context, key, value string) error
}
