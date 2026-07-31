package regulatory

import (
	"errors"
	"sort"
	"strings"
)

var ErrHoldoutOverlap = errors.New("blind holdout overlaps generation input")

type HoldoutOutcome string

const (
	HoldoutSupported   HoldoutOutcome = "SUPPORTED"
	HoldoutUnsupported HoldoutOutcome = "UNSUPPORTED"
	HoldoutMissed      HoldoutOutcome = "MISSED"
	HoldoutCorrected   HoldoutOutcome = "MANAGER_CORRECTED"
)

type HoldoutReview struct {
	StableRowIdentity string
	Outcome           HoldoutOutcome
}
type HoldoutEvaluationInput struct {
	GenerationInputStableRowIDs, HoldoutStableRowIDs []string
	Reviewed                                         []HoldoutReview
}
type HoldoutEvaluationResult struct{ ReviewedCount, SupportedCount, UnsupportedCount, MissedCount, ManagerCorrectionCount int }

// EvaluateBlindHoldout admits only exact reserved identities and rejects every
// input/holdout overlap independently of source filename or hash.
func EvaluateBlindHoldout(input HoldoutEvaluationInput) (HoldoutEvaluationResult, error) {
	inputIDs := map[string]bool{}
	for _, id := range input.GenerationInputStableRowIDs {
		if strings.TrimSpace(id) == "" || inputIDs[id] {
			return HoldoutEvaluationResult{}, ErrHoldoutOverlap
		}
		inputIDs[id] = true
	}
	holdoutIDs := map[string]bool{}
	for _, id := range input.HoldoutStableRowIDs {
		if strings.TrimSpace(id) == "" || inputIDs[id] || holdoutIDs[id] {
			return HoldoutEvaluationResult{}, ErrHoldoutOverlap
		}
		holdoutIDs[id] = true
	}
	seen := map[string]bool{}
	result := HoldoutEvaluationResult{}
	for _, review := range input.Reviewed {
		if !holdoutIDs[review.StableRowIdentity] || seen[review.StableRowIdentity] {
			return HoldoutEvaluationResult{}, ErrHoldoutOverlap
		}
		seen[review.StableRowIdentity] = true
		result.ReviewedCount++
		switch review.Outcome {
		case HoldoutSupported:
			result.SupportedCount++
		case HoldoutUnsupported:
			result.UnsupportedCount++
		case HoldoutMissed:
			result.MissedCount++
		case HoldoutCorrected:
			result.ManagerCorrectionCount++
		default:
			return HoldoutEvaluationResult{}, ErrHoldoutOverlap
		}
	}
	return result, nil
}

type AdaptiveControl struct {
	ID                                                                                 string
	Mandatory, SafetyCritical, SourceChanged, OpenFinding, RepeatFinding, HistoryKnown bool
}
type AdaptiveScopeDecision struct {
	Include bool
	Reason  string
}

func DecideAdaptiveScope(control AdaptiveControl) AdaptiveScopeDecision {
	for _, guarded := range []struct {
		condition bool
		reason    string
	}{{control.Mandatory, "mandatory"}, {control.SafetyCritical, "safety-critical"}, {control.SourceChanged, "source changed"}, {control.OpenFinding, "open Finding"}, {control.RepeatFinding, "repeat Finding"}, {!control.HistoryKnown, "unknown history"}} {
		if guarded.condition {
			return AdaptiveScopeDecision{Include: true, Reason: guarded.reason}
		}
	}
	return AdaptiveScopeDecision{Include: true, Reason: "advisory full-scope default"}
}

type RolloutStatus string

const RolloutReviewRequired RolloutStatus = "REVIEW_REQUIRED"

type RolloutReadinessInput struct {
	ProviderTypeID                              string
	SourcePackageReady, ResponsibleManagerReady bool
}
type RolloutReadinessDecision struct {
	Status   RolloutStatus
	Official bool
	Missing  []string
}

func DecideRolloutReadiness(input RolloutReadinessInput) RolloutReadinessDecision {
	missing := []string{}
	if strings.TrimSpace(input.ProviderTypeID) == "" {
		missing = append(missing, "provider identity")
	}
	if !input.SourcePackageReady {
		missing = append(missing, "source package")
	}
	if !input.ResponsibleManagerReady {
		missing = append(missing, "responsible manager")
	}
	sort.Strings(missing)
	return RolloutReadinessDecision{Status: RolloutReviewRequired, Official: false, Missing: missing}
}
