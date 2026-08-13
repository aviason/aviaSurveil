package organizations

import (
	"errors"

	"github.com/aviason/aviaSurveil/internal/identity"
)

var (
	ErrForbidden = errors.New("organization access forbidden")
	ErrNotFound  = errors.New("organization not found")
)

func CanView(principal identity.Principal, organizationID string) bool {
	if principal.HasRole(identity.RoleAuditee) {
		return principal.BelongsTo(organizationID)
	}
	return principal.IsCAA()
}

func CanListRegistry(principal identity.Principal) bool {
	return principal.IsCAA() ||
		(principal.HasRole(identity.RoleAuditee) && principal.OrganizationID != "")
}
