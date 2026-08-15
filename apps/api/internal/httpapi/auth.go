package httpapi

import (
	"context"
	"crypto/subtle"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/aviason/aviaSurveil/internal/identity"
	"github.com/aviason/aviaSurveil/internal/platform/session"
	"github.com/go-chi/chi/v5"
)

const (
	SessionCookieName = "__Host-avia_session"
	CSRFCookieName    = "__Host-avia_csrf"
	CSRFHeaderName    = "X-CSRF-Token"

	localSessionCookieName = "avia_session"
	localCSRFCookieName    = "avia_csrf"
	LoginCookieName        = "__Host-avia_login"
	localLoginCookieName   = "avia_login"
	loginCookieTTL         = 10 * time.Minute
)

type AuthSessionManager interface {
	NewLoginState(context.Context, string, string) (session.LoginRequest, error)
	ConsumeLoginState(context.Context, string, string) (session.LoginState, error)
	Create(context.Context, session.CreateInput) (session.BrowserSession, error)
	Authenticate(context.Context, string) (identity.Principal, error)
	ValidateCSRF(context.Context, string, string) error
	Revoke(context.Context, string) (string, error)
	RedeemProviderLogout(context.Context, string) (string, error)
}

type AuthBoundary struct {
	provider             identity.OIDCProvider
	sessions             AuthSessionManager
	cookieSecure         bool
	alwaysFreshAuthority bool
}

func NewAuthBoundary(provider identity.OIDCProvider, sessions AuthSessionManager) *AuthBoundary {
	return NewAuthBoundaryWithCookieSecure(provider, sessions, true)
}

// NewAuthBoundaryWithCookieSecure keeps the cookie transport policy explicit.
// Connected demo and production profiles use secure cookies; test profiles
// may provide an explicit loopback policy through their own harness.
func NewAuthBoundaryWithCookieSecure(
	provider identity.OIDCProvider,
	sessions AuthSessionManager,
	cookieSecure bool,
) *AuthBoundary {
	return &AuthBoundary{provider: provider, sessions: sessions, cookieSecure: cookieSecure}
}

// RequireFreshAuthorityOnEveryRequest is enabled only by the disposable
// canonical first-party demo. It makes lifecycle changes observable on every
// protected request instead of waiting for the normal heartbeat.
func (boundary *AuthBoundary) RequireFreshAuthorityOnEveryRequest() {
	boundary.alwaysFreshAuthority = true
}

func NewAuthHandler(provider identity.OIDCProvider, sessions AuthSessionManager) http.Handler {
	return NewAuthBoundary(provider, sessions).Handler()
}

func (boundary *AuthBoundary) Handler() http.Handler {
	router := chi.NewRouter()
	router.Get("/auth/login", boundary.login)
	router.Get("/auth/callback", boundary.callback)
	router.Get("/auth/session", boundary.sessionProjection)
	router.Post("/auth/logout", boundary.logout)
	router.Get("/auth/provider-logout", boundary.providerLogout)
	return router
}

func (boundary *AuthBoundary) Protect(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		principal, ok := boundary.authenticate(writer, request)
		if !ok {
			return
		}
		if isMutation(request.Method) && !boundary.validateCSRF(writer, request, principal.SessionID) {
			return
		}
		ctx := context.WithValue(request.Context(), principalContextKey{}, principal)
		next.ServeHTTP(writer, request.WithContext(ctx))
	})
}

// ProtectReadOnlyNeutral preserves the existing OIDC/session authority check
// while replacing its unauthenticated transport response with a caller-owned
// neutral denial. It is intentionally suitable only for read-only capability
// surfaces whose existence must not be disclosed.
func (boundary *AuthBoundary) ProtectReadOnlyNeutral(next http.Handler, deny func(http.ResponseWriter)) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		sink := &discardResponseWriter{header: make(http.Header)}
		principal, ok := boundary.authenticate(sink, request)
		if !ok {
			deny(writer)
			return
		}
		ctx := context.WithValue(request.Context(), principalContextKey{}, principal)
		next.ServeHTTP(writer, request.WithContext(ctx))
	})
}

type discardResponseWriter struct{ header http.Header }

