package administration

import (
	"errors"

	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/identity"
)

var (
	ErrForbidden                  = errors.New("administration authority required")
	ErrInvalid                    = errors.New("invalid user lifecycle request")
	ErrConflict                   = errors.New("user lifecycle conflict")
	ErrMembershipRevisionConflict = errors.New(
		"desired membership revision conflict",
	)
	ErrInvitationResendLimit = errors.New(
		"invitation resend limit exceeded",
	)
	ErrInvitationExpired = errors.New("identity invitation expired")
)

func CanManageUsers(actor identity.Principal) bool {
	return actor.OrganizationID == "CAA" && actor.HasRole(identity.RoleAdmin)
}
