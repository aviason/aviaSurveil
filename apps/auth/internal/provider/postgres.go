package provider

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/subtle"
	"crypto/x509"
	"errors"
	"fmt"
	"time"

	jose "github.com/go-jose/go-jose/v4"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/zitadel/oidc/v3/pkg/oidc"
	"github.com/zitadel/oidc/v3/pkg/op"
)

// PostgresStorage is the durable implementation of the selected OP library's
// required storage contract. It is deliberately not mounted by the candidate
// HTTP server until provider-owned identity handlers exist.
type PostgresStorage struct {
	pool          *pgxpool.Pool
	candidate     CandidateConfig
	encryptionKey []byte
	clock         func() time.Time
}

type PostgresStorageConfig struct {
	Candidate     CandidateConfig
	EncryptionKey []byte
	Clock         func() time.Time
}

func NewPostgresStorage(pool *pgxpool.Pool, configuration PostgresStorageConfig) (*PostgresStorage, error) {
	if pool == nil || len(configuration.EncryptionKey) != 32 {
		return nil, ErrProviderInvalid
	}
	if err := validateProviderConfig(configuration.Candidate); err != nil {
		return nil, err
	}
	if configuration.Clock == nil {
		configuration.Clock = time.Now
	}
	return &PostgresStorage{pool: pool, candidate: configuration.Candidate, encryptionKey: append([]byte(nil), configuration.EncryptionKey...), clock: configuration.Clock}, nil
}

// Bootstrap persists the exact client and encrypted active signing key for a
// disposable candidate. A different active key is rejected; callers must use
// RotateSigningKey so overlap is never skipped accidentally.
func (storage *PostgresStorage) Bootstrap(ctx context.Context) error {
	secretHash := sha256.Sum256([]byte(storage.candidate.ClientSecret))
	privateDER := x509.MarshalPKCS1PrivateKey(storage.candidate.SigningKey)
	privateCiphertext, err := storage.encryptKey(storage.candidate.SigningKeyID, privateDER)
	if err != nil {
		return err
	}
	publicDER, err := x509.MarshalPKIXPublicKey(&storage.candidate.SigningKey.PublicKey)
	if err != nil {
		return fmt.Errorf("marshal provider public key: %w", err)
	}
	now := storage.clock().UTC()
	tx, err := storage.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin provider material bootstrap: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `INSERT INTO auth_identity.provider_clients(client_id, secret_hash, redirect_uris, post_logout_redirect_uris, scopes, state, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, 'active', $6, $6) ON CONFLICT (client_id) DO UPDATE SET secret_hash = EXCLUDED.secret_hash, redirect_uris = EXCLUDED.redirect_uris, post_logout_redirect_uris = EXCLUDED.post_logout_redirect_uris, scopes = EXCLUDED.scopes, state = 'active', updated_at = EXCLUDED.updated_at`, storage.candidate.ClientID, secretHash[:], []string{storage.candidate.RedirectURI}, []string{storage.candidate.PostLogoutRedirectURI}, []string{oidc.ScopeOpenID, oidc.ScopeProfile, oidc.ScopeEmail, oidc.ScopeOfflineAccess}, now); err != nil {
		return fmt.Errorf("upsert provider client: %w", err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO auth_identity.provider_signing_keys(key_id, algorithm, private_key_ciphertext, public_key, state, created_at) VALUES ($1, 'RS256', $2, $3, 'active', $4) ON CONFLICT (key_id) DO UPDATE SET private_key_ciphertext = EXCLUDED.private_key_ciphertext, public_key = EXCLUDED.public_key, state = 'active', retire_at = NULL`, storage.candidate.SigningKeyID, privateCiphertext, publicDER, now); err != nil {
		return fmt.Errorf("upsert provider signing key: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit provider material bootstrap: %w", err)
	}
	return nil
}

func (storage *PostgresStorage) Health(ctx context.Context) error { return storage.pool.Ping(ctx) }

// RotateSigningKey makes nextKeyID active and keeps the previous active key in
// a finite verification overlap. It never replaces a key in place, and callers
// must later retire the overlap only after its explicit deadline.
func (storage *PostgresStorage) RotateSigningKey(ctx context.Context, nextKeyID string, nextKey *rsa.PrivateKey, overlap time.Duration) error {
	next := storage.candidate
	next.SigningKeyID = nextKeyID
	next.SigningKey = nextKey
	if err := validateProviderConfig(next); err != nil || overlap <= 0 || overlap > 7*24*time.Hour || nextKeyID == storage.candidate.SigningKeyID {
		return ErrProviderInvalid
	}
	privateDER := x509.MarshalPKCS1PrivateKey(nextKey)
	privateCiphertext, err := storage.encryptKey(nextKeyID, privateDER)
	if err != nil {
		return err
	}
	publicDER, err := x509.MarshalPKIXPublicKey(&nextKey.PublicKey)
	if err != nil {
		return fmt.Errorf("marshal rotated provider public key: %w", err)
	}
	now := storage.clock().UTC()
	tx, err := storage.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin provider signing-key rotation: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `SELECT key_id FROM auth_identity.provider_signing_keys WHERE state = 'active' FOR UPDATE`); err != nil {
		return fmt.Errorf("lock active provider signing key: %w", err)
	}
	if _, err := tx.Exec(ctx, `UPDATE auth_identity.provider_signing_keys SET state = 'overlap', retire_at = $1 WHERE state = 'active'`, now.Add(overlap)); err != nil {
		return fmt.Errorf("begin provider signing-key overlap: %w", err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO auth_identity.provider_signing_keys(key_id, algorithm, private_key_ciphertext, public_key, state, created_at) VALUES ($1, 'RS256', $2, $3, 'active', $4) ON CONFLICT (key_id) DO UPDATE SET private_key_ciphertext = EXCLUDED.private_key_ciphertext, public_key = EXCLUDED.public_key, state = 'active', retire_at = NULL`, nextKeyID, privateCiphertext, publicDER, now); err != nil {
		return fmt.Errorf("activate rotated provider signing key: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit provider signing-key rotation: %w", err)
	}
	return nil
}

