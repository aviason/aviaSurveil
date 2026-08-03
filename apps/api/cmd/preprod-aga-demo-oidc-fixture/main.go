package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/platform/database"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/preproddata/profiles"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/preproddata/scenarios"
)

func main() {
	if err := run(context.Background()); err != nil {
		slog.Error("preprod AGA demo OIDC qualification fixture failed", "error", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	runID := strings.TrimSpace(os.Getenv("AVIA_PREPROD_RUN_ID"))
	issuer := strings.TrimSpace(os.Getenv("AVIA_PREPROD_OIDC_ISSUER_URL"))
	if runID == "" || issuer == "" {
		return fmt.Errorf("exact predecessor run and OIDC issuer are required")
	}
	ownerPassword, err := readSecret("/run/secrets/preprod_app_database_password")
	if err != nil {
		return err
	}
	serviceSecret, err := readSecret("/run/secrets/preprod_keycloak_service_client_secret")
	if err != nil {
		return err
	}
	password, err := readSecret("/run/secrets/preprod_aga_demo_oidc_qualification_password")
	if err != nil {
		return err
	}
	profile, err := profiles.Lookup("smoke", "1.0.0")
	if err != nil {
		return err
	}
	pool, err := database.Open(ctx, databaseURL("aviasurveil360_preprod_loader", ownerPassword))
	if err != nil {
		return fmt.Errorf("open qualification PostgreSQL target: %w", err)
	}
	defer pool.Close()
	store, err := scenarios.NewPostgresStore(pool, profile, runID)
	if err != nil {
		return err
	}
	records, err := store.Records(ctx, "providerAccounts")
	if err != nil {
		return fmt.Errorf("read exact predecessor provider accounts: %w", err)
	}
	accounts, err := scenarios.QualificationAccounts(records)
	if err != nil {
		return err
	}
	keycloak, err := scenarios.NewKeycloakEndpoint(scenarios.KeycloakEndpointConfig{
		BaseURL: "http://preprod-keycloak:8080/identity", Realm: "aviasurveil360-local-preprod",
		ClientID: "aviasurveil360-local-preprod-lifecycle", ClientSecret: serviceSecret,
	})
	if err != nil {
		return err
	}
	if err := keycloak.ReconcileProviderAccounts(ctx, accounts); err != nil {
		return fmt.Errorf("preflight exact predecessor Keycloak accounts: %w", err)
	}
	if err := scenarios.ActivateQualificationAccounts(ctx, pool, accounts, issuer, time.Now()); err != nil {
		return err
	}
	if err := keycloak.QualifyExistingProviderAccounts(ctx, accounts, password); err != nil {
		return err
	}
	fmt.Printf("Disposable predecessor OIDC qualification verified: accounts=%d roleFamilies=%d\n", len(accounts), 8)
	return nil
}

func databaseURL(role, password string) string {
	return (&url.URL{Scheme: "postgres", User: url.UserPassword(role, password), Host: net.JoinHostPort("preprod-postgres", "5432"), Path: "aviasurveil360_local_preprod", RawQuery: "sslmode=disable"}).String()
}

func readSecret(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read private qualification secret: %w", err)
	}
	value := strings.TrimSpace(string(data))
	if value == "" {
		return "", fmt.Errorf("private qualification secret is empty")
	}
	return value, nil
}
