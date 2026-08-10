//go:build canonicaltest

package integration_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/application"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/caps"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/evidence"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/findings"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/httpapi"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/identity"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/platform/database"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/potentialfindings"
	fieldsync "github.com/MarlonJD/aviaSurveil360/apps/api/internal/sync"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/testprofile"
)

func TestFullFindingLifecycleAuthority(t *testing.T) {
	t.Run("Observation defaults to no CAP Evidence or Due Date", func(t *testing.T) {
		pool := canonicalDatabase(t, "finding_lifecycle_observation")
		service := testService(pool)
		result, err := service.ConvertPotentialFinding(
			context.Background(),
			principal("lead-001", "caa", "session-lead", identity.RoleLeadInspector),
			application.ConvertPotentialFindingCommand{
				OperationID:        "op-observation-convert",
				CorrelationID:      "corr-observation-convert",
				PotentialFindingID: "potential-cabin-001",
				ExpectedRevision:   1,
				Severity:           potentialfindings.SeverityObservation,
			},
		)
		if err != nil {
			t.Fatalf("convert Observation: %v", err)
		}
		var status string
		var capRequired, evidenceRequired bool
		var dueDate *string
		if err := pool.QueryRow(context.Background(), `
			SELECT status, cap_required, evidence_required, due_date::text
			FROM findings WHERE id = $1
		`, result.FindingID).Scan(
			&status,
			&capRequired,
			&evidenceRequired,
			&dueDate,
		); err != nil {
			t.Fatalf("read converted Observation: %v", err)
		}
		if status != string(findings.StatusPendingClosure) ||
			capRequired ||
			evidenceRequired ||
			dueDate != nil {
			t.Fatalf(
				"Observation defaults = status %q CAP %t Evidence %t Due Date %v",
				status,
				capRequired,
				evidenceRequired,
				dueDate,
			)
		}
	})

	t.Run("accepted CAP advances to closure review when Evidence is not required", func(t *testing.T) {
		pool := canonicalDatabase(t, "finding_lifecycle_cap_without_evidence")
		service := testService(pool)
		converted, err := service.ConvertPotentialFinding(
			context.Background(),
			principal("lead-001", "caa", "session-lead", identity.RoleLeadInspector),
			application.ConvertPotentialFindingCommand{
				OperationID:           "op-cap-only-convert",
				CorrelationID:         "corr-cap-only",
				PotentialFindingID:    "potential-cabin-001",
				ExpectedRevision:      1,
				Severity:              potentialfindings.SeverityObservation,
				CAPRequired:           true,
				EvidenceRequired:      false,
				RequirementsSpecified: true,
			},
		)
		if err != nil {
			t.Fatalf("convert CAP-only Observation: %v", err)
		}
		submitted, err := service.SubmitCAP(
			context.Background(),
			principal("auditee-xyz", "airline-xyz", "session-auditee", identity.RoleAuditee),
			application.SubmitCAPCommand{
				OperationID:             "op-cap-only-submit",
				CorrelationID:           "corr-cap-only",
				FindingID:               converted.FindingID,
				ExpectedFindingRevision: 1,
				RootCause:               "A configured control was not consistently applied.",
				CorrectiveAction:        "Apply and document the configured control.",
				PreventiveAction:        "Sample the configured control monthly.",
				ResponsiblePerson:       "Cabin Safety Manager",
				TargetCompletionDate:    canonicalNow.Add(30 * 24 * time.Hour),
				CommentToCAA:            "CAP-only Observation submitted for CAA review.",
			},
		)
		if err != nil {
			t.Fatalf("submit CAP-only Observation: %v", err)
		}
		_, err = service.ReviewCAP(
			context.Background(),
			principal(
				"inspector-cabin-001",
				"caa",
				"session-inspector",
				identity.RoleInspector,
			),
			application.ReviewCAPCommand{
				OperationID:             "op-cap-only-empty-comments",
				CorrelationID:           "corr-cap-only",
				CAPRevisionID:           submitted.CAPRevisionID,
				ExpectedCAPRevision:     submitted.CAPRevision,
				FindingID:               converted.FindingID,
				ExpectedFindingRevision: submitted.FindingRevision,
				Decision:                caps.DecisionAccept,
			},
		)
		if !errors.Is(err, application.ErrInvalid) {
			t.Fatalf("commentless CAP review error = %v", err)
		}
		_, err = service.ReviewCAP(
			context.Background(),
			principal(
				"manager-001",
				"caa",
				"session-manager",
				identity.RoleDepartmentManager,
			),
			application.ReviewCAPCommand{
				OperationID:             "op-cap-only-manager-review",
				CorrelationID:           "corr-cap-only",
				CAPRevisionID:           submitted.CAPRevisionID,
				ExpectedCAPRevision:     submitted.CAPRevision,
				FindingID:               converted.FindingID,
				ExpectedFindingRevision: submitted.FindingRevision,
				Decision:                caps.DecisionAccept,
				CommentToAuditee:        "Manager summary access is not CAP review authority.",
				InternalCAANote:         "The Inspector or Lead Inspector owns this decision.",
			},
		)
		if !errors.Is(err, application.ErrForbidden) {
			t.Fatalf("Department Manager CAP review error = %v", err)
		}
		accepted, err := service.ReviewCAP(
			context.Background(),
			principal(
				"inspector-cabin-001",
				"caa",
				"session-inspector",
				identity.RoleInspector,
			),
			application.ReviewCAPCommand{
				OperationID:             "op-cap-only-accept",
				CorrelationID:           "corr-cap-only",
				CAPRevisionID:           submitted.CAPRevisionID,
				ExpectedCAPRevision:     submitted.CAPRevision,
				FindingID:               converted.FindingID,
				ExpectedFindingRevision: submitted.FindingRevision,
				Decision:                caps.DecisionAccept,
				CommentToAuditee:        "CAP accepted; no Evidence was configured.",
				InternalCAANote:         "Proceed to the explicit closure review boundary.",
			},
		)
		if err != nil {
			t.Fatalf("accept CAP-only Observation: %v", err)
		}
		if accepted.FindingStatus != findings.StatusPendingClosure {
			t.Fatalf(
				"CAP-only acceptance status = %q, want %q",
				accepted.FindingStatus,
				findings.StatusPendingClosure,
			)
		}
	})

	t.Run("CAP review rejects an earlier immutable revision after resubmission", func(t *testing.T) {
		pool := canonicalDatabase(t, "finding_lifecycle_exact_cap")
		seedFinding(t, pool, "finding-exact-cap", "OPS-2026-094", "airline-xyz")
		if _, err := pool.Exec(context.Background(), `
			UPDATE findings
			SET status = 'CAP_SUBMITTED', cap_required = true,
			    evidence_required = true, next_action = 'CAA reviews CAP'
			WHERE id = 'finding-exact-cap'
		`); err != nil {
			t.Fatalf("seed exact CAP Finding state: %v", err)
		}
		if _, err := pool.Exec(context.Background(), `
			INSERT INTO cap_revisions (
				id, cap_id, finding_id, organization_id, revision, status,
				root_cause, corrective_action, preventive_action,
				target_completion_date, submitted_by_subject_id, submitted_at
			) VALUES
				(
					'cap-exact-r1', 'cap-exact', 'finding-exact-cap',
					'airline-xyz', 1, 'SUBMITTED', 'Initial root cause',
					'Initial corrective action', 'Initial preventive action',
					'2026-08-20', 'auditee-xyz', $1
				),
				(
					'cap-exact-r2', 'cap-exact', 'finding-exact-cap',
					'airline-xyz', 2, 'SUBMITTED', 'Revised root cause',
					'Revised corrective action', 'Revised preventive action',
					'2026-08-25', 'auditee-xyz', $1
				)
		`, canonicalNow); err != nil {
			t.Fatalf("seed exact CAP revisions: %v", err)
		}
		_, err := testService(pool).ReviewCAP(
			context.Background(),
			principal("lead-001", "caa", "session-lead", identity.RoleLeadInspector),
			application.ReviewCAPCommand{
				OperationID:             "op-review-earlier-cap",
				CorrelationID:           "corr-review-earlier-cap",
				CAPRevisionID:           "cap-exact-r1",
				ExpectedCAPRevision:     1,
				FindingID:               "finding-exact-cap",
				ExpectedFindingRevision: 1,
				Decision:                caps.DecisionAccept,
				CommentToAuditee:        "The earlier CAP revision must not be accepted.",
				InternalCAANote:         "A later immutable CAP revision exists.",
			},
		)
		if !errors.Is(err, application.ErrConflict) {
			t.Fatalf("earlier CAP review error = %v", err)
		}
		var status string
		var revision int64
		var decisions int
		if err := pool.QueryRow(context.Background(), `
			SELECT status, revision,
			       (SELECT count(*) FROM review_decisions
			        WHERE entity_type = 'cap_revision')
			FROM findings
			WHERE id = 'finding-exact-cap'
		`).Scan(&status, &revision, &decisions); err != nil {
			t.Fatalf("read earlier CAP review side effects: %v", err)
		}
		if status != "CAP_SUBMITTED" || revision != 1 || decisions != 0 {
			t.Fatalf(
				"earlier CAP review effects = status %s revision %d decisions %d",
				status,
				revision,
				decisions,
			)
		}
	})

	t.Run("Evidence-only Observation starts at Evidence submission authority", func(t *testing.T) {
		pool := canonicalDatabase(t, "finding_lifecycle_evidence_only")
		service := testService(pool)
		converted, err := service.ConvertPotentialFinding(
			context.Background(),
			principal("lead-001", "caa", "session-lead", identity.RoleLeadInspector),
			application.ConvertPotentialFindingCommand{
				OperationID:           "op-evidence-only-convert",
				CorrelationID:         "corr-evidence-only",
				PotentialFindingID:    "potential-cabin-001",
				ExpectedRevision:      1,
				Severity:              potentialfindings.SeverityObservation,
				CAPRequired:           false,
				EvidenceRequired:      true,
				RequirementsSpecified: true,
			},
		)
		if err != nil {
			t.Fatalf("convert Evidence-only Observation: %v", err)
		}
		var status, nextAction string
		if err := pool.QueryRow(context.Background(), `
			SELECT status, next_action
			FROM findings
			WHERE id = $1
		`, converted.FindingID).Scan(&status, &nextAction); err != nil {
			t.Fatalf("read Evidence-only Observation: %v", err)
		}
		if status != string(findings.StatusEvidenceRequired) ||
			nextAction != "Auditee submits Evidence" {
			t.Fatalf(
				"Evidence-only Observation = status %q next action %q",
				status,
				nextAction,
			)
		}
	})

	t.Run("only the exact latest scan-clean Evidence version can be reviewed", func(t *testing.T) {
		pool := canonicalDatabase(t, "finding_lifecycle_exact_evidence")
		seedReviewableEvidenceVersions(t, pool, "finding-exact-evidence", 2)
		service := testService(pool)
		_, err := service.ReviewEvidence(
			context.Background(),
			principal(
				"inspector-cabin-001",
				"caa",
				"session-inspector",
				identity.RoleInspector,
			),
			application.ReviewEvidenceCommand{
				OperationID:                     "op-review-stale-evidence",
				CorrelationID:                   "corr-review-stale-evidence",
				EvidenceVersionID:               "evidence-version-exact-1",
				ExpectedEvidenceVersionRevision: 1,
				FindingID:                       "finding-exact-evidence",
				ExpectedFindingRevision:         1,
				Decision:                        evidence.DecisionClose,
				CommentToAuditee:                "The earlier version must not close the Finding.",
				InternalCAANote:                 "A later immutable Evidence version exists.",
			},
		)
		if !errors.Is(err, application.ErrConflict) {
			t.Fatalf("earlier Evidence review error = %v", err)
		}
		assertNoFindingReviewSideEffects(
			t,
			pool,
			"finding-exact-evidence",
			string(findings.StatusPendingCAAReview),
		)
	})

	t.Run("Evidence decisions require separate public and Internal CAA comments", func(t *testing.T) {
		pool := canonicalDatabase(t, "finding_lifecycle_evidence_comments")
		seedReviewableEvidenceVersions(t, pool, "finding-evidence-comments", 1)
		service := testService(pool)
		_, err := service.ReviewEvidence(
			context.Background(),
			principal(
				"inspector-cabin-001",
				"caa",
				"session-inspector",
				identity.RoleInspector,
			),
			application.ReviewEvidenceCommand{
				OperationID:                     "op-review-without-comments",
				CorrelationID:                   "corr-review-without-comments",
				EvidenceVersionID:               "evidence-version-exact-1",
				ExpectedEvidenceVersionRevision: 1,
				FindingID:                       "finding-evidence-comments",
				ExpectedFindingRevision:         1,
				Decision:                        evidence.DecisionClose,
			},
		)
		if !errors.Is(err, application.ErrInvalid) {
			t.Fatalf("commentless Evidence review error = %v", err)
		}
		assertNoFindingReviewSideEffects(
			t,
			pool,
			"finding-evidence-comments",
			string(findings.StatusPendingCAAReview),
		)
	})

	t.Run("Auditee Evidence list and download fail closed before readiness disclosure", func(t *testing.T) {
		pool := canonicalDatabase(t, "finding_lifecycle_evidence_privacy")
		seedFinding(t, pool, "finding-other-empty", "OPS-2026-091", "airline-other")
		seedFinding(t, pool, "finding-other-pending", "OPS-2026-092", "airline-other")
		if _, err := pool.Exec(context.Background(), `
			INSERT INTO evidence_versions (
				id, evidence_id, finding_id, organization_id, version, filename,
				media_type, sha256, size_bytes, status, submitted_by_subject_id,
				submitted_at, revision
			) VALUES (
				'evidence-other-pending-v1', 'evidence-other-pending',
				'finding-other-pending', 'airline-other', 1, 'other-pending.pdf',
				'application/pdf', 'sha256:other-pending', 1024, 'PENDING',
				'auditee-other', $1, 1
			)
		`, canonicalNow); err != nil {
			t.Fatalf("seed cross-organization Evidence: %v", err)
		}
		uploadService := evidence.NewUploadService(
			pool,
			newMemoryObjectStore(),
			evidence.UploadServiceConfig{
				QuarantineBucket: "avia-quarantine",
				CanonicalBucket:  "avia-canonical",
				MaximumByteSize:  25 * 1024 * 1024,
				InstructionTTL:   10 * time.Minute,
				Clock:            func() time.Time { return canonicalNow },
			},
		)
		auditee := principal(
			"auditee-xyz",
			"airline-xyz",
			"session-auditee",
			identity.RoleAuditee,
		)
		if _, err := uploadService.ListVersions(
			context.Background(),
			auditee,
			"finding-other-empty",
		); !errors.Is(err, evidence.ErrEvidenceForbidden) {
			t.Fatalf("empty cross-organization Evidence list error = %v", err)
		}
		if _, err := uploadService.Download(
			context.Background(),
			auditee,
			"evidence-other-pending-v1",
		); !errors.Is(err, evidence.ErrEvidenceForbidden) {
			t.Fatalf("pending cross-organization Evidence download error = %v", err)
		}
	})

	t.Run("Potential Finding creation binds every exact Inspection Attachment", func(t *testing.T) {
		pool := canonicalDatabase(t, "finding_lifecycle_attachment_link")
		if err := testprofile.Reset(context.Background(), pool, canonicalNow); err != nil {
			t.Fatalf("reset canonical lifecycle profile: %v", err)
		}
		seedCanonicalTestProfileAttachmentPackage(t, pool)
		inspector, ok := testprofile.Principal(testprofile.CanonicalInspectorSubjectID)
		if !ok {
			t.Fatal("canonical Inspector principal is unavailable")
		}
		grant, err := fieldsync.NewGrantService(pool, fieldsync.GrantDependencies{
			Clock:       func() time.Time { return canonicalNow },
			IDGenerator: testprofile.NewGenerator().Next,
		}).Issue(context.Background(), inspector, fieldsync.CheckoutInput{
			OperationID:            "OP-TASK6-ATTACHMENT-CHECKOUT",
			CorrelationID:          "CORR-TASK6-ATTACHMENT-CHECKOUT",
			PackageID:              "PKG-TASK6-ATTACHMENT",
			ExpectedPackageVersion: 2,
			DeviceInstanceID:       "DEVICE-CANDIDATE-001",
		})
		if err != nil {
			t.Fatalf("checkout exact Potential Finding attachment package: %v", err)
		}
		if _, err := pool.Exec(context.Background(), `
			INSERT INTO checklist_responses (
				id, inspection_id, package_id, question_id,
				assigned_inspector_subject_id, response_value,
				comment_to_auditee, revision
			) VALUES (
				'RESP-TASK6-PBE-001', 'AUD-2026-001', 'PKG-TASK6-ATTACHMENT',
				'CAB-EMEQ-PBE-001', $1, 'NON_COMPLIANT',
				'PBE serviceability record is unavailable.', 1
			)
		`, testprofile.CanonicalInspectorSubjectID); err != nil {
			t.Fatalf("seed exact Potential Finding response: %v", err)
		}
		if _, err := pool.Exec(context.Background(), `
			INSERT INTO inspection_attachments (
				id, inspection_id, package_id, question_id,
				checklist_response_id, organization_id, created_by_subject_id,
				offline_grant_id, device_instance_id, file_name,
				declared_media_type, declared_size_bytes, declared_sha256,
				upload_state, scan_state, revision
			) VALUES (
				'ATT-TASK6-PBE-001', 'AUD-2026-001', 'PKG-TASK6-ATTACHMENT',
				'CAB-EMEQ-PBE-001', 'RESP-TASK6-PBE-001',
				'ORG-FLY-NAMIBIA', $1, $2, $3, 'pbe-position.png', 'image/png',
				67, 'sha256:task6-pbe', 'UPLOADED', 'CLEAN', 1
			)
		`, testprofile.CanonicalInspectorSubjectID, grant.ID, grant.DeviceInstanceID); err != nil {
			t.Fatalf("seed exact Potential Finding attachment: %v", err)
		}
		if _, err := pool.Exec(context.Background(), `
			INSERT INTO object_metadata (
				id, aggregate_type, aggregate_id, object_key, filename,
				declared_media_type, detected_media_type, sha256, size_bytes,
				scan_status, organization_id, bucket_name, object_state
			) VALUES (
				'OBJ-TASK6-PBE-001', 'INSPECTION_ATTACHMENT', 'ATT-TASK6-PBE-001',
				'inspection-attachments/task6/pbe-position.png', 'pbe-position.png',
				'image/png', 'image/png', 'sha256:task6-pbe', 67, 'CLEAN',
				'ORG-FLY-NAMIBIA', 'inspection-attachments', 'CANONICAL'
			);
			INSERT INTO inspection_attachment_versions (
				id, inspection_attachment_id, version, organization_id,
				source_object_metadata_id, file_name, media_type, sha256,
				size_bytes, submitted_by_subject_id, submitted_at
			) VALUES (
				'ATT-TASK6-PBE-001-V1', 'ATT-TASK6-PBE-001', 1,
				'ORG-FLY-NAMIBIA', 'OBJ-TASK6-PBE-001', 'pbe-position.png',
				'image/png', 'sha256:task6-pbe', 67,
				'154ec5ac-6f97-4f55-916f-d2f142fc6211', '2026-07-21T12:00:00Z'
			);
			UPDATE inspection_attachments
			SET object_metadata_id = 'OBJ-TASK6-PBE-001',
			    canonical_object_metadata_id = 'OBJ-TASK6-PBE-001',
			    current_version_id = 'ATT-TASK6-PBE-001-V1'
			WHERE id = 'ATT-TASK6-PBE-001'
		`); err != nil {
			t.Fatalf("seed immutable canonical Inspection Attachment version: %v", err)
		}
		api := httpapi.NewCanonicalAPI(httpapi.CanonicalAPIDependencies{
			Pool:        pool,
			Application: testService(pool),
			Clock:       func() time.Time { return canonicalNow },
		})
		handler := httpapi.NewCanonicalTestBoundary("task-6-token").Protect(api.Handler())
		const requestBody = `{
				"operationId":"OP-TASK6-PF-ATTACHMENT",
				"auditId":"AUD-2026-001",
				"questionId":"CAB-EMEQ-PBE-001",
				"checklistResponseId":"RESP-TASK6-PBE-001",
				"expectedChecklistResponseRevision":1,
				"title":"PBE serviceability record gap",
				"description":"The configured PBE record was unavailable.",
				"requiredComment":"Provide the exact serviceability record.",
				"inspectionAttachmentIds":["ATT-TASK6-PBE-001"]
			}`
		newRequest := func() *http.Request {
			request := httptest.NewRequest(
				http.MethodPost,
				"/v1/potential-findings",
				strings.NewReader(requestBody),
			)
			request.Header.Set(httpapi.CanonicalTestTokenHeader, "task-6-token")
			request.Header.Set(
				httpapi.CanonicalTestSubjectHeader,
				testprofile.CanonicalInspectorSubjectID,
			)
			request.Header.Set("Content-Type", "application/json")
			return request
		}
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, newRequest())
		if response.Code != http.StatusCreated {
			t.Fatalf(
				"create Potential Finding status=%d body=%s",
				response.Code,
				response.Body.String(),
			)
		}
		var potentialFindingID *string
		if err := pool.QueryRow(context.Background(), `
			SELECT potential_finding_id
			FROM inspection_attachments
			WHERE id = 'ATT-TASK6-PBE-001'
		`).Scan(&potentialFindingID); err != nil {
			t.Fatalf("read linked Inspection Attachment: %v", err)
		}
		if potentialFindingID == nil || *potentialFindingID == "" {
			t.Fatal("Inspection Attachment was not linked to the Potential Finding")
		}
		replay := httptest.NewRecorder()
		handler.ServeHTTP(replay, newRequest())
		if replay.Code != http.StatusCreated || replay.Body.String() != response.Body.String() {
			t.Fatalf(
				"Potential Finding replay status=%d body=%s, want body=%s",
				replay.Code,
				replay.Body.String(),
				response.Body.String(),
			)
		}
		var potentialCount, auditCount, outboxCount int
		if err := pool.QueryRow(context.Background(), `
			SELECT
				(SELECT count(*) FROM potential_findings
				 WHERE checklist_response_id = 'RESP-TASK6-PBE-001'),
				(SELECT count(*) FROM audit_events
				 WHERE operation_id = 'OP-TASK6-PF-ATTACHMENT'),
				(SELECT count(*) FROM outbox_messages
				 WHERE topic = 'potential_finding.created'
				   AND aggregate_id = $1)
		`, *potentialFindingID).Scan(
			&potentialCount,
			&auditCount,
			&outboxCount,
		); err != nil {
			t.Fatalf("read Potential Finding replay effects: %v", err)
		}
		if potentialCount != 1 || auditCount != 1 || outboxCount != 1 {
			t.Fatalf(
				"Potential Finding replay effects = potential %d audit %d outbox %d",
				potentialCount,
				auditCount,
				outboxCount,
			)
		}
	})

	t.Run("CAP reads derive immutable review status and keep Internal CAA Note private", func(t *testing.T) {
		pool := canonicalDatabase(t, "finding_lifecycle_cap_raw_wire")
		if err := testprofile.Reset(context.Background(), pool, canonicalNow); err != nil {
			t.Fatalf("reset canonical CAP raw-wire profile: %v", err)
		}
		if _, err := pool.Exec(context.Background(), `
			UPDATE report_approval_states
			SET status = 'LOCKED', revision = 4, issued_at = $1, updated_at = $1
			WHERE report_version_id = 'PR-2026-018-V1'
		`, canonicalNow); err != nil {
			t.Fatalf("seed issued Preliminary Report before CAP: %v", err)
		}
		if _, err := pool.Exec(context.Background(), `
			INSERT INTO findings (
				id, reference, inspection_id, organization_id, severity, status,
				owner_subject_id, next_action, due_date, revision, cap_required,
				evidence_required, issued_at, created_at, updated_at
			) VALUES (
				'FND-TASK6-CAP-WIRE', 'CAR-TASK6-CAP-WIRE', 'AUD-2026-001',
				'ORG-FLY-NAMIBIA', 'LEVEL_2_MAJOR', 'WAITING_FOR_CAP',
				'USR-AUDITEE-FLY', 'Auditee to submit CAP', '2026-08-15', 1,
				true, true, $1, $1, $1
			)
		`, canonicalNow); err != nil {
			t.Fatalf("seed CAP raw-wire Finding: %v", err)
		}
		service := testService(pool)
		auditee, _ := testprofile.Principal("USR-AUDITEE-FLY")
		submitted, err := service.SubmitCAP(
			context.Background(),
			auditee,
			application.SubmitCAPCommand{
				OperationID:             "op-cap-wire-submit",
				CorrelationID:           "corr-cap-wire",
				FindingID:               "FND-TASK6-CAP-WIRE",
				ExpectedFindingRevision: 1,
				RootCause:               "The handoff control was incomplete.",
				CorrectiveAction:        "Complete and record the handoff control.",
				PreventiveAction:        "Sample the handoff control monthly.",
				ResponsiblePerson:       "Cabin Safety Manager",
				TargetCompletionDate:    canonicalNow.Add(30 * 24 * time.Hour),
				CommentToCAA:            "CAP submitted for exact raw-wire review.",
			},
		)
		if err != nil {
			t.Fatalf("submit raw-wire CAP: %v", err)
		}
		lead, _ := testprofile.Principal("USR-LEAD-CANER")
		reviewed, err := service.ReviewCAP(
			context.Background(),
			lead,
			application.ReviewCAPCommand{
				OperationID:             "op-cap-wire-more-info",
				CorrelationID:           "corr-cap-wire",
				CAPRevisionID:           submitted.CAPRevisionID,
				ExpectedCAPRevision:     submitted.CAPRevision,
				FindingID:               "FND-TASK6-CAP-WIRE",
				ExpectedFindingRevision: submitted.FindingRevision,
				Decision:                caps.DecisionRequestInformation,
				CommentToAuditee:        "Clarify the preventive-action sampling evidence.",
				InternalCAANote:         "SECRET_TASK6_INTERNAL_CAP_NOTE",
			},
		)
		if err != nil {
			t.Fatalf("review raw-wire CAP: %v", err)
		}
		api := httpapi.NewCanonicalAPI(httpapi.CanonicalAPIDependencies{
			Pool: pool, Application: service, Clock: func() time.Time { return canonicalNow },
		})
		handler := httpapi.NewCanonicalTestBoundary("task-6-cap-wire").Protect(api.Handler())
		getCAP := func(subjectID string) *httptest.ResponseRecorder {
			request := httptest.NewRequest(
				http.MethodGet,
				"/v1/cap-revisions/"+submitted.CAPRevisionID,
				nil,
			)
			request.Header.Set(httpapi.CanonicalTestTokenHeader, "task-6-cap-wire")
			request.Header.Set(httpapi.CanonicalTestSubjectHeader, subjectID)
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			return response
		}
		caaResponse := getCAP("USR-LEAD-CANER")
		if caaResponse.Code != http.StatusOK ||
			!strings.Contains(caaResponse.Body.String(), `"status":"MORE_INFORMATION_REQUESTED"`) ||
			!strings.Contains(caaResponse.Body.String(), "SECRET_TASK6_INTERNAL_CAP_NOTE") {
			t.Fatalf(
				"CAA CAP raw wire status=%d body=%s",
				caaResponse.Code,
				caaResponse.Body.String(),
			)
		}
		auditeeResponse := getCAP("USR-AUDITEE-FLY")
		if auditeeResponse.Code != http.StatusOK ||
			!strings.Contains(auditeeResponse.Body.String(), `"status":"MORE_INFORMATION_REQUESTED"`) ||
			strings.Contains(auditeeResponse.Body.String(), "SECRET_TASK6_INTERNAL_CAP_NOTE") ||
			strings.Contains(auditeeResponse.Body.String(), "internalCaaNote") {
			t.Fatalf(
				"Auditee CAP raw wire status=%d body=%s",
				auditeeResponse.Code,
				auditeeResponse.Body.String(),
			)
		}
		if _, err := service.SubmitCAP(
			context.Background(),
			auditee,
			application.SubmitCAPCommand{
				OperationID:             "op-cap-wire-resubmit",
				CorrelationID:           "corr-cap-wire",
				FindingID:               "FND-TASK6-CAP-WIRE",
				ExpectedFindingRevision: reviewed.FindingRevision,
				RootCause:               "The revised handoff control is complete.",
				CorrectiveAction:        "Record the revised handoff control.",
				PreventiveAction:        "Sample and retain the revised control monthly.",
				ResponsiblePerson:       "Cabin Safety Manager",
				TargetCompletionDate:    canonicalNow.Add(35 * 24 * time.Hour),
				CommentToCAA:            "Revised CAP submitted for exact raw-wire review.",
			},
		); err != nil {
			t.Fatalf("resubmit raw-wire CAP: %v", err)
		}
		for _, response := range []*httptest.ResponseRecorder{
			getCAP("USR-LEAD-CANER"),
			getCAP("USR-AUDITEE-FLY"),
		} {
			if response.Code != http.StatusOK ||
				!strings.Contains(response.Body.String(), `"status":"SUPERSEDED"`) {
				t.Fatalf(
					"superseded CAP raw wire status=%d body=%s",
					response.Code,
					response.Body.String(),
				)
			}
		}
		if _, err := pool.Exec(context.Background(), `
			INSERT INTO cap_revisions (
				id, cap_id, finding_id, organization_id, revision, status,
				root_cause, corrective_action, preventive_action,
				target_completion_date, submitted_by_subject_id, submitted_at,
				responsible_person, comment_to_caa
			) VALUES (
				'CAP-TASK6-OTHER-R1', 'CAP-TASK6-OTHER',
				'FND-SKYCARGO-2026-099', 'ORG-SKYCARGO', 1, 'SUBMITTED',
				'OTHER_ORG_SECRET_ROOT_CAUSE', 'Other corrective action',
				'Other preventive action', '2026-08-30', 'USR-AUDITEE-FLY',
				$1, 'Other manager', 'Other CAA comment'
			)
		`, canonicalNow); err != nil {
			t.Fatalf("seed cross-organization CAP: %v", err)
		}
		otherRequest := httptest.NewRequest(
			http.MethodGet,
			"/v1/cap-revisions/CAP-TASK6-OTHER-R1",
			nil,
		)
		otherRequest.Header.Set(httpapi.CanonicalTestTokenHeader, "task-6-cap-wire")
		otherRequest.Header.Set(httpapi.CanonicalTestSubjectHeader, "USR-AUDITEE-FLY")
		otherResponse := httptest.NewRecorder()
		handler.ServeHTTP(otherResponse, otherRequest)
		if otherResponse.Code != http.StatusNotFound ||
			strings.Contains(otherResponse.Body.String(), "OTHER_ORG_SECRET") {
			t.Fatalf(
				"cross-organization CAP raw wire status=%d body=%s",
				otherResponse.Code,
				otherResponse.Body.String(),
			)
		}
	})

	t.Run("HTTP lifecycle mutations reject path and body identity drift", func(t *testing.T) {
		pool := canonicalDatabase(t, "finding_lifecycle_http_identity_drift")
		if err := testprofile.Reset(context.Background(), pool, canonicalNow); err != nil {
			t.Fatalf("reset canonical identity-drift profile: %v", err)
		}
		if _, err := pool.Exec(context.Background(), `
			INSERT INTO findings (
				id, reference, inspection_id, organization_id, severity, status,
				next_action, revision, cap_required, evidence_required,
				issued_at, created_at, updated_at
			) VALUES
				('FND-TASK6-CAP-DRIFT', 'CAR-TASK6-CAP-DRIFT', 'AUD-2026-001',
				 'ORG-FLY-NAMIBIA', 'LEVEL_2_MAJOR', 'CAP_SUBMITTED',
				 'CAA reviews CAP', 1, true, true, $1, $1, $1),
				('FND-TASK6-EVIDENCE-DRIFT', 'CAR-TASK6-EVIDENCE-DRIFT', 'AUD-2026-001',
				 'ORG-FLY-NAMIBIA', 'LEVEL_2_MAJOR', 'PENDING_CAA_REVIEW',
				 'CAA reviews Evidence', 1, true, true, $1, $1, $1),
				('FND-TASK6-CLOSE-DRIFT', 'CAR-TASK6-CLOSE-DRIFT', 'AUD-2026-001',
				 'ORG-FLY-NAMIBIA', 'LEVEL_2_MAJOR', 'PENDING_CLOSURE',
				 'CAA completes verification', 1, false, false, $1, $1, $1)
		`, canonicalNow); err != nil {
			t.Fatalf("seed lifecycle identity-drift Findings: %v", err)
		}
		if _, err := pool.Exec(context.Background(), `
			INSERT INTO cap_revisions (
				id, cap_id, finding_id, organization_id, revision, status,
				root_cause, corrective_action, preventive_action,
				target_completion_date, submitted_by_subject_id, submitted_at
			) VALUES (
				'CAP-TASK6-DRIFT-R1', 'CAP-TASK6-DRIFT',
				'FND-TASK6-CAP-DRIFT', 'ORG-FLY-NAMIBIA', 1, 'SUBMITTED',
				'Root cause', 'Corrective action', 'Preventive action',
				'2026-08-30', 'USR-AUDITEE-FLY', $1
			)
		`, canonicalNow); err != nil {
			t.Fatalf("seed lifecycle identity-drift CAP: %v", err)
		}
		if _, err := pool.Exec(context.Background(), `
			INSERT INTO evidence_versions (
				id, evidence_id, finding_id, organization_id, version, filename,
				media_type, sha256, size_bytes, status, submitted_by_subject_id,
				submitted_at, revision
			) VALUES (
				'EVD-TASK6-DRIFT-V1', 'EVD-TASK6-DRIFT',
				'FND-TASK6-EVIDENCE-DRIFT', 'ORG-FLY-NAMIBIA', 1,
				'evidence.pdf', 'application/pdf', 'sha256:task6-drift', 1024,
				'CLEAN', 'USR-AUDITEE-FLY', $1, 1
			)
		`, canonicalNow); err != nil {
			t.Fatalf("seed lifecycle identity-drift Evidence: %v", err)
		}
		api := httpapi.NewCanonicalAPI(httpapi.CanonicalAPIDependencies{
			Pool: pool, Application: testService(pool), Clock: func() time.Time { return canonicalNow },
		})
		handler := httpapi.NewCanonicalTestBoundary("task-6-identity-drift").Protect(api.Handler())
		for _, test := range []struct {
			name      string
			path      string
			subjectID string
			body      string
		}{
			{
				name:      "CAP review",
				path:      "/v1/caps/CAP-PATH-OTHER/reviews",
				subjectID: "USR-LEAD-CANER",
				body: `{
					"operationId":"OP-TASK6-CAP-DRIFT",
					"capRevisionId":"CAP-TASK6-DRIFT-R1",
					"expectedCapRevision":1,
					"findingId":"FND-TASK6-CAP-DRIFT",
					"expectedFindingRevision":1,
					"decision":"ACCEPT",
					"commentToAuditee":"CAP accepted.",
					"internalCaaNote":"Exact CAP reviewed."
				}`,
			},
			{
				name:      "Evidence review",
				path:      "/v1/evidence/EVD-PATH-OTHER/reviews",
				subjectID: "USR-LEAD-CANER",
				body: `{
					"operationId":"OP-TASK6-EVIDENCE-DRIFT",
					"evidenceVersionId":"EVD-TASK6-DRIFT-V1",
					"expectedEvidenceVersionRevision":1,
					"findingId":"FND-TASK6-EVIDENCE-DRIFT",
					"expectedFindingRevision":1,
					"decision":"CLOSE",
					"commentToAuditee":"Evidence accepted and verified.",
					"internalCaaNote":"Exact Evidence reviewed."
				}`,
			},
			{
				name:      "authorized closure",
				path:      "/v1/findings/FND-PATH-OTHER/authorized-closure",
				subjectID: "USR-MANAGER-NORA",
				body: `{
					"operationId":"OP-TASK6-CLOSE-DRIFT",
					"findingId":"FND-TASK6-CLOSE-DRIFT",
					"expectedFindingRevision":1,
					"reason":"Authorized alternate verification."
				}`,
			},
		} {
			t.Run(test.name, func(t *testing.T) {
				request := httptest.NewRequest(
					http.MethodPost,
					test.path,
					strings.NewReader(test.body),
				)
				request.Header.Set(
					httpapi.CanonicalTestTokenHeader,
					"task-6-identity-drift",
				)
				request.Header.Set(httpapi.CanonicalTestSubjectHeader, test.subjectID)
				request.Header.Set("Content-Type", "application/json")
				response := httptest.NewRecorder()
				handler.ServeHTTP(response, request)
				if response.Code != http.StatusUnprocessableEntity {
					t.Fatalf(
						"path/body drift status=%d body=%s",
						response.Code,
						response.Body.String(),
					)
				}
			})
		}
		var reviews, closures int
		if err := pool.QueryRow(context.Background(), `
			SELECT
				(SELECT count(*) FROM review_decisions
				 WHERE entity_id IN ('CAP-TASK6-DRIFT-R1', 'EVD-TASK6-DRIFT-V1')),
				(SELECT count(*) FROM findings
				 WHERE id = 'FND-TASK6-CLOSE-DRIFT' AND status = 'CLOSED')
		`).Scan(&reviews, &closures); err != nil {
			t.Fatalf("read identity-drift side effects: %v", err)
		}
		if reviews != 0 || closures != 0 {
			t.Fatalf(
				"identity-drift side effects = reviews %d closures %d",
				reviews,
				closures,
			)
		}
	})

	t.Run("Evidence upload mutations link exact transactional envelopes", func(t *testing.T) {
		pool := canonicalDatabase(t, "finding_lifecycle_upload_envelopes")
		seedFinding(t, pool, "finding-upload-envelope", "OPS-2026-093", "airline-xyz")
		if _, err := pool.Exec(
			context.Background(),
			"UPDATE findings SET status = 'EVIDENCE_REQUIRED' WHERE id = 'finding-upload-envelope'",
		); err != nil {
			t.Fatalf("seed Evidence-required Finding: %v", err)
		}
		objects := newMemoryObjectStore()
		uploadService := evidence.NewUploadService(
			pool,
			objects,
			evidence.UploadServiceConfig{
				QuarantineBucket: "avia-quarantine",
				CanonicalBucket:  "avia-canonical",
				MaximumByteSize:  25 * 1024 * 1024,
				InstructionTTL:   10 * time.Minute,
				Clock:            uploadClock,
				IDGenerator:      deterministicIDs(),
			},
		)
		auditee := principal(
			"auditee-xyz",
			"airline-xyz",
			"session-auditee",
			identity.RoleAuditee,
		)
		body := validPDF("task-six-transaction-envelope")
		digest := sha256Digest(body)
		beginInput := evidence.BeginUploadInput{
			OperationID:             "op-task6-evidence-begin",
			CorrelationID:           "corr-task6-evidence-upload",
			FindingID:               "finding-upload-envelope",
			ExpectedFindingRevision: 1,
			FileName:                "task-six-evidence.pdf",
			DeclaredMediaType:       "application/pdf",
			ByteSize:                int64(len(body)),
			SHA256:                  digest,
		}
		begin, err := uploadService.Begin(context.Background(), auditee, beginInput)
		if err != nil {
			t.Fatalf("begin Evidence upload: %v", err)
		}
		objects.Seed(
			"avia-quarantine",
			begin.StagingObjectKey,
			"application/pdf",
			body,
			map[string]string{"sha256": digest},
		)
		completeInput := evidence.CompleteUploadInput{
			OperationID:   "op-task6-evidence-complete",
			CorrelationID: "corr-task6-evidence-upload",
			UploadID:      begin.UploadID,
			SHA256:        digest,
			ByteSize:      int64(len(body)),
		}
		completed, err := uploadService.Complete(
			context.Background(),
			auditee,
			completeInput,
		)
		if err != nil {
			t.Fatalf("complete Evidence upload: %v", err)
		}

		envelopes := []struct {
			name          string
			scope         string
			operationID   string
			correlationID string
			action        string
			changeKind    string
			topic         string
			entityID      string
		}{
			{
				name:          "begin",
				scope:         "auditee-xyz:begin_evidence_upload",
				operationID:   beginInput.OperationID,
				correlationID: beginInput.CorrelationID,
				action:        "evidence.upload_started",
				changeKind:    "evidence_upload_session",
				topic:         "evidence.upload_started",
				entityID:      begin.UploadID,
			},
			{
				name:          "complete",
				scope:         "auditee-xyz:complete_evidence_upload",
				operationID:   completeInput.OperationID,
				correlationID: completeInput.CorrelationID,
				action:        "evidence.uploaded",
				changeKind:    "evidence_version",
				topic:         "evidence.scan_requested",
				entityID:      completed.EvidenceVersionID,
			},
		}
		for _, envelope := range envelopes {
			var count int
			if err := pool.QueryRow(context.Background(), `
				SELECT count(*)
				FROM command_transaction_links link
				JOIN idempotency_responses response
				  ON response.scope = link.idempotency_scope
				 AND response.operation_id = link.operation_id
				JOIN audit_events audit
				  ON audit.event_id = link.audit_event_id
				JOIN authorized_sync_changes sync_change
				  ON sync_change.sequence_id = link.change_sequence_id
				JOIN outbox_messages outbox
				  ON outbox.id = link.outbox_message_id
				WHERE link.idempotency_scope = $1
				  AND link.operation_id = $2
				  AND response.response_status = 200
				  AND audit.action = $3
				  AND audit.entity_id = $4
				  AND audit.operation_id = $2
				  AND audit.correlation_id = $5
				  AND sync_change.kind = $6
				  AND sync_change.entity_id = $4
				  AND sync_change.operation_id = $2
				  AND sync_change.correlation_id = $5
				  AND outbox.topic = $7
				  AND outbox.aggregate_id = $4
				  AND outbox.operation_id = $2
				  AND outbox.correlation_id = $5
			`,
				envelope.scope,
				envelope.operationID,
				envelope.action,
				envelope.entityID,
				envelope.correlationID,
				envelope.changeKind,
				envelope.topic,
			).Scan(&count); err != nil {
				t.Fatalf("read %s Evidence upload transaction envelope: %v", envelope.name, err)
			}
			if count != 1 {
				t.Fatalf("%s Evidence upload transaction envelopes = %d, want 1", envelope.name, count)
			}
		}
	})

	t.Run("concurrent Lead decisions permit one transition and exact replay", func(t *testing.T) {
		pool := canonicalDatabase(t, "finding_lifecycle_concurrent_lead")
		var identifier atomic.Int64
		service := application.NewService(pool, application.Dependencies{
			Clock: func() time.Time { return canonicalNow },
			IDGenerator: func(prefix string) string {
				return fmt.Sprintf("%s-concurrent-%03d", prefix, identifier.Add(1))
			},
		})
		lead := principal("lead-001", "caa", "session-lead", identity.RoleLeadInspector)
		commands := []application.DecidePotentialFindingCommand{
			{
				OperationID:        "op-concurrent-return",
				CorrelationID:      "corr-concurrent-lead",
				PotentialFindingID: "potential-cabin-001",
				ExpectedRevision:   1,
				Decision:           potentialfindings.DecisionReturn,
				Reason:             "Clarify the exact configured requirement.",
			},
			{
				OperationID:        "op-concurrent-dismiss",
				CorrelationID:      "corr-concurrent-lead",
				PotentialFindingID: "potential-cabin-001",
				ExpectedRevision:   1,
				Decision:           potentialfindings.DecisionDismiss,
				Reason:             "The documented exception is not supported.",
			},
		}
		type outcome struct {
			command application.DecidePotentialFindingCommand
			result  application.PotentialFindingResult
			err     error
		}
		start := make(chan struct{})
		outcomes := make(chan outcome, len(commands))
		var group sync.WaitGroup
		for _, command := range commands {
			command := command
			group.Add(1)
			go func() {
				defer group.Done()
				<-start
				result, err := service.DecidePotentialFinding(
					context.Background(),
					lead,
					command,
				)
				outcomes <- outcome{command: command, result: result, err: err}
			}()
		}
		close(start)
		group.Wait()
		close(outcomes)

		var succeeded *outcome
		conflicts := 0
		for candidate := range outcomes {
			if candidate.err == nil {
				value := candidate
				succeeded = &value
				continue
			}
			if errors.Is(candidate.err, application.ErrConflict) {
				conflicts++
				continue
			}
			t.Fatalf("concurrent Lead decision error = %v", candidate.err)
		}
		if succeeded == nil || conflicts != 1 {
			t.Fatalf(
				"concurrent Lead decisions = success %v conflicts %d",
				succeeded,
				conflicts,
			)
		}
		replayed, err := service.DecidePotentialFinding(
			context.Background(),
			lead,
			succeeded.command,
		)
		if err != nil || replayed != succeeded.result {
			t.Fatalf("winning Lead decision replay = %+v, err = %v", replayed, err)
		}
		var decisions, audits, outbox, links int
		if err := pool.QueryRow(context.Background(), `
			SELECT
				(SELECT count(*) FROM review_decisions
				 WHERE entity_type = 'potential_finding'
				   AND entity_id = 'potential-cabin-001'),
				(SELECT count(*) FROM audit_events
				 WHERE action = 'potential_finding.decided'
				   AND entity_id = 'potential-cabin-001'),
				(SELECT count(*) FROM outbox_messages
				 WHERE topic = 'potential_finding.decided'
				   AND aggregate_id = 'potential-cabin-001'),
				(SELECT count(*) FROM command_transaction_links
				 WHERE operation_id = $1)
		`, succeeded.command.OperationID).Scan(
			&decisions,
			&audits,
			&outbox,
			&links,
		); err != nil {
			t.Fatalf("read concurrent Lead transaction links: %v", err)
		}
		if decisions != 1 || audits != 1 || outbox != 1 || links != 1 {
			t.Fatalf(
				"concurrent Lead effects = decisions %d audits %d outbox %d links %d",
				decisions,
				audits,
				outbox,
				links,
			)
		}
	})
}

