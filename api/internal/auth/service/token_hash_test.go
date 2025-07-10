package service

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/hexabase/hexabase-ai/api/internal/auth/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

var (
	ErrTokenTooShortForTest = errors.New("token must be at least 8 characters long for security")
	ErrHashValidationFailed = errors.New("hash generation failed security validation")
)

// mockRepositoryForHashTest is a mock implementation of domain.Repository for testing hash/verify logic.
// It is defined here to be used within the 'service' package for testing private methods.
type mockRepositoryForHashTest struct {
	mock.Mock
}

func (m *mockRepositoryForHashTest) CreateUser(ctx context.Context, user *domain.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *mockRepositoryForHashTest) GetUser(ctx context.Context, userID string) (*domain.User, error) {
	args := m.Called(ctx, userID)
	if u, ok := args.Get(0).(*domain.User); ok {
		return u, args.Error(1)
	}

	return nil, args.Error(1)
}

func (m *mockRepositoryForHashTest) GetUserByExternalID(
	ctx context.Context,
	externalID, provider string,
) (*domain.User, error) {
	args := m.Called(ctx, externalID, provider)
	if u, ok := args.Get(0).(*domain.User); ok {
		return u, args.Error(1)
	}

	return nil, args.Error(1)
}

func (m *mockRepositoryForHashTest) GetUserByEmail(ctx context.Context, email string) (*domain.User, error) {
	args := m.Called(ctx, email)
	if u, ok := args.Get(0).(*domain.User); ok {
		return u, args.Error(1)
	}

	return nil, args.Error(1)
}

func (m *mockRepositoryForHashTest) UpdateUser(ctx context.Context, user *domain.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *mockRepositoryForHashTest) UpdateLastLogin(ctx context.Context, userID string) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func (m *mockRepositoryForHashTest) CreateSession(ctx context.Context, session *domain.Session) error {
	args := m.Called(ctx, session)
	return args.Error(0)
}

func (m *mockRepositoryForHashTest) GetSession(ctx context.Context, sessionID string) (*domain.Session, error) {
	args := m.Called(ctx, sessionID)
	if s, ok := args.Get(0).(*domain.Session); ok {
		return s, args.Error(1)
	}

	return nil, args.Error(1)
}

func (m *mockRepositoryForHashTest) GetSessionByRefreshTokenSelector(
	ctx context.Context,
	selector string,
) (*domain.Session, error) {
	args := m.Called(ctx, selector)
	if s, ok := args.Get(0).(*domain.Session); ok {
		return s, args.Error(1)
	}

	return nil, args.Error(1)
}

func (m *mockRepositoryForHashTest) GetAllActiveSessions(ctx context.Context) ([]*domain.Session, error) {
	args := m.Called(ctx)
	if s, ok := args.Get(0).([]*domain.Session); ok {
		return s, args.Error(1)
	}

	return nil, args.Error(1)
}

func (m *mockRepositoryForHashTest) ListUserSessions(ctx context.Context, userID string) ([]*domain.Session, error) {
	args := m.Called(ctx, userID)
	if s, ok := args.Get(0).([]*domain.Session); ok {
		return s, args.Error(1)
	}

	return nil, args.Error(1)
}

func (m *mockRepositoryForHashTest) UpdateSession(ctx context.Context, session *domain.Session) error {
	args := m.Called(ctx, session)
	return args.Error(0)
}

func (m *mockRepositoryForHashTest) DeleteSession(ctx context.Context, sessionID string) error {
	args := m.Called(ctx, sessionID)
	return args.Error(0)
}

func (m *mockRepositoryForHashTest) DeleteUserSessions(
	ctx context.Context,
	userID string,
	exceptSessionID string,
) error {
	args := m.Called(ctx, userID, exceptSessionID)
	return args.Error(0)
}

func (m *mockRepositoryForHashTest) CleanupExpiredSessions(ctx context.Context, before time.Time) error {
	args := m.Called(ctx, before)
	return args.Error(0)
}

func (m *mockRepositoryForHashTest) StoreAuthState(ctx context.Context, state *domain.AuthState) error {
	args := m.Called(ctx, state)
	return args.Error(0)
}

func (m *mockRepositoryForHashTest) GetAuthState(ctx context.Context, stateValue string) (*domain.AuthState, error) {
	args := m.Called(ctx, stateValue)
	if s, ok := args.Get(0).(*domain.AuthState); ok {
		return s, args.Error(1)
	}

	return nil, args.Error(1)
}

func (m *mockRepositoryForHashTest) DeleteAuthState(ctx context.Context, stateValue string) error {
	args := m.Called(ctx, stateValue)
	return args.Error(0)
}

func (m *mockRepositoryForHashTest) BlacklistRefreshToken(
	ctx context.Context,
	token string,
	expiresAt time.Time,
) error {
	args := m.Called(ctx, token, expiresAt)
	return args.Error(0)
}

