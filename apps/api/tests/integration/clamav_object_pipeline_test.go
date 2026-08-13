package integration_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/aviason/aviaSurveil/internal/evidence"
	"github.com/aviason/aviaSurveil/internal/identity"
	"github.com/aviason/aviaSurveil/internal/platform/database"
	"github.com/aviason/aviaSurveil/internal/platform/objectstore"
	"github.com/aviason/aviaSurveil/internal/platform/scanner"
	evidenceworker "github.com/aviason/aviaSurveil/internal/worker/evidence"
)

func TestLiveClamAVAdapterScansCleanAndEICARStreams(t *testing.T) {
	address := os.Getenv("AVIA_TEST_CLAMAV_ADDRESS")
	if address == "" {
		t.Skip("AVIA_TEST_CLAMAV_ADDRESS is required for the live ClamAV adapter gate")
	}
	client, err := scanner.NewClamAV(scanner.ClamAVConfig{
		Address:             address,
		DialTimeout:         5 * time.Second,
		MaximumSignatureAge: 48 * time.Hour,
	})
	if err != nil {
		t.Fatalf("NewClamAV() error = %v", err)
	}
	if err := client.Ready(context.Background()); err != nil {
		t.Fatalf("ClamAV readiness: %v", err)
	}
	clean, err := client.Scan(
		context.Background(),
		strings.NewReader("%PDF-1.4\nclean local verification\n%%EOF"),
	)
	if err != nil || !clean.Clean || clean.EngineVersion == "" ||
		clean.SignatureVersion == "" || clean.ScannedAt.IsZero() {
		t.Fatalf("clean ClamAV result = %+v, err = %v", clean, err)
	}
	eicar := "X5O!P%@AP[4\\PZX54(P^)7CC)7}$EICAR-STANDARD-ANTIVIRUS-TEST-FILE!$H+H*"
	infected, err := client.Scan(context.Background(), strings.NewReader(eicar))
	if err != nil {
		t.Fatalf("EICAR ClamAV scan error = %v", err)
	}
	if infected.Clean || infected.Reason == "" {
		t.Fatalf("EICAR ClamAV result = %+v", infected)
	}
}

