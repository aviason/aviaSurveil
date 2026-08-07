package planning

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/application"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/identity"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/platform/database"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/platform/idempotency"
	"github.com/jackc/pgx/v5"
)

type InspectionCategory string

const (
	InspectionCategoryRoutine InspectionCategory = "Routine / Announced"
	InspectionCategoryAdHoc   InspectionCategory = "Ad Hoc / Unannounced"
)

type NoticePolicy string

const (
	NoticePolicyAdvance  NoticePolicy = "ADVANCE"
	NoticePolicyWithheld NoticePolicy = "WITHHELD"
)

type IntakeDraftValues struct {
	OrganizationID     string             `json:"organizationId"`
	OrganizationName   string             `json:"organizationName"`
	ApplicationType    string             `json:"applicationType"`
	Domain             string             `json:"domain"`
	InspectionCategory InspectionCategory `json:"inspectionCategory"`
	NoticePolicy       NoticePolicy       `json:"noticePolicy"`
	Purpose            string             `json:"purpose"`
	TriggerType        string             `json:"triggerType"`
	RiskCategory       string             `json:"riskCategory"`
	PlannedDate        string             `json:"plannedDate"`
	Mode               string             `json:"mode"`
	Location           string             `json:"location"`
	TemplateVersionID  string             `json:"templateVersionId"`
	Scope              string             `json:"scope"`
	CatalogVersion     string             `json:"catalogVersion,omitempty"`
	ScopeDraftID       string             `json:"scopeDraftId,omitempty"`
	SelectionDigest    string             `json:"selectionDigest,omitempty"`
	SelectedQuestionVersionIDs []string   `json:"selectedQuestionVersionIds,omitempty"`
	EstimatedResourceRequirement float64   `json:"estimatedResourceRequirement,omitempty"`
	ProviderScopeID    string             `json:"providerScopeId,omitempty"`
	RegulatedTargetID  string             `json:"regulatedTargetId,omitempty"`
	RequestedBudget    float64            `json:"requestedBudget"`
	Currency           string             `json:"currency"`
}

type IntakeDraft struct {
	IntakeDraftValues
	ID                      string    `json:"id"`
	Revision                int64     `json:"revision"`
	SubmittedPlanningItemID *string   `json:"submittedPlanningItemId"`
	UpdatedAt               time.Time `json:"updatedAt"`
}

type SaveIntakeDraftCommand struct {
	OperationID      string
	IdempotencyKey   string
	DraftID          string
	ExpectedRevision int64
	Values           IntakeDraftValues
}

type SubmitIntakeCommand struct {
	OperationID      string
	IdempotencyKey   string
	DraftID          string
	PlanningItemID   string
	ExpectedRevision int64
}

type SubmitIntakeResult struct {
	Draft        IntakeDraft `json:"draft"`
	PlanningItem Item        `json:"planningItem"`
}

func (service *Service) GetIntakeDraft(
	ctx context.Context,
	actor identity.Principal,
	draftID string,
) (IntakeDraft, error) {
	if !actor.HasRole(identity.RoleDepartmentManager) {
		return IntakeDraft{}, fmt.Errorf("%w: Department Manager authority is required", application.ErrForbidden)
	}
	if strings.TrimSpace(draftID) == "" {
		return IntakeDraft{}, application.ErrInvalid
	}
	return getIntakeDraft(ctx, service.pool, draftID, false)
}

