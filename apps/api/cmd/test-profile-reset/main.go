package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/aviason/aviaSurveil/internal/platform/config"
	"github.com/aviason/aviaSurveil/internal/platform/database"
	"github.com/aviason/aviaSurveil/internal/platform/objectstore"
	"github.com/aviason/aviaSurveil/internal/testprofile"
	"github.com/aviason/aviaSurveil/migrations"
)

func main() {
	if err := run(context.Background()); err != nil {
		slog.Error("test-profile reset failed", "error", err)
		os.Exit(1)
	}
	fmt.Println("Canonical test profile reset: verified locally")
}

func run(ctx context.Context) error {
	settings, err := config.Load(os.LookupEnv)
	if err != nil {
		return fmt.Errorf("load configuration: %w", err)
	}
	if settings.Environment != "test" || !settings.CanonicalSeed || !settings.CanonicalTestProfile {
		return fmt.Errorf(
			"out-of-process reset requires the explicit canonical test profile",
		)
	}
	pool, err := database.Open(ctx, settings.DatabaseURL)
	if err != nil {
		return fmt.Errorf("open test database: %w", err)
	}
	defer pool.Close()
	if err := migrations.Apply(ctx, pool); err != nil {
		return fmt.Errorf("apply test migrations: %w", err)
	}
	objects, err := objectstore.NewMinIOStore(objectstore.MinIOConfig{
		Endpoint: settings.ObjectStoreEndpoint, PublicEndpoint: settings.ObjectStorePublicEndpoint,
		AccessKey: settings.ObjectStoreAccessKey, SecretKey: settings.ObjectStoreSecretKey,
		UseTLS: settings.ObjectStoreTLS, PublicUseTLS: settings.ObjectStorePublicTLS,
		Region: settings.ObjectStoreRegion, AllowServerManagedCORS: settings.AllowServerManagedCORS,
	})
	if err != nil {
		return fmt.Errorf("open test object store: %w", err)
	}
	buckets := []string{
		settings.QuarantineBucket,
		settings.CanonicalBucket,
		settings.AttachmentBucket,
		settings.DocumentBucket,
	}
	if err := objects.EnsurePrivateBuckets(ctx, buckets, settings.ObjectStoreCORSOrigins); err != nil {
		return fmt.Errorf("prepare private test buckets: %w", err)
	}
	if err := objects.ResetPrivateBuckets(ctx, buckets); err != nil {
		return fmt.Errorf("reset private test buckets: %w", err)
	}
	if err := testprofile.Reset(ctx, pool, testprofile.CanonicalScenarioTime()); err != nil {
		return fmt.Errorf("reset canonical database state: %w", err)
	}
	return nil
}