func TestLiveClamAVMinIOEvidencePipeline(t *testing.T) {
	apiObjects, workerObjects := liveObjectStores(t)
	clamav := liveClamAV(t)

	t.Run("clean exact version is promoted and downloadable", func(t *testing.T) {
		pool := canonicalDatabase(t, "live_clamav_minio_clean")
		body := validPDF("live clean Evidence")
		versionID, service := completeLiveEvidenceForScan(
			t,
			pool,
			apiObjects,
			"live-clean",
			body,
		)
		auditee := principal("auditee-xyz", "airline-xyz", "session-auditee", identity.RoleAuditee)
		if _, err := service.Download(
			context.Background(),
			auditee,
			versionID,
		); !errors.Is(err, evidence.ErrEvidenceNotReady) {
			t.Fatalf("pending live Evidence download error = %v", err)
		}
		worker := liveEvidenceWorker(pool, workerObjects, clamav, "live-clean", nil)
		if processed, err := worker.ProcessNext(context.Background()); err != nil || !processed {
			t.Fatalf("process live clean Evidence = %t, %v", processed, err)
		}
		assertEvidenceProcessingState(t, pool, versionID, "CLEAN", "PENDING_CAA_REVIEW")
		assertLiveScanMetadata(t, pool, versionID, true)
		assertLiveCanonicalBytes(t, workerObjects, versionID, body)

		download, err := service.Download(context.Background(), auditee, versionID)
		if err != nil {
			t.Fatalf("create live clean Evidence download: %v", err)
		}
		response, err := http.Get(download.URL)
		if err != nil {
			t.Fatalf("download live clean Evidence: %v", err)
		}
		defer response.Body.Close()
		observed, readErr := io.ReadAll(response.Body)
		if response.StatusCode != http.StatusOK || readErr != nil || !bytes.Equal(observed, body) {
			t.Fatalf(
				"live clean download = status %d, read %v, bytes match %t",
				response.StatusCode,
				readErr,
				bytes.Equal(observed, body),
			)
		}
	})

	t.Run("EICAR exact version remains quarantined and non-downloadable", func(t *testing.T) {
		pool := canonicalDatabase(t, "live_clamav_minio_eicar")
		body := validPDFWithEmbeddedEICAR()
		versionID, service := completeLiveEvidenceForScan(
			t,
			pool,
			apiObjects,
			"live-eicar",
			body,
		)
		worker := liveEvidenceWorker(pool, workerObjects, clamav, "live-eicar", nil)
		if processed, err := worker.ProcessNext(context.Background()); err != nil || !processed {
			t.Fatalf("process live EICAR Evidence = %t, %v", processed, err)
		}
		assertEvidenceProcessingState(t, pool, versionID, "QUARANTINED", "NOT_READY")
		assertLiveScanMetadata(t, pool, versionID, false)
		if _, _, err := workerObjects.Open(
			context.Background(),
			"evidence-clean",
			"organizations/airline-xyz/canonical-evidence/"+versionID,
		); !errors.Is(err, objectstore.ErrObjectNotFound) {
			t.Fatalf("EICAR canonical object error = %v", err)
		}
		if _, err := service.Download(
			context.Background(),
			principal("auditee-xyz", "airline-xyz", "session-auditee", identity.RoleAuditee),
			versionID,
		); !errors.Is(err, evidence.ErrEvidenceNotReady) {
			t.Fatalf("EICAR Evidence download error = %v", err)
		}
	})

	t.Run("restart after promotion does not duplicate the immutable version", func(t *testing.T) {
		pool := canonicalDatabase(t, "live_clamav_minio_restart")
		body := validPDF("live restart Evidence")
		versionID, _ := completeLiveEvidenceForScan(
			t,
			pool,
			apiObjects,
			"live-restart",
			body,
		)
		crash := errors.New("simulated worker restart after live MinIO promotion")
		worker := liveEvidenceWorker(pool, workerObjects, clamav, "live-crash", func() error {
			return crash
		})
		if processed, err := worker.ProcessNext(context.Background()); !processed || !errors.Is(err, crash) {
			t.Fatalf("live crash window = %t, %v", processed, err)
		}
		if _, err := pool.Exec(context.Background(), `
			UPDATE outbox_messages
			SET lease_expires_at = $1
			WHERE topic = 'evidence.scan_requested' AND aggregate_id = $2
		`, canonicalNow.Add(-time.Minute), versionID); err != nil {
			t.Fatalf("expire live worker lease: %v", err)
		}
		recovery := liveEvidenceWorker(pool, workerObjects, clamav, "live-recovery", nil)
		if processed, err := recovery.ProcessNext(context.Background()); err != nil || !processed {
			t.Fatalf("recover live scan request = %t, %v", processed, err)
		}
		assertEvidenceProcessingState(t, pool, versionID, "CLEAN", "PENDING_CAA_REVIEW")
		assertLiveCanonicalBytes(t, workerObjects, versionID, body)
		var versions, canonicalObjects int
		if err := pool.QueryRow(
			context.Background(),
			"SELECT count(*) FROM evidence_versions WHERE id = $1",
			versionID,
		).Scan(&versions); err != nil {
			t.Fatalf("count live immutable Evidence versions: %v", err)
		}
		if err := pool.QueryRow(context.Background(), `
			SELECT count(*)
			FROM object_metadata
			WHERE aggregate_type = 'evidence_version'
			  AND aggregate_id = $1
			  AND object_state = 'CANONICAL'
		`, versionID).Scan(&canonicalObjects); err != nil {
			t.Fatalf("count live canonical object metadata: %v", err)
		}
		if versions != 1 || canonicalObjects != 1 {
			t.Fatalf(
				"live restart immutable counts = versions %d, canonical objects %d",
				versions,
				canonicalObjects,
			)
		}
	})
}

