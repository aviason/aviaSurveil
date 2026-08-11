package provider

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/subtle"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	jose "github.com/go-jose/go-jose/v4"
	"github.com/zitadel/oidc/v3/pkg/oidc"
	"github.com/zitadel/oidc/v3/pkg/op"
)

var (
	ErrProviderNotFound    = errors.New("provider record not found")
	ErrProviderInvalid     = errors.New("provider record is invalid")
	ErrProviderUnavailable = errors.New("provider storage unavailable")
)

// CandidateConfig is deliberately explicit. It is used by the isolated
// protocol qualification harness; production wiring will supply the durable
// identity/session adapters after the remaining tasks are complete.
type CandidateConfig struct {
	Issuer                string
	AllowInsecure         bool
	CryptoKey             [32]byte
	CryptoKeyID           string
	SigningKey            *rsa.PrivateKey
	SigningKeyID          string
	ClientID              string
	ClientSecret          string
	RedirectURI           string
	PostLogoutRedirectURI string
	SubjectID             string
	Email                 string
	DisplayName           string
}

type Candidate struct {
	Handler  http.Handler
	Provider *op.Provider
	Storage  *MemoryStorage
}

func NewCandidate(configuration CandidateConfig) (*Candidate, error) {
	if err := validateCandidateConfig(configuration); err != nil {
		return nil, err
	}
	storage := NewMemoryStorage(configuration)
	providerConfig := &op.Config{
		CryptoKey:                configuration.CryptoKey,
		CryptoKeyId:              configuration.CryptoKeyID,
		DefaultLogoutRedirectURI: configuration.PostLogoutRedirectURI,
		CodeMethodS256:           true,
		AuthMethodPost:           false,
		AuthMethodPrivateKeyJWT:  false,
		GrantTypeRefreshToken:    true,
		RequestObjectSupported:   false,
		SupportedScopes:          []string{oidc.ScopeOpenID, oidc.ScopeProfile, oidc.ScopeEmail, oidc.ScopeOfflineAccess},
	}
	options := []op.Option{}
	if configuration.AllowInsecure {
		options = append(options, op.WithAllowInsecure())
	}
	provider, err := op.NewProvider(providerConfig, storage, op.StaticIssuer(configuration.Issuer), options...)
	if err != nil {
		return nil, fmt.Errorf("initialize OIDC provider: %w", err)
	}
	mux := http.NewServeMux()
	// The selected library validates PKCE when a challenge is present, while
	// AS360-OIDC-WEB-1 requires S256 PKCE for every browser authorization-code
	// request. Keep that stronger application contract at the narrow route
	// boundary so an omitted challenge cannot silently downgrade the flow.
	mux.HandleFunc("/authorize", func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Query().Get("response_type") == string(oidc.ResponseTypeCode) {
			if request.URL.Query().Get("code_challenge") == "" || request.URL.Query().Get("code_challenge_method") != "S256" {
				http.Error(writer, "PKCE S256 is required", http.StatusBadRequest)
				return
			}
		}
		provider.ServeHTTP(writer, request)
	})
	// The login endpoint represents the provider-owned identity UI boundary in
	// this disposable harness. It marks the preconfigured synthetic subject and
	// immediately returns to the library-owned callback.
	mux.HandleFunc("/login", func(writer http.ResponseWriter, request *http.Request) {
		requestID := request.URL.Query().Get("id")
		if err := storage.Authorize(requestID); err != nil {
			http.Error(writer, "login failed", http.StatusUnauthorized)
			return
		}
		callback := op.AuthCallbackURL(provider)(request.Context(), requestID)
		http.Redirect(writer, request, callback, http.StatusFound)
	})
	mux.Handle("/", provider)
	return &Candidate{Handler: mux, Provider: provider, Storage: storage}, nil
}

