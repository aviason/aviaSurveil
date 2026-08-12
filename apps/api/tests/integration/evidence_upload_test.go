package integration_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/aviason/aviaSurveil/internal/evidence"
	"github.com/aviason/aviaSurveil/internal/identity"
	"github.com/aviason/aviaSurveil/internal/platform/idempotency"
)

func TestOfficialEvidenceUploadIsBoundedIdempotentVersionedAndDownloadGated(t *testing.T) {
	pool := canonicalDatabase(t, "evidence_upload")
	seedFinding(t, pool, "finding-upload", "OPS-2026-040", "airline-xyz")
	if _, err := pool.Exec(context.Background(), "UPDATE findings SET status = 'EVIDENCE_REQUIRED' WHERE id = 'finding-upload'"); err != nil {
		t.Fatalf("seed Evidence-required Finding: %v", err)
	}
	objects := newMemoryObjectStore()
	service := evidence.NewUploadService(pool, objects, evidence.UploadServiceConfig{
		QuarantineBucket: "avia-quarantine", CanonicalBucket: "avia-canonical",
		MaximumByteSize: 25 * 1024 * 1024, InstructionTTL: 10 * time.Minute,
		Clock: uploadClock, IDGenerator: deterministicIDs(),
	})
	auditee := principal("auditee-xyz", "airline-xyz", "session-auditee", identity.RoleAuditee)
	bodyOne := validPDF("version-one")
	digestOne := sha256Digest(bodyOne)
	beginInput := evidence.BeginUploadInput{
		OperationID: "op-evidence-begin-001", CorrelationID: "corr-evidence-001",
		FindingID: "finding-upload", ExpectedFindingRevision: 1,
		FileName: "training-records.pdf", DeclaredMediaType: "application/pdf",
		ByteSize: int64(len(bodyOne)), SHA256: digestOne,
	}
	begin, err := service.Begin(context.Background(), auditee, beginInput)
	if err != nil {
		t.Fatalf("begin Evidence upload: %v", err)
	}
	if begin.UploadID == "" || begin.StagingObjectKey == "" || begin.MaximumByteSize != 25*1024*1024 || !begin.ExpiresAt.Equal(canonicalNow.Add(10*time.Minute)) {
		t.Fatalf("upload instruction = %+v", begin)
	}
	if begin.RequiredHeaders.IfNoneMatch != "*" {
		t.Fatalf("Evidence upload overwrite guard = %q", begin.RequiredHeaders.IfNoneMatch)
	}
	replay, err := service.Begin(context.Background(), auditee, beginInput)
	if err != nil || replay != begin {
		t.Fatalf("begin replay = %+v, err = %v", replay, err)
	}
	changed := beginInput
	changed.FileName = "changed.pdf"
	if _, err := service.Begin(context.Background(), auditee, changed); !errors.Is(err, idempotency.ErrOperationIDReuse) {
		t.Fatalf("changed begin operation error = %v", err)
	}

	objects.Seed("avia-quarantine", begin.StagingObjectKey, "application/pdf", bodyOne, map[string]string{"sha256": digestOne})
	completeInput := evidence.CompleteUploadInput{
		OperationID: "op-evidence-complete-001", CorrelationID: "corr-evidence-001",
		UploadID: begin.UploadID, SHA256: digestOne, ByteSize: int64(len(bodyOne)),
	}
	completed, err := service.Complete(context.Background(), auditee, completeInput)
	if err != nil {
		t.Fatalf("complete Evidence upload: %v", err)
	}
	if completed.Version != 1 || completed.UploadState != evidence.UploadStateUploaded || completed.ScanState != evidence.ScanStatePending || completed.ReviewState != evidence.ReviewStateNotReady {
		t.Fatalf("completed Evidence = %+v", completed)
	}
	completedReplay, err := service.Complete(context.Background(), auditee, completeInput)
	if err != nil || completedReplay != completed {
		t.Fatalf("complete lost-ack replay = %+v, err = %v", completedReplay, err)
	}
	versions, err := service.ListVersions(context.Background(), auditee, "finding-upload")
	if err != nil || len(versions) != 1 || versions[0].ID != completed.EvidenceVersionID {
		t.Fatalf("Evidence versions = %+v, err = %v", versions, err)
	}
	if _, err := service.Download(context.Background(), auditee, completed.EvidenceVersionID); !errors.Is(err, evidence.ErrEvidenceNotReady) {
		t.Fatalf("pending Evidence download error = %v", err)
	}
	var findingStatus string
	var findingRevision int64
	if err := pool.QueryRow(context.Background(), "SELECT status, revision FROM findings WHERE id = 'finding-upload'").Scan(&findingStatus, &findingRevision); err != nil {
		t.Fatalf("read Finding after upload: %v", err)
	}
	if findingStatus != "EVIDENCE_SUBMITTED" || findingRevision != 2 {
		t.Fatalf("Finding after upload = %s revision %d", findingStatus, findingRevision)
	}
	for table, expected := range map[string]int{
		"evidence_versions":         1,
		"evidence_version_states":   1,
		"audit_events":              2,
		"authorized_sync_changes":   2,
		"outbox_messages":           2,
		"idempotency_responses":     2,
		"command_transaction_links": 2,
	} {
		var count int
		if err := pool.QueryRow(context.Background(), "SELECT count(*) FROM "+table).Scan(&count); err != nil || count != expected {
			t.Fatalf("%s count = %d, want %d, err = %v", table, count, expected, err)
		}
	}
}

