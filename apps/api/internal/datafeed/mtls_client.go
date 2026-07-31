package datafeed

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

const aviaCoreV3MediaType = "application/vnd.aviacore.aviasurveil-events.v3+json"

// MTLSClientConfig carries only mounted-file references and approved public
// trust metadata. It deliberately has no plaintext or proxy certificate mode.
type MTLSClientConfig struct {
	Endpoint               string
	CABundleFile           string
	RevocationListFile     string
	ApprovedCABundleSHA256 string
	ClientCertificateFile  string
	ClientPrivateKeyFile   string
	ExpectedClientSAN      string
}

// MTLSClient is the Task 5 direct-mTLS transport boundary. Its later request
// methods stay separate from the immutable event/outbox persistence layer.
type MTLSClient struct {
	endpoint url.URL
	config   MTLSClientConfig
	http     *http.Client
}

// BatchRequest is intentionally closed to the three protocol headers. A
// publisher cannot inject forwarded certificate or proxy-derived headers.
type BatchRequest struct {
	RequestID      string
	IdempotencyKey string
	ReplayID       string
	Body           []byte
}

// NewMTLSClient validates every mounted trust artifact before a worker can
// claim an event. It does not use environment proxy configuration, TLS below
// 1.3, or a caller-supplied arbitrary destination.
func NewMTLSClient(config MTLSClientConfig) (*MTLSClient, error) {
	endpoint, err := url.Parse(strings.TrimSpace(config.Endpoint))
	if err != nil || endpoint.Scheme != "https" || endpoint.Host == "" || endpoint.User != nil {
		return nil, fmt.Errorf("datafeed publisher requires an approved https endpoint")
	}
	if endpoint.Path != "/v3/aviasurveil/event-batches" || endpoint.RawQuery != "" || endpoint.Fragment != "" {
		return nil, fmt.Errorf("datafeed publisher endpoint path is not the locked v3 event API")
	}
	if !isSHA256(config.ApprovedCABundleSHA256) {
		return nil, fmt.Errorf("datafeed publisher requires an approved CA bundle SHA-256")
	}
	if strings.TrimSpace(config.ClientCertificateFile) == "" || strings.TrimSpace(config.ClientPrivateKeyFile) == "" {
		return nil, fmt.Errorf("datafeed publisher requires mounted client certificate and private key files")
	}
	if strings.TrimSpace(config.ExpectedClientSAN) == "" {
		return nil, fmt.Errorf("datafeed publisher requires an approved client SAN mapping")
	}
	if strings.TrimSpace(config.CABundleFile) == "" {
		return nil, fmt.Errorf("datafeed publisher requires a mounted approved CA bundle")
	}
	if strings.TrimSpace(config.RevocationListFile) == "" {
		return nil, fmt.Errorf("datafeed publisher requires a mounted approved revocation list")
	}
	caBundle, err := os.ReadFile(config.CABundleFile)
	if err != nil {
		return nil, fmt.Errorf("read datafeed approved CA bundle: %w", err)
	}
	actualDigest := sha256.Sum256(caBundle)
	if hex.EncodeToString(actualDigest[:]) != config.ApprovedCABundleSHA256 {
		return nil, fmt.Errorf("datafeed publisher CA bundle fingerprint does not match its approved value")
	}
	rootCAs, trustedCAs, err := parseApprovedCABundle(caBundle)
	if err != nil {
		return nil, err
	}
	clientCertificate, err := tls.LoadX509KeyPair(config.ClientCertificateFile, config.ClientPrivateKeyFile)
	if err != nil {
		return nil, fmt.Errorf("load datafeed mounted client certificate: %w", err)
	}
	if len(clientCertificate.Certificate) == 0 {
		return nil, fmt.Errorf("datafeed publisher client certificate chain is empty")
	}
	leaf, err := x509.ParseCertificate(clientCertificate.Certificate[0])
	if err != nil {
		return nil, fmt.Errorf("parse datafeed client certificate leaf: %w", err)
	}
	now := time.Now().UTC()
	if now.Before(leaf.NotBefore) || !now.Before(leaf.NotAfter) {
		return nil, fmt.Errorf("datafeed publisher client certificate is not currently valid")
	}
	if !certificateHasExpectedSAN(leaf, config.ExpectedClientSAN) {
		return nil, fmt.Errorf("datafeed publisher client certificate SAN does not match the approved source/tenant mapping")
	}
	if _, err := leaf.Verify(x509.VerifyOptions{Roots: rootCAs, CurrentTime: now, KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}}); err != nil {
		return nil, fmt.Errorf("verify datafeed client certificate against approved CA: %w", err)
	}
	if err := verifyClientCertificateNotRevoked(config.RevocationListFile, trustedCAs, leaf, now); err != nil {
		return nil, err
	}
	clientCertificate.Leaf = leaf
	transport := &http.Transport{
		Proxy:                 nil,
		TLSClientConfig:       &tls.Config{MinVersion: tls.VersionTLS13, RootCAs: rootCAs, Certificates: []tls.Certificate{clientCertificate}},
		ForceAttemptHTTP2:     true,
		DisableCompression:    true,
		MaxIdleConns:          2,
		MaxConnsPerHost:       2,
		MaxIdleConnsPerHost:   2,
		IdleConnTimeout:       30 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 15 * time.Second,
	}
	return &MTLSClient{
		endpoint: *endpoint,
		config:   config,
		http: &http.Client{
			Transport: transport,
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return fmt.Errorf("datafeed publisher rejects redirect responses")
			},
		},
	}, nil
}

