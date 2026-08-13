//go:build canonicaltest

package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/aviason/aviaSurveil/internal/administration"
	"github.com/aviason/aviaSurveil/internal/assistant"
	"github.com/aviason/aviaSurveil/internal/httpapi"
	"github.com/aviason/aviaSurveil/internal/identity"
	"github.com/aviason/aviaSurveil/internal/risk"
	"github.com/aviason/aviaSurveil/internal/testprofile"
	"github.com/aviason/aviaSurveil/migrations"
)

func TestRiskProjectionsAreVersionedAdvisoryAndNonAuthoritative(t *testing.T) {
	pool := canonicalDatabase(t, "risk_projection")
	seedFinding(t, pool, "finding-risk-001", "CAR-RISK-001", "airline-xyz")
	service := risk.NewService(pool, risk.Dependencies{
		Clock:       func() time.Time { return canonicalNow },
		IDGenerator: scenarioIDGenerator(),
	})
	manager := principal(
		"manager-001", "caa", "session-manager", identity.RoleDepartmentManager,
	)

	overview, err := service.GetOverview(
		context.Background(), manager, "airline-xyz",
	)
	if err != nil {
		t.Fatalf("get governed risk overview: %v", err)
	}
	if overview.OrganizationID != "airline-xyz" ||
		overview.OpenFindingCount != 1 ||
		overview.RepeatFindingCount != 0 ||
		!overview.AdvisoryOnly ||
		overview.Source != "canonical finding lifecycle projection" ||
		overview.NonDecisionLabel != "Advisory management information — not an enforcement, certificate, or closure decision." ||
		overview.Revision != 1 ||
		!overview.CalculatedAt.Equal(canonicalNow) {
		t.Fatalf("risk overview = %+v", overview)
	}

	replayed, err := service.GetOverview(
		context.Background(), manager, "airline-xyz",
	)
	if err != nil || !reflect.DeepEqual(replayed, overview) {
		t.Fatalf("stable risk replay = %+v, err = %v", replayed, err)
	}
	var storedVersions int
	if err := pool.QueryRow(context.Background(), `
		SELECT count(*)
		FROM risk_projection_versions
		WHERE projection_kind = 'OVERVIEW'
		  AND organization_id = 'airline-xyz'
	`).Scan(&storedVersions); err != nil {
		t.Fatalf("count stored risk projections: %v", err)
	}
	if storedVersions != 1 {
		t.Fatalf("stable projection stored %d versions", storedVersions)
	}

	seedFinding(t, pool, "finding-risk-002", "CAR-RISK-002", "airline-xyz")
	refreshed, err := service.GetOverview(
		context.Background(), manager, "airline-xyz",
	)
	if err != nil {
		t.Fatalf("refresh governed risk overview: %v", err)
	}
	if refreshed.Revision != 2 || refreshed.OpenFindingCount != 2 ||
		refreshed.Source != overview.Source || !refreshed.AdvisoryOnly {
		t.Fatalf("refreshed risk overview = %+v", refreshed)
	}
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO cap_revisions (
			id, cap_id, finding_id, organization_id, revision, status,
			root_cause, corrective_action, preventive_action,
			target_completion_date, submitted_by_subject_id, submitted_at
		) VALUES (
			'cap-risk-001-v1', 'cap-risk-001', 'finding-risk-001',
			'airline-xyz', 1, 'ACCEPTED', 'Configured root cause',
			'Configured correction', 'Configured prevention', '2026-08-15',
			'auditee-xyz', $1
		)
	`, canonicalNow); err != nil {
		t.Fatalf("seed risk CAP projection source: %v", err)
	}

	management, err := service.GetManagementProjection(
		context.Background(), manager,
	)
	if err != nil {
		t.Fatalf("get management projection: %v", err)
	}
	if len(management.Findings) != 2 ||
		len(management.CAPEffectiveness) != 1 ||
		!management.AdvisoryOnly ||
		management.Source != "canonical finding and CAP lifecycle projection" ||
		management.NonDecisionLabel != overview.NonDecisionLabel ||
		management.Revision != 1 {
		t.Fatalf("management projection = %+v", management)
	}
	for _, finding := range management.Findings {
		if finding.RiskLevel != risk.ManagementRiskMedium {
			t.Errorf("finding %s risk level = %s", finding.FindingID, finding.RiskLevel)
		}
	}
	for _, item := range management.CAPEffectiveness {
		if item.State != risk.CAPEffectivenessNotEligible ||
			!strings.Contains(item.Reason, "Finding closure") {
			t.Errorf("CAP effectiveness item = %+v", item)
		}
	}

	if _, err := service.GetOverview(
		context.Background(),
		principal("auditee-xyz", "airline-xyz", "session-auditee", identity.RoleAuditee),
		"airline-xyz",
	); !errors.Is(err, risk.ErrForbidden) {
		t.Fatalf("Auditee risk overview error = %v", err)
	}
	if _, err := service.GetManagementProjection(
		context.Background(),
		principal("inspector-cabin-001", "caa", "session-inspector", identity.RoleInspector),
	); !errors.Is(err, risk.ErrForbidden) {
		t.Fatalf("Inspector management projection error = %v", err)
	}

	rows, err := pool.Query(context.Background(), `
		SELECT id
		FROM risk_projection_versions
		ORDER BY id
	`)
	if err != nil {
		t.Fatalf("list immutable risk projections: %v", err)
	}
	defer rows.Close()
	for rows.Next() {
		var projectionID string
		if err := rows.Scan(&projectionID); err != nil {
			t.Fatalf("scan immutable risk projection: %v", err)
		}
		if _, err := pool.Exec(context.Background(), `
			UPDATE risk_projection_versions
			SET source = 'rewritten source'
			WHERE id = $1
		`, projectionID); err == nil {
			t.Fatalf("risk projection %s accepted an in-place rewrite", projectionID)
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate immutable risk projections: %v", err)
	}

	var findingStatuses []string
	if err := pool.QueryRow(context.Background(), `
		SELECT array_agg(status ORDER BY id)
		FROM findings
		WHERE id IN ('finding-risk-001', 'finding-risk-002')
	`).Scan(&findingStatuses); err != nil {
		t.Fatalf("read Finding authority after projections: %v", err)
	}
	if got := strings.Join(findingStatuses, ","); got != "WAITING_FOR_CAP,WAITING_FOR_CAP" {
		t.Fatalf("risk projection changed Finding authority: %s", got)
	}
}

func TestAdministrationProjectionsAndFiltersAreRoleScoped(t *testing.T) {
	pool := canonicalDatabase(t, "administration_projection")
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO identity_references (subject_id, issuer, display_name)
		VALUES ('admin-001', 'test', 'Administrator')
	`); err != nil {
		t.Fatalf("seed administrator identity: %v", err)
	}
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO session_references (
			id, subject_id, organization_id, expires_at, last_seen_at,
			absolute_expires_at, roles
		) VALUES (
			'session-admin', 'admin-001', 'caa', $1, $2, $1, ARRAY['admin']
		)
	`, canonicalNow.Add(24*time.Hour), canonicalNow); err != nil {
		t.Fatalf("seed administrator session: %v", err)
	}
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO report_definition_versions (
			id, definition_id, version, title, description, definition,
			created_by_subject_id, created_at
		) VALUES (
			'report-definition-caps-v1', 'CAPS_BY_DUE_STATE', 1,
			'CAPs by Due State', 'Governed CAP due-state management projection.',
			'{"packageFields":["findingId","dueState"],"actionReason":"Review only; no automatic closure."}',
			'admin-001', $1
		)
	`, canonicalNow); err != nil {
		t.Fatalf("seed report definition: %v", err)
	}
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO audit_events (
			event_id, occurred_at, actor_subject_id, actor_role, organization_id,
			action, entity_type, entity_id, entity_version, after_status,
			operation_id, correlation_id, request_id, details
		) VALUES (
			'audit-admin-filter-001', $1, 'admin-001', 'admin', 'caa',
			'admin.report_definition_created', 'REPORT_DEFINITION',
			'report-definition-caps-v1', 1, 'PUBLISHED',
			'op-admin-filter-001', 'corr-admin-filter-001',
			'req-admin-filter-001', '{"system":"MANUAL"}'
		)
	`, canonicalNow); err != nil {
		t.Fatalf("seed admin audit event: %v", err)
	}
	if _, err := pool.Exec(context.Background(), `
			INSERT INTO identity_action_facts (
				id, request_id, fact_sequence, membership_id, subject_id,
				action_kind, state, delivery_attempt, expires_at,
				provider_acknowledged_at, reason, created_at
			) VALUES (
				'identity-action-directory-invitation-001',
				'fixture-membership-inspector-request', 1, NULL,
				'inspector-cabin-001', 'INVITATION',
					'DELIVERY_ACCEPTED', 1,
					$1::timestamptz + interval '24 hours', $1,
				'Approved seeded invitation projection.', $1
			)
		`, canonicalNow); err != nil {
		t.Fatalf("seed invitation delivery action fact: %v", err)
	}
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO notification_records (
			id, recipient_subject_id, title, body, related_entity_type,
			related_entity_id, deduplication_key, created_at
		) VALUES (
			'notification-directory-invitation-001', 'inspector-cabin-001',
			'Access invitation', 'Identity delivery fact.',
			'USER_LIFECYCLE_INVITATION', 'fixture-membership-inspector-request',
			'directory-invitation-001', $1
		)
	`, canonicalNow); err != nil {
		t.Fatalf("seed invitation notification fact: %v", err)
	}
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO notification_delivery_jobs (
			id, notification_id, recipient_subject_id, channel, status,
			idempotency_key, attempt_count, provider_message_id, accepted_at,
			created_at, updated_at
		) VALUES (
			'delivery-directory-invitation-001',
			'notification-directory-invitation-001', 'inspector-cabin-001',
			'EMAIL', 'DELIVERED', 'delivery-directory-invitation-001', 1,
			'<directory-invitation-001@aviasurveil360.local>', $1, $1, $1
		)
	`, canonicalNow); err != nil {
		t.Fatalf("seed invitation email delivery fact: %v", err)
	}
	service := administration.NewProjectionService(
		pool,
		administration.ProjectionDependencies{
			Clock:             func() time.Time { return canonicalNow },
			DirectoryProvider: canonicalDirectoryProvider{},
		},
	)
	admin := principal("admin-001", "caa", "session-admin", identity.RoleAdmin)

	screen, err := service.GetScreenProjection(
		context.Background(), admin, "admin-reports",
	)
	if err != nil {
		t.Fatalf("get admin screen projection: %v", err)
	}
	if screen.ScreenID != "admin-reports" || screen.State != administration.ScreenReady ||
		len(screen.VisibleActions) != 1 ||
		screen.VisibleActions[0].ID != "download-admin-report" {
		t.Fatalf("admin screen projection = %+v", screen)
	}
	if _, err := service.GetScreenProjection(
		context.Background(),
		principal("manager-001", "caa", "session-manager", identity.RoleDepartmentManager),
		"admin-reports",
	); !errors.Is(err, administration.ErrForbidden) {
		t.Fatalf("manager admin screen error = %v", err)
	}
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO communication_threads (
			id, organization_id, visibility, subject, revision, created_at, updated_at
		) VALUES (
			'thread-internal-only', NULL, 'INTERNAL_CAA',
			'Internal CAA workload discussion', 1, $1, $1
		)
	`, canonicalNow); err != nil {
		t.Fatalf("seed Internal CAA-only thread: %v", err)
	}
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO communication_messages (
			id, thread_id, organization_id, visibility, sender_subject_id,
			audience, direction, subject, body, idempotency_key, revision, created_at
		) VALUES (
			'message-internal-only', 'thread-internal-only', NULL, 'INTERNAL_CAA',
			'inspector-cabin-001', 'CAA', 'CAA_INTERNAL',
			'Internal CAA Note', 'Private workload discussion.',
			'idem-internal-only', 1, $1
		)
	`, canonicalNow); err != nil {
		t.Fatalf("seed Internal CAA-only message: %v", err)
	}
	auditeeMessages, err := service.GetScreenProjection(
		context.Background(),
		principal("auditee-xyz", "airline-xyz", "session-auditee", identity.RoleAuditee),
		"auditee-messages",
	)
	if err != nil || auditeeMessages.State != administration.ScreenEmpty {
		t.Fatalf(
			"Auditee message screen disclosed Internal CAA activity: projection=%+v, err=%v",
			auditeeMessages,
			err,
		)
	}
	action, err := service.InvokeVisibleAction(
		context.Background(), admin, "admin-reports", "download-admin-report",
	)
	if err != nil || action.Effect.Type != administration.ActionFileDownload ||
		action.Effect.File != "admin-report.csv" {
		t.Fatalf("admin visible action = %+v, err = %v", action, err)
	}

	reports, err := service.ListReportDefinitions(
		context.Background(), admin, "due",
	)
	if err != nil || len(reports) != 1 ||
		reports[0].ID != "CAPS_BY_DUE_STATE" ||
		reports[0].ActionReason != "Review only; no automatic closure." {
		t.Fatalf("admin report definitions = %+v, err = %v", reports, err)
	}
	directory, err := service.ListAccessDirectory(
		context.Background(),
		admin,
		administration.AccessDirectoryFilters{
			Search: "Cabin", Role: identity.RoleInspector,
		},
	)
	if err != nil || len(directory.Items) != 1 ||
		directory.Items[0].SubjectID != "inspector-cabin-001" ||
		directory.Items[0].Email != "inspector.cabin@example.test" ||
		!directory.Items[0].MFAEnrolled ||
		directory.Items[0].InvitationState != "delivered" ||
		directory.ProviderCalls != 2 {
		t.Fatalf("admin access directory = %+v, err = %v", directory, err)
	}
	preloginDirectory, err := service.ListAccessDirectory(
		context.Background(),
		admin,
		administration.AccessDirectoryFilters{
			Search: "Prelogin", Role: identity.RoleAuditee,
		},
	)
	if err != nil ||
		len(preloginDirectory.Items) != 1 ||
		preloginDirectory.Items[0].SubjectID != "provider-prelogin-001" ||
		preloginDirectory.Items[0].ApplicationProfile != "absent" ||
		preloginDirectory.Items[0].MembershipRevision != 0 ||
		preloginDirectory.Items[0].LastSuccessfulSession != nil {
		t.Fatalf("pre-login provider directory = %+v, err = %v", preloginDirectory, err)
	}
	organizations, err := service.ListOrganizations(
		context.Background(), admin,
		administration.OrganizationFilters{Search: "Airline", OrganizationType: "OPERATOR", Status: "ACTIVE"},
	)
	if err != nil || len(organizations) != 2 {
		t.Fatalf("admin organizations = %+v, err = %v", organizations, err)
	}
	if _, err := service.GetOrganization(
		context.Background(), admin, "airline-other",
	); !errors.Is(err, administration.ErrNotFound) {
		t.Fatalf("undeclared admin Organization detail error = %v", err)
	}
	auditEvents, err := service.ListAuditEvents(
		context.Background(), admin,
		administration.AuditEventFilters{
			Actor: "administrator", Action: "report_definition",
			Entity: "REPORT_DEFINITION report-definition-caps-v1",
			System: "MANUAL", DateText: "2026-07-21",
		},
	)
	if err != nil || len(auditEvents) != 1 ||
		auditEvents[0].EventID != "audit-admin-filter-001" {
		t.Fatalf("filtered admin audit events = %+v, err = %v", auditEvents, err)
	}
	if _, err := service.ListReportDefinitions(
		context.Background(),
		principal("inspector-cabin-001", "caa", "session-inspector", identity.RoleInspector),
		"",
	); !errors.Is(err, administration.ErrForbidden) {
		t.Fatalf("non-admin report definitions error = %v", err)
	}
}

type capturingAssistantProvider struct {
	requests []assistant.ProviderRequest
}

func (provider *capturingAssistantProvider) Generate(
	_ context.Context,
	request assistant.ProviderRequest,
) (assistant.ProviderResponse, error) {
	provider.requests = append(provider.requests, request)
	return assistant.ProviderResponse{
		Text:       "Captured deterministic advisory draft.",
		ProviderID: "capturing-test-provider",
	}, nil
}

func TestAssistantDraftIsMinimizedIdempotentAndDoesNotMutateFinding(t *testing.T) {
	pool := canonicalDatabase(t, "assistant_draft")
	seedFinding(t, pool, "finding-assistant-001", "CAR-ASSISTANT-001", "airline-xyz")
	if _, err := pool.Exec(context.Background(), `
		UPDATE findings
		SET next_action = 'Private enforcement deliberation must stay server-side.'
		WHERE id = 'finding-assistant-001'
	`); err != nil {
		t.Fatalf("seed forbidden Finding context: %v", err)
	}
	provider := &capturingAssistantProvider{}
	service := assistant.NewService(pool, assistant.Dependencies{
		Clock:       func() time.Time { return canonicalNow },
		IDGenerator: scenarioIDGenerator(),
		Provider:    provider,
	})
	inspector := principal(
		"inspector-cabin-001", "caa", "session-inspector", identity.RoleInspector,
	)

	guidance, err := service.GetGuidance(inspector)
	if err != nil ||
		!guidance.AdvisoryOnly ||
		strings.Join(guidance.ProhibitedActions, ",") != "create Finding,set severity,close Finding,enforcement action" {
		t.Fatalf("assistant guidance = %+v, err = %v", guidance, err)
	}
	command := assistant.CreateDraftCommand{
		OperationID:    "op-assistant-draft-001",
		IdempotencyKey: "idem-assistant-draft-001",
		FindingID:      "finding-assistant-001",
		Prompt:         "  Draft an evidence request only.  ",
	}
	draft, err := service.CreateDraft(context.Background(), inspector, command)
	if err != nil {
		t.Fatalf("create assistant draft: %v", err)
	}
	if draft.ID == "" || draft.FindingID != command.FindingID ||
		draft.Prompt != "Draft an evidence request only." ||
		draft.Draft != "Captured deterministic advisory draft." ||
		!draft.AdvisoryOnly || draft.CanCreateFinding ||
		draft.CanSetSeverity || draft.CanCloseFinding ||
		draft.ProviderID != "capturing-test-provider" ||
		!draft.GeneratedAt.Equal(canonicalNow) {
		t.Fatalf("assistant draft = %+v", draft)
	}
	replayed, err := service.CreateDraft(context.Background(), inspector, command)
	if err != nil || !reflect.DeepEqual(replayed, draft) {
		t.Fatalf("assistant draft replay = %+v, err = %v", replayed, err)
	}
	if len(provider.requests) != 1 {
		t.Fatalf("assistant provider invoked %d times for an idempotent command", len(provider.requests))
	}
	providerWire, err := json.Marshal(provider.requests[0])
	if err != nil {
		t.Fatalf("marshal minimized provider request: %v", err)
	}
	providerText := strings.ToLower(string(providerWire))
	for _, forbidden := range []string{
		"private enforcement", "internal", "comment", "organization",
		"owner", "next_action", "nextaction", "severity", "status",
	} {
		if strings.Contains(providerText, forbidden) {
			t.Errorf("provider request leaked forbidden context %q: %s", forbidden, providerWire)
		}
	}
	if provider.requests[0].FindingID != command.FindingID ||
		provider.requests[0].FindingReference != "CAR-ASSISTANT-001" ||
		provider.requests[0].Prompt != "Draft an evidence request only." {
		t.Fatalf("minimized provider request = %+v", provider.requests[0])
	}

	var findingStatus, nextAction string
	var findingRevision int64
	if err := pool.QueryRow(context.Background(), `
		SELECT status, next_action, revision
		FROM findings
		WHERE id = 'finding-assistant-001'
	`).Scan(&findingStatus, &nextAction, &findingRevision); err != nil {
		t.Fatalf("read Finding after assistant draft: %v", err)
	}
	if findingStatus != "WAITING_FOR_CAP" ||
		nextAction != "Private enforcement deliberation must stay server-side." ||
		findingRevision != 1 {
		t.Fatalf(
			"assistant mutated Finding authority: status=%q next=%q revision=%d",
			findingStatus, nextAction, findingRevision,
		)
	}
	var auditAction string
	var auditDetails []byte
	var linked int
	if err := pool.QueryRow(context.Background(), `
		SELECT action, details
		FROM audit_events
		WHERE operation_id = $1
	`, command.OperationID).Scan(&auditAction, &auditDetails); err != nil {
		t.Fatalf("read assistant audit event: %v", err)
	}
	if auditAction != "assistant.advisory_draft_generated" ||
		strings.Contains(strings.ToLower(string(auditDetails)), strings.ToLower(command.Prompt)) ||
		!strings.Contains(string(auditDetails), "capturing-test-provider") {
		t.Fatalf("assistant audit event action=%q details=%s", auditAction, auditDetails)
	}
	if err := pool.QueryRow(context.Background(), `
		SELECT count(*)
		FROM command_transaction_links
		WHERE operation_id = $1
	`, command.OperationID).Scan(&linked); err != nil {
		t.Fatalf("read assistant transaction link: %v", err)
	}
	if linked != 1 {
		t.Fatalf("assistant transaction envelope links = %d", linked)
	}
	if _, err := service.CreateDraft(
		context.Background(),
		principal("manager-001", "caa", "session-manager", identity.RoleDepartmentManager),
		assistant.CreateDraftCommand{
			OperationID: "op-assistant-denied", IdempotencyKey: "idem-assistant-denied",
			FindingID: command.FindingID, Prompt: "Draft.",
		},
	); !errors.Is(err, assistant.ErrForbidden) {
		t.Fatalf("manager assistant draft error = %v", err)
	}
}

func TestManagementAdminAssistantExactHTTPAndRawWirePrivacy(t *testing.T) {
	pool := createTestDatabase(t, "management_admin_assistant_http")
	if err := migrations.Apply(context.Background(), pool); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
	if err := testprofile.Reset(context.Background(), pool, canonicalNow); err != nil {
		t.Fatalf("reset canonical profile: %v", err)
	}
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO report_definition_versions (
			id, definition_id, version, title, description, definition,
			created_by_subject_id, created_at
		) VALUES (
			'RDEF-CAP-DUE-V1', 'CAPS_BY_DUE_STATE', 1,
			'CAPs by Due State', 'Governed CAP due-state management projection.',
			'{"packageFields":["findingId","dueState"],"actionReason":"Review only; no automatic closure."}',
			'USR-ADMIN-ADA', $1
		)
	`, canonicalNow); err != nil {
		t.Fatalf("seed HTTP report definition: %v", err)
	}
	generator := testprofile.NewGenerator()
	api := httpapi.NewCanonicalAPI(httpapi.CanonicalAPIDependencies{
		Pool: pool, Application: testService(pool),
		Risk: risk.NewService(pool, risk.Dependencies{
			Clock: func() time.Time { return canonicalNow }, IDGenerator: generator.Next,
		}),
		Administration: administration.NewProjectionService(
			pool,
			administration.ProjectionDependencies{
				Clock:             func() time.Time { return canonicalNow },
				DirectoryProvider: canonicalDirectoryProvider{},
			},
		),
		Assistant: assistant.NewService(pool, assistant.Dependencies{
			Clock: func() time.Time { return canonicalNow }, IDGenerator: generator.Next,
			Provider: assistant.NewDeterministicProvider(),
		}),
		Clock: func() time.Time { return canonicalNow },
	})
	handler := httpapi.NewCanonicalTestBoundary("task-9-token").Protect(api.Handler())
	request := func(method, path, body, subjectID string) *httptest.ResponseRecorder {
		httpRequest := httptest.NewRequest(method, path, bytes.NewBufferString(body))
		httpRequest.Header.Set(httpapi.CanonicalTestTokenHeader, "task-9-token")
		httpRequest.Header.Set(httpapi.CanonicalTestSubjectHeader, subjectID)
		if body != "" {
			httpRequest.Header.Set("Content-Type", "application/json")
		}
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httpRequest)
		return response
	}
	mutation := func(
		path, body, subjectID, idempotencyKey string,
	) *httptest.ResponseRecorder {
		httpRequest := httptest.NewRequest(
			http.MethodPost, path, bytes.NewBufferString(body),
		)
		httpRequest.Header.Set(httpapi.CanonicalTestTokenHeader, "task-9-token")
		httpRequest.Header.Set(httpapi.CanonicalTestSubjectHeader, subjectID)
		httpRequest.Header.Set("Content-Type", "application/json")
		httpRequest.Header.Set("Idempotency-Key", idempotencyKey)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httpRequest)
		return response
	}

	overview := request(
		http.MethodGet,
		"/v1/risk/overview?organizationId=ORG-SKYCARGO",
		"",
		"USR-MANAGER-NORA",
	)
	if overview.Code != http.StatusOK {
		t.Fatalf("risk overview status=%d body=%s", overview.Code, overview.Body.String())
	}
	assertClosedJSONKeys(t, overview.Body.Bytes(), []string{
		"organizationId", "overdueFindingCount", "openFindingCount",
		"repeatFindingCount", "revision",
	})
	management := request(
		http.MethodGet, "/v1/risk/management", "", "USR-MANAGER-NORA",
	)
	if management.Code != http.StatusOK {
		t.Fatalf(
			"risk management status=%d body=%s",
			management.Code,
			management.Body.String(),
		)
	}
	for _, forbidden := range []string{
		"internalCaaNote", "Internal CAA Note", "private enforcement",
		"closureReason",
	} {
		if strings.Contains(management.Body.String(), forbidden) {
			t.Errorf("risk management raw wire leaked %q: %s", forbidden, management.Body.String())
		}
	}
	auditeeRisk := request(
		http.MethodGet, "/v1/risk/overview", "", "USR-AUDITEE-FLY",
	)
	if auditeeRisk.Code != http.StatusForbidden {
		t.Fatalf(
			"Auditee risk status=%d body=%s",
			auditeeRisk.Code,
			auditeeRisk.Body.String(),
		)
	}

	adminScreen := request(
		http.MethodGet,
		"/v1/administration/screens/admin-reports",
		"",
		"USR-ADMIN-ADA",
	)
	if adminScreen.Code != http.StatusOK ||
		!strings.Contains(adminScreen.Body.String(), "download-admin-report") {
		t.Fatalf(
			"admin screen status=%d body=%s",
			adminScreen.Code,
			adminScreen.Body.String(),
		)
	}
	screenSubjects := []struct {
		subjectID string
		count     int
	}{
		{"USR-INSPECTOR-DAVID", 12},
		{"USR-LEAD-CANER", 15},
		{"USR-MANAGER-NORA", 25},
		{"USR-FINANCE-LINA", 2},
		{"USR-GM-OMAR", 7},
		{"USR-ED-ZARA", 8},
		{"USR-AUDITEE-FLY", 9},
		{"USR-ADMIN-ADA", 14},
	}
	uniqueScreens := map[string]struct{}{}
	type declaredAction struct {
		ID     string          `json:"id"`
		Effect json.RawMessage `json:"effect"`
	}
	type declaredScreen struct {
		ScreenID       string           `json:"screenId"`
		VisibleActions []declaredAction `json:"visibleActions"`
	}
	type actionInvocation struct {
		subjectID string
		screenID  string
		action    declaredAction
	}
	uniqueActions := map[string]actionInvocation{}
	for _, screenSubject := range screenSubjects {
		response := request(
			http.MethodGet,
			"/v1/administration/screens",
			"",
			screenSubject.subjectID,
		)
		if response.Code != http.StatusOK {
			t.Fatalf(
				"screen registry subject=%s status=%d body=%s",
				screenSubject.subjectID,
				response.Code,
				response.Body.String(),
			)
		}
		var screens []declaredScreen
		if err := json.Unmarshal(response.Body.Bytes(), &screens); err != nil {
			t.Fatalf("decode screen registry for %s: %v", screenSubject.subjectID, err)
		}
		if len(screens) != screenSubject.count {
			t.Fatalf(
				"screen registry subject=%s count=%d, want %d",
				screenSubject.subjectID,
				len(screens),
				screenSubject.count,
			)
		}
		for _, screen := range screens {
			uniqueScreens[screen.ScreenID] = struct{}{}
			for _, action := range screen.VisibleActions {
				key := screen.ScreenID + "/" + action.ID
				if _, exists := uniqueActions[key]; !exists {
					uniqueActions[key] = actionInvocation{
						subjectID: screenSubject.subjectID,
						screenID:  screen.ScreenID,
						action:    action,
					}
				}
			}
		}
	}
	if len(uniqueScreens) != 85 || len(uniqueActions) != 107 {
		t.Fatalf(
			"HTTP Administration registry screens=%d actions=%d",
			len(uniqueScreens),
			len(uniqueActions),
		)
	}
	for key, invocation := range uniqueActions {
		idempotencyKey := "IDEM-TASK9-ACTION-" +
			strings.NewReplacer("/", "-", "_", "-").Replace(key)
		body := fmt.Sprintf(
			`{"operationId":%q,"expectedRevision":null,"idempotencyKey":%q,"screenId":%q,"actionId":%q}`,
			idempotencyKey,
			idempotencyKey,
			invocation.screenID,
			invocation.action.ID,
		)
		response := mutation(
			"/v1/administration/screens/"+
				invocation.screenID+
				"/actions/"+
				invocation.action.ID,
			body,
			invocation.subjectID,
			idempotencyKey,
		)
		if response.Code != http.StatusOK {
			t.Fatalf(
				"invoke visible action %s status=%d body=%s",
				key,
				response.Code,
				response.Body.String(),
			)
		}
		var output struct {
			ScreenID string          `json:"screenId"`
			ActionID string          `json:"actionId"`
			Effect   json.RawMessage `json:"effect"`
		}
		if err := json.Unmarshal(response.Body.Bytes(), &output); err != nil {
			t.Fatalf("decode visible action %s: %v", key, err)
		}
		if output.ScreenID != invocation.screenID ||
			output.ActionID != invocation.action.ID ||
			!jsonBytesEqual(output.Effect, invocation.action.Effect) {
			t.Fatalf(
				"visible action %s output=%s declared=%s",
				key,
				response.Body.String(),
				invocation.action.Effect,
			)
		}
	}
	reportDefinitions := request(
		http.MethodGet,
		"/v1/admin/report-definitions?search=due",
		"",
		"USR-ADMIN-ADA",
	)
	if reportDefinitions.Code != http.StatusOK ||
		!strings.Contains(reportDefinitions.Body.String(), "CAPS_BY_DUE_STATE") ||
		!strings.Contains(reportDefinitions.Body.String(), "no automatic closure") {
		t.Fatalf(
			"admin report definitions status=%d body=%s",
			reportDefinitions.Code,
			reportDefinitions.Body.String(),
		)
	}
	accessDirectory := request(
		http.MethodGet,
		"/v1/admin/access-directory?search=David&role=inspector",
		"",
		"USR-ADMIN-ADA",
	)
	if accessDirectory.Code != http.StatusOK ||
		!strings.Contains(accessDirectory.Body.String(), "USR-INSPECTOR-DAVID") ||
		!strings.Contains(accessDirectory.Body.String(), "david.inspector@example.test") ||
		!strings.Contains(accessDirectory.Body.String(), `"mfaEnrolled":true`) ||
		strings.Contains(accessDirectory.Body.String(), "Not configured in demo") {
		t.Fatalf(
			"admin access directory status=%d body=%s",
			accessDirectory.Code,
			accessDirectory.Body.String(),
		)
	}
	var firstDirectoryPage struct {
		Items            []map[string]any `json:"items"`
		NextCursor       *string          `json:"nextCursor"`
		ConsistencyToken string           `json:"consistencyToken"`
		ProviderCalls    int              `json:"providerCalls"`
	}
	firstPageResponse := request(
		http.MethodGet,
		"/v1/admin/access-directory?limit=1",
		"",
		"USR-ADMIN-ADA",
	)
	if firstPageResponse.Code != http.StatusOK ||
		json.Unmarshal(firstPageResponse.Body.Bytes(), &firstDirectoryPage) != nil ||
		len(firstDirectoryPage.Items) != 1 ||
		firstDirectoryPage.NextCursor == nil ||
		firstDirectoryPage.ConsistencyToken == "" ||
		firstDirectoryPage.ProviderCalls != 2 {
		t.Fatalf(
			"first directory page status=%d body=%s",
			firstPageResponse.Code,
			firstPageResponse.Body.String(),
		)
	}
	secondPageResponse := request(
		http.MethodGet,
		"/v1/admin/access-directory?limit=1&cursor="+*firstDirectoryPage.NextCursor,
		"",
		"USR-ADMIN-ADA",
	)
	if secondPageResponse.Code != http.StatusOK ||
		!strings.Contains(secondPageResponse.Body.String(), "USR-INSPECTOR-DAVID") ||
		!strings.Contains(
			secondPageResponse.Body.String(),
			firstDirectoryPage.ConsistencyToken,
		) {
		t.Fatalf(
			"second directory page status=%d body=%s",
			secondPageResponse.Code,
			secondPageResponse.Body.String(),
		)
	}
	for _, forbidden := range []string{
		"password",
		"totpSecret",
		"providerTokens",
		"Internal CAA Note",
	} {
		if strings.Contains(accessDirectory.Body.String(), forbidden) {
			t.Fatalf("access directory disclosed %q: %s", forbidden, accessDirectory.Body.String())
		}
	}
	auditeeDirectory := request(
		http.MethodGet,
		"/v1/admin/access-directory",
		"",
		"USR-AUDITEE-ELENA",
	)
	if auditeeDirectory.Code != http.StatusForbidden ||
		strings.Contains(auditeeDirectory.Body.String(), "david.inspector@example.test") {
		t.Fatalf(
			"auditee access directory status=%d body=%s",
			auditeeDirectory.Code,
			auditeeDirectory.Body.String(),
		)
	}
	unavailableAPI := httpapi.NewCanonicalAPI(httpapi.CanonicalAPIDependencies{
		Pool:              pool,
		Application:       testService(pool),
		DirectoryProvider: unavailableDirectoryProvider{},
		Clock:             func() time.Time { return canonicalNow },
	})
	unavailableHandler := httpapi.NewCanonicalTestBoundary(
		"task-9-token",
	).Protect(unavailableAPI.Handler())
	unavailableRequest := httptest.NewRequest(
		http.MethodGet,
		"/v1/admin/access-directory",
		nil,
	)
	unavailableRequest.Header.Set(httpapi.CanonicalTestTokenHeader, "task-9-token")
	unavailableRequest.Header.Set(httpapi.CanonicalTestSubjectHeader, "USR-ADMIN-ADA")
	unavailableResponse := httptest.NewRecorder()
	unavailableHandler.ServeHTTP(unavailableResponse, unavailableRequest)
	if unavailableResponse.Code != http.StatusServiceUnavailable ||
		!strings.Contains(unavailableResponse.Body.String(), "PROVIDER_UNAVAILABLE") ||
		strings.Contains(unavailableResponse.Body.String(), "items") {
		t.Fatalf(
			"unavailable provider status=%d body=%s",
			unavailableResponse.Code,
			unavailableResponse.Body.String(),
		)
	}
	organizations := request(
		http.MethodGet,
		"/v1/admin/organizations?search=Fly&organizationType=OPERATOR&status=ACTIVE",
		"",
		"USR-ADMIN-ADA",
	)
	if organizations.Code != http.StatusOK ||
		!strings.Contains(organizations.Body.String(), "ORG-FLY-NAMIBIA") {
		t.Fatalf(
			"admin organizations status=%d body=%s",
			organizations.Code,
			organizations.Body.String(),
		)
	}

	guidance := request(
		http.MethodGet, "/v1/assistant/guidance", "", "USR-INSPECTOR-DAVID",
	)
	if guidance.Code != http.StatusOK ||
		!strings.Contains(guidance.Body.String(), "enforcement action") {
		t.Fatalf(
			"assistant guidance status=%d body=%s",
			guidance.Code,
			guidance.Body.String(),
		)
	}
	const draftBody = `{
		"operationId":"OP-TASK9-ASSISTANT",
		"expectedRevision":null,
		"idempotencyKey":"IDEM-TASK9-ASSISTANT",
		"findingId":"FND-SKYCARGO-2026-099",
		"prompt":"Draft an evidence request only."
	}`
	draft := mutation(
		"/v1/assistant/drafts",
		draftBody,
		"USR-INSPECTOR-DAVID",
		"IDEM-TASK9-ASSISTANT",
	)
	if draft.Code != http.StatusCreated {
		t.Fatalf("assistant draft status=%d body=%s", draft.Code, draft.Body.String())
	}
	assertClosedJSONKeys(t, draft.Body.Bytes(), []string{
		"id", "findingId", "prompt", "draft", "advisoryOnly",
		"canCreateFinding", "canSetSeverity", "canCloseFinding",
	})
	for _, forbidden := range []string{
		"providerId", "generatedAt", "nonDecisionLabel",
		"organizationId", "severity", "status", "nextAction",
		"Internal CAA Note", "private enforcement",
	} {
		if strings.Contains(draft.Body.String(), forbidden) {
			t.Errorf("assistant raw wire leaked %q: %s", forbidden, draft.Body.String())
		}
	}
	replay := mutation(
		"/v1/assistant/drafts",
		draftBody,
		"USR-INSPECTOR-DAVID",
		"IDEM-TASK9-ASSISTANT",
	)
	if replay.Code != http.StatusCreated || replay.Body.String() != draft.Body.String() {
		t.Fatalf(
			"assistant replay status=%d body=%s, want=%s",
			replay.Code,
			replay.Body.String(),
			draft.Body.String(),
		)
	}
	managerDraft := mutation(
		"/v1/assistant/drafts",
		strings.ReplaceAll(
			draftBody,
			"OP-TASK9-ASSISTANT",
			"OP-TASK9-ASSISTANT-DENIED",
		),
		"USR-MANAGER-NORA",
		"IDEM-TASK9-ASSISTANT",
	)
	if managerDraft.Code != http.StatusForbidden {
		t.Fatalf(
			"manager assistant status=%d body=%s",
			managerDraft.Code,
			managerDraft.Body.String(),
		)
	}
	auditEvents := request(
		http.MethodGet,
		"/v1/admin/audit-events?action=assistant.advisory&actor=David&system=MANUAL&dateText=2026-07-21",
		"",
		"USR-ADMIN-ADA",
	)
	if auditEvents.Code != http.StatusOK ||
		!strings.Contains(auditEvents.Body.String(), "assistant.advisory_draft_generated") {
		t.Fatalf(
			"admin filtered audit status=%d body=%s",
			auditEvents.Code,
			auditEvents.Body.String(),
		)
	}
	var findingStatus, nextAction string
	var findingRevision int64
	if err := pool.QueryRow(context.Background(), `
		SELECT status, next_action, revision
		FROM findings
		WHERE id = 'FND-SKYCARGO-2026-099'
	`).Scan(&findingStatus, &nextAction, &findingRevision); err != nil {
		t.Fatalf("read HTTP assistant Finding authority: %v", err)
	}
	if findingStatus != "OPEN" ||
		nextAction != "SkyCargo Air to submit CAP" ||
		findingRevision != 1 {
		t.Fatalf(
			"HTTP assistant mutated Finding: status=%s next=%s revision=%d",
			findingStatus,
			nextAction,
			findingRevision,
		)
	}
}

