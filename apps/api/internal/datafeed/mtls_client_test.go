package datafeed

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"encoding/pem"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewMTLSClientRejectsUnsafeTransportConfiguration(t *testing.T) {
	tests := []struct {
		name   string
		config MTLSClientConfig
	}{
		{
			name: "plaintext endpoint",
			config: MTLSClientConfig{
				Endpoint:               "http://ingest.example.test/v3/aviasurveil/event-batches",
				ApprovedCABundleSHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
				ClientCertificateFile:  "/test/client.crt",
				ClientPrivateKeyFile:   "/test/client.key",
				ExpectedClientSAN:      "spiffe://aviacore/tenant-contract-fixture/aviasurveil-production-api",
			},
		},
		{
			name: "missing approved CA fingerprint",
			config: MTLSClientConfig{
				Endpoint:              "https://ingest.example.test/v3/aviasurveil/event-batches",
				ClientCertificateFile: "/test/client.crt",
				ClientPrivateKeyFile:  "/test/client.key",
				ExpectedClientSAN:     "spiffe://aviacore/tenant-contract-fixture/aviasurveil-production-api",
			},
		},
		{
			name: "missing client certificate",
			config: MTLSClientConfig{
				Endpoint:               "https://ingest.example.test/v3/aviasurveil/event-batches",
				ApprovedCABundleSHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
				ClientPrivateKeyFile:   "/test/client.key",
				ExpectedClientSAN:      "spiffe://aviacore/tenant-contract-fixture/aviasurveil-production-api",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := NewMTLSClient(test.config); err == nil {
				t.Fatal("unsafe publisher transport configuration was accepted")
			}
		})
	}
}

func TestNewLocalCandidateClientIsExplicitlyInternalHTTPOnly(t *testing.T) {
	client, err := NewLocalCandidateClient("http://avia-data-admission:8080/v3/aviasurveil/event-batches")
	if err != nil || client == nil {
		t.Fatalf("local candidate client=%v err=%v", client, err)
	}
	if _, err := NewLocalCandidateClient("https://external.example/v3/aviasurveil/event-batches"); err == nil {
		t.Fatal("local candidate accepted an HTTPS external endpoint")
	}
}

