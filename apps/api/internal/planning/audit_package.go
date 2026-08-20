package planning

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/aviason/aviaSurveil/internal/application"
	"github.com/aviason/aviaSurveil/internal/identity"
	"github.com/aviason/aviaSurveil/internal/platform/database"
	"github.com/jackc/pgx/v5"
)

type AuditPackageSetupStatus string

const (
	AuditPackageSetupDraft              AuditPackageSetupStatus = "DRAFT"
	AuditPackageSetupSelectionConfirmed AuditPackageSetupStatus = "SELECTION_CONFIRMED"
	AuditPackageSetupFinalized          AuditPackageSetupStatus = "FINALIZED"
)

type AuditPackageSetup struct {
	PlanningItemID               string                  `json:"planningItemId"`
	PlanningSnapshotID           string                  `json:"planningSnapshotId"`
	PlanningSnapshotDigest       string                  `json:"planningSnapshotDigest"`
	ScopeDraftID                 string                  `json:"scopeDraftId"`
	Status                       AuditPackageSetupStatus `json:"status"`
	Revision                     int64                   `json:"revision"`
	CatalogVersion               string                  `json:"catalogVersion"`
	CatalogRootDigest            string                  `json:"catalogRootDigest"`
	SelectedCount                int64                   `json:"selectedCount"`
	SelectionDigest              string                  `json:"selectionDigest"`
	ApprovedChecklistItemCeiling int64                   `json:"approvedChecklistItemCeiling"`
	NextAction                   string                  `json:"nextAction"`
}

type EnsureAuditPackageSetupCommand struct {
	OperationID              string
	IdempotencyKey           string
	PlanningItemID           string
	ExpectedPlanningRevision int64
}

type FinalizeAuditPackageCommand struct {
	OperationID              string
	IdempotencyKey           string
	PlanningItemID           string
	ExpectedPlanningRevision int64
	ExpectedSetupRevision    int64
	ExpectedSelectionDigest  string
}

type proposalSnapshotFacts struct {
	ProviderScopeID             string `json:"providerScopeId"`
	RegulatedTargetID           string `json:"regulatedTargetId"`
	InspectionType              string `json:"inspectionType"`
	EstimatedChecklistItemCount int64  `json:"estimatedChecklistItemCount"`
}

