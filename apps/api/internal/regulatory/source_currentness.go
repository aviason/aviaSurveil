package regulatory

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/aviason/aviaSurveil/internal/application"
	"github.com/aviason/aviaSurveil/internal/identity"
	"github.com/aviason/aviaSurveil/internal/platform/database"
	"github.com/jackc/pgx/v5"
)

// SourceCurrentnessActivationCommand records a controlled, append-only fact
// that a source version is the current one for a source identity. It is not a
// legal, technical, or publication approval: it only creates the fail-closed
// impact-review boundary that a separately reviewed candidate may later bind.
type SourceCurrentnessActivationCommand struct {
	OperationID              string `json:"operationId"`
	IdempotencyKey           string `json:"idempotencyKey"`
	CurrentSourceSnapshotID  string `json:"currentSourceSnapshotId"`
	CurrentSourceHash        string `json:"currentSourceHash"`
	PreviousSourceSnapshotID string `json:"previousSourceSnapshotId,omitempty"`
	PreviousSourceHash       string `json:"previousSourceHash,omitempty"`
	Reason                   string `json:"reason"`
}

// SourceCurrentnessActivationView is a read-only receipt. ImpactReviewDraftID
// is nil for an initial baseline and non-nil for an exact source transition.
type SourceCurrentnessActivationView struct {
	EventID                  string    `json:"eventId"`
	ImpactReviewDraftID      *string   `json:"impactReviewDraftId"`
	SourceIdentity           string    `json:"sourceIdentity"`
	PreviousSourceSnapshotID *string   `json:"previousSourceSnapshotId"`
	PreviousSourceHash       *string   `json:"previousSourceHash"`
	CurrentSourceSnapshotID  string    `json:"currentSourceSnapshotId"`
	CurrentSourceHash        string    `json:"currentSourceHash"`
	Status                   string    `json:"status"`
	ActivatedAt              time.Time `json:"activatedAt"`
}

func sourceCurrentnessActivationSemantic(command SourceCurrentnessActivationCommand) (string, error) {
	return CanonicalSHA256(map[string]any{
		"operationId":              command.OperationID,
		"idempotencyKey":           command.IdempotencyKey,
		"currentSourceSnapshotId":  command.CurrentSourceSnapshotID,
		"currentSourceHash":        command.CurrentSourceHash,
		"previousSourceSnapshotId": command.PreviousSourceSnapshotID,
		"previousSourceHash":       command.PreviousSourceHash,
		"reason":                   command.Reason,
	})
}

func sourceCurrentnessTransitionDigest(sourceIdentity string, command SourceCurrentnessActivationCommand) (string, error) {
	return CanonicalSHA256(map[string]any{
		"sourceIdentity":           sourceIdentity,
		"currentSourceSnapshotId":  command.CurrentSourceSnapshotID,
		"currentSourceHash":        command.CurrentSourceHash,
		"previousSourceSnapshotId": command.PreviousSourceSnapshotID,
		"previousSourceHash":       command.PreviousSourceHash,
	})
}

func sourceCurrentnessSuffix(digest string) string {
	return strings.TrimPrefix(digest, "sha256:")[:24]
}

func validSourceCurrentnessActivation(command SourceCurrentnessActivationCommand) bool {
	if strings.TrimSpace(command.OperationID) == "" || strings.TrimSpace(command.IdempotencyKey) == "" ||
		strings.TrimSpace(command.CurrentSourceSnapshotID) == "" || !strings.HasPrefix(command.CurrentSourceHash, "sha256:") ||
		strings.TrimSpace(command.Reason) == "" {
		return false
	}
	hasPreviousID := strings.TrimSpace(command.PreviousSourceSnapshotID) != ""
	hasPreviousHash := strings.TrimSpace(command.PreviousSourceHash) != ""
	return hasPreviousID == hasPreviousHash &&
		(!hasPreviousHash || (strings.HasPrefix(command.PreviousSourceHash, "sha256:") && command.PreviousSourceSnapshotID != command.CurrentSourceSnapshotID))
}

