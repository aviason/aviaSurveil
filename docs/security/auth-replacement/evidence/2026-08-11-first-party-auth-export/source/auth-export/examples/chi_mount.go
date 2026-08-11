package examples

import (
	"encoding/json"
	"net/http"

	"example.invalid/aviasurveil360-auth-export/auth"
)

// MethodRouter is the subset of chi.Router used by this adapter. A *chi.Mux
// satisfies it without making chi a dependency of the exported auth package.
type MethodRouter interface {
	MethodFunc(method, pattern string, handler http.HandlerFunc)
}

// MountMetadata reproduces the only standalone HTTP routes used by the source
// runtime. These endpoints are informational; they do not make the package an
// OIDC authorization server.
func MountMetadata(router MethodRouter, tokens *auth.TokenService) {
	router.MethodFunc(http.MethodGet, "/.well-known/openid-configuration", jsonHandler(tokens.OpenIDConfiguration))
	router.MethodFunc(http.MethodGet, "/oauth2/jwks", jsonHandler(tokens.JWKS))
}

func jsonHandler(value func() map[string]any) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(value())
	}
}
