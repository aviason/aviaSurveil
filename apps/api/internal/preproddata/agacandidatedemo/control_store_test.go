package agacandidatedemo_test

import (
	"strings"
	"testing"
	"time"

	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/preproddata/agacandidatedemo"
)

func TestOverlayControlStoreAppendsIntentAndConsumesAuthorizationOnce(t *testing.T) {
	store, err := agacandidatedemo.NewControlStore(t.TempDir())
	if err != nil {
		t.Fatalf("new control store: %v", err)
	}
	intent, err := agacandidatedemo.BuildIntent(agacandidatedemo.IntentInput{RunID: "aga-demo-run-20260801", BaseRunID: "base-run-20260731", BaseIntentDigest: digest("a"), BaseResultDigest: digest("b"), BaseTargetDigest: digest("c"), CodeDigest: digest("d"), ContractDigest: digest("e"), ExpectedPackage: agacandidatedemo.ExactAcceptedPackage(), ExpectedRelationshipDigests: map[string]string{"forms": digest("f")}, Target: validOverlayTarget(), CreatedAt: time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)})
	if err != nil {
		t.Fatalf("build intent: %v", err)
	}
	if err := store.AppendIntent(intent); err != nil {
		t.Fatalf("append intent: %v", err)
	}
	if err := store.AppendIntent(intent); err != nil {
		t.Fatalf("idempotent intent append: %v", err)
	}
	authorization := agacandidatedemo.OperationAuthorization{SchemaVersion: "preprod-aga-candidate-demo-operation-authorization/v1", Token: "one-time-private-token", Operation: agacandidatedemo.LoadOverlayOperation, Issuer: "local-operator", Nonce: "aga-demo-nonce-001", RunID: intent.RunID, IntentDigest: intent.IntentDigest, TargetFingerprintDigest: intent.TargetFingerprintDigest, ExpiresAt: time.Date(2026, 8, 1, 12, 15, 0, 0, time.UTC)}
	if err := store.ConsumeAuthorization(authorization, agacandidatedemo.LoadOverlayOperation, time.Date(2026, 8, 1, 12, 1, 0, 0, time.UTC)); err != nil {
		t.Fatalf("consume authorization: %v", err)
	}
	if err := store.ConsumeAuthorization(authorization, agacandidatedemo.LoadOverlayOperation, time.Date(2026, 8, 1, 12, 1, 0, 0, time.UTC)); err == nil {
		t.Fatal("replayed authorization was accepted")
	}
	records, err := store.AuthorizationRecords()
	if err != nil {
		t.Fatalf("read authorization records: %v", err)
	}
	if strings.Contains(string(records), authorization.Token) || !strings.Contains(string(records), authorization.Hash()) {
		t.Fatal("authorization record did not retain only the token hash")
	}
	result, err := agacandidatedemo.BuildResult(agacandidatedemo.ResultInput{RunID: intent.RunID, IntentDigest: intent.IntentDigest, SealDigest: digest("a"), CompletedAt: time.Date(2026, 8, 1, 12, 2, 0, 0, time.UTC)})
	if err != nil {
		t.Fatalf("build result: %v", err)
	}
	if err := store.AppendResult(result); err != nil {
		t.Fatalf("append result: %v", err)
	}
	tombstone, err := agacandidatedemo.BuildCleanupTombstone(agacandidatedemo.CleanupTombstoneInput{RunID: intent.RunID, IntentDigest: intent.IntentDigest, ResultDigest: result.ResultDigest, CleanedAt: time.Date(2026, 8, 1, 13, 0, 0, 0, time.UTC)})
	if err != nil {
		t.Fatalf("build cleanup tombstone: %v", err)
	}
	if err := store.AppendCleanupTombstone(tombstone); err != nil {
		t.Fatalf("append cleanup tombstone: %v", err)
	}
	cleaned, err := store.IsCleaned(intent.RunID, intent.IntentDigest)
	if err != nil || !cleaned {
		t.Fatalf("cleaned run lookup = %t, %v", cleaned, err)
	}
	freshAuthorization := authorization
	freshAuthorization.Token = "another-private-token"
	freshAuthorization.Nonce = "aga-demo-nonce-002"
	if err := store.ConsumeAuthorization(freshAuthorization, agacandidatedemo.LoadOverlayOperation, time.Date(2026, 8, 1, 12, 1, 0, 0, time.UTC)); err == nil {
		t.Fatal("cleaned run accepted a fresh authorization")
	}
}
