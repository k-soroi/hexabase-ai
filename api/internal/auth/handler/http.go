package handler

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/hexabase/hexabase-ai/api/internal/auth/domain"
	"github.com/hexabase/hexabase-ai/api/internal/shared/utils/httpauth"
)

// Handler handles authentication-related HTTP requests
type Handler struct {
	service domain.Service
	logger  *slog.Logger
}

// NewHandler creates a new auth handler
func NewHandler(service domain.Service, logger *slog.Logger) *Handler {
	return &Handler{
		service: service,
		logger:  logger,
	}
}

// Login initiates OAuth flow with external provider
func (h *Handler) Login(c *gin.Context) {
	provider := c.Param("provider")

	var req domain.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// Use default if not provided
		req.Provider = provider
	}

	authURL, state, err := h.service.GetAuthURL(c.Request.Context(), &req)
	if err != nil {
		h.logger.Error("failed to generate auth URL",
			"error", err,
			"provider", provider)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid provider or configuration error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"provider": provider,
		"auth_url": authURL,
		"state":    state,
	})
}

// Callback handles OAuth callback from external provider
func (h *Handler) Callback(c *gin.Context) {
	provider := c.Param("provider")

	// Handle both query params (for redirect) and JSON body (for PKCE)
	code := c.Query("code")
	state := c.Query("state")
	var codeVerifier string

	// If not in query, check JSON body for PKCE flow
	if code == "" {
		var req domain.CallbackRequest
		if err := c.ShouldBindJSON(&req); err == nil {
			code = req.Code
			state = req.State
			codeVerifier = req.CodeVerifier
		}
	}

	if code == "" || state == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing authorization code or state"})
		return
	}

	clientIP := c.ClientIP()
	userAgent := c.GetHeader("User-Agent")

	req := &domain.CallbackRequest{
		Code:         code,
		State:        state,
		CodeVerifier: codeVerifier,
	}

	authResp, err := h.service.HandleCallback(c.Request.Context(), req, clientIP, userAgent)
	if err != nil {
		// Check if it's a session limit error
		if errors.Is(err, domain.ErrTooManySessions) {
			h.logger.Warn("concurrent session limit exceeded",
				"provider", provider,
				"client_ip", clientIP)
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "concurrent sessions exceeded"})

			return
		}

		h.logger.Error("OAuth callback failed",
			"error", err,
			"provider", provider)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "authentication failed"})
		return
	}

	h.logger.Info("OAuth login successful",
		"provider", provider,
		"user_id", authResp.User.ID,
		"email", authResp.User.Email)

	c.JSON(http.StatusOK, gin.H{
		"access_token":  authResp.AccessToken,
		"refresh_token": authResp.RefreshToken,
		"token_type":    authResp.TokenType,
		"expires_in":    authResp.ExpiresIn,
		"user": gin.H{
			"id":           authResp.User.ID,
			"email":        authResp.User.Email,
			"display_name": authResp.User.DisplayName,
			"provider":     authResp.User.Provider,
		},
	})
}

// RefreshToken handles token refresh requests
func (h *Handler) RefreshToken(c *gin.Context) {
	var req domain.RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	clientIP := c.ClientIP()
	userAgent := c.GetHeader("User-Agent")

	tokenPair, err := h.service.RefreshToken(c.Request.Context(), req.RefreshToken, clientIP, userAgent)
	if err != nil {
		h.logger.Error("token refresh failed", "error", err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired refresh token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token":  tokenPair.AccessToken,
		"refresh_token": tokenPair.RefreshToken,
		"token_type":    tokenPair.TokenType,
		"expires_in":    tokenPair.ExpiresIn,
	})
}