func validateCandidateConfig(configuration CandidateConfig) error {
	if strings.TrimSpace(configuration.Issuer) == "" || configuration.CryptoKeyID == "" || configuration.SigningKeyID == "" || configuration.ClientID == "" || configuration.RedirectURI == "" || configuration.PostLogoutRedirectURI == "" || configuration.SubjectID == "" || configuration.Email == "" {
		return ErrProviderInvalid
	}
	if configuration.SigningKey == nil || configuration.SigningKey.N == nil || configuration.SigningKey.N.BitLen() < 2048 {
		return errors.New("provider requires an RSA signing key of at least 2048 bits")
	}
	issuer, err := url.Parse(configuration.Issuer)
	if err != nil || issuer.User != nil || issuer.Host == "" || issuer.RawQuery != "" || issuer.Fragment != "" || (issuer.Scheme != "https" && !(configuration.AllowInsecure && issuer.Scheme == "http")) {
		return errors.New("provider issuer must be an absolute HTTPS URL (or explicitly allowed local HTTP)")
	}
	redirect, err := url.Parse(configuration.RedirectURI)
	if err != nil || !redirect.IsAbs() || redirect.User != nil || redirect.Fragment != "" {
		return errors.New("provider redirect URI is invalid")
	}
	logout, err := url.Parse(configuration.PostLogoutRedirectURI)
	if err != nil || !logout.IsAbs() || logout.User != nil || logout.Fragment != "" {
		return errors.New("provider post-logout redirect URI is invalid")
	}
	return nil
}

// MemoryStorage is a disposable, single-process implementation of the
// zitadel/oidc storage contract. Raw refresh credentials are returned once and
// retained only as SHA-256 hashes; authorization codes are encrypted by the
// library and remain single-use through DeleteAuthRequest.
type MemoryStorage struct {
	mu            sync.Mutex
	configuration CandidateConfig
	clients       map[string]*memoryClient
	authRequests  map[string]*memoryAuthRequest
	codes         map[string]string
	accessTokens  map[string]*memoryAccessToken
	refreshTokens map[[32]byte]*memoryRefreshToken
	signingKey    memorySigningKey
}

type memorySigningKey struct {
	id        string
	algorithm jose.SignatureAlgorithm
	private   *rsa.PrivateKey
}

func (key memorySigningKey) SignatureAlgorithm() jose.SignatureAlgorithm { return key.algorithm }
func (key memorySigningKey) Key() any                                    { return key.private }
func (key memorySigningKey) ID() string                                  { return key.id }

type memoryPublicKey struct{ memorySigningKey }

func (key memoryPublicKey) ID() string                         { return key.id }
func (key memoryPublicKey) Algorithm() jose.SignatureAlgorithm { return key.algorithm }
func (key memoryPublicKey) Use() string                        { return "sig" }
func (key memoryPublicKey) Key() any                           { return &key.private.PublicKey }

type memoryClient struct {
	id                 string
	secret             string
	redirectURI        string
	postLogoutRedirect string
	loginURL           string
}

func (client memoryClient) GetID() string          { return client.id }
func (client memoryClient) RedirectURIs() []string { return []string{client.redirectURI} }
func (client memoryClient) PostLogoutRedirectURIs() []string {
	return []string{client.postLogoutRedirect}
}
func (client memoryClient) ApplicationType() op.ApplicationType { return op.ApplicationTypeWeb }
func (client memoryClient) AuthMethod() oidc.AuthMethod         { return oidc.AuthMethodBasic }
func (client memoryClient) ResponseTypes() []oidc.ResponseType {
	return []oidc.ResponseType{oidc.ResponseTypeCode}
}
func (client memoryClient) GrantTypes() []oidc.GrantType {
	return []oidc.GrantType{oidc.GrantTypeCode, oidc.GrantTypeRefreshToken}
}
func (client memoryClient) LoginURL(id string) string {
	return client.loginURL + "?id=" + url.QueryEscape(id)
}
func (client memoryClient) AccessTokenType() op.AccessTokenType { return op.AccessTokenTypeBearer }
func (client memoryClient) IDTokenLifetime() time.Duration      { return time.Hour }
func (client memoryClient) DevMode() bool                       { return false }
func (client memoryClient) RestrictAdditionalIdTokenScopes() func([]string) []string {
	return func(scopes []string) []string { return scopes }
}
func (client memoryClient) RestrictAdditionalAccessTokenScopes() func([]string) []string {
	return func(scopes []string) []string { return scopes }
}
func (client memoryClient) IsScopeAllowed(scope string) bool {
	return scope == oidc.ScopeOpenID || scope == oidc.ScopeProfile || scope == oidc.ScopeEmail || scope == oidc.ScopeOfflineAccess
}
func (client memoryClient) IDTokenUserinfoClaimsAssertion() bool { return true }
func (client memoryClient) ClockSkew() time.Duration             { return 30 * time.Second }

