package agacandidatedemo_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/preproddata/agacandidatedemo"
)

func TestLoaderRejectsDigestMismatchBeforeDatabasePreflight(t *testing.T) {
	pkg := syntheticAcceptedPackage(t)
	digests, err := agacandidatedemo.RelationshipDigests(pkg)
	if err != nil {
		t.Fatal(err)
	}
	digests["forms"] = digestOf("wrong")
	intent := newOverlayIntent(t, digests)
	if err != nil {
		t.Fatal(err)
	}
	store := &fakeProjectionStore{}
	_, err = agacandidatedemo.LoadOverlay(context.Background(), agacandidatedemo.OverlayLoadInput{
		Intent: intent, Package: pkg, BaseEvidence: validBaseEvidence(intent), Store: store,
	})
	if err == nil || store.preflightCalls != 0 {
		t.Fatalf("expected preflight-free digest rejection, got %v and %d calls", err, store.preflightCalls)
	}
}

func TestLoaderUsesCommittedSealAsOnlyResultReceipt(t *testing.T) {
	pkg := syntheticAcceptedPackage(t)
	digests, err := agacandidatedemo.RelationshipDigests(pkg)
	if err != nil {
		t.Fatal(err)
	}
	intent := newOverlayIntent(t, digests)
	store := &fakeProjectionStore{receipt: agacandidatedemo.SealReceipt{PackageDigest: pkg.Identity.JSONSHA256, IntentDigest: intent.IntentDigest, TargetDigest: intent.TargetFingerprintDigest, ReconciliationDigest: digests["projection"], SealDigest: digestOf("seal"), SealedAt: time.Date(2026, 8, 2, 12, 0, 0, 0, time.UTC)}}
	result, err := agacandidatedemo.LoadOverlay(context.Background(), agacandidatedemo.OverlayLoadInput{Intent: intent, Package: pkg, BaseEvidence: validBaseEvidence(intent), Store: store})
	if err != nil {
		t.Fatal(err)
	}
	if result.SealDigest != store.receipt.SealDigest || store.materializeCalls != 1 || store.preflightCalls != 1 {
		t.Fatalf("result must be derived from the sole committed seal: %#v", result)
	}
}

func TestLoaderRejectsMalformedOrMismatchedSeal(t *testing.T) {
	pkg := syntheticAcceptedPackage(t)
	digests, err := agacandidatedemo.RelationshipDigests(pkg)
	if err != nil {
		t.Fatal(err)
	}
	intent := newOverlayIntent(t, digests)
	store := &fakeProjectionStore{receipt: agacandidatedemo.SealReceipt{PackageDigest: pkg.Identity.JSONSHA256, IntentDigest: intent.IntentDigest, TargetDigest: intent.TargetFingerprintDigest, ReconciliationDigest: digestOf("wrong"), SealDigest: digestOf("seal"), SealedAt: time.Now().UTC()}}
	_, err = agacandidatedemo.LoadOverlay(context.Background(), agacandidatedemo.OverlayLoadInput{Intent: intent, Package: pkg, BaseEvidence: validBaseEvidence(intent), Store: store})
	if err == nil {
		t.Fatal("expected seal reconciliation rejection")
	}
}

type fakeProjectionStore struct {
	preflightCalls, materializeCalls int
	receipt                          agacandidatedemo.SealReceipt
	err                              error
}

func (store *fakeProjectionStore) Preflight(context.Context, agacandidatedemo.IntentManifest) error {
	store.preflightCalls++
	return store.err
}
func (store *fakeProjectionStore) Materialize(_ context.Context, _ agacandidatedemo.IntentManifest, _ agacandidatedemo.AcceptedPackage, _ map[string]string) (agacandidatedemo.SealReceipt, error) {
	store.materializeCalls++
	if store.err != nil {
		return agacandidatedemo.SealReceipt{}, store.err
	}
	return store.receipt, nil
}
func (store *fakeProjectionStore) VerifySeal(context.Context, agacandidatedemo.IntentManifest) (agacandidatedemo.SealReceipt, error) {
	return store.receipt, store.err
}

