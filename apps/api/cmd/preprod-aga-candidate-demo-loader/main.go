package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/platform/database"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/preproddata/agacandidatedemo"
)

type runConfiguration struct {
	Environment           string                             `json:"environment"`
	RunID                 string                             `json:"runId"`
	CreatedAt             time.Time                          `json:"createdAt"`
	PackageFile           string                             `json:"packageFile"`
	ControlStoreDirectory string                             `json:"controlStoreDirectory"`
	AuthorizationFile     string                             `json:"authorizationFile"`
	BaseEvidenceFile      string                             `json:"baseEvidenceFile"`
	WriterPasswordFile    string                             `json:"writerPasswordFile"`
	CodeDigest            string                             `json:"codeDigest"`
	ContractDigest        string                             `json:"contractDigest"`
	Target                agacandidatedemo.TargetFingerprint `json:"target"`
}

type baseEvidenceDocument struct {
	RunID                   string `json:"runId"`
	IntentDigest            string `json:"intentDigest"`
	ResultDigest            string `json:"resultDigest"`
	TargetFingerprintDigest string `json:"targetFingerprintDigest"`
	Outcome                 string `json:"outcome"`
	Disposable              bool   `json:"disposable"`
}

type commandDependencies struct {
	openStore func(context.Context, runConfiguration) (agacandidatedemo.ProjectionStore, func(), error)
	now       func() time.Time
}

