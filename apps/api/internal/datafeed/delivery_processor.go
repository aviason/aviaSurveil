package datafeed

import (
	"context"
	"fmt"
	"time"
)

// DecisionRecorder persists a complete receipt decision set atomically. Its
// implementation must reject stale leases rather than convert them to an
// acknowledgement, retry, or quarantine transition.
type DecisionRecorder interface {
	Record(context.Context, []DeliveryDecision) error
}

// ReceiptProcessor has no transport authority: callers can only pass receipts
// after Publisher.Deliver has correlated and digest-bound them.
type ReceiptProcessor struct {
	Recorder DecisionRecorder
	Now      func() time.Time
	RandInt  func(int64) int64
}

func (processor ReceiptProcessor) Persist(ctx context.Context, items []BatchItem, receipts []ReceiptItem, attemptCounts map[string]int) error {
	if processor.Recorder == nil {
		return fmt.Errorf("datafeed receipt processor requires a fenced decision recorder")
	}
	now := time.Now().UTC()
	if processor.Now != nil {
		now = processor.Now().UTC()
	}
	decisions, err := DecideReceiptOutcomes(items, receipts, attemptCounts, now, processor.RandInt)
	if err != nil {
		return err
	}
	return processor.Recorder.Record(ctx, decisions)
}
