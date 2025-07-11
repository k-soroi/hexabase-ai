package auth

import (
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
}

type TokenClaims struct {
	jwt.RegisteredClaims
	KeyID string `json:"kid"`
}

func NewTokenHandler(privateKey *rsa.PrivateKey, issuerURL string, keyID string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		if err := r.ParseForm(); err != nil {
			http.Error(w, "Failed to parse form", http.StatusBadRequest)
			return
		}

		grantType := r.FormValue("grant_type")
		refreshToken := r.FormValue("refresh_token")

		if grantType != "refresh_token" && grantType != "client_credentials" {
			http.Error(w, "Invalid grant_type", http.StatusBadRequest)
			return
		}

		if grantType == "refresh_token" && refreshToken == "" {
			http.Error(w, "Missing refresh_token", http.StatusBadRequest)
			return
		}

		now := time.Now()
		expiresAt := now.Add(60 * time.Second)

		claims := &TokenClaims{
			RegisteredClaims: jwt.RegisteredClaims{
				Issuer:    issuerURL,
				Subject:   "mock-user",
				Audience:  []string{"kubernetes-api"},
				ExpiresAt: jwt.NewNumericDate(expiresAt),
				IssuedAt:  jwt.NewNumericDate(now),
			},
			KeyID: keyID,
		}

		token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
		token.Header["kid"] = keyID

		tokenString, err := token.SignedString(privateKey)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to sign token: %v", err), http.StatusInternalServerError)
			return
		}

		newRefreshToken := uuid.New().String()

		response := TokenResponse{
			AccessToken:  tokenString,
			RefreshToken: newRefreshToken,
			TokenType:    "Bearer",
			ExpiresIn:    60,
		}

		w.Header().Set("Content-Type", "application/json")
		
		if err := json.NewEncoder(w).Encode(response); err != nil {
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			return
		}
	})
}