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
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/preproddata/canonicalaga"
	"github.com/jackc/pgx/v5"
)

const loaderActorSubjectID = "canonical-aga-preprod-loader"

func main() {
	if err := run(context.Background(), os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "canonical AGA catalog loader:", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string, output io.Writer) error {
	if len(args) == 0 || (args[0] != "validate" && args[0] != "load") {
		return errors.New("usage: preprod-canonical-aga-loader validate|load --package PATH --catalog-version aga-preprod@1.0.0 [--database-url URL --provider-scope-id SCOPE --regulated-target-id TARGET --actor-subject-id SUBJECT --identity-namespace canonical-aga-preprod-exercise-v1]")
	}
	command := args[0]
	flags := flag.NewFlagSet("preprod-canonical-aga-loader", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	packagePath := flags.String("package", "", "absolute sealed AGA package path")
	catalogVersion := flags.String("catalog-version", "aga-preprod@1.0.0", "immutable catalog version")
	databaseURL := flags.String("database-url", os.Getenv("AVIA_CANONICAL_AGA_DATABASE_URL"), "disposable canonical preprod database URL")
	actorSubjectID := flags.String("actor-subject-id", os.Getenv("AVIA_CANONICAL_AGA_LOADER_ACTOR"), "deprecated; must be the fixed loader service identity")
	identityNamespace := flags.String("identity-namespace", os.Getenv("AVIA_PREPROD_IDENTITY_NAMESPACE"), "whole-namespace-disposable preprod identity namespace")
	providerScopeID := flags.String("provider-scope-id", os.Getenv("AVIA_CANONICAL_AGA_PROVIDER_SCOPE_ID"), "one explicit provider-scope identity eligible for the disposable exercise catalog")
	regulatedTargetID := flags.String("regulated-target-id", os.Getenv("AVIA_CANONICAL_AGA_REGULATED_TARGET_ID"), "one explicit regulated-target identity eligible for the disposable exercise catalog")
	if err := flags.Parse(args[1:]); err != nil {
		return errors.New("invalid canonical AGA loader arguments")
	}
	if !filepath.IsAbs(*packagePath) {
		return errors.New("--package must be an absolute regular package path")
	}
	if strings.TrimSpace(*catalogVersion) != "aga-preprod@1.0.0" {
		return errors.New("canonical AGA loader accepts only catalog version aga-preprod@1.0.0")
	}
	if strings.TrimSpace(*identityNamespace) != "canonical-aga-preprod-exercise-v1" {
		return errors.New("canonical AGA loader requires the dedicated disposable identity namespace canonical-aga-preprod-exercise-v1")
	}
	if strings.TrimSpace(*actorSubjectID) != "" && strings.TrimSpace(*actorSubjectID) != loaderActorSubjectID {
		return errors.New("--actor-subject-id cannot select a caller-supplied identity; use the fixed canonical-aga-preprod-loader service identity")
	}
	*actorSubjectID = loaderActorSubjectID
	pkg, err := canonicalaga.NewPackageReader().ReadAndValidate(ctx, *packagePath, canonicalaga.ExactAcceptedPackage())
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
	if strings.TrimSpace(*databaseURL) == "" {
		return errors.New("--database-url is required for load")
	}
	if strings.TrimSpace(*providerScopeID) == "" || strings.TrimSpace(*regulatedTargetID) == "" {
		return errors.New("--provider-scope-id and --regulated-target-id are required for load; exercise applicability must be explicitly bound")
	}
	pool, err := database.Open(ctx, *databaseURL)
	if err != nil {
		return fmt.Errorf("open disposable canonical database: %w", err)
	}
	defer pool.Close()
	// The loader is allowed to write only to the task-owned whole-namespace
	// database. Matching the URL alone is insufficient because a caller could
	// point it at a shared server with the same logical application name.
	var databaseName, databaseUser, databaseOwner string
	if err := pool.QueryRow(ctx, `
		SELECT current_database(), current_user, pg_get_userbyid(datdba)
		FROM pg_database
		WHERE datname = current_database()
	`).Scan(&databaseName, &databaseUser, &databaseOwner); err != nil {
		return fmt.Errorf("verify disposable database identity: %w", err)
	}
	if databaseName != "aviasurveil360_local_preprod" || databaseOwner != "aviasurveil360_preprod_loader" || databaseUser != "aviasurveil360_preprod_loader" {
		return fmt.Errorf("canonical AGA loader requires database aviasurveil360_local_preprod owned and accessed by aviasurveil360_preprod_loader (got %s owned by %s, accessed by %s)", databaseName, databaseOwner, databaseUser)
	}
	// The import actor is an explicit local service identity.  It is not a
	// stakeholder account and is created only inside this disposable namespace
	// so every immutable catalog row still has a valid identity FK.
	var issuer, displayName string
	err = pool.QueryRow(ctx, `SELECT issuer, display_name FROM identity_references WHERE subject_id = $1`, *actorSubjectID).Scan(&issuer, &displayName)
	if errors.Is(err, pgx.ErrNoRows) {
		_, err = pool.Exec(ctx, `
			INSERT INTO identity_references (subject_id, issuer, display_name)
			VALUES ($1, 'local-preprod-import', 'Canonical AGA catalog loader')
		`, *actorSubjectID)
	} else if err == nil && (issuer != "local-preprod-import" || displayName != "Canonical AGA catalog loader") {
		return fmt.Errorf("fixed loader identity %s is not the canonical local-preprod service identity", *actorSubjectID)
	}
	if err != nil {
		return fmt.Errorf("register canonical catalog loader actor: %w", err)
	}
	result, err := canonicalaga.LoadSealedCatalog(ctx, pool, pkg, *catalogVersion, *actorSubjectID, *providerScopeID, *regulatedTargetID, time.Now().UTC())
	if err != nil {
		return fmt.Errorf("load canonical catalog: %w", err)
	}
	_, err = fmt.Fprintf(output, "canonical AGA catalog loaded: catalog=%s forms=%d questions=%d digest=%s\n", result.CatalogVersion, result.FormCount, result.QuestionCount, result.ImportDigest)
	return err
}
