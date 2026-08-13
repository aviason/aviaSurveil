package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/aviason/aviaSurveil/internal/documents"
	"github.com/aviason/aviaSurveil/internal/identity"
	"github.com/aviason/aviaSurveil/internal/reports"
	"github.com/jackc/pgx/v5"
)

type CreateAdminOrganizationCommand struct {
	OperationID      string
	IdempotencyKey   string
	OrganizationID   string
	LegalName        string
	OrganizationType string
}

type AdminOrganization struct {
	ID               string `json:"id"`
	LegalName        string `json:"legalName"`
	OrganizationType string `json:"organizationType"`
	Status           string `json:"status"`
}

func (service *Service) CreateAdminOrganization(
	ctx context.Context,
	actor identity.Principal,
	command CreateAdminOrganizationCommand,
) (AdminOrganization, error) {
	if !actor.HasRole(identity.RoleAdmin) {
		return AdminOrganization{}, fmt.Errorf("%w: Admin authority is required", ErrForbidden)
	}
	command.OrganizationID = strings.TrimSpace(command.OrganizationID)
	command.LegalName = strings.TrimSpace(command.LegalName)
	command.OrganizationType = strings.TrimSpace(command.OrganizationType)
	if command.OperationID == "" || command.IdempotencyKey == "" ||
		command.OrganizationID == "" || command.LegalName == "" ||
		(command.OrganizationType != "AUTHORITY" &&
			command.OrganizationType != "OPERATOR" &&
			command.OrganizationType != "SERVICE_PROVIDER") {
		return AdminOrganization{}, ErrInvalid
	}
	return executeTransition(ctx, service, actor, commandEnvelope{
		OperationID: command.OperationID, IdempotencyKey: command.IdempotencyKey,
		CorrelationID: command.OperationID, Kind: "create_admin_organization",
		EntityID: command.OrganizationID, Semantic: command,
	}, func(ctx context.Context, transaction pgx.Tx) (transition[AdminOrganization], error) {
		now := service.clock().UTC()
		var output AdminOrganization
		if err := transaction.QueryRow(ctx, `
			INSERT INTO organizations (
				id, legal_name, organization_type, status, revision, created_at, updated_at
			) VALUES ($1, $2, $3, 'ACTIVE', 1, $4, $4)
			RETURNING id, legal_name, organization_type, status
		`, command.OrganizationID, command.LegalName, command.OrganizationType, now).Scan(
			&output.ID, &output.LegalName, &output.OrganizationType, &output.Status,
		); err != nil {
			return transition[AdminOrganization]{}, mapCreateConflict(err)
		}
		return transition[AdminOrganization]{
			Response: output, OrganizationID: command.OrganizationID,
			Action: "admin.organization_created", EntityType: "organization",
			EntityID: command.OrganizationID, EntityVersion: 1,
			AfterStatus: "ACTIVE", Reason: "Created organization master record.",
			SyncKind: "organization", OutboxTopic: "admin.organization_created",
		}, nil
	})
}

type CreatePlanningIntakeDraftCommand struct {
	OperationID    string
	IdempotencyKey string
	DraftID        string
	OrganizationID string
	Values         map[string]any
}

type PlanningIntakeDraft struct {
	ID                      string         `json:"id"`
	OrganizationID          string         `json:"organizationId"`
	Values                  map[string]any `json:"values"`
	Revision                int64          `json:"revision"`
	SubmittedPlanningItemID *string        `json:"submittedPlanningItemId"`
	UpdatedAt               string         `json:"updatedAt"`
}

