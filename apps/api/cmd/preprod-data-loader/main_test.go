package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/preproddata"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/preproddata/profiles"
)

func TestRunConnectedInvokesTheAuthorizedConnectedRunnerAndPrintsResult(
	t *testing.T,
) {
	configPath := filepath.Join(t.TempDir(), "loader-config.json")
	if err := os.WriteFile(configPath, []byte(`{
  "environment":"local-preprod",
  "profile":"smoke",
  "profileVersion":"1.0.0",
  "seedFile":"/run/secrets/preprod_seed",
  "authorizationFile":"/run/secrets/preprod_loader_authorization",
  "controlStoreDirectory":"/var/lib/aviasurveil360-preprod-control",
  "intentFile":"/var/lib/aviasurveil360-preprod-control/intents/run-task7.json",
  "routeCatalogFile":"/app/catalog/route-audit.json",
  "behaviorLedgerFile":"/app/catalog/behavior-ledger.json"
}`), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	var received runConfiguration
	dependencies := commandDependencies{
		runConnected: func(
			_ context.Context,
			configuration runConfiguration,
		) (preproddata.ResultManifest, error) {
			received = configuration
			return preproddata.ResultManifest{
				RunID:        "run-task7-connected-smoke",
				Outcome:      "SUCCEEDED",
				ResultDigest: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			}, nil
		},
	}
	var output bytes.Buffer
	if err := runWithDependencies(
		context.Background(),
		[]string{"run-connected", configPath},
		&output,
		dependencies,
	); err != nil {
		t.Fatalf("run connected command: %v", err)
	}
	if received.Profile != "smoke" ||
		received.RouteCatalogFile != "/app/catalog/route-audit.json" ||
		received.BehaviorLedgerFile != "/app/catalog/behavior-ledger.json" {
		t.Fatalf("connected runner configuration = %#v", received)
	}
	for _, expected := range []string{
		"run=run-task7-connected-smoke",
		"outcome=SUCCEEDED",
		"result=sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
	} {
		if !strings.Contains(output.String(), expected) {
			t.Fatalf("connected output %q omits %q", output.String(), expected)
		}
	}
}

func TestRunRecordCleanupInvokesRecorderAndPrintsExactEvidence(
	t *testing.T,
) {
	configPath := filepath.Join(t.TempDir(), "loader-config.json")
	if err := os.WriteFile(configPath, []byte(`{
  "environment":"local-preprod",
  "profile":"smoke",
  "profileVersion":"1.0.0",
  "runId":"run-task7-cleanup-command",
  "seedFile":"/run/secrets/preprod_seed",
  "authorizationFile":"/run/secrets/preprod_loader_authorization",
  "controlStoreDirectory":"/var/lib/aviasurveil360-preprod-control",
  "intentFile":"/var/lib/aviasurveil360-preprod-control/intents/run-task7-cleanup-command.json"
}`), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	var received runConfiguration
	dependencies := commandDependencies{
		recordCleanup: func(
			configuration runConfiguration,
		) (preproddata.CleanupAttestation, error) {
			received = configuration
			return preproddata.CleanupAttestation{
				RunID: "run-task7-cleanup-command",
				ResultDigest: "sha256:" +
					strings.Repeat("a", 64),
				AuthorizationHash: "sha256:" +
					strings.Repeat("b", 64),
				AttestationDigest: "sha256:" +
					strings.Repeat("c", 64),
			}, nil
		},
	}
	var output bytes.Buffer
	if err := runWithDependencies(
		context.Background(),
		[]string{"record-cleanup", configPath},
		&output,
		dependencies,
	); err != nil {
		t.Fatalf("record cleanup command: %v", err)
	}
	if received.RunID != "run-task7-cleanup-command" {
		t.Fatalf("cleanup recorder configuration = %#v", received)
	}
	for _, expected := range []string{
		"run=run-task7-cleanup-command",
		"result=sha256:" + strings.Repeat("a", 64),
		"authorization=sha256:" + strings.Repeat("b", 64),
		"attestation=sha256:" + strings.Repeat("c", 64),
	} {
		if !strings.Contains(output.String(), expected) {
			t.Fatalf("cleanup output %q omits %q", output.String(), expected)
		}
	}
}

func TestRecordCleanupRejectsLoadAuthorizationWithoutConsumingIt(
	t *testing.T,
) {
	now := time.Date(2026, 7, 28, 14, 0, 0, 0, time.UTC)
	configuration, _, _, store := cleanupCommandFixture(
		t,
		"run-task7-cleanup-load",
		preproddata.LoadEmptyTarget,
		now,
	)
	if _, err := recordCleanupData(configuration, now); err == nil {
		t.Fatalf("load authorization was accepted for cleanup")
	}
	records, err := store.AuthorizationRecords()
	if err != nil {
		t.Fatalf("read authorization records: %v", err)
	}
	if strings.TrimSpace(string(records)) != "" {
		t.Fatalf("wrong-operation authorization was consumed: %s", records)
	}
}

func TestRecordCleanupConsumesDropAuthorizationAndAppendsAttestation(
	t *testing.T,
) {
	now := time.Date(2026, 7, 28, 14, 5, 0, 0, time.UTC)
	configuration, intent, authorization, store := cleanupCommandFixture(
		t,
		"run-task7-cleanup-drop",
		preproddata.DropRecreateTarget,
		now,
	)
	attestation, err := recordCleanupData(configuration, now)
	if err != nil {
		t.Fatalf("record cleanup: %v", err)
	}
	if attestation.RunID != intent.RunID ||
		attestation.IntentDigest != intent.IntentDigest ||
		attestation.TargetDigest != intent.TargetFingerprintDigest ||
		attestation.AuthorizationHash != authorization.Hash() ||
		attestation.AttestationDigest == "" ||
		!attestation.CleanedAt.Equal(now) {
		t.Fatalf("cleanup attestation = %#v", attestation)
	}
	paths, err := filepath.Glob(filepath.Join(
		store.Root(),
		"cleanup",
		intent.RunID,
		"*.json",
	))
	if err != nil {
		t.Fatalf("glob cleanup attestations: %v", err)
	}
	if len(paths) != 1 {
		t.Fatalf("cleanup attestation files = %v", paths)
	}
	encoded, err := os.ReadFile(paths[0])
	if err != nil {
		t.Fatalf("read cleanup attestation: %v", err)
	}
	var persisted preproddata.CleanupAttestation
	if err := json.Unmarshal(encoded, &persisted); err != nil {
		t.Fatalf("decode cleanup attestation: %v", err)
	}
	if persisted != attestation {
		t.Fatalf("persisted cleanup = %#v, expected %#v", persisted, attestation)
	}
	authorizationRecords, err := store.AuthorizationRecords()
	if err != nil {
		t.Fatalf("read authorization records: %v", err)
	}
	if !strings.Contains(
		string(authorizationRecords),
		authorization.Hash(),
	) || strings.Contains(
		string(authorizationRecords),
		authorization.Token,
	) {
		t.Fatalf(
			"authorization records do not retain hash-only evidence: %s",
			authorizationRecords,
		)
	}
}

func cleanupCommandFixture(
	t *testing.T,
	runID string,
	operation preproddata.Operation,
	now time.Time,
) (
	runConfiguration,
	preproddata.IntentManifest,
	preproddata.OperationAuthorization,
	*preproddata.FileControlStore,
) {
	t.Helper()
	root := t.TempDir()
	profile, err := profiles.Lookup("smoke", "1.0.0")
	if err != nil {
		t.Fatalf("lookup smoke profile: %v", err)
	}
	intent, err := preproddata.BuildIntent(preproddata.IntentInput{
		RunID:   runID,
		Profile: profile,
		SeedHash: "sha256:" +
			strings.Repeat("a", 64),
		CodeDigest: "sha256:" +
			strings.Repeat("b", 64),
		ContractDigest: "sha256:" +
			strings.Repeat("c", 64),
		Target: preproddata.TargetFingerprint{
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
		},
	})
	if err != nil {
		t.Fatalf("build cleanup intent: %v", err)
	}
	controlStoreRoot := filepath.Join(root, "control-store")
	store, err := preproddata.NewFileControlStore(controlStoreRoot)
	if err != nil {
		t.Fatalf("new control store: %v", err)
	}
	if err := store.AppendIntent(intent); err != nil {
		t.Fatalf("append cleanup intent: %v", err)
	}
	intentPath := filepath.Join(root, "intent.json")
	encodedIntent, err := json.Marshal(intent)
	if err != nil {
		t.Fatalf("encode cleanup intent: %v", err)
	}
	if err := os.WriteFile(intentPath, encodedIntent, 0o600); err != nil {
		t.Fatalf("write cleanup intent: %v", err)
	}
	relationshipDigests := make(map[string]string, len(intent.ExpectedCounts))
	for family := range intent.ExpectedCounts {
		relationshipDigests[family] = "sha256:" + strings.Repeat("d", 64)
	}
	result, err := preproddata.BuildResult(preproddata.ResultInput{
		RunID:               runID,
		IntentDigest:        intent.IntentDigest,
		ActualCounts:        intent.ExpectedCounts,
		RelationshipDigests: relationshipDigests,
		CompletedAt:         now.Add(-time.Minute),
	})
	if err != nil {
		t.Fatalf("build successful cleanup result: %v", err)
	}
	if err := store.AppendResult(result); err != nil {
		t.Fatalf("append successful cleanup result: %v", err)
	}
	authorization := preproddata.OperationAuthorization{
		SchemaVersion:           "preprod-operation-authorization/v1",
		Token:                   "task7-cleanup-token-" + runID,
		Operation:               operation,
		Issuer:                  "plan-5-task-7-test",
		ExpiresAt:               now.Add(10 * time.Minute),
		Nonce:                   "cleanup-nonce-" + runID,
		RunID:                   runID,
		IntentDigest:            intent.IntentDigest,
		TargetFingerprintDigest: intent.TargetFingerprintDigest,
	}
	authorizationPath := filepath.Join(root, "authorization.json")
	if err := preproddata.WriteAuthorizationFile(
		authorizationPath,
		authorization,
	); err != nil {
		t.Fatalf("write cleanup authorization: %v", err)
	}
	return runConfiguration{
		Environment:           "local-preprod",
		Profile:               "smoke",
		ProfileVersion:        "1.0.0",
		RunID:                 runID,
		SeedFile:              filepath.Join(root, "unused-seed"),
		AuthorizationFile:     authorizationPath,
		ControlStoreDirectory: controlStoreRoot,
		IntentFile:            intentPath,
	}, intent, authorization, store
}

func TestLoadConnectedInputsBindsIntentSeedAuthorizationAndCanonicalCatalogs(
	t *testing.T,
) {
	now := time.Date(2026, 7, 28, 12, 0, 0, 0, time.UTC)
	root := t.TempDir()
	seedPath := filepath.Join(root, "seed")
	const seed = "task7-connected-seed"
	if err := os.WriteFile(seedPath, []byte(seed), 0o600); err != nil {
		t.Fatalf("write seed: %v", err)
	}
	profile, err := profiles.Lookup("smoke", "1.0.0")
	if err != nil {
		t.Fatalf("lookup profile: %v", err)
	}
	generator, err := preproddata.NewGenerator(profile, []byte(seed))
	if err != nil {
		t.Fatalf("new generator: %v", err)
	}
	const runID = "run-task7-connected-smoke"
	intent, err := preproddata.BuildIntent(preproddata.IntentInput{
		RunID:          runID,
		Profile:        profile,
		SeedHash:       generator.SeedHash(),
		CodeDigest:     "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		ContractDigest: "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
		Target: preproddata.TargetFingerprint{
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
		},
	})
	if err != nil {
		t.Fatalf("build intent: %v", err)
	}
	intentPath := filepath.Join(root, "intent.json")
	encodedIntent, err := json.Marshal(intent)
	if err != nil {
		t.Fatalf("encode intent: %v", err)
	}
	if err := os.WriteFile(intentPath, encodedIntent, 0o600); err != nil {
		t.Fatalf("write intent: %v", err)
	}
	authorizationPath := filepath.Join(root, "authorization.json")
	if err := preproddata.WriteAuthorizationFile(
		authorizationPath,
		preproddata.OperationAuthorization{
			SchemaVersion:           "preprod-operation-authorization/v1",
			Token:                   "task7-one-time-token",
			Operation:               preproddata.LoadEmptyTarget,
			Issuer:                  "plan-5-task-7-test",
			ExpiresAt:               now.Add(10 * time.Minute),
			Nonce:                   "task7-nonce",
			RunID:                   runID,
			IntentDigest:            intent.IntentDigest,
			TargetFingerprintDigest: intent.TargetFingerprintDigest,
		},
	); err != nil {
		t.Fatalf("write authorization: %v", err)
	}
	repositoryRoot := testRepositoryRoot(t)
	inputs, err := loadConnectedInputs(
		runConfiguration{
			Environment:           "local-preprod",
			Profile:               "smoke",
			ProfileVersion:        "1.0.0",
			SeedFile:              seedPath,
			AuthorizationFile:     authorizationPath,
			ControlStoreDirectory: filepath.Join(root, "control-store"),
			IntentFile:            intentPath,
			RouteCatalogFile: filepath.Join(
				repositoryRoot,
				"apps/web/src/parity/legacy-screen-source.json",
			),
			BehaviorLedgerFile: filepath.Join(
				repositoryRoot,
				"tests/parity/behavior-ledger.json",
			),
		},
		now,
	)
	if err != nil {
		t.Fatalf("load connected inputs: %v", err)
	}
	if inputs.Intent.IntentDigest != intent.IntentDigest ||
		inputs.Authorization.Operation != preproddata.LoadEmptyTarget ||
		inputs.Profile.Name != "smoke" ||
		string(inputs.Seed) != seed ||
		len(inputs.Catalog.Routes) != 86 ||
		len(inputs.Catalog.Actions) != 306 ||
		inputs.ControlStore == nil {
		t.Fatalf("connected inputs = %#v", inputs)
	}
}

func testRepositoryRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("resolve test source")
	}
	return filepath.Clean(filepath.Join(
		filepath.Dir(file),
		"..",
		"..",
		"..",
		"..",
	))
}