func (writer *discardResponseWriter) Header() http.Header        { return writer.header }
func (*discardResponseWriter) WriteHeader(int)                   {}
func (*discardResponseWriter) Write(payload []byte) (int, error) { return len(payload), nil }

func PrincipalFromContext(ctx context.Context) (identity.Principal, bool) {
	principal, ok := ctx.Value(principalContextKey{}).(identity.Principal)
	return principal, ok
}

func (boundary *AuthBoundary) login(writer http.ResponseWriter, request *http.Request) {
	if boundary.provider == nil || boundary.sessions == nil {
		writeProblem(writer, http.StatusServiceUnavailable, "Authentication unavailable", "OIDC authentication is not configured", "AUTH_UNAVAILABLE")
		return
	}
	login, err := boundary.sessions.NewLoginState(request.Context(), request.URL.Query().Get("returnTo"), boundary.loginBindingCookie(request))
	if err != nil {
		if errors.Is(err, session.ErrLoginRateLimited) {
			writer.Header().Set("Retry-After", "60")
			writeProblem(writer, http.StatusTooManyRequests, "Too many requests", "login admission limit reached", "RATE_LIMITED")
			return
		}
		writeProblem(writer, http.StatusServiceUnavailable, "Authentication unavailable", "login state could not be created", "AUTH_UNAVAILABLE")
		return
	}
	boundary.setLoginBindingCookie(writer, login.BrowserBinding)
	location := boundary.provider.AuthorizationURL(login.State, login.Nonce, login.PKCEChallenge)
	http.Redirect(writer, request, location, http.StatusFound)
}

func (boundary *AuthBoundary) callback(writer http.ResponseWriter, request *http.Request) {
	if boundary.provider == nil || boundary.sessions == nil {
		writeProblem(writer, http.StatusServiceUnavailable, "Authentication unavailable", "OIDC authentication is not configured", "AUTH_UNAVAILABLE")
		return
	}
	stateValue := strings.TrimSpace(request.URL.Query().Get("state"))
	code := strings.TrimSpace(request.URL.Query().Get("code"))
	if stateValue == "" || code == "" {
		writeProblem(writer, http.StatusBadRequest, "Invalid authentication response", "state and authorization code are required", "INVALID_OIDC_CALLBACK")
		return
	}
	loginState, err := boundary.sessions.ConsumeLoginState(request.Context(), stateValue, boundary.loginBindingCookie(request))
	if err != nil {
		writeProblem(writer, http.StatusUnauthorized, "Authentication failed", "OIDC state is invalid or expired", "INVALID_OIDC_STATE")
		return
	}
	authenticated, err := boundary.provider.Exchange(request.Context(), code, loginState.PKCEVerifier, loginState.Nonce)
	if err != nil {
		slog.Warn("OIDC callback rejected", "reason", err)
		writeProblem(writer, http.StatusUnauthorized, "Authentication failed", "OIDC token verification failed", "INVALID_OIDC_TOKEN")
		return
	}
	browserSession, err := boundary.sessions.Create(request.Context(), session.CreateInput{
		SubjectID: authenticated.SubjectID, Issuer: authenticated.Issuer, DisplayName: authenticated.DisplayName,
		Email:          authenticated.Email,
		OrganizationID: authenticated.OrganizationID, Roles: authenticated.Roles,
		ProviderSessionID: authenticated.ProviderSessionID, ProviderTokens: authenticated.Tokens,
		MembershipID: authenticated.MembershipID, MembershipRevision: authenticated.MembershipRevision,
		AuthRevision: authenticated.AuthRevision,
	})
	if err != nil {
		if errors.Is(err, session.ErrUnauthenticated) {
			slog.Warn(
				"OIDC browser session admission rejected",
				slog.String("diagnostic", session.AuthenticationFailureDiagnostic(err)),
			)
			boundary.expireBrowserSessionCookies(writer)
			writeProblem(
				writer,
				http.StatusUnauthorized,
				"Authentication failed",
				"provider and application authority are stale or invalid",
				"STALE_AUTHORITY",
			)
			return
		}
		writeProblem(writer, http.StatusInternalServerError, "Authentication failed", "browser session could not be created", "SESSION_CREATE_FAILED")
		return
	}
	boundary.setBrowserSessionCookies(writer, browserSession)
	http.Redirect(writer, request, loginState.ReturnTo, http.StatusFound)
}

