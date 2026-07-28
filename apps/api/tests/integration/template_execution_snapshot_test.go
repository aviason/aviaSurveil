//go:build canonicaltest

package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/application"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/configuration"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/httpapi"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/identity"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/platform/database"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/platform/idempotency"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/testprofile"
)

func TestChecklistQuestionDraftVersionAndPackageSnapshotWorkflow(t *testing.T) {
	pool := canonicalDatabase(t, "template_execution_snapshot")
	if err := testprofile.Reset(context.Background(), pool, canonicalNow); err != nil {
		t.Fatalf("reset canonical template profile: %v", err)
	}
	admin := principal("USR-ADMIN-ADA", "CAA", "TEST-USR-ADMIN-ADA", identity.RoleAdmin)
	manager := principal(
		"USR-MANAGER-NORA",
		"CAA",
		"TEST-USR-MANAGER-NORA",
		identity.RoleDepartmentManager,
	)
	projections := configuration.NewWorkspaceService(pool)
	workflow := testService(pool)

	publishedBefore := templateSnapshotJSON(t, pool, "CTV-CABIN-1")
	questions, err := projections.ListQuestions(context.Background(), admin, "", 100)
	if err != nil || len(questions) != 6 {
		t.Fatalf("initial Question Bank = %+v, err = %v", questions, err)
	}
	if _, err := projections.ListQuestions(context.Background(), manager, "", 100); !errors.Is(
		err,
		configuration.ErrWorkspaceForbidden,
	) {
		t.Fatalf("Department Manager Question Bank denial = %v", err)
	}

	created, err := workflow.CreateAdminQuestion(
		context.Background(),
		admin,
		application.CreateAdminQuestionCommand{
			OperationID: "op-template-question-create", IdempotencyKey: "idem-template-question-create",
			ExpectedRevision:    nil,
			Prompt:              "Is the multiline emergency equipment record complete?\nDoes it identify the exact cabin position?",
			ConfiguredReference: "Configured Cabin Inspection reference — EM EQ / PBE",
			ExpectedEvidence:    "PBE serviceability record\nCabin position confirmation",
		},
	)
	if err != nil {
		t.Fatalf("create Admin Question: %v", err)
	}
	replayed, err := workflow.CreateAdminQuestion(
		context.Background(),
		admin,
		application.CreateAdminQuestionCommand{
			OperationID: "op-template-question-create", IdempotencyKey: "idem-template-question-create",
			ExpectedRevision:    nil,
			Prompt:              "Is the multiline emergency equipment record complete?\nDoes it identify the exact cabin position?",
			ConfiguredReference: "Configured Cabin Inspection reference — EM EQ / PBE",
			ExpectedEvidence:    "PBE serviceability record\nCabin position confirmation",
		},
	)
	if err != nil || replayed != created {
		t.Fatalf("Question replay = %+v, err = %v", replayed, err)
	}
	if created.ID != "Q-ADMIN-2026-007" || created.Revision != 1 {
		t.Fatalf("created Question = %+v", created)
	}
	if _, err := workflow.CreateAdminQuestion(
		context.Background(),
		admin,
		application.CreateAdminQuestionCommand{
			OperationID:      "op-template-question-reused-key",
			IdempotencyKey:   "idem-template-question-create",
			ExpectedRevision: nil, Prompt: created.Prompt,
			ConfiguredReference: created.ConfiguredReference,
			ExpectedEvidence:    created.ExpectedEvidence,
		},
	); !errors.Is(err, idempotency.ErrOperationIDReuse) {
		t.Fatalf("reused Question idempotency key error = %v", err)
	}
	if _, err := workflow.CreateAdminQuestion(
		context.Background(),
		manager,
		application.CreateAdminQuestionCommand{
			OperationID:      "op-template-question-manager-denied",
			IdempotencyKey:   "idem-template-question-manager-denied",
			ExpectedRevision: nil, Prompt: "Manager must not create Admin Question.",
			ConfiguredReference: "Configured reference",
			ExpectedEvidence:    "Expected Evidence",
		},
	); !errors.Is(err, application.ErrForbidden) {
		t.Fatalf("Department Manager Question mutation denial = %v", err)
	}

	template, err := projections.GetTemplate(
		context.Background(),
		admin,
		"TPL-CABIN-2026",
	)
	if err != nil {
		t.Fatalf("get Template master: %v", err)
	}
	draft, err := workflow.CreateAdminTemplateDraft(
		context.Background(),
		admin,
		application.CreateAdminTemplateDraftCommand{
			OperationID: "op-template-draft-create", IdempotencyKey: "idem-template-draft-create",
			TemplateID: "TPL-CABIN-2026", ExpectedRevision: template.Revision,
			ChangeReason: "Add the multiline emergency equipment Question.",
		},
	)
	if err != nil {
		t.Fatalf("create Template Draft: %v", err)
	}
	if draft.ID != "CTV-CABIN-DRAFT-2" || draft.Status != "DRAFT" ||
		draft.Revision != 1 || len(draft.QuestionIDs) != 6 {
		t.Fatalf("created Template Draft = %+v", draft)
	}
	if _, err := workflow.CreateAdminTemplateDraft(
		context.Background(),
		admin,
		application.CreateAdminTemplateDraftCommand{
			OperationID: "op-template-draft-stale", IdempotencyKey: "idem-template-draft-stale",
			TemplateID: "TPL-CABIN-2026", ExpectedRevision: template.Revision,
			ChangeReason: "Stale duplicate Draft.",
		},
	); !errors.Is(err, application.ErrConflict) {
		t.Fatalf("stale Template Draft error = %v", err)
	}

	draft, err = workflow.AddAdminTemplateDraftQuestion(
		context.Background(),
		admin,
		application.AddAdminTemplateDraftQuestionCommand{
			OperationID: "op-template-question-add", IdempotencyKey: "idem-template-question-add",
			TemplateID: "TPL-CABIN-2026", DraftVersionID: draft.ID,
			QuestionID: created.ID, ExpectedRevision: draft.Revision,
		},
	)
	if err != nil {
		t.Fatalf("add Template Draft Question: %v", err)
	}
	draft, err = workflow.MoveAdminTemplateDraftQuestion(
		context.Background(),
		admin,
		application.MoveAdminTemplateDraftQuestionCommand{
			OperationID: "op-template-question-move", IdempotencyKey: "idem-template-question-move",
			TemplateID: "TPL-CABIN-2026", DraftVersionID: draft.ID,
			QuestionID: created.ID, Direction: "UP", ExpectedRevision: draft.Revision,
		},
	)
	if err != nil {
		t.Fatalf("move Template Draft Question: %v", err)
	}
	if draft.Revision != 3 ||
		draft.QuestionIDs[len(draft.QuestionIDs)-2] != created.ID {
		t.Fatalf("reordered Template Draft = %+v", draft)
	}
	if publishedAfter := templateSnapshotJSON(t, pool, "CTV-CABIN-1"); publishedAfter != publishedBefore {
		t.Fatalf("published checklist snapshot changed:\nbefore=%s\nafter=%s", publishedBefore, publishedAfter)
	}

	template, err = projections.GetTemplate(context.Background(), admin, "TPL-CABIN-2026")
	if err != nil || len(template.Versions) != 2 ||
		template.Versions[0].Status != "PUBLISHED" ||
		template.Versions[1].QuestionIDs[len(template.Versions[1].QuestionIDs)-2] != created.ID {
		t.Fatalf("Template version history = %+v, err = %v", template, err)
	}
	adminPackage, err := projections.GetInspectionPackage(
		context.Background(),
		admin,
		testprofile.CanonicalPackageID,
	)
	if err != nil {
		t.Fatalf("get Admin Inspection Package: %v", err)
	}
	if adminPackage.AuditID != testprofile.CanonicalAuditID ||
		len(adminPackage.QuestionIDs) != 6 ||
		len(adminPackage.ConfiguredReferences) != 6 ||
		len(adminPackage.ExpectedEvidence) != 6 ||
		len(adminPackage.RiskFocus) != 3 {
		t.Fatalf("Admin Inspection Package = %+v", adminPackage)
	}
	if containsString(adminPackage.QuestionIDs, created.ID) {
		t.Fatalf("existing Audit Package was changed by Draft edits: %+v", adminPackage)
	}
	for _, command := range []struct {
		operationID    string
		idempotencyKey string
		scope          string
		action         string
	}{
		{
			"op-template-question-create", "idem-template-question-create",
			"USR-ADMIN-ADA:create_admin_question", "admin.question_created",
		},
		{
			"op-template-draft-create", "idem-template-draft-create",
			"USR-ADMIN-ADA:create_admin_template_draft",
			"admin.template_draft_created",
		},
		{
			"op-template-question-add", "idem-template-question-add",
			"USR-ADMIN-ADA:add_admin_template_draft_question",
			"admin.template_question_added",
		},
		{
			"op-template-question-move", "idem-template-question-move",
			"USR-ADMIN-ADA:move_admin_template_draft_question",
			"admin.template_question_reordered",
		},
	} {
		assertTemplateCommandEnvelope(
			t, pool, command.operationID, command.idempotencyKey,
			command.scope, command.action,
		)
	}
}

