package documents

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/aviason/aviaSurveil/internal/identity"
	"github.com/aviason/aviaSurveil/internal/platform/database"
	"github.com/aviason/aviaSurveil/internal/platform/objectstore"
	"github.com/aviason/aviaSurveil/internal/platform/telemetry"
	"github.com/jackc/pgx/v5"
)

type Dependencies struct {
	Renderer            Renderer
	Bucket              string
	Clock               func() time.Time
	WorkerID            string
	AfterExternalEffect func() error
}

type Service struct {
	pool                *database.Pool
	objects             objectstore.Store
	renderer            Renderer
	bucket              string
	clock               func() time.Time
	workerID            string
	afterExternalEffect func() error
}

func NewService(pool *database.Pool, objects objectstore.Store, dependencies Dependencies) *Service {
	clock := dependencies.Clock
	if clock == nil {
		clock = time.Now
	}
	workerID := strings.TrimSpace(dependencies.WorkerID)
	if workerID == "" {
		workerID = "document-worker"
	}
	return &Service{
		pool: pool, objects: objects, renderer: dependencies.Renderer,
		bucket: dependencies.Bucket, clock: clock, workerID: workerID,
		afterExternalEffect: dependencies.AfterExternalEffect,
	}
}

type claimedJob struct {
	OutboxID        string
	JobID           string
	DocumentID      string
	OrganizationID  string
	Version         int64
	AttemptCount    int
	LeaseGeneration int64
	Snapshot        RenderSnapshot
	TraceParent     string
	CorrelationID   string
	AvailableAt     time.Time
}

const (
	documentLeaseDuration   = time.Minute
	documentLeaseRenewal    = 20 * time.Second
	documentAttemptLimit    = 5
	documentAttemptDeadline = 90 * time.Second
)