func (service *Service) SaveIntakeDraft(
	ctx context.Context,
	actor identity.Principal,
	command SaveIntakeDraftCommand,
) (IntakeDraft, error) {
	if !actor.HasRole(identity.RoleDepartmentManager) {
		return IntakeDraft{}, fmt.Errorf("%w: Department Manager authority is required", application.ErrForbidden)
	}
	command.Values = normalizedIntakeValues(command.Values)
	if strings.TrimSpace(command.OperationID) == "" ||
		strings.TrimSpace(command.IdempotencyKey) == "" ||
		strings.TrimSpace(command.DraftID) == "" ||
		command.ExpectedRevision <= 0 ||
		validateDraftValues(command.Values, false) != nil {
		return IntakeDraft{}, application.ErrInvalid
	}
	return executeIntakeCommand(ctx, service, actor, "save_planning_intake",
		command.OperationID, command.IdempotencyKey, command.DraftID, command,
		func(ctx context.Context, transaction pgx.Tx) (intakeCommandResult[IntakeDraft], error) {
			current, err := getIntakeDraft(ctx, transaction, command.DraftID, true)
			if err != nil {
				return intakeCommandResult[IntakeDraft]{}, err
			}
			if current.Revision != command.ExpectedRevision {
				return intakeCommandResult[IntakeDraft]{}, application.ErrConflict
			}
			if current.SubmittedPlanningItemID != nil {
				var status string
				if err := transaction.QueryRow(ctx, `
					SELECT status FROM surveillance_plan_items
					WHERE id = $1 AND tombstoned_at IS NULL
				`, *current.SubmittedPlanningItemID).Scan(&status); err != nil {
					return intakeCommandResult[IntakeDraft]{}, err
				}
				if Status(status) != StatusReturned {
					return intakeCommandResult[IntakeDraft]{}, application.ErrConflict
				}
			}
			var legalName string
			if err := transaction.QueryRow(ctx, `
				SELECT legal_name FROM organizations
				WHERE id = $1 AND tombstoned_at IS NULL
			`, command.Values.OrganizationID).Scan(&legalName); err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					return intakeCommandResult[IntakeDraft]{}, application.ErrNotFound
				}
				return intakeCommandResult[IntakeDraft]{}, err
			}
			command.Values.OrganizationName = legalName
			valuesJSON, err := json.Marshal(command.Values)
			if err != nil {
				return intakeCommandResult[IntakeDraft]{}, err
			}
			now := service.clock().UTC()
			var revision int64
			if err := transaction.QueryRow(ctx, `
				UPDATE planning_intake_drafts
				SET organization_id = $2, values = $3, revision = revision + 1,
				    updated_at = $4
				WHERE id = $1 AND revision = $5 AND tombstoned_at IS NULL
				RETURNING revision
			`, command.DraftID, command.Values.OrganizationID, valuesJSON, now,
				command.ExpectedRevision).Scan(&revision); err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					return intakeCommandResult[IntakeDraft]{}, application.ErrConflict
				}
				return intakeCommandResult[IntakeDraft]{}, err
			}
			output := IntakeDraft{
				IntakeDraftValues: command.Values, ID: command.DraftID,
				Revision: revision, SubmittedPlanningItemID: current.SubmittedPlanningItemID,
				UpdatedAt: now,
			}
			return intakeCommandResult[IntakeDraft]{
				Response: output, OrganizationID: command.Values.OrganizationID,
				Action: "planning.intake_saved", EntityType: "planning_intake_draft",
				EntityID: command.DraftID, EntityVersion: revision,
				BeforeStatus: "DRAFT", AfterStatus: "DRAFT",
			}, nil
		})
}

