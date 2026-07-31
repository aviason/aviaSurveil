package identity

import "time"

type Role string

const (
	RoleInspector         Role = "inspector"
	RoleLeadInspector     Role = "leadInspector"
	RoleDepartmentManager Role = "manager"
	RoleGeneralManager    Role = "gm"
	RoleFinance           Role = "finance"
	RoleExecutiveDirector Role = "executiveDirector"
	RoleAuditee           Role = "auditee"
	RoleAdmin             Role = "admin"
)

type Principal struct {
	SubjectID             string
	DisplayName           string
	OrganizationID        string
	Roles                 []Role
	SessionID             string
	DepartmentAssignments []DepartmentAssignment
}

// DepartmentAssignment is a resolved, effective-dated CAA authority fact.
// It is intentionally separate from organization membership and role claims.
type DepartmentAssignment struct {
	DepartmentID         string
	OrganizationalUnitID string
	EffectiveFrom        time.Time
	EffectiveTo          *time.Time
}

func (principal Principal) HasRole(expected ...Role) bool {
	for _, actual := range principal.Roles {
		for _, candidate := range expected {
			if actual == candidate {
				return true
			}
		}
	}
	return false
}

func (principal Principal) BelongsTo(organizationID string) bool {
	return principal.OrganizationID != "" && principal.OrganizationID == organizationID
}

func (principal Principal) IsCAA() bool {
	return principal.HasRole(
		RoleInspector,
		RoleLeadInspector,
		RoleDepartmentManager,
		RoleGeneralManager,
		RoleFinance,
		RoleExecutiveDirector,
		RoleAdmin,
	)
}
