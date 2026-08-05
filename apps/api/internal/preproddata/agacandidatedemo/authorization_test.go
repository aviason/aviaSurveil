package agacandidatedemo_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/preproddata/agacandidatedemo"
)

func TestOperationAuthorizationIsShortLivedSingleOperationAndBoundToIntent(t *testing.T) {
	intent, err := agacandidatedemo.BuildIntent(agacandidatedemo.IntentInput{
		RunID: "aga-demo-run-20260801", BaseRunID: "base-run-20260731", BaseIntentDigest: digest("a"), BaseResultDigest: digest("b"), BaseTargetDigest: digest("c"), CodeDigest: digest("d"), ContractDigest: digest("e"), ExpectedPackage: agacandidatedemo.ExactAcceptedPackage(), ExpectedRelationshipDigests: map[string]string{"forms": digest("f")}, Target: validOverlayTarget(), CreatedAt: time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("build intent: %v", err)
	}
	now := time.Date(2026, 8, 1, 12, 1, 0, 0, time.UTC)
	authorization := agacandidatedemo.OperationAuthorization{SchemaVersion: "preprod-aga-candidate-demo-operation-authorization/v1", Token: "one-time-private-token", Operation: agacandidatedemo.LoadOverlayOperation, Issuer: "local-operator", Nonce: "aga-demo-nonce-001", RunID: intent.RunID, IntentDigest: intent.IntentDigest, TargetFingerprintDigest: intent.TargetFingerprintDigest, InputDigest: intent.PackageZipDigest, CodeDigest: intent.CodeDigest, ContractDigest: intent.ContractDigest, ExpiresAt: now.Add(15 * time.Minute)}
	if err := authorization.Validate(intent, agacandidatedemo.LoadOverlayOperation, now); err != nil {
		t.Fatalf("validate authorization: %v", err)
	}
	if authorization.Hash() == authorization.Token {
		t.Fatal("authorization hash retained its token value")
	}
	if err := authorization.Validate(intent, agacandidatedemo.CleanupOverlayOperation, now); err == nil {
		t.Fatal("load authorization was accepted for cleanup")
	}
	authorization.ExpiresAt = now.Add(15*time.Minute + time.Second)
	if err := authorization.Validate(intent, agacandidatedemo.LoadOverlayOperation, now); err == nil {
		t.Fatal("overbroad authorization expiry was accepted")
	}
}

func TestAuthorizationFileRequiresPrivateAbsoluteRegularFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "authorization.json")
	authorization := agacandidatedemo.OperationAuthorization{SchemaVersion: "preprod-aga-candidate-demo-operation-authorization/v1", Token: "one-time-private-token", Issuer: "local-operator", Nonce: "aga-demo-nonce-001"}
	if err := agacandidatedemo.WriteAuthorizationFile(path, authorization); err != nil {
		t.Fatalf("write authorization: %v", err)
	}
	if _, err := agacandidatedemo.ReadAuthorizationFile(path); err != nil {
		t.Fatalf("read authorization: %v", err)
	}
	if err := os.Chmod(path, 0o644); err != nil {
		t.Fatalf("relax authorization mode: %v", err)
	}
	if _, err := agacandidatedemo.ReadAuthorizationFile(path); err == nil {
		t.Fatal("non-private authorization file was accepted")
	}
}
