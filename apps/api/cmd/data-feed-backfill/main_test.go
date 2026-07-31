package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadBackfillRequestRejectsUnknownFieldsAndDoesNotAcceptEventTimestamps(t *testing.T) {
	path := filepath.Join(t.TempDir(), "backfill-request.json")
	valid := `{
  "run_id":"10000000-0000-4000-8000-000000000131",
  "approval_id":"10000000-0000-4000-8000-000000000132",
  "tenant_id":"tenant-task6",
  "owning_organization_id":"organization-task6",
  "source_system":"aviasurveil-production-api",
  "contract_version":"3.0.0",
  "source_cut_id":"2026-07-29-source-consistent-cut",
  "source_manifest_sha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
  "cut_at":"2026-07-29T10:00:00Z",
  "requested_at":"2026-07-30T10:00:00Z",
  "event_ids":["10000000-0000-4000-8000-000000000133"]
}`
	if err := os.WriteFile(path, []byte(valid), 0o600); err != nil {
		t.Fatal(err)
	}
	request, err := loadBackfillRequest(path)
	if err != nil || request.RunID != "10000000-0000-4000-8000-000000000131" || len(request.EventIDs) != 1 {
		t.Fatalf("load request=%+v err=%v", request, err)
	}
	if err := os.WriteFile(path, []byte(valid[:len(valid)-2]+`,"occurred_at":"2026-07-01T00:00:00Z"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := loadBackfillRequest(path); err == nil {
		t.Fatal("backfill request accepted an event timestamp override")
	}
}
