package auth

import (
	"crypto/rsa"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/hexabase/hexabase-ai/api/internal/auth/domain"
)

// TokenManager handles JWT token generation and validation
type TokenManager struct {
	privateKey *rsa.PrivateKey
	publicKey  *rsa.PublicKey
	issuer     string
	expiration time.Duration
}

// WorkspaceClaims represents JWT claims for vCluster access
type WorkspaceClaims struct {
	jwt.RegisteredClaims
	UserID      string   `json:"user_id"`
	WorkspaceID string   `json:"workspace_id"`
	Groups      []string `json:"groups"`
}

// NewTokenManager creates a new token manager
func NewTokenManager(privateKey *rsa.PrivateKey, publicKey *rsa.PublicKey, issuer string, expiration time.Duration) *TokenManager {
	return &TokenManager{
		privateKey: privateKey,
		publicKey:  publicKey,
		issuer:     issuer,
		expiration: expiration,
	}
}

// SignClaims signs claims to create a JWT token
// This method is pure infrastructure - no business logic
func (tm *TokenManager) SignClaims(claims *domain.Claims) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	return token.SignedString(tm.privateKey)
}