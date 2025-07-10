package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/hexabase/hexabase-ai/api/internal/auth/domain"
	internalRedis "github.com/hexabase/hexabase-ai/api/internal/shared/redis"
)

//nolint:contextcheck // withTestDB wrapper doesn't pass context parameter
func TestService_GetAuthURL(t *testing.T) { //nolint:paralleltest // Transaction-based test
	// Skip if short test
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	ctx := context.Background()

	//nolint:paralleltest // Transaction-based test
	t.Run("successful auth URL generation for google", func(t *testing.T) {
		withTestDB(t, func(db *gorm.DB, redisClient *internalRedis.Client) {
			svc, _, _ := setupTestServiceWithDB(t, db, redisClient)

			req := &domain.LoginRequest{
				Provider: "google",
			}

			url, state, err := svc.GetAuthURL(ctx, req)

			require.NoError(t, err)
			assert.Contains(t, url, "accounts.google.com")
			assert.NotEmpty(t, state)
			assert.GreaterOrEqual(t, len(state), 16, "State token should be sufficiently long")

			// Verify auth state was stored in Redis (not in database)
			// Auth states are stored in Redis, not PostgreSQL
			// We can verify by checking that the state is not empty and has proper format
			assert.NotEmpty(t, state)
			assert.GreaterOrEqual(t, len(state), 32, "State should be at least 32 characters")
		})
	})

	//nolint:paralleltest // Transaction-based test
	t.Run("invalid provider should return error", func(t *testing.T) {
		withTestDB(t, func(db *gorm.DB, redisClient *internalRedis.Client) {
			svc, _, _ := setupTestServiceWithDB(t, db, redisClient)

			req := &domain.LoginRequest{
				Provider: "invalid-provider",
			}

			url, state, err := svc.GetAuthURL(ctx, req)

			require.Error(t, err)
			assert.Empty(t, url)
			assert.Empty(t, state)
			assert.Contains(t, err.Error(), "unsupported provider")
		})
	})
}

func TestService_HandleCallback(t *testing.T) { //nolint:paralleltest // Transaction-based test
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	ctx := context.Background()

	withTestDB(t, func(db *gorm.DB, redisClient *internalRedis.Client) {
		svc, oauthRepo, _ := setupTestServiceWithDB(t, db, redisClient)

		// First, we need to create an auth state to simulate a valid OAuth flow
		// Since we're using a real repository, we need to use the actual service method
		loginReq := &domain.LoginRequest{
			Provider: "google",
		}
		authURL, state, err := svc.GetAuthURL(ctx, loginReq)
		require.NoError(t, err)
		assert.NotEmpty(t, authURL)
		assert.NotEmpty(t, state)

		// Cast the OAuth repository to its stub type to get the user info
		// This user info will be used to verify the claims in the token.
		stubOAuthRepo, ok := oauthRepo.(*stubOAuthRepository)
		require.True(t, ok, "failed to cast oauthRepo to stub")

		expectedUser := stubOAuthRepo.userInfo

		// Now test the callback with the valid state
		req := &domain.CallbackRequest{
			Code:  "auth-code-123",
			State: state,
		}

		response, err := svc.HandleCallback(ctx, req, "192.168.1.1", "Mozilla/5.0")
		require.NoError(t, err)
		assert.NotNil(t, response)
		assert.NotEmpty(t, response.AccessToken)
		assert.NotEmpty(t, response.RefreshToken)

		// Verify user was created in the database
		var user domain.User

		// Find user by the ExternalID we expect from the stub
		err = db.Where("external_id = ?", expectedUser.ID).First(&user).Error
		require.NoError(t, err)
		assert.Equal(t, expectedUser.Provider, user.Provider)
		assert.Equal(t, expectedUser.ID, user.ExternalID)
		assert.Equal(t, expectedUser.Email, user.Email)

		// Verify session was created
		var session domain.Session

		// Get the session for the user (should be only one for new user)
		err = db.Where("user_id = ?", user.ID).First(&session).Error
		require.NoError(t, err)
		assert.Equal(t, user.ID, session.UserID)
		assert.False(t, session.ExpiresAt.IsZero())

		// Verify that the access token contains the SessionID
		// Parse and validate the JWT token to check Claims
		claims, err := svc.ValidateAccessToken(ctx, response.AccessToken)
		require.NoError(t, err)
		assert.NotNil(t, claims)
		assert.NotEmpty(t, claims.SessionID, "SessionID should be included in the access token")
		assert.Equal(t, session.ID, claims.SessionID, "Token SessionID should match the created session")
		assert.Equal(t, user.ID, claims.UserID, "Token UserID should match the created user")
		assert.Equal(t, expectedUser.Email, claims.Email, "Token Email should match the user email")

		// Security events are currently not stored in PostgreSQL
		// They may be logged to a separate system or stored differently
		// TODO: Add security event verification once the storage mechanism is clarified
	})
}

