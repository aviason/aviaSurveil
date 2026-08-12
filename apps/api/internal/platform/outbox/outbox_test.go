package outbox_test

import (
	"testing"
	"time"

	"github.com/aviason/aviaSurveil/internal/platform/outbox"
)

func TestMessageRequiresTopicAggregateAndPayload(t *testing.T) {
	t.Parallel()

	message := outbox.Message{
		ID:            "outbox-001",
		Topic:         "finding.changed",
		AggregateType: "finding",
		AggregateID:   "finding-001",
		Payload:       []byte(`{"status":"OPEN"}`),
		AvailableAt:   time.Date(2026, time.July, 21, 12, 0, 0, 0, time.UTC),
	}
	if err := message.Validate(); err != nil {
		t.Fatalf("valid outbox message rejected: %v", err)
	}
	message.Payload = nil
	if err := message.Validate(); err == nil {
		t.Fatal("message without payload accepted")
	}
}
