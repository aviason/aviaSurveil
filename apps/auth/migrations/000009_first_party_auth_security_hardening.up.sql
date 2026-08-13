ALTER TABLE auth_identity.oidc_auth_requests
    ADD COLUMN IF NOT EXISTS authenticating_auth_revision bigint,
    ADD COLUMN IF NOT EXISTS mfa_attempt_count integer NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS mfa_attempt_limit integer NOT NULL DEFAULT 5,
    ADD COLUMN IF NOT EXISTS invalidated_at timestamptz,
    ADD COLUMN IF NOT EXISTS invalidation_reason text,
    ADD COLUMN IF NOT EXISTS browser_binding_bootstrap boolean NOT NULL DEFAULT false;

ALTER TABLE auth_identity.oidc_auth_requests
    ADD CONSTRAINT oidc_auth_requests_auth_revision_check
        CHECK (authenticating_auth_revision IS NULL OR authenticating_auth_revision > 0),
    ADD CONSTRAINT oidc_auth_requests_mfa_attempt_count_check
        CHECK (mfa_attempt_count >= 0 AND mfa_attempt_count <= mfa_attempt_limit),
    ADD CONSTRAINT oidc_auth_requests_mfa_attempt_limit_check
        CHECK (mfa_attempt_limit BETWEEN 1 AND 20),
    ADD CONSTRAINT oidc_auth_requests_invalidation_check
        CHECK ((invalidated_at IS NULL AND invalidation_reason IS NULL)
            OR (invalidated_at IS NOT NULL AND btrim(invalidation_reason) <> '' AND length(invalidation_reason) <= 100));

CREATE INDEX IF NOT EXISTS auth_identity_oidc_auth_requests_subject_revision_idx
    ON auth_identity.oidc_auth_requests (subject_id, authenticating_auth_revision, created_at DESC);

CREATE INDEX IF NOT EXISTS auth_identity_oidc_auth_requests_outstanding_idx
    ON auth_identity.oidc_auth_requests (client_id, created_at, request_id)
    WHERE done = false AND invalidated_at IS NULL;

CREATE INDEX IF NOT EXISTS auth_identity_oidc_auth_requests_bootstrap_outstanding_idx
    ON auth_identity.oidc_auth_requests (expires_at, created_at, request_id)
    WHERE done = false AND invalidated_at IS NULL AND browser_binding_bootstrap = true;

CREATE INDEX IF NOT EXISTS auth_identity_oidc_auth_requests_cleanup_idx
    ON auth_identity.oidc_auth_requests (expires_at, created_at, request_id);

ALTER TABLE auth_identity.mfa_factors
    ADD COLUMN IF NOT EXISTS recovery_failure_window_started_at timestamptz,
    ADD COLUMN IF NOT EXISTS recovery_locked_until timestamptz;

ALTER TABLE auth_identity.mfa_factors
    ADD CONSTRAINT mfa_recovery_window_consistency_check
        CHECK (recovery_failure_window_started_at IS NULL OR recovery_failure_window_started_at <= updated_at),
    ADD CONSTRAINT mfa_recovery_lock_consistency_check
        CHECK (recovery_locked_until IS NULL OR recovery_failure_window_started_at IS NOT NULL);

WITH ranked AS (
    SELECT token_hash,
           row_number() OVER (
               PARTITION BY subject_id, purpose
               ORDER BY created_at DESC, token_hash DESC
           ) AS position
    FROM auth_identity.identity_challenges
    WHERE state = 'active'
)
UPDATE auth_identity.identity_challenges challenge
SET state = 'invalidated', invalidated_at = now(), updated_at = now()
FROM ranked
WHERE challenge.token_hash = ranked.token_hash
  AND ranked.position > 1;

CREATE UNIQUE INDEX IF NOT EXISTS auth_identity_identity_challenges_one_active_idx
    ON auth_identity.identity_challenges (subject_id, purpose)
    WHERE state = 'active';

ALTER TABLE auth_identity.mail_deliveries
    ADD COLUMN IF NOT EXISTS dedupe_key text;

ALTER TABLE auth_identity.mail_deliveries
    ADD CONSTRAINT mail_deliveries_dedupe_key_check
        CHECK (dedupe_key IS NULL OR (btrim(dedupe_key) <> '' AND length(dedupe_key) <= 160));

CREATE UNIQUE INDEX IF NOT EXISTS auth_identity_mail_deliveries_pending_dedupe_idx
    ON auth_identity.mail_deliveries (dedupe_key)
    WHERE dedupe_key IS NOT NULL AND state IN ('queued', 'leased', 'retryable');

CREATE INDEX IF NOT EXISTS auth_identity_throttle_buckets_cleanup_idx
    ON auth_identity.throttle_buckets (updated_at, bucket_key);

ALTER TABLE auth_identity.throttle_buckets
    ADD COLUMN IF NOT EXISTS window_ends_at timestamptz;

UPDATE auth_identity.throttle_buckets
SET window_ends_at = GREATEST(window_started, updated_at) + interval '1 hour'
WHERE window_ends_at IS NULL;

ALTER TABLE auth_identity.throttle_buckets
    ALTER COLUMN window_ends_at SET NOT NULL,
    ADD CONSTRAINT throttle_buckets_window_bounds_check
        CHECK (window_ends_at > window_started);

CREATE INDEX IF NOT EXISTS auth_identity_throttle_buckets_window_cleanup_idx
    ON auth_identity.throttle_buckets (window_ends_at, bucket_key);

CREATE INDEX IF NOT EXISTS auth_identity_mail_deliveries_retention_idx
    ON auth_identity.mail_deliveries (state, updated_at, delivery_id)
    WHERE state IN ('delivered', 'terminal-failure');
