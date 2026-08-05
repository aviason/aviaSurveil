package agademoworkspace

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	aga "github.com/MarlonJD/aviaSurveil360/apps/api/internal/agaapplicability"
	preprod "github.com/MarlonJD/aviaSurveil360/apps/api/internal/preproddata/agademoworkspace"
)

type LifecycleInspectionState string

const (
	InspectionReady      LifecycleInspectionState = "READY"
	InspectionInProgress LifecycleInspectionState = "IN_PROGRESS"
	InspectionSubmitted  LifecycleInspectionState = "SUBMITTED"
	InspectionCompleted  LifecycleInspectionState = "COMPLETED"
)

type PotentialFindingState string

const (
	PotentialFindingPending   PotentialFindingState = "PENDING_LEAD_REVIEW"
	PotentialFindingReturned  PotentialFindingState = "RETURNED"
	PotentialFindingDismissed PotentialFindingState = "DISMISSED"
	PotentialFindingConverted PotentialFindingState = "CONVERTED"
)

type FindingState string

const (
	FindingWaitingForCAP               FindingState = "WAITING_FOR_CAP"
	FindingCAPSubmitted                FindingState = "CAP_SUBMITTED"
	FindingCAPRejected                 FindingState = "CAP_REJECTED"
	FindingCAPMoreInformationRequested FindingState = "CAP_MORE_INFORMATION_REQUESTED"
	FindingEvidenceRequired            FindingState = "EVIDENCE_REQUIRED"
	FindingEvidenceSubmitted           FindingState = "EVIDENCE_SUBMITTED"
	FindingPendingCAAReview            FindingState = "PENDING_CAA_REVIEW"
	FindingEvidenceMoreInformation     FindingState = "EVIDENCE_MORE_INFORMATION_REQUESTED"
	FindingPendingClosure              FindingState = "PENDING_CLOSURE"
	FindingClosed                      FindingState = "CLOSED"
)

type CAPRevisionState string

const (
	CAPSubmitted        CAPRevisionState = "SUBMITTED"
	CAPPendingCAAReview CAPRevisionState = "PENDING_CAA_REVIEW"
	CAPAccepted         CAPRevisionState = "ACCEPTED"
	CAPRejected         CAPRevisionState = "REJECTED"
	CAPMoreInformation  CAPRevisionState = "MORE_INFORMATION_REQUESTED"
	CAPSuperseded       CAPRevisionState = "SUPERSEDED"
)

type EvidenceReviewState string

const (
	EvidencePendingCAAReview  EvidenceReviewState = "PENDING_CAA_REVIEW"
	EvidenceAccepted          EvidenceReviewState = "ACCEPTED"
	EvidencePartiallyAccepted EvidenceReviewState = "PARTIALLY_ACCEPTED"
	EvidenceRejected          EvidenceReviewState = "REJECTED"
	EvidenceMoreInformation   EvidenceReviewState = "MORE_INFORMATION_REQUESTED"
)

type ChecklistAnswer string

const (
	AnswerCompliant     ChecklistAnswer = "COMPLIANT"
	AnswerNonCompliant  ChecklistAnswer = "NON_COMPLIANT"
	AnswerObservation   ChecklistAnswer = "OBSERVATION"
	AnswerNotApplicable ChecklistAnswer = "NOT_APPLICABLE"
	AnswerNotChecked    ChecklistAnswer = "NOT_CHECKED"
)

type EvidenceVerificationOutcome string

const (
	EvidenceClose                  EvidenceVerificationOutcome = "CLOSE"
	EvidencePartiallyClose         EvidenceVerificationOutcome = "PARTIALLY_CLOSE"
	EvidenceNotClose               EvidenceVerificationOutcome = "NOT_CLOSE"
	EvidenceRequestMoreInformation EvidenceVerificationOutcome = "REQUEST_MORE_INFORMATION"
)

type LifecycleQuestionSnapshot struct {
	QuestionKey  string                 `json:"questionKey"`
	QuestionRef  aga.QuestionRef        `json:"questionRef"`
	RootSequence int                    `json:"rootSequence"`
	Projection   aga.ProposalProjection `json:"projection"`
}

// LifecycleQuestionSnapshot carries the accepted package order beside the
// compact base QuestionRef union. Restore that immutable ordering field after
// an append-only lifecycle event is decoded from PostgreSQL JSONB.
func (question *LifecycleQuestionSnapshot) UnmarshalJSON(data []byte) error {
	type lifecycleQuestionSnapshotAlias LifecycleQuestionSnapshot
	var decoded lifecycleQuestionSnapshotAlias
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	if decoded.QuestionRef.Base != nil {
		decoded.QuestionRef.RootSequence = decoded.RootSequence
	}
	*question = LifecycleQuestionSnapshot(decoded)
	return nil
}