func (boundary *AuthBoundary) sessionProjection(writer http.ResponseWriter, request *http.Request) {
	principal, ok := boundary.authenticate(writer, request)
	if !ok {
		return
	}
	writeJSON(writer, http.StatusOK, struct {
		SubjectID      string          `json:"subjectId"`
		DisplayName    string          `json:"displayName"`
		OrganizationID string          `json:"organizationId"`
		Roles          []identity.Role `json:"roles"`
	}{SubjectID: principal.SubjectID, DisplayName: principal.DisplayName, OrganizationID: principal.OrganizationID, Roles: principal.Roles})
}

func (boundary *AuthBoundary) logout(writer http.ResponseWriter, request *http.Request) {
	principal, ok := boundary.authenticate(writer, request)
	if !ok {
		return
	}
	if !boundary.validateCSRF(writer, request, principal.SessionID) {
		return
	}
	if boundary.provider == nil {
		writeProblem(writer, http.StatusServiceUnavailable, "Logout failed", "OIDC provider logout is unavailable", "PROVIDER_LOGOUT_UNAVAILABLE")
		return
	}
	providerLogoutTicket, err := boundary.sessions.Revoke(
		request.Context(),
		principal.SessionID,
	)
	if err != nil {
		writeProblem(writer, http.StatusInternalServerError, "Logout failed", "session revocation could not be recorded", "SESSION_REVOKE_FAILED")
		return
	}
	boundary.expireBrowserSessionCookies(writer)
	if strings.TrimSpace(providerLogoutTicket) == "" {
		writeProblem(writer, http.StatusServiceUnavailable, "Logout failed", "OIDC provider logout is unavailable", "PROVIDER_LOGOUT_UNAVAILABLE")
		return
	}
	writeJSON(writer, http.StatusOK, struct {
		LogoutURL string `json:"logoutUrl"`
	}{LogoutURL: "/auth/provider-logout?ticket=" + url.QueryEscape(providerLogoutTicket)})
}

func (boundary *AuthBoundary) providerLogout(writer http.ResponseWriter, request *http.Request) {
	if boundary.provider == nil || boundary.sessions == nil {
		writeProblem(writer, http.StatusServiceUnavailable, "Logout failed", "OIDC provider logout is unavailable", "PROVIDER_LOGOUT_UNAVAILABLE")
		return
	}
	providerIDToken, err := boundary.sessions.RedeemProviderLogout(
		request.Context(),
		request.URL.Query().Get("ticket"),
	)
	if err != nil {
		writeProblem(writer, http.StatusBadRequest, "Logout failed", "provider logout ticket is invalid or expired", "PROVIDER_LOGOUT_TICKET_INVALID")
		return
	}
	logoutURL := strings.TrimSpace(boundary.provider.LogoutURL(providerIDToken))
	if logoutURL == "" {
		writeProblem(writer, http.StatusServiceUnavailable, "Logout failed", "OIDC provider logout is unavailable", "PROVIDER_LOGOUT_UNAVAILABLE")
		return
	}
	writer.Header().Set("Cache-Control", "no-store")
	http.Redirect(writer, request, logoutURL, http.StatusSeeOther)
}

func (boundary *AuthBoundary) authenticate(writer http.ResponseWriter, request *http.Request) (identity.Principal, bool) {
	if boundary.sessions == nil {
		writeProblem(writer, http.StatusUnauthorized, "Authentication required", "no active browser session", "UNAUTHENTICATED")
		return identity.Principal{}, false
	}
	cookie, err := request.Cookie(boundary.cookieNames().session)
	if err != nil || strings.TrimSpace(cookie.Value) == "" {
		writeProblem(writer, http.StatusUnauthorized, "Authentication required", "no active browser session", "UNAUTHENTICATED")
		return identity.Principal{}, false
	}
	authenticationContext := request.Context()
	if isMutation(request.Method) || boundary.alwaysFreshAuthority {
		authenticationContext = session.RequireFreshAuthorityObservation(
			authenticationContext,
		)
	}
	principal, err := boundary.sessions.Authenticate(
		authenticationContext,
		cookie.Value,
	)
	if err != nil {
		diagnostic := session.AuthenticationFailureDiagnostic(err)
		if diagnostic == "context-expired" {
			slog.Debug(
				"browser session authentication request cancelled",
				"diagnostic",
				diagnostic,
			)
		} else {
			slog.Warn(
				"browser session authentication rejected",
				"diagnostic",
				diagnostic,
			)
		}
		if errors.Is(err, session.ErrUnauthenticated) {
			boundary.expireBrowserSessionCookies(writer)
		}
		writeProblem(writer, http.StatusUnauthorized, "Authentication required", "browser session is expired, revoked, or invalid", "UNAUTHENTICATED")
		return identity.Principal{}, false
	}
	return principal, true
}

