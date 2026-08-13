package integration_test

import (
	"context"
	"testing"
	"time"

	"github.com/aviason/aviaSurveil/internal/application"
	"github.com/aviason/aviaSurveil/internal/caps"
	"github.com/aviason/aviaSurveil/internal/evidence"
	"github.com/aviason/aviaSurveil/internal/findings"
	"github.com/aviason/aviaSurveil/internal/identity"
)

func TestCAPSubmissionAndCAAReviewRemainSeparateAndPreserveSubmittedRevision(t *testing.T) {
	pool := canonicalDatabase(t, "cap_transition")
	seedFinding(t, pool, "finding-cap", "OPS-2026-020", "airline-xyz")
	service := testService(pool)

	submitted, err := service.SubmitCAP(context.Background(), principal("auditee-xyz", "airline-xyz", "session-auditee", identity.RoleAuditee), application.SubmitCAPCommand{
		OperationID: "op-cap-submit", CorrelationID: "corr-cap", FindingID: "finding-cap", ExpectedFindingRevision: 1,
		RootCause: "Training roster handoff gap.", CorrectiveAction: "Reconcile all current records.",
		PreventiveAction: "Add monthly roster control.", ResponsiblePerson: "Training Manager",
		TargetCompletionDate: time.Date(2026, time.August, 15, 0, 0, 0, 0, time.UTC), CommentToCAA: "CAP submitted for CAA review.",
	})
	if err != nil {
		t.Fatalf("submit CAP: %v", err)
	}
	if submitted.CAPStatus != caps.StatusSubmitted || submitted.FindingStatus != findings.StatusCAPSubmitted {
		t.Fatalf("CAP submission = %+v", submitted)
	}

	accepted, err := service.ReviewCAP(context.Background(), principal("inspector-cabin-001", "caa", "session-inspector", identity.RoleInspector), application.ReviewCAPCommand{
		OperationID: "op-cap-review", CorrelationID: "corr-cap", CAPRevisionID: submitted.CAPRevisionID,
		ExpectedCAPRevision: 1, FindingID: "finding-cap", ExpectedFindingRevision: 2,
		Decision: caps.DecisionAccept, CommentToAuditee: "CAP accepted; Evidence remains required.", InternalCAANote: "Internal review complete.",
	})
	if err != nil {
		t.Fatalf("review CAP: %v", err)
	}
	if accepted.CAPStatus != caps.StatusAccepted || accepted.FindingStatus != findings.StatusEvidenceRequired || accepted.FindingStatus == findings.StatusClosed {
		t.Fatalf("CAP acceptance = %+v", accepted)
	}

	var storedCAPStatus string
	var capCount, decisionCount int
	if err := pool.QueryRow(context.Background(), "SELECT status FROM cap_revisions WHERE id = $1", submitted.CAPRevisionID).Scan(&storedCAPStatus); err != nil {
		t.Fatalf("read immutable CAP revision: %v", err)
	}
	if storedCAPStatus != "SUBMITTED" {
		t.Fatalf("submitted CAP revision was overwritten to %q", storedCAPStatus)
	}
	if err := pool.QueryRow(context.Background(), "SELECT count(*) FROM cap_revisions WHERE finding_id = 'finding-cap'").Scan(&capCount); err != nil || capCount != 1 {
		t.Fatalf("CAP revision count = %d, err = %v", capCount, err)
	}
	if err := pool.QueryRow(context.Background(), "SELECT count(*) FROM review_decisions WHERE entity_type = 'cap_revision'").Scan(&decisionCount); err != nil || decisionCount != 1 {
		t.Fatalf("CAP decision count = %d, err = %v", decisionCount, err)
	}
}

