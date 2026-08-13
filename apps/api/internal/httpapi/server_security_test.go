package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestApplicationHandlerAddsAPIAppropriateSecurityHeaders(t *testing.T) {
	t.Parallel()
	handler := NewApplicationHandler(readinessStub(func(context.Context) error { return nil }), nil, nil, nil)
	request := httptest.NewRequest(http.MethodGet, "/health/live", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d", response.Code)
	}
	expected := map[string]string{
		"Content-Security-Policy": "default-src 'none'; base-uri 'none'; frame-ancestors 'none'; form-action 'none'",
		"Permissions-Policy":      "camera=(), microphone=(), geolocation=()",
		"Referrer-Policy":         "no-referrer",
		"X-Content-Type-Options":  "nosniff",
		"X-Frame-Options":         "DENY",
		"Cache-Control":           "no-store",
	}
	for header, value := range expected {
		if actual := response.Header().Get(header); actual != value {
			t.Errorf("%s = %q, want %q", header, actual, value)
		}
	}
}

func TestApplicationRateLimitIsBoundedByClassAndUntrustedRemoteAddress(t *testing.T) {
	now := time.Date(2026, time.July, 21, 12, 0, 0, 0, time.UTC)
	api := http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusNoContent)
	})
	auth := http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusNoContent)
	})
	handler := newApplicationHandler(
		readinessStub(func(context.Context) error { return nil }),
		auth,
		api,
		nil,
		applicationSecurityOptions{
			clock:                     func() time.Time { return now },
			window:                    time.Minute,
			mutationRequestsPerWindow: 2,
		},
	)

	assertAllowed := func(method, target string) {
		t.Helper()
		request := httptest.NewRequest(method, target, nil)
		request.RemoteAddr = "192.0.2.10:4242"
		request.Header.Set("X-Forwarded-For", "198.51.100.99")
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code == http.StatusTooManyRequests {
			t.Fatalf("%s %s was limited early", method, target)
		}
	}

	assertLimited := func(method, target string) {
		t.Helper()
		request := httptest.NewRequest(method, target, nil)
		request.RemoteAddr = "192.0.2.10:4242"
		request.Header.Set("X-Forwarded-For", "203.0.113.55")
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusTooManyRequests || response.Header().Get("Retry-After") != "60" {
			t.Fatalf("%s %s response = %d Retry-After %q", method, target, response.Code, response.Header().Get("Retry-After"))
		}
		var problem struct {
			Code string `json:"code"`
		}
		if err := json.Unmarshal(response.Body.Bytes(), &problem); err != nil || problem.Code != "RATE_LIMITED" {
			t.Fatalf("problem = %s, err = %v", response.Body.String(), err)
		}
	}

	assertAllowed(http.MethodPost, "/v1/findings/FND-001/authorized-closure")
	assertAllowed(http.MethodPost, "/v1/findings/FND-002/authorized-closure")
	assertLimited(http.MethodPost, "/v1/findings/FND-003/authorized-closure")

	// Read operations use a distinct class and are not consumed by mutation traffic.
	assertAllowed(http.MethodGet, "/v1/findings")
	assertAllowed(http.MethodGet, "/v1/findings")

	// Login admission is durable and browser-bound; the application limiter does
	// not key it by the shared gateway socket peer.
	assertAllowed(http.MethodGet, "/auth/login")
	assertAllowed(http.MethodGet, "/auth/login")

	// The limiter keys the socket peer and never trusts spoofable forwarding headers.
	request := httptest.NewRequest(http.MethodPost, "/v1/findings/FND-004/authorized-closure", strings.NewReader("{}"))
	request.RemoteAddr = "192.0.2.11:5252"
	request.Header.Set("X-Forwarded-For", "192.0.2.10")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code == http.StatusTooManyRequests {
		t.Fatal("a different socket peer incorrectly shared the first peer's rate bucket")
	}

	now = now.Add(time.Minute)
	assertAllowed(http.MethodPost, "/v1/findings/FND-005/authorized-closure")
}

type readinessStub func(context.Context) error

func (stub readinessStub) Ready(ctx context.Context) error { return stub(ctx) }
