package service

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	internalAuth "github.com/hexabase/hexabase-ai/api/internal/auth"
	"github.com/hexabase/hexabase-ai/api/internal/auth/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// Mock repository
type mockRepository struct {
	mock.Mock
}

// Mock token domain service
type mockTokenDomainService struct {
	mock.Mock
}

// Mock session manager
type mockSessionManager struct {
	mock.Mock
}

// setupTestService creates a test service with all dependencies mocked
func setupTestService() (
	*service,
	*mockRepository,
	*mockOAuthRepository,
	*mockTokenDomainService,
	*mockSessionManager,
) {
	mockRepo := new(mockRepository)
	mockOAuthRepo := new(mockOAuthRepository)
	mockKeyRepo := new(mockKeyRepository)
	mockTokenDomainService := new(mockTokenDomainService)
	mockSessionManager := new(mockSessionManager)

	// Create a dummy TokenManager
	testPrivateKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	testPublicKey := &testPrivateKey.PublicKey
	tokenManager := internalAuth.NewTokenManager(testPrivateKey, testPublicKey, "test-issuer", time.Hour)

	svc := &service{
		repo:               mockRepo,
		oauthRepo:          mockOAuthRepo,
		keyRepo:            mockKeyRepo,
		tokenManager:       tokenManager,
		tokenDomainService: mockTokenDomainService,
		sessionManager:     mockSessionManager,
		logger:             slog.Default(),
		defaultTokenExpiry: 3600, // 1 hour default
	}

	return svc, mockRepo, mockOAuthRepo, mockTokenDomainService, mockSessionManager
}

func (m *mockSessionManager) CreateSession(ctx context.Context, userID, sessionID string) error {
	args := m.Called(ctx, userID, sessionID)
	return args.Error(0)
}

func (m *mockSessionManager) DeleteSession(ctx context.Context, userID, sessionID string) error {
	args := m.Called(ctx, userID, sessionID)
	return args.Error(0)
}

func (m *mockSessionManager) GetActiveSessionCount(ctx context.Context, userID string) (int, error) {
	args := m.Called(ctx, userID)
	return args.Int(0), args.Error(1)
}