// RetireExpiredSigningKeys removes only elapsed overlap keys from the JWKS.
// Private material remains encrypted in the durable audit/recovery record.
func (storage *PostgresStorage) RetireExpiredSigningKeys(ctx context.Context) error {
	if _, err := storage.pool.Exec(ctx, `UPDATE auth_identity.provider_signing_keys SET state = 'retired' WHERE state = 'overlap' AND retire_at IS NOT NULL AND retire_at <= $1`, storage.clock().UTC()); err != nil {
		return fmt.Errorf("retire elapsed provider signing keys: %w", err)
	}
	return nil
}

func (storage *PostgresStorage) CreateAuthRequest(ctx context.Context, request *oidc.AuthRequest, userID string) (op.AuthRequest, error) {
	if request == nil || request.ClientID == "" || request.RedirectURI == "" {
		return nil, ErrProviderInvalid
	}
	if len(request.Prompt) == 1 && request.Prompt[0] == oidc.PromptNone {
		return nil, oidc.ErrLoginRequired()
	}
	id, err := randomProviderID("req_")
	if err != nil {
		return nil, err
	}
	now := storage.clock().UTC()
	var challenge, method any
	if request.CodeChallenge != "" {
		challenge, method = request.CodeChallenge, request.CodeChallengeMethod
	}
	if _, err := storage.pool.Exec(ctx, `INSERT INTO auth_identity.oidc_auth_requests(request_id, client_id, redirect_uri, state_value, nonce_value, response_type, response_mode, scopes, code_challenge, code_challenge_method, subject_id, done, auth_time, created_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,NULLIF($11,''),$12,$13,$13)`, id, request.ClientID, request.RedirectURI, request.State, request.Nonce, request.ResponseType, request.ResponseMode, []string(request.Scopes), challenge, method, userID, userID != "", now); err != nil {
		return nil, fmt.Errorf("store auth request: %w", err)
	}
	return storage.AuthRequestByID(ctx, id)
}

func (storage *PostgresStorage) AuthRequestByID(ctx context.Context, id string) (op.AuthRequest, error) {
	return storage.authRequest(ctx, `SELECT request_id, client_id, redirect_uri, state_value, nonce_value, response_type, response_mode, scopes, code_challenge, code_challenge_method, COALESCE(subject_id, ''), done, auth_time FROM auth_identity.oidc_auth_requests WHERE request_id = $1`, id)
}

func (storage *PostgresStorage) AuthRequestByCode(ctx context.Context, code string) (op.AuthRequest, error) {
	hash := sha256.Sum256([]byte(code))
	return storage.authRequest(ctx, `SELECT r.request_id, r.client_id, r.redirect_uri, r.state_value, r.nonce_value, r.response_type, r.response_mode, r.scopes, r.code_challenge, r.code_challenge_method, COALESCE(r.subject_id, ''), r.done, r.auth_time FROM auth_identity.oidc_authorization_codes c JOIN auth_identity.oidc_auth_requests r ON r.request_id = c.request_id WHERE c.code_hash = $1`, hash[:])
}

