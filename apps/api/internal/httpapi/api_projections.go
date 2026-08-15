package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/aviason/aviaSurveil/internal/application"
	"github.com/aviason/aviaSurveil/internal/caps"
	capstore "github.com/aviason/aviaSurveil/internal/caps/store/postgres"
	"github.com/aviason/aviaSurveil/internal/checklists"
	"github.com/aviason/aviaSurveil/internal/documents"
	"github.com/aviason/aviaSurveil/internal/httpapi/generated"
	"github.com/aviason/aviaSurveil/internal/identity"
	"github.com/aviason/aviaSurveil/internal/potentialfindings"
	pfstore "github.com/aviason/aviaSurveil/internal/potentialfindings/store/postgres"
	"github.com/jackc/pgx/v5"
)

type packageQuestionSnapshot struct {
	ID                  string   `json:"id"`
	SectionID           string   `json:"sectionId"`
	Prompt              string   `json:"prompt"`
	RegulatoryReference string   `json:"regulatoryReference"`
	ExpectedEvidence    string   `json:"expectedEvidence"`
	Assigned            []string `json:"assignedInspectorUserIds"`
}

type packageSnapshot struct {
	SchemaVersion   int64                     `json:"schemaVersion"`
	ProtocolVersion int64                     `json:"protocolVersion"`
	Questions       []packageQuestionSnapshot `json:"questions"`
}

