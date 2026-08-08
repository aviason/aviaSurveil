//go:build preproddemo

package httpapi

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	aga "github.com/MarlonJD/aviaSurveil360/apps/api/internal/agaapplicability"
	workspace "github.com/MarlonJD/aviaSurveil360/apps/api/internal/agademoworkspace"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/identity"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/platform/session"
	preprod "github.com/MarlonJD/aviaSurveil360/apps/api/internal/preproddata/agademoworkspace"
)

type workspaceSessionDouble struct {
	principal identity.Principal
	csrfCalls int
}

func (sessions *workspaceSessionDouble) NewLoginState(context.Context, string) (session.LoginRequest, error) {
	return session.LoginRequest{}, nil
}
func (sessions *workspaceSessionDouble) ConsumeLoginState(context.Context, string) (session.LoginState, error) {
	return session.LoginState{}, nil
}
func (sessions *workspaceSessionDouble) Create(context.Context, session.CreateInput) (session.BrowserSession, error) {
	return session.BrowserSession{}, nil
}
func (sessions *workspaceSessionDouble) Authenticate(context.Context, string) (identity.Principal, error) {
	return sessions.principal, nil
}
func (sessions *workspaceSessionDouble) ValidateCSRF(context.Context, string, string) error {
	sessions.csrfCalls++
	return nil
}
func (sessions *workspaceSessionDouble) Revoke(context.Context, string) error { return nil }

type workspaceHTTPStore struct {
	applyErr error
}

func (workspaceHTTPStore) LoadAndSeal(context.Context, preprod.LoadInput) (preprod.WorkspaceSealReceipt, error) {
	return preprod.WorkspaceSealReceipt{}, nil
}
func (workspaceHTTPStore) Snapshot(context.Context) (preprod.LoadedWorkspace, error) {
	return preprod.LoadedWorkspace{Generation: preprod.Generation{GenerationID: "aga-ws-generation-0001", State: preprod.GenerationActive, Revision: 1}}, nil
}
func (store workspaceHTTPStore) ApplyDraftCommand(context.Context, aga.DraftCommand) (aga.Draft, error) {
	return aga.Draft{}, store.applyErr
}
func (workspaceHTTPStore) AppendQuestionVersion(context.Context, preprod.AppendQuestionVersionInput) (preprod.WorkspaceQuestionVersion, error) {
	return preprod.WorkspaceQuestionVersion{}, nil
}
func (workspaceHTTPStore) PutIdempotencyResponse(context.Context, preprod.IdempotencyResponse) (preprod.IdempotencyResponse, bool, error) {
	return preprod.IdempotencyResponse{}, false, nil
}
func (workspaceHTTPStore) GetIdempotencyResponse(context.Context, preprod.StoredResponseKey) (preprod.IdempotencyResponse, bool, error) {
	return preprod.IdempotencyResponse{}, false, nil
}
func (workspaceHTTPStore) ResetGeneration(context.Context, preprod.ResetInput) (preprod.Generation, preprod.ResetTombstone, error) {
	return preprod.Generation{}, preprod.ResetTombstone{}, nil
}

type readCounter struct {
	read   bool
	closed bool
}

func (counter *readCounter) Read([]byte) (int, error) { counter.read = true; return 0, io.EOF }
func (counter *readCounter) Close() error             { counter.closed = true; return nil }

func workspaceHTTPService(subject, organization string, role identity.Role, operationRoles ...string) *workspace.Service {
	return workspace.NewService(workspace.ServiceConfig{
		Store: workspaceHTTPStore{applyErr: aga.ErrNonCurrentQuestion},
		Resolver: workspace.StaticBindingResolver{Bindings: map[string]preprod.AuthorityBinding{subject: {
			BindingID: "binding", SubjectSlot: subject, MembershipSlot: "membership", OrganizationID: organization, DepartmentID: "department", OrganizationalUnitID: "unit", OperationRoles: operationRoles, Active: true,
		}}},
	})
}

