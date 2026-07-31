package regulatory

import (
	"errors"
	"testing"
)

func TestEvaluateBlindHoldoutRejectsInputOverlapAndCountsReviewOutcomes(t *testing.T) {
	_, err := EvaluateBlindHoldout(HoldoutEvaluationInput{
		GenerationInputStableRowIDs: []string{"CC:INPUT:1"},
		HoldoutStableRowIDs:         []string{"CC:INPUT:1"},
	})
	if !errors.Is(err, ErrHoldoutOverlap) {
		t.Fatalf("overlapping stable identity error=%v, want ErrHoldoutOverlap", err)
	}
	result, err := EvaluateBlindHoldout(HoldoutEvaluationInput{
		GenerationInputStableRowIDs: []string{"CC:INPUT:1"},
		HoldoutStableRowIDs:         []string{"CC:HOLDOUT:1", "CC:HOLDOUT:2"},
		Reviewed:                    []HoldoutReview{{StableRowIdentity: "CC:HOLDOUT:1", Outcome: HoldoutSupported}, {StableRowIdentity: "CC:HOLDOUT:2", Outcome: HoldoutMissed}},
	})
	if err != nil || result.ReviewedCount != 2 || result.SupportedCount != 1 || result.MissedCount != 1 || result.UnsupportedCount != 0 {
		t.Fatalf("holdout result=%+v err=%v", result, err)
	}
}

func TestAdaptiveScopeNeverDefersGuardedControlsOrClaimsOfficialRollout(t *testing.T) {
	for _, control := range []AdaptiveControl{{ID: "MANDATORY", Mandatory: true}, {ID: "SAFETY", SafetyCritical: true}, {ID: "CHANGED", SourceChanged: true}, {ID: "OPEN", OpenFinding: true}, {ID: "REPEAT", RepeatFinding: true}, {ID: "UNKNOWN", HistoryKnown: false}} {
		if decision := DecideAdaptiveScope(control); decision.Include != true || decision.Reason == "" {
			t.Fatalf("guarded control decision=%+v for %+v", decision, control)
		}
	}
	rollout := DecideRolloutReadiness(RolloutReadinessInput{ProviderTypeID: "AIR_OPERATOR", SourcePackageReady: true, ResponsibleManagerReady: false})
	if rollout.Status != RolloutReviewRequired || rollout.Official {
		t.Fatalf("rollout=%+v must remain non-official without manager", rollout)
	}
}
