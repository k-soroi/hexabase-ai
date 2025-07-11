package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

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
	cachePath := getCachePath()

	tokenCache, err := cache.Load(cachePath)
	if err != nil {
		logError("Failed to load token cache: %v", err)
		os.Exit(1)
	}

	if tokenCache == nil || tokenCache.AccessToken == "" {
		logError("No access token found in cache")
		os.Exit(1)
	}

	// ServiceAccountトークンの場合、有効期限を1時間後に設定
	expiryTime := time.Now().Add(1 * time.Hour)

	status := &ExecCredentialStatus{
		Token:               tokenCache.AccessToken,
		ExpirationTimestamp: &expiryTime,
	}

	writeCredential(status)
}