func assertClosedJSONKeys(t *testing.T, body []byte, expected []string) {
	t.Helper()
	var value map[string]json.RawMessage
	if err := json.Unmarshal(body, &value); err != nil {
		t.Fatalf("decode closed JSON object: %v\n%s", err, body)
	}
	actual := make([]string, 0, len(value))
	for key := range value {
		actual = append(actual, key)
	}
	if fmt.Sprint(sortedStrings(actual)) != fmt.Sprint(sortedStrings(expected)) {
		t.Fatalf("closed JSON keys = %v, want %v\n%s", actual, expected, body)
	}
}

func sortedStrings(values []string) []string {
	output := append([]string(nil), values...)
	for left := 0; left < len(output); left++ {
		for right := left + 1; right < len(output); right++ {
			if output[right] < output[left] {
				output[left], output[right] = output[right], output[left]
			}
		}
	}
	return output
}

func jsonBytesEqual(left, right []byte) bool {
	var leftValue, rightValue any
	if json.Unmarshal(left, &leftValue) != nil ||
		json.Unmarshal(right, &rightValue) != nil {
		return false
	}
	return reflect.DeepEqual(leftValue, rightValue)
}

type canonicalDirectoryProvider struct{}

func (canonicalDirectoryProvider) ListDirectory(
	_ context.Context,
	query identity.ProviderDirectoryQuery,
) (identity.ProviderDirectoryPage, error) {
	users := []identity.ProviderDirectoryUser{
		{
			SubjectID: "inspector-cabin-001",
			Email:     "inspector.cabin@example.test", DisplayName: "Cabin Inspector",
			OrganizationID: "CAA", Enabled: true, TOTPConfigured: true,
			Roles: []identity.Role{identity.RoleInspector},
		},
		{
			SubjectID: "USR-INSPECTOR-DAVID",
			Email:     "david.inspector@example.test", DisplayName: "David Inspector",
			OrganizationID: "CAA", Enabled: true, TOTPConfigured: true,
			Roles: []identity.Role{identity.RoleInspector},
		},
		{
			SubjectID: "provider-prelogin-001",
			Email:     "prelogin.auditee@example.test", DisplayName: "Prelogin Auditee",
			OrganizationID: "airline-xyz", Enabled: true,
			RequiredActions: []string{"UPDATE_PASSWORD", "VERIFY_EMAIL"},
			Roles:           []identity.Role{identity.RoleAuditee},
		},
	}
	needle := strings.ToLower(strings.TrimSpace(query.Search))
	filtered := make([]identity.ProviderDirectoryUser, 0, len(users))
	for _, user := range users {
		if needle == "" ||
			strings.Contains(strings.ToLower(
				user.SubjectID+" "+user.Email+" "+user.DisplayName,
			), needle) {
			filtered = append(filtered, user)
		}
	}
	first := query.First
	if first > len(filtered) {
		first = len(filtered)
	}
	end := first + query.Limit
	if end > len(filtered) {
		end = len(filtered)
	}
	nextFirst := 0
	if end < len(filtered) {
		nextFirst = end
	}
	return identity.ProviderDirectoryPage{
		Users:         append([]identity.ProviderDirectoryUser(nil), filtered[first:end]...),
		NextFirst:     nextFirst,
		ProviderCalls: 1 + end - first,
	}, nil
}

type unavailableDirectoryProvider struct{}

func (unavailableDirectoryProvider) ListDirectory(
	context.Context,
	identity.ProviderDirectoryQuery,
) (identity.ProviderDirectoryPage, error) {
	return identity.ProviderDirectoryPage{}, identity.ErrProviderUnavailable
}