func (m *mockTokenDomainService) RefreshToken(ctx context.Context, session *domain.Session, user *domain.User) (*domain.Claims, error) {
	args := m.Called(ctx, session, user)
	if args.Get(0) != nil {
		return args.Get(0).(*domain.Claims), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockTokenDomainService) ValidateRefreshEligibility(session *domain.Session) error {
	args := m.Called(session)
	return args.Error(0)
}

func (m *mockTokenDomainService) CreateSession(sessionID, userID, refreshToken, deviceID, clientIP, userAgent string) (*domain.Session, error) {
	args := m.Called(sessionID, userID, refreshToken, deviceID, clientIP, userAgent)
	if args.Get(0) != nil {
		return args.Get(0).(*domain.Session), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockTokenDomainService) ValidateTokenClaims(claims *domain.Claims) error {
	args := m.Called(claims)
	return args.Error(0)
}

func (m *mockTokenDomainService) ShouldRefreshToken(claims *domain.Claims) bool {
	args := m.Called(claims)
	return args.Bool(0)
}

func (m *mockRepository) CreateUser(ctx context.Context, user *domain.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *mockRepository) GetUser(ctx context.Context, userID string) (*domain.User, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) != nil {
		return args.Get(0).(*domain.User), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockRepository) GetUserByExternalID(ctx context.Context, externalID, provider string) (*domain.User, error) {
	args := m.Called(ctx, externalID, provider)
	if args.Get(0) != nil {
		return args.Get(0).(*domain.User), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockRepository) GetUserByEmail(ctx context.Context, email string) (*domain.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) != nil {
		return args.Get(0).(*domain.User), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockRepository) UpdateUser(ctx context.Context, user *domain.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *mockRepository) UpdateLastLogin(ctx context.Context, userID string) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func (m *mockRepository) CreateSession(ctx context.Context, session *domain.Session) error {
	args := m.Called(ctx, session)
	return args.Error(0)
}

func (m *mockRepository) GetSession(ctx context.Context, sessionID string) (*domain.Session, error) {
	args := m.Called(ctx, sessionID)
	if args.Get(0) != nil {
		return args.Get(0).(*domain.Session), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockRepository) GetSessionByRefreshTokenSelector(ctx context.Context, selector string) (*domain.Session, error) {
	args := m.Called(ctx, selector)
	if args.Get(0) != nil {
		return args.Get(0).(*domain.Session), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockRepository) ListUserSessions(ctx context.Context, userID string) ([]*domain.Session, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) != nil {
		return args.Get(0).([]*domain.Session), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockRepository) UpdateSession(ctx context.Context, session *domain.Session) error {
	args := m.Called(ctx, session)
	return args.Error(0)
}

func (m *mockRepository) DeleteSession(ctx context.Context, sessionID string) error {
	args := m.Called(ctx, sessionID)
	return args.Error(0)
}

func (m *mockRepository) DeleteUserSessions(ctx context.Context, userID string, exceptSessionID string) error {
	args := m.Called(ctx, userID, exceptSessionID)
	return args.Error(0)
}

func (m *mockRepository) CleanupExpiredSessions(ctx context.Context, before time.Time) error {
	args := m.Called(ctx, before)
	return args.Error(0)
}

func (m *mockRepository) StoreAuthState(ctx context.Context, state *domain.AuthState) error {
	args := m.Called(ctx, state)
	return args.Error(0)
}

func (m *mockRepository) GetAuthState(ctx context.Context, stateValue string) (*domain.AuthState, error) {
	args := m.Called(ctx, stateValue)
	if args.Get(0) != nil {
		return args.Get(0).(*domain.AuthState), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockRepository) DeleteAuthState(ctx context.Context, stateValue string) error {
	args := m.Called(ctx, stateValue)
	return args.Error(0)
}

func (m *mockRepository) BlacklistRefreshToken(ctx context.Context, token string, expiresAt time.Time) error {
	args := m.Called(ctx, token, expiresAt)
	return args.Error(0)
}

func (m *mockRepository) IsRefreshTokenBlacklisted(ctx context.Context, token string) (bool, error) {
	args := m.Called(ctx, token)
	return args.Bool(0), args.Error(1)
}

func (m *mockRepository) CreateSecurityEvent(ctx context.Context, event *domain.SecurityEvent) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}

func (m *mockRepository) ListSecurityEvents(ctx context.Context, filter domain.SecurityLogFilter) ([]*domain.SecurityEvent, error) {
	args := m.Called(ctx, filter)
	if args.Get(0) != nil {
		return args.Get(0).([]*domain.SecurityEvent), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockRepository) CleanupOldSecurityEvents(ctx context.Context, before time.Time) error {
	args := m.Called(ctx, before)
	return args.Error(0)
}

func (m *mockRepository) GetUserOrganizations(ctx context.Context, userID string) ([]string, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) != nil {
		return args.Get(0).([]string), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockRepository) GetUserWorkspaceGroups(ctx context.Context, userID, workspaceID string) ([]string, error) {
	args := m.Called(ctx, userID, workspaceID)
	if args.Get(0) != nil {
		return args.Get(0).([]string), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockRepository) HashToken(token string) (hashedToken string, salt string, err error) {
	args := m.Called(token)
	return args.String(0), args.String(1), args.Error(2)
}

func (m *mockRepository) VerifyToken(plainToken, hashedToken, salt string) bool {
	args := m.Called(plainToken, hashedToken, salt)
	return args.Bool(0)
}

func (m *mockRepository) GetAllActiveSessions(ctx context.Context) ([]*domain.Session, error) {
	args := m.Called(ctx)
	if args.Get(0) != nil {
		return args.Get(0).([]*domain.Session), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockRepository) BlockSession(ctx context.Context, sessionID string, expiresAt time.Time) error {
	args := m.Called(ctx, sessionID, expiresAt)
	return args.Error(0)
}

func (m *mockRepository) IsSessionBlocked(ctx context.Context, sessionID string) (bool, error) {
	args := m.Called(ctx, sessionID)
	return args.Bool(0), args.Error(1)
}

// Mock OAuth repository
type mockOAuthRepository struct {
	mock.Mock
}

func (m *mockOAuthRepository) GetProviderConfig(provider string) (*domain.ProviderConfig, error) {
	args := m.Called(provider)
	if args.Get(0) != nil {
		return args.Get(0).(*domain.ProviderConfig), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockOAuthRepository) GetAuthURL(provider, state string, params map[string]string) (string, error) {
	args := m.Called(provider, state, params)
	return args.String(0), args.Error(1)
}

func (m *mockOAuthRepository) ExchangeCode(ctx context.Context, provider, code string) (*domain.OAuthToken, error) {
	args := m.Called(ctx, provider, code)
	if args.Get(0) != nil {
		return args.Get(0).(*domain.OAuthToken), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockOAuthRepository) GetUserInfo(ctx context.Context, provider string, token *domain.OAuthToken) (*domain.UserInfo, error) {
	args := m.Called(ctx, provider, token)
	if args.Get(0) != nil {
		return args.Get(0).(*domain.UserInfo), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockOAuthRepository) RefreshOAuthToken(ctx context.Context, provider string, refreshToken string) (*domain.OAuthToken, error) {
	args := m.Called(ctx, provider, refreshToken)
	if args.Get(0) != nil {
		return args.Get(0).(*domain.OAuthToken), args.Error(1)
	}
	return nil, args.Error(1)
}

// Mock key repository
type mockKeyRepository struct {
	mock.Mock
}

func (m *mockKeyRepository) GetPrivateKey() ([]byte, error) {
	args := m.Called()
	if args.Get(0) != nil {
		return args.Get(0).([]byte), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockKeyRepository) GetPublicKey() ([]byte, error) {
	args := m.Called()
	if args.Get(0) != nil {
		return args.Get(0).([]byte), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockKeyRepository) GetJWKS() ([]byte, error) {
	args := m.Called()
	if args.Get(0) != nil {
		return args.Get(0).([]byte), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockKeyRepository) RotateKeys() error {
	args := m.Called()
	return args.Error(0)
}

func TestService_GetAuthURL(t *testing.T) {
	ctx := context.Background()

	mockRepo := new(mockRepository)
	mockOAuthRepo := new(mockOAuthRepository)
	mockKeyRepo := new(mockKeyRepository)
	mockTokenDomainService := new(mockTokenDomainService)

	// Create a dummy TokenManager
	testPrivateKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	testPublicKey := &testPrivateKey.PublicKey
	tokenManager := internalAuth.NewTokenManager(testPrivateKey, testPublicKey, "test-issuer", time.Hour)

	svc := &service{
		repo:               mockRepo,
		oauthRepo:          mockOAuthRepo,
		keyRepo:            mockKeyRepo,
		tokenManager:       tokenManager,
		tokenDomainService: mockTokenDomainService,
		logger:             slog.Default(),
		defaultTokenExpiry: 3600, // 1 hour default
	}

	t.Run("successful get auth URL", func(t *testing.T) {
		req := &domain.LoginRequest{
			Provider: "google",
		}

		expectedURL := "https://accounts.google.com/o/oauth2/v2/auth?state=random-state-123"

		// Mock repository calls - verify IsSignUp is false for login
		mockRepo.On("StoreAuthState", ctx, mock.MatchedBy(func(state *domain.AuthState) bool {
			return state.IsSignUp == false && state.Provider == "google"
		})).Return(nil)
		mockOAuthRepo.On("GetAuthURL", "google", mock.AnythingOfType("string"), mock.Anything).Return(expectedURL, nil)

		url, state, err := svc.GetAuthURL(ctx, req)
		assert.NoError(t, err)
		assert.Equal(t, expectedURL, url)
		assert.NotEmpty(t, state)

		mockRepo.AssertExpectations(t)
		mockOAuthRepo.AssertExpectations(t)
	})

	t.Run("invalid provider", func(t *testing.T) {
		req := &domain.LoginRequest{
			Provider: "invalid-provider",
		}

		mockRepo.On("StoreAuthState", ctx, mock.AnythingOfType("*domain.AuthState")).Return(nil)
		mockOAuthRepo.On("GetAuthURL", "invalid-provider", mock.AnythingOfType("string"), mock.Anything).
			Return("", errors.New("unsupported provider"))

		url, state, err := svc.GetAuthURL(ctx, req)
		assert.Error(t, err)
		assert.Empty(t, url)
		assert.Empty(t, state)
	})
}

//nolint:funlen // comprehensive test coverage for sign-up functionality
func TestService_GetAuthURLForSignUp(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("successful get auth URL for sign-up", func(t *testing.T) {
		t.Parallel()

		mockRepo := new(mockRepository)
		mockOAuthRepo := new(mockOAuthRepository)
		mockKeyRepo := new(mockKeyRepository)
		mockTokenDomainService := new(mockTokenDomainService)

		// Create a dummy TokenManager
		testPrivateKey, _ := rsa.GenerateKey(rand.Reader, 2048)
		testPublicKey := &testPrivateKey.PublicKey
		tokenManager := internalAuth.NewTokenManager(testPrivateKey, testPublicKey, "test-issuer", time.Hour)

		svc := &service{
			repo:               mockRepo,
			oauthRepo:          mockOAuthRepo,
			keyRepo:            mockKeyRepo,
			tokenManager:       tokenManager,
			tokenDomainService: mockTokenDomainService,
			logger:             slog.Default(),
			defaultTokenExpiry: 3600, // 1 hour default
		}

		req := &domain.SignUpAuthRequest{
			Provider:            "google",
			CodeChallenge:       "challenge123",
			CodeChallengeMethod: "S256",
		}

		// Mock storing auth state - verify IsSignUp is true for sign-up
		mockRepo.On("StoreAuthState", ctx, mock.MatchedBy(func(state *domain.AuthState) bool {
			return state.Provider == "google" && state.CodeChallenge == "challenge123" && state.IsSignUp == true
		})).Return(nil)

		// Mock getting auth URL
		expectedURL := "https://accounts.google.com/o/oauth2/v2/auth?client_id=123"
		mockOAuthRepo.On("GetAuthURL", "google", mock.AnythingOfType("string"),
			mock.MatchedBy(func(params map[string]string) bool {
				return params["code_challenge"] == "challenge123" && params["code_challenge_method"] == "S256"
			})).Return(expectedURL, nil)

		// Call the method
		url, state, err := svc.GetAuthURLForSignUp(ctx, req)

		// Assertions
		require.NoError(t, err)
		assert.Equal(t, expectedURL, url)
		assert.NotEmpty(t, state)

		mockRepo.AssertExpectations(t)
		mockOAuthRepo.AssertExpectations(t)
	})

	t.Run("successful get auth URL for sign-up without PKCE", func(t *testing.T) {
		t.Parallel()

		mockRepo := new(mockRepository)
		mockOAuthRepo := new(mockOAuthRepository)
		mockKeyRepo := new(mockKeyRepository)
		mockTokenDomainService := new(mockTokenDomainService)

		// Create a dummy TokenManager
		testPrivateKey, _ := rsa.GenerateKey(rand.Reader, 2048)
		testPublicKey := &testPrivateKey.PublicKey
		tokenManager := internalAuth.NewTokenManager(testPrivateKey, testPublicKey, "test-issuer", time.Hour)

		svc := &service{
			repo:               mockRepo,
			oauthRepo:          mockOAuthRepo,
			keyRepo:            mockKeyRepo,
			tokenManager:       tokenManager,
			tokenDomainService: mockTokenDomainService,
			logger:             slog.Default(),
			defaultTokenExpiry: 3600, // 1 hour default
		}

		req := &domain.SignUpAuthRequest{
			Provider: "github",
		}

		// Mock storing auth state - verify IsSignUp is true for sign-up
		mockRepo.On("StoreAuthState", ctx, mock.MatchedBy(func(state *domain.AuthState) bool {
			return state.Provider == "github" && state.CodeChallenge == "" && state.IsSignUp == true
		})).Return(nil)

		// Mock getting auth URL
		expectedURL := "https://github.com/login/oauth/authorize?client_id=456"
		mockOAuthRepo.On("GetAuthURL", "github", mock.AnythingOfType("string"),
			mock.MatchedBy(func(params map[string]string) bool {
				return len(params) == 0 || (params["code_challenge"] == "" && params["code_challenge_method"] == "")
			})).Return(expectedURL, nil)

		// Call the method
		url, state, err := svc.GetAuthURLForSignUp(ctx, req)

		// Assertions
		require.NoError(t, err)
		assert.Equal(t, expectedURL, url)
		assert.NotEmpty(t, state)

		mockRepo.AssertExpectations(t)
		mockOAuthRepo.AssertExpectations(t)
	})

	t.Run("invalid provider", func(t *testing.T) {
		t.Parallel()

		mockRepo := new(mockRepository)
		mockOAuthRepo := new(mockOAuthRepository)
		mockKeyRepo := new(mockKeyRepository)
		mockTokenDomainService := new(mockTokenDomainService)

		// Create a dummy TokenManager
		testPrivateKey, _ := rsa.GenerateKey(rand.Reader, 2048)
		testPublicKey := &testPrivateKey.PublicKey
		tokenManager := internalAuth.NewTokenManager(testPrivateKey, testPublicKey, "test-issuer", time.Hour)

		svc := &service{
			repo:               mockRepo,
			oauthRepo:          mockOAuthRepo,
			keyRepo:            mockKeyRepo,
			tokenManager:       tokenManager,
			tokenDomainService: mockTokenDomainService,
			logger:             slog.Default(),
			defaultTokenExpiry: 3600, // 1 hour default
		}

		req := &domain.SignUpAuthRequest{
			Provider: "invalid-provider",
		}

		// Mock storing auth state
		mockRepo.On("StoreAuthState", ctx, mock.AnythingOfType("*domain.AuthState")).Return(nil)

		// Mock getting auth URL - returns error for unsupported provider
		mockOAuthRepo.On("GetAuthURL", "invalid-provider", mock.AnythingOfType("string"),
			mock.AnythingOfType("map[string]string")).Return("", domain.ErrUnsupportedProvider)

		// Call the method
		url, state, err := svc.GetAuthURLForSignUp(ctx, req)

		// Assertions
		require.Error(t, err)
		assert.Empty(t, url)
		assert.Empty(t, state)
	})
}

//nolint:funlen // TODO: Refactor this test to reduce complexity
func TestService_HandleCallback(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	svc, mockRepo, mockOAuthRepo, mockTokenDomainService, mockSessionManager := setupTestService()

	t.Run("successful callback - new user", func(t *testing.T) {
		req := &domain.CallbackRequest{
			Code:  "auth-code-123",
			State: "valid-state-123",
		}
		clientIP := "192.168.1.1"
		userAgent := "Mozilla/5.0"

		authState := &domain.AuthState{
			State:     "valid-state-123",
			Provider:  "google",
			ExpiresAt: time.Now().Add(10 * time.Minute), // Valid for 10 minutes
		}

		oauthToken := &domain.OAuthToken{
			AccessToken:  "access-token-123",
			RefreshToken: "refresh-token-123",
		}

		userInfo := &domain.UserInfo{
			ID:       "google-123",
			Email:    "user@example.com",
			Name:     "Test User",
			Provider: "google",
		}

		// No longer need to generate private key for test since TokenManager handles it

		// Mock the flow
		mockRepo.On("GetAuthState", ctx, "valid-state-123").Return(authState, nil).Once()
		mockRepo.On("DeleteAuthState", ctx, "valid-state-123").Return(nil)
		mockOAuthRepo.On("ExchangeCode", ctx, "google", "auth-code-123").Return(oauthToken, nil)
		mockOAuthRepo.On("GetUserInfo", ctx, "google", oauthToken).Return(userInfo, nil)

		// User doesn't exist yet
		mockRepo.On("GetUserByExternalID", ctx, "google-123", "google").Return(nil, domain.ErrUserNotFound)
		mockRepo.On("CreateUser", ctx, mock.AnythingOfType("*domain.User")).Return(nil)
		mockRepo.On("CreateSecurityEvent", ctx, mock.AnythingOfType("*domain.SecurityEvent")).Return(nil).Times(2)

		// Generate tokens
		mockRepo.On("GetUserOrganizations", ctx, mock.AnythingOfType("string")).Return([]string{}, nil)

		// Hash token for session creation
		mockRepo.On("HashToken", mock.AnythingOfType("string")).Return(
			"1234567890123456789012345678901234567890123456789012345678901234", // 64 chars
			"abcdefabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcd", // 64 chars
			nil).Maybe()

		// Session manager CreateSession mock
		mockSessionManager.On("CreateSession", ctx, mock.AnythingOfType("string"),
			mock.AnythingOfType("string")).Return(nil)

		// Create session
		mockRepo.On("CreateSession", ctx, mock.AnythingOfType("*domain.Session")).Return(nil)
		mockTokenDomainService.On("CreateSession", mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(&domain.Session{
			ID:     "session-123",
			UserID: "user-123",
		}, nil)

		response, err := svc.HandleCallback(ctx, req, clientIP, userAgent)
		assert.NoError(t, err)
		assert.NotNil(t, response)
		assert.NotNil(t, response.User)
		assert.Equal(t, "user@example.com", response.User.Email)
		assert.NotEmpty(t, response.AccessToken)
		assert.NotEmpty(t, response.RefreshToken)

		mockRepo.AssertExpectations(t)
		mockOAuthRepo.AssertExpectations(t)
	})

	t.Run("database error when getting user by external ID", func(t *testing.T) {
		req := &domain.CallbackRequest{
			Code:  "auth-code-456",
			State: "valid-state-456",
		}
		clientIP := "192.168.1.1"
		userAgent := "Mozilla/5.0"

		authState := &domain.AuthState{
			State:     "valid-state-456",
			Provider:  "google",
			ExpiresAt: time.Now().Add(10 * time.Minute),
		}

		oauthToken := &domain.OAuthToken{
			AccessToken:  "access-token-456",
			RefreshToken: "refresh-token-456",
		}

		userInfo := &domain.UserInfo{
			ID:       "google-456",
			Email:    "existing@example.com",
			Name:     "Existing User",
			Provider: "google",
		}

		// Mock the flow
		mockRepo.On("GetAuthState", ctx, "valid-state-456").Return(authState, nil).Once()
		mockOAuthRepo.On("ExchangeCode", ctx, "google", "auth-code-456").Return(oauthToken, nil)
		mockOAuthRepo.On("GetUserInfo", ctx, "google", oauthToken).Return(userInfo, nil)

		// Database error occurs when getting user (not a "user not found" error)
		mockRepo.On("GetUserByExternalID", ctx, "google-456", "google").Return(nil, errors.New("database connection error"))

		// Should not create security event or create user since it's a database error
		// mockRepo.On("CreateSecurityEvent", ...) is NOT called
		// mockRepo.On("CreateUser", ...) is NOT called

		response, err := svc.HandleCallback(ctx, req, clientIP, userAgent)
		assert.Error(t, err)
		assert.Nil(t, response)
		assert.Contains(t, err.Error(), "failed to get user info")

		mockRepo.AssertExpectations(t)
		mockOAuthRepo.AssertExpectations(t)
	})

	t.Run("invalid state", func(t *testing.T) {
		req := &domain.CallbackRequest{
			Code:  "auth-code-789",
			State: "invalid-state",
		}
		clientIP := "192.168.1.1"
		userAgent := "Mozilla/5.0"

		mockRepo.On("GetAuthState", ctx, "invalid-state").Return(nil, domain.ErrAuthStateNotFound).Once()

		response, err := svc.HandleCallback(ctx, req, clientIP, userAgent)
		assert.Error(t, err)
		assert.Nil(t, response)
		assert.Contains(t, err.Error(), "auth state not found")

		mockRepo.AssertExpectations(t)
	})

	t.Run("successful callback with PKCE", func(t *testing.T) {
		req := &domain.CallbackRequest{
			Code:         "auth-code-pkce",
			State:        "valid-state-pkce",
			CodeVerifier: "verifier-123",
		}
		clientIP := "192.168.1.1"
		userAgent := "Mozilla/5.0"

		authState := &domain.AuthState{
			State:         "valid-state-pkce",
			Provider:      "google",
			CodeChallenge: "Ds3NpaREu9I2EYq6l0l3ZkFyv_Gt5O4EpGD6cZlY0Kg", // sha256("verifier-123")
			ExpiresAt:     time.Now().Add(10 * time.Minute),
		}

		oauthToken := &domain.OAuthToken{AccessToken: "access-token-pkce"}
		userInfo := &domain.UserInfo{ID: "google-pkce", Email: "pkce@example.com", Name: "PKCE User"}

		mockRepo.On("GetAuthState", ctx, req.State).Return(authState, nil).Once()
		mockOAuthRepo.On("ExchangeCode", ctx, "google", req.Code).Return(oauthToken, nil).Once()
		mockOAuthRepo.On("GetUserInfo", ctx, "google", oauthToken).Return(userInfo, nil).Once()
		mockRepo.On("GetUserByExternalID", ctx, userInfo.ID, "google").Return(nil, domain.ErrUserNotFound).Once()
		mockRepo.On("CreateUser", ctx, mock.AnythingOfType("*domain.User")).Return(nil).Once()
		mockRepo.On("GetUserOrganizations", ctx, mock.AnythingOfType("string")).Return([]string{}, nil).Once()
		mockRepo.On("HashToken", mock.AnythingOfType("string")).Return("hashed-token", "salt", nil).Once()
		mockRepo.On("CreateSession", ctx, mock.AnythingOfType("*domain.Session")).Return(nil).Once()
		mockRepo.On("DeleteAuthState", ctx, req.State).Return(nil).Once()
		mockRepo.On("CreateSecurityEvent", ctx, mock.AnythingOfType("*domain.SecurityEvent")).Return(nil).Twice()

		// No longer need to mock VerifyPKCE directly as it's now an internal helper

		response, err := svc.HandleCallback(ctx, req, clientIP, userAgent)
		assert.NoError(t, err)
		assert.NotNil(t, response)
	})

	t.Run("PKCE verification failed - wrong verifier", func(t *testing.T) {
		req := &domain.CallbackRequest{
			Code:         "auth-code-pkce-fail",
			State:        "state-pkce-fail",
			CodeVerifier: "wrong-verifier",
		}
		clientIP := "192.168.1.1"
		userAgent := "Mozilla/5.0"

		authState := &domain.AuthState{
			State:         "state-pkce-fail",
			Provider:      "google",
			CodeChallenge: "m6M3_w_n222e5N7g-aA4a-AYxEK299lF-iQ2pE79gA4", // sha256("verifier-123")
			ExpiresAt:     time.Now().Add(10 * time.Minute),
		}

		mockRepo.On("GetAuthState", ctx, req.State).Return(authState, nil).Once()
		mockRepo.On("CreateSecurityEvent", ctx, mock.AnythingOfType("*domain.SecurityEvent")).Return(nil).Once()

		response, err := svc.HandleCallback(ctx, req, clientIP, userAgent)
		assert.Error(t, err)
		assert.Nil(t, response)
		assert.Contains(t, err.Error(), "PKCE verification failed")
	})

	t.Run("PKCE error - verifier missing", func(t *testing.T) {
		req := &domain.CallbackRequest{
			Code:         "auth-code-pkce-missing",
			State:        "state-pkce-missing",
			CodeVerifier: "", // Verifier is missing
		}
		clientIP := "192.168.1.1"
		userAgent := "Mozilla/5.0"

		authState := &domain.AuthState{
			State:         "state-pkce-missing",
			Provider:      "google",
			CodeChallenge: "m6M3_w_n222e5N7g-aA4a-AYxEK299lF-iQ2pE79gA4",
			ExpiresAt:     time.Now().Add(10 * time.Minute),
		}

		mockRepo.On("GetAuthState", ctx, req.State).Return(authState, nil).Once()
		mockRepo.On("CreateSecurityEvent", ctx, mock.AnythingOfType("*domain.SecurityEvent")).Return(nil).Once()

		response, err := svc.HandleCallback(ctx, req, clientIP, userAgent)
		assert.Error(t, err)
		assert.Nil(t, response)
		assert.Contains(t, err.Error(), "code_verifier is required")
	})
}

func TestService_RefreshToken(t *testing.T) {
	ctx := context.Background()

	mockRepo := new(mockRepository)
	mockKeyRepo := new(mockKeyRepository)
	mockTokenDomainService := new(mockTokenDomainService)
	mockSessionManager := new(mockSessionManager)

	// Create a dummy TokenManager
	testPrivateKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	testPublicKey := &testPrivateKey.PublicKey
	tokenManager := internalAuth.NewTokenManager(testPrivateKey, testPublicKey, "test-issuer", time.Hour)

	svc := &service{
		repo:               mockRepo,
		keyRepo:            mockKeyRepo,
		tokenManager:       tokenManager,
		tokenDomainService: mockTokenDomainService,
		sessionManager:     mockSessionManager,
		logger:             slog.Default(),
		defaultTokenExpiry: 3600, // 1 hour default
	}

	// No longer need to generate private key for test since TokenManager handles it

	t.Run("successful refresh", func(t *testing.T) {
		selector := "123456789-123456789-123456789-s"
		verifier := "123456789-123456789-123456789-1v"
		refreshToken := selector + "." + verifier
		clientIP := "192.168.1.1"
		userAgent := "Mozilla/5.0"

		user := &domain.User{
			ID:          "user-123",
			Email:       "user@example.com",
			DisplayName: "Test User",
			Provider:    "google",
		}

		// Create session with mock hash and salt values (for testing purposes)
		// hashedToken := selector + "b8c8f5e6d4a7c2e9f1b3d6e8a5c7f9e2d4b6e8f1c3e5a7b9d2f4e6c8a1b3d5e712"
		hashedToken := refreshToken
		salt := "a1b2c3d4e5f67890123456789012345678901234567890123456789012345678"

		session := &domain.Session{
			ID:           uuid.New().String(),
			UserID:       "user-123",
			RefreshToken: hashedToken,
			Salt:         salt,
			ExpiresAt:    time.Now().Add(24 * time.Hour),
		}

		// Expected Claims from TokenDomainService
		expectedClaims := &domain.Claims{
			UserID:    "user-123",
			Email:     "user@example.com",
			Name:      "Test User",
			Provider:  "google",
			SessionID: session.ID,
		}

		// Mock repository calls - the VerifyToken should return true for our test token
		mockRepo.On("IsRefreshTokenBlacklisted", ctx, refreshToken).Return(false, nil)
		mockRepo.On("GetSessionByRefreshTokenSelector", ctx, selector).Return(session, nil)
		mockRepo.On("VerifyToken", verifier, hashedToken, salt).Return(true)
		mockRepo.On("GetUser", ctx, "user-123").Return(user, nil)
		mockTokenDomainService.On("RefreshToken", ctx, session, user).Return(expectedClaims, nil)
		mockRepo.On("BlacklistRefreshToken", ctx, refreshToken, session.ExpiresAt).Return(nil)

		// Hash token for session update with new refresh token
		mockRepo.On("HashToken", mock.AnythingOfType("string")).Return(
			refreshToken, // "9876543210987654321098765432109876543210987654321098765432109876", // 64 chars
			salt,         // "efghefghefghefghefghefghefghefghefghefghefghefghefghefghefghefgh", // 64 chars
			nil)

		// Mock SessionManager calls
		mockSessionManager.On("DeleteSession", ctx, "user-123", session.ID).Return(nil)
		mockSessionManager.On("CreateSession", ctx, "user-123", mock.AnythingOfType("string")).Return(nil)

		// Block old session and create new session (new behavior)
		mockRepo.On("BlockSession", ctx, session.ID, mock.AnythingOfType("time.Time")).Return(nil)
		mockRepo.On("CreateSession", ctx, mock.AnythingOfType("*domain.Session")).Return(nil)
		mockRepo.On("DeleteSession", ctx, session.ID).Return(nil)
		mockRepo.On("CreateSecurityEvent", ctx, mock.AnythingOfType("*domain.SecurityEvent")).Return(nil)

		response, err := svc.RefreshToken(ctx, refreshToken, clientIP, userAgent)
		assert.NoError(t, err)
		assert.NotNil(t, response)
		assert.NotEmpty(t, response.AccessToken)
		assert.NotEmpty(t, response.RefreshToken)
		assert.Equal(t, "Bearer", response.TokenType)
		assert.Equal(t, 3600, response.ExpiresIn)

		mockRepo.AssertExpectations(t)
	})

	t.Run("blacklisted token", func(t *testing.T) {
		refreshToken := "blacklisted-token"
		clientIP := "192.168.1.1"
		userAgent := "Mozilla/5.0"

		mockRepo.On("IsRefreshTokenBlacklisted", ctx, refreshToken).Return(true, nil)

		response, err := svc.RefreshToken(ctx, refreshToken, clientIP, userAgent)
		assert.Error(t, err)
		assert.Nil(t, response)
		assert.Contains(t, err.Error(), "invalid")

		mockRepo.AssertExpectations(t)
	})

	t.Run("expired session", func(t *testing.T) {
		selector := "expired-unique-123456789-12345s"
		verifier := "expired-unique-123456789-123456v"
		refreshToken := selector + "." + verifier
		clientIP := "192.168.1.1"
		userAgent := "Mozilla/5.0"

		// Create an expired session
		expiredSession := &domain.Session{
		 ID:                   "expired-session-id",
		 UserID:               "user-123",
		 RefreshToken:         refreshToken,
		 RefreshTokenSelector: selector,
		 Salt:                 "salt1234567890abcdef1234567890abcdef1234567890abcdef1234567890ab",
		 ExpiresAt:            time.Now().Add(-24 * time.Hour),
		}

		mockRepo.On("IsRefreshTokenBlacklisted", ctx, refreshToken).Return(false, nil).Once()
		mockRepo.On("GetSessionByRefreshTokenSelector", ctx, selector).Return(expiredSession, nil).Once()
		mockRepo.On("VerifyToken", verifier, expiredSession.RefreshToken, expiredSession.Salt).Return(true).Once()

		// Note: GetUser and TokenDomainService.RefreshToken are NOT called because session expires before those steps

		response, err := svc.RefreshToken(ctx, refreshToken, clientIP, userAgent)
		assert.Error(t, err)
		assert.Nil(t, response)
		assert.Contains(t, err.Error(), "session has expired")

		mockRepo.AssertExpectations(t)
		// Note: TokenDomainService.RefreshToken is NOT called because session expires before domain validation
	})
}

func TestService_RevokeSession(t *testing.T) {
	ctx := context.Background()
	svc, mockRepo, _, _, mockSessionManager := setupTestService()

	t.Run("successful revoke", func(t *testing.T) {
		userID := "user-123"
		sessionID := "session-123"
		refreshToken := "refresh-token-123"

		session := &domain.Session{
			ID:           sessionID,
			UserID:       userID,
			RefreshToken: refreshToken,
			ExpiresAt:    time.Now().Add(24 * time.Hour),
		}

		// Mock repository calls
		mockRepo.On("GetSession", ctx, sessionID).Return(session, nil)
		mockRepo.On("BlacklistRefreshToken", ctx, refreshToken, session.ExpiresAt).Return(nil)
		mockSessionManager.On("DeleteSession", ctx, userID, sessionID).Return(nil)
		mockRepo.On("DeleteSession", ctx, sessionID).Return(nil)
		mockRepo.On("CreateSecurityEvent", ctx, mock.AnythingOfType("*domain.SecurityEvent")).Return(nil)

		err := svc.RevokeSession(ctx, userID, sessionID)
		assert.NoError(t, err)

		mockRepo.AssertExpectations(t)
	})

	t.Run("session not found", func(t *testing.T) {
		userID := "user-456"
		sessionID := "non-existent"

		mockRepo.On("GetSession", ctx, sessionID).Return(nil, domain.ErrSessionNotFound)

		err := svc.RevokeSession(ctx, userID, sessionID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")

		mockRepo.AssertExpectations(t)
	})

	t.Run("user mismatch", func(t *testing.T) {
		userID := "user-789"
		sessionID := "session-789"

		session := &domain.Session{
			ID:     sessionID,
			UserID: "different-user",
		}

		mockRepo.On("GetSession", ctx, sessionID).Return(session, nil)

		err := svc.RevokeSession(ctx, userID, sessionID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unauthorized")

		mockRepo.AssertExpectations(t)
	})
}

// Test for generateTokenPair with sessionID parameter
func TestService_generateTokenPairWithSessionID(t *testing.T) {
	mockRepo := &mockRepository{}
	logger := slog.Default()

	// Create a test RSA key pair
	testPrivateKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	testPublicKey := &testPrivateKey.PublicKey
	tokenManager := internalAuth.NewTokenManager(testPrivateKey, testPublicKey, "test-issuer", time.Hour)
	tokenDomainService := &mockTokenDomainService{}

	svc := &service{
		repo:               mockRepo,
		tokenManager:       tokenManager,
		tokenDomainService: tokenDomainService,
		logger:             logger,
	}

	ctx := context.Background()
	user := &domain.User{
		ID:          "user-123",
		Email:       "test@example.com",
		DisplayName: "Test User",
		Provider:    "google",
	}

	sessionID := "session-123"
	orgIDs := []string{"org-1", "org-2"}
	mockRepo.On("GetUserOrganizations", ctx, user.ID).Return(orgIDs, nil)

	// Test generateTokenPair with sessionID parameter
	tokenPair, err := svc.generateTokenPair(ctx, user, sessionID)
	assert.NoError(t, err)
	assert.NotNil(t, tokenPair)
	assert.NotEmpty(t, tokenPair.AccessToken)
	assert.NotEmpty(t, tokenPair.RefreshToken)

	// Verify the token contains the session information
	// This test currently passes but we need to verify SessionID is properly set
	claims, err := tokenManager.ValidateToken(tokenPair.AccessToken)
	assert.NoError(t, err)
	assert.NotNil(t, claims)

	// Extract the actual token and decode it to verify SessionID is included
	// Since our implementation now uses domain.Claims with SessionID, it should be present
	token, err := jwt.ParseWithClaims(tokenPair.AccessToken, &domain.Claims{}, func(token *jwt.Token) (interface{}, error) {
		// For this test, we'll accept any signing method since we're testing the claims structure
		return []byte("dummy-key"), nil // This will fail validation but we only care about parsing structure
	})

	// Even though validation fails due to dummy key, we can still examine the claims
	if token != nil {
		if domainClaims, ok := token.Claims.(*domain.Claims); ok {
			// This should now contain a valid SessionID after our fix
			assert.NotEmpty(t, domainClaims.SessionID, "SessionID should now be included in the token after our fix")
		}
	}

	mockRepo.AssertExpectations(t)
}

// Test OAuth flow to verify sessionID is included in token
func TestService_OAuthFlowWithSessionID(t *testing.T) {
	mockRepo := &mockRepository{}
	mockRepo.On("HashToken", mock.AnythingOfType("string")).Return(
		"1234567890123456789012345678901234567890123456789012345678901234", // 64 chars
		"abcdefabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcd", // 64 chars
		nil,
	).Maybe()
	mockOAuthRepo := &mockOAuthRepository{}
	mockKeyRepo := &mockKeyRepository{}
	logger := slog.Default()

	// Create a test RSA key pair
	testPrivateKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	testPublicKey := &testPrivateKey.PublicKey
	tokenManager := internalAuth.NewTokenManager(testPrivateKey, testPublicKey, "test-issuer", time.Hour)
	tokenDomainService := &mockTokenDomainService{}

	mockSessionManager := &mockSessionManager{}

	svc := &service{
		repo:               mockRepo,
		oauthRepo:          mockOAuthRepo,
		keyRepo:            mockKeyRepo,
		tokenManager:       tokenManager,
		tokenDomainService: tokenDomainService,
		sessionManager:     mockSessionManager,
		logger:             logger,
		defaultTokenExpiry: 3600,
	}

	// Mock tokenDomainService.CreateSession
	tokenDomainService.On("CreateSession", mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(&domain.Session{
		ID:     "session-123",
		UserID: "user-123",
	}, nil)

	ctx := context.Background()
	req := &domain.CallbackRequest{
		Code:  "auth-code-123",
		State: "state-123",
	}
	clientIP := "127.0.0.1"
	userAgent := "test-agent"

	// Mock auth state
	authState := &domain.AuthState{
		State:         "state-123",
		Provider:      "google",
		RedirectURL:   "http://localhost:3000/callback",
		CodeChallenge: "",
		ExpiresAt:     time.Now().Add(10 * time.Minute),
	}

	// Set up mock expectations
	mockRepo.On("GetAuthState", ctx, req.State).Return(authState, nil)
	mockOAuthRepo.On("ExchangeCode", ctx, "google", req.Code).Return(&domain.OAuthToken{AccessToken: "access-token"}, nil)
	mockOAuthRepo.On("GetUserInfo", ctx, "google", mock.AnythingOfType("*domain.OAuthToken")).Return(&domain.UserInfo{
		ID:       "google-123",
		Email:    "test@example.com",
		Name:     "Test User",
		Provider: "google",
	}, nil)
	mockRepo.On("GetUserByExternalID", ctx, "google-123", "google").Return(nil, domain.ErrUserNotFound)
	mockRepo.On("CreateUser", ctx, mock.AnythingOfType("*domain.User")).Return(nil)
	mockRepo.On("GetUserOrganizations", ctx, mock.AnythingOfType("string")).Return([]string{"org-1"}, nil)
	mockSessionManager.On("CreateSession", ctx, mock.AnythingOfType("string"),
		mock.AnythingOfType("string")).Return(nil)
	mockRepo.On("CreateSession", ctx, mock.AnythingOfType("*domain.Session")).Return(nil)
	mockRepo.On("DeleteAuthState", ctx, req.State).Return(nil)
	mockRepo.On("CreateSecurityEvent", ctx, mock.AnythingOfType("*domain.SecurityEvent")).Return(nil)

	// Execute OAuth callback
	authResponse, err := svc.HandleCallback(ctx, req, clientIP, userAgent)
	assert.NoError(t, err)
	assert.NotNil(t, authResponse)
	assert.NotEmpty(t, authResponse.AccessToken)

	// Validate that the token contains session information
	// This test currently passes but we need to verify SessionID is properly set
	claims, err := tokenManager.ValidateToken(authResponse.AccessToken)
	assert.NoError(t, err)
	assert.NotNil(t, claims)

	// Extract the actual token and decode it to verify SessionID is included
	// Since our implementation now uses domain.Claims with SessionID, it should be present
	token, err := jwt.ParseWithClaims(authResponse.AccessToken, &domain.Claims{}, func(token *jwt.Token) (interface{}, error) {
		// For this test, we'll accept any signing method since we're testing the claims structure
		return []byte("dummy-key"), nil // This will fail validation but we only care about parsing structure
	})

	// Even though validation fails due to dummy key, we can still examine the claims
	if token != nil {
		if domainClaims, ok := token.Claims.(*domain.Claims); ok {
			// This should now contain a valid SessionID after our fix
			assert.NotEmpty(t, domainClaims.SessionID, "SessionID should now be included in the token after our fix")
		}
	}

	mockRepo.AssertExpectations(t)
	mockOAuthRepo.AssertExpectations(t)
}

func TestService_RefreshToken_OptimizedLookup(t *testing.T) {
	ctx := context.Background()

	mockRepo := new(mockRepository)
	mockKeyRepo := new(mockKeyRepository)
	mockTokenDomainService := new(mockTokenDomainService)
	mockSessionManager := new(mockSessionManager)

	// Create a dummy TokenManager
	testPrivateKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	testPublicKey := &testPrivateKey.PublicKey
	tokenManager := internalAuth.NewTokenManager(testPrivateKey, testPublicKey, "test-issuer", time.Hour)

	svc := &service{
		repo:               mockRepo,
		keyRepo:            mockKeyRepo,
		tokenManager:       tokenManager,
		tokenDomainService: mockTokenDomainService,
		sessionManager:     mockSessionManager,
		logger:             slog.Default(),
		defaultTokenExpiry: 3600, // 1 hour default
	}

	t.Run("optimized refresh token lookup with selector", func(t *testing.T) {
		// Test the new optimized implementation using selector.verifier pattern
		// Use proper base64 encoded values like the real implementation
		selector := "opt-unique-YWJjZGVmZ2hpams1Njc4"  // base64 encoded
		verifier := "opt-unique-eHl6OTg3NjU0MzIxMDk4N" // base64 encoded (43+ chars)
		refreshToken := selector + "." + verifier      // selector.verifier format
		clientIP := "192.168.1.1"
		userAgent := "Mozilla/5.0"

		user := &domain.User{
			ID:          "user-123",
			Email:       "user@example.com",
			DisplayName: "Test User",
			Provider:    "google",
		}

		// Create session with selector and hashed verifier
		salt := "a1b2c3d4e5f67890123456789012345678901234567890123456789012345678" // 64 chars

		session := &domain.Session{
			ID:                   uuid.New().String(),
			UserID:               "user-123",
			RefreshToken:         refreshToken,
			RefreshTokenSelector: selector,
			Salt:                 salt,
			ExpiresAt:            time.Now().Add(24 * time.Hour),
		}

		// Expected Claims from TokenDomainService
		expectedClaims := &domain.Claims{
			UserID:    "user-123",
			Email:     "user@example.com",
			Name:      "Test User",
			Provider:  "google",
			SessionID: session.ID,
		}

		// Mock repository calls - NEW optimized path
		mockRepo.On("IsRefreshTokenBlacklisted", ctx, refreshToken).Return(false, nil).Once()
		// Use new GetSessionByRefreshTokenSelector instead of GetAllActiveSessions
		mockRepo.On("GetSessionByRefreshTokenSelector", ctx, selector).Return(session, nil).Once()
		mockRepo.On("VerifyToken", verifier, refreshToken, salt).Return(true).Once()
		mockRepo.On("GetUser", ctx, "user-123").Return(user, nil).Once()
		mockTokenDomainService.On("RefreshToken", ctx, session, user).Return(expectedClaims, nil).Once()
		mockRepo.On("BlacklistRefreshToken", ctx, refreshToken, session.ExpiresAt).Return(nil).Once()

		// Hash token for session update with new refresh token
		mockRepo.On("HashToken", mock.AnythingOfType("string")).Return(
			refreshToken, // "9876543210987654321098765432109876543210987654321098765432109876", // 64 chars
			salt,         // "efghefghefghefghefghefghefghefghefghefghefghefghefghefghefghefgh", // 64 chars
			nil).Once()

		// Mock SessionManager calls
		mockSessionManager.On("DeleteSession", ctx, "user-123", session.ID).Return(nil)
		mockSessionManager.On("CreateSession", ctx, "user-123", mock.AnythingOfType("string")).Return(nil)

		mockRepo.On("CreateSecurityEvent", ctx, mock.AnythingOfType("*domain.SecurityEvent")).Return(nil).Once()
		mockRepo.On("BlockSession", ctx, session.ID, mock.AnythingOfType("time.Time")).Return(nil)
		mockRepo.On("CreateSession", ctx, mock.AnythingOfType("*domain.Session")).Return(nil)
		mockRepo.On("DeleteSession", ctx, session.ID).Return(nil)

		response, err := svc.RefreshToken(ctx, refreshToken, clientIP, userAgent)
		if err != nil {
			t.Logf("Error details: %v", err)
		}
		assert.NoError(t, err)
		if assert.NotNil(t, response) {
			assert.NotEmpty(t, response.AccessToken)
			assert.NotEmpty(t, response.RefreshToken)
			assert.Equal(t, "Bearer", response.TokenType)
			assert.Equal(t, 3600, response.ExpiresIn)
		}

		// Verify that GetAllActiveSessions was NOT called (optimization)
		mockRepo.AssertNotCalled(t, "GetAllActiveSessions")
		mockRepo.AssertExpectations(t)
	})

	t.Run("invalid refresh token format for optimized lookup", func(t *testing.T) {
		// Test with invalid token format (no selector.verifier structure)
		refreshToken := "invalid-token-without-selector"
		clientIP := "192.168.1.1"
		userAgent := "Mozilla/5.0"

		mockRepo.On("IsRefreshTokenBlacklisted", ctx, refreshToken).Return(false, nil)
		// For invalid format, getSessionByRefreshToken should return format error immediately
		// No need to mock GetAllActiveSessions as it won't be called

		response, err := svc.RefreshToken(ctx, refreshToken, clientIP, userAgent)
		assert.Error(t, err)
		assert.Nil(t, response)
		assert.Contains(t, err.Error(), "invalid refresh token format")

		mockRepo.AssertExpectations(t)
	})
}
