package service_test

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
	"github.com/hexabase/hexabase-ai/api/internal/auth/service"
	internalRedis "github.com/hexabase/hexabase-ai/api/internal/shared/redis"
)

const (
	testUserID = "user-123"
)

//nolint:paralleltest // Transaction-based test
func TestSessionManager_CreateSession_UnderLimit(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	ctx := context.Background()
	userID := testUserID
	sessionID := "session-new"

	withTestDB(t, func(db *gorm.DB, redisClient *internalRedis.Client) {
		_, _, _ = setupTestServiceWithDB(t, db, redisClient)
		// We get the sessionManager from the service, but we need to cast it to the concrete type
		// to access its methods directly if they are not on the interface.
		// For this test, we assume the service embeds or provides access to the session manager.
		// A better approach would be to have sessionManager returned from setup, but for now we work with what we have.
		// Let's assume we can get it from the service structure or we create it here for the test.
		repo := repository.NewCompositeRepository(repository.NewPostgresRepository(db), repository.NewRedisAuthRepository(redisClient), repository.NewTokenHashRepository())
		sessionLimiterRepo := repository.NewSessionLimiterRepository(redisClient)
		sessionManager := service.NewSessionManager(repo, sessionLimiterRepo)

		// Precondition: 2 sessions already exist
		err := sessionManager.CreateSession(ctx, userID, "session-1")
		require.NoError(t, err)
		err = sessionManager.CreateSession(ctx, userID, "session-2")
		require.NoError(t, err)

		// Execute
		err = sessionManager.CreateSession(ctx, userID, sessionID)
		require.NoError(t, err)

		// Verify that new session was added
		count, err := sessionManager.GetActiveSessionCount(ctx, userID)
		require.NoError(t, err)
		assert.Equal(t, 3, count)
	})
}

//nolint:paralleltest // Transaction-based test
func TestSessionManager_CreateSession_AtLimit_ReturnsError(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	ctx := context.Background()
	userID := testUserID
	newSessionID := "session-new"

	withTestDB(t, func(db *gorm.DB, redisClient *internalRedis.Client) {
		repo := repository.NewCompositeRepository(repository.NewPostgresRepository(db), repository.NewRedisAuthRepository(redisClient), repository.NewTokenHashRepository())
		sessionLimiterRepo := repository.NewSessionLimiterRepository(redisClient)
		sessionManager := service.NewSessionManager(repo, sessionLimiterRepo)

		// Precondition: 3 sessions already exist (at limit)
		err := sessionManager.CreateSession(ctx, userID, "session-1")
		require.NoError(t, err)
		err = sessionManager.CreateSession(ctx, userID, "session-2")
		require.NoError(t, err)
		err = sessionManager.CreateSession(ctx, userID, "session-3")
		require.NoError(t, err)

		// Execute
		err = sessionManager.CreateSession(ctx, userID, newSessionID)

		// Verify
		require.Error(t, err)
		assert.Equal(t, domain.ErrTooManySessions, err)

		// Check state - should remain unchanged
		count, err := sessionManager.GetActiveSessionCount(ctx, userID)
		require.NoError(t, err)
		assert.Equal(t, 3, count) // Still 3, new session NOT added
	})
}

//nolint:paralleltest // Transaction-based test
func TestSessionManager_DeleteSession(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	ctx := context.Background()
	userID := testUserID
	sessionID := "session-to-remove"

	withTestDB(t, func(db *gorm.DB, redisClient *internalRedis.Client) {
		repo := repository.NewCompositeRepository(repository.NewPostgresRepository(db), repository.NewRedisAuthRepository(redisClient), repository.NewTokenHashRepository())
		sessionLimiterRepo := repository.NewSessionLimiterRepository(redisClient)
		sessionManager := service.NewSessionManager(repo, sessionLimiterRepo)

		// Precondition: Sessions exist
		err := sessionManager.CreateSession(ctx, userID, sessionID)
		require.NoError(t, err)
		err = sessionManager.CreateSession(ctx, userID, "session-2")
		require.NoError(t, err)

		// Execute
		err = sessionManager.DeleteSession(ctx, userID, sessionID)
		require.NoError(t, err)

		// Verify removal
		count, err := sessionManager.GetActiveSessionCount(ctx, userID)
		require.NoError(t, err)
		assert.Equal(t, 1, count) // One session remaining
	})
}

//nolint:paralleltest // Transaction-based test
func TestSessionManager_GetActiveSessionCount(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	ctx := context.Background()
	userID := testUserID

	withTestDB(t, func(db *gorm.DB, redisClient *internalRedis.Client) {
		repo := repository.NewCompositeRepository(repository.NewPostgresRepository(db), repository.NewRedisAuthRepository(redisClient), repository.NewTokenHashRepository())
		sessionLimiterRepo := repository.NewSessionLimiterRepository(redisClient)
		sessionManager := service.NewSessionManager(repo, sessionLimiterRepo)

		// Precondition: 2 sessions exist
		err := sessionManager.CreateSession(ctx, userID, "session-1")
		require.NoError(t, err)
		err = sessionManager.CreateSession(ctx, userID, "session-2")
		require.NoError(t, err)

		// Execute
		count, err := sessionManager.GetActiveSessionCount(ctx, userID)

		// Verify
		require.NoError(t, err)
		assert.Equal(t, 2, count)
	})
}