// ActivateSourceCurrentness is intentionally separate from candidate import.
// A raw source row remains inert until this command commits its ledger entry;
// a later import can only bind to that exact entry and never creates one.
func (service *AdminService) ActivateSourceCurrentness(ctx context.Context, actor identity.Principal, command SourceCurrentnessActivationCommand) (SourceCurrentnessActivationView, error) {
	if err := service.requireAdmin(actor); err != nil {
		return SourceCurrentnessActivationView{}, err
	}
	if !validSourceCurrentnessActivation(command) {
		return SourceCurrentnessActivationView{}, application.ErrInvalid
	}
	semantic, err := sourceCurrentnessActivationSemantic(command)
	if err != nil {
		return SourceCurrentnessActivationView{}, err
	}
	view := SourceCurrentnessActivationView{}
	err = database.WithinTransaction(ctx, service.Pool, func(ctx context.Context, tx pgx.Tx) error {
		var storedOperationID, storedIdempotencyKey, storedSemantic string
		err := tx.QueryRow(ctx, `
			SELECT operation_id,idempotency_key,semantic_payload_digest
			FROM regulatory_source_currentness_events
			WHERE operation_id=$1 OR idempotency_key=$2`, command.OperationID, command.IdempotencyKey,
		).Scan(&storedOperationID, &storedIdempotencyKey, &storedSemantic)
		if err == nil {
			if storedOperationID != command.OperationID || storedIdempotencyKey != command.IdempotencyKey || storedSemantic != semantic {
				return application.ErrConflict
			}
			return loadSourceCurrentnessActivation(ctx, tx, "operation_id=$1", command.OperationID, &view)
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		return service.activateSourceCurrentnessInTransaction(ctx, tx, actor, command, semantic, &view)
	})
	return view, err
}

func (service *AdminService) activateSourceCurrentnessInTransaction(
	ctx context.Context,
	tx pgx.Tx,
	actor identity.Principal,
	command SourceCurrentnessActivationCommand,
	semantic string,
	view *SourceCurrentnessActivationView,
) error {
	currentIdentity, err := sourceIdentityForCurrentness(ctx, tx, command.CurrentSourceSnapshotID, command.CurrentSourceHash)
	if err != nil {
		return err
	}
	if strings.TrimSpace(command.PreviousSourceSnapshotID) != "" {
		previousIdentity, err := sourceIdentityForCurrentness(ctx, tx, command.PreviousSourceSnapshotID, command.PreviousSourceHash)
		if err != nil {
			return err
		}
		if previousIdentity != currentIdentity {
			return fmt.Errorf("%w: source-currentness predecessor belongs to another source identity", application.ErrInvalid)
		}
	}
	if _, err := tx.Exec(ctx, "SELECT pg_advisory_xact_lock(hashtextextended($1, 0))", SourceCurrentnessLockKey(currentIdentity)); err != nil {
		return err
	}
	// A concurrent identical command can miss the optimistic replay lookup in
	// ActivateSourceCurrentness, then wait on this source lock. Re-read after
	// the lock before checking the linear head so it receives the committed
	// immutable receipt rather than a false predecessor/baseline conflict.
	var storedOperationID, storedIdempotencyKey, storedSemantic string
	err = tx.QueryRow(ctx, `
		SELECT operation_id,idempotency_key,semantic_payload_digest
		FROM regulatory_source_currentness_events
		WHERE operation_id=$1 OR idempotency_key=$2`, command.OperationID, command.IdempotencyKey,
	).Scan(&storedOperationID, &storedIdempotencyKey, &storedSemantic)
	if err == nil {
		if storedOperationID != command.OperationID || storedIdempotencyKey != command.IdempotencyKey || storedSemantic != semantic {
			return application.ErrConflict
		}
		return loadSourceCurrentnessActivation(ctx, tx, "operation_id=$1", command.OperationID, view)
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return err
	}
	var activatedHeadID, activatedHeadHash string
	err = tx.QueryRow(ctx, `
		SELECT current_source_version_id,current_source_hash
		FROM regulatory_source_currentness_events
		WHERE source_identity=$1
		ORDER BY sequence_id DESC
		LIMIT 1`, currentIdentity,
	).Scan(&activatedHeadID, &activatedHeadHash)
	if errors.Is(err, pgx.ErrNoRows) {
		if strings.TrimSpace(command.PreviousSourceSnapshotID) != "" {
			return fmt.Errorf("%w: source-currentness chain requires an explicit initial baseline before a source change", application.ErrInvalid)
		}
	} else if err != nil {
		return err
	} else {
		if strings.TrimSpace(command.PreviousSourceSnapshotID) == "" {
			return fmt.Errorf("%w: source identity already has an activated baseline/head", application.ErrConflict)
		}
		if command.PreviousSourceSnapshotID != activatedHeadID || command.PreviousSourceHash != activatedHeadHash {
			return fmt.Errorf("%w: source-currentness predecessor must exactly match the latest activated source ID and hash", application.ErrInvalid)
		}
	}
	// An historical source snapshot cannot become current a second time. Its
	// original candidate/run identity is immutable, so reactivating it after a
	// later source would make a new currentness event unable to bind a fresh
	// candidate without rewriting or aliasing history. A controlled rollback
	// must therefore be supplied as a new immutable source-version identity,
	// even when the restored bytes have the same SHA-256.
	var previousActivationID string
	err = tx.QueryRow(ctx, `
		SELECT event_id
		FROM regulatory_source_currentness_events
		WHERE source_identity=$1
		  AND current_source_version_id=$2
		  AND current_source_hash=$3`,
		currentIdentity, command.CurrentSourceSnapshotID, command.CurrentSourceHash,
	).Scan(&previousActivationID)
	if err == nil {
		return fmt.Errorf("%w: source snapshot/hash was already activated as %s; restore it through a new immutable source-version identity", application.ErrConflict, previousActivationID)
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return err
	}
	var existingEventID string
	err = tx.QueryRow(ctx, `
		SELECT event_id
		FROM regulatory_source_currentness_events
		WHERE source_identity=$1
		  AND previous_source_version_id IS NOT DISTINCT FROM NULLIF($2, '')
		  AND previous_source_hash IS NOT DISTINCT FROM NULLIF($3, '')
		  AND current_source_version_id=$4
		  AND current_source_hash=$5`,
		currentIdentity, command.PreviousSourceSnapshotID, command.PreviousSourceHash,
		command.CurrentSourceSnapshotID, command.CurrentSourceHash,
	).Scan(&existingEventID)
	if err == nil {
		return fmt.Errorf("%w: exact source-currentness transition already activated as %s", application.ErrConflict, existingEventID)
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return err
	}
	transitionDigest, err := sourceCurrentnessTransitionDigest(currentIdentity, command)
	if err != nil {
		return err
	}
	suffix := sourceCurrentnessSuffix(transitionDigest)
	eventID := "SRC-CURRENTNESS-" + suffix
	activatedAt := service.Clock().UTC()
	if _, err := tx.Exec(ctx, `
		INSERT INTO regulatory_source_currentness_events
			(event_id,source_identity,previous_source_version_id,previous_source_hash,current_source_version_id,current_source_hash,actor_subject_id,operation_id,idempotency_key,semantic_payload_digest,reason,activated_at)
		VALUES
			($1,$2,NULLIF($3,''),NULLIF($4,''),$5,$6,$7,$8,$9,$10,$11,$12)`,
		eventID, currentIdentity, command.PreviousSourceSnapshotID, command.PreviousSourceHash,
		command.CurrentSourceSnapshotID, command.CurrentSourceHash, actor.SubjectID,
		command.OperationID, command.IdempotencyKey, semantic, command.Reason, activatedAt,
	); err != nil {
		return err
	}
	var impactReviewDraftID *string
	if strings.TrimSpace(command.PreviousSourceSnapshotID) != "" {
		impactID := "SRC-IMPACT-DRAFT-" + suffix
		if _, err := tx.Exec(ctx, `
			INSERT INTO regulatory_source_impact_review_drafts
				(id,currentness_event_id,source_identity,previous_source_version_id,previous_source_hash,current_source_version_id,current_source_hash,status)
			VALUES ($1,$2,$3,$4,$5,$6,$7,'IMPACT_REVIEW_DRAFT')`,
			impactID, eventID, currentIdentity, command.PreviousSourceSnapshotID, command.PreviousSourceHash,
			command.CurrentSourceSnapshotID, command.CurrentSourceHash,
		); err != nil {
			return err
		}
		impactReviewDraftID = &impactID
	}
	details, err := json.Marshal(map[string]any{
		"eventId":             eventID,
		"impactReviewDraftId": impactReviewDraftID,
		"sourceIdentity":      currentIdentity,
		"previous": map[string]any{
			"sourceId":   command.PreviousSourceSnapshotID,
			"sourceHash": command.PreviousSourceHash,
		},
		"current": map[string]any{
			"sourceId":   command.CurrentSourceSnapshotID,
			"sourceHash": command.CurrentSourceHash,
		},
	})
	if err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO audit_events
			(event_id,occurred_at,actor_subject_id,actor_role,organization_id,action,entity_type,entity_id,entity_version,before_status,after_status,reason,operation_id,correlation_id,request_id,details)
		VALUES
			($1,$2,$3,'admin',$4,'regulatory.source_currentness_activated','REGULATORY_SOURCE_CURRENTNESS',$5,1,NULLIF($6,''),$7,$8,$9,$9,$9,$10::jsonb)`,
		"AE-SOURCE-CURRENTNESS-"+suffix, activatedAt, actor.SubjectID, actor.OrganizationID, eventID,
		command.PreviousSourceHash, command.CurrentSourceHash, command.Reason, command.OperationID, string(details),
	); err != nil {
		return err
	}
	view.EventID = eventID
	view.ImpactReviewDraftID = impactReviewDraftID
	view.SourceIdentity = currentIdentity
	view.CurrentSourceSnapshotID = command.CurrentSourceSnapshotID
	view.CurrentSourceHash = command.CurrentSourceHash
	view.Status = "BASELINE_ACTIVATED"
	view.ActivatedAt = activatedAt
	if strings.TrimSpace(command.PreviousSourceSnapshotID) != "" {
		previousID, previousHash := command.PreviousSourceSnapshotID, command.PreviousSourceHash
		view.PreviousSourceSnapshotID = &previousID
		view.PreviousSourceHash = &previousHash
		view.Status = "IMPACT_REVIEW_DRAFT"
	}
	return nil
}

func sourceIdentityForCurrentness(ctx context.Context, query interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, sourceID, sourceHash string) (string, error) {
	var sourceIdentity string
	err := query.QueryRow(ctx, `
		SELECT source_identity
		FROM regulatory_source_versions
		WHERE id=$1 AND source_hash=$2`, sourceID, sourceHash,
	).Scan(&sourceIdentity)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", fmt.Errorf("%w: source-currentness snapshot/hash is not an exact persisted source", application.ErrInvalid)
	}
	return sourceIdentity, err
}

func loadSourceCurrentnessActivation(ctx context.Context, query interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, predicate string, value string, view *SourceCurrentnessActivationView) error {
	queryText := `
		SELECT event.event_id,draft.id,event.source_identity,
			event.previous_source_version_id,event.previous_source_hash,
			event.current_source_version_id,event.current_source_hash,event.activated_at
		FROM regulatory_source_currentness_events event
		LEFT JOIN regulatory_source_impact_review_drafts draft ON draft.currentness_event_id=event.event_id
		WHERE ` + predicate
	var previousID, previousHash *string
	if err := query.QueryRow(ctx, queryText, value).Scan(
		&view.EventID, &view.ImpactReviewDraftID, &view.SourceIdentity,
		&previousID, &previousHash, &view.CurrentSourceSnapshotID, &view.CurrentSourceHash, &view.ActivatedAt,
	); err != nil {
		return err
	}
	view.PreviousSourceSnapshotID = previousID
	view.PreviousSourceHash = previousHash
	view.Status = "BASELINE_ACTIVATED"
	if view.ImpactReviewDraftID != nil {
		view.Status = "IMPACT_REVIEW_DRAFT"
	}
	return nil
}