func (storage *PostgresStorage) authRequest(ctx context.Context, query string, argument any) (op.AuthRequest, error) {
	var request memoryAuthRequest
	var challenge, method *string
	err := storage.pool.QueryRow(ctx, query, argument).Scan(&request.id, &request.clientID, &request.redirectURI, &request.state, &request.nonce, &request.responseType, &request.responseMode, &request.scopes, &challenge, &method, &request.subject, &request.done, &request.authTime)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrProviderNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("load auth request: %w", err)
	}
	if challenge != nil && method != nil {
		request.codeChallenge = &oidc.CodeChallenge{Challenge: *challenge, Method: oidc.CodeChallengeMethod(*method)}
	}
	return &request, nil
}

func (storage *PostgresStorage) SaveAuthCode(ctx context.Context, requestID, code string) error {
	if requestID == "" || code == "" {
		return ErrProviderInvalid
	}
	hash := sha256.Sum256([]byte(code))
	command, err := storage.pool.Exec(ctx, `INSERT INTO auth_identity.oidc_authorization_codes(code_hash, request_id, created_at) VALUES ($1,$2,$3) ON CONFLICT DO NOTHING`, hash[:], requestID, storage.clock().UTC())
	if err != nil {
		return fmt.Errorf("store authorization code: %w", err)
	}
	if command.RowsAffected() != 1 {
		return ErrProviderInvalid
	}
	return nil
}

func (storage *PostgresStorage) DeleteAuthRequest(ctx context.Context, requestID string) error {
	if _, err := storage.pool.Exec(ctx, `DELETE FROM auth_identity.oidc_auth_requests WHERE request_id = $1`, requestID); err != nil {
		return fmt.Errorf("delete auth request: %w", err)
	}
	return nil
}

func (storage *PostgresStorage) CreateAccessToken(ctx context.Context, request op.TokenRequest) (string, time.Time, error) {
	return storage.createAccess(ctx, request)
}

func (storage *PostgresStorage) CreateAccessAndRefreshTokens(ctx context.Context, request op.TokenRequest, current string) (string, string, time.Time, error) {
	clientID := tokenClientID(request)
	if current != "" {
		hash := sha256.Sum256([]byte(current))
		command, err := storage.pool.Exec(ctx, `UPDATE auth_identity.oidc_refresh_tokens SET state = 'rotated', revoked_at = $2 WHERE token_hash = $1 AND client_id = $3 AND subject_id = $4 AND state = 'active' AND expires_at > $2`, hash[:], storage.clock().UTC(), clientID, request.GetSubject())
		if err != nil || command.RowsAffected() != 1 {
			return "", "", time.Time{}, oidc.ErrInvalidGrant()
		}
	}
	accessID, expiry, err := storage.createAccess(ctx, request)
	if err != nil {
		return "", "", time.Time{}, err
	}
	raw, err := randomProviderID("rt_")
	if err != nil {
		return "", "", time.Time{}, err
	}
	hash := sha256.Sum256([]byte(raw))
	authTime, amr := storage.clock().UTC(), []string{"pwd"}
	if value, ok := request.(interface{ GetAuthTime() time.Time }); ok {
		authTime = value.GetAuthTime()
	}
	if value, ok := request.(interface{ GetAMR() []string }); ok {
		amr = value.GetAMR()
	}
	if _, err := storage.pool.Exec(ctx, `INSERT INTO auth_identity.oidc_refresh_tokens(token_hash, access_token_id, client_id, subject_id, audience, scopes, auth_time, amr, expires_at, state, created_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,'active',$10)`, hash[:], accessID, clientID, request.GetSubject(), request.GetAudience(), request.GetScopes(), authTime, amr, expiry.Add(30*time.Minute), storage.clock().UTC()); err != nil {
		return "", "", time.Time{}, fmt.Errorf("store refresh token: %w", err)
	}
	return accessID, raw, expiry, nil
}

