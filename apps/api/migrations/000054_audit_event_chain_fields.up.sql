ALTER TABLE audit_events
    ADD COLUMN IF NOT EXISTS profile_key_id text,
    ADD COLUMN IF NOT EXISTS operation_id text,
    ADD COLUMN IF NOT EXISTS package_revision bigint,
    ADD COLUMN IF NOT EXISTS request_hash text,
    ADD COLUMN IF NOT EXISTS payload_hash text,
    ADD COLUMN IF NOT EXISTS server_revision bigint,
    ADD COLUMN IF NOT EXISTS result_code text,
    ADD COLUMN IF NOT EXISTS previous_event_hash text,
    ADD COLUMN IF NOT EXISTS event_hash text;

CREATE INDEX IF NOT EXISTS audit_events_chain_idx
    ON audit_events (organization_id, sequence_id, event_hash);
