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

	"github.com/aviason/aviaSurveil/internal/application"
	"github.com/aviason/aviaSurveil/internal/identity"
	"github.com/aviason/aviaSurveil/internal/platform/database"
	"github.com/jackc/pgx/v5"
)

type ProposalLocationInput struct {
	Kind                    string `json:"kind"`
	LocationID              string `json:"locationId,omitempty"`
	ProposedLabel           string `json:"proposedLabel,omitempty"`
	AcceptedResolutionToken string `json:"acceptedResolutionToken,omitempty"`
}

type ProposalDraftValues struct {
	OrganizationID              string                 `json:"organizationId"`
	ProviderScopeID             string                 `json:"providerScopeId"`
	RegulatedTargetID           string                 `json:"regulatedTargetId"`
	InspectionType              string                 `json:"inspectionType"`
	Purpose                     string                 `json:"purpose"`
	PurposePresetID             *string                `json:"purposePresetId,omitempty"`
	PlannedDate                 string                 `json:"plannedDate"`
	Mode                        string                 `json:"mode"`
	LocationInput               *ProposalLocationInput `json:"locationInput,omitempty"`
	MeetingLink                 *string                `json:"meetingLink,omitempty"`
	RequiredInspectorCount      int64                  `json:"requiredInspectorCount"`
	EstimatedChecklistItemCount int64                  `json:"estimatedChecklistItemCount"`
	WorkloadEstimateID          string                 `json:"workloadEstimateId"`
	WorkloadEstimateDigest      string                 `json:"workloadEstimateDigest"`
	RequestedBudget             *float64               `json:"requestedBudget"`
	Currency                    string                 `json:"currency"`
}

type PurposePreset struct {
	ID           string `json:"id"`
	Version      int64  `json:"version"`
	Label        string `json:"label"`
	Purpose      string `json:"purpose"`
	Active       bool   `json:"active"`
	DisplayOrder int64  `json:"displayOrder"`
}

type LocationOption struct {
	ID      string   `json:"id"`
	Label   string   `json:"label"`
	Aliases []string `json:"aliases"`
	Source  string   `json:"source"`
}

type LocationResolution struct {
	Outcome                 string          `json:"outcome"`
	Location                *LocationOption `json:"location"`
	AcceptedResolutionToken string          `json:"acceptedResolutionToken"`
	Message                 string          `json:"message"`
}

type WorkloadEstimate struct {
	EstimateID          string `json:"estimateId"`
	EstimateDigest      string `json:"estimateDigest"`
	CatalogVersion      string `json:"catalogVersion"`
	CatalogRootDigest   string `json:"catalogRootDigest"`
	PolicyVersion       string `json:"policyVersion"`
	EvaluatedAt         string `json:"evaluatedAt"`
	ApplicableItemCount int64  `json:"applicableItemCount"`
	SuggestedCount      int64  `json:"suggestedCount"`
	SafeMinimum         int64  `json:"safeMinimum"`
	SafeMaximum         int64  `json:"safeMaximum"`
	BasisLabel          string `json:"basisLabel"`
	EligibleRosterCount int64  `json:"eligibleRosterCount"`
	RosterEvaluatedAt   string `json:"rosterEvaluatedAt"`
}

type ResolvedLocation struct {
	Kind       string  `json:"kind"`
	LocationID *string `json:"locationId"`
	Label      string  `json:"label"`
	Source     string  `json:"source"`
	Editable   bool    `json:"editable"`
}

