package datafeed

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
)

const (
	maxBatchItems = 100
	maxEventBytes = 1 << 20
	maxBatchBytes = 10 << 20
)

// BatchItem is an immutable event reconstructed by the publisher from its
// encrypted outbox row under a fenced lease. Its digest is the locked v3
// event-content digest, not a transport or receipt value.
type BatchItem struct {
	Event              map[string]any
	EventID            string
	EventContentDigest string
	LeaseGeneration    int64
	// ReplayRunID is empty for the original delivery lane. A non-empty value
	// identifies a separate immutable replay lane for the same event bytes.
	ReplayRunID string
}

// PreparedBatch is exactly the JSON payload and identities submitted over the
// direct-mTLS boundary. It has no arbitrary headers or hidden members.
type PreparedBatch struct {
	BatchID               string
	AttemptID             string
	ExpectedItemSetDigest string
	Items                 []BatchItem
	Body                  []byte
}

func BuildBatch(items []BatchItem) (PreparedBatch, error) {
	return buildBatch(items, NewEventID)
}

func buildBatch(items []BatchItem, newID func() (string, error)) (PreparedBatch, error) {
	if len(items) == 0 || len(items) > maxBatchItems {
		return PreparedBatch{}, fmt.Errorf("datafeed batch must contain 1-%d items", maxBatchItems)
	}
	ordered := append([]BatchItem(nil), items...)
	sort.Slice(ordered, func(left, right int) bool { return ordered[left].EventID < ordered[right].EventID })
	seen := make(map[string]struct{}, len(ordered))
	setItems := make([]any, 0, len(ordered))
	events := make([]any, 0, len(ordered))
	for _, item := range ordered {
		if item.Event == nil || item.EventID == "" || item.LeaseGeneration < 1 || !isSHA256(item.EventContentDigest) {
			return PreparedBatch{}, fmt.Errorf("datafeed batch item is incomplete")
		}
		if eventID, _ := item.Event["event_id"].(string); eventID != item.EventID {
			return PreparedBatch{}, fmt.Errorf("datafeed batch event identity does not match its immutable header")
		}
		if _, exists := seen[item.EventID]; exists {
			return PreparedBatch{}, fmt.Errorf("datafeed batch contains duplicate event identity")
		}
		seen[item.EventID] = struct{}{}
		eventBytes, err := json.Marshal(item.Event)
		if err != nil || len(eventBytes) > maxEventBytes {
			return PreparedBatch{}, fmt.Errorf("datafeed batch event exceeds its locked byte limit")
		}
		setItems = append(setItems, map[string]any{"event_id": item.EventID, "event_content_digest": item.EventContentDigest})
		events = append(events, item.Event)
	}
	setDigest, err := expectedItemSetDigest(setItems)
	if err != nil {
		return PreparedBatch{}, err
	}
	batchID, err := newID()
	if err != nil {
		return PreparedBatch{}, err
	}
	attemptID, err := newID()
	if err != nil {
		return PreparedBatch{}, err
	}
	body, err := json.Marshal(map[string]any{
		"batch_id":                 batchID,
		"attempt_id":               attemptID,
		"expected_item_count":      len(ordered),
		"expected_item_set_digest": setDigest,
		"events":                   events,
	})
	if err != nil || len(body) > maxBatchBytes {
		return PreparedBatch{}, fmt.Errorf("datafeed batch exceeds its locked byte limit")
	}
	return PreparedBatch{BatchID: batchID, AttemptID: attemptID, ExpectedItemSetDigest: setDigest, Items: ordered, Body: body}, nil
}

func expectedItemSetDigest(items []any) (string, error) {
	encoded, err := typedLengthPrefixed(items)
	if err != nil {
		return "", fmt.Errorf("encode v3 expected item set: %w", err)
	}
	domain := []byte("aviacore.expected_item_set.v1\x00")
	sum := sha256.Sum256(append(append([]byte{}, domain...), append(typedLengthPrefixedDomain, encoded...)...))
	return hex.EncodeToString(sum[:]), nil
}
