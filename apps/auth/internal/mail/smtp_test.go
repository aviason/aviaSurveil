package mail

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/MarlonJD/aviaSurveil360/apps/auth/migrations"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestMailpitSTARTTLSDelivery(t *testing.T) {
	sender, mailpitURL := mailpitSenderFromEnvironment(t)
	if err := sender.Probe(context.Background()); err != nil {
		t.Fatalf("probe Mailpit STARTTLS transport: %v", err)
	}
	if err := sender.Send(context.Background(), "mailpit-recipient@example.invalid", "candidate delivery", "candidate-only delivery receipt"); err != nil {
		t.Fatalf("send through Mailpit: %v", err)
	}
	requireMailpitReceipt(t, mailpitURL, "mailpit-recipient@example.invalid")
}

func TestMailpitSTARTTLSOutboxRetryDelivery(t *testing.T) {
	databaseURL := strings.TrimSpace(os.Getenv("AVIA_AUTH_TEST_DATABASE_URL"))
	if databaseURL == "" {
		t.Skip("not run: AVIA_AUTH_TEST_DATABASE_URL is not configured")
	}
	sender, mailpitURL := mailpitSenderFromEnvironment(t)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("open disposable PostgreSQL: %v", err)
	}
	defer pool.Close()
	if err := migrations.Apply(ctx, pool); err != nil {
		t.Fatalf("apply auth migrations: %v", err)
	}
	clock := &outboxTestClock{now: time.Date(2026, 8, 11, 14, 0, 0, 0, time.UTC)}
	outbox, err := NewOutbox(OutboxConfig{
		Pool: pool, EncryptionKey: []byte("01234567890123456789012345678901"), Clock: clock.Now,
		LeaseTTL: 5 * time.Second, MaxAttempts: 3,
	})
	if err != nil {
		t.Fatalf("create PostgreSQL outbox: %v", err)
	}
	recipient := "mailpit-outbox@example.invalid"
	id, err := outbox.Enqueue(ctx, Delivery{Recipient: recipient, Subject: "candidate retry delivery", Body: "candidate-only retry receipt"})
	if err != nil {
		t.Fatalf("enqueue Mailpit delivery: %v", err)
	}
	defer func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM auth_identity.mail_deliveries WHERE delivery_id = $1`, id)
	}()
	if found, err := outbox.DeliverOnce(ctx, &outboxTestSender{err: errors.New("simulated transient SMTP failure")}); err != nil || !found {
		t.Fatalf("record transient SMTP failure = found:%v err:%v", found, err)
	}
	clock.Advance(time.Second)
	if found, err := outbox.DeliverOnce(ctx, sender); err != nil || !found {
		t.Fatalf("deliver retried mail through Mailpit = found:%v err:%v", found, err)
	}
	snapshot, err := outbox.Snapshot(ctx, id)
	if err != nil || snapshot.State != DeliveryDelivered || snapshot.AttemptCount != 2 {
		t.Fatalf("Mailpit outbox snapshot = %+v/%v", snapshot, err)
	}
	requireMailpitReceipt(t, mailpitURL, recipient)
}

func mailpitSenderFromEnvironment(t *testing.T) (*Sender, string) {
	t.Helper()
	address := strings.TrimSpace(os.Getenv("AVIA_AUTH_MAILPIT_ADDRESS"))
	mailpitURL := strings.TrimSpace(os.Getenv("AVIA_AUTH_MAILPIT_HTTP_URL"))
	passwordPath := strings.TrimSpace(os.Getenv("AVIA_AUTH_MAILPIT_PASSWORD_FILE"))
	certificatePath := strings.TrimSpace(os.Getenv("AVIA_AUTH_MAILPIT_CA_FILE"))
	if address == "" || mailpitURL == "" || passwordPath == "" || certificatePath == "" {
		t.Skip("not run: Mailpit integration environment is not configured")
	}
	password, err := os.ReadFile(passwordPath)
	if err != nil {
		t.Fatalf("read Mailpit password: %v", err)
	}
	certificate, err := os.ReadFile(certificatePath)
	if err != nil {
		t.Fatalf("read Mailpit CA: %v", err)
	}
	roots := x509.NewCertPool()
	if !roots.AppendCertsFromPEM(certificate) {
		t.Fatal("parse Mailpit CA")
	}
	sender, err := NewSender(Config{
		Address: address, Host: "127.0.0.1", From: "identity@aviasurveil360.local",
		Username: "aviasurveil360", Password: strings.TrimSpace(string(password)),
		TLSMode: TLSModeStartTLS, Timeout: 10 * time.Second,
		TLSConfig: &tls.Config{RootCAs: roots, ServerName: "127.0.0.1", MinVersion: tls.VersionTLS12},
	})
	if err != nil {
		t.Fatalf("new Mailpit sender: %v", err)
	}
	return sender, mailpitURL
}

func requireMailpitReceipt(t *testing.T, mailpitURL, recipient string) {
	t.Helper()
	client := &http.Client{Timeout: 10 * time.Second}
	request, err := http.NewRequestWithContext(context.Background(), http.MethodGet, mailpitURL+"/api/v1/messages", nil)
	if err != nil {
		t.Fatalf("construct Mailpit receipt request: %v", err)
	}
	response, err := client.Do(request)
	if err != nil {
		t.Fatalf("read Mailpit receipt: %v", err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read Mailpit receipt body: %v", err)
	}
	if response.StatusCode != http.StatusOK || !strings.Contains(string(body), recipient) {
		t.Fatalf("Mailpit delivery receipt status/body = %d/%q", response.StatusCode, body)
	}
}

func TestSMTPConfigurationRequiresAuthenticatedTLS(t *testing.T) {
	base := Config{Address: "smtp.example.invalid:587", Host: "smtp.example.invalid", From: "auth@example.invalid", Username: "auth", Password: "not-placeholder", TLSMode: TLSModeStartTLS}
	if _, err := NewSender(base); err != nil {
		t.Fatalf("valid STARTTLS config rejected: %v", err)
	}
	plaintext := base
	plaintext.TLSMode = "plain"
	if _, err := NewSender(plaintext); err == nil {
		t.Fatal("plaintext SMTP transport accepted")
	}
	unsafe := base
	unsafe.TLSConfig = &tls.Config{InsecureSkipVerify: true} //nolint:gosec -- asserting rejection of unsafe test input
	if _, err := NewSender(unsafe); err == nil {
		t.Fatal("insecure SMTP TLS config accepted")
	}
}

func TestSMTPMessageInputIsHeaderInjectionSafe(t *testing.T) {
	sender, err := NewSender(Config{Address: "smtp.example.invalid:587", Host: "smtp.example.invalid", From: "auth@example.invalid", Username: "auth", Password: "not-placeholder", TLSMode: TLSModeStartTLS})
	if err != nil {
		t.Fatal(err)
	}
	if err := sender.Send(context.Background(), "victim@example.invalid\r\nBcc: attacker@example.invalid", "subject", "body"); err == nil {
		t.Fatal("recipient header injection accepted")
	}
	if err := sender.Send(context.Background(), "victim@example.invalid", "subject\r\nBcc: attacker@example.invalid", "body"); err == nil {
		t.Fatal("subject header injection accepted")
	}
	if err := sender.Send(context.Background(), "victim@example.invalid", "subject", "body\x00"); err == nil {
		t.Fatal("NUL body accepted")
	}
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	if err := sender.Send(canceled, "victim@example.invalid", "subject", "body"); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled SMTP send = %v", err)
	}
}
