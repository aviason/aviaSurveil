package configuration

import "github.com/aviason/aviaSurveil/internal/identity"

func CanPreview(principal identity.Principal) bool {
	return principal.HasRole(identity.RoleAdmin)
}

func CanReadChecklistTemplateVersionDetail(principal identity.Principal) bool {
	return principal.HasRole(identity.RoleAdmin)
}
