package datafeed

import (
	"testing"
	"time"
)

func TestDecideReceiptOutcomesPreservesFencedTerminalSemantics(t *testing.T) {
	now := time.Date(2026, 7, 30, 9, 0, 0, 0, time.UTC)
	decisions, err := DecideReceiptOutcomes([]BatchItem{
		{EventID: "10000000-0000-4000-8000-000000000001", LeaseGeneration: 4},
		{EventID: "10000000-0000-4000-8000-000000000002", LeaseGeneration: 7},
		{EventID: "10000000-0000-4000-8000-000000000003", LeaseGeneration: 8},
	}, []ReceiptItem{
		{EventID: "10000000-0000-4000-8000-000000000001", Outcome: "accepted", ManifestReceiptDigest: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
		{EventID: "10000000-0000-4000-8000-000000000002", Outcome: "retryable_failure"},
		{EventID: "10000000-0000-4000-8000-000000000003", Outcome: "conflict"},
	}, map[string]int{"10000000-0000-4000-8000-000000000002": 3}, now, func(limit int64) int64 { return limit - 1 })
	if err != nil {
		t.Fatalf("decide outcomes: %v", err)
	}
	if got := decisions[0]; got.Action != DeliveryAcknowledge || got.OutcomeCode != "accepted" || got.ReceiptDigest != "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" {
		t.Fatalf("accepted decision = %+v", got)
	}
	if got := decisions[1]; got.Action != DeliveryRetry || got.OutcomeCode != "retryable" || !got.NextAttemptAt.Equal(now.Add(8*time.Second-time.Nanosecond)) {
		t.Fatalf("retry decision = %+v", got)
	}
	if got := decisions[2]; got.Action != DeliveryQuarantine || got.OutcomeCode != "conflict" || got.QuarantineOwnerRole != "data_feed_operator" || !got.QuarantineSLADueAt.Equal(now.Add(24*time.Hour)) {
		t.Fatalf("conflict decision = %+v", got)
	}
}

func TestDecideReceiptOutcomesRequiresOneReceiptForEveryFencedLease(t *testing.T) {
	_, err := DecideReceiptOutcomes(
		[]BatchItem{{EventID: "10000000-0000-4000-8000-000000000001", LeaseGeneration: 1}},
		nil, nil, time.Now().UTC(), nil,
	)
	if err == nil {
		t.Fatal("expected missing receipt decision to be rejected")
	}
}

func TestDecideReceiptOutcomesPreservesReplayRunFence(t *testing.T) {
	items := []BatchItem{{
		EventID:         "10000000-0000-4000-8000-000000000041",
		LeaseGeneration: 3,
		ReplayRunID:     "10000000-0000-4000-8000-000000000042",
	}}
	receipts := []ReceiptItem{{
		EventID:               "10000000-0000-4000-8000-000000000041",
		Outcome:               "accepted",
		ManifestReceiptDigest: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
	}}
	decisions, err := DecideReceiptOutcomes(items, receipts, nil, time.Date(2026, 7, 30, 11, 0, 0, 0, time.UTC), nil)
	if err != nil || len(decisions) != 1 || decisions[0].ReplayRunID != items[0].ReplayRunID {
		t.Fatalf("replay receipt decision=%+v err=%v", decisions, err)
	}
}

func TestDecideFailureOutcomesRetriesOnlyRetryableTransportFailures(t *testing.T) {
	now := time.Date(2026, 7, 30, 9, 0, 0, 0, time.UTC)
	items := []BatchItem{{EventID: "10000000-0000-4000-8000-000000000001", LeaseGeneration: 2}}
	retry, err := DecideFailureOutcomes(items, map[string]int{"10000000-0000-4000-8000-000000000001": 1}, &DeliveryFailure{HTTPStatus: 503, Retryable: true, Code: "unavailable"}, now, func(limit int64) int64 { return limit - 1 })
	if err != nil || retry[0].Action != DeliveryRetry || retry[0].OutcomeCode != "retryable" || !retry[0].NextAttemptAt.Equal(now.Add(2*time.Second-time.Nanosecond)) {
		t.Fatalf("retry decision = %+v, err = %v", retry, err)
	}
	quarantine, err := DecideFailureOutcomes(items, nil, &DeliveryFailure{HTTPStatus: 422, Code: "validation_rejected"}, now, nil)
	if err != nil || quarantine[0].Action != DeliveryQuarantine || quarantine[0].OutcomeCode != "validation_rejected" {
		t.Fatalf("quarantine decision = %+v, err = %v", quarantine, err)
	}
	exhausted, err := DecideFailureOutcomes(items, map[string]int{"10000000-0000-4000-8000-000000000001": 7}, &DeliveryFailure{HTTPStatus: 503, Retryable: true, Code: "unavailable"}, now, nil)
	if err != nil || exhausted[0].Action != DeliveryQuarantine || exhausted[0].OutcomeCode != "retry_exhausted" {
		t.Fatalf("exhausted retry decision = %+v, err = %v", exhausted, err)
	}
}
