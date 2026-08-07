-- Coverage is many-to-many.  PostgreSQL truncates the generated name for the
-- legacy UNIQUE(preparation_id, position) constraint, so remove both the
-- readable and truncated forms when upgrading databases that already ran
-- migration 30.
ALTER TABLE canonical_audit_preparation_questions
    DROP CONSTRAINT IF EXISTS canonical_audit_preparation_questions_preparation_id_position_key,
    DROP CONSTRAINT IF EXISTS canonical_audit_preparation_questio_preparation_id_position_key;
