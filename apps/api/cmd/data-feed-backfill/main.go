package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"time"

	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/datafeed"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/platform/config"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/platform/database"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/platform/telemetry"
	"github.com/MarlonJD/aviaSurveil360/apps/api/migrations"
)

const maxBackfillRequestBytes = 1 << 20

type backfillRequestFile struct {
	RunID                string    `json:"run_id"`
	ApprovalID           string    `json:"approval_id"`
	TenantID             string    `json:"tenant_id"`
	OwningOrganizationID string    `json:"owning_organization_id"`
	SourceSystem         string    `json:"source_system"`
	ContractVersion      string    `json:"contract_version"`
	SourceCutID          string    `json:"source_cut_id"`
	SourceManifestDigest string    `json:"source_manifest_sha256"`
	CutAt                time.Time `json:"cut_at"`
	RequestedAt          time.Time `json:"requested_at"`
	EventIDs             []string  `json:"event_ids"`
}

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: data-feed-backfill <approved-backfill-request.json>")
		os.Exit(64)
	}
	if err := run(context.Background(), os.Args[1]); err != nil {
		slog.Error("data feed backfill stopped", "error_code", "backfill_failed")
		os.Exit(1)
	}
}

func run(ctx context.Context, path string) error {
	request, err := loadBackfillRequest(path)
	if err != nil {
		return err
	}
	settings, err := config.LoadScheduler(os.LookupEnv)
	if err != nil {
		return fmt.Errorf("load backfill platform configuration: %w", err)
	}
	telemetryRuntime, err := telemetry.NewRuntime(ctx, telemetry.Config{ServiceName: "data-feed-backfill", ServiceVersion: "candidate", Environment: settings.Environment, OTLPHTTPEndpoint: settings.OTLPHTTPEndpoint})
	if err != nil {
		return fmt.Errorf("configure data feed backfill telemetry: %w", err)
	}
	defer telemetryRuntime.Shutdown(context.Background())
	slog.SetDefault(telemetry.NewJSONLogger(nil, "data-feed-backfill"))
	pool, err := database.OpenWithTracer(ctx, settings.DatabaseURL, telemetryRuntime.PostgresTracer("data-feed-backfill"))
	if err != nil {
		return err
	}
	defer pool.Close()
	if err := migrations.Apply(ctx, pool); err != nil {
		return err
	}
	result, err := (datafeed.PostgresReplayStore{Pool: pool}).CreateBackfillRun(ctx, request)
	if err != nil {
		return err
	}
	slog.Info("data feed backfill run recorded", "run_id", result.RunID, "request_digest", result.RequestDigest, "event_count", result.EventCount)
	return nil
}

func loadBackfillRequest(path string) (datafeed.BackfillRequest, error) {
	file, err := os.Open(path)
	if err != nil {
		return datafeed.BackfillRequest{}, fmt.Errorf("open approved backfill request: %w", err)
	}
	defer file.Close()
	decoder := json.NewDecoder(io.LimitReader(file, maxBackfillRequestBytes+1))
	decoder.DisallowUnknownFields()
	var wire backfillRequestFile
	if err := decoder.Decode(&wire); err != nil {
		return datafeed.BackfillRequest{}, fmt.Errorf("decode approved backfill request: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return datafeed.BackfillRequest{}, fmt.Errorf("decode approved backfill request: trailing JSON")
	}
	request := datafeed.BackfillRequest{
		RunID: wire.RunID, ApprovalID: wire.ApprovalID, TenantID: wire.TenantID,
		OwningOrganizationID: wire.OwningOrganizationID, SourceSystem: wire.SourceSystem,
		ContractVersion: wire.ContractVersion, SourceCutID: wire.SourceCutID,
		SourceManifestDigest: wire.SourceManifestDigest, CutAt: wire.CutAt,
		RequestedAt: wire.RequestedAt, EventIDs: wire.EventIDs,
	}
	if err := datafeed.ValidateBackfillRequest(request); err != nil {
		return datafeed.BackfillRequest{}, err
	}
	return request, nil
}
