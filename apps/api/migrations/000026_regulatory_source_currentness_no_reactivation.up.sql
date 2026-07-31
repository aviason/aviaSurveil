-- A controlled source restoration must enter as a new immutable source-version
-- identity. Re-activating an old current snapshot after a newer source would
-- collide with the old generation-run/candidate identity and could otherwise
-- imply that immutable historical output had become current again.

CREATE UNIQUE INDEX regulatory_source_currentness_events_source_current_unique
    ON regulatory_source_currentness_events (
        source_identity,
        current_source_version_id,
        current_source_hash
    );
