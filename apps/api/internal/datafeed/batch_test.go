package datafeed

import "testing"

func TestBuildBatchSortsItemsAndBindsTheExactV3ItemSet(t *testing.T) {
	batch, err := BuildBatch([]BatchItem{
		{Event: map[string]any{"event_id": "20000000-0000-4000-8000-000000000002"}, EventID: "20000000-0000-4000-8000-000000000002", EventContentDigest: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", LeaseGeneration: 3},
		{Event: map[string]any{"event_id": "10000000-0000-4000-8000-000000000001"}, EventID: "10000000-0000-4000-8000-000000000001", EventContentDigest: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", LeaseGeneration: 2},
	})
	if err != nil {
		t.Fatalf("build batch: %v", err)
	}
	if len(batch.Items) != 2 || batch.Items[0].EventID != "10000000-0000-4000-8000-000000000001" {
		t.Fatalf("batch item order = %+v, want event ID ascending", batch.Items)
	}
	if batch.ExpectedItemSetDigest != "56f3f28edccf4b95fb92195f3aadfad82c32420c645e713ad61811a3d7f1c778" {
		t.Fatalf("item-set digest = %q", batch.ExpectedItemSetDigest)
	}
	if len(batch.Body) == 0 || len(batch.Body) > maxBatchBytes {
		t.Fatalf("batch body length = %d", len(batch.Body))
	}
}

func TestBuildBatchRejectsDuplicatesAndOutOfRangeItems(t *testing.T) {
	item := BatchItem{Event: map[string]any{"event_id": "10000000-0000-4000-8000-000000000001"}, EventID: "10000000-0000-4000-8000-000000000001", EventContentDigest: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}
	if _, err := BuildBatch([]BatchItem{item, item}); err == nil {
		t.Fatal("duplicate event ID was accepted")
	}
	if _, err := BuildBatch(nil); err == nil {
		t.Fatal("empty batch was accepted")
	}
}