func parseApprovedCABundle(bundle []byte) (*x509.CertPool, []*x509.Certificate, error) {
	roots := x509.NewCertPool()
	var certificates []*x509.Certificate
	for remaining := bundle; len(remaining) > 0; {
		block, rest := pem.Decode(remaining)
		if block == nil || block.Type != "CERTIFICATE" {
			return nil, nil, fmt.Errorf("datafeed publisher approved CA bundle contains invalid PEM")
		}
		certificate, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return nil, nil, fmt.Errorf("parse datafeed publisher approved CA certificate: %w", err)
		}
		roots.AddCert(certificate)
		certificates = append(certificates, certificate)
		remaining = rest
	}
	if len(certificates) == 0 {
		return nil, nil, fmt.Errorf("datafeed publisher approved CA bundle contains no certificate")
	}
	return roots, certificates, nil
}

func verifyClientCertificateNotRevoked(filename string, trustedCAs []*x509.Certificate, leaf *x509.Certificate, now time.Time) error {
	contents, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("read datafeed approved revocation list: %w", err)
	}
	block, remaining := pem.Decode(contents)
	if block == nil || (block.Type != "X509 CRL" && block.Type != "CRL") || len(strings.TrimSpace(string(remaining))) != 0 {
		return fmt.Errorf("datafeed publisher approved revocation list contains invalid PEM")
	}
	list, err := x509.ParseRevocationList(block.Bytes)
	if err != nil {
		return fmt.Errorf("parse datafeed publisher approved revocation list: %w", err)
	}
	if now.Before(list.ThisUpdate) || !now.Before(list.NextUpdate) {
		return fmt.Errorf("datafeed publisher approved revocation list is not currently valid")
	}
	trustedIssuer := false
	for _, authority := range trustedCAs {
		if authority.Subject.String() != list.Issuer.String() {
			continue
		}
		if err := list.CheckSignatureFrom(authority); err != nil {
			continue
		}
		trustedIssuer = true
		break
	}
	if !trustedIssuer {
		return fmt.Errorf("datafeed publisher revocation list is not signed by an approved CA")
	}
	for _, entry := range list.RevokedCertificateEntries {
		if leaf.SerialNumber.Cmp(entry.SerialNumber) == 0 {
			return fmt.Errorf("datafeed publisher client certificate is revoked")
		}
	}
	return nil
}

// Submit performs one direct request to the exact locked v3 endpoint. It does
// not interpret success or advance a delivery state; the publisher validates
// the returned receipt before making any durable acknowledgement.
func (client *MTLSClient) Submit(ctx context.Context, request BatchRequest) (*http.Response, error) {
	if client == nil || client.http == nil {
		return nil, fmt.Errorf("datafeed direct mTLS client is not configured")
	}
	if strings.TrimSpace(request.RequestID) == "" || strings.TrimSpace(request.IdempotencyKey) == "" || strings.TrimSpace(request.ReplayID) == "" {
		return nil, fmt.Errorf("datafeed batch request requires request, idempotency, and replay identities")
	}
	if len(request.Body) == 0 {
		return nil, fmt.Errorf("datafeed batch request body is required")
	}
	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, client.endpoint.String(), strings.NewReader(string(request.Body)))
	if err != nil {
		return nil, fmt.Errorf("create datafeed direct mTLS request: %w", err)
	}
	httpRequest.Header.Set("Content-Type", aviaCoreV3MediaType)
	httpRequest.Header.Set("X-Request-Id", request.RequestID)
	httpRequest.Header.Set("Idempotency-Key", request.IdempotencyKey)
	httpRequest.Header.Set("X-Replay-Id", request.ReplayID)
	return client.http.Do(httpRequest)
}

func certificateHasExpectedSAN(certificate *x509.Certificate, expected string) bool {
	for _, uri := range certificate.URIs {
		if uri.String() == expected {
			return true
		}
	}
	return false
}

func isSHA256(value string) bool {
	if len(value) != 64 {
		return false
	}
	for _, character := range value {
		if !((character >= '0' && character <= '9') || (character >= 'a' && character <= 'f')) {
			return false
		}
	}
	return true
}
