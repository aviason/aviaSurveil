package planning

import (
	"errors"
	"testing"
	"time"

	"github.com/aviason/aviaSurveil/internal/application"
	"github.com/aviason/aviaSurveil/internal/identity"
)

func TestDecideTransitionKeepsPlanningAuthoritiesSeparate(t *testing.T) {
	finance := identity.Principal{Roles: []identity.Role{identity.RoleFinance}}
	gm := identity.Principal{Roles: []identity.Role{identity.RoleGeneralManager}}
	executive := identity.Principal{Roles: []identity.Role{identity.RoleExecutiveDirector}}
	manager := identity.Principal{Roles: []identity.Role{identity.RoleDepartmentManager}}

	status, owner, _, action, err := decideTransition(finance, StatusFinanceReview, DecisionApproveBudget)
	if err != nil || status != StatusGeneralManagerReview || owner != identity.RoleGeneralManager || action != "PLANNING_BUDGET_APPROVED" {
		t.Fatalf("Finance transition = %s %s %s, err=%v", status, owner, action, err)
	}
	status, owner, _, action, err = decideTransition(gm, status, DecisionForwardForFinalApproval)
	if err != nil || status != StatusExecutiveDirectorReview || owner != identity.RoleExecutiveDirector || action != "PLANNING_FORWARDED_FOR_FINAL_APPROVAL" {
		t.Fatalf("GM transition = %s %s %s, err=%v", status, owner, action, err)
	}
	status, owner, _, action, err = decideTransition(executive, status, DecisionApprovePlan)
	if err != nil || status != StatusGeneralManagerRelease || owner != identity.RoleGeneralManager || action != "PLANNING_APPROVED" {
		t.Fatalf("Executive transition = %s %s %s, err=%v", status, owner, action, err)
	}
	status, owner, _, action, err = decideTransition(gm, status, DecisionReleasePlan)
	if err != nil || status != StatusReleased || owner != identity.RoleDepartmentManager || action != "PLANNING_RELEASED" {
		t.Fatalf("release transition = %s %s %s, err=%v", status, owner, action, err)
	}
	if _, _, _, _, err := decideTransition(manager, StatusFinanceReview, DecisionApproveBudget); !errors.Is(err, application.ErrForbidden) {
		t.Fatalf("manager budget decision error = %v, want forbidden", err)
	}
}

func TestProposalWorkloadEstimateUsesCurrentGovernedCatalog(t *testing.T) {
	values := ProposalDraftValues{
		OrganizationID:    "ORG-NAMIBIA-DEMO-AGA-QUALIFICATION",
		ProviderScopeID:   "SCOPE-NAMIBIA-DEMO-AGA-QUALIFICATION",
		RegulatedTargetID: "TARGET-NAMIBIA-DEMO-AGA-QUALIFICATION-AERODROME",
		InspectionType:    "RAMP_INSPECTION",
	}

	estimate := proposalWorkloadEstimate(values, time.Date(2026, 8, 22, 0, 0, 0, 0, time.UTC))
	if estimate.CatalogVersion != "aga-approved-source@2.0.0" {
		t.Fatalf("catalog version = %q, want aga-approved-source@2.0.0", estimate.CatalogVersion)
	}
}
