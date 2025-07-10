package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/hexabase/hexabase-ai/api/internal/auth/domain"
	"github.com/hexabase/hexabase-ai/api/internal/shared/redis"
	goredis "github.com/redis/go-redis/v9"
)

// ErrLockNotHeld is returned when trying to release a lock that is not held or has a value mismatch
var ErrLockNotHeld = errors.New("lock not held or value mismatch")

// sessionLimiterRepository implements session limit management using Redis
type sessionLimiterRepository struct {
	client *goredis.Client
}

//nolint:ireturn // Returning interface is intentional for DI
func NewSessionLimiterRepository(redisClient *redis.Client) domain.SessionLimiterRepository {
	return &sessionLimiterRepository{
		client: redisClient.GetClient(),
	}
}

// AddSessionToUser adds a session ID to the user's active session set
func (r *sessionLimiterRepository) AddSessionToUser(
	ctx context.Context,
	userID, sessionID string,
	ttl time.Duration,
) error {
	key := r.getKey(userID)

	// Use transaction (MULTI/EXEC) to ensure atomic operation
	pipe := r.client.TxPipeline()
	pipe.SAdd(ctx, key, sessionID)
	pipe.Expire(ctx, key, ttl)

	// Execute transaction
	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to add session with TTL: %w", err)
	}

	return nil
}

// RemoveSessionFromUser removes a session ID from the user's active session set
func (r *sessionLimiterRepository) RemoveSessionFromUser(
	ctx context.Context,
	userID, sessionID string,
) error {
	key := r.getKey(userID)

	if err := r.client.SRem(ctx, key, sessionID).Err(); err != nil {
		return fmt.Errorf("failed to remove session from set: %w", err)
	}

	return nil
}

// GetUserSessionCount gets the count of active sessions for a user
func (r *sessionLimiterRepository) GetUserSessionCount(ctx context.Context, userID string) (int64, error) {
	key := r.getKey(userID)

	count, err := r.client.SCard(ctx, key).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to get session count: %w", err)
	}

	return count, nil
}

// GetUserSessionIDs gets all active session IDs for a user
func (r *sessionLimiterRepository) GetUserSessionIDs(ctx context.Context, userID string) ([]string, error) {
	key := r.getKey(userID)

	sessions, err := r.client.SMembers(ctx, key).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get session IDs: %w", err)
	}

	return sessions, nil
}

// IsSessionInUserSet checks if a specified session ID exists in the user's set
func (r *sessionLimiterRepository) IsSessionInUserSet(
	ctx context.Context,
	userID, sessionID string,
) (bool, error) {
	key := r.getKey(userID)

	exists, err := r.client.SIsMember(ctx, key, sessionID).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check session existence: %w", err)
	}

	return exists, nil
}

// AcquireLock tries to acquire a distributed lock
func (r *sessionLimiterRepository) AcquireLock(
	ctx context.Context,
	key, value string,
	ttl time.Duration,
) (bool, error) {
	// SET key value NX EX ttl
	result, err := r.client.SetNX(ctx, key, value, ttl).Result()
	if err != nil {
		return false, fmt.Errorf("failed to acquire lock: %w", err)
	}

	return result, nil
}

// ReleaseLock releases a distributed lock if the value matches
func (r *sessionLimiterRepository) ReleaseLock(ctx context.Context, key, value string) error {
	// Use Lua script to ensure atomic compare-and-delete
	// Only delete if the current value matches the provided value
	luaScript := `
		if redis.call("get", KEYS[1]) == ARGV[1] then
			return redis.call("del", KEYS[1])
		else
			return 0
		end
	`

	result, err := r.client.Eval(ctx, luaScript, []string{key}, value).Result()
	if err != nil {
		return fmt.Errorf("failed to release lock: %w", err)
	}

	// Check if the lock was actually released
	if deleted, ok := result.(int64); ok && deleted == 0 {
		return fmt.Errorf("failed to release lock: %w", ErrLockNotHeld)
	}

	return nil
}

// getKey generates the Redis key for a user's session set
func (r *sessionLimiterRepository) getKey(userID string) string {
	return "user_sessions:" + userID
}