func (service *Service) ProcessNext(
	ctx context.Context,
) (processed bool, resultErr error) {
	if service.pool == nil || service.objects == nil || service.renderer == nil ||
		strings.TrimSpace(service.bucket) == "" {
		return false, fmt.Errorf("%w: document renderer dependencies are incomplete", ErrNotReady)
	}
	claimed, found, err := service.claimNext(ctx)
	if err != nil || !found {
		return found, err
	}
	attemptContext, cancelAttempt := context.WithTimeout(ctx, documentAttemptDeadline)
	defer cancelAttempt()
	leaseContext, stopLease := context.WithCancel(ctx)
	defer stopLease()
	leaseLost := make(chan error, 1)
	go service.renewLease(leaseContext, claimed, leaseLost)
	jobContext, span := telemetry.StartPersistedJob(
		ctx,
		claimed.TraceParent,
		claimed.CorrelationID,
		"document",
		"native-pdf",
	)
	telemetry.RecordPersistedOutboxReadyAge(
		jobContext,
		"document",
		"document",
		claimed.AvailableAt,
		service.clock().UTC(),
	)
	defer func() {
		telemetry.FinishPersistedJob(
			jobContext,
			span,
			"document",
			"native-pdf",
			resultErr,
		)
	}()
	artifact, err := service.renderer.Render(attemptContext, claimed.Snapshot)
	if leaseErr := readLeaseError(leaseLost); leaseErr != nil {
		return true, service.recordFailure(ctx, claimed, leaseErr)
	}
	if err == nil {
		err = attemptContext.Err()
	}
	if err != nil {
		return true, service.recordFailure(ctx, claimed, err)
	}
	if err := validateRenderedArtifact(claimed.Snapshot, artifact); err != nil {
		return true, service.recordFailure(ctx, claimed, err)
	}
	if provenanceRenderer, ok := service.renderer.(interface{ Provenance() NativeProvenance }); ok {
		provenance := provenanceRenderer.Provenance()
		if artifact.RendererHash != provenance.RendererHash || artifact.TemplateHash != provenance.TemplateHash {
			return true, service.recordFailure(ctx, claimed, errors.New("renderer provenance identity mismatch"))
		}
	}
	digest := sha256.Sum256(artifact.Body)
	hash := "sha256:" + hex.EncodeToString(digest[:])
	key := fmt.Sprintf(
		"organizations/%s/documents/%s/version-%d.pdf",
		claimed.OrganizationID, claimed.JobID, claimed.Version,
	)
	written, writeErr := service.objects.Write(attemptContext, objectstore.WriteRequest{
		Bucket: service.bucket, Key: key, ContentType: artifact.MediaType,
		Size: int64(len(artifact.Body)), Metadata: map[string]string{
			"sha256":          hash,
			"renderer-sha256": artifact.RendererHash,
			"template-sha256": artifact.TemplateHash,
			"source-sha256":   artifact.SourceHash,
		},
		Body: bytes.NewReader(artifact.Body),
	})
	if writeErr != nil && !errors.Is(writeErr, objectstore.ErrObjectAlreadyExists) {
		return true, service.recordFailure(ctx, claimed, writeErr)
	}
	if writeErr != nil {
		reader, info, openErr := service.objects.Open(attemptContext, service.bucket, key)
		if openErr != nil {
			return true, service.recordFailure(ctx, claimed, openErr)
		}
		existing, readErr := io.ReadAll(io.LimitReader(
			reader, int64(len(artifact.Body))+1,
		))
		closeErr := reader.Close()
		if readErr != nil || closeErr != nil ||
			info.Size != int64(len(artifact.Body)) ||
			info.ContentType != artifact.MediaType ||
			!bytes.Equal(existing, artifact.Body) ||
			info.Metadata["sha256"] != hash ||
			info.Metadata["renderer-sha256"] != artifact.RendererHash ||
			info.Metadata["template-sha256"] != artifact.TemplateHash ||
			info.Metadata["source-sha256"] != artifact.SourceHash {
			return true, service.recordFailure(
				ctx,
				claimed,
				errors.Join(
					readErr,
					closeErr,
					errors.New("existing rendered object does not match the immutable output"),
				),
			)
		}
		written = info
	} else if service.afterExternalEffect != nil {
		if err := service.afterExternalEffect(); err != nil {
			return true, service.recordFailure(ctx, claimed, err)
		}
	}
	objectVersionID, objectETag, err := objectstore.ExactIdentityForPersistence(service.objects, written)
	if err != nil {
		return true, service.recordFailure(
			ctx,
			claimed,
			errors.New("rendered object is missing exact-version identity"),
		)
	}
	written.VersionID = objectVersionID
	written.ETag = objectETag
	if leaseErr := readLeaseError(leaseLost); leaseErr != nil {
		return true, service.recordFailure(ctx, claimed, leaseErr)
	}
	if err := service.finalize(attemptContext, claimed, artifact, key, hash, written); err != nil {
		return true, err
	}
	return true, nil
}

