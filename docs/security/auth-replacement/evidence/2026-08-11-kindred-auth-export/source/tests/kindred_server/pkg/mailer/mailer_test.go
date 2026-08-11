package mailer

import (
	"net/url"
	"strings"
	"testing"
)

func TestVerificationEmailBodyIncludesDeepLinkAndToken(t *testing.T) {
	email := "alice+test@example.com"
	token := "verify/token+123"
	body := verificationEmailBody(email, token, "kindred")

	wantLink := "kindred://auth/verify-email?email=" + url.QueryEscape(email) + "&token=" + url.QueryEscape(token)
	if !strings.Contains(body, wantLink) {
		t.Fatalf("verification body missing deep link %q:\n%s", wantLink, body)
	}
	if !strings.Contains(body, "Token: "+token) {
		t.Fatalf("verification body missing raw token fallback:\n%s", body)
	}
}

func TestPasswordResetEmailBodyIncludesDeepLinkAndToken(t *testing.T) {
	email := "alice+reset@example.com"
	token := "reset/token+123"
	body := passwordResetEmailBody(email, token, "kindred")

	wantLink := "kindred://auth/reset-password?email=" + url.QueryEscape(email) + "&token=" + url.QueryEscape(token)
	if !strings.Contains(body, wantLink) {
		t.Fatalf("reset body missing deep link %q:\n%s", wantLink, body)
	}
	if !strings.Contains(body, "Token: "+token) {
		t.Fatalf("reset body missing raw token fallback:\n%s", body)
	}
}

func TestNormalizeSchemeDefaultsToKindred(t *testing.T) {
	if got := normalizeScheme(""); got != "kindred" {
		t.Fatalf("empty scheme = %q, want kindred", got)
	}
	if got := normalizeScheme("kindred://"); got != "kindred" {
		t.Fatalf("scheme with suffix = %q, want kindred", got)
	}
}
