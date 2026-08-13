package provider

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/MarlonJD/aviaSurveil360/apps/auth/internal/throttle"
	jose "github.com/go-jose/go-jose/v4"
	"github.com/zitadel/oidc/v3/pkg/oidc"
)

func TestBrowserBindingRejectsForgedCookieAndAcceptsServerSignature(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "https://identity.example.invalid/authorize", nil)
	forged := base64.RawURLEncoding.EncodeToString(make([]byte, 32)) + "." + base64.RawURLEncoding.EncodeToString(make([]byte, sha256.Size))
	request.AddCookie(&http.Cookie{Name: "__Host-avia_login", Value: forged})
	if got := browserBinding(request, "client-secret"); got != "" {
		t.Fatalf("forged browser binding accepted: %q", got)
	}

	raw := []byte("01234567890123456789012345678901")
	keyHash := sha256.New()
	_, _ = keyHash.Write([]byte("as360-oidc-login-binding-key-v1\x00"))
	_, _ = keyHash.Write([]byte("client-secret"))
	mac := hmac.New(sha256.New, keyHash.Sum(nil))
	_, _ = mac.Write([]byte("as360-oidc-login-binding-v1\x00"))
	_, _ = mac.Write(raw)
	valid := base64.RawURLEncoding.EncodeToString(raw) + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	request = httptest.NewRequest(http.MethodGet, "https://identity.example.invalid/authorize", nil)
	request.AddCookie(&http.Cookie{Name: "__Host-avia_login", Value: valid})
	if got := browserBinding(request, "client-secret"); got != valid {
		t.Fatalf("server-signed browser binding = %q, want %q", got, valid)
	}
}