type memoryAuthRequest struct {
	id            string
	clientID      string
	redirectURI   string
	state         string
	nonce         string
	responseType  oidc.ResponseType
	responseMode  oidc.ResponseMode
	scopes        []string
	codeChallenge *oidc.CodeChallenge
	subject       string
	done          bool
	authTime      time.Time
}

func (request *memoryAuthRequest) GetID() string          { return request.id }
func (request *memoryAuthRequest) GetACR() string         { return "" }
func (request *memoryAuthRequest) GetAMR() []string       { return []string{"pwd"} }
func (request *memoryAuthRequest) GetAudience() []string  { return []string{request.clientID} }
func (request *memoryAuthRequest) GetAuthTime() time.Time { return request.authTime }
func (request *memoryAuthRequest) GetClientID() string    { return request.clientID }
func (request *memoryAuthRequest) GetCodeChallenge() *oidc.CodeChallenge {
	return request.codeChallenge
}
func (request *memoryAuthRequest) GetNonce() string                   { return request.nonce }
func (request *memoryAuthRequest) GetRedirectURI() string             { return request.redirectURI }
func (request *memoryAuthRequest) GetResponseType() oidc.ResponseType { return request.responseType }
func (request *memoryAuthRequest) GetResponseMode() oidc.ResponseMode { return request.responseMode }
func (request *memoryAuthRequest) GetScopes() []string {
	return append([]string(nil), request.scopes...)
}
func (request *memoryAuthRequest) GetState() string   { return request.state }
func (request *memoryAuthRequest) GetSubject() string { return request.subject }
func (request *memoryAuthRequest) Done() bool         { return request.done }

type memoryRefreshRequest struct{ *memoryRefreshToken }

func (request *memoryRefreshRequest) GetAMR() []string { return append([]string(nil), request.amr...) }
func (request *memoryRefreshRequest) GetAudience() []string {
	return append([]string(nil), request.audience...)
}
func (request *memoryRefreshRequest) GetAuthTime() time.Time { return request.authTime }
func (request *memoryRefreshRequest) GetClientID() string    { return request.clientID }
func (request *memoryRefreshRequest) GetScopes() []string {
	return append([]string(nil), request.scopes...)
}
func (request *memoryRefreshRequest) GetSubject() string { return request.subject }
func (request *memoryRefreshRequest) SetCurrentScopes(scopes []string) {
	request.scopes = append([]string(nil), scopes...)
}

type memoryAccessToken struct {
	id        string
	clientID  string
	subject   string
	scopes    []string
	expiresAt time.Time
}

type memoryRefreshToken struct {
	tokenHash [32]byte
	accessID  string
	clientID  string
	subject   string
	audience  []string
	scopes    []string
	authTime  time.Time
	amr       []string
	expiresAt time.Time
}

func NewMemoryStorage(configuration CandidateConfig) *MemoryStorage {
	client := &memoryClient{
		id: configuration.ClientID, secret: configuration.ClientSecret,
		redirectURI: configuration.RedirectURI, postLogoutRedirect: configuration.PostLogoutRedirectURI,
		loginURL: "/login",
	}
	return &MemoryStorage{
		configuration: configuration,
		clients:       map[string]*memoryClient{configuration.ClientID: client},
		authRequests:  make(map[string]*memoryAuthRequest),
		codes:         make(map[string]string),
		accessTokens:  make(map[string]*memoryAccessToken),
		refreshTokens: make(map[[32]byte]*memoryRefreshToken),
		signingKey:    memorySigningKey{id: configuration.SigningKeyID, algorithm: jose.RS256, private: configuration.SigningKey},
	}
}

func (storage *MemoryStorage) Health(context.Context) error { return nil }