func TestRunConfigurationCarriesAuthorizationByFileOnly(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "loader-config.json")
	if err := os.WriteFile(configPath, []byte(`{
  "environment": "local-preprod",
  "profile": "smoke",
  "profileVersion": "1.0.0",
  "seedFile": "/run/secrets/preprod_seed",
  "authorizationFile": "/run/secrets/preprod_loader_authorization",
  "controlStoreDirectory": "/var/lib/aviasurveil360-preprod-control",
  "intentFile": "/var/lib/aviasurveil360-preprod-control/intents/run-task6.json"
}`), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	configuration, err := loadRunConfiguration(configPath)
	if err != nil {
		t.Fatalf("load run configuration: %v", err)
	}
	if configuration.AuthorizationFile !=
		"/run/secrets/preprod_loader_authorization" {
		t.Fatalf("authorization file = %q", configuration.AuthorizationFile)
	}
	if configuration.ControlStoreDirectory !=
		"/var/lib/aviasurveil360-preprod-control" {
		t.Fatalf("control store = %q", configuration.ControlStoreDirectory)
	}
}

func TestRunConfigurationRejectsInlineAuthorizationAndWrongEnvironment(
	t *testing.T,
) {
	for name, body := range map[string]string{
		"inline token": `{
  "environment":"local-preprod",
  "profile":"smoke",
  "profileVersion":"1.0.0",
  "seedFile":"/run/secrets/preprod_seed",
  "authorizationFile":"/run/secrets/preprod_loader_authorization",
  "controlStoreDirectory":"/control",
  "intentFile":"/control/intent.json",
  "authorizationToken":"forbidden"
}`,
		"wrong environment": `{
  "environment":"production",
  "profile":"smoke",
  "profileVersion":"1.0.0",
  "seedFile":"/run/secrets/preprod_seed",
  "authorizationFile":"/run/secrets/preprod_loader_authorization",
  "controlStoreDirectory":"/control",
  "intentFile":"/control/intent.json"
}`,
	} {
		t.Run(name, func(t *testing.T) {
			configPath := filepath.Join(t.TempDir(), "loader-config.json")
			if err := os.WriteFile(configPath, []byte(body), 0o600); err != nil {
				t.Fatalf("write config: %v", err)
			}
			if _, err := loadRunConfiguration(configPath); err == nil {
				t.Fatalf("unsafe configuration was accepted")
			}
		})
	}
}

