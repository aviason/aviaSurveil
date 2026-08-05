package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	workspace "github.com/MarlonJD/aviaSurveil360/apps/api/internal/preproddata/agademoworkspace"
)

func main() {
	if err := run(context.Background(), os.Args[1:], os.Stdout, nil); err != nil {
		fmt.Fprintln(os.Stderr, "ERR_AGA_WORKSPACE_FIXTURE", controlledError(err))
		os.Exit(1)
	}
}

type accountDocument struct {
	Accounts []workspace.FixtureAccount `json:"accounts"`
}

func run(ctx context.Context, args []string, output io.Writer, source workspace.FixtureSource) error {
	if len(args) == 0 || (args[0] != "export" && args[0] != "verify") {
		return errors.New("usage: export|verify --template PATH --manifest PATH --target-digest DIGEST --base-run-id ID --provider-catalog-digest DIGEST [--accounts PATH]")
	}
	command := args[0]
	flags := flag.NewFlagSet("preprod-aga-demo-workspace-fixture-exporter", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	templatePath := flags.String("template", "", "tracked fixture template")
	manifestPath := flags.String("manifest", "", "private fixture manifest")
	targetDigest := flags.String("target-digest", "", "target fingerprint digest")
	baseRunID := flags.String("base-run-id", "", "predecessor run ID")
	providerDigest := flags.String("provider-catalog-digest", "", "provider catalog digest")
	accountsPath := flags.String("accounts", "", "private exact account snapshot")
	if err := flags.Parse(args[1:]); err != nil {
		return errors.New("invalid fixture exporter arguments")
	}
	template, err := workspace.LoadFixtureTemplate(*templatePath)
	if err != nil {
		return err
	}
	if source == nil {
		if *accountsPath == "" {
			return errors.New("fixture exporter requires an exact private account snapshot")
		}
		document, readErr := readAccounts(*accountsPath)
		if readErr != nil {
			return readErr
		}
		source = workspace.FixtureSourceFunc(func(context.Context, []string) ([]workspace.FixtureAccount, error) { return document.Accounts, nil })
	}
	switch command {
	case "export":
		manifest, err := workspace.ExportFixture(ctx, template, source, "aga-ws-fixture-export", *targetDigest, *baseRunID, *providerDigest, nowUTC())
		if err != nil {
			return err
		}
		if err := workspace.WriteFixtureManifest(*manifestPath, manifest); err != nil {
			return err
		}
		_, err = fmt.Fprintln(output, "AGA demo workspace fixture exported")
		return err
	case "verify":
		manifest, err := workspace.LoadFixtureManifest(*manifestPath)
		if err != nil {
			return err
		}
		if err := workspace.VerifyFixture(ctx, template, source, manifest); err != nil {
			return err
		}
		_, err = fmt.Fprintln(output, "AGA demo workspace fixture verified")
		return err
	default:
		return errors.New("unknown fixture exporter command")
	}
}

func readAccounts(path string) (accountDocument, error) {
	info, err := os.Stat(path)
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm() != 0o600 {
		return accountDocument{}, errors.New("account snapshot must be a private 0600 file")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return accountDocument{}, err
	}
	var document accountDocument
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&document); err != nil {
		return accountDocument{}, errors.New("account snapshot is invalid")
	}
	return document, nil
}

func nowUTC() time.Time            { return time.Now().UTC() }
func controlledError(error) string { return "contract" }
