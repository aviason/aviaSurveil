package identity

import (
	"context"
	"time"
)

type ProviderTokens struct {
	AccessToken  string    `json:"accessToken"`
	RefreshToken string    `json:"refreshToken"`
	IDToken      string    `json:"idToken"`
	Expiry       time.Time `json:"expiry"`
}

type OIDCIdentity struct {
	SubjectID          string
	Issuer             string
	DisplayName        string
	Email              string
	OrganizationID     string
	Roles              []Role
	ProviderSessionID  string
	MembershipID       string
	MembershipRevision int64
	AuthRevision       uint64
	AMR                []string
	AuthTime           time.Time
	Tokens             ProviderTokens
}

type OIDCProvider interface {
	AuthorizationURL(state, nonce, pkceChallenge string) string
	Exchange(context.Context, string, string, string) (OIDCIdentity, error)
	LogoutURL(idTokenHint string) string
}
