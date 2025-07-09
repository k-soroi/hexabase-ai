package auth_test

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hexabase/hexabase-ai/api/internal/auth/domain"
	"github.com/hexabase/hexabase-ai/api/internal/auth/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// mockTokenHashRepository implements repository-level token hashing for testing
type mockTokenHashRepository struct{}

// testService implements minimal Service interface for integration tests
type testService struct {
	repo domain.Repository
}

func (s *testService) getSessionByRefreshToken(ctx context.Context, refreshToken string) (*domain.Session, error) {
	// Get all active sessions from repository (pure data operation)
	sessions, err := s.repo.GetAllActiveSessions(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get active sessions: %w", err)
	}

	// Check each session using crypto operations
	for _, session := range sessions {
		if session.Salt != "" && s.repo.VerifyToken(refreshToken, session.RefreshToken, session.Salt) {
			return session, nil
		}
	}

	return nil, fmt.Errorf("session not found")
}

func (m *mockTokenHashRepository) HashToken(token string) (hashedToken string, salt string, err error) {

	// Generate cryptographically secure 32-byte salt
	saltBytes := make([]byte, 32)
	if _, err := rand.Read(saltBytes); err != nil {
		return "", "", fmt.Errorf("failed to generate salt: %w", err)
	}

	// Create hash: SHA256(token + salt)
	hasher := sha256.New()
	hasher.Write([]byte(token))
	hasher.Write(saltBytes)
	hashBytes := hasher.Sum(nil)

	return hex.EncodeToString(hashBytes), hex.EncodeToString(saltBytes), nil
}

func (m *mockTokenHashRepository) VerifyToken(plainToken, hashedToken, salt string) bool {
	// Decode salt from hex
	saltBytes, err := hex.DecodeString(salt)
	if err != nil {
		return false
	}

	// Compute hash using same method as HashToken
	hasher := sha256.New()
	hasher.Write([]byte(plainToken))
	hasher.Write(saltBytes)
	computedHashBytes := hasher.Sum(nil)
	computedHash := hex.EncodeToString(computedHashBytes)

	// Use constant-time comparison to prevent timing attacks
	return subtle.ConstantTimeCompare([]byte(computedHash), []byte(hashedToken)) == 1
}

// テスト用のダミーredisAuthRepository型
// CompositeRepositoryの引数用（何もしない）
type dummyRedisAuthRepository struct{}

// TestHashedRefreshTokenIntegration validates the complete authentication flow with hashed refresh tokens
func TestHashedRefreshTokenIntegration(t *testing.T) {
	// Setup in-memory database for integration testing
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	// Auto-migrate the session model with salt field
	err = db.AutoMigrate(&domain.Session{}, &domain.User{})
	require.NoError(t, err)

	// Setup repository (infrastructure has its own implementation)
	dbRepo := repository.NewPostgresRepository(db)
	cacheRepo := repository.NewRedisAuthRepository(nil) // テスト用にnilで生成
	tokenHashRepo := repository.NewTokenHashRepository()
	repo := repository.NewCompositeRepository(dbRepo, cacheRepo, tokenHashRepo)
	
	// Setup service layer for proper DDD architecture
	// Note: Using nil for other dependencies since we're only testing token hashing functionality
	svc := &testService{repo: repo}
	
	// Setup repository-level token hashing for testing crypto operations
	ctx := context.Background()

	t.Run("End-to-end hashed refresh token flow", func(t *testing.T) {
		// Step 1: Create a user
		user := &domain.User{
			ID:          uuid.New().String(),
			ExternalID:  "ext-user-123",
			Provider:    "github",
			Email:       "test@example.com",
			DisplayName: "Test User",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
		err := repo.CreateUser(ctx, user)
		require.NoError(t, err)

		// Step 2: Create a session with a plain refresh token
		plainRefreshToken := "original-plain-refresh-token-12345"
		session := &domain.Session{
			ID:           uuid.New().String(),
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
		retrievedSession, err := svc.getSessionByRefreshToken(ctx, plainRefreshToken)
		require.NoError(t, err)
		assert.Equal(t, session.ID, retrievedSession.ID)
		assert.Equal(t, session.UserID, retrievedSession.UserID)
		assert.Equal(t, session.RefreshToken, retrievedSession.RefreshToken) // Hashed value
		assert.Equal(t, session.Salt, retrievedSession.Salt)

		// Step 5: Verify wrong token doesn't match
		wrongToken := "wrong-refresh-token"
		_, err = svc.getSessionByRefreshToken(ctx, wrongToken)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "session not found")

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

		_, err = svc.getSessionByRefreshToken(ctx, plainRefreshToken)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "session not found")
	})

	t.Run("Security validation - no plain text tokens in database", func(t *testing.T) {
		// Create multiple sessions with different tokens
		testTokens := []string{
			"token-1-very-secret",
			"token-2-super-secret", 
			"token-3-ultra-secret",
		}

		var sessionIDs []string
		for i, token := range testTokens {
			user := &domain.User{
				ID:          uuid.New().String(),
				ExternalID:  "ext-user-" + uuid.New().String(),
				Provider:    "github",
				Email:       "test" + uuid.New().String() + "@example.com",
				DisplayName: "Test User " + uuid.New().String(),
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			}
			err := repo.CreateUser(ctx, user)
			require.NoError(t, err)

			// Hash token before creating session (simulating service layer)
			hashedToken, salt, err := tokenHashRepo.HashToken(token)
			require.NoError(t, err)

			session := &domain.Session{
				ID:           uuid.New().String(),
				UserID:       user.ID,
				RefreshToken: hashedToken,
				Salt:         salt,
				DeviceID:     "device-" + uuid.New().String(),
				IPAddress:    "192.168.1." + uuid.New().String(),
				UserAgent:    "Test-Agent/1.0",
				ExpiresAt:    time.Now().Add(24 * time.Hour),
				CreatedAt:    time.Now(),
				LastUsedAt:   time.Now(),
				Revoked:      false,
			}

			err = repo.CreateSession(ctx, session)
			require.NoError(t, err)
			sessionIDs = append(sessionIDs, session.ID)

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
			retrievedSession, err := svc.getSessionByRefreshToken(ctx, token)
			require.NoError(t, err)
			assert.NotEmpty(t, retrievedSession.ID)
		}
	})

	t.Run("Performance test - hash verification efficiency", func(t *testing.T) {
		// Create a session with hashed token
		plainToken := "performance-test-token-123"
		user := &domain.User{
			ID:          uuid.New().String(),
			ExternalID:  "perf-user",
			Provider:    "github",
			Email:       "perf@example.com",
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
			ID:           uuid.New().String(),
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
}