func (m *mockRepositoryForHashTest) IsRefreshTokenBlacklisted(ctx context.Context, token string) (bool, error) {
	args := m.Called(ctx, token)
	return args.Bool(0), args.Error(1)
}

func (m *mockRepositoryForHashTest) BlockSession(ctx context.Context, sessionID string, expiresAt time.Time) error {
	args := m.Called(ctx, sessionID, expiresAt)
	return args.Error(0)
}

func (m *mockRepositoryForHashTest) IsSessionBlocked(ctx context.Context, sessionID string) (bool, error) {
	args := m.Called(ctx, sessionID)
	return args.Bool(0), args.Error(1)
}

func (m *mockRepositoryForHashTest) CreateSecurityEvent(ctx context.Context, event *domain.SecurityEvent) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}

func (m *mockRepositoryForHashTest) ListSecurityEvents(
	ctx context.Context,
	filter domain.SecurityLogFilter,
) ([]*domain.SecurityEvent, error) {
	args := m.Called(ctx, filter)
	if e, ok := args.Get(0).([]*domain.SecurityEvent); ok {
		return e, args.Error(1)
	}

	return nil, args.Error(1)
}

func (m *mockRepositoryForHashTest) CleanupOldSecurityEvents(ctx context.Context, before time.Time) error {
	args := m.Called(ctx, before)
	return args.Error(0)
}

func (m *mockRepositoryForHashTest) GetUserOrganizations(ctx context.Context, userID string) ([]string, error) {
	args := m.Called(ctx, userID)
	if o, ok := args.Get(0).([]string); ok {
		return o, args.Error(1)
	}

	return nil, args.Error(1)
}

func (m *mockRepositoryForHashTest) GetUserWorkspaceGroups(
	ctx context.Context,
	userID, workspaceID string,
) ([]string, error) {
	args := m.Called(ctx, userID, workspaceID)
	if g, ok := args.Get(0).([]string); ok {
		return g, args.Error(1)
	}

	return nil, args.Error(1)
}

func (m *mockRepositoryForHashTest) HashToken(token string) (string, string, error) {
	args := m.Called(token)
	return args.String(0), args.String(1), args.Error(2)
}

func (m *mockRepositoryForHashTest) VerifyToken(plainToken, hashedToken, salt string) bool {
	args := m.Called(plainToken, hashedToken, salt)
	return args.Bool(0)
}

// TestService_HashToken_BusinessLogic tests the business validation logic within the hashToken service method.
// It ensures that input validation is performed correctly before delegating to the repository.
func TestService_HashToken_BusinessLogic(t *testing.T) { //nolint:paralleltest,funlen
	t.Run("Business validation - reject empty token", func(t *testing.T) {
		mockRepo := &mockRepositoryForHashTest{}
		s := NewService(mockRepo, nil, nil, nil, nil, nil, slog.Default(), 3600)
		svc, ok := s.(*service)
		require.True(t, ok)

		_, _, err := svc.hashToken("")

		require.Error(t, err)
		require.Error(t, err)
		assert.Equal(t, "token cannot be empty", err.Error())
		// Repository should not be called due to business validation failure
		mockRepo.AssertNotCalled(t, "HashToken")
	})

	t.Run("Business validation - reject short token", func(t *testing.T) {
		mockRepo := &mockRepositoryForHashTest{}
		s := NewService(mockRepo, nil, nil, nil, nil, nil, slog.Default(), 3600)
		svc, ok := s.(*service)
		require.True(t, ok)
		shortToken := "short"

		_, _, err := svc.hashToken(shortToken)

		require.Error(t, err)
		require.Error(t, err)
		assert.Equal(t, "token must be at least 8 characters long for security", err.Error())
		// Repository should not be called due to business validation failure
		mockRepo.AssertNotCalled(t, "HashToken")
	})

	t.Run("Business validation - accept valid token and delegate to repository", func(t *testing.T) {
		mockRepo := &mockRepositoryForHashTest{}
		s := NewService(mockRepo, nil, nil, nil, nil, nil, slog.Default(), 3600)
		svc, ok := s.(*service)
		require.True(t, ok)
		validToken := "valid-token-123"
		expectedHash := "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"
		expectedSalt := "abcd567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"

		// Mock repository to return valid crypto results
		mockRepo.On("HashToken", validToken).Return(expectedHash, expectedSalt, nil).Once()

		hashedToken, salt, err := svc.hashToken(validToken)

		require.NoError(t, err)
		assert.Equal(t, expectedHash, hashedToken)
		assert.Equal(t, expectedSalt, salt)
		mockRepo.AssertCalled(t, "HashToken", validToken)
	})

	t.Run("Business validation - reject invalid hash from repository", func(t *testing.T) { //nolint:paralleltest
		mockRepo := &mockRepositoryForHashTest{}
		s := NewService(mockRepo, nil, nil, nil, nil, nil, slog.Default(), 3600)
		svc, ok := s.(*service)
		require.True(t, ok)
		validToken := "valid-token-different" // Use different token to avoid mock conflicts
		invalidHash := "short"                // Not 64 chars
		validSalt := "abcd567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"

		// Mock repository to return invalid hash length
		mockRepo.On("HashToken", validToken).Return(invalidHash, validSalt, nil).Once()

		_, _, err := svc.hashToken(validToken)

		require.Error(t, err)
		require.Error(t, err)
		assert.Equal(t, "hash generation failed security validation", err.Error())
		mockRepo.AssertCalled(t, "HashToken", validToken)
	})

	t.Run("Business validation - reject invalid salt from repository", func(t *testing.T) { //nolint:paralleltest
		mockRepo := &mockRepositoryForHashTest{}
		s := NewService(mockRepo, nil, nil, nil, nil, nil, slog.Default(), 3600)
		svc, ok := s.(*service)
		require.True(t, ok)

		validToken := "valid-token-for-salt-test"
		validHash := "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"
		invalidSalt := "short"

		// Mock repository to return invalid salt length
		mockRepo.On("HashToken", validToken).Return(validHash, invalidSalt, nil).Once()

		_, _, err := svc.hashToken(validToken)

		require.Error(t, err)
		require.Error(t, err)
		assert.Equal(t, "hash generation failed security validation", err.Error())
		mockRepo.AssertCalled(t, "HashToken", validToken)
	})
}

