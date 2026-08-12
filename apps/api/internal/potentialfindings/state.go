package potentialfindings

import (
	"fmt"
	"strings"

	"github.com/aviason/aviaSurveil/internal/identity"
)

type Status string
type Decision string
type Severity string

const (
	StatusPendingLeadReview Status = "PENDING_LEAD_REVIEW"
	StatusReturned          Status = "RETURNED"
	StatusDismissed         Status = "DISMISSED"
	StatusConverted         Status = "CONVERTED"

	DecisionReturn  Decision = "RETURN"
	DecisionDismiss Decision = "DISMISS"
	DecisionConvert Decision = "CONVERT"

	SeverityLevel1Critical Severity = "LEVEL_1_CRITICAL"
	SeverityLevel2Major    Severity = "LEVEL_2_MAJOR"
	SeverityLevel3Minor    Severity = "LEVEL_3_MINOR"
	SeverityObservation    Severity = "OBSERVATION"
)

type DecideInput struct {
	Actor            identity.Principal
	Status           Status
	Revision         int64
	ExpectedRevision int64
	Decision         Decision
	Reason           string
	Severity         Severity
}

type DecideResult struct {
	Status        Status
	Revision      int64
	CreateFinding bool
	Severity      Severity
}

func Decide(input DecideInput) (DecideResult, error) {
	if !input.Actor.HasRole(identity.RoleLeadInspector) {
		return DecideResult{}, fmt.Errorf("only a Lead Inspector can decide a Potential Finding")
	}
	if input.Status != StatusPendingLeadReview {
		return DecideResult{}, fmt.Errorf("Potential Finding is not pending Lead review")
	}
	if input.Revision != input.ExpectedRevision {
		return DecideResult{}, fmt.Errorf("stale Potential Finding revision")
	}
	result := DecideResult{Revision: input.Revision + 1}
	switch input.Decision {
	case DecisionReturn:
		if strings.TrimSpace(input.Reason) == "" {
			return DecideResult{}, fmt.Errorf("return reason is required")
		}
		result.Status = StatusReturned
	case DecisionDismiss:
		if strings.TrimSpace(input.Reason) == "" {
			return DecideResult{}, fmt.Errorf("dismissal reason is required")
		}
		result.Status = StatusDismissed
	case DecisionConvert:
		if !validSeverity(input.Severity) {
			return DecideResult{}, fmt.Errorf("approved Finding severity is required")
		}
		result.Status = StatusConverted
		result.CreateFinding = true
		result.Severity = input.Severity
	default:
		return DecideResult{}, fmt.Errorf("unsupported Potential Finding decision")
	}
	return result, nil
}

func validSeverity(severity Severity) bool {
	return severity == SeverityLevel1Critical || severity == SeverityLevel2Major || severity == SeverityLevel3Minor || severity == SeverityObservation
}
