//go:build canonicaltest

package integration_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/application"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/checklistgovernance"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/httpapi"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/identity"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/platform/database"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/regulatory"
	"github.com/MarlonJD/aviaSurveil360/apps/api/migrations"
	"github.com/jackc/pgx/v5"
)

// Break caught: an approval response or identical replay loaded after releasing
// the root lock can report a later owner's state instead of its own commit.
func TestTask6FixRound3DifferentOwnerResultsAreCommittedInLockExactly(t *testing.T) {
	ctx := context.Background()
	service, foiManager, submitted := task6SubmittedCandidate(t, "task6_fix3_in_lock_result")
	if _, err := service.Pool.Exec(ctx, `
		INSERT INTO candidate_required_owner_assignments
			(id,candidate_draft_version_id,candidate_revision,candidate_content_digest,
			 department_id,organizational_unit_id,approval_required)
		VALUES ('OWNER-TASK6-FIX3-AIR',$1,$2,$3,
			'AIRWORTHINESS_INSPECTORATE','AIRWORTHINESS_INSPECTORATE',true)`,
		submitted.CandidateID, submitted.Revision, submitted.ContentDigest,
	); err != nil {
		t.Fatalf("add AIR required owner: %v", err)
	}
	airManager := identity.Principal{
		SubjectID: "USR-TASK6-AIR-MANAGER",
		Roles:     []identity.Role{identity.RoleDepartmentManager},
	}
	foiCommand := checklistgovernance.ReviewCommand{
		OperationID: "TASK6-FIX3-FOI-APPROVE", IdempotencyKey: "TASK6-FIX3-FOI-APPROVE",
		CandidateID: submitted.CandidateID, ExpectedRevision: submitted.Revision,
		ExpectedContentDigest: submitted.ContentDigest,
		Reason:                "FOI exact owner approval commits the partial result.",
	}
	airCommand := checklistgovernance.ReviewCommand{
		OperationID: "TASK6-FIX3-AIR-APPROVE", IdempotencyKey: "TASK6-FIX3-AIR-APPROVE",
		CandidateID: submitted.CandidateID, ExpectedRevision: submitted.Revision,
		ExpectedContentDigest: submitted.ContentDigest,
		Reason:                "AIR exact owner approval commits the final result.",
	}

	blocker, err := service.Pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer blocker.Rollback(ctx)
	if _, err := blocker.Exec(ctx,
		`SELECT pg_advisory_xact_lock(hashtextextended($1,0))`,
		submitted.CandidateRootID,
	); err != nil {
		t.Fatalf("hold root lock: %v", err)
	}
	type result struct {
		candidate regulatory.CandidateView
		err       error
	}
	foiResult := make(chan result, 1)
	go func() {
		candidate, err := service.Approve(ctx, foiManager, foiCommand)
		foiResult <- result{candidate: candidate, err: err}
	}()
	waitForTask6AdvisoryWaiters(t, service, 1)
	airResult := make(chan result, 1)
	go func() {
		candidate, err := service.Approve(ctx, airManager, airCommand)
		airResult <- result{candidate: candidate, err: err}
	}()
	waitForTask6AdvisoryWaiters(t, service, 2)
	if err := blocker.Commit(ctx); err != nil {
		t.Fatalf("release root lock: %v", err)
	}

	first, second := <-foiResult, <-airResult
	if first.err != nil || first.candidate.Status != "DEPARTMENT_REVIEW" {
		t.Fatalf("partial committed result=%+v err=%v", first.candidate, first.err)
	}
	if second.err != nil || second.candidate.Status != "TECHNICALLY_APPROVED" {
		t.Fatalf("final committed result=%+v err=%v", second.candidate, second.err)
	}
	replayed, err := service.Approve(ctx, foiManager, foiCommand)
	if err != nil || replayed.Status != "DEPARTMENT_REVIEW" {
		t.Fatalf("partial identical replay=%+v err=%v", replayed, err)
	}
	for label, candidate := range map[string]regulatory.CandidateView{
		"partial": first.candidate, "final": second.candidate, "replay": replayed,
	} {
		if candidate.CandidateID != submitted.CandidateID ||
			candidate.CandidateRootID != submitted.CandidateRootID ||
			candidate.Revision != submitted.Revision ||
			candidate.ContentDigest != submitted.ContentDigest {
			t.Fatalf("%s result lost exact candidate identity: %+v", label, candidate)
		}
	}

	actualInventory, err := task6FixRound4LoadApprovalInventory(
		ctx, service.Pool, submitted.CandidateID,
	)
	if err != nil {
		t.Fatalf("read independent approval inventories: %v", err)
	}
	expectedInventory := task6FixRound4ExpectedApprovalInventory(
		t, "task6_fix3_in_lock_result", submitted, foiCommand, airCommand,
	)
	task6FixRound4AssertApprovalInventory(t, actualInventory, expectedInventory)

	var currentLeaf, status string
	var publicationCount int
	if err := service.Pool.QueryRow(ctx, `
		SELECT candidate.id,candidate.status,
		       (SELECT COUNT(*) FROM checklist_publication_decisions publication
		         WHERE publication.candidate_draft_version_id=candidate.id)
		FROM template_draft_versions candidate
		WHERE candidate.candidate_root_id=$1
		  AND NOT EXISTS (
			SELECT 1 FROM template_draft_versions successor
			WHERE successor.supersedes_candidate_id=candidate.id
		  )`,
		submitted.CandidateRootID,
	).Scan(&currentLeaf, &status, &publicationCount); err != nil {
		t.Fatal(err)
	}
	if currentLeaf != submitted.CandidateID || status != "TECHNICALLY_APPROVED" ||
		publicationCount != 0 {
		t.Fatalf("leaf=%s status=%s publications=%d",
			currentLeaf, status, publicationCount)
	}
}

type task6FixRound4ApprovalInventory struct {
	Decisions []string `json:"decisions"`
	Audits    []string `json:"audits"`
	Owners    []string `json:"owners"`
	Commands  []string `json:"commands"`
}

type task6FixRound4Queryer interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
}

func task6FixRound4LoadApprovalInventory(
	ctx context.Context,
	queryer task6FixRound4Queryer,
	candidateID string,
) (task6FixRound4ApprovalInventory, error) {
	inventory := task6FixRound4ApprovalInventory{
		Decisions: []string{}, Audits: []string{},
		Owners: []string{}, Commands: []string{},
	}
	queries := []struct {
		target    *[]string
		statement string
	}{
		{&inventory.Decisions, `
			SELECT concat_ws(chr(31),
				decision.id,decision.decision,decision.candidate_root_id,
				decision.candidate_draft_version_id,decision.candidate_revision::text,
				decision.candidate_content_digest,decision.actor_subject_id,
				decision.actor_department_membership_id,membership.root_id,
				(NOT EXISTS (
					SELECT 1 FROM caa_department_memberships successor
					WHERE successor.supersedes_id=membership.id
				))::text,membership.status,membership.effective_from::text,
				COALESCE(membership.effective_to::text,''),
				decision.actor_department_id,decision.actor_organizational_unit_id,
				department.status,unit.status,decision.reason,
				to_char(decision.decided_at AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS"Z"'),
				decision.operation_id,decision.idempotency_key,
				decision.semantic_payload_digest,'AE-' || decision.operation_id)
			FROM department_review_decisions decision
			JOIN caa_department_memberships membership
			  ON membership.id=decision.actor_department_membership_id
			JOIN caa_departments department ON department.id=decision.actor_department_id
			JOIN caa_organizational_units unit
			  ON unit.id=decision.actor_organizational_unit_id
			 AND unit.department_id=decision.actor_department_id
			WHERE decision.candidate_draft_version_id=$1
			ORDER BY decision.operation_id`},
		{&inventory.Audits, `
			SELECT concat_ws(chr(31),
				audit.event_id,candidate.candidate_root_id,audit.entity_id,
				audit.entity_version::text,
				to_char(audit.occurred_at AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS"Z"'),
				COALESCE(audit.actor_subject_id,''),COALESCE(audit.actor_role,''),
				COALESCE(audit.organization_id,''),audit.action,audit.entity_type,
				COALESCE(audit.before_status,''),COALESCE(audit.after_status,''),
				COALESCE(audit.reason,''),COALESCE(audit.operation_id,''),
				COALESCE(audit.correlation_id,''),COALESCE(audit.request_id,''),
				audit.details::text)
			FROM audit_events audit
			JOIN template_draft_versions candidate ON candidate.id=audit.entity_id
			WHERE audit.entity_type='GOVERNED_CANDIDATE' AND audit.entity_id=$1
			ORDER BY audit.event_id`},
		{&inventory.Owners, `
			SELECT concat_ws(chr(31),
				owner.id,candidate.candidate_root_id,
				owner.candidate_draft_version_id,owner.candidate_revision::text,
				owner.candidate_content_digest,owner.department_id,
				owner.organizational_unit_id,owner.approval_required::text)
			FROM candidate_required_owner_assignments owner
			JOIN template_draft_versions candidate
			  ON candidate.id=owner.candidate_draft_version_id
			WHERE owner.candidate_draft_version_id=$1
			ORDER BY owner.id`},
		{&inventory.Commands, `
			SELECT concat_ws(chr(31),
				command.id,command.command_kind,command.operation_id,
				command.idempotency_key,command.semantic_payload_digest,
				command.generation_run_id,candidate.candidate_root_id,
				command.candidate_draft_version_id,command.candidate_revision::text,
				command.candidate_content_digest,command.actor_subject_id,
				command.reason,command.audit_event_id,
				to_char(command.created_at AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS"Z"'))
			FROM governed_candidate_commands command
			JOIN template_draft_versions candidate
			  ON candidate.id=command.candidate_draft_version_id
			WHERE command.candidate_draft_version_id=$1
			ORDER BY command.operation_id`},
	}
	for _, item := range queries {
		rows, err := queryer.Query(ctx, item.statement, candidateID)
		if err != nil {
			return inventory, err
		}
		for rows.Next() {
			var row string
			if err := rows.Scan(&row); err != nil {
				rows.Close()
				return inventory, err
			}
			*item.target = append(*item.target, row)
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return inventory, err
		}
		rows.Close()
	}
	return inventory, nil
}

func task6FixRound4InventoryRow(values ...any) string {
	parts := make([]string, len(values))
	for index, value := range values {
		parts[index] = fmt.Sprint(value)
	}
	return strings.Join(parts, string(rune(31)))
}

func task6FixRound4ExpectedApprovalInventory(
	t *testing.T,
	label string,
	candidate regulatory.CandidateView,
	foiCommand checklistgovernance.ReviewCommand,
	airCommand checklistgovernance.ReviewCommand,
) task6FixRound4ApprovalInventory {
	t.Helper()
	semantic := func(command checklistgovernance.ReviewCommand) string {
		value, err := regulatory.CanonicalSHA256(map[string]any{
			"command": "TECHNICALLY_APPROVED", "operationId": command.OperationID,
			"candidateId":           command.CandidateID,
			"expectedRevision":      command.ExpectedRevision,
			"expectedContentDigest": command.ExpectedContentDigest,
			"reason":                command.Reason,
		})
		if err != nil {
			t.Fatal(err)
		}
		return value
	}
	importOperation := "TASK6-IMPORT-" + label
	submitOperation := "TASK6-SUBMIT-" + label
	importSemantic, err := regulatory.CanonicalSHA256(map[string]any{
		"operationId":     importOperation,
		"candidateBundle": regulatory.SyntheticCandidateBundle(),
	})
	if err != nil {
		t.Fatal(err)
	}
	submitReason := "Submit exact synthetic candidate for Task 6 review."
	submitSemantic, err := regulatory.CanonicalSHA256(map[string]any{
		"operationId": submitOperation, "idempotencyKey": submitOperation,
		"candidateId":           candidate.CandidateID,
		"expectedContentDigest": candidate.ContentDigest,
		"reason":                submitReason, "expectedRevision": candidate.Revision,
	})
	if err != nil {
		t.Fatal(err)
	}
	at := "2026-07-29T16:00:00Z"
	audit := func(
		operation, actor, action, before, after, reason string,
	) string {
		return task6FixRound4InventoryRow(
			"AE-"+operation, candidate.CandidateRootID, candidate.CandidateID,
			candidate.Revision, at, actor,
			map[bool]string{true: "manager", false: "admin"}[strings.Contains(action, "APPROVAL")],
			"", action, "GOVERNED_CANDIDATE", before, after, reason,
			operation, operation, operation, "{}",
		)
	}
	return task6FixRound4ApprovalInventory{
		Decisions: []string{
			task6FixRound4InventoryRow(
				"DRD-"+airCommand.OperationID, "TECHNICALLY_APPROVED",
				candidate.CandidateRootID, candidate.CandidateID, candidate.Revision,
				candidate.ContentDigest, "USR-TASK6-AIR-MANAGER", "MEM-TASK6-AIR",
				"MEM-TASK6-AIR", true, "ACTIVE", "2025-01-01", "",
				"AIRWORTHINESS_INSPECTORATE", "AIRWORTHINESS_INSPECTORATE",
				"ACTIVE", "ACTIVE", airCommand.Reason, at, airCommand.OperationID,
				airCommand.IdempotencyKey, semantic(airCommand),
				"AE-"+airCommand.OperationID,
			),
			task6FixRound4InventoryRow(
				"DRD-"+foiCommand.OperationID, "TECHNICALLY_APPROVED",
				candidate.CandidateRootID, candidate.CandidateID, candidate.Revision,
				candidate.ContentDigest, "USR-TASK6-FOI-MANAGER", "MEM-TASK6-FOI",
				"MEM-TASK6-FOI", true, "ACTIVE", "2025-01-01", "",
				"FLIGHT_OPERATIONS_INSPECTORATE", "FLIGHT_OPERATIONS_INSPECTORATE",
				"ACTIVE", "ACTIVE", foiCommand.Reason, at, foiCommand.OperationID,
				foiCommand.IdempotencyKey, semantic(foiCommand),
				"AE-"+foiCommand.OperationID,
			),
		},
		Audits: []string{
			audit(airCommand.OperationID, "USR-TASK6-AIR-MANAGER",
				"TECHNICAL_APPROVAL_RECORDED", "DEPARTMENT_REVIEW",
				"TECHNICALLY_APPROVED", airCommand.Reason),
			audit(foiCommand.OperationID, "USR-TASK6-FOI-MANAGER",
				"TECHNICAL_APPROVAL_RECORDED", "DEPARTMENT_REVIEW",
				"DEPARTMENT_REVIEW", foiCommand.Reason),
			audit(importOperation, "USR-TASK6-ADMIN", "IMPORTED_GENERATION_RUN",
				"", "GENERATED_DRAFT", "Imported deterministic governed generation run."),
			audit(submitOperation, "USR-TASK6-ADMIN", "DEPARTMENT_REVIEW_SUBMITTED",
				"GENERATED_DRAFT", "DEPARTMENT_REVIEW", submitReason),
		},
		Owners: []string{
			task6FixRound4InventoryRow(
				"OWNER-"+candidate.CandidateID, candidate.CandidateRootID,
				candidate.CandidateID, candidate.Revision, candidate.ContentDigest,
				"FLIGHT_OPERATIONS_INSPECTORATE", "FLIGHT_OPERATIONS_INSPECTORATE", true,
			),
			task6FixRound4InventoryRow(
				"OWNER-TASK6-FIX3-AIR", candidate.CandidateRootID,
				candidate.CandidateID, candidate.Revision, candidate.ContentDigest,
				"AIRWORTHINESS_INSPECTORATE", "AIRWORTHINESS_INSPECTORATE", true,
			),
		},
		Commands: []string{
			task6FixRound4InventoryRow(
				"CMD-"+importOperation, "IMPORTED_GENERATION_RUN",
				importOperation, importOperation, importSemantic,
				candidate.GenerationRunID, candidate.CandidateRootID,
				candidate.CandidateID, candidate.Revision, candidate.ContentDigest,
				"USR-TASK6-ADMIN", "Imported deterministic governed generation run.",
				"AE-"+importOperation, at,
			),
			task6FixRound4InventoryRow(
				"CMD-"+submitOperation, "DEPARTMENT_REVIEW_SUBMITTED",
				submitOperation, submitOperation, submitSemantic,
				candidate.GenerationRunID, candidate.CandidateRootID,
				candidate.CandidateID, candidate.Revision, candidate.ContentDigest,
				"USR-TASK6-ADMIN", submitReason, "AE-"+submitOperation, at,
			),
		},
	}
}

