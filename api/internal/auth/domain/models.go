package domain

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// User represents an authenticated user with business rules
type User struct {
	ID          string    `json:"id"`
	ExternalID  string    `json:"external_id"`
	Provider    string    `json:"provider"`
	Email       string    `json:"email"`
	DisplayName string    `json:"display_name"`
	AvatarURL   string    `json:"avatar_url,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	LastLoginAt time.Time `json:"last_login_at"`
}

// Validate ensures user data meets business rules
func (u *User) Validate() error {
	if u.Email == "" {
		return errors.New("email is required")
	}
	if u.Provider == "" {
		return errors.New("provider is required")
	}
	if u.DisplayName == "" {
		return errors.New("display name is required")
	}
	if u.ExternalID == "" {
		return errors.New("external ID is required")
	}
	return nil
}

// TokenPair represents access and refresh tokens
type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
}

// Claims represents JWT claims with business validation
type Claims struct {
	jwt.RegisteredClaims
	UserID    string   `json:"user_id"`
	Email     string   `json:"email"`
	Name      string   `json:"name"`
	Provider  string   `json:"provider"`
	OrgIDs    []string `json:"org_ids,omitempty"`
	SessionID string   `json:"session_id"`
}

// ValidateBusinessRules ensures claims meet business requirements
func (c *Claims) ValidateBusinessRules() error {
	if c.UserID == "" {
		return errors.New("user_id claim is required")
	}
	if c.Email == "" {
		return errors.New("email claim is required")
	}
	if c.SessionID == "" {
		return errors.New("session_id claim is required")
	}
	
// Removed commented-out TODOs and dead code related to legacy session logging.

	// Validate expiration within acceptable range
	if c.ExpiresAt != nil {
		maxExpiry := time.Now().Add(24 * time.Hour)
		if c.ExpiresAt.After(maxExpiry) {
			return errors.New("token expiry exceeds maximum allowed duration")
		}
	}
	
	return nil
}

// WorkspaceClaims represents JWT claims for workspace access
type WorkspaceClaims struct {
	jwt.RegisteredClaims
	Subject     string   `json:"sub"`
	WorkspaceID string   `json:"workspace_id"`
	Groups      []string `json:"groups"`
}

// Session represents an authenticated session and implements the Selector/Validator model
// for refresh tokens to enhance security and performance.
//
// The Selector/Validator model splits a refresh token into two parts:
// 1. Selector: A public, unique identifier used for O(1) database lookups to find a session.
//    It is stored in plain text in the database.
// 2. Verifier: A high-entropy secret part that is hashed before being stored.
//    The client sends both parts, and the server verifies the verifier hash against the stored hash.
//
// This approach prevents timing attacks by ensuring the database lookup time is constant
// regardless of whether the token is valid, and it allows for efficient session retrieval
// without compromising the security of the refresh token itself.
type Session struct {
	ID                   string    `json:"id"`
	UserID               string    `json:"user_id"`
	RefreshToken         string    `json:"refresh_token"`         // Hashed verifier part of the refresh token
	RefreshTokenSelector string    `json:"refresh_token_selector"` // Selector part for O(1) session lookup
	Salt                 string    `json:"salt"`                   // Salt for refresh token hashing
	DeviceID             string    `json:"device_id,omitempty"`
	IPAddress            string    `json:"ip_address"`
	UserAgent            string    `json:"user_agent"`
	ExpiresAt            time.Time `json:"expires_at"`
	CreatedAt            time.Time `json:"created_at"`
	LastUsedAt           time.Time `json:"last_used_at"`
	Revoked              bool      `json:"revoked"`
}

// IsExpired checks if session has expired
func (s *Session) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}

// IsValid checks if session is valid for use
func (s *Session) IsValid() bool {
	return !s.Revoked && !s.IsExpired()
}

// Revoke marks the session as revoked
func (s *Session) Revoke() {
	s.Revoked = true
}

// UpdateLastUsed updates the last used timestamp
func (s *Session) UpdateLastUsed() {
	s.LastUsedAt = time.Now()
}

// ValidateRefreshTokenSelector validates the refresh token selector format
func (s *Session) ValidateRefreshTokenSelector() error {
	if s.RefreshTokenSelector == "" {
		return errors.New("refresh token selector is required")
	}
	if len(s.RefreshTokenSelector) < 8 {
		return errors.New("refresh token selector must be at least 8 characters long")
	}
	if len(s.RefreshTokenSelector) > 64 {
		return errors.New("refresh token selector must not exceed 64 characters")
	}
	return nil
}

// AuthState represents OAuth state data stored temporarily during the auth flow.
type AuthState struct {
	State         string    `gorm:"primaryKey" json:"state"` // gorm:primaryKey - Uniquely identifies the auth request.
	Provider      string    `gorm:"not null" json:"provider"`   // gorm:not null - Required to identify the auth provider on callback.
	RedirectURL   string    `json:"redirect_url,omitempty"`
	CodeChallenge string    `json:"code_challenge,omitempty"` // RFC 7636: Stores the code challenge (SHA256 hash of verifier)
	ClientIP      string    `json:"client_ip"`
	UserAgent     string    `json:"user_agent"`
	ExpiresAt    time.Time `gorm:"index;not null" json:"expires_at"` // gorm:index - For efficient lookup of active states. gorm:not null - States must expire.
	CreatedAt    time.Time `gorm:"not null" json:"created_at"`       // gorm:not null - Ensures creation timestamp exists for auditing.
}

// SecurityEvent represents a security-related event
type SecurityEvent struct {
	ID          string                 `json:"id"`
	UserID      string                 `json:"user_id"`
	EventType   string                 `json:"event_type"`
	Description string                 `json:"description"`
	IPAddress   string                 `json:"ip_address"`
	UserAgent   string                 `json:"user_agent"`
	Level       string                 `json:"level"` // info, warning, critical
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
}

// TODO: This TableName method is a temporary workaround to make GORM work with the domain model directly.
// In the future, a separate DTO for the repository layer should be created,
// and this method should be removed. The conversion between the domain model and the DTO
// will be handled within the repository.
func (SecurityEvent) TableName() string {
	return "security_events"
}

// LoginRequest represents OAuth login request
type LoginRequest struct {
	Provider            string `json:"provider" binding:"required"`
	CodeChallenge       string `json:"code_challenge,omitempty"`
	CodeChallengeMethod string `json:"code_challenge_method,omitempty"`
	RedirectURL         string `json:"redirect_url,omitempty"`
}

// CallbackRequest represents OAuth callback request
type CallbackRequest struct {
	Code         string `json:"code" binding:"required"`
	State        string `json:"state" binding:"required"`
	CodeVerifier string `json:"code_verifier,omitempty"`
}

// RefreshTokenRequest represents token refresh request
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// UserInfo represents user information from OAuth provider
type UserInfo struct {
	ID       string `json:"id"`
	Email    string `json:"email"`
	Name     string `json:"name"`
	Picture  string `json:"picture,omitempty"`
	Provider string `json:"provider"`
}

// AuthResponse represents authentication response
type AuthResponse struct {
	User         *User      `json:"user"`
	AccessToken  string     `json:"access_token"`
	RefreshToken string     `json:"refresh_token"`
	TokenType    string     `json:"token_type"`
	ExpiresIn    int        `json:"expires_in"`
}

// SessionInfo represents session information
type SessionInfo struct {
	ID         string    `json:"id"`
	DeviceID   string    `json:"device_id,omitempty"`
	IPAddress  string    `json:"ip_address"`
	UserAgent  string    `json:"user_agent"`
	Location   string    `json:"location,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
	LastUsedAt time.Time `json:"last_used_at"`
	IsCurrent  bool      `json:"is_current"`
}

// SecurityLogFilter represents filter options for security logs
type SecurityLogFilter struct {
	UserID    string
	EventType string
	Level     string
	StartDate *time.Time
	EndDate   *time.Time
	Limit     int
}