func (storage *MemoryStorage) CreateAuthRequest(_ context.Context, request *oidc.AuthRequest, _ string) (op.AuthRequest, error) {
	if request == nil || request.ClientID == "" || request.RedirectURI == "" {
		return nil, ErrProviderInvalid
	}
	if len(request.Prompt) == 1 && request.Prompt[0] == oidc.PromptNone {
		return nil, oidc.ErrLoginRequired()
	}
	requestID, err := randomProviderID("req_")
	if err != nil {
		return nil, err
	}
	copyRequest := &memoryAuthRequest{
		id: requestID, clientID: request.ClientID, redirectURI: request.RedirectURI,
		state: request.State, nonce: request.Nonce, responseType: request.ResponseType,
		responseMode: request.ResponseMode, scopes: append([]string(nil), request.Scopes...),
		subject: storage.configuration.SubjectID, authTime: time.Now().UTC(),
	}
	if request.CodeChallenge != "" {
		copyRequest.codeChallenge = &oidc.CodeChallenge{Challenge: request.CodeChallenge, Method: request.CodeChallengeMethod}
	}
	storage.mu.Lock()
	storage.authRequests[requestID] = copyRequest
	storage.mu.Unlock()
	return copyRequest, nil
}

func (storage *MemoryStorage) AuthRequestByID(_ context.Context, id string) (op.AuthRequest, error) {
	storage.mu.Lock()
	defer storage.mu.Unlock()
	request, ok := storage.authRequests[id]
	if !ok {
		return nil, ErrProviderNotFound
	}
	return request, nil
}

func (storage *MemoryStorage) AuthRequestByCode(ctx context.Context, code string) (op.AuthRequest, error) {
	storage.mu.Lock()
	requestID, ok := storage.codes[code]
	storage.mu.Unlock()
	if !ok {
		return nil, ErrProviderNotFound
	}
	return storage.AuthRequestByID(ctx, requestID)
}

func (storage *MemoryStorage) SaveAuthCode(_ context.Context, requestID, code string) error {
	if requestID == "" || code == "" {
		return ErrProviderInvalid
	}
	storage.mu.Lock()
	storage.codes[code] = requestID
	storage.mu.Unlock()
	return nil
}

func (storage *MemoryStorage) DeleteAuthRequest(_ context.Context, requestID string) error {
	storage.mu.Lock()
	defer storage.mu.Unlock()
	delete(storage.authRequests, requestID)
	for code, id := range storage.codes {
		if id == requestID {
			delete(storage.codes, code)
		}
	}
	return nil
}

func (storage *MemoryStorage) CreateAccessToken(_ context.Context, request op.TokenRequest) (string, time.Time, error) {
	clientID := ""
	if clientRequest, ok := request.(interface{ GetClientID() string }); ok {
		clientID = clientRequest.GetClientID()
	}
	return storage.createAccessToken(request.GetSubject(), request.GetAudience(), request.GetScopes(), clientID)
}

func (storage *MemoryStorage) CreateAccessAndRefreshTokens(_ context.Context, request op.TokenRequest, currentRefreshToken string) (string, string, time.Time, error) {
	clientID := ""
	if clientRequest, ok := request.(interface{ GetClientID() string }); ok {
		clientID = clientRequest.GetClientID()
	}
	if currentRefreshToken != "" {
		hash := sha256.Sum256([]byte(currentRefreshToken))
		storage.mu.Lock()
		previous, ok := storage.refreshTokens[hash]
		if ok {
			delete(storage.refreshTokens, hash)
		}
		storage.mu.Unlock()
		if !ok || previous.subject != request.GetSubject() || previous.clientID != clientID {
			return "", "", time.Time{}, oidc.ErrInvalidGrant()
		}
	}
	accessID, expiry, err := storage.createAccessToken(request.GetSubject(), request.GetAudience(), request.GetScopes(), clientID)
	if err != nil {
		return "", "", time.Time{}, err
	}
	rawRefresh, err := randomProviderID("rt_")
	if err != nil {
		return "", "", time.Time{}, err
	}
	hash := sha256.Sum256([]byte(rawRefresh))
	authTime := time.Now().UTC()
	if authRequest, ok := request.(interface{ GetAuthTime() time.Time }); ok {
		authTime = authRequest.GetAuthTime()
	}
	amr := []string{"pwd"}
	if authRequest, ok := request.(interface{ GetAMR() []string }); ok {
		amr = authRequest.GetAMR()
	}
	refresh := &memoryRefreshToken{
		tokenHash: hash, accessID: accessID, clientID: clientID, subject: request.GetSubject(),
		audience: request.GetAudience(), scopes: request.GetScopes(), authTime: authTime,
		amr: amr, expiresAt: expiry.Add(30 * time.Minute),
	}
	storage.mu.Lock()
	storage.refreshTokens[hash] = refresh
	storage.mu.Unlock()
	return accessID, rawRefresh, expiry, nil
}