func TestWorkspaceProtectorAuthenticatesCSRFFirst(t *testing.T) {
	body := &readCounter{}
	sessions := &workspaceSessionDouble{principal: identity.Principal{SubjectID: "manager", OrganizationID: "ORG", Roles: []identity.Role{identity.RoleDepartmentManager}, SessionID: "session"}}
	boundary := NewAuthBoundary(nil, sessions)
	service := workspaceHTTPService("manager", "ORG", identity.RoleDepartmentManager, "manager")
	wrapped := ProtectAGADemoWorkspace(boundary, service, http.HandlerFunc(func(http.ResponseWriter, *http.Request) { t.Fatal("handler reached with invalid CSRF") }))
	request := httptest.NewRequest(http.MethodPost, "/v1/preprod/aga-demo-workspace/classification/query", body)
	request.AddCookie(&http.Cookie{Name: SessionCookieName, Value: "session"})
	response := httptest.NewRecorder()
	wrapped.ServeHTTP(response, request)
	if response.Code != http.StatusNotFound || response.Body.String() != `{"error":"not found"}` {
		t.Fatalf("invalid CSRF response = %d %q", response.Code, response.Body.String())
	}
	if body.read || sessions.csrfCalls != 0 {
		t.Fatalf("body read=%v csrf validation calls=%d; CSRF must precede parsing", body.read, sessions.csrfCalls)
	}
}

func TestWorkspaceAuthorizedMalformedBodyIsControlledAfterBroadAuthority(t *testing.T) {
	sessions := &workspaceSessionDouble{principal: identity.Principal{SubjectID: "manager", OrganizationID: "ORG", Roles: []identity.Role{identity.RoleDepartmentManager}, SessionID: "session"}}
	boundary := NewAuthBoundary(nil, sessions)
	service := workspaceHTTPService("manager", "ORG", identity.RoleDepartmentManager, "manager")
	wrapper := ProtectAGADemoWorkspace(boundary, service, NewAGADemoWorkspaceHandler(service))
	request := httptest.NewRequest(http.MethodPost, "/v1/preprod/aga-demo-workspace/classification/query", strings.NewReader(`{"unknown":true}`))
	request.AddCookie(&http.Cookie{Name: SessionCookieName, Value: "session"})
	request.AddCookie(&http.Cookie{Name: CSRFCookieName, Value: "csrf"})
	request.Header.Set(CSRFHeaderName, "csrf")
	response := httptest.NewRecorder()
	wrapper.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("authorized malformed body status = %d", response.Code)
	}
	if sessions.csrfCalls != 1 {
		t.Fatalf("csrf validation calls = %d, want one", sessions.csrfCalls)
	}
}

func TestWorkspaceDirectIDDenialUsesNeutralNoStoreResponse(t *testing.T) {
	sessions := &workspaceSessionDouble{principal: identity.Principal{SubjectID: "manager", OrganizationID: "ORG", Roles: []identity.Role{identity.RoleDepartmentManager}, SessionID: "session"}}
	boundary := NewAuthBoundary(nil, sessions)
	service := workspaceHTTPService("manager", "ORG", identity.RoleDepartmentManager, "manager")
	wrapper := ProtectAGADemoWorkspace(boundary, service, NewAGADemoWorkspaceHandler(service))
	request := httptest.NewRequest(http.MethodPost, "/v1/preprod/aga-demo-workspace/classification/commands", strings.NewReader(`{"operationId":"INCLUDE","idempotencyKey":"key","expectedGenerationId":"aga-ws-generation-0001","expectedDraftRevision":1,"expectedDraftContentDigest":"sha256:0000000000000000000000000000000000000000000000000000000000000000","targetQuestionKey":"workspace\\u001fguess","reasonCode":"CLASSIFICATION_EXPERT_REVIEW"}`))
	request.AddCookie(&http.Cookie{Name: SessionCookieName, Value: "session"})
	request.AddCookie(&http.Cookie{Name: CSRFCookieName, Value: "csrf"})
	request.Header.Set(CSRFHeaderName, "csrf")
	request.Header.Set("Idempotency-Key", "key")
	request.Header.Set("If-Match", `"rev-1"`)
	response := httptest.NewRecorder()
	wrapper.ServeHTTP(response, request)
	if response.Code != http.StatusNotFound || response.Header().Get("Cache-Control") != "private, no-store" {
		t.Fatalf("direct ID response = %d headers=%v body=%q", response.Code, response.Header(), response.Body.String())
	}
}