func TestLiveClamAVMinIOPipelineFailsClosedDuringInjectedScannerLoss(t *testing.T) {
	failureMode := os.Getenv("AVIA_TEST_CLAMAV_FAILURE_MODE")
	if failureMode == "" {
		t.Skip("AVIA_TEST_CLAMAV_FAILURE_MODE is required for the live scanner-loss gate")
	}
	apiObjects, workerObjects := liveObjectStores(t)
	clamav := liveClamAV(t)
	pool := canonicalDatabase(t, "live_clamav_loss_"+failureMode)
	versionID, service := completeLiveEvidenceForScan(
		t,
		pool,
		apiObjects,
		"live-"+failureMode,
		validPDF("live scanner "+failureMode),
	)
	worker := evidenceworker.New(
		pool,
		workerObjects,
		clamav,
		evidenceworker.Config{
			WorkerID:         "live-" + failureMode,
			CanonicalBucket:  "evidence-clean",
			AttachmentBucket: "inspection-attachments",
			LeaseDuration:    time.Second,
			ScanTimeout:      250 * time.Millisecond,
			MaximumAttempts:  1,
			Clock:            uploadClock,
			IDGenerator:      liveIDGenerator("live-" + failureMode),
		},
	)
	processed, err := worker.ProcessNext(context.Background())
	if !processed || err == nil {
		t.Fatalf("live scanner-loss result = %t, %v", processed, err)
	}
	if failureMode == "timeout" && !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("live scanner timeout error = %v", err)
	}
	assertEvidenceProcessingState(t, pool, versionID, "FAILED", "NOT_READY")
	if _, err := service.Download(
		context.Background(),
		principal("auditee-xyz", "airline-xyz", "session-auditee", identity.RoleAuditee),
		versionID,
	); !errors.Is(err, evidence.ErrEvidenceNotReady) {
		t.Fatalf("scanner-loss Evidence download error = %v", err)
	}
	if _, _, err := workerObjects.Open(
		context.Background(),
		"evidence-clean",
		"organizations/airline-xyz/canonical-evidence/"+versionID,
	); !errors.Is(err, objectstore.ErrObjectNotFound) {
		t.Fatalf("scanner-loss canonical object error = %v", err)
	}
}

