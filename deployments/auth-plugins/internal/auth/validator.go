package auth

import (
	"context"

	"github.com/coreos/go-oidc/v3/oidc"
)

func NewValidator(ctx context.Context, issuerURL string, clientID string) (*oidc.IDTokenVerifier, error) {
	provider, err := oidc.NewProvider(ctx, issuerURL)
	if err != nil {
		return nil, err
	}

	config := &oidc.Config{
		ClientID: clientID,
	}

	verifier := provider.Verifier(config)
	return verifier, nil
}

func Validate(ctx context.Context, verifier *oidc.IDTokenVerifier, tokenString string) (*oidc.IDToken, error) {
	idToken, err := verifier.Verify(ctx, tokenString)
	if err != nil {
		return nil, err
	}

	return idToken, nil
}