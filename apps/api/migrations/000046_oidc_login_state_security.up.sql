-- A live state has no safe browser binding to invent. Expired ephemeral rows
-- are the only rows this migration is permitted to remove.
DELETE FROM oidc_login_states
WHERE expires_at <= now();

ALTER TABLE oidc_login_states
    ADD COLUMN browser_binding_hash text,
    ADD COLUMN browser_binding_bootstrap boolean NOT NULL DEFAULT false;

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM oidc_login_states WHERE browser_binding_hash IS NULL) THEN
        RAISE EXCEPTION 'cannot migrate live OIDC login state without browser binding';
    END IF;
END
$$;

ALTER TABLE oidc_login_states
    ALTER COLUMN browser_binding_hash SET NOT NULL,
    ADD CONSTRAINT oidc_login_states_browser_binding_hash_check
        CHECK (browser_binding_hash ~ '^[0-9a-f]{64}$');

CREATE INDEX oidc_login_states_expiry_created_idx
    ON oidc_login_states (expires_at, created_at, state_hash);

CREATE INDEX oidc_login_states_outstanding_idx
    ON oidc_login_states (created_at, state_hash)
    WHERE expires_at IS NOT NULL;

CREATE INDEX oidc_login_states_bootstrap_outstanding_idx
    ON oidc_login_states (expires_at, created_at, state_hash)
    WHERE browser_binding_bootstrap = true;

CREATE TABLE oidc_login_admission (
    rule_key text PRIMARY KEY CHECK (btrim(rule_key) <> '' AND length(rule_key) <= 256),
    window_started timestamptz NOT NULL,
    request_count integer NOT NULL CHECK (request_count >= 0 AND request_count <= 1000000),
    updated_at timestamptz NOT NULL
);

CREATE INDEX oidc_login_admission_cleanup_idx
    ON oidc_login_admission (updated_at, rule_key);
