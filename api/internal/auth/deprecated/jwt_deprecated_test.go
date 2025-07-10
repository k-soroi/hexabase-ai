//nolint:all // This package contains deprecated code that is not used in the current version
package deprecated_test

import (
	"crypto/rand"
	"crypto/rsa"
	"strings"
	"testing"
	"time"

	"github.com/hexabase/hexabase-ai/api/internal/auth/deprecated"
	"github.com/stretchr/testify/assert"
)

func generateTestKeyPair(t *testing.T) (*rsa.PrivateKey, *rsa.PublicKey) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	assert.NoError(t, err)
	return privateKey, &privateKey.PublicKey
}

func TestTokenManager_GenerateToken(t *testing.T) {
	privateKey, publicKey := generateTestKeyPair(t)
	tm := deprecated.NewTokenManager(privateKey, publicKey, "https://api.hexabase.test", 1*time.Hour)

	userID := "test-user-123"
	email := "test@example.com"
	name := "Test User"
	orgIDs := []string{"org-123", "org-456"}

	token, err := tm.GenerateToken(userID, email, name, orgIDs)
	assert.NoError(t, err)
	assert.NotEmpty(t, token)

	// Verify token structure (JWT has 3 parts separated by dots)
	parts := strings.Split(token, ".")
	assert.Len(t, parts, 3)
}

func TestTokenManager_ValidateToken(t *testing.T) {
	privateKey, publicKey := generateTestKeyPair(t)
	tm := deprecated.NewTokenManager(privateKey, publicKey, "https://api.hexabase.test", 1*time.Hour)

	userID := "test-user-123"
	email := "test@example.com"
	name := "Test User"
	orgIDs := []string{"org-123", "org-456"}

	// Generate token
	token, err := tm.GenerateToken(userID, email, name, orgIDs)
	assert.NoError(t, err)

	// Validate token
	claims, err := tm.ValidateToken(token)
	assert.NoError(t, err)
	assert.NotNil(t, claims)
	assert.Equal(t, userID, claims.UserID)
	assert.Equal(t, email, claims.Email)
	assert.Equal(t, name, claims.Name)
	assert.Equal(t, orgIDs, claims.OrgIDs)
	assert.Equal(t, "https://api.hexabase.test", claims.Issuer)
	assert.Contains(t, claims.Audience, "hexabase-api")
}

func TestTokenManager_ValidateToken_Expired(t *testing.T) {
	privateKey, publicKey := generateTestKeyPair(t)
	// Create token manager with negative expiration to generate expired tokens
	tm := deprecated.NewTokenManager(privateKey, publicKey, "https://api.hexabase.test", -1*time.Hour)

	token, err := tm.GenerateToken("user-123", "test@example.com", "Test User", nil)
	assert.NoError(t, err)

	// Validate expired token
	claims, err := tm.ValidateToken(token)
	assert.Error(t, err)
	assert.Nil(t, claims)
	assert.Contains(t, err.Error(), "token is expired")
}

func TestTokenManager_ValidateToken_InvalidSignature(t *testing.T) {
	privateKey1, publicKey1 := generateTestKeyPair(t)
	privateKey2, publicKey2 := generateTestKeyPair(t)

	// Create token with one key
	tm1 := deprecated.NewTokenManager(privateKey1, publicKey1, "https://api.hexabase.test", 1*time.Hour)
	token, err := tm1.GenerateToken("user-123", "test@example.com", "Test User", nil)
	assert.NoError(t, err)

	// Try to validate with different key
	tm2 := deprecated.NewTokenManager(privateKey2, publicKey2, "https://api.hexabase.test", 1*time.Hour)
	claims, err := tm2.ValidateToken(token)
	assert.Error(t, err)
	assert.Nil(t, claims)
}
