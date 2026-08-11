package notifications

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/tls"
	"embed"
	"errors"
	"fmt"
	htmltemplate "html/template"
	"net"
	"net/mail"
	"net/smtp"
	"net/textproto"
	"strconv"
	"strings"
	texttemplate "text/template"
	"time"
)

const (
	MaximumEmailTitleBytes   = 200
	maximumEmailSummaryBytes = 4 * 1024
	maximumEmailContextBytes = 4 * 1024
	maximumEmailMessageBytes = 64 * 1024
)

type EmailAudience string

const (
	EmailAudienceAuditee EmailAudience = "AUDITEE"
	EmailAudienceCAA     EmailAudience = "CAA"
)

type RenderedEmail struct {
	Subject string
	Text    string
	HTML    string
}

type AuditeeTemplateData struct {
	Title             string
	Summary           string
	OrganizationName  string
	RelatedRecordType string
	RelatedRecordID   string
}

type InternalCAATemplateData struct {
	Title             string
	Summary           string
	InternalContext   string
	RelatedRecordType string
	RelatedRecordID   string
}

//go:embed templates/*.tmpl
var emailTemplates embed.FS

func RenderAuditeeEmail(data AuditeeTemplateData) (RenderedEmail, error) {
	if err := validateTemplateValues(
		data.Title,
		data.Summary,
		"",
		data.OrganizationName,
		data.RelatedRecordType,
		data.RelatedRecordID,
	); err != nil {
		return RenderedEmail{}, err
	}
	return executeTemplates(
		"templates/auditee.text.tmpl",
		"templates/auditee.html.tmpl",
		data.Title,
		data,
	)
}

func RenderInternalCAAEmail(data InternalCAATemplateData) (RenderedEmail, error) {
	if err := validateTemplateValues(
		data.Title,
		data.Summary,
		data.InternalContext,
		"",
		data.RelatedRecordType,
		data.RelatedRecordID,
	); err != nil {
		return RenderedEmail{}, err
	}
	return executeTemplates(
		"templates/internal-caa.text.tmpl",
		"templates/internal-caa.html.tmpl",
		data.Title,
		data,
	)
}

func validateTemplateValues(
	title string,
	summary string,
	internalContext string,
	organizationName string,
	relatedRecordType string,
	relatedRecordID string,
) error {
	for _, entry := range []struct {
		name     string
		value    string
		maximum  int
		required bool
	}{
		{name: "title", value: title, maximum: MaximumEmailTitleBytes, required: true},
		{name: "summary", value: summary, maximum: maximumEmailSummaryBytes, required: true},
		{name: "internal context", value: internalContext, maximum: maximumEmailContextBytes},
		{name: "organization name", value: organizationName, maximum: 256},
		{name: "related record type", value: relatedRecordType, maximum: 64},
		{name: "related record ID", value: relatedRecordID, maximum: 256},
	} {
		trimmed := strings.TrimSpace(entry.value)
		if entry.required && trimmed == "" {
			return fmt.Errorf("%s is required", entry.name)
		}
		if len(entry.value) > entry.maximum {
			return fmt.Errorf("%s exceeds %d bytes", entry.name, entry.maximum)
		}
		if strings.ContainsAny(entry.value, "\r\n") && entry.name == "title" {
			return errors.New("title contains a forbidden header delimiter")
		}
	}
	return nil
}

func executeTemplates(
	textName string,
	htmlName string,
	subject string,
	data any,
) (RenderedEmail, error) {
	textSource, err := emailTemplates.ReadFile(textName)
	if err != nil {
		return RenderedEmail{}, fmt.Errorf("read text email template: %w", err)
	}
	htmlSource, err := emailTemplates.ReadFile(htmlName)
	if err != nil {
		return RenderedEmail{}, fmt.Errorf("read HTML email template: %w", err)
	}
	textParsed, err := texttemplate.New(textName).Option("missingkey=error").Parse(string(textSource))
	if err != nil {
		return RenderedEmail{}, fmt.Errorf("parse text email template: %w", err)
	}
	htmlParsed, err := htmltemplate.New(htmlName).Option("missingkey=error").Parse(string(htmlSource))
	if err != nil {
		return RenderedEmail{}, fmt.Errorf("parse HTML email template: %w", err)
	}
	var textBody bytes.Buffer
	if err := textParsed.Execute(&textBody, data); err != nil {
		return RenderedEmail{}, fmt.Errorf("render text email template: %w", err)
	}
	var htmlBody bytes.Buffer
	if err := htmlParsed.Execute(&htmlBody, data); err != nil {
		return RenderedEmail{}, fmt.Errorf("render HTML email template: %w", err)
	}
	if textBody.Len()+htmlBody.Len() > maximumEmailMessageBytes {
		return RenderedEmail{}, errors.New("rendered email exceeds the message-size limit")
	}
	return RenderedEmail{
		Subject: strings.TrimSpace(subject),
		Text:    strings.TrimSpace(textBody.String()),
		HTML:    strings.TrimSpace(htmlBody.String()),
	}, nil
}

