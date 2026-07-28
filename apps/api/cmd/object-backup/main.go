// Command object-backup copies every live application-object version to the
// logically isolated local backup store and emits a checksum-bound manifest.
package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

const (
	backupBucket  = "application-object-backups"
	catalogBucket = "recovery-catalog"
)

var recoveryPointPattern = regexp.MustCompile(
	`^rp-[a-z0-9TZ][a-z0-9TZ-]{5,95}$`,
)

type policy struct {
	ArtifactStatus string `json:"artifactStatus"`
	ObjectBackup   struct {
		SourceBuckets []string `json:"sourceBuckets"`
		ObjectLock    struct {
			RetentionDays int `json:"retentionDays"`
		} `json:"objectLock"`
	} `json:"objectBackup"`
}

type manifestEntry struct {
	SourceBucket         string            `json:"sourceBucket"`
	Key                  string            `json:"key"`
	VersionID            string            `json:"versionId"`
	IsLatest             bool              `json:"isLatest"`
	LastModified         string            `json:"lastModified"`
	ETag                 string            `json:"etag"`
	SHA256               string            `json:"sha256"`
	Size                 int64             `json:"size"`
	ContentType          string            `json:"contentType"`
	ContentEncoding      string            `json:"contentEncoding"`
	Metadata             map[string]string `json:"metadata"`
	RetentionUntil       string            `json:"retentionUntil"`
	DestinationKey       string            `json:"destinationKey"`
	DestinationVersionID string            `json:"destinationVersionId"`
}

type objectManifest struct {
	SchemaVersion   int             `json:"schemaVersion"`
	ArtifactStatus  string          `json:"artifactStatus"`
	RecoveryPointID string          `json:"recoveryPointId"`
	GeneratedAt     string          `json:"generatedAt"`
	Entries         []manifestEntry `json:"entries"`
	SHA256          string          `json:"sha256"`
}

type objectComponent struct {
	ApplicationObjects struct {
		Status           string `json:"status"`
		ArtifactStatus   string `json:"artifactStatus"`
		RecoveryPointID  string `json:"recoveryPointId"`
		GeneratedAt      string `json:"generatedAt"`
		ManifestSHA256   string `json:"manifestSha256"`
		ObjectCount      int    `json:"objectCount"`
		ManifestObject   string `json:"manifestObject"`
		RetentionUntil   string `json:"retentionUntil"`
		BackupBucket     string `json:"backupBucket"`
		SourceStoreScope string `json:"sourceStoreScope"`
	} `json:"applicationObjects"`
}

type objectRestoreResult struct {
	ObjectRestore struct {
		Status                    string   `json:"status"`
		ArtifactStatus            string   `json:"artifactStatus"`
		RecoveryPointID           string   `json:"recoveryPointId"`
		RestoredAt                string   `json:"restoredAt"`
		SourceManifestSHA256      string   `json:"sourceManifestSha256"`
		RestoredObjectFingerprint string   `json:"restoredObjectFingerprint"`
		RestoredObjectCount       int      `json:"restoredObjectCount"`
		Buckets                   []string `json:"buckets"`
	} `json:"objectRestore"`
}

type clients struct {
	source *minio.Client
	backup *minio.Client
}

