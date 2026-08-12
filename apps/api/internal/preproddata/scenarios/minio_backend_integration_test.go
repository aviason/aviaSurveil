//go:build integration

package scenarios_test

import (
	"context"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/aviason/aviaSurveil/internal/platform/objectstore"
	"github.com/aviason/aviaSurveil/internal/preproddata/scenarios"
)

func TestMinIOObjectBackendPersistsAndReconcilesExactScenarioJSON(
	t *testing.T,
) {
	endpointAddress := os.Getenv("AVIA_TEST_OBJECT_STORE_ENDPOINT")
	accessKey := os.Getenv("AVIA_TEST_OBJECT_STORE_ACCESS_KEY")
	secretKey := os.Getenv("AVIA_TEST_OBJECT_STORE_SECRET_KEY")
	if endpointAddress == "" || accessKey == "" || secretKey == "" {
		t.Skip("requires the task-owned MinIO integration endpoint")
	}
	ctx := context.Background()
	const bucket = "task7-connected-scenarios"
	store, err := objectstore.NewMinIOStore(objectstore.MinIOConfig{
		Endpoint:               endpointAddress,
		AccessKey:              accessKey,
		SecretKey:              secretKey,
		AllowServerManagedCORS: true,
	})
	if err != nil {
		t.Fatalf("new MinIO store: %v", err)
	}
	if err := store.EnsurePrivateBuckets(
		ctx,
		[]string{bucket},
		[]string{"http://127.0.0.1:4174"},
	); err != nil {
		t.Fatalf("ensure task-owned bucket: %v", err)
	}
	if err := store.ResetPrivateBuckets(ctx, []string{bucket}); err != nil {
		t.Fatalf("reset task-owned bucket before test: %v", err)
	}
	t.Cleanup(func() {
		if err := store.ResetPrivateBuckets(
			context.Background(),
			[]string{bucket},
		); err != nil {
			t.Errorf("reset task-owned bucket after test: %v", err)
		}
	})

	backend, err := scenarios.NewMinIOObjectBackend(
		scenarios.MinIOObjectBackendConfig{
			Endpoint:   endpointAddress,
			AccessKey:  accessKey,
			SecretKey:  secretKey,
			HTTPClient: &http.Client{Timeout: 5 * time.Second},
		},
	)
	if err != nil {
		t.Fatalf("new MinIO scenario backend: %v", err)
	}
	prefix := "runs/integration-" +
		time.Now().UTC().Format("20060102T150405.000000000") + "/"
	objectEndpoint, err := scenarios.NewConnectedObjectEndpoint(
		scenarios.ConnectedObjectEndpointConfig{
			Bucket:  bucket,
			Prefix:  prefix,
			Backend: backend,
		},
	)
	if err != nil {
		t.Fatalf("new connected object endpoint: %v", err)
	}
	if err := objectEndpoint.Preflight(ctx); err != nil {
		t.Fatalf("preflight connected object endpoint: %v", err)
	}
	version := scenarios.ObjectVersion{
		VersionID:      "synthetic-objectversions-0001",
		ObjectID:       "synthetic-objects-0001",
		OrganizationID: "AUDITEE-A",
		Bucket:         bucket,
		Key: prefix + "objects/" +
			"synthetic-objectversions-0001.json",
		ContentDigest: "sha256:a4731174dbb244fa6479a9081a8a11d4c0a182409a12645d9e7ea260990839eb",
		SizeBytes:     195,
		Content: []byte(
			`{"schemaVersion":"preprod-synthetic-object/v1","synthetic":true,"recordId":"synthetic-objectversions-0001","objectId":"synthetic-objects-0001","organizationId":"AUDITEE-A","binaryIncluded":false}`,
		),
	}
	if err := objectEndpoint.EnsureObjectVersion(ctx, version); err != nil {
		t.Fatalf("ensure MinIO object version: %v", err)
	}
	if err := objectEndpoint.ReconcileObjectVersions(
		ctx,
		[]scenarios.ObjectVersion{version},
	); err != nil {
		t.Fatalf("reconcile MinIO object version: %v", err)
	}
}
