-- Task 4: immutable, encrypted producer-side AviaCore v3 feed outbox.
-- This schema is deliberately separate from the legacy internal outbox.  It
-- records authoritative producer facts, never a scraper-derived projection.

CREATE TABLE datafeed_event_type_catalog (
    contract_version text NOT NULL CHECK (contract_version = '3.0.0'),
    event_type text NOT NULL CHECK (event_type ~ '^[a-z][a-z0-9_.-]+$'),
    PRIMARY KEY (contract_version, event_type)
);

INSERT INTO datafeed_event_type_catalog (contract_version, event_type) VALUES
    ('3.0.0', 'audit_assignment.versioned'),
    ('3.0.0', 'audit.completed'),
    ('3.0.0', 'audit.planned'),
    ('3.0.0', 'audit.started'),
    ('3.0.0', 'cap.reviewed'),
    ('3.0.0', 'cap.submitted'),
    ('3.0.0', 'checklist_configuration.published'),
    ('3.0.0', 'checklist_configuration.versioned'),
    ('3.0.0', 'checklist_generation.candidate_versioned'),
    ('3.0.0', 'checklist_generation.impact_review_recorded'),
    ('3.0.0', 'checklist_generation.rollout_readiness_recorded'),
    ('3.0.0', 'checklist_generation.run_recorded'),
    ('3.0.0', 'checklist_package.composed'),
    ('3.0.0', 'checklist_package.versioned'),
    ('3.0.0', 'checklist.completed'),
    ('3.0.0', 'checklist.response_recorded'),
    ('3.0.0', 'correction.recorded'),
    ('3.0.0', 'event.superseded'),
    ('3.0.0', 'evidence_reference.reviewed'),
    ('3.0.0', 'evidence_reference.submitted'),
    ('3.0.0', 'finding.closed'),
    ('3.0.0', 'finding.escalated'),
    ('3.0.0', 'finding.issued'),
    ('3.0.0', 'finding.reopened'),
    ('3.0.0', 'information_request.created'),
    ('3.0.0', 'information_request.responded'),
    ('3.0.0', 'organization.versioned'),
    ('3.0.0', 'potential_finding.decision_recorded'),
    ('3.0.0', 'potential_finding.versioned'),
    ('3.0.0', 'provider_scope.ended'),
    ('3.0.0', 'provider_scope.versioned'),
    ('3.0.0', 'regulatory_reference.versioned'),
    ('3.0.0', 'reminder.dispatch_recorded'),
    ('3.0.0', 'report.decision_recorded'),
    ('3.0.0', 'report.versioned'),
    ('3.0.0', 'surveillance_plan.approval_recorded'),
    ('3.0.0', 'surveillance_plan.status_changed'),
    ('3.0.0', 'surveillance_plan.versioned'),
    ('3.0.0', 'tombstone.recorded');

CREATE TABLE datafeed_events (
    event_id uuid PRIMARY KEY,
    contract_id text NOT NULL DEFAULT 'aviasurveil.production-feed.events'
        CHECK (contract_id = 'aviasurveil.production-feed.events'),
    contract_version text NOT NULL CHECK (contract_version = '3.0.0'),
    schema_version text NOT NULL DEFAULT '3.0.0' CHECK (schema_version = '3.0.0'),
    event_type text NOT NULL,
    event_version integer NOT NULL CHECK (event_version = 1),
    source_module text NOT NULL DEFAULT 'aviasurveil360' CHECK (source_module = 'aviasurveil360'),
    source_system text NOT NULL CHECK (source_system = 'aviasurveil-production-api'),
    tenant_id text NOT NULL CHECK (btrim(tenant_id) <> ''),
    owning_organization_id text NOT NULL CHECK (btrim(owning_organization_id) <> ''),
    actor_organization_id text,
    visibility_purpose_code text NOT NULL CHECK (visibility_purpose_code = 'regulated_oversight'),
    operation_id text NOT NULL CHECK (btrim(operation_id) <> ''),
    correlation_id uuid NOT NULL,
    causation_id uuid,
    aggregate_type text NOT NULL CHECK (btrim(aggregate_type) <> ''),
    aggregate_id text NOT NULL CHECK (btrim(aggregate_id) <> ''),
    aggregate_revision bigint NOT NULL CHECK (aggregate_revision >= 0),
    effective_at timestamptz NOT NULL,
    known_at timestamptz NOT NULL,
    occurred_at timestamptz NOT NULL,
    emitted_at timestamptz NOT NULL,
    entity_refs jsonb NOT NULL CHECK (jsonb_typeof(entity_refs) = 'object'),
    state_before text,
    state_after text NOT NULL CHECK (btrim(state_after) <> ''),
    privacy_class text NOT NULL DEFAULT 'P2' CHECK (privacy_class = 'P2'),
    payload_ciphertext bytea NOT NULL CHECK (octet_length(payload_ciphertext) > 0),
    payload_nonce bytea NOT NULL CHECK (octet_length(payload_nonce) = 12),
    payload_key_ref text NOT NULL CHECK (btrim(payload_key_ref) <> ''),
    payload_sha256 text NOT NULL CHECK (payload_sha256 ~ '^[0-9a-f]{64}$'),
    canonical_event_sha256 text NOT NULL CHECK (canonical_event_sha256 ~ '^[0-9a-f]{64}$'),
    retention_disposition text NOT NULL DEFAULT 'indefinite_immutable'
        CHECK (retention_disposition = 'indefinite_immutable'),
    legal_hold boolean NOT NULL DEFAULT false,
    created_at timestamptz NOT NULL DEFAULT now(),
    FOREIGN KEY (contract_version, event_type)
        REFERENCES datafeed_event_type_catalog (contract_version, event_type),
    UNIQUE (operation_id, event_type)
);

