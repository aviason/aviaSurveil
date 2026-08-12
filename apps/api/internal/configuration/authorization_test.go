package configuration

import (
	"testing"

	"github.com/aviason/aviaSurveil/internal/identity"
)

func TestChecklistTemplateVersionDetailReadRequiresAdmin(t *testing.T) {
	admin := identity.Principal{SubjectID: "USR-ADMIN-ADA", Roles: []identity.Role{identity.RoleAdmin}}
	if !CanReadChecklistTemplateVersionDetail(admin) {
		t.Fatalf("Admin should read checklist template detail")
	}

	for name, principal := range map[string]identity.Principal{
		"inspector":          {SubjectID: "USR-INSPECTOR-AMINA", Roles: []identity.Role{identity.RoleInspector}},
		"lead inspector":     {SubjectID: "USR-LEAD-CANER", Roles: []identity.Role{identity.RoleLeadInspector}},
		"manager":            {SubjectID: "USR-MANAGER-NORA", Roles: []identity.Role{identity.RoleDepartmentManager}},
		"finance":            {SubjectID: "USR-FINANCE-LINA", Roles: []identity.Role{identity.RoleFinance}},
		"gm":                 {SubjectID: "USR-GM-OMAR", Roles: []identity.Role{identity.RoleGeneralManager}},
		"executive director": {SubjectID: "USR-ED-ZARA", Roles: []identity.Role{identity.RoleExecutiveDirector}},
		"auditee":            {SubjectID: "USR-AUDITEE-FLY", OrganizationID: "ORG-FLY-NAMIBIA", Roles: []identity.Role{identity.RoleAuditee}},
	} {
		if CanReadChecklistTemplateVersionDetail(principal) {
			t.Fatalf("%s should not read checklist template detail", name)
		}
	}
}
