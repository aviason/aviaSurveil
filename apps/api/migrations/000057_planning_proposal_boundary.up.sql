-- New Audit is a Planning proposal, not an executable Audit-package scope.
-- Keep its mutable draft and immutable submitted evidence separate from the
-- legacy canonical selector tables used after release.
CREATE TABLE planning_proposal_drafts (
    id text PRIMARY KEY,
    organization_id text NOT NULL REFERENCES organizations(id),
    values jsonb NOT NULL,
    submitted_planning_item_id text REFERENCES surveillance_plan_items(id),
    planning_snapshot_id text,
    planning_snapshot_digest text,
    revision bigint NOT NULL DEFAULT 1 CHECK (revision > 0),
    tombstoned_at timestamptz,
    created_by_subject_id text NOT NULL REFERENCES identity_references(subject_id),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CHECK (jsonb_typeof(values) = 'object')
);

CREATE INDEX planning_proposal_drafts_org_updated_idx
    ON planning_proposal_drafts (organization_id, updated_at DESC, id)
    WHERE tombstoned_at IS NULL;

CREATE TABLE planning_proposal_snapshots (
    id text PRIMARY KEY,
    planning_item_id text NOT NULL REFERENCES surveillance_plan_items(id),
    draft_id text NOT NULL REFERENCES planning_proposal_drafts(id),
    revision bigint NOT NULL CHECK (revision > 0),
    snapshot_digest text NOT NULL,
    snapshot jsonb NOT NULL,
    created_by_subject_id text NOT NULL REFERENCES identity_references(subject_id),
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (planning_item_id, revision),
    CHECK (jsonb_typeof(snapshot) = 'object')
);

CREATE INDEX planning_proposal_snapshots_item_idx
    ON planning_proposal_snapshots (planning_item_id, revision DESC, id);

CREATE OR REPLACE FUNCTION planning_proposal_snapshot_immutable() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION 'planning proposal snapshots are append-only';
END;
$$;

DROP TRIGGER IF EXISTS planning_proposal_snapshots_append_only ON planning_proposal_snapshots;
CREATE TRIGGER planning_proposal_snapshots_append_only
BEFORE UPDATE OR DELETE ON planning_proposal_snapshots
FOR EACH ROW EXECUTE FUNCTION planning_proposal_snapshot_immutable();

-- Post-release Audit-package preparation is linked to the immutable Planning
-- proposal snapshot. The legacy intake foreign key remains readable for
-- historical rows, while new package setup rows use the proposal pin.
ALTER TABLE canonical_audit_scope_drafts
    ALTER COLUMN planning_intake_draft_id DROP NOT NULL,
    ADD COLUMN planning_proposal_snapshot_id text REFERENCES planning_proposal_snapshots(id),
    ADD COLUMN approved_checklist_item_ceiling integer CHECK (approved_checklist_item_ceiling IS NULL OR approved_checklist_item_ceiling > 0);

ALTER TABLE canonical_audit_scope_drafts
    DROP CONSTRAINT IF EXISTS canonical_audit_scope_drafts_status_check,
    ADD CONSTRAINT canonical_audit_scope_drafts_status_check
        CHECK (status IN ('DRAFT', 'SUBMITTED', 'RELEASED', 'SELECTION_CONFIRMED', 'FINALIZED', 'ABANDONED'));

CREATE UNIQUE INDEX canonical_audit_scope_drafts_planning_snapshot_idx
    ON canonical_audit_scope_drafts (planning_proposal_snapshot_id)
    WHERE planning_proposal_snapshot_id IS NOT NULL;