func TestMTLSClientPinsCAUsesTLS13AndPresentsApprovedClientSAN(t *testing.T) {
	t.Parallel()
	const clientSAN = "spiffe://aviacore/tenant-contract-fixture/aviasurveil-production-api"
	caCertificate, caKey, caPEM := issueTestCA(t)
	serverCertificate := issueTestLeaf(t, caCertificate, caKey, leafOptions{dnsNames: []string{"127.0.0.1"}, ipAddresses: []net.IP{net.ParseIP("127.0.0.1")}, serverAuth: true})
	clientCertificate := issueTestLeaf(t, caCertificate, caKey, leafOptions{uriSANs: []string{clientSAN}, clientAuth: true})

	directory := t.TempDir()
	caFile := filepath.Join(directory, "ca.pem")
	clientCertificateFile := filepath.Join(directory, "client.pem")
	clientKeyFile := filepath.Join(directory, "client.key")
	revocationListFile := filepath.Join(directory, "client.crl")
	if err := os.WriteFile(caFile, caPEM, 0o600); err != nil {
		t.Fatal(err)
	}
	writeCertificateAndKey(t, clientCertificateFile, clientKeyFile, clientCertificate)
	writeRevocationList(t, revocationListFile, caCertificate, caKey)
	caDigest := sha256.Sum256(caPEM)

	clientCAs := x509.NewCertPool()
	if !clientCAs.AppendCertsFromPEM(caPEM) {
		t.Fatal("add test CA to server client pool")
	}
	received := false
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		received = true
		if request.TLS == nil || request.TLS.Version != tls.VersionTLS13 {
			t.Error("request did not use TLS 1.3")
		}
		if request.URL.Path != "/v3/aviasurveil/event-batches" {
			t.Errorf("path = %q", request.URL.Path)
		}
		if got := request.Header.Get("Content-Type"); got != aviaCoreV3MediaType {
			t.Errorf("content type = %q", got)
		}
		if got := request.Header.Get("X-Forwarded-Client-Cert"); got != "" {
			t.Errorf("forwarded client certificate header = %q", got)
		}
		if len(request.TLS.PeerCertificates) != 1 || !certificateHasURI(request.TLS.PeerCertificates[0], clientSAN) {
			t.Error("approved client SAN was not presented")
		}
		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(http.StatusCreated)
		_, _ = writer.Write([]byte(`{"request_id":"REQUEST","batch_id":"BATCH","attempt_id":"ATTEMPT","batch_state":"sealed_terminal","items":[{"event_id":"10000000-0000-4000-8000-000000000004","attempt_id":"ATTEMPT","event_content_digest":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","outcome":"accepted","safe_code":"ACCEPTED","manifest_receipt_digest":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","acknowledgement_receipt_id":"receipt-1","winner_event_id":"10000000-0000-4000-8000-000000000004"}]}`))
	}))
	server.TLS = &tls.Config{MinVersion: tls.VersionTLS13, Certificates: []tls.Certificate{serverCertificate}, ClientAuth: tls.RequireAndVerifyClientCert, ClientCAs: clientCAs}
	server.StartTLS()
	t.Cleanup(server.Close)

	endpoint, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	endpoint.Path = "/v3/aviasurveil/event-batches"
	client, err := NewMTLSClient(MTLSClientConfig{
		Endpoint:               endpoint.String(),
		CABundleFile:           caFile,
		RevocationListFile:     revocationListFile,
		ApprovedCABundleSHA256: hex.EncodeToString(caDigest[:]),
		ClientCertificateFile:  clientCertificateFile,
		ClientPrivateKeyFile:   clientKeyFile,
		ExpectedClientSAN:      clientSAN,
	})
	if err != nil {
		t.Fatalf("new mTLS client: %v", err)
	}
	_, err = (Publisher{Client: client, NewID: sequenceIDs("BATCH", "ATTEMPT", "REQUEST")}).Deliver(context.Background(), []BatchItem{{Event: map[string]any{"event_id": "10000000-0000-4000-8000-000000000004"}, EventID: "10000000-0000-4000-8000-000000000004", EventContentDigest: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", LeaseGeneration: 1}}, "replay-task5-fixture")
	if err != nil {
		t.Fatalf("submit direct mTLS batch: %v", err)
	}
	if !received {
		t.Fatal("receiver was not reached")
	}
}

func TestNewMTLSClientRejectsRevokedClientCertificate(t *testing.T) {
	t.Parallel()
	caCertificate, caKey, caPEM := issueTestCA(t)
	clientCertificate := issueTestLeaf(t, caCertificate, caKey, leafOptions{uriSANs: []string{"spiffe://aviacore/tenant-contract-fixture/aviasurveil-production-api"}, clientAuth: true})
	directory := t.TempDir()
	caFile := filepath.Join(directory, "ca.pem")
	clientCertificateFile := filepath.Join(directory, "client.pem")
	clientKeyFile := filepath.Join(directory, "client.key")
	revocationListFile := filepath.Join(directory, "revoked.crl")
	if err := os.WriteFile(caFile, caPEM, 0o600); err != nil {
		t.Fatal(err)
	}
	writeCertificateAndKey(t, clientCertificateFile, clientKeyFile, clientCertificate)
	writeRevocationList(t, revocationListFile, caCertificate, caKey, clientCertificate.Certificate[0])
	caDigest := sha256.Sum256(caPEM)
	if _, err := NewMTLSClient(MTLSClientConfig{
		Endpoint:               "https://ingest.example.test/v3/aviasurveil/event-batches",
		CABundleFile:           caFile,
		RevocationListFile:     revocationListFile,
		ApprovedCABundleSHA256: hex.EncodeToString(caDigest[:]),
		ClientCertificateFile:  clientCertificateFile,
		ClientPrivateKeyFile:   clientKeyFile,
		ExpectedClientSAN:      "spiffe://aviacore/tenant-contract-fixture/aviasurveil-production-api",
	}); err == nil {
		t.Fatal("revoked client certificate was accepted")
	}
}

type issuedLeaf struct {
	certificate tls.Certificate
}

type leafOptions struct {
	dnsNames    []string
	ipAddresses []net.IP
	uriSANs     []string
	serverAuth  bool
	clientAuth  bool
}

func issueTestCA(t *testing.T) (*x509.Certificate, crypto.Signer, []byte) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	template := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "task5-test-ca"}, SubjectKeyId: []byte("task5-test-ca-key"), NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour), IsCA: true, BasicConstraintsValid: true, KeyUsage: x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature | x509.KeyUsageCRLSign}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	certificate, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatal(err)
	}
	return certificate, key, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
}

