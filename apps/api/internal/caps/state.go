package caps

import (
	"fmt"
	"strings"

	"github.com/aviason/aviaSurveil/internal/findings"
	"github.com/aviason/aviaSurveil/internal/identity"
)

type Status string
type ReviewDecision string

const (
	StatusDraft                    Status = "DRAFT"
	StatusSubmitted                Status = "SUBMITTED"
	StatusAccepted                 Status = "ACCEPTED"
	StatusRejected                 Status = "REJECTED"
	StatusMoreInformationRequested Status = "MORE_INFORMATION_REQUESTED"

	DecisionAccept             ReviewDecision = "ACCEPT"
	DecisionReject             ReviewDecision = "REJECT"
	DecisionRequestInformation ReviewDecision = "REQUEST_MORE_INFORMATION"
)

type SubmitInput struct {
	Actor                   identity.Principal
	FindingOrganizationID   string
	FindingStatus           findings.Status
	FindingRevision         int64
	ExpectedFindingRevision int64
}

type SubmitResult struct {
	CAPStatus     Status
	FindingStatus findings.Status
}

func Submit(input SubmitInput) (SubmitResult, error) {
	if !input.Actor.HasRole(identity.RoleAuditee) || !input.Actor.BelongsTo(input.FindingOrganizationID) {
		return SubmitResult{}, fmt.Errorf("Auditee is outside the Finding organization")
	}
	if input.FindingRevision != input.ExpectedFindingRevision {
		return SubmitResult{}, fmt.Errorf("stale Finding revision")
	}
	if input.FindingStatus != findings.StatusWaitingForCAP && input.FindingStatus != findings.StatusCAPRejected && input.FindingStatus != findings.StatusCAPMoreInformationRequested {
		return SubmitResult{}, fmt.Errorf("Finding is not accepting a CAP submission")
	}
	return SubmitResult{CAPStatus: StatusSubmitted, FindingStatus: findings.StatusCAPSubmitted}, nil
}

type ReviewInput struct {
	Actor               identity.Principal
	CAPStatus           Status
	CAPRevision         int64
	ExpectedCAPRevision int64
	FindingStatus       findings.Status
	EvidenceRequired    bool
	Decision            ReviewDecision
	Reason              string
}

type ReviewResult struct {
	CAPStatus     Status
	FindingStatus findings.Status
}

func Review(input ReviewInput) (ReviewResult, error) {
	if !input.Actor.HasRole(identity.RoleInspector, identity.RoleLeadInspector) {
		return ReviewResult{}, fmt.Errorf("role cannot review a CAP")
	}
	if input.CAPStatus != StatusSubmitted || input.FindingStatus != findings.StatusCAPSubmitted {
		return ReviewResult{}, fmt.Errorf("CAP is not pending CAA review")
	}
	if input.CAPRevision != input.ExpectedCAPRevision {
		return ReviewResult{}, fmt.Errorf("stale CAP revision")
	}
	switch input.Decision {
	case DecisionAccept:
		if !input.EvidenceRequired {
			return ReviewResult{CAPStatus: StatusAccepted, FindingStatus: findings.StatusPendingClosure}, nil
		}
		return ReviewResult{CAPStatus: StatusAccepted, FindingStatus: findings.StatusEvidenceRequired}, nil
	case DecisionReject:
		if strings.TrimSpace(input.Reason) == "" {
			return ReviewResult{}, fmt.Errorf("CAP rejection reason is required")
		}
		return ReviewResult{CAPStatus: StatusRejected, FindingStatus: findings.StatusCAPRejected}, nil
	case DecisionRequestInformation:
		if strings.TrimSpace(input.Reason) == "" {
			return ReviewResult{}, fmt.Errorf("CAP information request reason is required")
		}
		return ReviewResult{CAPStatus: StatusMoreInformationRequested, FindingStatus: findings.StatusCAPMoreInformationRequested}, nil
	default:
		return ReviewResult{}, fmt.Errorf("unsupported CAP review decision")
	}
}