func task6FixRound4AssertApprovalInventory(
	t *testing.T,
	actual, expected task6FixRound4ApprovalInventory,
) {
	t.Helper()
	actualBytes, actualErr := json.Marshal(actual)
	expectedBytes, expectedErr := json.Marshal(expected)
	if actualErr != nil || expectedErr != nil || !bytes.Equal(actualBytes, expectedBytes) {
		t.Fatalf("exact approval inventory mismatch\nactual=%s\nexpected=%s\nerrors=%v/%v",
			actualBytes, expectedBytes, actualErr, expectedErr)
	}
}

// Break caught: an Audit row for the exact candidate that is not linked to a
// review decision must still make the concurrent-approval effect inventory
// fail. An inner join between decisions and Audits cannot establish that.
func TestTask6FixRound4ConcurrentApprovalInventoryRejectsUnlinkedAudit(t *testing.T) {
	ctx := context.Background()
	service, manager, submitted := task6SubmittedCandidate(
		t, "task6_fix4_unlinked_audit_inventory",
	)
	if _, err := service.Approve(ctx, manager, checklistgovernance.ReviewCommand{
		OperationID:           "TASK6-FIX4-INVENTORY-APPROVE",
		IdempotencyKey:        "TASK6-FIX4-INVENTORY-APPROVE",
		CandidateID:           submitted.CandidateID,
		ExpectedRevision:      submitted.Revision,
		ExpectedContentDigest: submitted.ContentDigest,
		Reason:                "Create the exact reviewed approval effect.",
	}); err != nil {
		t.Fatalf("approve inventory candidate: %v", err)
	}
	expected, err := task6FixRound4LoadApprovalInventory(
		ctx, service.Pool, submitted.CandidateID,
	)
	if err != nil {
		t.Fatal(err)
	}
	tx, err := service.Pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `
		INSERT INTO audit_events
			(event_id,occurred_at,actor_subject_id,actor_role,action,entity_type,
			 entity_id,entity_version,before_status,after_status,reason,operation_id,
			 correlation_id,request_id,details)
		VALUES ('AE-TASK6-FIX4-UNLINKED','2026-07-29T16:00:00Z',
			'USR-TASK6-FOI-MANAGER','manager','TECHNICAL_APPROVAL_RECORDED',
			'GOVERNED_CANDIDATE',$1,$2,'DEPARTMENT_REVIEW','TECHNICALLY_APPROVED',
			'Unlinked extra Audit must be detected.','TASK6-FIX4-UNLINKED',
			'TASK6-FIX4-UNLINKED','TASK6-FIX4-UNLINKED','{}')`,
		submitted.CandidateID, submitted.Revision,
	); err != nil {
		t.Fatalf("insert unlinked Audit: %v", err)
	}
	actual, err := task6FixRound4LoadApprovalInventory(
		ctx, tx, submitted.CandidateID,
	)
	if err != nil {
		t.Fatal(err)
	}
	actualBytes, _ := json.Marshal(actual)
	expectedBytes, _ := json.Marshal(expected)
	if bytes.Equal(actualBytes, expectedBytes) {
		t.Fatal("unlinked extra Audit escaped the independent exact approval inventory")
	}
	if len(actual.Decisions) != len(expected.Decisions) ||
		len(actual.Owners) != len(expected.Owners) ||
		len(actual.Commands) != len(expected.Commands) ||
		len(actual.Audits) != len(expected.Audits)+1 {
		t.Fatalf(
			"unlinked Audit changed wrong inventories decisions=%d/%d Audits=%d/%d owners=%d/%d commands=%d/%d",
			len(actual.Decisions), len(expected.Decisions),
			len(actual.Audits), len(expected.Audits),
			len(actual.Owners), len(expected.Owners),
			len(actual.Commands), len(expected.Commands),
		)
	}
}

type task6FixRound4ParityOperation struct {
	OperationID    string `json:"operationId"`
	IdempotencyKey string `json:"idempotencyKey"`
	Reason         string `json:"reason"`
}

type task6FixRound4ParityActor struct {
	SubjectID            string `json:"subjectId"`
	MembershipID         string `json:"membershipId"`
	DepartmentID         string `json:"departmentId"`
	OrganizationalUnitID string `json:"organizationalUnitId"`
}

type task6FixRound4ArtifactParityContract struct {
	ContractID string `json:"contractId"`
	Clock      string `json:"clock"`
	Candidate  struct {
		CandidateID             string `json:"candidateId"`
		CandidateRootID         string `json:"candidateRootId"`
		Revision                int64  `json:"revision"`
		ContentDigest           string `json:"contentDigest"`
		AbsentTemplateVersionID string `json:"absentTemplateVersionId"`
		TamperedDigest          string `json:"tamperedDigest"`
	} `json:"candidate"`
	Actors struct {
		FOI task6FixRound4ParityActor `json:"foi"`
		AIR task6FixRound4ParityActor `json:"air"`
	} `json:"actors"`
	RequiredOwners []regulatory.RequiredOwner `json:"requiredOwners"`
	Operations     struct {
		Import                  task6FixRound4ParityOperation `json:"import"`
		Submit                  task6FixRound4ParityOperation `json:"submit"`
		FOIApproval             task6FixRound4ParityOperation `json:"foiApproval"`
		AIRApproval             task6FixRound4ParityOperation `json:"airApproval"`
		Publication             task6FixRound4ParityOperation `json:"publication"`
		DigestTamper            task6FixRound4ParityOperation `json:"digestTamper"`
		ConflictingReplayReason string                        `json:"conflictingReplayReason"`
	} `json:"operations"`
	Expected struct {
		PartialStatus       string `json:"partialStatus"`
		FinalStatus         string `json:"finalStatus"`
		PublishedStatus     string `json:"publishedStatus"`
		ArtifactCheckpoints struct {
			ApprovalOnly  task6FixRound5ArtifactCheckpoint `json:"approvalOnly"`
			JointComplete task6FixRound5ArtifactCheckpoint `json:"jointComplete"`
			Published     task6FixRound5ArtifactCheckpoint `json:"published"`
		} `json:"artifactCheckpoints"`
		ArtifactRows task6FixRound5ArtifactRows `json:"artifactRows"`
	} `json:"expected"`
	PublishedArtifact struct {
		Mappings  []regulatory.ComplianceMapping `json:"mappings"`
		Questions []regulatory.ChecklistQuestion `json:"questions"`
	} `json:"publishedArtifact"`
}

type task6FixRound4DecisionArtifact struct {
	DecisionID                  string `json:"decisionId"`
	Decision                    string `json:"decision"`
	CandidateRootID             string `json:"candidateRootId"`
	CandidateID                 string `json:"candidateId"`
	CandidateRevision           int64  `json:"candidateRevision"`
	CandidateContentDigest      string `json:"candidateContentDigest"`
	ActorSubjectID              string `json:"actorSubjectId"`
	ActorDepartmentMembershipID string `json:"actorDepartmentMembershipId"`
	ActorDepartmentID           string `json:"actorDepartmentId"`
	ActorOrganizationalUnitID   string `json:"actorOrganizationalUnitId"`
	Reason                      string `json:"reason"`
	DecidedAt                   string `json:"decidedAt"`
	OperationID                 string `json:"operationId"`
	IdempotencyKey              string `json:"idempotencyKey"`
	SemanticPayloadDigest       string `json:"semanticPayloadDigest"`
	AuditEventID                string `json:"auditEventId"`
}

type task6FixRound4PublicationArtifact struct {
	TemplateVersionID         string `json:"templateVersionId"`
	PublicationDecisionID     string `json:"publicationDecisionId"`
	CandidateRootID           string `json:"candidateRootId"`
	CandidateID               string `json:"candidateId"`
	CandidateRevision         int64  `json:"candidateRevision"`
	CandidateContentDigest    string `json:"candidateContentDigest"`
	ActorSubjectID            string `json:"actorSubjectId"`
	ActorMembershipID         string `json:"actorDepartmentMembershipId"`
	ActorDepartmentID         string `json:"actorDepartmentId"`
	ActorOrganizationalUnitID string `json:"actorOrganizationalUnitId"`
	Reason                    string `json:"reason"`
	DecidedAt                 string `json:"decidedAt"`
	PublishedAt               string `json:"publishedAt"`
	OperationID               string `json:"operationId"`
	IdempotencyKey            string `json:"idempotencyKey"`
	SemanticPayloadDigest     string `json:"semanticPayloadDigest"`
	AuditEventID              string `json:"auditEventId"`
}

type task6FixRound5ReviewDecisionArtifact struct {
	DecisionID                  string `json:"decisionId"`
	Decision                    string `json:"decision"`
	CandidateRootID             string `json:"candidateRootId"`
	CandidateID                 string `json:"candidateId"`
	CandidateRevision           int64  `json:"candidateRevision"`
	CandidateContentDigest      string `json:"candidateContentDigest"`
	ActorSubjectID              string `json:"actorSubjectId"`
	ActorDepartmentMembershipID string `json:"actorDepartmentMembershipId"`
	ActorMembershipIsCurrent    bool   `json:"actorMembershipIsCurrent"`
	ActorDepartmentID           string `json:"actorDepartmentId"`
	ActorOrganizationalUnitID   string `json:"actorOrganizationalUnitId"`
	Reason                      string `json:"reason"`
	DecidedAt                   string `json:"decidedAt"`
	OperationID                 string `json:"operationId"`
	IdempotencyKey              string `json:"idempotencyKey"`
	SemanticPayloadDigest       string `json:"semanticPayloadDigest"`
	AuditEventID                string `json:"auditEventId"`
}

type task6FixRound5AuditArtifact struct {
	EventID                     string `json:"eventId"`
	CandidateRootID             string `json:"candidateRootId"`
	CandidateID                 string `json:"candidateId"`
	CandidateRevision           int64  `json:"candidateRevision"`
	CandidateContentDigest      string `json:"candidateContentDigest"`
	ActorSubjectID              string `json:"actorSubjectId"`
	ActorRole                   string `json:"actorRole"`
	ActorDepartmentMembershipID string `json:"actorDepartmentMembershipId"`
	ActorMembershipIsCurrent    bool   `json:"actorMembershipIsCurrent"`
	ActorDepartmentID           string `json:"actorDepartmentId"`
	ActorOrganizationalUnitID   string `json:"actorOrganizationalUnitId"`
	Action                      string `json:"action"`
	EntityType                  string `json:"entityType"`
	EntityID                    string `json:"entityId"`
	BeforeStatus                string `json:"beforeStatus"`
	AfterStatus                 string `json:"afterStatus"`
	Reason                      string `json:"reason"`
	OccurredAt                  string `json:"occurredAt"`
	OperationID                 string `json:"operationId"`
	IdempotencyKey              string `json:"idempotencyKey"`
	SemanticPayloadDigest       string `json:"semanticPayloadDigest"`
	LinkedDecisionID            string `json:"linkedDecisionId"`
}

type task6FixRound5PublicationDecisionArtifact struct {
	PublicationDecisionID     string `json:"publicationDecisionId"`
	TemplateVersionID         string `json:"templateVersionId"`
	CandidateRootID           string `json:"candidateRootId"`
	CandidateID               string `json:"candidateId"`
	CandidateRevision         int64  `json:"candidateRevision"`
	CandidateContentDigest    string `json:"candidateContentDigest"`
	ActorSubjectID            string `json:"actorSubjectId"`
	ActorMembershipID         string `json:"actorDepartmentMembershipId"`
	ActorMembershipIsCurrent  bool   `json:"actorMembershipIsCurrent"`
	ActorDepartmentID         string `json:"actorDepartmentId"`
	ActorOrganizationalUnitID string `json:"actorOrganizationalUnitId"`
	Reason                    string `json:"reason"`
	DecidedAt                 string `json:"decidedAt"`
	CreatedAt                 string `json:"createdAt"`
	PublishedAt               string `json:"publishedAt"`
	OperationID               string `json:"operationId"`
	IdempotencyKey            string `json:"idempotencyKey"`
	SemanticPayloadDigest     string `json:"semanticPayloadDigest"`
	AuditEventID              string `json:"auditEventId"`
}

type task6FixRound5QuestionVersionPosition struct {
	QuestionVersionID string `json:"questionVersionId"`
	Position          int    `json:"position"`
}

type task6FixRound5ChecklistVersionArtifact struct {
	TemplateVersionID      string                                  `json:"templateVersionId"`
	TemplateID             string                                  `json:"templateId"`
	Version                int                                     `json:"version"`
	Title                  string                                  `json:"title"`
	PublishedAt            string                                  `json:"publishedAt"`
	CandidateRootID        string                                  `json:"candidateRootId"`
	CandidateID            string                                  `json:"candidateId"`
	CandidateRevision      int64                                   `json:"candidateRevision"`
	CandidateContentDigest string                                  `json:"candidateContentDigest"`
	PublicationDecisionID  string                                  `json:"publicationDecisionId"`
	AuditEventID           string                                  `json:"auditEventId"`
	QuestionVersionOrder   []task6FixRound5QuestionVersionPosition `json:"questionVersionOrder"`
	ImmutableSnapshot      any                                     `json:"immutableSnapshot"`
}

