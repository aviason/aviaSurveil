ALTER TABLE session_references
    ADD COLUMN provider_auth_revision bigint;

ALTER TABLE session_references
    ADD CONSTRAINT session_references_provider_auth_revision_check
    CHECK (provider_auth_revision IS NULL OR provider_auth_revision > 0);