func TestScanPipelinePersistsScannerIdentityAndPromotesToAggregateSpecificBuckets(t *testing.T) {
	t.Run("Evidence clean promotion", func(t *testing.T) {
		pool := canonicalDatabase(t, "clamav_pipeline_evidence_clean")
		objects := newMemoryObjectStore()
		versionID := completeEvidenceForScan(
			t,
			pool,
			objects,
			"finding-clamav-clean",
			"FND-CLAMAV-CLEAN",
			validPDF("clean evidence"),
		)
		scannedAt := canonicalNow.Add(5 * time.Minute)
		worker := evidenceworker.New(
			pool,
			objects,
			metadataScanner{result: scanner.Result{
				Clean:            true,
				EngineVersion:    "1.4.3",
				SignatureVersion: "27411",
				ScannedAt:        scannedAt,
			}},
			evidenceworker.Config{
				WorkerID:         "clamav-evidence-worker",
				CanonicalBucket:  "evidence-clean",
				AttachmentBucket: "inspection-attachments",
				Clock:            uploadClock,
				IDGenerator:      deterministicIDs(),
			},
		)
		processed, err := worker.ProcessNext(context.Background())
		if err != nil || !processed {
			t.Fatalf("ProcessNext() = %t, %v", processed, err)
		}

		var sourceEngine, sourceSignatures string
		var sourceScannedAt time.Time
		if err := pool.QueryRow(context.Background(), `
			SELECT metadata.scan_engine_version, metadata.scan_signature_version, metadata.scanned_at
			FROM evidence_versions version
			JOIN object_metadata metadata ON metadata.id = version.object_metadata_id
			WHERE version.id = $1
		`, versionID).Scan(&sourceEngine, &sourceSignatures, &sourceScannedAt); err != nil {
			t.Fatalf("read Evidence scan metadata: %v", err)
		}
		if sourceEngine != "1.4.3" || sourceSignatures != "27411" || !sourceScannedAt.Equal(scannedAt) {
			t.Fatalf("Evidence scan metadata = %q/%q/%s", sourceEngine, sourceSignatures, sourceScannedAt)
		}
		var canonicalBucket, canonicalEngine, canonicalSignatures string
		var canonicalScannedAt time.Time
		if err := pool.QueryRow(context.Background(), `
			SELECT metadata.bucket_name, metadata.scan_engine_version,
			       metadata.scan_signature_version, metadata.scanned_at
			FROM evidence_version_states state
			JOIN object_metadata metadata ON metadata.id = state.canonical_object_metadata_id
			WHERE state.evidence_version_id = $1
		`, versionID).Scan(
			&canonicalBucket,
			&canonicalEngine,
			&canonicalSignatures,
			&canonicalScannedAt,
		); err != nil {
			t.Fatalf("read canonical Evidence metadata: %v", err)
		}
		if canonicalBucket != "evidence-clean" ||
			canonicalEngine != "1.4.3" ||
			canonicalSignatures != "27411" ||
			!canonicalScannedAt.Equal(scannedAt) {
			t.Fatalf(
				"canonical Evidence metadata = %q/%q/%q/%s",
				canonicalBucket,
				canonicalEngine,
				canonicalSignatures,
				canonicalScannedAt,
			)
		}
		if len(objects.copies) != 1 ||
			objects.copies[0].DestinationBucket != "evidence-clean" {
			t.Fatalf("Evidence promotions = %+v", objects.copies)
		}
	})

	t.Run("infected Evidence stays quarantined", func(t *testing.T) {
		pool := canonicalDatabase(t, "clamav_pipeline_evidence_infected")
		objects := newMemoryObjectStore()
		versionID := completeEvidenceForScan(
			t,
			pool,
			objects,
			"finding-clamav-infected",
			"FND-CLAMAV-INFECTED",
			validPDF("infected evidence"),
		)
		worker := evidenceworker.New(
			pool,
			objects,
			metadataScanner{result: scanner.Result{
				Clean:            false,
				Reason:           "Win.Test.EICAR_HDB-1",
				EngineVersion:    "1.4.3",
				SignatureVersion: "27411",
				ScannedAt:        canonicalNow.Add(5 * time.Minute),
			}},
			evidenceworker.Config{
				WorkerID:         "clamav-infected-worker",
				CanonicalBucket:  "evidence-clean",
				AttachmentBucket: "inspection-attachments",
				Clock:            uploadClock,
				IDGenerator:      deterministicIDs(),
			},
		)
		processed, err := worker.ProcessNext(context.Background())
		if err != nil || !processed {
			t.Fatalf("ProcessNext() = %t, %v", processed, err)
		}
		assertEvidenceProcessingState(t, pool, versionID, "QUARANTINED", "NOT_READY")
		if len(objects.copies) != 0 {
			t.Fatalf("infected Evidence promotions = %+v", objects.copies)
		}
		var canonicalCount int
		if err := pool.QueryRow(context.Background(), `
			SELECT count(*) FROM evidence_version_states
			WHERE evidence_version_id = $1 AND canonical_object_metadata_id IS NOT NULL
		`, versionID).Scan(&canonicalCount); err != nil || canonicalCount != 0 {
			t.Fatalf("infected canonical metadata count = %d, err = %v", canonicalCount, err)
		}
	})

	t.Run("Inspection Attachment clean promotion", func(t *testing.T) {
		pool := canonicalDatabase(t, "clamav_pipeline_attachment_clean")
		objects := newMemoryObjectStore()
		attachmentID := completeAttachmentForScan(t, pool, objects, "clamav-clean")
		worker := evidenceworker.New(
			pool,
			objects,
			metadataScanner{result: scanner.Result{
				Clean:            true,
				EngineVersion:    "1.4.3",
				SignatureVersion: "27411",
				ScannedAt:        canonicalNow.Add(5 * time.Minute),
			}},
			evidenceworker.Config{
				WorkerID:         "clamav-attachment-worker",
				CanonicalBucket:  "evidence-clean",
				AttachmentBucket: "inspection-attachments",
				Clock:            uploadClock,
				IDGenerator:      deterministicIDs(),
			},
		)
		processed, err := worker.ProcessNext(context.Background())
		if err != nil || !processed {
			t.Fatalf("ProcessNext() = %t, %v", processed, err)
		}
		var canonicalBucket string
		if err := pool.QueryRow(context.Background(), `
			SELECT metadata.bucket_name
			FROM inspection_attachments attachment
			JOIN object_metadata metadata ON metadata.id = attachment.canonical_object_metadata_id
			WHERE attachment.id = $1
		`, attachmentID).Scan(&canonicalBucket); err != nil {
			t.Fatalf("read canonical Inspection Attachment metadata: %v", err)
		}
		if canonicalBucket != "inspection-attachments" ||
			len(objects.copies) != 1 ||
			objects.copies[0].DestinationBucket != "inspection-attachments" {
			t.Fatalf("Inspection Attachment promotion = bucket %q, copies %+v", canonicalBucket, objects.copies)
		}
	})
}

