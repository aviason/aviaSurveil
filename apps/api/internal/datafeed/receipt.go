package datafeed

import (
	"fmt"
	"regexp"
)

var safeCodePattern = regexp.MustCompile(`^[A-Z][A-Z0-9_]{0,95}$`)

// BatchReceipt is the closed, value-free subset of an AviaCore response used
// before any fenced producer acknowledgement can be persisted.
type BatchReceipt struct {
	RequestID                    string        `json:"request_id"`
	BatchID                      string        `json:"batch_id"`
	AttemptID                    string        `json:"attempt_id"`
	BatchState                   string        `json:"batch_state"`
	LandingManifestReceiptDigest *string       `json:"landing_manifest_receipt_digest"`
	Items                        []ReceiptItem `json:"items"`
}

type ReceiptItem struct {
	EventID                  string           `json:"event_id"`
	AttemptID                string           `json:"attempt_id"`
	EventContentDigest       string           `json:"event_content_digest"`
	Outcome                  string           `json:"outcome"`
	SafeCode                 string           `json:"safe_code"`
	RetryAfterSeconds        *int             `json:"retry_after_seconds"`
	Errors                   []ValueFreeError `json:"errors"`
	ManifestReceiptDigest    string           `json:"manifest_receipt_digest"`
	AcknowledgementReceiptID string           `json:"acknowledgement_receipt_id"`
	WinnerEventID            string           `json:"winner_event_id"`
}

type ValueFreeError struct {
	StableCode      string `json:"stable_code"`
	JSONPointer     string `json:"json_pointer"`
	ContractID      string `json:"contract_id"`
	ContractVersion string `json:"contract_version"`
}

// ValidateReceipt requires exactly one correlated result for every submitted
// event. It refuses batch-level success, missing/extra results, and any
// success that is not bound to the exact v3 event content digest.
func ValidateReceipt(batch PreparedBatch, requestID string, receipt BatchReceipt) ([]ReceiptItem, error) {
	if requestID == "" || receipt.RequestID != requestID || receipt.BatchID != batch.BatchID || receipt.AttemptID != batch.AttemptID {
		return nil, fmt.Errorf("datafeed receipt does not correlate to the submitted request, batch, and attempt")
	}
	if receipt.BatchState != "sealed_terminal" && receipt.BatchState != "unsealed_retryable" {
		return nil, fmt.Errorf("datafeed receipt has no valid batch state")
	}
	if receipt.LandingManifestReceiptDigest != nil && !isSHA256(*receipt.LandingManifestReceiptDigest) {
		return nil, fmt.Errorf("datafeed receipt has invalid landing manifest digest")
	}
	expected := make(map[string]BatchItem, len(batch.Items))
	for _, item := range batch.Items {
		expected[item.EventID] = item
	}
	if len(receipt.Items) != len(expected) {
		return nil, fmt.Errorf("datafeed receipt has missing or extra items")
	}
	seen := make(map[string]struct{}, len(receipt.Items))
	hasRetryable := false
	for _, item := range receipt.Items {
		expectedItem, exists := expected[item.EventID]
		if !exists || item.AttemptID != batch.AttemptID {
			return nil, fmt.Errorf("datafeed receipt item does not match the submitted batch")
		}
		if _, duplicate := seen[item.EventID]; duplicate {
			return nil, fmt.Errorf("datafeed receipt contains duplicate event identity")
		}
		seen[item.EventID] = struct{}{}
		if item.EventContentDigest != expectedItem.EventContentDigest {
			return nil, fmt.Errorf("datafeed receipt content digest does not match the submitted event")
		}
		if !safeCodePattern.MatchString(item.SafeCode) {
			return nil, fmt.Errorf("datafeed receipt item has no valid safe code")
		}
		if item.RetryAfterSeconds != nil && (*item.RetryAfterSeconds < 1 || *item.RetryAfterSeconds > 3600) {
			return nil, fmt.Errorf("datafeed receipt item has invalid retry delay")
		}
		if len(item.Errors) > 32 {
			return nil, fmt.Errorf("datafeed receipt item has too many value-free errors")
		}
		for _, itemError := range item.Errors {
			if !safeCodePattern.MatchString(itemError.StableCode) || len(itemError.JSONPointer) > 256 || itemError.ContractID != contractID || itemError.ContractVersion != contractVersion {
				return nil, fmt.Errorf("datafeed receipt item has invalid value-free error")
			}
		}
		switch item.Outcome {
		case "accepted", "duplicate":
			if !isSHA256(item.ManifestReceiptDigest) || item.AcknowledgementReceiptID == "" || item.WinnerEventID != item.EventID {
				return nil, fmt.Errorf("datafeed successful receipt lacks its exact manifest/winner binding")
			}
		case "conflict", "rejected_quarantine", "retryable_failure":
			if item.Outcome == "retryable_failure" {
				hasRetryable = true
			}
			if item.ManifestReceiptDigest != "" || item.AcknowledgementReceiptID != "" || item.WinnerEventID != "" {
				return nil, fmt.Errorf("datafeed non-success receipt contains an acknowledgement binding")
			}
		default:
			return nil, fmt.Errorf("datafeed receipt has an unknown outcome")
		}
	}
	if (receipt.BatchState == "unsealed_retryable") != hasRetryable {
		return nil, fmt.Errorf("datafeed receipt batch state does not match item terminality")
	}
	return receipt.Items, nil
}
