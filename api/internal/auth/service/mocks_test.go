package service_test

import (
	"context"
	"time"

	"github.com/stretchr/testify/mock"

	"github.com/hexabase/hexabase-ai/api/internal/auth/domain"
)

// Mock repository
type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) CreateUser(ctx context.Context, user *domain.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockRepository) GetUser(ctx context.Context, userID string) (*domain.User, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) != nil {
		user, ok := args.Get(0).(*domain.User)
		if !ok {
			return nil, args.Error(1)
		}

		return user, args.Error(1)
	}

	return nil, args.Error(1)
}

func (m *MockRepository) GetUserByExternalID(ctx context.Context, externalID, provider string) (*domain.User, error) {
	args := m.Called(ctx, externalID, provider)
	if args.Get(0) != nil {
		user, ok := args.Get(0).(*domain.User)
		if !ok {
			return nil, args.Error(1)
		}

		return user, args.Error(1)
	}

	return nil, args.Error(1)
}

func (m *MockRepository) GetUserByEmail(ctx context.Context, email string) (*domain.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) != nil {
		user, ok := args.Get(0).(*domain.User)
		if !ok {
			return nil, args.Error(1)
		}

		return user, args.Error(1)
	}

	return nil, args.Error(1)
}

func (m *MockRepository) UpdateUser(ctx context.Context, user *domain.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockRepository) UpdateLastLogin(ctx context.Context, userID string) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func (m *MockRepository) CreateSession(ctx context.Context, session *domain.Session) error {
	args := m.Called(ctx, session)
	return args.Error(0)
}

func (m *MockRepository) GetSession(ctx context.Context, sessionID string) (*domain.Session, error) {
	args := m.Called(ctx, sessionID)
	if args.Get(0) != nil {
		session, ok := args.Get(0).(*domain.Session)
		if !ok {
			return nil, args.Error(1)
		}

		return session, args.Error(1)
	}

	return nil, args.Error(1)
}

func (m *MockRepository) GetSessionByRefreshTokenSelector(
	ctx context.Context, selector string,
) (*domain.Session, error) {
	args := m.Called(ctx, selector)
	if args.Get(0) != nil {
		session, ok := args.Get(0).(*domain.Session)
		if !ok {
			return nil, args.Error(1)
		}

		return session, args.Error(1)
	}

	return nil, args.Error(1)
}

func (m *MockRepository) GetAllActiveSessions(ctx context.Context) ([]*domain.Session, error) {
	args := m.Called(ctx)
	if args.Get(0) != nil {
		sessions, ok := args.Get(0).([]*domain.Session)
		if !ok {
			return nil, args.Error(1)
		}

		return sessions, args.Error(1)
	}

	return nil, args.Error(1)
}

func (m *MockRepository) ListUserSessions(ctx context.Context, userID string) ([]*domain.Session, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) != nil {
		sessions, ok := args.Get(0).([]*domain.Session)
		if !ok {
			return nil, args.Error(1)
		}

		return sessions, args.Error(1)
	}

	return nil, args.Error(1)
}

func (m *MockRepository) UpdateSession(ctx context.Context, session *domain.Session) error {
	args := m.Called(ctx, session)
	return args.Error(0)
}

func (m *MockRepository) DeleteSession(ctx context.Context, sessionID string) error {
	args := m.Called(ctx, sessionID)
	return args.Error(0)
}

func (m *MockRepository) DeleteUserSessions(ctx context.Context, userID string, exceptSessionID string) error {
	args := m.Called(ctx, userID, exceptSessionID)
	return args.Error(0)
}

func (m *MockRepository) CleanupExpiredSessions(ctx context.Context, before time.Time) error {
	args := m.Called(ctx, before)
	return args.Error(0)
}

func (m *MockRepository) StoreAuthState(ctx context.Context, state *domain.AuthState) error {
	args := m.Called(ctx, state)
	return args.Error(0)
}

func (m *MockRepository) GetAuthState(ctx context.Context, stateValue string) (*domain.AuthState, error) {
	args := m.Called(ctx, stateValue)
	if args.Get(0) != nil {
		state, ok := args.Get(0).(*domain.AuthState)
		if !ok {
			return nil, args.Error(1)
		}

		return state, args.Error(1)
	}

	return nil, args.Error(1)
}

