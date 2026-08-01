package checklistgovernance

import "testing"

func TestEvaluateAuditPackageEligibilityIsSeparateFromPublication(t *testing.T) {
	view, err := EvaluateAuditPackageEligibilityView(AuditPackageEligibilityRequest{PublishedVersionID: "PV-1", Published: true, Current: true, Applicable: true, TechnicalDecisionCount: 1, RequiredOwnerCount: 1})
	if err != nil || !view.Eligible || view.Digest == "" {
		t.Fatalf("eligible view = %+v err=%v", view, err)
	}
	blocked, err := EvaluateAuditPackageEligibilityView(AuditPackageEligibilityRequest{PublishedVersionID: "PV-1", Published: true, Current: true, Applicable: true, TechnicalDecisionCount: 0, RequiredOwnerCount: 1})
	if err != nil || blocked.Eligible || len(blocked.Blockers) == 0 {
		t.Fatalf("technical approval bypass: %+v err=%v", blocked, err)
	}
}
