package datafeed

import (
	"context"
	"crypto/aes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
)

// WriterConfig contains deployment-owned values. The tenant is intentionally
// not inferred from any organization or request body. A worker has no access
// to this key; Task 5's publisher consumes only scoped, reconstructed events.
type WriterConfig struct {
	TenantID      string
	PayloadKey    []byte
	PayloadKeyRef string
}

type Writer struct {
	tenantID      string
	payloadKey    []byte
	payloadKeyRef string
}

func NewWriter(config WriterConfig) (*Writer, error) {
	if strings.TrimSpace(config.TenantID) == "" {
		return nil, fmt.Errorf("datafeed platform tenant is required")
	}
	if _, err := aes.NewCipher(config.PayloadKey); err != nil {
		return nil, fmt.Errorf("datafeed payload encryption key: %w", err)
	}
	keyRef := strings.TrimSpace(config.PayloadKeyRef)
	if keyRef == "" {
		keyRef = "deployment-managed-aes-gcm"
	}
	return &Writer{
		tenantID: config.TenantID, payloadKey: append([]byte(nil), config.PayloadKey...), payloadKeyRef: keyRef,
	}, nil
}

// Append persists only encrypted payload bytes and immutable, schema-checked
// headers. It must be invoked on the caller's business transaction.
func (writer *Writer) Append(ctx context.Context, transaction pgx.Tx, operationID string, input EventInput) (string, error) {
	if writer == nil || transaction == nil {
		return "", fmt.Errorf("datafeed writer and transaction are required")
	}
	if strings.TrimSpace(operationID) == "" {
		return "", fmt.Errorf("datafeed operation is required")
	}
	if input.TenantID != "" && input.TenantID != writer.tenantID {
		return "", fmt.Errorf("datafeed authenticated tenant mismatch")
	}
	input.TenantID = writer.tenantID
	event, err := BuildEvent(input)
	if err != nil {
		return "", err
	}
	payload, err := CanonicalJSON(event["payload"].(map[string]any))
	if err != nil {
		return "", fmt.Errorf("canonical event payload: %w", err)
	}
	sealed, err := EncryptPayload(writer.payloadKey, payload)
	if err != nil {
		return "", err
	}
	entityRefs, err := json.Marshal(event["entity_refs"])
	if err != nil {
		return "", fmt.Errorf("encode entity references: %w", err)
	}
	_, err = transaction.Exec(ctx, `
		INSERT INTO datafeed_events (
			event_id, contract_id, contract_version, schema_version, event_type,
			event_version, source_module, source_system, tenant_id,
			owning_organization_id, actor_organization_id, visibility_purpose_code,
			operation_id, correlation_id, causation_id, aggregate_type, aggregate_id,
			aggregate_revision, effective_at, known_at, occurred_at, emitted_at,
			entity_refs, state_before, state_after, privacy_class, payload_ciphertext,
			payload_nonce, payload_key_ref, payload_sha256, canonical_event_sha256
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15,
			$16, $17, $18, $19, $20, $21, $22, $23, $24, $25, $26, $27, $28, $29,
			$30, $31
		)
	`, event["event_id"], event["contract_id"], event["contract_version"], event["schema_version"],
		event["event_type"], event["event_version"], event["source_module"], event["source_system"],
		event["tenant_id"], event["owning_organization_id"], event["actor_organization_id"],
		event["visibility_purpose_code"], operationID, event["correlation_id"], event["causation_id"],
		event["aggregate_type"], event["aggregate_id"], event["aggregate_revision"], event["effective_at"],
		event["known_at"], event["occurred_at"], event["emitted_at"], entityRefs, event["state_before"],
		event["state_after"], event["privacy_class"], sealed.Ciphertext, sealed.Nonce, writer.payloadKeyRef,
		event["payload_sha256"], CanonicalDigest(event))
	if err != nil {
		return "", fmt.Errorf("append immutable datafeed event: %w", err)
	}
	eventID, _ := event["event_id"].(string)
	return eventID, nil
}