func TestChecklistConfigurationHTTPContract(t *testing.T) {
	pool := canonicalDatabase(t, "template_execution_http")
	if err := testprofile.Reset(context.Background(), pool, canonicalNow); err != nil {
		t.Fatalf("reset canonical Template HTTP profile: %v", err)
	}
	api := httpapi.NewCanonicalAPI(httpapi.CanonicalAPIDependencies{
		Pool: pool, Application: testService(pool), Clock: func() time.Time { return canonicalNow },
	})
	handler := httpapi.NewCanonicalTestBoundary("task-5-token").Protect(api.Handler())

	for _, route := range []struct {
		path     string
		required string
	}{
		{"/v1/admin/regulatory-references", `"id":"NAMCARS-CAB-001"`},
		{"/v1/admin/templates", `"id":"TPL-CABIN-2026"`},
		{"/v1/admin/questions", `"id":"CAB-EMEQ-PBE-001"`},
		{"/v1/admin/templates/TPL-CABIN-2026", `"publishedVersionId":"CTV-CABIN-1"`},
		{"/v1/admin/inspection-packages/PKG-CAB-2026-001", `"riskFocus"`},
	} {
		request := task5Request(http.MethodGet, route.path, "", "USR-ADMIN-ADA")
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusOK ||
			!strings.Contains(response.Body.String(), route.required) {
			t.Fatalf("GET %s status=%d body=%s", route.path, response.Code, response.Body.String())
		}
	}

	denied := task5Request(http.MethodGet, "/v1/admin/questions", "", "USR-MANAGER-NORA")
	deniedResponse := httptest.NewRecorder()
	handler.ServeHTTP(deniedResponse, denied)
	if deniedResponse.Code != http.StatusForbidden {
		t.Fatalf("Department Manager Admin Question denial status=%d body=%s",
			deniedResponse.Code, deniedResponse.Body.String())
	}

	createQuestion := task5Request(http.MethodPost, "/v1/admin/questions", `{
		"operationId":"OP-HTTP-TEMPLATE-QUESTION",
		"expectedRevision":null,
		"idempotencyKey":"IDEM-HTTP-TEMPLATE-QUESTION",
		"prompt":"Is the HTTP multiline record complete?\nDoes it identify the cabin position?",
		"configuredReference":"Configured Cabin Inspection reference — EM EQ / PBE",
		"expectedEvidence":"PBE record\nCabin position"
	}`, "USR-ADMIN-ADA")
	createQuestion.Header.Set("Idempotency-Key", "IDEM-HTTP-TEMPLATE-QUESTION")
	createQuestionResponse := httptest.NewRecorder()
	handler.ServeHTTP(createQuestionResponse, createQuestion)
	if createQuestionResponse.Code != http.StatusCreated ||
		!strings.Contains(createQuestionResponse.Body.String(), `"id":"Q-ADMIN-2026-007"`) ||
		!strings.Contains(createQuestionResponse.Body.String(), `\n`) {
		t.Fatalf("POST Admin Question status=%d body=%s",
			createQuestionResponse.Code, createQuestionResponse.Body.String())
	}

	createDraft := task5Request(
		http.MethodPost,
		"/v1/admin/templates/TPL-CABIN-2026/drafts",
		`{
			"operationId":"OP-HTTP-TEMPLATE-DRAFT",
			"expectedRevision":1,
			"idempotencyKey":"IDEM-HTTP-TEMPLATE-DRAFT",
			"templateId":"TPL-CABIN-2026",
			"changeReason":"Add HTTP multiline Question."
		}`,
		"USR-ADMIN-ADA",
	)
	createDraft.Header.Set("Idempotency-Key", "IDEM-HTTP-TEMPLATE-DRAFT")
	createDraft.Header.Set("If-Match", `"rev-1"`)
	createDraftResponse := httptest.NewRecorder()
	handler.ServeHTTP(createDraftResponse, createDraft)
	if createDraftResponse.Code != http.StatusCreated ||
		!strings.Contains(createDraftResponse.Body.String(), `"id":"CTV-CABIN-DRAFT-2"`) {
		t.Fatalf("POST Template Draft status=%d body=%s",
			createDraftResponse.Code, createDraftResponse.Body.String())
	}

	addQuestion := task5Request(
		http.MethodPost,
		"/v1/admin/templates/TPL-CABIN-2026/drafts/CTV-CABIN-DRAFT-2/questions",
		`{
			"operationId":"OP-HTTP-TEMPLATE-ADD",
			"expectedRevision":1,
			"idempotencyKey":"IDEM-HTTP-TEMPLATE-ADD",
			"templateId":"TPL-CABIN-2026",
			"draftVersionId":"CTV-CABIN-DRAFT-2",
			"questionId":"Q-ADMIN-2026-007"
		}`,
		"USR-ADMIN-ADA",
	)
	addQuestion.Header.Set("Idempotency-Key", "IDEM-HTTP-TEMPLATE-ADD")
	addQuestion.Header.Set("If-Match", `"rev-1"`)
	addQuestionResponse := httptest.NewRecorder()
	handler.ServeHTTP(addQuestionResponse, addQuestion)
	if addQuestionResponse.Code != http.StatusOK ||
		!strings.Contains(addQuestionResponse.Body.String(), `"revision":2`) {
		t.Fatalf("POST Template Question status=%d body=%s",
			addQuestionResponse.Code, addQuestionResponse.Body.String())
	}

	moveQuestion := task5Request(
		http.MethodPost,
		"/v1/admin/templates/TPL-CABIN-2026/drafts/CTV-CABIN-DRAFT-2/questions/Q-ADMIN-2026-007/moves",
		`{
			"operationId":"OP-HTTP-TEMPLATE-MOVE",
			"expectedRevision":2,
			"idempotencyKey":"IDEM-HTTP-TEMPLATE-MOVE",
			"templateId":"TPL-CABIN-2026",
			"draftVersionId":"CTV-CABIN-DRAFT-2",
			"questionId":"Q-ADMIN-2026-007",
			"direction":"UP"
		}`,
		"USR-ADMIN-ADA",
	)
	moveQuestion.Header.Set("Idempotency-Key", "IDEM-HTTP-TEMPLATE-MOVE")
	moveQuestion.Header.Set("If-Match", `"rev-2"`)
	moveQuestionResponse := httptest.NewRecorder()
	handler.ServeHTTP(moveQuestionResponse, moveQuestion)
	if moveQuestionResponse.Code != http.StatusOK ||
		!strings.Contains(moveQuestionResponse.Body.String(), `"revision":3`) {
		t.Fatalf("POST Template Question move status=%d body=%s",
			moveQuestionResponse.Code, moveQuestionResponse.Body.String())
	}

	packageRequest := task5Request(
		http.MethodGet,
		"/v1/admin/inspection-packages/PKG-CAB-2026-001",
		"",
		"USR-ADMIN-ADA",
	)
	packageResponse := httptest.NewRecorder()
	handler.ServeHTTP(packageResponse, packageRequest)
	if packageResponse.Code != http.StatusOK ||
		strings.Contains(packageResponse.Body.String(), "Q-ADMIN-2026-007") {
		t.Fatalf("immutable HTTP Package status=%d body=%s",
			packageResponse.Code, packageResponse.Body.String())
	}
}

