CREATE TABLE auth_identity.provider_sessions (
    session_id text PRIMARY KEY CHECK (session_id ~ '^ses_[A-Za-z0-9_-]{22}$'),
    subject_id text NOT NULL REFERENCES auth_identity.accounts(subject_id),
    family_id text NOT NULL UNIQUE CHECK (family_id ~ '^fam_[A-Za-z0-9_-]{22}$'),
    client_id text NOT NULL CHECK (btrim(client_id) <> ''),
    fingerprint_hash bytea NOT NULL CHECK (octet_length(fingerprint_hash) = 32),
    auth_revision bigint NOT NULL CHECK (auth_revision > 0),
    state text NOT NULL CHECK (state IN ('active', 'revoked', 'expired')),
    issued_at timestamptz NOT NULL,
    last_used_at timestamptz NOT NULL,
    idle_expires_at timestamptz NOT NULL,
    absolute_expires_at timestamptz NOT NULL,
    revoked_at timestamptz,
    CHECK (idle_expires_at <= absolute_expires_at)
);

CREATE INDEX auth_identity_provider_sessions_subject_idx
    ON auth_identity.provider_sessions (subject_id, state, issued_at);

CREATE TABLE auth_identity.refresh_families (
    family_id text PRIMARY KEY REFERENCES auth_identity.provider_sessions(family_id),
    session_id text NOT NULL UNIQUE REFERENCES auth_identity.provider_sessions(session_id),
    current_token_hash bytea NOT NULL UNIQUE CHECK (octet_length(current_token_hash) = 32),
    generation bigint NOT NULL CHECK (generation > 0),
    state text NOT NULL CHECK (state IN ('active', 'revoked', 'reuse-detected', 'expired')),
    issued_at timestamptz NOT NULL,
    last_used_at timestamptz NOT NULL,
    idle_expires_at timestamptz NOT NULL,
    absolute_expires_at timestamptz NOT NULL,
    revoked_at timestamptz,
    CHECK (idle_expires_at <= absolute_expires_at)
);

CREATE TABLE auth_identity.refresh_token_history (
    token_hash bytea PRIMARY KEY CHECK (octet_length(token_hash) = 32),
    family_id text NOT NULL REFERENCES auth_identity.refresh_families(family_id),
    used_at timestamptz NOT NULL
);

CREATE INDEX auth_identity_refresh_history_family_idx
    ON auth_identity.refresh_token_history (family_id, used_at DESC);

COMMENT ON TABLE auth_identity.provider_sessions IS
    'Opaque provider sessions; browser refresh credentials are never stored raw and are not exposed to application roles.';
COMMENT ON TABLE auth_identity.refresh_families IS
    'Single-use refresh rotation state; concurrent rotations must lock the family row before changing current_token_hash.';
COMMENT ON TABLE auth_identity.refresh_token_history IS
    'Hashed consumed refresh tokens retained for family-reuse detection and durable incident response.';