func (m *MockRepository) DeleteAuthState(ctx context.Context, stateValue string) error {
	args := m.Called(ctx, stateValue)
	return args.Error(0)
}

func (m *MockRepository) BlacklistRefreshToken(ctx context.Context, token string, expiresAt time.Time) error {
	args := m.Called(ctx, token, expiresAt)
	return args.Error(0)
}

func (m *MockRepository) IsRefreshTokenBlacklisted(ctx context.Context, token string) (bool, error) {
	args := m.Called(ctx, token)
	return args.Bool(0), args.Error(1)
}

func (m *MockRepository) BlockSession(ctx context.Context, sessionID string, expiresAt time.Time) error {
	args := m.Called(ctx, sessionID, expiresAt)
	return args.Error(0)
}

func (m *MockRepository) IsSessionBlocked(ctx context.Context, sessionID string) (bool, error) {
	args := m.Called(ctx, sessionID)
	return args.Bool(0), args.Error(1)
}

func (m *MockRepository) CreateSecurityEvent(ctx context.Context, event *domain.SecurityEvent) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}

func (m *MockRepository) ListSecurityEvents(
	ctx context.Context, filter domain.SecurityLogFilter,
) ([]*domain.SecurityEvent, error) {
	args := m.Called(ctx, filter)
	if args.Get(0) != nil {
		events, ok := args.Get(0).([]*domain.SecurityEvent)
		if !ok {
			return nil, args.Error(1)
		}

		return events, args.Error(1)
	}

	return nil, args.Error(1)
}

func (m *MockRepository) CleanupOldSecurityEvents(ctx context.Context, before time.Time) error {
	args := m.Called(ctx, before)
	return args.Error(0)
}

func (m *MockRepository) GetUserOrganizations(ctx context.Context, userID string) ([]string, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) != nil {
		orgs, ok := args.Get(0).([]string)
		if !ok {
			return nil, args.Error(1)
		}

		return orgs, args.Error(1)
	}

	return nil, args.Error(1)
}

func (m *MockRepository) GetUserWorkspaceGroups(ctx context.Context, userID, workspaceID string) ([]string, error) {
	args := m.Called(ctx, userID, workspaceID)
	if args.Get(0) != nil {
		groups, ok := args.Get(0).([]string)
		if !ok {
			return nil, args.Error(1)
		}

		return groups, args.Error(1)
	}

	return nil, args.Error(1)
}

func (m *MockRepository) HashToken(token string) (hashedToken string, salt string, err error) {
	args := m.Called(token)
	return args.String(0), args.String(1), args.Error(2)
}

func (m *MockRepository) VerifyToken(plainToken, hashedToken, salt string) bool {
	args := m.Called(plainToken, hashedToken, salt)
	return args.Bool(0)
}

// Mock token domain service
type MockTokenDomainService struct {
	mock.Mock
}

func (m *MockTokenDomainService) RefreshToken(
	ctx context.Context, session *domain.Session, user *domain.User,
) (*domain.Claims, error) {
	args := m.Called(ctx, session, user)
	if args.Get(0) != nil {
		claims, ok := args.Get(0).(*domain.Claims)
		if !ok {
			return nil, args.Error(1)
		}

		return claims, args.Error(1)
	}

	return nil, args.Error(1)
}

func (m *MockTokenDomainService) ValidateRefreshEligibility(session *domain.Session) error {
	args := m.Called(session)
	return args.Error(0)
}

func (m *MockTokenDomainService) CreateSession(
	sessionID, userID, refreshToken, deviceID, clientIP, userAgent string,
) (*domain.Session, error) {
	args := m.Called(sessionID, userID, refreshToken, deviceID, clientIP, userAgent)
	if args.Get(0) != nil {
		session, ok := args.Get(0).(*domain.Session)
		if !ok {
			return nil, args.Error(1)
		}

		return session, args.Error(1)
	}

	return nil, args.Error(1)
}

func (m *MockTokenDomainService) ValidateTokenClaims(claims *domain.Claims) error {
	args := m.Called(claims)
	return args.Error(0)
}

func (m *MockTokenDomainService) ShouldRefreshToken(claims *domain.Claims) bool {
	args := m.Called(claims)
	return args.Bool(0)
}