func (api *CanonicalAPI) assignmentProjection(ctx context.Context, actor identity.Principal, status *string, limit *int64) (generated.ListAssignmentsOutput, error) {
	query := `
		SELECT DISTINCT inspection.id, inspection.organization_id, organization.legal_name,
		       inspection.title, inspection.status, inspection.revision, COALESCE(inspection.due_date::text, ''),
		       package.id, audit_assignment.id, audit_assignment.revision
		FROM inspections inspection
		JOIN organizations organization ON organization.id = inspection.organization_id
		LEFT JOIN audit_assignments audit_assignment
		  ON audit_assignment.inspection_id = inspection.id
		 AND audit_assignment.tombstoned_at IS NULL
		LEFT JOIN LATERAL (
			SELECT id
			FROM inspection_packages
			WHERE inspection_id = inspection.id AND revoked_at IS NULL
			ORDER BY package_version DESC
			LIMIT 1
		) package ON true
		LEFT JOIN inspection_question_assignments question_assignment ON question_assignment.inspection_id = inspection.id
		WHERE ($1 = false OR question_assignment.subject_id = $2)
		  AND ($3 = false OR inspection.organization_id = $4)
		ORDER BY inspection.id
	`
	isInspector := actor.HasRole(identity.RoleInspector) && !actor.HasRole(identity.RoleLeadInspector)
	isAuditee := actor.HasRole(identity.RoleAuditee)
	rows, err := api.pool.Query(ctx, query, isInspector, actor.SubjectID, isAuditee, actor.OrganizationID)
	if err != nil {
		return generated.ListAssignmentsOutput{}, err
	}
	defer rows.Close()
	items := []generated.AssignmentSummary{}
	for rows.Next() {
		var item generated.AssignmentSummary
		var inspectionRevision int64
		var dueDate string
		if err := rows.Scan(&item.AuditId, &item.OrganizationId, &item.OrganizationName, &item.Title, &item.Status, &inspectionRevision, &dueDate, &item.PackageId, &item.AssignmentId, &item.Revision); err != nil {
			return generated.ListAssignmentsOutput{}, err
		}
		item.InspectionRevision = &inspectionRevision
		if status != nil && item.Status != *status {
			continue
		}
		if dueDate != "" {
			item.DueDate = &dueDate
		}
		item.DueState = dueState(dueDate, api.clock())
		if item.Status == "SCHEDULED" || item.Status == "AWAITING_AUDITEE_CONFIRMATION" {
			item.NextAction = "Start inspection when ready"
		} else {
			item.NextAction = "Continue Cabin Inspection checklist"
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return generated.ListAssignmentsOutput{}, err
	}
	if limit != nil && *limit >= 0 && int64(len(items)) > *limit {
		items = items[:*limit]
	}
	return generated.ListAssignmentsOutput{Items: items}, nil
}

func (api *CanonicalAPI) inspectionPackageProjection(ctx context.Context, actor identity.Principal, packageID string) (generated.InspectionPackage, error) {
	// This is an execution projection, not a general CAA document. Managers,
	// Finance, GM, ED, and administrators use preparation/report projections;
	// only the assigned Lead or a covered Inspector may obtain execution data.
	if actor.HasRole(identity.RoleAuditee) || (!actor.HasRole(identity.RoleInspector) && !actor.HasRole(identity.RoleLeadInspector)) {
		return generated.InspectionPackage{}, fmt.Errorf("%w: execution package is restricted to the assigned inspection team", application.ErrForbidden)
	}
	var output generated.InspectionPackage
	var snapshotBytes []byte
	var expiresAt *time.Time
	var revokedAt *time.Time
	if err := api.pool.QueryRow(ctx, `
		SELECT package.id, package.inspection_id, inspection.organization_id, organization.legal_name,
		       inspection.title, package.package_version, COALESCE(package.checklist_template_version_id, ''),
		       package.package_digest, package.expires_at, package.revoked_at, package.snapshot,
		       checklist.status, checklist.revision
		FROM inspection_packages package
		JOIN inspections inspection ON inspection.id = package.inspection_id
		JOIN organizations organization ON organization.id = inspection.organization_id
		JOIN inspection_checklists checklist ON checklist.inspection_id = inspection.id
		WHERE package.id = $1
		  AND package.canonical_scope_snapshot_id IS NOT NULL
	`, packageID).Scan(
		&output.Id, &output.AuditId, &output.OrganizationId, &output.OrganizationName, &output.Title,
		&output.PackageVersion, &output.TemplateVersionId, &output.PackageDigest, &expiresAt, &revokedAt,
		&snapshotBytes, &output.ChecklistStatus, &output.ChecklistRevision,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return generated.InspectionPackage{}, application.ErrNotFound
		}
		return generated.InspectionPackage{}, err
	}
	if output.ChecklistStatus != string(checklists.StatusInProgress) {
		return generated.InspectionPackage{}, fmt.Errorf("%w: execution package is unavailable before Inspector start", application.ErrConflict)
	}
	if revokedAt != nil || (expiresAt != nil && !api.clock().UTC().Before(expiresAt.UTC())) {
		return generated.InspectionPackage{}, fmt.Errorf("%w: execution package is expired or withdrawn", application.ErrConflict)
	}
	if actor.HasRole(identity.RoleLeadInspector) {
		var assignedLead bool
		if err := api.pool.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1
				FROM audit_assignments assignment
				WHERE assignment.inspection_id = $1
				  AND assignment.lead_subject_id = $2
				  AND assignment.tombstoned_at IS NULL
			)`, output.AuditId, actor.SubjectID).Scan(&assignedLead); err != nil {
			return generated.InspectionPackage{}, err
		}
		if !assignedLead {
			return generated.InspectionPackage{}, fmt.Errorf("%w: execution package is restricted to the assigned Lead Inspector", application.ErrForbidden)
		}
	}
	inspectorOnly := actor.HasRole(identity.RoleInspector) && !actor.HasRole(identity.RoleLeadInspector)
	assignedQuestionIDs := map[string]struct{}{}
	if inspectorOnly {
		rows, err := api.pool.Query(ctx, `
			SELECT question_assignment.question_id
			FROM inspection_question_assignments question_assignment
			JOIN audit_assignments audit_assignment
			  ON audit_assignment.inspection_id = question_assignment.inspection_id
			 AND audit_assignment.tombstoned_at IS NULL
			JOIN audit_team_members team_member
			  ON team_member.assignment_id = audit_assignment.id
			 AND team_member.subject_id = question_assignment.subject_id
			 AND team_member.removed_at IS NULL
			WHERE question_assignment.inspection_id = $1 AND question_assignment.subject_id = $2
		`, output.AuditId, actor.SubjectID)
		if err != nil {
			return generated.InspectionPackage{}, err
		}
		for rows.Next() {
			var questionID string
			if err := rows.Scan(&questionID); err != nil {
				rows.Close()
				return generated.InspectionPackage{}, err
			}
			assignedQuestionIDs[questionID] = struct{}{}
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return generated.InspectionPackage{}, err
		}
		rows.Close()
		if len(assignedQuestionIDs) == 0 {
			return generated.InspectionPackage{}, fmt.Errorf("%w: Inspector has no question coverage", application.ErrForbidden)
		}
	}
	var snapshot packageSnapshot
	if err := json.Unmarshal(snapshotBytes, &snapshot); err != nil {
		return generated.InspectionPackage{}, fmt.Errorf("decode inspection package snapshot: %w", err)
	}
	output.SchemaVersion = snapshot.SchemaVersion
	output.ProtocolVersion = snapshot.ProtocolVersion
	if expiresAt != nil {
		output.ExpiresAt = expiresAt.UTC().Format(time.RFC3339Nano)
	}
	output.Questions = make([]generated.InspectionQuestion, 0, len(snapshot.Questions))
	for _, question := range snapshot.Questions {
		if inspectorOnly {
			if _, ok := assignedQuestionIDs[question.ID]; !ok {
				continue
			}
		}
		regulatoryReference := question.RegulatoryReference
		expectedEvidence := question.ExpectedEvidence
		view := generated.InspectionQuestion{
			Id: question.ID, SectionId: question.SectionID, Prompt: question.Prompt,
			RegulatoryReference: &regulatoryReference, ExpectedEvidence: &expectedEvidence,
			AllowedAnswers: []generated.ChecklistAnswer{
				generated.ChecklistAnswerCOMPLIANT, generated.ChecklistAnswerNONCOMPLIANT,
				generated.ChecklistAnswerOBSERVATION, generated.ChecklistAnswerNOTAPPLICABLE,
				generated.ChecklistAnswerNOTCHECKED,
			},
			CommentRequiredFor:       []generated.ChecklistAnswer{generated.ChecklistAnswerNONCOMPLIANT, generated.ChecklistAnswerOBSERVATION},
			AssignedInspectorUserIds: append([]string(nil), question.Assigned...),
		}
		var response generated.ChecklistResponseView
		var updatedAt time.Time
		err := api.pool.QueryRow(ctx, `
			SELECT id, question_id, response_value, COALESCE(comment_to_auditee, ''), revision, updated_at
			FROM checklist_responses WHERE inspection_id = $1 AND question_id = $2
		`, output.AuditId, question.ID).Scan(&response.Id, &response.QuestionId, &response.Answer, &response.Comment, &response.Revision, &updatedAt)
		if err == nil {
			response.UpdatedAt = updatedAt.UTC().Format(time.RFC3339Nano)
			view.CurrentResponse = &response
		} else if !errors.Is(err, pgx.ErrNoRows) {
			return generated.InspectionPackage{}, err
		}
		output.Questions = append(output.Questions, view)
	}
	return output, nil
}

func (api *CanonicalAPI) checklistResponseProjection(ctx context.Context, responseID string) (generated.ChecklistResponseView, error) {
	var view generated.ChecklistResponseView
	var updatedAt time.Time
	if err := api.pool.QueryRow(ctx, `
		SELECT id, question_id, response_value, COALESCE(comment_to_auditee, ''), revision, updated_at
		FROM checklist_responses WHERE id = $1
	`, responseID).Scan(&view.Id, &view.QuestionId, &view.Answer, &view.Comment, &view.Revision, &updatedAt); err != nil {
		return generated.ChecklistResponseView{}, err
	}
	view.UpdatedAt = updatedAt.UTC().Format(time.RFC3339Nano)
	return view, nil
}

func (api *CanonicalAPI) potentialFindingProjection(ctx context.Context, potentialFindingID string) (generated.PotentialFindingView, error) {
	var view generated.PotentialFindingView
	var converted *string
	if err := api.pool.QueryRow(ctx, `
		SELECT id, inspection_id, question_id, organization_id, title, description, status, revision, converted_finding_id
		FROM potential_findings WHERE id = $1
	`, potentialFindingID).Scan(&view.Id, &view.AuditId, &view.QuestionId, &view.OrganizationId, &view.Title,
		&view.Description, &view.Status, &view.Revision, &converted); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return generated.PotentialFindingView{}, application.ErrNotFound
		}
		return generated.PotentialFindingView{}, err
	}
	view.ConvertedFindingId = converted
	return view, nil
}

func potentialFindingView(record pfstore.PotentialFinding) generated.PotentialFindingView {
	return generated.PotentialFindingView{
		Id: record.ID, AuditId: record.InspectionID, QuestionId: record.QuestionID,
		OrganizationId: record.OrganizationID, Title: record.Title, Description: record.Description,
		Status: generated.PotentialFindingStatus(record.Status), Revision: record.Revision,
		ConvertedFindingId: record.ConvertedFindingID,
	}
}

func (api *CanonicalAPI) potentialFindingsProjection(ctx context.Context, actor identity.Principal, status *string, limit *int64) (generated.ListPotentialFindingsOutput, error) {
	if err := potentialfindings.AuthorizeList(actor); err != nil {
		return generated.ListPotentialFindingsOutput{}, fmt.Errorf("%w: %v", application.ErrForbidden, err)
	}
	resultLimit := int32(50)
	if limit != nil && *limit > 0 && *limit < 100 {
		resultLimit = int32(*limit)
	} else if limit != nil && *limit >= 100 {
		resultLimit = 100
	}
	statusFilter := ""
	if status != nil {
		statusFilter = *status
	}
	records, err := pfstore.New(api.pool).ListPotentialFindings(ctx, pfstore.ListPotentialFindingsParams{
		StatusFilter: statusFilter,
		ResultLimit:  resultLimit,
	})
	if err != nil {
		return generated.ListPotentialFindingsOutput{}, err
	}
	items := make([]generated.PotentialFindingView, 0, len(records))
	for _, record := range records {
		items = append(items, potentialFindingView(record))
	}
	return generated.ListPotentialFindingsOutput{Items: items}, nil
}

func (api *CanonicalAPI) authorizedPotentialFindingProjection(ctx context.Context, actor identity.Principal, potentialFindingID string) (generated.PotentialFindingView, error) {
	store := pfstore.New(api.pool)
	record, err := store.GetPotentialFinding(ctx, potentialFindingID)
	if errors.Is(err, pgx.ErrNoRows) {
		return generated.PotentialFindingView{}, application.ErrNotFound
	}
	if err != nil {
		return generated.PotentialFindingView{}, err
	}
	assignments, err := store.ListAssignedInspectorSubjectIDs(ctx, potentialFindingID)
	if err != nil {
		return generated.PotentialFindingView{}, err
	}
	if err := potentialfindings.AuthorizeRead(potentialfindings.ReadAuthorizationInput{
		Actor:                       actor,
		AssignedInspectorSubjectIDs: assignments,
	}); err != nil {
		return generated.PotentialFindingView{}, fmt.Errorf("%w: %v", application.ErrForbidden, err)
	}
	return potentialFindingView(record), nil
}

func (api *CanonicalAPI) findingProjection(ctx context.Context, actor identity.Principal, findingID string) (generated.FindingView, error) {
	if _, err := api.application.GetFinding(ctx, actor, findingID); err != nil {
		return generated.FindingView{}, err
	}
	var view generated.FindingView
	var ownerSubject *string
	var dueDate string
	var createdAt time.Time
	var issuedAt, closedAt *time.Time
	var closureBasis *string
	var potentialTitle, potentialDescription, findingBasis *string
	if err := api.pool.QueryRow(ctx, `
		SELECT finding.id, finding.reference, finding.inspection_id, finding.organization_id,
		       organization.legal_name, potential.title, potential.description, potential.finding_basis,
		       finding.severity, finding.status, COALESCE(finding.due_date::text, ''), finding.owner_subject_id,
		       finding.next_action, finding.cap_required, finding.evidence_required, finding.created_at,
		       finding.issued_at, finding.closed_at, finding.closure_basis, finding.revision
		FROM findings finding
		JOIN organizations organization ON organization.id = finding.organization_id
		LEFT JOIN potential_findings potential ON potential.id = finding.potential_finding_id
		WHERE finding.id = $1
	`, findingID).Scan(
		&view.Id, &view.FindingNumber, &view.AuditId, &view.OrganizationId, &view.OrganizationName,
		&potentialTitle, &potentialDescription, &findingBasis, &view.Severity, &view.Status, &dueDate,
		&ownerSubject, &view.NextAction, &view.CapRequired, &view.EvidenceRequired, &createdAt,
		&issuedAt, &closedAt, &closureBasis, &view.Revision,
	); err != nil {
		return generated.FindingView{}, err
	}
	view.Title = valueOr(potentialTitle, view.FindingNumber)
	view.Description = valueOr(potentialDescription, "Configured oversight Finding")
	view.FindingBasis = valueOr(findingBasis, "Configured check result")
	configuredReference := "Configured oversight reference"
	if view.OrganizationId == "ORG-FLY-NAMIBIA" {
		configuredReference = "Configured Cabin Inspection reference — EM EQ / PBE"
	}
	view.RegulatoryReference = &configuredReference
	if dueDate != "" {
		view.DueDate = &dueDate
	}
	view.DueState = dueState(dueDate, api.clock())
	view.CreatedAt = createdAt.UTC().Format(time.RFC3339Nano)
	view.IssuedAt = formatOptionalInstant(issuedAt)
	view.ClosedAt = formatOptionalInstant(closedAt)
	if closureBasis != nil {
		publicBasis := *closureBasis
		if publicBasis == "AUTHORIZED_CLOSURE" {
			publicBasis = "AUTHORIZED"
		}
		view.ClosureBasis = &publicBasis
	}
	view.RepeatFinding = false
	view.CurrentOwnerType, view.CurrentOwnerId, view.CurrentOwnerRole = findingOwner(view.Status, view.OrganizationId, ownerSubject, view.ClosureBasis)
	return view, nil
}

func (api *CanonicalAPI) findingsProjection(ctx context.Context, actor identity.Principal, status *string, limit *int64) (generated.ListFindingsOutput, error) {
	projections, err := api.application.ListFindings(ctx, actor)
	if err != nil {
		return generated.ListFindingsOutput{}, err
	}
	items := make([]generated.FindingView, 0, len(projections))
	for _, projection := range projections {
		view, err := api.findingProjection(ctx, actor, projection.ID)
		if err != nil {
			return generated.ListFindingsOutput{}, err
		}
		if status != nil && string(view.Status) != *status {
			continue
		}
		items = append(items, view)
	}
	sort.Slice(items, func(left, right int) bool { return items[left].FindingNumber < items[right].FindingNumber })
	if limit != nil && *limit >= 0 && int64(len(items)) > *limit {
		items = items[:*limit]
	}
	return generated.ListFindingsOutput{Items: items}, nil
}

type capReviewSummary struct {
	decision         string
	commentToAuditee string
	internalCAANote  string
	decidedAt        time.Time
}

func (api *CanonicalAPI) latestCAPReview(ctx context.Context, capRevisionID string) (*capReviewSummary, error) {
	var summary capReviewSummary
	if err := api.pool.QueryRow(ctx, `
		SELECT decision, COALESCE(comment_to_auditee, ''), COALESCE(internal_caa_note, ''), decided_at
		FROM review_decisions
		WHERE entity_type = 'cap_revision' AND entity_id = $1
		ORDER BY decided_at DESC, id DESC
		LIMIT 1
	`, capRevisionID).Scan(&summary.decision, &summary.commentToAuditee, &summary.internalCAANote, &summary.decidedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &summary, nil
}

func capDate(value any) string {
	switch typed := value.(type) {
	case interface{ TimeValue() (time.Time, bool) }:
		if t, ok := typed.TimeValue(); ok {
			return t.UTC().Format("2006-01-02")
		}
	}
	return ""
}

func capRevisionInstant(value any) string {
	switch typed := value.(type) {
	case interface{ TimeValue() (time.Time, bool) }:
		if t, ok := typed.TimeValue(); ok {
			return t.UTC().Format(time.RFC3339Nano)
		}
	}
	return ""
}

func (api *CanonicalAPI) capRevisionProjection(ctx context.Context, record capstore.CapRevision, audience caps.RevisionAudience) (generated.CapRevisionView, error) {
	review, err := api.latestCAPReview(ctx, record.ID)
	if err != nil {
		return nil, err
	}
	var superseded bool
	if err := api.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM cap_revisions
			WHERE finding_id = $1 AND revision > $2
		)
	`, record.FindingID, record.Revision).Scan(&superseded); err != nil {
		return nil, err
	}
	responsiblePerson := valueOr(record.ResponsiblePerson, "")
	commentToCAA := valueOr(record.CommentToCaa, "")
	status := generated.CapStatus(record.Status)
	if superseded {
		status = generated.CapStatus("SUPERSEDED")
	} else if review != nil {
		switch review.decision {
		case "ACCEPT":
			status = generated.CapStatus("ACCEPTED")
		case "REJECT":
			status = generated.CapStatus("REJECTED")
		case "REQUEST_MORE_INFORMATION":
			status = generated.CapStatus("MORE_INFORMATION_REQUESTED")
		}
	}
	if audience == caps.AudienceAuditee {
		view := generated.AuditeeCapRevisionView{
			Audience: "AUDITEE", Id: record.ID, CapId: record.CapID, FindingId: record.FindingID,
			OrganizationId: record.OrganizationID, Revision: int64(record.Revision), Status: status,
			RootCause: record.RootCause, CorrectiveAction: record.CorrectiveAction, PreventiveAction: record.PreventiveAction,
			ResponsiblePerson: responsiblePerson, TargetCompletionDate: record.TargetCompletionDate.Time.Format("2006-01-02"),
			CommentToCaa: commentToCAA, SubmittedAt: record.SubmittedAt.Time.UTC().Format(time.RFC3339Nano),
		}
		if review != nil {
			view.LatestReview = &generated.CapReviewDecisionSummaryAuditee{
				Decision: review.decision, CommentToAuditee: review.commentToAuditee,
				DecidedAt: review.decidedAt.UTC().Format(time.RFC3339Nano),
			}
		}
		return json.Marshal(view)
	}
	view := generated.CaaCapRevisionView{
		Audience: "CAA", Id: record.ID, CapId: record.CapID, FindingId: record.FindingID,
		OrganizationId: record.OrganizationID, Revision: int64(record.Revision), Status: status,
		RootCause: record.RootCause, CorrectiveAction: record.CorrectiveAction, PreventiveAction: record.PreventiveAction,
		ResponsiblePerson: responsiblePerson, TargetCompletionDate: record.TargetCompletionDate.Time.Format("2006-01-02"),
		CommentToCaa: commentToCAA, SubmittedAt: record.SubmittedAt.Time.UTC().Format(time.RFC3339Nano),
	}
	if review != nil {
		view.LatestReview = &generated.CapReviewDecisionSummaryCaa{
			Decision: review.decision, CommentToAuditee: review.commentToAuditee,
			InternalCaaNote: review.internalCAANote, DecidedAt: review.decidedAt.UTC().Format(time.RFC3339Nano),
		}
	}
	return json.Marshal(view)
}

