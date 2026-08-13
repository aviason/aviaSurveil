package scanner

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/aviason/aviaSurveil/internal/platform/objectstore"
)

type exactStoreFixture struct {
	tags       map[string]string
	tagResults []map[string]string
	tagReads   int
	info       objectstore.ObjectInfo
	readErr    error
	body       []byte
	expected   objectstore.ExactObject
}

func (store *exactStoreFixture) OpenExact(_ context.Context, expected objectstore.ExactObject) (io.ReadCloser, objectstore.ObjectInfo, error) {
	store.expected = expected
	if store.readErr != nil {
		return nil, objectstore.ObjectInfo{}, store.readErr
	}
	return io.NopCloser(bytes.NewReader(store.body)), store.info, nil
}

func (store *exactStoreFixture) ReadTagsExact(_ context.Context, expected objectstore.ExactObject) (map[string]string, objectstore.ObjectInfo, error) {
	store.expected = expected
	if store.tagReads < len(store.tagResults) {
		tags := store.tagResults[store.tagReads]
		store.tagReads++
		return tags, store.info, store.readErr
	}
	store.tagReads++
	return store.tags, store.info, store.readErr
}

func (store *exactStoreFixture) CopyExact(context.Context, objectstore.ExactCopyRequest) (objectstore.ObjectInfo, error) {
	return objectstore.ObjectInfo{}, errors.New("not used")
}

func managedIdentity(body []byte) objectstore.ExactObject {
	digest := sha256.Sum256(body)
	return objectstore.ExactObject{
		Bucket: "quarantine", Key: "organizations/org/evidence/upload", VersionID: "version-7",
		ETag: "etag-7", SHA256: "sha256:" + hex.EncodeToString(digest[:]), Size: int64(len(body)),
	}
}

func TestGuardDutyS3MapsOnlyExactNoThreatsResultToClean(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, time.August, 10, 10, 30, 0, 0, time.UTC)
	body := []byte("exact clean managed object")
	store := &exactStoreFixture{tags: map[string]string{GuardDutyMalwareScanStatusTag: "NO_THREATS_FOUND"}, body: body}
	provider, err := NewGuardDutyS3ResultProvider(store, func() time.Time { return now })
	if err != nil {
		t.Fatalf("NewGuardDutyS3ResultProvider() error = %v", err)
	}
	identity := managedIdentity(body)
	result, err := provider.Resolve(context.Background(), identity)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if !result.Clean || result.EngineVersion != "guardduty-s3-managed" || result.SignatureVersion != "NO_THREATS_FOUND" || !result.ScannedAt.Equal(now) {
		t.Fatalf("clean managed result = %+v", result)
	}
	if store.expected != identity {
		t.Fatalf("provider read identity = %+v, want %+v", store.expected, identity)
	}
}

func TestGuardDutyS3ThreatsRemainQuarantined(t *testing.T) {
	t.Parallel()
	store := &exactStoreFixture{tags: map[string]string{GuardDutyMalwareScanStatusTag: "THREATS_FOUND"}}
	provider, _ := NewGuardDutyS3ResultProvider(store, nil)
	result, err := provider.Resolve(context.Background(), managedIdentity([]byte("threat")))
	if err != nil || result.Clean || result.Reason == "" {
		t.Fatalf("threat result = %+v, err = %v", result, err)
	}
}

func TestGuardDutyS3MissingMismatchedFailedAndTamperedResultsFailClosed(t *testing.T) {
	t.Parallel()
	tests := map[string]*exactStoreFixture{
		"missing":     {tags: map[string]string{}},
		"unsupported": {tags: map[string]string{GuardDutyMalwareScanStatusTag: "UNSUPPORTED"}},
		"access":      {tags: map[string]string{GuardDutyMalwareScanStatusTag: "ACCESS_DENIED"}},
		"failed":      {tags: map[string]string{GuardDutyMalwareScanStatusTag: "FAILED"}},
		"tampered":    {tags: map[string]string{GuardDutyMalwareScanStatusTag: "no_threats_found"}},
		"mismatched":  {readErr: errors.New("exact object identity mismatch")},
	}
	for name, store := range tests {
		t.Run(name, func(t *testing.T) {
			provider, _ := NewGuardDutyS3ResultProvider(store, nil)
			result, err := provider.Resolve(context.Background(), managedIdentity([]byte("expected")))
			if err == nil || result.Clean {
				t.Fatalf("unsafe result = %+v, error = %v", result, err)
			}
			if strings.Contains(err.Error(), "organizations/org") || strings.Contains(err.Error(), "no_threats_found") {
				t.Fatalf("managed result error leaked object/tag detail: %v", err)
			}
		})
	}
}

func TestGuardDutyS3CleanTagWithMismatchedBytesFailsClosed(t *testing.T) {
	t.Parallel()
	store := &exactStoreFixture{
		tags: map[string]string{GuardDutyMalwareScanStatusTag: "NO_THREATS_FOUND"},
		body: []byte("different bytes"),
	}
	provider, _ := NewGuardDutyS3ResultProvider(store, nil)
	result, err := provider.Resolve(context.Background(), managedIdentity([]byte("expected bytes")))
	if err == nil || result.Clean || strings.Contains(err.Error(), "different bytes") {
		t.Fatalf("mismatched content result = %+v, error = %v", result, err)
	}
}

func TestGuardDutyS3CleanTagChangedDuringVerificationFailsClosed(t *testing.T) {
	t.Parallel()
	body := []byte("exact bytes")
	store := &exactStoreFixture{
		tagResults: []map[string]string{
			{GuardDutyMalwareScanStatusTag: "NO_THREATS_FOUND"},
			{GuardDutyMalwareScanStatusTag: "THREATS_FOUND"},
		},
		body: body,
	}
	provider, _ := NewGuardDutyS3ResultProvider(store, nil)
	result, err := provider.Resolve(context.Background(), managedIdentity(body))
	if err == nil || result.Clean || store.tagReads != 2 {
		t.Fatalf("changed managed result = %+v, tag reads = %d, error = %v", result, store.tagReads, err)
	}
}
