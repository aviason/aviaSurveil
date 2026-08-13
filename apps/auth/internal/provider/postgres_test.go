package provider

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/MarlonJD/aviaSurveil360/apps/auth/internal/challenge"
	"github.com/MarlonJD/aviaSurveil360/apps/auth/internal/identity"
	"github.com/MarlonJD/aviaSurveil360/apps/auth/internal/mail"
	"github.com/MarlonJD/aviaSurveil360/apps/auth/internal/mfa"
	"github.com/MarlonJD/aviaSurveil360/apps/auth/internal/password"
	"github.com/MarlonJD/aviaSurveil360/apps/auth/internal/throttle"
	"github.com/MarlonJD/aviaSurveil360/apps/auth/migrations"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/zitadel/oidc/v3/pkg/oidc"
	"github.com/zitadel/oidc/v3/pkg/op"
)

func TestPostgreSQLStoragePersistsOIDCState(t *testing.T) {
	databaseURL := strings.TrimSpace(os.Getenv("AVIA_AUTH_TEST_DATABASE_URL"))
	if databaseURL == "" {
		t.Skip("not run: AVIA_AUTH_TEST_DATABASE_URL is not configured")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	if err := migrations.Apply(ctx, pool); err != nil {
		t.Fatal(err)
	}
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 8, 11, 18, 0, 0, 0, time.UTC)
	currentTime := now
	configuration := CandidateConfig{Issuer: "http://auth.candidate.invalid", AllowInsecure: true, CryptoKey: [32]byte{1}, CryptoKeyID: "candidate-crypto-key", SigningKey: key, SigningKeyID: "candidate-signing-key", ClientID: "candidate-web", ClientSecret: "candidate-test-secret", RedirectURI: "http://app.candidate.invalid/callback", PostLogoutRedirectURI: "http://app.candidate.invalid/logout", SubjectID: "usr_" + strings.Repeat("P", 22), Email: "provider@example.invalid", DisplayName: "Provider Test"}
	store, err := NewPostgresStorage(pool, PostgresStorageConfig{Candidate: configuration, EncryptionKey: []byte("01234567890123456789012345678901"), Clock: func() time.Time { return currentTime }})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Bootstrap(ctx); err != nil {
		t.Fatal(err)
	}
	hasher, err := password.New(password.Params{MemoryKiB: 16 * 1024, Time: 1, Threads: 1, KeyLength: 32, SaltLen: 16, MaxBytes: 1024, Capacity: 4})
	if err != nil {
		t.Fatal(err)
	}
	passwordHash, err := hasher.Hash([]byte("correct horse battery staple"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO auth_identity.accounts(subject_id, state, password_hash, email_verified, auth_revision, created_at, updated_at) VALUES ($1, 'active', $2, true, 1, $3, $3) ON CONFLICT (subject_id) DO UPDATE SET password_hash = EXCLUDED.password_hash, state = 'active', email_verified = true`, configuration.SubjectID, passwordHash, now); err != nil {
		t.Fatal(err)
	}
	defer func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM auth_identity.accounts WHERE subject_id=$1`, configuration.SubjectID)
	}()
	if _, err := pool.Exec(ctx, `INSERT INTO auth_identity.identifiers(subject_id, identifier_type, normalized_value, verified_at, created_at) VALUES ($1, 'email', 'provider@example.invalid', $2, $2) ON CONFLICT DO NOTHING`, configuration.SubjectID, now); err != nil {
		t.Fatal(err)
	}
	var encrypted []byte
	if err := pool.QueryRow(ctx, `SELECT private_key_ciphertext FROM auth_identity.provider_signing_keys WHERE key_id=$1`, configuration.SigningKeyID).Scan(&encrypted); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encrypted), string(x509.MarshalPKCS1PrivateKey(key))) {
		t.Fatal("provider signing key persisted as plaintext")
	}
	if err := store.AuthorizeClientIDSecret(ctx, configuration.ClientID, configuration.ClientSecret); err != nil {
		t.Fatal(err)
	}
	if err := store.AuthorizeClientIDSecret(ctx, configuration.ClientID, "wrong"); !errors.Is(err, oidc.ErrInvalidClient()) {
		t.Fatalf("wrong client secret = %v", err)
	}
	request, err := store.CreateAuthRequest(ctx, &oidc.AuthRequest{ClientID: configuration.ClientID, RedirectURI: configuration.RedirectURI, ResponseType: oidc.ResponseTypeCode, ResponseMode: oidc.ResponseModeQuery, Scopes: []string{oidc.ScopeOpenID, oidc.ScopeEmail}, State: "state", Nonce: "nonce", CodeChallenge: "challenge", CodeChallengeMethod: oidc.CodeChallengeMethodS256}, "")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Authorize(ctx, request.GetID(), configuration.SubjectID); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveAuthCode(ctx, request.GetID(), "authorization-code"); err != nil {
		t.Fatal(err)
	}
	loaded, err := store.AuthRequestByCode(ctx, "authorization-code")
	if err != nil || !loaded.Done() || loaded.GetSubject() != configuration.SubjectID {
		t.Fatalf("durable authorization code = %+v/%v", loaded, err)
	}
	access, refresh, _, err := store.CreateAccessAndRefreshTokens(ctx, loaded, "")
	if !errors.Is(err, oidc.ErrInvalidGrant()) || access != "" || refresh != "" {
		t.Fatalf("refresh-token issuance should be disabled = %q/%q/%v", access, refresh, err)
	}
	if _, err := store.TokenRequestByRefreshToken(ctx, "refresh-token-disabled"); !errors.Is(err, op.ErrInvalidRefreshToken) {
		t.Fatalf("disabled refresh lookup = %v", err)
	}
	if _, err := store.SigningKey(ctx); err != nil {
		t.Fatalf("load encrypted signing key: %v", err)
	}
	if keys, err := store.KeySet(ctx); err != nil || len(keys) != 1 {
		t.Fatalf("durable key set = %v/%v", keys, err)
	}
	rotatedKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.RotateSigningKey(ctx, configuration.SigningKeyID, rotatedKey, time.Hour); !errors.Is(err, ErrProviderInvalid) {
		t.Fatalf("same-ID signing-key rotation = %v", err)
	}
	if err := store.RotateSigningKey(ctx, "candidate-signing-key-3", rotatedKey, 8*24*time.Hour); !errors.Is(err, ErrProviderInvalid) {
		t.Fatalf("unbounded signing-key rotation = %v", err)
	}
	if err := store.RotateSigningKey(ctx, "candidate-signing-key-2", rotatedKey, time.Hour); err != nil {
		t.Fatalf("rotate durable signing key: %v", err)
	}
	activeKey, err := store.SigningKey(ctx)
	if err != nil || activeKey.ID() != "candidate-signing-key-2" {
		t.Fatalf("rotated active signing key = %v/%v", activeKey, err)
	}
	keys, err := store.KeySet(ctx)
	if err != nil || len(keys) != 2 {
		t.Fatalf("overlap durable key set = %v/%v", keys, err)
	}
	keyIDs := map[string]bool{}
	for _, entry := range keys {
		keyIDs[entry.ID()] = true
	}
	if !keyIDs[configuration.SigningKeyID] || !keyIDs["candidate-signing-key-2"] {
		t.Fatalf("overlap key IDs = %v", keyIDs)
	}
	currentTime = currentTime.Add(2 * time.Hour)
	if err := store.RetireExpiredSigningKeys(ctx); err != nil {
		t.Fatalf("retire elapsed signing overlap: %v", err)
	}
	keys, err = store.KeySet(ctx)
	if err != nil || len(keys) != 1 || keys[0].ID() != "candidate-signing-key-2" {
		t.Fatalf("retired durable key set = %v/%v", keys, err)
	}
	limiter, err := throttle.NewMemoryLimiter(time.Minute, 20, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	identities, err := identity.NewPostgresStore(pool, identity.Config{Hasher: hasher, PasswordPolicy: password.DefaultPolicy(), Limiter: limiter, Clock: func() time.Time { return now }})
	if err != nil {
		t.Fatal(err)
	}
	mfaStore, err := mfa.NewPostgresStore(pool, mfa.Config{EncryptionKey: []byte("0123456789abcdef0123456789abcdef"), Clock: func() time.Time { return now }})
	if err != nil {
		t.Fatal(err)
	}
	challenges, err := challenge.NewPostgresStore(pool, challenge.Config{Clock: func() time.Time { return now }})
	if err != nil {
		t.Fatal(err)
	}
	outbox, err := mail.NewOutbox(mail.OutboxConfig{Pool: pool, EncryptionKey: []byte("01234567890123456789012345678901"), Clock: func() time.Time { return now }})
	if err != nil {
		t.Fatal(err)
	}
	runtime, err := NewRuntimeCandidate(configuration, store, RuntimeDependencies{Identity: identities, MFA: mfaStore, Challenges: challenges, Outbox: outbox})
	if err != nil {
		t.Fatal(err)
	}
	authorizeQuery := url.Values{
		"client_id":             {configuration.ClientID},
		"redirect_uri":          {configuration.RedirectURI},
		"response_type":         {"code"},
		"scope":                 {"openid email"},
		"state":                 {"browser-state"},
		"nonce":                 {"browser-nonce"},
		"code_challenge":        {"browser-challenge"},
		"code_challenge_method": {"S256"},
	}
	authorizeHTTP := httptest.NewRequest(http.MethodGet, "/authorize?"+authorizeQuery.Encode(), nil)
	recorder := httptest.NewRecorder()
	runtime.Handler.ServeHTTP(recorder, authorizeHTTP)
	if recorder.Code != http.StatusFound || !strings.HasPrefix(recorder.Header().Get("Location"), "/login?id=") {
		t.Fatalf("durable authorize = %d %q", recorder.Code, recorder.Header().Get("Location"))
	}
	loginForm := httptest.NewRequest(http.MethodGet, recorder.Header().Get("Location"), nil)
	recorder = httptest.NewRecorder()
	runtime.Handler.ServeHTTP(recorder, loginForm)
	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), `lang="en"`) || !strings.Contains(recorder.Body.String(), `<main aria-labelledby="page-title">`) || !strings.Contains(recorder.Body.String(), `name="identifier"`) || !strings.Contains(recorder.Body.String(), `name="password"`) {
		t.Fatalf("durable login form = %d %q", recorder.Code, recorder.Body.String())
	}
	loginRequest, err := store.CreateAuthRequest(ctx, &oidc.AuthRequest{ClientID: configuration.ClientID, RedirectURI: configuration.RedirectURI, ResponseType: oidc.ResponseTypeCode, ResponseMode: oidc.ResponseModeQuery, Scopes: []string{oidc.ScopeOpenID}, State: "login-state", Nonce: "login-nonce", CodeChallenge: "login-challenge", CodeChallengeMethod: oidc.CodeChallengeMethodS256}, "")
	if err != nil {
		t.Fatal(err)
	}
	form := url.Values{"id": {loginRequest.GetID()}, "identifier": {"provider@example.invalid"}, "password": {"correct horse battery staple"}}
	loginHTTP := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
	loginHTTP.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	loginHTTP.RemoteAddr = "203.0.113.21:443"
	recorder = httptest.NewRecorder()
	runtime.Handler.ServeHTTP(recorder, loginHTTP)
	if recorder.Code != http.StatusFound {
		t.Fatalf("durable login = %d %s", recorder.Code, recorder.Body.String())
	}
	completed, err := store.AuthRequestByID(ctx, loginRequest.GetID())
	if err != nil || !completed.Done() || completed.GetSubject() != configuration.SubjectID {
		t.Fatalf("completed durable login = %+v/%v", completed, err)
	}
	_, err = mfaStore.Enroll(ctx, configuration.SubjectID, "AviaSurveil360", "provider@example.invalid")
	if err != nil {
		t.Fatal(err)
	}
	mfaCode, err := mfaStore.CurrentCodeForTesting(ctx, configuration.SubjectID, now)
	if err != nil {
		t.Fatal(err)
	}
	if err := mfaStore.ConfirmEnrollment(ctx, configuration.SubjectID, mfaCode); err != nil {
		t.Fatal(err)
	}
	now = now.Add(30 * time.Second)
	mfaCode, err = mfaStore.CurrentCodeForTesting(ctx, configuration.SubjectID, now)
	if err != nil {
		t.Fatal(err)
	}
	mfaLoginRequest, err := store.CreateAuthRequest(ctx, &oidc.AuthRequest{ClientID: configuration.ClientID, RedirectURI: configuration.RedirectURI, ResponseType: oidc.ResponseTypeCode, ResponseMode: oidc.ResponseModeQuery, Scopes: []string{oidc.ScopeOpenID}, State: "mfa-state", Nonce: "mfa-nonce", CodeChallenge: "mfa-challenge", CodeChallengeMethod: oidc.CodeChallengeMethodS256}, "")
	if err != nil {
		t.Fatal(err)
	}
	form = url.Values{"id": {mfaLoginRequest.GetID()}, "identifier": {"provider@example.invalid"}, "password": {"correct horse battery staple"}}
	loginHTTP = httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
	loginHTTP.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	loginHTTP.RemoteAddr = "203.0.113.22:443"
	recorder = httptest.NewRecorder()
	runtime.Handler.ServeHTTP(recorder, loginHTTP)
	if recorder.Code != http.StatusFound || !strings.HasPrefix(recorder.Header().Get("Location"), "/mfa?id=") {
		t.Fatalf("MFA-required login = %d %q", recorder.Code, recorder.Header().Get("Location"))
	}
	pending, err := store.AuthRequestByID(ctx, mfaLoginRequest.GetID())
	if err != nil || pending.Done() || pending.GetSubject() != configuration.SubjectID {
		t.Fatalf("pending MFA login = %+v/%v", pending, err)
	}
	form = url.Values{"id": {mfaLoginRequest.GetID()}, "code": {mfaCode}}
	mfaHTTP := httptest.NewRequest(http.MethodPost, "/mfa", strings.NewReader(form.Encode()))
	mfaHTTP.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	mfaHTTP.RemoteAddr = "203.0.113.22:443"
	recorder = httptest.NewRecorder()
	runtime.Handler.ServeHTTP(recorder, mfaHTTP)
	if recorder.Code != http.StatusFound {
		t.Fatalf("MFA completion = %d %s", recorder.Code, recorder.Body.String())
	}
	callbackHTTP := httptest.NewRequest(http.MethodGet, recorder.Header().Get("Location"), nil)
	recorder = httptest.NewRecorder()
	runtime.Handler.ServeHTTP(recorder, callbackHTTP)
	if recorder.Code != http.StatusFound || !strings.HasPrefix(recorder.Header().Get("Location"), configuration.RedirectURI+"?code=") {
		t.Fatalf("MFA callback = %d %q", recorder.Code, recorder.Header().Get("Location"))
	}
	completed, err = store.AuthRequestByID(ctx, mfaLoginRequest.GetID())
	if err != nil || !completed.Done() {
		t.Fatalf("completed MFA login = %+v/%v", completed, err)
	}
	passwordChallenge, err := challenges.Issue(ctx, configuration.SubjectID, challenge.PurposePasswordReset, time.Minute, 3)
	if err != nil {
		t.Fatal(err)
	}
	resetForm := url.Values{"subject": {configuration.SubjectID}, "token": {passwordChallenge.Token}, "password": {"reset correct password 4"}}
	resetRequest := httptest.NewRequest(http.MethodPost, "/recover/password?subject="+url.QueryEscape(configuration.SubjectID)+"&token="+url.QueryEscape(passwordChallenge.Token), strings.NewReader(resetForm.Encode()))
	resetRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	recorder = httptest.NewRecorder()
	runtime.Handler.ServeHTTP(recorder, resetRequest)
	if recorder.Code != http.StatusSeeOther || recorder.Header().Get("Location") != "/login" {
		t.Fatalf("password recovery reset = %d %q", recorder.Code, recorder.Header().Get("Location"))
	}
	if _, err := identities.Authenticate(ctx, identity.AuthenticationRequest{Identifier: configuration.Email, Password: []byte("reset correct password 4"), Source: throttle.ForwardedHeaders{RemoteAddr: "203.0.113.23:443"}, DeviceKey: "reset-device"}); err != nil {
		t.Fatalf("authenticate reset password = %v", err)
	}
	mfaChallenge, err := challenges.Issue(ctx, configuration.SubjectID, challenge.PurposeMFARecovery, time.Minute, 3)
	if err != nil {
		t.Fatal(err)
	}
	mfaResetForm := url.Values{"subject": {configuration.SubjectID}, "token": {mfaChallenge.Token}}
	mfaResetRequest := httptest.NewRequest(http.MethodPost, "/recover/mfa?subject="+url.QueryEscape(configuration.SubjectID)+"&token="+url.QueryEscape(mfaChallenge.Token), strings.NewReader(mfaResetForm.Encode()))
	mfaResetRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	recorder = httptest.NewRecorder()
	runtime.Handler.ServeHTTP(recorder, mfaResetRequest)
	if recorder.Code != http.StatusSeeOther || recorder.Header().Get("Location") != "/login" {
		t.Fatalf("MFA recovery reset = %d %q", recorder.Code, recorder.Header().Get("Location"))
	}
	if _, err := mfaStore.Snapshot(ctx, configuration.SubjectID); !errors.Is(err, mfa.ErrFactorNotFound) {
		t.Fatalf("MFA factor after recovery reset = %v", err)
	}
	logout := httptest.NewRequest(http.MethodGet, "/logout?client_id="+url.QueryEscape(configuration.ClientID), nil)
	recorder = httptest.NewRecorder()
	runtime.Handler.ServeHTTP(recorder, logout)
	if recorder.Code != http.StatusFound || !strings.HasPrefix(recorder.Header().Get("Location"), "/end_session?client_id=") {
		t.Fatalf("logout entry point = %d %q", recorder.Code, recorder.Header().Get("Location"))
	}
	endSession := httptest.NewRequest(http.MethodGet, "/end_session?"+url.Values{"client_id": {configuration.ClientID}, "post_logout_redirect_uri": {configuration.PostLogoutRedirectURI}}.Encode(), nil)
	recorder = httptest.NewRecorder()
	runtime.Handler.ServeHTTP(recorder, endSession)
	if recorder.Code != http.StatusFound || recorder.Header().Get("Location") != configuration.PostLogoutRedirectURI {
		t.Fatalf("durable end session = %d %q", recorder.Code, recorder.Header().Get("Location"))
	}
}