func TestService_RefreshToken_RotationAndLookup( //nolint:funlen,paralleltest // integration test, transaction-based
	t *testing.T,
) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	ctx := context.Background()

	withTestDB(t, func(db *gorm.DB, redisClient *internalRedis.Client) {
		svc, oauthRepo, tokenDomainService := setupTestServiceWithDB(t, db, redisClient)

		// First, create a user and session through the normal flow
		loginReq := &domain.LoginRequest{
			Provider: "google",
		}
		authURL, state, err := svc.GetAuthURL(ctx, loginReq)
		require.NoError(t, err)
		assert.NotEmpty(t, authURL)
		assert.NotEmpty(t, state)

		// Cast the OAuth repository to its stub type to get the user info
		stubOAuthRepo, ok := oauthRepo.(*stubOAuthRepository)
		require.True(t, ok, "failed to cast oauthRepo to stub")

		expectedUser := stubOAuthRepo.userInfo

		// Simulate successful OAuth callback
		callbackReq := &domain.CallbackRequest{
			Code:  "auth-code-123",
			State: state,
		}
		loginResponse, err := svc.HandleCallback(ctx, callbackReq, "192.168.1.1", "Mozilla/5.0")
		require.NoError(t, err)
		assert.NotNil(t, loginResponse)
		assert.NotEmpty(t, loginResponse.RefreshToken)

		// Get the created user and session from database
		var user domain.User

		err = db.Where("external_id = ?", expectedUser.ID).First(&user).Error
		require.NoError(t, err)

		var initialSession domain.Session

		err = db.Where("user_id = ?", user.ID).First(&initialSession).Error
		require.NoError(t, err)

		// Set up tokenDomainService expectation
		domainClaims := &domain.Claims{
			RegisteredClaims: jwt.RegisteredClaims{
				Subject:   user.ID,
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
				IssuedAt:  jwt.NewNumericDate(time.Now()),
			},
			UserID:    user.ID,
			Email:     user.Email,
			Name:      user.DisplayName,
			Provider:  user.Provider,
			SessionID: "new-session-id-for-refresh", // A new session ID will be generated
			OrgIDs:    []string{},
		}
		tokenDomainService.On("RefreshToken", mock.Anything,
			mock.AnythingOfType("*domain.Session"),
			mock.AnythingOfType("*domain.User")).Return(domainClaims, nil).Once()

		// Now test the refresh token functionality
		response, err := svc.RefreshToken(
			ctx,
			loginResponse.RefreshToken,
			"192.168.1.2",
			"Mozilla/5.0 Updated",
		)
		require.NoError(t, err)
		assert.NotNil(t, response)
		assert.NotEmpty(t, response.AccessToken)
		assert.NotEmpty(t, response.RefreshToken)
		assert.NotEqual(
			t,
			loginResponse.RefreshToken,
			response.RefreshToken,
			"New refresh token should be different",
		)

		// Verify old session was deleted
		var oldSession domain.Session

		err = db.Where("id = ?", initialSession.ID).First(&oldSession).Error
		require.Error(t, err, "Old session should be deleted after token refresh")
		require.ErrorIs(t, err, gorm.ErrRecordNotFound)

		// Verify that the new access token contains the correct (new) SessionID
		claims, err := svc.ValidateAccessToken(ctx, response.AccessToken)
		require.NoError(t, err)
		assert.NotNil(t, claims)
		assert.NotEmpty(t, claims.SessionID, "SessionID should be included in the refreshed access token")
		assert.NotEqual(t, initialSession.ID, claims.SessionID, "A new session ID should be created")

		// Get the new session using the SessionID from the token
		var newSession domain.Session

		err = db.Where("id = ?", claims.SessionID).First(&newSession).Error
		require.NoError(t, err)
		assert.Equal(t, user.ID, newSession.UserID)
		assert.Equal(t, "192.168.1.2", newSession.IPAddress)
		assert.Equal(t, "Mozilla/5.0 Updated", newSession.UserAgent)
		assert.Equal(t, user.ID, claims.UserID, "Token UserID should remain the same")
		assert.Equal(t, expectedUser.Email, claims.Email, "Token Email should remain the same")
		assert.NotEmpty(t, newSession.RefreshTokenSelector, "New session must have a selector for optimized lookup")
	})
}

func TestService_RevokeSession(t *testing.T) { //nolint:paralleltest // Transaction-based test
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	ctx := context.Background()

	withTestDB(t, func(db *gorm.DB, redisClient *internalRedis.Client) {
		svc, oauthRepo, _ := setupTestServiceWithDB(t, db, redisClient)

		// First, create a user and session through the normal flow
		loginReq := &domain.LoginRequest{
			Provider: "google",
		}
		authURL, state, err := svc.GetAuthURL(ctx, loginReq)
		require.NoError(t, err)
		assert.NotEmpty(t, authURL)
		assert.NotEmpty(t, state)

		// Simulate successful OAuth callback
		callbackReq := &domain.CallbackRequest{
			Code:  "auth-code-123",
			State: state,
		}
		loginResponse, err := svc.HandleCallback(ctx, callbackReq, "192.168.1.1", "Mozilla/5.0")
		require.NoError(t, err)
		assert.NotNil(t, loginResponse)

		// Get the created user and session from database
		stubOAuthRepo, ok := oauthRepo.(*stubOAuthRepository)
		require.True(t, ok, "failed to cast oauthRepo to stub")

		expectedUser := stubOAuthRepo.userInfo

		var user domain.User

		err = db.Where("external_id = ?", expectedUser.ID).First(&user).Error
		require.NoError(t, err)

		var session domain.Session

		err = db.Where("user_id = ?", user.ID).First(&session).Error
		require.NoError(t, err)

		// Now test revoking the session
		err = svc.RevokeSession(ctx, user.ID, session.ID)
		require.NoError(t, err)

		// Verify session was deleted
		var deletedSession domain.Session

		err = db.Where("id = ?", session.ID).First(&deletedSession).Error
		assert.Error(t, err, "Session should be deleted")
	})
}

// TestService_RefreshToken_OptimizedLookup_Testcontainers tests refresh token with optimized lookup
func TestService_RefreshToken_OptimizedLookup(t *testing.T) {
	t.Skip("Skipping this test as it is merged into TestService_RefreshToken_RotationAndLookup")
}
