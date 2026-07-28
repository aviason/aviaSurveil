package scenarios

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"sort"
	"strings"
)

var (
	ErrScenarioObjectNotFound = errors.New(
		"connected-scenario object not found",
	)
	ErrScenarioObjectAlreadyExists = errors.New(
		"connected-scenario object already exists",
	)
)

type ObjectBlob struct {
	Bucket      string
	Key         string
	ContentType string
	Metadata    map[string]string
	Content     []byte
}

type ObjectBackend interface {
	Check(context.Context) error
	List(context.Context, string, string) ([]ObjectBlob, error)
	Read(context.Context, string, string) (ObjectBlob, error)
	Create(context.Context, ObjectBlob) error
}

type streamingObjectBackend interface {
	Scan(
		context.Context,
		string,
		string,
		func(ObjectBlob) error,
	) error
}

type ConnectedObjectEndpointConfig struct {
	Bucket  string
	Prefix  string
	Backend ObjectBackend
}

type ConnectedObjectEndpoint struct {
	bucket  string
	prefix  string
	backend ObjectBackend
}

func NewConnectedObjectEndpoint(
	config ConnectedObjectEndpointConfig,
) (*ConnectedObjectEndpoint, error) {
	bucket := strings.TrimSpace(config.Bucket)
	prefix := strings.TrimSpace(config.Prefix)
	if bucket == "" ||
		!strings.HasPrefix(prefix, "runs/") ||
		!strings.HasSuffix(prefix, "/") ||
		strings.Contains(prefix, "..") ||
		config.Backend == nil {
		return nil, fmt.Errorf(
			"exact object bucket, run prefix, and backend are required",
		)
	}
	return &ConnectedObjectEndpoint{
		bucket:  bucket,
		prefix:  prefix,
		backend: config.Backend,
	}, nil
}

func (endpoint *ConnectedObjectEndpoint) Preflight(
	ctx context.Context,
) error {
	if err := endpoint.backend.Check(ctx); err != nil {
		return err
	}
	objects, err := endpoint.backend.List(
		ctx,
		endpoint.bucket,
		endpoint.prefix,
	)
	if err != nil {
		return err
	}
	if len(objects) != 0 {
		return fmt.Errorf(
			"connected-scenario object prefix retains %d objects",
			len(objects),
		)
	}
	return nil
}

func (endpoint *ConnectedObjectEndpoint) ResumePreflight(
	ctx context.Context,
) error {
	if err := endpoint.backend.Check(ctx); err != nil {
		return err
	}
	_, err := endpoint.backend.List(
		ctx,
		endpoint.bucket,
		endpoint.prefix,
	)
	return err
}

func (endpoint *ConnectedObjectEndpoint) EnsureObjectVersion(
	ctx context.Context,
	version ObjectVersion,
) error {
	expected, err := endpoint.blob(version)
	if err != nil {
		return err
	}
	actual, readErr := endpoint.backend.Read(
		ctx,
		expected.Bucket,
		expected.Key,
	)
	if readErr == nil {
		if !sameObjectBlob(actual, expected) {
			return fmt.Errorf(
				"existing connected-scenario object %s differs",
				expected.Key,
			)
		}
		return nil
	}
	if !errors.Is(readErr, ErrScenarioObjectNotFound) {
		return readErr
	}
	if err := endpoint.backend.Create(ctx, expected); err != nil {
		if !errors.Is(err, ErrScenarioObjectAlreadyExists) {
			return err
		}
	}
	actual, err = endpoint.backend.Read(
		ctx,
		expected.Bucket,
		expected.Key,
	)
	if err != nil {
		return err
	}
	if !sameObjectBlob(actual, expected) {
		return fmt.Errorf(
			"created connected-scenario object %s differs",
			expected.Key,
		)
	}
	return nil
}

func (endpoint *ConnectedObjectEndpoint) ReconcileObjectVersions(
	ctx context.Context,
	expected []ObjectVersion,
) error {
	listed, err := endpoint.backend.List(
		ctx,
		endpoint.bucket,
		endpoint.prefix,
	)
	if err != nil {
		return err
	}
	if len(listed) != len(expected) {
		return fmt.Errorf(
			"object prefix count = %d, expected %d",
			len(listed),
			len(expected),
		)
	}
	expectedKeys := make([]string, len(expected))
	for index, version := range expected {
		blob, err := endpoint.blob(version)
		if err != nil {
			return err
		}
		expectedKeys[index] = blob.Key
		actual, err := endpoint.backend.Read(ctx, blob.Bucket, blob.Key)
		if err != nil {
			return err
		}
		if !sameObjectBlob(actual, blob) {
			return fmt.Errorf(
				"connected-scenario object %s differs",
				blob.Key,
			)
		}
	}
	actualKeys := make([]string, len(listed))
	for index, blob := range listed {
		actualKeys[index] = blob.Key
	}
	sort.Strings(expectedKeys)
	sort.Strings(actualKeys)
	if !sameStrings(expectedKeys, actualKeys) {
		return fmt.Errorf("connected-scenario object keys differ")
	}
	return nil
}

