package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

type ExecCredential struct {
	APIVersion string `json:"apiVersion"`
	Kind       string `json:"kind"`
	Status     Status `json:"status"`
}

type Status struct {
	ExpirationTimestamp string `json:"expirationTimestamp"`
	Token               string `json:"token"`
}

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
}

func main() {
	// デバッグ用ログ出力
	fmt.Fprintf(os.Stderr, "=== EXEC AUTH PLUGIN STARTED ===\n")
	
	tokenURL := os.Getenv("KUBE_AUTH_TOKEN_URL")
	if tokenURL == "" {
		tokenURL = "http://192.168.11.10:8081/oauth/token"
	}
	
	fmt.Fprintf(os.Stderr, "Token URL: %s\n", tokenURL)

	// Get token from mock server
	resp, err := http.Post(tokenURL, "application/x-www-form-urlencoded", 
		strings.NewReader("grant_type=client_credentials"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get token: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to read response: %v\n", err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "Token Response Status: %s\n", resp.Status)
	fmt.Fprintf(os.Stderr, "Token Response Body: %s\n", string(body))

	var tokenResp TokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to parse token response: %v\n", err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "Access Token (first 50 chars): %s...\n", tokenResp.AccessToken[:50])

	// Create exec credential
	execCred := ExecCredential{
		APIVersion: "client.authentication.k8s.io/v1",
		Kind:       "ExecCredential",
		Status: Status{
			ExpirationTimestamp: time.Now().Add(time.Hour).Format(time.RFC3339),
			Token:               tokenResp.AccessToken,
		},
	}

	// Output as JSON
	output, err := json.Marshal(execCred)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to marshal exec credential: %v\n", err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "=== EXEC CREDENTIAL GENERATED ===\n")
	fmt.Fprintf(os.Stderr, "ExecCredential JSON Length: %d bytes\n", len(output))
	fmt.Fprintf(os.Stderr, "=== PLUGIN EXECUTION COMPLETED ===\n")

	fmt.Print(string(output))
}