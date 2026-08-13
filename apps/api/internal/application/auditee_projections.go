package application

import (
	"context"
	"fmt"
	"time"

	"github.com/aviason/aviaSurveil/internal/identity"
)

type AuditeeCAPProjection struct {
	ID                   string    `json:"id"`
	FindingID            string    `json:"findingId"`
	Revision             int64     `json:"revision"`
	SubmissionStatus     string    `json:"submissionStatus"`
	ReviewStatus         string    `json:"reviewStatus"`
	RootCause            string    `json:"rootCause"`
	CorrectiveAction     string    `json:"correctiveAction"`
	PreventiveAction     string    `json:"preventiveAction"`
	ResponsiblePerson    string    `json:"responsiblePerson"`
	TargetCompletionDate time.Time `json:"targetCompletionDate"`
	CommentToCAA         string    `json:"commentToCAA,omitempty"`
	ReviewComment        string    `json:"reviewComment,omitempty"`
}

type AuditeeEvidenceProjection struct {
	ID          string    `json:"id"`
	FindingID   string    `json:"findingId"`
	Version     int64     `json:"version"`
	Filename    string    `json:"filename"`
	MediaType   string    `json:"mediaType"`
	SizeBytes   int64     `json:"sizeBytes"`
	Status      string    `json:"status"`
	SubmittedAt time.Time `json:"submittedAt"`
}

type AuditeeReportProjection struct {
	ID           string     `json:"id"`
	ReportID     string     `json:"reportId"`
	InspectionID string     `json:"inspectionId"`
	Version      int64      `json:"version"`
	Status       string     `json:"status"`
	IssuedAt     *time.Time `json:"issuedAt,omitempty"`
}

type AuditeeAuditProjection struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Status  string `json:"status"`
	DueDate string `json:"dueDate,omitempty"`
}

type AuditeeChangeProjection struct {
	SequenceID     int64  `json:"sequenceId"`
	Kind           string `json:"kind"`
	EntityID       string `json:"entityId"`
	EntityRevision int64  `json:"entityRevision"`
}

type AuditeeDashboardProjection struct {
	OpenFindings   int `json:"openFindings"`
	ClosedFindings int `json:"closedFindings"`
	Overdue        int `json:"overdue"`
}

type AuditeeWorkspaceProjection struct {
	Findings  []FindingProjection         `json:"findings"`
	CAPs      []AuditeeCAPProjection      `json:"caps"`
	Evidence  []AuditeeEvidenceProjection `json:"evidence"`
	Reports   []AuditeeReportProjection   `json:"reports"`
	Audits    []AuditeeAuditProjection    `json:"audits"`
	Changes   []AuditeeChangeProjection   `json:"changes"`
	Dashboard AuditeeDashboardProjection  `json:"dashboard"`
}

type ConflictProjection struct {
	Code             string `json:"code"`
	EntityType       string `json:"entityType"`
	EntityID         string `json:"entityId"`
	ExpectedRevision int64  `json:"expectedRevision"`
	ActualRevision   int64  `json:"actualRevision"`
	Message          string `json:"message"`
}

func NewSafeConflict(entityType, entityID string, expectedRevision, actualRevision int64) ConflictProjection {
	return ConflictProjection{
		Code: "REVISION_CONFLICT", EntityType: entityType, EntityID: entityID,
		ExpectedRevision: expectedRevision, ActualRevision: actualRevision,
		Message: "The record changed on the server. Review the latest authorized version and retry.",
	}
}

