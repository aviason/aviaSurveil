package scenarios_test

import (
	"context"
	"maps"
	"slices"
	"testing"

	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/preproddata/scenarios"
)

func TestObjectEndpointWritesExactlyOnceAndRejectsContentDrift(
	t *testing.T,
) {
	ctx := context.Background()
	backend := newMemoryObjectBackend()
	endpoint, err := scenarios.NewConnectedObjectEndpoint(
		scenarios.ConnectedObjectEndpointConfig{
			Bucket:  "aviasurveil360-local-preprod",
			Prefix:  "runs/run-task7-connected-smoke/",
			Backend: backend,
		},
	)
	if err != nil {
		t.Fatalf("new object endpoint: %v", err)
	}
	if err := endpoint.Preflight(ctx); err != nil {
		t.Fatalf("preflight object endpoint: %v", err)
	}
	version := scenarios.ObjectVersion{
		VersionID:      "synthetic-objectversions-0001",
		ObjectID:       "synthetic-objects-0001",
		OrganizationID: "AUDITEE-A",
		Bucket:         "aviasurveil360-local-preprod",
		Key: "runs/run-task7-connected-smoke/objects/" +
			"synthetic-objectversions-0001.json",
		ContentDigest: "sha256:a4731174dbb244fa6479a9081a8a11d4c0a182409a12645d9e7ea260990839eb",
		SizeBytes:     195,
		Content: []byte(
			`{"schemaVersion":"preprod-synthetic-object/v1","synthetic":true,"recordId":"synthetic-objectversions-0001","objectId":"synthetic-objects-0001","organizationId":"AUDITEE-A","binaryIncluded":false}`,
		),
	}
	if err := endpoint.EnsureObjectVersion(ctx, version); err != nil {
		t.Fatalf("ensure object version: %v", err)
	}
	if err := endpoint.EnsureObjectVersion(ctx, version); err != nil {
		t.Fatalf("replay object version: %v", err)
	}
	if len(backend.blobs) != 1 {
		t.Fatalf("object count after replay = %d", len(backend.blobs))
	}
	if err := endpoint.ReconcileObjectVersions(
		ctx,
		[]scenarios.ObjectVersion{version},
	); err != nil {
		t.Fatalf("reconcile exact object version: %v", err)
	}
	if err := endpoint.ReconcileObjectVersionStream(
		ctx,
		func(yield func(scenarios.ObjectVersion) error) error {
			return yield(version)
		},
	); err != nil {
		t.Fatalf("stream-reconcile exact object version: %v", err)
	}
	blob := backend.blobs[version.Key]
	blob.Content = []byte(`{"synthetic":false}`)
	backend.blobs[version.Key] = blob
	if err := endpoint.ReconcileObjectVersions(
		ctx,
		[]scenarios.ObjectVersion{version},
	); err == nil {
		t.Fatalf("object content drift was accepted")
	}
}

type memoryObjectBackend struct {
	blobs map[string]scenarios.ObjectBlob
}

func newMemoryObjectBackend() *memoryObjectBackend {
	return &memoryObjectBackend{
		blobs: make(map[string]scenarios.ObjectBlob),
	}
}

func (*memoryObjectBackend) Check(context.Context) error {
	return nil
}

func (backend *memoryObjectBackend) List(
	_ context.Context,
	bucket,
	prefix string,
) ([]scenarios.ObjectBlob, error) {
	output := make([]scenarios.ObjectBlob, 0, len(backend.blobs))
	for _, blob := range backend.blobs {
		if blob.Bucket == bucket && len(blob.Key) >= len(prefix) &&
			blob.Key[:len(prefix)] == prefix {
			output = append(output, cloneObjectBlob(blob))
		}
	}
	slices.SortFunc(output, func(left, right scenarios.ObjectBlob) int {
		switch {
		case left.Key < right.Key:
			return -1
		case left.Key > right.Key:
			return 1
		default:
			return 0
		}
	})
	return output, nil
}

func (backend *memoryObjectBackend) Read(
	_ context.Context,
	bucket,
	key string,
) (scenarios.ObjectBlob, error) {
	blob, ok := backend.blobs[key]
	if !ok || blob.Bucket != bucket {
		return scenarios.ObjectBlob{}, scenarios.ErrScenarioObjectNotFound
	}
	return cloneObjectBlob(blob), nil
}

func (backend *memoryObjectBackend) Scan(
	ctx context.Context,
	bucket,
	prefix string,
	yield func(scenarios.ObjectBlob) error,
) error {
	blobs, err := backend.List(ctx, bucket, prefix)
	if err != nil {
		return err
	}
	for _, blob := range blobs {
		if err := yield(blob); err != nil {
			return err
		}
	}
	return nil
}

func (backend *memoryObjectBackend) Create(
	_ context.Context,
	blob scenarios.ObjectBlob,
) error {
	if _, exists := backend.blobs[blob.Key]; exists {
		return scenarios.ErrScenarioObjectAlreadyExists
	}
	backend.blobs[blob.Key] = cloneObjectBlob(blob)
	return nil
}

func cloneObjectBlob(source scenarios.ObjectBlob) scenarios.ObjectBlob {
	source.Content = append([]byte(nil), source.Content...)
	source.Metadata = maps.Clone(source.Metadata)
	return source
}

var _ scenarios.ObjectBackend = (*memoryObjectBackend)(nil)
