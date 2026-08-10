package identity

import (
	"context"
	"crypto/subtle"
	"fmt"
	"net/url"
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
	oauthConfig        oauth2.Config
	verifier           *oidc.IDTokenVerifier
	logoutEndpoint     url.URL
	postLogoutRedirect string
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
		Issuer             string `json:"issuer"`
		EndSessionEndpoint string `json:"end_session_endpoint"`
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
	logoutEndpoint, err := validatedOIDCLogoutEndpoint(
		discoveryMetadata.EndSessionEndpoint,
		config.IssuerURL,
	)
	if err != nil {
		return nil, err
	}
	postLogoutRedirect, err := oidcApplicationOrigin(config.RedirectURL)
	if err != nil {
		return nil, err
	}
	return &RemoteOIDCProvider{
		oauthConfig: oauth2.Config{
			ClientID: config.ClientID, ClientSecret: config.ClientSecret, Endpoint: provider.Endpoint(),
			RedirectURL: config.RedirectURL, Scopes: []string{oidc.ScopeOpenID, "profile", "email"},
		},
		verifier:           provider.Verifier(&oidc.Config{ClientID: config.ClientID}),
		logoutEndpoint:     logoutEndpoint,
		postLogoutRedirect: postLogoutRedirect,
	}, nil
}

func (provider *RemoteOIDCProvider) AuthorizationURL(state, nonce, pkceChallenge string) string {
	return provider.oauthConfig.AuthCodeURL(
		state,
		oidc.Nonce(nonce),
		oauth2.SetAuthURLParam("code_challenge", pkceChallenge),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"),
		oauth2.SetAuthURLParam("prompt", "login"),
		oauth2.SetAuthURLParam("max_age", "0"),
	)
}

// LogoutURL returns the provider-discovered RP-initiated logout endpoint.
// The endpoint and post-logout origin are validated when the provider is
// constructed; only the server-held ID-token hint varies per browser session.
func (provider *RemoteOIDCProvider) LogoutURL(idTokenHint string) string {
	logout := provider.logoutEndpoint
	query := logout.Query()
	query.Set("client_id", provider.oauthConfig.ClientID)
	query.Set("post_logout_redirect_uri", provider.postLogoutRedirect)
	if idTokenHint = strings.TrimSpace(idTokenHint); idTokenHint != "" {
		query.Set("id_token_hint", idTokenHint)
	}
	logout.RawQuery = query.Encode()
	return logout.String()
}

func validatedOIDCLogoutEndpoint(rawEndpoint, rawIssuer string) (url.URL, error) {
	endpoint, err := url.Parse(strings.TrimSpace(rawEndpoint))
	if err != nil || !endpoint.IsAbs() || endpoint.User != nil || endpoint.Fragment != "" || endpoint.RawQuery != "" {
		return url.URL{}, fmt.Errorf("OIDC discovery end_session_endpoint is invalid")
	}
	issuer, err := url.Parse(strings.TrimSpace(rawIssuer))
	if err != nil || !issuer.IsAbs() || issuer.User != nil || issuer.Host == "" {
		return url.URL{}, fmt.Errorf("configured OIDC issuer is invalid")
	}
	if endpoint.Scheme != issuer.Scheme || endpoint.Host != issuer.Host || (endpoint.Scheme != "https" && endpoint.Scheme != "http") {
		return url.URL{}, fmt.Errorf("OIDC end_session_endpoint does not match configured issuer origin")
	}
	return *endpoint, nil
}

func oidcApplicationOrigin(rawRedirect string) (string, error) {
	redirect, err := url.Parse(strings.TrimSpace(rawRedirect))
	if err != nil || !redirect.IsAbs() || redirect.User != nil || redirect.Host == "" || redirect.Fragment != "" || (redirect.Scheme != "https" && redirect.Scheme != "http") {
		return "", fmt.Errorf("OIDC redirect URL is invalid")
	}
	return (&url.URL{Scheme: redirect.Scheme, Host: redirect.Host, Path: "/"}).String(), nil
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
