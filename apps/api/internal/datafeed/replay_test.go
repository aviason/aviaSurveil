package datafeed

import (
	"testing"
	"time"
)

func TestValidateReplayRequestRequiresApprovalAndOneBoundedSelector(t *testing.T) {
	now := time.Date(2026, 7, 30, 10, 0, 0, 0, time.UTC)
	valid := ReplayRequest{
		RunID:                 "10000000-0000-4000-8000-000000000011",
		ApprovalID:            "10000000-0000-4000-8000-000000000012",
		TenantID:              "tenant-task6",
		OwningOrganizationID:  "organization-task6",
		SourceSystem:          sourceSystem,
		ContractVersion:       contractVersion,
		EventIDs:              []string{"10000000-0000-4000-8000-000000000013"},
		AllowedTerminalStates: []string{"QUARANTINED"},
		RequestedAt:           now,
	}
	if err := ValidateReplayRequest(valid); err != nil {
		t.Fatalf("valid replay request rejected: %v", err)
	}

	for name, mutate := range map[string]func(*ReplayRequest){
		"missing approval": func(request *ReplayRequest) { request.ApprovalID = "" },
		"no selector":      func(request *ReplayRequest) { request.EventIDs = nil },
		"two selectors": func(request *ReplayRequest) {
			request.WindowStart = now.Add(-time.Hour)
			request.WindowEnd = now
		},
		"unbounded window": func(request *ReplayRequest) {
			request.EventIDs = nil
			request.WindowStart = now.Add(-31 * 24 * time.Hour)
			request.WindowEnd = now
		},
		"acknowledged state": func(request *ReplayRequest) {
			request.AllowedTerminalStates = []string{"ACKNOWLEDGED"}
		},
	} {
		t.Run(name, func(t *testing.T) {
			request := valid
			mutate(&request)
			if err := ValidateReplayRequest(request); err == nil {
				t.Fatal("unsafe replay request was accepted")
			}
		})
	}
}

func TestReplayRequestDigestIsStableAcrossEventIDOrderButBindsEveryScopeField(t *testing.T) {
	requestedAt := time.Date(2026, 7, 30, 10, 0, 0, 0, time.UTC)
	first := ReplayRequest{
		RunID:                 "10000000-0000-4000-8000-000000000021",
		ApprovalID:            "10000000-0000-4000-8000-000000000022",
		TenantID:              "tenant-task6",
		OwningOrganizationID:  "organization-task6",
		SourceSystem:          sourceSystem,
		ContractVersion:       contractVersion,
		EventIDs:              []string{"10000000-0000-4000-8000-000000000023", "10000000-0000-4000-8000-000000000024"},
		AllowedTerminalStates: []string{"PENDING", "QUARANTINED"},
		RequestedAt:           requestedAt,
	}
	second := first
	second.EventIDs = []string{first.EventIDs[1], first.EventIDs[0]}
	second.AllowedTerminalStates = []string{"QUARANTINED", "PENDING"}

	firstDigest, err := ReplayRequestDigest(first)
	if err != nil {
		t.Fatalf("first digest: %v", err)
	}
	secondDigest, err := ReplayRequestDigest(second)
	if err != nil {
		t.Fatalf("second digest: %v", err)
	}
	if firstDigest != secondDigest {
		t.Fatalf("reordered equivalent request changed digest: %s != %s", firstDigest, secondDigest)
	}

	changed := first
	changed.OwningOrganizationID = "other-organization"
	changedDigest, err := ReplayRequestDigest(changed)
	if err != nil {
		t.Fatalf("changed digest: %v", err)
	}
	if changedDigest == firstDigest {
		t.Fatal("scope change did not change replay request digest")
	}
}

func TestValidateBackfillRequestBindsSourceConsistentCutWithoutRewritingEventTime(t *testing.T) {
	cutAt := time.Date(2026, 7, 29, 0, 0, 0, 0, time.UTC)
	request := BackfillRequest{
		RunID:                "10000000-0000-4000-8000-000000000031",
		ApprovalID:           "10000000-0000-4000-8000-000000000032",
		TenantID:             "tenant-task6",
		OwningOrganizationID: "organization-task6",
		SourceSystem:         sourceSystem,
		ContractVersion:      contractVersion,
		SourceCutID:          "2026-07-29-source-consistent-cut",
		SourceManifestDigest: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		CutAt:                cutAt,
		RequestedAt:          cutAt.Add(time.Hour),
		EventIDs:             []string{"10000000-0000-4000-8000-000000000033"},
	}
	if err := ValidateBackfillRequest(request); err != nil {
		t.Fatalf("valid source-consistent backfill rejected: %v", err)
	}
	for name, mutate := range map[string]func(*BackfillRequest){
		"missing approval": func(request *BackfillRequest) { request.ApprovalID = "" },
		"missing cut":      func(request *BackfillRequest) { request.SourceCutID = "" },
		"missing events":   func(request *BackfillRequest) { request.EventIDs = nil },
		"invalid manifest": func(request *BackfillRequest) { request.SourceManifestDigest = "sha256:opaque" },
		"cut after request": func(request *BackfillRequest) {
			request.CutAt = request.RequestedAt.Add(time.Second)
		},
	} {
		t.Run(name, func(t *testing.T) {
			candidate := request
			mutate(&candidate)
			if err := ValidateBackfillRequest(candidate); err == nil {
				t.Fatal("unsafe backfill request was accepted")
			}
		})
	}
}