func TestMemoryProviderAdmissionSeparatesBrowserBindingsAndCapsOutstanding(t *testing.T) {
	limiter, err := throttle.NewMemoryLimiter(time.Minute, 100, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	storage := NewMemoryStorage(CandidateConfig{
		ClientID: "candidate-web",
		AdmissionPolicy: AdmissionPolicy{
			Window: time.Minute, GlobalLimit: 100, ClientLimit: 100, BrowserLimit: 1, RequestLimit: 100, OutstandingLimit: 2,
		},
		Limiter: limiter,
	})
	request := func(state string) *oidc.AuthRequest {
		return &oidc.AuthRequest{ClientID: "candidate-web", RedirectURI: "https://app.example.invalid/callback", ResponseType: oidc.ResponseTypeCode, ResponseMode: oidc.ResponseModeQuery, Scopes: []string{oidc.ScopeOpenID}, State: state, Nonce: "nonce-" + state}
	}
	if _, err := storage.CreateAuthRequest(WithBrowserBinding(context.Background(), "browser-a"), request("one"), ""); err != nil {
		t.Fatal(err)
	}
	if _, err := storage.CreateAuthRequest(WithBrowserBinding(context.Background(), "browser-a"), request("two"), ""); !errors.Is(err, ErrProviderRateLimited) {
		t.Fatalf("same browser admission = %v", err)
	}
	if _, err := storage.CreateAuthRequest(WithBrowserBinding(context.Background(), "browser-b"), request("three"), ""); err != nil {
		t.Fatal(err)
	}
	if _, err := storage.CreateAuthRequest(WithBrowserBinding(context.Background(), "browser-c"), request("four"), ""); !errors.Is(err, ErrProviderRateLimited) {
		t.Fatalf("outstanding cap admission = %v", err)
	}
}

func TestMemoryProviderAdmissionReservesAnonymousOutstandingCapacity(t *testing.T) {
	limiter, err := throttle.NewMemoryLimiter(time.Minute, 100, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	storage := NewMemoryStorage(CandidateConfig{
		ClientID: "candidate-web",
		AdmissionPolicy: AdmissionPolicy{
			Window: time.Minute, GlobalLimit: 100, ClientLimit: 100, BrowserLimit: 100, RequestLimit: 100,
			OutstandingLimit: 2, AnonymousOutstandingLimit: 1,
		},
		Limiter: limiter,
	})
	request := func(state string) *oidc.AuthRequest {
		return &oidc.AuthRequest{ClientID: "candidate-web", RedirectURI: "https://app.example.invalid/callback", ResponseType: oidc.ResponseTypeCode, ResponseMode: oidc.ResponseModeQuery, Scopes: []string{oidc.ScopeOpenID}, State: state, Nonce: "nonce-" + state}
	}
	if _, err := storage.CreateAuthRequest(context.Background(), request("anonymous-one"), ""); err != nil {
		t.Fatal(err)
	}
	if _, err := storage.CreateAuthRequest(context.Background(), request("anonymous-two"), ""); !errors.Is(err, ErrProviderRateLimited) {
		t.Fatalf("anonymous bootstrap cap admission = %v", err)
	}
	bound, err := storage.CreateAuthRequest(WithBrowserBinding(context.Background(), "browser-bound"), request("bound-one"), "")
	if err != nil {
		t.Fatalf("bound capacity was not reserved = %v", err)
	}
	if _, err := storage.CreateAuthRequest(WithBrowserBinding(context.Background(), "browser-bound"), request("bound-two"), ""); !errors.Is(err, ErrProviderRateLimited) {
		t.Fatalf("outstanding cap after anonymous and bound state = %v", err)
	}
	if err := storage.DeleteAuthRequest(context.Background(), bound.GetID()); err != nil {
		t.Fatalf("delete bound request: %v", err)
	}
	if _, err := storage.CreateAuthRequest(WithBrowserBinding(context.Background(), "browser-bound"), request("bound-three"), ""); err != nil {
		t.Fatalf("capacity was not released after deleting bound request: %v", err)
	}
}

func TestMFARedirectRetainsIssuerPath(t *testing.T) {
	runtime := &runtimeLogin{issuer: "https://identity.example.invalid/identity"}
	if got := runtime.mfaPath("req_123"); got != "/identity/mfa?id=req_123" {
		t.Fatalf("issuer-prefixed MFA path = %q", got)
	}
	if got := (&runtimeLogin{issuer: "https://identity.example.invalid"}).mfaPath("req_123"); got != "/mfa?id=req_123" {
		t.Fatalf("root MFA path = %q", got)
	}
}

func TestRecoveryEndpointRetainsIssuerURL(t *testing.T) {
	if got := providerAbsoluteEndpoint("https://identity.example.invalid/identity", "recover/password"); got != "https://identity.example.invalid/identity/recover/password" {
		t.Fatalf("absolute recovery endpoint = %q", got)
	}
}

func TestMemoryStorageStoresOnlyOneAuthorizationCodePerRequest(t *testing.T) {
	storage := NewMemoryStorage(CandidateConfig{SubjectID: "subject", AdmissionPolicy: DefaultAdmissionPolicy()})
	request := &oidc.AuthRequest{ClientID: "client", RedirectURI: "https://app.example.invalid/callback", ResponseType: oidc.ResponseTypeCode, ResponseMode: oidc.ResponseModeQuery, Scopes: []string{oidc.ScopeOpenID}, State: "state", Nonce: "nonce"}
	authRequest, err := storage.CreateAuthRequest(context.Background(), request, "subject")
	if err != nil {
		t.Fatalf("create auth request: %v", err)
	}
	if err := storage.Authorize(authRequest.GetID()); err != nil {
		t.Fatalf("authorize request: %v", err)
	}
	if err := storage.SaveAuthCode(context.Background(), authRequest.GetID(), "code-one"); err != nil {
		t.Fatalf("save first authorization code: %v", err)
	}
	if err := storage.SaveAuthCode(context.Background(), authRequest.GetID(), "code-two"); !errors.Is(err, ErrProviderInvalid) {
		t.Fatalf("second authorization code error = %v", err)
	}
}

type providerTestServer struct {
	server    *httptest.Server
	candidate *Candidate
	issuer    string
}

func newProviderTestServer(t *testing.T) providerTestServer {
	t.Helper()
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate provider signing key: %v", err)
	}
	listenerServer := httptest.NewUnstartedServer(nil)
	issuer := "http://" + listenerServer.Listener.Addr().String()
	var cryptoKey [32]byte
	for index := range cryptoKey {
		cryptoKey[index] = byte(index + 1)
	}
	candidate, err := NewCandidate(CandidateConfig{
		Issuer: issuer, AllowInsecure: true, CryptoKey: cryptoKey, CryptoKeyID: "crypto-key-2026",
		SigningKey: privateKey, SigningKeyID: "signing-key-2026",
		ClientID: "as360-web", ClientSecret: "client-secret-2026",
		RedirectURI:           "https://app.example.invalid/oidc/callback",
		PostLogoutRedirectURI: "https://app.example.invalid/",
		SubjectID:             "usr_0123456789012345678901", Email: "pilot@example.invalid", DisplayName: "Pilot User",
	})
	if err != nil {
		t.Fatalf("new candidate provider: %v", err)
	}
	listenerServer.Config.Handler = candidate.Handler
	listenerServer.Start()
	t.Cleanup(listenerServer.Close)
	return providerTestServer{server: listenerServer, candidate: candidate, issuer: issuer}
}

func TestCandidateProviderDiscoveryJWKSAndAuthorizationCodePKCE(t *testing.T) {
	testServer := newProviderTestServer(t)
	client := &http.Client{CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }}

	discoveryResponse := mustRequest(t, client, http.MethodGet, testServer.issuer+"/.well-known/openid-configuration", nil, "")
	if discoveryResponse.StatusCode != http.StatusOK {
		t.Fatalf("discovery status = %d", discoveryResponse.StatusCode)
	}
	var discovery struct {
		Issuer                string   `json:"issuer"`
		AuthorizationEndpoint string   `json:"authorization_endpoint"`
		TokenEndpoint         string   `json:"token_endpoint"`
		JWKSURI               string   `json:"jwks_uri"`
		EndSessionEndpoint    string   `json:"end_session_endpoint"`
		CodeChallengeMethods  []string `json:"code_challenge_methods_supported"`
		IDTokenAlgorithms     []string `json:"id_token_signing_alg_values_supported"`
	}
	decodeJSON(t, discoveryResponse, &discovery)
	if discovery.Issuer != testServer.issuer || discovery.AuthorizationEndpoint != testServer.issuer+"/authorize" || discovery.TokenEndpoint != testServer.issuer+"/oauth/token" || discovery.JWKSURI != testServer.issuer+"/keys" || discovery.EndSessionEndpoint != testServer.issuer+"/end_session" {
		t.Fatalf("discovery endpoints = %+v", discovery)
	}
	if !contains(discovery.CodeChallengeMethods, "S256") || !contains(discovery.IDTokenAlgorithms, "RS256") {
		t.Fatalf("discovery security capabilities = %+v", discovery)
	}

	jwksResponse := mustRequest(t, client, http.MethodGet, testServer.issuer+"/keys", nil, "")
	if jwksResponse.StatusCode != http.StatusOK {
		t.Fatalf("jwks status = %d", jwksResponse.StatusCode)
	}
	var jwks jose.JSONWebKeySet
	decodeJSON(t, jwksResponse, &jwks)
	if len(jwks.Keys) != 1 || jwks.Keys[0].KeyID != "signing-key-2026" || jwks.Keys[0].Algorithm != string(jose.RS256) || jwks.Keys[0].Use != "sig" {
		t.Fatalf("jwks = %+v", jwks.Keys)
	}

	verifier := "verifier-for-as360-pkce-2026"
	authorizeURL := testServer.issuer + "/authorize?" + url.Values{
		"client_id": {"as360-web"}, "redirect_uri": {"https://app.example.invalid/oidc/callback"},
		"response_type": {"code"}, "scope": {"openid profile email"},
		"state": {"state-2026"}, "nonce": {"nonce-2026"},
		"code_challenge": {oidc.NewSHACodeChallenge(verifier)}, "code_challenge_method": {"S256"},
	}.Encode()
	authorizeResponse := mustRequest(t, client, http.MethodGet, authorizeURL, nil, "")
	if authorizeResponse.StatusCode != http.StatusFound || !strings.HasPrefix(authorizeResponse.Header.Get("Location"), "/login?id=") {
		t.Fatalf("authorize response = %d %q", authorizeResponse.StatusCode, authorizeResponse.Header.Get("Location"))
	}
	loginResponse := mustRequest(t, client, http.MethodGet, testServer.server.URL+authorizeResponse.Header.Get("Location"), nil, "")
	if loginResponse.StatusCode != http.StatusFound || !strings.Contains(loginResponse.Header.Get("Location"), "/callback?id=") {
		t.Fatalf("login response = %d %q", loginResponse.StatusCode, loginResponse.Header.Get("Location"))
	}
	callbackResponse := mustRequest(t, client, http.MethodGet, testServer.server.URL+loginResponse.Header.Get("Location"), nil, "")
	if callbackResponse.StatusCode != http.StatusFound {
		t.Fatalf("callback response = %d", callbackResponse.StatusCode)
	}
	redirected, err := url.Parse(callbackResponse.Header.Get("Location"))
	if err != nil {
		t.Fatalf("parse code redirect: %v", err)
	}
	if redirected.Host != "app.example.invalid" || redirected.Query().Get("state") != "state-2026" || redirected.Query().Get("code") == "" {
		t.Fatalf("code redirect = %s", redirected.String())
	}
	code := redirected.Query().Get("code")

	form := url.Values{"grant_type": {"authorization_code"}, "code": {code}, "redirect_uri": {"https://app.example.invalid/oidc/callback"}, "client_id": {"as360-web"}, "code_verifier": {verifier}}
	tokenResponse := mustAuthenticatedPost(t, client, testServer.issuer+"/oauth/token", form, "as360-web", "client-secret-2026")
	if tokenResponse.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(tokenResponse.Body)
		t.Fatalf("token status = %d body=%s", tokenResponse.StatusCode, body)
	}
	var tokens struct {
		AccessToken  string `json:"access_token"`
		IDToken      string `json:"id_token"`
		RefreshToken string `json:"refresh_token"`
		TokenType    string `json:"token_type"`
	}
	decodeJSON(t, tokenResponse, &tokens)
	if tokens.AccessToken == "" || tokens.IDToken == "" || tokens.RefreshToken != "" || tokens.TokenType != "Bearer" {
		t.Fatalf("token response = %+v", tokens)
	}
	verifyCandidateIDToken(t, testServer.candidate, tokens.IDToken, "nonce-2026")
	replayedCodeResponse := mustAuthenticatedPost(t, client, testServer.issuer+"/oauth/token", form, "as360-web", "client-secret-2026")
	if replayedCodeResponse.StatusCode < http.StatusBadRequest || replayedCodeResponse.StatusCode >= http.StatusInternalServerError {
		t.Fatalf("replayed authorization code status = %d", replayedCodeResponse.StatusCode)
	}

	refreshForm := url.Values{"grant_type": {"refresh_token"}, "refresh_token": {"refresh-token-disabled"}}
	refreshResponse := mustAuthenticatedPost(t, client, testServer.issuer+"/oauth/token", refreshForm, "as360-web", "client-secret-2026")
	if refreshResponse.StatusCode < http.StatusBadRequest || refreshResponse.StatusCode >= http.StatusInternalServerError {
		t.Fatalf("disabled refresh status = %d", refreshResponse.StatusCode)
	}
	logoutURL := testServer.issuer + "/end_session?" + url.Values{
		"post_logout_redirect_uri": {"https://app.example.invalid/"},
		"client_id":                {"as360-web"},
	}.Encode()
	logoutResponse := mustRequest(t, client, http.MethodGet, logoutURL, nil, "")
	if logoutResponse.StatusCode != http.StatusFound || logoutResponse.Header.Get("Location") != "https://app.example.invalid/" {
		t.Fatalf("logout response = %d %q", logoutResponse.StatusCode, logoutResponse.Header.Get("Location"))
	}
}

