package domain

import (
	"context"
)

// Service defines the interface for the authentication business logic
type Service interface {
	// OAuth operations
	GetAuthURL(ctx context.Context, req *LoginRequest) (string, string, error)
	GetAuthURLForSignUp(ctx context.Context, req *SignUpAuthRequest) (string, string, error)
	HandleCallback(ctx context.Context, req *CallbackRequest, clientIP, userAgent string) (*AuthResponse, error)

	// Token operations
	RefreshToken(ctx context.Context, refreshToken, clientIP, userAgent string) (*TokenPair, error)
	RevokeRefreshToken(ctx context.Context, refreshToken string) error
	ValidateAccessToken(ctx context.Context, token string) (*Claims, error)

	// Session operations
	CreateSession(ctx context.Context, sessionID, userID, refreshToken, deviceID, clientIP, userAgent string) (*Session, error)
	GetSession(ctx context.Context, sessionID string) (*Session, error)
	GetUserSessions(ctx context.Context, userID string) ([]*SessionInfo, error)
	RevokeSession(ctx context.Context, userID, sessionID string) error
	RevokeAllSessions(ctx context.Context, userID string, exceptSessionID string) error
	ValidateSession(ctx context.Context, sessionID, clientIP string) error

	// User operations
	GetUser(ctx context.Context, userID string) (*User, error)
	GetCurrentUser(ctx context.Context, token string) (*User, error)
	UpdateUserProfile(ctx context.Context, userID string, updates map[string]interface{}) (*User, error)

	// Workspace token operations
	GenerateWorkspaceToken(ctx context.Context, userID, workspaceID string) (string, error)

	// OIDC operations
	GetJWKS(ctx context.Context) ([]byte, error)

	// Auth state operations
	StoreAuthState(ctx context.Context, state *AuthState) error

	// Security operations
	LogSecurityEvent(ctx context.Context, event *SecurityEvent) error
	GetSecurityLogs(ctx context.Context, filter SecurityLogFilter) ([]*SecurityEvent, error)

	// Internal operations
	GenerateInternalAIOpsToken(ctx context.Context, userID string, orgIDs []string, activeWorkspaceID string) (string, error)

	// Session invalidation operations
	InvalidateSession(ctx context.Context, sessionID string) error
}

// TokenDomainService defines the interface for token business logic
type TokenDomainService interface {
	RefreshToken(ctx context.Context, session *Session, user *User) (*Claims, error)
	ValidateRefreshEligibility(session *Session) error
	CreateSession(sessionID, userID, refreshToken, deviceID, clientIP, userAgent string) (*Session, error)
	ValidateTokenClaims(claims *Claims) error
	ShouldRefreshToken(claims *Claims) bool
}
