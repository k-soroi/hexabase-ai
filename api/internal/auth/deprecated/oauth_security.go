// Package deprecated contains deprecated code that will be removed in future versions
//
//nolint:all // This package contains deprecated code that is not used in the current version
package deprecated

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/hexabase/hexabase-ai/api/internal/shared/config"
)

// SecureOAuthClient extends OAuthClient with enhanced security features
type SecureOAuthClient struct {
	*OAuthClient
	sessionManager *SessionManager
	jwtManager     *EnhancedJWTManager
	rateLimiter    *RateLimiter
	auditLogger    *AuditLogger
}

// TokenPair represents access and refresh tokens
type TokenPair struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	TokenType    string    `json:"token_type"`
	ExpiresIn    int       `json:"expires_in"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// EnhancedClaims extends JWT claims with security features
type EnhancedClaims struct {
	jwt.RegisteredClaims
	UserID        string   `json:"uid"`
	Email         string   `json:"email"`
	Provider      string   `json:"provider"`
	Organizations []string `json:"orgs,omitempty"`
	Permissions   []string `json:"perms,omitempty"`
	Fingerprint   string   `json:"fp,omitempty"`
	TokenType     string   `json:"typ"` // "access" or "refresh"
	SessionID     string   `json:"sid,omitempty"`
}

// SecureSession represents an authenticated session
type SecureSession struct {
	ID           string    `json:"id"`
	UserID       string    `json:"user_id"`
	DeviceID     string    `json:"device_id"`
	IPAddress    string    `json:"ip_address"`
	UserAgent    string    `json:"user_agent"`
	Provider     string    `json:"provider"`
	CreatedAt    time.Time `json:"created_at"`
	LastActive   time.Time `json:"last_active"`
	ExpiresAt    time.Time `json:"expires_at"`
	RefreshToken string    `json:"refresh_token,omitempty"`
}

// EnhancedJWTManager handles JWT operations with security enhancements
type EnhancedJWTManager struct {
	privateKey    interface{}
	publicKey     interface{}
	redis         RedisClient
	accessExpiry  time.Duration
	refreshExpiry time.Duration
}

// SessionManager handles secure session management
type SessionManager struct {
	redis           RedisClient
	maxConcurrent   int
	sessionTimeout  time.Duration
	absoluteTimeout time.Duration
	mu              sync.Mutex
}

// RateLimiter implements rate limiting for authentication endpoints
type RateLimiter struct {
	redis  RedisClient
	limit  int
	window time.Duration
}

// AuditLogger handles security audit logging
type AuditLogger struct {
	redis RedisClient
}

// AuditEvent represents a security audit event
type AuditEvent struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	UserID    string                 `json:"user_id"`
	IP        string                 `json:"ip"`
	UserAgent string                 `json:"user_agent"`
	Provider  string                 `json:"provider,omitempty"`
	Success   bool                   `json:"success"`
	Error     string                 `json:"error,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

// NewSecureOAuthClient creates an OAuth client with enhanced security
func NewSecureOAuthClient(
	cfg *config.Config,
	redis RedisClient,
	privateKey, publicKey interface{},
) *SecureOAuthClient {
	baseClient := NewOAuthClient(cfg, redis)

	return &SecureOAuthClient{
		OAuthClient:    baseClient,
		sessionManager: NewSessionManager(redis),
		jwtManager:     NewEnhancedJWTManager(privateKey, publicKey, redis),
		rateLimiter:    NewRateLimiter(redis, 10, time.Minute), //nolint:mnd // 10 attempts per minute
		auditLogger:    NewAuditLogger(redis),
	}
}

// NewEnhancedJWTManager creates a JWT manager with security enhancements
func NewEnhancedJWTManager(privateKey, publicKey interface{}, redis RedisClient) *EnhancedJWTManager {
	return &EnhancedJWTManager{
		privateKey:    privateKey,
		publicKey:     publicKey,
		redis:         redis,
		accessExpiry:  15 * time.Minute,
		refreshExpiry: 7 * 24 * time.Hour,
	}
}

