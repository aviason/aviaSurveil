package assignments

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/identity"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/platform/database"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/platform/idempotency"
	"github.com/jackc/pgx/v5"
)

var (
	ErrForbidden = errors.New("assignment forbidden")
	ErrConflict  = errors.New("assignment conflict")
	ErrInvalid   = errors.New("invalid assignment command")
	ErrNotFound  = errors.New("assignment not found")
)

type Dependencies struct {
	Clock       func() time.Time
	IDGenerator func(string) string
}

type Service struct {
	pool        *database.Pool
	clock       func() time.Time
	idGenerator func(string) string
}

func NewService(pool *database.Pool, dependencies Dependencies) *Service {
	clock := dependencies.Clock
	if clock == nil {
		clock = time.Now
	}
	idGenerator := dependencies.IDGenerator
	if idGenerator == nil {
		idGenerator = randomID
	}
	return &Service{pool: pool, clock: clock, idGenerator: idGenerator}
}

type PrepareCommand struct {
	OperationID              string
	IdempotencyKey           string
	PlanningItemID           string
	InspectionID             string
	ExpectedPlanningRevision int64
}

func (service *Service) Prepare(
	ctx context.Context,
	actor identity.Principal,
	command PrepareCommand,
) (Preparation, error) {
	if !CanPrepare(actor) {
		return Preparation{}, ErrForbidden
	}
	if blank(command.OperationID, command.IdempotencyKey, command.PlanningItemID, command.InspectionID) ||
		command.ExpectedPlanningRevision <= 0 {
		return Preparation{}, ErrInvalid
	}
	return executeCommand(ctx, service, actor, "prepare_audit", command.OperationID,
		command.IdempotencyKey, command.InspectionID, command,
		func(ctx context.Context, transaction pgx.Tx) (commandResult[Preparation], error) {
			var organizationID, title, inspectionType, status string
			var scheduledDate time.Time
			var revision int64
			if err := transaction.QueryRow(ctx, `
				SELECT organization_id, title, inspection_type, status, scheduled_date, revision
				FROM surveillance_plan_items
				WHERE id = $1 AND tombstoned_at IS NULL
				FOR UPDATE
			`, command.PlanningItemID).Scan(
				&organizationID, &title, &inspectionType, &status, &scheduledDate, &revision,
			); err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					return commandResult[Preparation]{}, ErrNotFound
				}
				return commandResult[Preparation]{}, err
			}
			if revision != command.ExpectedPlanningRevision || status != "RELEASED" {
				return commandResult[Preparation]{}, ErrConflict
			}
			var alreadyPrepared string
			err := transaction.QueryRow(ctx, `
				SELECT COALESCE(values->>'preparedAuditId', '')
				FROM planning_intake_drafts
				WHERE submitted_planning_item_id = $1 AND tombstoned_at IS NULL
				FOR UPDATE
			`, command.PlanningItemID).Scan(&alreadyPrepared)
			if err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					return commandResult[Preparation]{}, ErrNotFound
				}
				return commandResult[Preparation]{}, err
			}
			if alreadyPrepared != "" {
				return commandResult[Preparation]{}, ErrConflict
			}
			now := service.clock().UTC()
			if _, err := transaction.Exec(ctx, `
				INSERT INTO inspections (
					id, organization_id, assigned_inspector_subject_id, title,
					inspection_type, status, due_date, revision, created_at, updated_at
				) VALUES ($1, $2, NULL, $3, $4, $5, $6, 1, $7, $7)
			`, command.InspectionID, organizationID, title, inspectionType,
				string(StatusPreparation), scheduledDate, now); err != nil {
				return commandResult[Preparation]{}, err
			}
			if _, err := transaction.Exec(ctx, `
				UPDATE planning_intake_drafts
				SET values = jsonb_set(values, '{preparedAuditId}', to_jsonb($2::text), true),
				    updated_at = $3
				WHERE submitted_planning_item_id = $1 AND tombstoned_at IS NULL
			`, command.PlanningItemID, command.InspectionID, now); err != nil {
				return commandResult[Preparation]{}, err
			}
			output := Preparation{
				PlanningItemID: command.PlanningItemID, InspectionID: command.InspectionID,
				OrganizationID: organizationID, Status: StatusPreparation, Revision: 1,
			}
			return commandResult[Preparation]{
				Response: output, OrganizationID: organizationID,
				Action: "planning.preparation_started", EntityType: "inspection",
				EntityID: command.InspectionID, EntityVersion: 1,
				AfterStatus: string(StatusPreparation),
			}, nil
		})
}

