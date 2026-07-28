package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/administration"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/documents"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/identity"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/notifications"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/platform/config"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/platform/database"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/platform/objectstore"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/platform/scanner"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/platform/telemetry"
	evidenceworker "github.com/MarlonJD/aviaSurveil360/apps/api/internal/worker/evidence"
	"github.com/MarlonJD/aviaSurveil360/apps/api/migrations"
)

type scanProcessor interface {
	ProcessNext(context.Context) (bool, error)
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := run(ctx); err != nil {
		slog.Error("worker stopped", "error", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	settings, err := config.Load(os.LookupEnv)
	if err != nil {
		return fmt.Errorf("load configuration: %w", err)
	}
	telemetryRuntime, err := telemetry.NewRuntime(ctx, telemetry.Config{
		ServiceName:      "worker",
		ServiceVersion:   "candidate",
		Environment:      settings.Environment,
		OTLPHTTPEndpoint: settings.OTLPHTTPEndpoint,
	})
	if err != nil {
		return fmt.Errorf("configure telemetry: %w", err)
	}
	slog.SetDefault(telemetry.NewJSONLogger(nil, "worker"))
	defer func() {
		shutdownContext, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if shutdownErr := telemetryRuntime.Shutdown(shutdownContext); shutdownErr != nil {
			slog.Warn("telemetry shutdown incomplete", "errorClass", telemetry.ErrorClass(shutdownErr))
		}
	}()
	pool, err := database.OpenWithTracer(
		ctx,
		settings.DatabaseURL,
		telemetryRuntime.PostgresTracer("worker"),
	)
	if err != nil {
		return err
	}
	defer pool.Close()
	if err := migrations.Apply(ctx, pool); err != nil {
		return err
	}
	keycloakAdmin, err := newKeycloakAdminClient(settings)
	if err != nil {
		return err
	}
	var identityWorker scanProcessor
	if keycloakAdmin != nil {
		identityWorker = administration.NewUserLifecycleWorker(
			pool,
			keycloakAdmin,
			administration.UserLifecycleWorkerDependencies{
				WorkerID:      "identity-lifecycle-worker",
				LeaseDuration: time.Minute,
				RetryDelay:    5 * time.Second,
				Issuer:        settings.OIDCIssuerURL,
			},
		)
	}
	notificationSender, err := newNotificationSender(settings)
	if err != nil {
		return err
	}
	var notificationProcessor scanProcessor
	if notificationSender != nil {
		notificationProcessor = notifications.NewDeliveryService(
			pool,
			notifications.DeliveryDependencies{
				Adapter:  notificationSender,
				WorkerID: "notification-delivery-worker",
				Lease:    time.Minute,
			},
		)
	}

	var objects *objectstore.MinIOStore
	var evidenceProcessor scanProcessor
	var documentProcessor scanProcessor
	if settings.ObjectStoreEndpoint != "" {
		objects, err = objectstore.NewMinIOStore(objectstore.MinIOConfig{
			Endpoint: settings.ObjectStoreEndpoint, PublicEndpoint: settings.ObjectStorePublicEndpoint,
			AccessKey: settings.ObjectStoreAccessKey, SecretKey: settings.ObjectStoreSecretKey,
			UseTLS: settings.ObjectStoreTLS, PublicUseTLS: settings.ObjectStorePublicTLS,
			Region: settings.ObjectStoreRegion, AllowServerManagedCORS: settings.AllowServerManagedCORS,
		})
		if err != nil {
			return err
		}
		if settings.Environment != "production" {
			if err := objects.EnsurePrivateBuckets(ctx, []string{
				settings.QuarantineBucket,
				settings.CanonicalBucket,
				settings.AttachmentBucket,
				settings.DocumentBucket,
			}, settings.ObjectStoreCORSOrigins); err != nil {
				return err
			}
		}
		contentScanner, scannerErr := newEvidenceScanner(settings)
		if scannerErr != nil {
			return scannerErr
		}
		if contentScanner != nil {
			evidenceProcessor = evidenceworker.New(pool, objects, contentScanner, evidenceworker.Config{
				WorkerID: "evidence-worker", CanonicalBucket: settings.CanonicalBucket,
				AttachmentBucket: settings.AttachmentBucket, LeaseDuration: time.Minute,
			})
		}
		documentRenderer, rendererErr := newDocumentRenderer(settings)
		if rendererErr != nil {
			return rendererErr
		}
		if documentRenderer != nil {
			documentProcessor = documents.NewService(pool, objects, documents.Dependencies{
				Renderer: documentRenderer,
				Bucket:   settings.DocumentBucket, WorkerID: "document-worker",
			})
		}
	}

	readiness := database.Readiness{Pool: pool, RequiredMigrationVersion: migrations.LatestVersion}
	ticker := time.NewTicker(settings.WorkerInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := readiness.Ready(ctx); err != nil {
				return fmt.Errorf("worker dependency check: %w", err)
			}
			if identityWorker != nil {
				processed, err := processAvailableInstrumented(
					ctx,
					telemetryRuntime,
					identityWorker,
				)
				if err != nil {
					slog.Error(
						"identity lifecycle work batch failed",
						"processed",
						processed,
						"error",
						err,
					)
				} else if processed > 0 {
					slog.Info(
						"identity lifecycle work batch completed",
						"processed",
						processed,
					)
				}
			}
			if notificationProcessor != nil {
				processed, err := processAvailableInstrumented(
					ctx,
					telemetryRuntime,
					notificationProcessor,
				)
				if err != nil {
					slog.Error(
						"notification delivery work batch failed",
						"processed",
						processed,
						"errorCode",
						notifications.DeliveryFailureCode(err),
					)
				} else if processed > 0 {
					slog.Info(
						"notification delivery work batch completed",
						"processed",
						processed,
					)
				}
			}
			if objects != nil {
				if err := objects.Check(ctx); err != nil {
					return fmt.Errorf("worker object-store check: %w", err)
				}
				processed := 0
				if evidenceProcessor != nil {
					processed, err = processAvailableInstrumented(
						ctx,
						telemetryRuntime,
						evidenceProcessor,
					)
					if err != nil {
						slog.Error("scan work batch failed", "processed", processed, "error", err)
						continue
					}
				}
				rendered := 0
				if documentProcessor != nil {
					rendered, err = processAvailableInstrumented(
						ctx,
						telemetryRuntime,
						documentProcessor,
					)
					if err != nil {
						slog.Error("document work batch failed", "processed", rendered, "error", err)
						continue
					}
				}
				if processed > 0 {
					slog.Info("scan work batch completed", "processed", processed)
				}
				if rendered > 0 {
					slog.Info("document work batch completed", "processed", rendered)
				}
			}
		}
	}
}

func newNotificationSender(
	settings config.Settings,
) (notifications.DeliveryAdapter, error) {
	if settings.SMTPAddress == "" {
		if settings.Environment == "production" {
			return nil, errors.New(
				"SMTP notification delivery is required by the production worker",
			)
		}
		return nil, nil
	}
	sender, err := notifications.NewSMTPSender(notifications.SMTPConfig{
		Address:        settings.SMTPAddress,
		From:           settings.SMTPFrom,
		Username:       settings.SMTPUsername,
		Password:       settings.SMTPPassword,
		Timeout:        settings.SMTPTimeout,
		PrivateNetwork: settings.SMTPPrivateNetwork,
	})
	if err != nil {
		return nil, fmt.Errorf("configure SMTP notification delivery: %w", err)
	}
	return sender, nil
}

func newDocumentRenderer(settings config.Settings) (documents.Renderer, error) {
	if settings.ScannerMode == "deterministic-test" {
		if settings.Environment == "production" {
			return nil, errors.New(
				"deterministic document renderer is forbidden in production",
			)
		}
		return documents.DeterministicPDFRenderer{}, nil
	}
	if settings.GotenbergURL == "" {
		if settings.Environment == "production" {
			return nil, errors.New(
				"Gotenberg document rendering is required by the production worker",
			)
		}
		return nil, nil
	}
	renderer, err := documents.NewGotenbergRenderer(documents.GotenbergConfig{
		BaseURL: settings.GotenbergURL, Timeout: settings.GotenbergTimeout,
		RendererHash: settings.GotenbergRendererHash,
	})
	if err != nil {
		return nil, fmt.Errorf("configure Gotenberg document renderer: %w", err)
	}
	return renderer, nil
}

func newEvidenceScanner(settings config.Settings) (evidenceworker.Scanner, error) {
	switch settings.ScannerMode {
	case "":
		return nil, nil
	case "deterministic-test":
		if settings.Environment == "production" {
			return nil, errors.New("deterministic Evidence scanner is forbidden in production")
		}
		return scanner.SignatureScanner{}, nil
	case "clamav":
		return scanner.NewClamAV(scanner.ClamAVConfig{
			Address:             settings.ClamAVAddress,
			MaximumSignatureAge: settings.ClamAVMaximumSignatureAge,
		})
	default:
		return nil, fmt.Errorf("unsupported AVIA_SCANNER_MODE %q", settings.ScannerMode)
	}
}

func newKeycloakAdminClient(
	settings config.Settings,
) (*identity.KeycloakAdminClient, error) {
	if settings.KeycloakAdminURL == "" {
		if settings.Environment == "production" {
			return nil, fmt.Errorf(
				"Keycloak administration is required by the production worker",
			)
		}
		return nil, nil
	}
	client, err := identity.NewKeycloakAdminClient(identity.KeycloakAdminConfig{
		BaseURL: settings.KeycloakAdminURL, Realm: settings.KeycloakRealm,
		ClientID:     settings.KeycloakServiceClientID,
		ClientSecret: settings.KeycloakServiceClientSecret,
	})
	if err != nil {
		return nil, fmt.Errorf("configure Keycloak administration: %w", err)
	}
	return client, nil
}

func processAvailable(ctx context.Context, processor scanProcessor) (int, error) {
	processedCount := 0
	for {
		processed, err := processor.ProcessNext(ctx)
		if err != nil {
			return processedCount, err
		}
		if !processed {
			return processedCount, nil
		}
		processedCount++
	}
}

func processAvailableInstrumented(
	ctx context.Context,
	runtime *telemetry.Runtime,
	processor scanProcessor,
) (int, error) {
	return processAvailable(runtime.Context(ctx), processor)
}
