package datafeed

import (
	"fmt"
	"sort"
)

// RecoveryEvent is a value-free producer recovery frontier. It carries the
// immutable event identity/digest and current terminal knowledge only; source
// payload, private evidence, and diagnostics are intentionally unavailable.
type RecoveryEvent struct {
	EventID              string
	CanonicalEventSHA256 string
	DeliveryOutcome      string
	Tombstoned           bool
}

// SelectRecoveryReplayEvents returns only exact unacknowledged event IDs for
// a separately approved recovery run. A restored downstream state can never
// infer producer acknowledgement, and a tombstone can never be resurrected.
func SelectRecoveryReplayEvents(events []RecoveryEvent) ([]string, error) {
	if len(events) == 0 {
		return nil, fmt.Errorf("recovery frontier must contain events")
	}
	seen := make(map[string]struct{}, len(events))
	selected := make([]string, 0, len(events))
	for _, event := range events {
		if !validReplayUUID(event.EventID) || !validSHA256(event.CanonicalEventSHA256) {
			return nil, fmt.Errorf("recovery frontier has invalid immutable event identity or digest")
		}
		if _, exists := seen[event.EventID]; exists {
			return nil, fmt.Errorf("recovery frontier has duplicate event identity")
		}
		seen[event.EventID] = struct{}{}
		switch event.DeliveryOutcome {
		case "ACKNOWLEDGED":
			continue
		case "PENDING", "QUARANTINED":
			if !event.Tombstoned {
				selected = append(selected, event.EventID)
			}
		default:
			return nil, fmt.Errorf("recovery frontier has unsupported delivery outcome")
		}
	}
	sort.Strings(selected)
	return selected, nil
}
