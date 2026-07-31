-- Task 5: durable direct-mTLS publisher scheduling and manual quarantine.
-- This is forward-only: no immutable producer event or attempt is rewritten.

ALTER TABLE datafeed_delivery_state
    ADD COLUMN attempt_count integer NOT NULL DEFAULT 0 CHECK (attempt_count >= 0),
    ADD COLUMN next_attempt_at timestamptz,
    ADD COLUMN quarantine_owner_role text,
    ADD COLUMN quarantine_sla_due_at timestamptz;

ALTER TABLE datafeed_delivery_state
    DROP CONSTRAINT IF EXISTS datafeed_delivery_state_status_check;
ALTER TABLE datafeed_delivery_state
    ADD CONSTRAINT datafeed_delivery_state_status_check
    CHECK (status IN ('PENDING', 'LEASED', 'ACKNOWLEDGED', 'DEAD_LETTER', 'QUARANTINED', 'TOMBSTONED'));
ALTER TABLE datafeed_delivery_state
    ADD CONSTRAINT datafeed_delivery_state_quarantine_shape_check
    CHECK ((status = 'QUARANTINED') = (quarantine_owner_role IS NOT NULL AND quarantine_sla_due_at IS NOT NULL));

ALTER TABLE datafeed_delivery_attempts
    DROP CONSTRAINT IF EXISTS datafeed_delivery_attempts_outcome_code_check;
ALTER TABLE datafeed_delivery_attempts
    ADD CONSTRAINT datafeed_delivery_attempts_outcome_code_check
    CHECK (outcome_code IN ('claimed', 'accepted', 'duplicate', 'retryable', 'retry_exhausted', 'validation_rejected', 'conflict', 'dead_lettered'));

CREATE OR REPLACE FUNCTION datafeed_delivery_state_guard() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF NEW.lease_generation < OLD.lease_generation THEN
        RAISE EXCEPTION 'datafeed lease generation cannot move backwards';
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
    IF OLD.status = 'LEASED' AND NEW.status IN ('ACKNOWLEDGED', 'DEAD_LETTER', 'QUARANTINED')
       AND NEW.lease_generation = OLD.lease_generation THEN
        RETURN NEW;
    END IF;
    IF OLD.status = 'PENDING' AND NEW.status = 'DEAD_LETTER'
       AND NEW.lease_generation = OLD.lease_generation THEN
        RETURN NEW;
    END IF;
    IF OLD.status IN ('PENDING', 'DEAD_LETTER', 'QUARANTINED') AND NEW.status = 'TOMBSTONED'
       AND NEW.lease_generation = OLD.lease_generation THEN
        RETURN NEW;
    END IF;
    IF OLD.status IN ('ACKNOWLEDGED', 'TOMBSTONED', 'QUARANTINED') THEN
        RAISE EXCEPTION 'datafeed terminal delivery state is immutable';
    END IF;
    RAISE EXCEPTION 'datafeed delivery state requires a current fenced lease';
END;
$$;

CREATE INDEX datafeed_delivery_state_next_attempt_idx
    ON datafeed_delivery_state (status, next_attempt_at, event_id)
    WHERE status = 'PENDING';
