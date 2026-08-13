package provider

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/MarlonJD/aviaSurveil360/apps/auth/internal/challenge"
	"github.com/MarlonJD/aviaSurveil360/apps/auth/internal/identity"
	"github.com/MarlonJD/aviaSurveil360/apps/auth/internal/mail"
	"github.com/MarlonJD/aviaSurveil360/apps/auth/internal/mfa"
	"github.com/zitadel/oidc/v3/pkg/oidc"
	"github.com/zitadel/oidc/v3/pkg/op"
)

// RuntimeCandidate is the isolated, durable provider surface used by the
// canonical local-preprod topology.
type RuntimeCandidate struct {
	Handler  http.Handler
	Provider *op.Provider
	Storage  *PostgresStorage
}

type RuntimeDependencies struct {
	Identity   *identity.PostgresStore
	MFA        *mfa.PostgresStore
	Challenges *challenge.PostgresStore
	Outbox     *mail.Outbox
	Logger     *slog.Logger
}

func NewRuntimeCandidate(configuration CandidateConfig, storage *PostgresStorage, dependencies RuntimeDependencies) (*RuntimeCandidate, error) {
	if storage == nil || dependencies.Identity == nil || dependencies.MFA == nil || dependencies.Challenges == nil || dependencies.Outbox == nil {
		return nil, ErrProviderUnavailable
	}
	if err := validateProviderConfig(configuration); err != nil {
		return nil, err
	}
	providerConfig := &op.Config{CryptoKey: configuration.CryptoKey, CryptoKeyId: configuration.CryptoKeyID, DefaultLogoutRedirectURI: configuration.PostLogoutRedirectURI, CodeMethodS256: true, AuthMethodPost: false, AuthMethodPrivateKeyJWT: false, GrantTypeRefreshToken: false, RequestObjectSupported: false, SupportedScopes: []string{oidc.ScopeOpenID, oidc.ScopeProfile, oidc.ScopeEmail}}
	options := []op.Option{}
	if configuration.AllowInsecure {
		options = append(options, op.WithAllowInsecure())
	}
	provider, err := op.NewProvider(providerConfig, storage, op.StaticIssuer(configuration.Issuer), options...)
	if err != nil {
		return nil, fmt.Errorf("initialize durable OIDC provider: %w", err)
	}
	logger := dependencies.Logger
	if logger == nil {
		logger = slog.Default()
	}
	runtime := &runtimeLogin{storage: storage, identity: dependencies.Identity, mfa: dependencies.MFA, challenges: dependencies.Challenges, outbox: dependencies.Outbox, logger: logger, provider: provider, issuer: strings.TrimRight(configuration.Issuer, "/"), clientSecret: configuration.ClientSecret}
	mux := http.NewServeMux()
	mux.HandleFunc("/authorize", func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Query().Get("response_type") == string(oidc.ResponseTypeCode) && (request.URL.Query().Get("code_challenge") == "" || request.URL.Query().Get("code_challenge_method") != "S256") {
			http.Error(writer, "PKCE S256 is required", http.StatusBadRequest)
			return
		}
		provider.ServeHTTP(writer, request.WithContext(WithBrowserBinding(request.Context(), browserBinding(request, configuration.ClientSecret))))
	})
	// zitadel/oidc completes a provider-owned browser authorization request at
	// this callback path. Register it explicitly so it cannot be shadowed by
	// an application UI route as the isolated runtime grows.
	mux.Handle("/authorize/callback", provider)
	mux.HandleFunc("/login", runtime.login)
	mux.HandleFunc("/mfa", runtime.mfaChallenge)
	mux.HandleFunc("/activate", runtime.activate)
	mux.HandleFunc("/recover/password", runtime.passwordRecovery)
	mux.HandleFunc("/recover/mfa", runtime.mfaRecovery)
	// The selected provider owns `/end_session`; this alias makes the local UI
	// entry point explicit without accepting a broader logout protocol.
	mux.HandleFunc("/logout", runtime.logout)
	mux.Handle("/", provider)
	return &RuntimeCandidate{Handler: mux, Provider: provider, Storage: storage}, nil
}

