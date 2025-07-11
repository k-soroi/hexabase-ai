package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"kube-auth-plugin/internal/auth"
	"kube-auth-plugin/internal/jwks"
	"kube-auth-plugin/internal/util"
)

func main() {
	privateKey, err := util.GenerateRSAKeys()
	if err != nil {
		log.Fatalf("Failed to generate RSA keys: %v", err)
	}

	jwk := util.ConvertPrivateKeyToJWK(privateKey)
	publicJWK := jwk.Public()

	issuerURL := "http://172.31.78.208:8080"
	jwksURL := "http://172.31.78.208:8080/.well-known/jwks.json"

	mux := http.NewServeMux()

	discoveryHandler := jwks.NewDiscoveryHandler(issuerURL, jwksURL)
	mux.Handle("/.well-known/openid-configuration", discoveryHandler)

	jwksHandler := jwks.NewJWKSHandler(publicJWK)
	mux.Handle("/.well-known/jwks.json", jwksHandler)

	tokenHandler := auth.NewTokenHandler(privateKey, issuerURL, jwk.KeyID)
	mux.Handle("/oauth/token", tokenHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	fmt.Printf("Mock server starting on port %s\n", port)
	fmt.Printf("Issuer URL: %s\n", issuerURL)
	fmt.Printf("JWKS URL: %s\n", jwksURL)
	fmt.Printf("Token URL: %s/oauth/token\n", issuerURL)

	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}