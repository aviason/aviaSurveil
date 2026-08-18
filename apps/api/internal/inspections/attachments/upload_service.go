package attachments

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strconv"
	"time"

	"github.com/aviason/aviaSurveil/internal/identity"
	"github.com/aviason/aviaSurveil/internal/platform/database"
	"github.com/aviason/aviaSurveil/internal/platform/idempotency"
	"github.com/aviason/aviaSurveil/internal/platform/objectstore"
	"github.com/aviason/aviaSurveil/internal/platform/uploadpolicy"
	fieldsync "github.com/aviason/aviaSurveil/internal/sync"
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
	UploadID            string            `json:"uploadId"`
	StagingObjectKey    string            `json:"stagingObjectKey"`
	UploadURL           string            `json:"uploadUrl"`
	RequiredHeaders     RequiredHeaders   `json:"requiredHeaders"`
	ExpiresAt           time.Time         `json:"expiresAt"`
	MaximumByteSize     int64             `json:"maximumByteSize"`
	SessionEpoch        int64             `json:"sessionEpoch"`
	PartSize            int64             `json:"partSize"`
	ReceivedParts       []int64           `json:"receivedParts"`
	AcknowledgedOffsets []int64           `json:"acknowledgedOffsets"`
	PartHashes          map[string]string `json:"partHashes"`
	WholeFileSHA256     string            `json:"wholeFileSha256"`
}

type UploadPartReceipt struct {
	PartNumber         int64  `json:"partNumber"`
	ByteSize           int64  `json:"byteSize"`
	SHA256             string `json:"sha256"`
	AcknowledgedOffset int64  `json:"acknowledgedOffset"`
	ObjectVersion      string `json:"objectVersion"`
}

const defaultInspectionAttachmentPartSize int64 = 5 * 1024 * 1024

func expectedPartCount(byteSize, partSize int64) int64 {
	if byteSize <= 0 || partSize <= 0 {
		return 0
	}
	return (byteSize + partSize - 1) / partSize
}