type runtimeLogin struct {
	storage      *PostgresStorage
	identity     *identity.PostgresStore
	mfa          *mfa.PostgresStore
	challenges   *challenge.PostgresStore
	outbox       *mail.Outbox
	logger       *slog.Logger
	provider     *op.Provider
	issuer       string
	clientSecret string
}

func browserBinding(request *http.Request, clientSecret string) string {
	for _, name := range []string{"__Host-avia_login", "avia_login"} {
		cookie, err := request.Cookie(name)
		if err == nil && validateBrowserBinding(cookie.Value, clientSecret) {
			return cookie.Value
		}
	}
	return ""
}

func validateBrowserBinding(raw, clientSecret string) bool {
	parts := strings.Split(strings.TrimSpace(raw), ".")
	if len(parts) != 2 {
		return false
	}
	value, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil || len(value) != 32 {
		return false
	}
	signature, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil || len(signature) != sha256.Size {
		return false
	}
	keyHash := sha256.New()
	_, _ = keyHash.Write([]byte("as360-oidc-login-binding-key-v1\x00"))
	_, _ = keyHash.Write([]byte(clientSecret))
	mac := hmac.New(sha256.New, keyHash.Sum(nil))
	_, _ = mac.Write([]byte("as360-oidc-login-binding-v1\x00"))
	_, _ = mac.Write(value)
	return subtle.ConstantTimeCompare(signature, mac.Sum(nil)) == 1
}

func (runtime *runtimeLogin) passwordRecovery(writer http.ResponseWriter, request *http.Request) {
	runtime.recovery(writer, request, challenge.PurposePasswordReset)
}

func (runtime *runtimeLogin) mfaRecovery(writer http.ResponseWriter, request *http.Request) {
	runtime.recovery(writer, request, challenge.PurposeMFARecovery)
}

func (runtime *runtimeLogin) recovery(writer http.ResponseWriter, request *http.Request, purpose challenge.Purpose) {
	if request.Method == http.MethodGet && request.URL.Query().Get("token") == "" {
		renderForm(writer, "Recover access", "", "Email", "email", "", false, false)
		return
	}
	if request.Method == http.MethodPost && request.URL.Query().Get("token") == "" {
		request.Body = http.MaxBytesReader(writer, request.Body, 4<<10)
		if err := request.ParseForm(); err != nil {
			http.Error(writer, "invalid recovery request", http.StatusBadRequest)
			return
		}
		runtime.issueRecovery(writer, request, purpose, request.Form.Get("email"))
		return
	}
	if request.URL.Query().Get("token") == "" || request.URL.Query().Get("subject") == "" {
		http.Error(writer, "invalid recovery request", http.StatusBadRequest)
		return
	}
	if purpose == challenge.PurposePasswordReset {
		runtime.passwordReset(writer, request)
		return
	}
	runtime.resetMFA(writer, request)
}

