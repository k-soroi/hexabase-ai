//nolint:paralleltest
package repository_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/hexabase/hexabase-ai/api/internal/auth/domain"
	"github.com/hexabase/hexabase-ai/api/internal/auth/repository"
	internalRedis "github.com/hexabase/hexabase-ai/api/internal/shared/redis"
)

// uniqueID generates a unique identifier for test data
func uniqueID() string {
	return uuid.New().String()
}

// uniqueEmail generates a unique email address for test data
func uniqueEmail(prefix string) string {
	return fmt.Sprintf("%s-%s@example.com", prefix, uniqueID())
}

// uniqueExternalID generates a unique external ID for test data
func uniqueExternalID(prefix string) string {
	return fmt.Sprintf("%s-%s", prefix, uniqueID())
}

func TestPostgresRepository_CreateUser(t *testing.T) {
	ctx := context.Background()

	withTestDB(t, func(db *gorm.DB, redisClient *internalRedis.Client) {
		repo := repository.NewPostgresRepository(db)

		t.Run("successful user creation", func(t *testing.T) {
			user := &domain.User{
				ID:          uniqueID(),
				Email:       uniqueEmail("test"),
				DisplayName: "Test User",
				Provider:    "google",
				ExternalID:  uniqueExternalID("google-sub"),
			}

			err := repo.CreateUser(ctx, user)
			require.NoError(t, err)

			// Verify user was created by fetching it
			var savedUser domain.User

			err = db.Where("id = ?", user.ID).First(&savedUser).Error
			require.NoError(t, err)
			assert.Equal(t, user.Email, savedUser.Email)
			assert.Equal(t, user.DisplayName, savedUser.DisplayName)
			assert.Equal(t, user.Provider, savedUser.Provider)
			assert.Equal(t, user.ExternalID, savedUser.ExternalID)
		})

		t.Run("duplicate user error", func(t *testing.T) {
			// Create first user
			user1 := &domain.User{
				ID:          uniqueID(),
				Email:       uniqueEmail("duplicate"),
				DisplayName: "First User",
				Provider:    "google",
				ExternalID:  uniqueExternalID("google-sub-456"),
			}
			err := repo.CreateUser(ctx, user1)
			require.NoError(t, err)

			// Try to create user with same email
			user2 := &domain.User{
				ID:          uniqueID(),
				Email:       user1.Email, // Use the same email for duplicate test
				DisplayName: "Second User",
				Provider:    "google",
				ExternalID:  uniqueExternalID("google-sub-789"),
			}
			err = repo.CreateUser(ctx, user2)
			require.Error(t, err)
		})
	})
}

func TestPostgresRepository_GetUser(t *testing.T) {
	ctx := context.Background()

	withTestDB(t, func(db *gorm.DB, redisClient *internalRedis.Client) {
		repo := repository.NewPostgresRepository(db)

		t.Run("user found", func(t *testing.T) {
			// Create a user first
			user := &domain.User{
				ID:          uniqueID(),
				Email:       uniqueEmail("getuser"),
				DisplayName: "Get User Test",
				Provider:    "google",
				ExternalID:  uniqueExternalID("google-sub-getuser"),
			}
			err := repo.CreateUser(ctx, user)
			require.NoError(t, err)

			// Get the user
			foundUser, err := repo.GetUser(ctx, user.ID)
			require.NoError(t, err)
			assert.NotNil(t, foundUser)
			assert.Equal(t, user.ID, foundUser.ID)
			assert.Equal(t, user.Email, foundUser.Email)
		})

		t.Run("user not found", func(t *testing.T) {
			nonExistentID := uniqueID()
			foundUser, err := repo.GetUser(ctx, nonExistentID)
			require.Error(t, err)
			assert.Nil(t, foundUser)
			assert.Contains(t, err.Error(), "user not found")
		})
	})
}

func TestPostgresRepository_GetUserByEmail(t *testing.T) {
	ctx := context.Background()

	withTestDB(t, func(db *gorm.DB, redisClient *internalRedis.Client) {
		repo := repository.NewPostgresRepository(db)

		t.Run("user found", func(t *testing.T) {
			// Create a user first
			user := &domain.User{
				ID:          uniqueID(),
				Email:       uniqueEmail("getbyemail"),
				DisplayName: "Get By Email Test",
				Provider:    "google",
				ExternalID:  uniqueExternalID("google-sub-getbyemail"),
			}
			err := repo.CreateUser(ctx, user)
			require.NoError(t, err)

			// Get user by email
			foundUser, err := repo.GetUserByEmail(ctx, user.Email)
			require.NoError(t, err)
			assert.NotNil(t, foundUser)
			assert.Equal(t, user.Email, foundUser.Email)
			assert.Equal(t, user.ID, foundUser.ID)
		})

		t.Run("user not found", func(t *testing.T) {
			foundUser, err := repo.GetUserByEmail(ctx, "notfound@example.com")
			require.Error(t, err)
			assert.Nil(t, foundUser)
		})
	})
}

