package auth_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hexabase/hexabase-ai/api/internal/auth/domain"
	"github.com/hexabase/hexabase-ai/api/internal/auth/handler"
	"github.com/hexabase/hexabase-ai/api/internal/shared/infrastructure/server/ogen"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// Mock service for integration testing
type mockAuthService struct {
	mock.Mock
}

func (m *mockAuthService) GetAuthURL(ctx context.Context, req *domain.LoginRequest) (string, string, error) {
	args := m.Called(ctx, req)
	return args.String(0), args.String(1), args.Error(2)
}

func (m *mockAuthService) GetAuthURLForSignUp(
	ctx context.Context,
	req *domain.SignUpAuthRequest,
) (string, string, error) {
	args := m.Called(ctx, req)
	return args.String(0), args.String(1), args.Error(2)
}

func (m *mockAuthService) HandleCallback(
	ctx context.Context,
	req *domain.CallbackRequest,
	clientIP, userAgent string,
) (*domain.AuthResponse, error) {
	args := m.Called(ctx, req, clientIP, userAgent)
	if args.Get(0) != nil {
		if v, ok := args.Get(0).(*domain.AuthResponse); ok {
			return v, args.Error(1)
		}
	}

	return nil, args.Error(1)
}

func (m *mockAuthService) RefreshToken(
	ctx context.Context,
	refreshToken, clientIP, userAgent string,
) (*domain.TokenPair, error) {
	args := m.Called(ctx, refreshToken, clientIP, userAgent)
	if args.Get(0) != nil {
		if v, ok := args.Get(0).(*domain.TokenPair); ok {
			return v, args.Error(1)
		}
	}

	return nil, args.Error(1)
}

func (m *mockAuthService) RevokeRefreshToken(ctx context.Context, refreshToken string) error {
	args := m.Called(ctx, refreshToken)
	return args.Error(0)
}

func (m *mockAuthService) ValidateAccessToken(ctx context.Context, token string) (*domain.Claims, error) {
	args := m.Called(ctx, token)
	if args.Get(0) != nil {
		if v, ok := args.Get(0).(*domain.Claims); ok {
			return v, args.Error(1)
		}
	}

	return nil, args.Error(1)
}

func (m *mockAuthService) CreateSession(
	ctx context.Context,
	sessionID, userID, refreshToken, deviceID, clientIP, userAgent string,
) (*domain.Session, error) {
	args := m.Called(ctx, sessionID, userID, refreshToken, deviceID, clientIP, userAgent)
	if args.Get(0) != nil {
		if v, ok := args.Get(0).(*domain.Session); ok {
			return v, args.Error(1)
		}
	}

	return nil, args.Error(1)
}

func (m *mockAuthService) GetSession(ctx context.Context, sessionID string) (*domain.Session, error) {
	args := m.Called(ctx, sessionID)
	if args.Get(0) != nil {
		if v, ok := args.Get(0).(*domain.Session); ok {
			return v, args.Error(1)
		}
	}

	return nil, args.Error(1)
}

func (m *mockAuthService) GetUserSessions(ctx context.Context, userID string) ([]*domain.SessionInfo, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) != nil {
		if v, ok := args.Get(0).([]*domain.SessionInfo); ok {
			return v, args.Error(1)
		}
	}

	return nil, args.Error(1)
}

func (m *mockAuthService) RevokeSession(ctx context.Context, userID, sessionID string) error {
	args := m.Called(ctx, userID, sessionID)
	return args.Error(0)
}

func (m *mockAuthService) RevokeAllSessions(ctx context.Context, userID string, exceptSessionID string) error {
	args := m.Called(ctx, userID, exceptSessionID)
	return args.Error(0)
}

func (m *mockAuthService) ValidateSession(ctx context.Context, sessionID, clientIP string) error {
	args := m.Called(ctx, sessionID, clientIP)
	return args.Error(0)
}

func (m *mockAuthService) GetUser(ctx context.Context, userID string) (*domain.User, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) != nil {
		if v, ok := args.Get(0).(*domain.User); ok {
			return v, args.Error(1)
		}
	}

	return nil, args.Error(1)
}

func (m *mockAuthService) GetCurrentUser(ctx context.Context, token string) (*domain.User, error) {
	args := m.Called(ctx, token)
	if args.Get(0) != nil {
		if v, ok := args.Get(0).(*domain.User); ok {
			return v, args.Error(1)
		}
	}

	return nil, args.Error(1)
}

func (m *mockAuthService) UpdateUserProfile(
	ctx context.Context,
	userID string,
	updates map[string]interface{},
) (*domain.User, error) {
	args := m.Called(ctx, userID, updates)
	if args.Get(0) != nil {
		if v, ok := args.Get(0).(*domain.User); ok {
			return v, args.Error(1)
		}
	}

	return nil, args.Error(1)
}