type metadataScanner struct {
	result scanner.Result
	err    error
}

func (candidate metadataScanner) Scan(context.Context, io.Reader) (scanner.Result, error) {
	return candidate.result, candidate.err
}

func liveObjectStores(t *testing.T) (*objectstore.MinIOStore, *objectstore.MinIOStore) {
	t.Helper()
	endpoint := os.Getenv("AVIA_TEST_OBJECT_STORE_ENDPOINT")
	apiAccessKey := os.Getenv("AVIA_TEST_OBJECT_STORE_API_ACCESS_KEY")
	apiSecretKey := os.Getenv("AVIA_TEST_OBJECT_STORE_API_SECRET_KEY")
	workerAccessKey := os.Getenv("AVIA_TEST_OBJECT_STORE_WORKER_ACCESS_KEY")
	workerSecretKey := os.Getenv("AVIA_TEST_OBJECT_STORE_WORKER_SECRET_KEY")
	if endpoint == "" ||
		apiAccessKey == "" ||
		apiSecretKey == "" ||
		workerAccessKey == "" ||
		workerSecretKey == "" {
		t.Skip("live API and worker object-store endpoints and credentials are required")
	}
	apiObjects, err := objectstore.NewMinIOStore(objectstore.MinIOConfig{
		Endpoint: endpoint, AccessKey: apiAccessKey, SecretKey: apiSecretKey, Region: "local",
		Clock: uploadClock,
	})
	if err != nil {
		t.Fatalf("create live API object store: %v", err)
	}
	workerObjects, err := objectstore.NewMinIOStore(objectstore.MinIOConfig{
		Endpoint: endpoint, AccessKey: workerAccessKey, SecretKey: workerSecretKey, Region: "local",
		Clock: uploadClock,
	})
	if err != nil {
		t.Fatalf("create live worker object store: %v", err)
	}
	if err := apiObjects.Check(context.Background()); err != nil {
		t.Fatalf("live API object-store readiness: %v", err)
	}
	if err := workerObjects.Check(context.Background()); err != nil {
		t.Fatalf("live worker object-store readiness: %v", err)
	}
	return apiObjects, workerObjects
}

func liveClamAV(t *testing.T) *scanner.ClamAV {
	t.Helper()
	address := os.Getenv("AVIA_TEST_CLAMAV_ADDRESS")
	if address == "" {
		t.Skip("AVIA_TEST_CLAMAV_ADDRESS is required for the live ClamAV pipeline gate")
	}
	client, err := scanner.NewClamAV(scanner.ClamAVConfig{
		Address:             address,
		DialTimeout:         time.Second,
		MaximumSignatureAge: 48 * time.Hour,
	})
	if err != nil {
		t.Fatalf("create live ClamAV adapter: %v", err)
	}
	return client
}

