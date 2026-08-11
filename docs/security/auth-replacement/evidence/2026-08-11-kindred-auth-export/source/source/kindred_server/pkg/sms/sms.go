package sms

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"kindred_server/pkg/logger"
)

// Sender sends transactional SMS messages. Dev/test use LogSender; deployed
// environments can provide an HTTP webhook without binding the app to one SMS
// vendor SDK.
type Sender interface {
	SendPhoneVerification(ctx context.Context, to, code string) error
}

type LogSender struct {
	log *logger.Logger
}

func NewLogSender(log *logger.Logger) *LogSender {
	return &LogSender{log: log}
}

func (s *LogSender) SendPhoneVerification(_ context.Context, to, code string) error {
	s.log.Info("phone verification dispatched", map[string]any{
		"to":   to,
		"code": code,
	})
	return nil
}

type HTTPSender struct {
	url    string
	token  string
	client *http.Client
	log    *logger.Logger
}

func NewHTTPSender(url, token string, log *logger.Logger) *HTTPSender {
	return &HTTPSender{
		url:    url,
		token:  token,
		client: &http.Client{Timeout: 10 * time.Second},
		log:    log,
	}
}

func (s *HTTPSender) SendPhoneVerification(ctx context.Context, to, code string) error {
	body := map[string]string{
		"to":      to,
		"message": fmt.Sprintf("Kindred verification code: %s", code),
		"code":    code,
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.url, bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if s.token != "" {
		req.Header.Set("Authorization", "Bearer "+s.token)
	}
	resp, err := s.client.Do(req)
	if err != nil {
		s.log.Error("failed to send sms", err, map[string]any{"to": to})
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		err := fmt.Errorf("sms provider returned status %d", resp.StatusCode)
		s.log.Error("failed to send sms", err, map[string]any{"to": to})
		return err
	}
	s.log.Info("sms sent successfully", map[string]any{"to": to})
	return nil
}