// GenerateTokenPair generates both access and refresh tokens
func (m *EnhancedJWTManager) GenerateTokenPair(
	userInfo *UserInfo,
	deviceID, ipAddress string,
) (*TokenPair, error) {
	sessionID := uuid.New().String()
	fingerprint := m.generateFingerprint(deviceID, ipAddress)

	// Generate access token
	accessClaims := &EnhancedClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userInfo.ID,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(m.accessExpiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ID:        uuid.New().String(),
		},
		UserID:      userInfo.ID,
		Email:       userInfo.Email,
		Provider:    userInfo.Provider,
		Fingerprint: fingerprint,
		TokenType:   "access",
		SessionID:   sessionID,
	}

	accessToken := jwt.NewWithClaims(jwt.SigningMethodRS256, accessClaims)
	accessTokenString, err := accessToken.SignedString(m.privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to sign access token: %w", err)
	}

	// Generate refresh token
	refreshClaims := &EnhancedClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userInfo.ID,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(m.refreshExpiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ID:        uuid.New().String(),
		},
		UserID:      userInfo.ID,
		Email:       userInfo.Email,
		Provider:    userInfo.Provider,
		Fingerprint: fingerprint,
		TokenType:   "refresh",
		SessionID:   sessionID,
	}

	refreshToken := jwt.NewWithClaims(jwt.SigningMethodRS256, refreshClaims)
	refreshTokenString, err := refreshToken.SignedString(m.privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to sign refresh token: %w", err)
	}

	return &TokenPair{
		AccessToken:  accessTokenString,
		RefreshToken: refreshTokenString,
		TokenType:    "Bearer",
		ExpiresIn:    int(m.accessExpiry.Seconds()),
		ExpiresAt:    time.Now().Add(m.accessExpiry),
	}, nil
}

// ValidateAccessToken validates an access token
func (m *EnhancedJWTManager) ValidateAccessToken(tokenString string) (*EnhancedClaims, error) {
	// Check if token is revoked
	if m.redis != nil {
		key := fmt.Sprintf("revoked_token:%s", tokenString)
		exists, _ := m.redis.Exists(context.Background(), key)
		if exists > 0 {
			return nil, fmt.Errorf("token revoked")
		}
	}

	token, err := jwt.ParseWithClaims(tokenString, &EnhancedClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return m.publicKey, nil
	})
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*EnhancedClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	if claims.TokenType != "access" {
		return nil, fmt.Errorf("invalid token type")
	}

	return claims, nil
}

// ValidateRefreshToken validates a refresh token
func (m *EnhancedJWTManager) ValidateRefreshToken(tokenString string) (*EnhancedClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &EnhancedClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return m.publicKey, nil
	})
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*EnhancedClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	if claims.TokenType != "refresh" {
		return nil, fmt.Errorf("invalid token type")
	}

	return claims, nil
}

// ValidateWithFingerprint validates token with device fingerprint
func (m *EnhancedJWTManager) ValidateWithFingerprint(tokenString, deviceID, ipAddress string) (*EnhancedClaims, error) {
	claims, err := m.ValidateAccessToken(tokenString)
	if err != nil {
		return nil, err
	}

	expectedFingerprint := m.generateFingerprint(deviceID, ipAddress)
	if claims.Fingerprint != expectedFingerprint {
		return nil, fmt.Errorf("fingerprint mismatch")
	}

	return claims, nil
}

// RefreshTokens generates new token pair from refresh token
func (m *EnhancedJWTManager) RefreshTokens(refreshToken, deviceID, ipAddress string) (*TokenPair, error) {
	claims, err := m.ValidateRefreshToken(refreshToken)
	if err != nil {
		return nil, err
	}

	// Verify fingerprint
	expectedFingerprint := m.generateFingerprint(deviceID, ipAddress)
	if claims.Fingerprint != expectedFingerprint {
		return nil, fmt.Errorf("fingerprint mismatch")
	}

	// Generate new token pair
	userInfo := &UserInfo{
		ID:       claims.UserID,
		Email:    claims.Email,
		Provider: claims.Provider,
	}

	return m.GenerateTokenPair(userInfo, deviceID, ipAddress)
}

// RevokeToken revokes a token
func (m *EnhancedJWTManager) RevokeToken(ctx context.Context, tokenString string) error {
	if m.redis == nil {
		return fmt.Errorf("redis client required for token revocation")
	}

	// Parse token to get expiry
	token, _ := jwt.ParseWithClaims(tokenString, &EnhancedClaims{}, func(token *jwt.Token) (interface{}, error) {
		return m.publicKey, nil
	})

	if token != nil && token.Claims != nil {
		claims := token.Claims.(*EnhancedClaims)
		ttl := time.Until(claims.ExpiresAt.Time)
		if ttl > 0 {
			key := fmt.Sprintf("revoked_token:%s", tokenString)
			return m.redis.SetWithTTL(ctx, key, "revoked", ttl)
		}
	}

	return nil
}

