package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	internalAuth "github.com/hexabase/hexabase-ai/api/internal/auth"
	"github.com/hexabase/hexabase-ai/api/internal/auth/domain"
)

// InternalAIOpsClaims defines the structure for the internal AIOps token.
type InternalAIOpsClaims struct {
	jwt.RegisteredClaims
	UserID            string   `json:"user_id"`
	OrgIDs            []string `json:"org_ids"`
	ActiveWorkspaceID string   `json:"active_workspace_id"`
	TokenType         string   `json:"token_type"`
}

const (
	legacySessionID           = "legacy-session"
	refreshTokenSeparator     = "."
	refreshTokenExpectedParts = 2
)

// RefreshTokenParts represents the parsed components of a refresh token
type RefreshTokenParts struct {
	Selector string
	Verifier string
}

type service struct {
	repo               domain.Repository
	oauthRepo          domain.OAuthRepository
	keyRepo            domain.KeyRepository
	tokenManager       *internalAuth.TokenManager
	tokenDomainService domain.TokenDomainService
	sessionManager     domain.SessionManager
	logger             *slog.Logger
	defaultTokenExpiry int // Default token expiry in seconds when claims don't have expiry info
}

// NewService creates a new auth service
func NewService(
	repo domain.Repository,
	oauthRepo domain.OAuthRepository,
	keyRepo domain.KeyRepository,
	tokenManager *internalAuth.TokenManager,
	tokenDomainService domain.TokenDomainService,
	sessionManager domain.SessionManager,
	logger *slog.Logger,
	defaultTokenExpiry int,
) domain.Service {
	return &service{
		repo:               repo,
		oauthRepo:          oauthRepo,
		keyRepo:            keyRepo,
		tokenManager:       tokenManager,
		tokenDomainService: tokenDomainService,
		sessionManager:     sessionManager,
		logger:             logger,
		defaultTokenExpiry: defaultTokenExpiry,
	}
}

func (s *service) GetAuthURL(ctx context.Context, req *domain.LoginRequest) (string, string, error) {
	// Generate state
	stateBytes := make([]byte, 32)
	if _, err := rand.Read(stateBytes); err != nil {
		return "", "", fmt.Errorf("failed to generate state: %w", err)
	}
	state := base64.URLEncoding.EncodeToString(stateBytes)

	// Generate PKCE challenge if provided
	var codeChallenge string
	if req.CodeChallenge != "" {
		codeChallenge = req.CodeChallenge
	}

	// Store auth state
	authState := &domain.AuthState{
		State:         state,
		Provider:      req.Provider,
		RedirectURL:   req.RedirectURL,
		CodeChallenge: codeChallenge, // Store for later verification
		ExpiresAt:     time.Now().Add(10 * time.Minute),
		CreatedAt:     time.Now(),
	}

	if err := s.repo.StoreAuthState(ctx, authState); err != nil {
		return "", "", fmt.Errorf("failed to store auth state: %w", err)
	}

	// Get auth URL with parameters
	params := map[string]string{}
	if codeChallenge != "" {
		params["code_challenge"] = codeChallenge
		params["code_challenge_method"] = req.CodeChallengeMethod
	}

	authURL, err := s.oauthRepo.GetAuthURL(req.Provider, state, params)
	if err != nil {
		return "", "", fmt.Errorf("failed to get auth URL: %w", err)
	}

	return authURL, state, nil
}