type AssignLeadCommand struct {
	OperationID                string
	IdempotencyKey             string
	AssignmentID               string
	InspectionID               string
	ExpectedInspectionRevision int64
	LeadSubjectID              string
	ScheduledStartDate         string
	ScheduledEndDate           string
}

func (service *Service) AssignLead(
	ctx context.Context,
	actor identity.Principal,
	command AssignLeadCommand,
) (Assignment, error) {
	if !CanAssignLead(actor) {
		return Assignment{}, ErrForbidden
	}
	start, end, err := parseSchedule(command.ScheduledStartDate, command.ScheduledEndDate)
	if err != nil || blank(command.OperationID, command.IdempotencyKey, command.AssignmentID,
		command.InspectionID, command.LeadSubjectID) || command.ExpectedInspectionRevision <= 0 {
		return Assignment{}, ErrInvalid
	}
	return executeCommand(ctx, service, actor, "assign_lead", command.OperationID,
		command.IdempotencyKey, command.AssignmentID, command,
		func(ctx context.Context, transaction pgx.Tx) (commandResult[Assignment], error) {
			var organizationID, status string
			var revision int64
			if err := transaction.QueryRow(ctx, `
				SELECT organization_id, status, revision
				FROM inspections
				WHERE id = $1 AND tombstoned_at IS NULL
				FOR UPDATE
			`, command.InspectionID).Scan(&organizationID, &status, &revision); err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					return commandResult[Assignment]{}, ErrNotFound
				}
				return commandResult[Assignment]{}, err
			}
			if revision != command.ExpectedInspectionRevision || Status(status) != StatusPreparation {
				return commandResult[Assignment]{}, ErrConflict
			}
			if err := requireActiveRole(ctx, transaction, command.LeadSubjectID, identity.RoleLeadInspector); err != nil {
				return commandResult[Assignment]{}, err
			}
			now := service.clock().UTC()
			if _, err := transaction.Exec(ctx, `
				INSERT INTO audit_assignments (
					id, inspection_id, organization_id, lead_subject_id, status,
					scheduled_start_date, scheduled_end_date, revision, created_at, updated_at
				) VALUES ($1, $2, $3, $4, $5, $6, $7, 1, $8, $8)
			`, command.AssignmentID, command.InspectionID, organizationID, command.LeadSubjectID,
				string(StatusLeadAssigned), start, end, now); err != nil {
				return commandResult[Assignment]{}, err
			}
			if _, err := transaction.Exec(ctx, `
				INSERT INTO audit_team_members (
					assignment_id, subject_id, member_role, revision, created_at
				) VALUES ($1, $2, 'LEAD_INSPECTOR', 1, $3)
			`, command.AssignmentID, command.LeadSubjectID, now); err != nil {
				return commandResult[Assignment]{}, err
			}
			if _, err := transaction.Exec(ctx, `
				UPDATE inspections
				SET assigned_inspector_subject_id = $2, revision = revision + 1, updated_at = $3
				WHERE id = $1 AND revision = $4
			`, command.InspectionID, command.LeadSubjectID, now, revision); err != nil {
				return commandResult[Assignment]{}, err
			}
			output := Assignment{
				ID: command.AssignmentID, InspectionID: command.InspectionID,
				OrganizationID: organizationID, LeadSubjectID: command.LeadSubjectID,
				MemberSubjectIDs: []string{command.LeadSubjectID}, Status: StatusLeadAssigned,
				ScheduledStartDate: command.ScheduledStartDate,
				ScheduledEndDate:   command.ScheduledEndDate, Revision: 1,
			}
			return commandResult[Assignment]{
				Response: output, OrganizationID: organizationID,
				Action: "assignment.lead_assigned", EntityType: "audit_assignment",
				EntityID: command.AssignmentID, EntityVersion: 1,
				AfterStatus: string(StatusLeadAssigned),
			}, nil
		})
}

type AssignTeamCommand struct {
	OperationID      string
	IdempotencyKey   string
	AssignmentID     string
	ExpectedRevision int64
	MemberSubjectIDs []string
}

