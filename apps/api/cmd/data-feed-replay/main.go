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
		slog.Error("data feed replay stopped", "error_code", "startup_failed")
		os.Exit(1)
	}
}

// run consumes only the already immutable replay lane. It is intentionally a
// bounded one-shot process: a later approved invocation handles scheduled
// retries without turning recovery into an unbounded background exporter.
func run(ctx context.Context) error {
	settings, err := config.LoadScheduler(os.LookupEnv)
	if err != nil {
		return fmt.Errorf("load replay platform configuration: %w", err)
	}
	telemetryRuntime, err := telemetry.NewRuntime(ctx, telemetry.Config{ServiceName: "data-feed-replay", ServiceVersion: "candidate", Environment: settings.Environment, OTLPHTTPEndpoint: settings.OTLPHTTPEndpoint})
	if err != nil {
		return fmt.Errorf("configure data feed replay telemetry: %w", err)
	}
	defer telemetryRuntime.Shutdown(context.Background())
	slog.SetDefault(telemetry.NewJSONLogger(nil, "data-feed-replay"))
	replayConfig, err := datafeed.LoadReplayWorkerConfig(os.LookupEnv)
	if err != nil {
		return err
	}
	payloadKey, err := datafeed.LoadPayloadKeyFile(replayConfig.PayloadKeyFile)
	if err != nil {
		return err
	}
	client, err := datafeed.NewMTLSClient(replayConfig.MTLS)
	if err != nil {
		return err
	}
	pool, err := database.OpenWithTracer(ctx, settings.DatabaseURL, telemetryRuntime.PostgresTracer("data-feed-replay"))
	if err != nil {
		return err
	}
	defer pool.Close()
	if err := migrations.Apply(ctx, pool); err != nil {
		return err
	}
	worker := newReplayWorkerWithObserver(pool, replayConfig, payloadKey, client, replayOutcomeObserver{runtime: telemetryRuntime})
	jobContext, span := telemetry.StartPersistedJob(telemetryRuntime.Context(ctx), "", "", "datafeed", "aviacore")
	processed, err := worker.ProcessOnce(jobContext)
	outcome := "succeeded"
	if err != nil {
		outcome = "failed"
	}
	telemetryRuntime.RecordDataFeedItems(jobContext, processed, outcome)
	telemetry.FinishPersistedJob(jobContext, span, "datafeed", "aviacore", err)
	if err != nil {
		return fmt.Errorf("process replay run once: %w", err)
	}
	return nil
}

func newReplayWorker(pool *database.Pool, config datafeed.ReplayWorkerConfig, payloadKey []byte, client *datafeed.MTLSClient) datafeed.OutboxWorker {
	return newReplayWorkerWithObserver(pool, config, payloadKey, client, nil)
}

func newReplayWorkerWithObserver(pool *database.Pool, config datafeed.ReplayWorkerConfig, payloadKey []byte, client *datafeed.MTLSClient, observer datafeed.OutcomeObserver) datafeed.OutboxWorker {
	return datafeed.OutboxWorker{
		Source: datafeed.PostgresReplayLeaseSource{
			Pool: pool, RunID: config.ReplayRunID, TenantID: config.TenantID,
			OwningOrganizationID: config.OwningOrganizationID, PayloadKey: payloadKey,
		},
		Publisher: datafeed.Publisher{Client: client},
		Processor: datafeed.ReceiptProcessor{Recorder: datafeed.PostgresDecisionRecorder{Pool: pool}},
		ReplayID:  config.ReplayRunID, LeaseDuration: time.Minute, Limit: config.BatchLimit,
		Observer: observer,
	}
}

type replayOutcomeObserver struct{ runtime *telemetry.Runtime }

func (observer replayOutcomeObserver) RecordOutcome(ctx context.Context, outcome string, count int) {
	if observer.runtime != nil {
		observer.runtime.RecordDataFeedItems(ctx, count, outcome)
	}
}
func (observer replayOutcomeObserver) RecordPendingAge(ctx context.Context, age time.Duration) {
	if observer.runtime != nil {
		observer.runtime.RecordOutboxReadyAge(ctx, "datafeed", "aviacore", "ready", age)
	}
}
func (observer replayOutcomeObserver) RecordAcknowledgementLag(ctx context.Context, age time.Duration) {
	if observer.runtime != nil {
		observer.runtime.RecordOutboxReadyAge(ctx, "datafeed", "aviacore", "acknowledged", age)
	}
}
