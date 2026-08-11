CREATE TABLE auth_identity.mfa_factors (
    subject_id text PRIMARY KEY REFERENCES auth_identity.accounts(subject_id) ON DELETE CASCADE,
    secret_ciphertext bytea NOT NULL CHECK (octet_length(secret_ciphertext) > 28),
    enabled boolean NOT NULL DEFAULT false,
    last_used_counter bigint NOT NULL DEFAULT -1 CHECK (last_used_counter >= -1),
    recovery_failures integer NOT NULL DEFAULT 0 CHECK (recovery_failures >= 0 AND recovery_failures <= 20),
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL
);

CREATE TABLE auth_identity.mfa_recovery_codes (
    subject_id text NOT NULL REFERENCES auth_identity.mfa_factors(subject_id) ON DELETE CASCADE,
    code_hash bytea NOT NULL CHECK (octet_length(code_hash) = 32),
    created_at timestamptz NOT NULL,
    PRIMARY KEY (subject_id, code_hash)
);

COMMENT ON TABLE auth_identity.mfa_factors IS
    'Encrypted TOTP factors with durable replay and recovery-failure state; plaintext factor seeds are never persisted.';
COMMENT ON TABLE auth_identity.mfa_recovery_codes IS
    'Single-use hashed MFA recovery codes; raw recovery codes are never persisted.';
