package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/application"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/platform/config"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/platform/database"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/platform/telemetry"
	"github.com/MarlonJD/aviaSurveil360/apps/api/migrations"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

func main() {
	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()
	if err := run(ctx); err != nil {
		slog.Error("reminder scheduler stopped", "error", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	settings, err := config.LoadScheduler(os.LookupEnv)
	if err != nil {
		return fmt.Errorf("load configuration: %w", err)
	}
	telemetryRuntime, err := telemetry.NewRuntime(ctx, telemetry.Config{
		ServiceName:      "scheduler",
		ServiceVersion:   "candidate",
		Environment:      settings.Environment,
		OTLPHTTPEndpoint: settings.OTLPHTTPEndpoint,
	})
	if err != nil {
		return fmt.Errorf("configure telemetry: %w", err)
	}
	slog.SetDefault(telemetry.NewJSONLogger(nil, "scheduler"))
	defer func() {
		if shutdownErr := telemetryRuntime.Shutdown(context.Background()); shutdownErr != nil {
			slog.Warn("telemetry shutdown incomplete", "errorClass", telemetry.ErrorClass(shutdownErr))
		}
	}()
	pool, err := database.OpenWithTracer(
		ctx,
		settings.DatabaseURL,
		telemetryRuntime.PostgresTracer("scheduler"),
	)
	if err != nil {
		return err
	}
	defer pool.Close()
	if settings.Environment != "local-preprod" {
		if err := migrations.Apply(ctx, pool); err != nil {
			return err
		}
	}
	jobContext, span := telemetryRuntime.StartJob(
		ctx,
		nil,
		"reminder",
		"postgresql",
	)
	processed, err := application.NewCommunicationsWorkflow(
		pool,
		application.CommunicationsWorkflowDependencies{},
	).ScheduleDueReminders(jobContext)
	if err != nil {
		span.SetStatus(codes.Error, telemetry.ErrorClass(err))
		span.SetAttributes(attribute.String("outcome.class", "failed"))
		telemetryRuntime.RecordJobAttempt(jobContext, "reminder", "postgresql", "failed")
		span.End()
		return err
	}
	span.SetAttributes(attribute.String("outcome.class", "succeeded"))
	telemetryRuntime.RecordJobAttempt(jobContext, "reminder", "postgresql", "succeeded")
	span.End()
	slog.Info("reminder schedule completed", "processed", processed)
	return nil
}