func (service *Service) CreatePlanningIntakeDraft(
	ctx context.Context,
	actor identity.Principal,
	command CreatePlanningIntakeDraftCommand,
) (PlanningIntakeDraft, error) {
	if !actor.HasRole(identity.RoleDepartmentManager) {
		return PlanningIntakeDraft{}, fmt.Errorf(
			"%w: Department Manager authority is required",
			ErrForbidden,
		)
	}
	command.DraftID = strings.TrimSpace(command.DraftID)
	if command.DraftID == "" {
		digest := sha256.Sum256([]byte("planning-intake:" + command.OperationID))
		command.DraftID = "draft-planning-" + hex.EncodeToString(digest[:])[:24]
	}
	command.OrganizationID = strings.TrimSpace(command.OrganizationID)
	if command.OperationID == "" || command.IdempotencyKey == "" ||
		command.DraftID == "" || command.OrganizationID == "" ||
		len(command.Values) == 0 ||
		strings.TrimSpace(stringValue(command.Values["organizationId"])) != command.OrganizationID {
		return PlanningIntakeDraft{}, ErrInvalid
	}
	// New Audit is the canonical catalog/scope intake boundary. Legacy
	// checklist-template and free-form scope fields are intentionally rejected
	// before any draft row is created.
	if strings.TrimSpace(stringValue(command.Values["catalogVersion"])) == "" ||
		strings.TrimSpace(stringValue(command.Values["providerScopeId"])) == "" ||
		strings.TrimSpace(stringValue(command.Values["regulatedTargetId"])) == "" ||
		strings.TrimSpace(stringValue(command.Values["templateVersionId"])) != "" ||
		strings.TrimSpace(stringValue(command.Values["scope"])) != "" {
		return PlanningIntakeDraft{}, fmt.Errorf("%w: New Audit requires a canonical catalog and authorized scope", ErrInvalid)
	}
	return executeTransition(ctx, service, actor, commandEnvelope{
		OperationID: command.OperationID, IdempotencyKey: command.IdempotencyKey,
		CorrelationID: command.OperationID, Kind: "create_planning_intake_draft",
		EntityID: command.DraftID, Semantic: command,
	}, func(ctx context.Context, transaction pgx.Tx) (transition[PlanningIntakeDraft], error) {
		canonicalValues := NormalizeCanonicalPlanningValues(command.Values, command.DraftID)
		canonicalValues["organizationId"] = command.OrganizationID
		if canonicalValues["catalogVersion"] != "" || canonicalValues["scopeDraftId"] != "" || canonicalValues["providerScopeId"] != "" || canonicalValues["regulatedTargetId"] != "" {
			if canonicalValues["catalogVersion"] == "" {
				return transition[PlanningIntakeDraft]{}, fmt.Errorf("%w: canonical catalog identity is required when a scope is supplied", ErrInvalid)
			}
			if strings.TrimSpace(stringValue(command.Values["scopeDraftId"])) == "" {
				command.Values["scopeDraftId"] = canonicalValues["scopeDraftId"]
			}
			if _, err := ValidateCanonicalScopeMap(ctx, transaction, actor, canonicalValues, false); err != nil {
				return transition[PlanningIntakeDraft]{}, err
			}
		}
		var legalName string
		if err := transaction.QueryRow(ctx, `
			SELECT legal_name FROM organizations
			WHERE id = $1 AND tombstoned_at IS NULL
		`, command.OrganizationID).Scan(&legalName); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return transition[PlanningIntakeDraft]{}, ErrNotFound
			}
			return transition[PlanningIntakeDraft]{}, err
		}
		command.Values["organizationName"] = legalName
		values, err := json.Marshal(command.Values)
		if err != nil {
			return transition[PlanningIntakeDraft]{}, ErrInvalid
		}
		now := service.clock().UTC()
		var output PlanningIntakeDraft
		var outputValues []byte
		var updatedAt time.Time
		if err := transaction.QueryRow(ctx, `
			INSERT INTO planning_intake_drafts (
				id, organization_id, values, submitted_planning_item_id,
				revision, created_by_subject_id, created_at, updated_at
			) VALUES ($1, $2, $3, NULL, 1, $4, $5, $5)
			RETURNING id, organization_id, values, revision,
			          submitted_planning_item_id, updated_at
		`, command.DraftID, command.OrganizationID, values, actor.SubjectID, now).Scan(
			&output.ID, &output.OrganizationID, &outputValues, &output.Revision,
			&output.SubmittedPlanningItemID, &updatedAt,
		); err != nil {
			return transition[PlanningIntakeDraft]{}, mapCreateConflict(err)
		}
		if err := json.Unmarshal(outputValues, &output.Values); err != nil {
			return transition[PlanningIntakeDraft]{}, err
		}
		if canonicalValues["catalogVersion"] != "" {
			facts, err := ValidateCanonicalScopeMap(ctx, transaction, actor, canonicalValues, false)
			if err != nil {
				return transition[PlanningIntakeDraft]{}, err
			}
			if _, err := transaction.Exec(ctx, `
				INSERT INTO canonical_audit_scope_drafts (
					id, planning_intake_draft_id, organization_id, provider_scope_id,
					regulated_target_id, audit_type, catalog_id, usage_class, revision,
					status, selected_question_count, selection_digest, requested_budget,
					notice_policy, created_by_subject_id, created_at, updated_at
				) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, 1, 'DRAFT', 0, '', $9, $10, $11, $12, $12)
			`, facts.ScopeID, output.ID, stringValue(canonicalValues["organizationId"]), facts.ProviderScopeID,
				facts.RegulatedTargetID, stringValue(canonicalValues["applicationType"]), facts.CatalogID,
				facts.UsageClass, numberValue(canonicalValues["requestedBudget"]),
				stringValue(canonicalValues["noticePolicy"]), actor.SubjectID, now); err != nil {
				return transition[PlanningIntakeDraft]{}, mapCreateConflict(err)
			}
			// The scope draft identity is server-owned. Return it in the
			// authoritative draft projection so the first Step 1 save and a
			// browser refresh cannot lose the canonical aggregate binding.
			output.Values["scopeDraftId"] = facts.ScopeID
		}
		output.UpdatedAt = updatedAt.UTC().Format(time.RFC3339Nano)
		return transition[PlanningIntakeDraft]{
			Response: output, OrganizationID: command.OrganizationID,
			Action: "planning.intake_created", EntityType: "planning_intake_draft",
			EntityID: command.DraftID, EntityVersion: 1, AfterStatus: "DRAFT",
			Reason:      "Created planning intake draft.",
			SyncKind:    "planning_intake_draft",
			OutboxTopic: "planning.intake_created",
		}, nil
	})
}

