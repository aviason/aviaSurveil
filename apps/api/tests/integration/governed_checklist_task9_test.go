//go:build canonicaltest

package integration_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/aviason/aviaSurveil/internal/application"
	"github.com/aviason/aviaSurveil/internal/checklistgovernance"
	"github.com/aviason/aviaSurveil/internal/identity"
	"github.com/aviason/aviaSurveil/internal/regulatory"
	"github.com/aviason/aviaSurveil/internal/testprofile"
)

func TestTask9SyntheticPublicationAndBlockedRealPilotHaveSeparatePersistedEffects(t *testing.T) {
	ctx := context.Background()
	service, manager, submitted := task6SubmittedCandidate(t, "task9_boundary")
	approved, err := service.Approve(ctx, manager, checklistgovernance.ReviewCommand{OperationID: "TASK9-SYNTHETIC-APPROVE", IdempotencyKey: "TASK9-SYNTHETIC-APPROVE", CandidateID: submitted.CandidateID, ExpectedRevision: submitted.Revision, ExpectedContentDigest: submitted.ContentDigest, Reason: "Approve only the explicit synthetic test profile."})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.Pool.Exec(ctx, `
		INSERT INTO organizations (id, legal_name, organization_type, status)
		VALUES ('ORG-TASK9-OTHER', 'Task 9 Other Organization', 'OPERATOR', 'ACTIVE');
		INSERT INTO regulated_targets (id, target_kind, organization_id)
		VALUES ('TARGET-TASK9-OTHER', 'ORGANIZATION', 'ORG-TASK9-OTHER');
		INSERT INTO organization_service_provider_scopes
			(id, organization_id, service_provider_type_id, authorization_identifier, status, effective_from, primary_target_id)
		VALUES ('SCOPE-TASK9-OTHER', 'ORG-TASK9-OTHER', 'AIR_OPERATOR', 'AOC-TASK9-OTHER', 'ACTIVE', '2025-01-01', 'TARGET-TASK9-OTHER')`); err != nil {
		t.Fatal(err)
	}
	published, err := service.Publish(ctx, manager, checklistgovernance.PublicationCommand{OperationID: "TASK9-SYNTHETIC-PUBLISH", IdempotencyKey: "TASK9-SYNTHETIC-PUBLISH", CandidateID: approved.CandidateID, ExpectedRevision: approved.Revision, ExpectedContentDigest: approved.ContentDigest, Reason: "Publish only the explicit synthetic test profile."})
	if err != nil {
		t.Fatal(err)
	}
	auditee := identity.Principal{SubjectID: "TASK9-AUDITEE", OrganizationID: "ORG-SYNTHETIC-AOC", Roles: []identity.Role{identity.RoleAuditee}}
	if _, err := service.ListQueue(ctx, auditee); !errors.Is(err, application.ErrForbidden) {
		t.Fatalf("Auditee queue error=%v, want forbidden", err)
	}
	if _, err := service.GetPublishedVersion(ctx, auditee, published.TemplateVersionID); !errors.Is(err, application.ErrForbidden) {
		t.Fatalf("Auditee published version error=%v, want forbidden", err)
	}
	if _, err := (regulatory.NewAdminService(service.Pool, service.Clock)).ListSources(ctx, auditee); !errors.Is(err, application.ErrForbidden) {
		t.Fatalf("Auditee source lineage error=%v, want forbidden", err)
	}
	workspace, err := testService(service.Pool).GetAuditeeWorkspace(ctx, auditee)
	if err != nil {
		t.Fatalf("Auditee workspace error=%v", err)
	}
	encodedWorkspace, err := json.Marshal(workspace)
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{
		approved.CandidateID, published.TemplateVersionID, "SOURCE-SYNTHETIC-OPS-AOC",
		"TASK9-SYNTHETIC-APPROVE", "TASK9-SYNTHETIC-PUBLISH",
		"AE-TASK9-SYNTHETIC-APPROVE", "AE-TASK9-SYNTHETIC-PUBLISH",
		"PUBDEC-TASK9-SYNTHETIC-PUBLISH", "MEM-TASK6-FOI",
		"SCOPE-TASK9-OTHER", "TARGET-TASK9-OTHER", "ORG-TASK9-OTHER",
	} {
		if strings.Contains(string(encodedWorkspace), forbidden) {
			t.Fatalf("Auditee workspace leaked governed internal fact %q: %s", forbidden, encodedWorkspace)
		}
	}
	if err := testprofile.BootstrapBlockedRealOPSAOCGenerationInputs(ctx, service.Pool); err != nil {
		t.Fatal(err)
	}
	var runs, decisions, publications, audits int
	if err := service.Pool.QueryRow(ctx, `SELECT (SELECT count(*) FROM regulatory_generation_runs), (SELECT count(*) FROM department_review_decisions), (SELECT count(*) FROM checklist_publication_decisions), (SELECT count(*) FROM audit_events)`).Scan(&runs, &decisions, &publications, &audits); err != nil {
		t.Fatal(err)
	}
	if err := (regulatory.ImportStore{Pool: service.Pool}).ValidateBlockedRealOPSAOCRequest(ctx, regulatory.RealOPSAOCGenerationRequest()); !errors.Is(err, regulatory.ErrBlockedAuthority) {
		t.Fatalf("real OPS/AOC validation error=%v", err)
	}
	var afterRuns, afterDecisions, afterPublications, afterAudits int
	if err := service.Pool.QueryRow(ctx, `SELECT (SELECT count(*) FROM regulatory_generation_runs), (SELECT count(*) FROM department_review_decisions), (SELECT count(*) FROM checklist_publication_decisions), (SELECT count(*) FROM audit_events)`).Scan(&afterRuns, &afterDecisions, &afterPublications, &afterAudits); err != nil {
		t.Fatal(err)
	}
	if afterRuns != runs || afterDecisions != decisions || afterPublications != publications || afterAudits != audits {
		t.Fatalf("blocked real pilot mutated lifecycle runs=%d/%d decisions=%d/%d publications=%d/%d audits=%d/%d", runs, afterRuns, decisions, afterDecisions, publications, afterPublications, audits, afterAudits)
	}
}
