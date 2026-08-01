package regulatory

import (
	"testing"
	"time"
)

func TestReviewCommentLedgerSeparatesVisibilityAndReplays(t *testing.T) {
	ledger := NewReviewCommentLedger()
	first, err := ledger.Append("candidate-1", "admin-1", "INTERNAL_CAA", "private note", "op-1", "idem-1", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	replay, err := ledger.Append("candidate-1", "admin-1", "INTERNAL_CAA", "private note", "op-1", "idem-1", time.Now())
	if err != nil || replay.CommentID != first.CommentID {
		t.Fatalf("comment replay changed identity: first=%+v replay=%+v err=%v", first, replay, err)
	}
	if _, err := ledger.Append("candidate-1", "admin-1", "INTERNAL_CAA", "private note", "op-1", "idem-1", time.Now()); err != nil {
		t.Fatal(err)
	}
	if _, err := ledger.Append("candidate-1", "admin-1", "INTERNAL_CAA", "changed", "op-1", "idem-1", time.Now()); err == nil {
		t.Fatal("divergent replay must be rejected")
	}
}
