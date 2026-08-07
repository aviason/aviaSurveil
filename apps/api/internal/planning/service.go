package planning

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/application"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/identity"
	planningstore "github.com/MarlonJD/aviaSurveil360/apps/api/internal/planning/store/postgres"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/platform/database"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/platform/idempotency"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type Status string

const (
	StatusFinanceReview           Status = "FINANCE_REVIEW"
	StatusGeneralManagerReview    Status = "GM_REVIEW"
	StatusExecutiveDirectorReview Status = "EXECUTIVE_DIRECTOR_REVIEW"
	StatusGeneralManagerRelease   Status = "GM_RELEASE"
	StatusReleased                Status = "RELEASED"
	StatusReturned                Status = "RETURNED"
)

type Decision string

const (
	DecisionApproveBudget           Decision = "APPROVE_BUDGET"
	DecisionForwardForFinalApproval Decision = "FORWARD_FOR_FINAL_APPROVAL"
	DecisionApprovePlan             Decision = "APPROVE_PLAN"
	DecisionReleasePlan             Decision = "RELEASE_PLAN"
	DecisionReturnForRevision       Decision = "RETURN_FOR_REVISION"
)

type Item struct {
	ID                       string        `json:"id"`
	Title                    string        `json:"title"`
	PlanYear                 int32         `json:"planYear"`
	OrganizationID           string        `json:"organizationId"`
	OrganizationName         string        `json:"organizationName"`
	InspectionType           string        `json:"inspectionType"`
	ScheduledDate            string        `json:"scheduledDate"`
	EstimatedBudget          float64       `json:"estimatedBudget"`
	Status                   Status        `json:"status"`
	CurrentOwnerRole         identity.Role `json:"currentOwnerRole"`
	NextAction               string        `json:"nextAction"`
	Revision                 int64         `json:"revision"`
	SubmittedScopeSnapshotID string        `json:"submittedScopeSnapshotId,omitempty"`
	PlanningSnapshotDigest   string        `json:"planningSnapshotDigest,omitempty"`
}

type DecideCommand struct {
	OperationID                      string
	PlanningItemID                   string
	ExpectedRevision                 int64
	Decision                         Decision
	Reason                           string
	ExpectedSubmittedScopeSnapshotID string
	ExpectedPlanningSnapshotDigest   string
}

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

