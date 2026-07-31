-- Task 6: approval-bound replay/backfill run membership. These rows preserve
-- source-event identity and time snapshots; they never reopen or rewrite the
-- event's original delivery state or historical occurrence.

CREATE TABLE datafeed_replay_runs (
    run_id uuid PRIMARY KEY,
    run_kind text NOT NULL CHECK (run_kind IN ('REPLAY', 'BACKFILL')),
    approval_id uuid NOT NULL,
    request_sha256 text NOT NULL CHECK (request_sha256 ~ '^[0-9a-f]{64}$'),
    tenant_id text NOT NULL CHECK (btrim(tenant_id) <> ''),
    owning_organization_id text NOT NULL CHECK (btrim(owning_organization_id) <> ''),
    source_system text NOT NULL CHECK (source_system = 'aviasurveil-production-api'),
    contract_version text NOT NULL CHECK (contract_version = '3.0.0'),
    source_cut_id text,
    source_manifest_sha256 text CHECK (source_manifest_sha256 IS NULL OR source_manifest_sha256 ~ '^[0-9a-f]{64}$'),
    cut_at timestamptz,
    requested_at timestamptz NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (approval_id, request_sha256),
    CHECK (
        (run_kind = 'REPLAY' AND source_cut_id IS NULL AND source_manifest_sha256 IS NULL AND cut_at IS NULL)
        OR
        (run_kind = 'BACKFILL' AND btrim(source_cut_id) <> '' AND source_manifest_sha256 IS NOT NULL AND cut_at IS NOT NULL AND cut_at <= requested_at)
    )
);

CREATE TABLE datafeed_replay_run_events (
    run_id uuid NOT NULL REFERENCES datafeed_replay_runs(run_id),
    event_id uuid NOT NULL REFERENCES datafeed_events(event_id),
    canonical_event_sha256 text NOT NULL CHECK (canonical_event_sha256 ~ '^[0-9a-f]{64}$'),
    effective_at timestamptz NOT NULL,
    known_at timestamptz NOT NULL,
    occurred_at timestamptz NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (run_id, event_id)
);

CREATE TABLE datafeed_replay_delivery_state (
    run_id uuid NOT NULL,
    event_id uuid NOT NULL,
    status text NOT NULL DEFAULT 'PENDING'
        CHECK (status IN ('PENDING', 'LEASED', 'ACKNOWLEDGED', 'QUARANTINED')),
    lease_generation bigint NOT NULL DEFAULT 0 CHECK (lease_generation >= 0),
    lease_expires_at timestamptz,
    attempt_count integer NOT NULL DEFAULT 0 CHECK (attempt_count >= 0),
    next_attempt_at timestamptz,
    acknowledgement_receipt_digest text CHECK (acknowledgement_receipt_digest IS NULL OR acknowledgement_receipt_digest ~ '^[0-9a-f]{64}$'),
    acknowledged_at timestamptz,
    terminal_outcome_code text,
    quarantine_owner_role text,
    quarantine_sla_due_at timestamptz,
    updated_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (run_id, event_id),
    FOREIGN KEY (run_id, event_id) REFERENCES datafeed_replay_run_events(run_id, event_id),
    CHECK ((status = 'LEASED') = (lease_expires_at IS NOT NULL)),
    CHECK ((status = 'ACKNOWLEDGED') = (acknowledgement_receipt_digest IS NOT NULL AND acknowledged_at IS NOT NULL)),
    CHECK ((status <> 'ACKNOWLEDGED') = (acknowledgement_receipt_digest IS NULL AND acknowledged_at IS NULL)),
    CHECK ((status = 'QUARANTINED') = (quarantine_owner_role IS NOT NULL AND quarantine_sla_due_at IS NOT NULL))
);

CREATE TABLE datafeed_replay_delivery_attempts (
    attempt_id uuid PRIMARY KEY,
    run_id uuid NOT NULL,
    event_id uuid NOT NULL,
    lease_generation bigint NOT NULL CHECK (lease_generation > 0),
    outcome_code text NOT NULL CHECK (outcome_code IN ('claimed', 'accepted', 'duplicate', 'retryable', 'retry_exhausted', 'validation_rejected', 'conflict')),
    acknowledgement_receipt_digest text CHECK (acknowledgement_receipt_digest IS NULL OR acknowledgement_receipt_digest ~ '^[0-9a-f]{64}$'),
    diagnostic_code text CHECK (diagnostic_code IS NULL OR diagnostic_code ~ '^[a-z][a-z0-9_]{1,63}$'),
    occurred_at timestamptz NOT NULL,
    FOREIGN KEY (run_id, event_id) REFERENCES datafeed_replay_delivery_state(run_id, event_id),
    UNIQUE (run_id, event_id, lease_generation, outcome_code, occurred_at)
);

