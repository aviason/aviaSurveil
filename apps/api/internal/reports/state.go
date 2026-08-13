package reports

import (
	"fmt"
	"strings"

	"github.com/aviason/aviaSurveil/internal/identity"
)

type Status string
type Decision string
type Kind string

const (
	KindPreliminary Kind = "PRELIMINARY"
	KindFinal       Kind = "FINAL"

	StatusDraft                   Status   = "DRAFT"
	StatusDepartmentReview        Status   = "DEPARTMENT_REVIEW"
	StatusGeneralManagerReview    Status   = "GM_REVIEW"
	StatusExecutiveDirectorReview Status   = "EXECUTIVE_DIRECTOR_REVIEW"
	StatusReturned                Status   = "RETURNED"
	StatusIssued                  Status   = "ISSUED"
	StatusLocked                  Status   = "LOCKED"
	DecisionForward               Decision = "FORWARD"
	DecisionReturn                Decision = "RETURN"
	DecisionIssue                 Decision = "ISSUE"
)

type PrepareInput struct {
	ReportID    string
	Kind        Kind
	Version     int64
	FindingIDs  []string
	ContentHash string
	Ready       bool
}

type PreparedVersion struct {
	ReportID    string
	Kind        Kind
	Version     int64
	FindingIDs  []string
	ContentHash string
	Status      Status
}

func Prepare(input PrepareInput) (PreparedVersion, error) {
	if strings.TrimSpace(input.ReportID) == "" || input.Version <= 0 {
		return PreparedVersion{}, fmt.Errorf("report identity and positive version are required")
	}
	if input.Kind != KindPreliminary && input.Kind != KindFinal {
		return PreparedVersion{}, fmt.Errorf("typed Preliminary or Final Report family is required")
	}
	if !input.Ready {
		return PreparedVersion{}, fmt.Errorf("report preparation is not ready for Department review")
	}
	if !strings.HasPrefix(strings.TrimSpace(input.ContentHash), "sha256:") {
		return PreparedVersion{}, fmt.Errorf("report content hash is required")
	}
	findingIDs := append([]string(nil), input.FindingIDs...)
	for _, findingID := range findingIDs {
		if strings.TrimSpace(findingID) == "" {
			return PreparedVersion{}, fmt.Errorf("report Finding identities cannot be empty")
		}
	}
	return PreparedVersion{
		ReportID: input.ReportID, Kind: input.Kind, Version: input.Version,
		FindingIDs: findingIDs, ContentHash: input.ContentHash,
		Status: StatusDepartmentReview,
	}, nil
}

type DecideInput struct {
	Actor           identity.Principal
	Status          Status
	Version         int64
	ExpectedVersion int64
	Decision        Decision
	Reason          string
}

type DecideResult struct {
	Status Status
}

func Decide(input DecideInput) (DecideResult, error) {
	if input.Version != input.ExpectedVersion {
		return DecideResult{}, fmt.Errorf("stale report version")
	}
	switch input.Status {
	case StatusDepartmentReview:
		if !input.Actor.HasRole(identity.RoleDepartmentManager) {
			return DecideResult{}, fmt.Errorf("report is outside Department Manager authority")
		}
		return returnOrForward(input, StatusGeneralManagerReview)
	case StatusGeneralManagerReview:
		if !input.Actor.HasRole(identity.RoleGeneralManager) {
			return DecideResult{}, fmt.Errorf("report is outside General Manager authority")
		}
		return returnOrForward(input, StatusExecutiveDirectorReview)
	case StatusExecutiveDirectorReview:
		if !input.Actor.HasRole(identity.RoleExecutiveDirector) || input.Decision != DecisionIssue {
			return DecideResult{}, fmt.Errorf("only Executive Director can issue this report")
		}
		return DecideResult{Status: StatusLocked}, nil
	default:
		return DecideResult{}, fmt.Errorf("report stage is not decidable")
	}
}

func returnOrForward(input DecideInput, forward Status) (DecideResult, error) {
	switch input.Decision {
	case DecisionForward:
		return DecideResult{Status: forward}, nil
	case DecisionReturn:
		if strings.TrimSpace(input.Reason) == "" {
			return DecideResult{}, fmt.Errorf("report return reason is required")
		}
		return DecideResult{Status: StatusReturned}, nil
	default:
		return DecideResult{}, fmt.Errorf("unsupported report decision at this stage")
	}
}