func (service *Service) AssignTeam(
	ctx context.Context,
	actor identity.Principal,
	command AssignTeamCommand,
) (Assignment, error) {
	if !actor.HasRole(identity.RoleLeadInspector) {
		return Assignment{}, ErrForbidden
	}
	members, err := normalizedIDs(command.MemberSubjectIDs)
	if err != nil || blank(command.OperationID, command.IdempotencyKey, command.AssignmentID) ||
		command.ExpectedRevision <= 0 || len(members) == 0 {
		return Assignment{}, ErrInvalid
	}
	semantic := command
	semantic.MemberSubjectIDs = members
	return executeCommand(ctx, service, actor, "assign_team", command.OperationID,
		command.IdempotencyKey, command.AssignmentID, semantic,
		func(ctx context.Context, transaction pgx.Tx) (commandResult[Assignment], error) {
			current, err := getAssignmentForUpdate(ctx, transaction, command.AssignmentID)
			if err != nil {
				return commandResult[Assignment]{}, err
			}
			if !CanConfigureTeam(actor, current.LeadSubjectID) {
				return commandResult[Assignment]{}, ErrForbidden
			}
			if current.Revision != command.ExpectedRevision || current.Status != StatusLeadAssigned {
				return commandResult[Assignment]{}, ErrConflict
			}
			for _, subjectID := range members {
				if err := requireActiveRole(ctx, transaction, subjectID, identity.RoleInspector); err != nil {
					return commandResult[Assignment]{}, err
				}
			}
			now := service.clock().UTC()
			for _, subjectID := range members {
				if _, err := transaction.Exec(ctx, `
					INSERT INTO audit_team_members (
						assignment_id, subject_id, member_role, revision, created_at
					) VALUES ($1, $2, 'INSPECTOR', 1, $3)
					ON CONFLICT (assignment_id, subject_id) DO UPDATE
					SET member_role = 'INSPECTOR', removed_at = NULL,
					    revision = audit_team_members.revision + 1
				`, command.AssignmentID, subjectID, now); err != nil {
					return commandResult[Assignment]{}, err
				}
			}
			updated, err := updateAssignmentStatus(ctx, transaction, current,
				StatusTeamAssigned, now)
			if err != nil {
				return commandResult[Assignment]{}, err
			}
			updated.MemberSubjectIDs = append([]string{updated.LeadSubjectID}, members...)
			return commandResult[Assignment]{
				Response: updated, OrganizationID: updated.OrganizationID,
				Action: "assignment.team_assigned", EntityType: "audit_assignment",
				EntityID: updated.ID, EntityVersion: updated.Revision,
				BeforeStatus: string(current.Status), AfterStatus: string(updated.Status),
			}, nil
		})
}

type AssignQuestionsCommand struct {
	OperationID         string
	IdempotencyKey      string
	AssignmentID        string
	ExpectedRevision    int64
	QuestionAssignments []QuestionAssignment
}

