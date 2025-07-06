package repository //nolint:testpackage // testing internal implementation

import (
	"context"
	"encoding/json"
	"log/slog"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/hexabase/hexabase-ai/api/internal/auth/domain"
	"github.com/hexabase/hexabase-ai/api/internal/shared/config"
	"github.com/hexabase/hexabase-ai/api/internal/shared/redis"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupRedisTest creates a test Redis client and repository
func setupRedisTest(t *testing.T) (*redis.Client, *redisAuthRepository, func()) {
	t.Helper()

	// Setup miniredis
	mr, err := miniredis.Run()
	require.NoError(t, err)

	// Create Redis client config
	cfg := &config.RedisConfig{
		Host:     "localhost",
		Port:     mr.Port(),
		Password: "",
		DB:       0,
	}
	logger := slog.Default()
	client, err := redis.NewClient(cfg, logger)
	require.NoError(t, err)

	repo := NewRedisAuthRepository(client)

	cleanup := func() {
		_ = client.Close()
		mr.Close()
	}

	return client, repo, cleanup
}

func TestRedisAuthRepository_AuthStateIsSignUpTrue(t *testing.T) {
	t.Parallel()

	client, repo, cleanup := setupRedisTest(t)
	defer cleanup()

	ctx := t.Context()

	// Create auth state with IsSignUp = true
	authState := &domain.AuthState{
		State:         "test-state-123",
		Provider:      "github",
		CodeChallenge: "challenge123",
		IsSignUp:      true, // This should be preserved
		ExpiresAt:     time.Now().Add(10 * time.Minute),
		CreatedAt:     time.Now(),
	}

	// Store auth state
	err := repo.StoreAuthState(ctx, authState)
	require.NoError(t, err)

	// Retrieve auth state
	retrieved, err := repo.GetAuthState(ctx, "test-state-123")
	require.NoError(t, err)
	require.NotNil(t, retrieved)

	// Verify IsSignUp is preserved
	assert.True(t, retrieved.IsSignUp, "IsSignUp flag should be true")
	assert.Equal(t, authState.Provider, retrieved.Provider)
	assert.Equal(t, authState.CodeChallenge, retrieved.CodeChallenge)

	// Also check the raw JSON in Redis
	rawData, err := client.Get(ctx, "auth_state:test-state-123")
	require.NoError(t, err)

	var rawState map[string]interface{}

	err = json.Unmarshal([]byte(rawData), &rawState)
	require.NoError(t, err)

	// Verify is_sign_up field exists in JSON
	isSignUp, exists := rawState["is_sign_up"]
	assert.True(t, exists, "is_sign_up field should exist in JSON")
	assert.Equal(t, true, isSignUp, "is_sign_up should be true in raw JSON")
}

func TestRedisAuthRepository_AuthStateIsSignUpFalse(t *testing.T) {
	t.Parallel()

	_, repo, cleanup := setupRedisTest(t)
	defer cleanup()

	ctx := context.Background()

	// Create auth state with IsSignUp = false
	authState := &domain.AuthState{
		State:     "test-state-456",
		Provider:  "google",
		IsSignUp:  false, // This should be preserved as false
		ExpiresAt: time.Now().Add(10 * time.Minute),
		CreatedAt: time.Now(),
	}

	// Store auth state
	err := repo.StoreAuthState(ctx, authState)
	require.NoError(t, err)

	// Retrieve auth state
	retrieved, err := repo.GetAuthState(ctx, "test-state-456")
	require.NoError(t, err)
	require.NotNil(t, retrieved)

	// Verify IsSignUp is preserved as false
	assert.False(t, retrieved.IsSignUp, "IsSignUp flag should be false")
}
