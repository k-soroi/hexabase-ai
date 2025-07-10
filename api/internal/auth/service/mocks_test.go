package service_test

import (
	"context"

	"github.com/stretchr/testify/mock"

	"github.com/hexabase/hexabase-ai/api/internal/auth/domain"
)

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
