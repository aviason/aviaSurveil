//go:build canonicaltest

package httpapi_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aviason/aviaSurveil/internal/httpapi"
)

func TestCanonicalTestBoundaryUsesOnlyServerOwnedPrincipalMapping(t *testing.T) {
	boundary := httpapi.NewCanonicalTestBoundary("canonical-test-token")
	protected := boundary.Protect(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		principal, ok := httpapi.PrincipalFromContext(request.Context())
		if !ok {
			t.Fatal("principal missing from protected request")
		}
		if principal.SubjectID != "USR-AUDITEE-FLY" || principal.OrganizationID != "ORG-FLY-NAMIBIA" || len(principal.Roles) != 1 {
			t.Fatalf("server principal = %+v", principal)
		}
		writer.WriteHeader(http.StatusNoContent)
	}))

	request := httptest.NewRequest(http.MethodPost, "/v1/caps", nil)
	request.Header.Set(httpapi.CanonicalTestTokenHeader, "canonical-test-token")
	request.Header.Set(httpapi.CanonicalTestSubjectHeader, "USR-AUDITEE-FLY")
	request.Header.Set("X-Avia-Test-Role", "manager")
	request.Header.Set("X-Avia-Test-Organization", "ORG-SKYCARGO")
	response := httptest.NewRecorder()
	protected.ServeHTTP(response, request)
	if response.Code != http.StatusNoContent {
		t.Fatalf("protected response = %d: %s", response.Code, response.Body.String())
	}
}

func TestCanonicalTestBoundaryFailsClosed(t *testing.T) {
	boundary := httpapi.NewCanonicalTestBoundary("canonical-test-token")
	protected := boundary.Protect(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("protected handler must not run")
	}))
	for name, test := range map[string]struct {
		token, subject string
		expected       int
	}{
		"missing token":   {subject: "USR-AUDITEE-FLY", expected: http.StatusUnauthorized},
		"invalid token":   {token: "wrong", subject: "USR-AUDITEE-FLY", expected: http.StatusUnauthorized},
		"unknown subject": {token: "canonical-test-token", subject: "attacker", expected: http.StatusForbidden},
	} {
		t.Run(name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/v1/findings", nil)
			request.Header.Set(httpapi.CanonicalTestTokenHeader, test.token)
			request.Header.Set(httpapi.CanonicalTestSubjectHeader, test.subject)
			response := httptest.NewRecorder()
			protected.ServeHTTP(response, request)
			if response.Code != test.expected {
				t.Fatalf("status = %d, want %d", response.Code, test.expected)
			}
		})
	}
}
