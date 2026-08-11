package telemetry

import (
	"io"
	"log/slog"
	"strings"
)

// NewRedactedLogger emits structured JSON and replaces sensitive attribute
// values even if a future caller accidentally uses a credential-shaped key.
// Callers still must avoid placing secret material in error strings.
func NewRedactedLogger(writer io.Writer) *slog.Logger {
	if writer == nil {
		writer = io.Discard
	}
	return slog.New(slog.NewJSONHandler(writer, &slog.HandlerOptions{
		ReplaceAttr: func(_ []string, attribute slog.Attr) slog.Attr {
			if isSensitiveKey(attribute.Key) {
				return slog.String(attribute.Key, "[REDACTED]")
			}
			return attribute
		},
	}))
}

func isSensitiveKey(key string) bool {
	normalized := strings.ToLower(strings.ReplaceAll(key, "-", "_"))
	for _, marker := range []string{
		"password",
		"secret",
		"token",
		"private_key",
		"signing_key",
		"mfa",
		"recovery_code",
		"authorization",
		"cookie",
	} {
		if strings.Contains(normalized, marker) {
			return true
		}
	}
	return false
}
