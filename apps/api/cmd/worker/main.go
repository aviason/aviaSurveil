package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/aviason/aviaSurveil/internal/administration"
	"github.com/aviason/aviaSurveil/internal/application"
	"github.com/aviason/aviaSurveil/internal/documents"
	"github.com/aviason/aviaSurveil/internal/identity"
	"github.com/aviason/aviaSurveil/internal/notifications"
	"github.com/aviason/aviaSurveil/internal/platform/config"
	"github.com/aviason/aviaSurveil/internal/platform/database"
	"github.com/aviason/aviaSurveil/internal/platform/objectstore"
	"github.com/aviason/aviaSurveil/internal/platform/scanner"
	"github.com/aviason/aviaSurveil/internal/platform/telemetry"
	evidenceworker "github.com/aviason/aviaSurveil/internal/worker/evidence"
	"github.com/aviason/aviaSurveil/migrations"
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
	if settings.Environment != "local-preprod" && settings.RuntimeProfile != "aws-private-pilot" {
		if err := migrations.Apply(ctx, pool); err != nil {
			return err
		}
	}
	readiness := database.Readiness{Pool: pool, RequiredMigrationVersion: migrations.LatestVersion}
	if err := readiness.Ready(ctx); err != nil {
		return fmt.Errorf("worker migration precondition: %w", err)
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

	var objects objectstore.Store
	var evidenceProcessor scanProcessor
	var documentProcessor scanProcessor
	if settings.ObjectStoreMode != "" || settings.ObjectStoreEndpoint != "" {
		switch settings.ObjectStoreMode {
		case "aws-s3":
			objects, err = objectstore.NewAWSStore(objectstore.AWSConfig{
				Region: settings.ObjectStoreRegion, HealthBucket: settings.QuarantineBucket,
			})
		case "", "minio":
			var localObjects *objectstore.MinIOStore
			localObjects, err = objectstore.NewMinIOStore(objectstore.MinIOConfig{
				Endpoint: settings.ObjectStoreEndpoint, PublicEndpoint: settings.ObjectStorePublicEndpoint,
				AccessKey: settings.ObjectStoreAccessKey, SecretKey: settings.ObjectStoreSecretKey,
				UseTLS: settings.ObjectStoreTLS, PublicUseTLS: settings.ObjectStorePublicTLS,
				Region: settings.ObjectStoreRegion, AllowServerManagedCORS: settings.AllowServerManagedCORS,
			})
			objects = localObjects
			if err == nil && settings.Environment != "production" {
				err = localObjects.EnsurePrivateBuckets(ctx, []string{
					settings.QuarantineBucket,
					settings.CanonicalBucket,
					settings.AttachmentBucket,
					settings.DocumentBucket,
				}, settings.ObjectStoreCORSOrigins)
			}
		default:
			err = fmt.Errorf("unsupported AVIA_OBJECT_STORE_MODE %q", settings.ObjectStoreMode)
		}
		if err != nil {
			return err
		}
		contentScanner, scannerErr := newEvidenceScanner(settings)
		if scannerErr != nil {
			return scannerErr
		}
		var managedResultProvider scanner.ManagedResultProvider
		if settings.ScannerMode == "guardduty-s3" {
			exactObjects, ok := objects.(objectstore.ExactVersionStore)
			if !ok {
				return errors.New("GuardDuty S3 scanning requires exact-version object storage")
			}
			managedResultProvider, err = scanner.NewGuardDutyS3ResultProvider(exactObjects, nil)
			if err != nil {
				return err
			}
		}
		if contentScanner != nil || managedResultProvider != nil {
			evidenceProcessor = evidenceworker.New(pool, objects, contentScanner, evidenceworker.Config{
				WorkerID: "evidence-worker", CanonicalBucket: settings.CanonicalBucket,
				AttachmentBucket: settings.AttachmentBucket, LeaseDuration: time.Minute,
				ManagedResultProvider: managedResultProvider, ScanBackend: settings.ScannerMode,
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

	reminderWorkflow := application.NewCommunicationsWorkflow(
		pool,
		application.CommunicationsWorkflowDependencies{Clock: time.Now},
	)
	workerContext, cancelWorker := context.WithCancel(ctx)
	defer cancelWorker()
	var workerWaitGroup sync.WaitGroup
	workerWaitGroup.Add(1)
	go func() {
		defer workerWaitGroup.Done()
		runReminderController(workerContext, reminderControllerConfig{
			Interval: time.Minute,
			Deadline: 45 * time.Second,
			Schedule: reminderWorkflow,
		})
	}()
	processorErrors := make(chan error, 1)
	workerWaitGroup.Add(1)
	go func() {
		defer workerWaitGroup.Done()
		ticker := time.NewTicker(settings.WorkerInterval)
		defer ticker.Stop()
		for {
			select {
			case <-workerContext.Done():
				return
			case <-ticker.C:
				if err := readiness.Ready(workerContext); err != nil {
					processorErrors <- fmt.Errorf("worker dependency check: %w", err)
					return
				}
				if identityWorker != nil {
					processed, err := processAvailableInstrumented(workerContext, telemetryRuntime, identityWorker)
					if err != nil {
						slog.Error("identity lifecycle work batch failed", "processed", processed, "error", err)
					} else if processed > 0 {
						slog.Info("identity lifecycle work batch completed", "processed", processed)
					}
				}
				if notificationProcessor != nil {
					processed, err := processAvailableInstrumented(workerContext, telemetryRuntime, notificationProcessor)
					if err != nil {
						slog.Error("notification delivery work batch failed", "processed", processed, "errorCode", notifications.DeliveryFailureCode(err))
					} else if processed > 0 {
						slog.Info("notification delivery work batch completed", "processed", processed)
					}
				}
				if objects == nil {
					continue
				}
				if err := objects.Check(workerContext); err != nil {
					processorErrors <- fmt.Errorf("worker object-store check: %w", err)
					return
				}
				processed := 0
				if evidenceProcessor != nil {
					var batchErr error
					processed, batchErr = processAvailableInstrumented(workerContext, telemetryRuntime, evidenceProcessor)
					if batchErr != nil {
						slog.Error("scan work batch failed", "processed", processed, "error", batchErr)
						continue
					}
				}
				rendered := 0
				if documentProcessor != nil {
					var batchErr error
					rendered, batchErr = processAvailableInstrumented(workerContext, telemetryRuntime, documentProcessor)
					if batchErr != nil {
						slog.Error("document work batch failed", "processed", rendered, "error", batchErr)
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
	}()

	var processorErr error
	for {
		select {
		case <-ctx.Done():
			cancelWorker()
			workerWaitGroup.Wait()
			return nil
		case processorErr = <-processorErrors:
			cancelWorker()
			workerWaitGroup.Wait()
			return processorErr
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
		Transport:      settings.SMTPTransport,
		TLSServerName:  settings.SMTPTLSServerName,
	})
	if err != nil {
		return nil, fmt.Errorf("configure SMTP notification delivery: %w", err)
	}
	return sender, nil
}

func newDocumentRenderer(settings config.Settings) (documents.Renderer, error) {
	return documents.NewNativeRenderer()
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
	case "guardduty-s3":
		return nil, nil
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
