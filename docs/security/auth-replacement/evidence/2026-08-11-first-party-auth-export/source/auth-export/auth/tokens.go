package auth

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"os"
	"strings"
	"time"
)

const defaultAccessTokenTTL = 15 * time.Minute

type AccessToken struct {
	Token     string
	ExpiresAt time.Time
}

type TokenServiceConfig struct {
	AppEnv              string
	IssuerURL           string
	PrivateKeyPEM       string
	PrivateKeyFile      string
	KeyID               string
	AccessTokenAudience []string
	AppSyncClientID     string
	AllowedClientIDs    []string
	PreviousPublicPEM   string
	PreviousPublicFile  string
}

type AccessTokenInput struct {
	Subject     string
	SessionID   string
	ClientID    string
	Role        string
	Scope       string
	AuthTime    time.Time
	AuthVersion int64
}

type signingKey struct {
	kid     string
	private *rsa.PrivateKey
	public  *rsa.PublicKey
}

type verificationKey struct {
	kid    string
	public *rsa.PublicKey
}

type TokenService struct {
	issuer             string
	audience           []string
	audienceSet        map[string]bool
	allowedClientIDs   map[string]bool
	active             signingKey
	verificationKeys   map[string]verificationKey
	verificationKeyIDs []string
	now                func() time.Time
}

func NewTokenService(cfg TokenServiceConfig) (*TokenService, error) {
	appEnv := strings.TrimSpace(cfg.AppEnv)
	if appEnv == "" {
		appEnv = "development"
	}
	issuer := strings.TrimRight(strings.TrimSpace(cfg.IssuerURL), "/")
	if issuer == "" {
		issuer = "http://localhost:8080"
	}

	privatePEM := strings.TrimSpace(cfg.PrivateKeyPEM)
	if privatePEM == "" && strings.TrimSpace(cfg.PrivateKeyFile) != "" {
		data, err := os.ReadFile(strings.TrimSpace(cfg.PrivateKeyFile))
		if err != nil {
			return nil, err
		}
		privatePEM = string(data)
	}

	var privateKey *rsa.PrivateKey
	var err error
	if privatePEM != "" {
		privateKey, err = parseRSAPrivateKeyPEM(privatePEM)
		if err != nil {
			return nil, err
		}
	} else if appEnv == "development" {
		privateKey, err = rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			return nil, err
		}
	} else {
		return nil, ErrAuthUnavailable
	}

	kid := strings.TrimSpace(cfg.KeyID)
	if kid == "" {
		kid = publicKeyID(&privateKey.PublicKey)
	}
	audience := normalizedList(cfg.AccessTokenAudience)
	if len(audience) == 0 && strings.TrimSpace(cfg.AppSyncClientID) != "" {
		audience = []string{strings.TrimSpace(cfg.AppSyncClientID)}
	}
	if len(audience) == 0 {
		audience = []string{"emsi-api"}
	}
	allowedClientIDs := normalizedList(cfg.AllowedClientIDs)
	if strings.TrimSpace(cfg.AppSyncClientID) != "" {
		allowedClientIDs = append(allowedClientIDs, strings.TrimSpace(cfg.AppSyncClientID))
	}
	if len(allowedClientIDs) == 0 {
		allowedClientIDs = []string{"emsi.ios", "emsi.macos"}
	}

	service := &TokenService{
		issuer:           issuer,
		audience:         audience,
		audienceSet:      stringSet(audience),
		allowedClientIDs: stringSet(allowedClientIDs),
		active: signingKey{
			kid:     kid,
			private: privateKey,
			public:  &privateKey.PublicKey,
		},
		verificationKeys: map[string]verificationKey{},
		now:              func() time.Time { return time.Now().UTC() },
	}
	service.addVerificationKey(kid, &privateKey.PublicKey)

	previousPEM := strings.TrimSpace(cfg.PreviousPublicPEM)
	if previousPEM == "" && strings.TrimSpace(cfg.PreviousPublicFile) != "" {
		data, err := os.ReadFile(strings.TrimSpace(cfg.PreviousPublicFile))
		if err != nil {
			return nil, err
		}
		previousPEM = string(data)
	}
	if previousPEM != "" {
		publicKeys, err := parseRSAPublicKeysPEM(previousPEM)
		if err != nil {
			return nil, err
		}
		for _, publicKey := range publicKeys {
			service.addVerificationKey(publicKeyID(publicKey), publicKey)
		}
	}

	return service, nil
}