func seedReviewableEvidenceVersions(
	t *testing.T,
	pool *database.Pool,
	findingID string,
	versionCount int,
) {
	t.Helper()
	seedFinding(t, pool, findingID, "OPS-2026-090", "airline-xyz")
	if _, err := pool.Exec(
		context.Background(),
		"UPDATE findings SET status = 'PENDING_CAA_REVIEW' WHERE id = $1",
		findingID,
	); err != nil {
		t.Fatalf("seed Evidence review Finding state: %v", err)
	}
	for version := 1; version <= versionCount; version++ {
		versionID := fmt.Sprintf("evidence-version-exact-%d", version)
		if _, err := pool.Exec(context.Background(), `
			INSERT INTO evidence_versions (
				id, evidence_id, finding_id, organization_id, version, filename,
				media_type, sha256, size_bytes, status, submitted_by_subject_id,
				submitted_at, revision
			) VALUES (
				$1, 'evidence-exact', $2, 'airline-xyz', $3, $4,
				'application/pdf', $5, 1024, 'CLEAN', 'auditee-xyz', $6, 1
			)
		`,
			versionID,
			findingID,
			version,
			fmt.Sprintf("evidence-%d.pdf", version),
			fmt.Sprintf("sha256:evidence-%d", version),
			canonicalNow,
		); err != nil {
			t.Fatalf("seed Evidence version %d: %v", version, err)
		}
	}
}

