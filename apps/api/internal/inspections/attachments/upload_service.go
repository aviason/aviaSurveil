package attachments

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/identity"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/platform/database"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/platform/idempotency"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/platform/objectstore"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/platform/uploadpolicy"
	fieldsync "github.com/MarlonJD/aviaSurveil360/apps/api/internal/sync"
	"github.com/jackc/pgx/v5"
)

var (
	ErrAttachmentForbidden = errors.New("inspection attachment access forbidden")
	ErrInvalidUpload       = errors.New("invalid inspection attachment upload")
	ErrObjectMismatch      = errors.New("uploaded inspection attachment does not match declaration")
)

type UploadServiceConfig struct {
	QuarantineBucket string
	MaximumByteSize  int64
	InstructionTTL   time.Duration
	Clock            func() time.Time
	IDGenerator      func(string) string
}

type UploadService struct {
	pool             *database.Pool
	objects          objectstore.Store
	quarantineBucket string
	maximumByteSize  int64
	instructionTTL   time.Duration
	clock            func() time.Time
	idGenerator      func(string) string
}

func NewUploadService(pool *database.Pool, objects objectstore.Store, config UploadServiceConfig) *UploadService {
	clock := config.Clock
	if clock == nil {
		clock = time.Now
	}
	idGenerator := config.IDGenerator
	if idGenerator == nil {
		idGenerator = randomID
	}
	return &UploadService{
		pool: pool, objects: objects, quarantineBucket: config.QuarantineBucket,
		maximumByteSize: config.MaximumByteSize, instructionTTL: config.InstructionTTL,
		clock: clock, idGenerator: idGenerator,
	}
}

type RequiredHeaders struct {
	ContentType string
	SHA256      string
	IfNoneMatch string
}

func (headers RequiredHeaders) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]string{
		"Content-Type":      headers.ContentType,
		"x-amz-meta-sha256": headers.SHA256,
		"If-None-Match":     headers.IfNoneMatch,
	})
}

func (headers *RequiredHeaders) UnmarshalJSON(value []byte) error {
	var decoded map[string]string
	if err := json.Unmarshal(value, &decoded); err != nil {
		return err
	}
	headers.ContentType = decoded["Content-Type"]
	headers.SHA256 = decoded["x-amz-meta-sha256"]
	headers.IfNoneMatch = decoded["If-None-Match"]
	return nil
}

type BeginUploadInput struct {
	OperationID            string `json:"operationId"`
	CorrelationID          string `json:"correlationId"`
	InspectionAttachmentID string `json:"inspectionAttachmentId"`
	PackageID              string `json:"packageId"`
	FileName               string `json:"fileName"`
	DeclaredMediaType      string `json:"declaredMediaType"`
	ByteSize               int64  `json:"byteSize"`
	SHA256                 string `json:"sha256"`
}

type BeginUploadOutput struct {
	UploadID         string          `json:"uploadId"`
	StagingObjectKey string          `json:"stagingObjectKey"`
	UploadURL        string          `json:"uploadUrl"`
	RequiredHeaders  RequiredHeaders `json:"requiredHeaders"`
	ExpiresAt        time.Time       `json:"expiresAt"`
	MaximumByteSize  int64           `json:"maximumByteSize"`
}

