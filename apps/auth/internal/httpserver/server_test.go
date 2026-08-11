package httpserver

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/MarlonJD/aviaSurveil360/apps/auth/internal/config"
)

func TestTask2HealthBoundaryIsLivenessOnly(t *testing.T) {
	settings := config.Settings{
		HTTPAddress:       "127.0.0.1:18081",
		ReadHeaderTimeout: 5,
		ReadTimeout:       15,
		WriteTimeout:      15,
		IdleTimeout:       60,
		MaxHeaderBytes:    32 << 10,
	}
	server := New(settings, slog.New(slog.NewTextHandler(io.Discard, nil)))

	request := httptest.NewRequest(http.MethodGet, "/health/live", nil)
	recording := httptest.NewRecorder()
	server.httpServer.Handler.ServeHTTP(recording, request)
	if recording.Code != http.StatusOK || !strings.Contains(recording.Body.String(), `"status":"alive"`) {
		t.Fatalf("liveness = %d %s", recording.Code, recording.Body.String())
	}

	request = httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	recording = httptest.NewRecorder()
	server.httpServer.Handler.ServeHTTP(recording, request)
	if recording.Code != http.StatusServiceUnavailable || !strings.Contains(recording.Body.String(), notReadyReason) {
		t.Fatalf("readiness = %d %s", recording.Code, recording.Body.String())
	}

	request = httptest.NewRequest(http.MethodGet, "/.well-known/openid-configuration", nil)
	recording = httptest.NewRecorder()
	server.httpServer.Handler.ServeHTTP(recording, request)
	if recording.Code != http.StatusNotFound {
		t.Fatalf("unimplemented discovery status = %d, want 404", recording.Code)
	}
}

func TestTask2ReadinessCanOnlyBeOpenedExplicitly(t *testing.T) {
	settings := config.Settings{
		HTTPAddress:       "127.0.0.1:18082",
		ReadHeaderTimeout: 5,
		ReadTimeout:       15,
		WriteTimeout:      15,
		IdleTimeout:       60,
		MaxHeaderBytes:    32 << 10,
	}
	server := New(settings, slog.New(slog.NewTextHandler(io.Discard, nil)))
	server.Ready()
	recording := httptest.NewRecorder()
	server.httpServer.Handler.ServeHTTP(recording, httptest.NewRequest(http.MethodGet, "/health/ready", nil))
	if recording.Code != http.StatusOK {
		t.Fatalf("explicit readiness = %d, want 200", recording.Code)
	}
}
