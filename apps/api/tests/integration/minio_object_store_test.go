package integration_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/aviason/aviaSurveil/internal/platform/objectstore"
)

func TestMinIOObjectStoreKeepsObjectsPrivateAndHonorsSignedBoundaries(t *testing.T) {
	endpoint := os.Getenv("AVIA_TEST_OBJECT_STORE_ENDPOINT")
	if endpoint == "" {
		endpoint = "127.0.0.1:59001"
	}
	accessKey := os.Getenv("AVIA_TEST_OBJECT_STORE_ACCESS_KEY")
	if accessKey == "" {
		accessKey = "avia-local-access"
	}
	secretKey := os.Getenv("AVIA_TEST_OBJECT_STORE_SECRET_KEY")
	if secretKey == "" {
		secretKey = "avia-local-secret-key"
	}
	store, err := objectstore.NewMinIOStore(objectstore.MinIOConfig{
		Endpoint: endpoint, AccessKey: accessKey, SecretKey: secretKey,
		AllowServerManagedCORS: true,
	})
	if err != nil {
		t.Fatalf("create MinIO-compatible store: %v", err)
	}
	workerAccessKey := os.Getenv("AVIA_TEST_OBJECT_STORE_WORKER_ACCESS_KEY")
	if workerAccessKey == "" {
		workerAccessKey = "avia-local-worker"
	}
	workerSecretKey := os.Getenv("AVIA_TEST_OBJECT_STORE_WORKER_SECRET_KEY")
	if workerSecretKey == "" {
		workerSecretKey = "avia-local-worker-secret-key"
	}
	workerStore, err := objectstore.NewMinIOStore(objectstore.MinIOConfig{
		Endpoint: endpoint, AccessKey: workerAccessKey, SecretKey: workerSecretKey,
		AllowServerManagedCORS: true,
	})
	if err != nil {
		t.Fatalf("create MinIO worker store: %v", err)
	}
	ctx := context.Background()
	if err := store.EnsurePrivateBuckets(ctx, []string{"evidence-quarantine", "evidence-clean"}, []string{"http://127.0.0.1:4174"}); err != nil {
		t.Fatalf("initialize private object store: %v", err)
	}
	if err := store.Check(ctx); err != nil {
		t.Fatalf("object-store readiness: %v", err)
	}
	body := validPDF("minio-adapter")
	digest := sha256Digest(body)
	key := fmt.Sprintf("adapter-tests/%s-%d", strings.ReplaceAll(t.Name(), "/", "-"), time.Now().UnixNano())
	instruction, err := store.CreatePutInstruction(ctx, objectstore.PutRequest{
		Bucket: "evidence-quarantine", Key: key, ExpiresAt: time.Now().Add(time.Minute),
		RequiredHeaders: map[string]string{
			"Content-Type": "application/pdf", "x-amz-meta-sha256": digest,
			"If-None-Match": "*",
		},
	})
	if err != nil {
		t.Fatalf("create signed PUT: %v", err)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPut, instruction.URL, bytes.NewReader(body))
	if err != nil {
		t.Fatalf("create PUT request: %v", err)
	}
	for key, value := range instruction.Headers {
		request.Header.Set(key, value)
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("upload through signed instruction: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		contents, _ := io.ReadAll(response.Body)
		t.Fatalf("signed PUT status = %d: %s", response.StatusCode, contents)
	}
	repeatedRequest, err := http.NewRequestWithContext(
		ctx,
		http.MethodPut,
		instruction.URL,
		bytes.NewReader(body),
	)
	if err != nil {
		t.Fatalf("create repeated PUT request: %v", err)
	}
	for key, value := range instruction.Headers {
		repeatedRequest.Header.Set(key, value)
	}
	repeatedResponse, err := http.DefaultClient.Do(repeatedRequest)
	if err != nil {
		t.Fatalf("repeat signed PUT: %v", err)
	}
	defer repeatedResponse.Body.Close()
	if repeatedResponse.StatusCode != http.StatusPreconditionFailed {
		t.Fatalf(
			"repeated signed PUT status = %d, want 412",
			repeatedResponse.StatusCode,
		)
	}

	reader, info, err := store.Open(ctx, "evidence-quarantine", key)
	if err != nil {
		t.Fatalf("open uploaded object: %v", err)
	}
	observed, readErr := io.ReadAll(reader)
	closeErr := reader.Close()
	if readErr != nil || closeErr != nil || !bytes.Equal(observed, body) || info.Size != int64(len(body)) {
		t.Fatalf("observed object = %+v, read = %v, close = %v", info, readErr, closeErr)
	}
	destination := key + "-canonical"
	if err := workerStore.Copy(ctx, objectstore.CopyRequest{
		SourceBucket: "evidence-quarantine", SourceKey: key, DestinationBucket: "evidence-clean", DestinationKey: destination,
	}); err != nil {
		t.Fatalf("copy clean object: %v", err)
	}
	if err := workerStore.Copy(ctx, objectstore.CopyRequest{
		SourceBucket: "evidence-quarantine", SourceKey: key, DestinationBucket: "evidence-clean", DestinationKey: destination,
	}); err != objectstore.ErrObjectAlreadyExists {
		t.Fatalf("non-overwriting copy error = %v", err)
	}
	download, err := workerStore.CreateGetInstruction(ctx, objectstore.GetRequest{
		Bucket: "evidence-clean", Key: destination,
		DownloadFileName: "RPT-CAB-2026-001-v1.pdf",
		ExpiresAt:        time.Now().Add(time.Minute),
	})
	if err != nil {
		t.Fatalf("create signed GET: %v", err)
	}
	getResponse, err := http.Get(download.URL)
	if err != nil {
		t.Fatalf("download through signed instruction: %v", err)
	}
	defer getResponse.Body.Close()
	if getResponse.StatusCode != http.StatusOK {
		t.Fatalf("signed GET status = %d", getResponse.StatusCode)
	}
	if disposition := getResponse.Header.Get("Content-Disposition"); disposition !=
		`attachment; filename=RPT-CAB-2026-001-v1.pdf` {
		t.Fatalf("signed GET Content-Disposition = %q", disposition)
	}
	parsed, _ := url.Parse(download.URL)
	parsed.RawQuery = ""
	publicResponse, err := http.Get(parsed.String())
	if err != nil {
		t.Fatalf("try unsigned object access: %v", err)
	}
	defer publicResponse.Body.Close()
	if publicResponse.StatusCode != http.StatusForbidden {
		t.Fatalf("unsigned object access status = %d, want 403", publicResponse.StatusCode)
	}
}

func TestMinIOObjectStoreNeverOverwritesUnderConcurrentWritesAndCopies(t *testing.T) {
	endpoint := os.Getenv("AVIA_TEST_OBJECT_STORE_ENDPOINT")
	if endpoint == "" {
		endpoint = "127.0.0.1:59001"
	}
	accessKey := os.Getenv("AVIA_TEST_OBJECT_STORE_ACCESS_KEY")
	if accessKey == "" {
		accessKey = "avia-local-access"
	}
	secretKey := os.Getenv("AVIA_TEST_OBJECT_STORE_SECRET_KEY")
	if secretKey == "" {
		secretKey = "avia-local-secret-key"
	}
	store, err := objectstore.NewMinIOStore(objectstore.MinIOConfig{
		Endpoint: endpoint, AccessKey: accessKey, SecretKey: secretKey,
		AllowServerManagedCORS: true,
	})
	if err != nil {
		t.Fatalf("create MinIO-compatible store: %v", err)
	}
	workerAccessKey := os.Getenv("AVIA_TEST_OBJECT_STORE_WORKER_ACCESS_KEY")
	if workerAccessKey == "" {
		workerAccessKey = "avia-local-worker"
	}
	workerSecretKey := os.Getenv("AVIA_TEST_OBJECT_STORE_WORKER_SECRET_KEY")
	if workerSecretKey == "" {
		workerSecretKey = "avia-local-worker-secret-key"
	}
	workerStore, err := objectstore.NewMinIOStore(objectstore.MinIOConfig{
		Endpoint: endpoint, AccessKey: workerAccessKey, SecretKey: workerSecretKey,
		AllowServerManagedCORS: true,
	})
	if err != nil {
		t.Fatalf("create MinIO worker store: %v", err)
	}
	ctx := context.Background()
	if err := store.EnsurePrivateBuckets(
		ctx,
		[]string{"evidence-quarantine", "evidence-clean"},
		[]string{"http://127.0.0.1:4174"},
	); err != nil {
		t.Fatalf("initialize private object store: %v", err)
	}

	t.Run("concurrent writes", func(t *testing.T) {
		const contenders = 24
		key := fmt.Sprintf("adapter-tests/%s-%d", strings.ReplaceAll(t.Name(), "/", "-"), time.Now().UnixNano())
		bodies := make([][]byte, contenders)
		for index := range bodies {
			bodies[index] = bytes.Repeat([]byte(fmt.Sprintf("write-%02d|", index)), 128*1024)
		}
		results := raceObjectOperations(contenders, func(index int) error {
			_, err := workerStore.Write(ctx, objectstore.WriteRequest{
				Bucket: "evidence-clean", Key: key, Body: bytes.NewReader(bodies[index]),
				Size: int64(len(bodies[index])), ContentType: "application/octet-stream",
			})
			return err
		})
		winner := requireSingleObjectWinner(t, results)
		reader, _, err := store.Open(ctx, "evidence-clean", key)
		if err != nil {
			t.Fatalf("open concurrent write winner: %v", err)
		}
		observed, readErr := io.ReadAll(reader)
		closeErr := reader.Close()
		if readErr != nil || closeErr != nil || !bytes.Equal(observed, bodies[winner]) {
			t.Fatalf("concurrent write body mismatch: read = %v, close = %v", readErr, closeErr)
		}
	})

	t.Run("concurrent copies", func(t *testing.T) {
		const contenders = 24
		prefix := fmt.Sprintf("adapter-tests/%s-%d", strings.ReplaceAll(t.Name(), "/", "-"), time.Now().UnixNano())
		bodies := make([][]byte, contenders)
		for index := range bodies {
			bodies[index] = bytes.Repeat([]byte(fmt.Sprintf("copy-%02d|", index)), 128*1024)
			if _, err := store.Write(ctx, objectstore.WriteRequest{
				Bucket: "evidence-quarantine", Key: fmt.Sprintf("%s-source-%02d", prefix, index),
				Body: bytes.NewReader(bodies[index]), Size: int64(len(bodies[index])),
				ContentType: "application/octet-stream",
			}); err != nil {
				t.Fatalf("write copy source %d: %v", index, err)
			}
		}
		destination := prefix + "-destination"
		results := raceObjectOperations(contenders, func(index int) error {
			return workerStore.Copy(ctx, objectstore.CopyRequest{
				SourceBucket: "evidence-quarantine", SourceKey: fmt.Sprintf("%s-source-%02d", prefix, index),
				DestinationBucket: "evidence-clean", DestinationKey: destination,
			})
		})
		winner := requireSingleObjectWinner(t, results)
		reader, _, err := store.Open(ctx, "evidence-clean", destination)
		if err != nil {
			t.Fatalf("open concurrent copy winner: %v", err)
		}
		observed, readErr := io.ReadAll(reader)
		closeErr := reader.Close()
		if readErr != nil || closeErr != nil || !bytes.Equal(observed, bodies[winner]) {
			t.Fatalf("concurrent copy body mismatch: read = %v, close = %v", readErr, closeErr)
		}
	})
}

type objectOperationResult struct {
	index int
	err   error
}

func raceObjectOperations(count int, operation func(int) error) []objectOperationResult {
	start := make(chan struct{})
	results := make(chan objectOperationResult, count)
	var ready sync.WaitGroup
	ready.Add(count)
	for index := 0; index < count; index++ {
		go func() {
			ready.Done()
			<-start
			results <- objectOperationResult{index: index, err: operation(index)}
		}()
	}
	ready.Wait()
	close(start)
	observed := make([]objectOperationResult, 0, count)
	for range count {
		observed = append(observed, <-results)
	}
	return observed
}

func requireSingleObjectWinner(t *testing.T, results []objectOperationResult) int {
	t.Helper()
	winner := -1
	for _, result := range results {
		if result.err == nil {
			if winner != -1 {
				t.Fatalf("object operation allowed multiple winners: %d and %d", winner, result.index)
			}
			winner = result.index
			continue
		}
		if !errors.Is(result.err, objectstore.ErrObjectAlreadyExists) {
			t.Fatalf("object operation %d error = %v", result.index, result.err)
		}
	}
	if winner == -1 {
		t.Fatal("object operation allowed no winner")
	}
	return winner
}
