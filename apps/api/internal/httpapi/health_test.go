package httpapi_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aviason/aviaSurveil/internal/httpapi"
)

type readinessFunc func(context.Context) error

func (function readinessFunc) Ready(ctx context.Context) error {
	return function(ctx)
}

type readinessReporter struct {
	report httpapi.ReadinessReport
}

func (reporter readinessReporter) Ready(context.Context) error {
	if reporter.report.Status == httpapi.ReadinessStatusNotReady {
		return errors.New("required dependency unavailable")
	}
	return nil
}

func (reporter readinessReporter) Readiness(context.Context) httpapi.ReadinessReport {
	return reporter.report
}

func TestLivenessDoesNotDependOnPostgreSQL(t *testing.T) {
	t.Parallel()

	handler := httpapi.NewHealthHandler(readinessFunc(func(context.Context) error {
		return errors.New("database unavailable")
	}))
	request := httptest.NewRequest(http.MethodGet, "/health/live", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	assertJSONStatus(t, response, "ok")
}

func TestReadinessFailsClosedWhenPostgreSQLIsUnavailable(t *testing.T) {
	t.Parallel()

	handler := httpapi.NewHealthHandler(readinessFunc(func(context.Context) error {
		return errors.New("database unavailable")
	}))
	request := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusServiceUnavailable)
	}
	if contentType := response.Header().Get("Content-Type"); contentType != "application/problem+json" {
		t.Fatalf("Content-Type = %q", contentType)
	}
}

func TestReadinessSucceedsWhenRequiredDependenciesAreReady(t *testing.T) {
	t.Parallel()

	handler := httpapi.NewHealthHandler(readinessFunc(func(context.Context) error { return nil }))
	request := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	assertJSONStatus(t, response, "ok")
}

func TestReadinessReportsNamedRequiredDependenciesWithoutLeakingErrors(t *testing.T) {
	t.Parallel()

	handler := httpapi.NewHealthHandler(readinessReporter{
		report: httpapi.ReadinessReport{
			Status: httpapi.ReadinessStatusNotReady,
			Dependencies: []httpapi.DependencyReadiness{
				{Name: "postgresql", Required: true, Status: httpapi.DependencyStatusUnavailable},
				{Name: "minio", Required: true, Status: httpapi.DependencyStatusReady},
			},
		},
	})
	request := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusServiceUnavailable)
	}
	body := response.Body.String()
	for _, expected := range []string{`"status":"not_ready"`, `"name":"postgresql"`, `"status":"unavailable"`} {
		if !strings.Contains(body, expected) {
			t.Fatalf("readiness body omitted %q: %s", expected, body)
		}
	}
	for _, forbidden := range []string{"password", "secret", "postgres://", "required dependency unavailable"} {
		if strings.Contains(strings.ToLower(body), forbidden) {
			t.Fatalf("readiness body leaked %q: %s", forbidden, body)
		}
	}
}

func TestOptionalDeliveryFailureIsDegradedButReady(t *testing.T) {
	t.Parallel()

	handler := httpapi.NewHealthHandler(readinessReporter{
		report: httpapi.ReadinessReport{
			Status: httpapi.ReadinessStatusDegraded,
			Dependencies: []httpapi.DependencyReadiness{
				{Name: "postgresql", Required: true, Status: httpapi.DependencyStatusReady},
				{Name: "mailpit", Required: false, Status: httpapi.DependencyStatusUnavailable},
			},
		},
	})
	request := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	body := response.Body.String()
	if !strings.Contains(body, `"status":"degraded"`) ||
		!strings.Contains(body, `"name":"mailpit"`) {
		t.Fatalf("degraded readiness body = %s", body)
	}
}

func assertJSONStatus(t *testing.T, response *httptest.ResponseRecorder, expected string) {
	t.Helper()
	if contentType := response.Header().Get("Content-Type"); contentType != "application/json" {
		t.Fatalf("Content-Type = %q", contentType)
	}
	var body struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Status != expected {
		t.Fatalf("status body = %q, want %q", body.Status, expected)
	}
}
