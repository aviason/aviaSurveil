CREATE TABLE desired_membership_versions (
    membership_id text NOT NULL,
    subject_id text NOT NULL REFERENCES identity_references(subject_id),
    revision bigint NOT NULL CHECK (revision > 0),
    membership_state text NOT NULL CHECK (
        membership_state IN (
            'REQUESTED',
            'INVITED',
            'ACTIVE',
            'SUSPENDED',
            'DEACTIVATED',
            'REACTIVATION_PENDING'
        )
    ),
    organization_id text NOT NULL,
    roles text[] NOT NULL CHECK (cardinality(roles) > 0),
    requested_by_subject_id text NOT NULL,
    reason text NOT NULL,
    source_request_id text NOT NULL REFERENCES user_lifecycle_requests(id),
    requested_at timestamptz NOT NULL,
    effective_at timestamptz NOT NULL,
    observed_provider_enabled boolean NOT NULL,
    observed_organization_id text NOT NULL,
    observed_roles text[] NOT NULL,
    observed_at timestamptz NOT NULL,
    drift_state text NOT NULL CHECK (
        drift_state IN ('IN_SYNC', 'DRIFTED', 'PROVIDER_UNAVAILABLE', 'STALE')
    ),
    PRIMARY KEY (membership_id, revision),
    UNIQUE (subject_id, revision),
    UNIQUE (source_request_id)
);

CREATE INDEX desired_membership_versions_latest_idx
    ON desired_membership_versions (membership_id, revision DESC);

CREATE TRIGGER desired_membership_versions_immutable
BEFORE UPDATE OR DELETE ON desired_membership_versions
FOR EACH ROW EXECUTE FUNCTION reject_immutable_row_change();

CREATE TABLE desired_membership_sync (
    membership_id text PRIMARY KEY,
    subject_id text NOT NULL UNIQUE REFERENCES identity_references(subject_id),
    desired_revision bigint NOT NULL,
    observed_provider_enabled boolean NOT NULL,
    observed_organization_id text NOT NULL,
    observed_roles text[] NOT NULL,
    observed_at timestamptz NOT NULL,
    drift_state text NOT NULL CHECK (
        drift_state IN ('IN_SYNC', 'DRIFTED', 'PROVIDER_UNAVAILABLE', 'STALE')
    ),
    FOREIGN KEY (membership_id, desired_revision)
        REFERENCES desired_membership_versions(membership_id, revision)
);
