package integration_test

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/evidence"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/identity"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/inspections/attachments"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/platform/database"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/platform/objectstore"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/platform/scanner"
	evidenceworker "github.com/MarlonJD/aviaSurveil360/apps/api/internal/worker/evidence"
)

func TestEvidenceScanWorkerPromotesCleanExactVersionAndMakesItReviewable(t *testing.T) {
	pool := canonicalDatabase(t, "evidence_scan_clean")
	objects := newMemoryObjectStore()
	versionID := completeEvidenceForScan(t, pool, objects, "finding-scan-clean", "OPS-2026-050", validPDF("clean"))
	worker := evidenceworker.New(pool, objects, evidenceworker.SignatureScanner{}, evidenceworker.Config{
		WorkerID: "worker-clean", CanonicalBucket: "avia-canonical", LeaseDuration: time.Minute,
		Clock: uploadClock, IDGenerator: deterministicIDs(),
	})
	processed, err := worker.ProcessNext(context.Background())
	if err != nil || !processed {
		t.Fatalf("process clean Evidence = %v, err = %v", processed, err)
	}
	assertEvidenceProcessingState(t, pool, versionID, "CLEAN", "PENDING_CAA_REVIEW")
	var findingStatus string
	var findingRevision int64
	if err := pool.QueryRow(context.Background(), "SELECT status, revision FROM findings WHERE id = 'finding-scan-clean'").Scan(&findingStatus, &findingRevision); err != nil {
		t.Fatalf("read Finding after clean scan: %v", err)
	}
	if findingStatus != "PENDING_CAA_REVIEW" || findingRevision != 3 {
		t.Fatalf("Finding after clean scan = %s revision %d", findingStatus, findingRevision)
	}
	service := evidence.NewUploadService(pool, objects, evidence.UploadServiceConfig{
		QuarantineBucket: "avia-quarantine", CanonicalBucket: "avia-canonical", MaximumByteSize: 25 * 1024 * 1024,
		InstructionTTL: time.Minute, Clock: uploadClock, IDGenerator: deterministicIDs(),
	})
	download, err := service.Download(context.Background(), principal("auditee-xyz", "airline-xyz", "session-auditee", identity.RoleAuditee), versionID)
	if err != nil || !strings.HasPrefix(download.URL, "memory://download/avia-canonical/") {
		t.Fatalf("clean Evidence download = %+v, err = %v", download, err)
	}
	var deliveredAt *time.Time
	if err := pool.QueryRow(context.Background(), "SELECT delivered_at FROM outbox_messages WHERE topic = 'evidence.scan_requested'").Scan(&deliveredAt); err != nil || deliveredAt == nil {
		t.Fatalf("scan request delivery = %v, err = %v", deliveredAt, err)
	}
}

type managedMemoryObjectStore struct{ *memoryObjectStore }

func (*managedMemoryObjectStore) RequiresExactVersionIdentity() bool { return true }

