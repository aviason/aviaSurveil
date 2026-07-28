ALTER TABLE user_lifecycle_requests
    DROP CONSTRAINT IF EXISTS user_lifecycle_requests_requested_action_check,
    DROP CONSTRAINT IF EXISTS user_lifecycle_requests_status_check,
    ADD COLUMN expected_membership_revision bigint NOT NULL DEFAULT 0,
    ADD COLUMN reason text NOT NULL DEFAULT 'legacy lifecycle request',
    ADD COLUMN requested_effective_at timestamptz,
    ADD COLUMN membership_id text,
    ADD COLUMN resulting_membership_revision bigint,
    ADD COLUMN provider_failure_class text,
    ADD COLUMN provider_acknowledged_at timestamptz;

ALTER TABLE user_lifecycle_requests
    ALTER COLUMN expected_membership_revision DROP DEFAULT,
    ALTER COLUMN reason DROP DEFAULT,
    ADD CONSTRAINT user_lifecycle_requests_requested_action_check CHECK (
        requested_action IN (
            'PROVISION',
            'UPDATE_ROLES',
            'SUSPEND',
            'REACTIVATE',
            'DEACTIVATE',
            'TRANSFER_ORGANIZATION',
            'RESEND_INVITATION',
            'RESET_PASSWORD',
            'RESET_MFA',
            'FORCE_LOGOUT'
        )
    ),
    ADD CONSTRAINT user_lifecycle_requests_status_check CHECK (
        status IN (
            'PENDING',
            'RUNNING',
            'SUCCEEDED',
            'FAILED',
            'FAILED_RETRYABLE',
            'FAILED_PERMANENT',
            'MANUAL_REVIEW'
        )
    ),
    ADD CONSTRAINT user_lifecycle_requests_expected_revision_check CHECK (
        expected_membership_revision >= 0
    ),
    ADD CONSTRAINT user_lifecycle_requests_reason_check CHECK (
        btrim(reason) <> ''
    ),
    ADD CONSTRAINT user_lifecycle_requests_resulting_revision_check CHECK (
        resulting_membership_revision IS NULL
        OR resulting_membership_revision > 0
    ),
    ADD CONSTRAINT user_lifecycle_requests_failure_class_check CHECK (
        provider_failure_class IS NULL
        OR provider_failure_class IN ('RETRYABLE', 'PERMANENT', 'MANUAL_REVIEW')
    );

ALTER TABLE identity_references
    ADD COLUMN deactivated_at timestamptz;

ALTER TABLE desired_membership_versions
    DROP CONSTRAINT IF EXISTS desired_membership_versions_source_request_id_key;

CREATE TABLE identity_action_facts (
    id text PRIMARY KEY,
    request_id text NOT NULL REFERENCES user_lifecycle_requests(id),
    fact_sequence integer NOT NULL CHECK (fact_sequence > 0),
    membership_id text,
    subject_id text REFERENCES identity_references(subject_id),
    action_kind text NOT NULL CHECK (
        action_kind IN ('INVITATION', 'RECOVERY', 'MFA_RESET')
    ),
    state text NOT NULL CHECK (
        state IN (
            'ISSUED',
            'DELIVERY_ACCEPTED',
            'RETRYABLE_FAILURE',
            'TERMINAL_FAILURE',
            'EXPIRED',
            'CONSUMED',
            'CANCELLED',
            'RESET_PENDING',
            'RESET_COMPLETED'
        )
    ),
    delivery_attempt integer NOT NULL CHECK (delivery_attempt > 0),
    expires_at timestamptz,
    provider_acknowledged_at timestamptz,
    reason text NOT NULL CHECK (btrim(reason) <> ''),
    created_at timestamptz NOT NULL,
    UNIQUE (request_id, fact_sequence)
);

CREATE INDEX identity_action_facts_subject_created_idx
    ON identity_action_facts (subject_id, created_at DESC, id DESC);

CREATE INDEX identity_action_facts_membership_created_idx
    ON identity_action_facts (membership_id, created_at DESC, id DESC);

CREATE TRIGGER identity_action_facts_immutable
BEFORE UPDATE OR DELETE ON identity_action_facts
FOR EACH ROW EXECUTE FUNCTION reject_immutable_row_change();

CREATE TABLE identity_lifecycle_alerts (
    id text PRIMARY KEY,
    request_id text NOT NULL REFERENCES user_lifecycle_requests(id),
    severity text NOT NULL CHECK (severity IN ('WARNING', 'CRITICAL')),
    failure_class text NOT NULL CHECK (
        failure_class IN ('PERMANENT', 'MANUAL_REVIEW')
    ),
    reason_code text NOT NULL CHECK (btrim(reason_code) <> ''),
    created_at timestamptz NOT NULL
);

CREATE INDEX identity_lifecycle_alerts_created_idx
    ON identity_lifecycle_alerts (created_at DESC, id DESC);

CREATE TRIGGER identity_lifecycle_alerts_immutable
BEFORE UPDATE OR DELETE ON identity_lifecycle_alerts
FOR EACH ROW EXECUTE FUNCTION reject_immutable_row_change();
