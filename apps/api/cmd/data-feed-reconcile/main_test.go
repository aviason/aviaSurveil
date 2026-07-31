package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadManifestRejectsUnknownFields(t *testing.T) {
	path := filepath.Join(t.TempDir(), "producer.json")
	valid := `{
  "run_id":"10000000-0000-4000-8000-000000000141",
  "contract_version":"3.0.0",
  "events":[{
    "event_id":"10000000-0000-4000-8000-000000000142",
    "canonical_event_sha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
    "delivery_outcome":"ACKNOWLEDGED",
    "acknowledgement_receipt_digest":"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
  }]
}`
	if err := os.WriteFile(path, []byte(valid), 0o600); err != nil {
		t.Fatal(err)
	}
	manifest, err := loadManifest(path)
	if err != nil || manifest.RunID != "10000000-0000-4000-8000-000000000141" || len(manifest.Events) != 1 {
		t.Fatalf("load manifest=%+v err=%v", manifest, err)
	}
	if err := os.WriteFile(path, []byte(valid[:len(valid)-2]+`,"payload":"forbidden"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := loadManifest(path); err == nil {
		t.Fatal("reconciliation manifest accepted an unknown/payload field")
	}
}