type task6FixRound5ArtifactRows struct {
	ReviewDecisions      []task6FixRound5ReviewDecisionArtifact      `json:"reviewDecisions"`
	AuditEvents          []task6FixRound5AuditArtifact               `json:"auditEvents"`
	PublicationDecisions []task6FixRound5PublicationDecisionArtifact `json:"publicationDecisions"`
	ChecklistVersions    []task6FixRound5ChecklistVersionArtifact    `json:"checklistVersions"`
}

type task6FixRound5ArtifactCheckpoint struct {
	ReviewDecisionIDs      []string `json:"reviewDecisionIds"`
	AuditEventIDs          []string `json:"auditEventIds"`
	PublicationDecisionIDs []string `json:"publicationDecisionIds"`
	ChecklistVersionIDs    []string `json:"checklistVersionIds"`
}

func task6FixRound4LoadArtifactParityContract(
	t *testing.T,
) task6FixRound4ArtifactParityContract {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(
		apiModuleRoot(t), "tests", "fixtures", "task6",
		"manager-artifact-parity-contract.json",
	))
	if err != nil {
		t.Fatalf("read shared Task 6 artifact parity contract: %v", err)
	}
	var contract task6FixRound4ArtifactParityContract
	if err := json.Unmarshal(raw, &contract); err != nil {
		t.Fatalf("decode shared Task 6 artifact parity contract: %v", err)
	}
	return contract
}

func task6FixRound5SelectArtifactRows[T any](
	t *testing.T,
	rows []T,
	ids []string,
	identity func(T) string,
) []T {
	t.Helper()
	byID := make(map[string]T, len(rows))
	for _, row := range rows {
		id := identity(row)
		if _, exists := byID[id]; exists {
			t.Fatalf("shared Task 6 artifact contract repeats row identity %q", id)
		}
		byID[id] = row
	}
	selected := make([]T, 0, len(ids))
	seen := make(map[string]bool, len(ids))
	for _, id := range ids {
		if seen[id] {
			t.Fatalf("shared Task 6 artifact checkpoint repeats row identity %q", id)
		}
		seen[id] = true
		row, exists := byID[id]
		if !exists {
			t.Fatalf("shared Task 6 artifact contract is missing exact row %q", id)
		}
		selected = append(selected, row)
	}
	return selected
}

func task6FixRound5ExpectedArtifactCheckpoint(
	t *testing.T,
	contract task6FixRound4ArtifactParityContract,
	checkpoint task6FixRound5ArtifactCheckpoint,
) task6FixRound5ArtifactRows {
	t.Helper()
	return task6FixRound5ArtifactRows{
		ReviewDecisions: task6FixRound5SelectArtifactRows(
			t, contract.Expected.ArtifactRows.ReviewDecisions,
			checkpoint.ReviewDecisionIDs,
			func(row task6FixRound5ReviewDecisionArtifact) string {
				return row.DecisionID
			},
		),
		AuditEvents: task6FixRound5SelectArtifactRows(
			t, contract.Expected.ArtifactRows.AuditEvents,
			checkpoint.AuditEventIDs,
			func(row task6FixRound5AuditArtifact) string {
				return row.EventID
			},
		),
		PublicationDecisions: task6FixRound5SelectArtifactRows(
			t, contract.Expected.ArtifactRows.PublicationDecisions,
			checkpoint.PublicationDecisionIDs,
			func(row task6FixRound5PublicationDecisionArtifact) string {
				return row.PublicationDecisionID
			},
		),
		ChecklistVersions: task6FixRound5SelectArtifactRows(
			t, contract.Expected.ArtifactRows.ChecklistVersions,
			checkpoint.ChecklistVersionIDs,
			func(row task6FixRound5ChecklistVersionArtifact) string {
				return row.TemplateVersionID
			},
		),
	}
}

func task6FixRound5AssertArtifactRows(
	t *testing.T,
	actual task6FixRound5ArtifactRows,
	expected task6FixRound5ArtifactRows,
) {
	t.Helper()
	actualBytes, actualErr := json.Marshal(actual)
	expectedBytes, expectedErr := json.Marshal(expected)
	if actualErr != nil || expectedErr != nil {
		t.Fatalf("encode complete Task 6 artifact rows: actual=%v expected=%v",
			actualErr, expectedErr)
	}
	if !bytes.Equal(actualBytes, expectedBytes) {
		t.Fatalf("complete Task 6 artifact rows mismatch\nactual=%s\nwant=%s",
			actualBytes, expectedBytes)
	}
}

func task6FixRound5LoadArtifactRows(
	ctx context.Context,
	pool *database.Pool,
	candidateID string,
	at time.Time,
) (task6FixRound5ArtifactRows, error) {
	output := task6FixRound5ArtifactRows{
		ReviewDecisions:      []task6FixRound5ReviewDecisionArtifact{},
		AuditEvents:          []task6FixRound5AuditArtifact{},
		PublicationDecisions: []task6FixRound5PublicationDecisionArtifact{},
		ChecklistVersions:    []task6FixRound5ChecklistVersionArtifact{},
	}
	decisions, err := pool.Query(ctx, `
		SELECT decision.id,decision.decision,decision.candidate_root_id,
		       decision.candidate_draft_version_id,decision.candidate_revision,
		       decision.candidate_content_digest,decision.actor_subject_id,
		       decision.actor_department_membership_id,
		       membership.id=(
		         SELECT current.id FROM caa_department_memberships current
		         WHERE current.root_id=membership.root_id
		           AND current.effective_from <= $2::date
		         ORDER BY current.effective_from DESC,current.id DESC LIMIT 1
		       )
		       AND membership.status='ACTIVE'
		       AND (membership.effective_to IS NULL OR membership.effective_to > $2::date)
		       AND COALESCE((
		         SELECT status FROM caa_department_status_facts fact
		         WHERE fact.department_id=decision.actor_department_id
		           AND fact.effective_from <= $2::date
		         ORDER BY fact.effective_from DESC,fact.id DESC LIMIT 1
		       ),'')='ACTIVE'
		       AND COALESCE((
		         SELECT status FROM caa_organizational_unit_status_facts fact
		         WHERE fact.organizational_unit_id=decision.actor_organizational_unit_id
		           AND fact.effective_from <= $2::date
		         ORDER BY fact.effective_from DESC,fact.id DESC LIMIT 1
		       ),'')='ACTIVE',
		       decision.actor_department_id,decision.actor_organizational_unit_id,
		       decision.reason,
		       to_char(decision.decided_at AT TIME ZONE 'UTC',
		               'YYYY-MM-DD"T"HH24:MI:SS"Z"'),
		       decision.operation_id,decision.idempotency_key,
		       decision.semantic_payload_digest,COALESCE(audit.event_id,'')
		FROM department_review_decisions decision
		JOIN caa_department_memberships membership
		  ON membership.id=decision.actor_department_membership_id
		LEFT JOIN audit_events audit
		  ON audit.event_id='AE-' || decision.operation_id
		WHERE decision.candidate_draft_version_id=$1
		ORDER BY decision.id`,
		candidateID, at,
	)
	if err != nil {
		return output, err
	}
	for decisions.Next() {
		var row task6FixRound5ReviewDecisionArtifact
		if err := decisions.Scan(
			&row.DecisionID, &row.Decision, &row.CandidateRootID,
			&row.CandidateID, &row.CandidateRevision,
			&row.CandidateContentDigest, &row.ActorSubjectID,
			&row.ActorDepartmentMembershipID, &row.ActorMembershipIsCurrent,
			&row.ActorDepartmentID, &row.ActorOrganizationalUnitID,
			&row.Reason, &row.DecidedAt, &row.OperationID,
			&row.IdempotencyKey, &row.SemanticPayloadDigest,
			&row.AuditEventID,
		); err != nil {
			decisions.Close()
			return output, err
		}
		output.ReviewDecisions = append(output.ReviewDecisions, row)
	}
	if err := decisions.Err(); err != nil {
		decisions.Close()
		return output, err
	}
	decisions.Close()

	audits, err := pool.Query(ctx, `
		SELECT audit.event_id,candidate.candidate_root_id,candidate.id,
		       COALESCE(audit.entity_version,candidate.revision),
		       candidate.candidate_content_digest,
		       COALESCE(audit.actor_subject_id,''),COALESCE(audit.actor_role,''),
		       COALESCE(review.actor_department_membership_id,
		                publication.actor_department_membership_id,''),
		       CASE WHEN membership.id IS NULL THEN false ELSE
		         membership.id=(
		           SELECT current.id FROM caa_department_memberships current
		           WHERE current.root_id=membership.root_id
		             AND current.effective_from <= $2::date
		           ORDER BY current.effective_from DESC,current.id DESC LIMIT 1
		         )
		         AND membership.status='ACTIVE'
		         AND (membership.effective_to IS NULL
		              OR membership.effective_to > $2::date)
		         AND COALESCE((
		           SELECT status FROM caa_department_status_facts fact
		           WHERE fact.department_id=COALESCE(
		             review.actor_department_id,publication.actor_department_id)
		             AND fact.effective_from <= $2::date
		           ORDER BY fact.effective_from DESC,fact.id DESC LIMIT 1
		         ),'')='ACTIVE'
		         AND COALESCE((
		           SELECT status FROM caa_organizational_unit_status_facts fact
		           WHERE fact.organizational_unit_id=COALESCE(
		             review.actor_organizational_unit_id,
		             publication.actor_organizational_unit_id)
		             AND fact.effective_from <= $2::date
		           ORDER BY fact.effective_from DESC,fact.id DESC LIMIT 1
		         ),'')='ACTIVE'
		       END,
		       COALESCE(review.actor_department_id,
		                publication.actor_department_id,''),
		       COALESCE(review.actor_organizational_unit_id,
		                publication.actor_organizational_unit_id,''),
		       audit.action,audit.entity_type,audit.entity_id,
		       COALESCE(audit.before_status,''),COALESCE(audit.after_status,''),
		       COALESCE(audit.reason,''),
		       to_char(audit.occurred_at AT TIME ZONE 'UTC',
		               'YYYY-MM-DD"T"HH24:MI:SS"Z"'),
		       COALESCE(audit.operation_id,''),
		       COALESCE(review.idempotency_key,publication.idempotency_key,''),
		       COALESCE(review.semantic_payload_digest,
		                publication.semantic_payload_digest,''),
		       COALESCE(review.id,publication.id,'')
		FROM audit_events audit
		JOIN template_draft_versions candidate ON candidate.id=$1
		LEFT JOIN department_review_decisions review
		  ON review.operation_id=audit.operation_id
		 AND review.candidate_draft_version_id=candidate.id
		LEFT JOIN checklist_publication_decisions publication
		  ON publication.operation_id=audit.operation_id
		 AND publication.candidate_draft_version_id=candidate.id
		LEFT JOIN caa_department_memberships membership
		  ON membership.id=COALESCE(
		    review.actor_department_membership_id,
		    publication.actor_department_membership_id)
		WHERE audit.actor_role='manager'
		  AND (
		    audit.entity_id=candidate.id
		    OR review.id IS NOT NULL
		    OR publication.id IS NOT NULL
		  )
		ORDER BY audit.event_id`,
		candidateID, at,
	)
	if err != nil {
		return output, err
	}
	for audits.Next() {
		var row task6FixRound5AuditArtifact
		if err := audits.Scan(
			&row.EventID, &row.CandidateRootID, &row.CandidateID,
			&row.CandidateRevision, &row.CandidateContentDigest,
			&row.ActorSubjectID, &row.ActorRole,
			&row.ActorDepartmentMembershipID,
			&row.ActorMembershipIsCurrent, &row.ActorDepartmentID,
			&row.ActorOrganizationalUnitID, &row.Action, &row.EntityType,
			&row.EntityID, &row.BeforeStatus, &row.AfterStatus,
			&row.Reason, &row.OccurredAt, &row.OperationID,
			&row.IdempotencyKey, &row.SemanticPayloadDigest,
			&row.LinkedDecisionID,
		); err != nil {
			audits.Close()
			return output, err
		}
		output.AuditEvents = append(output.AuditEvents, row)
	}
	if err := audits.Err(); err != nil {
		audits.Close()
		return output, err
	}
	audits.Close()

	publications, err := pool.Query(ctx, `
		SELECT publication.id,COALESCE(version.id,''),
		       publication.candidate_root_id,
		       publication.candidate_draft_version_id,
		       publication.candidate_revision,
		       publication.candidate_content_digest,
		       publication.actor_subject_id,
		       publication.actor_department_membership_id,
		       membership.id=(
		         SELECT current.id FROM caa_department_memberships current
		         WHERE current.root_id=membership.root_id
		           AND current.effective_from <= $2::date
		         ORDER BY current.effective_from DESC,current.id DESC LIMIT 1
		       )
		       AND membership.status='ACTIVE'
		       AND (membership.effective_to IS NULL OR membership.effective_to > $2::date)
		       AND COALESCE((
		         SELECT status FROM caa_department_status_facts fact
		         WHERE fact.department_id=publication.actor_department_id
		           AND fact.effective_from <= $2::date
		         ORDER BY fact.effective_from DESC,fact.id DESC LIMIT 1
		       ),'')='ACTIVE'
		       AND COALESCE((
		         SELECT status FROM caa_organizational_unit_status_facts fact
		         WHERE fact.organizational_unit_id=
		               publication.actor_organizational_unit_id
		           AND fact.effective_from <= $2::date
		         ORDER BY fact.effective_from DESC,fact.id DESC LIMIT 1
		       ),'')='ACTIVE',
		       publication.actor_department_id,
		       publication.actor_organizational_unit_id,publication.reason,
		       to_char(publication.decided_at AT TIME ZONE 'UTC',
		               'YYYY-MM-DD"T"HH24:MI:SS"Z"'),
		       to_char(publication.created_at AT TIME ZONE 'UTC',
		               'YYYY-MM-DD"T"HH24:MI:SS"Z"'),
		       COALESCE(to_char(version.published_at AT TIME ZONE 'UTC',
		                        'YYYY-MM-DD"T"HH24:MI:SS"Z"'),''),
		       publication.operation_id,publication.idempotency_key,
		       publication.semantic_payload_digest,COALESCE(audit.event_id,'')
		FROM checklist_publication_decisions publication
		JOIN caa_department_memberships membership
		  ON membership.id=publication.actor_department_membership_id
		LEFT JOIN checklist_template_versions version
		  ON version.publication_decision_id=publication.id
		LEFT JOIN audit_events audit
		  ON audit.event_id='AE-' || publication.operation_id
		WHERE publication.candidate_draft_version_id=$1
		ORDER BY publication.id`,
		candidateID, at,
	)
	if err != nil {
		return output, err
	}
	for publications.Next() {
		var row task6FixRound5PublicationDecisionArtifact
		if err := publications.Scan(
			&row.PublicationDecisionID, &row.TemplateVersionID,
			&row.CandidateRootID, &row.CandidateID, &row.CandidateRevision,
			&row.CandidateContentDigest, &row.ActorSubjectID,
			&row.ActorMembershipID, &row.ActorMembershipIsCurrent,
			&row.ActorDepartmentID, &row.ActorOrganizationalUnitID,
			&row.Reason, &row.DecidedAt, &row.CreatedAt, &row.PublishedAt,
			&row.OperationID, &row.IdempotencyKey,
			&row.SemanticPayloadDigest, &row.AuditEventID,
		); err != nil {
			publications.Close()
			return output, err
		}
		output.PublicationDecisions = append(output.PublicationDecisions, row)
	}
	if err := publications.Err(); err != nil {
		publications.Close()
		return output, err
	}
	publications.Close()

	versions, err := pool.Query(ctx, `
		SELECT version.id,version.template_id,version.version,version.title,
		       to_char(version.published_at AT TIME ZONE 'UTC',
		               'YYYY-MM-DD"T"HH24:MI:SS"Z"'),
		       candidate.candidate_root_id,version.candidate_draft_version_id,
		       version.candidate_revision,version.candidate_content_digest,
		       version.publication_decision_id,COALESCE(audit.event_id,''),
		       version.snapshot
		FROM checklist_template_versions version
		JOIN template_draft_versions candidate
		  ON candidate.id=version.candidate_draft_version_id
		LEFT JOIN checklist_publication_decisions publication
		  ON publication.id=version.publication_decision_id
		LEFT JOIN audit_events audit
		  ON audit.event_id='AE-' || publication.operation_id
		WHERE version.candidate_draft_version_id=$1
		ORDER BY version.id`,
		candidateID,
	)
	if err != nil {
		return output, err
	}
	for versions.Next() {
		var row task6FixRound5ChecklistVersionArtifact
		var snapshot []byte
		if err := versions.Scan(
			&row.TemplateVersionID, &row.TemplateID, &row.Version, &row.Title,
			&row.PublishedAt, &row.CandidateRootID, &row.CandidateID,
			&row.CandidateRevision, &row.CandidateContentDigest,
			&row.PublicationDecisionID, &row.AuditEventID, &snapshot,
		); err != nil {
			versions.Close()
			return output, err
		}
		if err := json.Unmarshal(snapshot, &row.ImmutableSnapshot); err != nil {
			versions.Close()
			return output, err
		}
		questionRows, err := pool.Query(ctx, `
			SELECT question_version_id,position
			FROM template_version_questions
			WHERE template_version_id=$1
			ORDER BY position,question_version_id`,
			row.TemplateVersionID,
		)
		if err != nil {
			versions.Close()
			return output, err
		}
		row.QuestionVersionOrder = []task6FixRound5QuestionVersionPosition{}
		for questionRows.Next() {
			var question task6FixRound5QuestionVersionPosition
			if err := questionRows.Scan(
				&question.QuestionVersionID, &question.Position,
			); err != nil {
				questionRows.Close()
				versions.Close()
				return output, err
			}
			row.QuestionVersionOrder = append(row.QuestionVersionOrder, question)
		}
		if err := questionRows.Err(); err != nil {
			questionRows.Close()
			versions.Close()
			return output, err
		}
		questionRows.Close()
		output.ChecklistVersions = append(output.ChecklistVersions, row)
	}
	if err := versions.Err(); err != nil {
		versions.Close()
		return output, err
	}
	versions.Close()
	return output, nil
}

