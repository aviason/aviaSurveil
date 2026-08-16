package httpapi

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/aviason/aviaSurveil/internal/application"
	"github.com/jackc/pgx/v5"
)

func TestPublicErrorTitleRemovesSentinelPrefix(t *testing.T) {
	err := fmt.Errorf("%w: authorized closure reason is required", application.ErrConflict)
	if got := publicErrorTitle(err); got != "Authorized closure reason is required." {
		t.Fatalf("public error title = %q", got)
	}
}

func TestCreatedResponseMatchesOpenAPIStatusAndPreservesErrorMapping(t *testing.T) {
	api := &CanonicalAPI{}

	created := httptest.NewRecorder()
	api.respondCreated(created, map[string]string{"id": "created-record"}, nil)
	if created.Code != http.StatusCreated {
		t.Fatalf("created response status = %d, want %d", created.Code, http.StatusCreated)
	}

	forbidden := httptest.NewRecorder()
	api.respondCreated(forbidden, nil, application.ErrForbidden)
	if forbidden.Code != http.StatusForbidden {
		t.Fatalf("created error status = %d, want %d", forbidden.Code, http.StatusForbidden)
	}
}

func TestChecklistTemplateVersionDetailParsesImmutableSnapshot(t *testing.T) {
	publishedAt := time.Date(2026, 6, 15, 9, 0, 0, 0, time.UTC)
	view, err := checklistTemplateVersionDetailView(checklistTemplateVersionRecord{
		ID: "CTV-CABIN-1", TemplateID: "CABIN", Title: "Cabin Inspection checklist",
		Version: 1, PublishedAt: publishedAt, QuestionCount: 1,
		Snapshot: []byte(`{
			"schemaVersion": 1,
			"protocolVersion": 1,
			"questions": [{
				"id": "CAB-EMEQ-PBE-001",
				"sectionId": "EM EQ / PBE",
				"prompt": "PBE question",
				"regulatoryReference": "Configured Cabin Inspection reference - EM EQ / PBE",
				"expectedEvidence": "PBE serviceability record and cabin position confirmation",
				"allowedAnswers": ["COMPLIANT", "NON_COMPLIANT", "OBSERVATION", "NOT_APPLICABLE", "NOT_CHECKED"],
				"commentRequiredFor": ["NON_COMPLIANT", "OBSERVATION"],
				"assignedInspectorUserIds": ["USR-INSPECTOR-AMINA"]
			}]
		}`),
	})
	if err != nil {
		t.Fatalf("parse detail snapshot: %v", err)
	}
	if view.Id != "CTV-CABIN-1" || len(view.Questions) != 1 {
		t.Fatalf("detail view = %+v", view)
	}
	question := view.Questions[0]
	if len(question.AllowedAnswers) != 5 || len(question.CommentRequiredFor) != 2 {
		t.Fatalf("question arrays = allowed %+v required %+v", question.AllowedAnswers, question.CommentRequiredFor)
	}
	if strings.Contains(fmt.Sprintf("%+v", view), "USR-INSPECTOR-AMINA") {
		t.Fatalf("detail view leaked inspector assignment: %+v", view)
	}
}

func TestChecklistTemplateVersionDetailRejectsMalformedSnapshot(t *testing.T) {
	_, err := checklistTemplateVersionDetailView(checklistTemplateVersionRecord{
		ID: "CTV-CABIN-1", TemplateID: "CABIN", Title: "Cabin Inspection checklist",
		Version: 1, PublishedAt: time.Date(2026, 6, 15, 9, 0, 0, 0, time.UTC),
		Snapshot: []byte(`{"questions":[{"id":"CAB-EMEQ-PBE-001"}]}`),
	})
	if err == nil || !strings.Contains(err.Error(), "checklist template snapshot") {
		t.Fatalf("malformed snapshot err = %v", err)
	}
}

func TestChecklistTemplateVersionDetailMapsNotFound(t *testing.T) {
	if err := checklistTemplateVersionDetailStoreError(pgx.ErrNoRows); !errors.Is(err, application.ErrNotFound) {
		t.Fatalf("not-found mapping = %v", err)
	}
}

func TestCanonicalAPIRoutesChecklistTemplateVersionDirectLoad(t *testing.T) {
	handler := NewCanonicalAPI(CanonicalAPIDependencies{}).Handler()
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(
		http.MethodGet,
		"/v1/configuration/checklist-template-versions/CTV-CABIN-1",
		nil,
	))
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("direct detail route status = %d, want auth challenge before storage", response.Code)
	}
}

func TestCanonicalCatalogFacetWhereTypesEveryPreparedParameter(t *testing.T) {
	for _, exclude := range []string{"form", "domain", "topic", "risk", "focus", "recommendation"} {
		query := canonicalCatalogFacetWhere(exclude)
		for _, placeholder := range []string{
			"$1::text", "$2::text", "$3::text", "$4::text", "$5::text", "$6::text", "$7::text", "$8::text", "$9::text", "$10::text", "$11::text[]", "$12::text",
		} {
			if !strings.Contains(query, placeholder) {
				t.Fatalf("facet query for excluded %s does not type %s: %s", exclude, placeholder, query)
			}
		}
	}
}
