package mailer

import (
	"context"
	"fmt"
	"net/smtp"
	"net/url"
	"strings"

	"kindred_server/pkg/logger"
)

// Mailer sends transactional emails. The production implementation should be
// backed by SES/SMTP; tests and local development use LogMailer.
type Mailer interface {
	SendVerificationEmail(ctx context.Context, to, token string) error
	SendPasswordResetEmail(ctx context.Context, to, token string) error
}

// LogMailer writes message metadata to the logger instead of sending an email.
// It is safe for local development and tests.
type LogMailer struct {
	log            *logger.Logger
	deepLinkScheme string
}

func NewLogMailer(log *logger.Logger, deepLinkScheme string) *LogMailer {
	return &LogMailer{log: log, deepLinkScheme: normalizeScheme(deepLinkScheme)}
}

func (m *LogMailer) SendVerificationEmail(_ context.Context, to, token string) error {
	m.log.Info("email verification dispatched", map[string]any{
		"to": to,
	})
	return nil
}

func (m *LogMailer) SendPasswordResetEmail(_ context.Context, to, token string) error {
	m.log.Info("password reset dispatched", map[string]any{
		"to": to,
	})
	return nil
}

// SMTPMailer sends emails via SMTP (Google/Gmail or other providers).
type SMTPMailer struct {
	from     string
	password string
	host     string
	port     int
	log      *logger.Logger
	scheme   string
}

func NewSMTPMailer(from, password, host string, port int, log *logger.Logger, deepLinkScheme string) *SMTPMailer {
	return &SMTPMailer{
		from:     from,
		password: password,
		host:     host,
		port:     port,
		log:      log,
		scheme:   normalizeScheme(deepLinkScheme),
	}
}

func (m *SMTPMailer) SendVerificationEmail(ctx context.Context, to, token string) error {
	subject := "Kindred - Email Verification"
	body := verificationEmailBody(to, token, m.scheme)

	return m.send(ctx, to, subject, body)
}

func (m *SMTPMailer) SendPasswordResetEmail(ctx context.Context, to, token string) error {
	subject := "Kindred - Password Reset"
	body := passwordResetEmailBody(to, token, m.scheme)

	return m.send(ctx, to, subject, body)
}

func (m *SMTPMailer) send(ctx context.Context, to, subject, body string) error {
	addr := fmt.Sprintf("%s:%d", m.host, m.port)
	auth := smtp.PlainAuth("", m.from, m.password, m.host)

	// Prepare email message
	message := fmt.Sprintf("To: %s\r\nSubject: %s\r\n\r\n%s", to, subject, body)

	// Send email
	err := smtp.SendMail(addr, auth, m.from, []string{to}, []byte(message))
	if err != nil {
		m.log.Error("failed to send email", err, map[string]any{
			"to": to,
		})
		return err
	}

	m.log.Info("email sent successfully", map[string]any{
		"to":      to,
		"subject": subject,
	})
	return nil
}

func verificationEmailBody(to, token, scheme string) string {
	link := authDeepLink(scheme, "/verify-email", to, token)
	return fmt.Sprintf(`Hello,

Please verify your email address using this link or token:

%s

Token: %s

If you did not create this account, please ignore this email.

Best regards,
Kindred Team
`, link, token)
}

func passwordResetEmailBody(to, token, scheme string) string {
	link := authDeepLink(scheme, "/reset-password", to, token)
	return fmt.Sprintf(`Hello,

You requested to reset your password. Open this link or use the token below:

%s

Token: %s

If you did not request this, please ignore this email.

Best regards,
Kindred Team
`, link, token)
}

func authDeepLink(scheme, path, email, token string) string {
	q := url.Values{}
	q.Set("email", email)
	q.Set("token", token)
	u := url.URL{
		Scheme:   normalizeScheme(scheme),
		Host:     "auth",
		Path:     path,
		RawQuery: q.Encode(),
	}
	return u.String()
}

func normalizeScheme(scheme string) string {
	scheme = strings.TrimSpace(scheme)
	scheme = strings.TrimSuffix(scheme, "://")
	scheme = strings.TrimSuffix(scheme, ":")
	if scheme == "" {
		return "kindred"
	}
	return scheme
}