func task6FixRound5PublicReviewDecision(
	row task6FixRound5ReviewDecisionArtifact,
) task6FixRound4DecisionArtifact {
	return task6FixRound4DecisionArtifact{
		DecisionID: row.DecisionID, Decision: row.Decision,
		CandidateRootID: row.CandidateRootID, CandidateID: row.CandidateID,
		CandidateRevision:           row.CandidateRevision,
		CandidateContentDigest:      row.CandidateContentDigest,
		ActorSubjectID:              row.ActorSubjectID,
		ActorDepartmentMembershipID: row.ActorDepartmentMembershipID,
		ActorDepartmentID:           row.ActorDepartmentID,
		ActorOrganizationalUnitID:   row.ActorOrganizationalUnitID,
		Reason:                      row.Reason, DecidedAt: row.DecidedAt,
		OperationID: row.OperationID, IdempotencyKey: row.IdempotencyKey,
		SemanticPayloadDigest: row.SemanticPayloadDigest,
		AuditEventID:          row.AuditEventID,
	}
}

func task6FixRound5PublicPublication(
	row task6FixRound5PublicationDecisionArtifact,
) task6FixRound4PublicationArtifact {
	return task6FixRound4PublicationArtifact{
		TemplateVersionID:     row.TemplateVersionID,
		PublicationDecisionID: row.PublicationDecisionID,
		CandidateRootID:       row.CandidateRootID, CandidateID: row.CandidateID,
		CandidateRevision:         row.CandidateRevision,
		CandidateContentDigest:    row.CandidateContentDigest,
		ActorSubjectID:            row.ActorSubjectID,
		ActorMembershipID:         row.ActorMembershipID,
		ActorDepartmentID:         row.ActorDepartmentID,
		ActorOrganizationalUnitID: row.ActorOrganizationalUnitID,
		Reason:                    row.Reason, DecidedAt: row.DecidedAt,
		PublishedAt: row.PublishedAt, OperationID: row.OperationID,
		IdempotencyKey:        row.IdempotencyKey,
		SemanticPayloadDigest: row.SemanticPayloadDigest,
		AuditEventID:          row.AuditEventID,
	}
}

// Break caught: independent mock and PostgreSQL examples can drift while each
// remains green. This exact matrix is loaded from the same checked-in contract
// as the semantic mock and drives the real canonical PostgreSQL handler.
func TestTask6FixRound4SharedArtifactParityContractCanonicalHandler(t *testing.T) {
	ctx := context.Background()
	contract := task6FixRound4LoadArtifactParityContract(t)
	if contract.ContractID != "task6-manager-artifact-parity-v1" {
		t.Fatalf("shared artifact parity contract id=%q", contract.ContractID)
	}
	service, _, submitted := task6SubmittedCandidate(
		t, "task6_fix4_shared_artifact_parity",
	)
	if submitted.CandidateID != contract.Candidate.CandidateID ||
		submitted.CandidateRootID != contract.Candidate.CandidateRootID ||
		submitted.Revision != contract.Candidate.Revision ||
		submitted.ContentDigest != contract.Candidate.ContentDigest {
		t.Fatalf("canonical candidate identity=%+v contract=%+v",
			submitted, contract.Candidate)
	}
	seedStatements := []struct {
		statement string
		arguments []any
	}{
		{`
			INSERT INTO identity_references(subject_id,issuer,display_name)
			VALUES ($1,'task6-fix4','Task 6 parity FOI manager')
			ON CONFLICT (subject_id) DO NOTHING`,
			[]any{contract.Actors.FOI.SubjectID}},
		{`
			INSERT INTO caa_department_memberships
				(id,subject_id,department_id,organizational_unit_id,
				 membership_role,status,effective_from)
			VALUES ($1,$2,$3,$4,'DEPARTMENT_MANAGER','ACTIVE','2025-01-01')`,
			[]any{
				contract.Actors.FOI.MembershipID, contract.Actors.FOI.SubjectID,
				contract.Actors.FOI.DepartmentID,
				contract.Actors.FOI.OrganizationalUnitID,
			}},
		{`
			INSERT INTO candidate_required_owner_assignments
				(id,candidate_draft_version_id,candidate_revision,
				 candidate_content_digest,department_id,
				 organizational_unit_id,approval_required)
			VALUES ('OWNER-TASK6-FIX4-PARITY-AIR',$1,$2,$3,$4,$5,true)`,
			[]any{
				submitted.CandidateID, submitted.Revision, submitted.ContentDigest,
				contract.Actors.AIR.DepartmentID,
				contract.Actors.AIR.OrganizationalUnitID,
			}},
	}
	for _, seed := range seedStatements {
		if _, err := service.Pool.Exec(
			ctx, seed.statement, seed.arguments...,
		); err != nil {
			t.Fatalf("seed shared canonical parity actors/owners: %v", err)
		}
	}
	api := httpapi.NewCanonicalAPI(httpapi.CanonicalAPIDependencies{
		Pool: service.Pool, Clock: service.Clock,
	})
	handler := httpapi.NewCanonicalTestBoundary("task6-fix4-parity-token").
		Protect(api.Handler())
	request := func(
		method, path, subject string, body any,
	) *httptest.ResponseRecorder {
		var encoded []byte
		if body != nil {
			encoded, _ = json.Marshal(body)
		}
		httpRequest := httptest.NewRequest(method, path, bytes.NewReader(encoded))
		httpRequest.Header.Set(
			httpapi.CanonicalTestTokenHeader, "task6-fix4-parity-token",
		)
		httpRequest.Header.Set(httpapi.CanonicalTestSubjectHeader, subject)
		if body != nil {
			httpRequest.Header.Set("Content-Type", "application/json")
		}
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httpRequest)
		return response
	}
	commandBody := func(
		operation task6FixRound4ParityOperation,
		digest string,
	) map[string]any {
		return map[string]any{
			"operationId":           operation.OperationID,
			"idempotencyKey":        operation.IdempotencyKey,
			"candidateId":           submitted.CandidateID,
			"expectedRevision":      submitted.Revision,
			"expectedContentDigest": digest,
			"reason":                operation.Reason,
		}
	}
	approvalPath := "/v1/department-manager/governed-checklist/candidates/" +
		submitted.CandidateID + "/technical-approvals"
	publicationPath := "/v1/department-manager/governed-checklist/candidates/" +
		submitted.CandidateID + "/publications"
	detailPath := "/v1/department-manager/governed-checklist/candidates/" +
		submitted.CandidateID
	contractAt, err := time.Parse(time.RFC3339, contract.Clock)
	if err != nil {
		t.Fatalf("parse shared Task 6 artifact clock: %v", err)
	}
	assertArtifacts := func(checkpoint task6FixRound5ArtifactCheckpoint) {
		t.Helper()
		actual, err := task6FixRound5LoadArtifactRows(
			ctx, service.Pool, submitted.CandidateID, contractAt,
		)
		if err != nil {
			t.Fatalf("load complete PostgreSQL Task 6 artifact rows: %v", err)
		}
		task6FixRound5AssertArtifactRows(
			t, actual,
			task6FixRound5ExpectedArtifactCheckpoint(t, contract, checkpoint),
		)
	}

	foiBody := commandBody(contract.Operations.FOIApproval, submitted.ContentDigest)
	partial := request(http.MethodPost, approvalPath, contract.Actors.FOI.SubjectID, foiBody)
	if partial.Code != http.StatusOK ||
		!bytes.Contains(partial.Body.Bytes(), []byte(
			`"status":"`+contract.Expected.PartialStatus+`"`,
		)) {
		t.Fatalf("canonical parity partial status=%d body=%s",
			partial.Code, partial.Body.String())
	}
	partialReplay := request(
		http.MethodPost, approvalPath, contract.Actors.FOI.SubjectID, foiBody,
	)
	if partialReplay.Code != http.StatusOK ||
		!bytes.Equal(partialReplay.Body.Bytes(), partial.Body.Bytes()) {
		t.Fatalf("canonical parity partial replay status=%d\nfirst=%s\nreplay=%s",
			partialReplay.Code, partial.Body.Bytes(), partialReplay.Body.Bytes())
	}
	var detail struct {
		Decisions []task6FixRound4DecisionArtifact `json:"decisions"`
	}
	detailResponse := request(
		http.MethodGet, detailPath, contract.Actors.FOI.SubjectID, nil,
	)
	if detailResponse.Code != http.StatusOK ||
		json.Unmarshal(detailResponse.Body.Bytes(), &detail) != nil {
		t.Fatalf("canonical parity approval detail status=%d body=%s",
			detailResponse.Code, detailResponse.Body.String())
	}
	approvalOnlyExpected := task6FixRound5ExpectedArtifactCheckpoint(
		t, contract, contract.Expected.ArtifactCheckpoints.ApprovalOnly,
	)
	expectedFOI := task6FixRound5PublicReviewDecision(
		approvalOnlyExpected.ReviewDecisions[0],
	)
	if len(detail.Decisions) != 1 || detail.Decisions[0] != expectedFOI {
		t.Fatalf("canonical parity approval-only decisions=%+v want=%+v",
			detail.Decisions, expectedFOI)
	}
	absent := request(
		http.MethodGet,
		"/v1/department-manager/governed-checklist/published-versions/"+
			contract.Candidate.AbsentTemplateVersionID,
		contract.Actors.FOI.SubjectID, nil,
	)
	if absent.Code != http.StatusNotFound {
		t.Fatalf("canonical parity approval-only publication status=%d body=%s",
			absent.Code, absent.Body.String())
	}
	assertArtifacts(contract.Expected.ArtifactCheckpoints.ApprovalOnly)
	conflictingFOI := commandBody(
		contract.Operations.FOIApproval, submitted.ContentDigest,
	)
	conflictingFOI["reason"] = contract.Operations.ConflictingReplayReason
	if response := request(
		http.MethodPost, approvalPath, contract.Actors.FOI.SubjectID, conflictingFOI,
	); response.Code != http.StatusConflict {
		t.Fatalf("canonical parity conflicting approval status=%d body=%s",
			response.Code, response.Body.String())
	}
	assertArtifacts(contract.Expected.ArtifactCheckpoints.ApprovalOnly)

	airBody := commandBody(contract.Operations.AIRApproval, submitted.ContentDigest)
	complete := request(http.MethodPost, approvalPath, contract.Actors.AIR.SubjectID, airBody)
	if complete.Code != http.StatusOK ||
		!bytes.Contains(complete.Body.Bytes(), []byte(
			`"status":"`+contract.Expected.FinalStatus+`"`,
		)) {
		t.Fatalf("canonical parity final status=%d body=%s",
			complete.Code, complete.Body.String())
	}
	detailResponse = request(
		http.MethodGet, detailPath, contract.Actors.AIR.SubjectID, nil,
	)
	detail = struct {
		Decisions []task6FixRound4DecisionArtifact `json:"decisions"`
	}{}
	if detailResponse.Code != http.StatusOK ||
		json.Unmarshal(detailResponse.Body.Bytes(), &detail) != nil {
		t.Fatalf("canonical parity joint detail status=%d body=%s",
			detailResponse.Code, detailResponse.Body.String())
	}
	jointExpected := task6FixRound5ExpectedArtifactCheckpoint(
		t, contract, contract.Expected.ArtifactCheckpoints.JointComplete,
	)
	expectedAIR := task6FixRound5PublicReviewDecision(
		jointExpected.ReviewDecisions[0],
	)
	expectedFOI = task6FixRound5PublicReviewDecision(
		jointExpected.ReviewDecisions[1],
	)
	if len(detail.Decisions) != 2 ||
		detail.Decisions[0] != expectedAIR || detail.Decisions[1] != expectedFOI {
		t.Fatalf("canonical parity joint decisions=%+v want=[%+v %+v]",
			detail.Decisions, expectedAIR, expectedFOI)
	}
	assertArtifacts(contract.Expected.ArtifactCheckpoints.JointComplete)

	tamperBody := commandBody(
		contract.Operations.DigestTamper, contract.Candidate.TamperedDigest,
	)
	if response := request(
		http.MethodPost, publicationPath, contract.Actors.FOI.SubjectID, tamperBody,
	); response.Code != http.StatusConflict {
		t.Fatalf("canonical parity digest tamper status=%d body=%s",
			response.Code, response.Body.String())
	}
	assertArtifacts(contract.Expected.ArtifactCheckpoints.JointComplete)

	publishBody := commandBody(
		contract.Operations.Publication, submitted.ContentDigest,
	)
	published := request(
		http.MethodPost, publicationPath, contract.Actors.FOI.SubjectID, publishBody,
	)
	if published.Code != http.StatusCreated {
		t.Fatalf("canonical parity publication status=%d body=%s",
			published.Code, published.Body.String())
	}
	var publication task6FixRound4PublicationArtifact
	if err := json.Unmarshal(published.Body.Bytes(), &publication); err != nil {
		t.Fatal(err)
	}
	publishedExpected := task6FixRound5ExpectedArtifactCheckpoint(
		t, contract, contract.Expected.ArtifactCheckpoints.Published,
	)
	expectedPublication := task6FixRound5PublicPublication(
		publishedExpected.PublicationDecisions[0],
	)
	if publication != expectedPublication {
		t.Fatalf("canonical parity publication=%+v want=%+v",
			publication, expectedPublication)
	}
	var publishedDetail struct {
		Candidate struct {
			Status string `json:"status"`
		} `json:"candidate"`
	}
	publishedDetailResponse := request(
		http.MethodGet, detailPath, contract.Actors.FOI.SubjectID, nil,
	)
	if publishedDetailResponse.Code != http.StatusOK ||
		json.Unmarshal(
			publishedDetailResponse.Body.Bytes(), &publishedDetail,
		) != nil ||
		publishedDetail.Candidate.Status != contract.Expected.PublishedStatus {
		t.Fatalf("canonical parity published detail status=%d body=%s",
			publishedDetailResponse.Code, publishedDetailResponse.Body.String())
	}
	publishedReplay := request(
		http.MethodPost, publicationPath, contract.Actors.FOI.SubjectID, publishBody,
	)
	if publishedReplay.Code != http.StatusCreated ||
		!bytes.Equal(publishedReplay.Body.Bytes(), published.Body.Bytes()) {
		t.Fatalf("canonical parity publication replay status=%d\nfirst=%s\nreplay=%s",
			publishedReplay.Code, published.Body.Bytes(), publishedReplay.Body.Bytes())
	}
	artifactPath := "/v1/department-manager/governed-checklist/published-versions/" +
		publication.TemplateVersionID
	artifactResponse := request(
		http.MethodGet, artifactPath, contract.Actors.FOI.SubjectID, nil,
	)
	if artifactResponse.Code != http.StatusOK {
		t.Fatalf("canonical parity artifact status=%d body=%s",
			artifactResponse.Code, artifactResponse.Body.String())
	}
	var artifact struct {
		Publication task6FixRound4PublicationArtifact `json:"publication"`
		Mappings    []regulatory.ComplianceMapping    `json:"mappings"`
		Questions   []regulatory.ChecklistQuestion    `json:"questions"`
	}
	if err := json.Unmarshal(artifactResponse.Body.Bytes(), &artifact); err != nil {
		t.Fatal(err)
	}
	actualBytes, _ := json.Marshal(map[string]any{
		"mappings": artifact.Mappings, "questions": artifact.Questions,
	})
	expectedBytes, _ := json.Marshal(contract.PublishedArtifact)
	if artifact.Publication != expectedPublication ||
		!bytes.Equal(actualBytes, expectedBytes) {
		t.Fatalf("canonical parity immutable artifact=%s want=%s",
			artifactResponse.Body.Bytes(), expectedBytes)
	}
	artifact.Mappings[0].MappingID = "TAMPERED-CLONE"
	afterCloneTamper := request(
		http.MethodGet, artifactPath, contract.Actors.FOI.SubjectID, nil,
	)
	if afterCloneTamper.Code != http.StatusOK ||
		!bytes.Equal(afterCloneTamper.Body.Bytes(), artifactResponse.Body.Bytes()) {
		t.Fatalf("canonical parity clone tamper changed artifact status=%d",
			afterCloneTamper.Code)
	}
	conflictingPublication := commandBody(
		contract.Operations.Publication, submitted.ContentDigest,
	)
	conflictingPublication["reason"] = contract.Operations.ConflictingReplayReason
	if response := request(
		http.MethodPost, publicationPath,
		contract.Actors.FOI.SubjectID, conflictingPublication,
	); response.Code != http.StatusConflict {
		t.Fatalf("canonical parity conflicting publication status=%d body=%s",
			response.Code, response.Body.String())
	}
	for _, method := range []string{http.MethodPut, http.MethodDelete} {
		if response := request(
			method, artifactPath, contract.Actors.FOI.SubjectID,
			map[string]any{"mappings": []any{}},
		); response.Code != http.StatusMethodNotAllowed {
			t.Fatalf("canonical parity immutable %s status=%d body=%s",
				method, response.Code, response.Body.String())
		}
	}
	assertArtifacts(contract.Expected.ArtifactCheckpoints.Published)
}

