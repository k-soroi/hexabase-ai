package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"kube-auth-plugin/internal/auth"
	"kube-auth-plugin/internal/cache"
)

type ExecCredential struct {
	APIVersion string                `json:"apiVersion"`
	Kind       string                `json:"kind"`
	Status     *ExecCredentialStatus `json:"status"`
}

type ExecCredentialStatus struct {
	ExpirationTimestamp *time.Time `json:"expirationTimestamp,omitempty"`
	Token               string     `json:"token"`
}

func writeCredential(status *ExecCredentialStatus) {
	credential := &ExecCredential{
		APIVersion: "client.authentication.k8s.io/v1",
		Kind:       "ExecCredential",
		Status:     status,
	}

	if err := json.NewEncoder(os.Stdout).Encode(credential); err != nil {
		logError("Failed to encode credential: %v", err)
		os.Exit(1)
	}
}

func logError(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
}

func getCachePath() string {
	if path := os.Getenv("KUBE_AUTH_CACHE_PATH"); path != "" {
		return path
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		logError("Failed to get home directory: %v", err)
		os.Exit(1)
	}

	return filepath.Join(homeDir, ".kube", "cache", "auth-plugin", "tokens.json")
}

func main() {
	ctx := context.Background()

	issuerURL := os.Getenv("KUBE_AUTH_ISSUER_URL")
	if issuerURL == "" {
		issuerURL = "http://localhost:8080"
	}

	tokenURL := os.Getenv("KUBE_AUTH_TOKEN_URL")
	if tokenURL == "" {
		tokenURL = "http://localhost:8080/oauth/token"
	}

	audience := os.Getenv("KUBE_AUTH_AUDIENCE")
	if audience == "" {
		audience = "kubernetes-api"
	}

	cachePath := getCachePath()

	tokenCache, err := cache.Load(cachePath)
	if err != nil {
		logError("Failed to load token cache: %v", err)
		os.Exit(1)
	}

	if tokenCache == nil || tokenCache.AccessToken == "" {
		if tokenCache == nil {
			logError("No cached tokens found. Please authenticate manually and ensure refresh token is available.")
			os.Exit(1)
		}
		
		if tokenCache.RefreshToken == "" {
			logError("No refresh token found. Please authenticate manually and ensure refresh token is available.")
			os.Exit(1)
		}

		newAccessToken, newRefreshToken, err := auth.Refresh(ctx, tokenCache.RefreshToken, tokenURL)
		if err != nil {
			logError("Failed to refresh token: %v", err)
			os.Exit(1)
		}

		tokenCache.AccessToken = newAccessToken
		tokenCache.RefreshToken = newRefreshToken

		if err := cache.Save(cachePath, tokenCache); err != nil {
			logError("Failed to save token cache: %v", err)
			os.Exit(1)
		}
	}

	verifier, err := auth.NewValidator(ctx, issuerURL, audience)
	if err != nil {
		logError("Failed to create validator: %v", err)
		os.Exit(1)
	}

	idToken, err := auth.Validate(ctx, verifier, tokenCache.AccessToken)
	if err != nil {
		logError("Token validation failed, attempting refresh: %v", err)

		if tokenCache.RefreshToken == "" {
			logError("No refresh token available. Please authenticate manually.")
			os.Exit(1)
		}

		newAccessToken, newRefreshToken, err := auth.Refresh(ctx, tokenCache.RefreshToken, tokenURL)
		if err != nil {
			logError("Failed to refresh token: %v", err)
			os.Exit(1)
		}

		tokenCache.AccessToken = newAccessToken
		tokenCache.RefreshToken = newRefreshToken

		if err := cache.Save(cachePath, tokenCache); err != nil {
			logError("Failed to save token cache: %v", err)
			os.Exit(1)
		}

		idToken, err = auth.Validate(ctx, verifier, newAccessToken)
		if err != nil {
			logError("Token validation failed after refresh: %v", err)
			os.Exit(1)
		}
	}

	status := &ExecCredentialStatus{
		Token:               tokenCache.AccessToken,
		ExpirationTimestamp: &idToken.Expiry,
	}

	writeCredential(status)
}