func TestPostgresRepository_GetUserByExternalID(t *testing.T) {
	ctx := context.Background()

	withTestDB(t, func(db *gorm.DB, redisClient *internalRedis.Client) {
		repo := repository.NewPostgresRepository(db)

		t.Run("user found by external ID", func(t *testing.T) {
			// Create a user first
			user := &domain.User{
				ID:          uniqueID(),
				Email:       uniqueEmail("getbyexternal"),
				DisplayName: "Test User",
				Provider:    "google",
				ExternalID:  uniqueExternalID("google-sub-789"),
			}
			err := repo.CreateUser(ctx, user)
			require.NoError(t, err)

			// Get user by external ID
			foundUser, err := repo.GetUserByExternalID(ctx, user.ExternalID, user.Provider)
			require.NoError(t, err)
			assert.NotNil(t, foundUser)
			assert.Equal(t, user.ExternalID, foundUser.ExternalID)
			assert.Equal(t, user.Provider, foundUser.Provider)
		})
	})
}

func TestPostgresRepository_UpdateUser(t *testing.T) {
	ctx := context.Background()

	withTestDB(t, func(db *gorm.DB, redisClient *internalRedis.Client) {
		repo := repository.NewPostgresRepository(db)

		t.Run("successful user update", func(t *testing.T) {
			// Create a user first
			user := &domain.User{
				ID:          uniqueID(),
				Email:       uniqueEmail("original"),
				DisplayName: "Original Name",
				Provider:    "google",
				ExternalID:  uniqueExternalID("google-sub-999"),
			}
			err := repo.CreateUser(ctx, user)
			require.NoError(t, err)

			// Update user
			updatedEmail := uniqueEmail("updated")
			user.Email = updatedEmail
			user.DisplayName = "Updated Name"
			err = repo.UpdateUser(ctx, user)
			require.NoError(t, err)

			// Verify update
			var updatedUser domain.User

			err = db.Where("id = ?", user.ID).First(&updatedUser).Error
			require.NoError(t, err)
			assert.Equal(t, updatedEmail, updatedUser.Email)
			assert.Equal(t, "Updated Name", updatedUser.DisplayName)
		})
	})
}

func TestPostgresRepository_UpdateLastLogin(t *testing.T) {
	ctx := context.Background()

	withTestDB(t, func(db *gorm.DB, redisClient *internalRedis.Client) {
		repo := repository.NewPostgresRepository(db)

		t.Run("update last login time", func(t *testing.T) {
			// Create a user first
			user := &domain.User{
				ID:          uniqueID(),
				Email:       uniqueEmail("lastlogin"),
				DisplayName: "Last Login Test",
				Provider:    "google",
				ExternalID:  uniqueExternalID("google-sub-lastlogin"),
			}
			err := repo.CreateUser(ctx, user)
			require.NoError(t, err)

			// Update last login
			err = repo.UpdateLastLogin(ctx, user.ID)
			require.NoError(t, err)

			// Verify update
			var updatedUser domain.User

			err = db.Where("id = ?", user.ID).First(&updatedUser).Error
			require.NoError(t, err)
			assert.WithinDuration(t, time.Now(), updatedUser.LastLoginAt, 5*time.Second)
		})
	})
}