// Break caught: mock-only internal maps cannot prove that the canonical
// manager HTTP boundary exposes exact semantic decision/Audit identity and
// immutable ordered publication snapshots without a public mutation command.
func TestTask6FixRound3CanonicalManagerExposesImmutableSemanticArtifacts(t *testing.T) {
	ctx := context.Background()
	service, _, submitted := task6SubmittedCandidate(t, "task6_fix3_semantic_http")
	if _, err := service.Pool.Exec(ctx, `
		INSERT INTO identity_references (subject_id,issuer,display_name)
		VALUES ('USR-MANAGER-NORA','task6-fix3','Nora Department Manager')
		ON CONFLICT (subject_id) DO NOTHING;
		INSERT INTO caa_department_memberships
			(id,subject_id,department_id,organizational_unit_id,
			 membership_role,status,effective_from)
		VALUES ('MEM-TASK6-NORA','USR-MANAGER-NORA',
			'FLIGHT_OPERATIONS_INSPECTORATE','FLIGHT_OPERATIONS_INSPECTORATE',
			'DEPARTMENT_MANAGER','ACTIVE','2025-01-01')`); err != nil {
		t.Fatalf("seed canonical semantic manager: %v", err)
	}
	manager := identity.Principal{
		SubjectID: "USR-MANAGER-NORA",
		Roles:     []identity.Role{identity.RoleDepartmentManager},
	}
	approveCommand := checklistgovernance.ReviewCommand{
		OperationID:    "TASK6-FIX3-SEMANTIC-APPROVE",
		IdempotencyKey: "TASK6-FIX3-SEMANTIC-APPROVE",
		CandidateID:    submitted.CandidateID, ExpectedRevision: submitted.Revision,
		ExpectedContentDigest: submitted.ContentDigest,
		Reason:                "Expose exact approval identity through canonical HTTP.",
	}
	approved, err := service.Approve(ctx, manager, approveCommand)
	if err != nil {
		t.Fatalf("approve semantic artifact candidate: %v", err)
	}
	api := httpapi.NewCanonicalAPI(httpapi.CanonicalAPIDependencies{
		Pool: service.Pool, Clock: service.Clock,
	})
	handler := httpapi.NewCanonicalTestBoundary("task6-fix3-semantic-token").
		Protect(api.Handler())
	request := func(method, path string, body any) *httptest.ResponseRecorder {
		var encoded []byte
		if body != nil {
			encoded, _ = json.Marshal(body)
		}
		httpRequest := httptest.NewRequest(method, path, bytes.NewReader(encoded))
		httpRequest.Header.Set(
			httpapi.CanonicalTestTokenHeader, "task6-fix3-semantic-token",
		)
		httpRequest.Header.Set(
			httpapi.CanonicalTestSubjectHeader, manager.SubjectID,
		)
		if body != nil {
			httpRequest.Header.Set("Content-Type", "application/json")
		}
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httpRequest)
		return response
	}
	detail := request(
		http.MethodGet,
		"/v1/department-manager/governed-checklist/candidates/"+approved.CandidateID,
		nil,
	)
	if detail.Code != http.StatusOK {
		t.Fatalf("semantic candidate detail status=%d body=%s",
			detail.Code, detail.Body.String())
	}
	var decisionOutput struct {
		Decisions []struct {
			DecisionID, Decision, CandidateRootID, CandidateID string
			CandidateRevision                                  int64
			CandidateContentDigest, ActorSubjectID             string
			ActorDepartmentMembershipID, ActorDepartmentID     string
			ActorOrganizationalUnitID, Reason, DecidedAt       string
			OperationID, IdempotencyKey, SemanticPayloadDigest string
			AuditEventID                                       string
		} `json:"decisions"`
	}
	if err := json.Unmarshal(detail.Body.Bytes(), &decisionOutput); err != nil {
		t.Fatal(err)
	}
	if len(decisionOutput.Decisions) != 1 {
		t.Fatalf("semantic decision count=%d", len(decisionOutput.Decisions))
	}
	decision := decisionOutput.Decisions[0]
	var expectedApprovalSemantic string
	if err := service.Pool.QueryRow(ctx, `
		SELECT semantic_payload_digest
		FROM department_review_decisions WHERE operation_id=$1`,
		approveCommand.OperationID,
	).Scan(&expectedApprovalSemantic); err != nil {
		t.Fatal(err)
	}
	expectedAt := time.Date(2026, 7, 29, 16, 0, 0, 0, time.UTC)
	if decision.DecisionID != "DRD-"+approveCommand.OperationID ||
		decision.Decision != "TECHNICALLY_APPROVED" ||
		decision.CandidateRootID != approved.CandidateRootID ||
		decision.CandidateID != approved.CandidateID ||
		decision.CandidateRevision != approved.Revision ||
		decision.CandidateContentDigest != approved.ContentDigest ||
		decision.ActorSubjectID != manager.SubjectID ||
		decision.ActorDepartmentMembershipID != "MEM-TASK6-NORA" ||
		decision.ActorDepartmentID != "FLIGHT_OPERATIONS_INSPECTORATE" ||
		decision.ActorOrganizationalUnitID != "FLIGHT_OPERATIONS_INSPECTORATE" ||
		decision.Reason != approveCommand.Reason ||
		decision.DecidedAt != expectedAt.Format(time.RFC3339) ||
		decision.OperationID != approveCommand.OperationID ||
		decision.IdempotencyKey != approveCommand.IdempotencyKey ||
		decision.SemanticPayloadDigest != expectedApprovalSemantic ||
		decision.AuditEventID != "AE-"+approveCommand.OperationID {
		t.Fatalf("canonical exact review decision=%+v", decision)
	}
	absent := request(
		http.MethodGet,
		"/v1/department-manager/governed-checklist/published-versions/CTV-ABSENT",
		nil,
	)
	if absent.Code != http.StatusNotFound {
		t.Fatalf("approval-only publication artifact status=%d body=%s",
			absent.Code, absent.Body.String())
	}
	publishCommand := checklistgovernance.PublicationCommand{
		OperationID:    "TASK6-FIX3-SEMANTIC-PUBLISH",
		IdempotencyKey: "TASK6-FIX3-SEMANTIC-PUBLISH",
		CandidateID:    approved.CandidateID, ExpectedRevision: approved.Revision,
		ExpectedContentDigest: approved.ContentDigest,
		Reason:                "Expose exact immutable publication artifact.",
	}
	published, err := service.Publish(ctx, manager, publishCommand)
	if err != nil {
		t.Fatalf("publish semantic artifact: %v", err)
	}
	artifactResponse := request(
		http.MethodGet,
		"/v1/department-manager/governed-checklist/published-versions/"+
			published.TemplateVersionID,
		nil,
	)
	if artifactResponse.Code != http.StatusOK {
		t.Fatalf("published artifact status=%d body=%s",
			artifactResponse.Code, artifactResponse.Body.String())
	}
	var artifact struct {
		Publication struct {
			TemplateVersionID, PublicationDecisionID, CandidateRootID string
			CandidateID                                               string
			CandidateRevision                                         int64
			CandidateContentDigest, ActorSubjectID                    string
			ActorDepartmentMembershipID, ActorDepartmentID            string
			ActorOrganizationalUnitID, Reason, DecidedAt, PublishedAt string
			OperationID, IdempotencyKey, SemanticPayloadDigest        string
			AuditEventID                                              string
		} `json:"publication"`
		Mappings  []regulatory.ComplianceMapping `json:"mappings"`
		Questions []regulatory.ChecklistQuestion `json:"questions"`
	}
	if err := json.Unmarshal(artifactResponse.Body.Bytes(), &artifact); err != nil {
		t.Fatal(err)
	}
	if artifact.Publication.TemplateVersionID != published.TemplateVersionID ||
		artifact.Publication.PublicationDecisionID != published.PublicationDecisionID ||
		artifact.Publication.CandidateRootID != published.CandidateRootID ||
		artifact.Publication.CandidateID != published.CandidateID ||
		artifact.Publication.CandidateRevision != published.CandidateRevision ||
		artifact.Publication.CandidateContentDigest != published.CandidateContentDigest ||
		artifact.Publication.ActorSubjectID != published.ActorSubjectID ||
		artifact.Publication.ActorDepartmentMembershipID != published.ActorMembershipID ||
		artifact.Publication.ActorDepartmentID != published.ActorDepartmentID ||
		artifact.Publication.ActorOrganizationalUnitID != published.ActorUnitID ||
		artifact.Publication.Reason != published.Reason ||
		artifact.Publication.DecidedAt != published.DecidedAt.Format(time.RFC3339) ||
		artifact.Publication.PublishedAt != published.PublishedAt.Format(time.RFC3339) ||
		artifact.Publication.OperationID != published.OperationID ||
		artifact.Publication.IdempotencyKey != published.IdempotencyKey ||
		artifact.Publication.SemanticPayloadDigest != published.SemanticPayloadDigest ||
		artifact.Publication.AuditEventID != published.AuditEventID {
		t.Fatalf("canonical exact publication=%+v want=%+v",
			artifact.Publication, published)
	}
	actualBytes, _ := json.Marshal(map[string]any{
		"mappings": artifact.Mappings, "questions": artifact.Questions,
	})
	expectedBytes, _ := json.Marshal(map[string]any{
		"mappings": approved.Mappings, "questions": approved.Questions,
	})
	if !bytes.Equal(actualBytes, expectedBytes) {
		t.Fatalf("immutable snapshot bytes=%s want=%s", actualBytes, expectedBytes)
	}
	beforeTamper := append([]byte(nil), artifactResponse.Body.Bytes()...)
	conflicting := publishCommand
	conflicting.Reason = "Conflicting semantic tamper must have zero effects."
	if _, err := service.Publish(ctx, manager, conflicting); !errors.Is(err, application.ErrConflict) {
		t.Fatalf("conflicting publication replay error=%v, want conflict", err)
	}
	afterTamper := request(
		http.MethodGet,
		"/v1/department-manager/governed-checklist/published-versions/"+
			published.TemplateVersionID,
		nil,
	)
	if afterTamper.Code != http.StatusOK ||
		!bytes.Equal(afterTamper.Body.Bytes(), beforeTamper) {
		t.Fatalf("command tamper changed immutable artifact status=%d\nbefore=%s\nafter=%s",
			afterTamper.Code, beforeTamper, afterTamper.Body.Bytes())
	}
	mutationDenied := request(
		http.MethodPost,
		"/v1/department-manager/governed-checklist/published-versions/"+
			published.TemplateVersionID,
		map[string]any{"mappings": []any{}},
	)
	if mutationDenied.Code != http.StatusMethodNotAllowed {
		t.Fatalf("public mutation helper status=%d body=%s",
			mutationDenied.Code, mutationDenied.Body.String())
	}
}