// TestService_VerifyToken_BusinessLogic tests the business validation logic within the verifyToken service method.
// It ensures that input validation is performed correctly before delegating to the repository.
func TestService_VerifyToken_BusinessLogic(t *testing.T) {
	t.Run("Business validation - reject empty parameters", func(t *testing.T) {
		mockRepo := &mockRepositoryForHashTest{}
		s := NewService(mockRepo, nil, nil, nil, nil, nil, slog.Default(), 3600)
		svc, ok := s.(*service)
		require.True(t, ok)

		testCases := []struct {
			name        string
			plainToken  string
			hashedToken string
			salt        string
		}{
			{"empty plain token", "", "hash", "salt"},
			{"empty hashed token", "plain", "", "salt"},
			{"empty salt", "plain", "hash", ""},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) { //nolint:paralleltest
				isValid := svc.verifyToken(tc.plainToken, tc.hashedToken, tc.salt)

				assert.False(t, isValid)
				// Repository should not be called due to business validation failure
				mockRepo.AssertNotCalled(t, "VerifyToken")
			})
		}
	})

	t.Run("Business validation - reject short token", func(t *testing.T) {
		mockRepo := &mockRepositoryForHashTest{}
		s := NewService(mockRepo, nil, nil, nil, nil, nil, slog.Default(), 3600)
		svc, ok := s.(*service)
		require.True(t, ok)
		shortToken := "short"
		validHash := "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"
		validSalt := "abcd567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"

		isValid := svc.verifyToken(shortToken, validHash, validSalt)

		assert.False(t, isValid)
		// Repository should not be called due to business validation failure
		mockRepo.AssertNotCalled(t, "VerifyToken")
	})

	t.Run("Business validation - reject invalid hash/salt format", func(t *testing.T) {
		mockRepo := &mockRepositoryForHashTest{}
		s := NewService(mockRepo, nil, nil, nil, nil, nil, slog.Default(), 3600)
		svc, ok := s.(*service)
		require.True(t, ok)
		validToken := "valid-token-123"

		testCases := []struct {
			name        string
			hashedToken string
			salt        string
		}{
			{"short hash", "short", "abcd567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"},
			{"short salt", "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef", "short"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) { //nolint:paralleltest
				isValid := svc.verifyToken(validToken, tc.hashedToken, tc.salt)

				assert.False(t, isValid)
				// Repository should not be called due to business validation failure
				mockRepo.AssertNotCalled(t, "VerifyToken")
			})
		}
	})

	t.Run("Business validation - delegate to repository for valid inputs", func(t *testing.T) {
		mockRepo := &mockRepositoryForHashTest{}
		s := NewService(mockRepo, nil, nil, nil, nil, nil, slog.Default(), 3600)
		svc, ok := s.(*service)
		require.True(t, ok)
		validToken := "valid-token-123"
		validHash := "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef" // 64 chars
		validSalt := "abcd567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef" // 64 chars

		// Mock repository to return verification result
		mockRepo.On("VerifyToken", validToken, validHash, validSalt).Return(true).Once()

		isValid := svc.verifyToken(validToken, validHash, validSalt)

		assert.True(t, isValid)
		mockRepo.AssertCalled(t, "VerifyToken", validToken, validHash, validSalt)
	})
}