func TestLoaderPropagatesStoreFailure(t *testing.T) {
	pkg := syntheticAcceptedPackage(t)
	digests, _ := agacandidatedemo.RelationshipDigests(pkg)
	intent := newOverlayIntent(t, digests)
	_, err := agacandidatedemo.LoadOverlay(context.Background(), agacandidatedemo.OverlayLoadInput{Intent: intent, Package: pkg, BaseEvidence: validBaseEvidence(intent), Store: &fakeProjectionStore{err: errors.New("failed")}})
	if err == nil {
		t.Fatal("expected store failure")
	}
}

func TestRelationshipDigestsBindFormOrdering(t *testing.T) {
	pkg := syntheticAcceptedPackage(t)
	pkg.Forms = append(pkg.Forms, agacandidatedemo.FormCandidate{FormCode: "FSS-AGA-FORM-002", FormSHA256: digestOf("form-two"), ArchiveSHA256: pkg.Forms[0].ArchiveSHA256, DocumentTitle: "synthetic two", FormKind: "synthetic", PageCount: 1})
	first, err := agacandidatedemo.RelationshipDigests(pkg)
	if err != nil {
		t.Fatal(err)
	}
	pkg.Forms[0], pkg.Forms[1] = pkg.Forms[1], pkg.Forms[0]
	second, err := agacandidatedemo.RelationshipDigests(pkg)
	if err != nil {
		t.Fatal(err)
	}
	if first["forms"] == second["forms"] || first["projection"] == second["projection"] {
		t.Fatal("form order must be reconciliation-bound")
	}
}

func syntheticAcceptedPackage(t *testing.T) agacandidatedemo.AcceptedPackage {
	t.Helper()
	expected := agacandidatedemo.ExactAcceptedPackage()
	return agacandidatedemo.AcceptedPackage{
		Identity: agacandidatedemo.PackageIdentity{ZipSHA256: expected.ZipSHA256, ZipBytes: expected.ZipBytes, JSONSHA256: expected.JSONSHA256, JSONBytes: expected.JSONBytes, ManifestSHA256: expected.ManifestSHA256, PackageVersion: expected.PackageVersion, PackageStatus: expected.PackageStatus},
		Forms:    []agacandidatedemo.FormCandidate{{FormCode: "FSS-AGA-FORM-001", FormSHA256: digestOf("form"), ArchiveSHA256: expected.SourceArchiveSHA256, DocumentTitle: "synthetic", FormKind: "synthetic", PageCount: 1}},
	}
}

func newOverlayIntent(t *testing.T, digests map[string]string) agacandidatedemo.IntentManifest {
	t.Helper()
	intent, err := agacandidatedemo.BuildIntent(agacandidatedemo.IntentInput{
		RunID: "aga-demo-run-0001", BaseRunID: "base-run-0001", BaseIntentDigest: digestOf("base-intent"), BaseResultDigest: digestOf("base-result"), BaseTargetDigest: digestOf("base-target"), CodeDigest: digestOf("code"), ContractDigest: digestOf("contract"), ExpectedPackage: agacandidatedemo.ExactAcceptedPackage(), ExpectedRelationshipDigests: digests, Target: validTarget(), CreatedAt: time.Date(2026, 8, 2, 11, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatal(err)
	}
	return intent
}

func validTarget() agacandidatedemo.TargetFingerprint {
	return agacandidatedemo.TargetFingerprint{Environment: "local-preprod", DatabaseName: "aviasurveil360_local_preprod", DatabaseOwner: "aviasurveil360_preprod_loader", PostgresSystemIdentifier: "7421987349021349876", PostgresHost: "preprod-postgres", PostgresPort: 5432, ComposeProject: "aviasurveil360-local-preprod", OverlaySchema: "preprod_aga_demo"}
}

func validBaseEvidence(intent agacandidatedemo.IntentManifest) agacandidatedemo.BaseRunEvidence {
	return agacandidatedemo.BaseRunEvidence{RunID: intent.BaseRunID, IntentDigest: intent.BaseIntentDigest, ResultDigest: intent.BaseResultDigest, TargetFingerprintDigest: intent.BaseTargetDigest, Outcome: "SUCCEEDED", Disposable: true}
}

func digestOf(value string) string {
	return "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
}
