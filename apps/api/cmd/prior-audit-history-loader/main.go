// Command prior-audit-history-loader installs the immutable, qualification-only
// prior-Audit history used by the Namibia local/demo recommendation presentation.
// It is deliberately a no-op outside the exact approved local targets.
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

	"github.com/aviason/aviaSurveil/internal/platform/database"
	"github.com/aviason/aviaSurveil/internal/qualificationbootstrap"
)

func main() {
	if err := run(context.Background(), os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "prior-audit history loader:", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string, output io.Writer) error {
	if len(args) == 0 || (args[0] != "validate" && args[0] != "load") {
		return errors.New("usage: prior-audit-history-loader validate|load")
	}
	target := strings.TrimSpace(os.Getenv("AVIA_BOOTSTRAP_TARGET"))
	if target != "namibia/dev" && target != "namibia/demo" {
		_, err := fmt.Fprintf(output, "prior-Audit history loader skipped: target=%s\n", target)
		return err
	}
	flags := flag.NewFlagSet("prior-audit-history-loader", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	manifestPath := flags.String("manifest", os.Getenv("AVIA_PRIOR_AUDIT_HISTORY_MANIFEST_PATH"), "absolute prior-Audit history manifest path")
	databaseURLFile := flags.String("database-url-file", os.Getenv("AVIA_DATABASE_URL_FILE"), "private target database URL file")
	if err := flags.Parse(args[1:]); err != nil {
		return errors.New("invalid prior-Audit history loader arguments")
	}
	if !filepath.IsAbs(strings.TrimSpace(*manifestPath)) {
		return errors.New("prior-Audit history manifest path must be absolute")
	}
	manifest, manifestDigest, err := qualificationbootstrap.ReadPriorAuditHistoryManifest(*manifestPath, target)
	if err != nil {
		return err
	}
	if args[0] == "validate" {
		_, err := fmt.Fprintf(output, "prior-Audit history manifest validated: target=%s audits=%d digest=%s\n", target, len(manifest.Audits), manifestDigest)
		return err
	}
	databaseURL, err := readSecretFile(*databaseURLFile)
	if err != nil {
		return err
	}
	pool, err := database.Open(ctx, databaseURL)
	if err != nil {
		return fmt.Errorf("open prior-Audit history database: %w", err)
	}
	defer pool.Close()
	if err := qualificationbootstrap.LoadPriorAuditHistory(ctx, pool, manifest, qualificationbootstrap.PriorAuditHistoryLoaderActor, time.Now().UTC()); err != nil {
		return err
	}
	_, err = fmt.Fprintf(output, "prior-Audit history loaded: target=%s audits=%d manifest=%s\n", target, len(manifest.Audits), manifestDigest)
	return err
}

func readSecretFile(path string) (string, error) {
	if !filepath.IsAbs(path) {
		return "", errors.New("database URL file must be absolute")
	}
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || info.Size() < 1 || info.Size() > 8192 {
		return "", errors.New("database URL file is invalid")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	value := strings.TrimSpace(string(data))
	if value == "" || strings.ContainsAny(value, "\r\n") {
		return "", errors.New("database URL file is empty or malformed")
	}
	return value, nil
}
