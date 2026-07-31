package datafeed

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type batchSubmitter interface {
	Submit(context.Context, BatchRequest) (*http.Response, error)
}

// Publisher connects the pure bounded-batch and receipt validators to the
// direct-mTLS transport. It cannot acknowledge database state; callers must
// persist the validated receipt under each item's fenced lease.
type Publisher struct {
	Client batchSubmitter
	NewID  func() (string, error)
}

// DeliveryFailure exposes only a stable, value-free disposition for the
// worker. It deliberately does not carry receiver bodies, headers, payloads,
// certificate material, or arbitrary transport error strings.
type DeliveryFailure struct {
	HTTPStatus int
	Retryable  bool
	Code       string
}

func (failure *DeliveryFailure) Error() string {
	return "datafeed delivery " + failure.Code
}

func (publisher Publisher) Deliver(ctx context.Context, items []BatchItem, replayID string) ([]ReceiptItem, error) {
	if publisher.Client == nil || replayID == "" {
		return nil, fmt.Errorf("datafeed publisher requires direct mTLS client and replay identity")
	}
	newID := publisher.NewID
	if newID == nil {
		newID = NewEventID
	}
	batch, err := buildBatch(items, newID)
	if err != nil {
		return nil, err
	}
	requestID, err := newID()
	if err != nil {
		return nil, err
	}
	response, err := publisher.Client.Submit(ctx, BatchRequest{
		RequestID: requestID, IdempotencyKey: batch.BatchID + ":" + batch.ExpectedItemSetDigest,
		ReplayID: replayID, Body: batch.Body,
	})
	if err != nil {
		return nil, &DeliveryFailure{Retryable: true, Code: "transport_unavailable"}
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK && response.StatusCode != http.StatusCreated && response.StatusCode != http.StatusMultiStatus {
		return nil, classifyHTTPFailure(response.StatusCode)
	}
	limited := io.LimitReader(response.Body, maxBatchBytes)
	var receipt BatchReceipt
	decoder := json.NewDecoder(limited)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&receipt); err != nil {
		return nil, fmt.Errorf("decode datafeed batch receipt: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return nil, fmt.Errorf("decode datafeed batch receipt: trailing JSON")
	}
	return ValidateReceipt(batch, requestID, receipt)
}

func classifyHTTPFailure(status int) *DeliveryFailure {
	failure := &DeliveryFailure{HTTPStatus: status, Code: "receiver_failure"}
	switch status {
	case http.StatusTooManyRequests:
		failure.Retryable, failure.Code = true, "rate_limited"
	case http.StatusServiceUnavailable:
		failure.Retryable, failure.Code = true, "unavailable"
	case http.StatusInternalServerError:
		failure.Retryable, failure.Code = true, "receiver_error"
	case http.StatusConflict:
		failure.Code = "conflict"
	case http.StatusRequestEntityTooLarge:
		failure.Code = "payload_limit_exceeded"
	case http.StatusUnprocessableEntity:
		failure.Code = "validation_rejected"
	case http.StatusUnauthorized, http.StatusForbidden:
		failure.Code = "identity_denied"
	default:
		failure.Code = "unexpected_http_status"
	}
	return failure
}
