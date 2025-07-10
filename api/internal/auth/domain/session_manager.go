package domain

import (
	"context"
	"errors"
)

// SessionManager error definitions
var (
	ErrTooManySessions = errors.New("concurrent session limit exceeded")
)

// SessionManager is an interface for managing concurrent user sessions
type SessionManager interface {
	// CreateSession creates a new session and checks concurrent session limit.
	// If the limit is exceeded, it removes the oldest session before creating a new one
	CreateSession(ctx context.Context, userID, sessionID string) error

	// DeleteSession removes the specified session
	DeleteSession(ctx context.Context, userID, sessionID string) error

	// GetActiveSessionCount gets the count of active sessions for a user
	GetActiveSessionCount(ctx context.Context, userID string) (int, error)
}

// SessionManagerConfig is the configuration for SessionManager
type SessionManagerConfig struct {
	// MaxConcurrentSessions is the maximum number of concurrent sessions per user
	MaxConcurrentSessions int
	// SessionTimeout is the session timeout duration
	SessionTimeout int
}
