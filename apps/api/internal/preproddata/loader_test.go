package preproddata_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/aviason/aviaSurveil/internal/preproddata"
	"github.com/aviason/aviaSurveil/internal/preproddata/profiles"
)

func validTarget(runID string) preproddata.TargetFingerprint {
	return preproddata.TargetFingerprint{
		Environment:              "local-preprod",
		DatabaseName:             "aviasurveil360_local_preprod",
		DatabaseOwner:            "aviasurveil360_preprod_loader",
		PostgresSystemIdentifier: "7421987349021349876",
		PostgresHost:             "preprod-postgres",
		PostgresPort:             5432,
		ComposeProject:           "aviasurveil360-local-preprod",
		KeycloakRealm:            "aviasurveil360-local-preprod",
		KeycloakDatabase:         "keycloak_local_preprod",
		KeycloakServiceClientID:  "aviasurveil360-local-preprod-lifecycle",
		MailpitNamespace:         "aviasurveil360-local-preprod",
		ObjectBucket:             "aviasurveil360-local-preprod",
		ObjectPrefix:             "runs/" + runID + "/",
		LoaderQueueNamespace:     "aviasurveil360-local-preprod",
		ProfileName:              "smoke",
		ProfileVersion:           "1.0.0",
		RunID:                    runID,
	}
}

func privateTempDir(t *testing.T) string {
	t.Helper()
	directory := t.TempDir()
	if err := os.Chmod(directory, 0o700); err != nil {
		t.Fatalf("make temporary control store private: %v", err)
	}
	return directory
}

func TestDeterministicGenerationUsesOnlySeedProfileAndClock(t *testing.T) {
	profile, err := profiles.Lookup("smoke", "1.0.0")
	if err != nil {
		t.Fatalf("lookup profile: %v", err)
	}
	first, err := preproddata.NewGenerator(profile, []byte("task-6-seed"))
	if err != nil {
		t.Fatalf("new generator: %v", err)
	}
	second, err := preproddata.NewGenerator(profile, []byte("task-6-seed"))
	if err != nil {
		t.Fatalf("new second generator: %v", err)
	}
	different, err := preproddata.NewGenerator(profile, []byte("other-seed"))
	if err != nil {
		t.Fatalf("new different generator: %v", err)
	}

	if first.SeedHash() != second.SeedHash() {
		t.Fatalf("same seed hashes differ")
	}
	if first.ID("organization", 7) != second.ID("organization", 7) {
		t.Fatalf("same deterministic IDs differ")
	}
	if first.ID("organization", 7) == different.ID("organization", 7) {
		t.Fatalf("different seed reused deterministic ID")
	}
	if got := first.Instant(90); !got.Equal(
		time.Date(2026, 1, 1, 0, 1, 30, 0, time.UTC),
	) {
		t.Fatalf("deterministic instant = %s", got)
	}
	if got := first.SyntheticEmail("caa-user", 7); got !=
		"caa-user-0007@synthetic.invalid" {
		t.Fatalf("synthetic email = %q", got)
	}
	if text := first.Text("finding", 4); !strings.Contains(text, "SYNTHETIC") {
		t.Fatalf("synthetic text = %q", text)
	}
	if metadata := first.ObjectMetadata("evidence", 3, 4096); metadata.Bytes != nil ||
		metadata.SizeBytes != 4096 ||
		!strings.HasPrefix(metadata.ContentDigest, "sha256:") {
		t.Fatalf("object metadata = %#v", metadata)
	}
	if digest := first.RelationshipDigest([][]string{
		{"organization-2", "auditee"},
		{"organization-1", "caa"},
	}); digest != second.RelationshipDigest([][]string{
		{"organization-1", "caa"},
		{"organization-2", "auditee"},
	}) {
		t.Fatalf("relationship digest is not order-independent")
	}
}