func (service *Service) claimNext(ctx context.Context) (claimedJob, bool, error) {
	var claimed claimedJob
	var encoded []byte
	var workPayload []byte
	var found bool
	now := service.clock().UTC()
	err := database.WithinTransaction(ctx, service.pool, func(ctx context.Context, transaction pgx.Tx) error {
		err := transaction.QueryRow(ctx, `
			SELECT outbox.id, job.id, job.document_id, job.organization_id,
			       job.requested_version, job.attempt_count, job.lease_generation, job.input_snapshot,
			       COALESCE(outbox.traceparent, ''),
			       COALESCE(outbox.correlation_id, ''),
			       outbox.available_at, outbox.payload
			FROM outbox_messages outbox
			JOIN document_render_jobs job
			  ON job.id = NULLIF(outbox.payload ->> 'renderJobId', '')
			WHERE outbox.topic = 'document.render_requested'
			  AND outbox.delivered_at IS NULL
			  AND outbox.terminal_state IS NULL
			  AND job.input_snapshot ? 'source'
			  AND NOT EXISTS (
			      SELECT 1 FROM document_render_job_dispositions disposition
			      WHERE disposition.job_id = job.id
			        AND disposition.disposition LIKE 'SUPERSEDED_%'
			  )
			  AND outbox.available_at <= $1
			  AND (outbox.lease_expires_at IS NULL OR outbox.lease_expires_at <= $1)
			  AND job.status IN ('PENDING', 'RUNNING', 'FAILED')
			  AND (job.lease_expires_at IS NULL OR job.lease_expires_at <= $1)
			ORDER BY outbox.available_at, outbox.created_at, outbox.id
			FOR UPDATE OF outbox, job SKIP LOCKED
			LIMIT 1
		`, now).Scan(
			&claimed.OutboxID, &claimed.JobID, &claimed.DocumentID,
			&claimed.OrganizationID, &claimed.Version, &claimed.AttemptCount, &claimed.LeaseGeneration, &encoded,
			&claimed.TraceParent,
			&claimed.CorrelationID,
			&claimed.AvailableAt, &workPayload,
		)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("claim document render job: %w", err)
		}
		if err := json.Unmarshal(encoded, &claimed.Snapshot); err != nil {
			return fmt.Errorf("decode document render snapshot: %w", err)
		}
		if err := validateEnqueuedProvenance(workPayload, claimed); err != nil {
			return err
		}
		claimed.AttemptCount++
		claimed.LeaseGeneration++
		found = true
		if _, err := transaction.Exec(ctx, `
			UPDATE outbox_messages
			SET lease_owner = $2, lease_expires_at = $3, lease_generation = $5, claimed_at = $1,
			    attempt_count = attempt_count + 1
			WHERE id = $4
		`, now, service.workerID, now.Add(documentLeaseDuration), claimed.OutboxID, claimed.LeaseGeneration); err != nil {
			return err
		}
		_, err = transaction.Exec(ctx, `
			UPDATE document_render_jobs
			SET status = 'RUNNING', attempt_count = $2, lease_owner = $4,
			    lease_generation = $5, lease_expires_at = $6, last_error = NULL, updated_at = $3
			WHERE id = $1
		`, claimed.JobID, claimed.AttemptCount, now, service.workerID,
			claimed.LeaseGeneration, now.Add(documentLeaseDuration))
		return err
	})
	return claimed, found, err
}

func (service *Service) renewLease(ctx context.Context, claimed claimedJob, failures chan<- error) {
	ticker := time.NewTicker(documentLeaseRenewal)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			renewContext, cancel := context.WithTimeout(ctx, 5*time.Second)
			err := database.WithinTransaction(renewContext, service.pool, func(ctx context.Context, transaction pgx.Tx) error {
				result, err := transaction.Exec(ctx, `
					UPDATE document_render_jobs
					SET lease_expires_at = $4, updated_at = $4
					WHERE id = $1 AND lease_owner = $2 AND lease_generation = $3
				`, claimed.JobID, service.workerID, claimed.LeaseGeneration, now.UTC().Add(documentLeaseDuration))
				if err != nil {
					return err
				}
				if result.RowsAffected() != 1 {
					return errors.New("document render job lease generation was lost")
				}
				result, err = transaction.Exec(ctx, `
					UPDATE outbox_messages
					SET lease_expires_at = $4
					WHERE id = $1 AND lease_owner = $2 AND lease_generation = $3
				`, claimed.OutboxID, service.workerID, claimed.LeaseGeneration, now.UTC().Add(documentLeaseDuration))
				if err != nil {
					return err
				}
				if result.RowsAffected() != 1 {
					return errors.New("document render outbox lease generation was lost")
				}
				return nil
			})
			cancel()
			if err != nil {
				select {
				case failures <- fmt.Errorf("renew document render lease: %w", err):
				default:
				}
				return
			}
		}
	}
}