func (api *CanonicalAPI) capReadAudience(ctx context.Context, actor identity.Principal, findingID string, organizationID string) (caps.RevisionAudience, error) {
	_, findingErr := api.application.GetFinding(ctx, actor, findingID)
	if findingErr != nil {
		return "", findingErr
	}
	audience, err := caps.AuthorizeRevisionRead(caps.RevisionReadAuthorizationInput{
		Actor: actor, FindingOrganizationID: organizationID, FindingAuthorized: true,
	})
	if err != nil {
		return "", fmt.Errorf("%w: %v", application.ErrForbidden, err)
	}
	return audience, nil
}

func (api *CanonicalAPI) capRevisionsProjection(ctx context.Context, actor identity.Principal, findingID string) (generated.ListCapRevisionsOutput, error) {
	records, err := capstore.New(api.pool).ListCAPRevisionsForFinding(ctx, findingID)
	if err != nil {
		return generated.ListCapRevisionsOutput{}, err
	}
	if len(records) == 0 {
		if _, err := api.application.GetFinding(ctx, actor, findingID); err != nil {
			return generated.ListCapRevisionsOutput{}, err
		}
		return generated.ListCapRevisionsOutput{Items: []generated.CapRevisionView{}}, nil
	}
	audience, err := api.capReadAudience(ctx, actor, findingID, records[0].OrganizationID)
	if err != nil {
		return generated.ListCapRevisionsOutput{}, err
	}
	items := make([]generated.CapRevisionView, 0, len(records))
	for _, record := range records {
		view, err := api.capRevisionProjection(ctx, record, audience)
		if err != nil {
			return generated.ListCapRevisionsOutput{}, err
		}
		items = append(items, view)
	}
	return generated.ListCapRevisionsOutput{Items: items}, nil
}

