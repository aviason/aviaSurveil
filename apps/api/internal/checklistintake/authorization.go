package checklistintake

import "github.com/aviason/aviaSurveil/internal/identity"

func CanReceiveArchive(principal identity.Principal) bool {
	return principal.OrganizationID == "CAA" && principal.HasRole(identity.RoleAdmin) && principal.SubjectID != ""
}

func CanResolveIdentity(principal identity.Principal) bool {
	return CanReceiveArchive(principal)
}