func validateEnqueuedProvenance(payload []byte, claimed claimedJob) error {
	var values map[string]any
	if err := json.Unmarshal(payload, &values); err != nil {
		return fmt.Errorf("decode document render request provenance: %w", err)
	}
	jobID, _ := values["renderJobId"].(string)
	if jobID != claimed.JobID {
		return errors.New("document render request is not bound to its frozen render job")
	}
	_, sourceBytes, sourceHash, err := NewReportRenderSource(
		claimed.Snapshot.Source.ReportVersionID, claimed.Snapshot.Source.ReportID,
		claimed.Snapshot.Source.OrganizationID, claimed.Snapshot.Source.AuditID,
		claimed.Snapshot.Source.Version, claimed.Snapshot.Source.ActorSubjectID,
		claimed.Snapshot.Source.Content,
	)
	if err != nil {
		return fmt.Errorf("validate frozen render source: %w", err)
	}
	provenance, err := NativeRendererProvenance()
	if err != nil {
		return err
	}
	if stringValue(values, "sourceHash") != sourceHash ||
		stringValue(values, "rendererHash") != provenance.RendererHash ||
		stringValue(values, "templateHash") != provenance.TemplateHash ||
		stringValue(values, "fontHash") != provenance.FontHash ||
		stringValue(values, "renderer") != provenance.Renderer ||
		stringValue(values, "moduleChecksum") != provenance.ModuleChecksum ||
		stringValue(values, "layout") != provenance.Layout ||
		digest(sourceBytes) != sourceHash {
		return errors.New("document render request provenance identity mismatch")
	}
	return nil
}

func stringValue(values map[string]any, key string) string {
	value, _ := values[key].(string)
	return value
}

func readLeaseError(failures <-chan error) error {
	select {
	case err := <-failures:
		return err
	default:
		return nil
	}
}