//nolint:paralleltest,funlen // Transaction-based test with comprehensive integration scenarios
func TestSessionManager_CompleteSessionLifecycle(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	ctx := context.Background()

	withTestDB(t, func(db *gorm.DB, redisClient *internalRedis.Client) {
		// Test complete session lifecycle with PostgreSQL + Redis
		_, oauthRepo, _ := setupTestServiceWithDB(t, db, redisClient)
		stubRepo := oauthRepo.(*stubOAuthRepository)
		user := stubRepo.userInfo

		repo := repository.NewCompositeRepository(repository.NewPostgresRepository(db), repository.NewRedisAuthRepository(redisClient), repository.NewTokenHashRepository())
		sessionLimiterRepo := repository.NewSessionLimiterRepository(redisClient)
		sessionManager := service.NewSessionManager(repo, sessionLimiterRepo)

		userID := user.ID

		// Create a user first in PostgreSQL
		dbUser := &domain.User{
			ID:          userID,
			Email:       user.Email,
			ExternalID:  user.ID,
			Provider:    user.Provider,
			DisplayName: user.Name,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
			LastLoginAt: time.Now(),
		}
		err := repo.CreateUser(ctx, dbUser)
		require.NoError(t, err)

		// Create session in PostgreSQL
		session := &domain.Session{
			ID:        fmt.Sprintf("session-lifecycle-%d", time.Now().UnixNano()),
			UserID:    userID,
			IPAddress: "192.168.1.1",
			UserAgent: "Test Agent",
			ExpiresAt: time.Now().Add(1 * time.Hour),
			Salt:      "testsalt",
		}
		err = repo.CreateSession(ctx, session)
		require.NoError(t, err)

		// Test SessionManager with Redis tracking
		err = sessionManager.CreateSession(ctx, userID, session.ID)
		require.NoError(t, err)

		// Verify Redis tracking
		count, err := sessionManager.GetActiveSessionCount(ctx, userID)
		require.NoError(t, err)
		assert.Equal(t, 1, count)

		// Verify PostgreSQL data persistence
		retrievedSession, err := repo.GetSession(ctx, session.ID)
		require.NoError(t, err)
		assert.Equal(t, session.ID, retrievedSession.ID)
		assert.Equal(t, userID, retrievedSession.UserID)

		// Test deletion with both systems
		err = sessionManager.DeleteSession(ctx, userID, session.ID)
		require.NoError(t, err)

		// Verify Redis cleanup
		count, err = sessionManager.GetActiveSessionCount(ctx, userID)
		require.NoError(t, err)
		assert.Equal(t, 0, count)
	})
}

//nolint:paralleltest,funlen // Transaction-based test with comprehensive integration scenarios
func TestSessionManager_SessionLimitEnforcement(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	ctx := context.Background()

	withTestDB(t, func(db *gorm.DB, redisClient *internalRedis.Client) {
		_, oauthRepo, _ := setupTestServiceWithDB(t, db, redisClient)
		stubRepo := oauthRepo.(*stubOAuthRepository)
		user := stubRepo.userInfo

		repo := repository.NewCompositeRepository(repository.NewPostgresRepository(db), repository.NewRedisAuthRepository(redisClient), repository.NewTokenHashRepository())
		sessionLimiterRepo := repository.NewSessionLimiterRepository(redisClient)
		sessionManager := service.NewSessionManager(repo, sessionLimiterRepo)

		userID := user.ID

		// Create a user for this test
		dbUser := &domain.User{
			ID:          userID,
			Email:       user.Email,
			ExternalID:  user.ID,
			Provider:    user.Provider,
			DisplayName: user.Name,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
			LastLoginAt: time.Now(),
		}
		err := repo.CreateUser(ctx, dbUser)
		require.NoError(t, err)

		// Create 3 sessions to hit the limit
		for i := 0; i < 3; i++ {
			sessionID := fmt.Sprintf("session-limit-%d-%s", i, uuid.New().String())
			session := &domain.Session{
				ID:        sessionID,
				UserID:    userID,
				IPAddress: "192.168.1.1",
				UserAgent: "Test Agent",
				ExpiresAt: time.Now().Add(1 * time.Hour),
				Salt:      fmt.Sprintf("testsalt-%d", i),
			}
			err := repo.CreateSession(ctx, session)
			require.NoError(t, err)
			err = sessionManager.CreateSession(ctx, userID, sessionID)
			require.NoError(t, err)
		}

		// Try to create a 4th session
		err = sessionManager.CreateSession(ctx, userID, fmt.Sprintf("session-limit-4-%s", uuid.New().String()))

		// Verify it fails with the correct error
		require.Error(t, err)
		assert.Equal(t, domain.ErrTooManySessions, err)

		// Verify Redis count is still 3
		count, err := sessionManager.GetActiveSessionCount(ctx, userID)
		require.NoError(t, err)
		assert.Equal(t, 3, count)
	})
}
