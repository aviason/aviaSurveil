package config

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLoadAcceptsCompleteIsolatedCandidateConfiguration(t *testing.T) {
	values := validEnvironment(t)
	settings, err := Load(mapLookup(values), os.ReadFile)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if settings.Environment != EnvironmentLocalCandidate || settings.Profile != ProfileIsolatedCandidate {
		t.Fatalf("profile = %q/%q", settings.Environment, settings.Profile)
	}
	if settings.DatabaseRole != "auth_owner" || settings.DatabaseSchema != "auth_identity" {
		t.Fatalf("database isolation = %q/%q", settings.DatabaseRole, settings.DatabaseSchema)
	}
	if settings.signingKey == nil || settings.signingKey.N.BitLen() != 2048 {
		t.Fatalf("signing key was not loaded as a validated RSA key")
	}
	if settings.dataEncryptionKey == settings.mfaKey {
		t.Fatal("data-encryption and MFA keys must remain distinct")
	}
	if settings.MaxRequestBytes != defaultMaxRequestBytes || settings.MaxHeaderBytes != defaultMaxHeaderBytes {
		t.Fatalf("unexpected defaults: request=%d header=%d", settings.MaxRequestBytes, settings.MaxHeaderBytes)
	}
}

func TestLoadRejectsMissingSecretAndDoesNotEchoIt(t *testing.T) {
	values := validEnvironment(t)
	delete(values, "AVIA_AUTH_MFA_KEY_FILE")
	_, err := Load(mapLookup(values), os.ReadFile)
	if err == nil {
		t.Fatal("Load() accepted missing MFA key")
	}
	if strings.Contains(err.Error(), "smtp-credential-2026") || strings.Contains(err.Error(), "long-credential-2026") {
		t.Fatalf("configuration error echoed secret material: %v", err)
	}
}

func TestLoadRejectsInlineSecretsAndSharedDatabaseOwnership(t *testing.T) {
	values := validEnvironment(t)
	values["AVIA_AUTH_DATABASE_URL"] = "postgresql://auth_owner:long-credential-2026@127.0.0.1:5432/auth?sslmode=disable"
	if _, err := Load(mapLookup(values), os.ReadFile); err == nil {
		t.Fatal("Load() accepted inline database secret")
	}

	values = validEnvironment(t)
	values["AVIA_AUTH_DATABASE_ROLE"] = "aviasurveil360"
	if _, err := Load(mapLookup(values), os.ReadFile); err == nil {
		t.Fatal("Load() accepted shared application database role")
	}
}

func TestLoadRejectsWeakSigningKeyAndPlainSMTP(t *testing.T) {
	values := validEnvironment(t)
	weakKey, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		t.Fatal(err)
	}
	weakBytes := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(weakKey)})
	if err := os.Chmod(values["AVIA_AUTH_SIGNING_KEY_FILE"], 0o600); err != nil {
		t.Fatal(err)
	}
	writeSecret(t, values["AVIA_AUTH_SIGNING_KEY_FILE"], weakBytes, 0o400)
	if _, err := Load(mapLookup(values), os.ReadFile); err == nil {
		t.Fatal("Load() accepted a 1024-bit signing key")
	}

	values = validEnvironment(t)
	values["AVIA_AUTH_SMTP_TLS_MODE"] = "plain"
	if _, err := Load(mapLookup(values), os.ReadFile); err == nil {
		t.Fatal("Load() accepted plaintext SMTP")
	}
}

func TestLoadRejectsNormalOrProductionProfile(t *testing.T) {
	values := validEnvironment(t)
	values["AVIA_AUTH_PROFILE"] = "normal"
	if _, err := Load(mapLookup(values), os.ReadFile); err == nil {
		t.Fatal("Load() accepted normal profile routing")
	}

	values = validEnvironment(t)
	values["AVIA_AUTH_ENVIRONMENT"] = "production"
	if _, err := Load(mapLookup(values), os.ReadFile); err == nil {
		t.Fatal("Load() accepted production before cutover authorization")
	}
}

