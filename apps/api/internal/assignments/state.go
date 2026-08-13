package assignments

import (
	"time"

	"github.com/aviason/aviaSurveil/internal/identity"
)

type Status string

const (
	StatusPreparation                 Status = "PREPARATION"
	StatusLeadAssigned                Status = "LEAD_ASSIGNED"
	StatusTeamAssigned                Status = "TEAM_ASSIGNED"
	StatusQuestionsAssigned           Status = "QUESTIONS_ASSIGNED"
	StatusAwaitingAuditeeConfirmation Status = "AWAITING_AUDITEE_CONFIRMATION"
	StatusConfirmed                   Status = "CONFIRMED"
	StatusAlternativeProposed         Status = "ALTERNATIVE_PROPOSED"
	StatusScheduled                   Status = "SCHEDULED"
	StatusReady                       Status = "READY"
)

type Preparation struct {
	AssignmentID                string    `json:"assignmentId"`
	PlanningItemID              string    `json:"planningItemId"`
	InspectionID                string    `json:"inspectionId"`
	OrganizationID              string    `json:"organizationId"`
	Status                      Status    `json:"status"`
	Revision                    int64     `json:"revision"`
	PreparationID               string    `json:"preparationId,omitempty"`
	PreparationDigest           string    `json:"preparationDigest,omitempty"`
	SelectedQuestionCount       int       `json:"selectedQuestionCount,omitempty"`
	ConfirmedAt                 time.Time `json:"confirmedAt,omitempty"`
	ConfirmedAssignmentRevision int64     `json:"confirmedAssignmentRevision,omitempty"`
}

type Assignment struct {
	ID                      string               `json:"id"`
	PlanningItemID          string               `json:"planningItemId"`
	ReleasedScopeSnapshotID string               `json:"releasedScopeSnapshotId,omitempty"`
	InspectionID            string               `json:"inspectionId"`
	OrganizationID          string               `json:"organizationId"`
	LeadSubjectID           string               `json:"leadSubjectId"`
	MemberSubjectIDs        []string             `json:"memberSubjectIds"`
	QuestionAssignments     []QuestionAssignment `json:"questionAssignments"`
	// SelectedQuestionVersionIDs is the immutable released-scope question set.
	// It is exposed to the pre-materialization Lead workspace so coverage can
	// be assigned before an execution package exists.
	SelectedQuestionVersionIDs []string `json:"selectedQuestionVersionIds,omitempty"`
	Status                     Status   `json:"status"`
	ScheduledStartDate         string   `json:"scheduledStartDate"`
	ScheduledEndDate           string   `json:"scheduledEndDate"`
	Revision                   int64    `json:"revision"`
	// Preparation confirmation is an immutable receipt bound to the exact
	// assignment revision that the Department Manager confirmed.  It is part
	// of the restart-safe projection so materialization remains reachable after
	// a browser/session restart without re-confirming or trusting local state.
	PreparationID                          string     `json:"preparationId,omitempty"`
	PreparationDigest                      string     `json:"preparationDigest,omitempty"`
	PreparationConfirmedAt                 *time.Time `json:"preparationConfirmedAt,omitempty"`
	PreparationConfirmedAssignmentRevision int64      `json:"preparationConfirmedAssignmentRevision,omitempty"`
}

type QuestionAssignment struct {
	QuestionID string `json:"questionId"`
	SubjectID  string `json:"subjectId"`
}

type QuestionCoverageOperationKind string

const (
	QuestionCoverageAdd     QuestionCoverageOperationKind = "ADD"
	QuestionCoverageRemove  QuestionCoverageOperationKind = "REMOVE"
	QuestionCoverageReplace QuestionCoverageOperationKind = "REPLACE"
)

// PreparationEditPreview is a short-lived, server-issued receipt for a
// complete team or question-coverage set.  The receipt is consumed exactly
// once by the matching command; the mutable assignment tables remain only a
// projection of that immutable command snapshot.
type PreparationEditPreview struct {
	PreviewID           string               `json:"previewId"`
	AssignmentID        string               `json:"assignmentId"`
	AssignmentRevision  int64                `json:"assignmentRevision"`
	EditKind            string               `json:"editKind"`
	Digest              string               `json:"digest"`
	ExpiresAt           time.Time            `json:"expiresAt"`
	MemberSubjectIDs    []string             `json:"memberSubjectIds,omitempty"`
	QuestionAssignments []QuestionAssignment `json:"questionAssignments,omitempty"`
}

type AuditeeCoordination struct {
	InspectionID       string  `json:"auditId"`
	OrganizationID     string  `json:"organizationId"`
	OrganizationName   string  `json:"organizationName"`
	Title              string  `json:"title"`
	InspectionCategory string  `json:"inspectionCategory"`
	ScheduledStartDate string  `json:"scheduledStartDate"`
	Status             Status  `json:"status"`
	AlternativeDate    *string `json:"alternativeDate"`
	NextAction         string  `json:"nextAction"`
	Revision           int64   `json:"revision"`
}

type MaterializedInspection struct {
	InspectionID       string    `json:"inspectionId"`
	AssignmentID       string    `json:"assignmentId"`
	PackageID          string    `json:"packageId"`
	TemplateVersionID  string    `json:"templateVersionId"`
	PackageVersion     int64     `json:"packageVersion"`
	PackageDigest      string    `json:"packageDigest"`
	Status             Status    `json:"status"`
	NoticeWithheld     bool      `json:"noticeWithheld"`
	AssignmentRevision int64     `json:"assignmentRevision"`
	ExpiresAt          time.Time `json:"expiresAt"`
}

type TeamMember struct {
	SubjectID      string        `json:"subjectId"`
	DisplayName    string        `json:"displayName"`
	Role           identity.Role `json:"role"`
	OrganizationID *string       `json:"organizationId"`
	Revision       int64         `json:"revision"`
}

type TeamQuestionAssignment struct {
	QuestionID               string   `json:"questionId"`
	AssignedMemberSubjectIDs []string `json:"assignedMemberSubjectIds"`
}

type TeamHistory struct {
	EventID        string    `json:"eventId"`
	OccurredAt     time.Time `json:"occurredAt"`
	ActorSubjectID string    `json:"actorSubjectId"`
	Action         string    `json:"action"`
	Detail         string    `json:"detail"`
}

type TeamAudit struct {
	AuditID            string                   `json:"auditId"`
	OrganizationID     string                   `json:"organizationId"`
	OrganizationName   string                   `json:"organizationName"`
	Title              string                   `json:"title"`
	Status             string                   `json:"status"`
	ScheduledStartDate *string                  `json:"scheduledStartDate"`
	ScheduledEndDate   *string                  `json:"scheduledEndDate"`
	LeadInspector      TeamMember               `json:"leadInspector"`
	Members            []TeamMember             `json:"members"`
	Assignments        []TeamQuestionAssignment `json:"assignments"`
	History            []TeamHistory            `json:"history"`
	Revision           int64                    `json:"revision"`
}
