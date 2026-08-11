// Package provider contains the narrow protocol boundary owned by the
// first-party provider. The candidate protocol harness is exercised by later
// task tests, while cmd/auth remains liveness-only until durable runtime
// wiring is separately authorized.
package provider

import "github.com/zitadel/oidc/v3/pkg/op"

const (
	SelectedLibrary = "github.com/zitadel/oidc/v3"
	SelectedVersion = "v3.47.5"
)

// ProtocolBoundary is the immutable route vocabulary supplied by the selected
// provider library. It keeps the dependency visible in the provider module
// without pretending that the empty Task 2 scaffold is an OIDC server.
type ProtocolBoundary struct {
	AuthorizationEndpoint string
	TokenEndpoint         string
	JWKSURI               string
	EndSessionEndpoint    string
}

func NewProtocolBoundary() ProtocolBoundary {
	return ProtocolBoundary{
		AuthorizationEndpoint: op.DefaultEndpoints.Authorization.Relative(),
		TokenEndpoint:         op.DefaultEndpoints.Token.Relative(),
		JWKSURI:               op.DefaultEndpoints.JwksURI.Relative(),
		EndSessionEndpoint:    op.DefaultEndpoints.EndSession.Relative(),
	}
}
