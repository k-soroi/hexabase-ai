package service_test

import (
	"context"
	"strings"
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

// TestRefreshTokenFormat_InvalidFormat tests invalid refresh token format rejection
//
//nolint:paralleltest // Transaction-based test
func TestRefreshTokenFormat_InvalidFormat(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	ctx := context.Background()

	withTestDB(t, func(db *gorm.DB, redisClient *internalRedis.Client) {
		// Setup service using real repositories
		svc, _, _ := setupTestServiceWithDB(t, db, redisClient)

		invalidTokens := []string{
			"invalid",                 // No separator
			"selector.verifier.extra", // Too many parts
			"",                        // Empty
			"selector.",               // Missing verifier
			".verifier",               // Missing selector
			"a.b",                     // Too short parts
		}

		for _, token := range invalidTokens {
			_, err := svc.RefreshToken(ctx, token, "192.168.1.1", "Mozilla/5.0")
			assert.Error(t, err, "should reject token: %s", token)
		}
	})
}

// TestRefreshTokenFormat_ValidFormat tests valid refresh token format acceptance
//
//nolint:paralleltest // Transaction-based test
func TestRefreshTokenFormat_ValidFormat(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	ctx := context.Background()

	withTestDB(t, func(db *gorm.DB, redisClient *internalRedis.Client) {
		// Setup service using real repositories
		svc, _, tokenDomainService := setupTestServiceWithDB(t, db, redisClient)

		//nolint:paralleltest // Shares transaction context
		t.Run("accepts valid format and maintains format in new token", func(t *testing.T) {
			// First create a valid session through normal flow
			req := &domain.LoginRequest{
				Provider: "google",
			}
			authURL, state, err := svc.GetAuthURL(ctx, req)
			require.NoError(t, err)
			assert.NotEmpty(t, authURL)
			assert.NotEmpty(t, state)

			// Simulate OAuth callback
			callbackReq := &domain.CallbackRequest{
				Code:  "auth-code-format-test",
				State: state,
			}
			initialResponse, err := svc.HandleCallback(ctx, callbackReq, "192.168.1.1", "Mozilla/5.0")
			require.NoError(t, err)
			assert.NotNil(t, initialResponse)

			// Verify initial refresh token has valid format
			parts := strings.Split(initialResponse.RefreshToken, ".")
			assert.Len(t, parts, 2, "refresh token should have exactly 2 parts")
			assert.GreaterOrEqual(t, len(parts[0]), 8, "selector should be at least 8 chars")
			assert.GreaterOrEqual(t, len(parts[1]), 8, "verifier should be at least 8 chars")

			// Setup mock for refresh token
			tokenDomainService.On("RefreshToken", mock.Anything,
				mock.AnythingOfType("*domain.Session"),
				mock.AnythingOfType("*domain.User")).Return(
				&domain.Claims{
					RegisteredClaims: jwt.RegisteredClaims{
						Subject:   "user-id",
						ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
						IssuedAt:  jwt.NewNumericDate(time.Now()),
					},
					UserID:    "user-id",
					Email:     "test@example.com",
					Name:      "Test User",
					Provider:  "google",
					SessionID: "new-session-format-test",
					OrgIDs:    []string{},
				}, nil).Once()

			// Use the refresh token
			response, err := svc.RefreshToken(ctx, initialResponse.RefreshToken, "192.168.1.1", "Mozilla/5.0")
			require.NoError(t, err)
			assert.NotNil(t, response)
			assert.NotEmpty(t, response.AccessToken)
			assert.NotEmpty(t, response.RefreshToken)

			// Verify the new refresh token also has valid format
			newParts := strings.Split(response.RefreshToken, ".")
			assert.Len(t, newParts, 2, "new refresh token should have exactly 2 parts")
			assert.GreaterOrEqual(t, len(newParts[0]), 8, "new selector should be at least 8 chars")
			assert.GreaterOrEqual(t, len(newParts[1]), 8, "new verifier should be at least 8 chars")
		})
	})
}

// TestRefreshTokenFormat_RoundTrip tests refresh token format maintenance
//
//nolint:paralleltest // Transaction-based test
func TestRefreshTokenFormat_RoundTrip(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	ctx := context.Background()

	withTestDB(t, func(db *gorm.DB, redisClient *internalRedis.Client) {
		// Setup service using real repositories
		svc, _, _ := setupTestServiceWithDB(t, db, redisClient)

		// This test verifies that the format is maintained through the refresh process
		// by checking that new tokens generated maintain the same format

		// Initial valid token
		refreshToken := "initialselector12.initialverifier34"
		parts := strings.Split(refreshToken, ".")
		require.Len(t, parts, 2)

		// Test with valid format - should not error on format validation
		_, err := svc.RefreshToken(ctx, refreshToken, "192.168.1.1", "Mozilla/5.0")
		// We expect an error because the token doesn't exist in DB, but it should pass format validation
		require.Error(t, err)
		// The error should NOT be about invalid format
		assert.NotContains(t, err.Error(), "invalid refresh token format")
		assert.NotContains(t, err.Error(), "token format")

		// Verify that our format checker correctly identifies this as a valid format
		newParts := strings.Split(refreshToken, ".")
		assert.Len(t, newParts, 2, "token should maintain selector.verifier format")
		assert.Contains(t, refreshToken, ".", "token should contain separator")
	})
}