func TestEvidenceUploadRejectsAuthorityTypeSizeAndObservedObjectMismatchWithoutVersion(t *testing.T) {
	pool := canonicalDatabase(t, "evidence_upload_rejections")
	seedFinding(t, pool, "finding-upload-reject", "OPS-2026-041", "airline-xyz")
	if _, err := pool.Exec(context.Background(), "UPDATE findings SET status = 'EVIDENCE_REQUIRED' WHERE id = 'finding-upload-reject'"); err != nil {
		t.Fatalf("seed Evidence-required Finding: %v", err)
	}
	objects := newMemoryObjectStore()
	service := evidence.NewUploadService(pool, objects, evidence.UploadServiceConfig{
		QuarantineBucket: "avia-quarantine", CanonicalBucket: "avia-canonical",
		MaximumByteSize: 25 * 1024 * 1024, InstructionTTL: time.Minute,
		Clock: uploadClock, IDGenerator: deterministicIDs(),
	})
	body := validPDF("mismatch")
	valid := evidence.BeginUploadInput{
		OperationID: "op-evidence-reject-valid", CorrelationID: "corr-evidence-reject",
		FindingID: "finding-upload-reject", ExpectedFindingRevision: 1,
		FileName: "record.pdf", DeclaredMediaType: "application/pdf", ByteSize: int64(len(body)), SHA256: sha256Digest(body),
	}
	otherAuditee := principal("auditee-other", "airline-other", "session-auditee", identity.RoleAuditee)
	if _, err := service.Begin(context.Background(), otherAuditee, valid); !errors.Is(err, evidence.ErrEvidenceForbidden) {
		t.Fatalf("cross-organization begin error = %v", err)
	}
	oversized := valid
	oversized.OperationID = "op-evidence-too-large"
	oversized.ByteSize = 25*1024*1024 + 1
	if _, err := service.Begin(context.Background(), principal("auditee-xyz", "airline-xyz", "session-auditee", identity.RoleAuditee), oversized); !errors.Is(err, evidence.ErrInvalidUpload) {
		t.Fatalf("oversized begin error = %v", err)
	}
	archive := valid
	archive.OperationID = "op-evidence-archive"
	archive.FileName = "records.zip"
	archive.DeclaredMediaType = "application/zip"
	if _, err := service.Begin(context.Background(), principal("auditee-xyz", "airline-xyz", "session-auditee", identity.RoleAuditee), archive); !errors.Is(err, evidence.ErrInvalidUpload) {
		t.Fatalf("archive begin error = %v", err)
	}

	auditee := principal("auditee-xyz", "airline-xyz", "session-auditee", identity.RoleAuditee)
	begin, err := service.Begin(context.Background(), auditee, valid)
	if err != nil {
		t.Fatalf("begin valid upload: %v", err)
	}
	objects.Seed("avia-quarantine", begin.StagingObjectKey, "image/png", []byte("not a PDF"), nil)
	if _, err := service.Complete(context.Background(), auditee, evidence.CompleteUploadInput{
		OperationID: "op-evidence-mismatch-complete", CorrelationID: "corr-evidence-reject",
		UploadID: begin.UploadID, SHA256: valid.SHA256, ByteSize: valid.ByteSize,
	}); !errors.Is(err, evidence.ErrObjectMismatch) {
		t.Fatalf("observed mismatch completion error = %v", err)
	}
	wrongBucketInput := valid
	wrongBucketInput.OperationID = "op-evidence-wrong-bucket"
	wrongBucket, err := service.Begin(context.Background(), auditee, wrongBucketInput)
	if err != nil {
		t.Fatalf("begin wrong-bucket upload: %v", err)
	}
	if _, err := pool.Exec(
		context.Background(),
		"UPDATE upload_sessions SET bucket_name = 'avia-canonical' WHERE id = $1",
		wrongBucket.UploadID,
	); err != nil {
		t.Fatalf("move upload session to wrong bucket: %v", err)
	}
	objects.Seed(
		"avia-canonical",
		wrongBucket.StagingObjectKey,
		"application/pdf",
		body,
		map[string]string{"sha256": valid.SHA256},
	)
	if _, err := service.Complete(context.Background(), auditee, evidence.CompleteUploadInput{
		OperationID: "op-evidence-wrong-bucket-complete", CorrelationID: "corr-evidence-reject",
		UploadID: wrongBucket.UploadID, SHA256: valid.SHA256, ByteSize: valid.ByteSize,
	}); !errors.Is(err, evidence.ErrObjectMismatch) {
		t.Fatalf("wrong-bucket completion error = %v", err)
	}
	var versions int
	if err := pool.QueryRow(context.Background(), "SELECT count(*) FROM evidence_versions WHERE finding_id = 'finding-upload-reject'").Scan(&versions); err != nil || versions != 0 {
		t.Fatalf("rejected Evidence version count = %d, err = %v", versions, err)
	}
}

