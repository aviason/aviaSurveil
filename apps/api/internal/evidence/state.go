package evidence

import (
	"fmt"

	"github.com/aviason/aviaSurveil/internal/findings"
	"github.com/aviason/aviaSurveil/internal/identity"
)

type ScanStatus string
type Decision string

const (
	ScanPending     ScanStatus = "PENDING"
	ScanClean       ScanStatus = "CLEAN"
	ScanQuarantined ScanStatus = "QUARANTINED"

	DecisionClose              Decision = "CLOSE"
	DecisionPartiallyClose     Decision = "PARTIALLY_CLOSE"
	DecisionNotClose           Decision = "NOT_CLOSE"
	DecisionRequestInformation Decision = "REQUEST_MORE_INFORMATION"
)

type ReviewInput struct {
	Actor                   identity.Principal
	VersionID               string
	VersionRevision         int64
	ExpectedVersionRevision int64
	ScanStatus              ScanStatus
	FindingStatus           findings.Status
	Decision                Decision
}

type ReviewResult struct {
	FindingStatus findings.Status
	ClosureBasis  findings.ClosureBasis
}

func Review(input ReviewInput) (ReviewResult, error) {
	if !input.Actor.HasRole(identity.RoleInspector, identity.RoleLeadInspector) {
		return ReviewResult{}, fmt.Errorf("role cannot review Evidence")
	}
	if input.VersionID == "" || input.VersionRevision != input.ExpectedVersionRevision {
		return ReviewResult{}, fmt.Errorf("exact Evidence version revision is required")
	}
	if input.ScanStatus != ScanClean {
		return ReviewResult{}, fmt.Errorf("Evidence version is not scan-clean")
	}
	if input.FindingStatus != findings.StatusPendingCAAReview && input.FindingStatus != findings.StatusEvidenceSubmitted {
		return ReviewResult{}, fmt.Errorf("Finding is not pending Evidence review")
	}
	switch input.Decision {
	case DecisionClose:
		return ReviewResult{FindingStatus: findings.StatusClosed, ClosureBasis: findings.ClosureBasisEvidenceVerified}, nil
	case DecisionPartiallyClose:
		return ReviewResult{FindingStatus: findings.StatusEvidenceMoreInformationRequested}, nil
	case DecisionNotClose:
		return ReviewResult{FindingStatus: findings.StatusEvidenceMoreInformationRequested}, nil
	case DecisionRequestInformation:
		return ReviewResult{FindingStatus: findings.StatusEvidenceMoreInformationRequested}, nil
	default:
		return ReviewResult{}, fmt.Errorf("unsupported Evidence review decision")
	}
}