func TestGeneratorRejectsUnversionedMutationOfFrozenProfile(t *testing.T) {
	profile, err := profiles.Lookup("smoke", "1.0.0")
	if err != nil {
		t.Fatalf("lookup profile: %v", err)
	}
	profile.ExpectedCounts["organizations"] = 4
	profile.ExactDistributions["organizations"]["auditee"] = 3

	if _, err := preproddata.NewGenerator(
		profile,
		[]byte("task-6-seed"),
	); err == nil {
		t.Fatalf("generator accepted an unversioned frozen-profile mutation")
	}
	runID := "run-task6-mutated-profile"
	if _, err := preproddata.BuildIntent(preproddata.IntentInput{
		RunID: runID, Profile: profile,
		SeedHash:       "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		CodeDigest:     "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		ContractDigest: "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
		Target:         validTarget(runID),
	}); err == nil {
		t.Fatalf("intent accepted an unversioned frozen-profile mutation")
	}
}

func TestIntentAuthorizationAndControlRecordsAreImmutableAndBound(t *testing.T) {
	profile, err := profiles.Lookup("smoke", "1.0.0")
	if err != nil {
		t.Fatalf("lookup profile: %v", err)
	}
	generator, err := preproddata.NewGenerator(profile, []byte("task-6-seed"))
	if err != nil {
		t.Fatalf("new generator: %v", err)
	}
	runID := "run-task6-001"
	intent, err := preproddata.BuildIntent(preproddata.IntentInput{
		RunID:          runID,
		Profile:        profile,
		SeedHash:       generator.SeedHash(),
		CodeDigest:     "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		ContractDigest: "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		Target:         validTarget(runID),
	})
	if err != nil {
		t.Fatalf("build intent: %v", err)
	}
	if intent.IntentDigest == "" ||
		intent.Target.IntentDigest != intent.IntentDigest ||
		intent.TargetFingerprintDigest == "" {
		t.Fatalf("intent binding = %#v", intent)
	}
	if len(intent.ExactDistributions) != len(intent.ExpectedCounts) {
		t.Fatalf(
			"intent distributions = %d, counts = %d",
			len(intent.ExactDistributions),
			len(intent.ExpectedCounts),
		)
	}
	for family, count := range intent.ExpectedCounts {
		var total int64
		for _, value := range intent.ExactDistributions[family] {
			total += value
		}
		if total != count {
			t.Fatalf("%s distribution = %d, expected %d", family, total, count)
		}
	}

	store, err := preproddata.NewFileControlStore(privateTempDir(t))
	if err != nil {
		t.Fatalf("new control store: %v", err)
	}
	if err := store.AppendIntent(intent); err != nil {
		t.Fatalf("append intent: %v", err)
	}
	if err := store.AppendIntent(intent); err != nil {
		t.Fatalf("exact intent replay: %v", err)
	}
	conflicting := intent
	conflicting.CodeDigest =
		"sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
	if err := store.AppendIntent(conflicting); !errors.Is(
		err,
		preproddata.ErrRunIDConflict,
	) {
		t.Fatalf("conflicting intent error = %v", err)
	}

	authorization := preproddata.OperationAuthorization{
		SchemaVersion:           "preprod-operation-authorization/v1",
		Token:                   "single-use-task6-secret",
		Operation:               preproddata.LoadEmptyTarget,
		Issuer:                  "local-owner",
		ExpiresAt:               time.Now().UTC().Add(5 * time.Minute),
		Nonce:                   "nonce-task6-001",
		RunID:                   runID,
		IntentDigest:            intent.IntentDigest,
		TargetFingerprintDigest: intent.TargetFingerprintDigest,
	}
	authPath := filepath.Join(t.TempDir(), "authorization.json")
	if err := preproddata.WriteAuthorizationFile(authPath, authorization); err != nil {
		t.Fatalf("write authorization file: %v", err)
	}
	info, err := os.Stat(authPath)
	if err != nil {
		t.Fatalf("stat authorization file: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("authorization mode = %o", info.Mode().Perm())
	}
	loaded, err := preproddata.ReadAuthorizationFile(authPath)
	if err != nil {
		t.Fatalf("read authorization file: %v", err)
	}
	if err := loaded.Validate(intent, time.Now().UTC()); err != nil {
		t.Fatalf("validate authorization: %v", err)
	}
	wrongTarget := loaded
	wrongTarget.TargetFingerprintDigest =
		"sha256:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee"
	if err := wrongTarget.Validate(intent, time.Now().UTC()); !errors.Is(
		err,
		preproddata.ErrInvalidAuthorization,
	) {
		t.Fatalf("wrong-target authorization error = %v", err)
	}
	if err := store.ConsumeAuthorization(loaded, time.Now().UTC()); err != nil {
		t.Fatalf("consume authorization: %v", err)
	}
	if err := store.ConsumeAuthorization(loaded, time.Now().UTC()); !errors.Is(
		err,
		preproddata.ErrAuthorizationConsumed,
	) {
		t.Fatalf("second authorization consumption = %v", err)
	}
	records, err := store.AuthorizationRecords()
	if err != nil {
		t.Fatalf("read authorization records: %v", err)
	}
	if strings.Contains(string(records), authorization.Token) {
		t.Fatalf("control store retained plaintext token")
	}

	checkpoint := preproddata.Checkpoint{
		SchemaVersion: "preprod-run-checkpoint/v1",
		RunID:         runID, IntentDigest: intent.IntentDigest, Sequence: 1,
		Name:       "authoritative-command-boundary-ready",
		RecordedAt: time.Now().UTC(),
	}
	if err := store.AppendCheckpoint(checkpoint); err != nil {
		t.Fatalf("append checkpoint: %v", err)
	}
	if err := store.AppendCheckpoint(checkpoint); !errors.Is(
		err,
		preproddata.ErrAppendOnlyConflict,
	) {
		t.Fatalf("duplicate checkpoint error = %v", err)
	}
	mismatchedResult, err := preproddata.BuildResult(preproddata.ResultInput{
		RunID: runID, IntentDigest: intent.IntentDigest,
		ActualCounts: map[string]int64{"organizations": 3},
		RelationshipDigests: map[string]string{
			"organizations": generator.RelationshipDigest(
				[][]string{{"organization-1", "caa"}},
			),
		},
		Checkpoints: []string{checkpoint.Name},
		CompletedAt: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("build result: %v", err)
	}
	if err := store.AppendResult(mismatchedResult); err == nil {
		t.Fatalf("control store accepted a successful result outside its intent")
	}
	result, err := preproddata.BuildResult(preproddata.ResultInput{
		RunID: runID, IntentDigest: intent.IntentDigest,
		ActualCounts:        intent.ExpectedCounts,
		RelationshipDigests: relationshipDigests(intent),
		Checkpoints:         []string{checkpoint.Name},
		CompletedAt:         time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("build complete result: %v", err)
	}
	if err := store.AppendResult(result); err != nil {
		t.Fatalf("append result: %v", err)
	}
	if err := store.AppendResult(result); err != nil {
		t.Fatalf("exact result replay: %v", err)
	}
	cleanup := preproddata.CleanupAttestation{
		SchemaVersion: "preprod-cleanup-attestation/v1",
		RunID:         runID, IntentDigest: intent.IntentDigest,
		ResultDigest:      result.ResultDigest,
		TargetDigest:      intent.TargetFingerprintDigest,
		AuthorizationHash: authorization.Hash(),
		CleanedAt:         time.Now().UTC(),
	}
	if err := store.AppendCleanupAttestation(cleanup); err == nil {
		t.Fatalf("load authorization was accepted for cleanup attestation")
	}
	cleanupAuthorization := authorization
	cleanupAuthorization.Token = "single-use-task6-cleanup-secret"
	cleanupAuthorization.Operation = preproddata.DropRecreateTarget
	cleanupAuthorization.Nonce = "nonce-task6-cleanup-001"
	cleanupAuthorization.ExpiresAt = time.Now().UTC().Add(5 * time.Minute)
	if err := cleanupAuthorization.Validate(intent, time.Now().UTC()); err != nil {
		t.Fatalf("validate cleanup authorization: %v", err)
	}
	if err := store.ConsumeAuthorization(
		cleanupAuthorization,
		time.Now().UTC(),
	); err != nil {
		t.Fatalf("consume cleanup authorization: %v", err)
	}
	cleanup.AuthorizationHash = cleanupAuthorization.Hash()
	cleanup.CleanedAt = time.Now().UTC()
	if err := store.AppendCleanupAttestation(cleanup); err != nil {
		t.Fatalf("append cleanup attestation: %v", err)
	}
}

func TestControlStoreRejectsBroadOrNonPrivateRoots(t *testing.T) {
	if _, err := preproddata.NewFileControlStore("/"); err == nil {
		t.Fatalf("filesystem root was accepted as a control store")
	}
	directory := t.TempDir()
	if err := os.Chmod(directory, 0o755); err != nil {
		t.Fatalf("chmod test control store: %v", err)
	}
	if _, err := preproddata.NewFileControlStore(directory); err == nil {
		t.Fatalf("non-private control store was accepted")
	}
}

type recordingBoundary struct {
	preflight      int
	applied        []preproddata.AuthoritativeCommand
	reconciliation *preproddata.Reconciliation
}

type generatedCommandStream struct {
	remaining int
	index     int
}

func (stream *generatedCommandStream) Next(
	_ context.Context,
) (preproddata.AuthoritativeCommand, error) {
	if stream.remaining == 0 {
		return preproddata.AuthoritativeCommand{}, io.EOF
	}
	stream.index++
	stream.remaining--
	return preproddata.AuthoritativeCommand{
		Family:      "organizations",
		OperationID: fmt.Sprintf("synthetic-operation-%06d", stream.index),
		Payload:     []byte(`{"organizationId":"SYNTHETIC-ORG"}`),
	}, nil
}

type countingBoundary struct {
	applied        int
	reconciliation preproddata.Reconciliation
}

type failingBoundary struct {
	failure error
}

type failAtBoundary struct {
	applied        int
	failAt         int
	reconciliation preproddata.Reconciliation
}

func (boundary *failAtBoundary) Preflight(
	_ context.Context,
	_ preproddata.TargetFingerprint,
	_ preproddata.Operation,
) error {
	return nil
}

func (boundary *failAtBoundary) Apply(
	_ context.Context,
	_ preproddata.AuthoritativeCommand,
) error {
	boundary.applied++
	if boundary.failAt > 0 && boundary.applied == boundary.failAt {
		return errors.New("classified boundary failure")
	}
	return nil
}

func (boundary *failAtBoundary) Reconcile(
	_ context.Context,
) (preproddata.Reconciliation, error) {
	return boundary.reconciliation, nil
}

func (boundary *failingBoundary) Preflight(
	_ context.Context,
	_ preproddata.TargetFingerprint,
	_ preproddata.Operation,
) error {
	return nil
}

func (boundary *failingBoundary) Apply(
	_ context.Context,
	_ preproddata.AuthoritativeCommand,
) error {
	return boundary.failure
}

func (boundary *failingBoundary) Reconcile(
	_ context.Context,
) (preproddata.Reconciliation, error) {
	panic("reconciliation must not run after a command failure")
}

func (boundary *countingBoundary) Preflight(
	_ context.Context,
	_ preproddata.TargetFingerprint,
	_ preproddata.Operation,
) error {
	return nil
}

func (boundary *countingBoundary) Apply(
	_ context.Context,
	_ preproddata.AuthoritativeCommand,
) error {
	boundary.applied++
	return nil
}

func (boundary *countingBoundary) Reconcile(
	_ context.Context,
) (preproddata.Reconciliation, error) {
	return boundary.reconciliation, nil
}

func (boundary *recordingBoundary) Preflight(
	_ context.Context,
	target preproddata.TargetFingerprint,
	operation preproddata.Operation,
) error {
	boundary.preflight++
	if target.Environment != "local-preprod" ||
		operation != preproddata.LoadEmptyTarget {
		return errors.New("unexpected target preflight")
	}
	return nil
}

func (boundary *recordingBoundary) Apply(
	_ context.Context,
	command preproddata.AuthoritativeCommand,
) error {
	boundary.applied = append(boundary.applied, command)
	return nil
}

func (boundary *recordingBoundary) Reconcile(
	_ context.Context,
) (preproddata.Reconciliation, error) {
	if boundary.reconciliation != nil {
		return *boundary.reconciliation, nil
	}
	return preproddata.Reconciliation{
		ActualCounts: map[string]int64{"organizations": 1},
		RelationshipDigests: map[string]string{
			"organizations": "sha256:dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd",
		},
	}, nil
}

func TestRunnerPersistsIntentBeforeAuthoritativeCommands(t *testing.T) {
	frozenNow := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	profile, err := profiles.Lookup("smoke", "1.0.0")
	if err != nil {
		t.Fatalf("lookup profile: %v", err)
	}
	generator, err := preproddata.NewGenerator(profile, []byte("task-6-seed"))
	if err != nil {
		t.Fatalf("new generator: %v", err)
	}
	runID := "run-task6-002"
	intent, err := preproddata.BuildIntent(preproddata.IntentInput{
		RunID: runID, Profile: profile, SeedHash: generator.SeedHash(),
		CodeDigest:     "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		ContractDigest: "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		Target:         validTarget(runID),
	})
	if err != nil {
		t.Fatalf("build intent: %v", err)
	}
	store, err := preproddata.NewFileControlStore(privateTempDir(t))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	authorization := preproddata.OperationAuthorization{
		SchemaVersion: "preprod-operation-authorization/v1",
		Token:         "single-use-task6-runner-secret",
		Operation:     preproddata.LoadEmptyTarget,
		Issuer:        "local-owner",
		ExpiresAt:     frozenNow.Add(5 * time.Minute),
		Nonce:         "nonce-task6-002", RunID: runID,
		IntentDigest:            intent.IntentDigest,
		TargetFingerprintDigest: intent.TargetFingerprintDigest,
	}
	boundary := &recordingBoundary{
		reconciliation: &preproddata.Reconciliation{
			ActualCounts:        intent.ExpectedCounts,
			RelationshipDigests: relationshipDigests(intent),
		},
	}
	result, err := preproddata.Run(context.Background(), preproddata.RunInput{
		Intent: intent, Authorization: authorization, ControlStore: store,
		Boundary: boundary,
		Commands: preproddata.NewSliceCommandStream(preproddata.AuthoritativeCommand{
			Family: "organizations", OperationID: generator.ID("operation", 1),
			Payload: []byte(`{"organizationId":"SYNTHETIC-ORG-0001"}`),
		}),
		Clock: func() time.Time {
			return frozenNow
		},
	})
	if err != nil {
		t.Fatalf("run loader: %v", err)
	}
	if boundary.preflight != 1 || len(boundary.applied) != 1 {
		t.Fatalf("boundary calls = preflight %d apply %d",
			boundary.preflight, len(boundary.applied))
	}
	if result.IntentDigest != intent.IntentDigest ||
		result.ActualCounts["organizations"] != 3 {
		t.Fatalf("result = %#v", result)
	}

	replayBoundary := &recordingBoundary{}
	replayed, err := preproddata.Run(
		context.Background(),
		preproddata.RunInput{
			Intent: intent, Authorization: authorization, ControlStore: store,
			Boundary: replayBoundary,
			Commands: preproddata.NewSliceCommandStream(preproddata.AuthoritativeCommand{
				Family:      "organizations",
				OperationID: generator.ID("operation", 1),
				Payload:     []byte(`{"organizationId":"SYNTHETIC-ORG-0001"}`),
			}),
			Clock: func() time.Time {
				return frozenNow
			},
		},
	)
	if err != nil {
		t.Fatalf("exact completed-run replay: %v", err)
	}
	if replayed.ResultDigest != result.ResultDigest {
		t.Fatalf(
			"replay result digest = %s, expected %s",
			replayed.ResultDigest,
			result.ResultDigest,
		)
	}
	if replayBoundary.preflight != 0 || len(replayBoundary.applied) != 0 {
		t.Fatalf(
			"completed replay touched target: preflight %d apply %d",
			replayBoundary.preflight,
			len(replayBoundary.applied),
		)
	}
}

func relationshipDigests(
	intent preproddata.IntentManifest,
) map[string]string {
	digests := make(map[string]string, len(intent.ExpectedCounts))
	for family := range intent.ExpectedCounts {
		digests[family] =
			"sha256:dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd"
	}
	return digests
}

func TestRunnerRejectsReconciliationCountsOutsideIntent(t *testing.T) {
	frozenNow := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	profile, err := profiles.Lookup("smoke", "1.0.0")
	if err != nil {
		t.Fatalf("lookup profile: %v", err)
	}
	generator, err := preproddata.NewGenerator(profile, []byte("task-6-seed"))
	if err != nil {
		t.Fatalf("new generator: %v", err)
	}
	runID := "run-task6-003"
	intent, err := preproddata.BuildIntent(preproddata.IntentInput{
		RunID: runID, Profile: profile, SeedHash: generator.SeedHash(),
		CodeDigest:     "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		ContractDigest: "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		Target:         validTarget(runID),
	})
	if err != nil {
		t.Fatalf("build intent: %v", err)
	}
	store, err := preproddata.NewFileControlStore(privateTempDir(t))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	authorization := preproddata.OperationAuthorization{
		SchemaVersion: "preprod-operation-authorization/v1",
		Token:         "single-use-task6-mismatch-secret",
		Operation:     preproddata.LoadEmptyTarget,
		Issuer:        "local-owner",
		ExpiresAt:     frozenNow.Add(5 * time.Minute),
		Nonce:         "nonce-task6-003", RunID: runID,
		IntentDigest:            intent.IntentDigest,
		TargetFingerprintDigest: intent.TargetFingerprintDigest,
	}
	boundary := &recordingBoundary{
		reconciliation: &preproddata.Reconciliation{
			ActualCounts: map[string]int64{"organizations": 1},
			RelationshipDigests: map[string]string{
				"organizations": "sha256:dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd",
			},
		},
	}
	result, err := preproddata.Run(
		context.Background(),
		preproddata.RunInput{
			Intent: intent, Authorization: authorization, ControlStore: store,
			Boundary: boundary,
			Commands: preproddata.NewSliceCommandStream(preproddata.AuthoritativeCommand{
				Family:      "organizations",
				OperationID: generator.ID("operation", 1),
				Payload:     []byte(`{"organizationId":"SYNTHETIC-ORG-0001"}`),
			}),
			Clock: func() time.Time {
				return frozenNow
			},
		},
	)
	if err == nil {
		t.Fatalf("mismatched reconciliation was accepted: %#v", result)
	}
	if result.Outcome != "FAILED" ||
		result.IntentDigest != intent.IntentDigest ||
		len(result.Failures) != 1 {
		t.Fatalf("mismatched reconciliation result = %#v, err = %v", result, err)
	}
}

func TestRunnerBoundsCheckpointMemoryForStreamingProfiles(t *testing.T) {
	frozenNow := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	profile, err := profiles.Lookup("stress", "1.0.0")
	if err != nil {
		t.Fatalf("lookup profile: %v", err)
	}
	generator, err := preproddata.NewGenerator(profile, []byte("task-6-seed"))
	if err != nil {
		t.Fatalf("new generator: %v", err)
	}
	runID := "run-task6-bounded-stream"
	target := validTarget(runID)
	target.ProfileName = "stress"
	intent, err := preproddata.BuildIntent(preproddata.IntentInput{
		RunID: runID, Profile: profile, SeedHash: generator.SeedHash(),
		CodeDigest:     "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		ContractDigest: "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		Target:         target,
	})
	if err != nil {
		t.Fatalf("build intent: %v", err)
	}
	store, err := preproddata.NewFileControlStore(privateTempDir(t))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	boundary := &countingBoundary{
		reconciliation: preproddata.Reconciliation{
			ActualCounts:        intent.ExpectedCounts,
			RelationshipDigests: relationshipDigests(intent),
		},
	}
	result, err := preproddata.Run(
		context.Background(),
		preproddata.RunInput{
			Intent: intent,
			Authorization: preproddata.OperationAuthorization{
				SchemaVersion: "preprod-operation-authorization/v1",
				Token:         "single-use-task6-stream-secret",
				Operation:     preproddata.LoadEmptyTarget,
				Issuer:        "local-owner",
				ExpiresAt:     frozenNow.Add(5 * time.Minute),
				Nonce:         "nonce-task6-stream", RunID: runID,
				IntentDigest:            intent.IntentDigest,
				TargetFingerprintDigest: intent.TargetFingerprintDigest,
			},
			ControlStore: store,
			Boundary:     boundary,
			Commands:     &generatedCommandStream{remaining: 2049},
			Clock: func() time.Time {
				return frozenNow
			},
		},
	)
	if err != nil {
		t.Fatalf("run bounded stream: %v", err)
	}
	if boundary.applied != 2049 {
		t.Fatalf("applied commands = %d", boundary.applied)
	}
	if len(result.Checkpoints) > 1025 {
		t.Fatalf("unbounded checkpoint manifest = %d", len(result.Checkpoints))
	}
}

func TestRunnerRedactsBoundaryFailureFromResultAndError(t *testing.T) {
	frozenNow := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	profile, err := profiles.Lookup("smoke", "1.0.0")
	if err != nil {
		t.Fatalf("lookup profile: %v", err)
	}
	generator, err := preproddata.NewGenerator(profile, []byte("task-6-seed"))
	if err != nil {
		t.Fatalf("new generator: %v", err)
	}
	runID := "run-task6-redacted-failure"
	intent, err := preproddata.BuildIntent(preproddata.IntentInput{
		RunID: runID, Profile: profile, SeedHash: generator.SeedHash(),
		CodeDigest:     "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		ContractDigest: "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		Target:         validTarget(runID),
	})
	if err != nil {
		t.Fatalf("build intent: %v", err)
	}
	store, err := preproddata.NewFileControlStore(privateTempDir(t))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	const sentinel = "SECRET_SENTINEL_MUST_NOT_LEAVE_BOUNDARY"
	result, err := preproddata.Run(
		context.Background(),
		preproddata.RunInput{
			Intent: intent,
			Authorization: preproddata.OperationAuthorization{
				SchemaVersion: "preprod-operation-authorization/v1",
				Token:         "single-use-task6-redaction-secret",
				Operation:     preproddata.LoadEmptyTarget,
				Issuer:        "local-owner",
				ExpiresAt:     frozenNow.Add(5 * time.Minute),
				Nonce:         "nonce-task6-redaction", RunID: runID,
				IntentDigest:            intent.IntentDigest,
				TargetFingerprintDigest: intent.TargetFingerprintDigest,
			},
			ControlStore: store,
			Boundary: &failingBoundary{
				failure: errors.New(sentinel),
			},
			Commands: preproddata.NewSliceCommandStream(
				preproddata.AuthoritativeCommand{
					Family:      "organizations",
					OperationID: "synthetic-operation-redaction",
					Payload:     []byte(`{"organizationId":"SYNTHETIC-ORG"}`),
				},
			),
			Clock: func() time.Time {
				return frozenNow
			},
		},
	)
	if err == nil || result.Outcome != "FAILED" {
		t.Fatalf("boundary failure result = %#v, err = %v", result, err)
	}
	if strings.Contains(err.Error(), sentinel) ||
		strings.Contains(strings.Join(result.Failures, "\n"), sentinel) {
		t.Fatalf("boundary failure leaked into result or returned error")
	}
}

func TestRunnerResumesAfterLastDurableCheckpoint(t *testing.T) {
	frozenNow := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	profile, err := profiles.Lookup("smoke", "1.0.0")
	if err != nil {
		t.Fatalf("lookup profile: %v", err)
	}
	generator, err := preproddata.NewGenerator(profile, []byte("task-6-seed"))
	if err != nil {
		t.Fatalf("new generator: %v", err)
	}
	runID := "run-task6-resume"
	intent, err := preproddata.BuildIntent(preproddata.IntentInput{
		RunID: runID, Profile: profile, SeedHash: generator.SeedHash(),
		CodeDigest:     "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		ContractDigest: "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		Target:         validTarget(runID),
	})
	if err != nil {
		t.Fatalf("build intent: %v", err)
	}
	store, err := preproddata.NewFileControlStore(privateTempDir(t))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	commands := []preproddata.AuthoritativeCommand{
		{
			Family:      "organizations",
			OperationID: "synthetic-operation-resume-000001",
			Payload:     []byte(`{"organizationId":"SYNTHETIC-ORG-0001"}`),
		},
		{
			Family:      "organizations",
			OperationID: "synthetic-operation-resume-000002",
			Payload:     []byte(`{"organizationId":"SYNTHETIC-ORG-0002"}`),
		},
		{
			Family:      "organizations",
			OperationID: "synthetic-operation-resume-000003",
			Payload:     []byte(`{"organizationId":"SYNTHETIC-ORG-0003"}`),
		},
		{
			Family:      "auditEvents",
			OperationID: "synthetic-operation-resume-000004",
			Payload:     []byte(`{"auditEventId":"SYNTHETIC-AUDIT-0001"}`),
		},
	}
	loadAuthorization := preproddata.OperationAuthorization{
		SchemaVersion: "preprod-operation-authorization/v1",
		Token:         "single-use-task6-resume-load",
		Operation:     preproddata.LoadEmptyTarget,
		Issuer:        "local-owner",
		ExpiresAt:     frozenNow.Add(5 * time.Minute),
		Nonce:         "nonce-task6-resume-load", RunID: runID,
		IntentDigest:            intent.IntentDigest,
		TargetFingerprintDigest: intent.TargetFingerprintDigest,
	}
	firstBoundary := &failAtBoundary{failAt: 3}
	if result, err := preproddata.Run(
		context.Background(),
		preproddata.RunInput{
			Intent: intent, Authorization: loadAuthorization,
			ControlStore: store, Boundary: firstBoundary,
			Commands: preproddata.NewSliceCommandStream(commands...),
			Clock: func() time.Time {
				return frozenNow
			},
		},
	); err == nil || result.Outcome != "FAILED" {
		t.Fatalf("interrupted run result = %#v, err = %v", result, err)
	}

	resumeAuthorization := loadAuthorization
	resumeAuthorization.Token = "single-use-task6-resume-operation"
	resumeAuthorization.Operation = preproddata.ResumeRun
	resumeAuthorization.Nonce = "nonce-task6-resume-operation"
	secondBoundary := &failAtBoundary{
		reconciliation: preproddata.Reconciliation{
			ActualCounts:        intent.ExpectedCounts,
			RelationshipDigests: relationshipDigests(intent),
		},
	}
	result, err := preproddata.Run(
		context.Background(),
		preproddata.RunInput{
			Intent: intent, Authorization: resumeAuthorization,
			ControlStore: store, Boundary: secondBoundary,
			Commands: preproddata.NewSliceCommandStream(commands...),
			Clock: func() time.Time {
				return frozenNow
			},
		},
	)
	if err != nil {
		t.Fatalf("resume run: %v", err)
	}
	if result.Outcome != "SUCCEEDED" || secondBoundary.applied != 2 {
		t.Fatalf(
			"resume result = %#v, replayed commands = %d",
			result,
			secondBoundary.applied,
		)
	}
}
