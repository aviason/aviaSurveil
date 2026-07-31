package datafeed

import "testing"

func TestValidateReceiptRejectsMissingExtraAndDigestMismatchedItems(t *testing.T) {
	batch := PreparedBatch{BatchID: "batch-1", AttemptID: "attempt-1", Items: []BatchItem{{EventID: "event-1", EventContentDigest: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", LeaseGeneration: 1}}}
	valid := BatchReceipt{RequestID: "request-1", BatchID: "batch-1", AttemptID: "attempt-1", BatchState: "sealed_terminal", Items: []ReceiptItem{{EventID: "event-1", AttemptID: "attempt-1", EventContentDigest: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", Outcome: "accepted", SafeCode: "ACCEPTED", ManifestReceiptDigest: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", AcknowledgementReceiptID: "receipt-1", WinnerEventID: "event-1"}}}
	if _, err := ValidateReceipt(batch, "request-1", valid); err != nil {
		t.Fatalf("validate exact receipt: %v", err)
	}
	valid.Items[0].EventContentDigest = "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
	if _, err := ValidateReceipt(batch, "request-1", valid); err == nil {
		t.Fatal("digest-mismatched receipt was accepted")
	}
	valid.Items[0].EventContentDigest = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	valid.BatchState = ""
	if _, err := ValidateReceipt(batch, "request-1", valid); err == nil {
		t.Fatal("missing batch state was accepted")
	}
	valid.BatchState = "sealed_terminal"
	valid.Items[0].SafeCode = ""
	if _, err := ValidateReceipt(batch, "request-1", valid); err == nil {
		t.Fatal("missing item safe code was accepted")
	}
}

func TestValidateReceiptAcceptsLockedOptionalValueFreeFields(t *testing.T) {
	retryAfter := 10
	manifest := "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	batch := PreparedBatch{BatchID: "batch-1", AttemptID: "attempt-1", Items: []BatchItem{{EventID: "event-1", EventContentDigest: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", LeaseGeneration: 1}}}
	receipt := BatchReceipt{RequestID: "request-1", BatchID: "batch-1", AttemptID: "attempt-1", BatchState: "unsealed_retryable", LandingManifestReceiptDigest: &manifest, Items: []ReceiptItem{{EventID: "event-1", AttemptID: "attempt-1", EventContentDigest: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", Outcome: "retryable_failure", SafeCode: "RATE_LIMITED", RetryAfterSeconds: &retryAfter, Errors: []ValueFreeError{{StableCode: "RATE_LIMITED", JSONPointer: "/events/0", ContractID: contractID, ContractVersion: contractVersion}}}}}
	if _, err := ValidateReceipt(batch, "request-1", receipt); err != nil {
		t.Fatalf("optional locked receipt fields: %v", err)
	}
	invalid := "not-a-digest"
	receipt.LandingManifestReceiptDigest = &invalid
	if _, err := ValidateReceipt(batch, "request-1", receipt); err == nil {
		t.Fatal("invalid landing manifest digest was accepted")
	}
}