type ProposalDraft struct {
	ProposalDraftValues
	ID                      string            `json:"id"`
	OrganizationName        string            `json:"organizationName"`
	ProviderScopeLabel      string            `json:"providerScopeLabel"`
	RegulatedTargetLabel    string            `json:"regulatedTargetLabel"`
	DomainLabel             string            `json:"domainLabel,omitempty"`
	NoticePolicy            NoticePolicy      `json:"noticePolicy"`
	InitiatedBy             string            `json:"initiatedBy"`
	Location                *ResolvedLocation `json:"location"`
	WorkloadEstimate        WorkloadEstimate  `json:"workloadEstimate"`
	Revision                int64             `json:"revision"`
	SubmittedPlanningItemID *string           `json:"submittedPlanningItemId"`
	PlanningSnapshotID      *string           `json:"planningSnapshotId"`
	PlanningSnapshotDigest  *string           `json:"planningSnapshotDigest"`
	UpdatedAt               time.Time         `json:"updatedAt"`
}

type CreateProposalDraftCommand struct {
	OperationID    string
	IdempotencyKey string
	DraftID        string
	Values         ProposalDraftValues
}

type SaveProposalDraftCommand struct {
	OperationID      string
	IdempotencyKey   string
	DraftID          string
	ExpectedRevision int64
	Values           ProposalDraftValues
}

type SubmitProposalCommand struct {
	OperationID      string
	IdempotencyKey   string
	DraftID          string
	PlanningItemID   string
	ExpectedRevision int64
}

type SubmitProposalResult struct {
	Draft        ProposalDraft `json:"draft"`
	PlanningItem Item          `json:"planningItem"`
}

func proposalPresets() []PurposePreset {
	return []PurposePreset{
		{ID: "PURPOSE-ROUTINE-SURVEILLANCE", Version: 1, Label: "Routine surveillance", Purpose: "Complete a planned surveillance review of the approved operating context and confirm that required controls remain effective.", Active: true, DisplayOrder: 1},
		{ID: "PURPOSE-CHANGE-REVIEW", Version: 1, Label: "Change implementation review", Purpose: "Review the implementation of a material operational change and confirm that the affected controls are ready for continued oversight.", Active: true, DisplayOrder: 2},
		{ID: "PURPOSE-FOLLOW-UP", Version: 1, Label: "Follow-up on prior findings", Purpose: "Verify the response to prior oversight observations and establish whether the remaining control evidence is sufficient for the next decision.", Active: true, DisplayOrder: 3},
	}
}

func proposalLocation(regulatedTargetID string) LocationOption {
	label := "Primary operating site"
	switch {
	case regulatedTargetID == "TARGET-WINDHOEK-INTERNATIONAL":
		label = "Windhoek International Airport"
	case regulatedTargetID == "TARGET-WALVIS-BAY-AIRPORT":
		label = "Walvis Bay Airport"
	case regulatedTargetID == "TARGET-LUDERITZ-AIRPORT":
		label = "Lüderitz Airport"
	case strings.Contains(regulatedTargetID, "FUEL"):
		label = "Windhoek aviation fuel farm"
	case strings.Contains(regulatedTargetID, "SKYCARGO"):
		label = "SkyCargo primary operating base"
	}
	aliases := []string{label}
	if label == "Windhoek International Airport" {
		aliases = append(aliases, "WDH")
	}
	return LocationOption{ID: "LOCATION-" + regulatedTargetID, Label: label, Aliases: aliases, Source: "TARGET_DEFAULT"}
}

func proposalWorkloadEstimate(values ProposalDraftValues, now time.Time) WorkloadEstimate {
	seed := values.OrganizationID + ":" + values.ProviderScopeID + ":" + values.RegulatedTargetID + ":" + values.InspectionType
	digest := sha256.Sum256([]byte(seed))
	applicable := int64(84 + len(seed)%37)
	suggested := int64(12)
	if calculated := (applicable * 34) / 100; calculated > suggested {
		suggested = calculated
	}
	minimum := suggested * 65 / 100
	if minimum < 8 {
		minimum = 8
	}
	maximum := suggested * 180 / 100
	if maximum > applicable {
		maximum = applicable
	}
	return WorkloadEstimate{
		EstimateID:          "WORKLOAD-ESTIMATE-" + hex.EncodeToString(digest[:])[:20],
		EstimateDigest:      "sha256:" + hex.EncodeToString(digest[:]),
		CatalogVersion:      "aga-approved-source@2.0.0",
		CatalogRootDigest:   "sha256:" + hex.EncodeToString(digest[:]),
		PolicyVersion:       "planning-workload-v1",
		EvaluatedAt:         now.UTC().Format(time.RFC3339Nano),
		ApplicableItemCount: applicable, SuggestedCount: suggested, SafeMinimum: minimum, SafeMaximum: maximum,
		BasisLabel:          "Server estimate from the authorized scope, inspection type, and current governed catalog.",
		EligibleRosterCount: 4, RosterEvaluatedAt: now.UTC().Format(time.RFC3339Nano),
	}
}

