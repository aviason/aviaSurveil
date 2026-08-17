package evidence

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/aviason/aviaSurveil/internal/assignments"
	"github.com/aviason/aviaSurveil/internal/identity"
	"github.com/aviason/aviaSurveil/internal/platform/database"
	"github.com/aviason/aviaSurveil/internal/platform/idempotency"
	"github.com/aviason/aviaSurveil/internal/platform/objectstore"
	"github.com/aviason/aviaSurveil/internal/platform/uploadpolicy"
	"github.com/jackc/pgx/v5"
)

var (
	ErrEvidenceForbidden = errors.New("evidence access forbidden")
	ErrInvalidUpload     = errors.New("invalid evidence upload")
	ErrObjectMismatch    = errors.New("uploaded object does not match declaration")
	ErrEvidenceNotReady  = errors.New("evidence is not ready")
)

// BeginUploadError preserves the public error classification while exposing
// only a bounded internal stage for operational diagnosis. The underlying
// database/object-store error is never included in Error(), so handlers can
// log the stage without leaking credentials, SQL, or object-store details.
type BeginUploadError struct {
	Stage string
	Cause error
}

func (err *BeginUploadError) Error() string {
	return "begin Evidence upload failed at " + err.Stage
}

func (err *BeginUploadError) Unwrap() error { return err.Cause }

const (
	UploadStatePending   = "PENDING"
	UploadStateUploaded  = "UPLOADED"
	ScanStatePending     = "PENDING"
	ScanStateClean       = "CLEAN"
	ScanStateQuarantined = "QUARANTINED"
	ScanStateFailed      = "FAILED"
	ReviewStateNotReady  = "NOT_READY"
	ReviewStatePending   = "PENDING_CAA_REVIEW"
)

type UploadServiceConfig struct {
	QuarantineBucket string
	CanonicalBucket  string
	MaximumByteSize  int64
	InstructionTTL   time.Duration
	Clock            func() time.Time
	IDGenerator      func(string) string
}

