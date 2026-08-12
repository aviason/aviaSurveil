package idempotency_test

import (
	"testing"

	"github.com/aviason/aviaSurveil/internal/platform/idempotency"
)

func TestSemanticHashIsStableForEquivalentJSONObjects(t *testing.T) {
	t.Parallel()

	left, err := idempotency.SemanticHash(map[string]any{"status": "OPEN", "revision": 2})
	if err != nil {
		t.Fatalf("hash left payload: %v", err)
	}
	right, err := idempotency.SemanticHash(map[string]any{"revision": 2, "status": "OPEN"})
	if err != nil {
		t.Fatalf("hash right payload: %v", err)
	}
	if left != right {
		t.Fatalf("equivalent payload hashes differ: %s != %s", left, right)
	}
}

func TestSemanticHashChangesWithMeaningfulPayloadChange(t *testing.T) {
	t.Parallel()

	left, _ := idempotency.SemanticHash(map[string]any{"revision": 1})
	right, _ := idempotency.SemanticHash(map[string]any{"revision": 2})
	if left == right {
		t.Fatalf("different payloads share hash %s", left)
	}
}
