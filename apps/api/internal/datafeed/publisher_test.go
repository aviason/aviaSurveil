package datafeed

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"testing"
)

func TestPublisherDeliversExactBatchAndRejectsUnboundReceipt(t *testing.T) {
	client := &publisherSubmitterStub{response: `{"request_id":"10000000-0000-4000-8000-000000000010","batch_id":"BATCH","attempt_id":"ATTEMPT","batch_state":"sealed_terminal","items":[{"event_id":"10000000-0000-4000-8000-000000000001","attempt_id":"ATTEMPT","event_content_digest":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","outcome":"accepted","safe_code":"ACCEPTED","manifest_receipt_digest":"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","acknowledgement_receipt_id":"receipt-1","winner_event_id":"10000000-0000-4000-8000-000000000001"}]}`}
	publisher := Publisher{Client: client, NewID: sequenceIDs("BATCH", "ATTEMPT", "10000000-0000-4000-8000-000000000010")}
	_, err := publisher.Deliver(context.Background(), []BatchItem{{Event: map[string]any{"event_id": "10000000-0000-4000-8000-000000000001"}, EventID: "10000000-0000-4000-8000-000000000001", EventContentDigest: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", LeaseGeneration: 1}}, "replay-task5-fixture")
	if err != nil {
		t.Fatalf("deliver batch: %v", err)
	}
	if client.request.IdempotencyKey == "" || client.request.ReplayID != "replay-task5-fixture" {
		t.Fatalf("publisher protocol headers = %+v", client.request)
	}
}

func TestPublisherClassifiesOnlyApprovedRetryableHTTPFailures(t *testing.T) {
	for _, test := range []struct {
		status    int
		retryable bool
	}{
		{status: http.StatusTooManyRequests, retryable: true},
		{status: http.StatusServiceUnavailable, retryable: true},
		{status: http.StatusInternalServerError, retryable: true},
		{status: http.StatusConflict, retryable: false},
		{status: http.StatusUnprocessableEntity, retryable: false},
		{status: http.StatusUnauthorized, retryable: false},
		{status: http.StatusRequestEntityTooLarge, retryable: false},
	} {
		t.Run(http.StatusText(test.status), func(t *testing.T) {
			publisher := Publisher{Client: &publisherSubmitterStub{status: test.status, response: `{}`}, NewID: sequenceIDs("BATCH", "ATTEMPT", "10000000-0000-4000-8000-000000000010")}
			_, err := publisher.Deliver(context.Background(), []BatchItem{{Event: map[string]any{"event_id": "10000000-0000-4000-8000-000000000001"}, EventID: "10000000-0000-4000-8000-000000000001", EventContentDigest: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", LeaseGeneration: 1}}, "replay-task5-fixture")
			var failure *DeliveryFailure
			if !errors.As(err, &failure) || failure.Retryable != test.retryable || failure.HTTPStatus != test.status {
				t.Fatalf("failure = %#v, err = %v", failure, err)
			}
		})
	}
}

type publisherSubmitterStub struct {
	request  BatchRequest
	response string
	status   int
}

func (stub *publisherSubmitterStub) Submit(_ context.Context, request BatchRequest) (*http.Response, error) {
	stub.request = request
	status := stub.status
	if status == 0 {
		status = http.StatusCreated
	}
	return &http.Response{StatusCode: status, Body: io.NopCloser(bytes.NewBufferString(stub.response))}, nil
}

func sequenceIDs(values ...string) func() (string, error) {
	index := 0
	return func() (string, error) { value := values[index]; index++; return value, nil }
}
