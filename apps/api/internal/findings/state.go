package findings

import (
	"fmt"
	"strings"

	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/identity"
)

type Status string
type ClosureBasis string

const (
	StatusOpen                             Status = "OPEN"
	StatusWaitingForCAP                    Status = "WAITING_FOR_CAP"
	StatusCAPSubmitted                     Status = "CAP_SUBMITTED"
	StatusCAPRejected                      Status = "CAP_REJECTED"
	StatusCAPMoreInformationRequested      Status = "CAP_MORE_INFORMATION_REQUESTED"
	StatusEvidenceRequired                 Status = "EVIDENCE_REQUIRED"
	StatusEvidenceSubmitted                Status = "EVIDENCE_SUBMITTED"
	StatusPendingCAAReview                 Status = "PENDING_CAA_REVIEW"
	StatusEvidenceMoreInformationRequested Status = "EVIDENCE_MORE_INFORMATION_REQUESTED"
	StatusPendingClosure                   Status = "PENDING_CLOSURE"
	StatusClosed                           Status = "CLOSED"

	ClosureBasisEvidenceVerified ClosureBasis = "EVIDENCE_VERIFIED"
	ClosureBasisAuthorized       ClosureBasis = "AUTHORIZED_CLOSURE"
)

type AuthorizedCloseInput struct {
	Actor            identity.Principal
	Status           Status
	Revision         int64
	ExpectedRevision int64
	Reason           string
}

type TransitionResult struct {
	Status       Status
	Revision     int64
	ClosureBasis ClosureBasis
	Reason       string
}

func AuthorizedClose(input AuthorizedCloseInput) (TransitionResult, error) {
	if !input.Actor.HasRole(identity.RoleDepartmentManager) {
		return TransitionResult{}, fmt.Errorf("only a Department Manager can authorize closure")
	}
	if input.Status == StatusClosed {
		return TransitionResult{}, fmt.Errorf("Finding is already closed")
	}
	if input.Status != StatusPendingClosure {
		return TransitionResult{}, fmt.Errorf("Finding must be pending closure after Evidence verification")
	}
	if input.Revision != input.ExpectedRevision {
		return TransitionResult{}, fmt.Errorf("stale Finding revision")
	}
	if strings.TrimSpace(input.Reason) == "" {
		return TransitionResult{}, fmt.Errorf("authorized closure reason is required")
	}
	return TransitionResult{Status: StatusClosed, Revision: input.Revision + 1, ClosureBasis: ClosureBasisAuthorized, Reason: strings.TrimSpace(input.Reason)}, nil
}
