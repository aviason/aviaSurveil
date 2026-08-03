package agacandidatedemo_test

import (
	"testing"
	"time"

	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/preproddata/agacandidatedemo"
)

func TestBaseEvidenceMustBeSuccessfulDisposableAndExactlyBound(t *testing.T) {
	intent, err := agacandidatedemo.BuildIntent(agacandidatedemo.IntentInput{RunID: "aga-demo-run-20260801", BaseRunID: "base-run-20260731", BaseIntentDigest: digest("a"), BaseResultDigest: digest("b"), BaseTargetDigest: digest("c"), CodeDigest: digest("d"), ContractDigest: digest("e"), ExpectedPackage: agacandidatedemo.ExactAcceptedPackage(), ExpectedRelationshipDigests: map[string]string{"forms": digest("f")}, Target: validOverlayTarget(), CreatedAt: time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)})
	if err != nil {
		t.Fatalf("build intent: %v", err)
	}
	evidence := agacandidatedemo.BaseRunEvidence{RunID: intent.BaseRunID, IntentDigest: intent.BaseIntentDigest, ResultDigest: intent.BaseResultDigest, TargetFingerprintDigest: intent.BaseTargetDigest, Outcome: "SUCCEEDED", Disposable: true}
	if err := agacandidatedemo.VerifyBaseEvidence(intent, evidence); err != nil {
		t.Fatalf("verify base evidence: %v", err)
	}
	evidence.Outcome = "FAILED"
	if err := agacandidatedemo.VerifyBaseEvidence(intent, evidence); err == nil {
		t.Fatal("failed base evidence was accepted")
	}
}