func (api *CanonicalAPI) capRevisionByIDProjection(ctx context.Context, actor identity.Principal, capRevisionID string) (generated.CapRevisionView, error) {
	record, err := capstore.New(api.pool).GetCAPRevision(ctx, capRevisionID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, application.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	audience, err := api.capReadAudience(ctx, actor, record.FindingID, record.OrganizationID)
	if err != nil {
		return nil, err
	}
	return api.capRevisionProjection(ctx, record, audience)
}

func (api *CanonicalAPI) reportProjection(ctx context.Context, actor identity.Principal, reportVersionID string) (generated.ReportVersionView, error) {
	var view generated.ReportVersionView
	var snapshot []byte
	var issuedAt *time.Time
	if err := api.pool.QueryRow(ctx, `
		SELECT version.id, version.report_id, version.snapshot->>'kind', inspection.organization_id, version.inspection_id,
		       version.version, version.snapshot, state.status, state.revision, state.issued_at
		FROM report_versions version
		JOIN report_approval_states state ON state.report_version_id = version.id
		JOIN inspections inspection ON inspection.id = version.inspection_id
		WHERE version.id = $1
	`, reportVersionID).Scan(&view.ReportVersionId, &view.ReportId, &view.Kind, &view.OrganizationId, &view.AuditId,
		&view.Version, &snapshot, &view.Status, &view.Revision, &issuedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return generated.ReportVersionView{}, application.ErrNotFound
		}
		return generated.ReportVersionView{}, err
	}
	if actor.HasRole(identity.RoleAuditee) {
		return generated.ReportVersionView{}, fmt.Errorf(
			"%w: report version is unavailable to this Auditee; use released Auditee Reports",
			application.ErrForbidden,
		)
	}
	if !actor.IsCAA() {
		return generated.ReportVersionView{}, application.ErrForbidden
	}
	var payload struct {
		FindingIDs                 []string `json:"findingIds"`
		PotentialFindingIDs        []string `json:"potentialFindingIds"`
		PotentialFindingRootDigest string   `json:"potentialFindingRootDigest"`
		ContentHash                string   `json:"contentHash"`
	}
	if err := json.Unmarshal(snapshot, &payload); err != nil {
		return generated.ReportVersionView{}, err
	}
	view.FindingIds = payload.FindingIDs
	view.PotentialFindingIds = payload.PotentialFindingIDs
	view.PotentialFindingRootDigest = payload.PotentialFindingRootDigest
	view.ContentHash = payload.ContentHash
	view.IssuedAt = formatOptionalInstant(issuedAt)
	return view, nil
}

