package caps_test

import (
	"testing"

	"github.com/aviason/aviaSurveil/internal/caps"
	"github.com/aviason/aviaSurveil/internal/findings"
	"github.com/aviason/aviaSurveil/internal/identity"
)

func TestAuditeeSubmissionNeverAcceptsCAP(t *testing.T) {
	t.Parallel()
	auditee := identity.Principal{OrganizationID: "airline-xyz", Roles: []identity.Role{identity.RoleAuditee}}
	result, err := caps.Submit(caps.SubmitInput{
		Actor: auditee, FindingOrganizationID: "airline-xyz", FindingStatus: findings.StatusWaitingForCAP,
		FindingRevision: 2, ExpectedFindingRevision: 2,
	})
	if err != nil {
		t.Fatalf("submit CAP: %v", err)
	}
	if result.CAPStatus != caps.StatusSubmitted || result.FindingStatus != findings.StatusCAPSubmitted {
		t.Fatalf("submission result = %+v", result)
	}
}

func TestCAAReviewBindsExactRevisionAndAcceptanceKeepsFindingOpen(t *testing.T) {
	t.Parallel()
	inspector := identity.Principal{Roles: []identity.Role{identity.RoleInspector}}
	accepted, err := caps.Review(caps.ReviewInput{
		Actor: inspector, CAPStatus: caps.StatusSubmitted, CAPRevision: 1, ExpectedCAPRevision: 1,
		FindingStatus: findings.StatusCAPSubmitted, EvidenceRequired: true, Decision: caps.DecisionAccept,
	})
	if err != nil {
		t.Fatalf("accept CAP: %v", err)
	}
	if accepted.CAPStatus != caps.StatusAccepted || accepted.FindingStatus != findings.StatusEvidenceRequired {
		t.Fatalf("acceptance result = %+v", accepted)
	}
	if accepted.FindingStatus == findings.StatusClosed {
		t.Fatal("CAP acceptance closed Finding")
	}
	if _, err := caps.Review(caps.ReviewInput{
		Actor: inspector, CAPStatus: caps.StatusSubmitted, CAPRevision: 2, ExpectedCAPRevision: 1,
		FindingStatus: findings.StatusCAPSubmitted, EvidenceRequired: true, Decision: caps.DecisionReject,
	}); err == nil {
		t.Fatal("stale CAP review accepted")
	}
}

func TestCAPReviewSupportsRejectAndMoreInformation(t *testing.T) {
	t.Parallel()
	inspector := identity.Principal{Roles: []identity.Role{identity.RoleInspector}}
	for decision, expected := range map[caps.ReviewDecision]findings.Status{
		caps.DecisionReject:             findings.StatusCAPRejected,
		caps.DecisionRequestInformation: findings.StatusCAPMoreInformationRequested,
	} {
		result, err := caps.Review(caps.ReviewInput{
			Actor: inspector, CAPStatus: caps.StatusSubmitted, CAPRevision: 1, ExpectedCAPRevision: 1,
			FindingStatus: findings.StatusCAPSubmitted, EvidenceRequired: true,
			Decision: decision, Reason: "CAA review reason.",
		})
		if err != nil || result.FindingStatus != expected {
			t.Errorf("decision %s = %+v, err = %v", decision, result, err)
		}
	}
}
