package assignments

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
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
	OperationID    string
	IdempotencyKey string
	PlanningItemID string
	// InspectionID is retained only for source compatibility with the paused
	// donor tests. It is never read or persisted; canonical preparation owns no
	// inspection identity until materialization.
	InspectionID             string
	ExpectedPlanningRevision int64
}

// ConfirmPreparationCommand is the explicit Department Manager gate between
// Lead/team assignment and canonical materialization. It pins the released
// scope and exact per-question coverage in an immutable preparation snapshot;
// it does not open execution or notify the Auditee.
type ConfirmPreparationCommand struct {
	OperationID                string
	IdempotencyKey             string
	AssignmentID               string
	ExpectedAssignmentRevision int64
}

func (service *Service) ConfirmPreparation(
	ctx context.Context,
	actor identity.Principal,
	command ConfirmPreparationCommand,
) (Preparation, error) {
	if !CanPrepare(actor) {
		return Preparation{}, ErrForbidden
	}
	if blank(command.OperationID, command.IdempotencyKey, command.AssignmentID) || command.ExpectedAssignmentRevision <= 0 {
		return Preparation{}, ErrInvalid
	}
	return executeCommand(ctx, service, actor, "confirm_preparation", command.OperationID,
		command.IdempotencyKey, command.AssignmentID, command,
		func(ctx context.Context, transaction pgx.Tx) (commandResult[Preparation], error) {
			current, err := getAssignmentForUpdate(ctx, transaction, command.AssignmentID)
			if err != nil {
				return commandResult[Preparation]{}, err
			}
			if current.Revision != command.ExpectedAssignmentRevision || current.Status != StatusQuestionsAssigned {
				return commandResult[Preparation]{}, ErrConflict
			}
			if err := requireCurrentDepartmentScopeAuthority(ctx, transaction, actor, current.PlanningItemID, ""); err != nil {
				return commandResult[Preparation]{}, err
			}
			var inspectionID, planningItemID, snapshotID, selectionDigest string
			var selectedCount int
			if err := transaction.QueryRow(ctx, `
				SELECT COALESCE(assignment.inspection_id, ''), COALESCE(draft.submitted_planning_item_id, ''),
				       snapshot.id, snapshot.selection_digest, snapshot.selected_question_count
				FROM audit_assignments assignment
				JOIN planning_intake_drafts draft
				  ON draft.submitted_planning_item_id = assignment.planning_item_id
				JOIN canonical_audit_scope_drafts scope
				  ON scope.planning_intake_draft_id = draft.id
				JOIN canonical_audit_scope_snapshots snapshot
				  ON snapshot.id = assignment.released_scope_snapshot_id
				 AND snapshot.scope_draft_id = scope.id
				 AND snapshot.stage = 'RELEASED'
				WHERE assignment.id = $1
				  AND assignment.released_scope_snapshot_id IS NOT NULL
				  AND draft.tombstoned_at IS NULL
				FOR UPDATE OF assignment, draft
			`, command.AssignmentID).Scan(&inspectionID, &planningItemID, &snapshotID, &selectionDigest, &selectedCount); err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					return commandResult[Preparation]{}, ErrConflict
				}
				return commandResult[Preparation]{}, err
			}
			if planningItemID == "" || snapshotID == "" || selectionDigest == "" || selectedCount <= 0 {
				return commandResult[Preparation]{}, ErrConflict
			}
			var existingConfirmed bool
			if err := transaction.QueryRow(ctx, `
				SELECT EXISTS (
					SELECT 1
					FROM canonical_audit_preparation_snapshots
					WHERE assignment_id = $1
					  AND released_scope_snapshot_id = $2
					  AND status = 'CONFIRMED'
				)
			`, command.AssignmentID, snapshotID).Scan(&existingConfirmed); err != nil {
				return commandResult[Preparation]{}, err
			}
			if existingConfirmed {
				return commandResult[Preparation]{}, fmt.Errorf("%w: preparation is already confirmed for this released scope", ErrConflict)
			}
			type coverageEntry struct {
				QuestionID string   `json:"questionVersionId"`
				SubjectIDs []string `json:"subjectIds"`
			}
			coverageByQuestion := map[string][]string{}
			rows, err := transaction.Query(ctx, `
				SELECT question_id, subject_id
				FROM audit_question_assignments
				WHERE assignment_id = $1
				ORDER BY question_id, subject_id
			`, command.AssignmentID)
			if err != nil {
				return commandResult[Preparation]{}, err
			}
			for rows.Next() {
				var questionID, subjectID string
				if err := rows.Scan(&questionID, &subjectID); err != nil {
					rows.Close()
					return commandResult[Preparation]{}, err
				}
				coverageByQuestion[questionID] = append(coverageByQuestion[questionID], subjectID)
			}
			if err := rows.Err(); err != nil {
				rows.Close()
				return commandResult[Preparation]{}, err
			}
			rows.Close()
			selectedRows, err := transaction.Query(ctx, `
				SELECT question_version_id
				FROM canonical_audit_scope_snapshot_questions
				WHERE snapshot_id = $1
				ORDER BY position
			`, snapshotID)
			if err != nil {
				return commandResult[Preparation]{}, err
			}
			selectedIDs := make([]string, 0, selectedCount)
			for selectedRows.Next() {
				var questionID string
				if err := selectedRows.Scan(&questionID); err != nil {
					selectedRows.Close()
					return commandResult[Preparation]{}, err
				}
				selectedIDs = append(selectedIDs, questionID)
				if len(coverageByQuestion[questionID]) == 0 {
					selectedRows.Close()
					return commandResult[Preparation]{}, fmt.Errorf("%w: every released question needs team coverage", ErrConflict)
				}
			}
			if err := selectedRows.Err(); err != nil {
				selectedRows.Close()
				return commandResult[Preparation]{}, err
			}
			selectedRows.Close()
			if len(selectedIDs) != selectedCount || len(coverageByQuestion) != selectedCount {
				return commandResult[Preparation]{}, fmt.Errorf("%w: preparation coverage does not match released scope", ErrConflict)
			}
			coverageEntries := make([]coverageEntry, 0, len(selectedIDs))
			for _, questionID := range selectedIDs {
				subjects := append([]string(nil), coverageByQuestion[questionID]...)
				sort.Strings(subjects)
				coverageEntries = append(coverageEntries, coverageEntry{QuestionID: questionID, SubjectIDs: subjects})
			}
			payload, err := json.Marshal(struct {
				AssignmentID string          `json:"assignmentId"`
				SnapshotID   string          `json:"releasedScopeSnapshotId"`
				Digest       string          `json:"selectionDigest"`
				LeadSubject  string          `json:"leadSubjectId"`
				Coverage     []coverageEntry `json:"coverage"`
			}{command.AssignmentID, snapshotID, selectionDigest, current.LeadSubjectID, coverageEntries})
			if err != nil {
				return commandResult[Preparation]{}, err
			}
			digestBytes := sha256.Sum256(payload)
			preparationDigest := "sha256:" + hex.EncodeToString(digestBytes[:])
			var revision int64
			if err := transaction.QueryRow(ctx, `
				SELECT COALESCE(MAX(revision), 0) + 1
				FROM canonical_audit_preparation_snapshots
				WHERE released_scope_snapshot_id = $1
			`, snapshotID).Scan(&revision); err != nil {
				return commandResult[Preparation]{}, err
			}
			preparationID := fmt.Sprintf("preparation:%s:%d", command.AssignmentID, revision)
			now := service.clock().UTC()
			if _, err := transaction.Exec(ctx, `
					INSERT INTO canonical_audit_preparation_snapshots (
						id, assignment_id, released_scope_snapshot_id, lead_subject_id, revision, status,
						preparation_digest, confirmed_by_subject_id, confirmed_at, snapshot, created_at
					) VALUES ($1, $2, $3, $4, $5, 'CONFIRMED', $6, $7, $8, $9, $8)
				`, preparationID, command.AssignmentID, snapshotID, current.LeadSubjectID, revision,
				preparationDigest, actor.SubjectID, now, payload); err != nil {
				return commandResult[Preparation]{}, err
			}
			for position, item := range coverageEntries {
				for _, subjectID := range item.SubjectIDs {
					if _, err := transaction.Exec(ctx, `
						INSERT INTO canonical_audit_preparation_questions (
							preparation_id, released_scope_snapshot_id, question_version_id, subject_id, position
						) VALUES ($1, $2, $3, $4, $5)
					`, preparationID, snapshotID, item.QuestionID, subjectID, position); err != nil {
						return commandResult[Preparation]{}, err
					}
				}
			}
			// Confirmation is an explicit durable gate.  Keep the assignment in
			// QUESTIONS_ASSIGNED until materialization, but advance its aggregate
			// revision so the following materialize command is pinned to the exact
			// confirmed preparation rather than the pre-confirmation projection.
			updated, err := transaction.Exec(ctx, `
				UPDATE audit_assignments
				SET revision = revision + 1, updated_at = $2
				WHERE id = $1 AND status = 'QUESTIONS_ASSIGNED' AND revision = $3
			`, command.AssignmentID, now, current.Revision)
			if err != nil {
				return commandResult[Preparation]{}, err
			}
			if updated.RowsAffected() != 1 {
				return commandResult[Preparation]{}, ErrConflict
			}
			confirmedRevision := current.Revision + 1
			return commandResult[Preparation]{
				Response: Preparation{
					AssignmentID: command.AssignmentID, PlanningItemID: planningItemID, InspectionID: inspectionID,
					OrganizationID: current.OrganizationID, Status: current.Status,
					Revision: confirmedRevision, PreparationID: preparationID,
					PreparationDigest: preparationDigest, SelectedQuestionCount: selectedCount,
					ConfirmedAt: now,
				},
				OrganizationID: current.OrganizationID, Action: "planning.preparation_confirmed",
				EntityType: "audit_assignment", EntityID: command.AssignmentID,
				EntityVersion: confirmedRevision, BeforeStatus: string(current.Status),
				AfterStatus: string(current.Status),
			}, nil
		})
}

