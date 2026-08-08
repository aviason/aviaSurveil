-- Exercise review state is scoped to the disposable Audit scope that granted
-- access.  The catalog remains immutable and reusable, but review decisions
-- must never be visible or replayable across two provider/target scopes.
ALTER TABLE canonical_exercise_question_review_drafts
    ADD COLUMN scope_draft_id text REFERENCES canonical_audit_scope_drafts(id);
ALTER TABLE canonical_exercise_question_review_events
    ADD COLUMN scope_draft_id text REFERENCES canonical_audit_scope_drafts(id);

-- Historical exercise review rows have no authoritative scope.  Guessing a
-- provider/target after the fact would leak a decision across scopes, so the
-- upgrade fails closed with an actionable message instead of mutating the
-- append-only history.  A disposable preprod database may be repaired by
-- replaying the review commands with an explicit scope before rerunning.
DO $exercise_scope_history_guard$
BEGIN
    IF EXISTS (SELECT 1 FROM canonical_exercise_question_review_drafts WHERE scope_draft_id IS NULL)
       OR EXISTS (SELECT 1 FROM canonical_exercise_question_review_events WHERE scope_draft_id IS NULL) THEN
        RAISE EXCEPTION 'migration 39 cannot infer canonical exercise review scope from legacy unscoped append-only history; export and replay each decision with an explicit scope before retrying';
    END IF;
END
$exercise_scope_history_guard$;

DO $drop_legacy_exercise_review_key$
DECLARE constraint_name text;
BEGIN
    FOR constraint_name IN
        SELECT pg_constraint.conname
        FROM pg_constraint
        WHERE pg_constraint.conrelid = 'canonical_exercise_question_review_drafts'::regclass
          AND pg_constraint.contype = 'u'
          AND pg_get_constraintdef(pg_constraint.oid) = 'UNIQUE (catalog_id, question_version_id, revision)'
    LOOP
        EXECUTE format('ALTER TABLE canonical_exercise_question_review_drafts DROP CONSTRAINT %I', constraint_name);
    END LOOP;
END
$drop_legacy_exercise_review_key$;
ALTER TABLE canonical_exercise_question_review_drafts
    ADD CONSTRAINT canonical_exercise_question_review_drafts_scope_revision_key
    UNIQUE (scope_draft_id, catalog_id, question_version_id, revision);

ALTER TABLE canonical_exercise_question_review_drafts
    ADD CONSTRAINT canonical_exercise_question_review_drafts_scope_required
    CHECK (scope_draft_id IS NOT NULL) NOT VALID;
ALTER TABLE canonical_exercise_question_review_events
    ADD CONSTRAINT canonical_exercise_question_review_events_scope_required
    CHECK (scope_draft_id IS NOT NULL) NOT VALID;

ALTER TABLE canonical_exercise_question_review_drafts
    VALIDATE CONSTRAINT canonical_exercise_question_review_drafts_scope_required;
ALTER TABLE canonical_exercise_question_review_events
    VALIDATE CONSTRAINT canonical_exercise_question_review_events_scope_required;

CREATE INDEX canonical_exercise_question_review_drafts_scope_question_idx
    ON canonical_exercise_question_review_drafts (scope_draft_id, catalog_id, question_version_id, revision DESC);
CREATE INDEX canonical_exercise_question_review_events_scope_question_idx
    ON canonical_exercise_question_review_events (scope_draft_id, occurred_at DESC, event_id DESC);
