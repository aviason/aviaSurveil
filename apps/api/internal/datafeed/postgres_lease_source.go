package datafeed

import (
	"context"
	"fmt"
	"time"

	feedstore "github.com/aviason/aviaSurveil/internal/datafeed/store/postgres"
	"github.com/aviason/aviaSurveil/internal/platform/database"
)

// PostgresLeaseSource is the only worker-side reader of encrypted producer
// payloads. It filters every claim and re-read by the configured tenant and
// owning organization; it never publishes an unscoped event row.
type PostgresLeaseSource struct {
	Pool                 *database.Pool
	TenantID             string
	OwningOrganizationID string
	PayloadKey           []byte
	Clock                func() time.Time
}

func (source PostgresLeaseSource) Claim(ctx context.Context, leaseDuration time.Duration, limit int) ([]LeasedOutboxItem, error) {
	if source.Pool == nil || source.TenantID == "" || source.OwningOrganizationID == "" || len(source.PayloadKey) == 0 {
		return nil, fmt.Errorf("datafeed claim source requires scoped pool and payload key")
	}
	if limit < 1 || limit > maxBatchItems || leaseDuration <= 0 {
		return nil, fmt.Errorf("datafeed claim request is outside its bounded policy")
	}
	now := time.Now().UTC()
	if source.Clock != nil {
		now = source.Clock().UTC()
	}
	queries := feedstore.New(source.Pool)
	claimed, err := queries.ClaimPendingDelivery(ctx, feedstore.ClaimPendingDeliveryParams{
		LeaseExpiresAt: timestamp(now.Add(leaseDuration)), NowAt: timestamp(now), TenantID: source.TenantID,
		OwningOrganizationID: source.OwningOrganizationID, LimitCount: int32(limit),
	})
	if err != nil {
		return nil, fmt.Errorf("claim datafeed delivery leases: %w", err)
	}
	items := make([]LeasedOutboxItem, 0, len(claimed))
	for _, lease := range claimed {
		row, err := queries.GetEventForScopedPublication(ctx, feedstore.GetEventForScopedPublicationParams{
			EventID: lease.EventID, TenantID: source.TenantID, OwningOrganizationID: source.OwningOrganizationID,
		})
		if err != nil {
			return nil, fmt.Errorf("read claimed datafeed event: %w", err)
		}
		payload, err := DecryptPayload(source.PayloadKey, EncryptedPayload{Nonce: row.PayloadNonce, Ciphertext: row.PayloadCiphertext})
		if err != nil {
			return nil, fmt.Errorf("decrypt claimed datafeed payload: %w", err)
		}
		event, err := ReconstructPersistedEvent(row, payload)
		if err != nil {
			return nil, err
		}
		contentDigest, err := EventContentDigest(event)
		if err != nil {
			return nil, fmt.Errorf("calculate claimed v3 event content digest: %w", err)
		}
		items = append(items, LeasedOutboxItem{BatchItem: BatchItem{
			Event: event, EventID: row.EventID.String(), EventContentDigest: contentDigest, LeaseGeneration: lease.LeaseGeneration,
		}, AttemptCount: int(lease.AttemptCount), OccurredAt: lease.OccurredAt.Time.UTC()})
	}
	return items, nil
}
