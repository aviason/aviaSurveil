package datafeed

import "testing"

func TestSelectRecoveryReplayEventsNeverReplaysAcknowledgedOrTombstonedHistory(t *testing.T) {
	events := []RecoveryEvent{
		{EventID: "10000000-0000-4000-8000-000000000121", CanonicalEventSHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", DeliveryOutcome: "PENDING"},
		{EventID: "10000000-0000-4000-8000-000000000122", CanonicalEventSHA256: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", DeliveryOutcome: "ACKNOWLEDGED"},
		{EventID: "10000000-0000-4000-8000-000000000123", CanonicalEventSHA256: "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc", DeliveryOutcome: "QUARANTINED", Tombstoned: true},
	}
	selected, err := SelectRecoveryReplayEvents(events)
	if err != nil || len(selected) != 1 || selected[0] != "10000000-0000-4000-8000-000000000121" {
		t.Fatalf("recovery replay selection=%v err=%v", selected, err)
	}
	mutated := append([]RecoveryEvent(nil), events...)
	mutated[0].CanonicalEventSHA256 = "not-a-digest"
	if _, err := SelectRecoveryReplayEvents(mutated); err == nil {
		t.Fatal("recovery accepted an invalid immutable event digest")
	}
}