func (s *TokenService) IssueAccessToken(input AccessTokenInput, ttl time.Duration, now time.Time) (AccessToken, error) {
	if s == nil || s.active.private == nil {
		return AccessToken{}, ErrAuthUnavailable
	}
	input.Subject = strings.TrimSpace(input.Subject)
	input.SessionID = strings.TrimSpace(input.SessionID)
	input.ClientID = strings.TrimSpace(input.ClientID)
	if input.Subject == "" || input.SessionID == "" || input.ClientID == "" || input.AuthVersion <= 0 {
		return AccessToken{}, ErrUnauthorized
	}
	if !s.allowedClientIDs[input.ClientID] {
		return AccessToken{}, ErrUnauthorized
	}
	if ttl <= 0 {
		ttl = defaultAccessTokenTTL
	}
	if now.IsZero() {
		now = s.now()
	}
	now = now.UTC()
	authTime := input.AuthTime.UTC()
	if authTime.IsZero() {
		authTime = now
	}
	expiresAt := now.Add(ttl)
	tokenID, err := randomToken()
	if err != nil {
		return AccessToken{}, err
	}
	if strings.TrimSpace(input.Scope) == "" {
		input.Scope = "openid profile offline_access"
	}
	header := map[string]string{
		"alg": "RS256",
		"kid": s.active.kid,
		"typ": "JWT",
	}
	claims := map[string]any{
		"iss":          s.issuer,
		"sub":          input.Subject,
		"aud":          s.audience,
		"azp":          input.ClientID,
		"exp":          expiresAt.Unix(),
		"iat":          now.Unix(),
		"auth_time":    authTime.Unix(),
		"jti":          tokenID,
		"sid":          input.SessionID,
		"scope":        strings.TrimSpace(input.Scope),
		"client_id":    input.ClientID,
		"auth_version": input.AuthVersion,
	}
	if role := strings.TrimSpace(input.Role); role != "" {
		claims["role"] = role
	}
	token, err := s.signJWT(header, claims)
	if err != nil {
		return AccessToken{}, err
	}
	return AccessToken{Token: token, ExpiresAt: expiresAt}, nil
}

func (s *TokenService) VerifyAccessToken(token string, now time.Time) (User, error) {
	if s == nil {
		return User{}, ErrUnauthorized
	}
	if now.IsZero() {
		now = s.now()
	}
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return User{}, ErrUnauthorized
	}
	headerJSON, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return User{}, ErrUnauthorized
	}
	var header map[string]any
	if err := json.Unmarshal(headerJSON, &header); err != nil {
		return User{}, ErrUnauthorized
	}
	if stringClaim(header, "alg") != "RS256" {
		return User{}, ErrUnauthorized
	}
	kid := stringClaim(header, "kid")
	if kid == "" {
		return User{}, ErrUnauthorized
	}
	key, ok := s.verificationKeys[kid]
	if !ok || key.public == nil {
		return User{}, ErrUnauthorized
	}
	unsigned := parts[0] + "." + parts[1]
	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return User{}, ErrUnauthorized
	}
	digest := sha256.Sum256([]byte(unsigned))
	if err := rsa.VerifyPKCS1v15(key.public, crypto.SHA256, digest[:], signature); err != nil {
		return User{}, ErrUnauthorized
	}

	payloadJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return User{}, ErrUnauthorized
	}
	var claims map[string]any
	decoder := json.NewDecoder(strings.NewReader(string(payloadJSON)))
	decoder.UseNumber()
	if err := decoder.Decode(&claims); err != nil {
		return User{}, ErrUnauthorized
	}
	return s.userFromClaims(claims, now.UTC())
}