type UploadService struct {
	pool             *database.Pool
	objects          objectstore.Store
	quarantineBucket string
	canonicalBucket  string
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
		idGenerator = uploadRandomID
	}
	return &UploadService{
		pool: pool, objects: objects, quarantineBucket: config.QuarantineBucket,
		canonicalBucket: config.CanonicalBucket, maximumByteSize: config.MaximumByteSize,
		instructionTTL: config.InstructionTTL, clock: clock, idGenerator: idGenerator,
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
	OperationID             string `json:"operationId"`
	CorrelationID           string `json:"correlationId"`
	FindingID               string `json:"findingId"`
	ExpectedFindingRevision int64  `json:"expectedFindingRevision"`
	FileName                string `json:"fileName"`
	DeclaredMediaType       string `json:"declaredMediaType"`
	ByteSize                int64  `json:"byteSize"`
	SHA256                  string `json:"sha256"`
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
	if !actor.HasRole(identity.RoleAuditee) || actor.SubjectID == "" || actor.OrganizationID == "" {
		return BeginUploadOutput{}, ErrEvidenceForbidden
	}
	if input.OperationID == "" || input.CorrelationID == "" || input.FindingID == "" || input.ExpectedFindingRevision < 1 {
		return BeginUploadOutput{}, fmt.Errorf("%w: command metadata and Finding revision are required", ErrInvalidUpload)
	}
	if err := uploadpolicy.ValidateDeclaration(input.FileName, input.DeclaredMediaType, input.ByteSize, service.maximumByteSize, input.SHA256); err != nil {
		return BeginUploadOutput{}, fmt.Errorf("%w: %v", ErrInvalidUpload, err)
	}
	semanticHash, err := idempotency.SemanticHash(input)
	if err != nil {
		return BeginUploadOutput{}, err
	}
	scope := actor.SubjectID + ":begin_evidence_upload"
	var output BeginUploadOutput
	stage := "transaction"
	err = database.WithinTransaction(ctx, service.pool, func(ctx context.Context, transaction pgx.Tx) error {
		stage = "idempotency"
		replayed, err := loadIdempotent(ctx, transaction, scope, input.OperationID, semanticHash, &output)
		if err != nil || replayed {
			return err
		}
		stage = "finding-lock"
		var organizationID, status, inspectionID string
		var revision int64
		if err := transaction.QueryRow(ctx, `SELECT organization_id, inspection_id, status, revision FROM findings WHERE id = $1 FOR UPDATE`, input.FindingID).Scan(&organizationID, &inspectionID, &status, &revision); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrEvidenceForbidden
			}
			return err
		}
		if organizationID != actor.OrganizationID {
			return ErrEvidenceForbidden
		}
		if revision != input.ExpectedFindingRevision || (status != "EVIDENCE_REQUIRED" && status != "EVIDENCE_MORE_INFORMATION_REQUESTED") {
			return fmt.Errorf("%w: Finding is not at an Evidence submission boundary", ErrInvalidUpload)
		}
		stage = "preliminary-report"
		var preliminaryIssued bool
		if err := transaction.QueryRow(ctx, `
				SELECT EXISTS (
					SELECT 1
					FROM report_versions preliminary
					JOIN report_approval_states approval ON approval.report_version_id = preliminary.id
					WHERE preliminary.inspection_id = $1
					  AND preliminary.snapshot->>'kind' = 'PRELIMINARY'
					  AND approval.status IN ('ISSUED', 'LOCKED')
				)
			`, inspectionID).Scan(&preliminaryIssued); err != nil {
			return err
		}
		if !preliminaryIssued {
			return fmt.Errorf("%w: Preliminary Report must be approved and issued before Evidence", ErrInvalidUpload)
		}
		stage = "presign"
		now := service.clock().UTC()
		expiresAt := now.Add(service.instructionTTL)
		uploadID := service.idGenerator("upload")
		objectKey := fmt.Sprintf("organizations/%s/evidence/%s/%s", organizationID, input.FindingID, uploadID)
		requiredHeaders := map[string]string{
			"Content-Type":      input.DeclaredMediaType,
			"x-amz-meta-sha256": input.SHA256,
			"If-None-Match":     "*",
		}
		instruction, err := service.objects.CreatePutInstruction(ctx, objectstore.PutRequest{
			Bucket: service.quarantineBucket, Key: objectKey, RequiredHeaders: requiredHeaders, ExpiresAt: expiresAt,
		})
		if err != nil {
			return fmt.Errorf("create private Evidence upload instruction: %w", err)
		}
		output = BeginUploadOutput{
			UploadID: uploadID, StagingObjectKey: objectKey, UploadURL: instruction.URL,
			RequiredHeaders: RequiredHeaders{
				ContentType: input.DeclaredMediaType,
				SHA256:      input.SHA256,
				IfNoneMatch: "*",
			},
			ExpiresAt: expiresAt, MaximumByteSize: service.maximumByteSize,
		}
		stage = "upload-session"
		if _, err := transaction.Exec(ctx, `
			INSERT INTO upload_sessions (
				id, upload_kind, aggregate_id, organization_id, initiated_by_subject_id, bucket_name,
				staging_object_key, file_name, declared_media_type, declared_size_bytes, declared_sha256,
				expected_aggregate_revision, upload_state, expires_at, created_at
			) VALUES ($1, 'EVIDENCE', $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, 'PENDING', $12, $13)
		`, uploadID, input.FindingID, organizationID, actor.SubjectID, service.quarantineBucket, objectKey,
			input.FileName, input.DeclaredMediaType, input.ByteSize, input.SHA256, revision, expiresAt, now); err != nil {
			return fmt.Errorf("record Evidence upload session: %w", err)
		}
		responseBody, err := json.Marshal(output)
		if err != nil {
			return err
		}
		stage = "transaction-envelope"
		return persistUploadTransaction(ctx, transaction, uploadTransactionEnvelope{
			OperationID:      input.OperationID,
			CorrelationID:    input.CorrelationID,
			IdempotencyScope: scope,
			SemanticHash:     semanticHash,
			ResponseBody:     responseBody,
			ActorSubjectID:   actor.SubjectID,
			ActorRole:        string(identity.RoleAuditee),
			OrganizationID:   organizationID,
			Action:           "evidence.upload_started",
			EntityType:       "evidence_upload_session",
			EntityID:         uploadID,
			EntityVersion:    1,
			BeforeStatus:     "",
			AfterStatus:      UploadStatePending,
			SyncKind:         "evidence_upload_session",
			OutboxTopic:      "evidence.upload_started",
			OutboxKey:        "command:" + scope + ":idempotency:" + input.OperationID,
			AuditEventID:     service.idGenerator("audit"),
			OutboxMessageID:  service.idGenerator("outbox"),
			OccurredAt:       now,
		})
	})
	if err != nil {
		return output, &BeginUploadError{Stage: stage, Cause: err}
	}
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
	EvidenceVersionID string `json:"evidenceVersionId"`
	Version           int64  `json:"version"`
	UploadState       string `json:"uploadState"`
	ScanState         string `json:"scanState"`
	ReviewState       string `json:"reviewState"`
}

