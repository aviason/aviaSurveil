package evidenceworker

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/platform/objectstore"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/platform/scanner"
)

type managedProviderFixture struct {
	result   scanner.Result
	err      error
	received objectstore.ExactObject
}

func (provider *managedProviderFixture) Resolve(_ context.Context, expected objectstore.ExactObject) (scanner.Result, error) {
	provider.received = expected
	return provider.result, provider.err
}

type exactWorkerStoreFixture struct {
	copied   objectstore.ExactCopyRequest
	copyInfo objectstore.ObjectInfo
	copyErr  error
}

func (*exactWorkerStoreFixture) CreatePutInstruction(context.Context, objectstore.PutRequest) (objectstore.PutInstruction, error) {
	return objectstore.PutInstruction{}, errors.New("not used")
}
func (*exactWorkerStoreFixture) Write(context.Context, objectstore.WriteRequest) (objectstore.ObjectInfo, error) {
	return objectstore.ObjectInfo{}, errors.New("not used")
}
func (*exactWorkerStoreFixture) Open(context.Context, string, string) (io.ReadCloser, objectstore.ObjectInfo, error) {
	return io.NopCloser(bytes.NewReader(nil)), objectstore.ObjectInfo{}, nil
}
func (*exactWorkerStoreFixture) Copy(context.Context, objectstore.CopyRequest) error {
	return errors.New("legacy copy must not run")
}
func (*exactWorkerStoreFixture) CreateGetInstruction(context.Context, objectstore.GetRequest) (objectstore.GetInstruction, error) {
	return objectstore.GetInstruction{}, errors.New("not used")
}
func (*exactWorkerStoreFixture) Check(context.Context) error { return nil }
func (*exactWorkerStoreFixture) OpenExact(context.Context, objectstore.ExactObject) (io.ReadCloser, objectstore.ObjectInfo, error) {
	return nil, objectstore.ObjectInfo{}, errors.New("not used")
}
func (*exactWorkerStoreFixture) ReadTagsExact(context.Context, objectstore.ExactObject) (map[string]string, objectstore.ObjectInfo, error) {
	return nil, objectstore.ObjectInfo{}, errors.New("not used")
}
func (store *exactWorkerStoreFixture) CopyExact(_ context.Context, request objectstore.ExactCopyRequest) (objectstore.ObjectInfo, error) {
	store.copied = request
	return store.copyInfo, store.copyErr
}

func managedRecord() objectRecord {
	return objectRecord{
		OrganizationID: "org-1", SourceBucket: "quarantine", SourceKey: "organizations/org-1/upload-1",
		SHA256: "sha256:" + strings.Repeat("a", 64), Size: 128, VersionID: "version-1", ETag: "etag-1",
	}
}

func TestManagedResultUsesAndPromotesOnlyTheCapturedExactVersion(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, time.August, 10, 12, 0, 0, 0, time.UTC)
	provider := &managedProviderFixture{result: scanner.Result{Clean: true, EngineVersion: "guardduty-s3-managed", SignatureVersion: "NO_THREATS_FOUND", ScannedAt: now}}
	store := &exactWorkerStoreFixture{copyInfo: objectstore.ObjectInfo{Bucket: "clean", Key: "canonical", VersionID: "canonical-version", ETag: "canonical-etag", Size: 128}}
	worker := New(nil, store, nil, Config{ManagedResultProvider: provider, ScanBackend: "guardduty-s3", ScanTimeout: time.Second})
	record := managedRecord()
	result, err := worker.resolveScan(context.Background(), record)
	if err != nil || !result.Clean {
		t.Fatalf("managed resolve result = %+v, err = %v", result, err)
	}
	expected := objectstore.ExactObject{Bucket: record.SourceBucket, Key: record.SourceKey, VersionID: record.VersionID, ETag: record.ETag, SHA256: record.SHA256, Size: record.Size}
	if provider.received != expected {
		t.Fatalf("managed provider identity = %+v, want %+v", provider.received, expected)
	}
	promoted, err := worker.promote(context.Background(), record, "clean", "canonical")
	if err != nil {
		t.Fatalf("managed promotion error = %v", err)
	}
	if store.copied.Source != expected || store.copied.DestinationBucket != "clean" || store.copied.DestinationKey != "canonical" {
		t.Fatalf("managed copy request = %+v", store.copied)
	}
	if promoted.VersionID != "canonical-version" || promoted.ETag != "canonical-etag" {
		t.Fatalf("managed promotion identity = %+v", promoted)
	}
}

func TestManagedResultRejectsMissingVersionBeforeProviderOrCopy(t *testing.T) {
	t.Parallel()
	provider := &managedProviderFixture{result: scanner.Result{Clean: true}}
	store := &exactWorkerStoreFixture{}
	worker := New(nil, store, nil, Config{ManagedResultProvider: provider, ScanBackend: "guardduty-s3"})
	record := managedRecord()
	record.VersionID = ""
	if _, err := worker.resolveScan(context.Background(), record); err == nil || !strings.Contains(err.Error(), "exact object identity") {
		t.Fatalf("missing version resolve error = %v", err)
	}
	if provider.received != (objectstore.ExactObject{}) || store.copied != (objectstore.ExactCopyRequest{}) {
		t.Fatal("missing exact identity reached a managed external effect")
	}
}