func assertNoFindingReviewSideEffects(
	t *testing.T,
	pool *database.Pool,
	findingID string,
	expectedStatus string,
) {
	t.Helper()
	var status string
	if err := pool.QueryRow(
		context.Background(),
		"SELECT status FROM findings WHERE id = $1",
		findingID,
	).Scan(&status); err != nil {
		t.Fatalf("read Finding status: %v", err)
	}
	if status != expectedStatus {
		t.Fatalf("Finding status = %q, want %q", status, expectedStatus)
	}
	var decisions int
	if err := pool.QueryRow(
		context.Background(),
		"SELECT count(*) FROM review_decisions WHERE entity_type = 'evidence_version'",
	).Scan(&decisions); err != nil {
		t.Fatalf("count Evidence review decisions: %v", err)
	}
	if decisions != 0 {
		t.Fatalf("Evidence review decisions = %d, want 0", decisions)
	}
}

func seedCanonicalTestProfileAttachmentPackage(t *testing.T, pool *database.Pool) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO canonical_question_catalogs (
			id, catalog_version, usage_class, profile_name, profile_version,
			status, source_package_version, source_package_json_sha256,
			source_package_zip_sha256, root_digest, question_count, form_count,
			created_by_subject_id
		) VALUES (
			'CAT-TASK6-ATTACHMENT', 'task6-attachment@1.0.0',
			'PREPROD_EXERCISE', 'aga-preprod', '1.0.0', 'SEALED', '1.0.0',
			'sha256:task6-attachment-json', 'sha256:task6-attachment-zip',
			'sha256:task6-attachment-root', 1, 1, 'USR-MANAGER-NORA'
		);

		INSERT INTO canonical_question_catalog_forms (
			catalog_id, form_code, form_digest, archive_digest,
			question_count, source_gap_state
		) VALUES (
			'CAT-TASK6-ATTACHMENT', 'CABIN', 'sha256:task6-form',
			'sha256:task6-archive', 1, 'SOURCE_MAPPING_REQUIRED'
		);

		INSERT INTO canonical_question_version_provenance (
			question_version_id, usage_class, catalog_id
		) VALUES (
			'QV-CAB-EMEQ-PBE-001-V1', 'PREPROD_EXERCISE',
			'CAT-TASK6-ATTACHMENT'
		);

		INSERT INTO canonical_question_catalog_memberships (
			catalog_id, question_version_id, usage_class, form_code,
			proposal_id, ordinal, question_digest, source_locator,
			source_gap_state, proposed_domain, proposed_topic, proposed_risk_band
		) VALUES (
			'CAT-TASK6-ATTACHMENT', 'QV-CAB-EMEQ-PBE-001-V1',
			'PREPROD_EXERCISE', 'CABIN', 'PBE-001', 1,
			'sha256:task6-pbe-question', 'fixture://task6/pbe',
			'SOURCE_MAPPING_REQUIRED', 'Cabin Safety', 'PBE', 'MEDIUM'
		);

		INSERT INTO canonical_question_catalog_applicabilities (
			catalog_id, question_version_id, provider_scope_id,
			regulated_target_id, status, reason, actor_subject_id
		) VALUES (
			'CAT-TASK6-ATTACHMENT', 'QV-CAB-EMEQ-PBE-001-V1',
			'SCOPE-OPS-AOC-SOURCE-BOUND', 'TARGET-OPS-AOC-SOURCE-BOUND', 'ELIGIBLE',
			'Exact task-owned attachment fixture eligibility.', 'USR-MANAGER-NORA'
		);

		INSERT INTO canonical_audit_scope_drafts (
			id, planning_intake_draft_id, organization_id, provider_scope_id,
			regulated_target_id, audit_type, catalog_id, usage_class, revision,
			status, selected_question_count, selection_digest, requested_budget,
			notice_policy, created_by_subject_id
		) VALUES (
			'SCOPE-DRAFT-TASK6-ATTACHMENT', 'PLAN-DRAFT-2026-001',
			'ORG-FLY-NAMIBIA', 'SCOPE-OPS-AOC-SOURCE-BOUND',
			'TARGET-OPS-AOC-SOURCE-BOUND', 'CABIN',
			'CAT-TASK6-ATTACHMENT', 'PREPROD_EXERCISE', 1, 'RELEASED', 1,
			'8bf3518c051416c444a9b441fe44a67f9e17fd1c54723a2ef5cf91e1a67833e0',
			0, 'ADVANCE', 'USR-MANAGER-NORA'
		);

		INSERT INTO canonical_audit_scope_draft_questions (
			scope_draft_id, revision, catalog_id, question_version_id,
			position, selection_digest
		) VALUES (
			'SCOPE-DRAFT-TASK6-ATTACHMENT', 1, 'CAT-TASK6-ATTACHMENT',
			'QV-CAB-EMEQ-PBE-001-V1', 0,
			'8bf3518c051416c444a9b441fe44a67f9e17fd1c54723a2ef5cf91e1a67833e0'
		);

		INSERT INTO canonical_audit_scope_snapshots (
			id, scope_draft_id, revision, stage, catalog_id, usage_class,
			selection_digest, selected_question_count, snapshot,
			planning_snapshot_digest, created_by_subject_id
		) VALUES (
			'SCOPE-SNAPSHOT-TASK6-ATTACHMENT', 'SCOPE-DRAFT-TASK6-ATTACHMENT',
			1, 'RELEASED', 'CAT-TASK6-ATTACHMENT', 'PREPROD_EXERCISE',
			'8bf3518c051416c444a9b441fe44a67f9e17fd1c54723a2ef5cf91e1a67833e0',
			1, '{"catalogVersion":"task6-attachment@1.0.0","selectedQuestionVersionIds":["QV-CAB-EMEQ-PBE-001-V1"]}',
			'sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa',
			'USR-MANAGER-NORA'
		);

		INSERT INTO canonical_audit_scope_snapshot_questions (
			snapshot_id, catalog_id, question_version_id, position
		) VALUES (
			'SCOPE-SNAPSHOT-TASK6-ATTACHMENT', 'CAT-TASK6-ATTACHMENT',
			'QV-CAB-EMEQ-PBE-001-V1', 0
		);

		INSERT INTO inspection_packages (
			id, inspection_id, canonical_scope_snapshot_id, package_version,
			snapshot, expires_at, package_digest
		) VALUES (
			'PKG-TASK6-ATTACHMENT', 'AUD-2026-001',
			'SCOPE-SNAPSHOT-TASK6-ATTACHMENT', 2,
			'{"questionIds":["CAB-EMEQ-PBE-001"],"questionVersionIds":["QV-CAB-EMEQ-PBE-001-V1"]}',
			'2026-08-01T00:00:00Z', 'sha256:task6-attachment-package'
		)
	`); err != nil {
		t.Fatalf("seed canonical test-profile attachment package: %v", err)
	}
}
