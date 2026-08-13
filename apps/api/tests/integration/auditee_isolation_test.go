package integration_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/aviason/aviaSurveil/internal/application"
	"github.com/aviason/aviaSurveil/internal/identity"
)

func TestAuditeeListAndDirectIDProjectionAreOrganizationIsolatedAtRawJSONBoundary(t *testing.T) {
	pool := canonicalDatabase(t, "auditee_isolation")
	seedFinding(t, pool, "finding-xyz", "OPS-2026-001", "airline-xyz")
	seedFinding(t, pool, "finding-other", "OPS-2026-002", "airline-other")
	service := testService(pool)
	auditee := principal("auditee-xyz", "airline-xyz", "session-auditee", identity.RoleAuditee)

	items, err := service.ListFindings(context.Background(), auditee)
	if err != nil {
		t.Fatalf("list Auditee Findings: %v", err)
	}
	if len(items) != 1 || items[0].ID != "finding-xyz" {
		t.Fatalf("Auditee list = %+v", items)
	}
	raw, err := json.Marshal(items)
	if err != nil {
		t.Fatalf("marshal Auditee projection: %v", err)
	}
	for _, forbidden := range []string{
		"finding-other", "airline-other", "Internal CAA Note", "internalCaaNote", "internalRisk",
		"inspectorWorkload", "enforcementDeliberation", "unreleasedReport",
	} {
		if strings.Contains(string(raw), forbidden) {
			t.Errorf("Auditee raw JSON contains %q: %s", forbidden, raw)
		}
	}

	if _, err := service.GetFinding(context.Background(), auditee, "finding-other"); !errors.Is(err, application.ErrNotFound) {
		t.Fatalf("cross-organization direct-ID error = %v", err)
	}
	view, err := service.GetFinding(context.Background(), auditee, "finding-xyz")
	if err != nil || view.OrganizationID != "airline-xyz" {
		t.Fatalf("own direct-ID projection = %+v, err = %v", view, err)
	}
}