func TestReadIntentRejectsTrailingContent(t *testing.T) {
	profile, err := profiles.Lookup("smoke", "1.0.0")
	if err != nil {
		t.Fatalf("lookup profile: %v", err)
	}
	const runID = "run-task6-trailing-intent"
	intent, err := preproddata.BuildIntent(preproddata.IntentInput{
		RunID:          runID,
		Profile:        profile,
		SeedHash:       "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		CodeDigest:     "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		ContractDigest: "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
		Target: preproddata.TargetFingerprint{
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
		},
	})
	if err != nil {
		t.Fatalf("build intent: %v", err)
	}
	encoded, err := json.Marshal(intent)
	if err != nil {
		t.Fatalf("encode intent: %v", err)
	}
	intentPath := filepath.Join(t.TempDir(), "intent.json")
	encoded = append(encoded, []byte("\n{\"unexpected\":true}")...)
	if err := os.WriteFile(intentPath, encoded, 0o600); err != nil {
		t.Fatalf("write intent: %v", err)
	}

	if _, err := readIntent(intentPath); err == nil {
		t.Fatalf("intent reader accepted trailing JSON content")
	}
}

func TestQualificationInterruptionIsExplicitBoundedAndLoaderOnly(
	t *testing.T,
) {
	command := func(id string) preproddata.AuthoritativeCommand {
		return preproddata.AuthoritativeCommand{
			Family:      "organizations",
			OperationID: id,
			Payload:     []byte(`{"schemaVersion":"test/v1"}`),
		}
	}
	base := preproddata.NewSliceCommandStream(
		command("operation-1"),
		command("operation-2"),
	)
	lookup := func(name string) string {
		return map[string]string{
			"AVIA_PREPROD_PROFILE_QUALIFICATION":                  "true",
			"AVIA_PREPROD_QUALIFICATION_INTERRUPT_AFTER_COMMANDS": "1",
		}[name]
	}
	stream, err := qualificationCommandStream(base, lookup)
	if err != nil {
		t.Fatalf("qualification stream: %v", err)
	}
	if _, err := stream.Next(context.Background()); err != nil {
		t.Fatalf("first command: %v", err)
	}
	if _, err := stream.Next(context.Background()); !errors.Is(
		err,
		errQualificationInterruption,
	) {
		t.Fatalf("interruption error = %v", err)
	}
	if _, err := stream.Next(context.Background()); !errors.Is(
		err,
		errQualificationInterruption,
	) {
		t.Fatalf("post-interruption stream error = %v", err)
	}

	for name, environment := range map[string]map[string]string{
		"hook without qualification mode": {
			"AVIA_PREPROD_QUALIFICATION_INTERRUPT_AFTER_COMMANDS": "1",
		},
		"zero limit": {
			"AVIA_PREPROD_PROFILE_QUALIFICATION":                  "true",
			"AVIA_PREPROD_QUALIFICATION_INTERRUPT_AFTER_COMMANDS": "0",
		},
		"malformed limit": {
			"AVIA_PREPROD_PROFILE_QUALIFICATION":                  "true",
			"AVIA_PREPROD_QUALIFICATION_INTERRUPT_AFTER_COMMANDS": "one",
		},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := qualificationCommandStream(
				preproddata.NewSliceCommandStream(command("operation")),
				func(key string) string { return environment[key] },
			)
			if err == nil {
				t.Fatalf("unsafe qualification interruption was accepted")
			}
		})
	}
}
