CREATE TABLE auth_identity.provider_clients (
    client_id text PRIMARY KEY CHECK (client_id ~ '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$'),
    secret_hash bytea NOT NULL CHECK (octet_length(secret_hash) = 32),
    redirect_uris text[] NOT NULL CHECK (cardinality(redirect_uris) > 0),
    post_logout_redirect_uris text[] NOT NULL DEFAULT '{}',
    scopes text[] NOT NULL DEFAULT '{}',
    state text NOT NULL CHECK (state IN ('active', 'revoked')),
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL
);

CREATE TABLE auth_identity.provider_signing_keys (
    key_id text PRIMARY KEY CHECK (key_id ~ '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$'),
    algorithm text NOT NULL CHECK (algorithm = 'RS256'),
    private_key_ciphertext bytea NOT NULL,
    public_key bytea NOT NULL,
    state text NOT NULL CHECK (state IN ('active', 'overlap', 'retired')),
    created_at timestamptz NOT NULL,
    retire_at timestamptz
);

CREATE UNIQUE INDEX auth_identity_one_active_signing_key_idx
    ON auth_identity.provider_signing_keys ((state)) WHERE state = 'active';

CREATE TABLE auth_identity.audit_events (
    event_id text PRIMARY KEY CHECK (event_id ~ '^evt_[A-Fa-f0-9]{32}$'),
    event_at timestamptz NOT NULL,
    event_type text NOT NULL CHECK (btrim(event_type) <> ''),
    outcome text NOT NULL CHECK (btrim(outcome) <> ''),
    subject_id text REFERENCES auth_identity.accounts(subject_id),
    actor_id text,
    client_id text,
    request_id text,
    fields jsonb NOT NULL DEFAULT '{}'::jsonb,
    CHECK (jsonb_typeof(fields) = 'object')
);

CREATE INDEX auth_identity_audit_events_subject_idx
    ON auth_identity.audit_events (subject_id, event_at DESC);
CREATE INDEX auth_identity_audit_events_type_idx
    ON auth_identity.audit_events (event_type, event_at DESC);

COMMENT ON TABLE auth_identity.provider_clients IS
    'Exact first-party OIDC client registry; redirect and logout URI values are never wildcard patterns.';
COMMENT ON TABLE auth_identity.provider_signing_keys IS
    'RS256 key ring with explicit active/overlap/retired lifecycle and encrypted private material.';
COMMENT ON TABLE auth_identity.audit_events IS
    'Append-only redacted security events; application roles must not update or delete rows.';