func issueTestLeaf(t *testing.T, ca *x509.Certificate, caKey crypto.Signer, options leafOptions) tls.Certificate {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	uris := make([]*url.URL, 0, len(options.uriSANs))
	for _, rawURI := range options.uriSANs {
		uri, err := url.Parse(rawURI)
		if err != nil {
			t.Fatal(err)
		}
		uris = append(uris, uri)
	}
	extendedUsage := make([]x509.ExtKeyUsage, 0, 2)
	if options.serverAuth {
		extendedUsage = append(extendedUsage, x509.ExtKeyUsageServerAuth)
	}
	if options.clientAuth {
		extendedUsage = append(extendedUsage, x509.ExtKeyUsageClientAuth)
	}
	template := &x509.Certificate{SerialNumber: big.NewInt(time.Now().UnixNano()), Subject: pkix.Name{CommonName: "task5-test-leaf"}, NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour), DNSNames: options.dnsNames, IPAddresses: options.ipAddresses, URIs: uris, KeyUsage: x509.KeyUsageDigitalSignature, ExtKeyUsage: extendedUsage}
	der, err := x509.CreateCertificate(rand.Reader, template, ca, &key.PublicKey, caKey)
	if err != nil {
		t.Fatal(err)
	}
	return tls.Certificate{Certificate: [][]byte{der, ca.Raw}, PrivateKey: key}
}

func writeCertificateAndKey(t *testing.T, certificateFile, keyFile string, issued tls.Certificate) {
	t.Helper()
	if err := os.WriteFile(certificateFile, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: issued.Certificate[0]}), 0o600); err != nil {
		t.Fatal(err)
	}
	key, ok := issued.PrivateKey.(*rsa.PrivateKey)
	if !ok {
		t.Fatal("test client private key is not RSA")
	}
	if err := os.WriteFile(keyFile, pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)}), 0o600); err != nil {
		t.Fatal(err)
	}
}

func writeRevocationList(t *testing.T, filename string, ca *x509.Certificate, caKey crypto.Signer, revokedCertificateDER ...[]byte) {
	t.Helper()
	revoked := make([]x509.RevocationListEntry, 0, len(revokedCertificateDER))
	for _, rawCertificate := range revokedCertificateDER {
		certificate, err := x509.ParseCertificate(rawCertificate)
		if err != nil {
			t.Fatal(err)
		}
		revoked = append(revoked, x509.RevocationListEntry{SerialNumber: certificate.SerialNumber, RevocationTime: time.Now().Add(-time.Minute)})
	}
	der, err := x509.CreateRevocationList(rand.Reader, &x509.RevocationList{Number: big.NewInt(1), ThisUpdate: time.Now().Add(-time.Minute), NextUpdate: time.Now().Add(time.Hour), RevokedCertificateEntries: revoked}, ca, caKey)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filename, pem.EncodeToMemory(&pem.Block{Type: "X509 CRL", Bytes: der}), 0o600); err != nil {
		t.Fatal(err)
	}
}

func certificateHasURI(certificate *x509.Certificate, expected string) bool {
	for _, uri := range certificate.URIs {
		if uri.String() == expected {
			return true
		}
	}
	return false
}
