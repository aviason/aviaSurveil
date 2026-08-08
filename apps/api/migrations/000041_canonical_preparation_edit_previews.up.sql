-- Team and per-question coverage changes are submitted as bounded, server
-- issued previews.  The preview is immutable and its separate consumption
-- receipt is append-only, so retries cannot silently apply a different set.
CREATE TABLE canonical_audit_preparation_edit_previews (
    id text PRIMARY KEY,
    assignment_id text NOT NULL REFERENCES audit_assignments(id),
    assignment_revision bigint NOT NULL CHECK (assignment_revision > 0),
    edit_kind text NOT NULL CHECK (edit_kind IN ('TEAM', 'QUESTION_COVERAGE')),
    operation_id text NOT NULL,
    idempotency_key text NOT NULL,
    edit_digest text NOT NULL CHECK (btrim(edit_digest) <> ''),
    snapshot jsonb NOT NULL CHECK (jsonb_typeof(snapshot) = 'object'),
    actor_subject_id text NOT NULL REFERENCES identity_references(subject_id),
    expires_at timestamptz NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (assignment_id, edit_kind, operation_id),
    UNIQUE (assignment_id, edit_kind, idempotency_key)
);

CREATE TABLE canonical_audit_preparation_edit_preview_consumptions (
    preview_id text PRIMARY KEY REFERENCES canonical_audit_preparation_edit_previews(id),
    consumed_by_subject_id text NOT NULL REFERENCES identity_references(subject_id),
    consumed_at timestamptz NOT NULL DEFAULT now()
);

CREATE OR REPLACE FUNCTION reject_canonical_preparation_preview_change()
RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF TG_OP <> 'INSERT' THEN
        RAISE EXCEPTION 'canonical preparation previews and consumptions are append-only';
    END IF;
    RETURN NEW;
END;
$$;

CREATE TRIGGER canonical_preparation_edit_previews_append_only
BEFORE UPDATE OR DELETE ON canonical_audit_preparation_edit_previews
FOR EACH ROW EXECUTE FUNCTION reject_canonical_preparation_preview_change();

CREATE TRIGGER canonical_preparation_edit_preview_consumptions_append_only
BEFORE UPDATE OR DELETE ON canonical_audit_preparation_edit_preview_consumptions
FOR EACH ROW EXECUTE FUNCTION reject_canonical_preparation_preview_change();

CREATE INDEX canonical_audit_preparation_edit_previews_expiry_idx
    ON canonical_audit_preparation_edit_previews (assignment_id, expires_at);
