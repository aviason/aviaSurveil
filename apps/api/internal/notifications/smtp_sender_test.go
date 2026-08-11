package notifications

import (
	"bufio"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"
	"net"
	"strings"
	"sync"
	"testing"
	"time"
)

func smtpTestCertificate(t *testing.T) (tls.Certificate, *x509.CertPool) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate TLS key: %v", err)
	}
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "smtp.fixture"}, DNSNames: []string{"smtp.fixture"},
		NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour),
		KeyUsage: x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("create TLS certificate: %v", err)
	}
	keyDER, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatalf("marshal TLS key: %v", err)
	}
	certificate, err := tls.X509KeyPair(
		pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}),
		pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER}),
	)
	if err != nil {
		t.Fatalf("parse TLS key pair: %v", err)
	}
	roots := x509.NewCertPool()
	roots.AddCert(template)
	parsed, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatalf("parse TLS certificate: %v", err)
	}
	roots = x509.NewCertPool()
	roots.AddCert(parsed)
	return certificate, roots
}

func startSecureSMTPFixture(t *testing.T, transport string, advertiseStartTLS bool) (*smtpFixture, *x509.CertPool) {
	t.Helper()
	certificate, roots := smtpTestCertificate(t)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen secure SMTP fixture: %v", err)
	}
	fixture := &smtpFixture{address: listener.Addr().String(), close: func() { _ = listener.Close() }}
	t.Cleanup(fixture.close)
	go func() {
		connection, acceptErr := listener.Accept()
		if acceptErr != nil {
			return
		}
		defer connection.Close()
		if transport == SMTPTransportImplicitTLS {
			connection = tls.Server(connection, &tls.Config{Certificates: []tls.Certificate{certificate}, MinVersion: tls.VersionTLS12})
			if err := connection.(*tls.Conn).Handshake(); err != nil {
				return
			}
		}
		reader := bufio.NewReader(connection)
		writer := bufio.NewWriter(connection)
		write := func(value string) { _, _ = writer.WriteString(value); _ = writer.Flush() }
		write("220 smtp.fixture ESMTP\r\n")
		secure := transport == SMTPTransportImplicitTLS
		for {
			line, readErr := reader.ReadString('\n')
			if readErr != nil {
				return
			}
			switch {
			case strings.HasPrefix(line, "EHLO"):
				if advertiseStartTLS && !secure {
					write("250-smtp.fixture\r\n250-STARTTLS\r\n250 AUTH PLAIN\r\n")
				} else {
					write("250-smtp.fixture\r\n250 AUTH PLAIN\r\n")
				}
			case strings.HasPrefix(line, "STARTTLS") && advertiseStartTLS && !secure:
				write("220 2.0.0 begin TLS\r\n")
				tlsConnection := tls.Server(connection, &tls.Config{Certificates: []tls.Certificate{certificate}, MinVersion: tls.VersionTLS12})
				if err := tlsConnection.Handshake(); err != nil {
					return
				}
				connection = tlsConnection
				reader = bufio.NewReader(connection)
				writer = bufio.NewWriter(connection)
				secure = true
			case strings.HasPrefix(line, "AUTH PLAIN"):
				if !secure {
					write("530 5.7.0 encryption required\r\n")
					return
				}
				write("235 2.7.0 authenticated\r\n")
			case strings.HasPrefix(line, "MAIL FROM"), strings.HasPrefix(line, "RCPT TO"):
				write("250 OK\r\n")
			case strings.HasPrefix(line, "DATA"):
				write("354 send message\r\n")
				for {
					dataLine, dataErr := reader.ReadString('\n')
					if dataErr != nil {
						return
					}
					if dataLine == ".\r\n" {
						break
					}
				}
				write("250 queued\r\n")
			case strings.HasPrefix(line, "QUIT"):
				write("221 bye\r\n")
				return
			default:
				write("250 OK\r\n")
			}
		}
	}()
	return fixture, roots
}

type smtpFixture struct {
	address string
	body    string
	mu      sync.Mutex
	close   func()
}

