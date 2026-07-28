package scenarios

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type MinIOObjectBackendConfig struct {
	Endpoint   string
	AccessKey  string
	SecretKey  string
	UseTLS     bool
	Region     string
	HTTPClient *http.Client
}

type MinIOObjectBackend struct {
	client *minio.Client
}

func NewMinIOObjectBackend(
	config MinIOObjectBackendConfig,
) (*MinIOObjectBackend, error) {
	if strings.TrimSpace(config.Endpoint) == "" ||
		config.AccessKey == "" ||
		config.SecretKey == "" {
		return nil, fmt.Errorf(
			"MinIO endpoint and loader credentials are required",
		)
	}
	options := &minio.Options{
		Creds: credentials.NewStaticV4(
			config.AccessKey,
			config.SecretKey,
			"",
		),
		Secure: config.UseTLS,
		Region: config.Region,
	}
	if config.HTTPClient != nil && config.HTTPClient.Transport != nil {
		options.Transport = config.HTTPClient.Transport
	}
	client, err := minio.New(config.Endpoint, options)
	if err != nil {
		return nil, fmt.Errorf("create MinIO scenario client: %w", err)
	}
	return &MinIOObjectBackend{client: client}, nil
}

func (backend *MinIOObjectBackend) Check(ctx context.Context) error {
	if _, err := backend.client.ListBuckets(ctx); err != nil {
		return fmt.Errorf("check MinIO scenario backend: %w", err)
	}
	return nil
}

func (backend *MinIOObjectBackend) List(
	ctx context.Context,
	bucket,
	prefix string,
) ([]ObjectBlob, error) {
	objects := backend.client.ListObjects(
		ctx,
		bucket,
		minio.ListObjectsOptions{
			Prefix:    prefix,
			Recursive: true,
		},
	)
	var output []ObjectBlob
	for object := range objects {
		if object.Err != nil {
			return nil, mapScenarioObjectError(object.Err)
		}
		output = append(output, ObjectBlob{
			Bucket: bucket,
			Key:    object.Key,
		})
	}
	return output, nil
}

func (backend *MinIOObjectBackend) Read(
	ctx context.Context,
	bucket,
	key string,
) (ObjectBlob, error) {
	stat, err := backend.client.StatObject(
		ctx,
		bucket,
		key,
		minio.StatObjectOptions{},
	)
	if err != nil {
		return ObjectBlob{}, mapScenarioObjectError(err)
	}
	object, err := backend.client.GetObject(
		ctx,
		bucket,
		key,
		minio.GetObjectOptions{},
	)
	if err != nil {
		return ObjectBlob{}, mapScenarioObjectError(err)
	}
	content, readErr := io.ReadAll(io.LimitReader(object, stat.Size+1))
	closeErr := object.Close()
	if readErr != nil || closeErr != nil {
		return ObjectBlob{}, errors.Join(readErr, closeErr)
	}
	if int64(len(content)) != stat.Size {
		return ObjectBlob{}, fmt.Errorf(
			"MinIO object size changed while reading %s",
			key,
		)
	}
	metadata := make(map[string]string, len(stat.UserMetadata))
	for name, value := range stat.UserMetadata {
		metadata[strings.ToLower(name)] = value
	}
	return ObjectBlob{
		Bucket:      bucket,
		Key:         key,
		ContentType: stat.ContentType,
		Metadata:    metadata,
		Content:     content,
	}, nil
}

func (backend *MinIOObjectBackend) Create(
	ctx context.Context,
	blob ObjectBlob,
) error {
	if blob.Bucket == "" ||
		blob.Key == "" ||
		blob.ContentType == "" ||
		len(blob.Content) == 0 {
		return fmt.Errorf("complete MinIO scenario object is required")
	}
	options := minio.PutObjectOptions{
		ContentType:      blob.ContentType,
		UserMetadata:     mapsClone(blob.Metadata),
		DisableMultipart: true,
	}
	options.SetMatchETagExcept("*")
	_, err := backend.client.PutObject(
		ctx,
		blob.Bucket,
		blob.Key,
		bytes.NewReader(blob.Content),
		int64(len(blob.Content)),
		options,
	)
	if err != nil {
		return mapScenarioObjectError(err)
	}
	return nil
}

func mapScenarioObjectError(err error) error {
	if err == nil {
		return nil
	}
	response := minio.ToErrorResponse(err)
	switch response.Code {
	case "NoSuchKey", "NoSuchObject", "NoSuchBucket", "NotFound":
		return fmt.Errorf("%w: %v", ErrScenarioObjectNotFound, err)
	case "PreconditionFailed":
		return fmt.Errorf("%w: %v", ErrScenarioObjectAlreadyExists, err)
	default:
		return err
	}
}

func mapsClone(source map[string]string) map[string]string {
	output := make(map[string]string, len(source))
	for key, value := range source {
		output[key] = value
	}
	return output
}

var _ ObjectBackend = (*MinIOObjectBackend)(nil)
