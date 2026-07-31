package datafeed

import (
	"context"
	"testing"
	"time"
)

func TestReceiptProcessorPersistsEveryValidReceiptDecision(t *testing.T) {
	recorder := &decisionRecorderStub{}
	processor := ReceiptProcessor{
		Recorder: recorder,
		Now:      func() time.Time { return time.Date(2026, 7, 30, 9, 0, 0, 0, time.UTC) },
		RandInt:  func(limit int64) int64 { return limit - 1 },
	}
	err := processor.Persist(context.Background(), []BatchItem{
		{EventID: "10000000-0000-4000-8000-000000000001", LeaseGeneration: 2},
		{EventID: "10000000-0000-4000-8000-000000000002", LeaseGeneration: 3},
	}, []ReceiptItem{
		{EventID: "10000000-0000-4000-8000-000000000001", Outcome: "duplicate", ManifestReceiptDigest: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
		{EventID: "10000000-0000-4000-8000-000000000002", Outcome: "rejected_quarantine"},
	}, map[string]int{})
	if err != nil {
		t.Fatalf("persist receipt: %v", err)
	}
	if len(recorder.decisions) != 2 || recorder.decisions[0].Action != DeliveryAcknowledge || recorder.decisions[0].OutcomeCode != "duplicate" || recorder.decisions[1].Action != DeliveryQuarantine || recorder.decisions[1].OutcomeCode != "validation_rejected" {
		t.Fatalf("persisted decisions = %+v", recorder.decisions)
	}
}

type decisionRecorderStub struct{ decisions []DeliveryDecision }

func (stub *decisionRecorderStub) Record(_ context.Context, decisions []DeliveryDecision) error {
	stub.decisions = append(stub.decisions, decisions...)
	return nil
}