func (m *mockAuthService) GenerateWorkspaceToken(ctx context.Context, userID, workspaceID string) (string, error) {
	args := m.Called(ctx, userID, workspaceID)
	return args.String(0), args.Error(1)
}

func (m *mockAuthService) GetJWKS(ctx context.Context) ([]byte, error) {
	args := m.Called(ctx)
	if args.Get(0) != nil {
		if v, ok := args.Get(0).([]byte); ok {
			return v, args.Error(1)
		}
	}

	return nil, args.Error(1)
}

func (m *mockAuthService) StoreAuthState(ctx context.Context, state *domain.AuthState) error {
	args := m.Called(ctx, state)
	return args.Error(0)
}

func (m *mockAuthService) LogSecurityEvent(ctx context.Context, event *domain.SecurityEvent) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}

func (m *mockAuthService) GetSecurityLogs(
	ctx context.Context,
	filter domain.SecurityLogFilter,
) ([]*domain.SecurityEvent, error) {
	args := m.Called(ctx, filter)
	if args.Get(0) != nil {
		if v, ok := args.Get(0).([]*domain.SecurityEvent); ok {
			return v, args.Error(1)
		}
	}

	return nil, args.Error(1)
}

func (m *mockAuthService) GenerateInternalAIOpsToken(
	ctx context.Context,
	userID string,
	orgIDs []string,
	activeWorkspaceID string,
) (string, error) {
	args := m.Called(ctx, userID, orgIDs, activeWorkspaceID)
	return args.String(0), args.Error(1)
}

func (m *mockAuthService) InvalidateSession(ctx context.Context, sessionID string) error {
	args := m.Called(ctx, sessionID)
	return args.Error(0)
}

//nolint:funlen // integration test requires comprehensive scenarios
func TestSignUpEndpointIntegration(t *testing.T) {
	t.Parallel()

	t.Run("successful sign-up with Google", func(t *testing.T) {
		t.Parallel()

		// Setup
		mockService := new(mockAuthService)
		ogenHandler := handler.NewOgenAuthHandler(mockService, slog.Default())

		// Create ogen server
		server, err := ogen.NewServer(ogenHandler)
		require.NoError(t, err)

		// Setup expectations
		expectedAuthURL := "https://accounts.google.com/o/oauth2/v2/auth?client_id=123"
		expectedState := "random-state-123"

		mockService.On("GetAuthURLForSignUp",
			mock.AnythingOfType("*context.valueCtx"),
			mock.MatchedBy(func(req *domain.SignUpAuthRequest) bool {
				return req.Provider == "google" &&
					req.CodeChallenge == "test-challenge" &&
					req.CodeChallengeMethod == "S256"
			})).Return(expectedAuthURL, expectedState, nil)

		// Create request
		reqBody := map[string]interface{}{
			"provider":              "google",
			"code_challenge":        "test-challenge",
			"code_challenge_method": "S256",
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/auth/sign-up/google", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		// Create response recorder
		w := httptest.NewRecorder()

		// Execute request
		server.ServeHTTP(w, req)

		// Assert response
		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}

		err = json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)

		assert.Equal(t, "google", resp["provider"])
		assert.Equal(t, expectedAuthURL, resp["auth_url"])
		assert.Equal(t, expectedState, resp["state"])

		mockService.AssertExpectations(t)
	})

	t.Run("successful sign-up with GitHub without PKCE", func(t *testing.T) {
		t.Parallel()

		// Setup
		mockService := new(mockAuthService)
		ogenHandler := handler.NewOgenAuthHandler(mockService, slog.Default())

		// Create ogen server
		server, err := ogen.NewServer(ogenHandler)
		require.NoError(t, err)

		// Setup expectations
		expectedAuthURL := "https://github.com/login/oauth/authorize?client_id=456"
		expectedState := "random-state-456"

		mockService.On("GetAuthURLForSignUp",
			mock.AnythingOfType("*context.valueCtx"),
			mock.MatchedBy(func(req *domain.SignUpAuthRequest) bool {
				return req.Provider == "github" &&
					req.CodeChallenge == "" &&
					req.CodeChallengeMethod == ""
			})).Return(expectedAuthURL, expectedState, nil)

		// Create request
		reqBody := map[string]interface{}{
			"provider": "github",
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/auth/sign-up/github", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		// Create response recorder
		w := httptest.NewRecorder()

		// Execute request
		server.ServeHTTP(w, req)

		// Assert response
		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}

		err = json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)

		assert.Equal(t, "github", resp["provider"])
		assert.Equal(t, expectedAuthURL, resp["auth_url"])
		assert.Equal(t, expectedState, resp["state"])

		mockService.AssertExpectations(t)
	})
}