func (endpoint *ConnectedObjectEndpoint) ReconcileObjectVersionStream(
	ctx context.Context,
	scanExpected func(func(ObjectVersion) error) error,
) error {
	backend, ok := endpoint.backend.(streamingObjectBackend)
	if !ok || scanExpected == nil {
		return fmt.Errorf(
			"streaming object reconciliation is required for scale profiles",
		)
	}
	expectedDigest := newRelationshipDigestAccumulator()
	var expectedCount int64
	var previousExpectedKey string
	if err := scanExpected(func(version ObjectVersion) error {
		blob, err := endpoint.blob(version)
		if err != nil {
			return err
		}
		if previousExpectedKey != "" && blob.Key <= previousExpectedKey {
			return fmt.Errorf(
				"expected object stream is not strictly ordered",
			)
		}
		actual, err := endpoint.backend.Read(
			ctx,
			blob.Bucket,
			blob.Key,
		)
		if err != nil {
			return err
		}
		if !sameObjectBlob(actual, blob) {
			return fmt.Errorf(
				"connected-scenario object %s differs",
				blob.Key,
			)
		}
		expectedDigest.Add([]string{blob.Key})
		previousExpectedKey = blob.Key
		expectedCount++
		return nil
	}); err != nil {
		return err
	}
	if expectedCount == 0 {
		return fmt.Errorf("expected object stream is empty")
	}

	actualDigest := newRelationshipDigestAccumulator()
	var actualCount int64
	var previousActualKey string
	if err := backend.Scan(
		ctx,
		endpoint.bucket,
		endpoint.prefix,
		func(blob ObjectBlob) error {
			if blob.Bucket != endpoint.bucket ||
				!strings.HasPrefix(blob.Key, endpoint.prefix) ||
				len(blob.Key) > 1024 ||
				(previousActualKey != "" &&
					blob.Key <= previousActualKey) {
				return fmt.Errorf(
					"actual object stream is not exact and ordered",
				)
			}
			actualDigest.Add([]string{blob.Key})
			previousActualKey = blob.Key
			actualCount++
			return nil
		},
	); err != nil {
		return err
	}
	if actualCount != expectedCount ||
		actualDigest.Digest() != expectedDigest.Digest() {
		return fmt.Errorf(
			"object stream count or key digest differs",
		)
	}
	return nil
}

func (endpoint *ConnectedObjectEndpoint) blob(
	version ObjectVersion,
) (ObjectBlob, error) {
	if version.Bucket != endpoint.bucket ||
		version.Key != endpoint.prefix+
			"objects/"+version.VersionID+".json" ||
		!syntheticSubjectPattern.MatchString(version.VersionID) ||
		!syntheticSubjectPattern.MatchString(version.ObjectID) ||
		strings.TrimSpace(version.OrganizationID) == "" ||
		version.SizeBytes != int64(len(version.Content)) ||
		!json.Valid(version.Content) {
		return ObjectBlob{}, fmt.Errorf(
			"invalid connected-scenario object version",
		)
	}
	var payload struct {
		SchemaVersion  string `json:"schemaVersion"`
		Synthetic      bool   `json:"synthetic"`
		RecordID       string `json:"recordId"`
		ObjectID       string `json:"objectId"`
		OrganizationID string `json:"organizationId"`
		BinaryIncluded bool   `json:"binaryIncluded"`
		Padding        string `json:"padding,omitempty"`
	}
	if err := json.Unmarshal(version.Content, &payload); err != nil ||
		payload.SchemaVersion != "preprod-synthetic-object/v1" ||
		!payload.Synthetic ||
		payload.RecordID != version.VersionID ||
		payload.ObjectID != version.ObjectID ||
		payload.OrganizationID != version.OrganizationID ||
		payload.BinaryIncluded ||
		strings.Trim(payload.Padding, "S") != "" {
		return ObjectBlob{}, fmt.Errorf(
			"connected-scenario object content is not exact safe JSON",
		)
	}
	digest := sha256.Sum256(version.Content)
	actualDigest := "sha256:" + hex.EncodeToString(digest[:])
	if version.ContentDigest != actualDigest {
		return ObjectBlob{}, fmt.Errorf(
			"connected-scenario object digest differs",
		)
	}
	return ObjectBlob{
		Bucket:      version.Bucket,
		Key:         version.Key,
		ContentType: "application/json",
		Metadata: map[string]string{
			"sha256":          version.ContentDigest,
			"synthetic":       "true",
			"organization-id": version.OrganizationID,
			"object-id":       version.ObjectID,
		},
		Content: append([]byte(nil), version.Content...),
	}, nil
}

func sameObjectBlob(left, right ObjectBlob) bool {
	return left.Bucket == right.Bucket &&
		left.Key == right.Key &&
		left.ContentType == right.ContentType &&
		maps.Equal(left.Metadata, right.Metadata) &&
		bytes.Equal(left.Content, right.Content)
}
