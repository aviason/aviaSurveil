-- Governed Question Review dispositions are immutable candidate successors,
-- not an overlay that can silently diverge from technical approval/publication.
-- The existing candidate lineage tables remain authoritative; this migration
-- only permits the explicit audit command kind used to record the successor.
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

-- Pre-publication candidate review is intentionally catalog-free. Once the
-- exact candidate is published, a sealed operational catalog is projected by
-- the publication transaction. The event still pins question/candidate
-- identity; a NULL catalog means the direct candidate review boundary.
ALTER TABLE canonical_governed_question_review_events
    ALTER COLUMN catalog_id DROP NOT NULL;

-- A review event still points at the exact parent candidate it CAS-checked.
-- The successor is discoverable through template_draft_versions.supersedes_candidate_id;
-- no mutable pointer is added to the append-only event.
