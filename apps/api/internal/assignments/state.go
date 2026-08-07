package assignments

import (
	"time"

	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/identity"
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
	AssignmentID          string    `json:"assignmentId"`
	PlanningItemID        string    `json:"planningItemId"`
	InspectionID          string    `json:"inspectionId"`
	OrganizationID        string    `json:"organizationId"`
	Status                Status    `json:"status"`
	Revision              int64     `json:"revision"`
	PreparationID         string    `json:"preparationId,omitempty"`
	PreparationDigest     string    `json:"preparationDigest,omitempty"`
	SelectedQuestionCount int       `json:"selectedQuestionCount,omitempty"`
	ConfirmedAt           time.Time `json:"confirmedAt,omitempty"`
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
	Status                  Status               `json:"status"`
	ScheduledStartDate      string               `json:"scheduledStartDate"`
	ScheduledEndDate        string               `json:"scheduledEndDate"`
	Revision                int64                `json:"revision"`
}

type QuestionAssignment struct {
	QuestionID string `json:"questionId"`
	SubjectID  string `json:"subjectId"`
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