func (s *service) HandleCallback(ctx context.Context, req *domain.CallbackRequest, clientIP, userAgent string) (*domain.AuthResponse, error) {
	// Get auth state once and perform all validations
	authState, err := s.repo.GetAuthState(ctx, req.State)
	if err != nil {
		if errors.Is(err, domain.ErrAuthStateNotFound) {
			return nil, fmt.Errorf("auth state not found or expired: %w", err)
		}
		return nil, fmt.Errorf("failed to get auth state: %w", err)
	}

	// Verify state
	if err := s.verifyAuthState(authState, clientIP); err != nil {
		return nil, fmt.Errorf("invalid state: %w", err)
	}

	// PKCE-related security enhancements
	if authState.CodeChallenge != "" {
		// If a code_challenge was set, the client MUST provide a code_verifier.
		if req.CodeVerifier == "" {
			s.logger.Warn("PKCE verifier missing", "state", req.State, "client_ip", clientIP)
			s.logSecurityEvent(ctx, "", "pkce_missing_verifier", "Client did not provide a code_verifier despite a code_challenge being set.", clientIP, userAgent, "warning")
			return nil, fmt.Errorf("PKCE error: code_verifier is required")
		}

		// Perform the PKCE verification.
		if err := s.verifyPKCE(authState, req.CodeVerifier); err != nil {
			// Log the failure and return a generic error to the client.
			s.logger.Warn("PKCE verification failed", "state", req.State, "error", err, "client_ip", clientIP)
			s.logSecurityEvent(ctx, "", "pkce_verification_failed", err.Error(), clientIP, userAgent, "warning")
			return nil, fmt.Errorf("PKCE verification failed")
		}
	}

	// Exchange code for tokens
	oauthToken, err := s.oauthRepo.ExchangeCode(ctx, authState.Provider, req.Code)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}

	// Get user info from provider
	userInfo, err := s.oauthRepo.GetUserInfo(ctx, authState.Provider, oauthToken)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}

	// Find or create user
	user, err := s.repo.GetUserByExternalID(ctx, userInfo.ID, authState.Provider)
	if err != nil {
		// Check if it's a "user not found" error
		if errors.Is(err, domain.ErrUserNotFound) {
			// Create new user
			now := time.Now()
			user = &domain.User{
				ID:          uuid.New().String(),
				ExternalID:  userInfo.ID,
				Provider:    authState.Provider,
				Email:       userInfo.Email,
				DisplayName: userInfo.Name,
				AvatarURL:   userInfo.Picture,
				CreatedAt:   now,
				UpdatedAt:   now,
				LastLoginAt: now,
			}

			if err := s.repo.CreateUser(ctx, user); err != nil {
				return nil, fmt.Errorf("failed to create user: %w", err)
			}

			// Log security event
			s.logSecurityEvent(ctx, user.ID, "user_created", "New user created via OAuth", clientIP, userAgent, "info")
		} else {
			// Other errors (database errors, etc.)
			return nil, fmt.Errorf("failed to get user info: %w", err)
		}
	} else {
		// Update last login
		if err := s.repo.UpdateLastLogin(ctx, user.ID); err != nil {
			s.logger.Error("failed to update last login", "error", err)
		}
	}

	// Generate session ID first
	sessionID := uuid.New().String()

	// Generate tokens with session ID
	tokenPair, err := s.generateTokenPair(ctx, user, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate tokens: %w", err)
	}

	// Create session with the pre-generated session ID
	if _, err := s.CreateSession(ctx, sessionID, user.ID, tokenPair.RefreshToken, "", clientIP, userAgent); err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	// Clean up auth state
	if err := s.repo.DeleteAuthState(ctx, req.State); err != nil {
		s.logger.Error("failed to delete auth state", "error", err)
	}

	// Log security event
	s.logSecurityEvent(ctx, user.ID, "login_success", "Successful OAuth login", clientIP, userAgent, "info")

	return &domain.AuthResponse{
		User:         user,
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		TokenType:    tokenPair.TokenType,
		ExpiresIn:    tokenPair.ExpiresIn,
	}, nil
}

