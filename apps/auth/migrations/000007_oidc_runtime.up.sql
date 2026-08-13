CREATE TABLE auth_identity.oidc_auth_requests (
    request_id text PRIMARY KEY CHECK (request_id ~ '^req_[a-f0-9]{48}$'),
    client_id text NOT NULL REFERENCES auth_identity.provider_clients(client_id),
    redirect_uri text NOT NULL,
    state_value text NOT NULL,
    nonce_value text NOT NULL,
    response_type text NOT NULL CHECK (response_type = 'code'),
    response_mode text NOT NULL,
    scopes text[] NOT NULL CHECK (cardinality(scopes) > 0),
    code_challenge text,
    code_challenge_method text,
    subject_id text REFERENCES auth_identity.accounts(subject_id),
    done boolean NOT NULL DEFAULT false,
    auth_time timestamptz NOT NULL,
    created_at timestamptz NOT NULL,
    CHECK ((code_challenge IS NULL) = (code_challenge_method IS NULL)),
    CHECK (NOT done OR subject_id IS NOT NULL)
);

CREATE TABLE auth_identity.oidc_authorization_codes (
    code_hash bytea PRIMARY KEY CHECK (octet_length(code_hash) = 32),
    request_id text NOT NULL UNIQUE REFERENCES auth_identity.oidc_auth_requests(request_id) ON DELETE CASCADE,
    created_at timestamptz NOT NULL
);

CREATE TABLE auth_identity.oidc_access_tokens (
    token_id text PRIMARY KEY CHECK (token_id ~ '^at_[a-f0-9]{48}$'),
    client_id text NOT NULL REFERENCES auth_identity.provider_clients(client_id),
    subject_id text NOT NULL REFERENCES auth_identity.accounts(subject_id),
    audience text[] NOT NULL,
    scopes text[] NOT NULL,
    expires_at timestamptz NOT NULL,
    state text NOT NULL CHECK (state IN ('active', 'revoked')),
    created_at timestamptz NOT NULL,
    revoked_at timestamptz
);

CREATE INDEX auth_identity_oidc_access_tokens_subject_client_idx
    ON auth_identity.oidc_access_tokens (subject_id, client_id, state, expires_at);

CREATE TABLE auth_identity.oidc_refresh_tokens (
    token_hash bytea PRIMARY KEY CHECK (octet_length(token_hash) = 32),
    access_token_id text NOT NULL REFERENCES auth_identity.oidc_access_tokens(token_id),
    client_id text NOT NULL REFERENCES auth_identity.provider_clients(client_id),
    subject_id text NOT NULL REFERENCES auth_identity.accounts(subject_id),
    audience text[] NOT NULL,
    scopes text[] NOT NULL,
    auth_time timestamptz NOT NULL,
    amr text[] NOT NULL,
    expires_at timestamptz NOT NULL,
    state text NOT NULL CHECK (state IN ('active', 'rotated', 'revoked')),
    created_at timestamptz NOT NULL,
    revoked_at timestamptz
);

CREATE INDEX auth_identity_oidc_refresh_tokens_subject_client_idx
    ON auth_identity.oidc_refresh_tokens (subject_id, client_id, state, expires_at);

COMMENT ON TABLE auth_identity.oidc_auth_requests IS
    'Durable OIDC authorization requests; one code is stored only as a hash and is deleted after exchange.';
COMMENT ON TABLE auth_identity.oidc_authorization_codes IS
    'Hashed one-use authorization codes; raw authorization codes are never persisted.';
COMMENT ON TABLE auth_identity.oidc_access_tokens IS
    'Durable opaque OIDC access-token identifiers and revocation state.';
COMMENT ON TABLE auth_identity.oidc_refresh_tokens IS
    'Hashed OIDC refresh credentials with rotation/revocation state; raw refresh credentials are never persisted.';
