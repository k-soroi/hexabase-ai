package handler_test

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"testing"

	"github.com/hexabase/hexabase-ai/api/internal/auth/domain"
	"github.com/hexabase/hexabase-ai/api/internal/auth/handler"
	"github.com/hexabase/hexabase-ai/api/internal/shared/infrastructure/server/ogen"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const (
	testGoogleAuthURL = "https://accounts.google.com/o/oauth2/v2/auth?client_id=123"
	testGithubAuthURL = "https://github.com/login/oauth/authorize?client_id=456"
)

var errTestInternal = errors.New("some internal error")

// Mock service
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

//nolint:funlen // table-driven test requires comprehensive test cases
func TestOgenAuthHandler_StartAuthSignUp(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	tests := []struct {
		name      string
		req       ogen.OptSignUpRequest
		params    ogen.StartAuthSignUpParams
		setupMock func(*mockAuthService)
		wantErr   bool
		checkResp func(*testing.T, ogen.StartAuthSignUpRes)
	}{
		{
			name: "successful sign-up auth URL with PKCE",
			req: ogen.OptSignUpRequest{
				Value: ogen.SignUpRequest{
					Provider:            "google",
					CodeChallenge:       ogen.NewOptString("challenge123"),
					CodeChallengeMethod: ogen.NewOptString("S256"),
				},
				Set: true,
			},
			params: ogen.StartAuthSignUpParams{
				Provider: ogen.StartAuthSignUpProviderGoogle,
			},
			setupMock: func(m *mockAuthService) {
				authURL := testGoogleAuthURL
				state := "random-state-123"
				m.On("GetAuthURLForSignUp", ctx, mock.MatchedBy(func(req *domain.SignUpAuthRequest) bool {
					return req.Provider == "google" &&
						req.CodeChallenge == "challenge123" &&
						req.CodeChallengeMethod == "S256"
				})).Return(authURL, state, nil)
			},
			wantErr: false,
			checkResp: func(t *testing.T, result ogen.StartAuthSignUpRes) {
				t.Helper()
				resp, ok := result.(*ogen.SignUpResponse)
				require.True(t, ok)
				assert.Equal(t, "google", resp.Provider.Value)
				expectedURL := "https://accounts.google.com/o/oauth2/v2/auth?client_id=123"
				assert.Equal(t, expectedURL, resp.AuthURL.Value.String())
				assert.Equal(t, "random-state-123", resp.State.Value)
			},
		},
		{
			name: "successful sign-up auth URL without PKCE",
			req: ogen.OptSignUpRequest{
				Value: ogen.SignUpRequest{
					Provider: "github",
				},
				Set: true,
			},
			params: ogen.StartAuthSignUpParams{
				Provider: ogen.StartAuthSignUpProviderGithub,
			},
			setupMock: func(m *mockAuthService) {
				authURL := testGithubAuthURL
				state := "random-state-456"
				m.On("GetAuthURLForSignUp", ctx, mock.MatchedBy(func(req *domain.SignUpAuthRequest) bool {
					return req.Provider == "github" &&
						req.CodeChallenge == "" &&
						req.CodeChallengeMethod == ""
				})).Return(authURL, state, nil)
			},
			wantErr: false,
			checkResp: func(t *testing.T, result ogen.StartAuthSignUpRes) {
				t.Helper()
				resp, ok := result.(*ogen.SignUpResponse)
				require.True(t, ok)
				assert.Equal(t, "github", resp.Provider.Value)
				expectedURL := testGithubAuthURL
				assert.Equal(t, expectedURL, resp.AuthURL.Value.String())
				assert.Equal(t, "random-state-456", resp.State.Value)
			},
		},
		{
			name: "successful sign-up auth URL without request body",
			req: ogen.OptSignUpRequest{
				Set: false,
			},
			params: ogen.StartAuthSignUpParams{
				Provider: ogen.StartAuthSignUpProviderGithub,
			},
			setupMock: func(m *mockAuthService) {
				authURL := testGithubAuthURL
				state := "random-state-456"
				m.On("GetAuthURLForSignUp", ctx, mock.MatchedBy(func(req *domain.SignUpAuthRequest) bool {
					return req.Provider == "github" &&
						req.CodeChallenge == "" &&
						req.CodeChallengeMethod == ""
				})).Return(authURL, state, nil)
			},
			wantErr: false,
			checkResp: func(t *testing.T, result ogen.StartAuthSignUpRes) {
				t.Helper()
				resp, ok := result.(*ogen.SignUpResponse)
				require.True(t, ok)
				assert.Equal(t, "github", resp.Provider.Value)
				expectedURL := testGithubAuthURL
				assert.Equal(t, expectedURL, resp.AuthURL.Value.String())
				assert.Equal(t, "random-state-456", resp.State.Value)
			},
		},
		{
			name: "error from service",
			req: ogen.OptSignUpRequest{
				Value: ogen.SignUpRequest{
					Provider: "invalid",
				},
				Set: true,
			},
			params: ogen.StartAuthSignUpParams{
				Provider: ogen.StartAuthSignUpProviderGoogle,
			},
			setupMock: func(m *mockAuthService) {
				m.On("GetAuthURLForSignUp", ctx, mock.AnythingOfType("*domain.SignUpAuthRequest")).
					Return("", "", domain.ErrUnsupportedProvider)
			},
			wantErr: false, // Handler returns error in response, not as error
			checkResp: func(t *testing.T, result ogen.StartAuthSignUpRes) {
				t.Helper()
				errResp, ok := result.(*ogen.SignUpErrorResponseStatusCode)
				require.True(t, ok)
				assert.Equal(t, http.StatusBadRequest, errResp.StatusCode)
				assert.Equal(t, "unsupported authentication provider", errResp.Response.Error.Value)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Setup
			mockService := new(mockAuthService)
			tt.setupMock(mockService)
			authHandler := handler.NewOgenAuthHandler(mockService, slog.Default())

			// Execute
			result, err := authHandler.StartAuthSignUp(ctx, tt.req, tt.params)

			// Assert
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			if tt.checkResp != nil {
				tt.checkResp(t, result)
			}

			mockService.AssertExpectations(t)
		})
	}
}

func TestOgenAuthHandler_NewError(t *testing.T) {
	t.Parallel()

	authHandler := handler.NewOgenAuthHandler(nil, slog.Default())
	ctx := t.Context()

	// Table-driven test following t_wada's recommendations
	tests := []struct {
		name           string
		err            error
		wantStatusCode int
		wantMessage    string
	}{
		{
			name:           "unsupported provider error",
			err:            domain.ErrUnsupportedProvider,
			wantStatusCode: http.StatusBadRequest,
			wantMessage:    "unsupported authentication provider",
		},
		{
			name:           "invalid request error",
			err:            domain.ErrInvalidRequest,
			wantStatusCode: http.StatusBadRequest,
			wantMessage:    "invalid request parameters",
		},
		{
			name:           "generic error should return internal server error",
			err:            errTestInternal,
			wantStatusCode: http.StatusInternalServerError,
			wantMessage:    "authentication service error",
		},
		{
			name:           "nil error should return internal server error",
			err:            nil,
			wantStatusCode: http.StatusInternalServerError,
			wantMessage:    "authentication service error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Act
			result := authHandler.NewError(ctx, tt.err)

			// Assert
			assert.NotNil(t, result)
			assert.Equal(t, tt.wantStatusCode, result.StatusCode)
			assert.Equal(t, tt.wantMessage, result.Response.Error.Value)
		})
	}
}
