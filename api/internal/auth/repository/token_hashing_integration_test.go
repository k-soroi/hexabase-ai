//nolint:paralleltest
package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/hexabase/hexabase-ai/api/internal/auth/domain"
	"github.com/hexabase/hexabase-ai/api/internal/auth/repository"
	internalRedis "github.com/hexabase/hexabase-ai/api/internal/shared/redis"
)

// Helper function to find session by refresh token
// This simulates what the service layer would do
func findSessionByRefreshToken(
	ctx context.Context,
	repo domain.Repository,
	refreshToken string,
) (*domain.Session, error) {
	// Get all active sessions from repository
	sessions, err := repo.GetAllActiveSessions(ctx)
	if err != nil {
		return nil, err
	}

	// Check each session using crypto operations
	for _, session := range sessions {
		if session.Salt != "" && repo.VerifyToken(refreshToken, session.RefreshToken, session.Salt) {
			return session, nil
		}
	}

	return nil, gorm.ErrRecordNotFound
}

// TestHashedRefreshTokenIntegration validates the complete authentication flow with hashed refresh tokens
func TestHashedRefreshTokenIntegration(t *testing.T) { //nolint:funlen
	ctx := context.Background()

	t.Run("End-to-end hashed refresh token flow", func(t *testing.T) {
		withTestDB(t, func(db *gorm.DB, redisClient *internalRedis.Client) { //nolint:contextcheck
			// Setup repository
			dbRepo := repository.NewPostgresRepository(db)
			cacheRepo := repository.NewRedisAuthRepository(redisClient)
			tokenHashRepo := repository.NewTokenHashRepository()
			repo := repository.NewCompositeRepository(dbRepo, cacheRepo, tokenHashRepo)

			// Step 1: Create a user
			user := &domain.User{
				ID:          uniqueID(),
				ExternalID:  uniqueExternalID("ext-user"),
				Provider:    "github",
				Email:       uniqueEmail("test"),
				DisplayName: "Test User",
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			}
			err := repo.CreateUser(ctx, user)
			require.NoError(t, err)

			// Step 2: Create a session with a plain refresh token
			plainRefreshToken := "original-plain-refresh-token-12345"
			session := &domain.Session{
				ID:           uniqueID(),
				UserID:       user.ID,
				RefreshToken: plainRefreshToken, // This will be hashed by repository
				DeviceID:     "device-123",
				IPAddress:    "192.168.1.100",
				UserAgent:    "Test-Agent/1.0",
				ExpiresAt:    time.Now().Add(24 * time.Hour),
				CreatedAt:    time.Now(),
				LastUsedAt:   time.Now(),
				Revoked:      false,
			}

			// Step 3: Hash the token before saving (simulating service layer business logic)
			hashedToken, salt, err := tokenHashRepo.HashToken(plainRefreshToken)
			require.NoError(t, err)

			session.RefreshToken = hashedToken
			session.Salt = salt

			// Save session with already-hashed token
			err = repo.CreateSession(ctx, session)
			require.NoError(t, err)

			// Verify token was hashed and salt was generated
			assert.NotEqual(t, plainRefreshToken, session.RefreshToken)
			assert.NotEmpty(t, session.Salt)
			assert.Len(t, session.RefreshToken, 64) // SHA-256 hash = 64 hex chars
			assert.Len(t, session.Salt, 64)         // 32-byte salt = 64 hex chars

			// Step 4: Verify we can retrieve the session using the original plain token
			retrievedSession, err := findSessionByRefreshToken(ctx, repo, plainRefreshToken)
			require.NoError(t, err)
			assert.Equal(t, session.ID, retrievedSession.ID)
			assert.Equal(t, session.UserID, retrievedSession.UserID)
			assert.Equal(t, session.RefreshToken, retrievedSession.RefreshToken) // Hashed value
			assert.Equal(t, session.Salt, retrievedSession.Salt)

			// Step 5: Verify wrong token doesn't match
			wrongToken := "wrong-refresh-token"
			_, err = findSessionByRefreshToken(ctx, repo, wrongToken)
			require.Error(t, err)

			// Step 6: Verify the hash can be verified directly using crypto utilities
			isValid := tokenHashRepo.VerifyToken(plainRefreshToken, session.RefreshToken, session.Salt)
			assert.True(t, isValid)

			// Step 7: Verify wrong token fails crypto verification
			isValid = tokenHashRepo.VerifyToken(wrongToken, session.RefreshToken, session.Salt)
			assert.False(t, isValid)

			// Step 8: Test that revoked sessions are not found
			session.Revoked = true
			err = db.Save(session).Error
			require.NoError(t, err)

			_, err = findSessionByRefreshToken(ctx, repo, plainRefreshToken)
			require.Error(t, err)
		})
	})

	t.Run("Security validation - no plain text tokens in database", func(t *testing.T) {
		withTestDB(t, func(db *gorm.DB, redisClient *internalRedis.Client) { //nolint:contextcheck
			// Setup repository
			dbRepo := repository.NewPostgresRepository(db)
			cacheRepo := repository.NewRedisAuthRepository(redisClient)
			tokenHashRepo := repository.NewTokenHashRepository()
			repo := repository.NewCompositeRepository(dbRepo, cacheRepo, tokenHashRepo)

			// Create multiple sessions with different tokens
			testTokens := []string{
				"token-1-very-secret",
				"token-2-super-secret",
				"token-3-ultra-secret",
			}

			for i, token := range testTokens {
				user := &domain.User{
					ID:          uniqueID(),
					ExternalID:  uniqueExternalID("ext-user"),
					Provider:    "github",
					Email:       uniqueEmail("test"),
					DisplayName: "Test User " + uniqueID(),
					CreatedAt:   time.Now(),
					UpdatedAt:   time.Now(),
				}
				err := repo.CreateUser(ctx, user)
				require.NoError(t, err)

				// Hash token before creating session (simulating service layer)
				hashedToken, salt, err := tokenHashRepo.HashToken(token)
				require.NoError(t, err)

				session := &domain.Session{
					ID:           uniqueID(),
					UserID:       user.ID,
					RefreshToken: hashedToken,
					Salt:         salt,
					DeviceID:     "device-" + uniqueID(),
					IPAddress:    "192.168.1.100",
					UserAgent:    "Test-Agent/1.0",
					ExpiresAt:    time.Now().Add(24 * time.Hour),
					CreatedAt:    time.Now(),
					LastUsedAt:   time.Now(),
					Revoked:      false,
				}

				err = repo.CreateSession(ctx, session)
				require.NoError(t, err)

				// Verify token was hashed
				assert.NotEqual(t, token, session.RefreshToken, "Token %d should be hashed", i)
				assert.NotEmpty(t, session.Salt, "Token %d should have salt", i)
			}

			// Query database directly to ensure no plain text tokens exist
			var sessions []domain.Session

			err := db.Find(&sessions).Error
			require.NoError(t, err)

			for i, session := range sessions {
				for j, originalToken := range testTokens {
					assert.NotEqual(t, originalToken, session.RefreshToken,
						"Session %d should not contain plain text token %d", i, j)
				}

				// Verify all sessions have proper salt
				assert.NotEmpty(t, session.Salt, "Session %d should have salt", i)
				assert.Len(t, session.Salt, 64, "Session %d salt should be 64 hex chars", i)
			}

			// Verify each original token can still retrieve its corresponding session
			for _, token := range testTokens {
				retrievedSession, err := findSessionByRefreshToken(ctx, repo, token)
				require.NoError(t, err)
				assert.NotEmpty(t, retrievedSession.ID)
			}
		})
	})

	t.Run("Performance test - hash verification efficiency", func(t *testing.T) {
		withTestDB(t, func(db *gorm.DB, redisClient *internalRedis.Client) { //nolint:contextcheck
			// Setup repository
			dbRepo := repository.NewPostgresRepository(db)
			cacheRepo := repository.NewRedisAuthRepository(redisClient)
			tokenHashRepo := repository.NewTokenHashRepository()
			repo := repository.NewCompositeRepository(dbRepo, cacheRepo, tokenHashRepo)

			// Create a session with hashed token
			plainToken := "performance-test-token-123"
			user := &domain.User{
				ID:          uniqueID(),
				ExternalID:  uniqueExternalID("perf-user"),
				Provider:    "github",
				Email:       uniqueEmail("perf"),
				DisplayName: "Performance User",
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			}
			err := repo.CreateUser(ctx, user)
			require.NoError(t, err)

			// Hash token before creating session (simulating service layer)
			hashedToken, salt, err := tokenHashRepo.HashToken(plainToken)
			require.NoError(t, err)

			session := &domain.Session{
				ID:           uniqueID(),
				UserID:       user.ID,
				RefreshToken: hashedToken,
				Salt:         salt,
				DeviceID:     "perf-device",
				IPAddress:    "192.168.1.200",
				UserAgent:    "Perf-Agent/1.0",
				ExpiresAt:    time.Now().Add(24 * time.Hour),
				CreatedAt:    time.Now(),
				LastUsedAt:   time.Now(),
				Revoked:      false,
			}

			err = repo.CreateSession(ctx, session)
			require.NoError(t, err)

			// Measure hash verification performance
			start := time.Now()

			const iterations = 1000

			for i := 0; i < iterations; i++ {
				isValid := tokenHashRepo.VerifyToken(plainToken, session.RefreshToken, session.Salt)
				assert.True(t, isValid)
			}

			duration := time.Since(start)
			avgDuration := duration / iterations

			// Ensure hash verification is fast (should be well under 1ms per operation)
			assert.Less(t, avgDuration, 1*time.Millisecond,
				"Hash verification should be fast, got %v per operation", avgDuration)

			t.Logf("Hash verification performance: %v per operation (%d iterations in %v)",
				avgDuration, iterations, duration)
		})
	})
}