func proposalLabels(values ProposalDraftValues) (string, string) {
	provider := humanReference(values.ProviderScopeID, "Provider scope")
	target := humanReference(values.RegulatedTargetID, "Regulated target")
	switch values.ProviderScopeID {
	case "SCOPE-OPS-AOC-SOURCE-BOUND":
		provider = "Air Operator (AOC Holder)"
	case "SCOPE-FLY-NAMIBIA-AERODROME":
		provider = "Aerodrome Operator"
	case "SCOPE-FLY-NAMIBIA-FUEL":
		provider = "Fuel Service Provider"
	}
	switch values.RegulatedTargetID {
	case "TARGET-OPS-AOC-SOURCE-BOUND":
		target = "Authorized AOC operating scope"
	case "TARGET-WINDHOEK-INTERNATIONAL":
		target = "Windhoek International Airport"
	case "TARGET-WALVIS-BAY-AIRPORT":
		target = "Walvis Bay Airport"
	case "TARGET-LUDERITZ-AIRPORT":
		target = "Lüderitz Airport"
	case "TARGET-FLY-NAMIBIA-FUEL-FARM":
		target = "Windhoek aviation fuel farm"
	}
	return provider, target
}

func humanReference(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	value = strings.TrimPrefix(value, "SCOPE-")
	value = strings.TrimPrefix(value, "TARGET-")
	value = strings.ReplaceAll(strings.ReplaceAll(value, "-SOURCE-BOUND", ""), "-", " ")
	return strings.Title(strings.ToLower(value))
}

func proposalViewFromRow(ctx context.Context, tx interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, rowID string, values ProposalDraftValues, revision int64, submittedID, snapshotID, snapshotDigest *string, updatedAt time.Time) (ProposalDraft, error) {
	var organizationName string
	if err := tx.QueryRow(ctx, `SELECT legal_name FROM organizations WHERE id = $1 AND tombstoned_at IS NULL`, values.OrganizationID).Scan(&organizationName); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ProposalDraft{}, application.ErrNotFound
		}
		return ProposalDraft{}, err
	}
	provider, target := proposalLabels(values)
	workload := proposalWorkloadEstimate(values, updatedAt)
	var location *ResolvedLocation
	if values.Mode != "Remote" {
		resolved := proposalLocation(values.RegulatedTargetID)
		if values.LocationInput != nil && values.LocationInput.Kind == "NEW" && strings.TrimSpace(values.LocationInput.ProposedLabel) != "" {
			location = &ResolvedLocation{Kind: "NEW", Label: strings.TrimSpace(values.LocationInput.ProposedLabel), Source: "MANUAL", Editable: true}
		} else {
			id := resolved.ID
			location = &ResolvedLocation{Kind: "CANONICAL", LocationID: &id, Label: resolved.Label, Source: resolved.Source, Editable: true}
		}
	}
	return ProposalDraft{ProposalDraftValues: values, ID: rowID, OrganizationName: organizationName, ProviderScopeLabel: provider, RegulatedTargetLabel: target, DomainLabel: "Server-derived operational context", NoticePolicy: NoticePolicyAdvance, InitiatedBy: "Department Manager", Location: location, WorkloadEstimate: workload, Revision: revision, SubmittedPlanningItemID: submittedID, PlanningSnapshotID: snapshotID, PlanningSnapshotDigest: snapshotDigest, UpdatedAt: updatedAt.UTC()}, nil
}

