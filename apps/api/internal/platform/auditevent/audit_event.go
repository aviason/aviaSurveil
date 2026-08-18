package auditevent

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"
)

type Event struct {
	ID                string
	OccurredAt        time.Time
	ActorSubjectID    *string
	ActorRole         *string
	OrganizationID    *string
	Action            string
	EntityType        string
	EntityID          string
	RequestID         *string
	Details           json.RawMessage
	ProfileKeyID      *string
	OperationID       *string
	PackageRevision   *int64
	RequestHash       *string
	PayloadHash       *string
	ServerRevision    *int64
	ResultCode        *string
	PreviousEventHash *string
	EventHash         *string
}

func (event Event) CanonicalHash(previousEventHash string) (string, error) {
	payload, err := json.Marshal(struct {
		SchemaVersion string `json:"schemaVersion"`
		PreviousHash  string `json:"previousEventHash"`
		Event         Event  `json:"event"`
	}{"avia-audit-event/v1", previousEventHash, event})
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(payload)
	return "sha256:" + hex.EncodeToString(digest[:]), nil
}

func (event Event) Validate() error {
	if event.ID == "" {
		return fmt.Errorf("audit event ID is required")
	}
	if event.OccurredAt.IsZero() {
		return fmt.Errorf("audit event occurrence time is required")
	}
	if event.Action == "" {
		return fmt.Errorf("audit event action is required")
	}
	if event.EntityType == "" || event.EntityID == "" {
		return fmt.Errorf("audit event entity type and ID are required")
	}
	return nil
}

type Recorder interface {
	Append(context.Context, Event) error
}