type LifecycleResponse struct {
	ResponseID       string          `json:"responseId"`
	QuestionKey      string          `json:"questionKey"`
	Revision         int             `json:"revision"`
	Answer           ChecklistAnswer `json:"answer"`
	CommentToAuditee string          `json:"commentToAuditee,omitempty"`
	EvidenceFileName string          `json:"evidenceFileName,omitempty"`
	ActorSubjectID   string          `json:"actorSubjectId,omitempty"`
	CreatedAt        time.Time       `json:"createdAt"`
	ResponseDigest   string          `json:"responseDigest"`
}

type LifecyclePotentialFinding struct {
	PotentialFindingID string                `json:"potentialFindingId"`
	RootID             string                `json:"rootId"`
	Version            int                   `json:"version"`
	InspectionID       string                `json:"inspectionId"`
	QuestionKey        string                `json:"questionKey"`
	ResponseID         string                `json:"responseId"`
	ResponseRevision   int                   `json:"responseRevision"`
	ResponseDigest     string                `json:"responseDigest"`
	Answer             ChecklistAnswer       `json:"answer"`
	CommentToAuditee   string                `json:"commentToAuditee"`
	State              PotentialFindingState `json:"state"`
	ReasonCode         string                `json:"reasonCode,omitempty"`
	ActorSubjectID     string                `json:"actorSubjectId,omitempty"`
	CreatedAt          time.Time             `json:"createdAt"`
	Digest             string                `json:"digest"`
}

type LifecycleFinding struct {
	FindingID              string       `json:"findingId"`
	PotentialFindingRootID string       `json:"potentialFindingRootId"`
	InspectionID           string       `json:"inspectionId"`
	QuestionKey            string       `json:"questionKey"`
	Severity               string       `json:"severity"`
	State                  FindingState `json:"state"`
	NextAction             string       `json:"nextAction"`
	CAPRequired            bool         `json:"capRequired"`
	EvidenceRequired       bool         `json:"evidenceRequired"`
	DueDateRequired        bool         `json:"dueDateRequired"`
	DueDate                *time.Time   `json:"dueDate,omitempty"`
	ClosureBasis           string       `json:"closureBasis,omitempty"`
	Revision               int          `json:"revision"`
	CreatedAt              time.Time    `json:"createdAt"`
	Digest                 string       `json:"digest"`
}

type LifecycleCAPRevision struct {
	CAPID             string           `json:"capId"`
	FindingID         string           `json:"findingId"`
	Revision          int              `json:"revision"`
	State             CAPRevisionState `json:"state"`
	RootCause         string           `json:"rootCause"`
	CorrectiveAction  string           `json:"correctiveAction"`
	PreventiveAction  string           `json:"preventiveAction"`
	ResponsiblePerson string           `json:"responsiblePerson"`
	TargetDate        *time.Time       `json:"targetDate,omitempty"`
	CommentToAuditee  string           `json:"commentToAuditee,omitempty"`
	InternalCAANote   string           `json:"internalCaaNote,omitempty"`
	ActorSubjectID    string           `json:"actorSubjectId,omitempty"`
	CreatedAt         time.Time        `json:"createdAt"`
	Digest            string           `json:"digest"`
}

type LifecycleEvidenceVersion struct {
	EvidenceID       string              `json:"evidenceId"`
	FindingID        string              `json:"findingId"`
	Version          int                 `json:"version"`
	FileName         string              `json:"fileName"`
	ReviewState      EvidenceReviewState `json:"reviewState"`
	CommentToAuditee string              `json:"commentToAuditee,omitempty"`
	InternalCAANote  string              `json:"internalCaaNote,omitempty"`
	ActorSubjectID   string              `json:"actorSubjectId,omitempty"`
	CreatedAt        time.Time           `json:"createdAt"`
	Digest           string              `json:"digest"`
}

type LifecycleVerificationDecision struct {
	VerificationID   string                      `json:"verificationId"`
	FindingID        string                      `json:"findingId"`
	EvidenceID       string                      `json:"evidenceId"`
	EvidenceVersion  int                         `json:"evidenceVersion"`
	Outcome          EvidenceVerificationOutcome `json:"outcome"`
	CommentToAuditee string                      `json:"commentToAuditee"`
	InternalCAANote  string                      `json:"internalCaaNote,omitempty"`
	ActorSubjectID   string                      `json:"actorSubjectId,omitempty"`
	CreatedAt        time.Time                   `json:"createdAt"`
	Digest           string                      `json:"digest"`
}

