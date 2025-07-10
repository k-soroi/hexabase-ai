package deprecated

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"os"
)

// JWK represents a JSON Web Key
type JWK struct {
	Kty string `json:"kty"`
	Use string `json:"use"`
	Kid string `json:"kid"`
	Alg string `json:"alg"`
	N   string `json:"n"`
	E   string `json:"e"`
}

// JWKS represents a JSON Web Key Set
type JWKS struct {
	Keys []JWK `json:"keys"`
}

// KeyManager manages RSA keys for JWT signing
type KeyManager struct {
	privateKey *rsa.PrivateKey
	publicKey  *rsa.PublicKey
	keyID      string
}

// NewKeyManager creates a new key manager
func NewKeyManager() (*KeyManager, error) {
	// Try to load existing keys first
	privateKey, err := loadPrivateKey()
	if err != nil {
		// Generate new keys if not found
		privateKey, err = generateRSAKeyPair()
		if err != nil {
			return nil, fmt.Errorf("failed to generate RSA key pair: %w", err)
		}
		
		// Save keys for persistence
		if err := savePrivateKey(privateKey); err != nil {
			// Log error but continue - keys will be regenerated on restart
			fmt.Printf("Warning: failed to save private key: %v\n", err)
		}
	}

	return &KeyManager{
		privateKey: privateKey,
		publicKey:  &privateKey.PublicKey,
		keyID:      generateKeyID(),
	}, nil
}

// generateRSAKeyPair generates a new RSA key pair
func generateRSAKeyPair() (*rsa.PrivateKey, error) {
	return rsa.GenerateKey(rand.Reader, 2048)
}

// generateKeyID generates a unique key ID
func generateKeyID() string {
	// In production, this should be a stable ID that changes only when keys rotate
	return fmt.Sprintf("hexabase-key-%d", 1)
}

// GetPrivateKey returns the private key
func (km *KeyManager) GetPrivateKey() *rsa.PrivateKey {
	return km.privateKey
}

// GetPublicKey returns the public key
func (km *KeyManager) GetPublicKey() *rsa.PublicKey {
	return km.publicKey
}

// GetJWKS returns the JSON Web Key Set
func (km *KeyManager) GetJWKS() (*JWKS, error) {
	// Convert RSA public key to JWK format
	jwk := JWK{
		Kty: "RSA",
		Use: "sig",
		Kid: km.keyID,
		Alg: "RS256",
		N:   base64.RawURLEncoding.EncodeToString(km.publicKey.N.Bytes()),
		E:   base64.RawURLEncoding.EncodeToString([]byte{1, 0, 1}), // 65537
	}

	return &JWKS{
		Keys: []JWK{jwk},
	}, nil
}

// GetJWKSJSON returns the JWKS as JSON
func (km *KeyManager) GetJWKSJSON() ([]byte, error) {
	jwks, err := km.GetJWKS()
	if err != nil {
		return nil, err
	}
	
	return json.MarshalIndent(jwks, "", "  ")
}

// loadPrivateKey loads a private key from file
func loadPrivateKey() (*rsa.PrivateKey, error) {
	keyPath := os.Getenv("JWT_PRIVATE_KEY_PATH")
	if keyPath == "" {
		keyPath = "/tmp/hexabase-private-key.pem"
	}

	keyData, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, err
	}

	block, _ := pem.Decode(keyData)
	if block == nil {
		return nil, fmt.Errorf("failed to parse PEM block")
	}

	return x509.ParsePKCS1PrivateKey(block.Bytes)
}

// savePrivateKey saves a private key to file
func savePrivateKey(key *rsa.PrivateKey) error {
	keyPath := os.Getenv("JWT_PRIVATE_KEY_PATH")
	if keyPath == "" {
		keyPath = "/tmp/hexabase-private-key.pem"
	}

	keyBytes := x509.MarshalPKCS1PrivateKey(key)
	block := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: keyBytes,
	}

	file, err := os.OpenFile(keyPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer file.Close()

	return pem.Encode(file, block)
}

// GetPublicKeyPEM returns the public key in PEM format
func (km *KeyManager) GetPublicKeyPEM() (string, error) {
	pubKeyBytes, err := x509.MarshalPKIXPublicKey(km.publicKey)
	if err != nil {
		return "", err
	}

	block := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubKeyBytes,
	}

	return string(pem.EncodeToMemory(block)), nil
}