func stringValue(value any) string {
	text, _ := value.(string)
	return text
}

func mapCreateConflict(err error) error {
	if err == nil {
		return nil
	}
	if strings.Contains(err.Error(), "duplicate key") {
		return ErrConflict
	}
	return err
}

type CreateReminderRuleCommand struct {
	OperationID    string
	IdempotencyKey string
	RuleID         string
	Label          string
	OffsetDays     int64
	Channel        string
}

type ReminderRule struct {
	ID         string `json:"id"`
	Label      string `json:"label"`
	OffsetDays int64  `json:"offsetDays"`
	Channel    string `json:"channel"`
	Status     string `json:"status"`
	Revision   int64  `json:"revision"`
}

func (service *Service) CreateReminderRule(
	ctx context.Context,
	actor identity.Principal,
	command CreateReminderRuleCommand,
) (ReminderRule, error) {
	if !actor.HasRole(identity.RoleAdmin) {
		return ReminderRule{}, fmt.Errorf("%w: Admin authority is required", ErrForbidden)
	}
	if command.OperationID == "" || command.IdempotencyKey == "" ||
		strings.TrimSpace(command.RuleID) == "" ||
		strings.TrimSpace(command.Label) == "" ||
		(command.Channel != "IN_APP" && command.Channel != "EMAIL" &&
			command.Channel != "IN_APP_AND_EMAIL") {
		return ReminderRule{}, ErrInvalid
	}
	return executeTransition(ctx, service, actor, commandEnvelope{
		OperationID: command.OperationID, IdempotencyKey: command.IdempotencyKey,
		CorrelationID: command.OperationID, Kind: "create_reminder_rule",
		EntityID: command.RuleID, Semantic: command,
	}, func(ctx context.Context, transaction pgx.Tx) (transition[ReminderRule], error) {
		now := service.clock().UTC()
		output := ReminderRule{
			ID: command.RuleID, Label: command.Label, OffsetDays: command.OffsetDays,
			Channel: command.Channel, Status: "ACTIVE", Revision: 1,
		}
		if _, err := transaction.Exec(ctx, `
			INSERT INTO reminder_rules (
				id, label, offset_days, channel, status, revision, created_at, updated_at
			) VALUES ($1, $2, $3, $4, 'ACTIVE', 1, $5, $5)
		`, output.ID, output.Label, output.OffsetDays, output.Channel, now); err != nil {
			return transition[ReminderRule]{}, mapCreateConflict(err)
		}
		return transition[ReminderRule]{
			Response: output, OrganizationID: actor.OrganizationID,
			Action: "admin.reminder_rule_created", EntityType: "reminder_rule",
			EntityID: output.ID, EntityVersion: 1, AfterStatus: "ACTIVE",
			Reason: "Created reminder rule.", SyncKind: "reminder_rule",
			OutboxTopic: "admin.reminder_rule_created",
		}, nil
	})
}

