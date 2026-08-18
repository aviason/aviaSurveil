package finalization

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"sort"
	"time"

	"github.com/aviason/aviaSurveil/internal/identity"
	"github.com/aviason/aviaSurveil/internal/platform/database"
	"github.com/aviason/aviaSurveil/internal/platform/idempotency"
	"github.com/jackc/pgx/v5"
)

const CanonicalizationVersion = "avia-finalization-manifest/v1"

var (
	ErrFinalizationForbidden       = errors.New("inspection finalization forbidden")
	ErrFinalizationNotReady        = errors.New("inspection is not ready for finalization")
	ErrFinalizationAttachmentState = errors.New("inspection attachment is not clean and canonical")
)

type manifestEntry struct {
	EntityID string `json:"entityId"`
	Revision int64  `json:"revision"`
	Digest   string `json:"digest"`
}

func canonicalManifestDigest(kind string, entries []manifestEntry) (string, error) {
	ordered := append([]manifestEntry(nil), entries...)
	sort.Slice(ordered, func(left, right int) bool {
		if ordered[left].EntityID != ordered[right].EntityID {
			return ordered[left].EntityID < ordered[right].EntityID
		}
		if ordered[left].Revision != ordered[right].Revision {
			return ordered[left].Revision < ordered[right].Revision
		}
		return ordered[left].Digest < ordered[right].Digest
	})
	payload, err := json.Marshal(struct {
		CanonicalizationVersion string          `json:"canonicalizationVersion"`
		ManifestKind            string          `json:"manifestKind"`
		Entries                 []manifestEntry `json:"entries"`
	}{CanonicalizationVersion, kind, ordered})
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(payload)
	return "sha256:" + hex.EncodeToString(digest[:]), nil
}

func valueDigest(value any) (string, error) {
	payload, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(payload)
	return "sha256:" + hex.EncodeToString(digest[:]), nil
}

type Dependencies struct {
	Clock       func() time.Time
	IDGenerator func(string) string
}

type Service struct {
	pool        *database.Pool
	clock       func() time.Time
	idGenerator func(string) string
}

func NewService(pool *database.Pool, dependencies Dependencies) *Service {
	clock := dependencies.Clock
	if clock == nil {
		clock = time.Now
	}
	idGenerator := dependencies.IDGenerator
	if idGenerator == nil {
		idGenerator = func(prefix string) string { return prefix + "-" + time.Now().UTC().Format("20060102150405.000000000") }
	}
	return &Service{pool: pool, clock: clock, idGenerator: idGenerator}
}

type FinalizeInput struct {
	OperationID             string
	CorrelationID           string
	InspectionID            string
	ExpectedPackageRevision int64
}

type Receipt struct {
	ReceiptID               string    `json:"receiptId"`
	InspectionID            string    `json:"inspectionId"`
	PackageRevision         int64     `json:"packageRevision"`
	ServerRevision          int64     `json:"serverRevision"`
	AnswerManifestHash      string    `json:"answerManifestHash"`
	FindingManifestHash     string    `json:"findingManifestHash"`
	AttachmentManifestHash  string    `json:"attachmentManifestHash"`
	EventManifestHash       string    `json:"eventManifestHash"`
	CanonicalizationVersion string    `json:"canonicalizationVersion"`
	ServerTimestamp         time.Time `json:"serverTimestamp"`
}