func (s *service) RefreshToken(ctx context.Context, refreshToken, clientIP, userAgent string) (*domain.TokenPair, error) {
	// Infrastructure concerns: Check if token is blacklisted
	blacklisted, err := s.repo.IsRefreshTokenBlacklisted(ctx, refreshToken)
	if err != nil {
		return nil, fmt.Errorf("failed to check token blacklist: %w", err)
	}
	if blacklisted {
		return nil, fmt.Errorf("refresh token is invalid")
	}

	// Infrastructure concerns: Get session by refresh token
	session, err := s.getSessionByRefreshToken(ctx, refreshToken)
	if err != nil {
		return nil, fmt.Errorf("session not found: %w", err)
	}

	// Check if session is expired
	now := time.Now()
	s.logger.Info("[DEBUG] RefreshToken: session expiry check", "session.ExpiresAt", session.ExpiresAt, "now", now, "expired", session.IsExpired())
	if session.IsExpired() {
			s.logger.Info("[DEBUG] RefreshToken: session is expired, returning error")
			return nil, fmt.Errorf("session has expired")
	}

	// Infrastructure concerns: Get user
	user, err := s.repo.GetUser(ctx, session.UserID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	// Store old session ID before creating new one
	oldSessionID := session.ID

	// Generate new session ID to invalidate old access tokens
	newSessionID := uuid.New().String()

	// Business logic: Create new claims without modifying the original session
	newClaims, err := s.tokenDomainService.RefreshToken(ctx, session, user)
	if err != nil {
		return nil, fmt.Errorf("refresh validation failed: %w", err)
	}

	// Update claims with new session ID after domain logic validation
	newClaims.SessionID = newSessionID

	// Infrastructure concerns: Generate token pair using new claims
	tokenPair, err := s.generateTokenPairFromClaims(ctx, newClaims)
	if err != nil {
		return nil, fmt.Errorf("failed to generate token pair: %w", err)
	}

	// Infrastructure concerns: Blacklist old refresh token
	if err := s.repo.BlacklistRefreshToken(ctx, refreshToken, session.ExpiresAt); err != nil {
		s.logger.Error("failed to blacklist old refresh token", "error", err)
	}

	// Block the old session ID to invalidate all access tokens associated with it
	// This is a critical security operation - must not proceed if blocking fails
	if err := s.repo.BlockSession(ctx, oldSessionID, session.ExpiresAt); err != nil {
		return nil, fmt.Errorf("failed to block old session: %w", err)
	}

	// Remove old session from SessionManager
	// This is less critical - if it fails, the session will still be blocked and will expire from Redis
	if err := s.sessionManager.DeleteSession(ctx, session.UserID, oldSessionID); err != nil {
		s.logger.Error("failed to remove old session from SessionManager", "error", err)
	}

	// Infrastructure concerns: Hash new refresh token
	// Parse new refresh token to extract selector and verifier
	tokenParts, err := s.parseRefreshToken(tokenPair.RefreshToken)
	if err != nil {
		return nil, fmt.Errorf("failed to parse new refresh token: %w", err)
	}

	// Hash the new verifier
	hashedNewToken, newSalt, err := s.hashToken(tokenParts.Verifier)
	if err != nil {
		return nil, fmt.Errorf("failed to hash new refresh token: %w", err)
	}

	// Create new session with the new session ID
	newSession := &domain.Session{
		ID:           newSessionID,
		UserID:       session.UserID,
		RefreshToken: hashedNewToken,
		RefreshTokenSelector: tokenParts.Selector,
		Salt:         newSalt,
		DeviceID:     session.DeviceID,
		IPAddress:    clientIP,  // Update with current IP
		UserAgent:    userAgent, // Update with current user agent
		ExpiresAt:    session.ExpiresAt,
		CreatedAt:    now,
		LastUsedAt:   now,
		Revoked:      false,
	}

	// Add new session to SessionManager
	if err := s.sessionManager.CreateSession(ctx, session.UserID, newSessionID); err != nil {
		return nil, fmt.Errorf("failed to add new session to SessionManager: %w", err)
	}

	// Create the new session
	if err := s.repo.CreateSession(ctx, newSession); err != nil {
		// Rollback SessionManager changes if database creation fails
		if rollbackErr := s.sessionManager.DeleteSession(ctx, session.UserID, newSessionID); rollbackErr != nil {
			s.logger.Error("failed to rollback new session from SessionManager", "error", rollbackErr)
		}
		return nil, fmt.Errorf("failed to create new session: %w", err)
	}

	// Delete the old session
	if err := s.repo.DeleteSession(ctx, oldSessionID); err != nil {
		s.logger.Error("failed to delete old session", "error", err, "old_session_id", oldSessionID)
	}

	// Infrastructure concerns: Log security event
	s.logSecurityEvent(ctx, user.ID, "token_refreshed", fmt.Sprintf("Access token refreshed, old session %s replaced with %s", oldSessionID, newSessionID), clientIP, userAgent, "info")

	return tokenPair, nil
}

func (s *service) CreateSession(ctx context.Context, sessionID, userID, refreshToken, deviceID, clientIP, userAgent string) (*domain.Session, error) {
	// Parse refresh token to extract selector and verifier
	tokenParts, err := s.parseRefreshToken(refreshToken)
	if err != nil {
		return nil, err
	}

	// Create temporary session for validation
	tempSession := &domain.Session{RefreshTokenSelector: tokenParts.Selector}
	if err := tempSession.ValidateRefreshTokenSelector(); err != nil {
		return nil, fmt.Errorf("invalid refresh token format: %w", err)
	}

	// Hash the verifier part before storing (CRITICAL SECURITY FIX)
	hashedToken, salt, err := s.hashToken(tokenParts.Verifier)
	if err != nil {
		return nil, fmt.Errorf("failed to hash refresh token: %w", err)
	}

	// Check concurrent session limit using SessionManager
	if err := s.sessionManager.CreateSession(ctx, userID, sessionID); err != nil {
		// Propagate ErrTooManySessions without wrapping
		if errors.Is(err, domain.ErrTooManySessions) {
			return nil, err
		}

		return nil, fmt.Errorf("failed to enforce session limit: %w", err)
	}

	now := time.Now()
	session := &domain.Session{
		ID:                   sessionID, // Use the provided sessionID instead of generating new one
		UserID:               userID,
		RefreshToken:         hashedToken, // Store hashed verifier
		RefreshTokenSelector: tokenParts.Selector,    // Store selector for O(1) lookup
		Salt:                 salt,        // Store salt
		DeviceID:             deviceID,
		IPAddress:            clientIP,
		UserAgent:            userAgent,
		ExpiresAt:            now.Add(30 * 24 * time.Hour), // 30 days
		CreatedAt:            now,
		LastUsedAt:           now,
		Revoked:              false,
	}

	if err := s.repo.CreateSession(ctx, session); err != nil {
		// Rollback session from SessionManager if database creation fails
		if rollbackErr := s.sessionManager.DeleteSession(ctx, userID, sessionID); rollbackErr != nil {
			s.logger.Error("failed to rollback session from SessionManager", "error", rollbackErr)
		}
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	return session, nil
}

func (s *service) GetSession(ctx context.Context, sessionID string) (*domain.Session, error) {
	return s.repo.GetSession(ctx, sessionID)
}

func (s *service) GetUserSessions(ctx context.Context, userID string) ([]*domain.SessionInfo, error) {
	sessions, err := s.repo.ListUserSessions(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list sessions: %w", err)
	}

	sessionInfos := make([]*domain.SessionInfo, len(sessions))
	for i, session := range sessions {
		sessionInfos[i] = &domain.SessionInfo{
			ID:         session.ID,
			DeviceID:   session.DeviceID,
			IPAddress:  session.IPAddress,
			UserAgent:  session.UserAgent,
			CreatedAt:  session.CreatedAt,
			LastUsedAt: session.LastUsedAt,
			IsCurrent:  false, // TODO: Determine current session
		}

		// Parse user agent for better display
		if session.UserAgent != "" {
			// Simple parsing - in production use a proper UA parser
			if strings.Contains(session.UserAgent, "Chrome") {
				sessionInfos[i].Location = "Chrome Browser"
			} else if strings.Contains(session.UserAgent, "Firefox") {
				sessionInfos[i].Location = "Firefox Browser"
			} else if strings.Contains(session.UserAgent, "Safari") {
				sessionInfos[i].Location = "Safari Browser"
			}
		}
	}

	return sessionInfos, nil
}

func (s *service) RevokeSession(ctx context.Context, userID, sessionID string) error {
	// Verify session belongs to user
	session, err := s.repo.GetSession(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("session not found: %w", err)
	}

	if session.UserID != userID {
		return fmt.Errorf("unauthorized")
	}

	// Blacklist refresh token
	if err := s.repo.BlacklistRefreshToken(ctx, session.RefreshToken, session.ExpiresAt); err != nil {
		s.logger.Error("failed to blacklist refresh token", "error", err)
	}

	// Remove session from SessionManager
	if err := s.sessionManager.DeleteSession(ctx, userID, sessionID); err != nil {
		s.logger.Error("failed to remove session from SessionManager", "error", err)
	}

	// Delete session
	if err := s.repo.DeleteSession(ctx, sessionID); err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}

	// Log security event
	s.logSecurityEvent(ctx, userID, "session_revoked", fmt.Sprintf("Session %s revoked", sessionID), "", "", "info")

	return nil
}

func (s *service) RevokeAllSessions(ctx context.Context, userID string, exceptSessionID string) error {
	// Get all user sessions
	sessions, err := s.repo.ListUserSessions(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to list sessions: %w", err)
	}

	// Blacklist all refresh tokens except current
	for _, session := range sessions {
		if session.ID != exceptSessionID {
			if err := s.repo.BlacklistRefreshToken(ctx, session.RefreshToken, session.ExpiresAt); err != nil {
				s.logger.Error("failed to blacklist refresh token", "error", err)
			}

			// Remove session from SessionManager
			if err := s.sessionManager.DeleteSession(ctx, userID, session.ID); err != nil {
				s.logger.Error("failed to remove session from SessionManager", "error", err)
			}
		}
	}

	// Delete all sessions except current
	if err := s.repo.DeleteUserSessions(ctx, userID, exceptSessionID); err != nil {
		return fmt.Errorf("failed to delete sessions: %w", err)
	}

	// Log security event
	s.logSecurityEvent(ctx, userID, "all_sessions_revoked", "All sessions revoked except current", "", "", "warning")

	return nil
}

func (s *service) ValidateSession(ctx context.Context, sessionID, clientIP string) error {
	session, err := s.repo.GetSession(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("session not found: %w", err)
	}

	// Check if expired
	if session.ExpiresAt.Before(time.Now()) {
		return fmt.Errorf("session expired")
	}

	// Check IP change (optional security measure)
	if session.IPAddress != clientIP {
		s.logSecurityEvent(ctx, session.UserID, "session_ip_changed",
			fmt.Sprintf("Session IP changed from %s to %s", session.IPAddress, clientIP),
			clientIP, session.UserAgent, "warning")
	}

	// Update last used
	session.LastUsedAt = time.Now()
	if err := s.repo.UpdateSession(ctx, session); err != nil {
		s.logger.Error("failed to update session last used", "error", err)
	}

	return nil
}

func (s *service) ValidateAccessToken(ctx context.Context, tokenString string) (*domain.Claims, error) {
	// Handle development tokens
	if strings.HasPrefix(tokenString, "dev_token_") {
		// For development mode, return mock claims
		return &domain.Claims{
			RegisteredClaims: jwt.RegisteredClaims{
				Subject:   "dev-user-1",
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
				IssuedAt:  jwt.NewNumericDate(time.Now()),
			},
			UserID:    "dev-user-1",
			Email:     "test@hexabase.com",
			Name:      "Test User",
			Provider:  "credentials",
			OrgIDs:    []string{"dev-org-1"}, // Include development organization
			SessionID: "dev-session-1",
		}, nil
	}

	// Get public key for production tokens
	publicKeyPEM, err := s.keyRepo.GetPublicKey()
	if err != nil {
		return nil, fmt.Errorf("failed to get public key: %w", err)
	}

	// Parse and validate token
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Verify signing method
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		// Parse public key
		publicKey, err := jwt.ParseRSAPublicKeyFromPEM(publicKeyPEM)
		if err != nil {
			return nil, fmt.Errorf("failed to parse public key: %w", err)
		}

		return publicKey, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	if !token.Valid {
		return nil, fmt.Errorf("token is invalid")
	}

	// Extract claims
	mapClaims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("failed to extract claims")
	}

	claims := &domain.Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject: mapClaims["sub"].(string),
			ExpiresAt: func() *jwt.NumericDate {
				if exp, ok := mapClaims["exp"].(float64); ok {
					return jwt.NewNumericDate(time.Unix(int64(exp), 0))
				}
				return nil
			}(),
			IssuedAt: func() *jwt.NumericDate {
				if iat, ok := mapClaims["iat"].(float64); ok {
					return jwt.NewNumericDate(time.Unix(int64(iat), 0))
				}
				return nil
			}(),
		},
		UserID:   mapClaims["sub"].(string),
		Email:    mapClaims["email"].(string),
		Name:     mapClaims["name"].(string),
		Provider: mapClaims["provider"].(string),
	}

	// Set SessionID if present
	if sessionID, ok := mapClaims["session_id"].(string); ok {
		claims.SessionID = sessionID
	} else {
		claims.SessionID = legacySessionID
	}

	// Extract org IDs if present
	if orgIDs, ok := mapClaims["org_ids"].([]interface{}); ok {
		claims.OrgIDs = make([]string, len(orgIDs))
		for i, id := range orgIDs {
			claims.OrgIDs[i] = id.(string)
		}
	}

	// Check if session is blocked (skip legacy sessions for backwards compatibility)
	if claims.SessionID != legacySessionID {
		blocked, err := s.repo.IsSessionBlocked(ctx, claims.SessionID)
		if err != nil {
			s.logger.Error("failed to check session blocklist", "error", err, "session_id", claims.SessionID)
			// If Redis is down, we cannot confirm session validity, so we must deny access.
			return nil, fmt.Errorf("could not verify session validity: upstream service unavailable: %w", err)
		} else if blocked {
			return nil, fmt.Errorf("session has been invalidated")
		}
	}

	return claims, nil
}

