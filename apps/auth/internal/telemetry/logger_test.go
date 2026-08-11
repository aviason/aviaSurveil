package telemetry

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
)

func TestNewRedactedLoggerDoesNotEmitCredentialAttributes(t *testing.T) {
	var output bytes.Buffer
	logger := NewRedactedLogger(&output)
	logger.Info("auth scaffold", slog.String("password", "never-log-this"), slog.String("issuer_host", "127.0.0.1"))
	if strings.Contains(output.String(), "never-log-this") {
		t.Fatalf("logger emitted secret: %s", output.String())
	}
	if !strings.Contains(output.String(), "REDACTED") || !strings.Contains(output.String(), "127.0.0.1") {
		t.Fatalf("redaction or safe attribute missing: %s", output.String())
	}
}