func main() {
	if err := run(context.Background(), os.Args[1:], os.Stdout, commandDependencies{}); err != nil {
		slog.Error("AGA candidate demo loader failed", "error", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, arguments []string, output io.Writer, dependencies commandDependencies) error {
	if err := rejectAuthorizationEnvironment(os.Environ()); err != nil {
		return err
	}
	if len(arguments) != 2 || !validCommand(arguments[0]) {
		return fmt.Errorf("usage: preprod-aga-candidate-demo-loader prepare-aga-demo|verify-aga-demo-authorization|run-aga-demo|verify-aga-demo|cleanup-aga-demo CONFIG_FILE")
	}
	configuration, err := loadRunConfiguration(arguments[1])
	if err != nil {
		return err
	}
	intent, accepted, control, err := prepareInputs(ctx, configuration)
	if err != nil {
		return err
	}
	switch arguments[0] {
	case "prepare-aga-demo":
		if err := control.AppendIntent(intent); err != nil {
			return err
		}
		_, err := fmt.Fprintf(output, "AGA candidate demo intent prepared: run=%s digest=%s\n", intent.RunID, intent.IntentDigest)
		return err
	case "verify-aga-demo-authorization":
		authorization, err := readAuthorization(configuration.AuthorizationFile)
		if err != nil {
			return err
		}
		if err := authorization.Validate(intent, agacandidatedemo.LoadOverlayOperation, now(dependencies)); err != nil {
			return err
		}
		_, err = fmt.Fprintf(output, "AGA candidate demo authorization verified: run=%s hash=%s\n", intent.RunID, authorization.Hash())
		return err
	case "run-aga-demo":
		if err := control.AppendIntent(intent); err != nil {
			return err
		}
		base, err := readBaseEvidence(configuration.BaseEvidenceFile)
		if err != nil {
			return err
		}
		if err := agacandidatedemo.VerifyBaseEvidence(intent, base); err != nil {
			return err
		}
		authorization, err := readAuthorization(configuration.AuthorizationFile)
		if err != nil {
			return err
		}
		if err := control.ConsumeAuthorization(authorization, agacandidatedemo.LoadOverlayOperation, now(dependencies)); err != nil {
			return err
		}
		store, closeStore, err := openStore(ctx, configuration, dependencies)
		if err != nil {
			return err
		}
		defer closeStore()
		result, err := agacandidatedemo.LoadOverlay(ctx, agacandidatedemo.OverlayLoadInput{Intent: intent, Package: accepted, BaseEvidence: base, Store: store, Clock: func() time.Time { return now(dependencies) }})
		if err != nil {
			return err
		}
		if err := control.AppendResult(result); err != nil {
			return err
		}
		_, err = fmt.Fprintf(output, "AGA candidate demo sealed: run=%s result=%s\n", result.RunID, result.ResultDigest)
		return err
	case "verify-aga-demo":
		base, err := readBaseEvidence(configuration.BaseEvidenceFile)
		if err != nil {
			return err
		}
		if err := agacandidatedemo.VerifyBaseEvidence(intent, base); err != nil {
			return err
		}
		store, closeStore, err := openStore(ctx, configuration, dependencies)
		if err != nil {
			return err
		}
		defer closeStore()
		receipt, err := store.VerifySeal(ctx, intent)
		if err != nil {
			return err
		}
		_, err = fmt.Fprintf(output, "AGA candidate demo seal verified: run=%s seal=%s\n", intent.RunID, receipt.SealDigest)
		return err
	case "cleanup-aga-demo":
		authorization, err := readAuthorization(configuration.AuthorizationFile)
		if err != nil {
			return err
		}
		if err := control.ConsumeAuthorization(authorization, agacandidatedemo.CleanupOverlayOperation, now(dependencies)); err != nil {
			return err
		}
		result, err := control.Result(intent.RunID, intent.IntentDigest)
		if err != nil {
			return err
		}
		tombstone, err := agacandidatedemo.BuildCleanupTombstone(agacandidatedemo.CleanupTombstoneInput{RunID: intent.RunID, IntentDigest: intent.IntentDigest, ResultDigest: result.ResultDigest, CleanedAt: now(dependencies)})
		if err != nil {
			return err
		}
		if err := control.AppendCleanupTombstone(tombstone); err != nil {
			return err
		}
		_, err = fmt.Fprintf(output, "AGA candidate demo cleanup tombstoned: run=%s tombstone=%s\n", intent.RunID, tombstone.TombstoneDigest)
		return err
	}
	return fmt.Errorf("unreachable")
}

func validCommand(command string) bool {
	return command == "prepare-aga-demo" || command == "verify-aga-demo-authorization" || command == "run-aga-demo" || command == "verify-aga-demo" || command == "cleanup-aga-demo"
}

func rejectAuthorizationEnvironment(environment []string) error {
	for _, entry := range environment {
		if strings.HasPrefix(entry, "AVIA_AGA_DEMO_AUTHORIZATION_TOKEN=") || strings.HasPrefix(entry, "AVIA_AGA_DEMO_TOKEN=") {
			return fmt.Errorf("AGA demo authorization must be supplied only through its private file")
		}
	}
	return nil
}

func loadRunConfiguration(path string) (runConfiguration, error) {
	data, err := readPrivateRegularFile(path)
	if err != nil {
		return runConfiguration{}, fmt.Errorf("read AGA demo configuration: %w", err)
	}
	var configuration runConfiguration
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&configuration); err != nil {
		return runConfiguration{}, fmt.Errorf("decode AGA demo configuration: %w", err)
	}
	if configuration.Environment != "local-preprod" || configuration.RunID == "" || configuration.CreatedAt.IsZero() || configuration.PackageFile == "" || configuration.ControlStoreDirectory == "" || configuration.AuthorizationFile == "" || configuration.BaseEvidenceFile == "" || configuration.WriterPasswordFile == "" || !isDigest(configuration.CodeDigest) || !isDigest(configuration.ContractDigest) {
		return runConfiguration{}, fmt.Errorf("invalid AGA demo configuration")
	}
	for _, candidate := range []string{configuration.PackageFile, configuration.ControlStoreDirectory, configuration.AuthorizationFile, configuration.BaseEvidenceFile, configuration.WriterPasswordFile} {
		if !filepath.IsAbs(candidate) {
			return runConfiguration{}, fmt.Errorf("AGA demo configuration requires absolute paths")
		}
	}
	if err := configuration.Target.Validate(); err != nil {
		return runConfiguration{}, err
	}
	return configuration, nil
}

func prepareInputs(ctx context.Context, configuration runConfiguration) (agacandidatedemo.IntentManifest, agacandidatedemo.AcceptedPackage, *agacandidatedemo.ControlStore, error) {
	accepted, err := agacandidatedemo.NewPackageReader().ReadAndValidate(ctx, configuration.PackageFile, agacandidatedemo.ExactAcceptedPackage())
	if err != nil {
		return agacandidatedemo.IntentManifest{}, agacandidatedemo.AcceptedPackage{}, nil, err
	}
	digests, err := agacandidatedemo.RelationshipDigests(accepted)
	if err != nil {
		return agacandidatedemo.IntentManifest{}, agacandidatedemo.AcceptedPackage{}, nil, err
	}
	base, err := readBaseEvidence(configuration.BaseEvidenceFile)
	if err != nil {
		return agacandidatedemo.IntentManifest{}, agacandidatedemo.AcceptedPackage{}, nil, err
	}
	intent, err := agacandidatedemo.BuildIntent(agacandidatedemo.IntentInput{RunID: configuration.RunID, BaseRunID: base.RunID, BaseIntentDigest: base.IntentDigest, BaseResultDigest: base.ResultDigest, BaseTargetDigest: base.TargetFingerprintDigest, CodeDigest: configuration.CodeDigest, ContractDigest: configuration.ContractDigest, ExpectedPackage: agacandidatedemo.ExactAcceptedPackage(), ExpectedRelationshipDigests: digests, Target: configuration.Target, CreatedAt: configuration.CreatedAt})
	if err != nil {
		return agacandidatedemo.IntentManifest{}, agacandidatedemo.AcceptedPackage{}, nil, err
	}
	if err := agacandidatedemo.VerifyBaseEvidence(intent, base); err != nil {
		return agacandidatedemo.IntentManifest{}, agacandidatedemo.AcceptedPackage{}, nil, err
	}
	control, err := agacandidatedemo.NewControlStore(configuration.ControlStoreDirectory)
	if err != nil {
		return agacandidatedemo.IntentManifest{}, agacandidatedemo.AcceptedPackage{}, nil, err
	}
	return intent, accepted, control, nil
}

func readAuthorization(path string) (agacandidatedemo.OperationAuthorization, error) {
	if _, err := readPrivateRegularFile(path); err != nil {
		return agacandidatedemo.OperationAuthorization{}, err
	}
	return agacandidatedemo.ReadAuthorizationFile(path)
}

func readBaseEvidence(path string) (agacandidatedemo.BaseRunEvidence, error) {
	data, err := readPrivateRegularFile(path)
	if err != nil {
		return agacandidatedemo.BaseRunEvidence{}, err
	}
	var document baseEvidenceDocument
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&document); err != nil {
		return agacandidatedemo.BaseRunEvidence{}, err
	}
	return agacandidatedemo.BaseRunEvidence{RunID: document.RunID, IntentDigest: document.IntentDigest, ResultDigest: document.ResultDigest, TargetFingerprintDigest: document.TargetFingerprintDigest, Outcome: document.Outcome, Disposable: document.Disposable}, nil
}

