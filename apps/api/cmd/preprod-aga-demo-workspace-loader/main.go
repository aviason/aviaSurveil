package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	aga "github.com/MarlonJD/aviaSurveil360/apps/api/internal/agaapplicability"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/platform/database"
	workspace "github.com/MarlonJD/aviaSurveil360/apps/api/internal/preproddata/agademoworkspace"
)

func main() {
	if err := run(context.Background(), os.Args[1:], os.Stdout, nil); err != nil {
		fmt.Fprintln(os.Stderr, "ERR_AGA_WORKSPACE_LOAD", "contract")
		os.Exit(1)
	}
}

type loaderDependencies struct{ store workspace.Store }

func run(ctx context.Context, args []string, output io.Writer, dependencies *loaderDependencies) error {
	if len(args) == 0 || (args[0] != "validate" && args[0] != "load" && args[0] != "verify") {
		return errors.New("usage: validate|load|verify --candidate-dir PATH --fixture-manifest PATH --generation-id ID --taxonomy-version VERSION --taxonomy-digest DIGEST")
	}
	command := args[0]
	flags := flag.NewFlagSet("preprod-aga-demo-workspace-loader", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	candidateDir := flags.String("candidate-dir", "", "text-free candidate directory")
	fixturePath := flags.String("fixture-manifest", "", "private fixture manifest")
	generationID := flags.String("generation-id", "", "server-owned generation ID")
	taxonomyVersion := flags.String("taxonomy-version", "AGA_QUESTION_CLASSIFICATION_V1", "taxonomy version")
	taxonomyDigest := flags.String("taxonomy-digest", "", "taxonomy digest")
	databaseURL := flags.String("database-url", os.Getenv("AVIA_AGA_DEMO_WORKSPACE_LOADER_DATABASE_URL"), "loader database URL")
	if err := flags.Parse(args[1:]); err != nil {
		return errors.New("invalid workspace loader arguments")
	}
	if !filepath.IsAbs(*candidateDir) || !filepath.IsAbs(*fixturePath) {
		return errors.New("loader paths must be absolute")
	}
	fixture, err := workspace.LoadFixtureManifest(*fixturePath)
	if err != nil {
		return err
	}
	classification, err := readClassificationResult(*candidateDir)
	if err != nil {
		return err
	}
	input := workspace.LoadInput{GenerationID: *generationID, Classification: classification, Fixture: fixture, TaxonomyVersion: workspace.TaxonomyVersion{Version: *taxonomyVersion, Digest: *taxonomyDigest, PackageDigest: classification.RunReceipt.FixedInputDigests.PackageJSONSHA256, PublishedAt: nowUTC(), Sealed: true}, Now: nowUTC()}
	if command == "validate" {
		if err := validateOnly(input); err != nil {
			return err
		}
		_, err := fmt.Fprintln(output, "AGA demo workspace loader input validated")
		return err
	}
	if dependencies == nil {
		dependencies = &loaderDependencies{}
	}
	if dependencies.store == nil {
		if *databaseURL == "" {
			return errors.New("workspace loader database URL is required")
		}
		pool, err := database.Open(ctx, *databaseURL)
		if err != nil {
			return err
		}
		defer pool.Close()
		store, err := workspace.NewPostgresCommandStore(pool)
		if err != nil {
			return err
		}
		dependencies.store = store
	}
	if command == "verify" {
		_, err := dependencies.store.Snapshot(ctx)
		if err != nil {
			return err
		}
		_, err = fmt.Fprintln(output, "AGA demo workspace seal verified")
		return err
	}
	receipt, err := dependencies.store.LoadAndSeal(ctx, input)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(output, "AGA demo workspace sealed", receipt.GenerationID)
	return err
}

func validateOnly(input workspace.LoadInput) error {
	if input.GenerationID == "" || input.TaxonomyVersion.Version == "" || input.TaxonomyVersion.Digest == "" {
		return errors.New("workspace loader pins are required")
	}
	if input.Classification.State != aga.ClassificationRunSealed {
		return errors.New("classification candidate is not sealed")
	}
	return nil
}

func readClassificationResult(directory string) (aga.ClassificationResult, error) {
	path := filepath.Join(directory, "classification-result.json")
	info, err := os.Stat(path)
	if err != nil || !info.Mode().IsRegular() {
		return aga.ClassificationResult{}, errors.New("text-free classification-result.json is required")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return aga.ClassificationResult{}, err
	}
	var result aga.ClassificationResult
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&result); err != nil {
		return aga.ClassificationResult{}, errors.New("classification result is invalid")
	}
	return result, nil
}

func nowUTC() time.Time { return time.Now().UTC() }
