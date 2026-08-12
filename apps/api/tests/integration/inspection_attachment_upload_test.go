package integration_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/aviason/aviaSurveil/internal/identity"
	"github.com/aviason/aviaSurveil/internal/inspections/attachments"
)

func TestInspectionAttachmentUploadRequiresCurrentGrantAndNeverCreatesOfficialEvidence(t *testing.T) {
	pool := canonicalDatabase(t, "inspection_attachment_upload")
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO offline_grants (
			id, subject_id, device_id, package_id, inspection_id, assignment_revision, granted_at, expires_at,
			session_id, package_version, package_digest, allowed_command_types, assignment_scope, protocol_version
		) VALUES (
			'grant-attachment', 'inspector-cabin-001', 'managed-device-001', 'package-cabin-001', 'audit-cabin-001', 1, $1, $2,
			'session-inspector', 1, 'sha256:package-cabin-001', ARRAY['REGISTER_INSPECTION_ATTACHMENT'],
			'{"questionIds":["q-cabin-crew-training"]}', 1
		)
	`, canonicalNow, canonicalNow.Add(24*time.Hour)); err != nil {
		t.Fatalf("seed Inspection Attachment grant: %v", err)
	}
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO inspection_attachments (
			id, inspection_id, package_id, question_id, checklist_response_id, organization_id,
			created_by_subject_id, offline_grant_id, device_instance_id, file_name, declared_media_type,
			declared_size_bytes, declared_sha256, upload_state, scan_state, revision
		) VALUES (
			'attachment-cabin-001', 'audit-cabin-001', 'package-cabin-001', 'q-cabin-crew-training',
			'response-cabin-001', 'airline-xyz', 'inspector-cabin-001', 'grant-attachment', 'managed-device-001',
			'cabin-photo.png', 'image/png', 67, 'sha256:pending', 'PENDING', 'PENDING', 1
		)
	`); err != nil {
		t.Fatalf("seed Inspection Attachment: %v", err)
	}
	objects := newMemoryObjectStore()
	service := attachments.NewUploadService(pool, objects, attachments.UploadServiceConfig{
		QuarantineBucket: "avia-quarantine", MaximumByteSize: 25 * 1024 * 1024,
		InstructionTTL: 10 * time.Minute, Clock: uploadClock, IDGenerator: deterministicIDs(),
	})
	inspector := principal("inspector-cabin-001", "caa", "session-inspector", identity.RoleInspector)
	png := append([]byte("\x89PNG\r\n\x1a\n"), make([]byte, 59)...)
	digest := sha256Digest(png)
	begin, err := service.Begin(context.Background(), inspector, attachments.BeginUploadInput{
		OperationID: "op-attachment-begin", CorrelationID: "corr-attachment",
		InspectionAttachmentID: "attachment-cabin-001", PackageID: "package-cabin-001",
		ByteSize: int64(len(png)), SHA256: digest, FileName: "cabin-photo.png", DeclaredMediaType: "image/png",
	})
	if err != nil {
		t.Fatalf("begin Inspection Attachment upload: %v", err)
	}
	if begin.RequiredHeaders.IfNoneMatch != "*" {
		t.Fatalf("Inspection Attachment overwrite guard = %q", begin.RequiredHeaders.IfNoneMatch)
	}
	objects.Seed("avia-quarantine", begin.StagingObjectKey, "image/png", png, map[string]string{"sha256": digest})
	completed, err := service.Complete(context.Background(), inspector, attachments.CompleteUploadInput{
		OperationID: "op-attachment-complete", CorrelationID: "corr-attachment",
		UploadID: begin.UploadID, SHA256: digest, ByteSize: int64(len(png)),
	})
	if err != nil {
		t.Fatalf("complete Inspection Attachment upload: %v", err)
	}
	if completed.InspectionAttachmentID != "attachment-cabin-001" || completed.UploadState != "UPLOADED" || completed.ScanState != "PENDING" {
		t.Fatalf("attachment completion = %+v", completed)
	}
	if completed.InspectionAttachmentVersionID == "" || completed.Version != 1 {
		t.Fatalf("attachment immutable version identity = %+v", completed)
	}
	var evidenceCount int
	if err := pool.QueryRow(context.Background(), "SELECT count(*) FROM evidence_versions").Scan(&evidenceCount); err != nil || evidenceCount != 0 {
		t.Fatalf("Inspection Attachment implicitly created %d Evidence versions, err = %v", evidenceCount, err)
	}
	var currentVersionID, objectMetadataID, versionAttachmentID, sourceObjectMetadataID string
	var versionNumber int64
	if err := pool.QueryRow(context.Background(), `
		SELECT attachment.current_version_id, attachment.object_metadata_id,
		       version.inspection_attachment_id, version.version, version.source_object_metadata_id
		FROM inspection_attachments attachment
		JOIN inspection_attachment_versions version
		  ON version.id = attachment.current_version_id
		 AND version.inspection_attachment_id = attachment.id
		WHERE attachment.id = 'attachment-cabin-001'
	`).Scan(&currentVersionID, &objectMetadataID, &versionAttachmentID, &versionNumber, &sourceObjectMetadataID); err != nil {
		t.Fatalf("read immutable Inspection Attachment version: %v", err)
	}
	if currentVersionID != completed.InspectionAttachmentVersionID || versionAttachmentID != "attachment-cabin-001" ||
		versionNumber != 1 || sourceObjectMetadataID != objectMetadataID {
		t.Fatalf("immutable Inspection Attachment binding = version=%q attachment=%q number=%d source=%q object=%q",
			currentVersionID, versionAttachmentID, versionNumber, sourceObjectMetadataID, objectMetadataID)
	}
	view, err := service.Get(context.Background(), inspector, "attachment-cabin-001")
	if err != nil {
		t.Fatalf("get immutable Inspection Attachment history: %v", err)
	}
	if view.CurrentVersionID == nil || *view.CurrentVersionID != completed.InspectionAttachmentVersionID ||
		view.CurrentVersion != 1 || len(view.Versions) != 1 || !view.Versions[0].Current ||
		view.Versions[0].SourceObjectMetadataID != objectMetadataID {
		t.Fatalf("Inspection Attachment version projection = %+v", view)
	}
	if _, err := pool.Exec(context.Background(), `
		UPDATE inspection_attachment_versions SET file_name = 'tampered.png' WHERE id = $1
	`, completed.InspectionAttachmentVersionID); err == nil {
		t.Fatal("immutable Inspection Attachment version accepted an update")
	}
	if _, err := pool.Exec(context.Background(), `
		UPDATE inspection_attachments SET object_metadata_id = NULL WHERE id = 'attachment-cabin-001'
	`); err == nil {
		t.Fatal("completed Inspection Attachment detached its immutable version source object")
	}
}