func (s *TokenService) OpenIDConfiguration() map[string]any {
	return map[string]any{
		"issuer":                                s.issuer,
		"jwks_uri":                              s.issuer + "/oauth2/jwks",
		"response_types_supported":              []string{"token"},
		"subject_types_supported":               []string{"public"},
		"id_token_signing_alg_values_supported": []string{"RS256"},
		"scopes_supported":                      []string{"openid", "profile", "offline_access"},
		"claims_supported": []string{
			"iss", "sub", "aud", "azp", "exp", "iat", "auth_time",
			"jti", "sid", "scope", "client_id", "auth_version",
		},
	}
}

func (s *TokenService) JWKS() map[string]any {
	keys := make([]map[string]any, 0, len(s.verificationKeyIDs))
	for _, kid := range s.verificationKeyIDs {
		key := s.verificationKeys[kid]
		keys = append(keys, jwkFromRSA(key.kid, key.public))
	}
	return map[string]any{"keys": keys}
}

func (s *TokenService) Issuer() string {
	if s == nil {
		return ""
	}
	return s.issuer
}

func (s *TokenService) userFromClaims(claims map[string]any, now time.Time) (User, error) {
	if stringClaim(claims, "iss") != s.issuer {
		return User{}, ErrUnauthorized
	}
	if !audienceAllowed(stringListClaim(claims, "aud"), s.audienceSet) {
		return User{}, ErrUnauthorized
	}
	subject := strings.TrimSpace(stringClaim(claims, "sub"))
	sessionID := strings.TrimSpace(stringClaim(claims, "sid"))
	clientID := strings.TrimSpace(stringClaim(claims, "client_id"))
	if clientID == "" {
		clientID = strings.TrimSpace(stringClaim(claims, "azp"))
	}
	if subject == "" || sessionID == "" || clientID == "" || !s.allowedClientIDs[clientID] {
		return User{}, ErrUnauthorized
	}
	if azp := strings.TrimSpace(stringClaim(claims, "azp")); azp != "" && azp != clientID {
		return User{}, ErrUnauthorized
	}
	expiresAt, ok := int64Claim(claims["exp"])
	if !ok || now.Unix() >= expiresAt {
		return User{}, ErrUnauthorized
	}
	issuedAt, ok := int64Claim(claims["iat"])
	if !ok || issuedAt <= 0 || issuedAt > now.Add(time.Minute).Unix() {
		return User{}, ErrUnauthorized
	}
	authTimeUnix, ok := int64Claim(claims["auth_time"])
	if !ok || authTimeUnix <= 0 || authTimeUnix > now.Add(time.Minute).Unix() {
		return User{}, ErrUnauthorized
	}
	authVersion, ok := int64Claim(claims["auth_version"])
	if !ok || authVersion <= 0 {
		return User{}, ErrUnauthorized
	}
	tokenID := strings.TrimSpace(stringClaim(claims, "jti"))
	if tokenID == "" {
		return User{}, ErrUnauthorized
	}
	role := strings.TrimSpace(stringClaim(claims, "role"))
	return User{
		Role:        role,
		Subject:     subject,
		SessionID:   sessionID,
		ClientID:    clientID,
		TokenID:     tokenID,
		AuthTime:    time.Unix(authTimeUnix, 0).UTC(),
		AuthVersion: authVersion,
	}, nil
}

func (s *TokenService) signJWT(header map[string]string, claims map[string]any) (string, error) {
	headerJSON, err := json.Marshal(header)
	if err != nil {
		return "", err
	}
	claimsJSON, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	encodedHeader := base64.RawURLEncoding.EncodeToString(headerJSON)
	encodedClaims := base64.RawURLEncoding.EncodeToString(claimsJSON)
	unsigned := encodedHeader + "." + encodedClaims
	digest := sha256.Sum256([]byte(unsigned))
	signature, err := rsa.SignPKCS1v15(rand.Reader, s.active.private, crypto.SHA256, digest[:])
	if err != nil {
		return "", err
	}
	return unsigned + "." + base64.RawURLEncoding.EncodeToString(signature), nil
}