func TestWorkspaceRecommendationMissingServerFactsIsNeutralAndNoStore(t *testing.T) {
	sessions := &workspaceSessionDouble{principal: identity.Principal{SubjectID: "manager", OrganizationID: "ORG", Roles: []identity.Role{identity.RoleDepartmentManager}, SessionID: "session"}}
	boundary := NewAuthBoundary(nil, sessions)
	service := workspaceHTTPService("manager", "ORG", identity.RoleDepartmentManager, "manager")
	wrapper := ProtectAGADemoWorkspace(boundary, service, NewAGADemoWorkspaceHandler(service))
	body := `{"operationId":"CREATE_RECOMMENDATION","idempotencyKey":"recommendation-key","expectedGenerationId":"aga-ws-generation-0001","expectedDraftRevision":1,"draftRevision":1,"draftId":"aga-ws-draft-0001","draftContentDigest":"sha256:0000000000000000000000000000000000000000000000000000000000000000","organizationId":"ORG","providerScopeRootId":"aga-ws-scope-root-0001","providerScopeId":"aga-ws-scope-0001","providerScopeVersion":1,"providerTypeId":"provider-type","departmentId":"department","organizationalUnitId":"unit","targetId":"target","canonicalTargetKind":"SYSTEM","targetProfileCode":"AERODROME_MANAGEMENT_SYSTEM","inspectionProfileCode":"AERODROME_MANAGEMENT_SYSTEM","inspectionTypeCode":"PERIODIC_SURVEILLANCE","operationQualifiers":[{"key":"OPERATION_STATUS","value":"ACTIVE"}],"activityQualifiers":[{"key":"ACTIVITY_TYPE","value":"MAINTENANCE"}],"effectiveAt":"2026-08-04T12:00:00Z","taxonomyVersion":"AGA_QUESTION_CLASSIFICATION_V1","taxonomyDigest":"sha256:0000000000000000000000000000000000000000000000000000000000000000","classificationRunId":"aga-classification-run-test","classificationRunDigest":"sha256:0000000000000000000000000000000000000000000000000000000000000000","readinessEventId":"aga-ws-readiness-event-0001","readinessEventDigest":"sha256:0000000000000000000000000000000000000000000000000000000000000000"}`
	request := httptest.NewRequest(http.MethodPost, "/v1/preprod/aga-demo-workspace/recommendations/commands", strings.NewReader(body))
	request.AddCookie(&http.Cookie{Name: SessionCookieName, Value: "session"})
	request.AddCookie(&http.Cookie{Name: CSRFCookieName, Value: "csrf"})
	request.Header.Set(CSRFHeaderName, "csrf")
	request.Header.Set("Idempotency-Key", "recommendation-key")
	request.Header.Set("If-Match", `"rev-1"`)
	response := httptest.NewRecorder()
	wrapper.ServeHTTP(response, request)
	if response.Code != http.StatusNotFound || response.Header().Get("Cache-Control") != "private, no-store" {
		t.Fatalf("missing-facts recommendation response = %d headers=%v body=%q", response.Code, response.Header(), response.Body.String())
	}
}

func TestWorkspaceLifecycleQueryRejectsMissingInspectionBeforeStore(t *testing.T) {
	sessions := &workspaceSessionDouble{principal: identity.Principal{SubjectID: "manager", OrganizationID: "ORG", Roles: []identity.Role{identity.RoleDepartmentManager}, SessionID: "session"}}
	boundary := NewAuthBoundary(nil, sessions)
	service := workspaceHTTPService("manager", "ORG", identity.RoleDepartmentManager, "manager")
	wrapper := ProtectAGADemoWorkspace(boundary, service, NewAGADemoWorkspaceHandler(service))
	request := httptest.NewRequest(http.MethodPost, "/v1/preprod/aga-demo-workspace/lifecycle/query", strings.NewReader(`{"operationId":"GET_INSPECTION"}`))
	request.AddCookie(&http.Cookie{Name: SessionCookieName, Value: "session"})
	request.AddCookie(&http.Cookie{Name: CSRFCookieName, Value: "csrf"})
	request.Header.Set(CSRFHeaderName, "csrf")
	response := httptest.NewRecorder()
	wrapper.ServeHTTP(response, request)
	if response.Code != http.StatusNotFound || response.Header().Get("Cache-Control") != "private, no-store" {
		t.Fatalf("missing lifecycle inspection response = %d headers=%v body=%q", response.Code, response.Header(), response.Body.String())
	}
}
