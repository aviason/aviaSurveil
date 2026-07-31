package datafeed

import "testing"

func TestReconcileFeedManifestsRequiresExactEventIdentityDigestAndAcknowledgementFrontier(t *testing.T) {
	producer := ReconciliationManifest{
		RunID: "10000000-0000-4000-8000-000000000111", ContractVersion: contractVersion,
		Events: []ReconciliationEntry{
			{EventID: "10000000-0000-4000-8000-000000000112", CanonicalEventSHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", DeliveryOutcome: "ACKNOWLEDGED", AcknowledgementReceiptDigest: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"},
			{EventID: "10000000-0000-4000-8000-000000000113", CanonicalEventSHA256: "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc", DeliveryOutcome: "PENDING"},
		},
	}
	core := producer
	if result, err := ReconcileFeedManifests(producer, core); err != nil || result.ExpectedEventCount != 2 || result.AcknowledgedEventCount != 1 {
		t.Fatalf("exact reconciliation=%+v err=%v", result, err)
	}

	for name, mutate := range map[string]func(*ReconciliationManifest){
		"missing event": func(manifest *ReconciliationManifest) { manifest.Events = manifest.Events[:1] },
		"extra event": func(manifest *ReconciliationManifest) {
			manifest.Events = append(manifest.Events, ReconciliationEntry{EventID: "10000000-0000-4000-8000-000000000114", CanonicalEventSHA256: "dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd", DeliveryOutcome: "PENDING"})
		},
		"digest mutation": func(manifest *ReconciliationManifest) {
			manifest.Events[0].CanonicalEventSHA256 = "eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee"
		},
		"ack frontier mutation": func(manifest *ReconciliationManifest) {
			manifest.Events[0].AcknowledgementReceiptDigest = "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"
		},
	} {
		t.Run(name, func(t *testing.T) {
			candidate := core
			candidate.Events = append([]ReconciliationEntry(nil), core.Events...)
			mutate(&candidate)
			if _, err := ReconcileFeedManifests(producer, candidate); err == nil {
				t.Fatal("mismatched reconciliation manifest was accepted")
			}
		})
	}
}