type reportPublicSnapshot struct {
	Kind              string   `json:"kind"`
	Ready             bool     `json:"ready"`
	FindingIDs        []string `json:"findingIds"`
	ContentHash       string   `json:"contentHash"`
	ResponseDueDate   *string  `json:"responseDueDate"`
	CAAVisibleComment *string  `json:"caaVisibleComment"`
}

func (api *CanonicalAPI) auditeeReleasedReportProjection(
	ctx context.Context,
	actor identity.Principal,
	reportVersionID string,
) (generated.AuditeeReleasedReportView, error) {
	if !actor.HasRole(identity.RoleAuditee) || actor.OrganizationID == "" {
		return generated.AuditeeReleasedReportView{}, fmt.Errorf(
			"%w: Auditee authority is required for released report projections",
			application.ErrForbidden,
		)
	}
	var view generated.AuditeeReleasedReportView
	var snapshot []byte
	var issuedAt *time.Time
	if err := api.pool.QueryRow(ctx, `
		SELECT version.id, version.report_id, inspection.organization_id,
		       version.inspection_id, version.version, version.snapshot,
		       state.status, state.revision, state.issued_at
		FROM report_versions version
		JOIN report_approval_states state ON state.report_version_id = version.id
		JOIN inspections inspection ON inspection.id = version.inspection_id
		WHERE version.id = $1
	`, reportVersionID).Scan(
		&view.ReportVersionId, &view.ReportId, &view.OrganizationId, &view.AuditId,
		&view.Version, &snapshot, &view.Status, &view.Revision, &issuedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return generated.AuditeeReleasedReportView{}, application.ErrNotFound
		}
		return generated.AuditeeReleasedReportView{}, err
	}
	if view.OrganizationId != actor.OrganizationID || view.Status != "LOCKED" {
		return generated.AuditeeReleasedReportView{}, fmt.Errorf(
			"%w: released report version is unavailable to this Auditee",
			application.ErrForbidden,
		)
	}
	if issuedAt == nil {
		return generated.AuditeeReleasedReportView{}, fmt.Errorf(
			"%w: released report version has no issue instant",
			application.ErrForbidden,
		)
	}
	var payload reportPublicSnapshot
	if err := json.Unmarshal(snapshot, &payload); err != nil {
		return generated.AuditeeReleasedReportView{}, err
	}
	if payload.Kind != "PRELIMINARY" && payload.Kind != "FINAL" {
		return generated.AuditeeReleasedReportView{}, fmt.Errorf(
			"%w: released report has no typed public family",
			application.ErrForbidden,
		)
	}
	if len(payload.FindingIDs) > 0 {
		var authorizedCount int
		if err := api.pool.QueryRow(ctx, `
			SELECT count(*) FROM findings
			WHERE id = ANY($1::text[]) AND organization_id = $2
		`, payload.FindingIDs, actor.OrganizationID).Scan(&authorizedCount); err != nil {
			return generated.AuditeeReleasedReportView{}, err
		}
		if authorizedCount != len(payload.FindingIDs) {
			return generated.AuditeeReleasedReportView{}, fmt.Errorf(
				"%w: released report contains unavailable Finding identities",
				application.ErrForbidden,
			)
		}
	}
	view.Kind = payload.Kind
	view.FindingIds = append([]string{}, payload.FindingIDs...)
	view.IssuedAt = issuedAt.UTC().Format(time.RFC3339Nano)
	view.ResponseDueDate = payload.ResponseDueDate
	view.CaaVisibleComment = payload.CAAVisibleComment
	view.CaaVisibleCommentState = "NO_COMMENT_RECORDED"
	if payload.CAAVisibleComment != nil && strings.TrimSpace(*payload.CAAVisibleComment) != "" {
		view.CaaVisibleCommentState = "RECORDED"
	}
	return view, nil
}