func (service *Service) List(ctx context.Context, actor identity.Principal, limit int32) ([]Item, error) {
	if !actor.HasRole(
		identity.RoleInspector,
		identity.RoleLeadInspector,
		identity.RoleDepartmentManager,
		identity.RoleFinance,
		identity.RoleGeneralManager,
		identity.RoleExecutiveDirector,
		identity.RoleAdmin,
	) {
		return nil, fmt.Errorf("%w: CAA planning access is required", application.ErrForbidden)
	}
	records, err := planningstore.New(service.pool).ListSurveillancePlanItems(ctx, boundedLimit(limit))
	if err != nil {
		return nil, err
	}
	items := make([]Item, 0, len(records))
	for _, record := range records {
		item := itemFromList(record)
		item.SubmittedScopeSnapshotID, item.PlanningSnapshotDigest, err = service.submittedScopePin(ctx, service.pool, item.ID)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

func (service *Service) Decide(ctx context.Context, actor identity.Principal, command DecideCommand) (Item, error) {
	if strings.TrimSpace(command.OperationID) == "" || strings.TrimSpace(command.PlanningItemID) == "" || strings.TrimSpace(command.Reason) == "" ||
		strings.TrimSpace(command.ExpectedSubmittedScopeSnapshotID) == "" || strings.TrimSpace(command.ExpectedPlanningSnapshotDigest) == "" {
		return Item{}, fmt.Errorf("%w: operation, planning item, reason, submitted scope snapshot, and planning snapshot digest are required", application.ErrInvalid)
	}
	semanticHash, err := idempotency.SemanticHash(struct {
		PlanningItemID                   string   `json:"planningItemId"`
		ExpectedRevision                 int64    `json:"expectedRevision"`
		Decision                         Decision `json:"decision"`
		Reason                           string   `json:"reason"`
		ExpectedSubmittedScopeSnapshotID string   `json:"expectedSubmittedScopeSnapshotId"`
		ExpectedPlanningSnapshotDigest   string   `json:"expectedPlanningSnapshotDigest"`
	}{command.PlanningItemID, command.ExpectedRevision, command.Decision, strings.TrimSpace(command.Reason), strings.TrimSpace(command.ExpectedSubmittedScopeSnapshotID), strings.TrimSpace(command.ExpectedPlanningSnapshotDigest)})
	if err != nil {
		return Item{}, err
	}
	scope := actor.SubjectID + ":planning_decision"
	var output Item
	err = database.WithinTransaction(ctx, service.pool, func(ctx context.Context, transaction pgx.Tx) error {
		if _, err := transaction.Exec(ctx, "SELECT pg_advisory_xact_lock(hashtextextended($1, 0))", scope+":"+command.OperationID); err != nil {
			return err
		}
		var storedHash string
		var storedBody []byte
		err := transaction.QueryRow(ctx, `
			SELECT semantic_hash, response_body FROM idempotency_responses
			WHERE scope = $1 AND operation_id = $2
		`, scope, command.OperationID).Scan(&storedHash, &storedBody)
		if err == nil {
			if storedHash != semanticHash {
				return idempotency.ErrOperationIDReuse
			}
			return json.Unmarshal(storedBody, &output)
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return err
		}

		queries := planningstore.New(transaction)
		current, err := queries.GetSurveillancePlanItemForUpdate(ctx, command.PlanningItemID)
		if errors.Is(err, pgx.ErrNoRows) {
			return application.ErrNotFound
		}
		if err != nil {
			return err
		}
		var submittedScopeSnapshotID, planningSnapshotDigest, selectionDigest string
		var submittedSnapshot []byte
		if err := transaction.QueryRow(ctx, `
			SELECT snapshot.id, snapshot.planning_snapshot_digest, snapshot.selection_digest, snapshot.snapshot
			FROM canonical_audit_scope_snapshots snapshot
			JOIN canonical_audit_scope_drafts scope ON scope.id = snapshot.scope_draft_id
			JOIN planning_intake_drafts draft ON draft.id = scope.planning_intake_draft_id
			WHERE draft.submitted_planning_item_id = $1
			  AND snapshot.stage = 'SUBMITTED'
			ORDER BY snapshot.revision DESC, snapshot.id DESC
			LIMIT 1
		`, command.PlanningItemID).Scan(&submittedScopeSnapshotID, &planningSnapshotDigest, &selectionDigest, &submittedSnapshot); err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		if planningSnapshotDigest == "" && len(submittedSnapshot) > 0 {
			planningSnapshotDigest = derivedPlanningSnapshotDigest(submittedSnapshot)
		}
		if submittedScopeSnapshotID == "" || planningSnapshotDigest == "" {
			return fmt.Errorf("%w: submitted scope snapshot is required for every canonical planning decision", application.ErrConflict)
		}
		if command.ExpectedSubmittedScopeSnapshotID != submittedScopeSnapshotID || command.ExpectedPlanningSnapshotDigest != planningSnapshotDigest {
			return fmt.Errorf("%w: expected immutable submitted scope snapshot pin does not match", application.ErrConflict)
		}
		if current.Revision != command.ExpectedRevision {
			return fmt.Errorf("%w: Planning item revision conflict", application.ErrConflict)
		}
		status, owner, nextAction, auditAction, err := decideTransition(actor, Status(current.Status), command.Decision)
		if err != nil {
			return err
		}
		now := service.clock().UTC()
		updated, err := queries.UpdateSurveillancePlanDecision(ctx, planningstore.UpdateSurveillancePlanDecisionParams{
			ID: current.ID, Status: string(status), CurrentOwnerRole: string(owner), NextAction: nextAction,
			UpdatedAt: pgtype.Timestamptz{Time: now, Valid: true}, Revision: current.Revision,
		})
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("%w: Planning item revision conflict", application.ErrConflict)
		}
		if err != nil {
			return err
		}
		if command.Decision == DecisionReleasePlan {
			if err := releaseCanonicalScopeSnapshot(ctx, transaction, actor, current.ID, now); err != nil {
				return err
			}
		}
		output = Item{
			ID: updated.ID, Title: updated.Title, PlanYear: updated.PlanYear,
			OrganizationID: updated.OrganizationID, OrganizationName: current.LegalName,
			InspectionType: updated.InspectionType, ScheduledDate: updated.ScheduledDate.Time.Format("2006-01-02"),
			EstimatedBudget: updated.EstimatedBudget, Status: Status(updated.Status),
			CurrentOwnerRole: identity.Role(updated.CurrentOwnerRole), NextAction: updated.NextAction,
			Revision: updated.Revision, SubmittedScopeSnapshotID: submittedScopeSnapshotID,
			PlanningSnapshotDigest: planningSnapshotDigest,
		}
		responseBody, err := json.Marshal(output)
		if err != nil {
			return err
		}
		details := []byte(`{}`)
		if planningSnapshotDigest != "" {
			details, err = json.Marshal(map[string]any{
				"submittedScopeSnapshotId": submittedScopeSnapshotID,
				"planningSnapshotDigest":   planningSnapshotDigest,
				"selectionDigest":          selectionDigest,
			})
			if err != nil {
				return err
			}
		}
		actorRole := ""
		if len(actor.Roles) > 0 {
			actorRole = string(actor.Roles[0])
		}
		auditID := service.idGenerator("audit-plan")
		outboxID := service.idGenerator("outbox-plan")
		if _, err := transaction.Exec(ctx, `
			INSERT INTO audit_events (
				event_id, occurred_at, actor_subject_id, actor_role, organization_id, action,
				entity_type, entity_id, entity_version, before_status, after_status, reason,
				operation_id, correlation_id, request_id, details
			) VALUES ($1, $2, $3, $4, $5, $6, 'SURVEILLANCE_PLAN', $7, $8, $9, $10, $11, $12, $12, $12, $13)
		`, auditID, now, actor.SubjectID, actorRole, current.OrganizationID,
			auditAction, current.ID, updated.Revision, current.Status, updated.Status,
			strings.TrimSpace(command.Reason), command.OperationID, details); err != nil {
			return err
		}
		var changeID int64
		if err := transaction.QueryRow(ctx, `
			INSERT INTO authorized_sync_changes (
				subject_id, organization_id, kind, entity_id, entity_revision,
				payload, changed_at, operation_id, correlation_id
			) VALUES ($1, $2, 'SURVEILLANCE_PLAN', $3, $4, $5, $6, $7, $7)
			RETURNING sequence_id
		`, actor.SubjectID, current.OrganizationID, current.ID, updated.Revision,
			responseBody, now, command.OperationID).Scan(&changeID); err != nil {
			return err
		}
		if _, err := transaction.Exec(ctx, `
			INSERT INTO outbox_messages (
				id, topic, aggregate_type, aggregate_id, payload, available_at,
				idempotency_key, operation_id, correlation_id
			) VALUES (
				$1, 'planning.decision.recorded', 'SURVEILLANCE_PLAN', $2, $3, $4,
				$5, $6, $6
			)
		`, outboxID, current.ID, responseBody, now,
			planningCommandIdempotencyKey(scope, command.OperationID),
			command.OperationID); err != nil {
			return err
		}
		if _, err := transaction.Exec(ctx, `
			INSERT INTO idempotency_responses (
				scope, operation_id, semantic_hash, response_status, response_headers, response_body, created_at
			) VALUES ($1, $2, $3, 200, '{}'::jsonb, $4, $5)
		`, scope, command.OperationID, semanticHash, responseBody, now); err != nil {
			return err
		}
		if _, err := transaction.Exec(ctx, `
			INSERT INTO command_transaction_links (
				operation_id, idempotency_scope, audit_event_id,
				change_sequence_id, outbox_message_id, created_at
			) VALUES ($1, $2, $3, $4, $5, $6)
		`, command.OperationID, scope, auditID, changeID, outboxID, now); err != nil {
			return err
		}
		return nil
	})
	return output, err
}

// releaseCanonicalScopeSnapshot pins the exact question identities approved by
// Finance/GM/Executive Director to the GM Release transition. Materialization
// reads this immutable RELEASED snapshot; it never re-reads a mutable draft or
// a checklist template.
func releaseCanonicalScopeSnapshot(
	ctx context.Context,
	tx pgx.Tx,
	actor identity.Principal,
	planningItemID string,
	now time.Time,
) error {
	var scopeID, submittedID, catalogID, usageClass, digest, planningSnapshotDigest string
	var revision int64
	var selectedCount int
	var snapshot []byte
	err := tx.QueryRow(ctx, `
		SELECT scope.id, submitted.id, submitted.catalog_id, submitted.usage_class,
		       submitted.revision, submitted.selection_digest, submitted.planning_snapshot_digest,
		       submitted.selected_question_count, submitted.snapshot
		FROM canonical_audit_scope_drafts scope
		JOIN planning_intake_drafts draft ON draft.id = scope.planning_intake_draft_id
		JOIN LATERAL (
			SELECT snapshot.id, snapshot.catalog_id, snapshot.usage_class,
			       snapshot.revision, snapshot.selection_digest,
			       snapshot.planning_snapshot_digest, snapshot.selected_question_count, snapshot.snapshot
			FROM canonical_audit_scope_snapshots snapshot
			WHERE snapshot.scope_draft_id = scope.id AND snapshot.stage = 'SUBMITTED'
			ORDER BY snapshot.revision DESC, snapshot.id DESC
			LIMIT 1
		) submitted ON true
		WHERE draft.submitted_planning_item_id = $1
		  AND scope.status IN ('SUBMITTED', 'DRAFT')
		FOR UPDATE OF scope
	`, planningItemID).Scan(&scopeID, &submittedID, &catalogID, &usageClass, &revision, &digest, &planningSnapshotDigest, &selectedCount, &snapshot)
	if errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("%w: canonical submitted scope snapshot is required before release", application.ErrConflict)
	}
	if err != nil {
		return err
	}
	if planningSnapshotDigest == "" && len(snapshot) > 0 {
		planningSnapshotDigest = derivedPlanningSnapshotDigest(snapshot)
	}
	if selectedCount <= 0 || digest == "" || planningSnapshotDigest == "" || scopeID == "" || submittedID == "" {
		return fmt.Errorf("%w: canonical scope release requires a non-empty submitted snapshot", application.ErrConflict)
	}
	releasedID := "scope-snapshot:" + scopeID + ":released:" + fmt.Sprint(revision)
	if _, err := tx.Exec(ctx, `
		INSERT INTO canonical_audit_scope_snapshots (
			id, scope_draft_id, revision, stage, catalog_id, usage_class,
			selection_digest, planning_snapshot_digest, selected_question_count, snapshot,
			created_by_subject_id, created_at
		) VALUES ($1, $2, $3, 'RELEASED', $4, $5, $6, $7, $8, $9, $10, $11)
	`, releasedID, scopeID, revision, catalogID, usageClass, digest, planningSnapshotDigest, selectedCount, snapshot, actor.SubjectID, now); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO canonical_audit_scope_snapshot_questions (snapshot_id, catalog_id, question_version_id, position)
		SELECT $1, catalog_id, question_version_id, position
		FROM canonical_audit_scope_snapshot_questions
		WHERE snapshot_id = $2
		ORDER BY position
	`, releasedID, submittedID); err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `
		UPDATE canonical_audit_scope_drafts
		SET status = 'RELEASED', updated_at = $2
		WHERE id = $1
	`, scopeID, now)
	return err
}

func decideTransition(actor identity.Principal, status Status, decision Decision) (Status, identity.Role, string, string, error) {
	switch decision {
	case DecisionApproveBudget:
		if !actor.HasRole(identity.RoleFinance) {
			return "", "", "", "", fmt.Errorf("%w: Finance Review authority is required", application.ErrForbidden)
		}
		if status != StatusFinanceReview {
			return "", "", "", "", fmt.Errorf("%w: Planning item is not at Finance Review", application.ErrConflict)
		}
		return StatusGeneralManagerReview, identity.RoleGeneralManager, "General Manager to review operational scope", "PLANNING_BUDGET_APPROVED", nil
	case DecisionForwardForFinalApproval:
		if !actor.HasRole(identity.RoleGeneralManager) {
			return "", "", "", "", fmt.Errorf("%w: General Manager authority is required", application.ErrForbidden)
		}
		if status != StatusGeneralManagerReview {
			return "", "", "", "", fmt.Errorf("%w: Planning item is not at General Manager review", application.ErrConflict)
		}
		return StatusExecutiveDirectorReview, identity.RoleExecutiveDirector, "Executive Director to approve or return plan", "PLANNING_FORWARDED_FOR_FINAL_APPROVAL", nil
	case DecisionApprovePlan:
		if !actor.HasRole(identity.RoleExecutiveDirector) {
			return "", "", "", "", fmt.Errorf("%w: Executive Director authority is required", application.ErrForbidden)
		}
		if status != StatusExecutiveDirectorReview {
			return "", "", "", "", fmt.Errorf("%w: Planning item is not at Executive Director review", application.ErrConflict)
		}
		return StatusGeneralManagerRelease, identity.RoleGeneralManager, "General Manager to release approved plan", "PLANNING_APPROVED", nil
	case DecisionReleasePlan:
		if !actor.HasRole(identity.RoleGeneralManager) {
			return "", "", "", "", fmt.Errorf("%w: General Manager authority is required", application.ErrForbidden)
		}
		if status != StatusGeneralManagerRelease {
			return "", "", "", "", fmt.Errorf("%w: Planning item is not ready for General Manager release", application.ErrConflict)
		}
		return StatusReleased, identity.RoleDepartmentManager, "Department Manager to prepare the scheduled Audit", "PLANNING_RELEASED", nil
	case DecisionReturnForRevision:
		allowed := (actor.HasRole(identity.RoleFinance) && status == StatusFinanceReview) ||
			(actor.HasRole(identity.RoleGeneralManager) && (status == StatusGeneralManagerReview || status == StatusGeneralManagerRelease)) ||
			(actor.HasRole(identity.RoleExecutiveDirector) && status == StatusExecutiveDirectorReview)
		if !allowed {
			return "", "", "", "", fmt.Errorf("%w: current role and stage cannot return this item", application.ErrForbidden)
		}
		return StatusReturned, identity.RoleDepartmentManager, "Department Manager to revise and resubmit plan", "PLANNING_RETURNED_FOR_REVISION", nil
	default:
		return "", "", "", "", fmt.Errorf("%w: unsupported planning decision", application.ErrInvalid)
	}
}

func itemFromList(record planningstore.ListSurveillancePlanItemsRow) Item {
	return Item{
		ID: record.ID, Title: record.Title, PlanYear: record.PlanYear,
		OrganizationID: record.OrganizationID, OrganizationName: record.LegalName,
		InspectionType: record.InspectionType, ScheduledDate: record.ScheduledDate.Time.Format("2006-01-02"),
		EstimatedBudget: record.EstimatedBudget, Status: Status(record.Status),
		CurrentOwnerRole: identity.Role(record.CurrentOwnerRole), NextAction: record.NextAction,
		Revision: record.Revision,
	}
}

func (service *Service) submittedScopePin(ctx context.Context, query interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, planningItemID string) (string, string, error) {
	var snapshotID, digest string
	var snapshot []byte
	err := query.QueryRow(ctx, `
		SELECT snapshot.id, snapshot.planning_snapshot_digest, snapshot.snapshot
		FROM canonical_audit_scope_snapshots snapshot
		JOIN canonical_audit_scope_drafts scope ON scope.id = snapshot.scope_draft_id
		JOIN planning_intake_drafts draft ON draft.id = scope.planning_intake_draft_id
		WHERE draft.submitted_planning_item_id = $1 AND snapshot.stage = 'SUBMITTED'
		ORDER BY snapshot.revision DESC, snapshot.id DESC
		LIMIT 1
	`, planningItemID).Scan(&snapshotID, &digest, &snapshot)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", "", nil
	}
	if digest == "" && len(snapshot) > 0 {
		digest = derivedPlanningSnapshotDigest(snapshot)
	}
	return snapshotID, digest, err
}

func derivedPlanningSnapshotDigest(snapshot []byte) string {
	digest := sha256.Sum256(snapshot)
	return "sha256:" + hex.EncodeToString(digest[:])
}

func boundedLimit(limit int32) int32 {
	if limit <= 0 || limit > 100 {
		return 100
	}
	return limit
}

func randomID(prefix string) string {
	bytes := make([]byte, 12)
	if _, err := rand.Read(bytes); err != nil {
		panic(err)
	}
	return prefix + "-" + hex.EncodeToString(bytes)
}