func (s *service) GenerateWorkspaceToken(ctx context.Context, userID, workspaceID string) (string, error) {
	// Verify user exists
	_, err := s.repo.GetUser(ctx, userID)
	if err != nil {
		return "", fmt.Errorf("user not found: %w", err)
	}

	// Get user's workspace groups
	groups, err := s.repo.GetUserWorkspaceGroups(ctx, userID, workspaceID)
	if err != nil {
		return "", fmt.Errorf("failed to get user groups: %w", err)
	}

	// Create workspace claims
	claims := &domain.WorkspaceClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
		Subject:     userID,
		WorkspaceID: workspaceID,
		Groups:      groups,
	}

	// Get private key
	privateKeyPEM, err := s.keyRepo.GetPrivateKey()
	if err != nil {
		return "", fmt.Errorf("failed to get private key: %w", err)
	}

	// Parse private key
	privateKey, err := jwt.ParseRSAPrivateKeyFromPEM(privateKeyPEM)
	if err != nil {
		return "", fmt.Errorf("failed to parse private key: %w", err)
	}

	// Create token
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tokenString, err := token.SignedString(privateKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, nil
}

// generateTokenPairFromClaims creates a token pair from pre-generated claims
func (s *service) generateTokenPairFromClaims(ctx context.Context, claims *domain.Claims) (*domain.TokenPair, error) {
	// Use TokenManager to sign claims
	accessToken, err := s.tokenManager.SignClaims(claims)
	if err != nil {
		return nil, fmt.Errorf("failed to sign access token: %w", err)
	}

	// Generate refresh token
	refreshToken, err := s.generateRefreshToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	// Calculate expiration time from claims
	var expiresIn int
	if claims.ExpiresAt != nil && claims.IssuedAt != nil {
		expiresIn = int(claims.ExpiresAt.Sub(claims.IssuedAt.Time).Seconds())
	} else {
		// Use configurable default instead of hardcoded value
		// TODO: Consider making this configurable per user/organization/session type
		expiresIn = s.defaultTokenExpiry
		s.logger.Warn("using default token expiry due to missing claims timestamps",
			"default_expiry_seconds", expiresIn,
			"user_id", claims.UserID,
		)
	}

	return &domain.TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    expiresIn,
	}, nil
}

