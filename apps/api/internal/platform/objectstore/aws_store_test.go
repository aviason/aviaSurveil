package objectstore

import (
	"reflect"
	"strings"
	"testing"
)

func TestAWSStoreUsesOnlyIAMProviderChainConfiguration(t *testing.T) {
	t.Parallel()
	configType := reflect.TypeOf(AWSConfig{})
	for _, forbidden := range []string{"AccessKey", "SecretKey", "SessionToken", "Endpoint"} {
		if _, exists := configType.FieldByName(forbidden); exists {
			t.Fatalf("AWSConfig exposes forbidden static credential/endpoint field %s", forbidden)
		}
	}
	store, err := NewAWSStore(AWSConfig{Region: "eu-central-1", HealthBucket: "fixture-private-bucket"})
	if err != nil {
		t.Fatalf("NewAWSStore() error = %v", err)
	}
	if store.CredentialSource() != "aws-iam-provider-chain" {
		t.Fatalf("credential source = %q", store.CredentialSource())
	}
}

func TestExactIdentityRequiresVersionETagHashAndSize(t *testing.T) {
	t.Parallel()
	base := ExactObject{Bucket: "quarantine", Key: "object", VersionID: "version-1", ETag: "etag", SHA256: "sha256:" + strings.Repeat("a", 64), Size: 12}
	if err := validateExactObject(base); err != nil {
		t.Fatalf("complete exact identity error = %v", err)
	}
	for name, mutate := range map[string]func(*ExactObject){
		"bucket":  func(value *ExactObject) { value.Bucket = "" },
		"key":     func(value *ExactObject) { value.Key = "" },
		"version": func(value *ExactObject) { value.VersionID = "" },
		"etag":    func(value *ExactObject) { value.ETag = "" },
		"hash":    func(value *ExactObject) { value.SHA256 = "" },
		"size":    func(value *ExactObject) { value.Size = -1 },
	} {
		t.Run(name, func(t *testing.T) {
			candidate := base
			mutate(&candidate)
			if err := validateExactObject(candidate); err == nil {
				t.Fatal("incomplete exact identity was accepted")
			}
		})
	}
}

func TestExactIdentityComparisonFailsClosed(t *testing.T) {
	t.Parallel()
	expected := ExactObject{Bucket: "quarantine", Key: "object", VersionID: "version-1", ETag: "etag", SHA256: "sha256:" + strings.Repeat("a", 64), Size: 12}
	actual := ObjectInfo{Bucket: expected.Bucket, Key: expected.Key, VersionID: expected.VersionID, ETag: expected.ETag, Size: expected.Size, Metadata: map[string]string{"sha256": expected.SHA256}}
	if err := matchExactObject(expected, actual); err != nil {
		t.Fatalf("matching identity error = %v", err)
	}
	actual.VersionID = "stale-version"
	if err := matchExactObject(expected, actual); err == nil || !strings.Contains(err.Error(), "mismatch") {
		t.Fatalf("mismatched version error = %v", err)
	}
}

func TestExactIdentityPersistenceKeepsThePairAtomicAndLocalCompatible(t *testing.T) {
	t.Parallel()
	partial := ObjectInfo{VersionID: "", ETag: "etag-only"}
	versionID, etag, err := ExactIdentityForPersistence(&MinIOStore{}, partial)
	if err != nil || versionID != "" || etag != "" {
		t.Fatalf("local partial identity = %q/%q, err = %v", versionID, etag, err)
	}
	managed, err := NewAWSStore(AWSConfig{
		Region:       "eu-central-1",
		HealthBucket: "fixture-private-bucket",
	})
	if err != nil {
		t.Fatalf("create managed store: %v", err)
	}
	if _, _, err := ExactIdentityForPersistence(managed, partial); err == nil {
		t.Fatal("managed store accepted a partial identity")
	}
	complete := ObjectInfo{VersionID: "version-1", ETag: "etag-1"}
	versionID, etag, err = ExactIdentityForPersistence(managed, complete)
	if err != nil || versionID != complete.VersionID || etag != complete.ETag {
		t.Fatalf("managed complete identity = %q/%q, err = %v", versionID, etag, err)
	}
}
