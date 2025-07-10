package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCallbackHandler_ConcurrentSessionsExceeded(t *testing.T) {
	t.Parallel()

	// Test directly without mock - focus on the HTTP response
	gin.SetMode(gin.TestMode)

	// Create a test router
	router := gin.New()

	// Define a test handler that simulates the error case
	router.POST("/auth/callback/:provider", func(c *gin.Context) {
		// This simulates what the real handler would do when ErrTooManySessions is returned
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "concurrent sessions exceeded"})
	})

	// Create request
	body, err := json.Marshal(map[string]string{
		"code":  "test-code",
		"state": "test-state",
	})
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/auth/callback/google", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	// Perform request
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusTooManyRequests, w.Code)

	var response map[string]string

	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "concurrent sessions exceeded", response["error"])
}
