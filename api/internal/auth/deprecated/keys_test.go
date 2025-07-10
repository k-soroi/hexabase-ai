package deprecated_test

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/hexabase/hexabase-ai/api/internal/auth/deprecated"
	"github.com/stretchr/testify/assert"
)

func TestKeyManager_NewKeyManager(t *testing.T) {
	// Test with no existing keys
	km, err := deprecated.NewKeyManager()
	assert.NoError(t, err)
	assert.NotNil(t, km)
	assert.NotNil(t, km.GetPrivateKey())
	assert.NotNil(t, km.GetPublicKey())
}

func TestKeyManager_GetJWKS(t *testing.T) {
	km, err := deprecated.NewKeyManager()
	assert.NoError(t, err)

	jwks, err := km.GetJWKS()
	assert.NoError(t, err)
	assert.NotNil(t, jwks)
	assert.Len(t, jwks.Keys, 1)

	jwk := jwks.Keys[0]
	assert.Equal(t, "RSA", jwk.Kty)
	assert.Equal(t, "sig", jwk.Use)
	assert.Equal(t, "RS256", jwk.Alg)
	assert.NotEmpty(t, jwk.Kid)
	assert.NotEmpty(t, jwk.N)
	assert.NotEmpty(t, jwk.E)
}

func TestKeyManager_GetJWKSJSON(t *testing.T) {
	km, err := deprecated.NewKeyManager()
	assert.NoError(t, err)

	jwksJSON, err := km.GetJWKSJSON()
	assert.NoError(t, err)
	assert.NotEmpty(t, jwksJSON)

	// Verify it's valid JSON
	var jwks deprecated.JWKS
	err = json.Unmarshal(jwksJSON, &jwks)
	assert.NoError(t, err)
	assert.Len(t, jwks.Keys, 1)
}

func TestKeyManager_GetPublicKeyPEM(t *testing.T) {
	km, err := deprecated.NewKeyManager()
	assert.NoError(t, err)

	pem, err := km.GetPublicKeyPEM()
	assert.NoError(t, err)
	assert.NotEmpty(t, pem)
	assert.Contains(t, pem, "-----BEGIN PUBLIC KEY-----")
	assert.Contains(t, pem, "-----END PUBLIC KEY-----")
}

func TestKeyManager_Persistence(t *testing.T) {
	// Set custom key path for testing
	tempFile := "/tmp/test-hexabase-key.pem"
	os.Setenv("JWT_PRIVATE_KEY_PATH", tempFile)
	defer os.Unsetenv("JWT_PRIVATE_KEY_PATH")
	defer os.Remove(tempFile)

	// Create first key manager (should generate and save key)
	km1, err := deprecated.NewKeyManager()
	assert.NoError(t, err)
	key1PEM, err := km1.GetPublicKeyPEM()
	assert.NoError(t, err)

	// Verify key was saved
	_, err = os.Stat(tempFile)
	assert.NoError(t, err)

	// Create second key manager (should load existing key)
	km2, err := deprecated.NewKeyManager()
	assert.NoError(t, err)
	key2PEM, err := km2.GetPublicKeyPEM()
	assert.NoError(t, err)

	// Verify same key was loaded
	assert.Equal(t, key1PEM, key2PEM)
}