func completeLiveEvidenceForScan(
	t *testing.T,
	pool *database.Pool,
	apiObjects *objectstore.MinIOStore,
	suffix string,
	body []byte,
) (string, *evidence.UploadService) {
	t.Helper()
	uniqueSuffix := fmt.Sprintf("%s-%d", suffix, time.Now().UnixNano())
	findingID := "finding-" + uniqueSuffix
	seedFinding(t, pool, findingID, "LIVE-"+strings.ToUpper(suffix), "airline-xyz")
	if _, err := pool.Exec(
		context.Background(),
		"UPDATE findings SET status = 'EVIDENCE_REQUIRED' WHERE id = $1",
		findingID,
	); err != nil {
		t.Fatalf("seed live Evidence-required Finding: %v", err)
	}
	service := evidence.NewUploadService(pool, apiObjects, evidence.UploadServiceConfig{
		QuarantineBucket: "evidence-quarantine",
		CanonicalBucket:  "evidence-clean",
		MaximumByteSize:  25 * 1024 * 1024,
		InstructionTTL:   time.Minute,
		Clock:            uploadClock,
		IDGenerator:      liveIDGenerator(uniqueSuffix),
	})
	digest := sha256Digest(body)
	auditee := principal("auditee-xyz", "airline-xyz", "session-auditee", identity.RoleAuditee)
	begin, err := service.Begin(context.Background(), auditee, evidence.BeginUploadInput{
		OperationID:             "op-begin-" + uniqueSuffix,
		CorrelationID:           "corr-" + uniqueSuffix,
		FindingID:               findingID,
		ExpectedFindingRevision: 1,
		FileName:                "records-" + uniqueSuffix + ".pdf",
		DeclaredMediaType:       "application/pdf",
		ByteSize:                int64(len(body)),
		SHA256:                  digest,
	})
	if err != nil {
		t.Fatalf("begin live Evidence upload: %v", err)
	}
	request, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodPut,
		begin.UploadURL,
		bytes.NewReader(body),
	)
	if err != nil {
		t.Fatalf("create live Evidence PUT: %v", err)
	}
	request.Header.Set("Content-Type", begin.RequiredHeaders.ContentType)
	request.Header.Set("x-amz-meta-sha256", begin.RequiredHeaders.SHA256)
	request.Header.Set("If-None-Match", begin.RequiredHeaders.IfNoneMatch)
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("upload live Evidence bytes: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		contents, _ := io.ReadAll(response.Body)
		t.Fatalf("live Evidence PUT status = %d: %s", response.StatusCode, contents)
	}
	completed, err := service.Complete(context.Background(), auditee, evidence.CompleteUploadInput{
		OperationID:   "op-complete-" + uniqueSuffix,
		CorrelationID: "corr-" + uniqueSuffix,
		UploadID:      begin.UploadID,
		ByteSize:      int64(len(body)),
		SHA256:        digest,
	})
	if err != nil {
		t.Fatalf("complete live Evidence upload: %v", err)
	}
	return completed.EvidenceVersionID, service
}

func liveEvidenceWorker(
	pool *database.Pool,
	workerObjects *objectstore.MinIOStore,
	clamav *scanner.ClamAV,
	suffix string,
	afterExternalEffect func() error,
) *evidenceworker.Worker {
	return evidenceworker.New(
		pool,
		workerObjects,
		clamav,
		evidenceworker.Config{
			WorkerID:            "worker-" + suffix,
			CanonicalBucket:     "evidence-clean",
			AttachmentBucket:    "inspection-attachments",
			LeaseDuration:       time.Second,
			ScanTimeout:         30 * time.Second,
			MaximumAttempts:     3,
			Clock:               uploadClock,
			IDGenerator:         liveIDGenerator(suffix),
			AfterExternalEffect: afterExternalEffect,
		},
	)
}

func liveIDGenerator(suffix string) func(string) string {
	counter := 0
	return func(prefix string) string {
		counter++
		return fmt.Sprintf("%s-%s-%03d", prefix, suffix, counter)
	}
}

