package httpapi

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	aga "github.com/MarlonJD/aviaSurveil360/apps/api/internal/agacandidatedemo"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/identity"
)

func TestAGACandidateDemoDeniesBeforeQueryParsingOrReaderAccess(t *testing.T) {
	reader := &agaHTTPReader{}
	api := NewCanonicalAPI(CanonicalAPIDependencies{AGACandidateDemo: aga.NewService(reader)})
	request := httptest.NewRequest(http.MethodGet, "/v1/admin/governed-checklist/aga-candidate-demo/forms?limit=not-a-number", nil)
	request = request.WithContext(context.WithValue(request.Context(), principalContextKey{}, identity.Principal{OrganizationID: "OTHER", Roles: []identity.Role{identity.RoleAdmin}}))
	response := httptest.NewRecorder()
	api.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusNotFound || reader.calls != 0 {
		t.Fatalf("denial must precede parsing and reader access: %d/%d", response.Code, reader.calls)
	}
	if response.Header().Get("Cache-Control") != "private, no-store" || response.Header().Get("Vary") != "Cookie" || response.Header().Get("Content-Length") != "21" {
		t.Fatalf("missing no-store neutral response headers: %#v", response.Header())
	}
}

func TestAGACandidateDemoDenialMatrixIsByteAndHeaderNeutral(t *testing.T) {
	reader := &agaHTTPReader{}
	api := NewCanonicalAPI(CanonicalAPIDependencies{AGACandidateDemo: aga.NewService(reader)})
	paths := []string{
		"/v1/admin/governed-checklist/aga-candidate-demo/capability",
		"/v1/admin/governed-checklist/aga-candidate-demo/summary",
		"/v1/admin/governed-checklist/aga-candidate-demo/forms?limit=invalid",
		"/v1/admin/governed-checklist/aga-candidate-demo/forms/FSS-AGA-FORM-999",
		"/v1/admin/governed-checklist/aga-candidate-demo/questions?riskBand=INVALID",
	}
	actors := []identity.Principal{
		{},
		{OrganizationID: "CAA", Roles: []identity.Role{identity.RoleInspector}},
		{OrganizationID: "CAA", Roles: []identity.Role{identity.RoleLeadInspector}},
		{OrganizationID: "CAA", Roles: []identity.Role{identity.RoleDepartmentManager}},
		{OrganizationID: "CAA", Roles: []identity.Role{identity.RoleFinance}},
		{OrganizationID: "CAA", Roles: []identity.Role{identity.RoleGeneralManager}},
		{OrganizationID: "CAA", Roles: []identity.Role{identity.RoleExecutiveDirector}},
		{OrganizationID: "OTHER", Roles: []identity.Role{identity.RoleAdmin}},
		{OrganizationID: "ORG-AUDITEE", Roles: []identity.Role{identity.RoleAuditee}},
	}
	var expectedBody string
	var expectedHeaders http.Header
	for _, path := range paths {
		for _, actor := range actors {
			request := httptest.NewRequest(http.MethodGet, path, nil)
			if actor.OrganizationID != "" || len(actor.Roles) != 0 {
				request = request.WithContext(context.WithValue(request.Context(), principalContextKey{}, actor))
			}
			response := httptest.NewRecorder()
			api.Handler().ServeHTTP(response, request)
			if response.Code != http.StatusNotFound {
				t.Fatalf("%s/%#v status=%d", path, actor, response.Code)
			}
			body, err := io.ReadAll(response.Result().Body)
			if err != nil {
				t.Fatal(err)
			}
			if expectedBody == "" {
				expectedBody, expectedHeaders = string(body), response.Header().Clone()
			} else if string(body) != expectedBody || response.Header().Get("Cache-Control") != expectedHeaders.Get("Cache-Control") || response.Header().Get("Pragma") != expectedHeaders.Get("Pragma") || response.Header().Get("Vary") != expectedHeaders.Get("Vary") || response.Header().Get("Content-Length") != expectedHeaders.Get("Content-Length") {
				t.Fatalf("%s/%#v is distinguishable: body=%q headers=%#v", path, actor, body, response.Header())
			}
		}
	}
	if reader.calls != 0 {
		t.Fatalf("denial matrix accessed reader %d times", reader.calls)
	}
}

func TestAGACandidateDemoAdminGetsSealedSummary(t *testing.T) {
	reader := &agaHTTPReader{}
	api := NewCanonicalAPI(CanonicalAPIDependencies{AGACandidateDemo: aga.NewService(reader)})
	request := httptest.NewRequest(http.MethodGet, "/v1/admin/governed-checklist/aga-candidate-demo/summary", nil)
	request = request.WithContext(context.WithValue(request.Context(), principalContextKey{}, identity.Principal{OrganizationID: "CAA", Roles: []identity.Role{identity.RoleAdmin}}))
	response := httptest.NewRecorder()
	api.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK || reader.calls != 1 {
		t.Fatalf("admin summary=%d calls=%d", response.Code, reader.calls)
	}
}

type agaHTTPReader struct{ calls int }

func (reader *agaHTTPReader) Capability(context.Context) (aga.Capability, error) {
	reader.calls++
	return aga.Capability{Available: true}, nil
}
func (reader *agaHTTPReader) Summary(context.Context) (aga.Summary, error) {
	reader.calls++
	return aga.Summary{FormCount: 52, QuestionCount: 1310}, nil
}
func (reader *agaHTTPReader) Forms(context.Context, string, int) (aga.Page[aga.Form], error) {
	reader.calls++
	return aga.Page[aga.Form]{}, nil
}
func (reader *agaHTTPReader) Form(context.Context, string) (aga.Form, error) {
	reader.calls++
	return aga.Form{}, nil
}
func (reader *agaHTTPReader) Questions(context.Context, string, string, string, string, int) (aga.Page[aga.Question], error) {
	reader.calls++
	return aga.Page[aga.Question]{}, nil
}
