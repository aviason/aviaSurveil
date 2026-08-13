package integration_test

import (
	"context"
	"testing"
	"time"

	"github.com/aviason/aviaSurveil/internal/testprofile"
	"github.com/aviason/aviaSurveil/migrations"
)

func TestInspectionAttachmentNMinusOneUpgradeBackfillsImmutableCompletedVersion(t *testing.T) {
	pool := createTestDatabase(t, "inspection_attachment_n_minus_one")
	applyMigrationFilesThrough(t, pool, 39)
	if err := testprofile.Reset(context.Background(), pool, canonicalNow); err != nil {
		t.Fatalf("seed N-1 canonical profile: %v", err)
	}
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `
		INSERT INTO offline_grants (
			id, subject_id, device_id, package_id, inspection_id, assignment_revision, granted_at, expires_at,
			session_id, package_version, package_digest, allowed_command_types, assignment_scope, protocol_version
		) VALUES (
			'grant-attachment-n1', $1, 'n1-device', 'PKG-CAB-2026-001', 'AUD-2026-001', 1, $2, $3,
			'TEST-CANONICAL-INSPECTOR', 1, 'sha256:n1-package', ARRAY['REGISTER_INSPECTION_ATTACHMENT'],
			'{"questionIds":["CAB-GALLEY-001"]}', 1
		)
	`, testprofile.CanonicalInspectorSubjectID, canonicalNow, canonicalNow.Add(24*time.Hour)); err != nil {
		t.Fatalf("seed N-1 Inspection Attachment grant: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO object_metadata (
			id, aggregate_type, aggregate_id, organization_id, bucket_name, object_key, filename,
			declared_media_type, detected_media_type, sha256, size_bytes, scan_status, object_state, created_at
		) VALUES (
			'object-attachment-n1', 'inspection_attachment', 'attachment-n1', 'ORG-FLY-NAMIBIA',
			'avia-quarantine', 'organizations/ORG-FLY-NAMIBIA/attachments/n1.png', 'n1.png',
			'image/png', 'image/png', 'sha256:n1-object', 67, 'PENDING', 'QUARANTINED', $1
		)
	`, canonicalNow); err != nil {
		t.Fatalf("seed N-1 Inspection Attachment object: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO inspection_attachments (
			id, inspection_id, package_id, question_id, organization_id, created_by_subject_id,
			offline_grant_id, device_instance_id, file_name, declared_media_type, declared_size_bytes,
			declared_sha256, object_metadata_id, upload_state, scan_state, revision, created_at, updated_at
		) VALUES (
			'attachment-n1', 'AUD-2026-001', 'PKG-CAB-2026-001', 'CAB-GALLEY-001', 'ORG-FLY-NAMIBIA', $1,
			'grant-attachment-n1', 'n1-device', 'n1.png', 'image/png', 67, 'sha256:n1-object',
			'object-attachment-n1', 'UPLOADED', 'PENDING', 2, $2, $2
		)
	`, testprofile.CanonicalInspectorSubjectID, canonicalNow); err != nil {
		t.Fatalf("seed N-1 completed Inspection Attachment: %v", err)
	}

	if err := migrations.Apply(ctx, pool); err != nil {
		t.Fatalf("upgrade N-1 Inspection Attachment: %v", err)
	}
	var versionID, sourceMetadataID, currentMetadataID string
	var version int64
	if err := pool.QueryRow(context.Background(), `
		SELECT version.id, version.version, version.source_object_metadata_id, attachment.object_metadata_id
		FROM inspection_attachments attachment
		JOIN inspection_attachment_versions version
		  ON version.id = attachment.current_version_id
		 AND version.inspection_attachment_id = attachment.id
		WHERE attachment.id = 'attachment-n1'
	`).Scan(&versionID, &version, &sourceMetadataID, &currentMetadataID); err != nil {
		t.Fatalf("read N-1 immutable Inspection Attachment version: %v", err)
	}
	if versionID != "inspection-attachment-version:legacy:attachment-n1" || version != 1 ||
		sourceMetadataID != "object-attachment-n1" || currentMetadataID != sourceMetadataID {
		t.Fatalf("N-1 immutable attachment version = %q/%d source=%q current=%q",
			versionID, version, sourceMetadataID, currentMetadataID)
	}
	if _, err := pool.Exec(context.Background(), `
		UPDATE inspection_attachment_versions SET sha256 = 'sha256:tampered' WHERE id = $1
	`, versionID); err == nil {
		t.Fatal("N-1 immutable Inspection Attachment version accepted an update")
	}
}