func (service *Service) AssignQuestions(
	ctx context.Context,
	actor identity.Principal,
	command AssignQuestionsCommand,
) (Assignment, error) {
	if !actor.HasRole(identity.RoleLeadInspector) {
		return Assignment{}, ErrForbidden
	}
	questionAssignments, err := normalizedQuestionAssignments(command.QuestionAssignments)
	if err != nil || blank(command.OperationID, command.IdempotencyKey, command.AssignmentID) ||
		command.ExpectedRevision <= 0 || len(questionAssignments) == 0 {
		return Assignment{}, ErrInvalid
	}
	semantic := command
	semantic.QuestionAssignments = questionAssignments
	return executeCommand(ctx, service, actor, "assign_questions", command.OperationID,
		command.IdempotencyKey, command.AssignmentID, semantic,
		func(ctx context.Context, transaction pgx.Tx) (commandResult[Assignment], error) {
			current, err := getAssignmentForUpdate(ctx, transaction, command.AssignmentID)
			if err != nil {
				return commandResult[Assignment]{}, err
			}
			if !CanConfigureTeam(actor, current.LeadSubjectID) {
				return commandResult[Assignment]{}, ErrForbidden
			}
			if current.Revision != command.ExpectedRevision || current.Status != StatusTeamAssigned {
				return commandResult[Assignment]{}, ErrConflict
			}
			allowedQuestions, err := templateQuestionIDs(ctx, transaction, current.InspectionID)
			if err != nil {
				return commandResult[Assignment]{}, err
			}
			for _, assignment := range questionAssignments {
				if !allowedQuestions[assignment.QuestionID] {
					return commandResult[Assignment]{}, ErrInvalid
				}
				var exists bool
				if err := transaction.QueryRow(ctx, `
					SELECT EXISTS (
						SELECT 1 FROM audit_team_members
						WHERE assignment_id = $1 AND subject_id = $2 AND removed_at IS NULL
					)
				`, command.AssignmentID, assignment.SubjectID).Scan(&exists); err != nil {
					return commandResult[Assignment]{}, err
				}
				if !exists {
					return commandResult[Assignment]{}, ErrInvalid
				}
			}
			if _, err := transaction.Exec(ctx,
				"DELETE FROM audit_question_assignments WHERE assignment_id = $1",
				command.AssignmentID,
			); err != nil {
				return commandResult[Assignment]{}, err
			}
			now := service.clock().UTC()
			for _, assignment := range questionAssignments {
				if _, err := transaction.Exec(ctx, `
					INSERT INTO audit_question_assignments (
						assignment_id, question_id, subject_id, revision, created_at
					) VALUES ($1, $2, $3, 1, $4)
				`, command.AssignmentID, assignment.QuestionID, assignment.SubjectID, now); err != nil {
					return commandResult[Assignment]{}, err
				}
			}
			updated, err := updateAssignmentStatus(ctx, transaction, current,
				StatusQuestionsAssigned, now)
			if err != nil {
				return commandResult[Assignment]{}, err
			}
			updated.QuestionAssignments = questionAssignments
			updated.MemberSubjectIDs, err = listMemberIDs(ctx, transaction, command.AssignmentID)
			if err != nil {
				return commandResult[Assignment]{}, err
			}
			return commandResult[Assignment]{
				Response: updated, OrganizationID: updated.OrganizationID,
				Action: "assignment.questions_assigned", EntityType: "audit_assignment",
				EntityID: updated.ID, EntityVersion: updated.Revision,
				BeforeStatus: string(current.Status), AfterStatus: string(updated.Status),
			}, nil
		})
}

