package planning

import "github.com/aviason/aviaSurveil/internal/identity"

func CanEditBudget(principal identity.Principal) bool {
	return principal.HasRole(identity.RoleFinance)
}

// CanListQueue is deliberately narrower than assignment-scoped inspection
// access. Inspectors receive only the planning records attached to their
// canonical audit assignment; the global CAA queue is reserved for planning
// authorities and administrators.
func CanListQueue(principal identity.Principal) bool {
	return principal.HasRole(
		identity.RoleDepartmentManager,
		identity.RoleFinance,
		identity.RoleGeneralManager,
		identity.RoleExecutiveDirector,
		identity.RoleAdmin,
	)
}

func CanApproveOperationalScope(principal identity.Principal) bool {
	return principal.HasRole(identity.RoleDepartmentManager, identity.RoleGeneralManager, identity.RoleExecutiveDirector)
}

func CanIntermediateApprove(principal identity.Principal) bool {
	return principal.HasRole(identity.RoleGeneralManager)
}

func CanIssueReport(principal identity.Principal) bool {
	return principal.HasRole(identity.RoleExecutiveDirector)
}
