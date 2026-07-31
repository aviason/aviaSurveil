package datafeed

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// LeasedOutboxItem carries only an immutable reconstructed event and its
// current fence. Plaintext storage details remain behind the claim source.
type LeasedOutboxItem struct {
	BatchItem
	AttemptCount int
	OccurredAt   time.Time
}

type LeaseSource interface {
	Claim(context.Context, time.Duration, int) ([]LeasedOutboxItem, error)
}

type OutcomeObserver interface {
	RecordOutcome(context.Context, string, int)
	RecordPendingAge(context.Context, time.Duration)
	RecordAcknowledgementLag(context.Context, time.Duration)
}

// OutboxWorker executes one bounded delivery pass. Its source is responsible
// for scoped claims; its processor is responsible for atomic fenced updates.
type OutboxWorker struct {
	Source        LeaseSource
	Publisher     Publisher
	Processor     ReceiptProcessor
	ReplayID      string
	LeaseDuration time.Duration
	Limit         int
	Observer      OutcomeObserver
}

func (worker OutboxWorker) ProcessOnce(ctx context.Context) (int, error) {
	if worker.Source == nil || worker.ReplayID == "" || worker.LeaseDuration <= 0 || worker.Limit < 1 || worker.Limit > maxBatchItems {
		return 0, fmt.Errorf("datafeed worker requires a bounded scoped lease configuration")
	}
	items, err := worker.Source.Claim(ctx, worker.LeaseDuration, worker.Limit)
	if err != nil || len(items) == 0 {
		return len(items), err
	}
	batchItems := make([]BatchItem, 0, len(items))
	attemptCounts := make(map[string]int, len(items))
	for _, item := range items {
		if worker.Observer != nil && !item.OccurredAt.IsZero() {
			worker.Observer.RecordPendingAge(ctx, processorNow(worker.Processor).Sub(item.OccurredAt))
		}
		batchItems = append(batchItems, item.BatchItem)
		attemptCounts[item.EventID] = item.AttemptCount
	}
	receipts, err := worker.Publisher.Deliver(ctx, batchItems, worker.ReplayID)
	if err != nil {
		var failure *DeliveryFailure
		if !errors.As(err, &failure) {
			failure = &DeliveryFailure{Code: "protocol_integrity"}
		}
		decisions, decisionErr := DecideFailureOutcomes(batchItems, attemptCounts, failure, processorNow(worker.Processor), worker.Processor.RandInt)
		if decisionErr != nil {
			return 0, decisionErr
		}
		if worker.Processor.Recorder == nil {
			return 0, fmt.Errorf("datafeed worker receipt processor requires a recorder")
		}
		if recordErr := worker.Processor.Recorder.Record(ctx, decisions); recordErr != nil {
			return 0, recordErr
		}
		worker.recordDecisions(ctx, decisions)
		return len(items), nil
	}
	if err := worker.Processor.Persist(ctx, batchItems, receipts, attemptCounts); err != nil {
		return 0, err
	}
	byID := make(map[string]LeasedOutboxItem, len(items))
	for _, item := range items {
		byID[item.EventID] = item
	}
	for _, receipt := range receipts {
		worker.record(ctx, receipt.Outcome)
		if (receipt.Outcome == "accepted" || receipt.Outcome == "duplicate") && worker.Observer != nil && !byID[receipt.EventID].OccurredAt.IsZero() {
			worker.Observer.RecordAcknowledgementLag(ctx, processorNow(worker.Processor).Sub(byID[receipt.EventID].OccurredAt))
		}
	}
	return len(items), nil
}

func (worker OutboxWorker) recordDecisions(ctx context.Context, decisions []DeliveryDecision) {
	for _, decision := range decisions {
		worker.record(ctx, decision.OutcomeCode)
	}
}
func (worker OutboxWorker) record(ctx context.Context, outcome string) {
	if worker.Observer != nil {
		worker.Observer.RecordOutcome(ctx, outcome, 1)
	}
}

func processorNow(processor ReceiptProcessor) time.Time {
	if processor.Now != nil {
		return processor.Now().UTC()
	}
	return time.Now().UTC()
}
