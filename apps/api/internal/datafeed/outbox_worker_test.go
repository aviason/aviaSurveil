package datafeed

import (
	"context"
	"testing"
	"time"
)

func TestOutboxWorkerClaimsPublishesAndPersistsBoundReceipt(t *testing.T) {
	occurredAt := time.Date(2026, 7, 30, 8, 55, 0, 0, time.UTC)
	source := &leaseSourceStub{items: []LeasedOutboxItem{{BatchItem: BatchItem{Event: map[string]any{"event_id": "10000000-0000-4000-8000-000000000001"}, EventID: "10000000-0000-4000-8000-000000000001", EventContentDigest: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", LeaseGeneration: 1}, OccurredAt: occurredAt}}}
	recorder := &decisionRecorderStub{}
	observer := &outcomeObserverStub{}
	worker := OutboxWorker{
		Source:    source,
		Publisher: Publisher{Client: &publisherSubmitterStub{response: `{"request_id":"10000000-0000-4000-8000-000000000010","batch_id":"BATCH","attempt_id":"ATTEMPT","batch_state":"sealed_terminal","items":[{"event_id":"10000000-0000-4000-8000-000000000001","attempt_id":"ATTEMPT","event_content_digest":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","outcome":"accepted","safe_code":"ACCEPTED","manifest_receipt_digest":"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","acknowledgement_receipt_id":"receipt-1","winner_event_id":"10000000-0000-4000-8000-000000000001"}]}`}, NewID: sequenceIDs("BATCH", "ATTEMPT", "10000000-0000-4000-8000-000000000010")},
		Processor: ReceiptProcessor{Recorder: recorder, Now: func() time.Time { return time.Date(2026, 7, 30, 9, 0, 0, 0, time.UTC) }},
		ReplayID:  "task5-local-replay", LeaseDuration: time.Minute, Limit: 100, Observer: observer,
	}
	processed, err := worker.ProcessOnce(context.Background())
	if err != nil {
		t.Fatalf("process once: %v", err)
	}
	if processed != 1 || len(recorder.decisions) != 1 || recorder.decisions[0].Action != DeliveryAcknowledge || source.limit != 100 {
		t.Fatalf("processed=%d decisions=%+v source=%+v", processed, recorder.decisions, source)
	}
	if observer.outcomes["accepted"] != 1 {
		t.Fatalf("outcomes = %+v", observer.outcomes)
	}
	if len(observer.pendingAges) != 1 || observer.pendingAges[0] != 5*time.Minute {
		t.Fatalf("pending ages = %+v", observer.pendingAges)
	}
	if len(observer.acknowledgementLags) != 1 || observer.acknowledgementLags[0] != 5*time.Minute {
		t.Fatalf("acknowledgement lags = %+v", observer.acknowledgementLags)
	}
}

type leaseSourceStub struct {
	items []LeasedOutboxItem
	limit int
}

func (stub *leaseSourceStub) Claim(_ context.Context, _ time.Duration, limit int) ([]LeasedOutboxItem, error) {
	stub.limit = limit
	return stub.items, nil
}

type outcomeObserverStub struct {
	outcomes            map[string]int
	pendingAges         []time.Duration
	acknowledgementLags []time.Duration
}

func (stub *outcomeObserverStub) RecordOutcome(_ context.Context, outcome string, count int) {
	if stub.outcomes == nil {
		stub.outcomes = map[string]int{}
	}
	stub.outcomes[outcome] += count
}
func (stub *outcomeObserverStub) RecordPendingAge(_ context.Context, age time.Duration) {
	stub.pendingAges = append(stub.pendingAges, age)
}
func (stub *outcomeObserverStub) RecordAcknowledgementLag(_ context.Context, lag time.Duration) {
	stub.acknowledgementLags = append(stub.acknowledgementLags, lag)
}