// generateRefreshToken generates a new refresh token in selector.verifier format
func (s *service) generateRefreshToken() (string, error) {
	// Generate selector (16 bytes = 22 chars base64)
	selectorBytes := make([]byte, 16)
	if _, err := rand.Read(selectorBytes); err != nil {
		return "", fmt.Errorf("failed to generate selector: %w", err)
	}
	selector := base64.URLEncoding.EncodeToString(selectorBytes)

	// Generate verifier (32 bytes = 43 chars base64)
	verifierBytes := make([]byte, 32)
	if _, err := rand.Read(verifierBytes); err != nil {
		return "", fmt.Errorf("failed to generate verifier: %w", err)
	}
	verifier := base64.URLEncoding.EncodeToString(verifierBytes)

	// Return in selector.verifier format for O(1) lookup
	return s.buildRefreshToken(selector, verifier), nil
}

func (s *service) RevokeRefreshToken(ctx context.Context, refreshToken string) error {
	// Get session to find expiry
	session, err := s.getSessionByRefreshToken(ctx, refreshToken)
	if err != nil {
		return fmt.Errorf("session not found: %w", err)
	}

	// Blacklist token
	if err := s.repo.BlacklistRefreshToken(ctx, refreshToken, session.ExpiresAt); err != nil {
		return fmt.Errorf("failed to blacklist refresh token: %w", err)
	}

	return nil
}

