package service_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

var (
	ErrEmptyToken           = errors.New("token cannot be empty")
	ErrTokenTooShort        = errors.New("token must be at least 8 characters long")
	ErrHashValidationFailed = errors.New("hash generation failed security validation")
)

// tokenHashService is a minimal service implementation for testing
type tokenHashService struct {
	repo interface {
		HashToken(token string) (hashedToken string, salt string, err error)
		VerifyToken(plainToken, hashedToken, salt string) bool
	}
}

func (s *tokenHashService) hashToken(token string) (hashedToken string, salt string, err error) {
	// Business validation
	if token == "" {
		return "", "", ErrEmptyToken
	}

	if len(token) < 8 {
		return "", "", ErrTokenTooShort
	}

	// Delegate to repository
	hashedToken, salt, err = s.repo.HashToken(token)
	if err != nil {
		return "", "", err
	}

	// Validate output
	if len(hashedToken) != 64 {
		return "", "", ErrHashValidationFailed
	}

	return hashedToken, salt, nil
}

func (s *tokenHashService) verifyToken(plainToken, hashedToken, salt string) bool {
	// Business validation
	if plainToken == "" || hashedToken == "" || salt == "" {
		return false
	}

	// Check token length
	if len(plainToken) < 8 {
		return false
	}

	// Check hash/salt format
	if len(hashedToken) != 64 || len(salt) != 64 {
		return false
	}

	// Delegate to repository
	return s.repo.VerifyToken(plainToken, hashedToken, salt)
}

// Define minimal mock for this test
type mockTokenHashRepository struct {
	mock.Mock
}

func (m *mockTokenHashRepository) HashToken(token string) (hashedToken string, salt string, err error) {
	args := m.Called(token)
	return args.String(0), args.String(1), args.Error(2)
}

func (m *mockTokenHashRepository) VerifyToken(plainToken, hashedToken, salt string) bool {
	args := m.Called(plainToken, hashedToken, salt)
	return args.Bool(0)
}

func TestService_HashToken_BusinessLogic(t *testing.T) {
	t.Run("Business validation - reject empty token", func(t *testing.T) {
		mockRepo := &mockTokenHashRepository{}
		svc := &tokenHashService{repo: mockRepo}

		_, _, err := svc.hashToken("")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "token cannot be empty")
		// Repository should not be called due to business validation failure
		mockRepo.AssertNotCalled(t, "HashToken")
	})

	t.Run("Business validation - reject short token", func(t *testing.T) {
		mockRepo := &mockTokenHashRepository{}
		svc := &tokenHashService{repo: mockRepo}
		shortToken := "short"

		_, _, err := svc.hashToken(shortToken)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "token must be at least 8 characters long")
		// Repository should not be called due to business validation failure
		mockRepo.AssertNotCalled(t, "HashToken")
	})

	t.Run("Business validation - accept valid token and delegate to repository", func(t *testing.T) {
		mockRepo := &mockTokenHashRepository{}
		svc := &tokenHashService{repo: mockRepo}
		validToken := "valid-token-123"
		expectedHash := "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"
		expectedSalt := "abcd567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"

		// Mock repository to return valid crypto results
		mockRepo.On("HashToken", validToken).Return(expectedHash, expectedSalt, nil)

		hashedToken, salt, err := svc.hashToken(validToken)

		require.NoError(t, err)
		assert.Equal(t, expectedHash, hashedToken)
		assert.Equal(t, expectedSalt, salt)
		mockRepo.AssertCalled(t, "HashToken", validToken)
	})

	t.Run("Business validation - reject invalid output from repository", func(t *testing.T) {
		mockRepo := &mockTokenHashRepository{}
		svc := &tokenHashService{repo: mockRepo}
		validToken := "valid-token-different" // Use different token to avoid mock conflicts
		invalidHash := "short"                // Not 64 chars
		validSalt := "abcd567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"

		// Mock repository to return invalid hash length
		mockRepo.On("HashToken", validToken).Return(invalidHash, validSalt, nil)

		_, _, err := svc.hashToken(validToken)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "hash generation failed security validation")
	})
}

func TestService_VerifyToken_BusinessLogic(t *testing.T) {
	t.Run("Business validation - reject empty parameters", func(t *testing.T) {
		mockRepo := &mockTokenHashRepository{}
		svc := &tokenHashService{repo: mockRepo}

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
			t.Run(tc.name, func(t *testing.T) {
				isValid := svc.verifyToken(tc.plainToken, tc.hashedToken, tc.salt)

				assert.False(t, isValid)
				// Repository should not be called due to business validation failure
				mockRepo.AssertNotCalled(t, "VerifyToken")
			})
		}
	})

	t.Run("Business validation - reject short token", func(t *testing.T) {
		mockRepo := &mockTokenHashRepository{}
		svc := &tokenHashService{repo: mockRepo}
		shortToken := "short"
		validHash := "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"
		validSalt := "abcd567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"

		isValid := svc.verifyToken(shortToken, validHash, validSalt)

		assert.False(t, isValid)
		// Repository should not be called due to business validation failure
		mockRepo.AssertNotCalled(t, "VerifyToken")
	})

	t.Run("Business validation - reject invalid hash/salt format", func(t *testing.T) {
		mockRepo := &mockTokenHashRepository{}
		svc := &tokenHashService{repo: mockRepo}
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
			t.Run(tc.name, func(t *testing.T) {
				isValid := svc.verifyToken(validToken, tc.hashedToken, tc.salt)

				assert.False(t, isValid)
				// Repository should not be called due to business validation failure
				mockRepo.AssertNotCalled(t, "VerifyToken")
			})
		}
	})

	t.Run("Business validation - delegate to repository for valid inputs", func(t *testing.T) {
		mockRepo := &mockTokenHashRepository{}
		svc := &tokenHashService{repo: mockRepo}
		validToken := "valid-token-123"
		validHash := "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef" // 64 chars
		validSalt := "abcd567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef" // 64 chars

		// Mock repository to return verification result
		mockRepo.On("VerifyToken", validToken, validHash, validSalt).Return(true)

		isValid := svc.verifyToken(validToken, validHash, validSalt)

		assert.True(t, isValid)
		mockRepo.AssertCalled(t, "VerifyToken", validToken, validHash, validSalt)
	})
}
