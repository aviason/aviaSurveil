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

const actorSubjectID = "avia-bootstrap"

func main() {
	if err := run(context.Background(), os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "curated foundation loader:", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string, output io.Writer) error {
	if len(args) == 0 || (args[0] != "validate" && args[0] != "load") {
		return errors.New("usage: curated-foundation-loader validate|load")
	}
	command := args[0]
	flags := flag.NewFlagSet("curated-foundation-loader", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	manifestPath := flags.String("manifest", os.Getenv("AVIA_FOUNDATION_MANIFEST_PATH"), "absolute foundation manifest path")
	digest := flags.String("manifest-sha256", os.Getenv("AVIA_FOUNDATION_MANIFEST_SHA256"), "release-pinned manifest digest")
	target := flags.String("target", os.Getenv("AVIA_BOOTSTRAP_TARGET"), "exact target")
	databaseURLFile := flags.String("database-url-file", os.Getenv("AVIA_DATABASE_URL_FILE"), "private database URL file")
	if err := flags.Parse(args[1:]); err != nil {
		return errors.New("invalid foundation loader arguments")
	}
	if !filepath.IsAbs(*manifestPath) || strings.TrimSpace(*target) == "" {
		return errors.New("foundation loader requires absolute manifest and exact target")
	}
	manifest, manifestDigest, err := qualificationbootstrap.ReadFoundationManifest(*manifestPath, *digest, *target)
	if err != nil {
		return err
	}
	if command == "validate" {
		_, err := fmt.Fprintf(output, "foundation manifest validated: target=%s manifest=%s\n", manifest.Target, manifestDigest)
		return err
	}
	databaseURL, err := readSecretFile(*databaseURLFile)
	if err != nil {
		return err
	}
	pool, err := database.Open(ctx, databaseURL)
	if err != nil {
		return fmt.Errorf("open foundation database: %w", err)
	}
	defer pool.Close()
	if err := qualificationbootstrap.LoadFoundation(ctx, pool, manifest, manifestDigest, actorSubjectID, time.Now().UTC()); err != nil {
		return err
	}
	_, err = fmt.Fprintf(output, "foundation reconciled: target=%s organizations=2 providerScope=1 regulatedTarget=1\n", manifest.Target)
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