// generateFingerprint creates a device fingerprint
func (m *EnhancedJWTManager) generateFingerprint(deviceID, ipAddress string) string {
	data := fmt.Sprintf("%s:%s", deviceID, ipAddress)
	hash := sha256.Sum256([]byte(data))
	return base64.RawURLEncoding.EncodeToString(hash[:])
}

// NewSessionManager creates a new session manager
func NewSessionManager(redis RedisClient) *SessionManager {
	return &SessionManager{
		redis:           redis,
		maxConcurrent:   3,
		sessionTimeout:  30 * time.Minute,
		absoluteTimeout: 24 * time.Hour,
	}
}

// CreateSession creates a new secure session
func (sm *SessionManager) CreateSession(
	ctx context.Context,
	userInfo *UserInfo,
	deviceID, ipAddress, userAgent string,
) (*SecureSession, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Check concurrent sessions
	if err := sm.enforceSessionLimit(ctx, userInfo.ID); err != nil {
		return nil, err
	}

	session := &SecureSession{
		ID:         uuid.New().String(),
		UserID:     userInfo.ID,
		DeviceID:   deviceID,
		IPAddress:  ipAddress,
		UserAgent:  userAgent,
		Provider:   userInfo.Provider,
		CreatedAt:  time.Now(),
		LastActive: time.Now(),
		ExpiresAt:  time.Now().Add(sm.absoluteTimeout),
	}

	// Store session in Redis
	if sm.redis != nil {
		key := fmt.Sprintf("session:%s", session.ID)
		data, _ := json.Marshal(session)
		if err := sm.redis.SetWithTTL(ctx, key, string(data), sm.absoluteTimeout); err != nil {
			return nil, err
		}

		// Add to user's session list
		userKey := fmt.Sprintf("user_sessions:%s", userInfo.ID)
		sm.redis.SetWithTTL(ctx, userKey, session.ID, sm.absoluteTimeout)
	}

	return session, nil
}

// GetSession retrieves a session
func (sm *SessionManager) GetSession(ctx context.Context, sessionID string) (*SecureSession, error) {
	if sm.redis == nil {
		return nil, fmt.Errorf("redis client required")
	}

	key := fmt.Sprintf("session:%s", sessionID)
	data, err := sm.redis.Get(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("session not found")
	}

	var session SecureSession
	if err := json.Unmarshal([]byte(data), &session); err != nil {
		return nil, err
	}

	// Check if session expired
	if time.Now().After(session.ExpiresAt) {
		sm.redis.Delete(ctx, key)
		return nil, fmt.Errorf("session expired")
	}

	// Check idle timeout
	if time.Since(session.LastActive) > sm.sessionTimeout {
		sm.redis.Delete(ctx, key)
		return nil, fmt.Errorf("session idle timeout")
	}

	return &session, nil
}

// TouchSession updates session last active time
func (sm *SessionManager) TouchSession(ctx context.Context, sessionID string) error {
	session, err := sm.GetSession(ctx, sessionID)
	if err != nil {
		return err
	}

	session.LastActive = time.Now()

	key := fmt.Sprintf("session:%s", sessionID)
	data, _ := json.Marshal(session)
	return sm.redis.SetWithTTL(ctx, key, string(data), time.Until(session.ExpiresAt))
}

// ValidateSession validates session security
func (sm *SessionManager) ValidateSession(ctx context.Context, sessionID, ipAddress, userAgent string) error {
	session, err := sm.GetSession(ctx, sessionID)
	if err != nil {
		return err
	}

	// Check IP address
	if session.IPAddress != ipAddress {
		return fmt.Errorf("IP mismatch detected")
	}

	// Check user agent
	if session.UserAgent != userAgent {
		return fmt.Errorf("user agent mismatch detected")
	}

	// Touch session to update last active
	return sm.TouchSession(ctx, sessionID)
}

// GetUserSessions gets all sessions for a user
func (sm *SessionManager) GetUserSessions(ctx context.Context, userID string) ([]*SecureSession, error) {
	// In production, this would query Redis for all user sessions
	// For now, return mock data
	return []*SecureSession{}, nil
}

// enforceSessionLimit ensures user doesn't exceed max concurrent sessions
func (sm *SessionManager) enforceSessionLimit(ctx context.Context, userID string) error {
	sessions, err := sm.GetUserSessions(ctx, userID)
	if err != nil {
		return err
	}

	if len(sessions) >= sm.maxConcurrent {
		// Remove oldest session
		// In production, implement proper session cleanup
		return nil
	}

	return nil
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(redis RedisClient, limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		redis:  redis,
		limit:  limit,
		window: window,
	}
}

