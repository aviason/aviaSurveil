-- name: InsertEvent :one
INSERT INTO datafeed_events (
    event_id, contract_id, contract_version, schema_version, event_type,
    event_version, source_module, source_system, tenant_id,
    owning_organization_id, actor_organization_id, visibility_purpose_code,
    operation_id, correlation_id, causation_id, aggregate_type, aggregate_id,
    aggregate_revision, effective_at, known_at, occurred_at, emitted_at,
    entity_refs, state_before, state_after, privacy_class, payload_ciphertext,
    payload_nonce, payload_key_ref, payload_sha256, canonical_event_sha256
) VALUES (
    sqlc.arg(event_id), sqlc.arg(contract_id), sqlc.arg(contract_version),
    sqlc.arg(schema_version), sqlc.arg(event_type), sqlc.arg(event_version),
    sqlc.arg(source_module), sqlc.arg(source_system), sqlc.arg(tenant_id),
    sqlc.arg(owning_organization_id), sqlc.narg(actor_organization_id),
    sqlc.arg(visibility_purpose_code), sqlc.arg(operation_id),
    sqlc.arg(correlation_id), sqlc.narg(causation_id), sqlc.arg(aggregate_type),
    sqlc.arg(aggregate_id), sqlc.arg(aggregate_revision), sqlc.arg(effective_at),
    sqlc.arg(known_at), sqlc.arg(occurred_at), sqlc.arg(emitted_at),
    sqlc.arg(entity_refs), sqlc.narg(state_before), sqlc.arg(state_after),
    sqlc.arg(privacy_class), sqlc.arg(payload_ciphertext), sqlc.arg(payload_nonce),
    sqlc.arg(payload_key_ref), sqlc.arg(payload_sha256), sqlc.arg(canonical_event_sha256)
)
RETURNING event_id, created_at;

-- name: ClaimPendingDelivery :many
WITH eligible AS (
    SELECT state.event_id, event.occurred_at
    FROM datafeed_delivery_state state
    JOIN datafeed_events event ON event.event_id = state.event_id
    LEFT JOIN datafeed_replay_tombstones tombstone ON tombstone.event_id = event.event_id
    WHERE event.tenant_id = sqlc.arg(tenant_id)
      AND event.owning_organization_id = sqlc.arg(owning_organization_id)
      AND tombstone.event_id IS NULL
      AND (
            (state.status = 'PENDING' AND (state.next_attempt_at IS NULL OR state.next_attempt_at <= sqlc.arg(now_at)))
         OR (state.status = 'LEASED' AND state.lease_expires_at <= sqlc.arg(now_at))
      )
    ORDER BY event.occurred_at, event.event_id
    LIMIT sqlc.arg(limit_count)
    FOR UPDATE OF state SKIP LOCKED
)
UPDATE datafeed_delivery_state state
SET status = 'LEASED', lease_generation = state.lease_generation + 1,
    lease_expires_at = sqlc.arg(lease_expires_at), updated_at = sqlc.arg(now_at)
FROM eligible
WHERE state.event_id = eligible.event_id
RETURNING state.event_id, state.lease_generation, state.lease_expires_at, state.attempt_count, eligible.occurred_at;

-- name: AcknowledgeDelivery :one
UPDATE datafeed_delivery_state
SET status = 'ACKNOWLEDGED', acknowledgement_receipt_digest = sqlc.arg(receipt_digest),
    acknowledged_at = sqlc.arg(acknowledged_at), terminal_outcome_code = sqlc.arg(terminal_outcome_code),
    lease_expires_at = NULL, updated_at = sqlc.arg(acknowledged_at)
WHERE event_id = sqlc.arg(event_id)
  AND status = 'LEASED'
  AND lease_generation = sqlc.arg(lease_generation)
RETURNING event_id, lease_generation, status, acknowledgement_receipt_digest, acknowledged_at;

-- name: RetryDelivery :one
UPDATE datafeed_delivery_state
SET status = 'PENDING', lease_expires_at = NULL, next_attempt_at = sqlc.arg(next_attempt_at),
    attempt_count = attempt_count + 1, terminal_outcome_code = sqlc.arg(outcome_code),
    updated_at = sqlc.arg(updated_at)
WHERE event_id = sqlc.arg(event_id)
  AND status = 'LEASED'
  AND lease_generation = sqlc.arg(lease_generation)
RETURNING event_id, lease_generation, status, attempt_count, next_attempt_at;

-- name: QuarantineDelivery :one
UPDATE datafeed_delivery_state
SET status = 'QUARANTINED', lease_expires_at = NULL,
    quarantine_owner_role = sqlc.arg(quarantine_owner_role),
    quarantine_sla_due_at = sqlc.arg(quarantine_sla_due_at),
    terminal_outcome_code = sqlc.arg(outcome_code), updated_at = sqlc.arg(updated_at)
WHERE event_id = sqlc.arg(event_id)
  AND status = 'LEASED'
  AND lease_generation = sqlc.arg(lease_generation)
RETURNING event_id, lease_generation, status, terminal_outcome_code;

-- name: AppendDeliveryAttempt :exec
INSERT INTO datafeed_delivery_attempts (
    attempt_id, event_id, lease_generation, outcome_code,
    acknowledgement_receipt_digest, diagnostic_code, occurred_at
) VALUES (
    sqlc.arg(attempt_id), sqlc.arg(event_id), sqlc.arg(lease_generation),
    sqlc.arg(outcome_code), sqlc.narg(acknowledgement_receipt_digest),
    sqlc.narg(diagnostic_code), sqlc.arg(occurred_at)
);

-- name: GetEventForScopedPublication :one
SELECT event_id, contract_id, contract_version, schema_version, event_type, event_version,
       source_module, source_system, tenant_id, owning_organization_id,
       actor_organization_id, visibility_purpose_code, canonical_event_sha256,
       payload_ciphertext, payload_nonce, payload_key_ref, payload_sha256,
       entity_refs, state_before, state_after, effective_at,
       known_at, occurred_at, emitted_at, aggregate_type, aggregate_id,
       aggregate_revision, correlation_id, causation_id
FROM datafeed_events
WHERE event_id = sqlc.arg(event_id)
  AND tenant_id = sqlc.arg(tenant_id)
  AND owning_organization_id = sqlc.arg(owning_organization_id);
