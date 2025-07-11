package jwks

import (
	"encoding/json"
	"net/http"

	"gopkg.in/go-jose/go-jose.v2"
)

type JSONWebKeySet struct {
	Keys []jose.JSONWebKey `json:"keys"`
}

func NewJWKSHandler(publicKey jose.JSONWebKey) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		jwks := JSONWebKeySet{
			Keys: []jose.JSONWebKey{publicKey},
		}

		w.Header().Set("Content-Type", "application/json")
		
		if err := json.NewEncoder(w).Encode(jwks); err != nil {
			http.Error(w, "Failed to encode JWKS", http.StatusInternalServerError)
			return
		}
	})
}