func (service *UploadService) Complete(ctx context.Context, actor identity.Principal, input CompleteUploadInput) (CompleteUploadOutput, error) {
	if !actor.HasRole(identity.RoleAuditee) || actor.SubjectID == "" || input.OperationID == "" || input.CorrelationID == "" || input.UploadID == "" {
		return CompleteUploadOutput{}, ErrEvidenceForbidden
	}
	semanticHash, err := idempotency.SemanticHash(input)
	if err != nil {
		return CompleteUploadOutput{}, err
	}
	scope := actor.SubjectID + ":complete_evidence_upload"
	var output CompleteUploadOutput
	err = database.WithinTransaction(ctx, service.pool, func(ctx context.Context, transaction pgx.Tx) error {
		replayed, err := loadIdempotent(ctx, transaction, scope, input.OperationID, semanticHash, &output)
		if err != nil || replayed {
			return err
		}
		var findingID, organizationID, initiatedBy, bucket, key, fileName, mediaType, declaredSHA, uploadState string
		var declaredSize, expectedRevision int64
		var expiresAt time.Time
		if err := transaction.QueryRow(ctx, `
			SELECT aggregate_id, organization_id, initiated_by_subject_id, bucket_name, staging_object_key,
			       file_name, declared_media_type, declared_size_bytes, declared_sha256,
			       expected_aggregate_revision, upload_state, expires_at
			FROM upload_sessions WHERE id = $1 AND upload_kind = 'EVIDENCE' FOR UPDATE
		`, input.UploadID).Scan(&findingID, &organizationID, &initiatedBy, &bucket, &key, &fileName, &mediaType,
			&declaredSize, &declaredSHA, &expectedRevision, &uploadState, &expiresAt); err != nil {
			return ErrEvidenceForbidden
		}
		now := service.clock().UTC()
		if initiatedBy != actor.SubjectID || organizationID != actor.OrganizationID || uploadState != "PENDING" || now.After(expiresAt) {
			return ErrEvidenceForbidden
		}
		if bucket != service.quarantineBucket ||
			input.SHA256 != declaredSHA ||
			input.ByteSize != declaredSize {
			return ErrObjectMismatch
		}
		reader, info, err := service.objects.Open(ctx, bucket, key)
		if err != nil {
			return fmt.Errorf("%w: %v", ErrObjectMismatch, err)
		}
		defer reader.Close()
		observation, err := uploadpolicy.Observe(reader, service.maximumByteSize)
		if err != nil || info.Size != declaredSize || !uploadpolicy.MatchesDeclaration(observation, mediaType, declaredSHA, declaredSize) {
			return ErrObjectMismatch
		}
		objectVersionID, objectETag, err := objectstore.ExactIdentityForPersistence(service.objects, info)
		if err != nil {
			return ErrObjectMismatch
		}
		var findingStatus, inspectionID string
		var findingRevision int64
		if err := transaction.QueryRow(ctx, `SELECT inspection_id, status, revision FROM findings WHERE id = $1 AND organization_id = $2 FOR UPDATE`, findingID, organizationID).Scan(&inspectionID, &findingStatus, &findingRevision); err != nil {
			return ErrEvidenceForbidden
		}
		if findingRevision != expectedRevision || (findingStatus != "EVIDENCE_REQUIRED" && findingStatus != "EVIDENCE_MORE_INFORMATION_REQUESTED") {
			return fmt.Errorf("%w: Finding changed after upload began", ErrInvalidUpload)
		}
		var preliminaryIssued bool
		if err := transaction.QueryRow(ctx, `
				SELECT EXISTS (
					SELECT 1
					FROM report_versions preliminary
					JOIN report_approval_states approval ON approval.report_version_id = preliminary.id
					WHERE preliminary.inspection_id = $1
					  AND preliminary.snapshot->>'kind' = 'PRELIMINARY'
					  AND approval.status IN ('ISSUED', 'LOCKED')
				)
			`, inspectionID).Scan(&preliminaryIssued); err != nil {
			return err
		}
		if !preliminaryIssued {
			return fmt.Errorf("%w: Preliminary Report must be approved and issued before Evidence", ErrInvalidUpload)
		}
		objectMetadataID := service.idGenerator("object")
		if _, err := transaction.Exec(ctx, `
			INSERT INTO object_metadata (
				id, aggregate_type, aggregate_id, organization_id, bucket_name, object_key, filename,
				declared_media_type, detected_media_type, sha256, size_bytes, scan_status, object_state,
				upload_id, object_version_id, object_etag, created_at
			) VALUES ($1, 'evidence_upload', $2, $3, $4, $5, $6, $7, $7, $8, $9, 'PENDING', 'QUARANTINED', $10, NULLIF($11, ''), NULLIF($12, ''), $13)
		`, objectMetadataID, findingID, organizationID, bucket, key, fileName, mediaType, declaredSHA, declaredSize,
			input.UploadID, objectVersionID, objectETag, now); err != nil {
			return fmt.Errorf("record quarantined Evidence object: %w", err)
		}
		var evidenceID string
		var version int64
		err = transaction.QueryRow(ctx, `
			SELECT evidence_id, version FROM evidence_versions WHERE finding_id = $1 ORDER BY version DESC LIMIT 1
		`, findingID).Scan(&evidenceID, &version)
		if errors.Is(err, pgx.ErrNoRows) {
			evidenceID = service.idGenerator("evidence")
			version = 0
		} else if err != nil {
			return err
		}
		version++
		evidenceVersionID := service.idGenerator("evidence-version")
		if _, err := transaction.Exec(ctx, `
			INSERT INTO evidence_versions (
				id, evidence_id, finding_id, organization_id, version, object_metadata_id, filename,
				media_type, sha256, size_bytes, status, submitted_by_subject_id, submitted_at, revision
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, 'PENDING', $11, $12, 1)
		`, evidenceVersionID, evidenceID, findingID, organizationID, version, objectMetadataID, fileName,
			mediaType, declaredSHA, declaredSize, actor.SubjectID, now); err != nil {
			return fmt.Errorf("append immutable Evidence version: %w", err)
		}
		if _, err := transaction.Exec(ctx, `
			INSERT INTO evidence_version_states (
				evidence_version_id, upload_state, scan_state, review_state, revision, updated_at
			) VALUES ($1, 'UPLOADED', 'PENDING', 'NOT_READY', 1, $2)
		`, evidenceVersionID, now); err != nil {
			return fmt.Errorf("record Evidence processing state: %w", err)
		}
		if _, err := transaction.Exec(ctx, `
			UPDATE upload_sessions SET upload_state = 'COMPLETED', object_metadata_id = $2, completed_at = $3 WHERE id = $1
		`, input.UploadID, objectMetadataID, now); err != nil {
			return err
		}
		nextFindingRevision := findingRevision + 1
		if _, err := transaction.Exec(ctx, `
			UPDATE findings SET status = 'EVIDENCE_SUBMITTED', next_action = 'Evidence security scan', revision = $2, updated_at = $3 WHERE id = $1
		`, findingID, nextFindingRevision, now); err != nil {
			return err
		}
		output = CompleteUploadOutput{EvidenceVersionID: evidenceVersionID, Version: version, UploadState: UploadStateUploaded, ScanState: ScanStatePending, ReviewState: ReviewStateNotReady}
		responseBody, err := json.Marshal(output)
		if err != nil {
			return err
		}
		return persistUploadTransaction(ctx, transaction, uploadTransactionEnvelope{
			OperationID:      input.OperationID,
			CorrelationID:    input.CorrelationID,
			IdempotencyScope: scope,
			SemanticHash:     semanticHash,
			ResponseBody:     responseBody,
			ActorSubjectID:   actor.SubjectID,
			ActorRole:        string(identity.RoleAuditee),
			OrganizationID:   organizationID,
			Action:           "evidence.uploaded",
			EntityType:       "evidence_version",
			EntityID:         evidenceVersionID,
			EntityVersion:    1,
			BeforeStatus:     UploadStatePending,
			AfterStatus:      UploadStateUploaded,
			SyncKind:         "evidence_version",
			OutboxTopic:      "evidence.scan_requested",
			OutboxKey:        "evidence.scan_requested:" + evidenceVersionID,
			AuditEventID:     service.idGenerator("audit"),
			OutboxMessageID:  service.idGenerator("outbox"),
			OccurredAt:       now,
		})
	})
	return output, err
}

