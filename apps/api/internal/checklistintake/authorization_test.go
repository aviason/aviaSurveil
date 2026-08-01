package checklistintake

import (
	"testing"

	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/identity"
)

func TestAdminOnlyIntakeAuthorizationDoesNotElevateFunctionalAssignments(t *testing.T) {
	admin := identity.Principal{SubjectID: "admin", OrganizationID: "CAA", Roles: []identity.Role{identity.RoleAdmin}}
	manager := identity.Principal{SubjectID: "manager", OrganizationID: "CAA", Roles: []identity.Role{identity.RoleDepartmentManager}}
	if !CanReceiveArchive(admin) || !CanResolveIdentity(admin) {
		t.Fatal("Admin must be able to perform candidate-only intake commands")
	}
	if CanReceiveArchive(manager) || CanResolveIdentity(manager) {
		t.Fatal("Department Manager must not receive or resolve archive identity")
	}
}
