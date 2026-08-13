ALTER TABLE auth_identity.oidc_auth_requests
    ADD COLUMN IF NOT EXISTS expires_at timestamptz;

UPDATE auth_identity.oidc_auth_requests
SET expires_at = created_at + interval '10 minutes'
WHERE expires_at IS NULL;

ALTER TABLE auth_identity.oidc_auth_requests
    ALTER COLUMN expires_at SET NOT NULL,
    ALTER COLUMN expires_at SET DEFAULT (now() + interval '10 minutes'),
    ADD COLUMN IF NOT EXISTS amr text[] NOT NULL DEFAULT ARRAY['pwd']::text[];

ALTER TABLE auth_identity.oidc_authorization_codes
    ADD COLUMN IF NOT EXISTS expires_at timestamptz;

UPDATE auth_identity.oidc_authorization_codes
SET expires_at = created_at + interval '60 seconds'
WHERE expires_at IS NULL;

ALTER TABLE auth_identity.oidc_authorization_codes
    ALTER COLUMN expires_at SET NOT NULL,
    ALTER COLUMN expires_at SET DEFAULT (now() + interval '60 seconds');

CREATE INDEX IF NOT EXISTS auth_identity_oidc_auth_requests_expiry_idx
    ON auth_identity.oidc_auth_requests (expires_at)
    WHERE done = false;

CREATE INDEX IF NOT EXISTS auth_identity_oidc_authorization_codes_expiry_idx
    ON auth_identity.oidc_authorization_codes (expires_at);

CREATE TABLE auth_identity.provider_profiles (
    subject_id text PRIMARY KEY REFERENCES auth_identity.accounts(subject_id),
    display_name text NOT NULL CHECK (btrim(display_name) <> '' AND length(display_name) <= 200),
    given_name text NOT NULL DEFAULT '' CHECK (length(given_name) <= 100),
    family_name text NOT NULL DEFAULT '' CHECK (length(family_name) <= 100),
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL
);

CREATE TABLE auth_identity.application_authorities (
    subject_id text PRIMARY KEY REFERENCES auth_identity.accounts(subject_id),
    membership_id text NOT NULL CHECK (btrim(membership_id) <> '' AND length(membership_id) <= 160),
    organization_id text NOT NULL CHECK (btrim(organization_id) <> '' AND length(organization_id) <= 160),
    role text NOT NULL CHECK (btrim(role) <> '' AND length(role) <= 80),
    state text NOT NULL CHECK (state IN ('INVITED', 'ACTIVE', 'DISABLED', 'SUSPENDED', 'DEACTIVATED')),
    membership_revision bigint NOT NULL CHECK (membership_revision > 0),
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,
    UNIQUE (membership_id)
);

CREATE INDEX auth_identity_application_authorities_directory_idx
    ON auth_identity.application_authorities (organization_id, state, membership_revision, subject_id);

CREATE TABLE auth_identity.provider_admin_operation_receipts (
    operation_id text PRIMARY KEY CHECK (btrim(operation_id) <> '' AND length(operation_id) <= 160),
    request_hash bytea NOT NULL CHECK (octet_length(request_hash) = 32),
    operation_kind text NOT NULL CHECK (btrim(operation_kind) <> '' AND length(operation_kind) <= 100),
    state text NOT NULL CHECK (state IN ('PROCESSING', 'SUCCEEDED', 'FAILED')),
    response_status integer NOT NULL CHECK (response_status BETWEEN 200 AND 599),
    response_body jsonb NOT NULL CHECK (jsonb_typeof(response_body) = 'object'),
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL
);

CREATE INDEX auth_identity_provider_admin_operation_receipts_updated_idx
    ON auth_identity.provider_admin_operation_receipts (updated_at);

COMMENT ON TABLE auth_identity.provider_profiles IS
    'First-party provider profile projection; profile values are generated and owned by this provider.';
COMMENT ON TABLE auth_identity.application_authorities IS
    'Current provider-owned mirror of the application authority contract. membership_revision is independent from account auth_revision.';
COMMENT ON TABLE auth_identity.provider_admin_operation_receipts IS
    'Idempotent first-party provider-admin operation receipts keyed by operation ID and canonical request hash.';