func startSMTPFixture(
	t *testing.T,
	rcptStatus string,
	holdData bool,
) *smtpFixture {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen SMTP fixture: %v", err)
	}
	fixture := &smtpFixture{
		address: listener.Addr().String(),
		close:   func() { _ = listener.Close() },
	}
	t.Cleanup(fixture.close)
	go func() {
		connection, acceptErr := listener.Accept()
		if acceptErr != nil {
			return
		}
		defer connection.Close()
		reader := bufio.NewReader(connection)
		writer := bufio.NewWriter(connection)
		write := func(value string) {
			_, _ = writer.WriteString(value)
			_ = writer.Flush()
		}
		write("220 smtp.fixture ESMTP\r\n")
		for {
			line, readErr := reader.ReadString('\n')
			if readErr != nil {
				return
			}
			switch {
			case strings.HasPrefix(line, "EHLO"):
				write("250-smtp.fixture\r\n250-AUTH PLAIN\r\n250 OK\r\n")
			case strings.HasPrefix(line, "AUTH PLAIN"):
				write("235 2.7.0 authenticated\r\n")
			case strings.HasPrefix(line, "MAIL FROM"):
				write("250 2.1.0 sender accepted\r\n")
			case strings.HasPrefix(line, "RCPT TO"):
				if rcptStatus != "" {
					write(rcptStatus + "\r\n")
					return
				}
				write("250 2.1.5 recipient accepted\r\n")
			case strings.HasPrefix(line, "DATA"):
				write("354 send message\r\n")
				if holdData {
					_, _ = io.Copy(io.Discard, reader)
					return
				}
				var message strings.Builder
				for {
					dataLine, dataErr := reader.ReadString('\n')
					if dataErr != nil {
						return
					}
					if dataLine == ".\r\n" {
						break
					}
					message.WriteString(dataLine)
				}
				fixture.mu.Lock()
				fixture.body = message.String()
				fixture.mu.Unlock()
				write("250 2.0.0 queued\r\n")
			case strings.HasPrefix(line, "QUIT"):
				write("221 2.0.0 bye\r\n")
				return
			default:
				write("250 OK\r\n")
			}
		}
	}()
	return fixture
}

func TestSMTPSenderDeliversBoundedMultipartMessageWithStableMessageID(t *testing.T) {
	t.Parallel()
	fixture := startSMTPFixture(t, "", false)
	sender, err := NewSMTPSender(SMTPConfig{
		Address: fixture.address, From: "no-reply@aviasurveil360.local",
		Username: "aviasurveil360", Password: "test-secret",
		Timeout: time.Second, PrivateNetwork: true,
	})
	if err != nil {
		t.Fatalf("NewSMTPSender() error = %v", err)
	}
	err = sender.Deliver(context.Background(), EmailDelivery{
		JobID: "job-001", NotificationID: "notification-001",
		RecipientSubjectID: "auditee-001",
		RecipientEmail:     "auditee@example.test",
		RecipientAudience:  EmailAudienceAuditee,
		OrganizationID:     "ORG-001",
		Title:              "CAP update",
		Body:               "Open the authorized record.",
		RelatedEntityType:  "FINDING",
		RelatedEntityID:    "FND-001",
		ProviderMessageID:  "<notification-job-001@aviasurveil360.local>",
		Attempt:            1,
	})
	if err != nil {
		t.Fatalf("Deliver() error = %v", err)
	}
	fixture.mu.Lock()
	message := fixture.body
	fixture.mu.Unlock()
	for _, expected := range []string{
		"Message-ID: <notification-job-001@aviasurveil360.local>",
		"To: auditee@example.test",
		"Subject: CAP update",
		"multipart/alternative",
		"Open the authorized record.",
	} {
		if !strings.Contains(message, expected) {
			t.Fatalf("SMTP message omitted %q:\n%s", expected, message)
		}
	}
	if strings.Contains(message, "test-secret") {
		t.Fatal("SMTP message leaked credentials")
	}
}

func TestSMTPSenderClassifiesRefusalAndTimeoutWithoutLeakingProviderText(t *testing.T) {
	t.Parallel()
	refusing := startSMTPFixture(t, "550 5.1.1 private provider detail", false)
	sender, err := NewSMTPSender(SMTPConfig{
		Address: refusing.address, From: "no-reply@aviasurveil360.local",
		Username: "aviasurveil360", Password: "test-secret",
		Timeout: time.Second, PrivateNetwork: true,
	})
	if err != nil {
		t.Fatalf("NewSMTPSender() error = %v", err)
	}
	delivery := EmailDelivery{
		JobID: "job-refused", NotificationID: "notification-refused",
		RecipientSubjectID: "auditee-refused",
		RecipientEmail:     "refused@example.test",
		RecipientAudience:  EmailAudienceAuditee,
		Title:              "Notification",
		Body:               "Open the authorized record.",
		ProviderMessageID:  "<notification-job-refused@aviasurveil360.local>",
	}
	err = sender.Deliver(context.Background(), delivery)
	if err == nil || !IsPermanentDeliveryFailure(err) ||
		DeliveryFailureCode(err) != "SMTP_RECIPIENT_REJECTED" {
		t.Fatalf("SMTP refusal classification = %T %v", err, err)
	}
	if strings.Contains(err.Error(), "private provider detail") {
		t.Fatalf("SMTP refusal leaked provider response: %v", err)
	}

	holding := startSMTPFixture(t, "", true)
	timeoutSender, err := NewSMTPSender(SMTPConfig{
		Address: holding.address, From: "no-reply@aviasurveil360.local",
		Username: "aviasurveil360", Password: "test-secret",
		Timeout: 50 * time.Millisecond, PrivateNetwork: true,
	})
	if err != nil {
		t.Fatalf("NewSMTPSender(timeout) error = %v", err)
	}
	timeoutDelivery := delivery
	timeoutDelivery.JobID = "job-timeout"
	timeoutDelivery.ProviderMessageID = "<notification-job-timeout@aviasurveil360.local>"
	err = timeoutSender.Deliver(context.Background(), timeoutDelivery)
	if err == nil || IsPermanentDeliveryFailure(err) ||
		DeliveryFailureCode(err) != "SMTP_DATA_REJECTED_TIMEOUT" {
		t.Fatalf("SMTP timeout classification = %T %v", err, err)
	}
	if strings.Contains(err.Error(), holding.address) ||
		strings.Contains(err.Error(), "test-secret") {
		t.Fatalf("SMTP timeout leaked transport detail: %v", err)
	}
}