func (storage *MemoryStorage) TokenRequestByRefreshToken(_ context.Context, raw string) (op.RefreshTokenRequest, error) {
	hash := sha256.Sum256([]byte(raw))
	storage.mu.Lock()
	defer storage.mu.Unlock()
	refresh, ok := storage.refreshTokens[hash]
	if !ok || !time.Now().Before(refresh.expiresAt) {
		return nil, op.ErrInvalidRefreshToken
	}
	copyRefresh := *refresh
	return &memoryRefreshRequest{memoryRefreshToken: &copyRefresh}, nil
}

func (storage *MemoryStorage) TerminateSession(_ context.Context, subjectID, clientID string) error {
	storage.mu.Lock()
	defer storage.mu.Unlock()
	for id, token := range storage.accessTokens {
		if token.subject == subjectID && token.clientID == clientID {
			delete(storage.accessTokens, id)
		}
	}
	for hash, token := range storage.refreshTokens {
		if token.subject == subjectID && token.clientID == clientID {
			delete(storage.refreshTokens, hash)
		}
	}
	return nil
}

func (storage *MemoryStorage) GetRefreshTokenInfo(_ context.Context, clientID, raw string) (string, string, error) {
	hash := sha256.Sum256([]byte(raw))
	storage.mu.Lock()
	defer storage.mu.Unlock()
	refresh, ok := storage.refreshTokens[hash]
	if !ok || refresh.clientID != clientID {
		return "", "", op.ErrInvalidRefreshToken
	}
	return refresh.subject, refresh.accessID, nil
}

func (storage *MemoryStorage) RevokeToken(_ context.Context, tokenIDOrRaw, subjectID, clientID string) *oidc.Error {
	storage.mu.Lock()
	defer storage.mu.Unlock()
	if token, ok := storage.accessTokens[tokenIDOrRaw]; ok {
		if token.clientID != clientID || (subjectID != "" && token.subject != subjectID) {
			return oidc.ErrInvalidClient()
		}
		delete(storage.accessTokens, tokenIDOrRaw)
		return nil
	}
	hash := sha256.Sum256([]byte(tokenIDOrRaw))
	if token, ok := storage.refreshTokens[hash]; ok {
		if token.clientID != clientID || (subjectID != "" && token.subject != subjectID) {
			return oidc.ErrInvalidClient()
		}
		delete(storage.refreshTokens, hash)
		delete(storage.accessTokens, token.accessID)
	}
	return nil
}

func (storage *MemoryStorage) SigningKey(context.Context) (op.SigningKey, error) {
	return storage.signingKey, nil
}
func (storage *MemoryStorage) SignatureAlgorithms(context.Context) ([]jose.SignatureAlgorithm, error) {
	return []jose.SignatureAlgorithm{jose.RS256}, nil
}
func (storage *MemoryStorage) KeySet(context.Context) ([]op.Key, error) {
	return []op.Key{memoryPublicKey{storage.signingKey}}, nil
}

func (storage *MemoryStorage) GetClientByClientID(_ context.Context, clientID string) (op.Client, error) {
	storage.mu.Lock()
	defer storage.mu.Unlock()
	client, ok := storage.clients[clientID]
	if !ok {
		return nil, ErrProviderNotFound
	}
	return *client, nil
}

func (storage *MemoryStorage) AuthorizeClientIDSecret(_ context.Context, clientID, clientSecret string) error {
	storage.mu.Lock()
	client, ok := storage.clients[clientID]
	storage.mu.Unlock()
	if !ok || subtle.ConstantTimeCompare([]byte(client.secret), []byte(clientSecret)) != 1 {
		return oidc.ErrInvalidClient()
	}
	return nil
}