type task6FixRound3EditedSuccessor struct {
	RootID      string
	CandidateID string
	Digest      string
	Mappings    []regulatory.ComplianceMapping
}

type task6FixRound5FrozenDecision struct {
	ID                          string  `json:"id"`
	CandidateRootID             *string `json:"candidateRootId"`
	CandidateID                 string  `json:"candidateId"`
	CandidateRevision           int64   `json:"candidateRevision"`
	CandidateContentDigest      string  `json:"candidateContentDigest"`
	Decision                    string  `json:"decision"`
	ActorSubjectID              string  `json:"actorSubjectId"`
	ActorDepartmentMembershipID string  `json:"actorDepartmentMembershipId"`
	ActorDepartmentID           string  `json:"actorDepartmentId"`
	ActorOrganizationalUnitID   string  `json:"actorOrganizationalUnitId"`
	Reason                      string  `json:"reason"`
	DecidedAt                   string  `json:"decidedAt"`
	OperationID                 string  `json:"operationId"`
	IdempotencyKey              string  `json:"idempotencyKey"`
	SemanticPayloadDigest       string  `json:"semanticPayloadDigest"`
	CreatedAt                   string  `json:"createdAt"`
}

type task6FixRound5FrozenAudit struct {
	SequenceID     int64          `json:"sequenceId"`
	EventID        string         `json:"eventId"`
	OccurredAt     string         `json:"occurredAt"`
	ActorSubjectID string         `json:"actorSubjectId"`
	ActorRole      string         `json:"actorRole"`
	OrganizationID *string        `json:"organizationId"`
	Action         string         `json:"action"`
	EntityType     string         `json:"entityType"`
	EntityID       string         `json:"entityId"`
	EntityVersion  int64          `json:"entityVersion"`
	BeforeStatus   string         `json:"beforeStatus"`
	AfterStatus    string         `json:"afterStatus"`
	Reason         string         `json:"reason"`
	OperationID    string         `json:"operationId"`
	CorrelationID  string         `json:"correlationId"`
	RequestID      string         `json:"requestId"`
	ClosureBasis   *string        `json:"closureBasis"`
	Details        map[string]any `json:"details"`
}

type task6FixRound5FrozenPublication struct {
	PublicationDecisionID     string                                  `json:"publicationDecisionId"`
	TemplateVersionID         string                                  `json:"templateVersionId"`
	TemplateID                string                                  `json:"templateId"`
	Version                   int                                     `json:"version"`
	Title                     string                                  `json:"title"`
	CandidateRootID           string                                  `json:"candidateRootId"`
	CandidateID               string                                  `json:"candidateId"`
	CandidateRevision         int64                                   `json:"candidateRevision"`
	CandidateContentDigest    string                                  `json:"candidateContentDigest"`
	DecidedAt                 string                                  `json:"decidedAt"`
	CreatedAt                 string                                  `json:"createdAt"`
	PublishedAt               string                                  `json:"publishedAt"`
	ActorSubjectID            string                                  `json:"actorSubjectId"`
	ActorMembershipID         string                                  `json:"actorDepartmentMembershipId"`
	ActorDepartmentID         string                                  `json:"actorDepartmentId"`
	ActorOrganizationalUnitID string                                  `json:"actorOrganizationalUnitId"`
	Reason                    string                                  `json:"reason"`
	OperationID               string                                  `json:"operationId"`
	IdempotencyKey            string                                  `json:"idempotencyKey"`
	SemanticPayloadDigest     string                                  `json:"semanticPayloadDigest"`
	AuditEventID              string                                  `json:"auditEventId"`
	QuestionVersionOrder      []task6FixRound5QuestionVersionPosition `json:"questionVersionOrder"`
}

type task6FixRound5FrozenSuccessorContract struct {
	ContractID         string                          `json:"contractId"`
	Candidate          regulatory.CandidateView        `json:"candidate"`
	PreRepairDecision  task6FixRound5FrozenDecision    `json:"preRepairDecision"`
	PostRepairDecision task6FixRound5FrozenDecision    `json:"postRepairDecision"`
	Audit              task6FixRound5FrozenAudit       `json:"audit"`
	Publication        task6FixRound5FrozenPublication `json:"publication"`
}

func task6FixRound5LoadFrozenSuccessorContract(
	t *testing.T,
) task6FixRound5FrozenSuccessorContract {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(
		apiModuleRoot(t), "tests", "fixtures", "task6",
		"frozen-edited-successor-contract.json",
	))
	if err != nil {
		t.Fatalf("read frozen edited successor contract: %v", err)
	}
	var contract task6FixRound5FrozenSuccessorContract
	if err := json.Unmarshal(raw, &contract); err != nil {
		t.Fatalf("decode frozen edited successor contract: %v", err)
	}
	return contract
}

func task6FixRound5ExactJSON(
	t *testing.T,
	label string,
	actual any,
	expected any,
) []byte {
	t.Helper()
	actualBytes, actualErr := json.Marshal(actual)
	expectedBytes, expectedErr := json.Marshal(expected)
	if actualErr != nil || expectedErr != nil {
		t.Fatalf("encode %s: actual=%v expected=%v",
			label, actualErr, expectedErr)
	}
	if !bytes.Equal(actualBytes, expectedBytes) {
		t.Fatalf("%s mismatch\nactual=%s\nwant=%s",
			label, actualBytes, expectedBytes)
	}
	return actualBytes
}

func task6FixRound5LoadFrozenDecision(
	t *testing.T,
	pool *database.Pool,
	candidateID string,
	rootColumnExists bool,
) []task6FixRound5FrozenDecision {
	t.Helper()
	rootExpression := "NULL::text"
	if rootColumnExists {
		rootExpression = "candidate_root_id"
	}
	rows, err := pool.Query(context.Background(), fmt.Sprintf(`
		SELECT id,%s,candidate_draft_version_id,candidate_revision,
		       candidate_content_digest,decision,actor_subject_id,
		       actor_department_membership_id,actor_department_id,
		       actor_organizational_unit_id,reason,
		       to_char(decided_at AT TIME ZONE 'UTC',
		               'YYYY-MM-DD"T"HH24:MI:SS"Z"'),
		       operation_id,idempotency_key,semantic_payload_digest,
		       to_char(created_at AT TIME ZONE 'UTC',
		               'YYYY-MM-DD"T"HH24:MI:SS"Z"')
		FROM department_review_decisions
		WHERE candidate_draft_version_id=$1
		ORDER BY id`, rootExpression), candidateID)
	if err != nil {
		t.Fatalf("load complete frozen successor decision: %v", err)
	}
	defer rows.Close()
	output := []task6FixRound5FrozenDecision{}
	for rows.Next() {
		var row task6FixRound5FrozenDecision
		if err := rows.Scan(
			&row.ID, &row.CandidateRootID, &row.CandidateID,
			&row.CandidateRevision, &row.CandidateContentDigest, &row.Decision,
			&row.ActorSubjectID, &row.ActorDepartmentMembershipID,
			&row.ActorDepartmentID, &row.ActorOrganizationalUnitID,
			&row.Reason, &row.DecidedAt, &row.OperationID,
			&row.IdempotencyKey, &row.SemanticPayloadDigest, &row.CreatedAt,
		); err != nil {
			t.Fatalf("scan complete frozen successor decision: %v", err)
		}
		output = append(output, row)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate complete frozen successor decisions: %v", err)
	}
	return output
}

func task6FixRound5LoadFrozenAudit(
	t *testing.T,
	pool *database.Pool,
	candidateID string,
) []task6FixRound5FrozenAudit {
	t.Helper()
	rows, err := pool.Query(context.Background(), `
		SELECT sequence_id,event_id,
		       to_char(occurred_at AT TIME ZONE 'UTC',
		               'YYYY-MM-DD"T"HH24:MI:SS"Z"'),
		       actor_subject_id,actor_role,organization_id,action,entity_type,
		       entity_id,entity_version,before_status,after_status,reason,
		       operation_id,correlation_id,request_id,closure_basis,details
		FROM audit_events
		WHERE entity_id=$1
		ORDER BY sequence_id`, candidateID)
	if err != nil {
		t.Fatalf("load complete frozen successor Audit: %v", err)
	}
	defer rows.Close()
	output := []task6FixRound5FrozenAudit{}
	for rows.Next() {
		var row task6FixRound5FrozenAudit
		var details []byte
		if err := rows.Scan(
			&row.SequenceID, &row.EventID, &row.OccurredAt,
			&row.ActorSubjectID, &row.ActorRole, &row.OrganizationID,
			&row.Action, &row.EntityType, &row.EntityID, &row.EntityVersion,
			&row.BeforeStatus, &row.AfterStatus, &row.Reason,
			&row.OperationID, &row.CorrelationID, &row.RequestID,
			&row.ClosureBasis, &details,
		); err != nil {
			t.Fatalf("scan complete frozen successor Audit: %v", err)
		}
		if err := json.Unmarshal(details, &row.Details); err != nil {
			t.Fatalf("decode frozen successor Audit details: %v", err)
		}
		output = append(output, row)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate complete frozen successor Audits: %v", err)
	}
	return output
}