func (service *Service) EnsureAuditPackageSetup(ctx context.Context, actor identity.Principal, command EnsureAuditPackageSetupCommand) (AuditPackageSetup, error) {
	if !actor.HasRole(identity.RoleDepartmentManager) {
		return AuditPackageSetup{}, fmt.Errorf("%w: Department Manager authority is required", application.ErrForbidden)
	}
	if strings.TrimSpace(command.OperationID) == "" || strings.TrimSpace(command.IdempotencyKey) == "" || strings.TrimSpace(command.PlanningItemID) == "" || command.ExpectedPlanningRevision <= 0 {
		return AuditPackageSetup{}, application.ErrInvalid
	}
	var output AuditPackageSetup
	err := database.WithinTransaction(ctx, service.pool, func(ctx context.Context, tx pgx.Tx) error {
		var itemRevision int64
		var itemStatus, organizationID string
		var estimatedBudget float64
		if err := tx.QueryRow(ctx, `
			SELECT revision, status, organization_id, estimated_budget::float8
			FROM surveillance_plan_items
			WHERE id=$1 AND tombstoned_at IS NULL
			FOR UPDATE`, command.PlanningItemID).Scan(&itemRevision, &itemStatus, &organizationID, &estimatedBudget); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return application.ErrNotFound
			}
			return err
		}
		if itemStatus != string(StatusReleased) || itemRevision != command.ExpectedPlanningRevision {
			return fmt.Errorf("%w: Planning must be RELEASED at the expected revision before Audit-package setup", application.ErrConflict)
		}
		var snapshotID, snapshotDigest string
		var snapshot []byte
		if err := tx.QueryRow(ctx, `
			SELECT id, snapshot_digest, snapshot
			FROM planning_proposal_snapshots
			WHERE planning_item_id=$1
			ORDER BY revision DESC, id DESC
			LIMIT 1`, command.PlanningItemID).Scan(&snapshotID, &snapshotDigest, &snapshot); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return fmt.Errorf("%w: immutable Planning proposal snapshot is required", application.ErrConflict)
			}
			return err
		}
		var facts proposalSnapshotFacts
		if err := json.Unmarshal(snapshot, &facts); err != nil {
			return err
		}
		if facts.ProviderScopeID == "" || facts.RegulatedTargetID == "" || facts.InspectionType == "" || facts.EstimatedChecklistItemCount <= 0 {
			return fmt.Errorf("%w: Planning snapshot does not contain complete package scope facts", application.ErrConflict)
		}
		canonicalType, err := application.CanonicalExecutionType(facts.InspectionType)
		if err != nil {
			return err
		}
		var catalogID, catalogVersion, catalogRootDigest string
		if err := tx.QueryRow(ctx, `
			SELECT id, catalog_version, catalog_root_digest
			FROM canonical_question_catalogs
			WHERE usage_class='GOVERNED_OPERATIONAL'
			  AND status='SEALED'
			  AND source_origin='IMPORTED_APPROVED_SOURCE'
			ORDER BY created_at DESC, id DESC
			LIMIT 1`).Scan(&catalogID, &catalogVersion, &catalogRootDigest); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return fmt.Errorf("%w: governed catalog is unavailable for post-release setup", application.ErrConflict)
			}
			return err
		}
		var existingID string
		if err := tx.QueryRow(ctx, `
			SELECT id
			FROM canonical_audit_scope_drafts
			WHERE planning_proposal_snapshot_id=$1
		ORDER BY revision DESC, id DESC
			LIMIT 1`, snapshotID).Scan(&existingID); err == nil {
			output, err = loadAuditPackageSetup(ctx, tx, command.PlanningItemID)
			return err
		} else if !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		scopeDraftID := "audit-package-scope:" + command.PlanningItemID
		if _, err := tx.Exec(ctx, `
			INSERT INTO canonical_audit_scope_drafts (
				id, planning_intake_draft_id, planning_proposal_snapshot_id,
				organization_id, provider_scope_id, regulated_target_id, audit_type,
				catalog_id, usage_class, revision, status, selected_question_count,
				selection_digest, requested_budget, notice_policy, approved_checklist_item_ceiling,
				created_by_subject_id, created_at, updated_at
			) VALUES ($1, NULL, $2, $3, $4, $5, $6, $7, 'GOVERNED_OPERATIONAL', 1,
				'DRAFT', 0, '', $8, 'ADVANCE', $9, $10, now(), now())`,
			scopeDraftID, snapshotID, organizationID, facts.ProviderScopeID, facts.RegulatedTargetID,
			canonicalType, catalogID, estimatedBudget, facts.EstimatedChecklistItemCount, actor.SubjectID); err != nil {
			return err
		}
		output, err = loadAuditPackageSetup(ctx, tx, command.PlanningItemID)
		_ = catalogVersion
		_ = catalogRootDigest
		return err
	})
	return output, err
}

func (service *Service) GetAuditPackageSetup(ctx context.Context, actor identity.Principal, planningItemID string) (AuditPackageSetup, error) {
	if !actor.HasRole(identity.RoleDepartmentManager) {
		return AuditPackageSetup{}, fmt.Errorf("%w: Department Manager authority is required", application.ErrForbidden)
	}
	if strings.TrimSpace(planningItemID) == "" {
		return AuditPackageSetup{}, application.ErrInvalid
	}
	return loadAuditPackageSetup(ctx, service.pool, planningItemID)
}

