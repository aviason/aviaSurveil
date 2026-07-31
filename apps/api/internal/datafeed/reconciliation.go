package datafeed

import (
	"fmt"
	"strings"
	"time"
)

const (
	DeliveryAcknowledge = "acknowledge"
	DeliveryRetry       = "retry"
	DeliveryQuarantine  = "quarantine"

	quarantineOwnerRole = "data_feed_operator"
	quarantineSLATime   = 24 * time.Hour
)

// DeliveryDecision is a fully fenced persistence instruction derived only
// from a receipt already bound to the submitted v3 event bytes.
type DeliveryDecision struct {
	EventID             string
	LeaseGeneration     int64
	ReplayRunID         string
	RecordedAt          time.Time
	Action              string
	OutcomeCode         string
	DiagnosticCode      string
	ReceiptDigest       string
	NextAttemptAt       time.Time
	QuarantineOwnerRole string
	QuarantineSLADueAt  time.Time
}

// DecideReceiptOutcomes turns one validated receipt result into one explicit
// state transition per fenced event. No outcome is silently dropped.
func DecideReceiptOutcomes(items []BatchItem, receipts []ReceiptItem, attemptCounts map[string]int, now time.Time, randInt func(int64) int64) ([]DeliveryDecision, error) {
	if len(items) == 0 || len(items) != len(receipts) {
		return nil, fmt.Errorf("datafeed receipt decisions require one result for every fenced lease")
	}
	byID := make(map[string]ReceiptItem, len(receipts))
	for _, receipt := range receipts {
		if receipt.EventID == "" {
			return nil, fmt.Errorf("datafeed receipt decision has empty event identity")
		}
		if _, exists := byID[receipt.EventID]; exists {
			return nil, fmt.Errorf("datafeed receipt decision has duplicate event identity")
		}
		byID[receipt.EventID] = receipt
	}
	decisions := make([]DeliveryDecision, 0, len(items))
	for _, item := range items {
		receipt, exists := byID[item.EventID]
		if !exists || item.LeaseGeneration < 1 {
			return nil, fmt.Errorf("datafeed receipt decision does not match fenced lease")
		}
		decision := DeliveryDecision{EventID: item.EventID, LeaseGeneration: item.LeaseGeneration, ReplayRunID: item.ReplayRunID, RecordedAt: now}
		switch receipt.Outcome {
		case "accepted", "duplicate":
			decision.Action = DeliveryAcknowledge
			decision.OutcomeCode = receipt.Outcome
			decision.ReceiptDigest = receipt.ManifestReceiptDigest
		case "retryable_failure":
			setRetryOrExhausted(&decision, attemptCounts[item.EventID]+1, now, randInt)
		case "conflict", "rejected_quarantine":
			decision.Action = DeliveryQuarantine
			if receipt.Outcome == "rejected_quarantine" {
				decision.OutcomeCode = "validation_rejected"
			} else {
				decision.OutcomeCode = "conflict"
			}
			decision.QuarantineOwnerRole = quarantineOwnerRole
			decision.QuarantineSLADueAt = now.Add(quarantineSLATime)
		default:
			return nil, fmt.Errorf("datafeed receipt has unknown outcome")
		}
		decision.DiagnosticCode = strings.ToLower(receipt.SafeCode)
		decisions = append(decisions, decision)
	}
	return decisions, nil
}

// DecideFailureOutcomes gives every currently leased item an explicit safe
// disposition after a value-free publisher failure. It is never used for a
// malformed per-item receipt: such a receipt is a protocol integrity failure
// and therefore requires manual quarantine rather than an acknowledgement.
func DecideFailureOutcomes(items []BatchItem, attemptCounts map[string]int, failure *DeliveryFailure, now time.Time, randInt func(int64) int64) ([]DeliveryDecision, error) {
	if len(items) == 0 || failure == nil {
		return nil, fmt.Errorf("datafeed failure decisions require fenced items and a classified failure")
	}
	decisions := make([]DeliveryDecision, 0, len(items))
	for _, item := range items {
		if item.EventID == "" || item.LeaseGeneration < 1 {
			return nil, fmt.Errorf("datafeed failure decision does not match fenced lease")
		}
		decision := DeliveryDecision{EventID: item.EventID, LeaseGeneration: item.LeaseGeneration, ReplayRunID: item.ReplayRunID, RecordedAt: now}
		if failure.Retryable {
			setRetryOrExhausted(&decision, attemptCounts[item.EventID]+1, now, randInt)
		} else {
			decision.Action = DeliveryQuarantine
			decision.QuarantineOwnerRole = quarantineOwnerRole
			decision.QuarantineSLADueAt = now.Add(quarantineSLATime)
			if failure.Code == "conflict" {
				decision.OutcomeCode = "conflict"
			} else {
				decision.OutcomeCode = "validation_rejected"
			}
		}
		decision.DiagnosticCode = failure.Code
		decisions = append(decisions, decision)
	}
	return decisions, nil
}

func setRetryOrExhausted(decision *DeliveryDecision, attemptCount int, now time.Time, randInt func(int64) int64) {
	if OperatorAlertRequired(attemptCount) {
		decision.Action = DeliveryQuarantine
		decision.OutcomeCode = "retry_exhausted"
		decision.QuarantineOwnerRole = quarantineOwnerRole
		decision.QuarantineSLADueAt = now.Add(quarantineSLATime)
		return
	}
	decision.Action = DeliveryRetry
	decision.OutcomeCode = "retryable"
	decision.NextAttemptAt = now.Add(RetryDelay(attemptCount, randInt))
}