func (storage *MemoryStorage) SetUserinfoFromScopes(_ context.Context, userinfo *oidc.UserInfo, userID, _ string, scopes []string) error {
	return storage.setUserinfo(userinfo, userID, scopes)
}

func (storage *MemoryStorage) SetUserinfoFromRequest(_ context.Context, userinfo *oidc.UserInfo, request op.IDTokenRequest, scopes []string) error {
	return storage.setUserinfo(userinfo, request.GetSubject(), scopes)
}

func (storage *MemoryStorage) SetUserinfoFromToken(_ context.Context, userinfo *oidc.UserInfo, tokenID, subject, _ string) error {
	storage.mu.Lock()
	token, ok := storage.accessTokens[tokenID]
	storage.mu.Unlock()
	if !ok || token.subject != subject || !time.Now().Before(token.expiresAt) {
		return oidc.ErrUnauthorizedClient()
	}
	return storage.setUserinfo(userinfo, subject, token.scopes)
}

func (storage *MemoryStorage) SetIntrospectionFromToken(_ context.Context, introspection *oidc.IntrospectionResponse, tokenID, subject, clientID string) error {
	storage.mu.Lock()
	token, ok := storage.accessTokens[tokenID]
	storage.mu.Unlock()
	if !ok || token.subject != subject || token.clientID != clientID || !time.Now().Before(token.expiresAt) {
		return oidc.ErrUnauthorizedClient()
	}
	introspection.Active = true
	introspection.Subject = token.subject
	introspection.ClientID = token.clientID
	introspection.Expiration = oidc.FromTime(token.expiresAt)
	return nil
}

func (storage *MemoryStorage) GetPrivateClaimsFromScopes(_ context.Context, _ string, _ string, _ []string) (map[string]any, error) {
	return map[string]any{}, nil
}

func (storage *MemoryStorage) GetKeyByIDAndClientID(context.Context, string, string) (*jose.JSONWebKey, error) {
	return nil, ErrProviderNotFound
}

func (storage *MemoryStorage) ValidateJWTProfileScopes(context.Context, string, []string) ([]string, error) {
	return nil, oidc.ErrUnauthorizedClient()
}

func (storage *MemoryStorage) Authorize(requestID string) error {
	storage.mu.Lock()
	defer storage.mu.Unlock()
	request, ok := storage.authRequests[requestID]
	if !ok {
		return ErrProviderNotFound
	}
	request.subject = storage.configuration.SubjectID
	request.done = true
	request.authTime = time.Now().UTC()
	return nil
}

func (storage *MemoryStorage) createAccessToken(subject string, audience, scopes []string, clientID string) (string, time.Time, error) {
	id, err := randomProviderID("at_")
	if err != nil {
		return "", time.Time{}, err
	}
	expiresAt := time.Now().UTC().Add(time.Hour)
	storage.mu.Lock()
	storage.accessTokens[id] = &memoryAccessToken{id: id, clientID: clientID, subject: subject, scopes: append([]string(nil), scopes...), expiresAt: expiresAt}
	storage.mu.Unlock()
	_ = audience
	return id, expiresAt, nil
}

func (storage *MemoryStorage) setUserinfo(userinfo *oidc.UserInfo, subject string, scopes []string) error {
	if userinfo == nil || subject == "" {
		return ErrProviderInvalid
	}
	userinfo.Subject = subject
	if containsScope(scopes, oidc.ScopeProfile) {
		userinfo.Name = storage.configuration.DisplayName
		userinfo.PreferredUsername = subject
	}
	if containsScope(scopes, oidc.ScopeEmail) {
		userinfo.Email = storage.configuration.Email
		userinfo.EmailVerified = true
	}
	return nil
}

func containsScope(scopes []string, wanted string) bool {
	for _, scope := range scopes {
		if scope == wanted {
			return true
		}
	}
	return false
}

func randomProviderID(prefix string) (string, error) {
	var raw [24]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", err
	}
	return prefix + fmt.Sprintf("%x", raw[:]), nil
}

var _ op.Storage = (*MemoryStorage)(nil)