func (s *TokenService) addVerificationKey(kid string, publicKey *rsa.PublicKey) {
	kid = strings.TrimSpace(kid)
	if kid == "" || publicKey == nil {
		return
	}
	if _, exists := s.verificationKeys[kid]; !exists {
		s.verificationKeyIDs = append(s.verificationKeyIDs, kid)
	}
	s.verificationKeys[kid] = verificationKey{kid: kid, public: publicKey}
}

func parseRSAPrivateKeyPEM(value string) (*rsa.PrivateKey, error) {
	for {
		block, rest := pem.Decode([]byte(value))
		if block == nil {
			break
		}
		value = string(rest)
		if block.Type == "RSA PRIVATE KEY" {
			return x509.ParsePKCS1PrivateKey(block.Bytes)
		}
		if block.Type == "PRIVATE KEY" {
			key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
			if err != nil {
				return nil, err
			}
			rsaKey, ok := key.(*rsa.PrivateKey)
			if !ok {
				return nil, errors.New("private key is not RSA")
			}
			return rsaKey, nil
		}
	}
	return nil, errors.New("RSA private key PEM not found")
}

func parseRSAPublicKeysPEM(value string) ([]*rsa.PublicKey, error) {
	var keys []*rsa.PublicKey
	for {
		block, rest := pem.Decode([]byte(value))
		if block == nil {
			break
		}
		value = string(rest)
		var public any
		var err error
		switch block.Type {
		case "PUBLIC KEY":
			public, err = x509.ParsePKIXPublicKey(block.Bytes)
		case "RSA PUBLIC KEY":
			public, err = x509.ParsePKCS1PublicKey(block.Bytes)
		case "CERTIFICATE":
			var cert *x509.Certificate
			cert, err = x509.ParseCertificate(block.Bytes)
			if err == nil {
				public = cert.PublicKey
			}
		default:
			continue
		}
		if err != nil {
			return nil, err
		}
		rsaKey, ok := public.(*rsa.PublicKey)
		if !ok {
			return nil, errors.New("public key is not RSA")
		}
		keys = append(keys, rsaKey)
	}
	if len(keys) == 0 {
		return nil, errors.New("RSA public key PEM not found")
	}
	return keys, nil
}

func publicKeyID(publicKey *rsa.PublicKey) string {
	if publicKey == nil {
		return ""
	}
	sum := sha256.Sum256([]byte(publicKey.N.String() + ":" + fmt.Sprint(publicKey.E)))
	return "rsa-" + base64.RawURLEncoding.EncodeToString(sum[:12])
}

func jwkFromRSA(kid string, publicKey *rsa.PublicKey) map[string]any {
	exponent := big.NewInt(int64(publicKey.E)).Bytes()
	return map[string]any{
		"kty": "RSA",
		"use": "sig",
		"alg": "RS256",
		"kid": kid,
		"n":   base64.RawURLEncoding.EncodeToString(publicKey.N.Bytes()),
		"e":   base64.RawURLEncoding.EncodeToString(exponent),
	}
}

func normalizedList(values []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, raw := range values {
		for _, part := range strings.Split(raw, ",") {
			value := strings.TrimSpace(part)
			if value == "" || seen[value] {
				continue
			}
			seen[value] = true
			out = append(out, value)
		}
	}
	return out
}

func stringSet(values []string) map[string]bool {
	set := map[string]bool{}
	for _, value := range normalizedList(values) {
		set[value] = true
	}
	return set
}

func stringClaim(claims map[string]any, key string) string {
	value, _ := claims[key].(string)
	return value
}

func stringListClaim(claims map[string]any, key string) []string {
	value, ok := claims[key]
	if !ok {
		return nil
	}
	switch typed := value.(type) {
	case string:
		if strings.TrimSpace(typed) == "" {
			return nil
		}
		return []string{strings.TrimSpace(typed)}
	case []any:
		values := make([]string, 0, len(typed))
		for _, item := range typed {
			if text, ok := item.(string); ok && strings.TrimSpace(text) != "" {
				values = append(values, strings.TrimSpace(text))
			}
		}
		return values
	default:
		return nil
	}
}

func audienceAllowed(values []string, allowed map[string]bool) bool {
	for _, value := range values {
		if allowed[value] {
			return true
		}
	}
	return false
}