func validateResumablePartLayout(totalSize, partSize int64, receipts []UploadPartReceipt) error {
	if totalSize < 0 || partSize <= 0 || len(receipts) != int(expectedPartCount(totalSize, partSize)) {
		return ErrObjectMismatch
	}
	ordered := append([]UploadPartReceipt(nil), receipts...)
	sort.Slice(ordered, func(left, right int) bool { return ordered[left].PartNumber < ordered[right].PartNumber })
	var offset int64
	for index, receipt := range ordered {
		partNumber := int64(index + 1)
		expectedSize := partSize
		if partNumber == int64(len(ordered)) {
			expectedSize = totalSize - offset
		}
		if receipt.PartNumber != partNumber || receipt.ByteSize != expectedSize || receipt.AcknowledgedOffset != offset+receipt.ByteSize || receipt.SHA256 == "" || receipt.ObjectVersion == "" {
			return ErrObjectMismatch
		}
		offset += receipt.ByteSize
	}
	if offset != totalSize {
		return ErrObjectMismatch
	}
	return nil
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
		if err != nil {
			return err
		}
		if replayed {
			return service.refreshBeginUploadState(ctx, transaction, &output)
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
				  AND upload_state IN ('PENDING', 'OPEN', 'UPLOADING', 'PARTIALLY_COMMITTED', 'COMPLETING') AND expires_at > $2
			)`, input.InspectionAttachmentID, service.clock().UTC()).Scan(&activeUpload); err != nil {
			return err
		}
		if activeUpload {
			return ErrAttachmentForbidden
		}
		now := service.clock().UTC()
		expiresAt := now.Add(service.instructionTTL)
		var sessionEpoch int64
		if err := transaction.QueryRow(ctx, `
			SELECT COALESCE(MAX(session_epoch), 0) + 1
			FROM upload_sessions
			WHERE aggregate_id = $1 AND upload_kind = 'INSPECTION_ATTACHMENT'
		`, input.InspectionAttachmentID).Scan(&sessionEpoch); err != nil {
			return err
		}
		if _, err := transaction.Exec(ctx, `
			UPDATE upload_sessions SET upload_state = 'EXPIRED'
			WHERE aggregate_id = $1 AND upload_kind = 'INSPECTION_ATTACHMENT'
			  AND expires_at <= $2 AND upload_state IN ('OPEN', 'UPLOADING', 'PARTIALLY_COMMITTED', 'COMPLETING', 'PENDING')
		`, input.InspectionAttachmentID, now); err != nil {
			return err
		}
		uploadID := service.idGenerator("upload")
		key := fmt.Sprintf("organizations/%s/inspection-attachments/%s/%s", organizationID, input.InspectionAttachmentID, uploadID)
		partSize := defaultInspectionAttachmentPartSize
		output = BeginUploadOutput{
			UploadID: uploadID, StagingObjectKey: key, ExpiresAt: expiresAt,
			MaximumByteSize: service.maximumByteSize, SessionEpoch: sessionEpoch, PartSize: partSize,
			ReceivedParts: []int64{}, AcknowledgedOffsets: []int64{}, PartHashes: map[string]string{},
			WholeFileSHA256: input.SHA256,
		}
		if _, err := transaction.Exec(ctx, `
			INSERT INTO upload_sessions (
				id, upload_kind, aggregate_id, organization_id, initiated_by_subject_id, bucket_name,
				staging_object_key, file_name, declared_media_type, declared_size_bytes, declared_sha256,
				upload_state, expires_at, created_at, session_epoch, part_size_bytes, whole_file_sha256
			) VALUES ($1, 'INSPECTION_ATTACHMENT', $2, $3, $4, $5, $6, $7, $8, $9, $10, 'OPEN', $11, $12, $13, $14, $15)
		`, uploadID, input.InspectionAttachmentID, organizationID, actor.SubjectID, service.quarantineBucket,
			key, input.FileName, input.DeclaredMediaType, input.ByteSize, input.SHA256, expiresAt, now, sessionEpoch, partSize, input.SHA256); err != nil {
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

func (service *UploadService) refreshBeginUploadState(ctx context.Context, transaction pgx.Tx, output *BeginUploadOutput) error {
	if output.UploadID == "" {
		return ErrAttachmentForbidden
	}
	if err := transaction.QueryRow(ctx, `
		SELECT session_epoch, part_size_bytes, whole_file_sha256, expires_at
		FROM upload_sessions WHERE id = $1 AND upload_kind = 'INSPECTION_ATTACHMENT'
	`, output.UploadID).Scan(&output.SessionEpoch, &output.PartSize, &output.WholeFileSHA256, &output.ExpiresAt); err != nil {
		return ErrAttachmentForbidden
	}
	rows, err := transaction.Query(ctx, `
		SELECT part_number, byte_size, part_sha256
		FROM inspection_attachment_upload_parts
		WHERE upload_session_id = $1 AND session_epoch = $2 AND part_state = 'ACKNOWLEDGED'
		ORDER BY part_number
	`, output.UploadID, output.SessionEpoch)
	if err != nil {
		return err
	}
	defer rows.Close()
	output.ReceivedParts = []int64{}
	output.AcknowledgedOffsets = []int64{}
	output.PartHashes = map[string]string{}
	var offset int64
	for rows.Next() {
		var partNumber, byteSize int64
		var partHash string
		if err := rows.Scan(&partNumber, &byteSize, &partHash); err != nil {
			return err
		}
		output.ReceivedParts = append(output.ReceivedParts, partNumber)
		output.PartHashes[strconv.FormatInt(partNumber, 10)] = partHash
		offset += byteSize
		output.AcknowledgedOffsets = append(output.AcknowledgedOffsets, offset)
	}
	return rows.Err()
}

type BeginPartInput struct {
	OperationID   string `json:"operationId"`
	CorrelationID string `json:"correlationId"`
	UploadID      string `json:"uploadId"`
	SessionEpoch  int64  `json:"sessionEpoch"`
	PartNumber    int64  `json:"partNumber"`
	ByteSize      int64  `json:"byteSize"`
	SHA256        string `json:"sha256"`
}

type BeginPartOutput struct {
	UploadID        string          `json:"uploadId"`
	SessionEpoch    int64           `json:"sessionEpoch"`
	PartNumber      int64           `json:"partNumber"`
	PartObjectKey   string          `json:"partObjectKey"`
	UploadURL       string          `json:"uploadUrl"`
	RequiredHeaders RequiredHeaders `json:"requiredHeaders"`
	ExpiresAt       time.Time       `json:"expiresAt"`
}

func (service *UploadService) BeginPart(ctx context.Context, actor identity.Principal, input BeginPartInput) (BeginPartOutput, error) {
	if !actor.HasRole(identity.RoleInspector) || actor.SubjectID == "" || actor.SessionID == "" {
		return BeginPartOutput{}, ErrAttachmentForbidden
	}
	if input.OperationID == "" || input.CorrelationID == "" || input.UploadID == "" || input.SessionEpoch < 1 || input.PartNumber < 1 || input.ByteSize < 0 || input.SHA256 == "" {
		return BeginPartOutput{}, ErrInvalidUpload
	}
	semanticHash, err := idempotency.SemanticHash(input)
	if err != nil {
		return BeginPartOutput{}, err
	}
	scope := actor.SubjectID + ":begin_inspection_attachment_part"
	var output BeginPartOutput
	err = database.WithinTransaction(ctx, service.pool, func(ctx context.Context, transaction pgx.Tx) error {
		replayed, err := loadIdempotent(ctx, transaction, scope, input.OperationID, semanticHash, &output)
		if err != nil || replayed {
			return err
		}
		var aggregateID, organizationID, initiatedBy, bucket, stagingKey string
		var sessionEpoch, partSize, totalSize int64
		var expiresAt time.Time
		if err := transaction.QueryRow(ctx, `
			SELECT aggregate_id, organization_id, initiated_by_subject_id, bucket_name, staging_object_key,
			       session_epoch, part_size_bytes, declared_size_bytes, expires_at
			FROM upload_sessions
			WHERE id = $1 AND upload_kind = 'INSPECTION_ATTACHMENT'
			FOR UPDATE
		`, input.UploadID).Scan(&aggregateID, &organizationID, &initiatedBy, &bucket, &stagingKey,
			&sessionEpoch, &partSize, &totalSize, &expiresAt); err != nil {
			return ErrAttachmentForbidden
		}
		now := service.clock().UTC()
		if initiatedBy != actor.SubjectID || sessionEpoch != input.SessionEpoch || now.After(expiresAt) || partSize <= 0 {
			return ErrAttachmentForbidden
		}
		if input.PartNumber > expectedPartCount(totalSize, partSize) {
			return ErrInvalidUpload
		}
		expectedSize := partSize
		if input.PartNumber == expectedPartCount(totalSize, partSize) {
			expectedSize = totalSize - (input.PartNumber-1)*partSize
		}
		if input.ByteSize != expectedSize {
			return ErrObjectMismatch
		}
		var existingHash, existingKey, existingState string
		err = transaction.QueryRow(ctx, `
			SELECT part_sha256, part_object_key, part_state
			FROM inspection_attachment_upload_parts
			WHERE upload_session_id = $1 AND session_epoch = $2 AND part_number = $3
		`, input.UploadID, input.SessionEpoch, input.PartNumber).Scan(&existingHash, &existingKey, &existingState)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		partKey := fmt.Sprintf("%s/epoch-%d/part-%d", stagingKey, input.SessionEpoch, input.PartNumber)
		if err == nil {
			if existingHash != input.SHA256 || existingKey != partKey {
				return ErrObjectMismatch
			}
			partKey = existingKey
			_ = existingState
		} else {
			if _, err := transaction.Exec(ctx, `
				INSERT INTO inspection_attachment_upload_parts (
					id, upload_session_id, session_epoch, part_number, byte_size, part_sha256,
					part_object_key, part_state, created_at
				) VALUES ($1, $2, $3, $4, $5, $6, $7, 'OPEN', $8)
			`, service.idGenerator("upload-part"), input.UploadID, input.SessionEpoch, input.PartNumber,
				input.ByteSize, input.SHA256, partKey, now); err != nil {
				return err
			}
		}
		instruction, err := service.objects.CreatePutInstruction(ctx, objectstore.PutRequest{
			Bucket: bucket, Key: partKey, ExpiresAt: now.Add(service.instructionTTL),
			RequiredHeaders: map[string]string{
				"Content-Type": "application/octet-stream", "x-amz-meta-sha256": input.SHA256, "If-None-Match": "*",
			},
		})
		if err != nil {
			return err
		}
		output = BeginPartOutput{
			UploadID: input.UploadID, SessionEpoch: input.SessionEpoch, PartNumber: input.PartNumber,
			PartObjectKey: partKey, UploadURL: instruction.URL,
			RequiredHeaders: RequiredHeaders{ContentType: "application/octet-stream", SHA256: input.SHA256, IfNoneMatch: "*"},
			ExpiresAt:       instruction.ExpiresAt,
		}
		_, err = transaction.Exec(ctx, `UPDATE upload_sessions SET upload_state = 'UPLOADING' WHERE id = $1`, input.UploadID)
		if err != nil {
			return err
		}
		return saveIdempotent(ctx, transaction, scope, input.OperationID, semanticHash, output, now)
	})
	return output, err
}

func (service *UploadService) AcknowledgePart(ctx context.Context, actor identity.Principal, input BeginPartInput) (UploadPartReceipt, error) {
	if !actor.HasRole(identity.RoleInspector) || actor.SubjectID == "" || actor.SessionID == "" {
		return UploadPartReceipt{}, ErrAttachmentForbidden
	}
	if input.OperationID == "" || input.CorrelationID == "" || input.UploadID == "" || input.SessionEpoch < 1 || input.PartNumber < 1 {
		return UploadPartReceipt{}, ErrInvalidUpload
	}
	semanticHash, err := idempotency.SemanticHash(input)
	if err != nil {
		return UploadPartReceipt{}, err
	}
	scope := actor.SubjectID + ":ack_inspection_attachment_part"
	var output UploadPartReceipt
	err = database.WithinTransaction(ctx, service.pool, func(ctx context.Context, transaction pgx.Tx) error {
		replayed, err := loadIdempotent(ctx, transaction, scope, input.OperationID, semanticHash, &output)
		if err != nil || replayed {
			return err
		}
		var initiatedBy, bucket, partKey, partHash, partState string
		var declaredSize, sessionEpoch int64
		var expiresAt time.Time
		if err := transaction.QueryRow(ctx, `
			SELECT session.initiated_by_subject_id, session.bucket_name, session.session_epoch,
			       session.expires_at, part.part_object_key, part.part_sha256, part.byte_size, part.part_state
			FROM upload_sessions session
			JOIN inspection_attachment_upload_parts part ON part.upload_session_id = session.id
			WHERE session.id = $1 AND part.session_epoch = $2 AND part.part_number = $3
			FOR UPDATE OF session, part
		`, input.UploadID, input.SessionEpoch, input.PartNumber).Scan(&initiatedBy, &bucket, &sessionEpoch,
			&expiresAt, &partKey, &partHash, &declaredSize, &partState); err != nil {
			return ErrAttachmentForbidden
		}
		now := service.clock().UTC()
		if initiatedBy != actor.SubjectID || sessionEpoch != input.SessionEpoch || now.After(expiresAt) || partHash != input.SHA256 || declaredSize != input.ByteSize {
			return ErrObjectMismatch
		}
		if partState == "ACKNOWLEDGED" {
			var objectVersion string
			if err := transaction.QueryRow(ctx, `SELECT COALESCE(object_version_id, object_etag, part_object_key) FROM inspection_attachment_upload_parts WHERE upload_session_id = $1 AND session_epoch = $2 AND part_number = $3`, input.UploadID, input.SessionEpoch, input.PartNumber).Scan(&objectVersion); err != nil {
				return err
			}
			var offset int64
			if err := transaction.QueryRow(ctx, `SELECT COALESCE(SUM(byte_size), 0) FROM inspection_attachment_upload_parts WHERE upload_session_id = $1 AND session_epoch = $2 AND part_number <= $3 AND part_state = 'ACKNOWLEDGED'`, input.UploadID, input.SessionEpoch, input.PartNumber).Scan(&offset); err != nil {
				return err
			}
			output = UploadPartReceipt{PartNumber: input.PartNumber, ByteSize: input.ByteSize, SHA256: input.SHA256, AcknowledgedOffset: offset, ObjectVersion: objectVersion}
			return saveIdempotent(ctx, transaction, scope, input.OperationID, semanticHash, output, now)
		}
		reader, info, err := service.objects.Open(ctx, bucket, partKey)
		if err != nil {
			return ErrObjectMismatch
		}
		observation, observeErr := uploadpolicy.Observe(reader, service.maximumByteSize)
		_ = reader.Close()
		if observeErr != nil || info.Size != input.ByteSize || !uploadpolicy.MatchesDeclaration(observation, "application/octet-stream", input.SHA256, input.ByteSize) {
			return ErrObjectMismatch
		}
		objectVersion, objectETag, identityErr := objectstore.ExactIdentityForPersistence(service.objects, info)
		if identityErr != nil {
			return ErrObjectMismatch
		}
		if objectVersion == "" {
			objectVersion = objectETag
		}
		if objectVersion == "" {
			objectVersion = partKey
		}
		if _, err := transaction.Exec(ctx, `
			UPDATE inspection_attachment_upload_parts
			SET part_state = 'ACKNOWLEDGED', object_version_id = NULLIF($4, ''), object_etag = NULLIF($5, ''), acknowledged_at = $6
			WHERE upload_session_id = $1 AND session_epoch = $2 AND part_number = $3
		`, input.UploadID, input.SessionEpoch, input.PartNumber, objectVersion, objectETag, now); err != nil {
			return err
		}
		var offset int64
		if err := transaction.QueryRow(ctx, `SELECT COALESCE(SUM(byte_size), 0) FROM inspection_attachment_upload_parts WHERE upload_session_id = $1 AND session_epoch = $2 AND part_number <= $3 AND part_state = 'ACKNOWLEDGED'`, input.UploadID, input.SessionEpoch, input.PartNumber).Scan(&offset); err != nil {
			return err
		}
		if _, err := transaction.Exec(ctx, `UPDATE upload_sessions SET upload_state = 'PARTIALLY_COMMITTED' WHERE id = $1`, input.UploadID); err != nil {
			return err
		}
		output = UploadPartReceipt{PartNumber: input.PartNumber, ByteSize: input.ByteSize, SHA256: input.SHA256, AcknowledgedOffset: offset, ObjectVersion: objectVersion}
		return saveIdempotent(ctx, transaction, scope, input.OperationID, semanticHash, output, now)
	})
	return output, err
}

type CompleteUploadInput struct {
	OperationID   string              `json:"operationId"`
	CorrelationID string              `json:"correlationId"`
	UploadID      string              `json:"uploadId"`
	SessionEpoch  int64               `json:"sessionEpoch"`
	SHA256        string              `json:"sha256"`
	ByteSize      int64               `json:"byteSize"`
	Parts         []UploadPartReceipt `json:"parts"`
}

type CompleteUploadOutput struct {
	InspectionAttachmentID        string `json:"inspectionAttachmentId"`
	InspectionAttachmentVersionID string `json:"inspectionAttachmentVersionId"`
	Version                       int64  `json:"version"`
	UploadState                   string `json:"uploadState"`
	ScanState                     string `json:"scanState"`
	ByteSize                      int64  `json:"byteSize"`
	SHA256                        string `json:"sha256"`
	ObjectVersion                 string `json:"objectVersion"`
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
		var size, sessionEpoch, partSize int64
		var expiresAt time.Time
		if err := transaction.QueryRow(ctx, `
			SELECT aggregate_id, organization_id, initiated_by_subject_id, bucket_name, staging_object_key,
			       file_name, declared_media_type, declared_size_bytes, declared_sha256, upload_state, expires_at,
			       session_epoch, part_size_bytes
			FROM upload_sessions WHERE id = $1 AND upload_kind = 'INSPECTION_ATTACHMENT' FOR UPDATE
		`, input.UploadID).Scan(&attachmentID, &organizationID, &initiatedBy, &bucket, &key, &fileName,
			&mediaType, &size, &digest, &state, &expiresAt, &sessionEpoch, &partSize); err != nil {
			return ErrAttachmentForbidden
		}
		now := service.clock().UTC()
		if initiatedBy != actor.SubjectID || input.SessionEpoch != sessionEpoch || partSize <= 0 || now.After(expiresAt) ||
			(state != "OPEN" && state != "UPLOADING" && state != "PARTIALLY_COMMITTED" && state != "COMPLETING") {
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
		if bucket != service.quarantineBucket || input.ByteSize != size || input.SHA256 != digest || len(input.Parts) == 0 {
			return ErrObjectMismatch
		}
		orderedReceipts := append([]UploadPartReceipt(nil), input.Parts...)
		sort.Slice(orderedReceipts, func(left, right int) bool {
			return orderedReceipts[left].PartNumber < orderedReceipts[right].PartNumber
		})
		if err := validateResumablePartLayout(size, partSize, orderedReceipts); err != nil {
			return err
		}
		type storedPart struct {
			partNumber int64
			byteSize   int64
			sha256     string
			objectKey  string
			objectID   string
			state      string
		}
		stored := map[int64]storedPart{}
		rows, err := transaction.Query(ctx, `
			SELECT part_number, byte_size, part_sha256, part_object_key,
			       COALESCE(object_version_id, object_etag, part_object_key), part_state
			FROM inspection_attachment_upload_parts
			WHERE upload_session_id = $1 AND session_epoch = $2
			ORDER BY part_number
		`, input.UploadID, input.SessionEpoch)
		if err != nil {
			return err
		}
		for rows.Next() {
			var part storedPart
			if err := rows.Scan(&part.partNumber, &part.byteSize, &part.sha256, &part.objectKey, &part.objectID, &part.state); err != nil {
				rows.Close()
				return err
			}
			stored[part.partNumber] = part
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return err
		}
		rows.Close()
		if len(stored) != len(orderedReceipts) {
			return ErrObjectMismatch
		}
		partReaders := make([]io.Reader, 0, len(orderedReceipts))
		for _, receipt := range orderedReceipts {
			part, ok := stored[receipt.PartNumber]
			if !ok || part.state != "ACKNOWLEDGED" || part.byteSize != receipt.ByteSize || part.sha256 != receipt.SHA256 || part.objectID != receipt.ObjectVersion {
				return ErrObjectMismatch
			}
			reader, info, openErr := service.objects.Open(ctx, bucket, part.objectKey)
			if openErr != nil {
				return ErrObjectMismatch
			}
			partBytes, readErr := io.ReadAll(reader)
			_ = reader.Close()
			if readErr != nil || int64(len(partBytes)) != receipt.ByteSize {
				return ErrObjectMismatch
			}
			observation, observeErr := uploadpolicy.Observe(bytes.NewReader(partBytes), service.maximumByteSize)
			if observeErr != nil || info.Size != receipt.ByteSize || observation.Size != receipt.ByteSize || observation.SHA256 != receipt.SHA256 {
				return ErrObjectMismatch
			}
			partReaders = append(partReaders, bytes.NewReader(partBytes))
		}
		finalReader, finalInfo, openErr := service.objects.Open(ctx, bucket, key)
		if openErr == nil {
			finalBytes, readErr := io.ReadAll(finalReader)
			_ = finalReader.Close()
			observation, observeErr := uploadpolicy.Observe(bytes.NewReader(finalBytes), service.maximumByteSize)
			if readErr != nil || int64(len(finalBytes)) != size || observeErr != nil || !uploadpolicy.MatchesDeclaration(observation, mediaType, digest, size) {
				return ErrObjectMismatch
			}
		} else {
			finalInfo, err = service.objects.Write(ctx, objectstore.WriteRequest{
				Bucket: bucket, Key: key, ContentType: mediaType, Size: size,
				Metadata: map[string]string{"sha256": digest}, Body: io.MultiReader(partReaders...),
			})
			if err != nil {
				return ErrObjectMismatch
			}
		}
		objectVersionID, objectETag, err := objectstore.ExactIdentityForPersistence(service.objects, finalInfo)
		if err != nil {
			return ErrObjectMismatch
		}
		if objectVersionID == "" {
			objectVersionID = finalInfo.VersionID
		}
		if objectVersionID == "" {
			objectVersionID = finalInfo.ETag
		}
		if objectVersionID == "" {
			objectVersionID = key
		}
		objectMetadataID := service.idGenerator("object")
		if _, err := transaction.Exec(ctx, `
			INSERT INTO object_metadata (
				id, aggregate_type, aggregate_id, organization_id, bucket_name, object_key, filename,
				declared_media_type, detected_media_type, sha256, size_bytes, scan_status, object_state,
				upload_id, object_version_id, object_etag, created_at
			) VALUES ($1, 'inspection_attachment', $2, $3, $4, $5, $6, $7, $7, $8, $9,
				'PENDING', 'QUARANTINED', $10, NULLIF($11, ''), NULLIF($12, ''), $13)
		`, objectMetadataID, attachmentID, organizationID, bucket, key, fileName, mediaType, digest, size,
			input.UploadID, objectVersionID, objectETag, now); err != nil {
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
			UPDATE upload_sessions SET upload_state = 'COMPLETED', object_metadata_id = $2,
			       object_version_id = NULLIF($3, ''), object_etag = NULLIF($4, ''), completed_at = $5 WHERE id = $1
		`, input.UploadID, objectMetadataID, objectVersionID, objectETag, now); err != nil {
			return err
		}
		output = CompleteUploadOutput{
			InspectionAttachmentID: attachmentID, InspectionAttachmentVersionID: attachmentVersionID,
			Version: version, UploadState: "UPLOADED", ScanState: "PENDING", ByteSize: size,
			SHA256: digest, ObjectVersion: objectVersionID,
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
