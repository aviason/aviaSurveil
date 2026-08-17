package objectstore

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"
	"time"
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

func TestAWSStoreUsesPrivateCredentialProxyForPresigning(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/v2/credentials" {
			t.Fatalf("credential proxy path = %q", request.URL.Path)
		}
		if request.Header.Get("Authorization") != "fixture-token" {
			t.Fatalf("credential proxy authorization header was not forwarded")
		}
		writer.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(writer).Encode(map[string]string{
			"AccessKeyId":     "fixture-access-key",
			"SecretAccessKey": "fixture-secret-key",
			"Token":           "fixture-session-token",
			"Expiration":      time.Now().UTC().Add(time.Hour).Format(time.RFC3339),
			"Code":            "Success",
		})
	}))
	defer server.Close()
	t.Setenv("AWS_CONTAINER_CREDENTIALS_FULL_URI", server.URL+"/v2/credentials")
	t.Setenv("AWS_CONTAINER_AUTHORIZATION_TOKEN", "fixture-token")

	store, err := NewAWSStore(AWSConfig{Region: "eu-central-1", HealthBucket: "fixture-private-bucket"})
	if err != nil {
		t.Fatalf("NewAWSStore() error = %v", err)
	}
	instruction, err := store.CreatePutInstruction(context.Background(), PutRequest{
		Bucket: "fixture-private-bucket", Key: "fixture-key",
		RequiredHeaders: map[string]string{"Content-Type": "application/octet-stream"},
		ExpiresAt:       time.Now().UTC().Add(5 * time.Minute),
	})
	if err != nil {
		t.Fatalf("CreatePutInstruction() error = %v", err)
	}
	presigned, err := url.Parse(instruction.URL)
	if err != nil || presigned.Scheme != "https" || !strings.Contains(presigned.Host, ".s3.") || !strings.HasSuffix(presigned.Host, ".amazonaws.com") {
		t.Fatalf("presigned URL host is not AWS S3: %q", presigned.Host)
	}
}

func TestAWSStoreRejectsUnboundedCredentialProxy(t *testing.T) {
	t.Setenv("AWS_CONTAINER_CREDENTIALS_FULL_URI", "http://169.254.169.254:8181/v2/credentials")
	if _, err := NewAWSStore(AWSConfig{Region: "eu-central-1", HealthBucket: "fixture-private-bucket"}); err == nil {
		t.Fatal("NewAWSStore() accepted an unbounded credential proxy endpoint")
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
