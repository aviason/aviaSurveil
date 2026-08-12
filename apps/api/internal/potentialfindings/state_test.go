package potentialfindings_test

import (
	"testing"

	"github.com/aviason/aviaSurveil/internal/identity"
	"github.com/aviason/aviaSurveil/internal/potentialfindings"
)

func TestOnlyLeadCanReturnDismissOrConvertPotentialFinding(t *testing.T) {
	t.Parallel()
	lead := identity.Principal{Roles: []identity.Role{identity.RoleLeadInspector}}
	inspector := identity.Principal{Roles: []identity.Role{identity.RoleInspector}}

	for _, decision := range []potentialfindings.Decision{potentialfindings.DecisionReturn, potentialfindings.DecisionDismiss} {
		result, err := potentialfindings.Decide(potentialfindings.DecideInput{
			Actor: lead, Status: potentialfindings.StatusPendingLeadReview, Revision: 1, ExpectedRevision: 1,
			Decision: decision, Reason: "Recorded Lead decision.",
		})
		if err != nil || result.Status == potentialfindings.StatusPendingLeadReview {
			t.Fatalf("Lead decision %s failed: %+v, %v", decision, result, err)
		}
	}
	if _, err := potentialfindings.Decide(potentialfindings.DecideInput{
		Actor: inspector, Status: potentialfindings.StatusPendingLeadReview, Revision: 1, ExpectedRevision: 1,
		Decision: potentialfindings.DecisionDismiss, Reason: "not authorized",
	}); err == nil {
		t.Fatal("Inspector made Lead decision")
	}
}

func TestReturnAndDismissRequireReasonAndConversionRequiresSeverity(t *testing.T) {
	t.Parallel()
	lead := identity.Principal{Roles: []identity.Role{identity.RoleLeadInspector}}
	for _, decision := range []potentialfindings.Decision{potentialfindings.DecisionReturn, potentialfindings.DecisionDismiss} {
		if _, err := potentialfindings.Decide(potentialfindings.DecideInput{
			Actor: lead, Status: potentialfindings.StatusPendingLeadReview, Revision: 1, ExpectedRevision: 1, Decision: decision,
		}); err == nil {
			t.Errorf("%s without reason accepted", decision)
		}
	}
	if _, err := potentialfindings.Decide(potentialfindings.DecideInput{
		Actor: lead, Status: potentialfindings.StatusPendingLeadReview, Revision: 1, ExpectedRevision: 1,
		Decision: potentialfindings.DecisionConvert,
	}); err == nil {
		t.Fatal("conversion without severity accepted")
	}

	converted, err := potentialfindings.Decide(potentialfindings.DecideInput{
		Actor: lead, Status: potentialfindings.StatusPendingLeadReview, Revision: 1, ExpectedRevision: 1,
		Decision: potentialfindings.DecisionConvert, Severity: potentialfindings.SeverityLevel2Major,
	})
	if err != nil || !converted.CreateFinding || converted.Status != potentialfindings.StatusConverted {
		t.Fatalf("conversion = %+v, err = %v", converted, err)
	}
}