func TestLoadRejectsWritableSecretAndUnboundedResourceValues(t *testing.T) {
	values := validEnvironment(t)
	values["AVIA_AUTH_MAX_REQUEST_BYTES"] = "0"
	if _, err := Load(mapLookup(values), os.ReadFile); err == nil {
		t.Fatal("Load() accepted an unbounded request limit")
	}

	values = validEnvironment(t)
	if err := os.Chmod(values["AVIA_AUTH_MFA_KEY_FILE"], 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(mapLookup(values), os.ReadFile); err == nil {
		t.Fatal("Load() accepted a writable secret file")
	}
}

func validEnvironment(t *testing.T) map[string]string {
	t.Helper()
	directory := t.TempDir()
	signingKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	signingKeyBytes := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(signingKey)})
	databasePath := filepath.Join(directory, "database-url")
	signingPath := filepath.Join(directory, "signing-key.pem")
	dataKeyPath := filepath.Join(directory, "data-key")
	mfaKeyPath := filepath.Join(directory, "mfa-key")
	smtpPath := filepath.Join(directory, "smtp-password")
	oidcSecretPath := filepath.Join(directory, "oidc-client-secret")
	caPath := filepath.Join(directory, "smtp-ca.pem")
	certificate, err := certificateForTest(signingKey)
	if err != nil {
		t.Fatal(err)
	}
	writeSecret(t, databasePath, []byte("postgresql://auth_owner:long-credential-2026@127.0.0.1:5432/auth?sslmode=disable\n"), 0o400)
	writeSecret(t, signingPath, signingKeyBytes, 0o400)
	writeSecret(t, dataKeyPath, bytes.Repeat([]byte{0x42}, 32), 0o400)
	writeSecret(t, mfaKeyPath, bytes.Repeat([]byte{0x43}, 32), 0o400)
	writeSecret(t, smtpPath, []byte("smtp-credential-2026\n"), 0o400)
	writeSecret(t, oidcSecretPath, []byte("oidc-client-credential-2026\n"), 0o400)
	writeSecret(t, caPath, certificate, 0o400)
	return map[string]string{
		"AVIA_AUTH_ENVIRONMENT":                   EnvironmentLocalCandidate,
		"AVIA_AUTH_PROFILE":                       ProfileIsolatedCandidate,
		"AVIA_AUTH_HTTP_ADDRESS":                  "127.0.0.1:18081",
		"AVIA_AUTH_ISSUER_URL":                    "http://127.0.0.1:18081/",
		"AVIA_AUTH_DATABASE_URL_FILE":             databasePath,
		"AVIA_AUTH_DATABASE_ROLE":                 "auth_owner",
		"AVIA_AUTH_DATABASE_SCHEMA":               "auth_identity",
		"AVIA_AUTH_SIGNING_KEY_FILE":              signingPath,
		"AVIA_AUTH_SIGNING_KEY_ID":                "as360-auth-2026-08-11",
		"AVIA_AUTH_SIGNING_ALGORITHM":             "RS256",
		"AVIA_AUTH_DATA_ENCRYPTION_KEY_FILE":      dataKeyPath,
		"AVIA_AUTH_MFA_KEY_FILE":                  mfaKeyPath,
		"AVIA_AUTH_OIDC_CLIENT_ID":                "as360-local-candidate-web",
		"AVIA_AUTH_OIDC_REDIRECT_URI":             "http://127.0.0.1:18082/callback",
		"AVIA_AUTH_OIDC_POST_LOGOUT_REDIRECT_URI": "http://127.0.0.1:18082/logout",
		"AVIA_AUTH_OIDC_CLIENT_SECRET_FILE":       oidcSecretPath,
		"AVIA_AUTH_SMTP_ADDRESS":                  "mailpit:1025",
		"AVIA_AUTH_SMTP_FROM":                     "identity@aviasurveil360.local",
		"AVIA_AUTH_SMTP_USERNAME":                 "aviasurveil360",
		"AVIA_AUTH_SMTP_TLS_MODE":                 "starttls",
		"AVIA_AUTH_SMTP_PASSWORD_FILE":            smtpPath,
		"AVIA_AUTH_SMTP_CA_FILE":                  caPath,
	}
}

func certificateForTest(key *rsa.PrivateKey) ([]byte, error) {
	template := &x509.Certificate{SerialNumber: new(big.Int).SetInt64(1), NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour), KeyUsage: x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}, DNSNames: []string{"mailpit"}}
	encoded, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		return nil, err
	}
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: encoded}), nil
}

func writeSecret(t *testing.T, path string, contents []byte, mode os.FileMode) {
	t.Helper()
	if err := os.WriteFile(path, contents, mode); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, mode); err != nil {
		t.Fatal(err)
	}
}

func mapLookup(values map[string]string) LookupEnv {
	return func(name string) (string, bool) {
		value, ok := values[name]
		return value, ok
	}
}