// Allow checks if request is allowed
func (rl *RateLimiter) Allow(ctx context.Context, identifier, action string) (bool, error) {
	if rl.redis == nil {
		return true, nil // Allow if Redis not configured
	}

	key := fmt.Sprintf("rate_limit:%s:%s", action, identifier)

	// Simple implementation - in production use sliding window
	count, err := rl.redis.Exists(ctx, key)
	if err != nil {
		return false, err
	}

	if count >= int64(rl.limit) {
		return false, nil
	}

	// Increment counter
	rl.redis.SetWithTTL(ctx, key, "1", rl.window)

	return true, nil
}

// NewAuditLogger creates a new audit logger
func NewAuditLogger(redis RedisClient) *AuditLogger {
	return &AuditLogger{redis: redis}
}

// Log records an audit event
func (al *AuditLogger) Log(ctx context.Context, event AuditEvent) error {
	if al.redis == nil {
		return nil // Skip if Redis not configured
	}

	event.ID = uuid.New().String()
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	// Store in Redis with TTL
	key := fmt.Sprintf("audit:%s:%s", event.UserID, event.ID)
	data, _ := json.Marshal(event)

	return al.redis.SetWithTTL(ctx, key, string(data), 90*24*time.Hour) // 90 days retention
}

// GetUserLogs retrieves audit logs for a user
func (al *AuditLogger) GetUserLogs(ctx context.Context, userID string, limit int) ([]AuditEvent, error) {
	// In production, implement proper log retrieval
	return []AuditEvent{}, nil
}

// SecurityHeadersMiddleware adds security headers to responses
func SecurityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// HSTS
		w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")

		// Prevent MIME type sniffing
		w.Header().Set("X-Content-Type-Options", "nosniff")

		// Prevent clickjacking
		w.Header().Set("X-Frame-Options", "DENY")

		// XSS Protection (legacy but still useful)
		w.Header().Set("X-XSS-Protection", "1; mode=block")

		// Content Security Policy
		csp := "default-src 'self'; " +
			"script-src 'self' 'unsafe-inline' 'unsafe-eval' https://accounts.google.com; " +
			"style-src 'self' 'unsafe-inline'; " +
			"img-src 'self' data: https:; " +
			"font-src 'self' data:; " +
			"connect-src 'self' https://api.github.com https://accounts.google.com"
		w.Header().Set("Content-Security-Policy", csp)

		// Referrer Policy
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

		// Permissions Policy
		w.Header().Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")

		next.ServeHTTP(w, r)
	})
}

// ConfigureCORS returns a CORS middleware with strict configuration
func ConfigureCORS(allowedOrigins []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			// Check if origin is allowed
			allowed := false
			for _, allowedOrigin := range allowedOrigins {
				if origin == allowedOrigin {
					allowed = true
					break
				}
			}

			if allowed {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Credentials", "true")
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
				w.Header().
					Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Requested-With, X-CSRF-Token")
				w.Header().Set("Access-Control-Max-Age", "86400")
			}

			// Handle preflight
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// GetAuthURLWithPKCE generates OAuth URL with PKCE parameters
// Deprecated: This function is deprecated and will be removed in a future version
// func (c *SecureOAuthClient) GetAuthURLWithPKCE(provider, state, codeVerifier string) (string, error) {
// 	challenge := GenerateCodeChallenge(codeVerifier)
//
// 	cfg, ok := c.providers[provider]
// 	if !ok {
// 		return "", fmt.Errorf("provider %s not configured", provider)
// 	}
//
// 	// Add PKCE parameters
// 	authURL := cfg.AuthCodeURL(state,
// 		oauth2.AccessTypeOffline,
// 		oauth2.SetAuthURLParam("code_challenge", challenge),
// 		oauth2.SetAuthURLParam("code_challenge_method", "S256"),
// 	)
//
// 	return authURL, nil
// }

// ValidateProvider validates if provider is supported
func (c *SecureOAuthClient) ValidateProvider(provider string) error {
	validProviders := []string{"google", "github", "gitlab"}
	for _, valid := range validProviders {
		if provider == valid {
			return nil
		}
	}
	return fmt.Errorf("invalid provider: %s", provider)
}

// GenerateCodeChallenge generates SHA256 code challenge for PKCE
func GenerateCodeChallenge(verifier string) string {
	hash := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(hash[:])
}
