package deprecated_test

import (
	"context"
	"testing"
	"time"

	"github.com/hexabase/hexabase-ai/api/internal/auth/deprecated"
	"github.com/hexabase/hexabase-ai/api/internal/shared/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockRedisClient is a mock implementation of RedisClient
type MockRedisClient struct {
	mock.Mock
}

func (m *MockRedisClient) SetWithTTL(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	args := m.Called(ctx, key, value, ttl)
	return args.Error(0)
}

func (m *MockRedisClient) GetDel(ctx context.Context, key string) (string, error) {
	args := m.Called(ctx, key)
	return args.String(0), args.Error(1)
}

func (m *MockRedisClient) Get(ctx context.Context, key string) (string, error) {
	args := m.Called(ctx, key)
	return args.String(0), args.Error(1)
}

func (m *MockRedisClient) Delete(ctx context.Context, keys ...string) error {
	args := m.Called(ctx, keys)
	return args.Error(0)
}

func (m *MockRedisClient) Exists(ctx context.Context, keys ...string) (int64, error) {
	args := m.Called(ctx, keys)
	return args.Get(0).(int64), args.Error(1)
}

func TestOAuthClient_GenerateAndStoreState(t *testing.T) {
	cfg := &config.Config{}
	mockRedis := new(MockRedisClient)

	// Expect state to be stored in Redis
	mockRedis.On("SetWithTTL", mock.Anything, mock.MatchedBy(func(key string) bool {
		return len(key) > len("oauth_state:")
	}), "valid", 10*time.Minute).Return(nil)

	client := deprecated.NewOAuthClient(cfg, mockRedis)

	ctx := context.Background()
	state, err := client.GenerateAndStoreState(ctx)

	assert.NoError(t, err)
	assert.NotEmpty(t, state)
	mockRedis.AssertExpectations(t)
}

func TestOAuthClient_ValidateAndConsumeState_Valid(t *testing.T) {
	cfg := &config.Config{}
	mockRedis := new(MockRedisClient)

	testState := "test-state-123"
	expectedKey := "oauth_state:test-state-123"

	// Expect state to be retrieved and deleted from Redis
	mockRedis.On("GetDel", mock.Anything, expectedKey).Return("valid", nil)

	client := deprecated.NewOAuthClient(cfg, mockRedis)

	ctx := context.Background()
	err := client.ValidateAndConsumeState(ctx, testState)

	assert.NoError(t, err)
	mockRedis.AssertExpectations(t)
}

func TestOAuthClient_ValidateAndConsumeState_Invalid(t *testing.T) {
	cfg := &config.Config{}
	mockRedis := new(MockRedisClient)

	testState := "test-state-123"
	expectedKey := "oauth_state:test-state-123"

	// Expect state to be not found in Redis
	mockRedis.On("GetDel", mock.Anything, expectedKey).Return("", assert.AnError)

	client := deprecated.NewOAuthClient(cfg, mockRedis)

	ctx := context.Background()
	err := client.ValidateAndConsumeState(ctx, testState)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid or expired state")
	mockRedis.AssertExpectations(t)
}

func TestOAuthClient_ValidateAndConsumeState_EmptyState(t *testing.T) {
	cfg := &config.Config{}
	mockRedis := new(MockRedisClient)

	client := deprecated.NewOAuthClient(cfg, mockRedis)

	ctx := context.Background()
	err := client.ValidateAndConsumeState(ctx, "")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "empty state parameter")
	// No Redis calls should be made for empty state
	mockRedis.AssertNotCalled(t, "GetDel")
}

func TestOAuthClient_WithoutRedis(t *testing.T) {
	cfg := &config.Config{}
	client := deprecated.NewOAuthClient(cfg, nil)

	ctx := context.Background()

	// Without Redis, should still generate state but not store it
	state, err := client.GenerateAndStoreState(ctx)
	assert.NoError(t, err)
	assert.NotEmpty(t, state)

	// Without Redis, validation should pass for any non-empty state
	err = client.ValidateAndConsumeState(ctx, "any-state")
	assert.NoError(t, err)
}