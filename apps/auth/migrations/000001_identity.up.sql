CREATE SCHEMA IF NOT EXISTS auth_identity;

CREATE TABLE auth_identity.accounts (
    subject_id text PRIMARY KEY,
    state text NOT NULL CHECK (state IN (
        'invited', 'active', 'disabled', 'suspended', 'locked',
        'deletion-pending', 'deleted'
    )),
    password_hash text,
    email_verified boolean NOT NULL DEFAULT false,
    auth_revision bigint NOT NULL CHECK (auth_revision > 0),
    failed_login_count integer NOT NULL DEFAULT 0 CHECK (failed_login_count >= 0),
    locked_until timestamptz,
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,
    CHECK (subject_id ~ '^usr_[A-Za-z0-9_-]{22}$'),
    CHECK (state = 'invited' OR password_hash IS NOT NULL),
    CHECK (state <> 'active' OR email_verified = true)
);

CREATE TABLE auth_identity.identifiers (
    subject_id text NOT NULL REFERENCES auth_identity.accounts(subject_id),
    identifier_type text NOT NULL CHECK (identifier_type IN ('email', 'username')),
    normalized_value text NOT NULL CHECK (btrim(normalized_value) <> ''),
    verified_at timestamptz,
    created_at timestamptz NOT NULL,
    PRIMARY KEY (subject_id, identifier_type),
    UNIQUE (normalized_value)
);

CREATE INDEX auth_identity_identifiers_lookup_idx
    ON auth_identity.identifiers (normalized_value, identifier_type, subject_id);

CREATE TABLE auth_identity.password_history (
    subject_id text NOT NULL REFERENCES auth_identity.accounts(subject_id),
    history_revision bigint NOT NULL CHECK (history_revision > 0),
    password_hash text NOT NULL,
    created_at timestamptz NOT NULL,
    PRIMARY KEY (subject_id, history_revision)
);

CREATE TABLE auth_identity.invitations (
    invitation_id text PRIMARY KEY CHECK (invitation_id ~ '^inv_[A-Za-z0-9_-]{22}$'),
    subject_id text NOT NULL REFERENCES auth_identity.accounts(subject_id),
    token_hash bytea NOT NULL UNIQUE,
    state text NOT NULL CHECK (state IN (
        'issued', 'delivery-accepted', 'retryable-failure', 'terminal-failure',
        'expired', 'consumed', 'cancelled'
    )),
    resend_count integer NOT NULL DEFAULT 0 CHECK (resend_count >= 0 AND resend_count <= 3),
    issued_at timestamptz NOT NULL,
    expires_at timestamptz NOT NULL,
    invalidated_at timestamptz,
    consumed_at timestamptz,
    CHECK (expires_at > issued_at)
);

CREATE INDEX auth_identity_invitations_subject_idx
    ON auth_identity.invitations (subject_id, issued_at DESC, invitation_id DESC);

CREATE TABLE auth_identity.throttle_buckets (
    bucket_key text PRIMARY KEY CHECK (btrim(bucket_key) <> ''),
    window_started timestamptz NOT NULL,
    request_count integer NOT NULL CHECK (request_count >= 0),
    updated_at timestamptz NOT NULL
);

COMMENT ON SCHEMA auth_identity IS
    'Privileged first-party identity data; never grant this schema to the application or worker role.';
COMMENT ON TABLE auth_identity.identifiers IS
    'Canonical type-aware identifiers with global normalized-value uniqueness to prevent cross-field collisions.';
COMMENT ON TABLE auth_identity.throttle_buckets IS
    'Fail-closed admission state; sensitive operations must not proceed when this backend is unavailable.';