func main() {
	if err := run(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "object backup: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	var recoveryPointID string
	var publishCatalog bool
	var restoreManifest bool
	var verifyManifest bool
	var expectedManifestSHA256 string
	flag.StringVar(
		&recoveryPointID,
		"recovery-point",
		"",
		"immutable recovery point identifier",
	)
	flag.BoolVar(
		&publishCatalog,
		"publish-catalog",
		false,
		"publish a verified complete catalog read from stdin",
	)
	flag.BoolVar(
		&restoreManifest,
		"restore-manifest",
		false,
		"restore every checksum-bound manifest version into the configured object store",
	)
	flag.BoolVar(
		&verifyManifest,
		"verify-manifest",
		false,
		"verify the immutable object manifest without restoring objects",
	)
	flag.StringVar(
		&expectedManifestSHA256,
		"manifest-sha256",
		"",
		"expected canonical object-manifest checksum",
	)
	flag.Parse()

	if os.Getenv("AVIA_ENVIRONMENT") != "local-candidate" ||
		os.Getenv("AVIA_ENABLE_RECOVERY_BACKUP") != "true" {
		return errors.New(
			"AVIA_ENVIRONMENT=local-candidate and AVIA_ENABLE_RECOVERY_BACKUP=true are required",
		)
	}
	if err := validateRecoveryPointID(recoveryPointID); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()
	stores, err := newClients()
	if err != nil {
		return err
	}

	if publishCatalog {
		return publishCompleteCatalog(ctx, stores.backup, recoveryPointID)
	}
	if verifyManifest {
		manifest, entries, err := loadValidatedObjectManifest(
			ctx,
			stores.backup,
			recoveryPointID,
			expectedManifestSHA256,
		)
		if err != nil {
			return err
		}
		return json.NewEncoder(os.Stdout).Encode(map[string]any{
			"status":          "verified locally",
			"artifactStatus":  "candidate-only",
			"recoveryPointId": recoveryPointID,
			"manifestSha256":  manifest.SHA256,
			"objectCount":     len(entries),
		})
	}
	if restoreManifest {
		return restoreObjectManifest(
			ctx,
			stores,
			recoveryPointID,
			expectedManifestSHA256,
		)
	}
	return copyObjectVersions(ctx, stores, recoveryPointID)
}

func validateRecoveryPointID(value string) error {
	if !recoveryPointPattern.MatchString(value) {
		return fmt.Errorf("invalid recovery point identifier %q", value)
	}
	return nil
}

func newClients() (clients, error) {
	sourceAccess, err := readSecret(
		"AVIA_OBJECT_STORE_ACCESS_KEY_FILE",
		"/run/secrets/minio_root_user",
	)
	if err != nil {
		return clients{}, err
	}
	sourceSecret, err := readSecret(
		"AVIA_OBJECT_STORE_SECRET_KEY_FILE",
		"/run/secrets/minio_root_password",
	)
	if err != nil {
		return clients{}, err
	}
	backupAccess, err := readSecret(
		"AVIA_BACKUP_STORE_ACCESS_KEY_FILE",
		"/run/secrets/backup_object_access_key",
	)
	if err != nil {
		return clients{}, err
	}
	backupSecret, err := readSecret(
		"AVIA_BACKUP_STORE_SECRET_KEY_FILE",
		"/run/secrets/backup_object_secret_key",
	)
	if err != nil {
		return clients{}, err
	}

	source, err := minio.New(envOr("AVIA_OBJECT_STORE_ENDPOINT", "minio:9000"), &minio.Options{
		Creds:        credentials.NewStaticV4(sourceAccess, sourceSecret, ""),
		Secure:       false,
		Region:       "local",
		BucketLookup: minio.BucketLookupPath,
	})
	if err != nil {
		return clients{}, fmt.Errorf("create source client: %w", err)
	}

	transport, err := trustedTransport(
		envOr("AVIA_BACKUP_STORE_CA_FILE", "/certs/backup-minio.crt"),
	)
	if err != nil {
		return clients{}, err
	}
	backup, err := minio.New(envOr("AVIA_BACKUP_STORE_ENDPOINT", "backup-minio:9000"), &minio.Options{
		Creds:        credentials.NewStaticV4(backupAccess, backupSecret, ""),
		Secure:       true,
		Region:       "local",
		BucketLookup: minio.BucketLookupPath,
		Transport:    transport,
	})
	if err != nil {
		return clients{}, fmt.Errorf("create backup client: %w", err)
	}
	return clients{source: source, backup: backup}, nil
}

func trustedTransport(certificatePath string) (*http.Transport, error) {
	certificate, err := os.ReadFile(filepath.Clean(certificatePath))
	if err != nil {
		return nil, fmt.Errorf("read backup-store certificate: %w", err)
	}
	roots := x509.NewCertPool()
	if !roots.AppendCertsFromPEM(certificate) {
		return nil, errors.New("backup-store certificate is not valid PEM")
	}
	transport, err := minio.DefaultTransport(true)
	if err != nil {
		return nil, fmt.Errorf("create backup-store transport: %w", err)
	}
	transport.TLSClientConfig.RootCAs = roots
	transport.TLSClientConfig.ServerName = "backup-minio"
	return transport, nil
}

func readSecret(environmentName, fallbackPath string) (string, error) {
	secretPath := envOr(environmentName, fallbackPath)
	value, err := os.ReadFile(filepath.Clean(secretPath))
	if err != nil {
		return "", fmt.Errorf("read %s: %w", environmentName, err)
	}
	trimmed := strings.TrimSpace(string(value))
	if trimmed == "" {
		return "", fmt.Errorf("%s is empty", environmentName)
	}
	return trimmed, nil
}

func envOr(name, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return fallback
}

func loadPolicy() (policy, error) {
	source, err := os.ReadFile(
		filepath.Clean(
			envOr(
				"AVIA_BACKUP_POLICY_FILE",
				"/etc/aviasurveil360/minio-backup-policy.json",
			),
		),
	)
	if err != nil {
		return policy{}, fmt.Errorf("read backup policy: %w", err)
	}
	var value policy
	if err := json.Unmarshal(source, &value); err != nil {
		return policy{}, fmt.Errorf("parse backup policy: %w", err)
	}
	if value.ArtifactStatus != "candidate-only" ||
		len(value.ObjectBackup.SourceBuckets) == 0 ||
		value.ObjectBackup.ObjectLock.RetentionDays < 1 {
		return policy{}, errors.New("backup policy is incomplete")
	}
	return value, nil
}

func copyObjectVersions(
	ctx context.Context,
	stores clients,
	recoveryPointID string,
) error {
	backupPolicy, err := loadPolicy()
	if err != nil {
		return err
	}
	retentionUntil := time.Now().
		UTC().
		Add(time.Duration(backupPolicy.ObjectBackup.ObjectLock.RetentionDays) * 24 * time.Hour).
		Truncate(time.Second)

	entries := make([]manifestEntry, 0)
	for _, bucket := range backupPolicy.ObjectBackup.SourceBuckets {
		exists, err := stores.source.BucketExists(ctx, bucket)
		if err != nil || !exists {
			return fmt.Errorf("source bucket %s is unavailable: %w", bucket, err)
		}
		for listed := range stores.source.ListObjects(ctx, bucket, minio.ListObjectsOptions{
			Recursive:       true,
			WithVersions:    true,
			WithMetadata:    true,
			ReverseVersions: false,
		}) {
			if listed.Err != nil {
				return fmt.Errorf("list versions in %s: %w", bucket, listed.Err)
			}
			if listed.IsDeleteMarker {
				continue
			}
			entry, err := copyObjectVersion(
				ctx,
				stores,
				recoveryPointID,
				bucket,
				listed,
				retentionUntil,
			)
			if err != nil {
				return err
			}
			entries = append(entries, entry)
		}
	}

	manifestSHA, canonicalEntries, err := manifestDigest(entries)
	if err != nil {
		return err
	}
	manifest := objectManifest{
		SchemaVersion:   1,
		ArtifactStatus:  "candidate-only",
		RecoveryPointID: recoveryPointID,
		GeneratedAt:     time.Now().UTC().Format(time.RFC3339Nano),
		Entries:         canonicalEntries,
		SHA256:          manifestSHA,
	}
	manifestBytes, err := json.Marshal(manifest)
	if err != nil {
		return fmt.Errorf("encode object manifest: %w", err)
	}
	manifestKey := fmt.Sprintf("recovery-points/%s/manifest.json", recoveryPointID)
	if err := putRetainedBytes(
		ctx,
		stores.backup,
		backupBucket,
		manifestKey,
		manifestBytes,
		retentionUntil,
	); err != nil {
		return fmt.Errorf("store object manifest: %w", err)
	}

	var component objectComponent
	component.ApplicationObjects.Status = "verified locally"
	component.ApplicationObjects.ArtifactStatus = "candidate-only"
	component.ApplicationObjects.RecoveryPointID = recoveryPointID
	component.ApplicationObjects.GeneratedAt = manifest.GeneratedAt
	component.ApplicationObjects.ManifestSHA256 = manifestSHA
	component.ApplicationObjects.ObjectCount = len(canonicalEntries)
	component.ApplicationObjects.ManifestObject = manifestKey
	component.ApplicationObjects.RetentionUntil = retentionUntil.Format(time.RFC3339)
	component.ApplicationObjects.BackupBucket = backupBucket
	component.ApplicationObjects.SourceStoreScope = "primary-application-minio"
	return json.NewEncoder(os.Stdout).Encode(component)
}

func copyObjectVersion(
	ctx context.Context,
	stores clients,
	recoveryPointID string,
	bucket string,
	listed minio.ObjectInfo,
	retentionUntil time.Time,
) (manifestEntry, error) {
	versionOptions := minio.GetObjectOptions{VersionID: listed.VersionID}
	sourceInfo, err := stores.source.StatObject(
		ctx,
		bucket,
		listed.Key,
		minio.StatObjectOptions(versionOptions),
	)
	if err != nil {
		return manifestEntry{}, fmt.Errorf(
			"stat %s/%s version %s: %w",
			bucket,
			listed.Key,
			listed.VersionID,
			err,
		)
	}
	sourceSHA, sourceSize, err := hashObject(
		ctx,
		stores.source,
		bucket,
		listed.Key,
		versionOptions,
	)
	if err != nil {
		return manifestEntry{}, err
	}
	if sourceSize != sourceInfo.Size {
		return manifestEntry{}, fmt.Errorf(
			"source size changed for %s/%s version %s",
			bucket,
			listed.Key,
			listed.VersionID,
		)
	}

	source, err := stores.source.GetObject(
		ctx,
		bucket,
		listed.Key,
		versionOptions,
	)
	if err != nil {
		return manifestEntry{}, fmt.Errorf("open source object: %w", err)
	}
	defer source.Close()

	targetKey := destinationKey(
		recoveryPointID,
		bucket,
		listed.Key,
		listed.VersionID,
	)
	userMetadata := map[string]string{
		"source-bucket":     bucket,
		"source-key":        listed.Key,
		"source-version-id": listed.VersionID,
		"source-etag":       strings.Trim(sourceInfo.ETag, `"`),
		"source-sha256":     sourceSHA,
		"recovery-point-id": recoveryPointID,
		"artifact-status":   "candidate-only",
	}
	upload, err := stores.backup.PutObject(
		ctx,
		backupBucket,
		targetKey,
		source,
		sourceInfo.Size,
		minio.PutObjectOptions{
			UserMetadata:    userMetadata,
			ContentType:     sourceInfo.ContentType,
			ContentEncoding: sourceInfo.ContentEncoding,
			Mode:            minio.Compliance,
			RetainUntilDate: retentionUntil,
		},
	)
	if err != nil {
		return manifestEntry{}, fmt.Errorf("copy source object version: %w", err)
	}

	backupSHA, backupSize, err := hashObject(
		ctx,
		stores.backup,
		backupBucket,
		targetKey,
		minio.GetObjectOptions{VersionID: upload.VersionID},
	)
	if err != nil {
		return manifestEntry{}, err
	}
	if backupSHA != sourceSHA || backupSize != sourceInfo.Size {
		return manifestEntry{}, fmt.Errorf(
			"backup checksum or size differs for %s/%s version %s",
			bucket,
			listed.Key,
			listed.VersionID,
		)
	}

	return manifestEntry{
		SourceBucket:         bucket,
		Key:                  listed.Key,
		VersionID:            listed.VersionID,
		IsLatest:             listed.IsLatest,
		LastModified:         sourceInfo.LastModified.UTC().Format(time.RFC3339Nano),
		ETag:                 strings.Trim(sourceInfo.ETag, `"`),
		SHA256:               sourceSHA,
		Size:                 sourceInfo.Size,
		ContentType:          sourceInfo.ContentType,
		ContentEncoding:      sourceInfo.ContentEncoding,
		Metadata:             copyMetadata(sourceInfo.UserMetadata),
		RetentionUntil:       retentionUntil.Format(time.RFC3339),
		DestinationKey:       targetKey,
		DestinationVersionID: upload.VersionID,
	}, nil
}

func restoreObjectManifest(
	ctx context.Context,
	stores clients,
	recoveryPointID string,
	expectedManifestSHA256 string,
) error {
	manifest, entries, err := loadValidatedObjectManifest(
		ctx,
		stores.backup,
		recoveryPointID,
		expectedManifestSHA256,
	)
	if err != nil {
		return err
	}

	bucketSet := make(map[string]struct{})
	for _, entry := range entries {
		exists, err := stores.source.BucketExists(ctx, entry.SourceBucket)
		if err != nil {
			return fmt.Errorf(
				"check restore bucket %s: %w",
				entry.SourceBucket,
				err,
			)
		}
		if !exists {
			return fmt.Errorf(
				"restore bucket %s is unavailable",
				entry.SourceBucket,
			)
		}
		if err := restoreObjectVersion(
			ctx,
			stores,
			entry,
		); err != nil {
			return err
		}
		bucketSet[entry.SourceBucket] = struct{}{}
	}

	buckets := make([]string, 0, len(bucketSet))
	for bucket := range bucketSet {
		buckets = append(buckets, bucket)
	}
	sort.Strings(buckets)
	var result objectRestoreResult
	result.ObjectRestore.Status = "verified locally"
	result.ObjectRestore.ArtifactStatus = "candidate-only"
	result.ObjectRestore.RecoveryPointID = recoveryPointID
	result.ObjectRestore.RestoredAt = time.Now().UTC().Format(time.RFC3339Nano)
	result.ObjectRestore.SourceManifestSHA256 = manifest.SHA256
	result.ObjectRestore.RestoredObjectFingerprint = manifest.SHA256
	result.ObjectRestore.RestoredObjectCount = len(entries)
	result.ObjectRestore.Buckets = buckets
	return json.NewEncoder(os.Stdout).Encode(result)
}

func loadValidatedObjectManifest(
	ctx context.Context,
	backup *minio.Client,
	recoveryPointID string,
	expectedManifestSHA256 string,
) (objectManifest, []manifestEntry, error) {
	manifestKey := fmt.Sprintf(
		"recovery-points/%s/manifest.json",
		recoveryPointID,
	)
	object, err := backup.GetObject(
		ctx,
		backupBucket,
		manifestKey,
		minio.GetObjectOptions{},
	)
	if err != nil {
		return objectManifest{}, nil, fmt.Errorf("open object manifest: %w", err)
	}
	defer object.Close()
	body, err := io.ReadAll(io.LimitReader(object, 32<<20))
	if err != nil {
		return objectManifest{}, nil, fmt.Errorf("read object manifest: %w", err)
	}
	var manifest objectManifest
	if err := json.Unmarshal(body, &manifest); err != nil {
		return objectManifest{}, nil, fmt.Errorf("parse object manifest: %w", err)
	}
	entries, err := validateObjectManifest(
		manifest,
		recoveryPointID,
		expectedManifestSHA256,
	)
	if err != nil {
		return objectManifest{}, nil, err
	}
	return manifest, entries, nil
}

func validateObjectManifest(
	manifest objectManifest,
	expectedRecoveryPointID string,
	expectedSHA256 string,
) ([]manifestEntry, error) {
	if manifest.SchemaVersion != 1 ||
		manifest.ArtifactStatus != "candidate-only" ||
		manifest.RecoveryPointID != expectedRecoveryPointID {
		return nil, errors.New("object manifest identity mismatch")
	}
	if !regexp.MustCompile(`^[a-f0-9]{64}$`).MatchString(expectedSHA256) {
		return nil, errors.New("expected object manifest checksum is invalid")
	}
	digest, canonical, err := manifestDigest(manifest.Entries)
	if err != nil {
		return nil, err
	}
	if manifest.SHA256 != digest || expectedSHA256 != digest {
		return nil, errors.New("object manifest checksum mismatch")
	}
	return restoreOrder(canonical)
}

func restoreOrder(entries []manifestEntry) ([]manifestEntry, error) {
	ordered := append([]manifestEntry(nil), entries...)
	latestByObject := make(map[string]int)
	for _, entry := range ordered {
		if entry.SourceBucket == "" ||
			entry.Key == "" ||
			entry.VersionID == "" ||
			entry.DestinationKey == "" ||
			entry.DestinationVersionID == "" ||
			!regexp.MustCompile(`^[a-f0-9]{64}$`).MatchString(entry.SHA256) ||
			entry.Size < 0 {
			return nil, errors.New("object manifest entry is incomplete")
		}
		if _, err := time.Parse(time.RFC3339, entry.LastModified); err != nil {
			return nil, fmt.Errorf(
				"object manifest lastModified is invalid for %s/%s: %w",
				entry.SourceBucket,
				entry.Key,
				err,
			)
		}
		if entry.IsLatest {
			latestByObject[entry.SourceBucket+"\x00"+entry.Key]++
		}
	}
	for _, entry := range ordered {
		if latestByObject[entry.SourceBucket+"\x00"+entry.Key] != 1 {
			return nil, fmt.Errorf(
				"object manifest must identify one latest version for %s/%s",
				entry.SourceBucket,
				entry.Key,
			)
		}
	}
	sort.Slice(ordered, func(left, right int) bool {
		leftIdentity := ordered[left].SourceBucket + "\x00" + ordered[left].Key
		rightIdentity := ordered[right].SourceBucket + "\x00" + ordered[right].Key
		if leftIdentity != rightIdentity {
			return leftIdentity < rightIdentity
		}
		if ordered[left].LastModified != ordered[right].LastModified {
			return ordered[left].LastModified < ordered[right].LastModified
		}
		return ordered[left].VersionID < ordered[right].VersionID
	})
	return ordered, nil
}

func restoreObjectVersion(
	ctx context.Context,
	stores clients,
	entry manifestEntry,
) error {
	backupOptions := minio.GetObjectOptions{
		VersionID: entry.DestinationVersionID,
	}
	backupSHA, backupSize, err := hashObject(
		ctx,
		stores.backup,
		backupBucket,
		entry.DestinationKey,
		backupOptions,
	)
	if err != nil {
		return err
	}
	if backupSHA != entry.SHA256 || backupSize != entry.Size {
		return fmt.Errorf(
			"backup object checksum mismatch for %s/%s version %s",
			entry.SourceBucket,
			entry.Key,
			entry.VersionID,
		)
	}

	source, err := stores.backup.GetObject(
		ctx,
		backupBucket,
		entry.DestinationKey,
		backupOptions,
	)
	if err != nil {
		return fmt.Errorf("open backup object for restore: %w", err)
	}
	defer source.Close()
	upload, err := stores.source.PutObject(
		ctx,
		entry.SourceBucket,
		entry.Key,
		source,
		entry.Size,
		minio.PutObjectOptions{
			UserMetadata:    entry.Metadata,
			ContentType:     entry.ContentType,
			ContentEncoding: entry.ContentEncoding,
		},
	)
	if err != nil {
		return fmt.Errorf(
			"restore object %s/%s version %s: %w",
			entry.SourceBucket,
			entry.Key,
			entry.VersionID,
			err,
		)
	}
	restoredSHA, restoredSize, err := hashObject(
		ctx,
		stores.source,
		entry.SourceBucket,
		entry.Key,
		minio.GetObjectOptions{VersionID: upload.VersionID},
	)
	if err != nil {
		return err
	}
	if restoredSHA != entry.SHA256 || restoredSize != entry.Size {
		return fmt.Errorf(
			"restored object checksum mismatch for %s/%s version %s",
			entry.SourceBucket,
			entry.Key,
			entry.VersionID,
		)
	}
	return nil
}

func hashObject(
	ctx context.Context,
	client *minio.Client,
	bucket string,
	key string,
	options minio.GetObjectOptions,
) (string, int64, error) {
	object, err := client.GetObject(ctx, bucket, key, options)
	if err != nil {
		return "", 0, fmt.Errorf("open %s/%s for hashing: %w", bucket, key, err)
	}
	defer object.Close()
	digest := sha256.New()
	size, err := io.Copy(digest, object)
	if err != nil {
		return "", 0, fmt.Errorf("hash %s/%s: %w", bucket, key, err)
	}
	return hex.EncodeToString(digest.Sum(nil)), size, nil
}

func destinationKey(
	recoveryPointID string,
	sourceBucket string,
	sourceKey string,
	versionID string,
) string {
	return fmt.Sprintf(
		"recovery-points/%s/objects/%s/%s/versions/%s",
		recoveryPointID,
		sourceBucket,
		strings.TrimPrefix(sourceKey, "/"),
		strings.NewReplacer("/", "%2F", " ", "%20").Replace(versionID),
	)
}

func copyMetadata(source minio.StringMap) map[string]string {
	result := make(map[string]string, len(source))
	for key, value := range source {
		result[key] = value
	}
	return result
}

func manifestDigest(
	entries []manifestEntry,
) (string, []manifestEntry, error) {
	canonical := append([]manifestEntry(nil), entries...)
	sort.Slice(canonical, func(left, right int) bool {
		leftKey := canonical[left].SourceBucket + "\x00" +
			canonical[left].Key + "\x00" + canonical[left].VersionID
		rightKey := canonical[right].SourceBucket + "\x00" +
			canonical[right].Key + "\x00" + canonical[right].VersionID
		return leftKey < rightKey
	})
	encoded, err := json.Marshal(canonical)
	if err != nil {
		return "", nil, fmt.Errorf("encode canonical object manifest: %w", err)
	}
	digest := sha256.Sum256(encoded)
	return hex.EncodeToString(digest[:]), canonical, nil
}

func putRetainedBytes(
	ctx context.Context,
	client *minio.Client,
	bucket string,
	key string,
	body []byte,
	retentionUntil time.Time,
) error {
	_, err := client.PutObject(
		ctx,
		bucket,
		key,
		bytes.NewReader(body),
		int64(len(body)),
		minio.PutObjectOptions{
			ContentType:     "application/json",
			Mode:            minio.Compliance,
			RetainUntilDate: retentionUntil,
			UserMetadata: map[string]string{
				"artifact-status": "candidate-only",
				"sha256":          digestBytes(body),
			},
		},
	)
	return err
}

func publishCompleteCatalog(
	ctx context.Context,
	client *minio.Client,
	recoveryPointID string,
) error {
	body, err := io.ReadAll(io.LimitReader(os.Stdin, 4<<20))
	if err != nil {
		return fmt.Errorf("read catalog: %w", err)
	}
	var catalog struct {
		Status          string `json:"status"`
		RecoveryPointID string `json:"recoveryPointId"`
		RetentionUntil  string `json:"retentionUntil"`
	}
	if err := json.Unmarshal(body, &catalog); err != nil {
		return fmt.Errorf("parse catalog: %w", err)
	}
	if catalog.Status != "complete" || catalog.RecoveryPointID != recoveryPointID {
		return errors.New("only the matching complete catalog may be published")
	}
	retentionUntil, err := time.Parse(time.RFC3339, catalog.RetentionUntil)
	if err != nil || !retentionUntil.After(time.Now().UTC()) {
		return errors.New("catalog retentionUntil must be in the future")
	}
	key := fmt.Sprintf("recovery-points/%s/catalog.json", recoveryPointID)
	if err := putRetainedBytes(
		ctx,
		client,
		catalogBucket,
		key,
		body,
		retentionUntil,
	); err != nil {
		return fmt.Errorf("publish recovery catalog: %w", err)
	}
	return json.NewEncoder(os.Stdout).Encode(map[string]string{
		"status":          "verified locally",
		"artifactStatus":  "candidate-only",
		"recoveryPointId": recoveryPointID,
		"catalogObject":   key,
		"sha256":          digestBytes(body),
	})
}

func digestBytes(body []byte) string {
	digest := sha256.Sum256(body)
	return hex.EncodeToString(digest[:])
}