func (service *Service) ListWorkload(
	ctx context.Context,
	actor identity.Principal,
) (map[string]int64, error) {
	if !CanViewWorkload(actor) {
		return nil, ErrForbidden
	}
	rows, err := service.pool.Query(ctx, `
		SELECT member.subject_id, COUNT(DISTINCT member.assignment_id)
		FROM audit_team_members member
		JOIN audit_assignments assignment ON assignment.id = member.assignment_id
		WHERE member.removed_at IS NULL
		  AND member.member_role = 'INSPECTOR'
		  AND assignment.tombstoned_at IS NULL
		  AND assignment.status IN (
		      'LEAD_ASSIGNED', 'TEAM_ASSIGNED', 'QUESTIONS_ASSIGNED',
		      'AWAITING_AUDITEE_CONFIRMATION', 'CONFIRMED', 'SCHEDULED', 'READY'
		  )
		GROUP BY member.subject_id
		ORDER BY member.subject_id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	output := map[string]int64{}
	for rows.Next() {
		var subjectID string
		var count int64
		if err := rows.Scan(&subjectID, &count); err != nil {
			return nil, err
		}
		output[subjectID] = count
	}
	return output, rows.Err()
}

func (service *Service) ListAuditeeCoordination(
	ctx context.Context,
	actor identity.Principal,
) ([]AuditeeCoordination, error) {
	if !CanViewAuditeeCoordination(actor) {
		return nil, ErrForbidden
	}
	rows, err := service.pool.Query(ctx, `
		SELECT inspection.id, inspection.organization_id, organization.legal_name,
		       inspection.title, draft.values->>'inspectionCategory',
		       assignment.scheduled_start_date, assignment.status,
		       NULLIF(draft.values->>'alternativeDate', ''), assignment.revision
		FROM audit_assignments assignment
		JOIN inspections inspection ON inspection.id = assignment.inspection_id
		JOIN organizations organization ON organization.id = inspection.organization_id
		JOIN planning_intake_drafts draft
		  ON draft.values->>'preparedAuditId' = inspection.id
		WHERE inspection.organization_id = $1
		  AND draft.values->>'noticePolicy' = 'ADVANCE'
		  AND assignment.status IN (
		      'AWAITING_AUDITEE_CONFIRMATION', 'CONFIRMED', 'ALTERNATIVE_PROPOSED'
		  )
		  AND assignment.tombstoned_at IS NULL
		  AND inspection.tombstoned_at IS NULL
		ORDER BY assignment.scheduled_start_date, inspection.id
	`, actor.OrganizationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	output := []AuditeeCoordination{}
	for rows.Next() {
		var item AuditeeCoordination
		var scheduledDate time.Time
		if err := rows.Scan(
			&item.InspectionID, &item.OrganizationID, &item.OrganizationName,
			&item.Title, &item.InspectionCategory, &scheduledDate, &item.Status,
			&item.AlternativeDate, &item.Revision,
		); err != nil {
			return nil, err
		}
		item.ScheduledStartDate = scheduledDate.Format("2006-01-02")
		item.NextAction = coordinationNextAction(item.Status)
		output = append(output, item)
	}
	return output, rows.Err()
}

type CoordinationDecision string

const (
	CoordinationConfirm            CoordinationDecision = "CONFIRM"
	CoordinationProposeAlternative CoordinationDecision = "PROPOSE_ALTERNATIVE"
)

type RespondCoordinationCommand struct {
	OperationID      string
	IdempotencyKey   string
	InspectionID     string
	OrganizationID   string
	ExpectedRevision int64
	Decision         CoordinationDecision
	AlternativeDate  *string
}

func (service *Service) RespondAuditeeCoordination(
	ctx context.Context,
	actor identity.Principal,
	command RespondCoordinationCommand,
) (AuditeeCoordination, error) {
	if !CanViewAuditeeCoordination(actor) || actor.OrganizationID != command.OrganizationID {
		return AuditeeCoordination{}, ErrForbidden
	}
	if blank(command.OperationID, command.IdempotencyKey, command.InspectionID,
		command.OrganizationID) || command.ExpectedRevision <= 0 {
		return AuditeeCoordination{}, ErrInvalid
	}
	switch command.Decision {
	case CoordinationConfirm:
		if command.AlternativeDate != nil {
			return AuditeeCoordination{}, ErrInvalid
		}
	case CoordinationProposeAlternative:
		if command.AlternativeDate == nil {
			return AuditeeCoordination{}, ErrInvalid
		}
		if _, err := time.Parse("2006-01-02", *command.AlternativeDate); err != nil {
			return AuditeeCoordination{}, ErrInvalid
		}
	default:
		return AuditeeCoordination{}, ErrInvalid
	}
	return executeCommand(ctx, service, actor, "respond_auditee_coordination",
		command.OperationID, command.IdempotencyKey, command.InspectionID, command,
		func(ctx context.Context, transaction pgx.Tx) (commandResult[AuditeeCoordination], error) {
			var assignmentID, organizationName, title, category, status string
			var scheduledDate time.Time
			var revision int64
			if err := transaction.QueryRow(ctx, `
				SELECT assignment.id, organization.legal_name, inspection.title,
				       draft.values->>'inspectionCategory', assignment.scheduled_start_date,
				       assignment.status, assignment.revision
				FROM audit_assignments assignment
				JOIN inspections inspection ON inspection.id = assignment.inspection_id
				JOIN organizations organization ON organization.id = inspection.organization_id
				JOIN planning_intake_drafts draft
				  ON draft.values->>'preparedAuditId' = inspection.id
				WHERE inspection.id = $1
				  AND inspection.organization_id = $2
				  AND draft.values->>'noticePolicy' = 'ADVANCE'
				FOR UPDATE OF assignment, draft
			`, command.InspectionID, command.OrganizationID).Scan(
				&assignmentID, &organizationName, &title, &category, &scheduledDate,
				&status, &revision,
			); err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					return commandResult[AuditeeCoordination]{}, ErrNotFound
				}
				return commandResult[AuditeeCoordination]{}, err
			}
			if revision != command.ExpectedRevision ||
				(Status(status) != StatusAwaitingAuditeeConfirmation &&
					Status(status) != StatusAlternativeProposed) {
				return commandResult[AuditeeCoordination]{}, ErrConflict
			}
			nextStatus := StatusConfirmed
			if command.Decision == CoordinationProposeAlternative {
				nextStatus = StatusAlternativeProposed
			}
			now := service.clock().UTC()
			if _, err := transaction.Exec(ctx, `
				UPDATE audit_assignments
				SET status = $2, revision = revision + 1, updated_at = $3
				WHERE id = $1 AND revision = $4
			`, assignmentID, string(nextStatus), now, revision); err != nil {
				return commandResult[AuditeeCoordination]{}, err
			}
			if command.AlternativeDate != nil {
				if _, err := transaction.Exec(ctx, `
					UPDATE planning_intake_drafts
					SET values = jsonb_set(values, '{alternativeDate}', to_jsonb($2::text), true),
					    updated_at = $3
					WHERE values->>'preparedAuditId' = $1
				`, command.InspectionID, *command.AlternativeDate, now); err != nil {
					return commandResult[AuditeeCoordination]{}, err
				}
			}
			output := AuditeeCoordination{
				InspectionID: command.InspectionID, OrganizationID: command.OrganizationID,
				OrganizationName: organizationName, Title: title, InspectionCategory: category,
				ScheduledStartDate: scheduledDate.Format("2006-01-02"), Status: nextStatus,
				AlternativeDate: command.AlternativeDate, NextAction: coordinationNextAction(nextStatus),
				Revision: revision + 1,
			}
			return commandResult[AuditeeCoordination]{
				Response: output, OrganizationID: command.OrganizationID,
				Action: "auditee.coordination_responded", EntityType: "audit_assignment",
				EntityID: assignmentID, EntityVersion: output.Revision,
				BeforeStatus: status, AfterStatus: string(nextStatus),
			}, nil
		})
}

type commandResult[T any] struct {
	Response       T
	OrganizationID string
	Action         string
	EntityType     string
	EntityID       string
	EntityVersion  int64
	BeforeStatus   string
	AfterStatus    string
}

func executeCommand[T any](
	ctx context.Context,
	service *Service,
	actor identity.Principal,
	kind, operationID, idempotencyKey, entityID string,
	semantic any,
	handler func(context.Context, pgx.Tx) (commandResult[T], error),
) (T, error) {
	var zero T
	semanticHash, err := idempotency.SemanticHash(semantic)
	if err != nil {
		return zero, err
	}
	scope := actor.SubjectID + ":" + kind
	var output T
	err = database.WithinTransaction(ctx, service.pool, func(ctx context.Context, transaction pgx.Tx) error {
		if _, err := transaction.Exec(ctx,
			"SELECT pg_advisory_xact_lock(hashtextextended($1, 0))",
			scope+":idempotency:"+idempotencyKey,
		); err != nil {
			return err
		}
		if _, err := transaction.Exec(ctx,
			"SELECT pg_advisory_xact_lock(hashtextextended($1, 0))",
			scope+":operation:"+operationID,
		); err != nil {
			return err
		}
		var storedHash string
		var responseBody []byte
		err := transaction.QueryRow(ctx, `
			SELECT semantic_hash, response_body
			FROM idempotency_responses
			WHERE scope = $1 AND operation_id = $2
		`, scope, operationID).Scan(&storedHash, &responseBody)
		if err == nil {
			if storedHash != semanticHash {
				return idempotency.ErrOperationIDReuse
			}
			return json.Unmarshal(responseBody, &output)
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		var reused bool
		if err := transaction.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM outbox_messages WHERE idempotency_key = $1
			)
		`, commandIdempotencyKey(scope, idempotencyKey)).Scan(&reused); err != nil {
			return err
		}
		if reused {
			return idempotency.ErrOperationIDReuse
		}
		result, err := handler(ctx, transaction)
		if err != nil {
			return err
		}
		output = result.Response
		responseBody, err = json.Marshal(output)
		if err != nil {
			return err
		}
		now := service.clock().UTC()
		role := ""
		if len(actor.Roles) > 0 {
			role = string(actor.Roles[0])
		}
		auditID := service.idGenerator("audit-assignment")
		outboxID := service.idGenerator("outbox-assignment")
		if _, err := transaction.Exec(ctx, `
			INSERT INTO audit_events (
				event_id, occurred_at, actor_subject_id, actor_role, organization_id,
				action, entity_type, entity_id, entity_version, before_status,
				after_status, operation_id, correlation_id, request_id, details
			) VALUES (
				$1, $2, $3, $4, $5, $6, $7, $8, $9, NULLIF($10, ''),
				NULLIF($11, ''), $12, $12, $12, '{}'::jsonb
			)
		`, auditID, now, actor.SubjectID, role, result.OrganizationID,
			result.Action, result.EntityType, result.EntityID, result.EntityVersion,
			result.BeforeStatus, result.AfterStatus, operationID); err != nil {
			return err
		}
		var changeID int64
		if err := transaction.QueryRow(ctx, `
			INSERT INTO authorized_sync_changes (
				subject_id, organization_id, kind, entity_id, entity_revision,
				payload, changed_at, operation_id, correlation_id
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $8)
			RETURNING sequence_id
		`, actor.SubjectID, result.OrganizationID, result.EntityType, result.EntityID,
			result.EntityVersion, responseBody, now, operationID).Scan(&changeID); err != nil {
			return err
		}
		if _, err := transaction.Exec(ctx, `
			INSERT INTO outbox_messages (
				id, topic, aggregate_type, aggregate_id, payload, available_at,
				idempotency_key, operation_id, correlation_id
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $8)
		`, outboxID, result.Action, result.EntityType, entityID, responseBody, now,
			commandIdempotencyKey(scope, idempotencyKey), operationID); err != nil {
			return err
		}
		if _, err := transaction.Exec(ctx, `
			INSERT INTO idempotency_responses (
				scope, operation_id, semantic_hash, response_status,
				response_headers, response_body, created_at
			) VALUES ($1, $2, $3, 200, '{}'::jsonb, $4, $5)
		`, scope, operationID, semanticHash, responseBody, now); err != nil {
			return err
		}
		if _, err := transaction.Exec(ctx, `
			INSERT INTO command_transaction_links (
				operation_id, idempotency_scope, audit_event_id,
				change_sequence_id, outbox_message_id, created_at
			) VALUES ($1, $2, $3, $4, $5, $6)
		`, operationID, scope, auditID, changeID, outboxID, now); err != nil {
			return err
		}
		return nil
	})
	return output, err
}