func (service *Service) finalize(
	ctx context.Context,
	claimed claimedJob,
	artifact RenderedArtifact,
	key string,
	hash string,
	object objectstore.ObjectInfo,
) error {
	now := service.clock().UTC()
	documentVersionID := claimed.JobID + "-version"
	objectMetadataID := claimed.JobID + "-object"
	return database.WithinTransaction(ctx, service.pool, func(ctx context.Context, transaction pgx.Tx) error {
		var status, leaseOwner string
		var leaseGeneration int64
		var outputID *string
		err := transaction.QueryRow(ctx, `
			SELECT status, output_document_version_id, COALESCE(lease_owner, ''), lease_generation
			FROM document_render_jobs WHERE id = $1 FOR UPDATE
		`, claimed.JobID).Scan(&status, &outputID, &leaseOwner, &leaseGeneration)
		if errors.Is(err, pgx.ErrNoRows) {
			// The canonical test reset may remove a claimed test-only job.
			// Production workflows never delete render jobs.
			return nil
		}
		if err != nil {
			return err
		}
		if leaseOwner != service.workerID || leaseGeneration != claimed.LeaseGeneration {
			return errors.New("document render job lease generation was lost before finalization")
		}
		if status == string(JobSucceeded) {
			return service.markDelivered(ctx, transaction, claimed, now)
		}
		if _, err := transaction.Exec(ctx, `
			INSERT INTO object_metadata (
				id, aggregate_type, aggregate_id, organization_id, bucket_name,
				object_key, filename, declared_media_type, detected_media_type,
				sha256, size_bytes, object_version_id, object_etag,
				scan_status, object_state, created_at
			) VALUES (
				$1, 'document_version', $2, $3, $4, $5, $6, $7, $7,
				$8, $9, NULLIF($10, ''), NULLIF($11, ''),
				'CLEAN', 'CANONICAL', $12
			)
			ON CONFLICT (object_key) DO NOTHING
		`, objectMetadataID, documentVersionID, claimed.OrganizationID,
			service.bucket, key, artifact.FileName, artifact.MediaType, hash,
			len(artifact.Body), object.VersionID, object.ETag, now); err != nil {
			return fmt.Errorf("record rendered private object: %w", err)
		}
		if _, err := transaction.Exec(ctx, `
			INSERT INTO document_versions (
				id, document_id, organization_id, version, visibility, status,
				file_name, media_type, sha256, size_bytes, object_metadata_id,
				created_by_subject_id, created_at, renderer_hash, template_hash,
				source_hash
			) VALUES (
				$1, $2, $3, $4, 'AUDITEE_VISIBLE', 'RELEASED',
				$5, $6, $7, $8, $9, $10, $11, $12, $13, $14
			)
			ON CONFLICT (document_id, version) DO NOTHING
		`, documentVersionID, claimed.DocumentID, claimed.OrganizationID,
			claimed.Version, artifact.FileName, artifact.MediaType, hash,
			len(artifact.Body), objectMetadataID, claimed.Snapshot.CreatedBySubject, now,
			artifact.RendererHash, artifact.TemplateHash, artifact.SourceHash); err != nil {
			return fmt.Errorf("append immutable DocumentVersion: %w", err)
		}
		result, err := transaction.Exec(ctx, `
			UPDATE document_render_jobs
			SET status = 'SUCCEEDED', output_document_version_id = $2,
			    lease_owner = NULL, lease_expires_at = NULL,
			    last_error = NULL, updated_at = $3
			WHERE id = $1 AND status = 'RUNNING' AND lease_owner = $4
			  AND lease_generation = $5
		`, claimed.JobID, documentVersionID, now, service.workerID, claimed.LeaseGeneration)
		if err != nil {
			return err
		}
		if result.RowsAffected() != 1 {
			return errors.New("document render job state changed before completion")
		}
		operationID := "render:" + claimed.JobID
		if _, err := transaction.Exec(ctx, `
			INSERT INTO audit_events (
				event_id, occurred_at, actor_subject_id, actor_role, organization_id,
				action, entity_type, entity_id, entity_version, before_status,
				after_status, operation_id, correlation_id, request_id, details
			) VALUES (
				$1, $2, $3, 'system', $4, 'document.render_completed',
				'document_version', $5, $6, 'PENDING', 'RELEASED',
				$7, $7, $7, jsonb_build_object(
					'sha256', $8::text,
					'rendererHash', $9::text,
					'templateHash', $10::text,
					'sourceHash', $11::text,
					'reportVersionId', $12::text
				)
			)
			ON CONFLICT (event_id) DO NOTHING
		`, claimed.JobID+"-audit", now, claimed.Snapshot.CreatedBySubject,
			claimed.OrganizationID, documentVersionID, claimed.Version,
			operationID, hash, artifact.RendererHash, artifact.TemplateHash,
			artifact.SourceHash, claimed.Snapshot.ReportVersionID); err != nil {
			return fmt.Errorf("append document render audit: %w", err)
		}
		projection, err := json.Marshal(map[string]any{
			"documentVersionId": documentVersionID, "documentId": claimed.DocumentID,
			"version": claimed.Version, "sha256": hash, "status": "RELEASED",
			"rendererHash":    artifact.RendererHash,
			"templateHash":    artifact.TemplateHash,
			"sourceHash":      artifact.SourceHash,
			"reportVersionId": claimed.Snapshot.ReportVersionID,
		})
		if err != nil {
			return err
		}
		if _, err := transaction.Exec(ctx, `
			INSERT INTO authorized_sync_changes (
				subject_id, organization_id, kind, entity_id, entity_revision,
				payload, changed_at, operation_id, correlation_id
			) VALUES ($1, $2, 'document_version', $3, $4, $5, $6, $7, $7)
		`, claimed.Snapshot.CreatedBySubject, claimed.OrganizationID,
			documentVersionID, claimed.Version, projection, now, operationID); err != nil {
			return fmt.Errorf("append document render change: %w", err)
		}
		if _, err := transaction.Exec(ctx, `
			INSERT INTO outbox_messages (
				id, topic, aggregate_type, aggregate_id, payload, available_at,
				event_version, idempotency_key, operation_id, correlation_id
			) VALUES (
				$1, 'document.render_completed', 'document_version', $2, $3, $4,
				1, $5, $6, $6
			)
			ON CONFLICT (idempotency_key) WHERE idempotency_key IS NOT NULL DO NOTHING
		`, claimed.JobID+"-completed-outbox", documentVersionID, projection, now,
			"document.render_completed:"+claimed.JobID, operationID); err != nil {
			return fmt.Errorf("enqueue document render completion: %w", err)
		}
		return service.markDelivered(ctx, transaction, claimed, now)
	})
}