func (s *service) InvalidateSession(ctx context.Context, sessionID string) error {
	// Get session to determine TTL
	session, err := s.repo.GetSession(ctx, sessionID)
	if err != nil {
		// If session doesn't exist, it's already effectively invalidated
		s.logger.Warn("session not found during invalidation", "session_id", sessionID, "error", err)
		return nil // Don't return error for missing sessions during logout
	}

	// Block session in Redis with TTL matching session expiry
	if err := s.repo.BlockSession(ctx, sessionID, session.ExpiresAt); err != nil {
		return fmt.Errorf("failed to block session: %w", err)
	}

	// Log security event for session invalidation
	s.logSecurityEvent(ctx, session.UserID, "session_invalidated", "Session manually invalidated", "", "", "info")

	return nil
}

func (s *service) GetUser(ctx context.Context, userID string) (*domain.User, error) {
	return s.repo.GetUser(ctx, userID)
}

func (s *service) GetCurrentUser(ctx context.Context, token string) (*domain.User, error) {
	// Validate token and get claims
	claims, err := s.ValidateAccessToken(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	// Get user
	user, err := s.repo.GetUser(ctx, claims.UserID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	return user, nil
}

func (s *service) UpdateUserProfile(ctx context.Context, userID string, updates map[string]interface{}) (*domain.User, error) {
	user, err := s.repo.GetUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	// Apply updates
	if displayName, ok := updates["display_name"].(string); ok {
		user.DisplayName = displayName
	}
	if avatarURL, ok := updates["avatar_url"].(string); ok {
		user.AvatarURL = avatarURL
	}

	user.UpdatedAt = time.Now()

	if err := s.repo.UpdateUser(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	return user, nil
}

func (s *service) LogSecurityEvent(ctx context.Context, event *domain.SecurityEvent) error {
	event.ID = uuid.New().String()
	event.CreatedAt = time.Now()

	if err := s.repo.CreateSecurityEvent(ctx, event); err != nil {
		return fmt.Errorf("failed to log security event: %w", err)
	}

	return nil
}

func (s *service) GetSecurityLogs(ctx context.Context, filter domain.SecurityLogFilter) ([]*domain.SecurityEvent, error) {
	return s.repo.ListSecurityEvents(ctx, filter)
}

func (s *service) GetJWKS(ctx context.Context) ([]byte, error) {
	return s.keyRepo.GetJWKS()
}

func (s *service) verifyAuthState(authState *domain.AuthState, clientIP string) error {
	// Check expiry
	if authState.ExpiresAt.Before(time.Now()) {
		return fmt.Errorf("auth state expired")
	}

	// Optionally check IP
	// if authState.ClientIP != clientIP {
	// 	return fmt.Errorf("client IP mismatch")
	// }

	return nil
}

func (s *service) verifyPKCE(authState *domain.AuthState, codeVerifier string) error {
	if authState.CodeChallenge == "" {
		return nil // PKCE not required
	}

	// Verify code verifier matches stored challenge
	// Use RawURLEncoding (no padding) as required by RFC 7636
	h := sha256.New()
	h.Write([]byte(codeVerifier))
	computedChallenge := base64.RawURLEncoding.EncodeToString(h.Sum(nil))

	if computedChallenge != authState.CodeChallenge {
		s.logger.Warn("PKCE verification failed", "state", authState.State)
		return fmt.Errorf("PKCE verification failed")
	}

	return nil
}

func (s *service) StoreAuthState(ctx context.Context, state *domain.AuthState) error {
	return s.repo.StoreAuthState(ctx, state)
}

// Helper functions

func (s *service) generateTokenPair(ctx context.Context, user *domain.User, sessionID string) (*domain.TokenPair, error) {
	// Get user's organizations
	orgIDs, err := s.repo.GetUserOrganizations(ctx, user.ID)
	if err != nil {
		s.logger.Warn("failed to get user organizations", "error", err)
		orgIDs = []string{}
	}

	// Create domain claims
	now := time.Now()
	claims := &domain.Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.ID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(15 * time.Minute)),
			Issuer:    "https://api.hexabase-kaas.io",
			Audience:  []string{"hexabase-api"},
		},
		UserID:    user.ID,
		Email:     user.Email,
		Name:      user.DisplayName,
		Provider:  user.Provider,
		OrgIDs:    orgIDs,
		SessionID: sessionID,
	}

	// Use common token pair generation logic
	return s.generateTokenPairFromClaims(ctx, claims)
}

