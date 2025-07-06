package handler_test

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/hexabase/hexabase-ai/api/internal/auth/domain"
	"github.com/hexabase/hexabase-ai/api/internal/auth/handler"
	"github.com/hexabase/hexabase-ai/api/internal/shared/infrastructure/server/ogen"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestOgenSignUpIntegration(t *testing.T) {
	t.Parallel()

	// Setup
	gin.SetMode(gin.TestMode)

	mockService := new(mockAuthService)
	ogenHandler := handler.NewOgenAuthHandler(mockService, slog.Default())

	// Create ogen server
	server, err := ogen.NewServer(ogenHandler)
	require.NoError(t, err)

	// Setup expectations
	expectedAuthURL := testGoogleAuthURL
	expectedState := "random-state-123"

	mockService.On("GetAuthURLForSignUp",
		mock.Anything,
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
	body, err := json.Marshal(reqBody)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/auth/sign-up/google", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	// Create response recorder
	w := httptest.NewRecorder()

	// Execute request through ogen server
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
}
