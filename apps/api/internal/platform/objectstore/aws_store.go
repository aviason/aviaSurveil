package objectstore

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/minio/minio-go/v7/pkg/credentials"
)

// AWSConfig intentionally has no access-key or secret-key field. The client
// uses the AWS IAM credential-provider chain (ECS/EKS/EC2 instance metadata)
// and the private-pilot profile separately rejects static credential env vars.
type AWSConfig struct {
	Region       string
	HealthBucket string
	Clock        func() time.Time
}

type AWSStore struct {
	delegate     *MinIOStore
	healthBucket string
}

func NewAWSStore(config AWSConfig) (*AWSStore, error) {
	region := strings.TrimSpace(config.Region)
	if region == "" || strings.TrimSpace(config.HealthBucket) == "" {
		return nil, errors.New("AWS S3 region and health bucket are required")
	}
	endpoint := "s3." + region + ".amazonaws.com"
	delegate, err := newMinIOStore(MinIOConfig{
		Endpoint: endpoint, PublicEndpoint: endpoint, UseTLS: true, PublicUseTLS: true,
		Region: region, Clock: config.Clock,
	}, credentials.NewIAM(""))
	if err != nil {
		return nil, fmt.Errorf("create AWS instance-profile S3 client: %w", err)
	}
	return &AWSStore{delegate: delegate, healthBucket: strings.TrimSpace(config.HealthBucket)}, nil
}

func (*AWSStore) CredentialSource() string           { return "aws-iam-provider-chain" }
func (*AWSStore) RequiresExactVersionIdentity() bool { return true }

func (store *AWSStore) CreatePutInstruction(ctx context.Context, request PutRequest) (PutInstruction, error) {
	return store.delegate.CreatePutInstruction(ctx, request)
}

func (store *AWSStore) Write(ctx context.Context, request WriteRequest) (ObjectInfo, error) {
	return store.delegate.Write(ctx, request)
}

func (store *AWSStore) Open(ctx context.Context, bucket, key string) (io.ReadCloser, ObjectInfo, error) {
	return store.delegate.Open(ctx, bucket, key)
}

func (store *AWSStore) Copy(ctx context.Context, request CopyRequest) error {
	return store.delegate.Copy(ctx, request)
}

func (store *AWSStore) CreateGetInstruction(ctx context.Context, request GetRequest) (GetInstruction, error) {
	return store.delegate.CreateGetInstruction(ctx, request)
}

func (store *AWSStore) Check(ctx context.Context) error {
	_, err := store.delegate.client.GetBucketLocation(ctx, store.healthBucket)
	if err != nil {
		return fmt.Errorf("check configured S3 bucket: %w", mapObjectError(err))
	}
	return nil
}

func (store *AWSStore) OpenExact(ctx context.Context, expected ExactObject) (io.ReadCloser, ObjectInfo, error) {
	return store.delegate.OpenExact(ctx, expected)
}

func (store *AWSStore) ReadTagsExact(ctx context.Context, expected ExactObject) (map[string]string, ObjectInfo, error) {
	return store.delegate.ReadTagsExact(ctx, expected)
}

func (store *AWSStore) CopyExact(ctx context.Context, request ExactCopyRequest) (ObjectInfo, error) {
	return store.delegate.CopyExact(ctx, request)
}

var _ Store = (*AWSStore)(nil)
var _ ExactVersionStore = (*AWSStore)(nil)
