CREATE TABLE auth_identity.mail_deliveries (
    delivery_id text PRIMARY KEY CHECK (delivery_id ~ '^dly_[A-Za-z0-9_-]{22}$'),
    recipient_ciphertext bytea NOT NULL CHECK (octet_length(recipient_ciphertext) > 28),
    subject_ciphertext bytea NOT NULL CHECK (octet_length(subject_ciphertext) > 28),
    body_ciphertext bytea NOT NULL CHECK (octet_length(body_ciphertext) > 28),
    state text NOT NULL CHECK (state IN ('queued', 'leased', 'retryable', 'delivered', 'terminal-failure')),
    attempt_count integer NOT NULL DEFAULT 0 CHECK (attempt_count >= 0 AND attempt_count <= 12),
    available_at timestamptz NOT NULL,
    lease_until timestamptz,
    lease_token_hash bytea CHECK (lease_token_hash IS NULL OR octet_length(lease_token_hash) = 32),
    delivered_at timestamptz,
    last_error_class text,
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,
    CHECK (
        (state = 'leased' AND lease_until IS NOT NULL AND lease_token_hash IS NOT NULL)
        OR (state <> 'leased' AND lease_until IS NULL AND lease_token_hash IS NULL)
    ),
    CHECK ((state = 'delivered') = (delivered_at IS NOT NULL))
);

CREATE INDEX auth_identity_mail_deliveries_claim_idx
    ON auth_identity.mail_deliveries (state, available_at, created_at)
    WHERE state IN ('queued', 'retryable');

CREATE INDEX auth_identity_mail_deliveries_lease_idx
    ON auth_identity.mail_deliveries (lease_until)
    WHERE state = 'leased';

COMMENT ON TABLE auth_identity.mail_deliveries IS
    'Encrypted-at-rest mail delivery outbox. Plaintext messages, verification tokens, and SMTP credentials are never persisted.';
