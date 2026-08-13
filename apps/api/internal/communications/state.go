package communications

import (
	"errors"
	"strings"
	"time"

	"github.com/aviason/aviaSurveil/internal/identity"
	"github.com/aviason/aviaSurveil/internal/notifications"
)

var (
	ErrForbidden = errors.New("communication forbidden")
	ErrInvalid   = errors.New("invalid communication")
)

type Visibility string

const (
	VisibilityAuditeeVisible Visibility = "AUDITEE_VISIBLE"
	VisibilityInternalCAA    Visibility = "INTERNAL_CAA"
)

type Audience string

const (
	AudienceCAA     Audience = "CAA"
	AudienceAuditee Audience = "AUDITEE"
)

type Direction string

const (
	DirectionCAAToAuditee Direction = "CAA_TO_AUDITEE"
	DirectionAuditeeToCAA Direction = "AUDITEE_TO_CAA"
	DirectionCAAInternal  Direction = "CAA_INTERNAL"
)

type Message struct {
	ID              string     `json:"id"`
	ThreadID        string     `json:"threadId"`
	OrganizationID  string     `json:"organizationId"`
	Visibility      Visibility `json:"visibility"`
	SenderSubjectID string     `json:"senderSubjectId"`
	Audience        Audience   `json:"audience"`
	Direction       Direction  `json:"direction"`
	Subject         string     `json:"subject"`
	Body            string     `json:"body"`
	Revision        int64      `json:"revision"`
	CreatedAt       time.Time  `json:"createdAt"`
}

type Attachment struct {
	ID               string    `json:"id"`
	MessageID        string    `json:"messageId"`
	OrganizationID   string    `json:"organizationId"`
	ObjectMetadataID string    `json:"objectMetadataId"`
	FileName         string    `json:"fileName"`
	MediaType        string    `json:"mediaType"`
	SizeBytes        int64     `json:"sizeBytes"`
	SHA256           string    `json:"sha256"`
	CreatedAt        time.Time `json:"createdAt"`
}

type CalendarItem struct {
	ID               string                 `json:"id"`
	AuditID          string                 `json:"auditId"`
	OrganizationID   string                 `json:"organizationId"`
	OrganizationName string                 `json:"organizationName"`
	Title            string                 `json:"title"`
	NextAction       string                 `json:"nextAction"`
	ScheduledDate    string                 `json:"scheduledDate"`
	DueState         notifications.DueState `json:"dueState"`
}

type Policy struct {
	Visibility Visibility
	Direction  Direction
}

func CanUseCommunications(actor identity.Principal) bool {
	return actor.HasRole(
		identity.RoleInspector,
		identity.RoleLeadInspector,
		identity.RoleDepartmentManager,
		identity.RoleAuditee,
	)
}

func CanUseCalendar(actor identity.Principal) bool {
	return actor.HasRole(
		identity.RoleInspector,
		identity.RoleLeadInspector,
		identity.RoleDepartmentManager,
		identity.RoleAuditee,
	)
}

func ResolvePolicy(
	actor identity.Principal,
	organizationID string,
	audience Audience,
) (Policy, error) {
	organizationID = strings.TrimSpace(organizationID)
	if !CanUseCommunications(actor) {
		return Policy{}, ErrForbidden
	}
	switch {
	case actor.HasRole(identity.RoleAuditee):
		if audience != AudienceCAA || organizationID == "" ||
			organizationID != actor.OrganizationID {
			return Policy{}, ErrForbidden
		}
		return Policy{
			Visibility: VisibilityAuditeeVisible,
			Direction:  DirectionAuditeeToCAA,
		}, nil
	case actor.HasRole(
		identity.RoleInspector,
		identity.RoleLeadInspector,
		identity.RoleDepartmentManager,
	):
		switch audience {
		case AudienceAuditee:
			if organizationID == "" || organizationID == actor.OrganizationID {
				return Policy{}, ErrInvalid
			}
			return Policy{
				Visibility: VisibilityAuditeeVisible,
				Direction:  DirectionCAAToAuditee,
			}, nil
		case AudienceCAA:
			return Policy{
				Visibility: VisibilityInternalCAA,
				Direction:  DirectionCAAInternal,
			}, nil
		default:
			return Policy{}, ErrInvalid
		}
	default:
		return Policy{}, ErrForbidden
	}
}

func CanRead(
	actor identity.Principal,
	message Message,
) bool {
	if CanUseCommunications(actor) && actor.IsCAA() {
		return true
	}
	if !actor.HasRole(identity.RoleAuditee) ||
		message.Visibility != VisibilityAuditeeVisible ||
		message.OrganizationID == "" ||
		message.OrganizationID != actor.OrganizationID {
		return false
	}
	switch message.Direction {
	case DirectionCAAToAuditee:
		return message.Audience == AudienceAuditee
	case DirectionAuditeeToCAA:
		return message.Audience == AudienceCAA &&
			message.SenderSubjectID == actor.SubjectID
	default:
		return false
	}
}