func (service *UploadService) Begin(ctx context.Context, actor identity.Principal, input BeginUploadInput) (BeginUploadOutput, error) {
	if !actor.HasRole(identity.RoleInspector) || actor.SubjectID == "" || actor.SessionID == "" {
		return BeginUploadOutput{}, ErrAttachmentForbidden
	}
	if input.OperationID == "" || input.CorrelationID == "" || input.InspectionAttachmentID == "" || input.PackageID == "" {
		return BeginUploadOutput{}, ErrInvalidUpload
	}
	var organizationID, createdBy, grantID, deviceID, packageID, uploadState string
	if err := service.pool.QueryRow(ctx, `
		SELECT organization_id, created_by_subject_id, offline_grant_id, device_instance_id, package_id, upload_state
		FROM inspection_attachments WHERE id = $1
	`, input.InspectionAttachmentID).Scan(&organizationID, &createdBy, &grantID, &deviceID, &packageID, &uploadState); err != nil {
		return BeginUploadOutput{}, ErrAttachmentForbidden
	}
	if createdBy != actor.SubjectID || packageID != input.PackageID || uploadState != "PENDING" {
		return BeginUploadOutput{}, ErrAttachmentForbidden
	}
	grantService := fieldsync.NewGrantService(service.pool, fieldsync.GrantDependencies{Clock: service.clock})
	if err := grantService.Authorize(ctx, actor, fieldsync.AuthorizationInput{
		GrantID: grantID, PackageID: packageID, DeviceInstanceID: deviceID,
		ServerNow: service.clock().UTC(), CommandType: "REGISTER_INSPECTION_ATTACHMENT",
	}); err != nil {
		return BeginUploadOutput{}, fmt.Errorf("%w: %v", ErrAttachmentForbidden, err)
	}
	if err := uploadpolicy.ValidateDeclaration(input.FileName, input.DeclaredMediaType, input.ByteSize, service.maximumByteSize, input.SHA256); err != nil {
		return BeginUploadOutput{}, fmt.Errorf("%w: %v", ErrInvalidUpload, err)
	}
	semanticHash, err := idempotency.SemanticHash(input)
	if err != nil {
		return BeginUploadOutput{}, err
	}
	scope := actor.SubjectID + ":begin_inspection_attachment_upload"
	var output BeginUploadOutput
	err = database.WithinTransaction(ctx, service.pool, func(ctx context.Context, transaction pgx.Tx) error {
		replayed, err := loadIdempotent(ctx, transaction, scope, input.OperationID, semanticHash, &output)
		if err != nil || replayed {
			return err
		}
		var currentOrganizationID, currentPackageID, currentUploadState, inspectionStatus, checklistStatus string
		if err := transaction.QueryRow(ctx, `
			SELECT attachment.organization_id, attachment.package_id, attachment.upload_state,
			       inspection.status, checklist.status
			FROM inspection_attachments attachment
			JOIN inspections inspection ON inspection.id = attachment.inspection_id
			JOIN inspection_checklists checklist ON checklist.inspection_id = inspection.id
			WHERE attachment.id = $1
			FOR UPDATE OF attachment`, input.InspectionAttachmentID).Scan(
			&currentOrganizationID, &currentPackageID, &currentUploadState,
			&inspectionStatus, &checklistStatus); err != nil {
			return ErrAttachmentForbidden
		}
		if currentOrganizationID != organizationID || currentPackageID != input.PackageID ||
			currentUploadState != "PENDING" || inspectionStatus != "IN_PROGRESS" || checklistStatus != "IN_PROGRESS" {
			return ErrAttachmentForbidden
		}
		var activeUpload bool
		if err := transaction.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM upload_sessions
				WHERE aggregate_id = $1 AND upload_kind = 'INSPECTION_ATTACHMENT'
				  AND upload_state = 'PENDING' AND expires_at > $2
			)`, input.InspectionAttachmentID, service.clock().UTC()).Scan(&activeUpload); err != nil {
			return err
		}
		if activeUpload {
			return ErrAttachmentForbidden
		}
		now := service.clock().UTC()
		expiresAt := now.Add(service.instructionTTL)
		uploadID := service.idGenerator("upload")
		key := fmt.Sprintf("organizations/%s/inspection-attachments/%s/%s", organizationID, input.InspectionAttachmentID, uploadID)
		instruction, err := service.objects.CreatePutInstruction(ctx, objectstore.PutRequest{
			Bucket: service.quarantineBucket, Key: key, ExpiresAt: expiresAt,
			RequiredHeaders: map[string]string{
				"Content-Type":      input.DeclaredMediaType,
				"x-amz-meta-sha256": input.SHA256,
				"If-None-Match":     "*",
			},
		})
		if err != nil {
			return err
		}
		output = BeginUploadOutput{
			UploadID: uploadID, StagingObjectKey: key, UploadURL: instruction.URL,
			RequiredHeaders: RequiredHeaders{
				ContentType: input.DeclaredMediaType,
				SHA256:      input.SHA256,
				IfNoneMatch: "*",
			},
			ExpiresAt: expiresAt, MaximumByteSize: service.maximumByteSize,
		}
		if _, err := transaction.Exec(ctx, `
			INSERT INTO upload_sessions (
				id, upload_kind, aggregate_id, organization_id, initiated_by_subject_id, bucket_name,
				staging_object_key, file_name, declared_media_type, declared_size_bytes, declared_sha256,
				upload_state, expires_at, created_at
			) VALUES ($1, 'INSPECTION_ATTACHMENT', $2, $3, $4, $5, $6, $7, $8, $9, $10, 'PENDING', $11, $12)
		`, uploadID, input.InspectionAttachmentID, organizationID, actor.SubjectID, service.quarantineBucket,
			key, input.FileName, input.DeclaredMediaType, input.ByteSize, input.SHA256, expiresAt, now); err != nil {
			return err
		}
		if _, err := transaction.Exec(ctx, `
			UPDATE inspection_attachments SET file_name = $2, declared_media_type = $3,
			       declared_size_bytes = $4, declared_sha256 = $5, updated_at = $6 WHERE id = $1
		`, input.InspectionAttachmentID, input.FileName, input.DeclaredMediaType, input.ByteSize, input.SHA256, now); err != nil {
			return err
		}
		return saveIdempotent(ctx, transaction, scope, input.OperationID, semanticHash, output, now)
	})
	return output, err
}

type CompleteUploadInput struct {
	OperationID   string `json:"operationId"`
	CorrelationID string `json:"correlationId"`
	UploadID      string `json:"uploadId"`
	SHA256        string `json:"sha256"`
	ByteSize      int64  `json:"byteSize"`
}

type CompleteUploadOutput struct {
	InspectionAttachmentID        string `json:"inspectionAttachmentId"`
	InspectionAttachmentVersionID string `json:"inspectionAttachmentVersionId"`
	Version                       int64  `json:"version"`
	UploadState                   string `json:"uploadState"`
	ScanState                     string `json:"scanState"`
}

type AttachmentView struct {
	ID                  string                  `json:"id"`
	InspectionID        string                  `json:"inspectionId"`
	PackageID           string                  `json:"packageId"`
	QuestionID          string                  `json:"questionId"`
	ChecklistResponseID string                  `json:"checklistResponseId"`
	FileName            string                  `json:"fileName"`
	UploadState         string                  `json:"uploadState"`
	ScanState           string                  `json:"scanState"`
	Revision            int64                   `json:"revision"`
	CurrentVersionID    *string                 `json:"currentVersionId"`
	CurrentVersion      int64                   `json:"currentVersion"`
	Versions            []AttachmentVersionView `json:"versions"`
}

type AttachmentVersionView struct {
	ID                     string    `json:"id"`
	Version                int64     `json:"version"`
	SourceObjectMetadataID string    `json:"sourceObjectMetadataId"`
	FileName               string    `json:"fileName"`
	MediaType              string    `json:"mediaType"`
	SHA256                 string    `json:"sha256"`
	SizeBytes              int64     `json:"sizeBytes"`
	SubmittedBySubjectID   string    `json:"submittedBySubjectId"`
	SubmittedAt            time.Time `json:"submittedAt"`
	Current                bool      `json:"current"`
}

func (service *UploadService) Get(ctx context.Context, actor identity.Principal, attachmentID string) (AttachmentView, error) {
	if !canAccessAttachment(actor) || attachmentID == "" {
		return AttachmentView{}, ErrAttachmentForbidden
	}
	var view AttachmentView
	var memberSubject string
	if err := service.pool.QueryRow(ctx, `
		SELECT attachment.id, attachment.inspection_id, attachment.package_id,
		       attachment.question_id, COALESCE(attachment.checklist_response_id, ''),
		       attachment.file_name, attachment.upload_state, attachment.scan_state,
		       attachment.revision, attachment.current_version_id,
		       COALESCE(current_version.version, 0), member.subject_id
		FROM inspection_attachments attachment
		LEFT JOIN inspection_attachment_versions current_version
		  ON current_version.id = attachment.current_version_id
		 AND current_version.inspection_attachment_id = attachment.id
		JOIN audit_assignments assignment
		  ON assignment.inspection_id = attachment.inspection_id
		 AND assignment.tombstoned_at IS NULL
		JOIN audit_team_members member
		  ON member.assignment_id = assignment.id
		 AND member.removed_at IS NULL
		 AND member.member_role IN ('INSPECTOR', 'LEAD_INSPECTOR')
		WHERE attachment.id = $1 AND member.subject_id = $2
	`, attachmentID, actor.SubjectID).Scan(
		&view.ID, &view.InspectionID, &view.PackageID, &view.QuestionID,
		&view.ChecklistResponseID, &view.FileName, &view.UploadState,
		&view.ScanState, &view.Revision, &view.CurrentVersionID, &view.CurrentVersion, &memberSubject,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return AttachmentView{}, ErrAttachmentForbidden
		}
		return AttachmentView{}, err
	}
	rows, err := service.pool.Query(ctx, `
		SELECT version.id, version.version, version.source_object_metadata_id,
		       version.file_name, version.media_type, version.sha256, version.size_bytes,
		       version.submitted_by_subject_id, version.submitted_at,
		       (version.id = attachment.current_version_id)
		FROM inspection_attachment_versions version
		JOIN inspection_attachments attachment ON attachment.id = version.inspection_attachment_id
		WHERE version.inspection_attachment_id = $1
		ORDER BY version.version DESC
	`, attachmentID)
	if err != nil {
		return AttachmentView{}, err
	}
	defer rows.Close()
	view.Versions = make([]AttachmentVersionView, 0)
	for rows.Next() {
		var item AttachmentVersionView
		if err := rows.Scan(&item.ID, &item.Version, &item.SourceObjectMetadataID, &item.FileName,
			&item.MediaType, &item.SHA256, &item.SizeBytes, &item.SubmittedBySubjectID,
			&item.SubmittedAt, &item.Current); err != nil {
			return AttachmentView{}, err
		}
		view.Versions = append(view.Versions, item)
	}
	if err := rows.Err(); err != nil {
		return AttachmentView{}, err
	}
	return view, nil
}

func (service *UploadService) Download(ctx context.Context, actor identity.Principal, attachmentID string) (objectstore.GetInstruction, error) {
	if !canAccessAttachment(actor) || attachmentID == "" {
		return objectstore.GetInstruction{}, ErrAttachmentForbidden
	}
	var organizationID, bucket, key, fileName string
	if err := service.pool.QueryRow(ctx, `
		SELECT attachment.organization_id, metadata.bucket_name, metadata.object_key, metadata.filename
		FROM inspection_attachments attachment
		JOIN audit_assignments assignment
		  ON assignment.inspection_id = attachment.inspection_id
		 AND assignment.tombstoned_at IS NULL
		JOIN audit_team_members member
		  ON member.assignment_id = assignment.id
		 AND member.subject_id = $2
		 AND member.removed_at IS NULL
		 AND member.member_role IN ('INSPECTOR', 'LEAD_INSPECTOR')
		JOIN inspection_attachment_versions version
		  ON version.id = attachment.current_version_id
		 AND version.inspection_attachment_id = attachment.id
		JOIN object_metadata metadata
		  ON metadata.id = attachment.canonical_object_metadata_id
		 AND metadata.scan_status = 'CLEAN'
		 AND metadata.object_state = 'CANONICAL'
		WHERE attachment.id = $1
		  AND attachment.upload_state = 'UPLOADED'
		  AND attachment.scan_state = 'CLEAN'
	`, attachmentID, actor.SubjectID).Scan(&organizationID, &bucket, &key, &fileName); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return objectstore.GetInstruction{}, ErrAttachmentForbidden
		}
		return objectstore.GetInstruction{}, err
	}
	if organizationID == "" || bucket == "" || key == "" {
		return objectstore.GetInstruction{}, ErrAttachmentForbidden
	}
	return service.objects.CreateGetInstruction(ctx, objectstore.GetRequest{
		Bucket: bucket, Key: key, DownloadFileName: fileName,
		ExpiresAt: service.clock().UTC().Add(5 * time.Minute),
	})
}

func canAccessAttachment(actor identity.Principal) bool {
	return actor.SubjectID != "" && (actor.HasRole(identity.RoleInspector) || actor.HasRole(identity.RoleLeadInspector))
}

func (service *UploadService) Complete(ctx context.Context, actor identity.Principal, input CompleteUploadInput) (CompleteUploadOutput, error) {
	if !actor.HasRole(identity.RoleInspector) || actor.SubjectID == "" || input.OperationID == "" || input.UploadID == "" {
		return CompleteUploadOutput{}, ErrAttachmentForbidden
	}
	semanticHash, err := idempotency.SemanticHash(input)
	if err != nil {
		return CompleteUploadOutput{}, err
	}
	scope := actor.SubjectID + ":complete_inspection_attachment_upload"
	var output CompleteUploadOutput
	err = database.WithinTransaction(ctx, service.pool, func(ctx context.Context, transaction pgx.Tx) error {
		replayed, err := loadIdempotent(ctx, transaction, scope, input.OperationID, semanticHash, &output)
		if err != nil || replayed {
			return err
		}
		var attachmentID, organizationID, initiatedBy, bucket, key, fileName, mediaType, digest, state string
		var size int64
		var expiresAt time.Time
		if err := transaction.QueryRow(ctx, `
			SELECT aggregate_id, organization_id, initiated_by_subject_id, bucket_name, staging_object_key,
			       file_name, declared_media_type, declared_size_bytes, declared_sha256, upload_state, expires_at
			FROM upload_sessions WHERE id = $1 AND upload_kind = 'INSPECTION_ATTACHMENT' FOR UPDATE
		`, input.UploadID).Scan(&attachmentID, &organizationID, &initiatedBy, &bucket, &key, &fileName,
			&mediaType, &size, &digest, &state, &expiresAt); err != nil {
			return ErrAttachmentForbidden
		}
		now := service.clock().UTC()
		if initiatedBy != actor.SubjectID || state != "PENDING" || now.After(expiresAt) {
			return ErrAttachmentForbidden
		}
		var currentPackageID, currentUploadState, inspectionStatus, checklistStatus string
		if err := transaction.QueryRow(ctx, `
			SELECT attachment.package_id, attachment.upload_state,
			       inspection.status, checklist.status
			FROM inspection_attachments attachment
			JOIN inspections inspection ON inspection.id = attachment.inspection_id
			JOIN inspection_checklists checklist ON checklist.inspection_id = inspection.id
			JOIN audit_assignments assignment
			  ON assignment.inspection_id = attachment.inspection_id
			 AND assignment.tombstoned_at IS NULL
			JOIN audit_team_members member
			  ON member.assignment_id = assignment.id
			 AND member.subject_id = $2
			 AND member.member_role IN ('INSPECTOR', 'LEAD_INSPECTOR')
			 AND member.removed_at IS NULL
			WHERE attachment.id = $1
			FOR UPDATE OF attachment`, attachmentID, actor.SubjectID).Scan(
			&currentPackageID, &currentUploadState, &inspectionStatus, &checklistStatus); err != nil {
			return ErrAttachmentForbidden
		}
		if currentPackageID == "" || currentUploadState != "PENDING" ||
			inspectionStatus != "IN_PROGRESS" || checklistStatus != "IN_PROGRESS" {
			return ErrAttachmentForbidden
		}
		if bucket != service.quarantineBucket ||
			input.ByteSize != size ||
			input.SHA256 != digest {
			return ErrObjectMismatch
		}
		reader, info, err := service.objects.Open(ctx, bucket, key)
		if err != nil {
			return ErrObjectMismatch
		}
		defer reader.Close()
		observation, err := uploadpolicy.Observe(reader, service.maximumByteSize)
		if err != nil || info.Size != size || !uploadpolicy.MatchesDeclaration(observation, mediaType, digest, size) {
			return ErrObjectMismatch
		}
		objectMetadataID := service.idGenerator("object")
		if _, err := transaction.Exec(ctx, `
			INSERT INTO object_metadata (
				id, aggregate_type, aggregate_id, organization_id, bucket_name, object_key, filename,
				declared_media_type, detected_media_type, sha256, size_bytes, scan_status, object_state,
				upload_id, created_at
			) VALUES ($1, 'inspection_attachment', $2, $3, $4, $5, $6, $7, $7, $8, $9,
				'PENDING', 'QUARANTINED', $10, $11)
		`, objectMetadataID, attachmentID, organizationID, bucket, key, fileName, mediaType, digest, size, input.UploadID, now); err != nil {
			return err
		}
		var version int64
		if err := transaction.QueryRow(ctx, `
			SELECT COALESCE(MAX(version), 0) + 1
			FROM inspection_attachment_versions
			WHERE inspection_attachment_id = $1
		`, attachmentID).Scan(&version); err != nil {
			return err
		}
		attachmentVersionID := service.idGenerator("inspection-attachment-version")
		if _, err := transaction.Exec(ctx, `
			INSERT INTO inspection_attachment_versions (
				id, inspection_attachment_id, version, organization_id,
				source_object_metadata_id, upload_session_id, file_name, media_type,
				sha256, size_bytes, submitted_by_subject_id, submitted_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		`, attachmentVersionID, attachmentID, version, organizationID,
			objectMetadataID, input.UploadID, fileName, mediaType,
			digest, size, actor.SubjectID, now); err != nil {
			return fmt.Errorf("append immutable Inspection Attachment version: %w", err)
		}
		if _, err := transaction.Exec(ctx, `
			UPDATE inspection_attachments SET object_metadata_id = $2, upload_state = 'UPLOADED',
			       scan_state = 'PENDING', current_version_id = $3, revision = revision + 1, updated_at = $4 WHERE id = $1
		`, attachmentID, objectMetadataID, attachmentVersionID, now); err != nil {
			return err
		}
		if _, err := transaction.Exec(ctx, `
			UPDATE upload_sessions SET upload_state = 'COMPLETED', object_metadata_id = $2, completed_at = $3 WHERE id = $1
		`, input.UploadID, objectMetadataID, now); err != nil {
			return err
		}
		output = CompleteUploadOutput{
			InspectionAttachmentID: attachmentID, InspectionAttachmentVersionID: attachmentVersionID,
			Version: version, UploadState: "UPLOADED", ScanState: "PENDING",
		}
		body, _ := json.Marshal(output)
		if _, err := transaction.Exec(ctx, `
			INSERT INTO outbox_messages (
				id, topic, aggregate_type, aggregate_id, payload, available_at, event_version, idempotency_key
			) VALUES ($1, 'inspection_attachment.scan_requested', 'inspection_attachment', $2, $3, $4, 1, $5)
		`, service.idGenerator("outbox"), attachmentID, body, now, "inspection_attachment.scan_requested:"+attachmentID); err != nil {
			return err
		}
		return saveIdempotent(ctx, transaction, scope, input.OperationID, semanticHash, output, now)
	})
	return output, err
}

func loadIdempotent(ctx context.Context, transaction pgx.Tx, scope, operationID, semanticHash string, output any) (bool, error) {
	if _, err := transaction.Exec(ctx, "SELECT pg_advisory_xact_lock(hashtextextended($1, 0))", scope+":"+operationID); err != nil {
		return false, err
	}
	var storedHash string
	var response []byte
	err := transaction.QueryRow(ctx, `SELECT semantic_hash, response_body FROM idempotency_responses WHERE scope = $1 AND operation_id = $2`, scope, operationID).Scan(&storedHash, &response)
	if err == nil {
		if storedHash != semanticHash {
			return false, idempotency.ErrOperationIDReuse
		}
		return true, json.Unmarshal(response, output)
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return false, err
	}
	return false, nil
}

func saveIdempotent(ctx context.Context, transaction pgx.Tx, scope, operationID, semanticHash string, output any, now time.Time) error {
	body, err := json.Marshal(output)
	if err != nil {
		return err
	}
	_, err = transaction.Exec(ctx, `
		INSERT INTO idempotency_responses (scope, operation_id, semantic_hash, response_status, response_headers, response_body, created_at)
		VALUES ($1, $2, $3, 200, '{}'::jsonb, $4, $5)
	`, scope, operationID, semanticHash, body, now)
	return err
}

func randomID(prefix string) string {
	value := make([]byte, 16)
	if _, err := rand.Read(value); err != nil {
		panic(fmt.Sprintf("generate attachment upload identifier: %v", err))
	}
	return prefix + "-" + hex.EncodeToString(value)
}
