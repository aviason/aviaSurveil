package agademoworkspace

import (
	"strings"
	"testing"

	aga "github.com/MarlonJD/aviaSurveil360/apps/api/internal/agaapplicability"
	preprod "github.com/MarlonJD/aviaSurveil360/apps/api/internal/preproddata/agademoworkspace"
)

func TestBatchItemsDigestSupportsAffectedIdentityLists(t *testing.T) {
	items := []preprod.ClassificationItem{
		{Identity: aga.BaseIdentity{PackageVersion: "2026-08-01", PackageJSONSHA256: "package", FormCode: "FSS-AGA-FORM-002", ProposalID: "q-2", Ordinal: 2, TextDigest: "sha256:" + "2" + "0"}},
		{Identity: aga.BaseIdentity{PackageVersion: "2026-08-01", PackageJSONSHA256: "package", FormCode: "FSS-AGA-FORM-002", ProposalID: "q-1", Ordinal: 1, TextDigest: "sha256:" + "1" + "0"}},
	}

	digest := batchItemsDigest(items)
	if len(digest) != len("sha256:")+64 || !strings.HasPrefix(digest, "sha256:") {
		t.Fatalf("batch affected identity digest must be a server-valid digest, got %q", digest)
	}
	if digest != batchItemsDigest([]preprod.ClassificationItem{items[1], items[0]}) {
		t.Fatal("batch affected identity digest must be independent of source order")
	}
	if got, want := batchQuestionKey(items[0]), aga.BaseQuestionReference(items[0].Identity).Key(); got != want {
		t.Fatalf("batch question key must use the question-reference union: got %q want %q", got, want)
	}
}

func TestPreviewProjectionRetainsCanonicalFilterForConsume(t *testing.T) {
	filter := BatchFilter{FormCode: "FSS-AGA-FORM-002", Disposition: "UNSET"}
	record := preprod.SelectionBatchPreviewRecord{
		PreviewID:               "aga-ws-preview-test",
		GenerationID:            "aga-ws-generation-test",
		DraftID:                 "aga-ws-draft-test",
		DraftRevision:           1,
		DraftContentDigest:      "sha256:" + strings.Repeat("a", 64),
		ClassificationRunDigest: "sha256:" + strings.Repeat("b", 64),
		FilterDigest:            "sha256:" + strings.Repeat("c", 64),
		Action:                  string(BatchExclude),
		ReasonCode:              "MANAGER_SCOPE_DECISION",
		PreviewDigest:           "sha256:" + strings.Repeat("d", 64),
	}
	projection := previewProjection(record, nil, SimulationSetupProjection{}, filter)
	if projection.Filter != filter {
		t.Fatalf("preview filter must remain available for the consume command: got %#v want %#v", projection.Filter, filter)
	}
}