func (service *Service) SubmitIntake(
	ctx context.Context,
	actor identity.Principal,
	command SubmitIntakeCommand,
) (SubmitIntakeResult, error) {
	if !actor.HasRole(identity.RoleDepartmentManager) {
		return SubmitIntakeResult{}, fmt.Errorf("%w: Department Manager authority is required", application.ErrForbidden)
	}
	if strings.TrimSpace(command.OperationID) == "" ||
		strings.TrimSpace(command.IdempotencyKey) == "" ||
		strings.TrimSpace(command.DraftID) == "" ||
		command.ExpectedRevision <= 0 {
		return SubmitIntakeResult{}, application.ErrInvalid
	}
	if strings.TrimSpace(command.PlanningItemID) == "" {
		digest := sha256.Sum256([]byte("planning-item:" + command.DraftID + ":" + command.OperationID))
		command.PlanningItemID = "plan-intake-" + hex.EncodeToString(digest[:])[:24]
	}
	return executeIntakeCommand(ctx, service, actor, "submit_planning_intake",
		command.OperationID, command.IdempotencyKey, command.PlanningItemID, command,
		func(ctx context.Context, transaction pgx.Tx) (intakeCommandResult[SubmitIntakeResult], error) {
			draft, err := getIntakeDraft(ctx, transaction, command.DraftID, true)
			if err != nil {
				return intakeCommandResult[SubmitIntakeResult]{}, err
			}
			if draft.Revision != command.ExpectedRevision ||
				validateDraftValues(draft.IntakeDraftValues, true) != nil {
				return intakeCommandResult[SubmitIntakeResult]{}, application.ErrConflict
			}
			title := fmt.Sprintf("%s — %s", draft.InspectionCategory, draft.OrganizationName)
			inspectionType := draft.ApplicationType + " · " + draft.Domain
			plannedDate, _ := time.Parse("2006-01-02", draft.PlannedDate)
			now := service.clock().UTC()
			var item Item
			if draft.SubmittedPlanningItemID == nil {
				if err := transaction.QueryRow(ctx, `
					INSERT INTO surveillance_plan_items (
						id, title, plan_year, organization_id, inspection_type,
						scheduled_date, estimated_budget, status, current_owner_role,
						next_action, revision, created_at, updated_at
					) VALUES (
						$1, $2, $3, $4, $5, $6, $7, 'FINANCE_REVIEW', 'finance',
						'Finance to review budget and resources', 1, $8, $8
					)
					RETURNING id, title, plan_year, organization_id, inspection_type,
					          scheduled_date, estimated_budget::float8, status,
					          current_owner_role, next_action, revision
				`, command.PlanningItemID, title, plannedDate.Year(), draft.OrganizationID,
					inspectionType, plannedDate, draft.RequestedBudget, now).Scan(
					&item.ID, &item.Title, &item.PlanYear, &item.OrganizationID,
					&item.InspectionType, &plannedDate, &item.EstimatedBudget, &item.Status,
					&item.CurrentOwnerRole, &item.NextAction, &item.Revision,
				); err != nil {
					return intakeCommandResult[SubmitIntakeResult]{}, err
				}
			} else {
				if *draft.SubmittedPlanningItemID != command.PlanningItemID {
					return intakeCommandResult[SubmitIntakeResult]{}, application.ErrConflict
				}
				if err := transaction.QueryRow(ctx, `
					UPDATE surveillance_plan_items
					SET title = $2, plan_year = $3, organization_id = $4,
					    inspection_type = $5, scheduled_date = $6, estimated_budget = $7,
					    status = 'FINANCE_REVIEW', current_owner_role = 'finance',
					    next_action = 'Finance to review budget and resources',
					    revision = revision + 1, updated_at = $8
					WHERE id = $1 AND status = 'RETURNED' AND tombstoned_at IS NULL
					RETURNING id, title, plan_year, organization_id, inspection_type,
					          scheduled_date, estimated_budget::float8, status,
					          current_owner_role, next_action, revision
				`, command.PlanningItemID, title, plannedDate.Year(), draft.OrganizationID,
					inspectionType, plannedDate, draft.RequestedBudget, now).Scan(
					&item.ID, &item.Title, &item.PlanYear, &item.OrganizationID,
					&item.InspectionType, &plannedDate, &item.EstimatedBudget, &item.Status,
					&item.CurrentOwnerRole, &item.NextAction, &item.Revision,
				); err != nil {
					if errors.Is(err, pgx.ErrNoRows) {
						return intakeCommandResult[SubmitIntakeResult]{}, application.ErrConflict
					}
					return intakeCommandResult[SubmitIntakeResult]{}, err
				}
			}
			item.OrganizationName = draft.OrganizationName
			item.ScheduledDate = plannedDate.Format("2006-01-02")
			submittedID := command.PlanningItemID
			if err := transaction.QueryRow(ctx, `
				UPDATE planning_intake_drafts
				SET submitted_planning_item_id = $2, revision = revision + 1,
				    updated_at = $3
				WHERE id = $1 AND revision = $4 AND tombstoned_at IS NULL
				RETURNING revision
			`, command.DraftID, submittedID, now, command.ExpectedRevision).Scan(
				&draft.Revision,
			); err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					return intakeCommandResult[SubmitIntakeResult]{}, application.ErrConflict
				}
				return intakeCommandResult[SubmitIntakeResult]{}, err
			}
			draft.SubmittedPlanningItemID = &submittedID
			draft.UpdatedAt = now
			output := SubmitIntakeResult{Draft: draft, PlanningItem: item}
			return intakeCommandResult[SubmitIntakeResult]{
				Response: output, OrganizationID: draft.OrganizationID,
				Action: "planning.intake_submitted", EntityType: "SURVEILLANCE_PLAN",
				EntityID: item.ID, EntityVersion: item.Revision,
				BeforeStatus: "DRAFT", AfterStatus: string(StatusFinanceReview),
			}, nil
		})
}

type intakeCommandResult[T any] struct {
	Response       T
	OrganizationID string
	Action         string
	EntityType     string
	EntityID       string
	EntityVersion  int64
	BeforeStatus   string
	AfterStatus    string
}

