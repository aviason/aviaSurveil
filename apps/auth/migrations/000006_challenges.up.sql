CREATE TABLE auth_identity.identity_challenges (
    token_hash bytea PRIMARY KEY CHECK (octet_length(token_hash) = 32),
    subject_id text NOT NULL REFERENCES auth_identity.accounts(subject_id) ON DELETE CASCADE,
    purpose text NOT NULL CHECK (purpose IN (
        'email-verification', 'password-reset', 'mfa-recovery', 'admin-recovery'
    )),
    state text NOT NULL CHECK (state IN ('active', 'consumed', 'invalidated', 'locked')),
    attempt_count integer NOT NULL DEFAULT 0 CHECK (attempt_count >= 0 AND attempt_count <= 10),
    max_attempts integer NOT NULL CHECK (max_attempts >= 1 AND max_attempts <= 10),
    expires_at timestamptz NOT NULL,
    created_at timestamptz NOT NULL,
    consumed_at timestamptz,
    invalidated_at timestamptz,
    updated_at timestamptz NOT NULL,
    CHECK (expires_at > created_at),
    CHECK ((state = 'consumed') = (consumed_at IS NOT NULL)),
    CHECK ((state = 'invalidated') = (invalidated_at IS NOT NULL))
);

CREATE INDEX auth_identity_identity_challenges_subject_purpose_idx
    ON auth_identity.identity_challenges (subject_id, purpose, state, expires_at);

CREATE INDEX auth_identity_identity_challenges_cleanup_idx
    ON auth_identity.identity_challenges (expires_at, state);

COMMENT ON TABLE auth_identity.identity_challenges IS
    'Hashed, subject- and purpose-bound identity challenges. Consume and rejection paths lock rows so attempts and use remain atomic.';
