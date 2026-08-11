package analytics

import "time"

type Purpose string

const (
	PurposeAnalytics         Purpose = "analytics"
	PurposePersonalization   Purpose = "personalization"
	PurposeMarketing         Purpose = "marketing"
	PurposePreciseLocation   Purpose = "precise_location"
	PurposeHeatmap           Purpose = "heatmap"
	PurposeMessagingMetadata Purpose = "messaging_metadata"
)

func AllPurposes() []Purpose {
	return []Purpose{
		PurposeAnalytics,
		PurposePersonalization,
		PurposeMarketing,
		PurposePreciseLocation,
		PurposeHeatmap,
		PurposeMessagingMetadata,
	}
}

func (p Purpose) Valid() bool {
	switch p {
	case PurposeAnalytics,
		PurposePersonalization,
		PurposeMarketing,
		PurposePreciseLocation,
		PurposeHeatmap,
		PurposeMessagingMetadata:
		return true
	default:
		return false
	}
}

type EventName string

const (
	EventSessionStarted       EventName = "session_started"
	EventScreenViewed         EventName = "screen_viewed"
	EventTabSelected          EventName = "tab_selected"
	EventHomeFeedLoaded       EventName = "home_feed_loaded"
	EventItemImpression       EventName = "item_impression"
	EventItemViewed           EventName = "item_viewed"
	EventItemViewDuration     EventName = "item_view_duration"
	EventItemImageViewed      EventName = "item_image_viewed"
	EventUIInteraction        EventName = "ui_interaction"
	EventLocationObserved     EventName = "location_observed"
	EventItemCreated          EventName = "item_created"
	EventRequestCreated       EventName = "request_created"
	EventRequestStatusChanged EventName = "request_status_changed"
	EventPointsChanged        EventName = "points_changed"
	EventMessageThreadOpened  EventName = "message_thread_opened"
	EventMessageSentMetadata  EventName = "message_sent_metadata"
	EventNotificationOpened   EventName = "notification_opened"
	EventProfileUpdated       EventName = "profile_updated"
)

func (n EventName) Valid() bool {
	switch n {
	case EventSessionStarted,
		EventScreenViewed,
		EventTabSelected,
		EventHomeFeedLoaded,
		EventItemImpression,
		EventItemViewed,
		EventItemViewDuration,
		EventItemImageViewed,
		EventUIInteraction,
		EventLocationObserved,
		EventItemCreated,
		EventRequestCreated,
		EventRequestStatusChanged,
		EventPointsChanged,
		EventMessageThreadOpened,
		EventMessageSentMetadata,
		EventNotificationOpened,
		EventProfileUpdated:
		return true
	default:
		return false
	}
}

type EventEnvelope struct {
	EventID       string         `json:"eventId"`
	EventName     EventName      `json:"eventName"`
	SchemaVersion int            `json:"schemaVersion"`
	EventTime     time.Time      `json:"eventTime"`
	SessionID     string         `json:"sessionId"`
	AnonymousID   string         `json:"anonymousId,omitempty"`
	Source        string         `json:"source"`
	AppVersion    string         `json:"appVersion,omitempty"`
	Screen        string         `json:"screen,omitempty"`
	Purposes      []Purpose      `json:"purposes"`
	Properties    map[string]any `json:"properties,omitempty"`
}

type BatchRequest struct {
	Events []EventEnvelope `json:"events"`
}

type BatchResponse struct {
	Accepted int `json:"accepted"`
	Rejected int `json:"rejected"`
}

type EnrichedEvent struct {
	EventEnvelope
	UserID         string    `json:"userId"`
	DeviceIDHash   string    `json:"deviceIdHash,omitempty"`
	ReceivedAt     time.Time `json:"receivedAt"`
	RecordSource   string    `json:"recordSource"`
	Environment    string    `json:"environment"`
	ConsentVersion int       `json:"consentVersion"`
}

type DataDeletionRecord struct {
	RecordType     string    `json:"recordType"`
	RecordID       string    `json:"recordId"`
	UserID         string    `json:"userId"`
	RequestedAt    time.Time `json:"requestedAt"`
	DeletedAt      time.Time `json:"deletedAt"`
	RecordSource   string    `json:"recordSource"`
	Environment    string    `json:"environment"`
	DeleteCategory string    `json:"deleteCategory"`
}

type ConsentRecord struct {
	UserID    string    `json:"userId" dynamodbav:"userId"`
	Purpose   Purpose   `json:"purpose" dynamodbav:"purpose"`
	Granted   bool      `json:"granted" dynamodbav:"granted"`
	Version   int       `json:"version" dynamodbav:"version"`
	UpdatedAt time.Time `json:"updatedAt" dynamodbav:"updatedAt"`
	Source    string    `json:"source,omitempty" dynamodbav:"source,omitempty"`
}

type ConsentUpdateRequest struct {
	Consents map[string]bool `json:"consents"`
}

type ConsentStatus struct {
	Granted   bool       `json:"granted"`
	Version   int        `json:"version"`
	UpdatedAt *time.Time `json:"updatedAt,omitempty"`
}

type ConsentStateResponse struct {
	Consents map[Purpose]ConsentStatus `json:"consents"`
}