func (api *CanonicalAPI) auditeeReleasedReportsProjection(
	ctx context.Context,
	actor identity.Principal,
	kind string,
) (generated.AuditeeReleasedReportPage, error) {
	if !actor.HasRole(identity.RoleAuditee) || actor.OrganizationID == "" {
		return generated.AuditeeReleasedReportPage{}, fmt.Errorf(
			"%w: Auditee authority is required for released report projections",
			application.ErrForbidden,
		)
	}
	if kind != "" && kind != "PRELIMINARY" && kind != "FINAL" {
		return generated.AuditeeReleasedReportPage{}, fmt.Errorf(
			"%w: unsupported Auditee report kind",
			application.ErrInvalid,
		)
	}
	rows, err := api.pool.Query(ctx, `
		SELECT version.id
		FROM report_versions version
		JOIN report_approval_states state ON state.report_version_id = version.id
		JOIN inspections inspection ON inspection.id = version.inspection_id
		WHERE inspection.organization_id = $1
		  AND state.status = 'LOCKED'
		  AND state.issued_at IS NOT NULL
		ORDER BY version.report_id, version.version
	`, actor.OrganizationID)
	if err != nil {
		return generated.AuditeeReleasedReportPage{}, err
	}
	defer rows.Close()
	ids := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return generated.AuditeeReleasedReportPage{}, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return generated.AuditeeReleasedReportPage{}, err
	}
	items := []generated.AuditeeReleasedReportView{}
	for _, id := range ids {
		item, err := api.auditeeReleasedReportProjection(ctx, actor, id)
		if errors.Is(err, application.ErrForbidden) {
			continue
		}
		if err != nil {
			return generated.AuditeeReleasedReportPage{}, err
		}
		if kind == "" || item.Kind == kind {
			items = append(items, item)
		}
	}
	return generated.AuditeeReleasedReportPage{Items: items}, nil
}