func TestCandidateProviderRejectsRedirectAndPKCENegatives(t *testing.T) {
	testServer := newProviderTestServer(t)
	client := &http.Client{CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }}
	base := url.Values{
		"client_id": {"as360-web"}, "response_type": {"code"}, "scope": {"openid"},
		"state": {"state"}, "nonce": {"nonce"}, "code_challenge": {oidc.NewSHACodeChallenge("expected-verifier")}, "code_challenge_method": {"S256"},
	}
	wrongRedirect := cloneValues(base)
	wrongRedirect.Set("redirect_uri", "https://attacker.example.invalid/callback")
	response := mustRequest(t, client, http.MethodGet, testServer.issuer+"/authorize?"+wrongRedirect.Encode(), nil, "")
	if response.StatusCode < http.StatusBadRequest || response.StatusCode >= http.StatusInternalServerError || strings.Contains(response.Header.Get("Location"), "attacker.example.invalid") {
		t.Fatalf("wrong redirect response = %d %q", response.StatusCode, response.Header.Get("Location"))
	}

	missingPKCE := cloneValues(base)
	missingPKCE.Set("redirect_uri", "https://app.example.invalid/oidc/callback")
	missingPKCE.Del("code_challenge")
	missingPKCE.Del("code_challenge_method")
	response = mustRequest(t, client, http.MethodGet, testServer.issuer+"/authorize?"+missingPKCE.Encode(), nil, "")
	if response.StatusCode < http.StatusBadRequest || response.StatusCode >= http.StatusInternalServerError {
		t.Fatalf("missing PKCE response = %d", response.StatusCode)
	}
}

