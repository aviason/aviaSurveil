package planning

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/aviason/aviaSurveil/internal/application"
	"github.com/aviason/aviaSurveil/internal/identity"
	"github.com/aviason/aviaSurveil/internal/platform/database"
	"github.com/aviason/aviaSurveil/internal/platform/idempotency"
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
	OrganizationID               string             `json:"organizationId"`
	OrganizationName             string             `json:"organizationName"`
	ApplicationType              string             `json:"applicationType"`
	Domain                       string             `json:"domain"`
	InspectionCategory           InspectionCategory `json:"inspectionCategory"`
	NoticePolicy                 NoticePolicy       `json:"noticePolicy"`
	Purpose                      string             `json:"purpose"`
	TriggerType                  string             `json:"triggerType"`
	RiskCategory                 string             `json:"riskCategory"`
	PlannedDate                  string             `json:"plannedDate"`
	Mode                         string             `json:"mode"`
	Location                     string             `json:"location"`
	TemplateVersionID            string             `json:"templateVersionId"`
	Scope                        string             `json:"scope"`
	CatalogVersion               string             `json:"catalogVersion,omitempty"`
	ScopeDraftID                 string             `json:"scopeDraftId,omitempty"`
	SelectionDigest              string             `json:"selectionDigest,omitempty"`
	SelectedQuestionVersionIDs   []string           `json:"selectedQuestionVersionIds,omitempty"`
	EstimatedResourceRequirement float64            `json:"estimatedResourceRequirement,omitempty"`
	FormDistribution             map[string]any     `json:"formDistribution,omitempty"`
	DomainDistribution           map[string]any     `json:"domainDistribution,omitempty"`
	ProviderScopeID              string             `json:"providerScopeId,omitempty"`
	RegulatedTargetID            string             `json:"regulatedTargetId,omitempty"`
	RequestedBudget              float64            `json:"requestedBudget"`
	Currency                     string             `json:"currency"`
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
	draft, err := getIntakeDraft(ctx, service.pool, draftID, false, actor.SubjectID)
	if err != nil {
		return IntakeDraft{}, err
	}
	// Reads are also authority checks. A manager who created a draft but has
	// since lost the provider/depart­ment responsibility must not retain a
	// usable canonical planning projection.
	values, err := canonicalValuesMap(draft.IntakeDraftValues, draft.ID)
	if err != nil {
		return IntakeDraft{}, application.ErrInvalid
	}
	if _, err := application.ValidateCanonicalScopeMap(ctx, service.pool, actor, values, false); err != nil {
		return IntakeDraft{}, err
	}
	return draft, nil
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
	if command.Values.CatalogVersion == "" ||
		command.Values.ProviderScopeID == "" || command.Values.RegulatedTargetID == "" ||
		command.Values.TemplateVersionID != "" || command.Values.Scope != "" {
		return IntakeDraft{}, fmt.Errorf("%w: New Audit requires a canonical catalog and authorized scope", application.ErrInvalid)
	}
	return executeIntakeCommand(ctx, service, actor, "save_planning_intake",
		command.OperationID, command.IdempotencyKey, command.DraftID, command,
		func(ctx context.Context, transaction pgx.Tx) (intakeCommandResult[IntakeDraft], error) {
			current, err := getIntakeDraft(ctx, transaction, command.DraftID, true, actor.SubjectID)
			if err != nil {
				return intakeCommandResult[IntakeDraft]{}, err
			}
			if current.Revision != command.ExpectedRevision {
				return intakeCommandResult[IntakeDraft]{}, application.ErrConflict
			}
			returned := false
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
				returned = true
			}
			if returned {
				// A returned Planning item reopens only the mutable scope draft. The
				// previous SUBMITTED snapshot remains append-only and is replaced by
				// a new revision on the next Finance submission.
				if _, err := transaction.Exec(ctx, `
					UPDATE canonical_audit_scope_drafts
					SET status = 'DRAFT', updated_at = $2
					WHERE planning_intake_draft_id = $1 AND status = 'SUBMITTED'
				`, command.DraftID, service.clock().UTC()); err != nil {
					return intakeCommandResult[IntakeDraft]{}, err
				}
			}
			canonicalValues, err := canonicalValuesMap(command.Values, command.DraftID)
			if err != nil {
				return intakeCommandResult[IntakeDraft]{}, application.ErrInvalid
			}
			facts, err := application.ValidateCanonicalScopeMap(ctx, transaction, actor, canonicalValues, false)
			if err != nil {
				return intakeCommandResult[IntakeDraft]{}, err
			}
			if facts.ScopeID != "" {
				if err := application.ValidateCanonicalScopeDraft(ctx, transaction, command.DraftID, facts); err != nil {
					return intakeCommandResult[IntakeDraft]{}, err
				}
				// The canonical scope aggregate is the authority for the exact
				// selection and its server-derived summary. Never persist a client
				// supplied resource estimate or distribution.
				command.Values.CatalogVersion = facts.CatalogVersion
				command.Values.ScopeDraftID = current.ScopeDraftID
				command.Values.SelectionDigest = current.SelectionDigest
				command.Values.SelectedQuestionVersionIDs = append([]string(nil), current.SelectedQuestionVersionIDs...)
				formDistribution, domainDistribution, resourceRequirement, summaryErr := canonicalSelectionSummary(ctx, transaction, facts.CatalogVersion, current.SelectedQuestionVersionIDs)
				if summaryErr != nil {
					return intakeCommandResult[IntakeDraft]{}, summaryErr
				}
				command.Values.FormDistribution = formDistribution
				command.Values.DomainDistribution = domainDistribution
				command.Values.EstimatedResourceRequirement = resourceRequirement
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
			draft, err := getIntakeDraft(ctx, transaction, command.DraftID, true, actor.SubjectID)
			if err != nil {
				return intakeCommandResult[SubmitIntakeResult]{}, err
			}
			if draft.Revision != command.ExpectedRevision ||
				validateDraftValues(draft.IntakeDraftValues, true) != nil {
				return intakeCommandResult[SubmitIntakeResult]{}, application.ErrConflict
			}
			canonicalValues, err := canonicalValuesMap(draft.IntakeDraftValues, draft.ID)
			if err != nil {
				return intakeCommandResult[SubmitIntakeResult]{}, application.ErrInvalid
			}
			facts, err := application.ValidateCanonicalScopeMap(ctx, transaction, actor, canonicalValues, true)
			if err != nil {
				return intakeCommandResult[SubmitIntakeResult]{}, err
			}
			if facts.ScopeID != "" {
				if err := application.ValidateCanonicalScopeDraft(ctx, transaction, command.DraftID, facts); err != nil {
					return intakeCommandResult[SubmitIntakeResult]{}, err
				}
				formDistribution, domainDistribution, resourceRequirement, summaryErr := canonicalSelectionSummary(ctx, transaction, facts.CatalogVersion, draft.SelectedQuestionVersionIDs)
				if summaryErr != nil {
					return intakeCommandResult[SubmitIntakeResult]{}, summaryErr
				}
				draft.CatalogVersion = facts.CatalogVersion
				draft.ScopeDraftID = facts.ScopeID
				draft.SelectionDigest = facts.SelectionDigest
				draft.FormDistribution = formDistribution
				draft.DomainDistribution = domainDistribution
				draft.EstimatedResourceRequirement = resourceRequirement
				valuesJSON, marshalErr := json.Marshal(draft.IntakeDraftValues)
				if marshalErr != nil {
					return intakeCommandResult[SubmitIntakeResult]{}, marshalErr
				}
				if _, err := transaction.Exec(ctx, `
					UPDATE planning_intake_drafts
					SET values = $2, updated_at = $3
					WHERE id = $1 AND revision = $4 AND tombstoned_at IS NULL
				`, command.DraftID, valuesJSON, service.clock().UTC(), command.ExpectedRevision); err != nil {
					return intakeCommandResult[SubmitIntakeResult]{}, err
				}
				if err := persistSubmittedCanonicalScope(ctx, transaction, actor, draft, facts, service.clock().UTC()); err != nil {
					return intakeCommandResult[SubmitIntakeResult]{}, err
				}
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
	ownerSubjectID string,
) (IntakeDraft, error) {
	query := `
		SELECT draft.id, draft.values, draft.submitted_planning_item_id,
		       draft.revision, draft.updated_at, organization.legal_name
		FROM planning_intake_drafts draft
		JOIN organizations organization ON organization.id = draft.organization_id
		WHERE draft.id = $1
		  AND ($2 = '' OR draft.created_by_subject_id = $2)
		  AND draft.tombstoned_at IS NULL
		  AND organization.tombstoned_at IS NULL
	`
	if lock {
		query += " FOR UPDATE OF draft"
	}
	var output IntakeDraft
	var valuesJSON []byte
	var organizationName string
	if err := querier.QueryRow(ctx, query, draftID, ownerSubjectID).Scan(
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
	// Selection commits are owned by the canonical scope aggregate, not by
	// the mutable planning JSON. Overlay the latest non-preview receipt so a
	// refresh between selection and Save/Submit cannot resurrect an empty or
	// stale question set.
	var canonicalScopeID, canonicalDigest string
	var selectedJSON []byte
	if err := querier.QueryRow(ctx, `
		SELECT scope.id, scope.selection_digest, catalog.catalog_version,
	       COALESCE((
		   SELECT jsonb_agg(selected.question_version_id ORDER BY selected.position)
		   FROM canonical_audit_scope_selection_operations latest
		   JOIN canonical_audit_scope_selection_questions selected
		     ON selected.operation_id = latest.id
		   WHERE latest.id = (
			   SELECT latest_operation.id
			   FROM canonical_audit_scope_selection_operations latest_operation
			   WHERE latest_operation.scope_draft_id = scope.id
			     AND latest_operation.operation_kind <> 'PREVIEW'
			   ORDER BY latest_operation.created_at DESC, latest_operation.id DESC
			   LIMIT 1
		   )
	       ), '[]'::jsonb)
		FROM canonical_audit_scope_drafts scope
		JOIN canonical_question_catalogs catalog ON catalog.id = scope.catalog_id
		WHERE scope.planning_intake_draft_id = $1
		  AND scope.status IN ('DRAFT', 'SUBMITTED', 'RELEASED')
		ORDER BY scope.updated_at DESC, scope.id DESC
		LIMIT 1
	`, draftID).Scan(&canonicalScopeID, &canonicalDigest, &output.CatalogVersion, &selectedJSON); err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return IntakeDraft{}, err
	} else if err == nil {
		var selected []string
		if err := json.Unmarshal(selectedJSON, &selected); err != nil {
			return IntakeDraft{}, err
		}
		output.SelectionDigest = strings.TrimSpace(canonicalDigest)
		output.SelectedQuestionVersionIDs = selected
		output.ScopeDraftID = strings.TrimSpace(canonicalScopeID)
		if queryer, ok := querier.(interface {
			Query(context.Context, string, ...any) (pgx.Rows, error)
		}); ok {
			forms, domains, resourceRequirement, summaryErr := canonicalSelectionSummary(ctx, queryer, output.CatalogVersion, selected)
			if summaryErr != nil {
				return IntakeDraft{}, summaryErr
			}
			output.FormDistribution = forms
			output.DomainDistribution = domains
			output.EstimatedResourceRequirement = resourceRequirement
		}
	}
	return output, nil
}

// canonicalSelectionSummary derives the Planning selection receipt only from
// immutable catalog memberships. It deliberately accepts arbitrary catalog
// labels (including spaces) because labels are data, not identifiers.
func canonicalSelectionSummary(
	ctx context.Context,
	querier interface {
		Query(context.Context, string, ...any) (pgx.Rows, error)
	},
	catalogVersion string,
	ids []string,
) (map[string]any, map[string]any, float64, error) {
	forms := make(map[string]any)
	domains := make(map[string]any)
	if len(ids) == 0 {
		return forms, domains, 0, nil
	}
	rows, err := querier.Query(ctx, `
		SELECT membership.form_code,
		       COALESCE(NULLIF(membership.proposed_domain, ''), 'Unclassified')
		FROM canonical_question_catalogs catalog
		JOIN canonical_question_catalog_memberships membership
		  ON membership.catalog_id = catalog.id
		WHERE catalog.catalog_version = $1
		  AND catalog.status = 'SEALED'
		  AND catalog.source_origin = 'IMPORTED_APPROVED_SOURCE'
		  AND membership.question_version_id = ANY($2::text[])
		ORDER BY membership.ordinal, membership.question_version_id
	`, catalogVersion, ids)
	if err != nil {
		return nil, nil, 0, err
	}
	defer rows.Close()
	var count int
	for rows.Next() {
		var formCode, domain string
		if err := rows.Scan(&formCode, &domain); err != nil {
			return nil, nil, 0, err
		}
		forms[formCode] = int64MapValue(forms[formCode]) + 1
		domains[domain] = int64MapValue(domains[domain]) + 1
		count++
	}
	if err := rows.Err(); err != nil {
		return nil, nil, 0, err
	}
	if count != len(ids) {
		return nil, nil, 0, fmt.Errorf("%w: server selection summary did not resolve every immutable question", application.ErrConflict)
	}
	return forms, domains, float64(count), nil
}

func int64MapValue(value any) int64 {
	switch typed := value.(type) {
	case int64:
		return typed
	case int:
		return int64(typed)
	case float64:
		return int64(typed)
	default:
		return 0
	}
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

func canonicalValuesMap(values IntakeDraftValues, draftID string) (map[string]any, error) {
	encoded, err := json.Marshal(values)
	if err != nil {
		return nil, err
	}
	var output map[string]any
	if err := json.Unmarshal(encoded, &output); err != nil {
		return nil, err
	}
	return application.NormalizeCanonicalPlanningValues(output, draftID), nil
}

func persistSubmittedCanonicalScope(
	ctx context.Context,
	tx pgx.Tx,
	actor identity.Principal,
	draft IntakeDraft,
	facts application.CanonicalScopeFacts,
	now time.Time,
) error {
	ids := append([]string(nil), draft.SelectedQuestionVersionIDs...)
	if len(ids) == 0 || facts.SelectionDigest == "" {
		return application.ErrInvalid
	}
	var protectedOmissions int64
	if err := tx.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM canonical_question_catalog_ai_enrichments enrichment
		WHERE enrichment.catalog_id=$1
		  AND enrichment.mandatory_control=true
		  AND NOT (enrichment.question_version_id = ANY($2::text[]))
	`, facts.CatalogID, ids).Scan(&protectedOmissions); err != nil {
		return err
	}
	if protectedOmissions > 0 {
		return fmt.Errorf("%w: submitted scope omits a mandatory server-protected question", application.ErrConflict)
	}
	snapshotID := "scope-snapshot:" + draft.ID + ":submitted:" + strconv.FormatInt(draft.Revision, 10)
	snapshot, err := json.Marshal(map[string]any{
		"draftId":                      draft.ID,
		"organizationId":               draft.OrganizationID,
		"organizationName":             draft.OrganizationName,
		"applicationType":              draft.ApplicationType,
		"domain":                       draft.Domain,
		"inspectionCategory":           draft.InspectionCategory,
		"purpose":                      draft.Purpose,
		"triggerType":                  draft.TriggerType,
		"riskCategory":                 draft.RiskCategory,
		"plannedDate":                  draft.PlannedDate,
		"mode":                         draft.Mode,
		"location":                     draft.Location,
		"providerScopeId":              facts.ProviderScopeID,
		"regulatedTargetId":            facts.RegulatedTargetID,
		"catalogVersion":               facts.CatalogVersion,
		"usageClass":                   facts.UsageClass,
		"selectionDigest":              facts.SelectionDigest,
		"selectedQuestionVersionIds":   ids,
		"estimatedResourceRequirement": draft.EstimatedResourceRequirement,
		"formDistribution":             draft.FormDistribution,
		"domainDistribution":           draft.DomainDistribution,
		"requestedBudget":              draft.RequestedBudget,
		"currency":                     draft.Currency,
		"noticePolicy":                 draft.NoticePolicy,
	})
	if err != nil {
		return err
	}
	digestBytes := sha256.Sum256(snapshot)
	planningSnapshotDigest := "sha256:" + hex.EncodeToString(digestBytes[:])
	// Canonical selection digests intentionally remain bare in the scope
	// contract. Append-only recommendation receipts use the governed digest
	// format enforced by their database checks.
	governedSelectionDigest := "sha256:" + facts.SelectionDigest
	// Freeze the server-evaluated recommendation boundary alongside the
	// immutable submitted scope. The current catalog projection may have no
	// omission candidates, but the append-only receipt still binds the exact
	// evaluation/snapshot/deviation/freeze digests used by later stages.
	recommendationEvaluationID := "prior-audit-evaluation:" + facts.ScopeID + ":" + strconv.FormatInt(draft.Revision, 10)
	if _, err := tx.Exec(ctx, `
		INSERT INTO prior_audit_recommendation_evaluations (
			evaluation_id, organization_id, provider_scope_root_id, provider_scope_id,
			regulated_target_id, location, audit_type, catalog_version, usage_class,
			evaluation_as_of, history_window_months, comparable_audit_ids,
			recommendation_snapshot_digest, created_by_subject_id, created_at
		) VALUES ($1,$2,'',$3,$4,$5,$6,$7,$8,$9,36,'[]'::jsonb,$10,$11,$9)
		ON CONFLICT (evaluation_id) DO NOTHING
	`, recommendationEvaluationID, draft.OrganizationID, facts.ProviderScopeID, facts.RegulatedTargetID,
		draft.Location, draft.ApplicationType, facts.CatalogVersion, facts.UsageClass,
		now, governedSelectionDigest, actor.SubjectID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO canonical_audit_question_recommendation_snapshots (
			evaluation_id, question_version_id, recommendation_state, classification,
			included_by_default, can_defer, history_count, comparable_audit_count,
			last_comparable_result, last_comparable_audit_id, signal_codes, rationale,
			guardrails, snapshot_digest, created_at
		)
		SELECT $1, enrichment.question_version_id,
			'SUGGESTED_NOW',
			CASE WHEN enrichment.mandatory_control OR enrichment.safety_critical OR enrichment.risk_tier='HIGH' THEN 'MANDATORY_CORE' ELSE 'ROTATIONAL_SAMPLE' END,
			true, false, 0, 0, NULL, NULL,
			'["MANDATORY_FLOOR_ENFORCED"]'::jsonb,
			'Server-evaluated recommendation snapshot retained with the frozen Planning scope.',
			'["MANDATORY_FLOOR_ENFORCED","FULL_CATALOG_OVERRIDE_ALLOWED"]'::jsonb,
			$2, $3
		FROM canonical_question_catalog_ai_enrichments enrichment
		WHERE enrichment.catalog_id=$4 AND enrichment.question_version_id=ANY($5::text[])
		ON CONFLICT (evaluation_id, question_version_id) DO NOTHING
	`, recommendationEvaluationID, governedSelectionDigest, now, facts.CatalogID, ids); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO canonical_audit_scope_freezes (
			freeze_id, scope_draft_id, evaluation_id, recommendation_snapshot_digest,
			deviation_digest, selection_digest, freeze_digest,
			selected_question_version_ids, created_by_subject_id, created_at
		) VALUES ($1,$2,$3,$4,$4,$5,$4,$6,$7,$8)
		ON CONFLICT (freeze_id) DO NOTHING
	`, "prior-audit-freeze:"+facts.ScopeID+":"+strconv.FormatInt(draft.Revision, 10), facts.ScopeID,
		recommendationEvaluationID, governedSelectionDigest, facts.SelectionDigest, ids, actor.SubjectID, now); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
			INSERT INTO canonical_audit_scope_snapshots (
				id, scope_draft_id, revision, stage, catalog_id, usage_class,
				catalog_root_digest,
				selection_digest, planning_snapshot_digest, selected_question_count, snapshot,
				created_by_subject_id, created_at
			) VALUES ($1, $2, $3, 'SUBMITTED', $4, $5, $6, $7, $8, $9, $10, $11, $12)
		`, snapshotID, facts.ScopeID, draft.Revision, facts.CatalogID, facts.UsageClass,
		facts.CatalogRootDigest, facts.SelectionDigest, planningSnapshotDigest, len(ids), snapshot, actor.SubjectID, now); err != nil {
		return err
	}
	for position, questionVersionID := range ids {
		if _, err := tx.Exec(ctx, `
			INSERT INTO canonical_audit_scope_snapshot_questions (
				snapshot_id, catalog_id, question_version_id, position
			) VALUES ($1, $2, $3, $4)
		`, snapshotID, facts.CatalogID, questionVersionID, position); err != nil {
			return err
		}
	}
	if _, err := tx.Exec(ctx, `
		UPDATE canonical_audit_scope_drafts
		SET status = 'SUBMITTED', updated_at = $2
		WHERE id = $1 AND status = 'DRAFT'
	`, facts.ScopeID, now); err != nil {
		return err
	}
	return nil
}

func validateDraftValues(values IntakeDraftValues, complete bool) error {
	if values.OrganizationID == "" || values.RequestedBudget < 0 {
		return application.ErrInvalid
	}
	if values.InspectionCategory != InspectionCategoryRoutine &&
		values.InspectionCategory != InspectionCategoryAdHoc {
		return application.ErrInvalid
	}
	if values.PlannedDate != "" {
		if _, err := time.Parse("2006-01-02", values.PlannedDate); err != nil {
			return application.ErrInvalid
		}
	} else if complete {
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
		// The successor New Audit route is catalog/scope-only. A checklist
		// template identifier is never a valid pre-approval source authority.
		canonicalScope := values.CatalogVersion != "" && values.ScopeDraftID != "" && values.SelectionDigest != "" && len(values.SelectedQuestionVersionIDs) > 0
		if values.TemplateVersionID != "" || values.Scope != "" || !canonicalScope {
			return application.ErrInvalid
		}
	}
	return nil
}

func planningCommandIdempotencyKey(scope, key string) string {
	return "command:" + scope + ":idempotency:" + key
}