func (api *CanonicalAPI) documentProjection(
	ctx context.Context,
	actor identity.Principal,
	documentID string,
	includeDownload bool,
) (generated.DocumentMetadataView, error) {
	var reportID, organizationID string
	var version, revision int64
	var createdAt time.Time
	var issuedAt *time.Time
	err := api.pool.QueryRow(ctx, `
		SELECT version.report_id, inspection.organization_id, version.version,
		       state.revision, version.created_at, state.issued_at
		FROM report_versions version
		JOIN report_approval_states state ON state.report_version_id = version.id
		JOIN inspections inspection ON inspection.id = version.inspection_id
		WHERE version.id = $1
	`, documentID).Scan(
		&reportID, &organizationID, &version, &revision, &createdAt, &issuedAt,
	)
	if err == nil {
		view := generated.DocumentMetadataView{
			Id: documentID, OrganizationId: organizationID, Title: "Report " + reportID,
			Kind: "REPORT", Version: version, Revision: revision,
			CreatedAt: createdAt.UTC().Format(time.RFC3339Nano),
		}
		if actor.HasRole(identity.RoleAuditee) {
			released, err := api.auditeeReleasedReportProjection(ctx, actor, documentID)
			if err != nil {
				return generated.DocumentMetadataView{}, err
			}
			releasedLabel := "RELEASED"
			view.CreatedAt = released.IssuedAt
			view.PublicReviewResult = &releasedLabel
		} else if !actor.IsCAA() {
			return generated.DocumentMetadataView{}, application.ErrForbidden
		} else if issuedAt != nil {
			view.CreatedAt = issuedAt.UTC().Format(time.RFC3339Nano)
		}
		var renderStatus string
		var outputDocumentVersionID *string
		renderErr := api.pool.QueryRow(ctx, `
			SELECT status, output_document_version_id
			FROM document_render_jobs
			WHERE idempotency_key = 'report-render:' || $1
		`, documentID).Scan(&renderStatus, &outputDocumentVersionID)
		if errors.Is(renderErr, pgx.ErrNoRows) {
			return view, nil
		}
		if renderErr != nil {
			return generated.DocumentMetadataView{}, renderErr
		}
		view.RenderStatus = &renderStatus
		if outputDocumentVersionID == nil {
			return view, nil
		}
		var fileName, pdfHash, rendererHash, templateHash, sourceHash string
		if err := api.pool.QueryRow(ctx, `
			SELECT file_name, sha256, renderer_hash, template_hash, source_hash
			FROM document_versions
			WHERE id = $1
		`, *outputDocumentVersionID).Scan(
			&fileName, &pdfHash, &rendererHash, &templateHash, &sourceHash,
		); err != nil {
			return generated.DocumentMetadataView{}, err
		}
		view.DocumentVersionId = outputDocumentVersionID
		view.DownloadFileName = &fileName
		view.Sha256 = &pdfHash
		view.RendererHash = &rendererHash
		view.TemplateHash = &templateHash
		view.SourceHash = &sourceHash
		if includeDownload && api.documents != nil {
			download, err := api.documents.AuthorizeDownload(
				ctx, actor, *outputDocumentVersionID,
			)
			if errors.Is(err, documents.ErrForbidden) {
				return generated.DocumentMetadataView{}, application.ErrForbidden
			}
			if errors.Is(err, documents.ErrNotFound) {
				return generated.DocumentMetadataView{}, application.ErrNotFound
			}
			if errors.Is(err, documents.ErrNotReady) {
				return view, nil
			}
			if err != nil {
				return generated.DocumentMetadataView{}, err
			}
			view.DownloadUrl = &download.URL
			expiresAt := download.ExpiresAt.UTC().Format(time.RFC3339Nano)
			view.DownloadExpiresAt = &expiresAt
		}
		return view, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return generated.DocumentMetadataView{}, err
	}

	var evidence generated.DocumentMetadataView
	var reviewState string
	var submittedAt time.Time
	if err := api.pool.QueryRow(ctx, `
		SELECT version.id, version.organization_id, version.filename, version.version,
		       state.revision, state.review_state, version.submitted_at
		FROM evidence_versions version
		JOIN evidence_version_states state ON state.evidence_version_id = version.id
		WHERE version.id = $1
	`, documentID).Scan(
		&evidence.Id, &evidence.OrganizationId, &evidence.Title, &evidence.Version,
		&evidence.Revision, &reviewState, &submittedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return generated.DocumentMetadataView{}, application.ErrNotFound
		}
		return generated.DocumentMetadataView{}, err
	}
	evidence.Kind = "EVIDENCE"
	evidence.CreatedAt = submittedAt.UTC().Format(time.RFC3339Nano)
	if actor.HasRole(identity.RoleAuditee) {
		if actor.OrganizationID != evidence.OrganizationId {
			return generated.DocumentMetadataView{}, application.ErrForbidden
		}
		evidence.PublicReviewResult = &reviewState
		fileName := evidence.Title
		evidence.DownloadFileName = &fileName
	} else if !actor.IsCAA() {
		return generated.DocumentMetadataView{}, application.ErrForbidden
	}
	return evidence, nil
}

