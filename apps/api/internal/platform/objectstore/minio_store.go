package objectstore

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/cors"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type MinIOConfig struct {
	Endpoint               string
	PublicEndpoint         string
	AccessKey              string
	SecretKey              string
	UseTLS                 bool
	PublicUseTLS           bool
	Region                 string
	AllowServerManagedCORS bool
	Clock                  func() time.Time
}

type MinIOStore struct {
	client                 *minio.Client
	signer                 *minio.Client
	allowServerManagedCORS bool
	clock                  func() time.Time
}

func NewMinIOStore(config MinIOConfig) (*MinIOStore, error) {
	if strings.TrimSpace(config.Endpoint) == "" || config.AccessKey == "" || config.SecretKey == "" {
		return nil, errors.New("object-store endpoint and credentials are required")
	}
	client, err := minio.New(config.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(config.AccessKey, config.SecretKey, ""),
		Secure: config.UseTLS, Region: config.Region,
	})
	if err != nil {
		return nil, fmt.Errorf("create MinIO-compatible client: %w", err)
	}
	publicEndpoint := strings.TrimSpace(config.PublicEndpoint)
	publicUseTLS := config.PublicUseTLS
	if publicEndpoint == "" {
		publicEndpoint = config.Endpoint
		publicUseTLS = config.UseTLS
	}
	signer, err := minio.New(publicEndpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(config.AccessKey, config.SecretKey, ""),
		Secure: publicUseTLS,
		Region: config.Region,
	})
	if err != nil {
		return nil, fmt.Errorf("create public MinIO-compatible signer: %w", err)
	}
	clock := config.Clock
	if clock == nil {
		clock = time.Now
	}
	return &MinIOStore{
		client: client, signer: signer,
		allowServerManagedCORS: config.AllowServerManagedCORS, clock: clock,
	}, nil
}

func (store *MinIOStore) EnsurePrivateBuckets(ctx context.Context, buckets []string, allowedOrigins []string) error {
	if len(allowedOrigins) == 0 {
		return errors.New("at least one explicit object-store CORS origin is required")
	}
	for _, bucket := range buckets {
		exists, err := store.client.BucketExists(ctx, bucket)
		if err != nil {
			return fmt.Errorf("check private bucket %s: %w", bucket, err)
		}
		if !exists {
			if err := store.client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{}); err != nil {
				return fmt.Errorf("create private bucket %s: %w", bucket, err)
			}
		}
		configuration := cors.NewConfig([]cors.Rule{{
			ID: "avia-private-browser", AllowedOrigin: append([]string(nil), allowedOrigins...),
			AllowedMethod: []string{http.MethodGet, http.MethodHead, http.MethodPut},
			AllowedHeader: []string{"Content-Type", "x-amz-meta-sha256", "If-None-Match"},
			ExposeHeader:  []string{"ETag"}, MaxAgeSeconds: 300,
		}})
		if err := store.client.SetBucketCors(ctx, bucket, configuration); err != nil {
			response := minio.ToErrorResponse(err)
			// Some disposable MinIO releases reject the unsupported CORS
			// action at the IAM layer as AccessDenied before returning the
			// underlying NotImplemented response.  The local-preprod profile
			// preconfigures no browser CORS because the server does not expose
			// that API; it still exercises real bucket/object authorization.
			if !store.allowServerManagedCORS ||
				(response.Code != "NotImplemented" && response.Code != "AccessDenied") {
				return fmt.Errorf("configure private bucket CORS %s: %w", bucket, err)
			}
		}
	}
	return nil
}

func (store *MinIOStore) CreatePutInstruction(ctx context.Context, request PutRequest) (PutInstruction, error) {
	duration := request.ExpiresAt.Sub(store.clock().UTC())
	if duration < time.Second {
		return PutInstruction{}, errors.New("upload instruction expiry must be in the future")
	}
	headers := make(http.Header, len(request.RequiredHeaders))
	for key, value := range request.RequiredHeaders {
		headers.Set(key, value)
	}
	presigned, err := store.signer.PresignHeader(ctx, http.MethodPut, request.Bucket, request.Key, duration, nil, headers)
	if err != nil {
		return PutInstruction{}, fmt.Errorf("presign private PUT: %w", err)
	}
	return PutInstruction{URL: presigned.String(), Headers: cloneHeaders(request.RequiredHeaders), ExpiresAt: request.ExpiresAt}, nil
}

func (store *MinIOStore) Write(ctx context.Context, request WriteRequest) (ObjectInfo, error) {
	if request.Body == nil || request.Size < 0 {
		return ObjectInfo{}, errors.New("private object body and non-negative size are required")
	}
	options := minio.PutObjectOptions{
		ContentType:      request.ContentType,
		UserMetadata:     cloneHeaders(request.Metadata),
		DisableMultipart: true,
	}
	options.SetMatchETagExcept("*")
	info, err := store.client.PutObject(ctx, request.Bucket, request.Key, request.Body, request.Size, options)
	if err != nil {
		return ObjectInfo{}, fmt.Errorf("write private object: %w", mapObjectError(err))
	}
	return ObjectInfo{
		Bucket: request.Bucket, Key: request.Key, Size: info.Size,
		ContentType: request.ContentType, Metadata: cloneHeaders(request.Metadata),
	}, nil
}