type LifecycleBindingPin struct {
	BindingID            string `json:"bindingId"`
	BindingRevision      int    `json:"bindingRevision"`
	SubjectID            string `json:"subjectId"`
	MembershipSlot       string `json:"membershipSlot"`
	OrganizationID       string `json:"organizationId"`
	SourceOrganizationID string `json:"sourceOrganizationId,omitempty"`
	DepartmentID         string `json:"departmentId"`
	OrganizationalUnitID string `json:"organizationalUnitId"`
}

type LifecycleAggregate struct {
	InspectionID          string                          `json:"inspectionId"`
	GenerationID          string                          `json:"generationId"`
	RecommendationID      string                          `json:"recommendationId"`
	RecommendationDigest  string                          `json:"recommendationDigest"`
	OrganizationID        string                          `json:"organizationId"`
	ProviderScopeID       string                          `json:"providerScopeId"`
	ProviderScopeVersion  int                             `json:"providerScopeVersion"`
	State                 LifecycleInspectionState        `json:"state"`
	Revision              int                             `json:"revision"`
	Inspector             LifecycleBindingPin             `json:"inspector"`
	Lead                  LifecycleBindingPin             `json:"lead"`
	Auditee               LifecycleBindingPin             `json:"auditee"`
	Questions             []LifecycleQuestionSnapshot     `json:"questions"`
	Responses             []LifecycleResponse             `json:"responses"`
	PotentialFindings     []LifecyclePotentialFinding     `json:"potentialFindings"`
	Findings              []LifecycleFinding              `json:"findings"`
	CAPRevisions          []LifecycleCAPRevision          `json:"capRevisions"`
	EvidenceVersions      []LifecycleEvidenceVersion      `json:"evidenceVersions"`
	VerificationDecisions []LifecycleVerificationDecision `json:"verificationDecisions"`
	CreatedAt             time.Time                       `json:"createdAt"`
	UpdatedAt             time.Time                       `json:"updatedAt"`
	Digest                string                          `json:"digest"`
}

// LifecycleProjection deliberately contains only the public synthetic
// inspection/checklist projection. CAA-only fields live in a different type.
type LifecycleProjection struct {
	InspectionID          string                          `json:"inspectionId"`
	GenerationID          string                          `json:"generationId"`
	OrganizationID        string                          `json:"organizationId"`
	ProviderScopeID       string                          `json:"providerScopeId"`
	State                 LifecycleInspectionState        `json:"state"`
	Revision              int                             `json:"revision"`
	Questions             []LifecycleQuestionSnapshot     `json:"questions"`
	Responses             []LifecycleResponse             `json:"responses"`
	PotentialFindings     []LifecyclePotentialFinding     `json:"potentialFindings"`
	Findings              []LifecycleFinding              `json:"findings"`
	CAPRevisions          []LifecycleCAPRevision          `json:"capRevisions"`
	EvidenceVersions      []LifecycleEvidenceVersion      `json:"evidenceVersions"`
	VerificationDecisions []LifecycleVerificationDecision `json:"verificationDecisions"`
	CurrentOwnerRole      string                          `json:"currentOwnerRole"`
	NextAction            string                          `json:"nextAction"`
	UpdatedAt             time.Time                       `json:"updatedAt"`
	Digest                string                          `json:"digest"`
}

type LifecycleCAAProjection struct {
	LifecycleProjection
	RecommendationID     string               `json:"recommendationId"`
	RecommendationDigest string               `json:"recommendationDigest"`
	Inspector            LifecycleBindingPin  `json:"inspector"`
	Lead                 LifecycleBindingPin  `json:"lead"`
	Auditee              LifecycleBindingPin  `json:"auditee"`
	RoleHistory          []LifecycleRoleEvent `json:"roleHistory"`
}

type LifecycleAuditeeProjection struct {
	LifecycleProjection
	PublicOwnerLabel string `json:"publicOwnerLabel"`
}

type LifecycleRoleEvent struct {
	Role       string    `json:"role"`
	Action     string    `json:"action"`
	OccurredAt time.Time `json:"occurredAt"`
}

type LifecycleBindingFact struct {
	BindingID            string
	BindingRevision      int
	SubjectID            string
	MembershipSlot       string
	OrganizationID       string
	SourceOrganizationID string
	DepartmentID         string
	OrganizationalUnitID string
	ProviderScopeID      string
	Active               bool
}

type LifecycleBindingResolver func(context.Context, preprod.LoadedWorkspace, aga.Recommendation, string) ([]LifecycleBindingFact, error)