func task5Request(method, path, body, subjectID string) *http.Request {
	request := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	request.Header.Set(httpapi.CanonicalTestTokenHeader, "task-5-token")
	request.Header.Set(httpapi.CanonicalTestSubjectHeader, subjectID)
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
	}
	return request
}

func templateSnapshotJSON(
	t *testing.T,
	pool *database.Pool,
	templateVersionID string,
) string {
	t.Helper()
	var snapshot []byte
	if err := pool.QueryRow(
		context.Background(),
		"SELECT snapshot FROM checklist_template_versions WHERE id = $1",
		templateVersionID,
	).Scan(&snapshot); err != nil {
		t.Fatalf("read checklist template snapshot: %v", err)
	}
	var normalized any
	if err := json.Unmarshal(snapshot, &normalized); err != nil {
		t.Fatalf("decode checklist template snapshot: %v", err)
	}
	encoded, err := json.Marshal(normalized)
	if err != nil {
		t.Fatalf("encode checklist template snapshot: %v", err)
	}
	return string(encoded)
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func assertTemplateCommandEnvelope(
	t *testing.T,
	pool *database.Pool,
	operationID string,
	idempotencyKey string,
	scope string,
	action string,
) {
	t.Helper()
	var auditAction, changeOperationID, outboxOperationID, outboxIdempotencyKey string
	if err := pool.QueryRow(context.Background(), `
		SELECT audit.action, change.operation_id, outbox.operation_id, outbox.idempotency_key
		FROM command_transaction_links link
		JOIN audit_events audit ON audit.event_id = link.audit_event_id
		JOIN authorized_sync_changes change ON change.sequence_id = link.change_sequence_id
		JOIN outbox_messages outbox ON outbox.id = link.outbox_message_id
		WHERE link.operation_id = $1 AND link.idempotency_scope = $2
	`, operationID, scope).Scan(
		&auditAction, &changeOperationID, &outboxOperationID, &outboxIdempotencyKey,
	); err != nil {
		t.Fatalf("Template command envelope %s: %v", operationID, err)
	}
	expectedIdempotencyKey := "command:" + scope + ":idempotency:" + idempotencyKey
	if auditAction != action || changeOperationID != operationID ||
		outboxOperationID != operationID || outboxIdempotencyKey != expectedIdempotencyKey {
		t.Fatalf(
			"Template command envelope %s action=%q change=%q outbox=%q idempotency=%q",
			operationID, auditAction, changeOperationID, outboxOperationID,
			outboxIdempotencyKey,
		)
	}
}
