package httpapi_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/httpapi"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/identity"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/platform/session"
)

type fakeOIDCProvider struct {
	authorizationState     string
	authorizationNonce     string
	authorizationChallenge string
	exchangeCode           string
	exchangeVerifier       string
	exchangeNonce          string
	identity               identity.OIDCIdentity
	exchangeErr            error
}

func (provider *fakeOIDCProvider) AuthorizationURL(state, nonce, challenge string) string {
	provider.authorizationState = state
	provider.authorizationNonce = nonce
	provider.authorizationChallenge = challenge
	return "https://identity.example/authorize?state=" + state
}

func (provider *fakeOIDCProvider) Exchange(_ context.Context, code, verifier, nonce string) (identity.OIDCIdentity, error) {
	provider.exchangeCode = code
	provider.exchangeVerifier = verifier
	provider.exchangeNonce = nonce
	return provider.identity, provider.exchangeErr
}

type fakeAuthSessions struct {
	loginRequest       session.LoginRequest
	loginState         session.LoginState
	created            session.BrowserSession
	createInput        session.CreateInput
	principal          identity.Principal
	newLoginReturnTo   string
	consumedState      string
	authenticatedToken string
	csrfSessionID      string
	csrfToken          string
	revokedSessionID   string
	createErr          error
	authenticateErr    error
	csrfErr            error
}

func (sessions *fakeAuthSessions) NewLoginState(_ context.Context, returnTo string) (session.LoginRequest, error) {
	sessions.newLoginReturnTo = returnTo
	return sessions.loginRequest, nil
}

func (sessions *fakeAuthSessions) ConsumeLoginState(_ context.Context, rawState string) (session.LoginState, error) {
	sessions.consumedState = rawState
	return sessions.loginState, nil
}

func (sessions *fakeAuthSessions) Create(_ context.Context, input session.CreateInput) (session.BrowserSession, error) {
	sessions.createInput = input
	return sessions.created, sessions.createErr
}

func (sessions *fakeAuthSessions) Authenticate(_ context.Context, token string) (identity.Principal, error) {
	sessions.authenticatedToken = token
	return sessions.principal, sessions.authenticateErr
}

func (sessions *fakeAuthSessions) ValidateCSRF(_ context.Context, sessionID, rawCSRF string) error {
	sessions.csrfSessionID = sessionID
	sessions.csrfToken = rawCSRF
	return sessions.csrfErr
}

func (sessions *fakeAuthSessions) Revoke(_ context.Context, sessionID string) error {
	sessions.revokedSessionID = sessionID
	return nil
}

