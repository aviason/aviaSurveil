package planning

import "github.com/aviason/aviaSurveil/internal/identity"

func CanEditBudget(principal identity.Principal) bool {
	return principal.HasRole(identity.RoleFinance)
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