func TestInspectionAttachmentUploadFailsClosedOnWrongScopeAndExpiredGrant(t *testing.T) {
	pool := canonicalDatabase(t, "inspection_attachment_scope")
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO offline_grants (
			id, subject_id, device_id, package_id, inspection_id, assignment_revision, granted_at, expires_at,
			session_id, package_version, package_digest, allowed_command_types, assignment_scope, protocol_version
		) VALUES (
			'grant-attachment-expired', 'inspector-cabin-001', 'managed-device-001', 'package-cabin-001', 'audit-cabin-001', 1, $1, $1,
			'session-inspector', 1, 'sha256:package-cabin-001', ARRAY['REGISTER_INSPECTION_ATTACHMENT'],
			'{"questionIds":["q-cabin-crew-training"]}', 1
		)
	`, canonicalNow.Add(-time.Hour)); err != nil {
		t.Fatalf("seed expired attachment grant: %v", err)
	}
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO inspection_attachments (
			id, inspection_id, package_id, question_id, checklist_response_id, organization_id,
			created_by_subject_id, offline_grant_id, device_instance_id, file_name, declared_media_type,
			declared_size_bytes, declared_sha256, upload_state, scan_state, revision
		) VALUES (
			'attachment-expired', 'audit-cabin-001', 'package-cabin-001', 'q-cabin-crew-training',
			'response-cabin-001', 'airline-xyz', 'inspector-cabin-001', 'grant-attachment-expired', 'managed-device-001',
			'cabin-photo.png', 'image/png', 67, 'sha256:pending', 'PENDING', 'PENDING', 1
		)
	`); err != nil {
		t.Fatalf("seed expired Inspection Attachment: %v", err)
	}
	service := attachments.NewUploadService(pool, newMemoryObjectStore(), attachments.UploadServiceConfig{
		QuarantineBucket: "avia-quarantine", MaximumByteSize: 25 * 1024 * 1024,
		InstructionTTL: time.Minute, Clock: uploadClock, IDGenerator: deterministicIDs(),
	})
	input := attachments.BeginUploadInput{
		OperationID: "op-attachment-expired", CorrelationID: "corr-attachment-expired",
		InspectionAttachmentID: "attachment-expired", PackageID: "package-cabin-001", ByteSize: 67,
		SHA256: "sha256:pending", FileName: "cabin-photo.png", DeclaredMediaType: "image/png",
	}
	if _, err := service.Begin(context.Background(), principal("inspector-other", "caa", "session-inspector", identity.RoleInspector), input); !errors.Is(err, attachments.ErrAttachmentForbidden) {
		t.Fatalf("wrong Inspector scope error = %v", err)
	}
	if _, err := service.Begin(context.Background(), principal("inspector-cabin-001", "caa", "session-inspector", identity.RoleInspector), input); !errors.Is(err, attachments.ErrAttachmentForbidden) {
		t.Fatalf("expired grant error = %v", err)
	}
}