func readPrivateRegularFile(path string) ([]byte, error) {
	if !filepath.IsAbs(path) {
		return nil, fmt.Errorf("path must be absolute")
	}
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm() != 0o600 {
		return nil, fmt.Errorf("path must be a private 0600 regular file")
	}
	return os.ReadFile(path)
}

func openStore(ctx context.Context, configuration runConfiguration, dependencies commandDependencies) (agacandidatedemo.ProjectionStore, func(), error) {
	if dependencies.openStore != nil {
		return dependencies.openStore(ctx, configuration)
	}
	if err := agacandidatedemo.ValidateOverlayDatabaseUser(agacandidatedemo.OverlayWriterRole); err != nil {
		return nil, nil, err
	}
	password, err := readPrivateRegularFile(configuration.WriterPasswordFile)
	if err != nil {
		return nil, nil, err
	}
	connectionURL := writerDatabaseURL(configuration.Target, strings.TrimSpace(string(password)))
	pool, err := database.Open(ctx, connectionURL)
	if err != nil {
		return nil, nil, err
	}
	store, err := agacandidatedemo.NewPostgresStore(pool)
	if err != nil {
		pool.Close()
		return nil, nil, err
	}
	return store, pool.Close, nil
}

func writerDatabaseURL(target agacandidatedemo.TargetFingerprint, password string) string {
	return (&url.URL{Scheme: "postgres", User: url.UserPassword(agacandidatedemo.OverlayWriterRole, password), Host: net.JoinHostPort(target.PostgresHost, fmt.Sprintf("%d", target.PostgresPort)), Path: target.DatabaseName, RawQuery: "sslmode=disable"}).String()
}

func now(dependencies commandDependencies) time.Time {
	if dependencies.now != nil {
		return dependencies.now().UTC()
	}
	return time.Now().UTC()
}
func isDigest(value string) bool {
	return strings.HasPrefix(value, "sha256:") && len(value) == len("sha256:")+64
}