func (storage *PostgresStorage) createAccess(ctx context.Context, request op.TokenRequest) (string, time.Time, error) {
	id, err := randomProviderID("at_")
	if err != nil {
		return "", time.Time{}, err
	}
	expires := storage.clock().UTC().Add(time.Hour)
	if _, err := storage.pool.Exec(ctx, `INSERT INTO auth_identity.oidc_access_tokens(token_id, client_id, subject_id, audience, scopes, expires_at, state, created_at) VALUES ($1,$2,$3,$4,$5,$6,'active',$7)`, id, tokenClientID(request), request.GetSubject(), request.GetAudience(), request.GetScopes(), expires, storage.clock().UTC()); err != nil {
		return "", time.Time{}, fmt.Errorf("store access token: %w", err)
	}
	return id, expires, nil
}

func tokenClientID(request op.TokenRequest) string {
	if value, ok := request.(interface{ GetClientID() string }); ok {
		return value.GetClientID()
	}
	return ""
}

func (storage *PostgresStorage) TokenRequestByRefreshToken(ctx context.Context, raw string) (op.RefreshTokenRequest, error) {
	hash := sha256.Sum256([]byte(raw))
	var token memoryRefreshToken
	var storedHash []byte
	err := storage.pool.QueryRow(ctx, `SELECT token_hash, access_token_id, client_id, subject_id, audience, scopes, auth_time, amr, expires_at FROM auth_identity.oidc_refresh_tokens WHERE token_hash = $1 AND state = 'active' AND expires_at > $2`, hash[:], storage.clock().UTC()).Scan(&storedHash, &token.accessID, &token.clientID, &token.subject, &token.audience, &token.scopes, &token.authTime, &token.amr, &token.expiresAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, op.ErrInvalidRefreshToken
	}
	if err != nil {
		return nil, fmt.Errorf("load refresh token: %w", err)
	}
	if len(storedHash) != len(token.tokenHash) {
		return nil, op.ErrInvalidRefreshToken
	}
	copy(token.tokenHash[:], storedHash)
	return &memoryRefreshRequest{memoryRefreshToken: &token}, nil
}

func (storage *PostgresStorage) TerminateSession(ctx context.Context, subject, client string) error {
	tx, err := storage.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin OIDC session termination: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	now := storage.clock().UTC()
	if _, err := tx.Exec(ctx, `UPDATE auth_identity.oidc_access_tokens SET state = 'revoked', revoked_at = $3 WHERE subject_id = $1 AND client_id = $2 AND state = 'active'`, subject, client, now); err != nil {
		return fmt.Errorf("revoke OIDC access tokens: %w", err)
	}
	if _, err := tx.Exec(ctx, `UPDATE auth_identity.oidc_refresh_tokens SET state = 'revoked', revoked_at = $3 WHERE subject_id = $1 AND client_id = $2 AND state = 'active'`, subject, client, now); err != nil {
		return fmt.Errorf("revoke OIDC refresh tokens: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit OIDC session termination: %w", err)
	}
	return nil
}

func (storage *PostgresStorage) GetRefreshTokenInfo(ctx context.Context, client, raw string) (string, string, error) {
	hash := sha256.Sum256([]byte(raw))
	var subject, access string
	err := storage.pool.QueryRow(ctx, `SELECT subject_id, access_token_id FROM auth_identity.oidc_refresh_tokens WHERE token_hash = $1 AND client_id = $2 AND state = 'active' AND expires_at > $3`, hash[:], client, storage.clock().UTC()).Scan(&subject, &access)
	if err != nil {
		return "", "", op.ErrInvalidRefreshToken
	}
	return subject, access, nil
}

func (storage *PostgresStorage) RevokeToken(ctx context.Context, token, subject, client string) *oidc.Error {
	now := storage.clock().UTC()
	if subject != "" {
		command, err := storage.pool.Exec(ctx, `UPDATE auth_identity.oidc_access_tokens SET state = 'revoked', revoked_at = $4 WHERE token_id = $1 AND subject_id = $2 AND client_id = $3 AND state = 'active'`, token, subject, client, now)
		if err != nil || command.RowsAffected() == 0 {
			return oidc.ErrInvalidClient()
		}
		return nil
	}
	hash := sha256.Sum256([]byte(token))
	command, err := storage.pool.Exec(ctx, `UPDATE auth_identity.oidc_refresh_tokens SET state = 'revoked', revoked_at = $3 WHERE token_hash = $1 AND client_id = $2 AND state = 'active'`, hash[:], client, now)
	if err != nil || command.RowsAffected() == 0 {
		return oidc.ErrInvalidClient()
	}
	return nil
}

