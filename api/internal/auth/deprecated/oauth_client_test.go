package deprecated_test

import (
	"context"
	"encoding/base64"
	"testing"

	"github.com/hexabase/hexabase-ai/api/internal/auth/deprecated"
	"github.com/hexabase/hexabase-ai/api/internal/shared/config"
	"github.com/stretchr/testify/assert"
	"golang.org/x/oauth2"
)

func TestOAuthClient_NewOAuthClient(t *testing.T) {
	cfg := &config.Config{
		Auth: config.AuthConfig{
			ExternalProviders: map[string]config.OAuthProvider{
				"google": {
					ClientID:     "google-client-id",
					ClientSecret: "google-client-secret",
					RedirectURL:  "https://api.hexabase.test/auth/callback/google",
					Scopes:       []string{"openid", "email", "profile"},
				},
				"github": {
					ClientID:     "github-client-id",
					ClientSecret: "github-client-secret",
					RedirectURL:  "https://api.hexabase.test/auth/callback/github",
					Scopes:       []string{"user:email", "read:user"},
				},
			},
		},
	}

	client := deprecated.NewOAuthClient(cfg, nil)
	assert.NotNil(t, client)
}

func TestOAuthClient_GetAuthURL(t *testing.T) {
	cfg := &config.Config{
		Auth: config.AuthConfig{
			ExternalProviders: map[string]config.OAuthProvider{
				"google": {
					ClientID:     "google-client-id",
					ClientSecret: "google-client-secret",
					RedirectURL:  "https://api.hexabase.test/auth/callback/google",
					Scopes:       []string{"openid", "email", "profile"},
				},
				"github": {
					ClientID:     "github-client-id",
					ClientSecret: "github-client-secret",
					RedirectURL:  "https://api.hexabase.test/auth/callback/github",
					Scopes:       []string{"user:email", "read:user"},
				},
			},
		},
	}

	client := deprecated.NewOAuthClient(cfg, nil)

	// Test Google provider
	authURL, err := client.GetAuthURL("google", "test-state")
	assert.NoError(t, err)
	assert.NotEmpty(t, authURL)
	assert.Contains(t, authURL, "accounts.google.com")
	assert.Contains(t, authURL, "client_id=google-client-id")
	assert.Contains(t, authURL, "state=test-state")

	// Test GitHub provider
	authURL, err = client.GetAuthURL("github", "test-state")
	assert.NoError(t, err)
	assert.NotEmpty(t, authURL)
	assert.Contains(t, authURL, "github.com")
	assert.Contains(t, authURL, "client_id=github-client-id")
	assert.Contains(t, authURL, "state=test-state")

	// Test invalid provider
	authURL, err = client.GetAuthURL("invalid", "test-state")
	assert.Error(t, err)
	assert.Empty(t, authURL)
	assert.Contains(t, err.Error(), "not configured")
}

func TestOAuthClient_GenerateState(t *testing.T) {
	cfg := &config.Config{}
	client := deprecated.NewOAuthClient(cfg, nil)

	state1, err := client.GenerateState()
	assert.NoError(t, err)
	assert.NotEmpty(t, state1)

	// Verify it's base64 encoded
	decoded, err := base64.URLEncoding.DecodeString(state1)
	assert.NoError(t, err)
	assert.Len(t, decoded, 32) // 32 bytes

	// Generate another state to ensure they're unique
	state2, err := client.GenerateState()
	assert.NoError(t, err)
	assert.NotEqual(t, state1, state2)
}

func TestOAuthClient_ValidateState(t *testing.T) {
	cfg := &config.Config{}
	client := deprecated.NewOAuthClient(cfg, nil)

	// Valid state
	state, err := client.GenerateState()
	assert.NoError(t, err)
	err = client.ValidateState(state)
	assert.NoError(t, err)

	// Invalid state - empty
	err = client.ValidateState("")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "empty state")
}

func TestOAuthClient_ExchangeCode_InvalidProvider(t *testing.T) {
	cfg := &config.Config{}
	client := deprecated.NewOAuthClient(cfg, nil)

	ctx := context.Background()
	token, err := client.ExchangeCode(ctx, "invalid", "test-code")
	assert.Error(t, err)
	assert.Nil(t, token)
	assert.Contains(t, err.Error(), "not configured")
}

func TestOAuthClient_GetUserInfo_InvalidProvider(t *testing.T) {
	cfg := &config.Config{}
	client := deprecated.NewOAuthClient(cfg, nil)

	ctx := context.Background()
	token := &oauth2.Token{
		AccessToken: "test-access-token",
		TokenType:   "Bearer",
	}

	// Test with unconfigured provider
	userInfo, err := client.GetUserInfo(ctx, "invalid", token)
	assert.Error(t, err)
	assert.Nil(t, userInfo)
	assert.Contains(t, err.Error(), "not configured")
}