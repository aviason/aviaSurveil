package objectstore

import (
	"context"
	"errors"
	"io"
	"strings"
	"time"
)

var (
	ErrObjectNotFound      = errors.New("object not found")
	ErrObjectAlreadyExists = errors.New("object already exists")
)

type PutRequest struct {
	Bucket          string
	Key             string
	RequiredHeaders map[string]string
	ExpiresAt       time.Time
}

type PutInstruction struct {
	URL       string            `json:"url"`
	Headers   map[string]string `json:"headers"`
	ExpiresAt time.Time         `json:"expiresAt"`
}

type ObjectInfo struct {
	Bucket       string
	Key          string
	VersionID    string
	ETag         string
	Size         int64
	ContentType  string
	Metadata     map[string]string
	LastModified time.Time
}

type CopyRequest struct {
	SourceBucket      string
	SourceKey         string
	DestinationBucket string
	DestinationKey    string
}

// ExactObject identifies one immutable object version. Production managed-scan
// decisions must bind every field before bytes can leave quarantine.
type ExactObject struct {
	Bucket    string
	Key       string
	VersionID string
	ETag      string
	SHA256    string
	Size      int64
}

type ExactCopyRequest struct {
	Source            ExactObject
	DestinationBucket string
	DestinationKey    string
}

// ExactVersionStore is deliberately separate from Store. Local MinIO callers
// keep the existing stream-scanner contract; the AWS private-pilot worker must
// explicitly require this stronger version/tag/copy boundary.
type ExactVersionStore interface {
	OpenExact(context.Context, ExactObject) (io.ReadCloser, ObjectInfo, error)
	ReadTagsExact(context.Context, ExactObject) (map[string]string, ObjectInfo, error)
	CopyExact(context.Context, ExactCopyRequest) (ObjectInfo, error)
}

type ExactIdentityPolicy interface {
	RequiresExactVersionIdentity() bool
}

func RequiresExactVersionIdentity(store Store) bool {
	policy, ok := store.(ExactIdentityPolicy)
	return ok && policy.RequiresExactVersionIdentity()
}

// ExactIdentityForPersistence keeps the database pair atomic. Local stores
// that do not require exact identity may expose an ETag without versioning;
// that partial observation is deliberately not persisted. Managed production
// stores must supply both values or the caller fails closed.
func ExactIdentityForPersistence(store Store, info ObjectInfo) (string, string, error) {
	if !RequiresExactVersionIdentity(store) {
		return "", "", nil
	}
	versionID := strings.TrimSpace(info.VersionID)
	etag := strings.TrimSpace(info.ETag)
	if versionID == "" || etag == "" {
		return "", "", errors.New("exact object version identity is incomplete")
	}
	return versionID, etag, nil
}

type WriteRequest struct {
	Bucket      string
	Key         string
	ContentType string
	Size        int64
	Metadata    map[string]string
	Body        io.Reader
}

type GetRequest struct {
	Bucket           string
	Key              string
	DownloadFileName string
	ExpiresAt        time.Time
}

type GetInstruction struct {
	URL       string    `json:"url"`
	ExpiresAt time.Time `json:"expiresAt"`
}

// Store is backend-neutral. Implementations must keep buckets private and
// issue short-lived instructions instead of exposing durable public URLs.
type Store interface {
	CreatePutInstruction(context.Context, PutRequest) (PutInstruction, error)
	Write(context.Context, WriteRequest) (ObjectInfo, error)
	Open(context.Context, string, string) (io.ReadCloser, ObjectInfo, error)
	Copy(context.Context, CopyRequest) error
	CreateGetInstruction(context.Context, GetRequest) (GetInstruction, error)
	Check(context.Context) error
}

// TestResetter is intentionally separate from Store so production services
// cannot acquire destructive bucket-reset authority.
type TestResetter interface {
	ResetPrivateBuckets(context.Context, []string) error
}