type CreateReportVersionCommand struct {
	OperationID     string
	IdempotencyKey  string
	ReportVersionID string
	ReportID        string
	AuditID         string
	Kind            reports.Kind
	Version         int64
	Status          string
	FindingIDs      []string
	Content         map[string]any
}

type CreatedReportVersion struct {
	ReportVersionID string   `json:"reportVersionId"`
	ReportID        string   `json:"reportId"`
	OrganizationID  string   `json:"organizationId"`
	AuditID         string   `json:"auditId"`
	FindingIDs      []string `json:"findingIds"`
	ContentHash     string   `json:"contentHash"`
	Version         int64    `json:"version"`
	Status          string   `json:"status"`
	Revision        int64    `json:"revision"`
	IssuedAt        *string  `json:"issuedAt"`
}

func (service *Service) CreateReportVersion(
	ctx context.Context,
	actor identity.Principal,
	command CreateReportVersionCommand,
) (CreatedReportVersion, error) {
	if !actor.HasRole(identity.RoleLeadInspector) {
		return CreatedReportVersion{}, fmt.Errorf(
			"%w: Lead Inspector authority is required",
			ErrForbidden,
		)
	}
	if command.OperationID == "" || command.IdempotencyKey == "" ||
		command.ReportVersionID == "" || command.ReportID == "" ||
		command.AuditID == "" || command.Version <= 0 ||
		(command.Kind != reports.KindPreliminary && command.Kind != reports.KindFinal) ||
		(command.Status != "RETURNED" && command.Status != "DEPARTMENT_REVIEW") {
		return CreatedReportVersion{}, ErrInvalid
	}
	var err error
	command.FindingIDs, err = normalizedUniqueIDs(command.FindingIDs)
	if err != nil {
		return CreatedReportVersion{}, fmt.Errorf("%w: Finding IDs must be unique", ErrInvalid)
	}
	return executeTransition(ctx, service, actor, commandEnvelope{
		OperationID: command.OperationID, IdempotencyKey: command.IdempotencyKey,
		CorrelationID: command.OperationID, Kind: "create_report_version",
		EntityID: command.ReportVersionID, Semantic: command,
	}, func(ctx context.Context, transaction pgx.Tx) (transition[CreatedReportVersion], error) {
		var organizationID string
		if err := transaction.QueryRow(ctx, `
			SELECT organization_id FROM inspections
			WHERE id = $1 AND tombstoned_at IS NULL
		`, command.AuditID).Scan(&organizationID); err != nil {
			if err == pgx.ErrNoRows {
				return transition[CreatedReportVersion]{}, ErrNotFound
			}
			return transition[CreatedReportVersion]{}, err
		}
		// For canonical audits, the Lead Inspector recorded on the assignment
		// is the only authority for report version creation. Assignment-less
		// records fail closed instead of falling back to a global role.
		var hasAssignment, isLead bool
		if err := transaction.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM audit_assignments
				WHERE inspection_id = $1 AND tombstoned_at IS NULL
			), EXISTS (
				SELECT 1
				FROM audit_assignments assignment
				JOIN audit_team_members member ON member.assignment_id = assignment.id
				WHERE assignment.inspection_id = $1
				  AND assignment.tombstoned_at IS NULL
				  AND assignment.lead_subject_id = $2
				  AND member.subject_id = $2
				  AND member.member_role = 'LEAD_INSPECTOR'
				  AND member.removed_at IS NULL
			)
		`, command.AuditID, actor.SubjectID).Scan(&hasAssignment, &isLead); err != nil {
			return transition[CreatedReportVersion]{}, err
		}
		if !hasAssignment {
			return transition[CreatedReportVersion]{}, fmt.Errorf("%w: report requires a canonical Audit assignment", ErrForbidden)
		}
		if !isLead {
			return transition[CreatedReportVersion]{}, fmt.Errorf("%w: report authority belongs to the assigned Lead Inspector", ErrForbidden)
		}
		if command.Kind == reports.KindPreliminary {
			var checklistStatus string
			if err := transaction.QueryRow(ctx, `
				SELECT status FROM inspection_checklists WHERE inspection_id = $1
			`, command.AuditID).Scan(&checklistStatus); err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					return transition[CreatedReportVersion]{}, fmt.Errorf("%w: checklist execution is required before Preliminary Report", ErrConflict)
				}
				return transition[CreatedReportVersion]{}, err
			}
			if checklistStatus != "SUBMITTED" {
				return transition[CreatedReportVersion]{}, fmt.Errorf("%w: checklist must be submitted before Preliminary Report", ErrConflict)
			}
		}
		if command.Kind == reports.KindFinal {
			var preliminaryIssued bool
			if err := transaction.QueryRow(ctx, `
				SELECT EXISTS (
					SELECT 1
					FROM report_versions preliminary
					JOIN report_approval_states approval ON approval.report_version_id = preliminary.id
					WHERE preliminary.inspection_id = $1
					  AND preliminary.snapshot->>'kind' = 'PRELIMINARY'
					  AND approval.status IN ('ISSUED', 'LOCKED')
				)
			`, command.AuditID).Scan(&preliminaryIssued); err != nil {
				return transition[CreatedReportVersion]{}, err
			}
			if !preliminaryIssued {
				return transition[CreatedReportVersion]{}, fmt.Errorf("%w: Preliminary Report must be approved and issued before Final Report", ErrConflict)
			}
			var openFindingCount int64
			if err := transaction.QueryRow(ctx, `SELECT COUNT(*) FROM findings WHERE inspection_id = $1 AND status <> 'CLOSED'`, command.AuditID).Scan(&openFindingCount); err != nil {
				return transition[CreatedReportVersion]{}, err
			}
			if openFindingCount != 0 {
				return transition[CreatedReportVersion]{}, fmt.Errorf("%w: all Findings must be closed before Final Report", ErrConflict)
			}
		}
		var auditFindingCount, linkedFindingCount int
		if err := transaction.QueryRow(ctx, `
			SELECT COUNT(*) FROM findings WHERE inspection_id = $1
		`, command.AuditID).Scan(&auditFindingCount); err != nil {
			return transition[CreatedReportVersion]{}, err
		}
		if err := transaction.QueryRow(ctx, `
			SELECT COUNT(*) FROM findings
			WHERE inspection_id = $1 AND id = ANY($2::text[])
		`, command.AuditID, command.FindingIDs).Scan(&linkedFindingCount); err != nil {
			return transition[CreatedReportVersion]{}, err
		}
		if linkedFindingCount != auditFindingCount || linkedFindingCount != len(command.FindingIDs) {
			return transition[CreatedReportVersion]{}, fmt.Errorf("%w: report must link the exact immutable Finding set for this Audit", ErrConflict)
		}
		var potentialFindingIDs []string
		if err := transaction.QueryRow(ctx, `
			SELECT COALESCE(array_agg(id ORDER BY id), ARRAY[]::text[])
			FROM potential_findings
			WHERE inspection_id = $1
		`, command.AuditID).Scan(&potentialFindingIDs); err != nil {
			return transition[CreatedReportVersion]{}, err
		}
		contentBytes, err := json.Marshal(command.Content)
		if err != nil {
			return transition[CreatedReportVersion]{}, ErrInvalid
		}
		content, err := documents.DecodeReportContent(contentBytes)
		if err != nil {
			return transition[CreatedReportVersion]{}, fmt.Errorf("%w: canonical report content: %v", ErrInvalid, err)
		}
		canonicalContent, err := json.Marshal(content)
		if err != nil {
			return transition[CreatedReportVersion]{}, ErrInvalid
		}
		contentDigest := sha256.Sum256(canonicalContent)
		contentHash := fmt.Sprintf("sha256:%x", contentDigest[:])
		snapshot := map[string]any{
			"kind": command.Kind, "ready": true,
			"findingIds": command.FindingIDs, "contentHash": contentHash,
			"createdBySubject": actor.SubjectID,
			// Freeze the Potential Finding roots alongside the formal Finding
			// set. A later conversion is valid only for a root explicitly present
			// in this immutable Preliminary snapshot.
			"potentialFindingIds": potentialFindingIDs,
			"responseDueDate":     nil, "caaVisibleComment": nil,
			"content": content,
		}
		if _, err := reports.Prepare(reports.PrepareInput{
			ReportID: command.ReportID, Kind: command.Kind,
			Version: command.Version, FindingIDs: command.FindingIDs,
			ContentHash: contentHash, Ready: true,
		}); err != nil {
			return transition[CreatedReportVersion]{}, fmt.Errorf("%w: %v", ErrInvalid, err)
		}
		snapshotBytes, _ := json.Marshal(snapshot)
		now := service.clock().UTC()
		if _, err := transaction.Exec(ctx, `
			INSERT INTO report_versions (
				id, report_id, inspection_id, version, status, snapshot, created_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7)
		`, command.ReportVersionID, command.ReportID, command.AuditID,
			command.Version, command.Status, snapshotBytes, now); err != nil {
			return transition[CreatedReportVersion]{}, mapCreateConflict(err)
		}
		if _, err := transaction.Exec(ctx, `
			INSERT INTO report_approval_states (
				report_version_id, status, revision, updated_at
			) VALUES ($1, $2, 1, $3)
		`, command.ReportVersionID, command.Status, now); err != nil {
			return transition[CreatedReportVersion]{}, err
		}
		output := CreatedReportVersion{
			ReportVersionID: command.ReportVersionID, ReportID: command.ReportID,
			OrganizationID: organizationID, AuditID: command.AuditID,
			FindingIDs:  append([]string(nil), command.FindingIDs...),
			ContentHash: contentHash, Version: command.Version,
			Status: command.Status, Revision: 1,
		}
		return transition[CreatedReportVersion]{
			Response: output, OrganizationID: organizationID,
			Action: "report.version_created", EntityType: "report_version",
			EntityID: command.ReportVersionID, EntityVersion: 1,
			AfterStatus: command.Status, Reason: "Created immutable report version.",
			SyncKind: "report_version", OutboxTopic: "report.version_created",
		}, nil
	})
}