func (s *service) logSecurityEvent(ctx context.Context, userID, eventType, description, ipAddress, userAgent, level string) {
	event := &domain.SecurityEvent{
		ID:          uuid.New().String(),
		UserID:      userID,
		EventType:   eventType,
		Description: description,
		IPAddress:   ipAddress,
		UserAgent:   userAgent,
		Level:       level,
		CreatedAt:   time.Now(),
	}

	if err := s.repo.CreateSecurityEvent(ctx, event); err != nil {
		s.logger.Error("failed to log security event", "error", err)
	}
}

func (s *service) GenerateInternalAIOpsToken(ctx context.Context, userID string, orgIDs []string, activeWorkspaceID string) (string, error) {
	claims := &InternalAIOpsClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(10 * time.Second)), // Very short-lived
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "hexabase-control-plane",
			Audience:  jwt.ClaimStrings{"hexabase-aiops-service"},
		},
		UserID:            userID,
		OrgIDs:            orgIDs,
		ActiveWorkspaceID: activeWorkspaceID,
		TokenType:         "internal-aiops-v1",
	}

	privateKeyPEM, err := s.keyRepo.GetPrivateKey()
	if err != nil {
		return "", fmt.Errorf("failed to get private key for internal token: %w", err)
	}

	privateKey, err := jwt.ParseRSAPrivateKeyFromPEM(privateKeyPEM)
	if err != nil {
		return "", fmt.Errorf("failed to parse private key for internal token: %w", err)
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tokenString, err := token.SignedString(privateKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign internal token: %w", err)
	}

	return tokenString, nil
}

