package auditlog

import (
	"fmt"
	"strings"

	"github.com/aviason/aviaSurveil/internal/identity"
)

func CanReadInternal(principal identity.Principal) bool {
	return principal.HasRole(
		identity.RoleInspector,
		identity.RoleLeadInspector,
		identity.RoleDepartmentManager,
		identity.RoleGeneralManager,
		identity.RoleExecutiveDirector,
		identity.RoleAdmin,
	)
}

type Event struct {
	ActorSubjectID string
	ActorRole      identity.Role
	OrganizationID string
	Action         string
	EntityType     string
	EntityID       string
	EntityVersion  int64
	BeforeStatus   string
	AfterStatus    string
	Reason         string
	OperationID    string
	CorrelationID  string
	ClosureBasis   string
}

func (event Event) Validate() error {
	for name, value := range map[string]string{
		"actor": event.ActorSubjectID, "role": string(event.ActorRole), "organization": event.OrganizationID,
		"action": event.Action, "entity type": event.EntityType, "entity ID": event.EntityID,
		"operation ID": event.OperationID, "correlation ID": event.CorrelationID,
	} {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("audit %s is required", name)
		}
	}
	if event.EntityVersion < 1 {
		return fmt.Errorf("audit entity version must be positive")
	}
	if event.BeforeStatus == event.AfterStatus {
		return fmt.Errorf("audit transition must change status")
	}
	return nil
}