func (storage *PostgresStorage) SigningKey(ctx context.Context) (op.SigningKey, error) {
	var id string
	var encrypted []byte
	err := storage.pool.QueryRow(ctx, `SELECT key_id, private_key_ciphertext FROM auth_identity.provider_signing_keys WHERE state = 'active'`).Scan(&id, &encrypted)
	if err != nil {
		return nil, fmt.Errorf("load active signing key: %w", err)
	}
	raw, err := storage.decryptKey(id, encrypted)
	if err != nil {
		return nil, err
	}
	key, err := x509.ParsePKCS1PrivateKey(raw)
	if err != nil {
		return nil, ErrProviderUnavailable
	}
	return memorySigningKey{id: id, algorithm: jose.RS256, private: key}, nil
}
func (storage *PostgresStorage) SignatureAlgorithms(context.Context) ([]jose.SignatureAlgorithm, error) {
	return []jose.SignatureAlgorithm{jose.RS256}, nil
}
func (storage *PostgresStorage) KeySet(ctx context.Context) ([]op.Key, error) {
	rows, err := storage.pool.Query(ctx, `SELECT key_id, public_key FROM auth_identity.provider_signing_keys WHERE state = 'active' OR (state = 'overlap' AND (retire_at IS NULL OR retire_at > $1))`, storage.clock().UTC())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	keys := []op.Key{}
	for rows.Next() {
		var id string
		var raw []byte
		if err := rows.Scan(&id, &raw); err != nil {
			return nil, err
		}
		key, err := x509.ParsePKIXPublicKey(raw)
		if err != nil {
			return nil, ErrProviderUnavailable
		}
		rsaKey, ok := key.(*rsa.PublicKey)
		if !ok {
			return nil, ErrProviderUnavailable
		}
		keys = append(keys, durablePublicKey{id: id, key: rsaKey})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return keys, nil
}

type durablePublicKey struct {
	id  string
	key *rsa.PublicKey
}

func (key durablePublicKey) ID() string                         { return key.id }
func (key durablePublicKey) Algorithm() jose.SignatureAlgorithm { return jose.RS256 }
func (key durablePublicKey) Use() string                        { return "sig" }
func (key durablePublicKey) Key() any                           { return key.key }

func (storage *PostgresStorage) GetClientByClientID(ctx context.Context, id string) (op.Client, error) {
	var redirects, logouts, scopes []string
	var state string
	err := storage.pool.QueryRow(ctx, `SELECT redirect_uris, post_logout_redirect_uris, scopes, state FROM auth_identity.provider_clients WHERE client_id = $1`, id).Scan(&redirects, &logouts, &scopes, &state)
	if errors.Is(err, pgx.ErrNoRows) || state != "active" {
		return nil, ErrProviderNotFound
	}
	if err != nil {
		return nil, err
	}
	if len(redirects) != 1 || len(logouts) != 1 || !scopesAllowed(scopes) {
		return nil, ErrProviderInvalid
	}
	return memoryClient{id: id, redirectURI: redirects[0], postLogoutRedirect: logouts[0], loginURL: "/login"}, nil
}
func (storage *PostgresStorage) AuthorizeClientIDSecret(ctx context.Context, id, secret string) error {
	var expected []byte
	var state string
	err := storage.pool.QueryRow(ctx, `SELECT secret_hash, state FROM auth_identity.provider_clients WHERE client_id = $1`, id).Scan(&expected, &state)
	actual := sha256.Sum256([]byte(secret))
	if err != nil || state != "active" || subtle.ConstantTimeCompare(expected, actual[:]) != 1 {
		return oidc.ErrInvalidClient()
	}
	return nil
}
func scopesAllowed(scopes []string) bool {
	for _, scope := range scopes {
		if !containsScope([]string{oidc.ScopeOpenID, oidc.ScopeProfile, oidc.ScopeEmail, oidc.ScopeOfflineAccess}, scope) {
			return false
		}
	}
	return true
}

func (storage *PostgresStorage) SetUserinfoFromScopes(ctx context.Context, info *oidc.UserInfo, subject, _ string, scopes []string) error {
	return storage.setUserinfo(ctx, info, subject, scopes)
}
func (storage *PostgresStorage) SetUserinfoFromRequest(ctx context.Context, info *oidc.UserInfo, request op.IDTokenRequest, scopes []string) error {
	return storage.setUserinfo(ctx, info, request.GetSubject(), scopes)
}
func (storage *PostgresStorage) SetUserinfoFromToken(ctx context.Context, info *oidc.UserInfo, token, subject, _ string) error {
	var scopes []string
	err := storage.pool.QueryRow(ctx, `SELECT scopes FROM auth_identity.oidc_access_tokens WHERE token_id=$1 AND subject_id=$2 AND state='active' AND expires_at>$3`, token, subject, storage.clock().UTC()).Scan(&scopes)
	if err != nil {
		return oidc.ErrUnauthorizedClient()
	}
	return storage.setUserinfo(ctx, info, subject, scopes)
}
func (storage *PostgresStorage) SetIntrospectionFromToken(ctx context.Context, info *oidc.IntrospectionResponse, token, subject, client string) error {
	var expires time.Time
	err := storage.pool.QueryRow(ctx, `SELECT expires_at FROM auth_identity.oidc_access_tokens WHERE token_id=$1 AND subject_id=$2 AND client_id=$3 AND state='active' AND expires_at>$4`, token, subject, client, storage.clock().UTC()).Scan(&expires)
	if err != nil {
		return oidc.ErrUnauthorizedClient()
	}
	info.Active = true
	info.Subject = subject
	info.ClientID = client
	info.Expiration = oidc.FromTime(expires)
	return nil
}
func (storage *PostgresStorage) setUserinfo(ctx context.Context, info *oidc.UserInfo, subject string, scopes []string) error {
	if info == nil || subject == "" {
		return ErrProviderInvalid
	}
	info.Subject = subject
	if containsScope(scopes, oidc.ScopeProfile) {
		info.Name = subject
		info.PreferredUsername = subject
	}
	if containsScope(scopes, oidc.ScopeEmail) {
		var email string
		var verified bool
		err := storage.pool.QueryRow(ctx, `SELECT i.normalized_value, a.email_verified FROM auth_identity.accounts a JOIN auth_identity.identifiers i ON i.subject_id=a.subject_id AND i.identifier_type='email' WHERE a.subject_id=$1`, subject).Scan(&email, &verified)
		if err != nil {
			return ErrProviderNotFound
		}
		info.Email = email
		info.EmailVerified = oidc.Bool(verified)
	}
	return nil
}
func (storage *PostgresStorage) GetPrivateClaimsFromScopes(context.Context, string, string, []string) (map[string]any, error) {
	return map[string]any{}, nil
}
func (storage *PostgresStorage) GetKeyByIDAndClientID(context.Context, string, string) (*jose.JSONWebKey, error) {
	return nil, ErrProviderNotFound
}
func (storage *PostgresStorage) ValidateJWTProfileScopes(context.Context, string, []string) ([]string, error) {
	return nil, oidc.ErrUnauthorizedClient()
}

func (storage *PostgresStorage) Authorize(ctx context.Context, requestID, subject string) error {
	command, err := storage.pool.Exec(ctx, `UPDATE auth_identity.oidc_auth_requests SET subject_id=$2, done=true, auth_time=$3 WHERE request_id=$1 AND done=false`, requestID, subject, storage.clock().UTC())
	if err != nil {
		return err
	}
	if command.RowsAffected() != 1 {
		return ErrProviderNotFound
	}
	return nil
}

// StageAuthenticatedSubject records a successful primary authentication while
// deliberately leaving the request unfinished until MFA is completed.
func (storage *PostgresStorage) StageAuthenticatedSubject(ctx context.Context, requestID, subject string) error {
	command, err := storage.pool.Exec(ctx, `UPDATE auth_identity.oidc_auth_requests SET subject_id = $2, auth_time = $3 WHERE request_id = $1 AND done = false`, requestID, subject, storage.clock().UTC())
	if err != nil {
		return fmt.Errorf("stage authenticated subject: %w", err)
	}
	if command.RowsAffected() != 1 {
		return ErrProviderNotFound
	}
	return nil
}

func (storage *PostgresStorage) encryptKey(id string, plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(storage.encryptionKey)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	return gcm.Seal(nonce, nonce, plaintext, []byte("as360-oidc-key-v1\x00"+id)), nil
}
func (storage *PostgresStorage) decryptKey(id string, ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(storage.encryptionKey)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil || len(ciphertext) < gcm.NonceSize() {
		return nil, ErrProviderUnavailable
	}
	return gcm.Open(nil, ciphertext[:gcm.NonceSize()], ciphertext[gcm.NonceSize():], []byte("as360-oidc-key-v1\x00"+id))
}

var _ op.Storage = (*PostgresStorage)(nil)