func getAssignmentForUpdate(
	ctx context.Context,
	transaction pgx.Tx,
	assignmentID string,
) (Assignment, error) {
	var output Assignment
	var start, end time.Time
	err := transaction.QueryRow(ctx, `
		SELECT id, inspection_id, organization_id, lead_subject_id, status,
		       scheduled_start_date, scheduled_end_date, revision
		FROM audit_assignments
		WHERE id = $1 AND tombstoned_at IS NULL
		FOR UPDATE
	`, assignmentID).Scan(
		&output.ID, &output.InspectionID, &output.OrganizationID, &output.LeadSubjectID,
		&output.Status, &start, &end, &output.Revision,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Assignment{}, ErrNotFound
	}
	if err != nil {
		return Assignment{}, err
	}
	output.ScheduledStartDate = start.Format("2006-01-02")
	output.ScheduledEndDate = end.Format("2006-01-02")
	return output, nil
}

func updateAssignmentStatus(
	ctx context.Context,
	transaction pgx.Tx,
	current Assignment,
	status Status,
	now time.Time,
) (Assignment, error) {
	var start, end time.Time
	var output Assignment
	err := transaction.QueryRow(ctx, `
		UPDATE audit_assignments
		SET status = $2, revision = revision + 1, updated_at = $3
		WHERE id = $1 AND revision = $4 AND tombstoned_at IS NULL
		RETURNING id, inspection_id, organization_id, lead_subject_id, status,
		          scheduled_start_date, scheduled_end_date, revision
	`, current.ID, string(status), now, current.Revision).Scan(
		&output.ID, &output.InspectionID, &output.OrganizationID, &output.LeadSubjectID,
		&output.Status, &start, &end, &output.Revision,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Assignment{}, ErrConflict
	}
	if err != nil {
		return Assignment{}, err
	}
	output.ScheduledStartDate = start.Format("2006-01-02")
	output.ScheduledEndDate = end.Format("2006-01-02")
	return output, nil
}