func (service *Service) GetAuditeeWorkspace(ctx context.Context, actor identity.Principal) (AuditeeWorkspaceProjection, error) {
	if !actor.HasRole(identity.RoleAuditee) || actor.SubjectID == "" || actor.OrganizationID == "" {
		return AuditeeWorkspaceProjection{}, ErrForbidden
	}
	findings, err := service.ListFindings(ctx, actor)
	if err != nil {
		return AuditeeWorkspaceProjection{}, err
	}
	workspace := AuditeeWorkspaceProjection{
		Findings: findings,
		CAPs:     []AuditeeCAPProjection{}, Evidence: []AuditeeEvidenceProjection{},
		Reports: []AuditeeReportProjection{}, Audits: []AuditeeAuditProjection{}, Changes: []AuditeeChangeProjection{},
	}

	capRows, err := service.pool.Query(ctx, `
		SELECT cap.id, cap.finding_id, cap.revision, cap.status,
		       COALESCE(review.decision, ''), cap.root_cause, cap.corrective_action, cap.preventive_action,
		       COALESCE(cap.responsible_person, ''), cap.target_completion_date,
		       COALESCE(cap.comment_to_caa, ''), COALESCE(review.comment_to_auditee, '')
		FROM cap_revisions cap
		LEFT JOIN LATERAL (
			SELECT decision, comment_to_auditee
			FROM review_decisions
			WHERE entity_type = 'cap_revision' AND entity_id = cap.id
			ORDER BY decided_at DESC, id DESC LIMIT 1
		) review ON true
		WHERE cap.organization_id = $1
		ORDER BY cap.finding_id, cap.revision
	`, actor.OrganizationID)
	if err != nil {
		return AuditeeWorkspaceProjection{}, fmt.Errorf("list Auditee CAP projections: %w", err)
	}
	for capRows.Next() {
		var item AuditeeCAPProjection
		var decision string
		if err := capRows.Scan(
			&item.ID, &item.FindingID, &item.Revision, &item.SubmissionStatus, &decision,
			&item.RootCause, &item.CorrectiveAction, &item.PreventiveAction, &item.ResponsiblePerson,
			&item.TargetCompletionDate, &item.CommentToCAA, &item.ReviewComment,
		); err != nil {
			capRows.Close()
			return AuditeeWorkspaceProjection{}, fmt.Errorf("scan Auditee CAP projection: %w", err)
		}
		item.ReviewStatus = publicCAPReviewStatus(decision)
		workspace.CAPs = append(workspace.CAPs, item)
	}
	if err := capRows.Err(); err != nil {
		capRows.Close()
		return AuditeeWorkspaceProjection{}, fmt.Errorf("iterate Auditee CAP projections: %w", err)
	}
	capRows.Close()

	evidenceRows, err := service.pool.Query(ctx, `
		SELECT id, finding_id, version, filename, media_type, size_bytes, status, submitted_at
		FROM evidence_versions WHERE organization_id = $1 ORDER BY finding_id, version
	`, actor.OrganizationID)
	if err != nil {
		return AuditeeWorkspaceProjection{}, fmt.Errorf("list Auditee Evidence projections: %w", err)
	}
	for evidenceRows.Next() {
		var item AuditeeEvidenceProjection
		if err := evidenceRows.Scan(&item.ID, &item.FindingID, &item.Version, &item.Filename, &item.MediaType, &item.SizeBytes, &item.Status, &item.SubmittedAt); err != nil {
			evidenceRows.Close()
			return AuditeeWorkspaceProjection{}, fmt.Errorf("scan Auditee Evidence projection: %w", err)
		}
		workspace.Evidence = append(workspace.Evidence, item)
	}
	if err := evidenceRows.Err(); err != nil {
		evidenceRows.Close()
		return AuditeeWorkspaceProjection{}, fmt.Errorf("iterate Auditee Evidence projections: %w", err)
	}
	evidenceRows.Close()

	reportRows, err := service.pool.Query(ctx, `
		SELECT version.id, version.report_id, version.inspection_id, version.version, state.status, state.issued_at
		FROM report_versions version
		JOIN report_approval_states state ON state.report_version_id = version.id
		JOIN inspections inspection ON inspection.id = version.inspection_id
		WHERE inspection.organization_id = $1 AND state.status = 'LOCKED'
		ORDER BY version.report_id, version.version
	`, actor.OrganizationID)
	if err != nil {
		return AuditeeWorkspaceProjection{}, fmt.Errorf("list released Auditee reports: %w", err)
	}
	for reportRows.Next() {
		var item AuditeeReportProjection
		if err := reportRows.Scan(&item.ID, &item.ReportID, &item.InspectionID, &item.Version, &item.Status, &item.IssuedAt); err != nil {
			reportRows.Close()
			return AuditeeWorkspaceProjection{}, fmt.Errorf("scan released Auditee report: %w", err)
		}
		workspace.Reports = append(workspace.Reports, item)
	}
	if err := reportRows.Err(); err != nil {
		reportRows.Close()
		return AuditeeWorkspaceProjection{}, fmt.Errorf("iterate released Auditee reports: %w", err)
	}
	reportRows.Close()

	auditRows, err := service.pool.Query(ctx, `
		SELECT id, title, status, COALESCE(due_date::text, '')
		FROM inspections WHERE organization_id = $1 ORDER BY due_date, id
	`, actor.OrganizationID)
	if err != nil {
		return AuditeeWorkspaceProjection{}, fmt.Errorf("list Auditee Audit projections: %w", err)
	}
	for auditRows.Next() {
		var item AuditeeAuditProjection
		if err := auditRows.Scan(&item.ID, &item.Title, &item.Status, &item.DueDate); err != nil {
			auditRows.Close()
			return AuditeeWorkspaceProjection{}, fmt.Errorf("scan Auditee Audit projection: %w", err)
		}
		workspace.Audits = append(workspace.Audits, item)
	}
	if err := auditRows.Err(); err != nil {
		auditRows.Close()
		return AuditeeWorkspaceProjection{}, fmt.Errorf("iterate Auditee Audit projections: %w", err)
	}
	auditRows.Close()

	changeRows, err := service.pool.Query(ctx, `
		SELECT sequence_id, kind, entity_id, COALESCE(entity_revision, 0)
		FROM authorized_sync_changes
		WHERE subject_id = $1 AND organization_id = $2
		ORDER BY sequence_id
	`, actor.SubjectID, actor.OrganizationID)
	if err != nil {
		return AuditeeWorkspaceProjection{}, fmt.Errorf("list Auditee change projections: %w", err)
	}
	for changeRows.Next() {
		var item AuditeeChangeProjection
		if err := changeRows.Scan(&item.SequenceID, &item.Kind, &item.EntityID, &item.EntityRevision); err != nil {
			changeRows.Close()
			return AuditeeWorkspaceProjection{}, fmt.Errorf("scan Auditee change projection: %w", err)
		}
		workspace.Changes = append(workspace.Changes, item)
	}
	if err := changeRows.Err(); err != nil {
		changeRows.Close()
		return AuditeeWorkspaceProjection{}, fmt.Errorf("iterate Auditee change projections: %w", err)
	}
	changeRows.Close()

	nowDate := service.clock().UTC().Format("2006-01-02")
	for _, finding := range findings {
		if finding.Status == "CLOSED" {
			workspace.Dashboard.ClosedFindings++
		} else {
			workspace.Dashboard.OpenFindings++
		}
		if finding.DueDate != "" && finding.DueDate < nowDate && finding.Status != "CLOSED" {
			workspace.Dashboard.Overdue++
		}
	}
	return workspace, nil
}

func publicCAPReviewStatus(decision string) string {
	switch decision {
	case "ACCEPT":
		return "ACCEPTED"
	case "REJECT":
		return "REJECTED"
	case "REQUEST_MORE_INFORMATION":
		return "MORE_INFORMATION_REQUESTED"
	default:
		return "PENDING_CAA_REVIEW"
	}
}
