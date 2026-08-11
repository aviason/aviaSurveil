package integration_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/platform/objectstore"
)

type memoryObject struct {
	body        []byte
	contentType string
	metadata    map[string]string
	versionID   string
	etag        string
	tags        map[string]string
}

type memoryObjectStore struct {
	mu      sync.Mutex
	objects map[string]memoryObject
	copies  []objectstore.CopyRequest
}

func newMemoryObjectStore() *memoryObjectStore {
	return &memoryObjectStore{objects: map[string]memoryObject{}}
}

func objectLocation(bucket, key string) string { return bucket + "/" + key }

func (store *memoryObjectStore) CreatePutInstruction(_ context.Context, request objectstore.PutRequest) (objectstore.PutInstruction, error) {
	return objectstore.PutInstruction{
		URL:       "memory://" + objectLocation(request.Bucket, request.Key),
		Headers:   cloneStrings(request.RequiredHeaders),
		ExpiresAt: request.ExpiresAt,
	}, nil
}

func (store *memoryObjectStore) Write(_ context.Context, request objectstore.WriteRequest) (objectstore.ObjectInfo, error) {
	body, err := io.ReadAll(request.Body)
	if err != nil {
		return objectstore.ObjectInfo{}, err
	}
	if int64(len(body)) != request.Size {
		return objectstore.ObjectInfo{}, fmt.Errorf("object size %d does not match %d", len(body), request.Size)
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	location := objectLocation(request.Bucket, request.Key)
	if _, exists := store.objects[location]; exists {
		return objectstore.ObjectInfo{}, objectstore.ErrObjectAlreadyExists
	}
	store.objects[location] = memoryObject{
		body: append([]byte(nil), body...), contentType: request.ContentType,
		metadata: cloneStrings(request.Metadata), versionID: fmt.Sprintf("version-%d", len(store.objects)+1),
		etag: sha256Digest(body), tags: map[string]string{},
	}
	stored := store.objects[location]
	return objectstore.ObjectInfo{
		Bucket: request.Bucket, Key: request.Key, Size: request.Size, VersionID: stored.versionID, ETag: stored.etag,
		ContentType: request.ContentType, Metadata: cloneStrings(request.Metadata),
	}, nil
}

func (store *memoryObjectStore) Open(_ context.Context, bucket, key string) (io.ReadCloser, objectstore.ObjectInfo, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	object, ok := store.objects[objectLocation(bucket, key)]
	if !ok {
		return nil, objectstore.ObjectInfo{}, objectstore.ErrObjectNotFound
	}
	copyBody := append([]byte(nil), object.body...)
	return io.NopCloser(bytes.NewReader(copyBody)), objectstore.ObjectInfo{
		Bucket: bucket, Key: key, Size: int64(len(copyBody)), VersionID: object.versionID, ETag: object.etag, ContentType: object.contentType,
		Metadata: cloneStrings(object.metadata),
	}, nil
}

func (store *memoryObjectStore) Copy(_ context.Context, request objectstore.CopyRequest) error {
	store.mu.Lock()
	defer store.mu.Unlock()
	source, ok := store.objects[objectLocation(request.SourceBucket, request.SourceKey)]
	if !ok {
		return objectstore.ErrObjectNotFound
	}
	destination := objectLocation(request.DestinationBucket, request.DestinationKey)
	if _, exists := store.objects[destination]; exists {
		return objectstore.ErrObjectAlreadyExists
	}
	store.objects[destination] = memoryObject{
		body: append([]byte(nil), source.body...), contentType: source.contentType,
		metadata: cloneStrings(source.metadata), versionID: fmt.Sprintf("version-%d", len(store.objects)+1),
		etag: sha256Digest(source.body), tags: map[string]string{},
	}
	store.copies = append(store.copies, request)
	return nil
}

func (store *memoryObjectStore) CreateGetInstruction(_ context.Context, request objectstore.GetRequest) (objectstore.GetInstruction, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	if _, ok := store.objects[objectLocation(request.Bucket, request.Key)]; !ok {
		return objectstore.GetInstruction{}, objectstore.ErrObjectNotFound
	}
	return objectstore.GetInstruction{
		URL: "memory://download/" + objectLocation(request.Bucket, request.Key), ExpiresAt: request.ExpiresAt,
	}, nil
}

func (store *memoryObjectStore) Check(context.Context) error { return nil }

func (store *memoryObjectStore) Seed(bucket, key, contentType string, body []byte, metadata map[string]string) {
	store.mu.Lock()
	defer store.mu.Unlock()
	store.objects[objectLocation(bucket, key)] = memoryObject{
		body: append([]byte(nil), body...), contentType: contentType, metadata: cloneStrings(metadata),
		versionID: fmt.Sprintf("version-%d", len(store.objects)+1), etag: sha256Digest(body), tags: map[string]string{},
	}
}

func (store *memoryObjectStore) SetTags(bucket, key string, tags map[string]string) {
	store.mu.Lock()
	defer store.mu.Unlock()
	location := objectLocation(bucket, key)
	object := store.objects[location]
	object.tags = cloneStrings(tags)
	store.objects[location] = object
}

func (store *memoryObjectStore) OpenExact(ctx context.Context, expected objectstore.ExactObject) (io.ReadCloser, objectstore.ObjectInfo, error) {
	reader, info, err := store.Open(ctx, expected.Bucket, expected.Key)
	if err != nil {
		return nil, objectstore.ObjectInfo{}, err
	}
	if info.VersionID != expected.VersionID || info.ETag != expected.ETag || info.Size != expected.Size || info.Metadata["sha256"] != expected.SHA256 {
		_ = reader.Close()
		return nil, objectstore.ObjectInfo{}, errors.New("exact object identity mismatch")
	}
	return reader, info, nil
}

func (store *memoryObjectStore) ReadTagsExact(ctx context.Context, expected objectstore.ExactObject) (map[string]string, objectstore.ObjectInfo, error) {
	reader, info, err := store.OpenExact(ctx, expected)
	if err != nil {
		return nil, objectstore.ObjectInfo{}, err
	}
	_ = reader.Close()
	store.mu.Lock()
	defer store.mu.Unlock()
	return cloneStrings(store.objects[objectLocation(expected.Bucket, expected.Key)].tags), info, nil
}

func (store *memoryObjectStore) CopyExact(ctx context.Context, request objectstore.ExactCopyRequest) (objectstore.ObjectInfo, error) {
	reader, sourceInfo, err := store.OpenExact(ctx, request.Source)
	if err != nil {
		return objectstore.ObjectInfo{}, err
	}
	body, readErr := io.ReadAll(reader)
	closeErr := reader.Close()
	if readErr != nil || closeErr != nil {
		return objectstore.ObjectInfo{}, errors.Join(readErr, closeErr)
	}
	return store.Write(ctx, objectstore.WriteRequest{
		Bucket: request.DestinationBucket, Key: request.DestinationKey, ContentType: sourceInfo.ContentType,
		Size: int64(len(body)), Metadata: sourceInfo.Metadata, Body: bytes.NewReader(body),
	})
}

func (store *memoryObjectStore) Has(bucket, key string) bool {
	store.mu.Lock()
	defer store.mu.Unlock()
	_, ok := store.objects[objectLocation(bucket, key)]
	return ok
}

func cloneStrings(values map[string]string) map[string]string {
	cloned := make(map[string]string, len(values))
	for key, value := range values {
		cloned[key] = value
	}
	return cloned
}

func sha256Digest(body []byte) string {
	digest := sha256.Sum256(body)
	return "sha256:" + hex.EncodeToString(digest[:])
}

func validPDF(label string) []byte {
	return []byte(fmt.Sprintf("%%PDF-1.7\n1 0 obj\n<</Type/Catalog/Label(%s)>>\nendobj\n%%%%EOF\n", label))
}

func deterministicIDs() func(string) string {
	var mu sync.Mutex
	counters := map[string]int{}
	return func(prefix string) string {
		mu.Lock()
		defer mu.Unlock()
		counters[prefix]++
		return fmt.Sprintf("%s-test-%03d", prefix, counters[prefix])
	}
}

func uploadClock() time.Time { return canonicalNow }
