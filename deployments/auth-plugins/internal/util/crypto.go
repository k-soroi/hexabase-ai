package util

import (
	"crypto/rand"
	"crypto/rsa"

	"github.com/google/uuid"
	"gopkg.in/go-jose/go-jose.v2"
)

func GenerateRSAKeys() (*rsa.PrivateKey, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}
	return privateKey, nil
}

func ConvertPrivateKeyToJWK(privateKey *rsa.PrivateKey) jose.JSONWebKey {
	keyID := uuid.New().String()
	
	jwk := jose.JSONWebKey{
		Key:       privateKey,
		KeyID:     keyID,
		Algorithm: string(jose.RS256),
		Use:       "sig",
	}
	
	return jwk
}