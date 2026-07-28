package identity

import (
	"context"
	"crypto/subtle"
	"fmt"
	"strings"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

type RemoteOIDCConfig struct {
	IssuerURL    string
	DiscoveryURL string
	ClientID     string
	ClientSecret string
	RedirectURL  string
}

type RemoteOIDCProvider struct {
	oauthConfig oauth2.Config
	verifier    *oidc.IDTokenVerifier
}

func NewRemoteOIDCProvider(ctx context.Context, config RemoteOIDCConfig) (*RemoteOIDCProvider, error) {
	if strings.TrimSpace(config.IssuerURL) == "" || strings.TrimSpace(config.ClientID) == "" || strings.TrimSpace(config.ClientSecret) == "" || strings.TrimSpace(config.RedirectURL) == "" {
		return nil, fmt.Errorf("OIDC issuer, client ID, client secret, and redirect URL are required")
	}
	discoveryURL := strings.TrimSpace(config.DiscoveryURL)
	if discoveryURL == "" {
		discoveryURL = config.IssuerURL
	}
	discoveryContext := ctx
	if discoveryURL != config.IssuerURL {
		discoveryContext = oidc.InsecureIssuerURLContext(ctx, config.IssuerURL)
	}
	provider, err := oidc.NewProvider(discoveryContext, discoveryURL)
	if err != nil {
		return nil, fmt.Errorf("discover OIDC provider: %w", err)
	}
	var discoveryMetadata struct {
		Issuer string `json:"issuer"`
	}
	if err := provider.Claims(&discoveryMetadata); err != nil {
		return nil, fmt.Errorf("decode OIDC discovery metadata: %w", err)
	}
	if discoveryMetadata.Issuer != config.IssuerURL {
		return nil, fmt.Errorf(
			"OIDC discovery issuer %q does not match configured issuer %q",
			discoveryMetadata.Issuer,
			config.IssuerURL,
		)
	}
	return &RemoteOIDCProvider{
		oauthConfig: oauth2.Config{
			ClientID: config.ClientID, ClientSecret: config.ClientSecret, Endpoint: provider.Endpoint(),
			RedirectURL: config.RedirectURL, Scopes: []string{oidc.ScopeOpenID, "profile", "email"},
		},
		verifier: provider.Verifier(&oidc.Config{ClientID: config.ClientID}),
	}, nil
}

func (provider *RemoteOIDCProvider) AuthorizationURL(state, nonce, pkceChallenge string) string {
	return provider.oauthConfig.AuthCodeURL(
		state,
		oidc.Nonce(nonce),
		oauth2.SetAuthURLParam("code_challenge", pkceChallenge),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"),
	)
}

func (provider *RemoteOIDCProvider) Exchange(ctx context.Context, code, pkceVerifier, expectedNonce string) (OIDCIdentity, error) {
	if strings.TrimSpace(code) == "" || strings.TrimSpace(pkceVerifier) == "" || strings.TrimSpace(expectedNonce) == "" {
		return OIDCIdentity{}, fmt.Errorf("authorization code, PKCE verifier, and expected nonce are required")
	}
	token, err := provider.oauthConfig.Exchange(ctx, code, oauth2.VerifierOption(pkceVerifier))
	if err != nil {
		return OIDCIdentity{}, fmt.Errorf("exchange OIDC authorization code: %w", err)
	}
	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok || rawIDToken == "" {
		return OIDCIdentity{}, fmt.Errorf("OIDC token response omitted id_token")
	}
	verifiedToken, err := provider.verifier.Verify(ctx, rawIDToken)
	if err != nil {
		return OIDCIdentity{}, fmt.Errorf("verify OIDC ID token: %w", err)
	}
	if !constantTimeEqual(verifiedToken.Nonce, expectedNonce) {
		return OIDCIdentity{}, fmt.Errorf("OIDC ID token nonce mismatch")
	}
	var claims struct {
		Name              string   `json:"name"`
		PreferredUsername string   `json:"preferred_username"`
		Email             string   `json:"email"`
		OrganizationID    string   `json:"organization_id"`
		Roles             []string `json:"roles"`
		SID               string   `json:"sid"`
		RealmAccess       struct {
			Roles []string `json:"roles"`
		} `json:"realm_access"`
	}
	if err := verifiedToken.Claims(&claims); err != nil {
		return OIDCIdentity{}, fmt.Errorf("decode verified OIDC claims: %w", err)
	}
	if strings.TrimSpace(verifiedToken.Subject) == "" ||
		strings.TrimSpace(claims.OrganizationID) == "" ||
		!validOIDCEmail(claims.Email) {
		return OIDCIdentity{}, fmt.Errorf(
			"verified OIDC subject, organization_id, and email are required",
		)
	}
	roles := canonicalRoles(append(append([]string(nil), claims.Roles...), claims.RealmAccess.Roles...))
	if err := ValidateApplicationAuthority(
		strings.TrimSpace(claims.OrganizationID),
		roles,
	); err != nil {
		return OIDCIdentity{}, fmt.Errorf(
			"verified OIDC token contains invalid AviaSurveil360 authority: %w",
			err,
		)
	}
	displayName := strings.TrimSpace(claims.Name)
	if displayName == "" {
		displayName = strings.TrimSpace(claims.PreferredUsername)
	}
	if displayName == "" {
		displayName = verifiedToken.Subject
	}
	return OIDCIdentity{
		SubjectID: verifiedToken.Subject, Issuer: verifiedToken.Issuer, DisplayName: displayName,
		Email:          strings.ToLower(strings.TrimSpace(claims.Email)),
		OrganizationID: strings.TrimSpace(claims.OrganizationID), Roles: roles,
		ProviderSessionID: strings.TrimSpace(claims.SID),
		Tokens: ProviderTokens{
			AccessToken: token.AccessToken, RefreshToken: token.RefreshToken, IDToken: rawIDToken, Expiry: token.Expiry,
		},
	}, nil
}

func validOIDCEmail(value string) bool {
	value = strings.TrimSpace(value)
	at := strings.LastIndexByte(value, '@')
	return at > 0 && at < len(value)-1 &&
		!strings.ContainsAny(value, "\r\n\t ,;<>")
}

func canonicalRoles(rawRoles []string) []Role {
	seen := map[Role]bool{}
	roles := make([]Role, 0, len(rawRoles))
	for _, rawRole := range rawRoles {
		role := Role(strings.TrimSpace(rawRole))
		switch role {
		case RoleInspector, RoleLeadInspector, RoleDepartmentManager, RoleGeneralManager,
			RoleFinance, RoleExecutiveDirector, RoleAuditee, RoleAdmin:
			if !seen[role] {
				seen[role] = true
				roles = append(roles, role)
			}
		}
	}
	return roles
}

func constantTimeEqual(actual, expected string) bool {
	if len(actual) != len(expected) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(actual), []byte(expected)) == 1
}
