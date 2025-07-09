package deprecated

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/hexabase/hexabase-ai/api/internal/shared/config"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
	"golang.org/x/oauth2/google"
)

// OAuthClient handles OAuth2 authentication with external providers
type OAuthClient struct {
	providers map[string]*oauth2.Config
	config    *config.Config
	redis     RedisClient
}

// RedisClient interface for state storage
type RedisClient interface {
	SetWithTTL(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	GetDel(ctx context.Context, key string) (string, error)
	Get(ctx context.Context, key string) (string, error)
	Delete(ctx context.Context, keys ...string) error
	Exists(ctx context.Context, keys ...string) (int64, error)
}

// UserInfo represents user information from OAuth providers
type UserInfo struct {
	ID          string `json:"id"`
	Email       string `json:"email"`
	Name        string `json:"name"`
	Picture     string `json:"picture,omitempty"`
	Provider    string `json:"provider"`
}

// NewOAuthClient creates a new OAuth client
func NewOAuthClient(cfg *config.Config, redisClient RedisClient) *OAuthClient {
	providers := make(map[string]*oauth2.Config)

	// Configure Google OAuth2
	if googleCfg, ok := cfg.Auth.ExternalProviders["google"]; ok {
		providers["google"] = &oauth2.Config{
			ClientID:     googleCfg.ClientID,
			ClientSecret: googleCfg.ClientSecret,
			RedirectURL:  googleCfg.RedirectURL,
			Scopes:       googleCfg.Scopes,
			Endpoint:     google.Endpoint,
		}
	}

	// Configure GitHub OAuth2
	if githubCfg, ok := cfg.Auth.ExternalProviders["github"]; ok {
		providers["github"] = &oauth2.Config{
			ClientID:     githubCfg.ClientID,
			ClientSecret: githubCfg.ClientSecret,
			RedirectURL:  githubCfg.RedirectURL,
			Scopes:       githubCfg.Scopes,
			Endpoint:     github.Endpoint,
		}
	}

	return &OAuthClient{
		providers: providers,
		config:    cfg,
		redis:     redisClient,
	}
}

// GenerateAndStoreState generates a secure random state for OAuth2 and stores it in Redis
func (c *OAuthClient) GenerateAndStoreState(ctx context.Context) (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	state := base64.URLEncoding.EncodeToString(b)

	// Store state in Redis with 10 minute expiration
	if c.redis != nil {
		key := fmt.Sprintf("oauth_state:%s", state)
		if err := c.redis.SetWithTTL(ctx, key, "valid", 10*time.Minute); err != nil {
			return "", fmt.Errorf("failed to store state in Redis: %w", err)
		}
	}

	return state, nil
}

// GenerateState generates a secure random state (legacy method for tests)
func (c *OAuthClient) GenerateState() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// GetAuthURL returns the OAuth2 authorization URL for a provider
func (c *OAuthClient) GetAuthURL(provider, state string) (string, error) {
	cfg, ok := c.providers[provider]
	if !ok {
		return "", fmt.Errorf("provider %s not configured", provider)
	}

	return cfg.AuthCodeURL(state, oauth2.AccessTypeOffline), nil
}

// ExchangeCode exchanges an authorization code for tokens
func (c *OAuthClient) ExchangeCode(ctx context.Context, provider, code string) (*oauth2.Token, error) {
	cfg, ok := c.providers[provider]
	if !ok {
		return nil, fmt.Errorf("provider %s not configured", provider)
	}

	return cfg.Exchange(ctx, code)
}

// GetUserInfo fetches user information from the OAuth provider
func (c *OAuthClient) GetUserInfo(ctx context.Context, provider string, token *oauth2.Token) (*UserInfo, error) {
	cfg, ok := c.providers[provider]
	if !ok {
		return nil, fmt.Errorf("provider %s not configured", provider)
	}

	client := cfg.Client(ctx, token)

	switch provider {
	case "google":
		return c.getGoogleUserInfo(ctx, client)
	case "github":
		return c.getGitHubUserInfo(ctx, client)
	default:
		return nil, fmt.Errorf("unsupported provider: %s", provider)
	}
}

// getGoogleUserInfo fetches user info from Google
func (c *OAuthClient) getGoogleUserInfo(ctx context.Context, client *http.Client) (*UserInfo, error) {
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}
	defer resp.Body.Close()

	var googleUser struct {
		ID      string `json:"id"`
		Email   string `json:"email"`
		Name    string `json:"name"`
		Picture string `json:"picture"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&googleUser); err != nil {
		return nil, fmt.Errorf("failed to decode user info: %w", err)
	}

	return &UserInfo{
		ID:       googleUser.ID,
		Email:    googleUser.Email,
		Name:     googleUser.Name,
		Picture:  googleUser.Picture,
		Provider: "google",
	}, nil
}

// getGitHubUserInfo fetches user info from GitHub
func (c *OAuthClient) getGitHubUserInfo(ctx context.Context, client *http.Client) (*UserInfo, error) {
	// Get user info
	resp, err := client.Get("https://api.github.com/user")
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}
	defer resp.Body.Close()

	var githubUser struct {
		ID       int64  `json:"id"`
		Login    string `json:"login"`
		Name     string `json:"name"`
		Email    string `json:"email"`
		Avatar   string `json:"avatar_url"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&githubUser); err != nil {
		return nil, fmt.Errorf("failed to decode user info: %w", err)
	}

	// GitHub might not return email in the user endpoint
	if githubUser.Email == "" {
		// Fetch primary email
		emailResp, err := client.Get("https://api.github.com/user/emails")
		if err == nil {
			defer emailResp.Body.Close()
			
			var emails []struct {
				Email    string `json:"email"`
				Primary  bool   `json:"primary"`
				Verified bool   `json:"verified"`
			}
			
			if err := json.NewDecoder(emailResp.Body).Decode(&emails); err == nil {
				for _, email := range emails {
					if email.Primary && email.Verified {
						githubUser.Email = email.Email
						break
					}
				}
			}
		}
	}

	// Use login as name if name is empty
	if githubUser.Name == "" {
		githubUser.Name = githubUser.Login
	}

	return &UserInfo{
		ID:       fmt.Sprintf("%d", githubUser.ID),
		Email:    githubUser.Email,
		Name:     githubUser.Name,
		Picture:  githubUser.Avatar,
		Provider: "github",
	}, nil
}

// ValidateAndConsumeState validates the OAuth state parameter and removes it from Redis
func (c *OAuthClient) ValidateAndConsumeState(ctx context.Context, state string) error {
	if state == "" {
		return fmt.Errorf("empty state parameter")
	}

	// Validate state from Redis
	if c.redis != nil {
		key := fmt.Sprintf("oauth_state:%s", state)
		_, err := c.redis.GetDel(ctx, key)
		if err != nil {
			return fmt.Errorf("invalid or expired state: %w", err)
		}
	}

	return nil
}

// ValidateState validates the OAuth state parameter (legacy method for tests)
func (c *OAuthClient) ValidateState(state string) error {
	if state == "" {
		return fmt.Errorf("empty state parameter")
	}
	return nil
}