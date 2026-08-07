-- Preparation edits are append-only receipts. The mutable assignment tables
-- remain projections for existing workflow readers, while every team and
-- question-coverage revision is retained as an immutable command snapshot.
CREATE TABLE IF NOT EXISTS canonical_audit_preparation_edit_events (
    id text PRIMARY KEY,
    assignment_id text NOT NULL REFERENCES audit_assignments(id),
    assignment_revision bigint NOT NULL CHECK (assignment_revision > 0),
    edit_kind text NOT NULL CHECK (edit_kind IN ('TEAM', 'QUESTION_COVERAGE')),
    edit_digest text NOT NULL CHECK (btrim(edit_digest) <> ''),
    snapshot jsonb NOT NULL CHECK (jsonb_typeof(snapshot) = 'object'),
    actor_subject_id text NOT NULL REFERENCES identity_references(subject_id),
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (assignment_id, assignment_revision, edit_kind)
);

CREATE OR REPLACE FUNCTION reject_canonical_preparation_edit_change()
RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF TG_OP <> 'INSERT' THEN
        RAISE EXCEPTION 'canonical preparation edit history is append-only';
    END IF;
    RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS canonical_preparation_edit_events_immutable
    ON canonical_audit_preparation_edit_events;
CREATE TRIGGER canonical_preparation_edit_events_immutable
BEFORE UPDATE OR DELETE ON canonical_audit_preparation_edit_events
FOR EACH ROW EXECUTE FUNCTION reject_canonical_preparation_edit_change();
