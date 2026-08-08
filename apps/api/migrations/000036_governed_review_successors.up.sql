-- Governed Question Review dispositions are immutable candidate successors,
-- not an overlay that can silently diverge from technical approval/publication.
-- The existing candidate lineage tables remain authoritative; this migration
-- only permits the explicit audit command kind used to record the successor.
-- A genuine version-21 database may have recorded migration 21 before the
-- Task-5 Admin boundary was introduced.  In that case the normal forward
-- repair runs only after all numbered migrations.  Keep this numbered
-- migration a no-op for the missing table; the final idempotent repair creates
-- the complete ledger without making the N-1 fixture pretend it had it.
DO $command_boundary$
BEGIN
    IF to_regclass('public.governed_candidate_commands') IS NOT NULL THEN
        ALTER TABLE governed_candidate_commands
            DROP CONSTRAINT IF EXISTS governed_candidate_commands_command_kind_check;
        ALTER TABLE governed_candidate_commands
            ADD CONSTRAINT governed_candidate_commands_command_kind_check
            CHECK (command_kind IN (
                'IMPORTED_GENERATION_RUN',
                'FAILED_IMPORT',
                'REVISION_CREATED',
                'DEPARTMENT_REVIEW_SUBMITTED',
                'QUESTION_REVIEW_SUCCESSOR'
            ));
    END IF;
END
$command_boundary$;

-- Pre-publication candidate review is intentionally catalog-free. Once the
-- exact candidate is published, a sealed operational catalog is projected by
-- the publication transaction. The event still pins question/candidate
-- identity; a NULL catalog means the direct candidate review boundary.
DO $review_event_boundary$
BEGIN
    IF to_regclass('public.canonical_governed_question_review_events') IS NOT NULL THEN
        ALTER TABLE canonical_governed_question_review_events
            ALTER COLUMN catalog_id DROP NOT NULL;
    END IF;
END
$review_event_boundary$;

-- A review event still points at the exact parent candidate it CAS-checked.
-- The successor is discoverable through template_draft_versions.supersedes_candidate_id;
-- no mutable pointer is added to the append-only event.
