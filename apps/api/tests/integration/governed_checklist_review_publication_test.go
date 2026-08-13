//go:build canonicaltest

package integration

import (
	"testing"

	"github.com/aviason/aviaSurveil/internal/regulatory"
)

func TestGovernedChecklistReviewPublication(t *testing.T) {
	result := regulatory.EvaluateAuditPackageEligibility(regulatory.AuditPackageEligibilityInput{Published: true, Current: true, Applicable: true, TechnicalDecisionCount: 1, RequiredOwnerCount: 1})
	if !result.Eligible {
		t.Fatalf("complete synthetic approval/currentness facts must be eligible: %+v", result)
	}
	blocked := regulatory.EvaluateAuditPackageEligibility(regulatory.AuditPackageEligibilityInput{Published: true, Current: false, Applicable: true, TechnicalDecisionCount: 1, RequiredOwnerCount: 1})
	if blocked.Eligible || len(blocked.Blockers) == 0 {
		t.Fatalf("source change must block eligibility: %+v", blocked)
	}
}