func TestAuditeeWorkspaceRawJSONUsesClosedSafeProjectionsAcrossDomainFamilies(t *testing.T) {
	pool := canonicalDatabase(t, "auditee_workspace")
	seedFinding(t, pool, "finding-workspace-xyz", "OPS-2026-030", "airline-xyz")
	seedFinding(t, pool, "finding-workspace-other", "OPS-2026-031", "airline-other")
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO cap_revisions (
			id, cap_id, finding_id, organization_id, revision, status, root_cause, corrective_action,
			preventive_action, target_completion_date, submitted_by_subject_id, submitted_at,
			responsible_person, comment_to_caa
		) VALUES (
			'cap-workspace-xyz-v1', 'cap-workspace-xyz', 'finding-workspace-xyz', 'airline-xyz', 1, 'SUBMITTED',
			'Roster handoff gap', 'Reconcile records', 'Monthly roster control', '2026-08-15', 'auditee-xyz', now(),
			'Training Manager', 'CAA-visible CAP comment'
		);
		INSERT INTO review_decisions (
			id, entity_type, entity_id, expected_revision, decision, reason, comment_to_auditee,
			internal_caa_note, decided_by_subject_id, decided_at
		) VALUES (
			'decision-workspace-cap', 'cap_revision', 'cap-workspace-xyz-v1', 1, 'ACCEPT', 'CAP accepted',
			'CAP accepted', 'SECRET_INTERNAL_CAP_NOTE', 'inspector-cabin-001', now()
		);
		INSERT INTO object_metadata (
			id, aggregate_type, aggregate_id, object_key, filename, declared_media_type,
			detected_media_type, sha256, size_bytes, scan_status
		) VALUES (
			'object-workspace-xyz', 'evidence_version', 'evidence-workspace-xyz-v1',
			'PRIVATE_OBJECT_KEY_SHOULD_NOT_LEAK', 'training-records.pdf', 'application/pdf',
			'application/pdf', 'sha256:workspace', 1200, 'CLEAN'
		);
		INSERT INTO evidence_versions (
			id, evidence_id, finding_id, organization_id, version, object_metadata_id, filename,
			media_type, sha256, size_bytes, status, submitted_by_subject_id, submitted_at, revision
		) VALUES (
			'evidence-workspace-xyz-v1', 'evidence-workspace-xyz', 'finding-workspace-xyz', 'airline-xyz', 1,
			'object-workspace-xyz', 'training-records.pdf', 'application/pdf', 'sha256:workspace', 1200,
			'CLEAN', 'auditee-xyz', now(), 1
		);
		INSERT INTO report_versions (id, report_id, inspection_id, version, status, snapshot)
		VALUES
			('report-workspace-released-v1', 'report-workspace-released', 'audit-cabin-001', 1, 'DRAFT', '{"public":"released","internalRisk":"SECRET_INTERNAL_REPORT_RISK"}'),
			('report-workspace-unreleased-v1', 'report-workspace-unreleased', 'audit-cabin-001', 1, 'DRAFT', '{"marker":"UNRELEASED_REPORT_SECRET"}');
		INSERT INTO report_approval_states (report_version_id, status, revision, issued_at)
		VALUES
			('report-workspace-released-v1', 'LOCKED', 4, now()),
			('report-workspace-unreleased-v1', 'GM_REVIEW', 2, NULL);
		INSERT INTO authorized_sync_changes (
			subject_id, organization_id, kind, entity_id, entity_revision, payload
		) VALUES
			('auditee-xyz', 'airline-xyz', 'finding', 'finding-workspace-xyz', 1, '{"internalCaaNote":"SECRET_SYNC_INTERNAL_NOTE"}'),
			('auditee-other', 'airline-other', 'finding', 'finding-workspace-other', 1, '{"marker":"OTHER_ORG_SYNC_SECRET"}')
	`); err != nil {
		t.Fatalf("seed Auditee workspace: %v", err)
	}
	service := testService(pool)
	auditee := principal("auditee-xyz", "airline-xyz", "session-auditee", identity.RoleAuditee)
	workspace, err := service.GetAuditeeWorkspace(context.Background(), auditee)
	if err != nil {
		t.Fatalf("get Auditee workspace: %v", err)
	}
	if len(workspace.CAPs) != 1 || workspace.CAPs[0].ReviewStatus != "ACCEPTED" || len(workspace.Evidence) != 1 || len(workspace.Reports) != 1 || workspace.Reports[0].ID != "report-workspace-released-v1" {
		t.Fatalf("Auditee workspace = %+v", workspace)
	}
	conflict := application.NewSafeConflict("checklist_response", "response-cabin-001", 1, 2)
	raw, err := json.Marshal(struct {
		Workspace application.AuditeeWorkspaceProjection `json:"workspace"`
		Conflict  application.ConflictProjection         `json:"conflict"`
	}{Workspace: workspace, Conflict: conflict})
	if err != nil {
		t.Fatalf("marshal Auditee workspace: %v", err)
	}
	for _, forbidden := range []string{
		"internalCaaNote", "internal_caa_note", "SECRET_INTERNAL_CAP_NOTE", "PRIVATE_OBJECT_KEY_SHOULD_NOT_LEAK",
		"SECRET_INTERNAL_REPORT_RISK", "UNRELEASED_REPORT_SECRET", "SECRET_SYNC_INTERNAL_NOTE",
		"OTHER_ORG_SYNC_SECRET", "airline-other", "finding-workspace-other", "inspector-cabin-001",
		"assignedInspector", "internalRisk", "inspectorWorkload", "enforcementDeliberation", "snapshot", "payload",
	} {
		if strings.Contains(string(raw), forbidden) {
			t.Errorf("Auditee workspace raw JSON contains %q: %s", forbidden, raw)
		}
	}
}