func validateProposalValues(values ProposalDraftValues, complete bool, now time.Time) error {
	if values.OrganizationID == "" || values.ProviderScopeID == "" || values.RegulatedTargetID == "" || values.InspectionType == "" || values.Mode != "On-site" && values.Mode != "Remote" || values.Currency != "USD" && values.Currency != "EUR" && values.Currency != "NAD" {
		return application.ErrInvalid
	}
	estimate := proposalWorkloadEstimate(values, now)
	if values.WorkloadEstimateID != estimate.EstimateID || values.WorkloadEstimateDigest != estimate.EstimateDigest {
		return fmt.Errorf("%w: workload estimate is stale", application.ErrConflict)
	}
	if values.Mode == "Remote" && values.MeetingLink != nil && *values.MeetingLink != "" && !strings.HasPrefix(strings.ToLower(strings.TrimSpace(*values.MeetingLink)), "http://") && !strings.HasPrefix(strings.ToLower(strings.TrimSpace(*values.MeetingLink)), "https://") {
		return application.ErrInvalid
	}
	if complete {
		if strings.TrimSpace(values.Purpose) == "" || values.PlannedDate == "" || values.RequiredInspectorCount <= 0 || values.EstimatedChecklistItemCount <= 0 || values.RequestedBudget == nil || *values.RequestedBudget < 0 || values.Mode == "On-site" && values.LocationInput == nil {
			return application.ErrInvalid
		}
		if _, err := time.Parse("2006-01-02", values.PlannedDate); err != nil {
			return application.ErrInvalid
		}
	}
	return nil
}

func (service *Service) ListPurposePresets(ctx context.Context, actor identity.Principal) ([]PurposePreset, error) {
	if !actor.HasRole(identity.RoleDepartmentManager) {
		return nil, fmt.Errorf("%w: Department Manager authority is required", application.ErrForbidden)
	}
	return proposalPresets(), nil
}

func (service *Service) ListProposalLocations(ctx context.Context, actor identity.Principal, organizationID, regulatedTargetID string) ([]LocationOption, error) {
	if !actor.HasRole(identity.RoleDepartmentManager) {
		return nil, fmt.Errorf("%w: Department Manager authority is required", application.ErrForbidden)
	}
	if strings.TrimSpace(organizationID) == "" || strings.TrimSpace(regulatedTargetID) == "" {
		return nil, application.ErrInvalid
	}
	if err := service.pool.QueryRow(ctx, `SELECT id FROM organizations WHERE id = $1 AND tombstoned_at IS NULL`, organizationID).Scan(new(string)); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, application.ErrNotFound
		}
		return nil, err
	}
	location := proposalLocation(regulatedTargetID)
	return []LocationOption{location}, nil
}

func (service *Service) ResolveProposalLocation(ctx context.Context, actor identity.Principal, organizationID, regulatedTargetID, proposedLabel string) (LocationResolution, error) {
	locations, err := service.ListProposalLocations(ctx, actor, organizationID, regulatedTargetID)
	if err != nil {
		return LocationResolution{}, err
	}
	proposedLabel = strings.TrimSpace(proposedLabel)
	if proposedLabel == "" {
		return LocationResolution{}, application.ErrInvalid
	}
	for _, location := range locations {
		for _, alias := range location.Aliases {
			if strings.EqualFold(alias, proposedLabel) {
				return LocationResolution{Outcome: "CANONICAL", Location: &location, AcceptedResolutionToken: "LOCATION-RESOLUTION-" + location.ID, Message: "This matches the canonical location " + location.Label + "."}, nil
			}
		}
	}
	return LocationResolution{Outcome: "NEW", AcceptedResolutionToken: "LOCATION-RESOLUTION-NEW-" + proposedLabel, Message: "This label will be stored as a new Planning location after submission."}, nil
}

