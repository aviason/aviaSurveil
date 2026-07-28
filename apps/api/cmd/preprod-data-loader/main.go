package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/platform/database"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/preproddata"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/preproddata/profiles"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/preproddata/scenarios"
)

type runConfiguration struct {
	Environment           string                        `json:"environment"`
	Profile               string                        `json:"profile"`
	ProfileVersion        string                        `json:"profileVersion"`
	RunID                 string                        `json:"runId,omitempty"`
	SeedFile              string                        `json:"seedFile"`
	AuthorizationFile     string                        `json:"authorizationFile"`
	ControlStoreDirectory string                        `json:"controlStoreDirectory"`
	IntentFile            string                        `json:"intentFile"`
	RouteCatalogFile      string                        `json:"routeCatalogFile,omitempty"`
	BehaviorLedgerFile    string                        `json:"behaviorLedgerFile,omitempty"`
	CodeDigest            string                        `json:"codeDigest,omitempty"`
	ContractDigest        string                        `json:"contractDigest,omitempty"`
	Target                preproddata.TargetFingerprint `json:"target,omitempty"`
}

type commandDependencies struct {
	runConnected func(
		context.Context,
		runConfiguration,
	) (preproddata.ResultManifest, error)
	recordCleanup func(
		runConfiguration,
	) (preproddata.CleanupAttestation, error)
}

type connectedInputs struct {
	Intent        preproddata.IntentManifest
	Authorization preproddata.OperationAuthorization
	Profile       profiles.Profile
	Seed          []byte
	Catalog       scenarios.Catalog
	ControlStore  *preproddata.FileControlStore
}

