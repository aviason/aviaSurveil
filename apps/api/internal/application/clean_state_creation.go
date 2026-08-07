package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/datafeed"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/identity"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/reports"
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
	return executeTransition(ctx, service, actor, commandEnvelope{
		OperationID: command.OperationID, IdempotencyKey: command.IdempotencyKey,
		CorrelationID: command.OperationID, Kind: "create_planning_intake_draft",
		EntityID: command.DraftID, Semantic: command,
	}, func(ctx context.Context, transaction pgx.Tx) (transition[PlanningIntakeDraft], error) {
		canonicalValues := NormalizeCanonicalPlanningValues(command.Values, command.DraftID)
		canonicalValues["organizationId"] = command.OrganizationID
		if canonicalValues["catalogVersion"] != "" {
			if canonicalValues["scopeDraftId"] == "" {
				canonicalValues["scopeDraftId"] = "scope-draft-" + command.DraftID
				command.Values["scopeDraftId"] = canonicalValues["scopeDraftId"]
			}
			if _, err := ValidateCanonicalScopeMap(ctx, transaction, actor, canonicalValues, false); err != nil {
				return transition[PlanningIntakeDraft]{}, err
			}
		}
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

type AuditWorkspaceQuestion struct {
	QuestionID                  string   `json:"questionId"`
	AssignedInspectorSubjectIDs []string `json:"assignedInspectorSubjectIds"`
}

type CreateAuditWorkspaceCommand struct {
	OperationID              string
	IdempotencyKey           string
	PlanningItemID           string
	ExpectedPlanningRevision int64
	AuditID                  string
	AssignmentID             string
	PackageID                string
	PackageDraftID           string
	TemplateID               string
	TemplateVersionID        string
	LeadInspectorSubjectID   string
	MemberSubjectIDs         []string
	ScheduledStartDate       string
	ScheduledEndDate         string
	ExpiresAt                time.Time
	Questions                []AuditWorkspaceQuestion
}

type AuditWorkspace struct {
	AuditID           string `json:"auditId"`
	AssignmentID      string `json:"assignmentId"`
	PackageID         string `json:"packageId"`
	PackageDraftID    string `json:"packageDraftId"`
	TemplateVersionID string `json:"templateVersionId"`
	PackageVersion    int64  `json:"packageVersion"`
	Revision          int64  `json:"revision"`
}

func (service *Service) CreateAuditWorkspace(
	ctx context.Context,
	actor identity.Principal,
	command CreateAuditWorkspaceCommand,
) (AuditWorkspace, error) {
	if !actor.HasRole(identity.RoleDepartmentManager) {
		return AuditWorkspace{}, fmt.Errorf(
			"%w: Department Manager authority is required",
			ErrForbidden,
		)
	}
	start, startErr := time.Parse("2006-01-02", command.ScheduledStartDate)
	end, endErr := time.Parse("2006-01-02", command.ScheduledEndDate)
	if command.OperationID == "" || command.IdempotencyKey == "" ||
		command.PlanningItemID == "" || command.ExpectedPlanningRevision <= 0 ||
		command.AuditID == "" || command.AssignmentID == "" ||
		command.PackageID == "" || command.PackageDraftID == "" ||
		command.TemplateID == "" || command.TemplateVersionID == "" ||
		command.LeadInspectorSubjectID == "" ||
		len(command.MemberSubjectIDs) == 0 || len(command.Questions) == 0 ||
		command.ExpiresAt.IsZero() || startErr != nil || endErr != nil ||
		end.Before(start) {
		return AuditWorkspace{}, ErrInvalid
	}
	assignedQuestions := map[string][]string{}
	for _, question := range command.Questions {
		if question.QuestionID == "" || len(question.AssignedInspectorSubjectIDs) == 0 {
			return AuditWorkspace{}, ErrInvalid
		}
		if _, exists := assignedQuestions[question.QuestionID]; exists {
			return AuditWorkspace{}, ErrInvalid
		}
		assignedQuestions[question.QuestionID] = append(
			[]string(nil),
			question.AssignedInspectorSubjectIDs...,
		)
	}
	return executeTransition(ctx, service, actor, commandEnvelope{
		OperationID: command.OperationID, IdempotencyKey: command.IdempotencyKey,
		CorrelationID: command.OperationID, Kind: "create_audit_workspace",
		EntityID: command.AuditID, Semantic: command,
	}, func(ctx context.Context, transaction pgx.Tx) (transition[AuditWorkspace], error) {
		var organizationID, title, inspectionType, planStatus string
		var planRevision int64
		var dueDate time.Time
		if err := transaction.QueryRow(ctx, `
			SELECT organization_id, title, inspection_type, status, revision, scheduled_date
			FROM surveillance_plan_items
			WHERE id = $1 AND tombstoned_at IS NULL
			FOR UPDATE
		`, command.PlanningItemID).Scan(
			&organizationID, &title, &inspectionType, &planStatus,
			&planRevision, &dueDate,
		); err != nil {
			if err == pgx.ErrNoRows {
				return transition[AuditWorkspace]{}, ErrNotFound
			}
			return transition[AuditWorkspace]{}, err
		}
		if planStatus != "RELEASED" || planRevision != command.ExpectedPlanningRevision {
			return transition[AuditWorkspace]{}, ErrConflict
		}
		auditScopeCode, err := dataFeedAuditScopeCode(inspectionType)
		if err != nil {
			return transition[AuditWorkspace]{}, err
		}
		if service.dataFeedWriter == nil {
			return transition[AuditWorkspace]{}, ErrDataFeedNotConfigured
		}
		var noticePolicy string
		if err := transaction.QueryRow(ctx, `SELECT COALESCE(values->>'noticePolicy','ADVANCE') FROM planning_intake_drafts WHERE values->>'preparedAuditId'=$1 AND tombstoned_at IS NULL FOR UPDATE`, command.AuditID).Scan(&noticePolicy); err != nil {
			if err == pgx.ErrNoRows {
				return transition[AuditWorkspace]{}, ErrNotFound
			}
			return transition[AuditWorkspace]{}, err
		}
		var templateSnapshot []byte
		var templateID string
		if err := transaction.QueryRow(ctx, `
			SELECT template_id, snapshot
			FROM checklist_template_versions
			WHERE id = $1
		`, command.TemplateVersionID).Scan(&templateID, &templateSnapshot); err != nil {
			if err == pgx.ErrNoRows {
				return transition[AuditWorkspace]{}, ErrNotFound
			}
			return transition[AuditWorkspace]{}, err
		}
		if templateID != command.TemplateID {
			return transition[AuditWorkspace]{}, ErrConflict
		}
		var snapshot struct {
			SchemaVersion   int64            `json:"schemaVersion"`
			ProtocolVersion int64            `json:"protocolVersion"`
			Questions       []map[string]any `json:"questions"`
		}
		if err := json.Unmarshal(templateSnapshot, &snapshot); err != nil {
			return transition[AuditWorkspace]{}, ErrInvalid
		}
		for _, question := range snapshot.Questions {
			questionID, _ := question["id"].(string)
			assigned, exists := assignedQuestions[questionID]
			if !exists {
				return transition[AuditWorkspace]{}, ErrConflict
			}
			question["assignedInspectorUserIds"] = assigned
			delete(assignedQuestions, questionID)
		}
		if len(assignedQuestions) != 0 || len(snapshot.Questions) == 0 {
			return transition[AuditWorkspace]{}, ErrConflict
		}
		packageSnapshot, err := json.Marshal(snapshot)
		if err != nil {
			return transition[AuditWorkspace]{}, err
		}
		digest := sha256.Sum256(packageSnapshot)
		packageDigest := fmt.Sprintf("sha256:%x", digest[:])
		now := service.clock().UTC()
		firstAssigned := command.Questions[0].AssignedInspectorSubjectIDs[0]
		if _, err := transaction.Exec(ctx, `
			INSERT INTO inspections (
				id, organization_id, assigned_inspector_subject_id, title,
				inspection_type, status, due_date, revision, created_at, updated_at
			) VALUES ($1, $2, $3, $4, $5, 'READY_TO_EXECUTE', $6, 1, $7, $7)
		`, command.AuditID, organizationID, firstAssigned, title,
			inspectionType, dueDate, now); err != nil {
			return transition[AuditWorkspace]{}, mapCreateConflict(err)
		}
		if _, err := transaction.Exec(ctx, `
			INSERT INTO audit_assignments (
				id, inspection_id, organization_id, lead_subject_id, status,
				scheduled_start_date, scheduled_end_date, revision, created_at, updated_at
			) VALUES (
				$1, $2, $3, $4, $5,
				$6, $7, 1, $8, $8
			)
		`, command.AssignmentID, command.AuditID, organizationID,
			command.LeadInspectorSubjectID, func() string {
				if noticePolicy == "WITHHELD" {
					return "SCHEDULED"
				}
				return "AWAITING_AUDITEE_CONFIRMATION"
			}(), start, end, now); err != nil {
			return transition[AuditWorkspace]{}, mapCreateConflict(err)
		}
		if _, err := transaction.Exec(ctx, `
			INSERT INTO audit_team_members (
				assignment_id, subject_id, member_role, revision, created_at
			) VALUES ($1, $2, 'LEAD_INSPECTOR', 1, $3)
		`, command.AssignmentID, command.LeadInspectorSubjectID, now); err != nil {
			return transition[AuditWorkspace]{}, err
		}
		for _, subjectID := range command.MemberSubjectIDs {
			if subjectID == command.LeadInspectorSubjectID {
				continue
			}
			if _, err := transaction.Exec(ctx, `
				INSERT INTO audit_team_members (
					assignment_id, subject_id, member_role, revision, created_at
				) VALUES ($1, $2, 'INSPECTOR', 1, $3)
			`, command.AssignmentID, subjectID, now); err != nil {
				return transition[AuditWorkspace]{}, err
			}
		}
		for _, question := range command.Questions {
			for _, subjectID := range question.AssignedInspectorSubjectIDs {
				if _, err := transaction.Exec(ctx, `
					INSERT INTO inspection_question_assignments (
						inspection_id, question_id, subject_id, assignment_revision
					) VALUES ($1, $2, $3, 1)
				`, command.AuditID, question.QuestionID, subjectID); err != nil {
					return transition[AuditWorkspace]{}, err
				}
				if _, err := transaction.Exec(ctx, `
					INSERT INTO audit_question_assignments (
						assignment_id, question_id, subject_id, revision, created_at
					) VALUES ($1, $2, $3, 1, $4)
				`, command.AssignmentID, question.QuestionID, subjectID, now); err != nil {
					return transition[AuditWorkspace]{}, err
				}
			}
		}
		if _, err := transaction.Exec(ctx, `
			INSERT INTO inspection_packages (
				id, inspection_id, checklist_template_version_id, package_version,
				snapshot, expires_at, created_at, package_digest
			) VALUES ($1, $2, $3, 1, $4, $5, $6, $7)
		`, command.PackageID, command.AuditID, command.TemplateVersionID,
			packageSnapshot, command.ExpiresAt.UTC(), now, packageDigest); err != nil {
			return transition[AuditWorkspace]{}, mapCreateConflict(err)
		}
		if _, err := transaction.Exec(ctx, `
			INSERT INTO inspection_checklists (inspection_id, status, revision)
			VALUES ($1, 'NOT_STARTED', 1)
		`, command.AuditID); err != nil {
			return transition[AuditWorkspace]{}, err
		}
		draftQuestions := make([]map[string]any, 0, len(snapshot.Questions))
		for _, question := range snapshot.Questions {
			expectedEvidence, _ := question["expectedEvidence"].(string)
			draftQuestions = append(draftQuestions, map[string]any{
				"id": question["id"], "prompt": question["prompt"],
				"whyIncluded":         "Included by the authorized published checklist version.",
				"expectedEvidence":    []string{expectedEvidence},
				"configuredReference": question["regulatoryReference"],
			})
		}
		draftQuestionJSON, _ := json.Marshal(draftQuestions)
		riskFocusJSON, _ := json.Marshal([]string{"Authorized checklist scope"})
		if _, err := transaction.Exec(ctx, `
			INSERT INTO inspection_package_drafts (
				id, source_inspection_id, organization_id, status, package_version,
				risk_focus, question_snapshot, revision, created_by_subject_id,
				created_at, updated_at
			) VALUES ($1, $2, $3, 'DRAFT', 1, $4, $5, 1, $6, $7, $7)
		`, command.PackageDraftID, command.AuditID, organizationID,
			riskFocusJSON, draftQuestionJSON, actor.SubjectID, now); err != nil {
			return transition[AuditWorkspace]{}, mapCreateConflict(err)
		}
		output := AuditWorkspace{
			AuditID: command.AuditID, AssignmentID: command.AssignmentID,
			PackageID: command.PackageID, PackageDraftID: command.PackageDraftID,
			TemplateVersionID: command.TemplateVersionID,
			PackageVersion:    1, Revision: 1,
		}
		correlationID, err := datafeed.NewEventID()
		if err != nil {
			return transition[AuditWorkspace]{}, fmt.Errorf("allocate datafeed correlation id: %w", err)
		}
		plannedEventID, err := datafeed.NewEventID()
		if err != nil {
			return transition[AuditWorkspace]{}, fmt.Errorf("allocate planned datafeed event id: %w", err)
		}
		plannedStart := time.Date(dueDate.Year(), dueDate.Month(), dueDate.Day(), 0, 0, 0, 0, time.UTC)
		dataFeedEvents := []datafeed.EventInput{
			{
				EventID: plannedEventID, EventType: "audit.planned",
				OwningOrganizationID: organizationID, ActorOrganizationID: actor.OrganizationID,
				CorrelationID: correlationID, AggregateType: "audit", AggregateID: command.AuditID,
				AggregateRevision: 1, EffectiveAt: now, KnownAt: now, OccurredAt: now, EmittedAt: now,
				VisibilityPurposeCode: "regulated_oversight", EntityRefs: map[string]any{"audit_id": command.AuditID},
				StateBefore: nil, StateAfter: "audit_planned",
				Payload: map[string]any{
					"audit_program_ref": command.PlanningItemID,
					"audit_scope_code":  auditScopeCode,
					"planned_start_at":  plannedStart.Format(time.RFC3339Nano),
				},
			},
		}
		return transition[AuditWorkspace]{
			Response: output, OrganizationID: organizationID,
			Action: "audit.workspace_created", EntityType: "inspection",
			EntityID: command.AuditID, EntityVersion: 1,
			AfterStatus: "READY_TO_EXECUTE",
			Reason:      "Created audit workspace from a released plan and published checklist.",
			SyncKind:    "inspection", OutboxTopic: "audit.workspace_created",
			DataFeedEvents: dataFeedEvents,
		}, nil
	})
}

// dataFeedAuditScopeCode permits only source inspection-type values already
// represented by this candidate. It intentionally does not normalize input:
// an unknown type, including any ambiguous combined value, cannot be emitted.
func dataFeedAuditScopeCode(inspectionType string) (string, error) {
	switch inspectionType {
	case "RAMP":
		return "ramp", nil
	case "CABIN":
		return "cabin", nil
	case "RAMP_INSPECTION":
		return "ramp_inspection", nil
	case "CABIN_INSPECTION":
		return "cabin_inspection", nil
	default:
		return "", fmt.Errorf("%w: unsupported exact inspection type for datafeed: %q", ErrInvalid, inspectionType)
	}
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
		command.AuditID == "" || command.Version < 0 ||
		(command.Kind != reports.KindPreliminary && command.Kind != reports.KindFinal) ||
		(command.Status != "RETURNED" && command.Status != "DEPARTMENT_REVIEW") {
		return CreatedReportVersion{}, ErrInvalid
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
		contentBytes, err := json.Marshal(command.Content)
		if err != nil {
			return transition[CreatedReportVersion]{}, ErrInvalid
		}
		contentDigest := sha256.Sum256(contentBytes)
		contentHash := fmt.Sprintf("sha256:%x", contentDigest[:])
		snapshot := map[string]any{
			"kind": command.Kind, "ready": true,
			"findingIds": command.FindingIDs, "contentHash": contentHash,
			"responseDueDate": nil, "caaVisibleComment": nil,
			"content": command.Content,
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