func (service *Service) FinalizeAuditPackage(ctx context.Context, actor identity.Principal, command FinalizeAuditPackageCommand) (AuditPackageSetup, error) {
	if !actor.HasRole(identity.RoleDepartmentManager) {
		return AuditPackageSetup{}, fmt.Errorf("%w: Department Manager authority is required", application.ErrForbidden)
	}
	if strings.TrimSpace(command.OperationID) == "" || strings.TrimSpace(command.IdempotencyKey) == "" || strings.TrimSpace(command.PlanningItemID) == "" || command.ExpectedPlanningRevision <= 0 || command.ExpectedSetupRevision <= 0 {
		return AuditPackageSetup{}, application.ErrInvalid
	}
	var output AuditPackageSetup
	err := database.WithinTransaction(ctx, service.pool, func(ctx context.Context, tx pgx.Tx) error {
		var itemRevision int64
		var itemStatus string
		if err := tx.QueryRow(ctx, `SELECT revision, status FROM surveillance_plan_items WHERE id=$1 AND tombstoned_at IS NULL`, command.PlanningItemID).Scan(&itemRevision, &itemStatus); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return application.ErrNotFound
			}
			return err
		}
		if itemStatus != string(StatusReleased) || itemRevision != command.ExpectedPlanningRevision {
			return fmt.Errorf("%w: released Planning revision is stale", application.ErrConflict)
		}
		var scopeID, snapshotID, snapshotDigest, status, selectionDigest, catalogID, usageClass, catalogRootDigest string
		var setupRevision, selectedCount, ceiling int64
		var catalogVersion string
		if err := tx.QueryRow(ctx, `
			SELECT scope.id, scope.planning_proposal_snapshot_id, proposal.snapshot_digest,
			       scope.status, scope.revision, scope.selected_question_count, scope.selection_digest,
			       scope.approved_checklist_item_ceiling, scope.catalog_id, scope.usage_class,
			       catalog.catalog_version, catalog.catalog_root_digest
			FROM canonical_audit_scope_drafts scope
			JOIN planning_proposal_snapshots proposal ON proposal.id=scope.planning_proposal_snapshot_id
			JOIN canonical_question_catalogs catalog ON catalog.id=scope.catalog_id
			WHERE proposal.planning_item_id=$1
			FOR UPDATE OF scope`, command.PlanningItemID).Scan(&scopeID, &snapshotID, &snapshotDigest, &status, &setupRevision, &selectedCount, &selectionDigest, &ceiling, &catalogID, &usageClass, &catalogVersion, &catalogRootDigest); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return application.ErrNotFound
			}
			return err
		}
		if status == string(AuditPackageSetupFinalized) {
			loaded, loadErr := loadAuditPackageSetup(ctx, tx, command.PlanningItemID)
			output = loaded
			return loadErr
		}
		if setupRevision != command.ExpectedSetupRevision || selectionDigest != command.ExpectedSelectionDigest {
			return fmt.Errorf("%w: checklist selection is stale; reload the post-release setup", application.ErrConflict)
		}
		if selectedCount <= 0 {
			return fmt.Errorf("%w: at least one checklist item must be confirmed", application.ErrInvalid)
		}
		if ceiling <= 0 || selectedCount > ceiling {
			return fmt.Errorf("%w: PLANNING_AMENDMENT_REQUIRED: selected checklist items exceed the approved ceiling", application.ErrConflict)
		}
		var operationID string
		var selectedJSON []byte
		if err := tx.QueryRow(ctx, `
			SELECT id, affected_question_version_ids
			FROM canonical_audit_scope_selection_operations
			WHERE scope_draft_id=$1 AND operation_kind <> 'PREVIEW'
			ORDER BY created_at DESC, id DESC
			LIMIT 1`, scopeID).Scan(&operationID, &selectedJSON); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return fmt.Errorf("%w: confirmed checklist selection receipt is required", application.ErrConflict)
			}
			return err
		}
		var selectedIDs []string
		if err := json.Unmarshal(selectedJSON, &selectedIDs); err != nil || len(selectedIDs) != int(selectedCount) {
			return fmt.Errorf("%w: selection receipt is incomplete", application.ErrConflict)
		}
		packageSnapshotID := "audit-package-snapshot:" + command.PlanningItemID + ":final"
		packageSnapshot, err := json.Marshal(map[string]any{
			"planningItemId": command.PlanningItemID, "planningSnapshotId": snapshotID,
			"planningSnapshotDigest": snapshotDigest, "scopeDraftId": scopeID,
			"catalogId": catalogID, "catalogVersion": catalogVersion, "catalogRootDigest": catalogRootDigest,
			"usageClass": usageClass, "selectionDigest": selectionDigest,
			"selectedCount": selectedCount, "approvedChecklistItemCeiling": ceiling,
			"selectionOperationId": operationID,
		})
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO canonical_audit_scope_snapshots (
				id, scope_draft_id, revision, stage, catalog_id, usage_class,
				selection_digest, selected_question_count, snapshot, created_by_subject_id, created_at
			) VALUES ($1, $2, $3, 'RELEASED', $4, $5, $6, $7, $8, $9, now())
			ON CONFLICT (scope_draft_id, stage, revision) DO NOTHING`,
			packageSnapshotID, scopeID, setupRevision, catalogID, usageClass, selectionDigest, selectedCount, packageSnapshot, actor.SubjectID); err != nil {
			return err
		}
		for position, questionID := range selectedIDs {
			if _, err := tx.Exec(ctx, `
				INSERT INTO canonical_audit_scope_snapshot_questions (snapshot_id, catalog_id, question_version_id, position)
				VALUES ($1, $2, $3, $4) ON CONFLICT DO NOTHING`, packageSnapshotID, catalogID, questionID, position); err != nil {
				return err
			}
		}
		if _, err := tx.Exec(ctx, `UPDATE canonical_audit_scope_drafts SET status='FINALIZED', revision=revision+1, updated_at=now() WHERE id=$1 AND revision=$2`, scopeID, setupRevision); err != nil {
			return err
		}
		output, err = loadAuditPackageSetup(ctx, tx, command.PlanningItemID)
		return err
	})
	return output, err
}

type auditPackageSetupQueryer interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

func loadAuditPackageSetup(ctx context.Context, queryer auditPackageSetupQueryer, planningItemID string) (AuditPackageSetup, error) {
	var output AuditPackageSetup
	var status string
	if err := queryer.QueryRow(ctx, `
		SELECT proposal.planning_item_id, proposal.id, proposal.snapshot_digest,
		       scope.id, scope.status, scope.revision, catalog.catalog_version,
		       catalog.catalog_root_digest, scope.selected_question_count,
		       scope.selection_digest, scope.approved_checklist_item_ceiling
		FROM canonical_audit_scope_drafts scope
		JOIN planning_proposal_snapshots proposal ON proposal.id=scope.planning_proposal_snapshot_id
		JOIN canonical_question_catalogs catalog ON catalog.id=scope.catalog_id
		WHERE proposal.planning_item_id=$1
		ORDER BY scope.revision DESC, scope.id DESC
		LIMIT 1`, planningItemID).Scan(
		&output.PlanningItemID, &output.PlanningSnapshotID, &output.PlanningSnapshotDigest,
		&output.ScopeDraftID, &status, &output.Revision, &output.CatalogVersion,
		&output.CatalogRootDigest, &output.SelectedCount, &output.SelectionDigest,
		&output.ApprovedChecklistItemCeiling,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return AuditPackageSetup{}, application.ErrNotFound
		}
		return AuditPackageSetup{}, err
	}
	output.Status = AuditPackageSetupStatus(status)
	output.NextAction = "Review the governed checklist and confirm the exact Audit-package selection."
	if output.Status == AuditPackageSetupFinalized {
		output.NextAction = "Audit-package scope is finalized; proceed to Department Manager preparation."
	}
	return output, nil
}