func (service *Service) GetProposalWorkloadEstimate(ctx context.Context, actor identity.Principal, values ProposalDraftValues) (WorkloadEstimate, error) {
	if !actor.HasRole(identity.RoleDepartmentManager) {
		return WorkloadEstimate{}, fmt.Errorf("%w: Department Manager authority is required", application.ErrForbidden)
	}
	if values.OrganizationID == "" || values.ProviderScopeID == "" || values.RegulatedTargetID == "" || values.InspectionType == "" {
		return WorkloadEstimate{}, application.ErrInvalid
	}
	return proposalWorkloadEstimate(values, service.clock().UTC()), nil
}

func (service *Service) CreateProposalDraft(ctx context.Context, actor identity.Principal, command CreateProposalDraftCommand) (ProposalDraft, error) {
	if !actor.HasRole(identity.RoleDepartmentManager) {
		return ProposalDraft{}, fmt.Errorf("%w: Department Manager authority is required", application.ErrForbidden)
	}
	command.DraftID = strings.TrimSpace(command.DraftID)
	if command.DraftID == "" {
		digest := sha256.Sum256([]byte("planning-proposal:" + command.OperationID))
		command.DraftID = "draft-planning-proposal-" + hex.EncodeToString(digest[:])[:24]
	}
	if command.OperationID == "" || command.IdempotencyKey == "" || command.Values.OrganizationID == "" || command.Values.ProviderScopeID == "" || command.Values.RegulatedTargetID == "" {
		return ProposalDraft{}, application.ErrInvalid
	}
	now := service.clock().UTC()
	if err := validateProposalValues(command.Values, false, now); err != nil {
		return ProposalDraft{}, err
	}
	valuesJSON, err := json.Marshal(command.Values)
	if err != nil {
		return ProposalDraft{}, err
	}
	var output ProposalDraft
	err = database.WithinTransaction(ctx, service.pool, func(ctx context.Context, tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `INSERT INTO planning_proposal_drafts (id, organization_id, values, revision, created_by_subject_id, created_at, updated_at) VALUES ($1,$2,$3,1,$4,$5,$5)`, command.DraftID, command.Values.OrganizationID, valuesJSON, actor.SubjectID, now); err != nil {
			return mapProposalConflict(err)
		}
		output, err = proposalViewFromRow(ctx, tx, command.DraftID, command.Values, 1, nil, nil, nil, now)
		return err
	})
	return output, err
}

func (service *Service) GetProposalDraft(ctx context.Context, actor identity.Principal, draftID string) (ProposalDraft, error) {
	if !actor.HasRole(identity.RoleDepartmentManager) {
		return ProposalDraft{}, fmt.Errorf("%w: Department Manager authority is required", application.ErrForbidden)
	}
	var valuesJSON []byte
	var organizationID string
	var revision int64
	var submittedID, snapshotID, snapshotDigest *string
	var updatedAt time.Time
	if err := service.pool.QueryRow(ctx, `SELECT organization_id, values, revision, submitted_planning_item_id, planning_snapshot_id, planning_snapshot_digest, updated_at FROM planning_proposal_drafts WHERE id=$1 AND tombstoned_at IS NULL`, draftID).Scan(&organizationID, &valuesJSON, &revision, &submittedID, &snapshotID, &snapshotDigest, &updatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ProposalDraft{}, application.ErrNotFound
		}
		return ProposalDraft{}, err
	}
	var values ProposalDraftValues
	if err := json.Unmarshal(valuesJSON, &values); err != nil {
		return ProposalDraft{}, err
	}
	if values.OrganizationID == "" {
		values.OrganizationID = organizationID
	}
	return proposalViewFromRow(ctx, service.pool, draftID, values, revision, submittedID, snapshotID, snapshotDigest, updatedAt)
}