type SMTPConfig struct {
	Address        string
	From           string
	Username       string
	Password       string
	Timeout        time.Duration
	PrivateNetwork bool
	Transport      string
	TLSServerName  string
	TLSConfig      *tls.Config
}

const (
	SMTPTransportPrivatePlaintext = "private-plaintext"
	SMTPTransportStartTLS         = "starttls"
	SMTPTransportImplicitTLS      = "implicit-tls"
)

type SMTPSender struct {
	address   string
	host      string
	from      string
	username  string
	password  string
	timeout   time.Duration
	transport string
	tlsConfig *tls.Config
}

func NewSMTPSender(config SMTPConfig) (*SMTPSender, error) {
	address := strings.TrimSpace(config.Address)
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return nil, errors.New("SMTP address must contain host and port")
	}
	from, err := mail.ParseAddress(strings.TrimSpace(config.From))
	if err != nil || from.Address == "" {
		return nil, errors.New("SMTP sender address is invalid")
	}
	if strings.TrimSpace(config.Username) == "" ||
		strings.TrimSpace(config.Password) == "" {
		return nil, errors.New("SMTP username and password are required")
	}
	if config.Timeout <= 0 || config.Timeout > time.Minute {
		return nil, errors.New("SMTP timeout must be positive and no greater than one minute")
	}
	transport := strings.TrimSpace(config.Transport)
	if transport == "" {
		transport = SMTPTransportPrivatePlaintext
	}
	var tlsConfig *tls.Config
	switch transport {
	case SMTPTransportPrivatePlaintext:
		if !config.PrivateNetwork {
			return nil, errors.New("plaintext SMTP credentials require an explicitly private network")
		}
	case SMTPTransportStartTLS, SMTPTransportImplicitTLS:
		if config.TLSConfig != nil {
			tlsConfig = config.TLSConfig.Clone()
		} else {
			tlsConfig = &tls.Config{}
		}
		if tlsConfig.InsecureSkipVerify {
			return nil, errors.New("SMTP TLS certificate verification cannot be disabled")
		}
		serverName := strings.TrimSpace(config.TLSServerName)
		if serverName == "" {
			serverName = host
		}
		if net.ParseIP(serverName) == nil && strings.ContainsAny(serverName, " /@") {
			return nil, errors.New("SMTP TLS server name is invalid")
		}
		tlsConfig.ServerName = serverName
		if tlsConfig.MinVersion == 0 || tlsConfig.MinVersion < tls.VersionTLS12 {
			tlsConfig.MinVersion = tls.VersionTLS12
		}
	default:
		return nil, errors.New("SMTP transport must be private-plaintext, starttls, or implicit-tls")
	}
	return &SMTPSender{
		address: address, host: host, from: from.Address,
		username: strings.TrimSpace(config.Username),
		password: config.Password, timeout: config.Timeout,
		transport: transport, tlsConfig: tlsConfig,
	}, nil
}

