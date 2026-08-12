package potentialfindings

import (
	"testing"

	"github.com/aviason/aviaSurveil/internal/identity"
)

func TestAuthorizePotentialFindingListAndRead(t *testing.T) {
	lead := identity.Principal{SubjectID: "USR-LEAD-CANER", Roles: []identity.Role{identity.RoleLeadInspector}}
	inspector := identity.Principal{SubjectID: "USR-INSPECTOR-AMINA", Roles: []identity.Role{identity.RoleInspector}}
	finance := identity.Principal{SubjectID: "USR-FINANCE-LINA", Roles: []identity.Role{identity.RoleFinance}}

	if err := AuthorizeList(lead); err != nil {
		t.Fatalf("lead list authorization failed: %v", err)
	}
	if err := AuthorizeList(inspector); err == nil {
		t.Fatal("inspector must not list the Lead Potential Finding queue")
	}
	if err := AuthorizeRead(ReadAuthorizationInput{
		Actor: inspector,
		AssignedInspectorSubjectIDs: []string{"USR-INSPECTOR-AMINA"},
	}); err != nil {
		t.Fatalf("assigned inspector read authorization failed: %v", err)
	}
	if err := AuthorizeRead(ReadAuthorizationInput{
		Actor: inspector,
		AssignedInspectorSubjectIDs: []string{"USR-INSPECTOR-DAVID"},
	}); err == nil {
		t.Fatal("unassigned inspector must not read another inspector's Potential Finding")
	}
	if err := AuthorizeRead(ReadAuthorizationInput{Actor: finance}); err == nil {
		t.Fatal("finance must not read lifecycle Potential Findings")
	}
}