func (service *Service) Finalize(ctx context.Context, actor identity.Principal, input FinalizeInput) (Receipt, error) {
	if !actor.HasRole(identity.RoleInspector) || actor.SubjectID == "" || actor.SessionID == "" ||
		input.OperationID == "" || input.CorrelationID == "" || input.InspectionID == "" || input.ExpectedPackageRevision < 1 {
		return Receipt{}, ErrFinalizationForbidden
	}
	semanticHash, err := idempotency.SemanticHash(input)
	if err != nil {
		return Receipt{}, err
	}
	scope := actor.SubjectID + ":inspection_finalize"
	var output Receipt
	err = database.WithinTransaction(ctx, service.pool, func(ctx context.Context, transaction pgx.Tx) error {
		if _, err := transaction.Exec(ctx, "SELECT pg_advisory_xact_lock(hashtextextended($1, 0))", scope+":"+input.OperationID); err != nil {
			return err
		}
		var storedHash string
		var storedBody []byte
		err := transaction.QueryRow(ctx, `SELECT semantic_hash, response_body FROM idempotency_responses WHERE scope = $1 AND operation_id = $2`, scope, input.OperationID).Scan(&storedHash, &storedBody)
		if err == nil {
			if storedHash != semanticHash {
				return idempotency.ErrOperationIDReuse
			}
			return json.Unmarshal(storedBody, &output)
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		if err := transaction.QueryRow(ctx, `
			SELECT receipt_id, inspection_id, package_revision, server_revision,
			       answer_manifest_hash, finding_manifest_hash, attachment_manifest_hash,
			       event_manifest_hash, canonicalization_version, server_timestamp
			FROM inspection_finalization_receipts WHERE inspection_id = $1
		`, input.InspectionID).Scan(&output.ReceiptID, &output.InspectionID, &output.PackageRevision, &output.ServerRevision,
			&output.AnswerManifestHash, &output.FindingManifestHash, &output.AttachmentManifestHash,
			&output.EventManifestHash, &output.CanonicalizationVersion, &output.ServerTimestamp); err == nil {
			return saveIdempotent(ctx, transaction, scope, input.OperationID, semanticHash, output, service.clock().UTC())
		} else if !errors.Is(err, pgx.ErrNoRows) {
			return err
		}

		var checklistStatus, organizationID string
		var packageRevision, serverRevision int64
		if err := transaction.QueryRow(ctx, `
			SELECT checklist.status, inspection.organization_id, package.package_version, inspection.revision
			FROM inspections inspection
			JOIN inspection_checklists checklist ON checklist.inspection_id = inspection.id
			JOIN inspection_packages package ON package.inspection_id = inspection.id
			WHERE inspection.id = $1
			ORDER BY package.package_version DESC
			LIMIT 1
			FOR UPDATE OF inspection, checklist, package
		`, input.InspectionID).Scan(&checklistStatus, &organizationID, &packageRevision, &serverRevision); err != nil {
			return ErrFinalizationNotReady
		}
		if checklistStatus != "SUBMITTED" || packageRevision != input.ExpectedPackageRevision {
			return ErrFinalizationNotReady
		}

		answerEntries := []manifestEntry{}
		rows, err := transaction.Query(ctx, `
			SELECT id, revision, question_id, response_value, COALESCE(comment_to_auditee, '')
			FROM checklist_responses WHERE inspection_id = $1 ORDER BY id
		`, input.InspectionID)
		if err != nil {
			return err
		}
		for rows.Next() {
			var entry manifestEntry
			var questionID, answer, comment string
			if err := rows.Scan(&entry.EntityID, &entry.Revision, &questionID, &answer, &comment); err != nil {
				rows.Close()
				return err
			}
			entry.Digest, err = valueDigest(map[string]any{"questionId": questionID, "answer": answer, "comment": comment})
			if err != nil {
				rows.Close()
				return err
			}
			answerEntries = append(answerEntries, entry)
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return err
		}
		answerHash, err := canonicalManifestDigest("answers", answerEntries)
		if err != nil {
			return err
		}

		findingEntries := []manifestEntry{}
		rows, err = transaction.Query(ctx, `
			SELECT id, revision, status, finding_basis, COALESCE(comment_to_auditee, ''), COALESCE(converted_finding_id, '')
			FROM potential_findings WHERE inspection_id = $1 ORDER BY id
		`, input.InspectionID)
		if err != nil {
			return err
		}
		for rows.Next() {
			var entry manifestEntry
			var status, basis, comment, converted string
			if err := rows.Scan(&entry.EntityID, &entry.Revision, &status, &basis, &comment, &converted); err != nil {
				rows.Close()
				return err
			}
			entry.Digest, err = valueDigest(map[string]any{"status": status, "findingBasis": basis, "comment": comment, "convertedFindingId": converted})
			if err != nil {
				rows.Close()
				return err
			}
			findingEntries = append(findingEntries, entry)
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return err
		}
		findingHash, err := canonicalManifestDigest("potential-findings", findingEntries)
		if err != nil {
			return err
		}

		attachmentEntries := []manifestEntry{}
		rows, err = transaction.Query(ctx, `
			SELECT attachment.id, attachment.revision, attachment.declared_sha256, attachment.declared_size_bytes,
			       attachment.upload_state, attachment.scan_state, COALESCE(metadata.object_state, ''),
			       COALESCE(metadata.scan_status, ''), COALESCE(metadata.object_version_id, metadata.object_etag, '')
			FROM inspection_attachments attachment
			LEFT JOIN object_metadata metadata ON metadata.id = attachment.canonical_object_metadata_id
			WHERE attachment.inspection_id = $1 ORDER BY attachment.id
		`, input.InspectionID)
		if err != nil {
			return err
		}
		for rows.Next() {
			var entry manifestEntry
			var declaredSHA, uploadState, scanState, objectState, objectScan, objectVersion string
			var declaredSize int64
			if err := rows.Scan(&entry.EntityID, &entry.Revision, &declaredSHA, &declaredSize, &uploadState, &scanState, &objectState, &objectScan, &objectVersion); err != nil {
				rows.Close()
				return err
			}
			if uploadState != "UPLOADED" || scanState != "CLEAN" || objectState != "CANONICAL" || objectScan != "CLEAN" || objectVersion == "" {
				rows.Close()
				return ErrFinalizationAttachmentState
			}
			entry.Digest, err = valueDigest(map[string]any{"sha256": declaredSHA, "byteSize": declaredSize, "objectVersion": objectVersion})
			if err != nil {
				rows.Close()
				return err
			}
			attachmentEntries = append(attachmentEntries, entry)
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return err
		}
		attachmentHash, err := canonicalManifestDigest("attachments", attachmentEntries)
		if err != nil {
			return err
		}

		eventEntries := []manifestEntry{}
		rows, err = transaction.Query(ctx, `
			SELECT event_id, sequence_id, action, entity_type, entity_id
			FROM audit_events WHERE organization_id = $1 ORDER BY sequence_id
		`, organizationID)
		if err != nil {
			return err
		}
		for rows.Next() {
			var entry manifestEntry
			var sequence int64
			var action, entityType string
			if err := rows.Scan(&entry.EntityID, &sequence, &action, &entityType, &entry.Digest); err != nil {
				rows.Close()
				return err
			}
			entry.Revision = sequence
			entry.Digest, err = valueDigest(map[string]any{"action": action, "entityType": entityType, "entityId": entry.Digest})
			if err != nil {
				rows.Close()
				return err
			}
			eventEntries = append(eventEntries, entry)
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return err
		}
		eventHash, err := canonicalManifestDigest("events", eventEntries)
		if err != nil {
			return err
		}

		now := service.clock().UTC()
		output = Receipt{
			ReceiptID: service.idGenerator("finalization-receipt"), InspectionID: input.InspectionID,
			PackageRevision: packageRevision, ServerRevision: serverRevision,
			AnswerManifestHash: answerHash, FindingManifestHash: findingHash,
			AttachmentManifestHash: attachmentHash, EventManifestHash: eventHash,
			CanonicalizationVersion: CanonicalizationVersion, ServerTimestamp: now,
		}
		if _, err := transaction.Exec(ctx, `
			INSERT INTO inspection_finalization_receipts (
				receipt_id, inspection_id, package_revision, server_revision,
				answer_manifest_hash, finding_manifest_hash, attachment_manifest_hash,
				event_manifest_hash, canonicalization_version, server_timestamp, created_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $10)
		`, output.ReceiptID, output.InspectionID, output.PackageRevision, output.ServerRevision,
			output.AnswerManifestHash, output.FindingManifestHash, output.AttachmentManifestHash,
			output.EventManifestHash, output.CanonicalizationVersion, now); err != nil {
			return err
		}
		if _, err := transaction.Exec(ctx, `
			INSERT INTO audit_events (sequence_id, event_id, occurred_at, actor_subject_id, actor_role, organization_id, action, entity_type, entity_id, request_id, details)
			VALUES (nextval(pg_get_serial_sequence('audit_events', 'sequence_id')), $1, $2, $3, 'INSPECTOR', $4, 'inspection.finalized', 'inspection', $5, $6, $7)
		`, service.idGenerator("audit"), now, actor.SubjectID, organizationID, input.InspectionID, input.OperationID,
			map[string]any{"receiptId": output.ReceiptID, "canonicalizationVersion": CanonicalizationVersion}); err != nil {
			return err
		}
		return saveIdempotent(ctx, transaction, scope, input.OperationID, semanticHash, output, now)
	})
	return output, err
}

func saveIdempotent(ctx context.Context, transaction pgx.Tx, scope, operationID, semanticHash string, output Receipt, now time.Time) error {
	body, err := json.Marshal(output)
	if err != nil {
		return err
	}
	_, err = transaction.Exec(ctx, `
		INSERT INTO idempotency_responses (scope, operation_id, semantic_hash, response_status, response_headers, response_body, created_at, terminal_at)
		VALUES ($1, $2, $3, 200, '{}'::jsonb, $4, $5, $5 + INTERVAL '400 days')
	`, scope, operationID, semanticHash, body, now)
	return err
}