func (boundary *AuthBoundary) validateCSRF(writer http.ResponseWriter, request *http.Request, sessionID string) bool {
	headerToken := strings.TrimSpace(request.Header.Get(CSRFHeaderName))
	cookie, err := request.Cookie(boundary.cookieNames().csrf)
	if err != nil || headerToken == "" || strings.TrimSpace(cookie.Value) == "" || subtle.ConstantTimeCompare([]byte(headerToken), []byte(cookie.Value)) != 1 {
		writeProblem(writer, http.StatusForbidden, "Request forbidden", "CSRF token is missing or invalid", "CSRF_INVALID")
		return false
	}
	if err := boundary.sessions.ValidateCSRF(request.Context(), sessionID, headerToken); err != nil {
		writeProblem(writer, http.StatusForbidden, "Request forbidden", "CSRF token is missing or invalid", "CSRF_INVALID")
		return false
	}
	return true
}

type browserCookieNames struct {
	session string
	csrf    string
	login   string
}

func (boundary *AuthBoundary) cookieNames() browserCookieNames {
	if boundary.cookieSecure {
		return browserCookieNames{session: SessionCookieName, csrf: CSRFCookieName, login: LoginCookieName}
	}
	return browserCookieNames{session: localSessionCookieName, csrf: localCSRFCookieName, login: localLoginCookieName}
}

func (boundary *AuthBoundary) loginBindingCookie(request *http.Request) string {
	cookie, err := request.Cookie(boundary.cookieNames().login)
	if err != nil {
		return ""
	}
	return cookie.Value
}

func (boundary *AuthBoundary) setLoginBindingCookie(writer http.ResponseWriter, value string) {
	http.SetCookie(writer, &http.Cookie{
		Name: boundary.cookieNames().login, Value: value, Path: "/", Secure: boundary.cookieSecure,
		HttpOnly: true, SameSite: http.SameSiteLaxMode, MaxAge: int(loginCookieTTL / time.Second),
	})
}

func (boundary *AuthBoundary) setBrowserSessionCookies(writer http.ResponseWriter, browserSession session.BrowserSession) {
	cookieNames := boundary.cookieNames()
	http.SetCookie(writer, &http.Cookie{
		Name: cookieNames.session, Value: browserSession.Token, Path: "/", Secure: boundary.cookieSecure, HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})
	http.SetCookie(writer, &http.Cookie{
		Name: cookieNames.csrf, Value: browserSession.CSRFToken, Path: "/", Secure: boundary.cookieSecure, HttpOnly: false,
		SameSite: http.SameSiteStrictMode,
	})
}

func (boundary *AuthBoundary) expireBrowserSessionCookies(writer http.ResponseWriter) {
	expiredAt := time.Unix(1, 0).UTC()
	cookieNames := boundary.cookieNames()
	for _, cookie := range []*http.Cookie{
		{Name: cookieNames.session, Path: "/", Secure: boundary.cookieSecure, HttpOnly: true, SameSite: http.SameSiteStrictMode, Expires: expiredAt, MaxAge: -1},
		{Name: cookieNames.csrf, Path: "/", Secure: boundary.cookieSecure, HttpOnly: false, SameSite: http.SameSiteStrictMode, Expires: expiredAt, MaxAge: -1},
	} {
		http.SetCookie(writer, cookie)
	}
}

func isMutation(method string) bool {
	switch method {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	default:
		return false
	}
}

type principalContextKey struct{}