func (service *Service) Prepare(
	ctx context.Context,
	actor identity.Principal,
	command PrepareCommand,
) (Preparation, error) {
	if !CanPrepare(actor) {
		return Preparation{}, ErrForbidden
	}
	if blank(command.OperationID, command.IdempotencyKey, command.PlanningItemID) ||
		command.ExpectedPlanningRevision <= 0 {
		return Preparation{}, ErrInvalid
	}
	return executeCommand(ctx, service, actor, "prepare_audit", command.OperationID,
		command.IdempotencyKey, "assignment:"+command.PlanningItemID, command,
		func(ctx context.Context, transaction pgx.Tx) (commandResult[Preparation], error) {
			var organizationID, status string
			var scheduledDate time.Time
			var revision int64
			if err := transaction.QueryRow(ctx, `
				SELECT organization_id, status, scheduled_date, revision
				FROM surveillance_plan_items
				WHERE id = $1 AND tombstoned_at IS NULL
				FOR UPDATE
			`, command.PlanningItemID).Scan(
				&organizationID, &status, &scheduledDate, &revision,
			); err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					return commandResult[Preparation]{}, ErrNotFound
				}
				return commandResult[Preparation]{}, err
			}
			if revision != command.ExpectedPlanningRevision || status != "RELEASED" {
				return commandResult[Preparation]{}, ErrConflict
			}
			if err := requireCurrentDepartmentScopeAuthority(ctx, transaction, actor, command.PlanningItemID, ""); err != nil {
				return commandResult[Preparation]{}, err
			}
			var draftID string
			if err := transaction.QueryRow(ctx, `
				SELECT id
				FROM planning_intake_drafts
				WHERE submitted_planning_item_id = $1 AND tombstoned_at IS NULL
				ORDER BY revision DESC, updated_at DESC, id DESC
				LIMIT 1
				FOR UPDATE
			`, command.PlanningItemID).Scan(&draftID); err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					return commandResult[Preparation]{}, ErrNotFound
				}
				return commandResult[Preparation]{}, err
			}
			assignmentID := "assignment:" + command.PlanningItemID
			var releasedScopeSnapshotID *string
			if err := transaction.QueryRow(ctx, `
				SELECT snapshot.id
				FROM canonical_audit_scope_snapshots snapshot
				JOIN canonical_audit_scope_drafts scope ON scope.id = snapshot.scope_draft_id
				JOIN planning_intake_drafts draft ON draft.id = scope.planning_intake_draft_id
				WHERE draft.submitted_planning_item_id = $1
				  AND draft.tombstoned_at IS NULL
				  AND snapshot.stage = 'RELEASED'
				ORDER BY snapshot.revision DESC, snapshot.id DESC
				LIMIT 1
			`, command.PlanningItemID).Scan(&releasedScopeSnapshotID); err != nil && !errors.Is(err, pgx.ErrNoRows) {
				return commandResult[Preparation]{}, err
			}
			now := service.clock().UTC()
			if _, err := transaction.Exec(ctx, `
				INSERT INTO audit_assignments (
					id, inspection_id, planning_item_id, organization_id, lead_subject_id, status,
					scheduled_start_date, scheduled_end_date, revision, created_at, updated_at,
					released_scope_snapshot_id
				) VALUES ($1, NULL, $2, $3, NULL, 'PREPARATION', $4, $4, 1, $5, $5, $6)
			`, assignmentID, command.PlanningItemID, organizationID, scheduledDate, now, releasedScopeSnapshotID); err != nil {
				return commandResult[Preparation]{}, err
			}
			output := Preparation{
				AssignmentID: assignmentID, PlanningItemID: command.PlanningItemID, InspectionID: "",
				OrganizationID: organizationID, Status: StatusPreparation, Revision: 1,
			}
			return commandResult[Preparation]{
				Response: output, OrganizationID: organizationID,
				Action: "planning.preparation_started", EntityType: "audit_assignment",
				EntityID: assignmentID, EntityVersion: 1,
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
	if blank(command.OperationID, command.IdempotencyKey, command.AssignmentID, command.LeadSubjectID) || command.ExpectedInspectionRevision <= 0 {
		return Assignment{}, ErrInvalid
	}
	return executeCommand(ctx, service, actor, "assign_lead", command.OperationID,
		command.IdempotencyKey, command.AssignmentID, command,
		func(ctx context.Context, transaction pgx.Tx) (commandResult[Assignment], error) {
			current, err := getAssignmentForUpdate(ctx, transaction, command.AssignmentID)
			if err != nil {
				return commandResult[Assignment]{}, err
			}
			if current.Revision != command.ExpectedInspectionRevision || current.Status != StatusPreparation || current.PlanningItemID == "" {
				return commandResult[Assignment]{}, ErrConflict
			}
			var releasedPlannedDate time.Time
			if err := transaction.QueryRow(ctx, `
				SELECT plan.scheduled_date
				FROM planning_intake_drafts draft
				JOIN surveillance_plan_items plan ON plan.id = draft.submitted_planning_item_id
				WHERE draft.submitted_planning_item_id = $1
				  AND draft.tombstoned_at IS NULL
				ORDER BY draft.revision DESC, draft.updated_at DESC, draft.id DESC
				LIMIT 1
				FOR UPDATE
			`, current.PlanningItemID).Scan(&releasedPlannedDate); err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					return commandResult[Assignment]{}, ErrNotFound
				}
				return commandResult[Assignment]{}, err
			}
			if err := requireCurrentDepartmentScopeAuthority(ctx, transaction, actor, current.PlanningItemID, ""); err != nil {
				return commandResult[Assignment]{}, err
			}
			if err := requireActiveRole(ctx, transaction, command.LeadSubjectID, identity.RoleLeadInspector); err != nil {
				return commandResult[Assignment]{}, err
			}
			if current.ReleasedScopeSnapshotID == "" {
				return commandResult[Assignment]{}, fmt.Errorf("%w: canonical released scope snapshot is required before Lead assignment", ErrConflict)
			}
			now := service.clock().UTC()
			updated, err := transaction.Exec(ctx, `
				UPDATE audit_assignments
				SET lead_subject_id = $2, status = 'LEAD_ASSIGNED',
				    scheduled_start_date = $3, scheduled_end_date = $3,
				    revision = revision + 1, updated_at = $4
				WHERE id = $1 AND status = 'PREPARATION' AND revision = $5
			`, command.AssignmentID, command.LeadSubjectID, releasedPlannedDate, now, current.Revision)
			if err != nil {
				return commandResult[Assignment]{}, err
			}
			if updated.RowsAffected() != 1 {
				return commandResult[Assignment]{}, ErrConflict
			}
			if _, err := transaction.Exec(ctx, `
				INSERT INTO audit_team_members (
					assignment_id, subject_id, member_role, revision, created_at
				) VALUES ($1, $2, 'LEAD_INSPECTOR', 1, $3)
			`, command.AssignmentID, command.LeadSubjectID, now); err != nil {
				return commandResult[Assignment]{}, err
			}
			output := Assignment{
				ID: command.AssignmentID, InspectionID: "", PlanningItemID: current.PlanningItemID,
				OrganizationID: current.OrganizationID, LeadSubjectID: command.LeadSubjectID,
				MemberSubjectIDs: []string{command.LeadSubjectID}, Status: StatusLeadAssigned,
				ScheduledStartDate: releasedPlannedDate.Format("2006-01-02"),
				ScheduledEndDate:   releasedPlannedDate.Format("2006-01-02"), Revision: current.Revision + 1,
			}
			return commandResult[Assignment]{
				Response: output, OrganizationID: current.OrganizationID,
				Action: "assignment.lead_assigned", EntityType: "audit_assignment",
				EntityID: command.AssignmentID, EntityVersion: output.Revision,
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
			if err := requireActiveRole(ctx, transaction, actor.SubjectID, identity.RoleLeadInspector); err != nil {
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
			if err := recordPreparationEdit(ctx, transaction, command.AssignmentID, updated.Revision, "TEAM", actor, map[string]any{
				"memberSubjectIds": members,
			}); err != nil {
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
			if err := requireActiveRole(ctx, transaction, actor.SubjectID, identity.RoleLeadInspector); err != nil {
				return commandResult[Assignment]{}, ErrForbidden
			}
			if current.Revision != command.ExpectedRevision || current.Status != StatusTeamAssigned {
				return commandResult[Assignment]{}, ErrConflict
			}
			allowedQuestions, err := templateQuestionIDs(ctx, transaction, current.InspectionID, current.PlanningItemID)
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
			if err := recordPreparationEdit(ctx, transaction, command.AssignmentID, updated.Revision, "QUESTION_COVERAGE", actor, map[string]any{
				"questionAssignments": questionAssignments,
			}); err != nil {
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

// recordPreparationEdit is the durable preview/confirm trail for mutable
// assignment projections. It is written in the same transaction as the
// status CAS, so a successful team/coverage command always has an immutable
// revision receipt and a failed command has none.
func recordPreparationEdit(
	ctx context.Context,
	transaction pgx.Tx,
	assignmentID string,
	assignmentRevision int64,
	editKind string,
	actor identity.Principal,
	snapshot any,
) error {
	body, err := json.Marshal(snapshot)
	if err != nil {
		return err
	}
	digest, err := idempotency.SemanticHash(snapshot)
	if err != nil {
		return err
	}
	_, err = transaction.Exec(ctx, `
		INSERT INTO canonical_audit_preparation_edit_events (
			id, assignment_id, assignment_revision, edit_kind, edit_digest,
			snapshot, actor_subject_id
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, fmt.Sprintf("preparation-edit:%s:%d:%s", assignmentID, assignmentRevision, strings.ToLower(editKind)), assignmentID, assignmentRevision, editKind, digest, body, actor.SubjectID)
	return err
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
		  OR draft.submitted_planning_item_id = assignment.planning_item_id
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

type ReviewCoordinationDecision string

const (
	ReviewCoordinationAccept ReviewCoordinationDecision = "ACCEPT_ALTERNATIVE"
	ReviewCoordinationReturn ReviewCoordinationDecision = "RETURN_TO_AUDITEE"
)

type ReviewCoordinationCommand struct {
	OperationID      string
	IdempotencyKey   string
	InspectionID     string
	OrganizationID   string
	ExpectedRevision int64
	Decision         ReviewCoordinationDecision
	Reason           string
}

// ReviewAuditeeCoordination is the distinct CAA command boundary for an
// announced Auditee alternative. The Auditee can propose; only the scoped
// Department Manager can accept or return that proposal.
func (service *Service) ReviewAuditeeCoordination(
	ctx context.Context,
	actor identity.Principal,
	command ReviewCoordinationCommand,
) (AuditeeCoordination, error) {
	if !actor.HasRole(identity.RoleDepartmentManager) || blank(command.OperationID, command.IdempotencyKey, command.InspectionID, command.OrganizationID, command.Reason) || command.ExpectedRevision <= 0 {
		return AuditeeCoordination{}, ErrForbidden
	}
	if command.Decision != ReviewCoordinationAccept && command.Decision != ReviewCoordinationReturn {
		return AuditeeCoordination{}, ErrInvalid
	}
	return executeCommand(ctx, service, actor, "review_auditee_coordination", command.OperationID, command.IdempotencyKey, command.InspectionID, command,
		func(ctx context.Context, transaction pgx.Tx) (commandResult[AuditeeCoordination], error) {
			var assignmentID, organizationName, title, category, status string
			var scheduledDate time.Time
			var revision int64
			var alternativeDate *string
			if err := transaction.QueryRow(ctx, `
				SELECT assignment.id, organization.legal_name, inspection.title,
				       draft.values->>'inspectionCategory', assignment.scheduled_start_date,
				       assignment.status, assignment.revision,
				       NULLIF(draft.values->>'alternativeDate', '')
				FROM audit_assignments assignment
				JOIN inspections inspection ON inspection.id = assignment.inspection_id
				JOIN organizations organization ON organization.id = inspection.organization_id
				JOIN planning_intake_drafts draft
				  ON draft.values->>'preparedAuditId' = inspection.id
				  OR draft.submitted_planning_item_id = assignment.planning_item_id
				WHERE inspection.id = $1 AND inspection.organization_id = $2
				  AND draft.values->>'noticePolicy' = 'ADVANCE'
				  AND assignment.status = 'ALTERNATIVE_PROPOSED'
				  AND assignment.tombstoned_at IS NULL
				FOR UPDATE OF assignment
			`, command.InspectionID, command.OrganizationID).Scan(&assignmentID, &organizationName, &title, &category, &scheduledDate, &status, &revision, &alternativeDate); err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					return commandResult[AuditeeCoordination]{}, ErrNotFound
				}
				return commandResult[AuditeeCoordination]{}, err
			}
			if revision != command.ExpectedRevision {
				return commandResult[AuditeeCoordination]{}, ErrConflict
			}
			if err := requireCurrentDepartmentScopeAuthority(ctx, transaction, actor, "", command.InspectionID); err != nil {
				return commandResult[AuditeeCoordination]{}, err
			}
			if alternativeDate == nil || strings.TrimSpace(*alternativeDate) == "" {
				return commandResult[AuditeeCoordination]{}, fmt.Errorf("%w: an Auditee alternative date is required", ErrConflict)
			}
			alternative, err := time.Parse("2006-01-02", *alternativeDate)
			if err != nil {
				return commandResult[AuditeeCoordination]{}, ErrConflict
			}
			now := service.clock().UTC()
			nextStatus := StatusAwaitingAuditeeConfirmation
			if command.Decision == ReviewCoordinationAccept {
				nextStatus = StatusConfirmed
				if _, err := transaction.Exec(ctx, `
					UPDATE audit_assignments
					SET status = $2, scheduled_start_date = $3,
					    scheduled_end_date = GREATEST(COALESCE(scheduled_end_date, $3), $3),
					    revision = revision + 1, updated_at = $4
					WHERE id = $1 AND revision = $5
				`, assignmentID, string(nextStatus), alternative, now, revision); err != nil {
					return commandResult[AuditeeCoordination]{}, err
				}
				if _, err := transaction.Exec(ctx, `
					UPDATE inspections
					SET status = 'SCHEDULED', revision = revision + 1, updated_at = $2
					WHERE id = $1 AND status = 'AWAITING_AUDITEE_CONFIRMATION'
				`, command.InspectionID, now); err != nil {
					return commandResult[AuditeeCoordination]{}, err
				}
				scheduledDate = alternative
			} else if _, err := transaction.Exec(ctx, `
				UPDATE audit_assignments SET status = $2, revision = revision + 1, updated_at = $3
				WHERE id = $1 AND revision = $4
			`, assignmentID, string(nextStatus), now, revision); err != nil {
				return commandResult[AuditeeCoordination]{}, err
			}
			output := AuditeeCoordination{
				InspectionID: command.InspectionID, OrganizationID: command.OrganizationID,
				OrganizationName: organizationName, Title: title, InspectionCategory: category,
				ScheduledStartDate: scheduledDate.Format("2006-01-02"), Status: nextStatus,
				AlternativeDate: alternativeDate, NextAction: coordinationNextAction(nextStatus), Revision: revision + 1,
			}
			return commandResult[AuditeeCoordination]{Response: output, OrganizationID: command.OrganizationID,
				Action: "caa.coordination_reviewed", EntityType: "audit_assignment", EntityID: assignmentID,
				EntityVersion: revision + 1, BeforeStatus: status, AfterStatus: string(nextStatus),
			}, nil
		})
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
		  OR draft.submitted_planning_item_id = assignment.planning_item_id
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
				(Status(status) != StatusAwaitingAuditeeConfirmation && Status(status) != StatusAlternativeProposed) {
				return commandResult[AuditeeCoordination]{}, ErrConflict
			}
			// An Auditee may confirm the announced date or propose an alternative,
			// but it cannot self-accept an alternative after the CAA must review it.
			if command.Decision == CoordinationConfirm && Status(status) != StatusAwaitingAuditeeConfirmation {
				return commandResult[AuditeeCoordination]{}, fmt.Errorf("%w: CAA must accept or return the proposed alternative date", ErrForbidden)
			}
			nextStatus := StatusConfirmed
			if command.Decision == CoordinationProposeAlternative {
				nextStatus = StatusAlternativeProposed
			}
			now := service.clock().UTC()
			assignmentUpdate, err := transaction.Exec(ctx, `
				UPDATE audit_assignments
				SET status = $2, revision = revision + 1, updated_at = $3
				WHERE id = $1 AND revision = $4
			`, assignmentID, string(nextStatus), now, revision)
			if err != nil {
				return commandResult[AuditeeCoordination]{}, err
			}
			if assignmentUpdate.RowsAffected() != 1 {
				return commandResult[AuditeeCoordination]{}, ErrConflict
			}
			if command.Decision == CoordinationConfirm {
				inspectionUpdate, err := transaction.Exec(ctx, `
						UPDATE inspections
						SET status = 'SCHEDULED', revision = revision + 1, updated_at = $2
						WHERE id = $1 AND status = 'AWAITING_AUDITEE_CONFIRMATION'
					`, command.InspectionID, now)
				if err != nil {
					return commandResult[AuditeeCoordination]{}, err
				}
				if inspectionUpdate.RowsAffected() != 1 {
					return commandResult[AuditeeCoordination]{}, ErrConflict
				}
			}
			if command.AlternativeDate != nil {
				if _, err := transaction.Exec(ctx, `
					UPDATE planning_intake_drafts
					SET values = jsonb_set(values, '{alternativeDate}', to_jsonb($2::text), true),
					    updated_at = $3
					WHERE values->>'preparedAuditId' = $1
					   OR submitted_planning_item_id = (SELECT planning_item_id FROM audit_assignments WHERE id = $4)
				`, command.InspectionID, *command.AlternativeDate, now, assignmentID); err != nil {
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
		SELECT id, COALESCE(inspection_id, ''), COALESCE(planning_item_id, ''), COALESCE(released_scope_snapshot_id, ''), organization_id,
		       COALESCE(lead_subject_id, ''), status,
		       COALESCE(scheduled_start_date, CURRENT_DATE), COALESCE(scheduled_end_date, CURRENT_DATE), revision
		FROM audit_assignments
		WHERE id = $1 AND tombstoned_at IS NULL
		FOR UPDATE
	`, assignmentID).Scan(
		&output.ID, &output.InspectionID, &output.PlanningItemID, &output.ReleasedScopeSnapshotID, &output.OrganizationID, &output.LeadSubjectID,
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
		RETURNING id, COALESCE(inspection_id, ''), COALESCE(planning_item_id, ''), COALESCE(released_scope_snapshot_id, ''), organization_id,
		          COALESCE(lead_subject_id, ''), status,
		          COALESCE(scheduled_start_date, CURRENT_DATE), COALESCE(scheduled_end_date, CURRENT_DATE), revision
	`, current.ID, string(status), now, current.Revision).Scan(
		&output.ID, &output.InspectionID, &output.PlanningItemID, &output.ReleasedScopeSnapshotID, &output.OrganizationID, &output.LeadSubjectID,
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
			  AND session.expires_at > now()
			  AND (session.absolute_expires_at IS NULL OR session.absolute_expires_at > now())
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

// requireCurrentDepartmentScopeAuthority binds canonical post-release
// preparation commands to the same current provider/department authority used
// by Planning scope selection. Legacy/template planning rows have no canonical
// scope row and remain readable only until the explicit donor-removal gate;
// canonical rows fail closed when their released scope is stale, revoked, or
// outside the actor's current Department Manager responsibility.
func requireCurrentDepartmentScopeAuthority(
	ctx context.Context,
	transaction pgx.Tx,
	actor identity.Principal,
	planningItemID string,
	inspectionID string,
) error {
	var canonicalScope, authorized bool
	if err := transaction.QueryRow(ctx, `
		WITH matching_scope AS (
			SELECT scope.id, scope.provider_scope_id, scope.regulated_target_id,
			       scope.status, draft.submitted_planning_item_id,
			       draft.values->>'preparedAuditId' AS prepared_audit_id
			FROM planning_intake_drafts draft
			JOIN canonical_audit_scope_drafts scope
			  ON scope.planning_intake_draft_id = draft.id
			WHERE draft.tombstoned_at IS NULL
			  AND (($2 <> '' AND draft.submitted_planning_item_id = $2)
			       OR ($3 <> '' AND (
						 draft.values->>'preparedAuditId' = $3
						 OR EXISTS (
							 SELECT 1 FROM audit_assignments canonical_assignment
							 WHERE canonical_assignment.inspection_id = $3
							   AND canonical_assignment.planning_item_id = draft.submitted_planning_item_id
							   AND canonical_assignment.tombstoned_at IS NULL
						 )
					 )))
		), authorized_scope AS (
			SELECT 1
			FROM matching_scope selected
			JOIN LATERAL (
				SELECT snapshot.id
				FROM canonical_audit_scope_snapshots snapshot
				WHERE snapshot.scope_draft_id = selected.id
				  AND snapshot.stage = 'RELEASED'
				ORDER BY snapshot.revision DESC, snapshot.id DESC
				LIMIT 1
			) released ON true
			JOIN organization_service_provider_scopes selected_scope
			  ON selected_scope.id = selected.provider_scope_id
			JOIN LATERAL (
				SELECT current_scope.*
				FROM organization_service_provider_scopes current_scope
				WHERE current_scope.root_id = selected_scope.root_id
				  AND current_scope.effective_from <= CURRENT_DATE
				ORDER BY current_scope.effective_from DESC, current_scope.id DESC
				LIMIT 1
			) current_scope ON current_scope.id = selected_scope.id
			JOIN regulated_targets target
			  ON target.id = selected.regulated_target_id
			JOIN service_provider_unit_responsibilities responsibility
			  ON responsibility.service_provider_type_id = selected_scope.service_provider_type_id
			JOIN caa_organizational_units unit
			  ON unit.id = responsibility.organizational_unit_id
			JOIN LATERAL (
				SELECT membership.department_id, membership.organizational_unit_id
				FROM (
					SELECT DISTINCT ON (root_id) *
					FROM caa_department_memberships
					WHERE subject_id = $1 AND effective_from <= CURRENT_DATE
					ORDER BY root_id, effective_from DESC, id DESC
				) membership
				WHERE membership.organizational_unit_id = unit.id
				  AND membership.membership_role = 'DEPARTMENT_MANAGER'
				  AND membership.status = 'ACTIVE'
				  AND (membership.effective_to IS NULL OR membership.effective_to > CURRENT_DATE)
				LIMIT 1
			) membership ON true
			JOIN LATERAL (
				SELECT status
				FROM caa_department_status_facts
				WHERE department_id = membership.department_id
				  AND effective_from <= CURRENT_DATE
				ORDER BY effective_from DESC, id DESC
				LIMIT 1
			) department_status ON department_status.status = 'ACTIVE'
			JOIN LATERAL (
				SELECT status
				FROM caa_organizational_unit_status_facts
				WHERE organizational_unit_id = membership.organizational_unit_id
				  AND effective_from <= CURRENT_DATE
				ORDER BY effective_from DESC, id DESC
				LIMIT 1
			) unit_status ON unit_status.status = 'ACTIVE'
			WHERE selected.status = 'RELEASED'
			  AND selected_scope.status = 'ACTIVE'
			  AND selected_scope.effective_from <= CURRENT_DATE
			  AND (selected_scope.effective_to IS NULL OR selected_scope.effective_to > CURRENT_DATE)
			  AND (target.organization_id IS NULL OR target.organization_id = selected_scope.organization_id)
			  AND (target.owner_organization_id IS NULL OR target.owner_organization_id = selected_scope.organization_id)
			  AND (selected_scope.primary_target_id = target.id OR EXISTS (
				  SELECT 1
				  FROM organization_service_provider_scope_targets linked
				  WHERE linked.organization_service_provider_scope_id = selected_scope.id
				    AND linked.regulated_target_id = target.id
			  ))
		)
		SELECT EXISTS (SELECT 1 FROM matching_scope),
		       EXISTS (SELECT 1 FROM authorized_scope)
	`, actor.SubjectID, planningItemID, inspectionID).Scan(&canonicalScope, &authorized); err != nil {
		return err
	}
	if canonicalScope && !authorized {
		return ErrForbidden
	}
	return nil
}

// RequireCurrentDepartmentScopeAuthority is shared with the application
// materializer so every canonical preparation transition enforces the same
// current provider/department authority boundary.
func RequireCurrentDepartmentScopeAuthority(
	ctx context.Context,
	transaction pgx.Tx,
	actor identity.Principal,
	planningItemID string,
	inspectionID string,
) error {
	return requireCurrentDepartmentScopeAuthority(ctx, transaction, actor, planningItemID, inspectionID)
}

func templateQuestionIDs(
	ctx context.Context,
	transaction pgx.Tx,
	inspectionID string,
	planningItemID string,
) (map[string]bool, error) {
	var snapshot []byte
	if err := transaction.QueryRow(ctx, `
		SELECT template.snapshot
		FROM planning_intake_drafts draft
		JOIN checklist_template_versions template
		  ON template.id = draft.values->>'templateVersionId'
			WHERE (($1 <> '' AND draft.values->>'preparedAuditId' = $1)
			   OR ($2 <> '' AND draft.submitted_planning_item_id = $2))
		  AND draft.tombstoned_at IS NULL
		`, inspectionID, planningItemID).Scan(&snapshot); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// Canonical successor audits are selected from an immutable RELEASED
			// scope snapshot. They intentionally have no checklist-template
			// reference, so assignment eligibility is derived from question
			// version identities instead.
			rows, canonicalErr := transaction.Query(ctx, `
				SELECT question_version_id
				FROM canonical_audit_scope_snapshot_questions
					WHERE snapshot_id = (
					SELECT snapshot.id
					FROM canonical_audit_scope_snapshots snapshot
					JOIN canonical_audit_scope_drafts scope ON scope.id = snapshot.scope_draft_id
					JOIN planning_intake_drafts draft ON draft.id = scope.planning_intake_draft_id
						WHERE (($1 <> '' AND draft.values->>'preparedAuditId' = $1)
						   OR ($2 <> '' AND draft.submitted_planning_item_id = $2))
					  AND snapshot.stage = 'RELEASED'
					ORDER BY snapshot.revision DESC, snapshot.id DESC
					LIMIT 1
				)
				ORDER BY position
				`, inspectionID, planningItemID)
			if canonicalErr != nil {
				return nil, canonicalErr
			}
			defer rows.Close()
			output := map[string]bool{}
			for rows.Next() {
				var questionVersionID string
				if err := rows.Scan(&questionVersionID); err != nil {
					return nil, err
				}
				output[questionVersionID] = true
			}
			if err := rows.Err(); err != nil {
				return nil, err
			}
			if len(output) == 0 {
				return nil, ErrNotFound
			}
			return output, nil
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
