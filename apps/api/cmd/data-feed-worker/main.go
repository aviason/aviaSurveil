package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/datafeed"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/platform/config"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/platform/database"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/platform/telemetry"
	"github.com/MarlonJD/aviaSurveil360/apps/api/migrations"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := run(ctx); err != nil {
		slog.Error("data feed worker stopped", "error_code", "startup_failed")
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	settings, err := config.LoadScheduler(os.LookupEnv)
	if err != nil {
		return fmt.Errorf("load worker platform configuration: %w", err)
	}
	telemetryRuntime, err := telemetry.NewRuntime(ctx, telemetry.Config{ServiceName: "data-feed-worker", ServiceVersion: "candidate", Environment: settings.Environment, OTLPHTTPEndpoint: settings.OTLPHTTPEndpoint})
	if err != nil {
		return fmt.Errorf("configure data feed telemetry: %w", err)
	}
	defer telemetryRuntime.Shutdown(context.Background())
	slog.SetDefault(telemetry.NewJSONLogger(nil, "data-feed-worker"))
	workerConfig, err := datafeed.LoadWorkerConfig(os.LookupEnv)
	if err != nil {
		return err
	}
	payloadKey, err := datafeed.LoadPayloadKeyFile(workerConfig.PayloadKeyFile)
	if err != nil {
		return err
	}
	client, err := datafeed.NewMTLSClient(workerConfig.MTLS)
	if err != nil {
		return err
	}
	pool, err := database.OpenWithTracer(ctx, settings.DatabaseURL, telemetryRuntime.PostgresTracer("data-feed-worker"))
	if err != nil {
		return err
	}
	defer pool.Close()
	if err := migrations.Apply(ctx, pool); err != nil {
		return err
	}
	worker := datafeed.OutboxWorker{
		Source: datafeed.PostgresLeaseSource{
			Pool: pool, TenantID: workerConfig.TenantID, OwningOrganizationID: workerConfig.OwningOrganizationID,
			PayloadKey: payloadKey,
		},
		Publisher: datafeed.Publisher{Client: client},
		Processor: datafeed.ReceiptProcessor{Recorder: datafeed.PostgresDecisionRecorder{Pool: pool}},
		ReplayID:  workerConfig.ReplayID, LeaseDuration: time.Minute, Limit: workerConfig.BatchLimit,
		Observer: dataFeedOutcomeObserver{runtime: telemetryRuntime},
	}
	ticker := time.NewTicker(settings.WorkerInterval)
	defer ticker.Stop()
	for {
		jobContext, span := telemetry.StartPersistedJob(telemetryRuntime.Context(ctx), "", "", "datafeed", "aviacore")
		processed, err := worker.ProcessOnce(jobContext)
		outcome := "succeeded"
		if err != nil {
			outcome = "failed"
		}
		telemetryRuntime.RecordDataFeedItems(jobContext, processed, outcome)
		telemetry.FinishPersistedJob(jobContext, span, "datafeed", "aviacore", err)
		if err != nil {
			slog.Error("data feed delivery pass failed", "error_code", "delivery_pass_failed")
		}
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
	}
}

type dataFeedOutcomeObserver struct{ runtime *telemetry.Runtime }

func (observer dataFeedOutcomeObserver) RecordOutcome(ctx context.Context, outcome string, count int) {
	observer.runtime.RecordDataFeedItems(ctx, count, outcome)
}
func (observer dataFeedOutcomeObserver) RecordPendingAge(ctx context.Context, age time.Duration) {
	observer.runtime.RecordOutboxReadyAge(ctx, "datafeed", "aviacore", "ready", age)
}
func (observer dataFeedOutcomeObserver) RecordAcknowledgementLag(ctx context.Context, age time.Duration) {
	observer.runtime.RecordOutboxReadyAge(ctx, "datafeed", "aviacore", "acknowledged", age)
}

func process(ctx context.Context, worker datafeed.OutboxWorker) error {
	_, err := worker.ProcessOnce(ctx)
	return err
}
