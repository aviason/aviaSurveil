package main

import (
	"testing"

	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/datafeed"
)

func TestNewReplayWorkerUsesOnlyTheImmutableReplayLane(t *testing.T) {
	config := datafeed.ReplayWorkerConfig{
		WorkerConfig: datafeed.WorkerConfig{
			TenantID: "tenant-task6", OwningOrganizationID: "organization-task6", ReplayID: "10000000-0000-4000-8000-000000000101", BatchLimit: 100,
		},
		ReplayRunID: "10000000-0000-4000-8000-000000000101",
	}
	worker := newReplayWorker(nil, config, nil, nil)
	source, ok := worker.Source.(datafeed.PostgresReplayLeaseSource)
	if !ok || source.RunID != config.ReplayRunID || source.TenantID != config.TenantID || worker.ReplayID != config.ReplayRunID {
		t.Fatalf("replay worker=%+v source=%+v", worker, source)
	}
}