func validateRenderedArtifact(
	snapshot RenderSnapshot,
	artifact RenderedArtifact,
) error {
	if strings.TrimSpace(artifact.FileName) == "" ||
		strings.ContainsAny(artifact.FileName, `/\`) ||
		!strings.HasSuffix(strings.ToLower(artifact.FileName), ".pdf") {
		return fmt.Errorf("rendered PDF filename is invalid")
	}
	if artifact.MediaType != "application/pdf" ||
		len(artifact.Body) < len("%PDF-") ||
		!bytes.Equal(artifact.Body[:len("%PDF-")], []byte("%PDF-")) {
		return fmt.Errorf("renderer did not return a PDF artifact")
	}
	if len(artifact.Body) > maximumPDFResponseSize {
		return fmt.Errorf("rendered PDF exceeds %d bytes", maximumPDFResponseSize)
	}
	if !validSHA256(artifact.RendererHash) ||
		!validSHA256(artifact.TemplateHash) ||
		!validSHA256(artifact.SourceHash) {
		return fmt.Errorf("complete renderer, template, and source sha256 provenance is required")
	}
	_, source, sourceHash, err := NewReportRenderSource(
		snapshot.Source.ReportVersionID, snapshot.Source.ReportID,
		snapshot.Source.OrganizationID, snapshot.Source.AuditID,
		snapshot.Source.Version, snapshot.Source.ActorSubjectID, snapshot.Source.Content,
	)
	if err != nil {
		return fmt.Errorf("encode immutable render source: %w", err)
	}
	if artifact.SourceHash != sourceHash || artifact.SourceHash != digest(source) {
		return fmt.Errorf("rendered source sha256 does not match the immutable snapshot")
	}
	return nil
}

func (service *Service) recordFailure(ctx context.Context, claimed claimedJob, cause error) error {
	now := service.clock().UTC()
	errorClass := telemetry.ErrorClass(cause)
	err := database.WithinTransaction(ctx, service.pool, func(ctx context.Context, transaction pgx.Tx) error {
		result, err := transaction.Exec(ctx, `
			UPDATE document_render_jobs
			SET status = 'FAILED', attempt_count = $2, last_error = $3,
			    lease_owner = NULL, lease_expires_at = NULL, updated_at = $4
			WHERE id = $1 AND lease_owner = $5 AND lease_generation = $6
		`, claimed.JobID, claimed.AttemptCount, errorClass, now,
			service.workerID, claimed.LeaseGeneration)
		if err != nil {
			return err
		}
		if result.RowsAffected() != 1 {
			return errors.New("document render failure lease generation was lost")
		}
		terminal := claimed.AttemptCount >= documentAttemptLimit
		result, err = transaction.Exec(ctx, `
			UPDATE outbox_messages
			SET last_error = $2, available_at = $3, lease_owner = NULL,
			    lease_expires_at = NULL, terminal_state = CASE WHEN $5 THEN 'DEAD_LETTER' ELSE terminal_state END
			WHERE id = $1 AND lease_owner = $4 AND lease_generation = $6
		`, claimed.OutboxID, errorClass, now.Add(5*time.Second), service.workerID,
			terminal, claimed.LeaseGeneration)
		if err != nil {
			return err
		}
		if result.RowsAffected() != 1 {
			return errors.New("document render outbox failure lease generation was lost")
		}
		if !terminal {
			return nil
		}
		details, err := json.Marshal(map[string]any{"errorClass": errorClass, "jobId": claimed.JobID})
		if err != nil {
			return err
		}
		if _, err := transaction.Exec(ctx, `
			INSERT INTO document_render_job_dispositions
				(id, job_id, disposition, attempt_count, details, created_at)
			VALUES ($1, $2, 'DEAD_LETTER', $3, $4, $5)
			ON CONFLICT (job_id, disposition, attempt_count) DO NOTHING
		`, claimed.JobID+"-dead-letter-"+fmt.Sprint(claimed.AttemptCount), claimed.JobID,
			claimed.AttemptCount, details, now); err != nil {
			return err
		}
		return nil
	})
	return errors.Join(cause, err)
}

func (service *Service) markDelivered(
	ctx context.Context,
	transaction pgx.Tx,
	claimed claimedJob,
	now time.Time,
) error {
	result, err := transaction.Exec(ctx, `
		UPDATE outbox_messages
		SET delivered_at = $3, lease_owner = NULL, lease_expires_at = NULL
		WHERE id = $1 AND delivered_at IS NULL AND lease_owner = $2
		  AND lease_generation = $4
	`, claimed.OutboxID, service.workerID, now, claimed.LeaseGeneration)
	if err != nil {
		return err
	}
	if result.RowsAffected() != 1 {
		return errors.New("document render outbox lease generation was lost")
	}
	return nil
}

func (service *Service) AuthorizeDownload(
	ctx context.Context,
	actor identity.Principal,
	documentVersionID string,
) (Download, error) {
	var organizationID, visibility, status, fileName, mediaType, hash, kind string
	var size int64
	var bucket, key, objectState, scanStatus *string
	var sourceSnapshot []byte
	err := service.pool.QueryRow(ctx, `
		SELECT version.organization_id, version.visibility, version.status,
		       version.file_name, version.media_type, version.sha256, version.size_bytes,
		       record.kind, metadata.bucket_name, metadata.object_key,
		       metadata.object_state, metadata.scan_status, job.input_snapshot
		FROM document_versions version
		JOIN document_records record ON record.id = version.document_id
		LEFT JOIN object_metadata metadata ON metadata.id = version.object_metadata_id
		LEFT JOIN document_render_jobs job
		  ON job.output_document_version_id = version.id
		WHERE version.id = $1
	`, documentVersionID).Scan(
		&organizationID, &visibility, &status, &fileName, &mediaType, &hash, &size,
		&kind, &bucket, &key, &objectState, &scanStatus, &sourceSnapshot,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Download{}, ErrNotFound
	}
	if err != nil {
		return Download{}, err
	}
	if actor.HasRole(identity.RoleAuditee) {
		if actor.OrganizationID != organizationID ||
			visibility != "AUDITEE_VISIBLE" || status != "RELEASED" {
			return Download{}, ErrForbidden
		}
		if kind == "REPORT" {
			var snapshot RenderSnapshot
			if len(sourceSnapshot) == 0 || json.Unmarshal(sourceSnapshot, &snapshot) != nil ||
				snapshot.OrganizationID != organizationID ||
				snapshot.AuditID == "" {
				return Download{}, ErrForbidden
			}
			if len(snapshot.FindingIDs) > 0 {
				var authorizedFindingCount int
				if err := service.pool.QueryRow(ctx, `
					SELECT count(*)
					FROM findings
					WHERE id = ANY($1::text[])
					  AND organization_id = $2
					  AND inspection_id = $3
				`, snapshot.FindingIDs, organizationID, snapshot.AuditID).Scan(&authorizedFindingCount); err != nil {
					return Download{}, err
				}
				if authorizedFindingCount != len(snapshot.FindingIDs) {
					return Download{}, ErrForbidden
				}
			}
		}
	} else if !actor.IsCAA() {
		return Download{}, ErrForbidden
	}
	if bucket == nil || key == nil || objectState == nil || scanStatus == nil ||
		*objectState != "CANONICAL" || *scanStatus != "CLEAN" {
		return Download{}, ErrNotReady
	}
	expiresAt := service.clock().UTC().Add(5 * time.Minute)
	instruction, err := service.objects.CreateGetInstruction(ctx, objectstore.GetRequest{
		Bucket: *bucket, Key: *key, DownloadFileName: fileName,
		ExpiresAt: expiresAt,
	})
	if err != nil {
		return Download{}, err
	}
	return Download{
		DocumentVersionID: documentVersionID, FileName: fileName,
		MediaType: mediaType, SHA256: hash, SizeBytes: size,
		URL: instruction.URL, ExpiresAt: instruction.ExpiresAt,
	}, nil
}

// ManualRetry appends a new render request for a terminal/failed job. The
// original job, lease generation, and outbox row remain immutable history; a
// retry receives a new idempotency key and a new fenced lease sequence.
func (service *Service) ManualRetry(ctx context.Context, jobID string) (string, error) {
	jobID = strings.TrimSpace(jobID)
	if jobID == "" || service.pool == nil {
		return "", ErrInvalid
	}
	now := service.clock().UTC()
	retryID := fmt.Sprintf("%s-manual-%d", jobID, now.UnixNano())
	err := database.WithinTransaction(ctx, service.pool, func(ctx context.Context, transaction pgx.Tx) error {
		var documentID, organizationID, status, idempotencyKey string
		var requestedVersion int64
		var attempts int
		var snapshot []byte
		if err := transaction.QueryRow(ctx, `
			SELECT document_id, organization_id, status, idempotency_key,
			       requested_version, attempt_count, input_snapshot
			FROM document_render_jobs WHERE id = $1 FOR UPDATE
		`, jobID).Scan(&documentID, &organizationID, &status, &idempotencyKey,
			&requestedVersion, &attempts, &snapshot); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrNotFound
			}
			return err
		}
		if status == string(JobSucceeded) || status == string(JobRunning) {
			return fmt.Errorf("%w: only failed or pending render jobs can be retried", ErrConflict)
		}
		var payload map[string]any
		if err := json.Unmarshal(snapshot, &payload); err != nil {
			return fmt.Errorf("decode immutable render snapshot: %w", err)
		}
		var nativeSnapshot RenderSnapshot
		if err := json.Unmarshal(snapshot, &nativeSnapshot); err != nil ||
			nativeSnapshot.Source.ReportVersionID == "" ||
			nativeSnapshot.Source.Content.Schema != ReportContentSchema {
			return fmt.Errorf("%w: legacy renderer jobs are superseded and cannot be retried", ErrConflict)
		}
		payload["renderJobId"] = retryID
		payload["manualRetryOf"] = jobID
		encodedPayload, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		if _, err := transaction.Exec(ctx, `
			INSERT INTO document_render_jobs
				(id, document_id, organization_id, requested_version, status,
				 idempotency_key, input_snapshot)
			VALUES ($1, $2, $3, $4, 'PENDING', $5, $6)
		`, retryID, documentID, organizationID, requestedVersion,
			idempotencyKey+":manual:"+fmt.Sprint(now.UnixNano()), snapshot); err != nil {
			return err
		}
		if _, err := transaction.Exec(ctx, `
			INSERT INTO outbox_messages
				(id, topic, aggregate_type, aggregate_id, payload, available_at,
				 event_version, idempotency_key, operation_id, correlation_id)
			VALUES ($1, 'document.render_requested', 'report_version', $2, $3, $4,
				 1, $5, $6, $6)
		`, retryID+"-outbox", jobID, encodedPayload, now,
			"document.render_requested:"+retryID, "manual-render:"+retryID); err != nil {
			return err
		}
		details, err := json.Marshal(map[string]any{"replacementJobId": retryID})
		if err != nil {
			return err
		}
		_, err = transaction.Exec(ctx, `
			INSERT INTO document_render_job_dispositions
				(id, job_id, disposition, attempt_count, details, created_at)
			VALUES ($1, $2, 'MANUAL_RETRY', $3, $4, $5)
			ON CONFLICT (job_id, disposition, attempt_count) DO NOTHING
		`, jobID+"-manual-retry-"+fmt.Sprint(now.UnixNano()), jobID, attempts, details, now)
		return err
	})
	if err != nil {
		return "", err
	}
	return retryID, nil
}
