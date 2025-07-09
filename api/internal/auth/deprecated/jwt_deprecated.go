// Package deprecated contains deprecated authentication-related code that is no longer in active use
// but is kept for reference or potential future needs.
//
//nolint:all // This package contains deprecated code that is not used in the current version
package deprecated

import (
	"crypto/rsa"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// TokenManager is a struct to hold required components for token generation.
// This is a replica of the original auth.TokenManager to avoid circular dependencies and
// to keep deprecated code self-contained.
type TokenManager struct {
	PrivateKey *rsa.PrivateKey
	PublicKey  *rsa.PublicKey
	Issuer     string
	Expiration time.Duration
}

// NewTokenManager creates a new token manager for deprecated functions.
func NewTokenManager(privateKey *rsa.PrivateKey, publicKey *rsa.PublicKey, issuer string, expiration time.Duration) *TokenManager {
	return &TokenManager{
		PrivateKey: privateKey,
		PublicKey:  publicKey,
		Issuer:     issuer,
		Expiration: expiration,
	}
}

// Claims represents JWT claims for Hexabase
type Claims struct {
	jwt.RegisteredClaims
	UserID string   `json:"user_id"`
	Email  string   `json:"email"`
	Name   string   `json:"name"`
	OrgIDs []string `json:"org_ids,omitempty"`
}

// GenerateToken generates a JWT token for a user
func (tm *TokenManager) GenerateToken(userID, email, name string, orgIDs []string) (string, error) {
	now := time.Now()
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    tm.Issuer,
			Subject:   userID,
			Audience:  []string{"hexabase-api"},
			ExpiresAt: jwt.NewNumericDate(now.Add(tm.Expiration)),
			NotBefore: jwt.NewNumericDate(now),
			IssuedAt:  jwt.NewNumericDate(now),
			ID:        uuid.New().String(),
		},
		UserID: userID,
		Email:  email,
		Name:   name,
		OrgIDs: orgIDs,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	return token.SignedString(tm.PrivateKey)
}

// ValidateToken validates a JWT token
func (tm *TokenManager) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// Validate signing method
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return tm.PublicKey, nil
	})
	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token")
}