func TestManagedEvidenceScanPersistsExactSourceAndCanonicalVersions(t *testing.T) {
	pool := canonicalDatabase(t, "evidence_scan_managed_exact")
	objects := &managedMemoryObjectStore{newMemoryObjectStore()}
	body := validPDF("managed-exact")
	versionID := completeEvidenceForScan(t, pool, objects, "finding-scan-managed-exact", "OPS-2026-054", body)

	var sourceBucket, sourceKey, sourceVersion, sourceETag string
	if err := pool.QueryRow(context.Background(), `
		SELECT metadata.bucket_name, metadata.object_key, metadata.object_version_id, metadata.object_etag
		FROM evidence_versions version
		JOIN object_metadata metadata ON metadata.id = version.object_metadata_id
		WHERE version.id = $1
	`, versionID).Scan(&sourceBucket, &sourceKey, &sourceVersion, &sourceETag); err != nil {
		t.Fatalf("read exact managed source identity: %v", err)
	}
	if sourceVersion == "" || sourceETag == "" {
		t.Fatal("managed upload did not persist an exact source identity")
	}
	objects.SetTags(sourceBucket, sourceKey, map[string]string{
		scanner.GuardDutyMalwareScanStatusTag: "NO_THREATS_FOUND",
	})
	provider, err := scanner.NewGuardDutyS3ResultProvider(objects, uploadClock)
	if err != nil {
		t.Fatalf("create managed result provider: %v", err)
	}
	worker := evidenceworker.New(pool, objects, nil, evidenceworker.Config{
		WorkerID: "worker-managed-exact", CanonicalBucket: "avia-canonical", LeaseDuration: time.Minute,
		Clock: uploadClock, IDGenerator: deterministicIDs(), ManagedResultProvider: provider, ScanBackend: "guardduty-s3",
	})
	if processed, err := worker.ProcessNext(context.Background()); err != nil || !processed {
		t.Fatalf("process exact managed Evidence = %v, err = %v", processed, err)
	}
	assertEvidenceProcessingState(t, pool, versionID, "CLEAN", "PENDING_CAA_REVIEW")

	var canonicalVersion, canonicalETag, canonicalHash string
	if err := pool.QueryRow(context.Background(), `
		SELECT object_version_id, object_etag, sha256
		FROM object_metadata
		WHERE aggregate_id = $1 AND object_state = 'CANONICAL'
	`, versionID).Scan(&canonicalVersion, &canonicalETag, &canonicalHash); err != nil {
		t.Fatalf("read exact canonical identity: %v", err)
	}
	if canonicalVersion == "" || canonicalETag == "" || canonicalHash != sha256Digest(body) {
		t.Fatalf("canonical identity = %q/%q/%q", canonicalVersion, canonicalETag, canonicalHash)
	}
}