func TestPostgresRepository_CreateSession(t *testing.T) {
	ctx := context.Background()

	withTestDB(t, func(db *gorm.DB, redisClient *internalRedis.Client) {
		repo := repository.NewPostgresRepository(db)

		t.Run("create new session with already-hashed refresh token", func(t *testing.T) {
			// Create a user first
			user := &domain.User{
				ID:          uniqueID(),
				Email:       uniqueEmail("createsession"),
				DisplayName: "Create Session Test",
				Provider:    "google",
				ExternalID:  uniqueExternalID("google-sub-createsession"),
			}
			err := repo.CreateUser(ctx, user)
			require.NoError(t, err)

			plainToken := "refresh-token-123"

			// Simulate service layer hashing the token first
			tokenHashRepo := repository.NewTokenHashRepository()
			hashedToken, salt, err := tokenHashRepo.HashToken(plainToken)
			require.NoError(t, err)

			session := &domain.Session{
				ID:                   uniqueID(),
				UserID:               user.ID,
				RefreshToken:         hashedToken, // Already hashed by service layer
				RefreshTokenSelector: "test-selector-" + uniqueID(),
				Salt:                 salt, // Generated by service layer
				DeviceID:             "device-123",
				IPAddress:            "192.168.1.100",
				UserAgent:            "Mozilla/5.0...",
				ExpiresAt:            time.Now().Add(24 * time.Hour),
			}

			err = repo.CreateSession(ctx, session)
			require.NoError(t, err)

			// Verify session was created
			var savedSession domain.Session

			err = db.Where("id = ?", session.ID).First(&savedSession).Error
			require.NoError(t, err)
			assert.Equal(t, session.UserID, savedSession.UserID)
			assert.Equal(t, session.RefreshToken, savedSession.RefreshToken)
			assert.NotEqual(t, plainToken, savedSession.RefreshToken)
			assert.NotEmpty(t, savedSession.Salt)
			assert.Len(t, savedSession.RefreshToken, 64) // SHA-256 hash as hex = 64 chars
			assert.Len(t, savedSession.Salt, 64)         // 32-byte salt as hex = 64 chars
		})
	})
}

func TestPostgresRepository_GetAllActiveSessions(t *testing.T) {
	ctx := context.Background()

	withTestDB(t, func(db *gorm.DB, redisClient *internalRedis.Client) {
		repo := repository.NewPostgresRepository(db)

		t.Run("get all active sessions", func(t *testing.T) {
			// Create users
			user1 := &domain.User{
				ID:          uniqueID(),
				Email:       uniqueEmail("user1"),
				DisplayName: "User 1",
				Provider:    "google",
				ExternalID:  uniqueExternalID("google-sub-1"),
			}
			user2 := &domain.User{
				ID:          uniqueID(),
				Email:       uniqueEmail("user2"),
				DisplayName: "User 2",
				Provider:    "google",
				ExternalID:  uniqueExternalID("google-sub-2"),
			}
			err := repo.CreateUser(ctx, user1)
			require.NoError(t, err)
			err = repo.CreateUser(ctx, user2)
			require.NoError(t, err)

			// Create sessions
			session1 := &domain.Session{
				ID:           uniqueID(),
				UserID:       user1.ID,
				RefreshToken: "hash1",
				Salt:         "salt1",
				DeviceID:     "device-123",
				IPAddress:    "192.168.1.1",
				UserAgent:    "test-agent",
				ExpiresAt:    time.Now().Add(time.Hour),
			}
			session2 := &domain.Session{
				ID:           uniqueID(),
				UserID:       user2.ID,
				RefreshToken: "hash2",
				Salt:         "salt2",
				DeviceID:     "device-456",
				IPAddress:    "192.168.1.2",
				UserAgent:    "test-agent",
				ExpiresAt:    time.Now().Add(2 * time.Hour),
			}
			err = repo.CreateSession(ctx, session1)
			require.NoError(t, err)
			err = repo.CreateSession(ctx, session2)
			require.NoError(t, err)

			// Get all active sessions
			sessions, err := repo.GetAllActiveSessions(ctx)
			require.NoError(t, err)
			// At least the 2 sessions we created should be in the results
			assert.GreaterOrEqual(t, len(sessions), 2)

			// Verify our sessions are in the results
			sessionIDs := make(map[string]bool)
			for _, s := range sessions {
				sessionIDs[s.ID] = true
			}

			assert.True(t, sessionIDs[session1.ID], "session1 should be in results")
			assert.True(t, sessionIDs[session2.ID], "session2 should be in results")
		})
	})
}

func TestPostgresRepository_GetSession(t *testing.T) {
	ctx := context.Background()

	withTestDB(t, func(db *gorm.DB, redisClient *internalRedis.Client) {
		repo := repository.NewPostgresRepository(db)

		t.Run("get session by ID", func(t *testing.T) {
			// Create a user and session
			user := &domain.User{
				ID:          uniqueID(),
				Email:       uniqueEmail("getsession"),
				DisplayName: "Get Session Test User",
				Provider:    "google",
				ExternalID:  uniqueExternalID("google-sub-getsession"),
			}
			err := repo.CreateUser(ctx, user)
			require.NoError(t, err)

			session := &domain.Session{
				ID:           uniqueID(),
				UserID:       user.ID,
				RefreshToken: "refresh-token-123",
				DeviceID:     "device-123",
				IPAddress:    "192.168.1.100",
				UserAgent:    "Mozilla/5.0...",
				ExpiresAt:    time.Now().Add(24 * time.Hour),
			}
			err = repo.CreateSession(ctx, session)
			require.NoError(t, err)

			// Get the session
			foundSession, err := repo.GetSession(ctx, session.ID)
			require.NoError(t, err)
			assert.NotNil(t, foundSession)
			assert.Equal(t, session.ID, foundSession.ID)
			assert.Equal(t, session.UserID, foundSession.UserID)
		})
	})
}