type VersionView struct {
	ID             string    `json:"id"`
	FindingID      string    `json:"findingId"`
	OrganizationID string    `json:"organizationId"`
	Version        int64     `json:"version"`
	FileName       string    `json:"fileName"`
	SubmittedAt    time.Time `json:"submittedAt"`
	UploadState    string    `json:"uploadState"`
	ScanState      string    `json:"scanState"`
	ReviewState    string    `json:"reviewState"`
	Revision       int64     `json:"revision"`
}

func (service *UploadService) ListVersions(ctx context.Context, actor identity.Principal, findingID string) ([]VersionView, error) {
	if actor.SubjectID == "" {
		return nil, ErrEvidenceForbidden
	}
	var findingOrganizationID string
	if err := service.pool.QueryRow(
		ctx,
		"SELECT organization_id FROM findings WHERE id = $1",
		findingID,
	).Scan(&findingOrganizationID); err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			return nil, err
		}
		if actor.HasRole(identity.RoleAuditee) {
			return nil, ErrEvidenceForbidden
		}
		return []VersionView{}, nil
	}
	if err := authorizeEvidenceOrganization(actor, findingOrganizationID); err != nil {
		return nil, err
	}
	if err := service.authorizeFindingScope(ctx, actor, findingID); err != nil {
		return nil, err
	}
	rows, err := service.pool.Query(ctx, `
		SELECT version.id, version.finding_id, version.organization_id, version.version, version.filename,
		       version.submitted_at, state.upload_state, state.scan_state, state.review_state, state.revision
		FROM evidence_versions version
		JOIN evidence_version_states state ON state.evidence_version_id = version.id
		WHERE version.finding_id = $1 ORDER BY version.version
	`, findingID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	views := []VersionView{}
	for rows.Next() {
		var view VersionView
		if err := rows.Scan(&view.ID, &view.FindingID, &view.OrganizationID, &view.Version, &view.FileName,
			&view.SubmittedAt, &view.UploadState, &view.ScanState, &view.ReviewState, &view.Revision); err != nil {
			return nil, err
		}
		if view.OrganizationID != findingOrganizationID {
			return nil, ErrEvidenceForbidden
		}
		views = append(views, view)
	}
	return views, rows.Err()
}