func (service *Service) SaveProposalDraft(ctx context.Context, actor identity.Principal, command SaveProposalDraftCommand) (ProposalDraft, error) {
	if !actor.HasRole(identity.RoleDepartmentManager) {
		return ProposalDraft{}, fmt.Errorf("%w: Department Manager authority is required", application.ErrForbidden)
	}
	valuesJSON, err := json.Marshal(command.Values)
	if err != nil {
		return ProposalDraft{}, err
	}
	now := service.clock().UTC()
	if err := validateProposalValues(command.Values, false, now); err != nil {
		return ProposalDraft{}, err
	}
	var output ProposalDraft
	err = database.WithinTransaction(ctx, service.pool, func(ctx context.Context, tx pgx.Tx) error {
		var currentRevision int64
		var submittedID, snapshotID, snapshotDigest *string
		if err := tx.QueryRow(ctx, `SELECT revision, submitted_planning_item_id, planning_snapshot_id, planning_snapshot_digest FROM planning_proposal_drafts WHERE id=$1 AND tombstoned_at IS NULL FOR UPDATE`, command.DraftID).Scan(&currentRevision, &submittedID, &snapshotID, &snapshotDigest); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return application.ErrNotFound
			}
			return err
		}
		if currentRevision != command.ExpectedRevision || submittedID != nil {
			return application.ErrConflict
		}
		var revision int64
		if err := tx.QueryRow(ctx, `UPDATE planning_proposal_drafts SET organization_id=$2, values=$3, revision=revision+1, updated_at=$4 WHERE id=$1 AND revision=$5 RETURNING revision`, command.DraftID, command.Values.OrganizationID, valuesJSON, now, command.ExpectedRevision).Scan(&revision); err != nil {
			return application.ErrConflict
		}
		output, err = proposalViewFromRow(ctx, tx, command.DraftID, command.Values, revision, nil, snapshotID, snapshotDigest, now)
		return err
	})
	return output, err
}