CREATE OR REPLACE FUNCTION datafeed_replay_run_event_guard() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM datafeed_replay_runs run
        JOIN datafeed_events event ON event.event_id = NEW.event_id
        WHERE run.run_id = NEW.run_id
          AND run.tenant_id = event.tenant_id
          AND run.owning_organization_id = event.owning_organization_id
          AND run.source_system = event.source_system
          AND run.contract_version = event.contract_version
          AND NEW.canonical_event_sha256 = event.canonical_event_sha256
          AND NEW.effective_at = event.effective_at
          AND NEW.known_at = event.known_at
          AND NEW.occurred_at = event.occurred_at
    ) THEN
        RAISE EXCEPTION 'datafeed replay membership must preserve the exact scoped source event snapshot';
    END IF;
    RETURN NEW;
END;
$$;

CREATE OR REPLACE FUNCTION datafeed_replay_delivery_state_guard() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF NEW.lease_generation < OLD.lease_generation THEN
        RAISE EXCEPTION 'datafeed replay lease generation cannot move backwards';
    END IF;
    IF OLD.status = 'PENDING' AND NEW.status = 'LEASED'
       AND NEW.lease_generation = OLD.lease_generation + 1 THEN
        RETURN NEW;
    END IF;
    IF OLD.status = 'LEASED' AND NEW.status = 'LEASED'
       AND NEW.lease_generation = OLD.lease_generation + 1
       AND OLD.lease_expires_at <= now() THEN
        RETURN NEW;
    END IF;
    IF OLD.status = 'LEASED' AND NEW.status = 'PENDING'
       AND NEW.lease_generation = OLD.lease_generation
       AND NEW.lease_expires_at IS NULL AND NEW.next_attempt_at IS NOT NULL
       AND NEW.attempt_count = OLD.attempt_count + 1 THEN
        RETURN NEW;
    END IF;
    IF OLD.status = 'LEASED' AND NEW.status IN ('ACKNOWLEDGED', 'QUARANTINED')
       AND NEW.lease_generation = OLD.lease_generation THEN
        RETURN NEW;
    END IF;
    IF OLD.status IN ('ACKNOWLEDGED', 'QUARANTINED') THEN
        RAISE EXCEPTION 'datafeed replay terminal delivery state is immutable';
    END IF;
    RAISE EXCEPTION 'datafeed replay delivery state requires a current fenced lease';
END;
$$;

CREATE TRIGGER datafeed_replay_runs_append_only
BEFORE UPDATE OR DELETE ON datafeed_replay_runs
FOR EACH ROW EXECUTE FUNCTION reject_immutable_row_change();
CREATE TRIGGER datafeed_replay_run_events_append_only
BEFORE UPDATE OR DELETE ON datafeed_replay_run_events
FOR EACH ROW EXECUTE FUNCTION reject_immutable_row_change();
CREATE TRIGGER datafeed_replay_run_events_source_snapshot_guard
BEFORE INSERT ON datafeed_replay_run_events
FOR EACH ROW EXECUTE FUNCTION datafeed_replay_run_event_guard();
CREATE TRIGGER datafeed_replay_delivery_state_transition_guard
BEFORE UPDATE ON datafeed_replay_delivery_state
FOR EACH ROW EXECUTE FUNCTION datafeed_replay_delivery_state_guard();
CREATE TRIGGER datafeed_replay_delivery_attempts_append_only
BEFORE UPDATE OR DELETE ON datafeed_replay_delivery_attempts
FOR EACH ROW EXECUTE FUNCTION reject_immutable_row_change();

CREATE INDEX datafeed_replay_runs_scope_requested_idx
    ON datafeed_replay_runs (tenant_id, owning_organization_id, requested_at, run_id);
CREATE INDEX datafeed_replay_run_events_event_idx
    ON datafeed_replay_run_events (event_id, run_id);
CREATE INDEX datafeed_replay_delivery_state_claim_idx
    ON datafeed_replay_delivery_state (status, next_attempt_at, lease_generation, run_id, event_id)
    WHERE status = 'PENDING';
