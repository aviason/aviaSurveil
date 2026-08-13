package assignments

import "github.com/aviason/aviaSurveil/internal/identity"

func CanPrepare(actor identity.Principal) bool {
	return actor.HasRole(identity.RoleDepartmentManager)
}

func CanAssignLead(actor identity.Principal) bool {
	return actor.HasRole(identity.RoleDepartmentManager)
}

func CanConfigureTeam(actor identity.Principal, leadSubjectID string) bool {
	return actor.HasRole(identity.RoleLeadInspector) && actor.SubjectID == leadSubjectID
}

func CanViewWorkload(actor identity.Principal) bool {
	return actor.HasRole(identity.RoleDepartmentManager, identity.RoleLeadInspector)
}

func CanViewAuditeeCoordination(actor identity.Principal) bool {
	return actor.HasRole(identity.RoleAuditee) && actor.OrganizationID != ""
}

func CanViewTeamMembers(actor identity.Principal) bool {
	return actor.HasRole(identity.RoleDepartmentManager, identity.RoleAdmin)
}

func CanViewAuditTeams(actor identity.Principal) bool {
	return actor.HasRole(identity.RoleDepartmentManager)
}