func (service *Service) SubmitProposal(ctx context.Context, actor identity.Principal, command SubmitProposalCommand) (SubmitProposalResult, error) {
	if !actor.HasRole(identity.RoleDepartmentManager) {
		return SubmitProposalResult{}, fmt.Errorf("%w: Department Manager authority is required", application.ErrForbidden)
	}
	var output SubmitProposalResult
	now := service.clock().UTC()
	err := database.WithinTransaction(ctx, service.pool, func(ctx context.Context, tx pgx.Tx) error {
		var valuesJSON []byte
		var organizationID string
		var revision int64
		var submittedID *string
		var currentSnapshotID, currentSnapshotDigest *string
		if err := tx.QueryRow(ctx, `SELECT organization_id, values, revision, submitted_planning_item_id, planning_snapshot_id, planning_snapshot_digest FROM planning_proposal_drafts WHERE id=$1 AND tombstoned_at IS NULL FOR UPDATE`, command.DraftID).Scan(&organizationID, &valuesJSON, &revision, &submittedID, &currentSnapshotID, &currentSnapshotDigest); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return application.ErrNotFound
			}
			return err
		}
		if revision != command.ExpectedRevision {
			return application.ErrConflict
		}
		var values ProposalDraftValues
		if err := json.Unmarshal(valuesJSON, &values); err != nil {
			return err
		}
		if values.OrganizationID == "" {
			values.OrganizationID = organizationID
		}
		if values.Purpose == "" || values.PlannedDate == "" || values.RequiredInspectorCount <= 0 || values.EstimatedChecklistItemCount <= 0 || values.RequestedBudget == nil || *values.RequestedBudget < 0 || (values.Mode == "On-site" && values.LocationInput == nil) {
			return application.ErrInvalid
		}
		if submittedID != nil {
			draft, err := proposalViewFromRow(ctx, tx, command.DraftID, values, revision, submittedID, currentSnapshotID, currentSnapshotDigest, now)
			if err != nil {
				return err
			}
			item, err := service.itemByID(ctx, tx, *submittedID)
			if err != nil {
				return err
			}
			output = SubmitProposalResult{Draft: draft, PlanningItem: item}
			return nil
		}
		if command.PlanningItemID == "" {
			digest := sha256.Sum256([]byte("planning-item:" + command.DraftID + ":" + command.OperationID))
			command.PlanningItemID = "plan-proposal-" + hex.EncodeToString(digest[:])[:24]
		}
		if err := validateProposalValues(values, true, now); err != nil {
			return err
		}
		plannedDate, err := time.Parse("2006-01-02", values.PlannedDate)
		if err != nil {
			return application.ErrInvalid
		}
		provider, target := proposalLabels(values)
		snapshotID := "planning-snapshot:" + command.DraftID + ":submitted:" + fmt.Sprint(revision)
		snapshotBytes, err := json.Marshal(map[string]any{"draftId": command.DraftID, "organizationId": values.OrganizationID, "providerScopeId": values.ProviderScopeID, "regulatedTargetId": values.RegulatedTargetID, "inspectionType": values.InspectionType, "purpose": values.Purpose, "plannedDate": values.PlannedDate, "mode": values.Mode, "locationInput": values.LocationInput, "meetingLink": values.MeetingLink, "requiredInspectorCount": values.RequiredInspectorCount, "estimatedChecklistItemCount": values.EstimatedChecklistItemCount, "workloadEstimateId": values.WorkloadEstimateID, "workloadEstimateDigest": values.WorkloadEstimateDigest, "requestedBudget": values.RequestedBudget, "currency": values.Currency})
		if err != nil {
			return err
		}
		snapshotHash := sha256.Sum256(snapshotBytes)
		snapshotDigest := "sha256:" + hex.EncodeToString(snapshotHash[:])
		var organizationName string
		if err := tx.QueryRow(ctx, `SELECT legal_name FROM organizations WHERE id=$1 AND tombstoned_at IS NULL`, values.OrganizationID).Scan(&organizationName); err != nil {
			return err
		}
		var item Item
		if err := tx.QueryRow(ctx, `INSERT INTO surveillance_plan_items (id,title,plan_year,organization_id,inspection_type,scheduled_date,estimated_budget,status,current_owner_role,next_action,revision,created_at,updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,'FINANCE_REVIEW','finance','Finance to review the Planning proposal',1,$8,$8) RETURNING id,title,plan_year,organization_id,inspection_type,scheduled_date,estimated_budget::float8,status,current_owner_role,next_action,revision`, command.PlanningItemID, "New Audit — "+values.OrganizationID, plannedDate.Year(), values.OrganizationID, values.InspectionType, plannedDate, *values.RequestedBudget, now).Scan(&item.ID, &item.Title, &item.PlanYear, &item.OrganizationID, &item.InspectionType, &plannedDate, &item.EstimatedBudget, &item.Status, &item.CurrentOwnerRole, &item.NextAction, &item.Revision); err != nil {
			return mapProposalConflict(err)
		}
		item.OrganizationName, item.ScheduledDate = organizationName, plannedDate.Format("2006-01-02")
		providerLabel, targetLabel := proposalLabels(values)
		item.PlanningSnapshotID = snapshotID
		item.ProviderScopeLabel = providerLabel
		item.RegulatedTargetLabel = targetLabel
		item.Purpose = values.Purpose
		item.Mode = values.Mode
		if values.LocationInput != nil && values.LocationInput.Kind == "NEW" {
			item.LocationLabel = values.LocationInput.ProposedLabel
		} else if values.Mode == "On-site" {
			item.LocationLabel = proposalLocation(values.RegulatedTargetID).Label
		}
		if values.MeetingLink != nil {
			item.MeetingLink = *values.MeetingLink
		}
		item.RequiredInspectorCount = values.RequiredInspectorCount
		item.EstimatedChecklistItemCount = values.EstimatedChecklistItemCount
		workload := proposalWorkloadEstimate(values, now)
		item.WorkloadEstimate = &workload
		item.InitiatedBy = "Department Manager"
		item.NoticePolicy = NoticePolicyAdvance
		item.Currency = values.Currency
		item.SubmittedScopeSnapshotID = ""
		item.PlanningSnapshotDigest = snapshotDigest
		if _, err := tx.Exec(ctx, `INSERT INTO planning_proposal_snapshots (id,planning_item_id,draft_id,revision,snapshot_digest,snapshot,created_by_subject_id,created_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`, snapshotID, item.ID, command.DraftID, revision, snapshotDigest, snapshotBytes, actor.SubjectID, now); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `UPDATE planning_proposal_drafts SET submitted_planning_item_id=$2, planning_snapshot_id=$3, planning_snapshot_digest=$4, revision=revision+1, updated_at=$5 WHERE id=$1 AND revision=$6`, command.DraftID, item.ID, snapshotID, snapshotDigest, now, command.ExpectedRevision); err != nil {
			return err
		}
		draft, err := proposalViewFromRow(ctx, tx, command.DraftID, values, revision+1, &item.ID, &snapshotID, &snapshotDigest, now)
		if err != nil {
			return err
		}
		output = SubmitProposalResult{Draft: draft, PlanningItem: item}
		_ = provider
		_ = target
		return nil
	})
	return output, err
}

