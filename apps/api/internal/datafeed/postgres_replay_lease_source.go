package datafeed

import (
	"context"
	"fmt"
	"time"

	feedstore "github.com/MarlonJD/aviaSurveil360/apps/api/internal/datafeed/store/postgres"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/platform/database"
	"github.com/jackc/pgx/v5/pgtype"
)

// PostgresReplayLeaseSource claims only an immutable replay run's separate
// delivery lane. It deliberately never changes the source event's original
// delivery state, event bytes, occurrence time, or revision.
type PostgresReplayLeaseSource struct {
	Pool                 *database.Pool
	RunID                string
	TenantID             string
	OwningOrganizationID string
	PayloadKey           []byte
	Clock                func() time.Time
}

func (source PostgresReplayLeaseSource) Claim(ctx context.Context, leaseDuration time.Duration, limit int) ([]LeasedOutboxItem, error) {
	if source.Pool == nil || !validReplayUUID(source.RunID) || source.TenantID == "" || source.OwningOrganizationID == "" || len(source.PayloadKey) == 0 {
		return nil, fmt.Errorf("datafeed replay claim source requires a scoped immutable run and payload key")
	}
	if limit < 1 || limit > maxBatchItems || leaseDuration <= 0 {
		return nil, fmt.Errorf("datafeed replay claim request is outside its bounded policy")
	}
	now := time.Now().UTC()
	if source.Clock != nil {
		now = source.Clock().UTC()
	}
	rows, err := source.Pool.Query(ctx, `
		WITH eligible AS (
			SELECT replay.run_id, replay.event_id, event.occurred_at
			FROM datafeed_replay_delivery_state replay
			JOIN datafeed_replay_runs run ON run.run_id = replay.run_id
			JOIN datafeed_events event ON event.event_id = replay.event_id
			WHERE replay.run_id = $1
			  AND run.tenant_id = $2 AND run.owning_organization_id = $3
			  AND event.tenant_id = $2 AND event.owning_organization_id = $3
			  AND (
				(replay.status = 'PENDING' AND (replay.next_attempt_at IS NULL OR replay.next_attempt_at <= $4))
				OR (replay.status = 'LEASED' AND replay.lease_expires_at <= $4)
			  )
			ORDER BY event.occurred_at, replay.event_id
			LIMIT $5
			FOR UPDATE OF replay SKIP LOCKED
		)
		UPDATE datafeed_replay_delivery_state replay
		SET status = 'LEASED', lease_generation = replay.lease_generation + 1,
		    lease_expires_at = $6, updated_at = $4
		FROM eligible
		WHERE replay.run_id = eligible.run_id AND replay.event_id = eligible.event_id
		RETURNING replay.event_id, replay.lease_generation, replay.attempt_count, eligible.occurred_at
	`, source.RunID, source.TenantID, source.OwningOrganizationID, now, limit, now.Add(leaseDuration))
	if err != nil {
		return nil, fmt.Errorf("claim datafeed replay delivery leases: %w", err)
	}
	defer rows.Close()
	queries := feedstore.New(source.Pool)
	items := make([]LeasedOutboxItem, 0)
	for rows.Next() {
		var eventID pgtype.UUID
		var leaseGeneration int64
		var attemptCount int32
		var occurredAt pgtype.Timestamptz
		if err := rows.Scan(&eventID, &leaseGeneration, &attemptCount, &occurredAt); err != nil {
			return nil, fmt.Errorf("read datafeed replay delivery lease: %w", err)
		}
		row, err := queries.GetEventForScopedPublication(ctx, feedstore.GetEventForScopedPublicationParams{
			EventID: eventID, TenantID: source.TenantID, OwningOrganizationID: source.OwningOrganizationID,
		})
		if err != nil {
			return nil, fmt.Errorf("read claimed datafeed replay event: %w", err)
		}
		payload, err := DecryptPayload(source.PayloadKey, EncryptedPayload{Nonce: row.PayloadNonce, Ciphertext: row.PayloadCiphertext})
		if err != nil {
			return nil, fmt.Errorf("decrypt claimed datafeed replay payload: %w", err)
		}
		event, err := ReconstructPersistedEvent(row, payload)
		if err != nil {
			return nil, err
		}
		contentDigest, err := EventContentDigest(event)
		if err != nil {
			return nil, fmt.Errorf("calculate claimed replay v3 event content digest: %w", err)
		}
		items = append(items, LeasedOutboxItem{BatchItem: BatchItem{
			Event: event, EventID: eventID.String(), EventContentDigest: contentDigest,
			LeaseGeneration: leaseGeneration, ReplayRunID: source.RunID,
		}, AttemptCount: int(attemptCount), OccurredAt: occurredAt.Time.UTC()})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate datafeed replay delivery leases: %w", err)
	}
	return items, nil
}