func TestRejectedCAPResubmissionAppendsARevisionAndPreservesPriorContent(t *testing.T) {
	pool := canonicalDatabase(t, "cap_resubmission")
	seedFinding(t, pool, "finding-cap-resubmit", "OPS-2026-024", "airline-xyz")
	service := testService(pool)
	auditee := principal("auditee-xyz", "airline-xyz", "session-auditee", identity.RoleAuditee)
	first, err := service.SubmitCAP(context.Background(), auditee, application.SubmitCAPCommand{
		OperationID: "op-cap-first", CorrelationID: "corr-cap-resubmit", FindingID: "finding-cap-resubmit", ExpectedFindingRevision: 1,
		RootCause: "Original root cause", CorrectiveAction: "Original action", PreventiveAction: "Original prevention",
		ResponsiblePerson: "Training Manager", TargetCompletionDate: time.Date(2026, time.August, 15, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("submit first CAP: %v", err)
	}
	if _, err := service.ReviewCAP(context.Background(), principal("inspector-cabin-001", "caa", "session-inspector", identity.RoleInspector), application.ReviewCAPCommand{
		OperationID: "op-cap-reject", CorrelationID: "corr-cap-resubmit", CAPRevisionID: first.CAPRevisionID,
		ExpectedCAPRevision: 1, FindingID: "finding-cap-resubmit", ExpectedFindingRevision: 2,
		Decision: caps.DecisionReject, CommentToAuditee: "Root cause needs more detail.", InternalCAANote: "Prior revision retained.",
	}); err != nil {
		t.Fatalf("reject first CAP: %v", err)
	}
	second, err := service.SubmitCAP(context.Background(), auditee, application.SubmitCAPCommand{
		OperationID: "op-cap-second", CorrelationID: "corr-cap-resubmit", FindingID: "finding-cap-resubmit", ExpectedFindingRevision: 3,
		RootCause: "Revised root cause", CorrectiveAction: "Revised action", PreventiveAction: "Revised prevention",
		ResponsiblePerson: "Quality Manager", TargetCompletionDate: time.Date(2026, time.August, 20, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("submit revised CAP: %v", err)
	}
	if second.CAPID != first.CAPID || second.CAPRevision != 2 || second.CAPRevisionID == first.CAPRevisionID {
		t.Fatalf("CAP revision chain = first %+v, second %+v", first, second)
	}
	var revisions int
	var firstCause, secondCause string
	if err := pool.QueryRow(context.Background(), `
		SELECT count(*), min(root_cause) FILTER (WHERE revision = 1), min(root_cause) FILTER (WHERE revision = 2)
		FROM cap_revisions WHERE cap_id = $1
	`, first.CAPID).Scan(&revisions, &firstCause, &secondCause); err != nil {
		t.Fatalf("read CAP revision chain: %v", err)
	}
	if revisions != 2 || firstCause != "Original root cause" || secondCause != "Revised root cause" {
		t.Fatalf("CAP revision preservation = count %d, first %q, second %q", revisions, firstCause, secondCause)
	}
}

func TestEvidenceVerifiedAndAuthorizedClosureRemainDistinctAuditedPaths(t *testing.T) {
	t.Run("evidence verified", func(t *testing.T) {
		pool := canonicalDatabase(t, "evidence_close")
		seedFinding(t, pool, "finding-evidence", "OPS-2026-021", "airline-xyz")
		if _, err := pool.Exec(context.Background(), `
			UPDATE findings SET status = 'PENDING_CAA_REVIEW' WHERE id = 'finding-evidence'
		`); err != nil {
			t.Fatalf("seed Evidence review state: %v", err)
		}
		if _, err := pool.Exec(context.Background(), `
			INSERT INTO evidence_versions (
				id, evidence_id, finding_id, organization_id, version, filename, media_type, sha256, size_bytes,
				status, submitted_by_subject_id, submitted_at, revision
			) VALUES (
				'evidence-version-001', 'evidence-001', 'finding-evidence', 'airline-xyz', 1, 'training-records.pdf',
				'application/pdf', 'sha256:evidence-001', 1024, 'CLEAN', 'auditee-xyz', $1, 1
			)
		`, canonicalNow); err != nil {
			t.Fatalf("seed Evidence: %v", err)
		}
		service := testService(pool)
		result, err := service.ReviewEvidence(context.Background(), principal("inspector-cabin-001", "caa", "session-inspector", identity.RoleInspector), application.ReviewEvidenceCommand{
			OperationID: "op-evidence-close", CorrelationID: "corr-evidence", EvidenceVersionID: "evidence-version-001",
			ExpectedEvidenceVersionRevision: 1, FindingID: "finding-evidence", ExpectedFindingRevision: 1,
			Decision: evidence.DecisionClose, CommentToAuditee: "Evidence verified.", InternalCAANote: "Exact clean version reviewed.",
		})
		if err != nil {
			t.Fatalf("review Evidence: %v", err)
		}
		if result.FindingStatus != findings.StatusClosed || result.ClosureBasis != findings.ClosureBasisEvidenceVerified {
			t.Fatalf("Evidence closure = %+v", result)
		}
		var evidenceCount int
		if err := pool.QueryRow(context.Background(), "SELECT count(*) FROM evidence_versions WHERE evidence_id = 'evidence-001'").Scan(&evidenceCount); err != nil || evidenceCount != 1 {
			t.Fatalf("Evidence version count = %d, err = %v", evidenceCount, err)
		}
	})

	t.Run("authorized closure", func(t *testing.T) {
		pool := canonicalDatabase(t, "authorized_close")
		seedFinding(t, pool, "finding-authorized", "OPS-2026-022", "airline-xyz")
		if _, err := pool.Exec(context.Background(), `
			UPDATE findings SET status = 'PENDING_CLOSURE', next_action = 'CAA verifies closure path'
			WHERE id = 'finding-authorized'
		`); err != nil {
			t.Fatalf("seed pending closure state: %v", err)
		}
		service := testService(pool)
		result, err := service.AuthorizedCloseFinding(context.Background(), principal("manager-001", "caa", "session-manager", identity.RoleDepartmentManager), application.AuthorizedCloseFindingCommand{
			OperationID: "op-authorized-close", CorrelationID: "corr-authorized", FindingID: "finding-authorized",
			ExpectedFindingRevision: 1, Reason: "Alternate verification authorized and documented.",
		})
		if err != nil {
			t.Fatalf("authorized close: %v", err)
		}
		if result.FindingStatus != findings.StatusClosed || result.ClosureBasis != findings.ClosureBasisAuthorized {
			t.Fatalf("authorized closure = %+v", result)
		}
		var closureBasis string
		if err := pool.QueryRow(context.Background(), "SELECT closure_basis FROM audit_events WHERE operation_id = 'op-authorized-close'").Scan(&closureBasis); err != nil || closureBasis != string(findings.ClosureBasisAuthorized) {
			t.Fatalf("authorized closure audit basis = %q, err = %v", closureBasis, err)
		}
	})
}
