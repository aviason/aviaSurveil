// Command preprod-canonical-aga-loader is the import-only bridge from the
// sealed AGA package into the canonical question catalog. It is deliberately
// a separate command package and is not linked into the normal API/worker.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/platform/database"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/preproddata/agacandidatedemo"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/preproddata/canonicalaga"
)

func main() {
	if err := run(context.Background(), os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "canonical AGA catalog loader:", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string, output io.Writer) error {
	if len(args) == 0 || (args[0] != "validate" && args[0] != "load") {
		return errors.New("usage: preprod-canonical-aga-loader validate|load --package PATH --catalog-version aga-preprod@1.0.0 [--database-url URL --actor-subject-id SUBJECT]")
	}
	command := args[0]
	flags := flag.NewFlagSet("preprod-canonical-aga-loader", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	packagePath := flags.String("package", "", "absolute sealed AGA package path")
	catalogVersion := flags.String("catalog-version", "aga-preprod@1.0.0", "immutable catalog version")
	databaseURL := flags.String("database-url", os.Getenv("AVIA_CANONICAL_AGA_DATABASE_URL"), "disposable canonical preprod database URL")
	actorSubjectID := flags.String("actor-subject-id", os.Getenv("AVIA_CANONICAL_AGA_LOADER_ACTOR"), "existing loader actor subject identity")
	if err := flags.Parse(args[1:]); err != nil {
		return errors.New("invalid canonical AGA loader arguments")
	}
	if !filepath.IsAbs(*packagePath) {
		return errors.New("--package must be an absolute regular package path")
	}
	if strings.TrimSpace(*catalogVersion) != "aga-preprod@1.0.0" {
		return errors.New("canonical AGA loader accepts only catalog version aga-preprod@1.0.0")
	}
	pkg, err := agacandidatedemo.NewPackageReader().ReadAndValidate(ctx, *packagePath, agacandidatedemo.ExactAcceptedPackage())
	if err != nil {
		return fmt.Errorf("validate sealed package: %w", err)
	}
	manifest, err := canonicalaga.BuildImportManifest(pkg, *catalogVersion)
	if err != nil {
		return fmt.Errorf("build canonical import manifest: %w", err)
	}
	if command == "validate" {
		_, err := fmt.Fprintf(output, "canonical AGA import validated: catalog=%s forms=%d questions=%d digest=%s\n", manifest.CatalogVersion, len(manifest.Forms), len(manifest.Rows), manifest.ImportDigest)
		return err
	}
	if strings.TrimSpace(*databaseURL) == "" || strings.TrimSpace(*actorSubjectID) == "" {
		return errors.New("--database-url and --actor-subject-id are required for load")
	}
	pool, err := database.Open(ctx, *databaseURL)
	if err != nil {
		return fmt.Errorf("open disposable canonical database: %w", err)
	}
	defer pool.Close()
	result, err := canonicalaga.LoadSealedCatalog(ctx, pool, pkg, *catalogVersion, *actorSubjectID, time.Now().UTC())
	if err != nil {
		return fmt.Errorf("load canonical catalog: %w", err)
	}
	_, err = fmt.Fprintf(output, "canonical AGA catalog loaded: catalog=%s forms=%d questions=%d digest=%s\n", result.CatalogVersion, result.FormCount, result.QuestionCount, result.ImportDigest)
	return err
}
