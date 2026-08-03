package agacandidatedemo_test

import (
	"testing"
	"time"

	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/preproddata/agacandidatedemo"
)

func TestOverlayResultAndCleanupTombstoneAreAppendOnly(t *testing.T) {
	result, err := agacandidatedemo.BuildResult(agacandidatedemo.ResultInput{RunID: "aga-demo-run-20260801", IntentDigest: digest("a"), SealDigest: digest("b"), CompletedAt: time.Date(2026, 8, 1, 12, 10, 0, 0, time.UTC)})
	if err != nil {
		t.Fatalf("build result: %v", err)
	}
	if err := result.Validate(); err != nil {
		t.Fatalf("validate result: %v", err)
	}
	tombstone, err := agacandidatedemo.BuildCleanupTombstone(agacandidatedemo.CleanupTombstoneInput{RunID: result.RunID, IntentDigest: result.IntentDigest, ResultDigest: result.ResultDigest, CleanedAt: time.Date(2026, 8, 1, 13, 0, 0, 0, time.UTC)})
	if err != nil {
		t.Fatalf("build cleanup tombstone: %v", err)
	}
	if err := tombstone.Validate(); err != nil {
		t.Fatalf("validate cleanup tombstone: %v", err)
	}
}