func TestOIDCLoginAndCallbackUseOneTimeStatePKCEAndSecureBrowserCookies(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, time.July, 21, 12, 0, 0, 0, time.UTC)
	provider := &fakeOIDCProvider{identity: identity.OIDCIdentity{
		SubjectID: "inspector-001", Issuer: "https://identity.example/realms/avia", DisplayName: "Inspector One",
		Email:          "inspector.one@example.test",
		OrganizationID: "caa", Roles: []identity.Role{identity.RoleInspector}, ProviderSessionID: "provider-session",
		Tokens: identity.ProviderTokens{AccessToken: "server-only-access"},
	}}
	sessions := &fakeAuthSessions{
		loginRequest: session.LoginRequest{State: "raw-state", Nonce: "raw-nonce", PKCEChallenge: "pkce-challenge", ReturnTo: "/findings"},
		loginState:   session.LoginState{Nonce: "raw-nonce", PKCEVerifier: "pkce-verifier", ReturnTo: "/findings"},
		created: session.BrowserSession{
			ID: "session-001", Token: "opaque-session-token", CSRFToken: "opaque-csrf-token",
			ExpiresAt: now.Add(30 * time.Minute), AbsoluteExpiresAt: now.Add(8 * time.Hour),
		},
	}
	handler := httpapi.NewAuthHandler(provider, sessions)

	login := httptest.NewRecorder()
	handler.ServeHTTP(login, httptest.NewRequest(http.MethodGet, "/auth/login?returnTo=%2Ffindings", nil))
	if login.Code != http.StatusFound || login.Header().Get("Location") != "https://identity.example/authorize?state=raw-state" {
		t.Fatalf("login response = %d, location %q", login.Code, login.Header().Get("Location"))
	}
	if sessions.newLoginReturnTo != "/findings" || provider.authorizationState != "raw-state" || provider.authorizationNonce != "raw-nonce" || provider.authorizationChallenge != "pkce-challenge" {
		t.Fatalf("login authority = sessions %q provider %+v", sessions.newLoginReturnTo, provider)
	}

	callback := httptest.NewRecorder()
	handler.ServeHTTP(callback, httptest.NewRequest(http.MethodGet, "/auth/callback?state=raw-state&code=authorization-code", nil))
	if callback.Code != http.StatusFound || callback.Header().Get("Location") != "/findings" {
		t.Fatalf("callback response = %d, location %q, body %s", callback.Code, callback.Header().Get("Location"), callback.Body.String())
	}
	if sessions.consumedState != "raw-state" || provider.exchangeCode != "authorization-code" || provider.exchangeVerifier != "pkce-verifier" || provider.exchangeNonce != "raw-nonce" {
		t.Fatalf("callback exchange = provider %+v, consumed %q", provider, sessions.consumedState)
	}
	if sessions.createInput.ProviderTokens.AccessToken != "server-only-access" ||
		sessions.createInput.SubjectID != "inspector-001" ||
		sessions.createInput.Email != "inspector.one@example.test" {
		t.Fatalf("session creation input = %+v", sessions.createInput)
	}

	cookies := callback.Result().Cookies()
	if len(cookies) != 2 {
		t.Fatalf("callback cookies = %+v", cookies)
	}
	byName := map[string]*http.Cookie{}
	for _, cookie := range cookies {
		byName[cookie.Name] = cookie
	}
	sessionCookie := byName[httpapi.SessionCookieName]
	csrfCookie := byName[httpapi.CSRFCookieName]
	if sessionCookie == nil || sessionCookie.Value != "opaque-session-token" || !sessionCookie.Secure || !sessionCookie.HttpOnly || sessionCookie.SameSite != http.SameSiteStrictMode || sessionCookie.Path != "/" {
		t.Fatalf("session cookie = %+v", sessionCookie)
	}
	if !sessionCookie.Expires.IsZero() || sessionCookie.MaxAge != 0 {
		t.Fatalf("rolling server session must use a browser-session cookie: %+v", sessionCookie)
	}
	if csrfCookie == nil || csrfCookie.Value != "opaque-csrf-token" || !csrfCookie.Secure || csrfCookie.HttpOnly || csrfCookie.SameSite != http.SameSiteStrictMode || csrfCookie.Path != "/" {
		t.Fatalf("CSRF cookie = %+v", csrfCookie)
	}
	if !csrfCookie.Expires.IsZero() || csrfCookie.MaxAge != 0 {
		t.Fatalf("rolling server session must use a browser-session CSRF cookie: %+v", csrfCookie)
	}
}

