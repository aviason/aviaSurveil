package regulatory

import (
	"testing"
	"time"

	"github.com/aviason/aviaSurveil/internal/identity"
)

func TestAppendSourceAuthorityDecisionRequiresExactSourceOwnerAssignment(t *testing.T) {
	at := time.Now().UTC()
	assignment := identity.FunctionalAssignment{RootID: "root", Version: 1, SubjectID: "owner", MembershipID: "membership", Permission: identity.FunctionalPermissionRegulatorySourceOwner, Scope: identity.FunctionalAssignmentScope{Kind: identity.ScopeSourceIdentity, SourceID: "source", SourceVersionID: "v1", ChainRole: "REGULATORY_AUTHORITY", DepartmentID: "D1", OrganizationalUnitID: "U1"}, EffectiveFrom: at.Add(-time.Hour), Status: identity.FunctionalAssignmentActive}
	decision, err := AppendSourceAuthorityDecision([]identity.FunctionalAssignment{assignment}, SourceAuthorityDecisionInput{SubjectID: "owner", MembershipID: "membership", SourceID: "source", SourceVersionID: "v1", SourceHash: "sha256:hash", ChainRole: "REGULATORY_AUTHORITY", Outcome: "ACCEPT", At: at, Reason: "verified"})
	if err != nil || decision.DecisionSubjectDigest == "" {
		t.Fatalf("decision=%+v err=%v", decision, err)
	}
	if _, err := AppendSourceAuthorityDecision([]identity.FunctionalAssignment{assignment}, SourceAuthorityDecisionInput{SubjectID: "owner", MembershipID: "membership", SourceID: "other", SourceVersionID: "v1", SourceHash: "sha256:hash", ChainRole: "REGULATORY_AUTHORITY", Outcome: "ACCEPT", At: at, Reason: "wrong source"}); err == nil {
		t.Fatal("cross-source authority was accepted")
	}
}

func TestEvaluateAuditPackageEligibilityIsComputedAndFailClosed(t *testing.T) {
	result := EvaluateAuditPackageEligibility(AuditPackageEligibilityInput{Published: true, Current: true, Applicable: true, TechnicalDecisionCount: 1, RequiredOwnerCount: 1})
	if !result.Eligible || len(result.Blockers) != 0 {
		t.Fatalf("eligible result=%+v", result)
	}
	result = EvaluateAuditPackageEligibility(AuditPackageEligibilityInput{Published: true, Current: false, Applicable: true, TechnicalDecisionCount: 1, RequiredOwnerCount: 1})
	if result.Eligible || len(result.Blockers) != 1 || result.Blockers[0] != "SOURCE_NOT_CURRENT" {
		t.Fatalf("stale result=%+v", result)
	}
}
