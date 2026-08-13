package potentialfindings

import (
	"fmt"

	"github.com/aviason/aviaSurveil/internal/identity"
)

type ReadAuthorizationInput struct {
	Actor                       identity.Principal
	AssignedInspectorSubjectIDs []string
}

func AuthorizeList(actor identity.Principal) error {
	if actor.HasRole(identity.RoleLeadInspector) {
		return nil
	}
	return fmt.Errorf("Lead Inspector authority is required to list Potential Findings")
}

func AuthorizeRead(input ReadAuthorizationInput) error {
	if input.Actor.HasRole(identity.RoleLeadInspector) {
		return nil
	}
	if input.Actor.HasRole(identity.RoleInspector) {
		for _, subjectID := range input.AssignedInspectorSubjectIDs {
			if subjectID == input.Actor.SubjectID {
				return nil
			}
		}
	}
	return fmt.Errorf("Potential Finding read authority is unavailable")
}
