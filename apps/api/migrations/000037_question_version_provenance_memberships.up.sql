-- A question_versions row is immutable and may be referenced by more than one
-- sealed catalog within the same usage class. Provenance is therefore a
-- usage-class guard plus catalog membership, not a singleton ownership row.
ALTER TABLE canonical_question_version_provenance
    DROP CONSTRAINT IF EXISTS canonical_question_version_provenance_pkey;
ALTER TABLE canonical_question_version_provenance
    ADD PRIMARY KEY (question_version_id, usage_class, catalog_id);

CREATE OR REPLACE FUNCTION reject_catalog_membership_provenance_mismatch() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    PERFORM pg_advisory_xact_lock(hashtextextended('qv-provenance:' || NEW.question_version_id, 0));
    IF EXISTS (
        SELECT 1
        FROM canonical_question_version_provenance provenance
        WHERE provenance.question_version_id = NEW.question_version_id
          AND provenance.usage_class <> NEW.usage_class
    ) THEN
        RAISE EXCEPTION 'catalog membership usage class conflicts with immutable question provenance';
    END IF;
    IF NOT EXISTS (
        SELECT 1
        FROM canonical_question_version_provenance provenance
        WHERE provenance.question_version_id = NEW.question_version_id
          AND provenance.usage_class = NEW.usage_class
          AND provenance.catalog_id = NEW.catalog_id
    ) THEN
        RAISE EXCEPTION 'catalog membership must bind the exact immutable question provenance';
    END IF;
    RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS canonical_catalog_membership_provenance_guard
    ON canonical_question_catalog_memberships;
CREATE TRIGGER canonical_catalog_membership_provenance_guard
BEFORE INSERT ON canonical_question_catalog_memberships
FOR EACH ROW EXECUTE FUNCTION reject_catalog_membership_provenance_mismatch();