func requireActiveRole(
	ctx context.Context,
	transaction pgx.Tx,
	subjectID string,
	role identity.Role,
) error {
	var exists bool
	if err := transaction.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM identity_references identity
			JOIN session_references session ON session.subject_id = identity.subject_id
			WHERE identity.subject_id = $1
			  AND identity.tombstoned_at IS NULL
			  AND session.revoked_at IS NULL
			  AND $2 = ANY(session.roles)
		)
	`, subjectID, string(role)).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return ErrInvalid
	}
	return nil
}

func templateQuestionIDs(
	ctx context.Context,
	transaction pgx.Tx,
	inspectionID string,
) (map[string]bool, error) {
	var snapshot []byte
	if err := transaction.QueryRow(ctx, `
		SELECT template.snapshot
		FROM planning_intake_drafts draft
		JOIN checklist_template_versions template
		  ON template.id = draft.values->>'templateVersionId'
		WHERE draft.values->>'preparedAuditId' = $1
		  AND draft.tombstoned_at IS NULL
	`, inspectionID).Scan(&snapshot); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	var decoded struct {
		Questions []struct {
			ID string `json:"id"`
		} `json:"questions"`
	}
	if err := json.Unmarshal(snapshot, &decoded); err != nil {
		return nil, ErrInvalid
	}
	output := make(map[string]bool, len(decoded.Questions))
	for _, question := range decoded.Questions {
		output[question.ID] = true
	}
	return output, nil
}

func listMemberIDs(
	ctx context.Context,
	transaction pgx.Tx,
	assignmentID string,
) ([]string, error) {
	rows, err := transaction.Query(ctx, `
		SELECT subject_id
		FROM audit_team_members
		WHERE assignment_id = $1 AND removed_at IS NULL
		ORDER BY CASE member_role WHEN 'LEAD_INSPECTOR' THEN 0 ELSE 1 END, subject_id
	`, assignmentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	output := []string{}
	for rows.Next() {
		var subjectID string
		if err := rows.Scan(&subjectID); err != nil {
			return nil, err
		}
		output = append(output, subjectID)
	}
	return output, rows.Err()
}

func normalizedIDs(values []string) ([]string, error) {
	seen := map[string]bool{}
	output := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			return nil, ErrInvalid
		}
		seen[value] = true
		output = append(output, value)
	}
	sort.Strings(output)
	return output, nil
}

func normalizedQuestionAssignments(
	values []QuestionAssignment,
) ([]QuestionAssignment, error) {
	seen := map[string]bool{}
	output := make([]QuestionAssignment, 0, len(values))
	for _, value := range values {
		value.QuestionID = strings.TrimSpace(value.QuestionID)
		value.SubjectID = strings.TrimSpace(value.SubjectID)
		key := value.QuestionID + "\x00" + value.SubjectID
		if value.QuestionID == "" || value.SubjectID == "" || seen[key] {
			return nil, ErrInvalid
		}
		seen[key] = true
		output = append(output, value)
	}
	sort.Slice(output, func(left, right int) bool {
		if output[left].QuestionID == output[right].QuestionID {
			return output[left].SubjectID < output[right].SubjectID
		}
		return output[left].QuestionID < output[right].QuestionID
	})
	return output, nil
}

func parseSchedule(startValue, endValue string) (time.Time, time.Time, error) {
	start, err := time.Parse("2006-01-02", startValue)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	end, err := time.Parse("2006-01-02", endValue)
	if err != nil || end.Before(start) {
		return time.Time{}, time.Time{}, ErrInvalid
	}
	return start, end, nil
}

func coordinationNextAction(status Status) string {
	switch status {
	case StatusAwaitingAuditeeConfirmation:
		return "Confirm proposed date or provide an alternative date"
	case StatusAlternativeProposed:
		return "CAA to accept or return the proposed alternative date"
	case StatusConfirmed:
		return "CAA prepares the confirmed inspection for execution"
	default:
		return ""
	}
}

func commandIdempotencyKey(scope, key string) string {
	return "command:" + scope + ":idempotency:" + key
}

func blank(values ...string) bool {
	for _, value := range values {
		if strings.TrimSpace(value) == "" {
			return true
		}
	}
	return false
}

func randomID(prefix string) string {
	var bytes [12]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		panic(fmt.Sprintf("generate assignment identifier: %v", err))
	}
	return prefix + "-" + hex.EncodeToString(bytes[:])
}
