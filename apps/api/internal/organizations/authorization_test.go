package organizations_test

import (
	"testing"

	"github.com/aviason/aviaSurveil/internal/identity"
	"github.com/aviason/aviaSurveil/internal/organizations"
)

func TestAuditeeCanViewOnlyItsOrganization(t *testing.T) {
	t.Parallel()
	principal := identity.Principal{SubjectID: "auditee-001", OrganizationID: "airline-xyz", Roles: []identity.Role{identity.RoleAuditee}}
	if !organizations.CanView(principal, "airline-xyz") {
		t.Fatal("Auditee denied its own organization")
	}
	if organizations.CanView(principal, "airline-other") {
		t.Fatal("Auditee allowed another organization")
	}
	if !organizations.CanListRegistry(principal) {
		t.Fatal("Auditee denied its organization-scoped registry projection")
	}
}

func TestCAAOperationalRolesCanViewAuthorizedOrganizationRecords(t *testing.T) {
	t.Parallel()
	principal := identity.Principal{SubjectID: "manager-001", OrganizationID: "caa", Roles: []identity.Role{identity.RoleDepartmentManager}}
	if !organizations.CanView(principal, "airline-xyz") {
		t.Fatal("Department Manager denied CAA organization record access")
	}
}