CREATE INDEX datafeed_events_tenant_organization_pending_idx
    ON datafeed_events (tenant_id, owning_organization_id, occurred_at, event_id);
CREATE INDEX datafeed_events_aggregate_idx
    ON datafeed_events (tenant_id, aggregate_type, aggregate_id, aggregate_revision);

CREATE TABLE datafeed_delivery_state (
    event_id uuid PRIMARY KEY REFERENCES datafeed_events(event_id),
    status text NOT NULL DEFAULT 'PENDING'
        CHECK (status IN ('PENDING', 'LEASED', 'ACKNOWLEDGED', 'DEAD_LETTER', 'TOMBSTONED')),
    lease_generation bigint NOT NULL DEFAULT 0 CHECK (lease_generation >= 0),
    lease_expires_at timestamptz,
    acknowledgement_receipt_digest text CHECK (acknowledgement_receipt_digest IS NULL OR acknowledgement_receipt_digest ~ '^[0-9a-f]{64}$'),
    acknowledged_at timestamptz,
    terminal_outcome_code text,
    dead_letter_code text,
    updated_at timestamptz NOT NULL DEFAULT now(),
    CHECK ((status = 'LEASED') = (lease_expires_at IS NOT NULL)),
    CHECK ((status = 'ACKNOWLEDGED') = (acknowledgement_receipt_digest IS NOT NULL AND acknowledged_at IS NOT NULL)),
    CHECK ((status <> 'ACKNOWLEDGED') = (acknowledgement_receipt_digest IS NULL AND acknowledged_at IS NULL)),
    CHECK ((status = 'DEAD_LETTER') = (dead_letter_code IS NOT NULL)),
    CHECK ((status <> 'DEAD_LETTER') = (dead_letter_code IS NULL))
);

CREATE INDEX datafeed_delivery_state_claim_idx
    ON datafeed_delivery_state (status, lease_expires_at, lease_generation, event_id);

CREATE TABLE datafeed_delivery_attempts (
    attempt_id uuid PRIMARY KEY,
    event_id uuid NOT NULL REFERENCES datafeed_events(event_id),
    lease_generation bigint NOT NULL CHECK (lease_generation > 0),
    outcome_code text NOT NULL CHECK (outcome_code IN ('claimed', 'accepted', 'duplicate', 'retryable', 'validation_rejected', 'conflict', 'dead_lettered')),
    acknowledgement_receipt_digest text CHECK (acknowledgement_receipt_digest IS NULL OR acknowledgement_receipt_digest ~ '^[0-9a-f]{64}$'),
    diagnostic_code text CHECK (diagnostic_code IS NULL OR diagnostic_code ~ '^[a-z][a-z0-9_]{1,63}$'),
    occurred_at timestamptz NOT NULL,
    UNIQUE (event_id, lease_generation, outcome_code, occurred_at)
);
CREATE INDEX datafeed_delivery_attempts_event_idx
    ON datafeed_delivery_attempts (event_id, lease_generation, occurred_at);

