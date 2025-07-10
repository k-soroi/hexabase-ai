package repository_test

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/hexabase/hexabase-ai/api/internal/auth/domain"
	"github.com/hexabase/hexabase-ai/api/internal/auth/repository"
	"github.com/hexabase/hexabase-ai/api/internal/shared/config"
	internalRedis "github.com/hexabase/hexabase-ai/api/internal/shared/redis"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testUserID    = "user-123"
	testSessionID = "session-456"
)

// testContext returns a context for testing that's cancelled when the test ends
func testContext(t *testing.T) context.Context {
	t.Helper()

	ctx, cancel := context.WithCancel(t.Context())
	t.Cleanup(cancel)

	return ctx
}

func setupRedisTest(t *testing.T) (*miniredis.Miniredis, domain.SessionLimiterRepository) { //nolint:ireturn
	t.Helper()
	// Create miniredis instance
	mr, err := miniredis.Run()
	require.NoError(t, err)

	// Create wrapped Redis client for internal usage
	logger := slog.Default()
	wrappedClient, err := internalRedis.NewClient(&config.RedisConfig{
		Host: mr.Host(),
		Port: mr.Port(),
		DB:   0,
	}, logger)
	require.NoError(t, err)

	// Create repository using the public constructor
	limiter := repository.NewSessionLimiterRepository(wrappedClient)

	// Cleanup
	t.Cleanup(func() {
		_ = wrappedClient.Close()
		mr.Close()
	})

	return mr, limiter
}

func TestSessionLimiter_AddSession_UnderLimit(t *testing.T) {
	t.Parallel()
	// Test: When creating a new session, it succeeds if the user's concurrent session count is less than 3

	ctx := testContext(t)
	userID := testUserID
	sessionID := testSessionID
	ttl := 30 * time.Minute

	// Setup
	mr, limiter := setupRedisTest(t)

	// Precondition: 2 sessions already exist
	_, err2 := mr.SAdd("user_sessions:"+testUserID, "session-111", "session-222")
	require.NoError(t, err2)

	// Execute
	err := limiter.AddSessionToUser(ctx, userID, sessionID, ttl)

	// Verify
	require.NoError(t, err)

	// Verify that the session was added
	members, err := mr.SMembers("user_sessions:" + testUserID)
	require.NoError(t, err)
	assert.Len(t, members, 3)
	assert.Contains(t, members, sessionID)

	// Verify that TTL was set
	ttlDuration := mr.TTL("user_sessions:" + testUserID)
	assert.Greater(t, ttlDuration, time.Duration(0))
	assert.LessOrEqual(t, ttlDuration, ttl)
}

func TestSessionLimiter_GetUserSessionCount(t *testing.T) {
	t.Parallel()
	// Test: Can correctly get the user's active session count

	ctx := testContext(t)
	userID := testUserID

	// Setup
	mr, limiter := setupRedisTest(t)

	// Precondition: Set up 2 sessions
	_, err2 := mr.SAdd("user_sessions:"+testUserID, "session-111", "session-222")
	require.NoError(t, err2)

	// Execute
	count, err := limiter.GetUserSessionCount(ctx, userID)

	// Verify
	require.NoError(t, err)
	assert.Equal(t, int64(2), count)
}

func TestSessionLimiter_RemoveSession(t *testing.T) {
	t.Parallel()
	// Test: When removing a session, the session count decreases correctly

	ctx := testContext(t)
	userID := testUserID
	sessionID := testSessionID

	// Setup
	mr, limiter := setupRedisTest(t)

	// Precondition: Set up 3 sessions
	_, err2 := mr.SAdd("user_sessions:"+testUserID, "session-111", "session-222", sessionID)
	require.NoError(t, err2)

	// Execute
	err := limiter.RemoveSessionFromUser(ctx, userID, sessionID)

	// Verify
	require.NoError(t, err)

	// Verify that the session was removed
	members, err := mr.SMembers("user_sessions:" + testUserID)
	require.NoError(t, err)
	assert.Len(t, members, 2)
	assert.NotContains(t, members, sessionID)
}

func TestSessionLimiter_GetUserSessionIDs(t *testing.T) {
	t.Parallel()
	// Test: Can get all active session IDs for a user

	ctx := testContext(t)
	userID := testUserID
	expectedSessions := []string{"session-1", "session-2", "session-3"}

	// Setup
	mr, limiter := setupRedisTest(t)

	// Precondition: Set up sessions
	for _, session := range expectedSessions {
		_, err := mr.SAdd("user_sessions:"+testUserID, session)
		require.NoError(t, err)
	}

	// Execute
	sessions, err := limiter.GetUserSessionIDs(ctx, userID)

	// Verify
	require.NoError(t, err)
	assert.ElementsMatch(t, expectedSessions, sessions)
}

func TestSessionLimiter_IsSessionInUserSet(t *testing.T) {
	t.Parallel()
	// Test: Can check if a specified session ID exists in the user's set

	ctx := testContext(t)
	userID := testUserID
	sessionID := testSessionID

	// Setup
	mr, limiter := setupRedisTest(t)

	// Precondition: Set up sessions
	_, err2 := mr.SAdd("user_sessions:"+testUserID, sessionID, "session-111")
	require.NoError(t, err2)

	// Execute
	exists, err := limiter.IsSessionInUserSet(ctx, userID, sessionID)

	// Verify
	require.NoError(t, err)
	assert.True(t, exists)

	// Check for non-existent session
	exists, err = limiter.IsSessionInUserSet(ctx, userID, "non-existent")
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestSessionLimiter_AcquireReleaseLock(t *testing.T) {
	t.Parallel()
	// Test: Distributed lock with compare-and-delete functionality

	ctx := testContext(t)
	lockKey := "test:lock:resource"
	lockValue1 := "owner-1"
	lockValue2 := "owner-2"

	// Setup
	_, limiter := setupRedisTest(t)

	// Test 1: Acquire lock successfully
	acquired, err := limiter.AcquireLock(ctx, lockKey, lockValue1, 5*time.Second)
	require.NoError(t, err)
	assert.True(t, acquired)

	// Test 2: Cannot acquire lock with different value
	acquired, err = limiter.AcquireLock(ctx, lockKey, lockValue2, 5*time.Second)
	require.NoError(t, err)
	assert.False(t, acquired)

	// Test 3: Release lock with wrong value fails
	err = limiter.ReleaseLock(ctx, lockKey, lockValue2)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "lock not held or value mismatch")

	// Test 4: Release lock with correct value succeeds
	err = limiter.ReleaseLock(ctx, lockKey, lockValue1)
	require.NoError(t, err)

	// Test 5: Can acquire lock after release
	acquired, err = limiter.AcquireLock(ctx, lockKey, lockValue2, 5*time.Second)
	require.NoError(t, err)
	assert.True(t, acquired)

	// Clean up
	err = limiter.ReleaseLock(ctx, lockKey, lockValue2)
	require.NoError(t, err)
}
