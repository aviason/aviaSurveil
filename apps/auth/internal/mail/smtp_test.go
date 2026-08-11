package mail

import (
	"context"
	"crypto/tls"
	"errors"
	"testing"
)

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