func (sender *SMTPSender) Deliver(
	ctx context.Context,
	delivery EmailDelivery,
) error {
	if sender == nil {
		return NewPermanentDeliveryFailure("SMTP_NOT_CONFIGURED")
	}
	if _, err := mail.ParseAddress(delivery.RecipientEmail); err != nil {
		return NewPermanentDeliveryFailure("SMTP_RECIPIENT_INVALID")
	}
	rendered, err := renderDelivery(delivery)
	if err != nil {
		return NewPermanentDeliveryFailure("EMAIL_TEMPLATE_INVALID")
	}
	message, err := buildSMTPMessage(sender.from, delivery, rendered)
	if err != nil {
		return NewPermanentDeliveryFailure("EMAIL_MESSAGE_INVALID")
	}
	if len(message) > maximumEmailMessageBytes {
		return NewPermanentDeliveryFailure("EMAIL_MESSAGE_TOO_LARGE")
	}

	deadline := time.Now().Add(sender.timeout)
	if contextDeadline, ok := ctx.Deadline(); ok && contextDeadline.Before(deadline) {
		deadline = contextDeadline
	}
	dialer := net.Dialer{Timeout: time.Until(deadline)}
	connection, err := dialer.DialContext(ctx, "tcp", sender.address)
	if err != nil {
		return classifySMTPFailure("SMTP_UNAVAILABLE", err, false)
	}
	defer connection.Close()
	if err := connection.SetDeadline(deadline); err != nil {
		return classifySMTPFailure("SMTP_UNAVAILABLE", err, false)
	}
	if sender.transport == SMTPTransportImplicitTLS {
		secureConnection := tls.Client(connection, sender.tlsConfig.Clone())
		if err := secureConnection.HandshakeContext(ctx); err != nil {
			return classifySMTPFailure("SMTP_TLS_VERIFICATION_FAILED", err, true)
		}
		connection = secureConnection
	}
	client, err := smtp.NewClient(connection, sender.host)
	if err != nil {
		return classifySMTPFailure("SMTP_PROTOCOL_ERROR", err, false)
	}
	defer client.Close()
	if sender.transport == SMTPTransportStartTLS {
		if supported, _ := client.Extension("STARTTLS"); !supported {
			return NewPermanentDeliveryFailure("SMTP_STARTTLS_REQUIRED")
		}
		if err := client.StartTLS(sender.tlsConfig.Clone()); err != nil {
			return classifySMTPFailure("SMTP_TLS_VERIFICATION_FAILED", err, true)
		}
	}
	if err := client.Auth(privatePlainAuth{
		username: sender.username,
		password: sender.password,
	}); err != nil {
		return classifySMTPFailure("SMTP_AUTHENTICATION_FAILED", err, true)
	}
	if err := client.Mail(sender.from); err != nil {
		return classifySMTPFailure("SMTP_SENDER_REJECTED", err, true)
	}
	if err := client.Rcpt(delivery.RecipientEmail); err != nil {
		return classifySMTPFailure("SMTP_RECIPIENT_REJECTED", err, true)
	}
	writer, err := client.Data()
	if err != nil {
		return classifySMTPFailure("SMTP_DATA_REJECTED", err, false)
	}
	if _, err := writer.Write(message); err != nil {
		_ = writer.Close()
		return classifySMTPFailure("SMTP_WRITE_FAILED", err, false)
	}
	if err := writer.Close(); err != nil {
		return classifySMTPFailure("SMTP_DATA_REJECTED", err, false)
	}
	_ = client.Quit()
	return nil
}

func renderDelivery(delivery EmailDelivery) (RenderedEmail, error) {
	switch delivery.RecipientAudience {
	case EmailAudienceAuditee:
		return RenderAuditeeEmail(AuditeeTemplateData{
			Title: delivery.Title, Summary: delivery.Body,
			OrganizationName:  delivery.OrganizationName,
			RelatedRecordType: delivery.RelatedEntityType,
			RelatedRecordID:   delivery.RelatedEntityID,
		})
	case EmailAudienceCAA:
		return RenderInternalCAAEmail(InternalCAATemplateData{
			Title: delivery.Title, Summary: delivery.Body,
			InternalContext:   delivery.InternalContext,
			RelatedRecordType: delivery.RelatedEntityType,
			RelatedRecordID:   delivery.RelatedEntityID,
		})
	default:
		return RenderedEmail{}, errors.New("email audience is invalid")
	}
}