func executeIntakeCommand[T any](
	ctx context.Context,
	service *Service,
	actor identity.Principal,
	kind, operationID, idempotencyKey, aggregateID string,
	semantic any,
	handler func(context.Context, pgx.Tx) (intakeCommandResult[T], error),
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
		`, planningCommandIdempotencyKey(scope, idempotencyKey)).Scan(&reused); err != nil {
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
		auditID := service.idGenerator("audit-planning-intake")
		outboxID := service.idGenerator("outbox-planning-intake")
		if _, err := transaction.Exec(ctx, `
			INSERT INTO audit_events (
				event_id, occurred_at, actor_subject_id, actor_role, organization_id,
				action, entity_type, entity_id, entity_version, before_status,
				after_status, operation_id, correlation_id, request_id, details
			) VALUES (
				$1, $2, $3, $4, $5, $6, $7, $8, $9, NULLIF($10, ''),
				NULLIF($11, ''), $12, $12, $12, '{}'::jsonb
			)
		`, auditID, now, actor.SubjectID, role, result.OrganizationID, result.Action,
			result.EntityType, result.EntityID, result.EntityVersion,
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
		`, outboxID, result.Action, result.EntityType, aggregateID, responseBody, now,
			planningCommandIdempotencyKey(scope, idempotencyKey), operationID); err != nil {
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

type rowQuerier interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

func getIntakeDraft(
	ctx context.Context,
	querier rowQuerier,
	draftID string,
	lock bool,
) (IntakeDraft, error) {
	query := `
		SELECT draft.id, draft.values, draft.submitted_planning_item_id,
		       draft.revision, draft.updated_at, organization.legal_name
		FROM planning_intake_drafts draft
		JOIN organizations organization ON organization.id = draft.organization_id
		WHERE draft.id = $1
		  AND draft.tombstoned_at IS NULL
		  AND organization.tombstoned_at IS NULL
	`
	if lock {
		query += " FOR UPDATE OF draft"
	}
	var output IntakeDraft
	var valuesJSON []byte
	var organizationName string
	if err := querier.QueryRow(ctx, query, draftID).Scan(
		&output.ID, &valuesJSON, &output.SubmittedPlanningItemID,
		&output.Revision, &output.UpdatedAt, &organizationName,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return IntakeDraft{}, application.ErrNotFound
		}
		return IntakeDraft{}, err
	}
	if err := json.Unmarshal(valuesJSON, &output.IntakeDraftValues); err != nil {
		return IntakeDraft{}, err
	}
	if output.OrganizationName == "" {
		output.OrganizationName = organizationName
	}
	return output, nil
}

func normalizedIntakeValues(values IntakeDraftValues) IntakeDraftValues {
	values.OrganizationID = strings.TrimSpace(values.OrganizationID)
	values.OrganizationName = strings.TrimSpace(values.OrganizationName)
	values.ApplicationType = strings.TrimSpace(values.ApplicationType)
	values.Domain = strings.TrimSpace(values.Domain)
	values.Purpose = strings.TrimSpace(values.Purpose)
	values.TriggerType = strings.TrimSpace(values.TriggerType)
	values.RiskCategory = strings.TrimSpace(values.RiskCategory)
	values.PlannedDate = strings.TrimSpace(values.PlannedDate)
	values.Mode = strings.TrimSpace(values.Mode)
	values.Location = strings.TrimSpace(values.Location)
	values.TemplateVersionID = strings.TrimSpace(values.TemplateVersionID)
	values.Scope = strings.TrimSpace(values.Scope)
	values.CatalogVersion = strings.TrimSpace(values.CatalogVersion)
	values.ScopeDraftID = strings.TrimSpace(values.ScopeDraftID)
	values.SelectionDigest = strings.TrimSpace(values.SelectionDigest)
	values.ProviderScopeID = strings.TrimSpace(values.ProviderScopeID)
	values.RegulatedTargetID = strings.TrimSpace(values.RegulatedTargetID)
	values.Currency = strings.TrimSpace(values.Currency)
	switch values.InspectionCategory {
	case InspectionCategoryRoutine:
		values.NoticePolicy = NoticePolicyAdvance
	case InspectionCategoryAdHoc:
		values.NoticePolicy = NoticePolicyWithheld
	}
	return values
}

func validateDraftValues(values IntakeDraftValues, complete bool) error {
	if values.OrganizationID == "" || values.RequestedBudget < 0 {
		return application.ErrInvalid
	}
	if values.InspectionCategory != InspectionCategoryRoutine &&
		values.InspectionCategory != InspectionCategoryAdHoc {
		return application.ErrInvalid
	}
	if _, err := time.Parse("2006-01-02", values.PlannedDate); err != nil {
		return application.ErrInvalid
	}
	if values.Mode != "On-site" && values.Mode != "Remote" {
		return application.ErrInvalid
	}
	if values.Currency != "USD" && values.Currency != "EUR" && values.Currency != "NAD" {
		return application.ErrInvalid
	}
	if complete && (values.ApplicationType == "" || values.Domain == "" ||
		values.Purpose == "" || values.TriggerType == "" || values.RiskCategory == "" || values.Location == "") {
		return application.ErrInvalid
	}
	if complete {
		legacyScope := values.TemplateVersionID != "" && values.Scope != ""
		canonicalScope := values.CatalogVersion != "" && values.ScopeDraftID != "" && values.SelectionDigest != "" && len(values.SelectedQuestionVersionIDs) > 0
		if !legacyScope && !canonicalScope { return application.ErrInvalid }
	}
	return nil
}

func planningCommandIdempotencyKey(scope, key string) string {
	return "command:" + scope + ":idempotency:" + key
}
