// Package jwks serves the public JWK set at /.well-known/jwks.json. The
// endpoint is unauthenticated by design — API Gateway's JWT authorizer polls
// it to validate tokens at the edge.
package jwks

import (
	"encoding/json"
	"net/http"

	"kindred_server/internal/auth"
	"kindred_server/internal/platform/router"
)

type Handler struct {
	jwks   auth.JWKS
	issuer string
}

func NewHandler(jwks auth.JWKS, issuer string) *Handler {
	return &Handler{jwks: jwks, issuer: issuer}
}

func RegisterRoutes(r *router.Router, h *Handler) {
	r.Handle(http.MethodGet, "/.well-known/jwks.json", http.HandlerFunc(h.ServeJWKS))
	// OIDC discovery — APIGW HTTP API JWT authorizer probes this URL when
	// the authorizer is created and refuses any other discovery shape.
	r.Handle(http.MethodGet, "/.well-known/openid-configuration", http.HandlerFunc(h.ServeOIDC))
}

func (h *Handler) ServeJWKS(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=300")
	_ = json.NewEncoder(w).Encode(h.jwks)
}

// ServeOIDC returns the minimal OIDC discovery document APIGW requires.
// `issuer` MUST match the value Terraform passes to the JWT authorizer.
func (h *Handler) ServeOIDC(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=300")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"issuer":                                h.issuer,
		"jwks_uri":                              h.issuer + "/.well-known/jwks.json",
		"id_token_signing_alg_values_supported": []string{"RS256"},
		"response_types_supported":              []string{"id_token"},
		"subject_types_supported":               []string{"public"},
	})
}