func (service *UploadService) Download(ctx context.Context, actor identity.Principal, evidenceVersionID string) (objectstore.GetInstruction, error) {
	if actor.SubjectID == "" {
		return objectstore.GetInstruction{}, ErrEvidenceForbidden
	}
	var organizationID, findingID string
	if err := service.pool.QueryRow(
		ctx,
		"SELECT organization_id, finding_id FROM evidence_versions WHERE id = $1",
		evidenceVersionID,
	).Scan(&organizationID, &findingID); err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			return objectstore.GetInstruction{}, err
		}
		if actor.HasRole(identity.RoleAuditee) {
			return objectstore.GetInstruction{}, ErrEvidenceForbidden
		}
		return objectstore.GetInstruction{}, ErrEvidenceNotReady
	}
	if err := authorizeEvidenceOrganization(actor, organizationID); err != nil {
		return objectstore.GetInstruction{}, err
	}
	if err := service.authorizeFindingScope(ctx, actor, findingID); err != nil {
		return objectstore.GetInstruction{}, err
	}
	var scanState, bucket, key string
	if err := service.pool.QueryRow(ctx, `
		SELECT state.scan_state, metadata.bucket_name, metadata.object_key
		FROM evidence_versions version
		JOIN evidence_version_states state ON state.evidence_version_id = version.id
		JOIN object_metadata metadata ON metadata.id = state.canonical_object_metadata_id
		WHERE version.id = $1
	`, evidenceVersionID).Scan(&scanState, &bucket, &key); err != nil {
		return objectstore.GetInstruction{}, ErrEvidenceNotReady
	}
	if scanState != ScanStateClean {
		return objectstore.GetInstruction{}, ErrEvidenceNotReady
	}
	return service.objects.CreateGetInstruction(ctx, objectstore.GetRequest{Bucket: bucket, Key: key, ExpiresAt: service.clock().UTC().Add(5 * time.Minute)})
}