// hashToken applies business validation and delegates to repository for crypto operations
func (s *service) hashToken(token string) (hashedToken string, salt string, err error) {
	// Business validation
	if token == "" {
		return "", "", fmt.Errorf("token cannot be empty")
	}

	if len(token) < 8 {
		return "", "", fmt.Errorf("token must be at least 8 characters long for security")
	}

	// Delegate to repository for crypto implementation
	hashedToken, salt, err = s.repo.HashToken(token)
	if err != nil {
		return "", "", fmt.Errorf("failed to hash token: %w", err)
	}

	// Business validation: ensure output meets security requirements
	if len(hashedToken) != 64 || len(salt) != 64 {
		return "", "", fmt.Errorf("hash generation failed security validation")
	}

	return hashedToken, salt, nil
}

// verifyToken applies business validation and delegates to repository for crypto operations
func (s *service) verifyToken(plainToken, hashedToken, salt string) bool {
	// Business validation
	if plainToken == "" || hashedToken == "" || salt == "" {
		return false
	}

	// Business rule: tokens must meet minimum security requirements
	if len(plainToken) < 8 {
		return false
	}

	// Business validation: hash and salt must be proper format
	if len(hashedToken) != 64 || len(salt) != 64 {
		return false
	}

	// Delegate to repository for crypto verification
	return s.repo.VerifyToken(plainToken, hashedToken, salt)
}

// getSessionByRefreshToken implements optimized session lookup using selector/verifier pattern
func (s *service) getSessionByRefreshToken(ctx context.Context, refreshToken string) (*domain.Session, error) {
	s.logger.Info("getSessionByRefreshToken called", "token_length", len(refreshToken))

	// Parse refresh token in selector.verifier format for O(1) lookup
	tokenParts, err := s.parseRefreshToken(refreshToken)
	if err != nil {
		s.logger.Warn("failed to parse refresh token", "error", err, "token_length", len(refreshToken))
		return nil, err
	}

	s.logger.Debug("parsed refresh token", "selector", tokenParts.Selector, "verifier_length", len(tokenParts.Verifier))

	// Create temporary session for validation
	tempSession := &domain.Session{RefreshTokenSelector: tokenParts.Selector}
	if err := tempSession.ValidateRefreshTokenSelector(); err != nil {
		s.logger.Warn("refresh token selector validation failed", "error", err, "selector", tokenParts.Selector)
		return nil, fmt.Errorf("invalid refresh token format: %w", err)
	}

	// O(1) database lookup using selector
	session, err := s.repo.GetSessionByRefreshTokenSelector(ctx, tokenParts.Selector)
	if err != nil {
		s.logger.Warn("session lookup failed", "error", err, "selector", tokenParts.Selector)
		return nil, fmt.Errorf("session not found")
	}

	s.logger.Debug("found session", "session_id", session.ID, "has_salt", session.Salt != "")

	// Verify the verifier part using crypto hash comparison
	if session.Salt != "" && s.verifyToken(tokenParts.Verifier, session.RefreshToken, session.Salt) {
		s.logger.Debug("token verification successful")
		return session, nil
	}

	s.logger.Warn("token verification failed", "has_salt", session.Salt != "", "verifier_length", len(tokenParts.Verifier))
	return nil, fmt.Errorf("session not found")
}

// Helper functions for refresh token processing

// parseRefreshToken parses a refresh token into selector and verifier components
func (s *service) parseRefreshToken(refreshToken string) (*RefreshTokenParts, error) {
	parts := strings.Split(refreshToken, refreshTokenSeparator)
	if len(parts) != refreshTokenExpectedParts || parts[0] == "" || parts[1] == "" {
		return nil, fmt.Errorf("invalid refresh token format: expected selector%sverifier", refreshTokenSeparator)
	}

	return &RefreshTokenParts{
		Selector: parts[0],
		Verifier: parts[1],
	}, nil
}

// buildRefreshToken combines selector and verifier into a refresh token
func (s *service) buildRefreshToken(selector, verifier string) string {
	return selector + refreshTokenSeparator + verifier
}
