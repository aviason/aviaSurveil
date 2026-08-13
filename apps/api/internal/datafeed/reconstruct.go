package datafeed

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

	feedstore "github.com/aviason/aviaSurveil/internal/datafeed/store/postgres"
	"github.com/jackc/pgx/v5/pgtype"
)

// ReconstructPersistedEvent recreates the exact locked envelope from immutable
// headers plus decrypted payload. It verifies the stored canonical digest
// before the event can cross the network boundary.
func ReconstructPersistedEvent(row feedstore.GetEventForScopedPublicationRow, payloadBytes []byte) (map[string]any, error) {
	payloadDigest := sha256.Sum256(payloadBytes)
	if hex.EncodeToString(payloadDigest[:]) != row.PayloadSha256 {
		return nil, fmt.Errorf("datafeed encrypted payload digest does not match")
	}
	var payload map[string]any
	if err := json.Unmarshal(payloadBytes, &payload); err != nil || payload == nil {
		return nil, fmt.Errorf("decode encrypted datafeed payload")
	}
	var entityRefs map[string]any
	if err := json.Unmarshal(row.EntityRefs, &entityRefs); err != nil || entityRefs == nil {
		return nil, fmt.Errorf("decode immutable datafeed entity references")
	}
	if !row.EventID.Valid || !row.CorrelationID.Valid || !row.EffectiveAt.Valid || !row.KnownAt.Valid || !row.OccurredAt.Valid || !row.EmittedAt.Valid {
		return nil, fmt.Errorf("datafeed immutable event row has incomplete required headers")
	}
	var actorOrganizationID any
	if row.ActorOrganizationID != nil {
		actorOrganizationID = *row.ActorOrganizationID
	}
	event := map[string]any{
		"event_id":                row.EventID.String(),
		"event_type":              row.EventType,
		"event_version":           int(row.EventVersion),
		"contract_id":             row.ContractID,
		"contract_version":        row.ContractVersion,
		"schema_version":          row.SchemaVersion,
		"source_module":           row.SourceModule,
		"source_system":           row.SourceSystem,
		"tenant_id":               row.TenantID,
		"owning_organization_id":  row.OwningOrganizationID,
		"actor_organization_id":   actorOrganizationID,
		"visibility_purpose_code": row.VisibilityPurposeCode,
		"correlation_id":          row.CorrelationID.String(),
		"causation_id":            nullableUUID(row.CausationID),
		"aggregate_type":          row.AggregateType,
		"aggregate_id":            row.AggregateID,
		"aggregate_revision":      row.AggregateRevision,
		"business_key":            row.AggregateID,
		"effective_at":            row.EffectiveAt.Time.UTC().Format("2006-01-02T15:04:05.999999999Z07:00"),
		"known_at":                row.KnownAt.Time.UTC().Format("2006-01-02T15:04:05.999999999Z07:00"),
		"occurred_at":             row.OccurredAt.Time.UTC().Format("2006-01-02T15:04:05.999999999Z07:00"),
		"emitted_at":              row.EmittedAt.Time.UTC().Format("2006-01-02T15:04:05.999999999Z07:00"),
		"entity_refs":             entityRefs,
		"state_before":            nullableString(row.StateBefore),
		"state_after":             row.StateAfter,
		"payload":                 payload,
		"payload_sha256":          row.PayloadSha256,
		"privacy_class":           "P2",
	}
	if CanonicalDigest(event) != row.CanonicalEventSha256 {
		return nil, fmt.Errorf("datafeed immutable event canonical digest does not match")
	}
	return event, nil
}

func nullableUUID(value pgtype.UUID) any {
	if !value.Valid {
		return nil
	}
	return value.String()
}

func nullableString(value *string) any {
	if value == nil {
		return nil
	}
	return *value
}