func main() {
	if err := run(context.Background(), os.Args[1:], os.Stdout); err != nil {
		slog.Error("preprod data loader failed", "error", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, arguments []string, output io.Writer) error {
	return runWithDependencies(
		ctx,
		arguments,
		output,
		commandDependencies{
			runConnected: runConnectedData,
			recordCleanup: func(
				configuration runConfiguration,
			) (preproddata.CleanupAttestation, error) {
				return recordCleanupData(
					configuration,
					time.Now().UTC(),
				)
			},
		},
	)
}

func runWithDependencies(
	ctx context.Context,
	arguments []string,
	output io.Writer,
	dependencies commandDependencies,
) error {
	if len(arguments) != 2 ||
		(arguments[0] != "prepare" &&
			arguments[0] != "verify-authorization" &&
			arguments[0] != "run-connected" &&
			arguments[0] != "record-cleanup") {
		return fmt.Errorf(
			"usage: preprod-data-loader prepare|verify-authorization|run-connected|record-cleanup CONFIG_FILE",
		)
	}
	configuration, err := loadRunConfiguration(arguments[1])
	if err != nil {
		return err
	}
	switch arguments[0] {
	case "prepare":
		intent, err := prepareIntent(configuration)
		if err != nil {
			return err
		}
		_, err = fmt.Fprintf(
			output,
			"preprod intent prepared: run=%s digest=%s target=%s\n",
			intent.RunID,
			intent.IntentDigest,
			intent.TargetFingerprintDigest,
		)
		return err
	case "verify-authorization":
		intent, err := readIntent(configuration.IntentFile)
		if err != nil {
			return err
		}
		authorization, err := preproddata.ReadAuthorizationFile(
			configuration.AuthorizationFile,
		)
		if err != nil {
			return err
		}
		if err := authorization.Validate(intent, time.Now().UTC()); err != nil {
			return err
		}
		_, err = fmt.Fprintf(
			output,
			"preprod authorization verified: run=%s operation=%s hash=%s\n",
			intent.RunID,
			authorization.Operation,
			authorization.Hash(),
		)
		return err
	case "run-connected":
		if dependencies.runConnected == nil {
			return fmt.Errorf("connected runner dependency is required")
		}
		result, err := dependencies.runConnected(ctx, configuration)
		if err != nil {
			return err
		}
		_, err = fmt.Fprintf(
			output,
			"preprod connected run: run=%s outcome=%s result=%s\n",
			result.RunID,
			result.Outcome,
			result.ResultDigest,
		)
		return err
	case "record-cleanup":
		if dependencies.recordCleanup == nil {
			return fmt.Errorf("cleanup recorder dependency is required")
		}
		attestation, err := dependencies.recordCleanup(configuration)
		if err != nil {
			return err
		}
		_, err = fmt.Fprintf(
			output,
			"preprod cleanup attested: run=%s result=%s authorization=%s attestation=%s\n",
			attestation.RunID,
			attestation.ResultDigest,
			attestation.AuthorizationHash,
			attestation.AttestationDigest,
		)
		return err
	}
	panic("unreachable")
}

func recordCleanupData(
	configuration runConfiguration,
	now time.Time,
) (preproddata.CleanupAttestation, error) {
	intent, err := readIntent(configuration.IntentFile)
	if err != nil {
		return preproddata.CleanupAttestation{}, err
	}
	if configuration.Environment != "local-preprod" ||
		configuration.Profile != intent.ProfileName ||
		configuration.ProfileVersion != intent.ProfileVersion ||
		(configuration.RunID != "" && configuration.RunID != intent.RunID) ||
		(configuration.Target.Environment != "" &&
			configuration.Target != intent.Target) {
		return preproddata.CleanupAttestation{}, fmt.Errorf(
			"cleanup configuration differs from immutable intent",
		)
	}
	authorization, err := preproddata.ReadAuthorizationFile(
		configuration.AuthorizationFile,
	)
	if err != nil {
		return preproddata.CleanupAttestation{}, err
	}
	now = now.UTC()
	if err := authorization.Validate(intent, now); err != nil {
		return preproddata.CleanupAttestation{}, err
	}
	if authorization.Operation != preproddata.DropRecreateTarget {
		return preproddata.CleanupAttestation{}, fmt.Errorf(
			"cleanup requires a DROP_RECREATE_TARGET authorization",
		)
	}
	controlStore, err := preproddata.NewFileControlStore(
		configuration.ControlStoreDirectory,
	)
	if err != nil {
		return preproddata.CleanupAttestation{}, err
	}
	result, err := controlStore.SuccessfulResult(
		intent.RunID,
		intent.IntentDigest,
	)
	if err != nil {
		return preproddata.CleanupAttestation{}, err
	}
	if err := controlStore.ConsumeAuthorization(
		authorization,
		now,
	); err != nil {
		return preproddata.CleanupAttestation{}, err
	}
	attestation, err := preproddata.BuildCleanupAttestation(
		preproddata.CleanupAttestationInput{
			RunID:             intent.RunID,
			IntentDigest:      intent.IntentDigest,
			ResultDigest:      result.ResultDigest,
			TargetDigest:      intent.TargetFingerprintDigest,
			AuthorizationHash: authorization.Hash(),
			CleanedAt:         now,
		},
	)
	if err != nil {
		return preproddata.CleanupAttestation{}, err
	}
	if err := controlStore.AppendCleanupAttestation(attestation); err != nil {
		return preproddata.CleanupAttestation{}, err
	}
	return attestation, nil
}

func runConnectedData(
	ctx context.Context,
	configuration runConfiguration,
) (preproddata.ResultManifest, error) {
	inputs, err := loadConnectedInputs(configuration, time.Now().UTC())
	if err != nil {
		return preproddata.ResultManifest{}, err
	}
	if err := validateConnectedRuntimeBinding(inputs.Intent.Target); err != nil {
		return preproddata.ResultManifest{}, err
	}
	databasePassword, err := readRuntimeSecret(
		requiredEnvironment("AVIA_DATABASE_PASSWORD_FILE"),
	)
	if err != nil {
		return preproddata.ResultManifest{}, err
	}
	databaseURL := url.URL{
		Scheme: "postgres",
		User: url.UserPassword(
			inputs.Intent.Target.DatabaseOwner,
			databasePassword,
		),
		Host: net.JoinHostPort(
			inputs.Intent.Target.PostgresHost,
			strconv.Itoa(inputs.Intent.Target.PostgresPort),
		),
		Path: "/" + inputs.Intent.Target.DatabaseName,
	}
	databaseURL.RawQuery = "sslmode=disable"
	pool, err := database.Open(ctx, databaseURL.String())
	if err != nil {
		return preproddata.ResultManifest{}, err
	}
	defer pool.Close()
	var databaseName, databaseOwner, systemIdentifier string
	if err := pool.QueryRow(ctx, `
		SELECT current_database(), current_user,
		       system_identifier::text
		FROM pg_control_system()
	`).Scan(
		&databaseName,
		&databaseOwner,
		&systemIdentifier,
	); err != nil {
		return preproddata.ResultManifest{}, fmt.Errorf(
			"read PostgreSQL target fingerprint: %w",
			err,
		)
	}
	if databaseName != inputs.Intent.Target.DatabaseName ||
		databaseOwner != inputs.Intent.Target.DatabaseOwner ||
		systemIdentifier != inputs.Intent.Target.PostgresSystemIdentifier {
		return preproddata.ResultManifest{}, fmt.Errorf(
			"PostgreSQL runtime fingerprint differs from immutable intent",
		)
	}
	store, err := scenarios.NewPostgresStore(
		pool,
		inputs.Profile,
		inputs.Intent.RunID,
	)
	if err != nil {
		return preproddata.ResultManifest{}, err
	}
	keycloakSecret, err := readRuntimeSecret(requiredEnvironment(
		"AVIA_KEYCLOAK_SERVICE_CLIENT_SECRET_FILE",
	))
	if err != nil {
		return preproddata.ResultManifest{}, err
	}
	httpClient := &http.Client{Timeout: 10 * time.Second}
	keycloak, err := scenarios.NewKeycloakEndpoint(
		scenarios.KeycloakEndpointConfig{
			BaseURL:      requiredEnvironment("AVIA_KEYCLOAK_ADMIN_URL"),
			Realm:        inputs.Intent.Target.KeycloakRealm,
			ClientID:     inputs.Intent.Target.KeycloakServiceClientID,
			ClientSecret: keycloakSecret,
			HTTPClient:   httpClient,
		},
	)
	if err != nil {
		return preproddata.ResultManifest{}, err
	}
	invitations, err := scenarios.NewMailpitInvitationEndpoint(
		scenarios.MailpitInvitationEndpointConfig{
			Keycloak: keycloak,
			BaseURL: requiredEnvironment(
				"AVIA_PREPROD_MAILPIT_HTTP_URL",
			),
			HTTPClient: httpClient,
		},
	)
	if err != nil {
		return preproddata.ResultManifest{}, err
	}
	objectAccessKey, err := readRuntimeSecret(requiredEnvironment(
		"AVIA_OBJECT_STORE_ACCESS_KEY_FILE",
	))
	if err != nil {
		return preproddata.ResultManifest{}, err
	}
	objectSecretKey, err := readRuntimeSecret(requiredEnvironment(
		"AVIA_OBJECT_STORE_SECRET_KEY_FILE",
	))
	if err != nil {
		return preproddata.ResultManifest{}, err
	}
	objectBackend, err := scenarios.NewMinIOObjectBackend(
		scenarios.MinIOObjectBackendConfig{
			Endpoint:   requiredEnvironment("AVIA_OBJECT_STORE_ENDPOINT"),
			AccessKey:  objectAccessKey,
			SecretKey:  objectSecretKey,
			HTTPClient: httpClient,
		},
	)
	if err != nil {
		return preproddata.ResultManifest{}, err
	}
	objects, err := scenarios.NewConnectedObjectEndpoint(
		scenarios.ConnectedObjectEndpointConfig{
			Bucket:  inputs.Intent.Target.ObjectBucket,
			Prefix:  inputs.Intent.Target.ObjectPrefix,
			Backend: objectBackend,
		},
	)
	if err != nil {
		return preproddata.ResultManifest{}, err
	}
	boundary, err := scenarios.NewConnectedBoundary(
		scenarios.ConnectedBoundaryConfig{
			Target:      inputs.Intent.Target,
			Store:       store,
			Identity:    keycloak,
			Invitations: invitations,
			Objects:     objects,
		},
	)
	if err != nil {
		return preproddata.ResultManifest{}, err
	}
	stream, err := scenarios.NewStream(
		inputs.Profile,
		inputs.Seed,
		inputs.Catalog,
	)
	if err != nil {
		return preproddata.ResultManifest{}, err
	}
	return preproddata.Run(
		ctx,
		preproddata.RunInput{
			Intent:        inputs.Intent,
			Authorization: inputs.Authorization,
			ControlStore:  inputs.ControlStore,
			Boundary:      boundary,
			Commands:      stream,
			Clock:         time.Now,
		},
	)
}

func validateConnectedRuntimeBinding(
	target preproddata.TargetFingerprint,
) error {
	expected := map[string]string{
		"AVIA_ENVIRONMENT":                        target.Environment,
		"AVIA_PREPROD_DATABASE_NAME":              target.DatabaseName,
		"AVIA_PREPROD_DATABASE_OWNER":             target.DatabaseOwner,
		"AVIA_PREPROD_DATABASE_HOST":              target.PostgresHost,
		"AVIA_PREPROD_DATABASE_PORT":              strconv.Itoa(target.PostgresPort),
		"AVIA_PREPROD_COMPOSE_PROJECT":            target.ComposeProject,
		"AVIA_PREPROD_KEYCLOAK_REALM":             target.KeycloakRealm,
		"AVIA_PREPROD_KEYCLOAK_DATABASE":          target.KeycloakDatabase,
		"AVIA_PREPROD_KEYCLOAK_SERVICE_CLIENT_ID": target.KeycloakServiceClientID,
		"AVIA_PREPROD_MAILPIT_NAMESPACE":          target.MailpitNamespace,
		"AVIA_PREPROD_OBJECT_BUCKET":              target.ObjectBucket,
		"AVIA_PREPROD_LOADER_QUEUE_NAMESPACE":     target.LoaderQueueNamespace,
	}
	for name, value := range expected {
		if requiredEnvironment(name) != value {
			return fmt.Errorf(
				"runtime %s differs from immutable target",
				name,
			)
		}
	}
	for _, name := range []string{
		"AVIA_DATABASE_PASSWORD_FILE",
		"AVIA_KEYCLOAK_ADMIN_URL",
		"AVIA_KEYCLOAK_SERVICE_CLIENT_SECRET_FILE",
		"AVIA_PREPROD_MAILPIT_HTTP_URL",
		"AVIA_OBJECT_STORE_ENDPOINT",
		"AVIA_OBJECT_STORE_ACCESS_KEY_FILE",
		"AVIA_OBJECT_STORE_SECRET_KEY_FILE",
	} {
		if requiredEnvironment(name) == "" {
			return fmt.Errorf("runtime %s is required", name)
		}
	}
	return nil
}

func requiredEnvironment(name string) string {
	return strings.TrimSpace(os.Getenv(name))
}

func readRuntimeSecret(path string) (string, error) {
	if !filepath.IsAbs(path) {
		return "", fmt.Errorf("runtime secret path must be absolute")
	}
	info, err := os.Lstat(path)
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() ||
		info.Mode()&os.ModeSymlink != 0 ||
		info.Size() < 1 ||
		info.Size() > 4096 {
		return "", fmt.Errorf("runtime secret must be a bounded regular file")
	}
	value, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	secret := strings.TrimSpace(string(value))
	if secret == "" {
		return "", fmt.Errorf("runtime secret is empty")
	}
	return secret, nil
}

func loadConnectedInputs(
	configuration runConfiguration,
	now time.Time,
) (connectedInputs, error) {
	intent, err := readIntent(configuration.IntentFile)
	if err != nil {
		return connectedInputs{}, err
	}
	if configuration.Environment != "local-preprod" ||
		configuration.Profile != intent.ProfileName ||
		configuration.ProfileVersion != intent.ProfileVersion ||
		(configuration.RunID != "" && configuration.RunID != intent.RunID) ||
		(configuration.Target.Environment != "" &&
			configuration.Target != intent.Target) {
		return connectedInputs{}, fmt.Errorf(
			"connected configuration differs from immutable intent",
		)
	}
	profile, err := profiles.Lookup(
		configuration.Profile,
		configuration.ProfileVersion,
	)
	if err != nil {
		return connectedInputs{}, err
	}
	seed, err := readPrivateSeed(configuration.SeedFile)
	if err != nil {
		return connectedInputs{}, err
	}
	generator, err := preproddata.NewGenerator(profile, seed)
	if err != nil {
		return connectedInputs{}, err
	}
	if generator.SeedHash() != intent.SeedHash {
		return connectedInputs{}, fmt.Errorf(
			"connected seed does not match immutable intent",
		)
	}
	authorization, err := preproddata.ReadAuthorizationFile(
		configuration.AuthorizationFile,
	)
	if err != nil {
		return connectedInputs{}, err
	}
	if err := authorization.Validate(intent, now.UTC()); err != nil {
		return connectedInputs{}, err
	}
	routeSource, err := readCanonicalCatalogFile(
		configuration.RouteCatalogFile,
	)
	if err != nil {
		return connectedInputs{}, fmt.Errorf("read route catalog: %w", err)
	}
	ledgerSource, err := readCanonicalCatalogFile(
		configuration.BehaviorLedgerFile,
	)
	if err != nil {
		return connectedInputs{}, fmt.Errorf(
			"read behavior ledger: %w",
			err,
		)
	}
	catalog, err := scenarios.ParseCatalogs(routeSource, ledgerSource)
	if err != nil {
		return connectedInputs{}, err
	}
	controlStore, err := preproddata.NewFileControlStore(
		configuration.ControlStoreDirectory,
	)
	if err != nil {
		return connectedInputs{}, err
	}
	return connectedInputs{
		Intent:        intent,
		Authorization: authorization,
		Profile:       profile,
		Seed:          seed,
		Catalog:       catalog,
		ControlStore:  controlStore,
	}, nil
}

func readPrivateSeed(path string) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() ||
		info.Mode()&os.ModeSymlink != 0 ||
		info.Mode().Perm() != 0o600 {
		return nil, fmt.Errorf("seed file must be a private 0600 regular file")
	}
	seed, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	seed = bytes.TrimSpace(seed)
	if len(seed) == 0 || len(seed) > 4096 {
		return nil, fmt.Errorf("seed file content must be bounded and non-empty")
	}
	return seed, nil
}

func readCanonicalCatalogFile(path string) ([]byte, error) {
	if !filepath.IsAbs(path) {
		return nil, fmt.Errorf("catalog path must be absolute")
	}
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() ||
		info.Mode()&os.ModeSymlink != 0 ||
		info.Size() < 2 ||
		info.Size() > 16<<20 {
		return nil, fmt.Errorf("catalog must be a bounded regular file")
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	return io.ReadAll(io.LimitReader(file, 16<<20))
}

func loadRunConfiguration(path string) (runConfiguration, error) {
	if !filepath.IsAbs(path) {
		return runConfiguration{}, fmt.Errorf("configuration path must be absolute")
	}
	info, err := os.Stat(path)
	if err != nil {
		return runConfiguration{}, err
	}
	if !info.Mode().IsRegular() || info.Mode().Perm()&0o077 != 0 {
		return runConfiguration{}, fmt.Errorf(
			"configuration file must be a private regular file",
		)
	}
	file, err := os.Open(path)
	if err != nil {
		return runConfiguration{}, err
	}
	defer file.Close()
	decoder := json.NewDecoder(io.LimitReader(file, 128*1024))
	decoder.DisallowUnknownFields()
	var configuration runConfiguration
	if err := decoder.Decode(&configuration); err != nil {
		return runConfiguration{}, fmt.Errorf("decode configuration: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return runConfiguration{}, fmt.Errorf("configuration contains trailing data")
	}
	if configuration.Environment != "local-preprod" {
		return runConfiguration{}, fmt.Errorf(
			"environment must be exactly local-preprod",
		)
	}
	for name, value := range map[string]string{
		"seedFile":              configuration.SeedFile,
		"authorizationFile":     configuration.AuthorizationFile,
		"controlStoreDirectory": configuration.ControlStoreDirectory,
		"intentFile":            configuration.IntentFile,
	} {
		if !filepath.IsAbs(value) {
			return runConfiguration{}, fmt.Errorf("%s must be an absolute path", name)
		}
	}
	if configuration.Profile == "" || configuration.ProfileVersion == "" {
		return runConfiguration{}, fmt.Errorf("profile and profileVersion are required")
	}
	return configuration, nil
}

func prepareIntent(configuration runConfiguration) (preproddata.IntentManifest, error) {
	if configuration.RunID == "" ||
		configuration.CodeDigest == "" ||
		configuration.ContractDigest == "" {
		return preproddata.IntentManifest{}, fmt.Errorf(
			"runId, codeDigest, and contractDigest are required for prepare",
		)
	}
	seedInfo, err := os.Stat(configuration.SeedFile)
	if err != nil {
		return preproddata.IntentManifest{}, err
	}
	if !seedInfo.Mode().IsRegular() || seedInfo.Mode().Perm() != 0o600 {
		return preproddata.IntentManifest{}, fmt.Errorf(
			"seed file must be a 0600 regular file",
		)
	}
	seed, err := os.ReadFile(configuration.SeedFile)
	if err != nil {
		return preproddata.IntentManifest{}, err
	}
	profile, err := profiles.Lookup(
		configuration.Profile,
		configuration.ProfileVersion,
	)
	if err != nil {
		return preproddata.IntentManifest{}, err
	}
	generator, err := preproddata.NewGenerator(profile, bytes.TrimSpace(seed))
	if err != nil {
		return preproddata.IntentManifest{}, err
	}
	intent, err := preproddata.BuildIntent(preproddata.IntentInput{
		RunID: configuration.RunID, Profile: profile,
		SeedHash: generator.SeedHash(), CodeDigest: configuration.CodeDigest,
		ContractDigest: configuration.ContractDigest, Target: configuration.Target,
	})
	if err != nil {
		return preproddata.IntentManifest{}, err
	}
	store, err := preproddata.NewFileControlStore(
		configuration.ControlStoreDirectory,
	)
	if err != nil {
		return preproddata.IntentManifest{}, err
	}
	if err := store.AppendIntent(intent); err != nil {
		return preproddata.IntentManifest{}, err
	}
	encoded, err := json.Marshal(intent)
	if err != nil {
		return preproddata.IntentManifest{}, err
	}
	if err := writeImmutable(configuration.IntentFile, encoded); err != nil {
		return preproddata.IntentManifest{}, err
	}
	return intent, nil
}

func readIntent(path string) (preproddata.IntentManifest, error) {
	file, err := os.Open(path)
	if err != nil {
		return preproddata.IntentManifest{}, err
	}
	defer file.Close()
	decoder := json.NewDecoder(io.LimitReader(file, 512*1024))
	decoder.DisallowUnknownFields()
	var intent preproddata.IntentManifest
	if err := decoder.Decode(&intent); err != nil {
		return preproddata.IntentManifest{}, err
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return preproddata.IntentManifest{}, fmt.Errorf(
			"intent contains trailing data",
		)
	}
	if err := intent.Validate(); err != nil {
		return preproddata.IntentManifest{}, err
	}
	return intent, nil
}

func writeImmutable(path string, contents []byte) error {
	if !filepath.IsAbs(path) {
		return fmt.Errorf("output path must be absolute")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	file, err := os.OpenFile(
		path,
		os.O_WRONLY|os.O_CREATE|os.O_EXCL,
		0o600,
	)
	if errors.Is(err, os.ErrExist) {
		existing, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		if bytes.Equal(existing, contents) {
			return nil
		}
		return preproddata.ErrAppendOnlyConflict
	}
	if err != nil {
		return err
	}
	if _, err := file.Write(contents); err != nil {
		_ = file.Close()
		return err
	}
	return file.Close()
}
