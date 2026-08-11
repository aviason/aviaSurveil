package config

import (
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"net"
	"net/mail"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// LookupEnv and ReadFile make configuration validation deterministic in tests
// while keeping production secret reads on the process filesystem.
type LookupEnv func(string) (string, bool)
type ReadFile func(string) ([]byte, error)

const (
	EnvironmentLocalCandidate = "local-candidate"
	EnvironmentTest           = "test"
	ProfileIsolatedCandidate  = "isolated-candidate"

	defaultMaxRequestBytes   int64 = 1 << 20
	defaultMaxHeaderBytes    int   = 32 << 10
	defaultReadHeaderTimeout       = 5 * time.Second
	defaultReadTimeout             = 15 * time.Second
	defaultWriteTimeout            = 15 * time.Second
	defaultIdleTimeout             = 60 * time.Second
	minSecretBytes                 = 16
	maxSMTPPasswordBytes           = 4096
)

var keyIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{7,127}$`)

// Settings contains validated non-secret runtime settings and private secret
// material held only for the future provider initialization boundary. It has
// no String method that could accidentally serialize key or password bytes.
type Settings struct {
	Environment string
	Profile     string

	HTTPAddress string
	IssuerURL   string

	DatabaseURL    string
	DatabaseRole   string
	DatabaseSchema string

	SigningKeyID      string
	OIDCClientID      string
	OIDCRedirectURI   string
	OIDCLogoutURI     string
	MaxRequestBytes   int64
	MaxHeaderBytes    int
	ReadHeaderTimeout time.Duration
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration

	SMTPAddress  string
	SMTPFrom     string
	SMTPUsername string
	SMTPTLSMode  string

	signingKey        *rsa.PrivateKey
	dataEncryptionKey [32]byte
	mfaKey            [32]byte
	oidcClientSecret  []byte
	smtpPassword      []byte
	smtpTLSConfig     *tls.Config
}

// Load validates the isolated provider contract. It intentionally accepts
// only local candidate/test profiles until the later cutover authorization.
func Load(lookup LookupEnv, readFile ReadFile) (Settings, error) {
	if lookup == nil {
		lookup = os.LookupEnv
	}
	if readFile == nil {
		readFile = os.ReadFile
	}

	for _, name := range []string{
		"AVIA_AUTH_DATABASE_URL",
		"AVIA_AUTH_SIGNING_KEY",
		"AVIA_AUTH_DATA_ENCRYPTION_KEY",
		"AVIA_AUTH_MFA_KEY",
		"AVIA_AUTH_SMTP_PASSWORD",
	} {
		if _, present := lookup(name); present {
			return Settings{}, invalid(name, "inline secret environment variables are forbidden; use a read-only secret file")
		}
	}

	environment, err := required(lookup, "AVIA_AUTH_ENVIRONMENT")
	if err != nil {
		return Settings{}, err
	}
	if environment != EnvironmentLocalCandidate && environment != EnvironmentTest {
		return Settings{}, invalid("AVIA_AUTH_ENVIRONMENT", "only local-candidate and test are enabled before cutover")
	}
	profile, err := required(lookup, "AVIA_AUTH_PROFILE")
	if err != nil {
		return Settings{}, err
	}
	if profile != ProfileIsolatedCandidate {
		return Settings{}, invalid("AVIA_AUTH_PROFILE", "must be isolated-candidate")
	}

	address, err := required(lookup, "AVIA_AUTH_HTTP_ADDRESS")
	if err != nil {
		return Settings{}, err
	}
	if err := validateListenAddress(address); err != nil {
		return Settings{}, invalid("AVIA_AUTH_HTTP_ADDRESS", err.Error())
	}

	issuer, err := required(lookup, "AVIA_AUTH_ISSUER_URL")
	if err != nil {
		return Settings{}, err
	}
	if err := validateIssuer(environment, issuer); err != nil {
		return Settings{}, invalid("AVIA_AUTH_ISSUER_URL", err.Error())
	}

	databaseURLPath, err := required(lookup, "AVIA_AUTH_DATABASE_URL_FILE")
	if err != nil {
		return Settings{}, err
	}
	databaseURLBytes, err := readSecretFile(readFile, databaseURLPath, "AVIA_AUTH_DATABASE_URL_FILE", 1, 16<<10)
	if err != nil {
		return Settings{}, err
	}
	databaseURL := strings.TrimSpace(string(databaseURLBytes))
	databaseRole, err := required(lookup, "AVIA_AUTH_DATABASE_ROLE")
	if err != nil {
		return Settings{}, err
	}
	databaseSchema, err := required(lookup, "AVIA_AUTH_DATABASE_SCHEMA")
	if err != nil {
		return Settings{}, err
	}
	if err := validateDatabase(databaseURL, databaseRole, databaseSchema); err != nil {
		return Settings{}, invalid("AVIA_AUTH_DATABASE_URL_FILE", err.Error())
	}

	signingKeyPath, err := required(lookup, "AVIA_AUTH_SIGNING_KEY_FILE")
	if err != nil {
		return Settings{}, err
	}
	signingKeyBytes, err := readSecretFile(readFile, signingKeyPath, "AVIA_AUTH_SIGNING_KEY_FILE", minSecretBytes, 64<<10)
	if err != nil {
		return Settings{}, err
	}
	signingKey, err := parseSigningKey(signingKeyBytes)
	if err != nil {
		return Settings{}, invalid("AVIA_AUTH_SIGNING_KEY_FILE", err.Error())
	}
	signingKeyID, err := required(lookup, "AVIA_AUTH_SIGNING_KEY_ID")
	if err != nil {
		return Settings{}, err
	}
	if !keyIDPattern.MatchString(signingKeyID) || containsPlaceholder(signingKeyID) {
		return Settings{}, invalid("AVIA_AUTH_SIGNING_KEY_ID", "must be a non-placeholder stable key identifier")
	}
	algorithm, err := required(lookup, "AVIA_AUTH_SIGNING_ALGORITHM")
	if err != nil {
		return Settings{}, err
	}
	if algorithm != "RS256" {
		return Settings{}, invalid("AVIA_AUTH_SIGNING_ALGORITHM", "only RS256 is enabled")
	}

	dataKeyPath, err := required(lookup, "AVIA_AUTH_DATA_ENCRYPTION_KEY_FILE")
	if err != nil {
		return Settings{}, err
	}
	dataKeyBytes, err := readSecretFile(readFile, dataKeyPath, "AVIA_AUTH_DATA_ENCRYPTION_KEY_FILE", 32, 32)
	if err != nil {
		return Settings{}, err
	}
	dataKey, err := exactKey(dataKeyBytes)
	if err != nil {
		return Settings{}, invalid("AVIA_AUTH_DATA_ENCRYPTION_KEY_FILE", err.Error())
	}

	mfaKeyPath, err := required(lookup, "AVIA_AUTH_MFA_KEY_FILE")
	if err != nil {
		return Settings{}, err
	}
	mfaKeyBytes, err := readSecretFile(readFile, mfaKeyPath, "AVIA_AUTH_MFA_KEY_FILE", 32, 32)
	if err != nil {
		return Settings{}, err
	}
	mfaKey, err := exactKey(mfaKeyBytes)
	if err != nil {
		return Settings{}, invalid("AVIA_AUTH_MFA_KEY_FILE", err.Error())
	}
	oidcClientID, err := required(lookup, "AVIA_AUTH_OIDC_CLIENT_ID")
	if err != nil {
		return Settings{}, err
	}
	oidcRedirectURI, err := required(lookup, "AVIA_AUTH_OIDC_REDIRECT_URI")
	if err != nil {
		return Settings{}, err
	}
	oidcLogoutURI, err := required(lookup, "AVIA_AUTH_OIDC_POST_LOGOUT_REDIRECT_URI")
	if err != nil {
		return Settings{}, err
	}
	if err := validateOIDCURIs(oidcRedirectURI, oidcLogoutURI); err != nil {
		return Settings{}, invalid("AVIA_AUTH_OIDC_REDIRECT_URI", err.Error())
	}
	oidcClientSecretPath, err := required(lookup, "AVIA_AUTH_OIDC_CLIENT_SECRET_FILE")
	if err != nil {
		return Settings{}, err
	}
	oidcClientSecret, err := readSecretFile(readFile, oidcClientSecretPath, "AVIA_AUTH_OIDC_CLIENT_SECRET_FILE", minSecretBytes, maxSMTPPasswordBytes)
	if err != nil {
		return Settings{}, err
	}
	if containsPlaceholder(string(oidcClientSecret)) {
		return Settings{}, invalid("AVIA_AUTH_OIDC_CLIENT_SECRET_FILE", "placeholder secret is forbidden")
	}

	smtpAddress, err := required(lookup, "AVIA_AUTH_SMTP_ADDRESS")
	if err != nil {
		return Settings{}, err
	}
	if err := validateHostPort(smtpAddress); err != nil {
		return Settings{}, invalid("AVIA_AUTH_SMTP_ADDRESS", err.Error())
	}
	smtpFrom, err := required(lookup, "AVIA_AUTH_SMTP_FROM")
	if err != nil {
		return Settings{}, err
	}
	if err := validateMailFrom(smtpFrom); err != nil {
		return Settings{}, invalid("AVIA_AUTH_SMTP_FROM", err.Error())
	}
	smtpUsername, err := required(lookup, "AVIA_AUTH_SMTP_USERNAME")
	if err != nil {
		return Settings{}, err
	}
	smtpTLSMode, err := required(lookup, "AVIA_AUTH_SMTP_TLS_MODE")
	if err != nil {
		return Settings{}, err
	}
	if smtpTLSMode != "starttls" && smtpTLSMode != "implicit-tls" {
		return Settings{}, invalid("AVIA_AUTH_SMTP_TLS_MODE", "plain SMTP is forbidden")
	}
	smtpPasswordPath, err := required(lookup, "AVIA_AUTH_SMTP_PASSWORD_FILE")
	if err != nil {
		return Settings{}, err
	}
	smtpPassword, err := readSecretFile(readFile, smtpPasswordPath, "AVIA_AUTH_SMTP_PASSWORD_FILE", 12, maxSMTPPasswordBytes)
	if err != nil {
		return Settings{}, err
	}
	if containsPlaceholder(string(smtpPassword)) {
		return Settings{}, invalid("AVIA_AUTH_SMTP_PASSWORD_FILE", "placeholder secret is forbidden")
	}
	smtpCAPath, err := required(lookup, "AVIA_AUTH_SMTP_CA_FILE")
	if err != nil {
		return Settings{}, err
	}
	smtpCA, err := readSecretFile(readFile, smtpCAPath, "AVIA_AUTH_SMTP_CA_FILE", minSecretBytes, 64<<10)
	if err != nil {
		return Settings{}, err
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(smtpCA) {
		return Settings{}, invalid("AVIA_AUTH_SMTP_CA_FILE", "must contain at least one PEM certificate")
	}

	maxRequestBytes, err := optionalInt64(lookup, "AVIA_AUTH_MAX_REQUEST_BYTES", defaultMaxRequestBytes, 1<<10, 16<<20)
	if err != nil {
		return Settings{}, err
	}
	maxHeaderBytes, err := optionalInt(lookup, "AVIA_AUTH_MAX_HEADER_BYTES", defaultMaxHeaderBytes, 8<<10, 1<<20)
	if err != nil {
		return Settings{}, err
	}
	readHeaderTimeout, err := optionalDuration(lookup, "AVIA_AUTH_READ_HEADER_TIMEOUT", defaultReadHeaderTimeout, time.Second, time.Minute)
	if err != nil {
		return Settings{}, err
	}
	readTimeout, err := optionalDuration(lookup, "AVIA_AUTH_READ_TIMEOUT", defaultReadTimeout, time.Second, 2*time.Minute)
	if err != nil {
		return Settings{}, err
	}
	writeTimeout, err := optionalDuration(lookup, "AVIA_AUTH_WRITE_TIMEOUT", defaultWriteTimeout, time.Second, 2*time.Minute)
	if err != nil {
		return Settings{}, err
	}
	idleTimeout, err := optionalDuration(lookup, "AVIA_AUTH_IDLE_TIMEOUT", defaultIdleTimeout, time.Second, 5*time.Minute)
	if err != nil {
		return Settings{}, err
	}

	return Settings{
		Environment:       environment,
		Profile:           profile,
		HTTPAddress:       address,
		IssuerURL:         issuer,
		DatabaseURL:       databaseURL,
		DatabaseRole:      databaseRole,
		DatabaseSchema:    databaseSchema,
		SigningKeyID:      signingKeyID,
		OIDCClientID:      oidcClientID,
		OIDCRedirectURI:   oidcRedirectURI,
		OIDCLogoutURI:     oidcLogoutURI,
		MaxRequestBytes:   maxRequestBytes,
		MaxHeaderBytes:    maxHeaderBytes,
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
		SMTPAddress:       smtpAddress,
		SMTPFrom:          smtpFrom,
		SMTPUsername:      smtpUsername,
		SMTPTLSMode:       smtpTLSMode,
		signingKey:        signingKey,
		dataEncryptionKey: dataKey,
		mfaKey:            mfaKey,
		oidcClientSecret:  append([]byte(nil), oidcClientSecret...),
		smtpPassword:      smtpPassword,
		smtpTLSConfig:     &tls.Config{RootCAs: pool, MinVersion: tls.VersionTLS12},
	}, nil
}

// SigningKey returns a private-key copy for the isolated provider bootstrap.
func (settings Settings) SigningKey() *rsa.PrivateKey {
	if settings.signingKey == nil {
		return nil
	}
	copyKey, err := x509.ParsePKCS1PrivateKey(x509.MarshalPKCS1PrivateKey(settings.signingKey))
	if err != nil {
		return nil
	}
	return copyKey
}

func (settings Settings) DataEncryptionKey() [32]byte { return settings.dataEncryptionKey }
func (settings Settings) MFAKey() [32]byte            { return settings.mfaKey }
func (settings Settings) OIDCClientSecret() string {
	return strings.TrimSpace(string(settings.oidcClientSecret))
}
func (settings Settings) SMTPPassword() string {
	return strings.TrimSpace(string(settings.smtpPassword))
}

func (settings Settings) SMTPTLSConfig() *tls.Config {
	if settings.smtpTLSConfig == nil {
		return nil
	}
	return settings.smtpTLSConfig.Clone()
}

func invalid(field, reason string) error {
	return fmt.Errorf("%s: %s", field, reason)
}

func required(lookup LookupEnv, name string) (string, error) {
	value, ok := lookup(name)
	value = strings.TrimSpace(value)
	if !ok || value == "" {
		return "", invalid(name, "required")
	}
	if containsPlaceholder(value) {
		return "", invalid(name, "placeholder value is forbidden")
	}
	return value, nil
}

func containsPlaceholder(value string) bool {
	normalized := strings.ToLower(strings.TrimSpace(value))
	for _, marker := range []string{"changeme", "change-me", "replace-me", "placeholder", "dummy", "example", "your-", "your_", "<secret>", "todo"} {
		if strings.Contains(normalized, marker) {
			return true
		}
	}
	return normalized == "secret" || normalized == "password" || normalized == "test-secret"
}

func readSecretFile(readFile ReadFile, path, field string, minSize, maxSize int) ([]byte, error) {
	path = strings.TrimSpace(path)
	if path == "" || !filepath.IsAbs(path) {
		return nil, invalid(field, "must be an absolute read-only secret-file path")
	}
	if strings.HasPrefix(filepath.Clean(path), "/dev/") {
		return nil, invalid(field, "device and standard-input paths are forbidden")
	}
	info, err := os.Stat(path)
	if err != nil {
		return nil, invalid(field, "secret file is unavailable")
	}
	if !info.Mode().IsRegular() || info.Mode().Perm()&0o222 != 0 {
		return nil, invalid(field, "secret file must be a regular non-writable file")
	}
	contents, err := readFile(path)
	if err != nil {
		return nil, invalid(field, "secret file cannot be read")
	}
	if len(contents) < minSize || len(contents) > maxSize {
		return nil, invalid(field, "secret file has an invalid bounded size")
	}
	return contents, nil
}

func validateListenAddress(address string) error {
	host, port, err := net.SplitHostPort(address)
	if err != nil || strings.TrimSpace(host) == "" {
		return errors.New("must be host:port")
	}
	portNumber, err := strconv.Atoi(port)
	if err != nil || portNumber < 1024 || portNumber > 65535 {
		return errors.New("port must be between 1024 and 65535")
	}
	return nil
}

func validateIssuer(environment, issuer string) error {
	parsed, err := url.Parse(issuer)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return errors.New("must be an absolute issuer URL without credentials, query, or fragment")
	}
	if environment == EnvironmentLocalCandidate || environment == EnvironmentTest {
		if parsed.Scheme != "https" && !isLoopbackHost(parsed.Hostname()) {
			return errors.New("non-HTTPS issuer is allowed only on loopback in local/test profiles")
		}
	} else if parsed.Scheme != "https" {
		return errors.New("issuer must use HTTPS")
	}
	return nil
}

func validateOIDCURIs(redirectURI, logoutURI string) error {
	for _, raw := range []string{redirectURI, logoutURI} {
		parsed, err := url.Parse(raw)
		if err != nil || !parsed.IsAbs() || parsed.User != nil || parsed.Fragment != "" || parsed.RawQuery != "" || (parsed.Scheme != "https" && parsed.Scheme != "http") {
			return errors.New("OIDC redirect URIs must be absolute HTTP(S) URLs without credentials, query, or fragment")
		}
	}
	return nil
}

func isLoopbackHost(host string) bool {
	return host == "localhost" || host == "127.0.0.1" || host == "::1"
}

func validateDatabase(rawURL, role, schema string) error {
	if containsPlaceholder(role) || containsPlaceholder(schema) {
		return errors.New("database role/schema cannot be placeholders")
	}
	if role == "postgres" || role == "root" || role == "keycloak" || role == "aviasurveil360" || schema == "public" || schema == "keycloak" || schema == "aviasurveil360" {
		return errors.New("database role and schema must be separate from application and Keycloak ownership")
	}
	parsed, err := url.Parse(rawURL)
	if err != nil || (parsed.Scheme != "postgres" && parsed.Scheme != "postgresql") || parsed.Host == "" || parsed.Path == "" || parsed.User == nil {
		return errors.New("must be a complete PostgreSQL URL")
	}
	username := parsed.User.Username()
	password, hasPassword := parsed.User.Password()
	if !hasPassword || username != role || len(password) < 12 || containsPlaceholder(password) {
		return errors.New("database URL must contain the dedicated role and a non-placeholder password")
	}
	if parsed.Query().Get("sslmode") == "" {
		return errors.New("database URL must declare sslmode explicitly")
	}
	return nil
}

func parseSigningKey(contents []byte) (*rsa.PrivateKey, error) {
	block, rest := pem.Decode(contents)
	if block == nil || len(strings.TrimSpace(string(rest))) != 0 {
		return nil, errors.New("must contain one PEM private key")
	}
	var parsed any
	var err error
	switch block.Type {
	case "RSA PRIVATE KEY":
		parsed, err = x509.ParsePKCS1PrivateKey(block.Bytes)
	case "PRIVATE KEY":
		parsed, err = x509.ParsePKCS8PrivateKey(block.Bytes)
	default:
		return nil, errors.New("must be an RSA private key")
	}
	if err != nil {
		return nil, errors.New("private key cannot be parsed")
	}
	key, ok := parsed.(*rsa.PrivateKey)
	if !ok || key.N == nil || key.N.BitLen() < 2048 || key.N.BitLen() > 8192 {
		return nil, errors.New("RSA signing key must be between 2048 and 8192 bits")
	}
	if err := key.Validate(); err != nil {
		return nil, errors.New("RSA signing key failed validation")
	}
	return key, nil
}

func exactKey(contents []byte) ([32]byte, error) {
	var key [32]byte
	if len(contents) != len(key) {
		return key, errors.New("must be exactly 32 non-zero bytes")
	}
	copy(key[:], contents)
	if allZero(key[:]) {
		return [32]byte{}, errors.New("must not be all zero")
	}
	return key, nil
}

func allZero(value []byte) bool {
	for _, byteValue := range value {
		if byteValue != 0 {
			return false
		}
	}
	return true
}

func validateHostPort(value string) error {
	host, port, err := net.SplitHostPort(value)
	if err != nil || strings.TrimSpace(host) == "" {
		return errors.New("must be host:port")
	}
	portNumber, err := strconv.Atoi(port)
	if err != nil || portNumber < 1 || portNumber > 65535 {
		return errors.New("port must be between 1 and 65535")
	}
	return nil
}

func validateMailFrom(value string) error {
	parsed, err := mail.ParseAddress(value)
	if err != nil || parsed.Address != value || !strings.Contains(parsed.Address, "@") {
		return errors.New("must be one plain RFC 5322 mailbox address")
	}
	return nil
}

func optionalInt64(lookup LookupEnv, name string, fallback, minValue, maxValue int64) (int64, error) {
	value, ok := lookup(name)
	if !ok || strings.TrimSpace(value) == "" {
		return fallback, nil
	}
	parsed, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
	if err != nil || parsed < minValue || parsed > maxValue {
		return 0, invalid(name, "must be a bounded integer")
	}
	return parsed, nil
}

func optionalInt(lookup LookupEnv, name string, fallback, minValue, maxValue int) (int, error) {
	value, ok := lookup(name)
	if !ok || strings.TrimSpace(value) == "" {
		return fallback, nil
	}
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || parsed < minValue || parsed > maxValue {
		return 0, invalid(name, "must be a bounded integer")
	}
	return parsed, nil
}

func optionalDuration(lookup LookupEnv, name string, fallback, minValue, maxValue time.Duration) (time.Duration, error) {
	value, ok := lookup(name)
	if !ok || strings.TrimSpace(value) == "" {
		return fallback, nil
	}
	parsed, err := time.ParseDuration(strings.TrimSpace(value))
	if err != nil || parsed < minValue || parsed > maxValue {
		return 0, invalid(name, "must be a bounded duration")
	}
	return parsed, nil
}