// Logout invalidates user session
func (h *Handler) Logout(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	_ = c.ShouldBindJSON(&req)

	userID := c.GetString("user_id")
	sessionID := c.GetString("session_id")

	h.logger.Info("logout attempt", "user_id", userID, "session_id", sessionID)

	// Invalidate session immediately if session_id is available
	if sessionID != "" {
		if err := h.service.InvalidateSession(c.Request.Context(), sessionID); err != nil {
			h.logger.Error("failed to invalidate session", "error", err, "session_id", sessionID)
			// Don't return error - continue with logout process
		} else {
			h.logger.Info("session invalidated successfully", "session_id", sessionID)
		}
	}

	// Revoke refresh token if provided
	if req.RefreshToken != "" {
		if err := h.service.RevokeRefreshToken(c.Request.Context(), req.RefreshToken); err != nil {
			h.logger.Warn("failed to revoke refresh token", "error", err)
		}
	}

	// Log security event
	event := &domain.SecurityEvent{
		UserID:      userID,
		EventType:   "logout",
		Description: "User logged out",
		IPAddress:   c.ClientIP(),
		UserAgent:   c.GetHeader("User-Agent"),
		Level:       "info",
		Metadata:    make(map[string]any),
	}

	if err := h.service.LogSecurityEvent(c.Request.Context(), event); err != nil {
		h.logger.Error("failed to log security event", "error", err)
	}

	c.JSON(http.StatusOK, gin.H{"message": "successfully logged out"})
}

// GetCurrentUser returns current user information
func (h *Handler) GetCurrentUser(c *gin.Context) {
	// Get token from header
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing authorization header"})
		return
	}

	token := httpauth.TrimBearerPrefix(authHeader)

	user, err := h.service.GetCurrentUser(c.Request.Context(), token)
	if err != nil {
		h.logger.Error("failed to get current user", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get user information"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user": gin.H{
			"id":           user.ID,
			"email":        user.Email,
			"display_name": user.DisplayName,
			"provider":     user.Provider,
			"created_at":   user.CreatedAt,
			"last_login":   user.LastLoginAt,
		},
	})
}

// GetSessions returns active sessions for the current user
func (h *Handler) GetSessions(c *gin.Context) {
	userID := c.GetString("user_id")

	sessions, err := h.service.GetUserSessions(c.Request.Context(), userID)
	if err != nil {
		h.logger.Error("failed to get sessions", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve sessions"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"sessions": sessions,
		"total":    len(sessions),
	})
}

// RevokeSession revokes a specific session
func (h *Handler) RevokeSession(c *gin.Context) {
	sessionID := c.Param("sessionId")
	userID := c.GetString("user_id")

	if err := h.service.RevokeSession(c.Request.Context(), userID, sessionID); err != nil {
		h.logger.Error("failed to revoke session", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to revoke session"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "session revoked successfully"})
}

// RevokeAllSessions revokes all sessions except current
func (h *Handler) RevokeAllSessions(c *gin.Context) {
	userID := c.GetString("user_id")
	sessionID := c.GetString("session_id") // Set by middleware

	if err := h.service.RevokeAllSessions(c.Request.Context(), userID, sessionID); err != nil {
		h.logger.Error("failed to revoke sessions", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to revoke sessions"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "all other sessions revoked successfully"})
}

// GetSecurityLogs returns security logs for the current user
func (h *Handler) GetSecurityLogs(c *gin.Context) {
	userID := c.GetString("user_id")

	filter := domain.SecurityLogFilter{
		UserID: userID,
		Limit:  50, // Last 50 events
	}

	logs, err := h.service.GetSecurityLogs(c.Request.Context(), filter)
	if err != nil {
		h.logger.Error("failed to get security logs", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve security logs"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"logs":  logs,
		"total": len(logs),
	})
}

// JWKS returns JSON Web Key Set for token verification
func (h *Handler) JWKS(c *gin.Context) {
	jwks, err := h.service.GetJWKS(c.Request.Context())
	if err != nil {
		h.logger.Error("failed to get JWKS", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve public keys"})
		return
	}

	c.Header("Content-Type", "application/json")
	c.Data(http.StatusOK, "application/json", jwks)
}

// AuthMiddleware validates JWT tokens and sets user context
func (h *Handler) AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "missing authorization header"})
			c.Abort()
			return
		}

		// Extract Bearer token
		if !httpauth.HasBearerPrefix(authHeader) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization header format"})
			c.Abort()
			return
		}
		token := httpauth.TrimBearerPrefix(authHeader)

		// Validate JWT token
		claims, err := h.service.ValidateAccessToken(c.Request.Context(), token)
		if err != nil {
			h.logger.Debug("token validation failed", "error", err)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			c.Abort()
			return
		}

		// Set user info in context
		c.Set("user_id", claims.Subject)
		c.Set("user_email", claims.Email)
		c.Set("user_name", claims.Name)
		c.Set("provider", claims.Provider)
		c.Set("org_ids", claims.OrgIDs)
		c.Set("session_id", claims.SessionID)

		c.Next()
	}
}
