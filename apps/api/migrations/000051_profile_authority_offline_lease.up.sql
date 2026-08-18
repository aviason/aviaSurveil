ALTER TABLE offline_grants
    ADD COLUMN IF NOT EXISTS profile_key_id text NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS profile_public_jwk jsonb NOT NULL DEFAULT '{}'::jsonb,
    ADD COLUMN IF NOT EXISTS lease_issued_at timestamptz,
    ADD COLUMN IF NOT EXISTS lease_expires_at timestamptz;

UPDATE offline_grants
SET lease_issued_at = COALESCE(lease_issued_at, granted_at),
    lease_expires_at = COALESCE(lease_expires_at, expires_at)
WHERE lease_issued_at IS NULL OR lease_expires_at IS NULL;

CREATE INDEX IF NOT EXISTS offline_grants_profile_key_scope_idx
    ON offline_grants (subject_id, profile_key_id, package_id);
