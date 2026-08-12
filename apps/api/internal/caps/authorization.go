package caps

import (
	"fmt"

	"github.com/aviason/aviaSurveil/internal/identity"
)

type RevisionAudience string

const (
	AudienceCAA     RevisionAudience = "CAA"
	AudienceAuditee RevisionAudience = "AUDITEE"
)

type RevisionReadAuthorizationInput struct {
	Actor                 identity.Principal
	FindingOrganizationID string
	FindingAuthorized     bool
}

func AuthorizeRevisionRead(input RevisionReadAuthorizationInput) (RevisionAudience, error) {
	if input.Actor.HasRole(identity.RoleInspector, identity.RoleLeadInspector, identity.RoleDepartmentManager) {
		if !input.FindingAuthorized {
			return "", fmt.Errorf("Finding authority is required to read CAP revisions")
		}
		return AudienceCAA, nil
	}
	if input.Actor.HasRole(identity.RoleAuditee) && input.Actor.BelongsTo(input.FindingOrganizationID) {
		return AudienceAuditee, nil
	}
	return "", fmt.Errorf("CAP revision read authority is unavailable")
}