func TestManagedEvidenceScanCrashRecoveryDoesNotDuplicateCanonicalVersion(t *testing.T) {
	pool := canonicalDatabase(t, "evidence_scan_managed_recovery")
	objects := &managedMemoryObjectStore{newMemoryObjectStore()}
	versionID := completeEvidenceForScan(t, pool, objects, "finding-scan-managed-recovery", "OPS-2026-055", validPDF("managed-recovery"))
	var sourceBucket, sourceKey string
	if err := pool.QueryRow(context.Background(), `
		SELECT metadata.bucket_name, metadata.object_key
		FROM evidence_versions version JOIN object_metadata metadata ON metadata.id = version.object_metadata_id
		WHERE version.id = $1
	`, versionID).Scan(&sourceBucket, &sourceKey); err != nil {
		t.Fatalf("read managed source: %v", err)
	}
	objects.SetTags(sourceBucket, sourceKey, map[string]string{scanner.GuardDutyMalwareScanStatusTag: "NO_THREATS_FOUND"})
	provider, _ := scanner.NewGuardDutyS3ResultProvider(objects, uploadClock)
	crash := errors.New("managed crash after exact copy")
	worker := evidenceworker.New(pool, objects, nil, evidenceworker.Config{
		WorkerID: "worker-managed-crash", CanonicalBucket: "avia-canonical", LeaseDuration: time.Minute,
		Clock: uploadClock, IDGenerator: deterministicIDs(), ManagedResultProvider: provider, ScanBackend: "guardduty-s3",
		AfterExternalEffect: func() error { return crash },
	})
	if processed, err := worker.ProcessNext(context.Background()); !processed || !errors.Is(err, crash) {
		t.Fatalf("managed crash window = %v, err = %v", processed, err)
	}
	if _, err := pool.Exec(context.Background(), `UPDATE outbox_messages SET lease_expires_at = $1 WHERE topic = 'evidence.scan_requested'`, canonicalNow.Add(-time.Minute)); err != nil {
		t.Fatalf("expire managed lease: %v", err)
	}
	recovery := evidenceworker.New(pool, objects, nil, evidenceworker.Config{
		WorkerID: "worker-managed-recovery", CanonicalBucket: "avia-canonical", LeaseDuration: time.Minute,
		Clock: uploadClock, IDGenerator: deterministicIDs(), ManagedResultProvider: provider, ScanBackend: "guardduty-s3",
	})
	if processed, err := recovery.ProcessNext(context.Background()); err != nil || !processed {
		t.Fatalf("recover managed scan = %v, err = %v", processed, err)
	}
	var canonicalRows int
	if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM object_metadata WHERE aggregate_id = $1 AND object_state = 'CANONICAL'`, versionID).Scan(&canonicalRows); err != nil || canonicalRows != 1 {
		t.Fatalf("managed canonical metadata rows = %d, err = %v", canonicalRows, err)
	}
}

func TestEvidenceScanWorkerKeepsRejectedObjectQuarantinedAndNotReviewable(t *testing.T) {
	pool := canonicalDatabase(t, "evidence_scan_quarantine")
	objects := newMemoryObjectStore()
	body := validPDF("EICAR-STANDARD-ANTIVIRUS-TEST-FILE")
	versionID := completeEvidenceForScan(t, pool, objects, "finding-scan-quarantine", "OPS-2026-051", body)
	worker := evidenceworker.New(pool, objects, evidenceworker.SignatureScanner{}, evidenceworker.Config{
		WorkerID: "worker-quarantine", CanonicalBucket: "avia-canonical", LeaseDuration: time.Minute,
		Clock: uploadClock, IDGenerator: deterministicIDs(),
	})
	if processed, err := worker.ProcessNext(context.Background()); err != nil || !processed {
		t.Fatalf("process quarantined Evidence = %v, err = %v", processed, err)
	}
	assertEvidenceProcessingState(t, pool, versionID, "QUARANTINED", "NOT_READY")
	var findingStatus string
	if err := pool.QueryRow(context.Background(), "SELECT status FROM findings WHERE id = 'finding-scan-quarantine'").Scan(&findingStatus); err != nil || findingStatus != "EVIDENCE_MORE_INFORMATION_REQUESTED" {
		t.Fatalf("Finding after rejected scan = %s, err = %v", findingStatus, err)
	}
	if len(objects.copies) != 0 {
		t.Fatalf("quarantined object was promoted: %+v", objects.copies)
	}
}

func TestEvidenceScanWorkerRecoversAfterCopyBeforeAcknowledgementWithoutDuplicateVersion(t *testing.T) {
	pool := canonicalDatabase(t, "evidence_scan_recovery")
	objects := newMemoryObjectStore()
	versionID := completeEvidenceForScan(t, pool, objects, "finding-scan-recovery", "OPS-2026-052", validPDF("recovery"))
	crash := errors.New("simulated crash after object copy")
	worker := evidenceworker.New(pool, objects, evidenceworker.SignatureScanner{}, evidenceworker.Config{
		WorkerID: "worker-crash", CanonicalBucket: "avia-canonical", LeaseDuration: time.Minute,
		Clock: uploadClock, IDGenerator: deterministicIDs(), AfterExternalEffect: func() error { return crash },
	})
	if processed, err := worker.ProcessNext(context.Background()); !processed || !errors.Is(err, crash) {
		t.Fatalf("crash window result = %v, err = %v", processed, err)
	}
	if len(objects.copies) != 1 {
		t.Fatalf("external copy count after crash = %d", len(objects.copies))
	}
	if _, err := pool.Exec(context.Background(), `
		UPDATE outbox_messages SET lease_expires_at = $1 WHERE topic = 'evidence.scan_requested'
	`, canonicalNow.Add(-time.Minute)); err != nil {
		t.Fatalf("expire worker lease: %v", err)
	}
	recovery := evidenceworker.New(pool, objects, evidenceworker.SignatureScanner{}, evidenceworker.Config{
		WorkerID: "worker-recovery", CanonicalBucket: "avia-canonical", LeaseDuration: time.Minute,
		Clock: uploadClock, IDGenerator: deterministicIDs(),
	})
	if processed, err := recovery.ProcessNext(context.Background()); err != nil || !processed {
		t.Fatalf("recover scan request = %v, err = %v", processed, err)
	}
	assertEvidenceProcessingState(t, pool, versionID, "CLEAN", "PENDING_CAA_REVIEW")
	if len(objects.copies) != 1 {
		t.Fatalf("recovery duplicated external copy, copies = %d", len(objects.copies))
	}
	var versions int
	if err := pool.QueryRow(context.Background(), "SELECT count(*) FROM evidence_versions WHERE id = $1", versionID).Scan(&versions); err != nil || versions != 1 {
		t.Fatalf("immutable Evidence version count = %d, err = %v", versions, err)
	}
}

func TestEvidenceScanWorkerMakesTerminalFailureAndTimeoutVisibleButNotReviewable(t *testing.T) {
	for _, testCase := range []struct {
		name    string
		scanner evidenceworker.Scanner
		timeout time.Duration
	}{
		{name: "scanner failure", scanner: failingScanner{err: errors.New("scanner unavailable")}, timeout: time.Minute},
		{name: "scanner timeout", scanner: blockingScanner{}, timeout: time.Millisecond},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			pool := canonicalDatabase(t, "evidence_scan_"+strings.ReplaceAll(testCase.name, " ", "_"))
			objects := newMemoryObjectStore()
			findingID := "finding-scan-" + strings.ReplaceAll(testCase.name, " ", "-")
			versionID := completeEvidenceForScan(t, pool, objects, findingID, "OPS-2026-053", validPDF(testCase.name))
			worker := evidenceworker.New(pool, objects, testCase.scanner, evidenceworker.Config{
				WorkerID: "worker-terminal", CanonicalBucket: "avia-canonical", LeaseDuration: time.Minute,
				ScanTimeout: testCase.timeout, MaximumAttempts: 1, Clock: uploadClock, IDGenerator: deterministicIDs(),
			})
			if processed, err := worker.ProcessNext(context.Background()); !processed || err == nil {
				t.Fatalf("terminal scan result = %v, err = %v", processed, err)
			}
			assertEvidenceProcessingState(t, pool, versionID, "FAILED", "NOT_READY")
			var findingStatus, terminalState string
			if err := pool.QueryRow(context.Background(), "SELECT status FROM findings WHERE id = $1", findingID).Scan(&findingStatus); err != nil || findingStatus != "EVIDENCE_MORE_INFORMATION_REQUESTED" {
				t.Fatalf("Finding after terminal scan = %s, err = %v", findingStatus, err)
			}
			if err := pool.QueryRow(context.Background(), "SELECT terminal_state FROM outbox_messages WHERE topic = 'evidence.scan_requested'").Scan(&terminalState); err != nil || terminalState != "FAILED" {
				t.Fatalf("terminal outbox state = %s, err = %v", terminalState, err)
			}
			service := evidence.NewUploadService(pool, objects, evidence.UploadServiceConfig{
				QuarantineBucket: "avia-quarantine", CanonicalBucket: "avia-canonical",
				MaximumByteSize: 25 * 1024 * 1024, InstructionTTL: time.Minute, Clock: uploadClock,
			})
			if _, err := service.Download(context.Background(), principal("auditee-xyz", "airline-xyz", "session-auditee", identity.RoleAuditee), versionID); !errors.Is(err, evidence.ErrEvidenceNotReady) {
				t.Fatalf("terminal-failure download error = %v", err)
			}
		})
	}
}

func TestInspectionAttachmentScanNeverCreatesOfficialEvidence(t *testing.T) {
	for _, testCase := range []struct {
		name                string
		scanner             evidenceworker.Scanner
		maximum             int
		expectedState       string
		expectError         bool
		expectTerminalEvent bool
		expectDownload      bool
	}{
		{name: "clean", scanner: evidenceworker.SignatureScanner{}, expectedState: "CLEAN", expectTerminalEvent: true, expectDownload: true},
		{name: "quarantined", scanner: quarantinedScanner{}, expectedState: "QUARANTINED", expectTerminalEvent: true},
		{name: "terminal failure", scanner: failingScanner{err: errors.New("scanner unavailable")}, maximum: 1, expectedState: "FAILED", expectError: true},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			pool := canonicalDatabase(t, "attachment_scan_"+strings.ReplaceAll(testCase.name, " ", "_"))
			objects := newMemoryObjectStore()
			attachmentID := completeAttachmentForScan(t, pool, objects, strings.ReplaceAll(testCase.name, " ", "-"))
			worker := evidenceworker.New(pool, objects, testCase.scanner, evidenceworker.Config{
				WorkerID: "worker-attachment", CanonicalBucket: "avia-canonical", LeaseDuration: time.Minute,
				MaximumAttempts: testCase.maximum, Clock: uploadClock, IDGenerator: deterministicIDs(),
			})
			processed, err := worker.ProcessNext(context.Background())
			if !processed || (err != nil) != testCase.expectError {
				t.Fatalf("process Inspection Attachment = %v, err = %v", processed, err)
			}
			var scanState string
			if err := pool.QueryRow(context.Background(), "SELECT scan_state FROM inspection_attachments WHERE id = $1", attachmentID).Scan(&scanState); err != nil || scanState != testCase.expectedState {
				t.Fatalf("Inspection Attachment scan state = %s, err = %v", scanState, err)
			}
			var evidenceCount int
			if err := pool.QueryRow(context.Background(), "SELECT count(*) FROM evidence_versions").Scan(&evidenceCount); err != nil || evidenceCount != 0 {
				t.Fatalf("Inspection Attachment created %d Evidence versions, err = %v", evidenceCount, err)
			}
			if testCase.expectTerminalEvent {
				assertInspectionAttachmentTerminalEvent(t, pool, attachmentID, testCase.expectedState)
			}
			uploads := attachments.NewUploadService(pool, objects, attachments.UploadServiceConfig{
				QuarantineBucket: "avia-quarantine", MaximumByteSize: 25 * 1024 * 1024,
				InstructionTTL: time.Minute, Clock: uploadClock, IDGenerator: deterministicIDs(),
			})
			download, downloadErr := uploads.Download(context.Background(), principal(
				"inspector-cabin-001", "caa", "session-inspector", identity.RoleInspector,
			), attachmentID)
			if testCase.expectDownload {
				if downloadErr != nil || !strings.HasPrefix(download.URL, "memory://download/avia-canonical/") {
					t.Fatalf("clean Inspection Attachment download = %+v, err = %v", download, downloadErr)
				}
			} else if !errors.Is(downloadErr, attachments.ErrAttachmentForbidden) {
				t.Fatalf("non-clean Inspection Attachment download error = %v", downloadErr)
			}
		})
	}
}

func assertInspectionAttachmentTerminalEvent(t *testing.T, pool *database.Pool, attachmentID, expectedState string) {
	t.Helper()
	var versionID string
	if err := pool.QueryRow(context.Background(), `
		SELECT current_version_id FROM inspection_attachments WHERE id = $1
	`, attachmentID).Scan(&versionID); err != nil || versionID == "" {
		t.Fatalf("read immutable Inspection Attachment version = %q, err = %v", versionID, err)
	}
	terminalAction := "inspection_attachment.scan_completed"
	if expectedState == "FAILED" {
		terminalAction = "inspection_attachment.scan_failed"
	}
	var auditState string
	if err := pool.QueryRow(context.Background(), `
		SELECT after_status
		FROM audit_events
		WHERE action = $1
		  AND entity_type = 'inspection_attachment_version'
		  AND entity_id = $2
	`, terminalAction, versionID).Scan(&auditState); err != nil || auditState != expectedState {
		t.Fatalf("Inspection Attachment terminal audit = %q, err = %v", auditState, err)
	}
	var syncPayload string
	if err := pool.QueryRow(context.Background(), `
		SELECT payload::text
		FROM authorized_sync_changes
		WHERE kind = 'inspection_attachment_version' AND entity_id = $1
		ORDER BY sequence_id DESC
		LIMIT 1
	`, versionID).Scan(&syncPayload); err != nil || !strings.Contains(syncPayload, `"scanState": "`+expectedState+`"`) {
		t.Fatalf("Inspection Attachment terminal sync = %s, err = %v", syncPayload, err)
	}
	var aggregateType, idempotencyKey, outboxPayload string
	if err := pool.QueryRow(context.Background(), `
		SELECT aggregate_type, idempotency_key, payload::text
		FROM outbox_messages
		WHERE topic = $2 AND aggregate_id = $1
	`, versionID, terminalAction).Scan(&aggregateType, &idempotencyKey, &outboxPayload); err != nil ||
		aggregateType != "inspection_attachment_version" ||
		idempotencyKey != terminalAction+":"+versionID ||
		!strings.Contains(outboxPayload, `"scanState": "`+expectedState+`"`) {
		t.Fatalf("Inspection Attachment terminal outbox = %q/%q/%s, err = %v", aggregateType, idempotencyKey, outboxPayload, err)
	}
}

type failingScanner struct{ err error }

func (scanner failingScanner) Scan(context.Context, io.Reader) (evidenceworker.ScanResult, error) {
	return evidenceworker.ScanResult{}, scanner.err
}

type quarantinedScanner struct{}

func (quarantinedScanner) Scan(context.Context, io.Reader) (evidenceworker.ScanResult, error) {
	return evidenceworker.ScanResult{
		Clean:            false,
		Reason:           "deterministic quarantine",
		EngineVersion:    "test-engine",
		SignatureVersion: "test-signatures",
		ScannedAt:        canonicalNow,
	}, nil
}

type blockingScanner struct{}

func (blockingScanner) Scan(ctx context.Context, _ io.Reader) (evidenceworker.ScanResult, error) {
	<-ctx.Done()
	return evidenceworker.ScanResult{}, ctx.Err()
}

func completeAttachmentForScan(t *testing.T, pool *database.Pool, objects *memoryObjectStore, suffix string) string {
	t.Helper()
	attachmentID := "attachment-scan-" + suffix
	grantID := "grant-scan-" + suffix
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO offline_grants (
			id, subject_id, device_id, package_id, inspection_id, assignment_revision, granted_at, expires_at,
			session_id, package_version, package_digest, allowed_command_types, assignment_scope, protocol_version
		) VALUES ($1, 'inspector-cabin-001', 'managed-device-001', 'package-cabin-001', 'audit-cabin-001', 1, $2, $3,
			'session-inspector', 1, 'sha256:package-cabin-001', ARRAY['REGISTER_INSPECTION_ATTACHMENT'],
			'{"questionIds":["q-cabin-crew-training"]}', 1)
	`, grantID, canonicalNow, canonicalNow.Add(24*time.Hour)); err != nil {
		t.Fatalf("seed scan grant: %v", err)
	}
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO inspection_attachments (
			id, inspection_id, package_id, question_id, checklist_response_id, organization_id,
			created_by_subject_id, offline_grant_id, device_instance_id, file_name, declared_media_type,
			declared_size_bytes, declared_sha256, upload_state, scan_state, revision
		) VALUES ($1, 'audit-cabin-001', 'package-cabin-001', 'q-cabin-crew-training',
			'response-cabin-001', 'airline-xyz', 'inspector-cabin-001', $2, 'managed-device-001',
			'cabin-photo.png', 'image/png', 67, 'sha256:pending', 'PENDING', 'PENDING', 1)
	`, attachmentID, grantID); err != nil {
		t.Fatalf("seed scan Inspection Attachment: %v", err)
	}
	png := append([]byte("\x89PNG\r\n\x1a\n"), make([]byte, 59)...)
	digest := sha256Digest(png)
	service := attachments.NewUploadService(pool, objects, attachments.UploadServiceConfig{
		QuarantineBucket: "avia-quarantine", MaximumByteSize: 25 * 1024 * 1024,
		InstructionTTL: time.Minute, Clock: uploadClock, IDGenerator: deterministicIDs(),
	})
	inspector := principal("inspector-cabin-001", "caa", "session-inspector", identity.RoleInspector)
	begin, err := service.Begin(context.Background(), inspector, attachments.BeginUploadInput{
		OperationID: "op-attachment-begin-" + suffix, CorrelationID: "corr-attachment-" + suffix,
		InspectionAttachmentID: attachmentID, PackageID: "package-cabin-001",
		ByteSize: int64(len(png)), SHA256: digest, FileName: "cabin-photo.png", DeclaredMediaType: "image/png",
	})
	if err != nil {
		t.Fatalf("begin scan Inspection Attachment: %v", err)
	}
	objects.Seed("avia-quarantine", begin.StagingObjectKey, "image/png", png, map[string]string{"sha256": digest})
	if _, err := service.Complete(context.Background(), inspector, attachments.CompleteUploadInput{
		OperationID: "op-attachment-complete-" + suffix, CorrelationID: "corr-attachment-" + suffix,
		UploadID: begin.UploadID, SHA256: digest, ByteSize: int64(len(png)),
	}); err != nil {
		t.Fatalf("complete scan Inspection Attachment: %v", err)
	}
	return attachmentID
}

type seedableObjectStore interface {
	objectstore.Store
	Seed(string, string, string, []byte, map[string]string)
}

func completeEvidenceForScan(t *testing.T, pool *database.Pool, objects seedableObjectStore, findingID, reference string, body []byte) string {
	t.Helper()
	seedFinding(t, pool, findingID, reference, "airline-xyz")
	if _, err := pool.Exec(context.Background(), "UPDATE findings SET status = 'EVIDENCE_REQUIRED' WHERE id = $1", findingID); err != nil {
		t.Fatalf("seed Evidence-required Finding: %v", err)
	}
	service := evidence.NewUploadService(pool, objects, evidence.UploadServiceConfig{
		QuarantineBucket: "avia-quarantine", CanonicalBucket: "avia-canonical", MaximumByteSize: 25 * 1024 * 1024,
		InstructionTTL: time.Minute, Clock: uploadClock, IDGenerator: deterministicIDs(),
	})
	auditee := principal("auditee-xyz", "airline-xyz", "session-auditee", identity.RoleAuditee)
	digest := sha256Digest(body)
	begin, err := service.Begin(context.Background(), auditee, evidence.BeginUploadInput{
		OperationID: "op-begin-" + findingID, CorrelationID: "corr-" + findingID, FindingID: findingID,
		ExpectedFindingRevision: 1, FileName: "records.pdf", DeclaredMediaType: "application/pdf",
		ByteSize: int64(len(body)), SHA256: digest,
	})
	if err != nil {
		t.Fatalf("begin Evidence for scan: %v", err)
	}
	objects.Seed("avia-quarantine", begin.StagingObjectKey, "application/pdf", body, map[string]string{"sha256": digest})
	completed, err := service.Complete(context.Background(), auditee, evidence.CompleteUploadInput{
		OperationID: "op-complete-" + findingID, CorrelationID: "corr-" + findingID,
		UploadID: begin.UploadID, ByteSize: int64(len(body)), SHA256: digest,
	})
	if err != nil {
		t.Fatalf("complete Evidence for scan: %v", err)
	}
	return completed.EvidenceVersionID
}

func assertEvidenceProcessingState(t *testing.T, pool *database.Pool, versionID, scanState, reviewState string) {
	t.Helper()
	var actualScan, actualReview string
	if err := pool.QueryRow(context.Background(), `
		SELECT scan_state, review_state FROM evidence_version_states WHERE evidence_version_id = $1
	`, versionID).Scan(&actualScan, &actualReview); err != nil {
		t.Fatalf("read Evidence processing state: %v", err)
	}
	if actualScan != scanState || actualReview != reviewState {
		t.Fatalf("Evidence state = %s/%s, want %s/%s", actualScan, actualReview, scanState, reviewState)
	}
}

var _ objectstore.Store = (*memoryObjectStore)(nil)
