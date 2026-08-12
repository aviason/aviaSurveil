package objectstore_test

import (
	"context"
	"net/url"
	"testing"
	"time"

	"github.com/aviason/aviaSurveil/internal/platform/objectstore"
)

func TestMinIOPutInstructionMeasuresExpiryFromInjectedClock(t *testing.T) {
	scenarioTime := time.Date(2026, time.June, 15, 9, 0, 0, 0, time.UTC)
	store, err := objectstore.NewMinIOStore(objectstore.MinIOConfig{
		Endpoint:  "objects.invalid",
		AccessKey: "access",
		SecretKey: "secret",
		Region:    "us-east-1",
		Clock:     func() time.Time { return scenarioTime },
	})
	if err != nil {
		t.Fatalf("create MinIO-compatible store: %v", err)
	}

	instruction, err := store.CreatePutInstruction(context.Background(), objectstore.PutRequest{
		Bucket:    "quarantine",
		Key:       "inspection-attachments/example.pdf",
		ExpiresAt: scenarioTime.Add(10 * time.Minute),
		RequiredHeaders: map[string]string{
			"Content-Type":      "application/pdf",
			"x-amz-meta-sha256": "sha256:example",
		},
	})
	if err != nil {
		t.Fatalf("create signed PUT from scenario clock: %v", err)
	}
	parsed, err := url.Parse(instruction.URL)
	if err != nil {
		t.Fatalf("parse signed PUT URL: %v", err)
	}
	if got := parsed.Query().Get("X-Amz-Expires"); got != "600" {
		t.Fatalf("signed PUT expiry = %q, want 600", got)
	}
}

func TestMinIOInstructionsUseTheSeparatePublicHTTPSSigner(t *testing.T) {
	scenarioTime := time.Date(2026, time.July, 24, 9, 0, 0, 0, time.UTC)
	store, err := objectstore.NewMinIOStore(objectstore.MinIOConfig{
		Endpoint:       "minio:9000",
		PublicEndpoint: "localhost:8443",
		PublicUseTLS:   true,
		AccessKey:      "access",
		SecretKey:      "secret",
		Region:         "local",
		Clock:          func() time.Time { return scenarioTime },
	})
	if err != nil {
		t.Fatalf("create MinIO-compatible store: %v", err)
	}

	instruction, err := store.CreatePutInstruction(context.Background(), objectstore.PutRequest{
		Bucket: "evidence-quarantine",
		Key:    "organizations/airline-xyz/evidence/finding-1/upload-1",
		RequiredHeaders: map[string]string{
			"Content-Type":      "application/pdf",
			"x-amz-meta-sha256": "sha256:example",
		},
		ExpiresAt: scenarioTime.Add(10 * time.Minute),
	})
	if err != nil {
		t.Fatalf("CreatePutInstruction() error = %v", err)
	}
	parsed, err := url.Parse(instruction.URL)
	if err != nil {
		t.Fatalf("parse signed PUT URL: %v", err)
	}
	if parsed.Scheme != "https" || parsed.Host != "localhost:8443" {
		t.Fatalf("signed PUT origin = %s://%s", parsed.Scheme, parsed.Host)
	}
	if parsed.Path != "/evidence-quarantine/organizations/airline-xyz/evidence/finding-1/upload-1" {
		t.Fatalf("signed PUT path = %q", parsed.Path)
	}
}
