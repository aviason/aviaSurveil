-- Canonical preparation is created after Planning release and before an
-- inspection exists.  Keep the existing assignment aggregate as the durable
-- post-release preparation owner, but make its inspection/lead references
-- attachable only at Lead assignment/materialization time.
ALTER TABLE audit_assignments
    ALTER COLUMN inspection_id DROP NOT NULL,
    ALTER COLUMN lead_subject_id DROP NOT NULL;

ALTER TABLE audit_assignments
    ADD COLUMN planning_item_id text REFERENCES surveillance_plan_items(id);

CREATE UNIQUE INDEX audit_assignments_canonical_planning_item_idx
    ON audit_assignments (planning_item_id)
    WHERE planning_item_id IS NOT NULL;

ALTER TABLE audit_assignments
    ADD CONSTRAINT audit_assignments_canonical_preparation_shape
    CHECK (
        (status = 'PREPARATION' AND inspection_id IS NULL AND lead_subject_id IS NULL)
        OR
        (status IN ('LEAD_ASSIGNED', 'TEAM_ASSIGNED', 'QUESTIONS_ASSIGNED')
         AND inspection_id IS NULL AND lead_subject_id IS NOT NULL)
        OR
        (status NOT IN ('PREPARATION', 'LEAD_ASSIGNED', 'TEAM_ASSIGNED', 'QUESTIONS_ASSIGNED')
         AND inspection_id IS NOT NULL AND lead_subject_id IS NOT NULL)
    ) NOT VALID;

DO $migration$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM audit_assignments
        WHERE (status = 'PREPARATION' AND (inspection_id IS NOT NULL OR lead_subject_id IS NOT NULL))
           OR (status IN ('LEAD_ASSIGNED', 'TEAM_ASSIGNED', 'QUESTIONS_ASSIGNED')
               AND (inspection_id IS NOT NULL OR lead_subject_id IS NULL))
           OR (status NOT IN ('PREPARATION', 'LEAD_ASSIGNED', 'TEAM_ASSIGNED', 'QUESTIONS_ASSIGNED')
               AND (inspection_id IS NULL OR lead_subject_id IS NULL))
    ) THEN
        ALTER TABLE audit_assignments
            VALIDATE CONSTRAINT audit_assignments_canonical_preparation_shape;
    END IF;
END
$migration$;
