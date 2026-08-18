package auditevent_test

import (
	"strings"
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

func TestEventCanonicalHashBindsThePreviousChainHash(t *testing.T) {
	event := auditevent.Event{ID: "audit-001", OccurredAt: time.Date(2026, time.July, 21, 12, 0, 0, 0, time.UTC), Action: "inspection.finalized", EntityType: "inspection", EntityID: "INS-001"}
	first, err := event.CanonicalHash("sha256:" + strings.Repeat("a", 64))
	if err != nil {
		t.Fatalf("hash event: %v", err)
	}
	second, err := event.CanonicalHash("sha256:" + strings.Repeat("b", 64))
	if err != nil || first == second {
		t.Fatalf("previous event hash must bind the next event hash: %s == %s (err=%v)", first, second, err)
	}
}