func task6FixRound5LoadFrozenQuestions(
	t *testing.T,
	pool *database.Pool,
	candidateID string,
) []regulatory.ChecklistQuestion {
	t.Helper()
	rows, err := pool.Query(context.Background(), `
		SELECT snapshot
		FROM regulatory_generated_question_snapshots
		WHERE candidate_draft_version_id=$1
		ORDER BY question_id`, candidateID)
	if err != nil {
		t.Fatalf("load frozen successor question snapshots: %v", err)
	}
	defer rows.Close()
	questions := []regulatory.ChecklistQuestion{}
	for rows.Next() {
		var raw []byte
		var question regulatory.ChecklistQuestion
		if err := rows.Scan(&raw); err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(raw, &question); err != nil {
			t.Fatal(err)
		}
		questions = append(questions, question)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return questions
}

func task6FixRound4ApplyFrozenEditedSuccessor(
	t *testing.T,
) (*database.Pool, task6FixRound3EditedSuccessor) {
	t.Helper()
	ctx := context.Background()
	pool := task6FixRound2ApplyFrozenV21(t)
	rootID, mappings := task6FixRound2SeedFrozenHistory(t, pool)
	raw, err := os.ReadFile(filepath.Join(
		apiModuleRoot(t), "tests", "fixtures", "task6", "pre-task6-v21.sql",
	))
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.SplitN(
		string(raw), task6FixRound4FrozenSuccessorMarker, 2,
	)
	if len(parts) != 2 {
		t.Fatal("frozen v21 SQL is missing checked-in successor data")
	}
	dataAndTail := strings.SplitN(
		parts[1], task6FixRound4FrozenSuccessorEndMarker, 2,
	)
	if len(dataAndTail) != 2 {
		t.Fatal("frozen v21 SQL is missing checked-in successor end marker")
	}
	if _, err := pool.Exec(ctx, dataAndTail[0]); err != nil {
		t.Fatalf("apply checked-in frozen edited successor: %v", err)
	}
	appended := mappings[0]
	appended.MappingID = "MAP-N-FROZEN-APPENDED"
	return pool, task6FixRound3EditedSuccessor{
		RootID:      rootID,
		CandidateID: "CAND-TASK6-FIX4-EDITED",
		Digest: "sha256:" +
			"97a386e78b9555dd3eec5be0b7e499d5ab6236d7285c5a5a4faad80137ba762c",
		Mappings: append(mappings, appended),
	}
}

func task6FixRound3SeedEditedSuccessor(
	t *testing.T,
	pool *database.Pool,
	unrecoverable bool,
) task6FixRound3EditedSuccessor {
	t.Helper()
	ctx := context.Background()
	rootID, mappings := task6FixRound2SeedFrozenHistory(t, pool)
	if unrecoverable {
		orphan := mappings[0]
		orphan.MappingID = "MAP-UNRECOVERABLE-NEW"
		mappings = []regulatory.ComplianceMapping{orphan}
	}
	question := regulatory.SyntheticCandidateBundle().InspectionChecklist.Questions[0]
	question.QuestionID = "Q-TASK6-FIX3-EDITED"
	if unrecoverable {
		question.MappingIDs = []string{"MAP-NOT-PRESENT-IN-SUCCESSOR"}
	} else {
		question.MappingIDs = []string{mappings[0].MappingID, mappings[1].MappingID}
	}
	digest, err := regulatory.CanonicalSHA256(map[string]any{
		"complianceMappings": mappings,
		"inspectionChecklist": map[string]any{
			"checklistId": "TPL-TASK6-FIX2-FROZEN",
			"questions":   []regulatory.ChecklistQuestion{question},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	candidateID := "CAND-TASK6-FIX3-EDITED"
	if unrecoverable {
		candidateID = "CAND-TASK6-FIX3-UNRECOVERABLE"
	}
	questionVersionID := "QV-" + candidateID
	if _, err := pool.Exec(ctx, `
		INSERT INTO question_versions
			(id,question_id,version,prompt,configured_reference,expected_evidence,
			 created_by_subject_id)
		VALUES ($1,$2,2,$3,$4,$5,'USR-TASK6-FIX2-FROZEN-ADMIN')`,
		questionVersionID, question.QuestionID, question.Prompt,
		joinTask6Strings(question.MappingIDs), joinTask6Strings(question.ExpectedEvidence),
	); err != nil {
		t.Fatalf("seed edited successor question version: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO template_draft_versions
			(id,template_id,version,status,owner_role,creator_subject_id,change_reason,
			 question_version_ids,revision,generation_run_id,candidate_content_digest,
			 candidate_schema_version,candidate_root_id,supersedes_candidate_id)
		SELECT $1,template_id,version+1,'TECHNICALLY_APPROVED',owner_role,
		       'USR-TASK6-FIX2-FROZEN-ADMIN','Frozen edited successor.',
		       ARRAY[$2],revision+1,generation_run_id,$3,candidate_schema_version,
		       candidate_root_id,id
		FROM template_draft_versions WHERE id=$4`,
		candidateID, questionVersionID, digest, rootID,
	); err != nil {
		t.Fatalf("seed edited successor: %v", err)
	}
	for _, mapping := range mappings {
		raw, err := json.Marshal(mapping)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := pool.Exec(ctx, `
			INSERT INTO regulatory_generated_mapping_snapshots
				(candidate_draft_version_id,mapping_id,snapshot)
			VALUES ($1,$2,$3::jsonb)`,
			candidateID, mapping.MappingID, string(raw),
		); err != nil {
			t.Fatalf("seed successor mapping %s: %v", mapping.MappingID, err)
		}
	}
	questionRaw, err := json.Marshal(question)
	if err != nil {
		t.Fatal(err)
	}
	operationID := "TASK6-FIX3-EDITED-APPROVE"
	if unrecoverable {
		operationID = "TASK6-FIX3-UNRECOVERABLE-APPROVE"
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO regulatory_generated_question_snapshots
			(candidate_draft_version_id,question_id,snapshot)
		VALUES ($1,$2,$3::jsonb)`,
		candidateID, question.QuestionID, string(questionRaw),
	); err != nil {
		t.Fatalf("seed edited successor question snapshot: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO candidate_required_owner_assignments
			(id,candidate_draft_version_id,candidate_revision,candidate_content_digest,
			 department_id,organizational_unit_id,approval_required)
		VALUES ('OWNER-' || $1,$1,2,$2,
			'FLIGHT_OPERATIONS_INSPECTORATE','FLIGHT_OPERATIONS_INSPECTORATE',true)`,
		candidateID, digest,
	); err != nil {
		t.Fatalf("seed edited successor owner: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO audit_events
			(event_id,occurred_at,actor_subject_id,actor_role,action,entity_type,
			 entity_id,entity_version,before_status,after_status,reason,operation_id,
			 correlation_id,request_id,details)
		VALUES ('AE-' || $2,'2026-01-04T00:00:00Z',
			'USR-TASK6-FIX2-FROZEN-MANAGER','manager',
			'TECHNICAL_APPROVAL_RECORDED','GOVERNED_CANDIDATE',$1,2,
			'DEPARTMENT_REVIEW','TECHNICALLY_APPROVED',
			'Frozen edited successor approval.',$2,$2,$2,'{}')`,
		candidateID, operationID,
	); err != nil {
		t.Fatalf("seed edited successor Audit: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO department_review_decisions
			(id,candidate_draft_version_id,candidate_revision,candidate_content_digest,
			 decision,actor_subject_id,actor_department_membership_id,
			 actor_department_id,actor_organizational_unit_id,reason,decided_at,
			 operation_id,idempotency_key,semantic_payload_digest)
		VALUES ('DRD-' || $3,$1,2,$2,'TECHNICALLY_APPROVED',
			'USR-TASK6-FIX2-FROZEN-MANAGER','MEM-TASK6-FIX2-FROZEN',
			'FLIGHT_OPERATIONS_INSPECTORATE','FLIGHT_OPERATIONS_INSPECTORATE',
			'Frozen edited successor approval.','2026-01-04T00:00:00Z',
			$3,$3,'sha256:dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd')`,
		candidateID, digest, operationID,
	); err != nil {
		t.Fatalf("seed edited successor history: %v", err)
	}
	return task6FixRound3EditedSuccessor{
		RootID: rootID, CandidateID: candidateID, Digest: digest, Mappings: mappings,
	}
}

func joinTask6Strings(values []string) string {
	var output string
	for index, value := range values {
		if index > 0 {
			output += ","
		}
		output += value
	}
	return output
}

// Break caught: a dynamically seeded positive successor can pass repair tests
// while the checked-in frozen v21 fixture still contains no edited history.
func TestTask6FixRound4FrozenSQLContainsEditedSuccessorAndAppendedMapping(
	t *testing.T,
) {
	raw, err := os.ReadFile(filepath.Join(
		apiModuleRoot(t), "tests", "fixtures", "task6", "pre-task6-v21.sql",
	))
	if err != nil {
		t.Fatal(err)
	}
	fixture := string(raw)
	for _, required := range []string{
		"CAND-TASK6-FIX4-EDITED",
		"MAP-Z-FROZEN-FIRST",
		"MAP-A-FROZEN-SECOND",
		"MAP-N-FROZEN-APPENDED",
		"Q-TASK6-FIX4-EDITED",
		"DRD-TASK6-FIX4-EDITED-APPROVE",
		"AE-TASK6-FIX4-EDITED-APPROVE",
	} {
		if !strings.Contains(fixture, required) {
			t.Fatalf("frozen v21 SQL is missing positive successor artifact %q", required)
		}
	}
}

// Break caught: a checked-in reviewed successor with reverse/non-lexical
// inherited identities plus a new mapping must inherit predecessor order,
// append the new reference from its ordered question snapshot, and retain
// exact projection/publication bytes and history.
func TestTask6FixRound4FrozenEditedSuccessorRepairPreservesExactOrderDigestAndHistory(
	t *testing.T,
) {
	ctx := context.Background()
	pool, fixture := task6FixRound4ApplyFrozenEditedSuccessor(t)
	contract := task6FixRound5LoadFrozenSuccessorContract(t)
	if fixture.RootID != contract.Candidate.CandidateRootID ||
		fixture.CandidateID != contract.Candidate.CandidateID ||
		fixture.Digest != contract.Candidate.ContentDigest {
		t.Fatalf("checked-in successor fixture identity=%+v contract=%+v",
			fixture, contract.Candidate)
	}
	preDecisionBytes := task6FixRound5ExactJSON(
		t, "complete pre-repair successor decision",
		task6FixRound5LoadFrozenDecision(t, pool, fixture.CandidateID, false),
		[]task6FixRound5FrozenDecision{contract.PreRepairDecision},
	)
	preAuditBytes := task6FixRound5ExactJSON(
		t, "complete pre-repair successor Audit",
		task6FixRound5LoadFrozenAudit(t, pool, fixture.CandidateID),
		[]task6FixRound5FrozenAudit{contract.Audit},
	)
	preQuestionBytes := task6FixRound5ExactJSON(
		t, "complete pre-repair successor question snapshots",
		task6FixRound5LoadFrozenQuestions(t, pool, fixture.CandidateID),
		contract.Candidate.Questions,
	)
	if err := migrations.Apply(ctx, pool); err != nil {
		t.Fatalf("repair frozen edited successor: %v", err)
	}
	postDecisions := task6FixRound5LoadFrozenDecision(
		t, pool, fixture.CandidateID, true,
	)
	postDecisionBytes := task6FixRound5ExactJSON(
		t, "complete post-repair successor decision",
		postDecisions,
		[]task6FixRound5FrozenDecision{contract.PostRepairDecision},
	)
	if bytes.Equal(preDecisionBytes, postDecisionBytes) {
		t.Fatal("repair did not add the required successor decision root")
	}
	postDecisions[0].CandidateRootID = nil
	if !bytes.Equal(
		task6FixRound5ExactJSON(
			t, "successor decision with repaired root removed",
			postDecisions,
			[]task6FixRound5FrozenDecision{contract.PreRepairDecision},
		),
		preDecisionBytes,
	) {
		t.Fatal("successor decision changed beyond its repaired root")
	}
	postAuditBytes := task6FixRound5ExactJSON(
		t, "complete post-repair successor Audit",
		task6FixRound5LoadFrozenAudit(t, pool, fixture.CandidateID),
		[]task6FixRound5FrozenAudit{contract.Audit},
	)
	if !bytes.Equal(preAuditBytes, postAuditBytes) {
		t.Fatal("repair changed frozen successor Audit bytes")
	}
	postQuestionBytes := task6FixRound5ExactJSON(
		t, "complete post-repair successor question snapshots",
		task6FixRound5LoadFrozenQuestions(t, pool, fixture.CandidateID),
		contract.Candidate.Questions,
	)
	if !bytes.Equal(preQuestionBytes, postQuestionBytes) {
		t.Fatal("repair changed frozen successor question bytes")
	}
	var mappingIDs []string
	if err := pool.QueryRow(ctx, `
		SELECT array_agg(mapping_id ORDER BY mapping_ordinal)
		FROM regulatory_generated_mapping_snapshots
		WHERE candidate_draft_version_id=$1`,
		fixture.CandidateID,
	).Scan(&mappingIDs); err != nil {
		t.Fatal(err)
	}
	if len(mappingIDs) != 3 ||
		mappingIDs[0] != fixture.Mappings[0].MappingID ||
		mappingIDs[1] != fixture.Mappings[1].MappingID ||
		mappingIDs[2] != fixture.Mappings[2].MappingID {
		t.Fatalf("edited successor mapping order=%v, want [%s %s %s]",
			mappingIDs, fixture.Mappings[0].MappingID,
			fixture.Mappings[1].MappingID, fixture.Mappings[2].MappingID)
	}
	candidate, err := regulatory.LoadCandidateForGovernance(ctx, pool, fixture.CandidateID)
	if err != nil {
		t.Fatal(err)
	}
	task6FixRound5ExactJSON(
		t, "complete repaired successor projection",
		candidate, contract.Candidate,
	)
	recomputedDigest, err := regulatory.CanonicalSHA256(map[string]any{
		"complianceMappings": contract.Candidate.Mappings,
		"inspectionChecklist": map[string]any{
			"checklistId": contract.Candidate.TemplateID,
			"questions":   contract.Candidate.Questions,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if recomputedDigest != contract.Candidate.ContentDigest {
		t.Fatalf("repaired successor recomputed digest=%s want=%s",
			recomputedDigest, contract.Candidate.ContentDigest)
	}
	manager := identity.Principal{
		SubjectID: "USR-TASK6-FIX2-FROZEN-MANAGER",
		Roles:     []identity.Role{identity.RoleDepartmentManager},
	}
	service := checklistgovernance.NewService(pool, func() time.Time {
		return time.Date(2026, 7, 29, 21, 0, 0, 0, time.UTC)
	})
	published, err := service.Publish(ctx, manager, checklistgovernance.PublicationCommand{
		OperationID:           contract.Publication.OperationID,
		IdempotencyKey:        contract.Publication.IdempotencyKey,
		CandidateID:           contract.Candidate.CandidateID,
		ExpectedRevision:      contract.Candidate.Revision,
		ExpectedContentDigest: contract.Candidate.ContentDigest,
		Reason:                contract.Publication.Reason,
	})
	if err != nil {
		t.Fatalf("publish repaired edited successor: %v", err)
	}
	if published.TemplateVersionID != contract.Publication.TemplateVersionID ||
		published.PublicationDecisionID != contract.Publication.PublicationDecisionID ||
		published.SemanticPayloadDigest != contract.Publication.SemanticPayloadDigest {
		t.Fatalf("published successor identity=%+v want=%+v",
			published, contract.Publication)
	}
	artifactRows, err := task6FixRound5LoadArtifactRows(
		ctx, pool, fixture.CandidateID,
		time.Date(2026, 7, 29, 21, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatal(err)
	}
	expectedPublication := task6FixRound5PublicationDecisionArtifact{
		PublicationDecisionID:     contract.Publication.PublicationDecisionID,
		TemplateVersionID:         contract.Publication.TemplateVersionID,
		CandidateRootID:           contract.Publication.CandidateRootID,
		CandidateID:               contract.Publication.CandidateID,
		CandidateRevision:         contract.Publication.CandidateRevision,
		CandidateContentDigest:    contract.Publication.CandidateContentDigest,
		ActorSubjectID:            contract.Publication.ActorSubjectID,
		ActorMembershipID:         contract.Publication.ActorMembershipID,
		ActorMembershipIsCurrent:  true,
		ActorDepartmentID:         contract.Publication.ActorDepartmentID,
		ActorOrganizationalUnitID: contract.Publication.ActorOrganizationalUnitID,
		Reason:                    contract.Publication.Reason,
		DecidedAt:                 contract.Publication.DecidedAt,
		CreatedAt:                 contract.Publication.CreatedAt,
		PublishedAt:               contract.Publication.PublishedAt,
		OperationID:               contract.Publication.OperationID,
		IdempotencyKey:            contract.Publication.IdempotencyKey,
		SemanticPayloadDigest:     contract.Publication.SemanticPayloadDigest,
		AuditEventID:              contract.Publication.AuditEventID,
	}
	task6FixRound5ExactJSON(
		t, "complete repaired successor publication decision",
		artifactRows.PublicationDecisions,
		[]task6FixRound5PublicationDecisionArtifact{expectedPublication},
	)
	var expectedSnapshot any
	expectedSnapshotBytes, err := json.Marshal(map[string]any{
		"candidateId":            contract.Candidate.CandidateID,
		"candidateRevision":      contract.Candidate.Revision,
		"candidateContentDigest": contract.Candidate.ContentDigest,
		"complianceMappings":     contract.Candidate.Mappings,
		"questions":              contract.Candidate.Questions,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(expectedSnapshotBytes, &expectedSnapshot); err != nil {
		t.Fatal(err)
	}
	expectedVersion := task6FixRound5ChecklistVersionArtifact{
		TemplateVersionID:      contract.Publication.TemplateVersionID,
		TemplateID:             contract.Publication.TemplateID,
		Version:                contract.Publication.Version,
		Title:                  contract.Publication.Title,
		PublishedAt:            contract.Publication.PublishedAt,
		CandidateRootID:        contract.Publication.CandidateRootID,
		CandidateID:            contract.Publication.CandidateID,
		CandidateRevision:      contract.Publication.CandidateRevision,
		CandidateContentDigest: contract.Publication.CandidateContentDigest,
		PublicationDecisionID:  contract.Publication.PublicationDecisionID,
		AuditEventID:           contract.Publication.AuditEventID,
		QuestionVersionOrder:   contract.Publication.QuestionVersionOrder,
		ImmutableSnapshot:      expectedSnapshot,
	}
	task6FixRound5ExactJSON(
		t, "complete repaired successor published mapping/question version",
		artifactRows.ChecklistVersions,
		[]task6FixRound5ChecklistVersionArtifact{expectedVersion},
	)
}

func reflectTask6JSONEqual(left, right any) bool {
	leftBytes, leftErr := json.Marshal(left)
	rightBytes, rightErr := json.Marshal(right)
	return leftErr == nil && rightErr == nil && bytes.Equal(leftBytes, rightBytes)
}

// Break caught: a successor mapping order absent from both its predecessor and
// ordered question snapshots must abort the complete forward repair instead of
// receiving lexical ordinals or partially mutating governed history.
func TestTask6FixRound3UnrecoverableSuccessorOrderRollsBackRepair(t *testing.T) {
	ctx := context.Background()
	pool := task6FixRound2ApplyFrozenV21(t)
	fixture := task6FixRound3SeedEditedSuccessor(t, pool, true)
	before := task6FixRound2HistorySnapshot(t, pool, fixture.RootID)
	var successorBefore string
	if err := pool.QueryRow(ctx, `
		SELECT jsonb_build_object(
			'candidate',to_jsonb(candidate),
			'mappings',(SELECT jsonb_agg(to_jsonb(mapping) ORDER BY mapping.mapping_id)
			            FROM regulatory_generated_mapping_snapshots mapping
			            WHERE mapping.candidate_draft_version_id=candidate.id),
			'decisions',(SELECT jsonb_agg(to_jsonb(decision) ORDER BY decision.id)
			             FROM department_review_decisions decision
			             WHERE decision.candidate_draft_version_id=candidate.id),
			'audits',(SELECT jsonb_agg(to_jsonb(audit) ORDER BY audit.event_id)
			          FROM audit_events audit WHERE audit.entity_id=candidate.id)
		)::text
		FROM template_draft_versions candidate WHERE candidate.id=$1`,
		fixture.CandidateID,
	).Scan(&successorBefore); err != nil {
		t.Fatal(err)
	}
	if err := migrations.Apply(ctx, pool); err == nil {
		t.Fatal("unrecoverable successor mapping order unexpectedly repaired")
	}
	after := task6FixRound2HistorySnapshot(t, pool, fixture.RootID)
	for label, expected := range before {
		if after[label] != expected {
			t.Fatalf("failed repair mutated frozen root %s history", label)
		}
	}
	var successorAfter string
	if err := pool.QueryRow(ctx, `
		SELECT jsonb_build_object(
			'candidate',to_jsonb(candidate),
			'mappings',(SELECT jsonb_agg(to_jsonb(mapping) ORDER BY mapping.mapping_id)
			            FROM regulatory_generated_mapping_snapshots mapping
			            WHERE mapping.candidate_draft_version_id=candidate.id),
			'decisions',(SELECT jsonb_agg(to_jsonb(decision) ORDER BY decision.id)
			             FROM department_review_decisions decision
			             WHERE decision.candidate_draft_version_id=candidate.id),
			'audits',(SELECT jsonb_agg(to_jsonb(audit) ORDER BY audit.event_id)
			          FROM audit_events audit WHERE audit.entity_id=candidate.id)
		)::text
		FROM template_draft_versions candidate WHERE candidate.id=$1`,
		fixture.CandidateID,
	).Scan(&successorAfter); err != nil {
		t.Fatal(err)
	}
	if successorAfter != successorBefore {
		t.Fatalf("failed repair changed edited successor history\nbefore=%s\nafter=%s",
			successorBefore, successorAfter)
	}
	var ordinalColumns int
	if err := pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM information_schema.columns
		WHERE table_schema='public'
		  AND table_name='regulatory_generated_mapping_snapshots'
		  AND column_name='mapping_ordinal'`,
	).Scan(&ordinalColumns); err != nil || ordinalColumns != 0 {
		t.Fatalf("failed repair left mapping ordinal column count=%d err=%v",
			ordinalColumns, err)
	}
}

type task6FixRound3NormalizedInventory struct {
	Columns     map[string]string `json:"columns"`
	Constraints map[string]string `json:"constraints"`
	Indexes     map[string]string `json:"indexes"`
	Triggers    map[string]string `json:"triggers"`
	Functions   map[string]string `json:"functions"`
}

var task6FixRound3InventoryRelations = []string{
	"candidate_required_owner_assignments",
	"checklist_publication_decisions",
	"checklist_template_versions",
	"department_review_decisions",
	"regulatory_generated_mapping_snapshots",
	"template_draft_versions",
}

func task6FixRound3DefinitionSHA256(definition string) string {
	normalized := strings.Join(strings.Fields(definition), " ")
	digest := sha256.Sum256([]byte(normalized))
	return fmt.Sprintf("sha256:%x", digest)
}

func task6FixRound3LoadNormalizedInventory(
	t *testing.T,
	pool *database.Pool,
) task6FixRound3NormalizedInventory {
	t.Helper()
	ctx := context.Background()
	inventory := task6FixRound3NormalizedInventory{
		Columns: map[string]string{}, Constraints: map[string]string{},
		Indexes: map[string]string{}, Triggers: map[string]string{},
		Functions: map[string]string{},
	}
	rows, err := pool.Query(ctx, `
		SELECT relation.relname,attribute.attname,
		       format_type(attribute.atttypid,attribute.atttypmod),
		       attribute.attnotnull,
		       COALESCE(pg_get_expr(default_value.adbin,default_value.adrelid),'')
		FROM pg_class relation
		JOIN pg_namespace namespace ON namespace.oid=relation.relnamespace
		JOIN pg_attribute attribute ON attribute.attrelid=relation.oid
		LEFT JOIN pg_attrdef default_value
		  ON default_value.adrelid=relation.oid
		 AND default_value.adnum=attribute.attnum
		WHERE namespace.nspname='public'
		  AND relation.relname=ANY($1)
		  AND attribute.attnum>0
		  AND NOT attribute.attisdropped
		ORDER BY relation.relname,attribute.attnum`,
		task6FixRound3InventoryRelations,
	)
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var relation, column, dataType, defaultValue string
		var notNull bool
		if err := rows.Scan(&relation, &column, &dataType, &notNull, &defaultValue); err != nil {
			rows.Close()
			t.Fatal(err)
		}
		key := relation + "." + column
		inventory.Columns[key] = task6FixRound3DefinitionSHA256(fmt.Sprintf(
			"type=%s|nullability=%t|default=%s", dataType, !notNull, defaultValue,
		))
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		t.Fatal(err)
	}
	rows.Close()
	rows, err = pool.Query(ctx, `
		SELECT relation.relname,constraint_value.conname,
		       pg_get_constraintdef(constraint_value.oid,true)
		FROM pg_constraint constraint_value
		JOIN pg_class relation ON relation.oid=constraint_value.conrelid
		JOIN pg_namespace namespace ON namespace.oid=relation.relnamespace
		WHERE namespace.nspname='public'
		  AND relation.relname=ANY($1)
		ORDER BY relation.relname,constraint_value.conname`,
		task6FixRound3InventoryRelations,
	)
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var relation, name, definition string
		if err := rows.Scan(&relation, &name, &definition); err != nil {
			rows.Close()
			t.Fatal(err)
		}
		inventory.Constraints[relation+"."+name] =
			task6FixRound3DefinitionSHA256(definition)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		t.Fatal(err)
	}
	rows.Close()
	rows, err = pool.Query(ctx, `
		SELECT relation.relname,index_relation.relname,
		       pg_get_indexdef(index_relation.oid)
		FROM pg_index index_value
		JOIN pg_class relation ON relation.oid=index_value.indrelid
		JOIN pg_class index_relation ON index_relation.oid=index_value.indexrelid
		JOIN pg_namespace namespace ON namespace.oid=relation.relnamespace
		WHERE namespace.nspname='public'
		  AND relation.relname=ANY($1)
		ORDER BY relation.relname,index_relation.relname`,
		task6FixRound3InventoryRelations,
	)
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var relation, name, definition string
		if err := rows.Scan(&relation, &name, &definition); err != nil {
			rows.Close()
			t.Fatal(err)
		}
		inventory.Indexes[relation+"."+name] =
			task6FixRound3DefinitionSHA256(definition)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		t.Fatal(err)
	}
	rows.Close()
	rows, err = pool.Query(ctx, `
		SELECT relation.relname,trigger_value.tgname,
		       CASE
			       WHEN (trigger_value.tgtype & 2)=2 THEN 'BEFORE'
			       WHEN (trigger_value.tgtype & 64)=64 THEN 'INSTEAD OF'
			       ELSE 'AFTER'
		       END,
		       trigger_value.tgtype::integer,
		       trigger_value.tgenabled::text,
		       function_namespace.nspname || '.' || function_value.proname
		FROM pg_trigger trigger_value
		JOIN pg_class relation ON relation.oid=trigger_value.tgrelid
		JOIN pg_namespace namespace ON namespace.oid=relation.relnamespace
		JOIN pg_proc function_value ON function_value.oid=trigger_value.tgfoid
		JOIN pg_namespace function_namespace
		  ON function_namespace.oid=function_value.pronamespace
		WHERE namespace.nspname='public'
		  AND relation.relname=ANY($1)
		  AND NOT trigger_value.tgisinternal
		ORDER BY relation.relname,trigger_value.tgname`,
		task6FixRound3InventoryRelations,
	)
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var relation, name, timing, enabled, function string
		var eventMask int
		if err := rows.Scan(
			&relation, &name, &timing, &eventMask, &enabled, &function,
		); err != nil {
			rows.Close()
			t.Fatal(err)
		}
		definition := fmt.Sprintf(
			"relation=%s|timing=%s|eventMask=%d|enabled=%s|function=%s",
			relation, timing, eventMask, enabled, function,
		)
		inventory.Triggers[relation+"."+name] =
			task6FixRound3DefinitionSHA256(definition)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		t.Fatal(err)
	}
	rows.Close()
	rows, err = pool.Query(ctx, `
		SELECT DISTINCT function_namespace.nspname || '.' || function_value.proname,
		       pg_get_functiondef(function_value.oid)
		FROM pg_trigger trigger_value
		JOIN pg_class relation ON relation.oid=trigger_value.tgrelid
		JOIN pg_namespace namespace ON namespace.oid=relation.relnamespace
		JOIN pg_proc function_value ON function_value.oid=trigger_value.tgfoid
		JOIN pg_namespace function_namespace
		  ON function_namespace.oid=function_value.pronamespace
		WHERE namespace.nspname='public'
		  AND relation.relname=ANY($1)
		  AND NOT trigger_value.tgisinternal
		ORDER BY function_namespace.nspname || '.' || function_value.proname`,
		task6FixRound3InventoryRelations,
	)
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var name, definition string
		if err := rows.Scan(&name, &definition); err != nil {
			rows.Close()
			t.Fatal(err)
		}
		inventory.Functions[name] = task6FixRound3DefinitionSHA256(definition)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		t.Fatal(err)
	}
	rows.Close()
	return inventory
}

func task6FixRound3ExpectedInventory(t *testing.T) task6FixRound3NormalizedInventory {
	t.Helper()
	path := filepath.Join(
		apiModuleRoot(t), "tests", "fixtures", "task6",
		"frozen-v21-normalized-inventory.json",
	)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read frozen normalized Task 6 inventory: %v", err)
	}
	var inventory task6FixRound3NormalizedInventory
	if err := json.Unmarshal(raw, &inventory); err != nil {
		t.Fatalf("decode frozen normalized Task 6 inventory: %v", err)
	}
	return inventory
}

func task6FixRound3AssertInventory(
	t *testing.T,
	actual, expected task6FixRound3NormalizedInventory,
) {
	t.Helper()
	actualJSON, err := json.MarshalIndent(actual, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	expectedJSON, err := json.MarshalIndent(expected, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(actualJSON, expectedJSON) {
		t.Fatalf("complete normalized Task 6 inventory mismatch\nactual=%s\nwant=%s",
			actualJSON, expectedJSON)
	}
}

// Break caught: a changed Task 6 default, constraint clause, trigger event
// mask, or trigger function body must be restored to the complete reviewed
// frozen-v21 inventory; partial phrase checks cannot admit catalog drift.
func TestTask6FixRound3FrozenV21RepairsCompletePinnedInventory(t *testing.T) {
	ctx := context.Background()
	expected := task6FixRound3ExpectedInventory(t)
	clean := task6FixRound2ApplyFrozenV21(t)
	task6FixRound2SeedFrozenHistory(t, clean)
	if err := migrations.Apply(ctx, clean); err != nil {
		t.Fatalf("repair clean frozen-v21 fixture: %v", err)
	}
	task6FixRound3AssertInventory(
		t, task6FixRound3LoadNormalizedInventory(t, clean), expected,
	)

	mutated := task6FixRound2ApplyFrozenV21(t)
	task6FixRound2SeedFrozenHistory(t, mutated)
	if _, err := mutated.Exec(ctx, `
		ALTER TABLE department_review_decisions
			ALTER COLUMN decided_at SET DEFAULT now();
		ALTER TABLE department_review_decisions
			DROP CONSTRAINT department_review_decisions_reason_check;
		ALTER TABLE department_review_decisions
			ADD CONSTRAINT department_review_decisions_reason_check CHECK (true);
		DROP TRIGGER checklist_publication_decisions_approval_guard
			ON checklist_publication_decisions;
		CREATE TRIGGER checklist_publication_decisions_approval_guard
			BEFORE INSERT OR UPDATE ON checklist_publication_decisions
			FOR EACH ROW
			EXECUTE FUNCTION validate_governed_publication_approval();
		CREATE OR REPLACE FUNCTION validate_governed_publication_approval()
			RETURNS trigger LANGUAGE plpgsql AS $broken$
		BEGIN
			RETURN NEW;
		END;
		$broken$;
	`); err != nil {
		t.Fatalf("mutate frozen catalog definitions: %v", err)
	}
	if err := migrations.Apply(ctx, mutated); err != nil {
		t.Fatalf("repair mutated frozen-v21 fixture: %v", err)
	}
	task6FixRound3AssertInventory(
		t, task6FixRound3LoadNormalizedInventory(t, mutated), expected,
	)
}
