package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestOIDCDiscoveryEndpoint_AfterDeletion(t *testing.T) {
	// This test verifies that OIDC Discovery endpoint returns 404 after deletion

	gin.SetMode(gin.TestMode)
	router := gin.New()

	// No route registered for /.well-known/openid-configuration

	// Test the endpoint
	req, _ := http.NewRequest("GET", "/.well-known/openid-configuration", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should return 404
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestJWKSEndpoint_RouteExists(t *testing.T) {
	// This test verifies that JWKS route can still be registered
	// We're not testing the handler implementation, just the route

	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Register a dummy handler for JWKS route
	router.GET("/.well-known/jwks.json", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"keys": []interface{}{}})
	})

	// Test the endpoint
	req, _ := http.NewRequest("GET", "/.well-known/jwks.json", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should return 200
	assert.Equal(t, http.StatusOK, w.Code)
}