func buildSMTPMessage(
	from string,
	delivery EmailDelivery,
	rendered RenderedEmail,
) ([]byte, error) {
	if strings.TrimSpace(delivery.ProviderMessageID) == "" ||
		strings.ContainsAny(delivery.ProviderMessageID, "\r\n") {
		return nil, errors.New("provider message ID is invalid")
	}
	boundaryHash := sha256.Sum256([]byte(delivery.JobID))
	boundary := "avia-" + fmt.Sprintf("%x", boundaryHash[:12])
	var message bytes.Buffer
	for _, header := range []string{
		"From: " + from,
		"To: " + delivery.RecipientEmail,
		"Message-ID: " + delivery.ProviderMessageID,
		"Subject: " + rendered.Subject,
		"MIME-Version: 1.0",
		`Content-Type: multipart/alternative; boundary="` + boundary + `"`,
	} {
		message.WriteString(header)
		message.WriteString("\r\n")
	}
	message.WriteString("\r\n--")
	message.WriteString(boundary)
	message.WriteString("\r\nContent-Type: text/plain; charset=utf-8\r\n")
	message.WriteString("Content-Transfer-Encoding: 8bit\r\n\r\n")
	message.WriteString(normalizeCRLF(rendered.Text))
	message.WriteString("\r\n--")
	message.WriteString(boundary)
	message.WriteString("\r\nContent-Type: text/html; charset=utf-8\r\n")
	message.WriteString("Content-Transfer-Encoding: 8bit\r\n\r\n")
	message.WriteString(normalizeCRLF(rendered.HTML))
	message.WriteString("\r\n--")
	message.WriteString(boundary)
	message.WriteString("--\r\n")
	return message.Bytes(), nil
}

func normalizeCRLF(value string) string {
	normalized := strings.ReplaceAll(value, "\r\n", "\n")
	normalized = strings.ReplaceAll(normalized, "\r", "\n")
	return strings.ReplaceAll(normalized, "\n", "\r\n")
}

type privatePlainAuth struct {
	username string
	password string
}

func (auth privatePlainAuth) Start(
	_ *smtp.ServerInfo,
) (string, []byte, error) {
	return "PLAIN", []byte("\x00" + auth.username + "\x00" + auth.password), nil
}

func (auth privatePlainAuth) Next(
	_ []byte,
	more bool,
) ([]byte, error) {
	if more {
		return nil, errors.New("SMTP PLAIN authentication challenge is unsupported")
	}
	return nil, nil
}

type DeliveryFailure struct {
	code      string
	permanent bool
}

func (failure *DeliveryFailure) Error() string {
	return failure.code
}

func NewPermanentDeliveryFailure(code string) error {
	return &DeliveryFailure{code: canonicalFailureCode(code), permanent: true}
}

func newRetryableDeliveryFailure(code string) error {
	return &DeliveryFailure{code: canonicalFailureCode(code)}
}

func IsPermanentDeliveryFailure(err error) bool {
	var failure *DeliveryFailure
	return errors.As(err, &failure) && failure.permanent
}

func DeliveryFailureCode(err error) string {
	var failure *DeliveryFailure
	if errors.As(err, &failure) {
		return failure.code
	}
	return "SMTP_DELIVERY_FAILED"
}

func canonicalFailureCode(code string) string {
	code = strings.TrimSpace(code)
	if code == "" || len(code) > 64 {
		return "SMTP_DELIVERY_FAILED"
	}
	for _, character := range code {
		if !(character == '_' ||
			character >= 'A' && character <= 'Z' ||
			character >= '0' && character <= '9') {
			return "SMTP_DELIVERY_FAILED"
		}
	}
	return code
}

func classifySMTPFailure(
	code string,
	err error,
	permanentByDefault bool,
) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return newRetryableDeliveryFailure(code + "_TIMEOUT")
	}
	var networkError net.Error
	if errors.As(err, &networkError) && networkError.Timeout() {
		return newRetryableDeliveryFailure(code + "_TIMEOUT")
	}
	var protocolError *textproto.Error
	if errors.As(err, &protocolError) {
		if protocolError.Code >= 500 {
			return NewPermanentDeliveryFailure(code)
		}
		if protocolError.Code >= 400 {
			return newRetryableDeliveryFailure(code)
		}
	}
	if permanentByDefault {
		return NewPermanentDeliveryFailure(code)
	}
	return newRetryableDeliveryFailure(code)
}

func stableProviderMessageID(jobID string) string {
	digest := sha256.Sum256([]byte(strings.TrimSpace(jobID)))
	return "<notification-" + strconv.FormatUint(
		uint64(digest[0])<<56|
			uint64(digest[1])<<48|
			uint64(digest[2])<<40|
			uint64(digest[3])<<32|
			uint64(digest[4])<<24|
			uint64(digest[5])<<16|
			uint64(digest[6])<<8|
			uint64(digest[7]),
		16,
	) + "@aviasurveil360.local>"
}
