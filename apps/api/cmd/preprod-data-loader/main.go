package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/preproddata"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/preproddata/profiles"
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
	CodeDigest            string                        `json:"codeDigest,omitempty"`
	ContractDigest        string                        `json:"contractDigest,omitempty"`
	Target                preproddata.TargetFingerprint `json:"target,omitempty"`
}

func main() {
	if err := run(context.Background(), os.Args[1:], os.Stdout); err != nil {
		slog.Error("preprod data loader failed", "error", err)
		os.Exit(1)
	}
}

func run(_ context.Context, arguments []string, output io.Writer) error {
	if len(arguments) != 2 ||
		(arguments[0] != "prepare" && arguments[0] != "verify-authorization") {
		return fmt.Errorf(
			"usage: preprod-data-loader prepare|verify-authorization CONFIG_FILE",
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
	}
	panic("unreachable")
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