func TestSessionProjectionNeverReturnsCredentialsAndLogoutRequiresCSRF(t *testing.T) {
	t.Parallel()
	principal := identity.Principal{
		SubjectID: "auditee-xyz", OrganizationID: "airline-xyz", SessionID: "session-auditee",
		DisplayName: "Fly Namibia Auditee",
		Roles:       []identity.Role{identity.RoleAuditee},
	}
	provider := &fakeOIDCProvider{}
	sessions := &fakeAuthSessions{principal: principal}
	handler := httpapi.NewAuthHandler(provider, sessions)

	request := httptest.NewRequest(http.MethodGet, "/auth/session", nil)
	request.AddCookie(&http.Cookie{Name: httpapi.SessionCookieName, Value: "raw-browser-token"})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("session response = %d, %s", response.Code, response.Body.String())
	}
	if strings.Contains(response.Body.String(), "raw-browser-token") || strings.Contains(strings.ToLower(response.Body.String()), "csrf") || strings.Contains(strings.ToLower(response.Body.String()), "provider") {
		t.Fatalf("session response exposed credentials: %s", response.Body.String())
	}
	var projection struct {
		SubjectID      string          `json:"subjectId"`
		DisplayName    string          `json:"displayName"`
		OrganizationID string          `json:"organizationId"`
		Roles          []identity.Role `json:"roles"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &projection); err != nil || projection.SubjectID != principal.SubjectID || projection.DisplayName != principal.DisplayName || projection.OrganizationID != principal.OrganizationID || len(projection.Roles) != 1 {
		t.Fatalf("session projection = %+v, err = %v", projection, err)
	}

	missingCSRF := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	missingCSRF.AddCookie(&http.Cookie{Name: httpapi.SessionCookieName, Value: "raw-browser-token"})
	missingResponse := httptest.NewRecorder()
	handler.ServeHTTP(missingResponse, missingCSRF)
	if missingResponse.Code != http.StatusForbidden || sessions.revokedSessionID != "" {
		t.Fatalf("missing CSRF logout = %d, revoked %q", missingResponse.Code, sessions.revokedSessionID)
	}

	logout := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	logout.AddCookie(&http.Cookie{Name: httpapi.SessionCookieName, Value: "raw-browser-token"})
	logout.AddCookie(&http.Cookie{Name: httpapi.CSRFCookieName, Value: "raw-csrf-token"})
	logout.Header.Set(httpapi.CSRFHeaderName, "raw-csrf-token")
	logoutResponse := httptest.NewRecorder()
	handler.ServeHTTP(logoutResponse, logout)
	if logoutResponse.Code != http.StatusNoContent || sessions.revokedSessionID != principal.SessionID || sessions.csrfSessionID != principal.SessionID || sessions.csrfToken != "raw-csrf-token" {
		t.Fatalf("logout response = %d, sessions %+v", logoutResponse.Code, sessions)
	}
	for _, cookie := range logoutResponse.Result().Cookies() {
		if cookie.MaxAge >= 0 || !cookie.Expires.Before(time.Now()) {
			t.Fatalf("logout cookie was not expired: %+v", cookie)
		}
	}
}

func TestAuthenticationFailuresUseClosedProblemResponses(t *testing.T) {
	t.Parallel()
	sessions := &fakeAuthSessions{authenticateErr: session.ErrUnauthenticated}
	handler := httpapi.NewAuthHandler(&fakeOIDCProvider{exchangeErr: errors.New("invalid ID token")}, sessions)

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/auth/session", nil))
	if response.Code != http.StatusUnauthorized || response.Header().Get("Content-Type") != "application/problem+json" {
		t.Fatalf("unauthenticated response = %d %q", response.Code, response.Header().Get("Content-Type"))
	}
}

func TestTask4OIDCCallbackRejectsStaleAuthorityAndExpiresCookies(t *testing.T) {
	t.Parallel()
	provider := &fakeOIDCProvider{
		identity: identity.OIDCIdentity{
			SubjectID:      "stale-inspector",
			Issuer:         "https://identity.example/realms/avia",
			DisplayName:    "Stale Inspector",
			OrganizationID: "CAA",
			Roles:          []identity.Role{identity.RoleInspector},
		},
	}
	sessions := &fakeAuthSessions{
		loginState: session.LoginState{
			Nonce:        "nonce",
			PKCEVerifier: "verifier",
			ReturnTo:     "/",
		},
		createErr: session.ErrUnauthenticated,
	}
	response := httptest.NewRecorder()
	httpapi.NewAuthHandler(provider, sessions).ServeHTTP(
		response,
		httptest.NewRequest(
			http.MethodGet,
			"/auth/callback?state=state&code=code",
			nil,
		),
	)
	if response.Code != http.StatusUnauthorized ||
		!strings.Contains(response.Body.String(), `"code":"STALE_AUTHORITY"`) {
		t.Fatalf(
			"stale-authority callback = %d, %s",
			response.Code,
			response.Body.String(),
		)
	}
	cookies := response.Result().Cookies()
	if len(cookies) != 2 {
		t.Fatalf("expired callback cookies = %+v", cookies)
	}
	for _, cookie := range cookies {
		if cookie.MaxAge != -1 || !cookie.Expires.Before(time.Now()) {
			t.Fatalf("callback cookie was not expired: %+v", cookie)
		}
	}
}