CREATE TABLE datafeed_replay_tombstones (
    event_id uuid PRIMARY KEY REFERENCES datafeed_events(event_id),
    tombstone_reason_code text NOT NULL CHECK (tombstone_reason_code IN ('legal_hold_release', 'replay_suppression')),
    recorded_at timestamptz NOT NULL,
    recorded_by_role text NOT NULL DEFAULT 'data_feed_worker' CHECK (recorded_by_role = 'data_feed_worker')
);

CREATE OR REPLACE FUNCTION datafeed_delivery_state_guard() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF NEW.lease_generation < OLD.lease_generation THEN
        RAISE EXCEPTION 'datafeed lease generation cannot move backwards';
    END IF;
    IF OLD.status = 'PENDING' AND NEW.status = 'LEASED' AND NEW.lease_generation = OLD.lease_generation + 1 THEN
        RETURN NEW;
    END IF;
    IF OLD.status = 'LEASED' AND NEW.status = 'LEASED' AND NEW.lease_generation = OLD.lease_generation + 1 AND OLD.lease_expires_at <= now() THEN
        RETURN NEW;
    END IF;
    IF OLD.status = 'LEASED' AND NEW.status IN ('ACKNOWLEDGED', 'DEAD_LETTER') AND NEW.lease_generation = OLD.lease_generation THEN
        RETURN NEW;
    END IF;
    IF OLD.status = 'PENDING' AND NEW.status = 'DEAD_LETTER' AND NEW.lease_generation = OLD.lease_generation THEN
        RETURN NEW;
    END IF;
    IF OLD.status IN ('PENDING', 'DEAD_LETTER') AND NEW.status = 'TOMBSTONED' AND NEW.lease_generation = OLD.lease_generation THEN
        RETURN NEW;
    END IF;
    IF OLD.status = 'ACKNOWLEDGED' OR OLD.status = 'TOMBSTONED' THEN
        RAISE EXCEPTION 'datafeed terminal delivery state is immutable';
    END IF;
    RAISE EXCEPTION 'datafeed delivery state requires a current fenced lease';
END;
$$;

CREATE OR REPLACE FUNCTION datafeed_replay_tombstone_guard() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF EXISTS (SELECT 1 FROM datafeed_events WHERE event_id = NEW.event_id AND legal_hold) THEN
        RAISE EXCEPTION 'datafeed legal-hold event cannot be tombstoned';
    END IF;
    UPDATE datafeed_delivery_state
       SET status = 'TOMBSTONED', lease_expires_at = NULL, updated_at = NEW.recorded_at
     WHERE event_id = NEW.event_id AND status IN ('PENDING', 'DEAD_LETTER');
    IF NOT FOUND THEN
        RAISE EXCEPTION 'datafeed replay tombstone requires an unacknowledged event';
    END IF;
    RETURN NEW;
END;
$$;

CREATE TRIGGER datafeed_event_type_catalog_append_only
BEFORE UPDATE OR DELETE ON datafeed_event_type_catalog
FOR EACH ROW EXECUTE FUNCTION reject_immutable_row_change();
CREATE TRIGGER datafeed_events_append_only
BEFORE UPDATE OR DELETE ON datafeed_events
FOR EACH ROW EXECUTE FUNCTION reject_immutable_row_change();
CREATE TRIGGER datafeed_delivery_attempts_append_only
BEFORE UPDATE OR DELETE ON datafeed_delivery_attempts
FOR EACH ROW EXECUTE FUNCTION reject_immutable_row_change();
CREATE TRIGGER datafeed_replay_tombstones_append_only
BEFORE UPDATE OR DELETE ON datafeed_replay_tombstones
FOR EACH ROW EXECUTE FUNCTION reject_immutable_row_change();
CREATE TRIGGER datafeed_delivery_state_transition_guard
BEFORE UPDATE ON datafeed_delivery_state
FOR EACH ROW EXECUTE FUNCTION datafeed_delivery_state_guard();
CREATE TRIGGER datafeed_replay_tombstone_transition_guard
BEFORE INSERT ON datafeed_replay_tombstones
FOR EACH ROW EXECUTE FUNCTION datafeed_replay_tombstone_guard();

CREATE OR REPLACE FUNCTION datafeed_create_delivery_state() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    INSERT INTO datafeed_delivery_state (event_id) VALUES (NEW.event_id);
    RETURN NEW;
END;
$$;
CREATE TRIGGER datafeed_events_create_delivery_state
AFTER INSERT ON datafeed_events
FOR EACH ROW EXECUTE FUNCTION datafeed_create_delivery_state();
