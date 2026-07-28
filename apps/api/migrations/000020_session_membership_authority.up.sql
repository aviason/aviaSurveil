ALTER TABLE session_references
    ADD COLUMN membership_id text,
    ADD COLUMN membership_revision bigint,
    ADD COLUMN authority_observed_at timestamptz,
    ADD COLUMN authority_state text;

ALTER TABLE session_references
    ADD CONSTRAINT session_references_membership_revision_pair
    CHECK (
        (membership_id IS NULL AND membership_revision IS NULL)
        OR
        (membership_id IS NOT NULL AND membership_revision > 0)
    ),
    ADD CONSTRAINT session_references_authority_state
    CHECK (
        authority_state IS NULL
        OR authority_state IN (
            'ACTIVE',
            'REVOCATION_PENDING',
            'DENIED_STALE_AUTHORITY'
        )
    ),
    ADD CONSTRAINT session_references_desired_membership_revision_fk
    FOREIGN KEY (membership_id, membership_revision)
    REFERENCES desired_membership_versions(membership_id, revision);

CREATE INDEX session_references_membership_active_idx
    ON session_references (
        membership_id,
        membership_revision,
        authority_observed_at,
        id
    )
    WHERE revoked_at IS NULL;