var (
	ErrLifecycleNotFound            = errors.New("AGA demo lifecycle object is unavailable")
	ErrLifecycleConflict            = errors.New("AGA demo lifecycle compare-and-swap conflict")
	ErrLifecycleTransition          = errors.New("AGA demo lifecycle transition is invalid")
	ErrLifecycleBindingMismatch     = errors.New("AGA demo lifecycle binding is stale or mismatched")
	ErrLifecycleRecommendationStale = errors.New("AGA demo recommendation is stale")
	ErrLifecycleCommentRequired     = errors.New("AGA demo lifecycle comment is required")
	ErrLifecycleChoiceInvalid       = errors.New("AGA demo lifecycle choice is invalid")
)

func (aggregate LifecycleAggregate) Validate() error {
	if aggregate.InspectionID == "" || aggregate.GenerationID == "" || aggregate.RecommendationID == "" || aggregate.RecommendationDigest == "" || aggregate.OrganizationID == "" || aggregate.ProviderScopeID == "" || aggregate.ProviderScopeVersion < 1 || aggregate.Revision < 1 || aggregate.CreatedAt.IsZero() || aggregate.UpdatedAt.IsZero() || aggregate.State == "" || aggregate.Digest == "" {
		return fmt.Errorf("%w: aggregate pins", ErrLifecycleConflict)
	}
	if aggregate.Inspector.SubjectID == "" || aggregate.Lead.SubjectID == "" || aggregate.Inspector.BindingID == "" || aggregate.Lead.BindingID == "" {
		return fmt.Errorf("%w: binding pins", ErrLifecycleBindingMismatch)
	}
	if len(aggregate.Questions) == 0 {
		return fmt.Errorf("%w: empty question snapshot", ErrLifecycleRecommendationStale)
	}
	for _, question := range aggregate.Questions {
		if question.QuestionKey == "" || question.QuestionKey != question.QuestionRef.Key() || question.RootSequence < 1 || question.QuestionRef.RootSequence != question.RootSequence || aga.ValidateQuestionRef(question.QuestionRef) != nil || aga.ValidateProjection(aga.FrozenTaxonomy(), question.Projection) != nil {
			return fmt.Errorf("%w: question snapshot", ErrLifecycleRecommendationStale)
		}
	}
	return nil
}

func validChecklistAnswer(value ChecklistAnswer) bool {
	switch value {
	case AnswerCompliant, AnswerNonCompliant, AnswerObservation, AnswerNotApplicable, AnswerNotChecked:
		return true
	default:
		return false
	}
}

func findingEligibleAnswer(value ChecklistAnswer) bool {
	return value == AnswerNonCompliant || value == AnswerObservation
}

func isTerminalPotentialFinding(state PotentialFindingState) bool {
	return state == PotentialFindingDismissed || state == PotentialFindingConverted
}

func (aggregate LifecycleAggregate) latestResponse(questionKey string) (LifecycleResponse, bool) {
	var result LifecycleResponse
	found := false
	for _, response := range aggregate.Responses {
		if response.QuestionKey == questionKey && (!found || response.Revision > result.Revision) {
			result, found = response, true
		}
	}
	return result, found
}

func (aggregate LifecycleAggregate) latestPotential(rootID string) (LifecyclePotentialFinding, bool) {
	var result LifecyclePotentialFinding
	found := false
	for _, potential := range aggregate.PotentialFindings {
		if potential.RootID == rootID && (!found || potential.Version > result.Version) {
			result, found = potential, true
		}
	}
	return result, found
}

func (aggregate LifecycleAggregate) latestFinding(findingID string) (LifecycleFinding, bool) {
	for _, finding := range aggregate.Findings {
		if finding.FindingID == findingID {
			return finding, true
		}
	}
	return LifecycleFinding{}, false
}

func (aggregate LifecycleAggregate) latestCAP(findingID string) (LifecycleCAPRevision, bool) {
	var result LifecycleCAPRevision
	found := false
	for _, cap := range aggregate.CAPRevisions {
		if cap.FindingID == findingID && (!found || cap.Revision >= result.Revision) {
			result, found = cap, true
		}
	}
	return result, found
}

func (aggregate LifecycleAggregate) latestEvidence(findingID string) (LifecycleEvidenceVersion, bool) {
	var result LifecycleEvidenceVersion
	found := false
	for _, evidence := range aggregate.EvidenceVersions {
		if evidence.FindingID == findingID && (!found || evidence.Version >= result.Version) {
			result, found = evidence, true
		}
	}
	return result, found
}

func (aggregate LifecycleAggregate) questionExists(questionKey string) bool {
	for _, question := range aggregate.Questions {
		if question.QuestionRef.Key() == questionKey || question.QuestionRef.Key() == strings.TrimSpace(questionKey) {
			return true
		}
	}
	return false
}

func marshalLifecycle(value any) json.RawMessage {
	data, _ := json.Marshal(value)
	return data
}
