package notifications

import (
	"time"

	"github.com/aviason/aviaSurveil/internal/identity"
)

type DueState string

type EmailDeliveryStatus string

const (
	DueStateNone     DueState = "NONE"
	DueStateNotDue   DueState = "NOT_DUE"
	DueStateDueSoon  DueState = "DUE_SOON"
	DueStateDueToday DueState = "DUE_TODAY"
	DueStateOverdue  DueState = "OVERDUE"
)

const (
	EmailDeliveryNotConfigured EmailDeliveryStatus = "NOT_CONFIGURED"
	EmailDeliveryPending       EmailDeliveryStatus = "PENDING"
	EmailDeliveryRetrying      EmailDeliveryStatus = "RETRYING"
	EmailDeliveryDelivered     EmailDeliveryStatus = "DELIVERED"
	EmailDeliveryFailed        EmailDeliveryStatus = "FAILED"
)

type Notification struct {
	ID                    string              `json:"id"`
	RecipientSubjectID    string              `json:"recipientSubjectId"`
	OrganizationID        string              `json:"organizationId"`
	Title                 string              `json:"title"`
	Body                  string              `json:"body"`
	RelatedEntityType     string              `json:"relatedEntityType"`
	RelatedEntityID       string              `json:"relatedEntityId"`
	DeduplicationKey      string              `json:"deduplicationKey"`
	ReadAt                *time.Time          `json:"readAt"`
	EmailDeliveryStatus   EmailDeliveryStatus `json:"emailDeliveryStatus"`
	EmailDeliveryAttempts int                 `json:"emailDeliveryAttempts"`
	EmailAcceptedAt       *time.Time          `json:"emailAcceptedAt"`
	EmailNextAttemptAt    *time.Time          `json:"emailNextAttemptAt"`
	Revision              int64               `json:"revision"`
	CreatedAt             time.Time           `json:"createdAt"`
}

type Page struct {
	Items       []Notification `json:"items"`
	UnreadCount int            `json:"unreadCount"`
}

func CanUse(actor identity.Principal) bool {
	return actor.HasRole(
		identity.RoleInspector,
		identity.RoleLeadInspector,
		identity.RoleDepartmentManager,
		identity.RoleFinance,
		identity.RoleGeneralManager,
		identity.RoleExecutiveDirector,
		identity.RoleAuditee,
		identity.RoleAdmin,
	)
}

func DueStateFor(dueDate, now time.Time) DueState {
	due := dateOnly(dueDate)
	today := dateOnly(now)
	days := int(due.Sub(today) / (24 * time.Hour))
	switch {
	case days < 0:
		return DueStateOverdue
	case days == 0:
		return DueStateDueToday
	case days <= 7:
		return DueStateDueSoon
	default:
		return DueStateNotDue
	}
}

func RuleMatches(offsetDays int, dueDate, now time.Time) bool {
	days := int(dateOnly(dueDate).Sub(dateOnly(now)) / (24 * time.Hour))
	if offsetDays == -1 {
		return days < 0
	}
	return days == offsetDays
}

func dateOnly(value time.Time) time.Time {
	return time.Date(
		value.UTC().Year(),
		value.UTC().Month(),
		value.UTC().Day(),
		0,
		0,
		0,
		0,
		time.UTC,
	)
}