func (runtime *runtimeLogin) issueRecovery(writer http.ResponseWriter, request *http.Request, purpose challenge.Purpose, email string) {
	// Always return the same no-store response, including for malformed and
	// unknown emails, so this route does not become an account oracle.
	defer recoveryAccepted(writer)
	if err := runtime.storage.AdmitRecovery(request.Context(), string(purpose), email); err != nil {
		runtime.logAdmission(string(purpose), err)
		return
	}
	tx, err := runtime.storage.pool.Begin(request.Context())
	if err != nil {
		return
	}
	defer func() { _ = tx.Rollback(request.Context()) }()
	account, err := runtime.identity.LookupEmailTx(request.Context(), tx, strings.TrimSpace(email))
	if err != nil || account.State != identity.AccountActive || !account.EmailVerified {
		return
	}
	lockKey := "recovery:" + account.SubjectID + ":" + string(purpose)
	if _, err := tx.Exec(request.Context(), `SELECT pg_advisory_xact_lock(hashtext($1))`, lockKey); err != nil {
		return
	}
	now := runtime.storage.clock().UTC()
	dedupeKey := recoveryDedupeKey(account.SubjectID, purpose)
	var pending bool
	if err := tx.QueryRow(request.Context(), `SELECT EXISTS (SELECT 1 FROM auth_identity.mail_deliveries WHERE dedupe_key = $1 AND state IN ('queued', 'leased', 'retryable'))`, dedupeKey).Scan(&pending); err != nil {
		return
	}
	activeChallenge, err := runtime.challenges.ActiveTx(request.Context(), tx, account.SubjectID, purpose, now)
	if err != nil {
		return
	}
	var delivered bool
	if err := tx.QueryRow(request.Context(), `SELECT EXISTS (SELECT 1 FROM auth_identity.mail_deliveries WHERE dedupe_key = $1 AND state = 'delivered')`, dedupeKey).Scan(&delivered); err != nil {
		return
	}
	// A delivered message coalesces only while its challenge remains usable. A
	// retained delivered row must not suppress a fresh recovery after the
	// original 15-minute challenge has expired.
	if pending || (delivered && activeChallenge) {
		if runtime.logger != nil {
			runtime.logger.Info("recovery request coalesced", slog.String("operation_class", string(purpose)), slog.String("reason", "existing_delivery_or_challenge"))
		}
		if err := tx.Commit(request.Context()); err != nil {
			return
		}
		return
	}
	// Invalidate every active row, including an already-expired row, before the
	// replacement insert. This keeps the partial unique active-challenge index
	// aligned with the expiry check above.
	if _, err := runtime.challenges.InvalidateTx(request.Context(), tx, account.SubjectID, purpose, now); err != nil {
		return
	}
	issued, err := runtime.challenges.IssueTx(request.Context(), tx, account.SubjectID, purpose, 15*time.Minute, 5)
	if err != nil {
		return
	}
	path := "recover/password"
	if purpose == challenge.PurposeMFARecovery {
		path = "recover/mfa"
	}
	link := providerAbsoluteEndpoint(runtime.issuer, path) + "?" + url.Values{"subject": {account.SubjectID}, "token": {issued.Token}}.Encode()
	if _, err := runtime.outbox.EnqueueTx(request.Context(), tx, mail.Delivery{Recipient: account.Email, Subject: "AviaSurveil360 account recovery", Body: "Use this one-time recovery link within 15 minutes: " + link, DedupeKey: dedupeKey}); err != nil {
		return
	}
	if err := tx.Commit(request.Context()); err != nil {
		return
	}
}

func recoveryDedupeKey(subjectID string, purpose challenge.Purpose) string {
	return fmt.Sprintf("recovery:%x", sha256.Sum256([]byte("as360-recovery-mail-v1\x00"+subjectID+"\x00"+string(purpose))))
}

