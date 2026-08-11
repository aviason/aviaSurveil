package mail

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/mail"
	"net/smtp"
	"regexp"
	"strings"
	"time"
)

type TLSMode string

const (
	TLSModeStartTLS TLSMode = "starttls"
	TLSModeImplicit TLSMode = "implicit-tls"
)

var hostPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9.-]*$`)

type Config struct {
	Address   string
	Host      string
	From      string
	Username  string
	Password  string
	TLSMode   TLSMode
	Timeout   time.Duration
	TLSConfig *tls.Config
}

type Sender struct {
	address  string
	host     string
	from     mail.Address
	username string
	password string
	mode     TLSMode
	timeout  time.Duration
	tls      *tls.Config
}

func NewSender(configuration Config) (*Sender, error) {
	if strings.TrimSpace(configuration.Address) == "" || strings.TrimSpace(configuration.Host) == "" {
		return nil, errors.New("SMTP address and host are required")
	}
	if _, _, err := net.SplitHostPort(configuration.Address); err != nil {
		return nil, errors.New("SMTP address must contain host and port")
	}
	if !hostPattern.MatchString(configuration.Host) {
		return nil, errors.New("SMTP host is invalid")
	}
	parsedFrom, err := mail.ParseAddress(configuration.From)
	if err != nil || parsedFrom.Address != configuration.From || strings.ContainsAny(configuration.From, "\r\n") {
		return nil, errors.New("SMTP from address is invalid")
	}
	if strings.TrimSpace(configuration.Username) == "" || configuration.Password == "" {
		return nil, errors.New("SMTP username and password are required")
	}
	if configuration.Timeout <= 0 || configuration.Timeout > time.Minute {
		configuration.Timeout = 30 * time.Second
	}
	switch configuration.TLSMode {
	case TLSModeStartTLS, TLSModeImplicit:
	default:
		return nil, errors.New("SMTP plaintext transport is forbidden")
	}
	tlsConfig := &tls.Config{ServerName: configuration.Host, MinVersion: tls.VersionTLS12}
	if configuration.TLSConfig != nil {
		if configuration.TLSConfig.InsecureSkipVerify {
			return nil, errors.New("SMTP TLS certificate verification cannot be disabled")
		}
		copyConfig := configuration.TLSConfig.Clone()
		if copyConfig.ServerName == "" {
			copyConfig.ServerName = configuration.Host
		}
		if copyConfig.MinVersion < tls.VersionTLS12 {
			copyConfig.MinVersion = tls.VersionTLS12
		}
		tlsConfig = copyConfig
	}
	return &Sender{address: configuration.Address, host: configuration.Host, from: *parsedFrom, username: configuration.Username, password: configuration.Password, mode: configuration.TLSMode, timeout: configuration.Timeout, tls: tlsConfig}, nil
}

func (sender *Sender) Send(ctx context.Context, recipient, subject, body string) error {
	parsedRecipient, err := mail.ParseAddress(recipient)
	if err != nil || parsedRecipient.Address != recipient || strings.ContainsAny(recipient, "\r\n") {
		return errors.New("SMTP recipient is invalid")
	}
	if strings.TrimSpace(subject) == "" || strings.ContainsAny(subject, "\r\n") {
		return errors.New("SMTP subject is invalid")
	}
	if strings.ContainsAny(body, "\x00") {
		return errors.New("SMTP body contains an invalid NUL")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	dialer := net.Dialer{Timeout: sender.timeout}
	var connection net.Conn
	if sender.mode == TLSModeImplicit {
		connection, err = tls.DialWithDialer(&dialer, "tcp", sender.address, sender.tls)
	} else {
		connection, err = dialer.DialContext(ctx, "tcp", sender.address)
	}
	if err != nil {
		return fmt.Errorf("SMTP connect: %w", err)
	}
	defer connection.Close()
	_ = connection.SetDeadline(time.Now().Add(sender.timeout))
	client, err := smtp.NewClient(connection, sender.host)
	if err != nil {
		return fmt.Errorf("SMTP protocol: %w", err)
	}
	defer client.Close()
	if sender.mode == TLSModeStartTLS {
		supported, _ := client.Extension("STARTTLS")
		if !supported {
			return errors.New("SMTP server does not advertise STARTTLS")
		}
		if err := client.StartTLS(sender.tls); err != nil {
			return fmt.Errorf("SMTP STARTTLS: %w", err)
		}
	}
	if err := client.Auth(smtp.PlainAuth("", sender.username, sender.password, sender.host)); err != nil {
		return fmt.Errorf("SMTP authentication: %w", err)
	}
	if err := client.Mail(sender.from.Address); err != nil {
		return fmt.Errorf("SMTP sender: %w", err)
	}
	if err := client.Rcpt(parsedRecipient.Address); err != nil {
		return fmt.Errorf("SMTP recipient: %w", err)
	}
	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("SMTP data: %w", err)
	}
	message := bytes.NewBuffer(nil)
	fmt.Fprintf(message, "From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=utf-8\r\n\r\n%s\r\n", sender.from.String(), parsedRecipient.String(), subject, body)
	if _, err := message.WriteTo(writer); err != nil {
		_ = writer.Close()
		return fmt.Errorf("SMTP write: %w", err)
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("SMTP data close: %w", err)
	}
	if err := client.Quit(); err != nil {
		return fmt.Errorf("SMTP quit: %w", err)
	}
	return nil
}
