package jwks

import (
	"encoding/json"
	"net/http"
)

type OIDCDiscovery struct {
	Issuer                           string   `json:"issuer"`
	JWKSURI                         string   `json:"jwks_uri"`
	ResponseTypesSupported          []string `json:"response_types_supported"`
	SubjectTypesSupported           []string `json:"subject_types_supported"`
	IDTokenSigningAlgValuesSupported []string `json:"id_token_signing_alg_values_supported"`
}

func NewDiscoveryHandler(issuerURL, jwksURL string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		discovery := OIDCDiscovery{
			Issuer:                           issuerURL,
			JWKSURI:                         jwksURL,
			ResponseTypesSupported:          []string{"code"},
			SubjectTypesSupported:           []string{"public"},
			IDTokenSigningAlgValuesSupported: []string{"RS256"},
		}

		w.Header().Set("Content-Type", "application/json")
		
		if err := json.NewEncoder(w).Encode(discovery); err != nil {
			http.Error(w, "Failed to encode discovery", http.StatusInternalServerError)
			return
		}
	})
}