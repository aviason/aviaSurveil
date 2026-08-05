package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"

	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/platform/database"
	workspace "github.com/MarlonJD/aviaSurveil360/apps/api/internal/preproddata/agademoworkspace"
)

func main() {
	if err := run(context.Background(), os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "ERR_AGA_WORKSPACE_ROLE_PROVISION", controlledError(err))
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string, output io.Writer) error {
	flags := flag.NewFlagSet("preprod-aga-demo-workspace-role-provisioner", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	commandURL := flags.String("database-url", os.Getenv("AVIA_AGA_DEMO_WORKSPACE_OWNER_DATABASE_URL"), "bootstrap database URL")
	if err := flags.Parse(args); err != nil || flags.NArg() > 1 {
		return errors.New("usage: provision [--database-url URL]|revoke [--database-url URL]")
	}
	command := "provision"
	if flags.NArg() == 1 {
		command = flags.Arg(0)
	}
	if *commandURL == "" {
		return errors.New("workspace bootstrap database URL is required")
	}
	passwords, err := passwordsFromEnvironment()
	if err != nil {
		return err
	}
	pool, err := database.Open(ctx, *commandURL)
	if err != nil {
		return fmt.Errorf("open workspace bootstrap database: %w", err)
	}
	defer pool.Close()
	switch command {
	case "provision":
		if err := workspace.ProvisionWorkspaceSchema(ctx, pool, passwords); err != nil {
			return err
		}
		_, err := fmt.Fprintln(output, "AGA demo workspace schema and role contract provisioned")
		return err
	case "revoke":
		if err := workspace.RevokeWorkspaceOneShotLogins(ctx, pool); err != nil {
			return err
		}
		_, err := fmt.Fprintln(output, "AGA demo workspace one-shot logins revoked")
		return err
	default:
		return errors.New("unknown workspace role provisioner command")
	}
}

func passwordsFromEnvironment() (workspace.WorkspacePasswords, error) {
	values := workspace.WorkspacePasswords{Exporter: os.Getenv("AVIA_AGA_DEMO_WORKSPACE_EXPORTER_PASSWORD"), Loader: os.Getenv("AVIA_AGA_DEMO_WORKSPACE_LOADER_PASSWORD"), Reader: os.Getenv("AVIA_AGA_DEMO_WORKSPACE_READER_PASSWORD"), Command: os.Getenv("AVIA_AGA_DEMO_WORKSPACE_COMMAND_PASSWORD")}
	if err := values.Validate(); err != nil {
		return workspace.WorkspacePasswords{}, errors.New("workspace role passwords must be supplied through private secret mounts")
	}
	return values, nil
}

func databaseURL(role, password, host, databaseName string) string {
	return (&url.URL{Scheme: "postgres", User: url.UserPassword(role, password), Host: host, Path: databaseName, RawQuery: "sslmode=disable"}).String()
}

func controlledError(err error) string {
	if err == nil {
		return ""
	}
	if strings.Contains(err.Error(), "workspace") {
		return "contract"
	}
	return "failed"
}