func assertLiveScanMetadata(t *testing.T, pool *database.Pool, versionID string, clean bool) {
	t.Helper()
	var engineVersion, signatureVersion string
	var scannedAt time.Time
	var reason *string
	if err := pool.QueryRow(context.Background(), `
		SELECT metadata.scan_engine_version, metadata.scan_signature_version,
		       metadata.scanned_at, state.scan_reason
		FROM evidence_versions version
		JOIN object_metadata metadata ON metadata.id = version.object_metadata_id
		JOIN evidence_version_states state ON state.evidence_version_id = version.id
		WHERE version.id = $1
	`, versionID).Scan(&engineVersion, &signatureVersion, &scannedAt, &reason); err != nil {
		t.Fatalf("read live scan metadata: %v", err)
	}
	if engineVersion == "" || signatureVersion == "" || scannedAt.IsZero() {
		t.Fatalf(
			"live scan metadata = engine %q, signatures %q, scanned at %s",
			engineVersion,
			signatureVersion,
			scannedAt,
		)
	}
	if clean && reason != nil {
		t.Fatalf("clean live scan reason = %q", *reason)
	}
	if !clean && (reason == nil || *reason == "") {
		t.Fatal("infected live scan reason is empty")
	}
}

func assertLiveCanonicalBytes(
	t *testing.T,
	objects *objectstore.MinIOStore,
	versionID string,
	expected []byte,
) {
	t.Helper()
	reader, info, err := objects.Open(
		context.Background(),
		"evidence-clean",
		"organizations/airline-xyz/canonical-evidence/"+versionID,
	)
	if err != nil {
		t.Fatalf("open live canonical Evidence: %v", err)
	}
	observed, readErr := io.ReadAll(reader)
	closeErr := reader.Close()
	if readErr != nil || closeErr != nil || info.Size != int64(len(expected)) ||
		!bytes.Equal(observed, expected) {
		t.Fatalf(
			"live canonical Evidence = size %d, read %v, close %v, bytes match %t",
			info.Size,
			readErr,
			closeErr,
			bytes.Equal(observed, expected),
		)
	}
}

func validPDFWithEmbeddedEICAR() []byte {
	eicar := []byte(
		"X5O!P%@AP[4\\PZX54(P^)7CC)7}$EICAR-STANDARD-ANTIVIRUS-TEST-FILE!$H+H*",
	)
	var document bytes.Buffer
	document.WriteString("%PDF-1.7\n")
	offsets := make([]int, 7)
	writeObject := func(number int, body string) {
		offsets[number] = document.Len()
		fmt.Fprintf(&document, "%d 0 obj\n%s\nendobj\n", number, body)
	}
	writeObject(
		1,
		"<</Type/Catalog/Pages 2 0 R/Names<</EmbeddedFiles<</Names[(eicar.com) 5 0 R]>>>>>>",
	)
	writeObject(2, "<</Type/Pages/Kids[3 0 R]/Count 1>>")
	writeObject(3, "<</Type/Page/Parent 2 0 R/MediaBox[0 0 100 100]/Contents 4 0 R>>")
	writeObject(4, "<</Length 0>>\nstream\n\nendstream")
	writeObject(5, "<</Type/Filespec/F(eicar.com)/EF<</F 6 0 R>>>>")
	offsets[6] = document.Len()
	fmt.Fprintf(
		&document,
		"6 0 obj\n<</Type/EmbeddedFile/Length %d>>\nstream\n",
		len(eicar),
	)
	document.Write(eicar)
	document.WriteString("\nendstream\nendobj\n")
	xrefOffset := document.Len()
	document.WriteString("xref\n0 7\n0000000000 65535 f \n")
	for objectNumber := 1; objectNumber <= 6; objectNumber++ {
		fmt.Fprintf(&document, "%010d 00000 n \n", offsets[objectNumber])
	}
	fmt.Fprintf(
		&document,
		"trailer\n<</Size 7/Root 1 0 R>>\nstartxref\n%d\n%%%%EOF\n",
		xrefOffset,
	)
	return document.Bytes()
}