func TestSMTPSenderRequiresExplicitPrivateNetworkForPlaintextCredentials(t *testing.T) {
	t.Parallel()
	_, err := NewSMTPSender(SMTPConfig{
		Address: "mailpit:1025", From: "no-reply@aviasurveil360.local",
		Username: "aviasurveil360", Password: "test-secret",
		Timeout: time.Second,
	})
	if err == nil || !strings.Contains(err.Error(), "private network") {
		t.Fatalf("plaintext SMTP config error = %v", err)
	}
}

func secureDelivery() EmailDelivery {
	return EmailDelivery{
		JobID: "job-secure", NotificationID: "notification-secure", RecipientSubjectID: "auditee-secure",
		RecipientEmail: "auditee@example.test", RecipientAudience: EmailAudienceAuditee,
		Title: "Secure notification", Body: "Open the authorized record.", ProviderMessageID: "<notification-secure@aviasurveil360.local>",
	}
}

func TestSMTPSenderSupportsVerifiedImplicitTLSAndMandatorySTARTTLS(t *testing.T) {
	for _, transport := range []string{SMTPTransportImplicitTLS, SMTPTransportStartTLS} {
		t.Run(transport, func(t *testing.T) {
			fixture, roots := startSecureSMTPFixture(t, transport, true)
			sender, err := NewSMTPSender(SMTPConfig{
				Address: fixture.address, From: "no-reply@aviasurveil360.local", Username: "pilot", Password: "test-secret",
				Timeout: time.Second, Transport: transport, TLSServerName: "smtp.fixture", TLSConfig: &tls.Config{RootCAs: roots},
			})
			if err != nil {
				t.Fatalf("NewSMTPSender() error = %v", err)
			}
			if err := sender.Deliver(context.Background(), secureDelivery()); err != nil {
				t.Fatalf("Deliver() error = %v", err)
			}
		})
	}
}

func TestSMTPSenderFailsClosedForMissingSTARTTLSWrongCertificateAndVerificationBypass(t *testing.T) {
	missing, roots := startSecureSMTPFixture(t, SMTPTransportStartTLS, false)
	sender, err := NewSMTPSender(SMTPConfig{
		Address: missing.address, From: "no-reply@aviasurveil360.local", Username: "pilot", Password: "test-secret",
		Timeout: time.Second, Transport: SMTPTransportStartTLS, TLSServerName: "smtp.fixture", TLSConfig: &tls.Config{RootCAs: roots},
	})
	if err != nil {
		t.Fatalf("NewSMTPSender() error = %v", err)
	}
	if err := sender.Deliver(context.Background(), secureDelivery()); err == nil || DeliveryFailureCode(err) != "SMTP_STARTTLS_REQUIRED" {
		t.Fatalf("missing STARTTLS error = %v", err)
	}

	wrong, wrongRoots := startSecureSMTPFixture(t, SMTPTransportImplicitTLS, true)
	wrongSender, err := NewSMTPSender(SMTPConfig{
		Address: wrong.address, From: "no-reply@aviasurveil360.local", Username: "pilot", Password: "test-secret",
		Timeout: time.Second, Transport: SMTPTransportImplicitTLS, TLSServerName: "wrong.fixture", TLSConfig: &tls.Config{RootCAs: wrongRoots},
	})
	if err != nil {
		t.Fatalf("NewSMTPSender(wrong name) error = %v", err)
	}
	if err := wrongSender.Deliver(context.Background(), secureDelivery()); err == nil || DeliveryFailureCode(err) != "SMTP_TLS_VERIFICATION_FAILED" || strings.Contains(err.Error(), "test-secret") {
		t.Fatalf("wrong certificate error = %v", err)
	}

	_, err = NewSMTPSender(SMTPConfig{
		Address: "smtp.example.invalid:465", From: "no-reply@example.invalid", Username: "pilot", Password: "test-secret",
		Timeout: time.Second, Transport: SMTPTransportImplicitTLS, TLSConfig: &tls.Config{InsecureSkipVerify: true},
	})
	if err == nil || !strings.Contains(err.Error(), "cannot be disabled") {
		t.Fatalf("TLS bypass error = %v", err)
	}
}

func ExampleDeliveryFailureCode() {
	err := NewPermanentDeliveryFailure("SMTP_RECIPIENT_REJECTED")
	fmt.Println(DeliveryFailureCode(err))
	// Output: SMTP_RECIPIENT_REJECTED
}