func TestPostgresRepository_StoreAuthState(t *testing.T) {
	ctx := context.Background()

	withTestDB(t, func(db *gorm.DB, redisClient *internalRedis.Client) {
		repo := repository.NewPostgresRepository(db)

		t.Run("successfully stores auth state with code challenge", func(t *testing.T) {
			authState := &domain.AuthState{
				State:         "test-state-" + uniqueID(),
				Provider:      "google",
				RedirectURL:   "https://example.com/callback",
				CodeChallenge: "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM", // RFC 7636 compliant challenge
				ClientIP:      "192.168.1.1",
				UserAgent:     "Mozilla/5.0",
				ExpiresAt:     time.Now().Add(10 * time.Minute),
				CreatedAt:     time.Now(),
			}

			err := repo.StoreAuthState(ctx, authState)
			require.NoError(t, err)

			// Verify auth state was stored
			var savedState domain.AuthState

			err = db.Where("state = ?", authState.State).First(&savedState).Error
			require.NoError(t, err)
			assert.Equal(t, authState.State, savedState.State)
			assert.Equal(t, authState.Provider, savedState.Provider)
			assert.Equal(t, authState.CodeChallenge, savedState.CodeChallenge)
		})

		t.Run("successfully stores auth state without PKCE", func(t *testing.T) {
			authState := &domain.AuthState{
				State:         "test-state-" + uniqueID(),
				Provider:      "github",
				RedirectURL:   "https://example.com/callback",
				CodeChallenge: "", // No PKCE
				ClientIP:      "192.168.1.2",
				UserAgent:     "Chrome/91.0",
				ExpiresAt:     time.Now().Add(10 * time.Minute),
				CreatedAt:     time.Now(),
			}

			err := repo.StoreAuthState(ctx, authState)
			require.NoError(t, err)

			// Verify auth state was stored
			var savedState domain.AuthState

			err = db.Where("state = ?", authState.State).First(&savedState).Error
			require.NoError(t, err)
			assert.Equal(t, authState.State, savedState.State)
			assert.Empty(t, savedState.CodeChallenge)
		})
	})
}

func TestPostgresRepository_GetAuthState(t *testing.T) {
	ctx := context.Background()

	withTestDB(t, func(db *gorm.DB, redisClient *internalRedis.Client) {
		repo := repository.NewPostgresRepository(db)

		t.Run("successfully retrieves auth state with code challenge", func(t *testing.T) {
			// Store auth state first
			authState := &domain.AuthState{
				State:         "test-state-" + uniqueID(),
				Provider:      "google",
				RedirectURL:   "https://example.com/callback",
				CodeChallenge: "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM",
				ClientIP:      "192.168.1.1",
				UserAgent:     "Mozilla/5.0",
				ExpiresAt:     time.Now().Add(5 * time.Minute),
				CreatedAt:     time.Now(),
			}
			err := repo.StoreAuthState(ctx, authState)
			require.NoError(t, err)

			// Get auth state
			foundState, err := repo.GetAuthState(ctx, authState.State)
			require.NoError(t, err)
			assert.NotNil(t, foundState)
			assert.Equal(t, authState.State, foundState.State)
			assert.Equal(t, authState.CodeChallenge, foundState.CodeChallenge)
		})

		t.Run("returns error for expired auth state", func(t *testing.T) {
			// Store expired auth state
			authState := &domain.AuthState{
				State:       "expired-state-" + uniqueID(),
				Provider:    "google",
				RedirectURL: "https://example.com/callback",
				ExpiresAt:   time.Now().Add(-5 * time.Minute), // Already expired
				CreatedAt:   time.Now().Add(-10 * time.Minute),
			}
			err := repo.StoreAuthState(ctx, authState)
			require.NoError(t, err)

			// Try to get expired state
			foundState, err := repo.GetAuthState(ctx, authState.State)
			require.Error(t, err)
			assert.Nil(t, foundState)
			assert.Contains(t, err.Error(), "auth state not found")
		})

		t.Run("handles auth state without PKCE", func(t *testing.T) {
			// Store auth state without PKCE
			authState := &domain.AuthState{
				State:         "no-pkce-state-" + uniqueID(),
				Provider:      "github",
				RedirectURL:   "https://example.com/callback",
				CodeChallenge: "", // Empty code challenge
				ClientIP:      "192.168.1.2",
				UserAgent:     "Chrome/91.0",
				ExpiresAt:     time.Now().Add(5 * time.Minute),
				CreatedAt:     time.Now(),
			}
			err := repo.StoreAuthState(ctx, authState)
			require.NoError(t, err)

			// Get auth state
			foundState, err := repo.GetAuthState(ctx, authState.State)
			require.NoError(t, err)
			assert.NotNil(t, foundState)
			assert.Equal(t, authState.State, foundState.State)
			assert.Empty(t, foundState.CodeChallenge)
		})
	})
}

