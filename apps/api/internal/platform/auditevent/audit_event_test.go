package auditevent_test

import (
	"testing"
	"time"

	"github.com/aviason/aviaSurveil/internal/platform/auditevent"
)

func TestEventRequiresStableIdentityActionAndEntity(t *testing.T) {
	t.Parallel()

	event := auditevent.Event{
		ID:         "audit-001",
		OccurredAt: time.Date(2026, time.July, 21, 12, 0, 0, 0, time.UTC),
		Action:     "inspection.created",
		EntityType: "inspection",
		EntityID:   "audit-cabin-001",
	}
	if err := event.Validate(); err != nil {
		t.Fatalf("valid event rejected: %v", err)
	}
	event.Action = ""
	if err := event.Validate(); err == nil {
		t.Fatal("event without action accepted")
	}
}