func (api *CanonicalAPI) documentsProjection(
	ctx context.Context,
	actor identity.Principal,
	organizationID string,
) (generated.ListDocumentsOutput, error) {
	if actor.HasRole(identity.RoleAuditee) {
		if organizationID != "" && organizationID != actor.OrganizationID {
			return generated.ListDocumentsOutput{}, application.ErrForbidden
		}
		organizationID = actor.OrganizationID
	} else if !actor.IsCAA() {
		return generated.ListDocumentsOutput{}, application.ErrForbidden
	}
	rows, err := api.pool.Query(ctx, `
		SELECT version.id
		FROM report_versions version
		JOIN inspections inspection ON inspection.id = version.inspection_id
		WHERE ($1 = '' OR inspection.organization_id = $1)
		ORDER BY version.created_at DESC, version.id DESC
	`, organizationID)
	if err != nil {
		return generated.ListDocumentsOutput{}, err
	}
	ids := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return generated.ListDocumentsOutput{}, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return generated.ListDocumentsOutput{}, err
	}
	rows.Close()
	evidenceRows, err := api.pool.Query(ctx, `
		SELECT id FROM evidence_versions
		WHERE ($1 = '' OR organization_id = $1)
		ORDER BY submitted_at DESC, id DESC
	`, organizationID)
	if err != nil {
		return generated.ListDocumentsOutput{}, err
	}
	for evidenceRows.Next() {
		var id string
		if err := evidenceRows.Scan(&id); err != nil {
			evidenceRows.Close()
			return generated.ListDocumentsOutput{}, err
		}
		ids = append(ids, id)
	}
	if err := evidenceRows.Err(); err != nil {
		evidenceRows.Close()
		return generated.ListDocumentsOutput{}, err
	}
	evidenceRows.Close()

	items := []generated.DocumentMetadataView{}
	for _, id := range ids {
		item, err := api.documentProjection(ctx, actor, id, false)
		if actor.HasRole(identity.RoleAuditee) && errors.Is(err, application.ErrForbidden) {
			continue
		}
		if err != nil {
			return generated.ListDocumentsOutput{}, err
		}
		items = append(items, item)
	}
	return generated.ListDocumentsOutput{Items: items}, nil
}

func (api *CanonicalAPI) managerProjection(ctx context.Context, actor identity.Principal) (generated.ManagerDashboardProjection, error) {
	if !actor.HasRole(identity.RoleDepartmentManager, identity.RoleGeneralManager, identity.RoleExecutiveDirector) {
		return generated.ManagerDashboardProjection{}, fmt.Errorf("%w: management dashboard authority is required", application.ErrForbidden)
	}
	var view generated.ManagerDashboardProjection
	if err := api.pool.QueryRow(ctx, `
		SELECT
			count(*) FILTER (WHERE status <> 'CLOSED'),
			count(*) FILTER (WHERE status = 'CLOSED'),
			count(*) FILTER (WHERE status <> 'CLOSED' AND due_date < $1::date),
			count(*) FILTER (WHERE status = 'CAP_SUBMITTED'),
			count(*) FILTER (WHERE status = 'PENDING_CAA_REVIEW'),
			(SELECT count(*) FROM report_versions WHERE status = 'DEPARTMENT_REVIEW')
		FROM findings
	`, api.clock().UTC().Format("2006-01-02")).Scan(&view.OpenFindings, &view.ClosedFindings, &view.OverdueFindings,
		&view.PendingCapReviews, &view.PendingEvidenceReviews, &view.PendingReportReviews); err != nil {
		return generated.ManagerDashboardProjection{}, err
	}
	rows, err := api.pool.Query(ctx, `SELECT reference FROM findings ORDER BY updated_at DESC, reference LIMIT 5`)
	if err != nil {
		return generated.ManagerDashboardProjection{}, err
	}
	defer rows.Close()
	view.RecentFindingNumbers = []string{}
	for rows.Next() {
		var reference string
		if err := rows.Scan(&reference); err != nil {
			return generated.ManagerDashboardProjection{}, err
		}
		view.RecentFindingNumbers = append(view.RecentFindingNumbers, reference)
	}
	view.GeneratedAt = api.clock().UTC().Format(time.RFC3339Nano)
	return view, rows.Err()
}

func dueState(dueDate string, now time.Time) generated.DueState {
	if dueDate == "" {
		return generated.DueStateNONE
	}
	due, err := time.Parse("2006-01-02", dueDate)
	if err != nil {
		return generated.DueStateNONE
	}
	today, _ := time.Parse("2006-01-02", now.UTC().Format("2006-01-02"))
	if due.Before(today) {
		return generated.DueStateOVERDUE
	}
	if due.Sub(today) <= 7*24*time.Hour {
		return generated.DueStateDUESOON
	}
	return generated.DueStateNOTDUE
}

func formatOptionalInstant(value *time.Time) *string {
	if value == nil {
		return nil
	}
	formatted := value.UTC().Format(time.RFC3339Nano)
	return &formatted
}

func valueOr(value *string, fallback string) string {
	if value == nil || *value == "" {
		return fallback
	}
	return *value
}

func findingOwner(status generated.FindingStatus, organizationID string, owner *string, closureBasis *string) (string, string, generated.Role) {
	switch status {
	case generated.FindingStatusWAITINGFORCAP, generated.FindingStatusCAPREJECTED,
		generated.FindingStatusCAPMOREINFORMATIONREQUESTED, generated.FindingStatusEVIDENCEREQUIRED,
		generated.FindingStatusEVIDENCEMOREINFORMATIONREQUESTED:
		return "AUDITEE", organizationID, generated.RoleAuditee
	case generated.FindingStatusCLOSED:
		if closureBasis != nil && *closureBasis == "AUTHORIZED" {
			return "CAA", "USR-MANAGER-NORA", generated.RoleManager
		}
		return "CAA", "USR-LEAD-CANER", generated.RoleLeadInspector
	default:
		ownerID := "USR-LEAD-CANER"
		if owner != nil && *owner != "" {
			ownerID = *owner
		}
		return "CAA", ownerID, generated.RoleLeadInspector
	}
}