func TestEvidenceUploadExpiresIncompleteSessionAndIssuesFreshNonOverwritingRetry(t *testing.T) {
	pool := canonicalDatabase(t, "evidence_upload_expiry")
	seedFinding(t, pool, "finding-upload-expiry", "OPS-2026-041", "airline-xyz")
	if _, err := pool.Exec(context.Background(), "UPDATE findings SET status = 'EVIDENCE_REQUIRED' WHERE id = 'finding-upload-expiry'"); err != nil {
		t.Fatalf("seed Evidence-required Finding: %v", err)
	}
	now := canonicalNow
	objects := newMemoryObjectStore()
	service := evidence.NewUploadService(pool, objects, evidence.UploadServiceConfig{
		QuarantineBucket: "avia-quarantine", CanonicalBucket: "avia-canonical",
		MaximumByteSize: 25 * 1024 * 1024, InstructionTTL: time.Minute,
		Clock: func() time.Time { return now }, IDGenerator: deterministicIDs(),
	})
	body := validPDF("expiry-retry")
	auditee := principal("auditee-xyz", "airline-xyz", "session-auditee", identity.RoleAuditee)
	input := evidence.BeginUploadInput{
		OperationID: "op-evidence-expiry-1", CorrelationID: "corr-evidence-expiry",
		FindingID: "finding-upload-expiry", ExpectedFindingRevision: 1,
		FileName: "records-2026-001.pdf", DeclaredMediaType: "application/pdf",
		ByteSize: int64(len(body)), SHA256: sha256Digest(body),
	}
	first, err := service.Begin(context.Background(), auditee, input)
	if err != nil {
		t.Fatalf("begin incomplete upload: %v", err)
	}
	now = canonicalNow.Add(2 * time.Minute)
	if expired, err := service.ReconcileExpired(context.Background()); err != nil || expired != 1 {
		t.Fatalf("reconcile incomplete upload = %d, err = %v", expired, err)
	}
	if _, err := service.Complete(context.Background(), auditee, evidence.CompleteUploadInput{
		OperationID: "op-evidence-expired-complete", CorrelationID: "corr-evidence-expiry",
		UploadID: first.UploadID, SHA256: input.SHA256, ByteSize: input.ByteSize,
	}); !errors.Is(err, evidence.ErrEvidenceForbidden) {
		t.Fatalf("expired completion error = %v", err)
	}
	input.OperationID = "op-evidence-expiry-2"
	second, err := service.Begin(context.Background(), auditee, input)
	if err != nil {
		t.Fatalf("begin fresh retry: %v", err)
	}
	if second.UploadID == first.UploadID || second.StagingObjectKey == first.StagingObjectKey || !second.ExpiresAt.After(first.ExpiresAt) {
		t.Fatalf("fresh retry reused expired instruction: first=%+v second=%+v", first, second)
	}
	var versions int
	if err := pool.QueryRow(context.Background(), "SELECT count(*) FROM evidence_versions WHERE finding_id = 'finding-upload-expiry'").Scan(&versions); err != nil || versions != 0 {
		t.Fatalf("incomplete retry Evidence version count = %d, err = %v", versions, err)
	}
}