func (store *MinIOStore) Open(ctx context.Context, bucket, key string) (io.ReadCloser, ObjectInfo, error) {
	stat, err := store.client.StatObject(ctx, bucket, key, minio.StatObjectOptions{})
	if err != nil {
		return nil, ObjectInfo{}, mapObjectError(err)
	}
	object, err := store.client.GetObject(ctx, bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, ObjectInfo{}, mapObjectError(err)
	}
	metadata := make(map[string]string, len(stat.UserMetadata))
	for key, value := range stat.UserMetadata {
		metadata[strings.ToLower(key)] = value
	}
	return object, ObjectInfo{
		Bucket: bucket, Key: key, Size: stat.Size, ContentType: stat.ContentType, Metadata: metadata,
	}, nil
}

func (store *MinIOStore) Copy(ctx context.Context, request CopyRequest) error {
	sourceInfo, err := store.client.StatObject(
		ctx,
		request.SourceBucket,
		request.SourceKey,
		minio.StatObjectOptions{},
	)
	if err != nil {
		return fmt.Errorf("inspect private copy source: %w", mapObjectError(err))
	}
	source, err := store.client.GetObject(
		ctx,
		request.SourceBucket,
		request.SourceKey,
		minio.GetObjectOptions{},
	)
	if err != nil {
		return fmt.Errorf("open private copy source: %w", mapObjectError(err))
	}
	options := minio.PutObjectOptions{
		ContentType:      sourceInfo.ContentType,
		UserMetadata:     cloneHeaders(sourceInfo.UserMetadata),
		DisableMultipart: true,
	}
	options.SetMatchETagExcept("*")
	_, writeErr := store.client.PutObject(
		ctx,
		request.DestinationBucket,
		request.DestinationKey,
		source,
		sourceInfo.Size,
		options,
	)
	closeErr := source.Close()
	if writeErr != nil || closeErr != nil {
		if writeErr != nil {
			writeErr = mapObjectError(writeErr)
			if errors.Is(writeErr, ErrObjectAlreadyExists) {
				writeErr = ErrObjectAlreadyExists
			} else {
				writeErr = fmt.Errorf("copy private object: %w", writeErr)
			}
		}
		if closeErr == nil {
			return writeErr
		}
		if writeErr == nil {
			return closeErr
		}
		return errors.Join(writeErr, closeErr)
	}
	return nil
}

func (store *MinIOStore) CreateGetInstruction(ctx context.Context, request GetRequest) (GetInstruction, error) {
	if _, err := store.client.StatObject(ctx, request.Bucket, request.Key, minio.StatObjectOptions{}); err != nil {
		return GetInstruction{}, mapObjectError(err)
	}
	duration := request.ExpiresAt.Sub(store.clock().UTC())
	if duration < time.Second {
		return GetInstruction{}, errors.New("download instruction expiry must be in the future")
	}
	responseParameters := url.Values{}
	if fileName := strings.TrimSpace(request.DownloadFileName); fileName != "" {
		disposition := mime.FormatMediaType(
			"attachment",
			map[string]string{"filename": fileName},
		)
		if disposition == "" {
			return GetInstruction{}, errors.New("download filename cannot form Content-Disposition")
		}
		responseParameters.Set("response-content-disposition", disposition)
	}
	presigned, err := store.signer.PresignedGetObject(
		ctx,
		request.Bucket,
		request.Key,
		duration,
		responseParameters,
	)
	if err != nil {
		return GetInstruction{}, fmt.Errorf("presign private GET: %w", err)
	}
	return GetInstruction{URL: presigned.String(), ExpiresAt: request.ExpiresAt}, nil
}

func (store *MinIOStore) Check(ctx context.Context) error {
	_, err := store.client.ListBuckets(ctx)
	if err != nil {
		return fmt.Errorf("check object-store connectivity: %w", err)
	}
	return nil
}

func (store *MinIOStore) ResetPrivateBuckets(ctx context.Context, buckets []string) error {
	for _, bucket := range buckets {
		objects := store.client.ListObjects(ctx, bucket, minio.ListObjectsOptions{Recursive: true})
		for object := range objects {
			if object.Err != nil {
				return fmt.Errorf("list test objects in %s: %w", bucket, object.Err)
			}
			if err := store.client.RemoveObject(ctx, bucket, object.Key, minio.RemoveObjectOptions{}); err != nil {
				return fmt.Errorf("remove test object %s/%s: %w", bucket, object.Key, err)
			}
		}
	}
	return nil
}

func mapObjectError(err error) error {
	if err == nil {
		return nil
	}
	response := minio.ToErrorResponse(err)
	switch response.Code {
	case "NoSuchKey", "NoSuchObject", "NoSuchBucket", "NotFound":
		return fmt.Errorf("%w: %v", ErrObjectNotFound, err)
	case "PreconditionFailed":
		return fmt.Errorf("%w: %v", ErrObjectAlreadyExists, err)
	default:
		return err
	}
}

func cloneHeaders(values map[string]string) map[string]string {
	cloned := make(map[string]string, len(values))
	for key, value := range values {
		cloned[key] = value
	}
	return cloned
}