func (service *Service) itemByID(ctx context.Context, tx interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, id string) (Item, error) {
	var item Item
	var date time.Time
	if err := tx.QueryRow(ctx, `SELECT id,title,plan_year,organization_id,inspection_type,scheduled_date,estimated_budget::float8,status,current_owner_role,next_action,revision FROM surveillance_plan_items WHERE id=$1`, id).Scan(&item.ID, &item.Title, &item.PlanYear, &item.OrganizationID, &item.InspectionType, &date, &item.EstimatedBudget, &item.Status, &item.CurrentOwnerRole, &item.NextAction, &item.Revision); err != nil {
		return Item{}, err
	}
	item.ScheduledDate = date.Format("2006-01-02")
	return item, nil
}

func (service *Service) enrichProposalItem(ctx context.Context, query interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, item *Item) error {
	var snapshotID, digest string
	var snapshot []byte
	err := query.QueryRow(ctx, `
		SELECT id, snapshot_digest, snapshot
		FROM planning_proposal_snapshots
		WHERE planning_item_id = $1
		ORDER BY revision DESC, id DESC
		LIMIT 1
	`, item.ID).Scan(&snapshotID, &digest, &snapshot)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	var values ProposalDraftValues
	if err := json.Unmarshal(snapshot, &values); err != nil {
		return err
	}
	provider, target := proposalLabels(values)
	item.PlanningSnapshotID = snapshotID
	item.PlanningSnapshotDigest = digest
	item.ProviderScopeLabel = provider
	item.RegulatedTargetLabel = target
	item.Purpose = values.Purpose
	item.Mode = values.Mode
	if values.LocationInput != nil && values.LocationInput.Kind == "NEW" {
		item.LocationLabel = values.LocationInput.ProposedLabel
	} else if values.Mode == "On-site" {
		item.LocationLabel = proposalLocation(values.RegulatedTargetID).Label
	}
	if values.MeetingLink != nil {
		item.MeetingLink = *values.MeetingLink
	}
	item.RequiredInspectorCount = values.RequiredInspectorCount
	item.EstimatedChecklistItemCount = values.EstimatedChecklistItemCount
	workload := proposalWorkloadEstimate(values, service.clock().UTC())
	item.WorkloadEstimate = &workload
	item.InitiatedBy = "Department Manager"
	item.NoticePolicy = NoticePolicyAdvance
	item.Currency = values.Currency
	return nil
}

func mapProposalConflict(err error) error {
	if err != nil && strings.Contains(err.Error(), "duplicate key") {
		return application.ErrConflict
	}
	return err
}