func verifyCandidateIDToken(t *testing.T, candidate *Candidate, raw, expectedNonce string) {
	t.Helper()
	parsed, err := jose.ParseSigned(raw, []jose.SignatureAlgorithm{jose.RS256})
	if err != nil {
		t.Fatalf("parse ID token: %v", err)
	}
	payload, err := parsed.Verify(candidate.Storage.signingKey.private.Public())
	if err != nil {
		t.Fatalf("verify ID token: %v", err)
	}
	var claims struct {
		Issuer    string   `json:"iss"`
		Subject   string   `json:"sub"`
		Audience  []string `json:"aud"`
		Nonce     string   `json:"nonce"`
		Algorithm string   `json:"alg"`
	}
	if err := json.Unmarshal(payload, &claims); err != nil {
		t.Fatalf("decode ID token: %v", err)
	}
	if claims.Issuer == "" || claims.Subject == "" || !contains(claims.Audience, "as360-web") || claims.Nonce != expectedNonce || len(parsed.Signatures) != 1 || parsed.Signatures[0].Protected.Algorithm != string(jose.RS256) {
		t.Fatalf("ID token claims = %+v", claims)
	}
}

func mustRequest(t *testing.T, client *http.Client, method, endpoint string, body io.Reader, contentType string) *http.Response {
	t.Helper()
	request, err := http.NewRequest(method, endpoint, body)
	if err != nil {
		t.Fatalf("new HTTP request: %v", err)
	}
	if contentType != "" {
		request.Header.Set("Content-Type", contentType)
	}
	response, err := client.Do(request)
	if err != nil {
		t.Fatalf("HTTP request %s %s: %v", method, endpoint, err)
	}
	return response
}

func mustAuthenticatedPost(t *testing.T, client *http.Client, endpoint string, values url.Values, clientID, secret string) *http.Response {
	t.Helper()
	request, err := http.NewRequest(http.MethodPost, endpoint, strings.NewReader(values.Encode()))
	if err != nil {
		t.Fatalf("new authenticated token request: %v", err)
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.SetBasicAuth(clientID, secret)
	response, err := client.Do(request)
	if err != nil {
		t.Fatalf("authenticated token request: %v", err)
	}
	return response
}

func decodeJSON(t *testing.T, response *http.Response, target any) {
	t.Helper()
	defer response.Body.Close()
	if err := json.NewDecoder(response.Body).Decode(target); err != nil {
		t.Fatalf("decode JSON response: %v", err)
	}
}

func cloneValues(values url.Values) url.Values {
	clone := make(url.Values, len(values))
	for key, list := range values {
		clone[key] = append([]string(nil), list...)
	}
	return clone
}

func contains(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}
