package identity

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

var ErrInvalidApplicationAuthority = errors.New(
	"invalid application authority",
)

type AuthorityObservation struct {
	SubjectID       string
	Enabled         bool
	Locked          bool
	OrganizationID  string
	Roles           []Role
	RequiredActions []string
	MFAEnrolled     bool
	ObservedAt      time.Time
}

type AuthorityObserver interface {
	ObserveUserAuthority(
		context.Context,
		string,
	) (AuthorityObservation, error)
}

func ValidateApplicationAuthority(
	organizationID string,
	roles []Role,
) error {
	trimmedOrganizationID := strings.TrimSpace(organizationID)
	if organizationID != trimmedOrganizationID ||
		trimmedOrganizationID == "" ||
		len(roles) != 1 {
		return ErrInvalidApplicationAuthority
	}
	switch roles[0] {
	case RoleAuditee:
		if organizationID == "CAA" {
			return ErrInvalidApplicationAuthority
		}
	case RoleInspector, RoleLeadInspector, RoleDepartmentManager,
		RoleGeneralManager, RoleFinance, RoleExecutiveDirector, RoleAdmin:
		if organizationID != "CAA" {
			return ErrInvalidApplicationAuthority
		}
	default:
		return fmt.Errorf(
			"%w: unsupported role %q",
			ErrInvalidApplicationAuthority,
			roles[0],
		)
	}
	return nil
}

func EqualApplicationAuthority(
	leftOrganizationID string,
	leftRoles []Role,
	rightOrganizationID string,
	rightRoles []Role,
) bool {
	if ValidateApplicationAuthority(leftOrganizationID, leftRoles) != nil ||
		ValidateApplicationAuthority(rightOrganizationID, rightRoles) != nil {
		return false
	}
	return leftOrganizationID == rightOrganizationID &&
		leftRoles[0] == rightRoles[0]
}
