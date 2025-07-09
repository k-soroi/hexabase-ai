package service_test

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	internalAuth "github.com/hexabase/hexabase-ai/api/internal/auth"
	"github.com/hexabase/hexabase-ai/api/internal/auth/domain"
	"github.com/hexabase/hexabase-ai/api/internal/auth/repository"
	"github.com/hexabase/hexabase-ai/api/internal/auth/service"
	internalRedis "github.com/hexabase/hexabase-ai/api/internal/shared/redis"
	"github.com/hexabase/hexabase-ai/api/internal/shared/testutil"
)

// Define static errors for testing
var (
	errProviderNotFound    = errors.New("provider not found")
	errUnsupportedProvider = errors.New("unsupported provider")
)

// TestMain sets up test infrastructure once per package
func TestMain(m *testing.M) {
	code := testutil.SetupTestDatabase(m)
	if code != 0 {
		return
	}
}

// withTestDB creates a new database from template for each test
func withTestDB(t *testing.T, fn func(db *gorm.DB, redisClient *internalRedis.Client)) {
	t.Helper()

	testutil.WithTestDB(t, fn)
}

// Note: withTestTx has been replaced with withTestDB
// Note: setupTestServiceWithTx has been replaced with setupTestServiceWithDB

// setupTestServiceWithDB creates a test service using database-isolated repositories
func setupTestServiceWithDB(
	t *testing.T, db *gorm.DB, redisClient *internalRedis.Client,
) (domain.Service, domain.OAuthRepository, *MockTokenDomainService) {
	t.Helper()

	// Create repository components
	postgresRepo := repository.NewPostgresRepository(db)
	redisAuthRepo := repository.NewRedisAuthRepository(redisClient)
	tokenHashRepo := repository.NewTokenHashRepository()

	// Create composite repository
	repo := repository.NewCompositeRepository(postgresRepo, redisAuthRepo, tokenHashRepo)

	// Create key repository
	keyRepo, err := repository.NewKeyRepository()
	require.NoError(t, err)

	// Get keys from real key repository
	privKeyPEM, err := keyRepo.GetPrivateKey()
	require.NoError(t, err)
	pubKeyPEM, err := keyRepo.GetPublicKey()
	require.NoError(t, err)

	// Parse keys
	testPrivateKey, testPublicKey := setupTestKeys(t, privKeyPEM, pubKeyPEM)

	// Create real token manager and domain service
	tokenManager := internalAuth.NewTokenManager(
		testPrivateKey,
		testPublicKey,
		"test-issuer",
		time.Hour,
	)
	tokenDomainService := new(MockTokenDomainService)

	// Create session limiter repository using Redis
	sessionLimiterRepo := repository.NewSessionLimiterRepository(redisClient)
	sessionManager := service.NewSessionManager(repo, sessionLimiterRepo)

	// Create OAuth stub for testing with a UNIQUE user for each test
	// This prevents session limit collisions between concurrent tests.
	uniqueID := uuid.New().String()
	userInfo := &domain.UserInfo{
		ID:       "google-" + uniqueID,
		Email:    fmt.Sprintf("test-%s@example.com", uniqueID),
		Name:     "Test User",
		Provider: "google",
	}
	oauthRepo := newStubOAuthRepository(userInfo)

	// Create service with real dependencies
	svc := service.NewService(
		repo,
		oauthRepo,
		keyRepo,
		tokenManager,
		tokenDomainService,
		sessionManager,
		slog.Default(),
		3600,
	)

	return svc, oauthRepo, tokenDomainService
}

// setupTestKeys parses test RSA keys from PEM format
func setupTestKeys(t *testing.T, privKeyPEM, pubKeyPEM []byte) (*rsa.PrivateKey, *rsa.PublicKey) {
	t.Helper()

	// Parse PEM keys
	privBlock, _ := pem.Decode(privKeyPEM)
	require.NotNil(t, privBlock, "failed to decode private key PEM")
	testPrivateKey, err := x509.ParsePKCS1PrivateKey(privBlock.Bytes)
	require.NoError(t, err)

	pubBlock, _ := pem.Decode(pubKeyPEM)
	require.NotNil(t, pubBlock, "failed to decode public key PEM")
	pubKeyInterface, err := x509.ParsePKIXPublicKey(pubBlock.Bytes)
	require.NoError(t, err)

	testPublicKey, ok := pubKeyInterface.(*rsa.PublicKey)
	require.True(t, ok, "failed to parse public key as RSA public key")

	return testPrivateKey, testPublicKey
}

// stubOAuthRepository provides a simple OAuth implementation for testing
type stubOAuthRepository struct {
	userInfo *domain.UserInfo
}

func newStubOAuthRepository(userInfo *domain.UserInfo) domain.OAuthRepository {
	// Fallback to default user if none provided, for safety, though tests should provide one.
	if userInfo == nil {
		userInfo = &domain.UserInfo{
			ID:       "google-123",
			Email:    "test@example.com",
			Name:     "Test User",
			Provider: "google",
		}
	}

	return &stubOAuthRepository{userInfo: userInfo}
}

func (s *stubOAuthRepository) GetProviderConfig(provider string) (*domain.ProviderConfig, error) {
	configs := map[string]*domain.ProviderConfig{
		"google": {
			ClientID:     "test-client-id",
			ClientSecret: "test-client-secret",
			RedirectURL:  "http://localhost:8080/callback",
			Scopes:       []string{"openid", "email", "profile"},
		},
	}

	config, ok := configs[provider]
	if !ok {
		return nil, fmt.Errorf("%w: %s", errProviderNotFound, provider)
	}

	return config, nil
}

func (s *stubOAuthRepository) GetAuthURL(
	provider, state string,
	params map[string]string,
) (string, error) {
	if provider == "google" {
		url := "https://accounts.google.com/o/oauth2/v2/auth?state=" + state
		// Add any additional parameters (like PKCE)
		for key, value := range params {
			url += "&" + key + "=" + value
		}

		return url, nil
	}

	return "", fmt.Errorf("%w: %s", errUnsupportedProvider, provider)
}

func (s *stubOAuthRepository) ExchangeCode(
	ctx context.Context,
	provider, code string,
) (*domain.OAuthToken, error) {
	return &domain.OAuthToken{
		AccessToken:  "test-access-token",
		RefreshToken: "test-refresh-token",
		TokenType:    "Bearer",
		ExpiresIn:    3600,
	}, nil
}

func (s *stubOAuthRepository) GetUserInfo(
	ctx context.Context, provider string, token *domain.OAuthToken,
) (*domain.UserInfo, error) {
	// Return the specific user info configured for this test instance
	return s.userInfo, nil
}

func (s *stubOAuthRepository) RefreshOAuthToken(
	ctx context.Context, provider string, refreshToken string,
) (*domain.OAuthToken, error) {
	return &domain.OAuthToken{
		AccessToken:  "new-access-token",
		RefreshToken: "new-refresh-token",
		TokenType:    "Bearer",
		ExpiresIn:    3600,
	}, nil
}