func authorizeEvidenceOrganization(actor identity.Principal, organizationID string) error {
	if actor.HasRole(identity.RoleAuditee) {
		if !actor.BelongsTo(organizationID) {
			return ErrEvidenceForbidden
		}
		return nil
	}
	if !actor.IsCAA() {
		return ErrEvidenceForbidden
	}
	return nil
}

// authorizeFindingScope prevents a CAA user with a broad role from reading
// Evidence belonging to an unrelated Audit. Auditee organization privacy is
// handled separately above; CAA access is limited to the active assigned
// Inspector/Lead team for the Finding's inspection.
func (service *UploadService) authorizeFindingScope(ctx context.Context, actor identity.Principal, findingID string) error {
	if actor.HasRole(identity.RoleAuditee) {
		return nil
	}
	if !actor.IsCAA() {
		return ErrEvidenceForbidden
	}
	if actor.HasRole(identity.RoleDepartmentManager) {
		var inspectionID string
		if err := service.pool.QueryRow(ctx, `SELECT inspection_id FROM findings WHERE id = $1`, findingID).Scan(&inspectionID); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrEvidenceForbidden
			}
			return err
		}
		return database.WithinTransaction(ctx, service.pool, func(ctx context.Context, transaction pgx.Tx) error {
			return assignments.RequireCurrentDepartmentScopeAuthority(ctx, transaction, actor, "", inspectionID)
		})
	}
	var assigned bool
	if err := service.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM findings finding
			JOIN audit_assignments assignment
			  ON assignment.inspection_id = finding.inspection_id
			 AND assignment.tombstoned_at IS NULL
			LEFT JOIN audit_team_members member
			  ON member.assignment_id = assignment.id
			 AND member.subject_id = $2
			 AND member.removed_at IS NULL
			WHERE finding.id = $1
			  AND (assignment.lead_subject_id = $2 OR member.subject_id IS NOT NULL)
		)
	`, findingID, actor.SubjectID).Scan(&assigned); err != nil {
		return err
	}
	if !assigned {
		return ErrEvidenceForbidden
	}
	return nil
}

func (service *UploadService) ReconcileExpired(ctx context.Context) (int64, error) {
	result, err := service.pool.Exec(ctx, `
		UPDATE upload_sessions SET upload_state = 'EXPIRED'
		WHERE upload_kind = 'EVIDENCE' AND upload_state = 'PENDING' AND expires_at < $1
	`, service.clock().UTC())
	if err != nil {
		return 0, err
	}
	return result.RowsAffected(), nil
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

type uploadTransactionEnvelope struct {
	OperationID      string
	CorrelationID    string
	IdempotencyScope string
	SemanticHash     string
	ResponseBody     []byte
	ActorSubjectID   string
	ActorRole        string
	OrganizationID   string
	Action           string
	EntityType       string
	EntityID         string
	EntityVersion    int64
	BeforeStatus     string
	AfterStatus      string
	SyncKind         string
	OutboxTopic      string
	OutboxKey        string
	AuditEventID     string
	OutboxMessageID  string
	OccurredAt       time.Time
}

func persistUploadTransaction(
	ctx context.Context,
	transaction pgx.Tx,
	record uploadTransactionEnvelope,
) error {
	if _, err := transaction.Exec(ctx, `
		INSERT INTO audit_events (
			event_id, occurred_at, actor_subject_id, actor_role, organization_id,
			action, entity_type, entity_id, entity_version, before_status, after_status,
			operation_id, correlation_id, request_id, details
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8, $9, NULLIF($10, ''), $11,
			$12, $13, $13, '{}'::jsonb
		)
	`, record.AuditEventID, record.OccurredAt, record.ActorSubjectID, record.ActorRole,
		record.OrganizationID, record.Action, record.EntityType, record.EntityID,
		record.EntityVersion, record.BeforeStatus, record.AfterStatus,
		record.OperationID, record.CorrelationID); err != nil {
		return fmt.Errorf("append Evidence upload audit: %w", err)
	}

	var changeSequenceID int64
	if err := transaction.QueryRow(ctx, `
		INSERT INTO authorized_sync_changes (
			subject_id, organization_id, kind, entity_id, entity_revision,
			payload, changed_at, operation_id, correlation_id
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING sequence_id
	`, record.ActorSubjectID, record.OrganizationID, record.SyncKind, record.EntityID,
		record.EntityVersion, record.ResponseBody, record.OccurredAt, record.OperationID,
		record.CorrelationID).Scan(&changeSequenceID); err != nil {
		return fmt.Errorf("append Evidence upload authorized change: %w", err)
	}

	if _, err := transaction.Exec(ctx, `
		INSERT INTO outbox_messages (
			id, topic, aggregate_type, aggregate_id, payload, available_at,
			event_version, idempotency_key, operation_id, correlation_id
		) VALUES ($1, $2, $3, $4, $5, $6, 1, $7, $8, $9)
	`, record.OutboxMessageID, record.OutboxTopic, record.EntityType, record.EntityID,
		record.ResponseBody, record.OccurredAt, record.OutboxKey, record.OperationID,
		record.CorrelationID); err != nil {
		return fmt.Errorf("enqueue Evidence upload outbox: %w", err)
	}

	if _, err := transaction.Exec(ctx, `
		INSERT INTO idempotency_responses (scope, operation_id, semantic_hash, response_status, response_headers, response_body, created_at)
		VALUES ($1, $2, $3, 200, '{}'::jsonb, $4, $5)
	`, record.IdempotencyScope, record.OperationID, record.SemanticHash,
		record.ResponseBody, record.OccurredAt); err != nil {
		return fmt.Errorf("store Evidence upload idempotent response: %w", err)
	}

	if _, err := transaction.Exec(ctx, `
		INSERT INTO command_transaction_links (
			operation_id, idempotency_scope, audit_event_id,
			change_sequence_id, outbox_message_id, created_at
		) VALUES ($1, $2, $3, $4, $5, $6)
	`, record.OperationID, record.IdempotencyScope, record.AuditEventID,
		changeSequenceID, record.OutboxMessageID, record.OccurredAt); err != nil {
		return fmt.Errorf("link Evidence upload transaction records: %w", err)
	}
	return nil
}

func uploadRandomID(prefix string) string {
	value := make([]byte, 16)
	if _, err := rand.Read(value); err != nil {
		panic(fmt.Sprintf("generate upload identifier: %v", err))
	}
	return prefix + "-" + hex.EncodeToString(value)
}
