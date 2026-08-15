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

	"github.com/aviason/aviaSurveil/internal/identity"
	"github.com/aviason/aviaSurveil/internal/platform/database"
	"github.com/aviason/aviaSurveil/internal/qualificationbootstrap"
)

const actorSubjectID = "avia-bootstrap"

func main() {
	if err := run(context.Background(), os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "prepared roster loader:", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string, output io.Writer) error {
	if len(args) == 0 || (args[0] != "validate" && args[0] != "load") {
		return errors.New("usage: prepared-roster-loader validate|load")
	}
	command := args[0]
	flags := flag.NewFlagSet("prepared-roster-loader", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	manifestPath := flags.String("manifest", os.Getenv("AVIA_ROSTER_MANIFEST_PATH"), "absolute roster manifest path")
	digest := flags.String("manifest-sha256", os.Getenv("AVIA_ROSTER_MANIFEST_SHA256"), "release-pinned manifest digest")
	target := flags.String("target", os.Getenv("AVIA_BOOTSTRAP_TARGET"), "exact target")
	databaseURLFile := flags.String("database-url-file", os.Getenv("AVIA_DATABASE_URL_FILE"), "private database URL file")
	issuer := flags.String("issuer", os.Getenv("AVIA_OIDC_ISSUER_URL"), "provider issuer")
	secretDirectory := flags.String("credential-directory", os.Getenv("AVIA_ROSTER_CREDENTIAL_DIRECTORY"), "private roster credential directory")
	if err := flags.Parse(args[1:]); err != nil {
		return errors.New("invalid roster loader arguments")
	}
	if !filepath.IsAbs(*manifestPath) || strings.TrimSpace(*target) == "" || strings.TrimSpace(*issuer) == "" {
		return errors.New("roster loader requires absolute manifest, exact target, and issuer")
	}
	manifest, manifestDigest, err := qualificationbootstrap.ReadRosterManifest(*manifestPath, *digest, *target)
	if err != nil {
		return err
	}
	if command == "validate" {
		_, err := fmt.Fprintf(output, "roster manifest validated: target=%s accounts=%d manifest=%s\n", manifest.Target, len(manifest.Accounts), manifestDigest)
		return err
	}
	databaseURL, err := readSecretFile(*databaseURLFile)
	if err != nil {
		return err
	}
	provider, err := identity.NewFirstPartyAdminClient(identity.FirstPartyAdminConfig{BaseURL: os.Getenv("AVIA_AUTH_ADMIN_URL"), BootstrapSecretFile: os.Getenv("AVIA_AUTH_BOOTSTRAP_SECRET_FILE"), Target: *target})
	if err != nil {
		return err
	}
	pool, err := database.Open(ctx, databaseURL)
	if err != nil {
		return fmt.Errorf("open roster database: %w", err)
	}
	defer pool.Close()
	if err := qualificationbootstrap.LoadRoster(ctx, pool, provider, manifest, manifestDigest, *target, *issuer, *secretDirectory, actorSubjectID, time.Now().UTC()); err != nil {
		return err
	}
	_, err = fmt.Fprintf(output, "roster reconciled: target=%s accounts=%d mode=%s\n", manifest.Target, len(manifest.Accounts), manifest.OnboardingMode)
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