func (runtime *runtimeLogin) passwordReset(writer http.ResponseWriter, request *http.Request) {
	if request.Method == http.MethodGet {
		renderResetForm(writer, request, "Set a new password", "New password")
		return
	}
	if request.Method != http.MethodPost {
		writer.Header().Set("Allow", "GET, POST")
		http.Error(writer, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	request.Body = http.MaxBytesReader(writer, request.Body, 4<<10)
	if err := request.ParseForm(); err != nil {
		http.Error(writer, "invalid reset request", http.StatusBadRequest)
		return
	}
	subject, token, newPassword := strings.TrimSpace(request.Form.Get("subject")), strings.TrimSpace(request.Form.Get("token")), []byte(request.Form.Get("password"))
	if _, err := runtime.identity.ResetPasswordWithChallenge(request.Context(), subject, string(challenge.PurposePasswordReset), token, newPassword); err != nil {
		http.Error(writer, "password reset failed", http.StatusBadRequest)
		return
	}
	http.Redirect(writer, request, runtime.loginPath(), http.StatusSeeOther)
}

func (runtime *runtimeLogin) activate(writer http.ResponseWriter, request *http.Request) {
	if request.Method == http.MethodGet {
		renderResetForm(writer, request, "Activate account", "New password")
		return
	}
	if request.Method != http.MethodPost {
		writer.Header().Set("Allow", "GET, POST")
		http.Error(writer, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	request.Body = http.MaxBytesReader(writer, request.Body, 4<<10)
	if err := request.ParseForm(); err != nil {
		http.Error(writer, "invalid activation request", http.StatusBadRequest)
		return
	}
	subject, token, newPassword := strings.TrimSpace(request.Form.Get("subject")), strings.TrimSpace(request.Form.Get("token")), []byte(request.Form.Get("password"))
	if _, err := runtime.identity.ActivateWithInvitation(request.Context(), subject, token, newPassword); err != nil {
		http.Error(writer, "activation failed", http.StatusBadRequest)
		return
	}
	http.Redirect(writer, request, runtime.loginPath(), http.StatusSeeOther)
}

func (runtime *runtimeLogin) resetMFA(writer http.ResponseWriter, request *http.Request) {
	if request.Method == http.MethodGet {
		renderResetForm(writer, request, "Reset MFA", "")
		return
	}
	if request.Method != http.MethodPost {
		writer.Header().Set("Allow", "GET, POST")
		http.Error(writer, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	request.Body = http.MaxBytesReader(writer, request.Body, 4<<10)
	if err := request.ParseForm(); err != nil {
		http.Error(writer, "invalid recovery request", http.StatusBadRequest)
		return
	}
	subject, token := strings.TrimSpace(request.Form.Get("subject")), strings.TrimSpace(request.Form.Get("token"))
	if runtime.mfa.ResetWithChallenge(request.Context(), subject, string(challenge.PurposeMFARecovery), token) != nil {
		http.Error(writer, "invalid recovery request", http.StatusBadRequest)
		return
	}
	http.Redirect(writer, request, runtime.loginPath(), http.StatusSeeOther)
}

func (runtime *runtimeLogin) logout(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		writer.Header().Set("Allow", "GET")
		http.Error(writer, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	http.Redirect(writer, request, providerEndpoint(runtime.issuer, "end_session")+"?"+request.URL.Query().Encode(), http.StatusFound)
}

func (runtime *runtimeLogin) login(writer http.ResponseWriter, request *http.Request) {
	if request.Method == http.MethodGet {
		runtime.loginForm(writer, request, "")
		return
	}
	if request.Method != http.MethodPost {
		writer.Header().Set("Allow", "GET, POST")
		http.Error(writer, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	request.Body = http.MaxBytesReader(writer, request.Body, 8<<10)
	if err := request.ParseForm(); err != nil {
		http.Error(writer, "invalid login request", http.StatusBadRequest)
		return
	}
	id, username, password := strings.TrimSpace(request.Form.Get("id")), strings.TrimSpace(request.Form.Get("identifier")), request.Form.Get("password")
	auth, err := runtime.storage.AuthRequestByID(request.Context(), id)
	if err != nil || auth.Done() {
		http.Error(writer, "invalid login request", http.StatusBadRequest)
		return
	}
	result, err := runtime.identity.Authenticate(request.Context(), identity.AuthenticationRequest{Identifier: username, Password: []byte(password), BrowserBinding: browserBinding(request, runtime.clientSecret), DeviceKey: request.UserAgent()})
	if err != nil {
		if errors.Is(err, identity.ErrAuthenticationUnavailable) {
			http.Error(writer, "authentication unavailable", http.StatusServiceUnavailable)
			return
		}
		if errors.Is(err, identity.ErrAuthenticationRateLimited) {
			runtime.logAdmission("password", err)
			http.Error(writer, "too many login attempts", http.StatusTooManyRequests)
			return
		}
		runtime.loginForm(writer, request, "invalid credentials")
		return
	}
	factor, err := runtime.mfa.Snapshot(request.Context(), result.Account.SubjectID)
	if err != nil && !errors.Is(err, mfa.ErrFactorNotFound) {
		http.Error(writer, "MFA unavailable", http.StatusServiceUnavailable)
		return
	}
	if err := runtime.storage.StageAuthenticatedSubject(request.Context(), id, result.Account.SubjectID, result.Account.AuthRevision); err != nil {
		http.Error(writer, "authentication unavailable", http.StatusServiceUnavailable)
		return
	}
	if factor.Enabled {
		http.Redirect(writer, request, runtime.mfaPath(id), http.StatusFound)
		return
	}
	runtime.complete(writer, request, id, result.Account.SubjectID, []string{"pwd"})
}

func (runtime *runtimeLogin) mfaChallenge(writer http.ResponseWriter, request *http.Request) {
	if request.Method == http.MethodGet {
		runtime.mfaForm(writer, request, "")
		return
	}
	if request.Method != http.MethodPost {
		writer.Header().Set("Allow", "GET, POST")
		http.Error(writer, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	request.Body = http.MaxBytesReader(writer, request.Body, 4<<10)
	if err := request.ParseForm(); err != nil {
		http.Error(writer, "invalid MFA request", http.StatusBadRequest)
		return
	}
	id, code := strings.TrimSpace(request.Form.Get("id")), strings.TrimSpace(request.Form.Get("code"))
	auth, err := runtime.storage.AuthRequestByID(request.Context(), id)
	if err != nil || auth.Done() || auth.GetSubject() == "" {
		http.Error(writer, "invalid MFA request", http.StatusBadRequest)
		return
	}
	amr := []string{"pwd", "otp"}
	if err := runtime.storage.AdmitMFA(request.Context(), id, auth.GetSubject()); err != nil {
		if errors.Is(err, ErrProviderRateLimited) {
			runtime.logAdmission("mfa", err)
			http.Error(writer, "too many MFA attempts", http.StatusTooManyRequests)
			return
		}
		http.Error(writer, "authentication unavailable", http.StatusServiceUnavailable)
		return
	}
	reservation, err := runtime.storage.ReserveMFAAttempt(request.Context(), id, auth.GetSubject())
	if err != nil {
		if errors.Is(err, ErrProviderRateLimited) {
			runtime.logAdmission("mfa", err)
			http.Error(writer, "too many MFA attempts", http.StatusTooManyRequests)
			return
		}
		runtime.mfaForm(writer, request, "invalid MFA code")
		return
	}
	if request.Form.Get("recovery") == "1" {
		err = runtime.mfa.ConsumeRecoveryCode(request.Context(), auth.GetSubject(), code)
		amr = []string{"pwd", "mfa"}
	} else {
		err = runtime.mfa.Verify(request.Context(), auth.GetSubject(), code)
	}
	if err != nil {
		if reservation.Final {
			runtime.logger.Warn("MFA attempt budget exhausted", slog.String("operation_class", "mfa"), slog.String("reason", "verification_failed"))
			_ = runtime.storage.RejectMFAAttempt(request.Context(), reservation)
		}
		runtime.mfaForm(writer, request, "invalid MFA code")
		return
	}
	runtime.complete(writer, request, id, auth.GetSubject(), amr)
}

func (runtime *runtimeLogin) complete(writer http.ResponseWriter, request *http.Request, id, subject string, amr []string) {
	if len(amr) > 0 {
		if err := runtime.storage.SetAuthenticatedAMR(request.Context(), id, amr); err != nil {
			http.Error(writer, "authentication unavailable", http.StatusServiceUnavailable)
			return
		}
	}
	if err := runtime.storage.Authorize(request.Context(), id, subject); err != nil {
		runtime.logger.Warn("authorization transition rejected", slog.String("operation_class", "authorize"), slog.String("reason", "revision_or_state_mismatch"))
		http.Error(writer, "authentication unavailable", http.StatusServiceUnavailable)
		return
	}
	callback := runtime.authCallbackURL(id)
	http.Redirect(writer, request, callback, http.StatusFound)
}

func (runtime *runtimeLogin) logAdmission(operation string, err error) {
	if runtime.logger == nil {
		return
	}
	reason := "unavailable"
	if errors.Is(err, ErrProviderRateLimited) || errors.Is(err, identity.ErrAuthenticationRateLimited) {
		reason = "rate_limited"
	}
	runtime.logger.Warn("authentication admission denied", slog.String("operation_class", operation), slog.String("reason", reason))
}

func (runtime *runtimeLogin) loginForm(writer http.ResponseWriter, request *http.Request, message string) {
	renderForm(writer, "Sign in", request.URL.Query().Get("id"), "Username or email", "identifier", message, false, true)
}

func (runtime *runtimeLogin) loginPath() string {
	return providerEndpoint(runtime.issuer, "login")
}

func (runtime *runtimeLogin) mfaPath(id string) string {
	return providerEndpoint(runtime.issuer, "mfa") + "?id=" + url.QueryEscape(id)
}

func (runtime *runtimeLogin) authCallbackURL(id string) string {
	return providerEndpoint(runtime.issuer, "authorize/callback") + "?id=" + url.QueryEscape(id)
}
func (runtime *runtimeLogin) mfaForm(writer http.ResponseWriter, request *http.Request, message string) {
	id := request.URL.Query().Get("id")
	if id == "" {
		id = request.Form.Get("id")
	}
	renderForm(writer, "Verify MFA", id, "One-time code", "code", message, true, false)
}

var formTemplate = template.Must(template.New("form").Parse(`<!doctype html><html lang="en"><head><meta charset="utf-8"><meta name="referrer" content="no-referrer"><title>{{.Title}}</title></head><body><main aria-labelledby="page-title"><h1 id="page-title">{{.Title}}</h1>{{if .Message}}<p role="alert">{{.Message}}</p>{{end}}<form method="post" autocomplete="off"><input type="hidden" name="id" value="{{.ID}}"><label for="{{.Name}}">{{.Field}}</label><input id="{{.Name}}" name="{{.Name}}" required>{{if .Password}}<label for="password">Password</label><input id="password" name="password" type="password" required>{{end}}{{if .MFA}}<label for="recovery"><input id="recovery" type="checkbox" name="recovery" value="1">Use recovery code</label>{{end}}<button type="submit">Continue</button></form></main></body></html>`))

var resetTemplate = template.Must(template.New("reset").Parse(`<!doctype html><html lang="en"><head><meta charset="utf-8"><meta name="referrer" content="no-referrer"><title>{{.Title}}</title></head><body><main aria-labelledby="page-title"><h1 id="page-title">{{.Title}}</h1><form method="post" autocomplete="off"><input type="hidden" name="subject" value="{{.Subject}}"><input type="hidden" name="token" value="{{.Token}}">{{if .Field}}<label for="password">{{.Field}}</label><input id="password" name="password" type="password" required>{{end}}<button type="submit">Continue</button></form></main></body></html>`))

func renderForm(writer http.ResponseWriter, title, id, field, name, message string, isMFA, includePassword bool) {
	writer.Header().Set("Content-Type", "text/html; charset=utf-8")
	writer.Header().Set("Cache-Control", "no-store")
	writer.Header().Set("Content-Security-Policy", "default-src 'none'; form-action 'self'; base-uri 'none'")
	_ = formTemplate.Execute(writer, struct {
		Title, ID, Field, Name, Message string
		MFA, Password                   bool
	}{title, id, field, name, message, isMFA, includePassword})
}

func renderResetForm(writer http.ResponseWriter, request *http.Request, title, field string) {
	writer.Header().Set("Content-Type", "text/html; charset=utf-8")
	writer.Header().Set("Cache-Control", "no-store")
	writer.Header().Set("Content-Security-Policy", "default-src 'none'; form-action 'self'; base-uri 'none'")
	_ = resetTemplate.Execute(writer, struct{ Title, Field, Subject, Token string }{title, field, request.URL.Query().Get("subject"), request.URL.Query().Get("token")})
}

func recoveryAccepted(writer http.ResponseWriter) {
	writer.Header().Set("Cache-Control", "no-store")
	writer.Header().Set("Content-Type", "text/plain; charset=utf-8")
	writer.WriteHeader(http.StatusAccepted)
	_, _ = writer.Write([]byte("If the account is eligible, recovery instructions will be sent."))
}