func TestPostgresRepository_RefreshTokenBlacklist(t *testing.T) {
	ctx := context.Background()

	withTestDB(t, func(db *gorm.DB, redisClient *internalRedis.Client) {
		repo := repository.NewPostgresRepository(db)

		t.Run("blacklist and check token", func(t *testing.T) {
			token := "test-refresh-token-" + uniqueID()
			expiresAt := time.Now().Add(1 * time.Hour)

			// Blacklist the token
			err := repo.BlacklistRefreshToken(ctx, token, expiresAt)
			require.NoError(t, err)

			// Check if token is blacklisted
			isBlacklisted, err := repo.IsRefreshTokenBlacklisted(ctx, token)
			require.NoError(t, err)
			assert.True(t, isBlacklisted)
		})

		t.Run("token not in blacklist", func(t *testing.T) {
			token := "non-blacklisted-token-" + uniqueID()

			isBlacklisted, err := repo.IsRefreshTokenBlacklisted(ctx, token)
			require.NoError(t, err)
			assert.False(t, isBlacklisted)
		})

		t.Run("expired token in blacklist", func(t *testing.T) {
			token := "expired-token-" + uniqueID()
			expiresAt := time.Now().Add(-1 * time.Hour) // Already expired

			// Blacklist the expired token
			err := repo.BlacklistRefreshToken(ctx, token, expiresAt)
			require.NoError(t, err)

			// Check if token is blacklisted - should return false since it's expired
			isBlacklisted, err := repo.IsRefreshTokenBlacklisted(ctx, token)
			require.NoError(t, err)
			assert.False(t, isBlacklisted)
		})
	})
}

func TestPostgresRepository_GetSessionByRefreshTokenSelector(t *testing.T) {
	ctx := context.Background()

	withTestDB(t, func(db *gorm.DB, redisClient *internalRedis.Client) {
		repo := repository.NewPostgresRepository(db)

		t.Run("find session by valid selector", func(t *testing.T) {
			// Create a user first
			user := &domain.User{
				ID:          uniqueID(),
				Email:       uniqueEmail("selector-test"),
				DisplayName: "Selector Test User",
				Provider:    "google",
				ExternalID:  uniqueExternalID("google-sub-selector"),
			}
			err := repo.CreateUser(ctx, user)
			require.NoError(t, err)

			// Create session with selector
			selector := "selector-" + uniqueID()
			session := &domain.Session{
				ID:                   uniqueID(),
				UserID:               user.ID,
				RefreshToken:         "hashed-verifier-789",
				RefreshTokenSelector: selector,
				Salt:                 "salt-def456",
				DeviceID:             "device-xyz",
				IPAddress:            "192.168.1.1",
				UserAgent:            "Mozilla/5.0",
				ExpiresAt:            time.Now().Add(24 * time.Hour),
			}
			err = repo.CreateSession(ctx, session)
			require.NoError(t, err)

			// Find session by selector
			foundSession, err := repo.GetSessionByRefreshTokenSelector(ctx, selector)
			require.NoError(t, err)
			assert.NotNil(t, foundSession)
			assert.Equal(t, selector, foundSession.RefreshTokenSelector)
			assert.Equal(t, user.ID, foundSession.UserID)
		})

		t.Run("return error for nonexistent selector", func(t *testing.T) {
			selector := "nonexistent-selector"

			foundSession, err := repo.GetSessionByRefreshTokenSelector(ctx, selector)
			require.Error(t, err)
			assert.Nil(t, foundSession)
			assert.Contains(t, err.Error(), "session not found")
		})
	})
}
