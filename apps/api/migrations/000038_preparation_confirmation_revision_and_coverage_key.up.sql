-- Bind the immutable Department Manager confirmation to the assignment
-- revision it confirmed.  Materialization must reject later mutable
-- projections instead of silently using an older receipt.
ALTER TABLE canonical_audit_preparation_snapshots
    ADD COLUMN confirmed_assignment_revision bigint;

-- Migration 30 introduced assignment ownership after migration 29 had already
-- made preparation receipts immutable. Reconcile any pre-38 confirmed receipt
-- by inserting an immutable successor bound to the exact current assignment
-- revision; never UPDATE the legacy receipt (the append-only trigger forbids
-- that and the old row remains a truthful historical record).
CREATE TEMP TABLE canonical_preparation_confirmation_bindings (
    legacy_id text PRIMARY KEY,
    successor_id text NOT NULL UNIQUE,
    assignment_id text NOT NULL,
    released_scope_snapshot_id text NOT NULL,
    lead_subject_id text NOT NULL,
    successor_revision bigint NOT NULL,
    assignment_revision bigint NOT NULL
) ON COMMIT DROP;

INSERT INTO canonical_preparation_confirmation_bindings (
    legacy_id, successor_id, assignment_id, released_scope_snapshot_id,
    lead_subject_id, successor_revision, assignment_revision
)
SELECT legacy.id,
       'preparation:legacy-binding:' || legacy.id,
       assignment.id,
       legacy.released_scope_snapshot_id,
       legacy.lead_subject_id,
       scope_revisions.max_revision + row_number() OVER (
           PARTITION BY legacy.released_scope_snapshot_id
           ORDER BY legacy.revision, legacy.id
       ),
       assignment.revision
FROM canonical_audit_preparation_snapshots legacy
JOIN LATERAL (
    SELECT assignment.*
    FROM audit_assignments assignment
    WHERE (legacy.assignment_id IS NOT NULL AND assignment.id = legacy.assignment_id)
       OR (legacy.assignment_id IS NULL AND assignment.released_scope_snapshot_id = legacy.released_scope_snapshot_id)
    ORDER BY (assignment.id = legacy.assignment_id) DESC, assignment.updated_at DESC, assignment.id
    LIMIT 1
) assignment ON TRUE
JOIN LATERAL (
    SELECT COALESCE(MAX(existing.revision), 0) AS max_revision
    FROM canonical_audit_preparation_snapshots existing
    WHERE existing.released_scope_snapshot_id = legacy.released_scope_snapshot_id
) scope_revisions ON TRUE
WHERE legacy.status = 'CONFIRMED'
  AND legacy.confirmed_assignment_revision IS NULL;

INSERT INTO canonical_audit_preparation_snapshots (
    id, assignment_id, released_scope_snapshot_id, lead_subject_id, revision,
    status, preparation_digest, confirmed_by_subject_id, confirmed_at,
    confirmed_assignment_revision, snapshot, created_at
)
SELECT binding.successor_id,
       binding.assignment_id,
       binding.released_scope_snapshot_id,
       binding.lead_subject_id,
       binding.successor_revision,
       legacy.status,
       legacy.preparation_digest,
       legacy.confirmed_by_subject_id,
       legacy.confirmed_at,
       binding.assignment_revision,
       legacy.snapshot,
       now()
FROM canonical_preparation_confirmation_bindings binding
JOIN canonical_audit_preparation_snapshots legacy ON legacy.id = binding.legacy_id;

INSERT INTO canonical_audit_preparation_questions (
    preparation_id, released_scope_snapshot_id, question_version_id,
    subject_id, position, created_at
)
SELECT binding.successor_id,
       questions.released_scope_snapshot_id,
       questions.question_version_id,
       questions.subject_id,
       questions.position,
       questions.created_at
FROM canonical_preparation_confirmation_bindings binding
JOIN canonical_audit_preparation_questions questions ON questions.preparation_id = binding.legacy_id;

DO $migration$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM canonical_audit_preparation_snapshots legacy
        WHERE legacy.status = 'CONFIRMED'
          AND legacy.confirmed_assignment_revision IS NULL
          AND NOT EXISTS (
              SELECT 1
              FROM canonical_audit_preparation_snapshots successor
              WHERE successor.id = 'preparation:legacy-binding:' || legacy.id
                AND successor.confirmed_assignment_revision IS NOT NULL
          )
    ) THEN
        RAISE EXCEPTION 'cannot reconcile confirmed preparation without an exact audit assignment: migration 38 is fail-closed';
    END IF;
END
$migration$;

ALTER TABLE canonical_audit_preparation_snapshots
    ADD CONSTRAINT canonical_audit_preparation_confirmed_revision_shape
    CHECK (
        (status = 'CONFIRMED' AND confirmed_assignment_revision IS NOT NULL AND confirmed_assignment_revision > 0)
        OR status <> 'CONFIRMED'
    ) NOT VALID;

-- Coverage is per question and per assigned Inspector. The former primary
-- key omitted subject_id, so a valid question covered by two Inspectors could
-- never be copied into the immutable confirmation receipt. A coverage row
-- must also carry the exact question/position pair from its released scope:
-- the existing independent snapshot/question FK did not prevent question A
-- from being written with question B's position.
ALTER TABLE canonical_audit_scope_snapshot_questions
    ADD CONSTRAINT canonical_scope_snapshot_question_position_key
    UNIQUE (snapshot_id, question_version_id, position);

ALTER TABLE canonical_audit_preparation_questions
    DROP CONSTRAINT IF EXISTS canonical_audit_preparation_questions_pkey;
ALTER TABLE canonical_audit_preparation_questions
    ADD PRIMARY KEY (preparation_id, question_version_id, subject_id);
ALTER TABLE canonical_audit_preparation_questions
    DROP CONSTRAINT IF EXISTS canonical_audit_preparation_questions_preparation_id_position_key;
CREATE UNIQUE INDEX canonical_audit_preparation_questions_position_subject_key
    ON canonical_audit_preparation_questions (preparation_id, position, subject_id);
ALTER TABLE canonical_audit_preparation_questions
    ADD CONSTRAINT canonical_preparation_question_scope_position_fkey
    FOREIGN KEY (released_scope_snapshot_id, question_version_id, position)
    REFERENCES canonical_audit_scope_snapshot_questions (
        snapshot_id, question_version_id, position
    );
