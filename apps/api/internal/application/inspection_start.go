package application

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/assignments"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/datafeed"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/identity"
	"github.com/jackc/pgx/v5"
)

// StartInspectionCommand is deliberately separate from materialization.  A
// materialized package is a durable, read-only preparation artifact; this
// command is the only transition that opens execution and grants field access.
type StartInspectionCommand struct {
	OperationID                string
	CorrelationID              string
	InspectionID               string
	ExpectedInspectionRevision int64
}

type StartedInspection struct {
	InspectionID       string             `json:"inspectionId"`
	AssignmentID       string             `json:"assignmentId"`
	InspectionStatus   string             `json:"inspectionStatus"`
	AssignmentStatus   assignments.Status `json:"assignmentStatus"`
	InspectionRevision int64              `json:"inspectionRevision"`
	ChecklistRevision  int64              `json:"checklistRevision"`
	StartedAt          time.Time          `json:"startedAt"`
}

func (service *Service) StartInspection(
	ctx context.Context,
	actor identity.Principal,
	command StartInspectionCommand,
) (StartedInspection, error) {
	return executeTransition(ctx, service, actor, commandEnvelope{
		OperationID: command.OperationID, CorrelationID: command.CorrelationID,
		Kind: "start_inspection", EntityID: command.InspectionID,
		Semantic: struct {
			ExpectedInspectionRevision int64 `json:"expectedInspectionRevision"`
		}{command.ExpectedInspectionRevision},
	}, func(ctx context.Context, transaction pgx.Tx) (transition[StartedInspection], error) {
		if !actor.HasRole(identity.RoleInspector) || actor.SubjectID == "" {
			return transition[StartedInspection]{}, fmt.Errorf("%w: assigned Inspector role required", ErrForbidden)
		}
		if command.ExpectedInspectionRevision <= 0 || command.InspectionID == "" {
			return transition[StartedInspection]{}, ErrInvalid
		}
		var assignmentID, organizationID, inspectionType, inspectionStatus, assignmentStatus, checklistStatus string
		var inspectionRevision, assignmentRevision, checklistRevision int64
		var assigned bool
		if err := transaction.QueryRow(ctx, `
			SELECT assignment.id, inspection.organization_id, inspection.inspection_type,
			       inspection.status, inspection.revision, assignment.status, assignment.revision,
			       checklist.status, checklist.revision,
			       EXISTS (SELECT 1 FROM inspection_question_assignments question_assignment
			               WHERE question_assignment.inspection_id=inspection.id
			                 AND question_assignment.subject_id=$2)
			FROM inspections inspection
			JOIN audit_assignments assignment ON assignment.inspection_id=inspection.id
			JOIN inspection_checklists checklist ON checklist.inspection_id=inspection.id
			WHERE inspection.id=$1 AND inspection.tombstoned_at IS NULL
		  AND assignment.tombstoned_at IS NULL
			FOR UPDATE OF inspection, assignment, checklist
		`, command.InspectionID, actor.SubjectID).Scan(
			&assignmentID, &organizationID, &inspectionType, &inspectionStatus,
			&inspectionRevision, &assignmentStatus, &assignmentRevision,
			&checklistStatus, &checklistRevision, &assigned,
		); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return transition[StartedInspection]{}, ErrNotFound
			}
			return transition[StartedInspection]{}, err
		}
		if !assigned {
			return transition[StartedInspection]{}, fmt.Errorf("%w: Inspector is not assigned to this Audit", ErrForbidden)
		}
		if inspectionRevision != command.ExpectedInspectionRevision || inspectionStatus != "READY_TO_EXECUTE" || checklistStatus != "NOT_STARTED" {
			return transition[StartedInspection]{}, ErrConflict
		}
		if assignmentStatus != string(assignments.StatusConfirmed) && assignmentStatus != string(assignments.StatusScheduled) && assignmentStatus != string(assignments.StatusReady) {
			return transition[StartedInspection]{}, ErrConflict
		}
		var packageID string
		var packageExpiry *time.Time
		var packageRevoked *time.Time
		if err := transaction.QueryRow(ctx, `
			SELECT id, expires_at, revoked_at FROM inspection_packages
			WHERE inspection_id=$1 ORDER BY package_version DESC LIMIT 1 FOR UPDATE
		`, command.InspectionID).Scan(&packageID, &packageExpiry, &packageRevoked); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return transition[StartedInspection]{}, ErrNotFound
			}
			return transition[StartedInspection]{}, err
		}
		now := service.clock().UTC()
		if packageRevoked != nil || (packageExpiry != nil && !now.Before(*packageExpiry)) {
			return transition[StartedInspection]{}, fmt.Errorf("%w: inspection package is expired or withdrawn", ErrConflict)
		}
		if _, err := transaction.Exec(ctx, `UPDATE inspections SET status='IN_PROGRESS', revision=revision+1, updated_at=$2 WHERE id=$1 AND status='READY_TO_EXECUTE' AND revision=$3`, command.InspectionID, now, command.ExpectedInspectionRevision); err != nil {
			return transition[StartedInspection]{}, err
		}
		if _, err := transaction.Exec(ctx, `UPDATE inspection_checklists SET status='IN_PROGRESS', revision=revision+1 WHERE inspection_id=$1 AND status='NOT_STARTED' AND revision=$2`, command.InspectionID, checklistRevision); err != nil {
			return transition[StartedInspection]{}, err
		}
		if _, err := transaction.Exec(ctx, `UPDATE audit_assignments SET status='READY', revision=revision+1, updated_at=$2 WHERE id=$1 AND revision=$3`, assignmentID, now, assignmentRevision); err != nil {
			return transition[StartedInspection]{}, err
		}
		scopeCode, err := dataFeedAuditScopeCode(inspectionType)
		if err != nil {
			return transition[StartedInspection]{}, err
		}
		eventID, err := datafeed.NewEventID()
		if err != nil {
			return transition[StartedInspection]{}, err
		}
		correlationID := command.CorrelationID
		if correlationID == "" {
			correlationID = command.OperationID
		}
		response := StartedInspection{
			InspectionID: command.InspectionID, AssignmentID: assignmentID,
			InspectionStatus: "IN_PROGRESS", AssignmentStatus: assignments.StatusReady,
			InspectionRevision: inspectionRevision + 1, ChecklistRevision: checklistRevision + 1,
			StartedAt: now,
		}
		return transition[StartedInspection]{
			Response: response, OrganizationID: organizationID,
			Action: "inspection.started", EntityType: "inspection", EntityID: command.InspectionID,
			EntityVersion: inspectionRevision + 1, BeforeStatus: "READY_TO_EXECUTE", AfterStatus: "IN_PROGRESS",
			Reason:   "Inspector start opened the separately authorized execution window.",
			SyncKind: "inspection", OutboxTopic: "inspection.started",
			DataFeedEvents: []datafeed.EventInput{{
				EventID: eventID, EventType: "audit.started", OwningOrganizationID: organizationID,
				ActorOrganizationID: actor.OrganizationID, CorrelationID: correlationID,
				AggregateType: "audit", AggregateID: command.InspectionID, AggregateRevision: inspectionRevision + 1,
				EffectiveAt: now, KnownAt: now, OccurredAt: now, EmittedAt: now,
				VisibilityPurposeCode: "regulated_oversight", EntityRefs: map[string]any{"audit_id": command.InspectionID},
				StateBefore: "audit_planned", StateAfter: "audit_in_progress",
				Payload: map[string]any{"started_at": now.Format(time.RFC3339Nano), "audit_scope_code": scopeCode, "package_id": packageID},
			}},
		}, nil
	})
}
