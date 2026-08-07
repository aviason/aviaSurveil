-- Canonical audit packages are materialized from the released scope snapshot,
-- not from a pre-approval checklist-template field. Legacy packages retain
-- their immutable template reference; successor packages pin the canonical
-- snapshot explicitly and leave that legacy reference NULL.
ALTER TABLE inspection_packages
    ALTER COLUMN checklist_template_version_id DROP NOT NULL;

ALTER TABLE inspection_packages
    ADD COLUMN canonical_scope_snapshot_id text
        REFERENCES canonical_audit_scope_snapshots(id);

ALTER TABLE inspection_packages
    ADD CONSTRAINT inspection_packages_source_identity_check
    CHECK ((checklist_template_version_id IS NOT NULL) <> (canonical_scope_snapshot_id IS NOT NULL));

-- A canonical package is bound to exactly one source authority. The legacy
-- template identity and the released canonical scope are deliberately
-- mutually exclusive; accepting both would let a caller smuggle a mutable
-- checklist-template fallback alongside the released question subset.
ALTER TABLE audit_assignments
    ADD COLUMN released_scope_snapshot_id text
        REFERENCES canonical_audit_scope_snapshots(id);

-- The selection digest identifies the exact question set. This companion
-- digest covers the complete immutable Planning snapshot (setup, resource,
-- notice, scope, catalog, and selection facts) carried through every later
-- decision and release.
ALTER TABLE canonical_audit_scope_snapshots
    ADD COLUMN planning_snapshot_digest text NOT NULL DEFAULT '';

-- N-1 databases already contain immutable scope snapshots created by
-- migration 29.  The one-time digest repair is performed with the append-only
-- trigger suspended by the table owner, then the default is removed and a
-- format check is installed.  Runtime code can never update this column.
ALTER TABLE canonical_audit_scope_snapshots
    DISABLE TRIGGER canonical_audit_scope_snapshots_append_only;
UPDATE canonical_audit_scope_snapshots
SET planning_snapshot_digest = governed_jsonb_sha256(snapshot)
WHERE btrim(planning_snapshot_digest) = '';
ALTER TABLE canonical_audit_scope_snapshots
    ENABLE TRIGGER canonical_audit_scope_snapshots_append_only;
ALTER TABLE canonical_audit_scope_snapshots
    ALTER COLUMN planning_snapshot_digest DROP DEFAULT;
ALTER TABLE canonical_audit_scope_snapshots
    ADD CONSTRAINT canonical_audit_scope_snapshot_planning_digest
    CHECK (governed_sha256(planning_snapshot_digest));

-- Preparation is an assignment-owned receipt, not a free-floating
-- confirmation for the latest scope revision. Canonical materialization must
-- consume the exact assignment and released snapshot that the Lead/team
-- prepared.
ALTER TABLE canonical_audit_preparation_snapshots
    ADD COLUMN assignment_id text REFERENCES audit_assignments(id);

-- Existing preparation receipts are append-only and may predate the
-- assignment column, so they remain readable with a NULL legacy value. The
-- NOT VALID check enforces assignment identity on every new row without
-- attempting an UPDATE that the migration 29 append-only trigger must reject.
ALTER TABLE canonical_audit_preparation_snapshots
    ADD CONSTRAINT canonical_audit_preparation_assignment_required
    CHECK (assignment_id IS NOT NULL) NOT VALID;

DO $migration$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM canonical_audit_preparation_snapshots
        WHERE assignment_id IS NULL
    ) THEN
        ALTER TABLE canonical_audit_preparation_snapshots
            VALIDATE CONSTRAINT canonical_audit_preparation_assignment_required;
    END IF;
END
$migration$;
CREATE UNIQUE INDEX canonical_audit_preparation_assignment_revision_idx
    ON canonical_audit_preparation_snapshots (assignment_id, revision);

-- A governed catalog is bound to one exact published candidate decision. The
-- exercise catalog leaves these fields NULL and is structurally incapable of
-- acquiring an operational publication identity.
ALTER TABLE canonical_question_catalogs
    ADD COLUMN governed_publication_decision_id text,
    ADD COLUMN governed_candidate_draft_version_id text,
    ADD COLUMN governed_candidate_revision bigint,
    ADD COLUMN governed_candidate_content_digest text;
ALTER TABLE canonical_question_catalogs
    ADD CONSTRAINT canonical_question_catalog_publication_shape
    CHECK (
        (usage_class = 'PREPROD_EXERCISE'
         AND governed_publication_decision_id IS NULL
         AND governed_candidate_draft_version_id IS NULL
         AND governed_candidate_revision IS NULL
         AND governed_candidate_content_digest IS NULL)
        OR
        (usage_class = 'GOVERNED_OPERATIONAL'
         AND governed_publication_decision_id IS NOT NULL
         AND governed_candidate_draft_version_id IS NOT NULL
         AND governed_candidate_revision IS NOT NULL
         AND governed_candidate_content_digest IS NOT NULL)
    );
ALTER TABLE canonical_question_catalogs
    ADD CONSTRAINT canonical_question_catalog_publication_fk
    FOREIGN KEY (governed_publication_decision_id)
    REFERENCES checklist_publication_decisions(id);
ALTER TABLE canonical_question_catalogs
    ADD CONSTRAINT canonical_question_catalog_candidate_fk
    FOREIGN KEY (governed_candidate_draft_version_id, governed_candidate_revision, governed_candidate_content_digest)
    REFERENCES template_draft_versions(id, revision, candidate_content_digest);

CREATE OR REPLACE FUNCTION validate_canonical_governed_catalog_membership() RETURNS trigger
LANGUAGE plpgsql AS $$
DECLARE catalog record;
DECLARE published_count integer;
BEGIN
    SELECT * INTO catalog FROM canonical_question_catalogs WHERE id = NEW.catalog_id;
    IF catalog.usage_class = 'GOVERNED_OPERATIONAL' THEN
        IF catalog.governed_publication_decision_id IS NULL
           OR catalog.governed_candidate_draft_version_id IS NULL
           OR catalog.governed_candidate_revision IS NULL
           OR catalog.governed_candidate_content_digest IS NULL THEN
            RAISE EXCEPTION 'governed catalog membership requires one exact publication decision and candidate digest';
        END IF;
        SELECT count(*) INTO published_count
        FROM template_version_questions link
        JOIN checklist_template_versions template
          ON template.id = link.template_version_id
        WHERE link.question_version_id = NEW.question_version_id
          AND template.candidate_draft_version_id = catalog.governed_candidate_draft_version_id
          AND template.candidate_revision = catalog.governed_candidate_revision
          AND template.candidate_content_digest = catalog.governed_candidate_content_digest
          AND template.publication_decision_id = catalog.governed_publication_decision_id
          AND template.published_at IS NOT NULL;
        IF published_count <> 1 THEN
            RAISE EXCEPTION 'governed catalog question must be present in the exact published candidate template';
        END IF;
    END IF;
    RETURN NEW;
END;
$$;

CREATE TRIGGER canonical_governed_catalog_membership_guard
BEFORE INSERT ON canonical_question_catalog_memberships
FOR EACH ROW EXECUTE FUNCTION validate_canonical_governed_catalog_membership();

-- Question versions remain the sole immutable body/version authority, while
-- this append-only provenance row makes the exercise/governed usage class
-- structurally exclusive. A PREPROD_EXERCISE question version can therefore
-- never be reused by the governed candidate/publication tables.
CREATE TABLE canonical_question_version_provenance (
    question_version_id text PRIMARY KEY REFERENCES question_versions(id),
    usage_class text NOT NULL CHECK (usage_class IN ('GOVERNED_OPERATIONAL', 'PREPROD_EXERCISE')),
    catalog_id text NOT NULL REFERENCES canonical_question_catalogs(id),
    recorded_at timestamptz NOT NULL DEFAULT now()
);

-- Migration 29 may already have sealed memberships. Backfill their immutable
-- usage provenance before installing the new publication guards; otherwise
-- legacy exercise rows would remain invisible to the no-promotion triggers.
DO $migration$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM canonical_question_catalog_memberships membership
        JOIN canonical_question_catalogs catalog ON catalog.id = membership.catalog_id
        WHERE membership.usage_class <> catalog.usage_class
    ) THEN
        RAISE EXCEPTION 'existing canonical membership usage does not match its catalog';
    END IF;
    IF EXISTS (
        SELECT membership.question_version_id
        FROM canonical_question_catalog_memberships membership
        GROUP BY membership.question_version_id
        HAVING COUNT(DISTINCT membership.usage_class) > 1
            OR COUNT(DISTINCT membership.catalog_id) > 1
    ) THEN
        RAISE EXCEPTION 'existing question version has conflicting canonical provenance';
    END IF;
END
$migration$;

INSERT INTO canonical_question_version_provenance
    (question_version_id, usage_class, catalog_id, recorded_at)
SELECT membership.question_version_id, membership.usage_class, membership.catalog_id, now()
FROM canonical_question_catalog_memberships membership
ON CONFLICT (question_version_id) DO NOTHING;

-- A legacy database may already contain a candidate or governed template that
-- references a question now classified as PREPROD_EXERCISE. Do not install a
-- partial guard and silently preserve that promotion; fail the upgrade closed
-- so the operator can reconcile the conflicting immutable lineage first.
DO $migration$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM template_draft_versions draft
        JOIN canonical_question_version_provenance provenance
          ON provenance.question_version_id = ANY(draft.question_version_ids)
        WHERE provenance.usage_class = 'PREPROD_EXERCISE'
    ) THEN
        RAISE EXCEPTION 'existing governed candidate contains PREPROD_EXERCISE question versions';
    END IF;
    IF EXISTS (
        SELECT 1
        FROM checklist_template_versions template
        JOIN template_version_questions template_question
          ON template_question.template_version_id = template.id
        JOIN canonical_question_version_provenance provenance
          ON provenance.question_version_id = template_question.question_version_id
        WHERE provenance.usage_class = 'PREPROD_EXERCISE'
          AND template.candidate_draft_version_id IS NOT NULL
    ) THEN
        RAISE EXCEPTION 'existing governed published template contains PREPROD_EXERCISE question versions';
    END IF;
END
$migration$;

CREATE OR REPLACE FUNCTION reject_catalog_membership_provenance_mismatch() RETURNS trigger
LANGUAGE plpgsql AS $$
DECLARE recorded_usage text;
DECLARE recorded_catalog text;
BEGIN
    SELECT usage_class, catalog_id INTO recorded_usage, recorded_catalog
    FROM canonical_question_version_provenance
    WHERE question_version_id = NEW.question_version_id;
    IF recorded_usage IS NULL OR recorded_usage <> NEW.usage_class OR recorded_catalog <> NEW.catalog_id THEN
        RAISE EXCEPTION 'catalog membership must bind the exact immutable question provenance';
    END IF;
    RETURN NEW;
END;
$$;

CREATE TRIGGER canonical_catalog_membership_provenance_guard
BEFORE INSERT ON canonical_question_catalog_memberships
FOR EACH ROW EXECUTE FUNCTION reject_catalog_membership_provenance_mismatch();

CREATE OR REPLACE FUNCTION reject_exercise_question_version_reuse() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM canonical_question_version_provenance provenance
        WHERE provenance.question_version_id = ANY(NEW.question_version_ids)
          AND provenance.usage_class = 'PREPROD_EXERCISE'
    ) THEN
        RAISE EXCEPTION 'PREPROD_EXERCISE question versions cannot enter a governed candidate';
    END IF;
    RETURN NEW;
END;
$$;

CREATE TRIGGER template_draft_versions_exercise_provenance_guard
BEFORE INSERT OR UPDATE OF question_version_ids ON template_draft_versions
FOR EACH ROW EXECUTE FUNCTION reject_exercise_question_version_reuse();

CREATE OR REPLACE FUNCTION reject_exercise_published_question_version() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM template_draft_versions draft
        JOIN canonical_question_version_provenance provenance
          ON provenance.question_version_id = ANY(draft.question_version_ids)
        WHERE draft.id = NEW.candidate_draft_version_id
          AND provenance.usage_class = 'PREPROD_EXERCISE'
    ) THEN
        RAISE EXCEPTION 'PREPROD_EXERCISE question versions cannot be published';
    END IF;
    RETURN NEW;
END;
$$;

CREATE TRIGGER checklist_publication_decisions_exercise_provenance_guard
BEFORE INSERT ON checklist_publication_decisions
FOR EACH ROW EXECUTE FUNCTION reject_exercise_published_question_version();

CREATE OR REPLACE FUNCTION reject_exercise_template_question_link() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM checklist_template_versions template
        JOIN canonical_question_version_provenance provenance
          ON provenance.question_version_id = NEW.question_version_id
        WHERE template.id = NEW.template_version_id
          AND template.candidate_draft_version_id IS NOT NULL
          AND provenance.usage_class = 'PREPROD_EXERCISE'
    ) THEN
        RAISE EXCEPTION 'PREPROD_EXERCISE question versions cannot enter a governed published template';
    END IF;
    RETURN NEW;
END;
$$;

CREATE TRIGGER template_version_questions_exercise_provenance_guard
BEFORE INSERT ON template_version_questions
FOR EACH ROW EXECUTE FUNCTION reject_exercise_template_question_link();

CREATE OR REPLACE FUNCTION reject_canonical_snapshot_usage_mismatch() RETURNS trigger
LANGUAGE plpgsql AS $$
DECLARE snapshot_usage text;
DECLARE membership_usage text;
BEGIN
    SELECT usage_class INTO snapshot_usage
    FROM canonical_audit_scope_snapshots
    WHERE id = NEW.snapshot_id;
    SELECT usage_class INTO membership_usage
    FROM canonical_question_catalog_memberships
    WHERE catalog_id = NEW.catalog_id AND question_version_id = NEW.question_version_id;
    IF snapshot_usage IS NULL OR membership_usage IS NULL OR snapshot_usage <> membership_usage THEN
        RAISE EXCEPTION 'canonical scope snapshot usage class does not match question provenance';
    END IF;
    RETURN NEW;
END;
$$;

CREATE TRIGGER canonical_scope_snapshot_question_usage_guard
BEFORE INSERT ON canonical_audit_scope_snapshot_questions
FOR EACH ROW EXECUTE FUNCTION reject_canonical_snapshot_usage_mismatch();

CREATE TRIGGER canonical_question_version_provenance_append_only
BEFORE UPDATE OR DELETE ON canonical_question_version_provenance
FOR EACH ROW EXECUTE FUNCTION reject_immutable_row_change();

-- Governed Question Review disposition/reclassification commands are an
-- append-only projection over the existing candidate authority.  The
-- candidate revision and digest are pinned here; technical approval and
-- publication continue to use checklist-governance's own aggregates.
CREATE TABLE canonical_governed_question_review_events (
    event_id text PRIMARY KEY,
    operation_id text NOT NULL UNIQUE,
    idempotency_key text NOT NULL UNIQUE,
    catalog_id text NOT NULL,
    question_version_id text NOT NULL REFERENCES question_versions(id),
    candidate_draft_version_id text NOT NULL,
    candidate_revision bigint NOT NULL CHECK (candidate_revision > 0),
    candidate_content_digest text NOT NULL,
    action text NOT NULL CHECK (action IN ('RETAIN', 'INCLUDE', 'EXCLUDE', 'DEFER', 'DOMAIN_RECLASSIFIED', 'TOPIC_RECLASSIFIED')),
    reason text NOT NULL CHECK (btrim(reason) <> ''),
    reviewed_domain text,
    reviewed_topic text,
    actor_subject_id text NOT NULL REFERENCES identity_references(subject_id),
    created_at timestamptz NOT NULL DEFAULT now(),
    FOREIGN KEY (catalog_id, question_version_id)
        REFERENCES canonical_question_catalog_memberships(catalog_id, question_version_id),
    FOREIGN KEY (candidate_draft_version_id, candidate_revision, candidate_content_digest)
        REFERENCES template_draft_versions(id, revision, candidate_content_digest)
);

CREATE TRIGGER canonical_governed_question_review_events_append_only
BEFORE UPDATE OR DELETE ON canonical_governed_question_review_events
FOR EACH ROW EXECUTE FUNCTION reject_immutable_row_change();

ALTER TABLE canonical_audit_scope_selection_operations
    ADD COLUMN expires_at timestamptz;

CREATE TABLE canonical_audit_scope_selection_preview_consumptions (
    preview_operation_id text PRIMARY KEY
        REFERENCES canonical_audit_scope_selection_operations(operation_id),
    commit_operation_id text NOT NULL UNIQUE,
    actor_subject_id text NOT NULL REFERENCES identity_references(subject_id),
    consumed_at timestamptz NOT NULL DEFAULT now()
);

CREATE TRIGGER canonical_audit_scope_selection_preview_consumptions_append_only
BEFORE UPDATE OR DELETE ON canonical_audit_scope_selection_preview_consumptions
FOR EACH ROW EXECUTE FUNCTION reject_immutable_row_change();

CREATE OR REPLACE FUNCTION preserve_inspection_package_snapshot() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    IF NEW.inspection_id IS DISTINCT FROM OLD.inspection_id
       OR NEW.checklist_template_version_id IS DISTINCT FROM OLD.checklist_template_version_id
       OR NEW.canonical_scope_snapshot_id IS DISTINCT FROM OLD.canonical_scope_snapshot_id
       OR NEW.package_version IS DISTINCT FROM OLD.package_version
       OR NEW.snapshot IS DISTINCT FROM OLD.snapshot
       OR NEW.expires_at IS DISTINCT FROM OLD.expires_at
       OR NEW.package_digest IS DISTINCT FROM OLD.package_digest THEN
        RAISE EXCEPTION 'inspection package snapshots are immutable' USING ERRCODE = '55000';
    END IF;
    RETURN NEW;
END;
$$;

CREATE TRIGGER inspection_packages_snapshot_immutable
BEFORE UPDATE ON inspection_packages
FOR EACH ROW EXECUTE FUNCTION preserve_inspection_package_snapshot();

CREATE INDEX inspection_packages_canonical_scope_snapshot_idx
    ON inspection_packages (canonical_scope_snapshot_id)
    WHERE canonical_scope_snapshot_id IS NOT NULL;

-- Governed catalog eligibility is explicit and scope-bound. A catalog row is
-- never considered package-eligible merely because it exists in a sealed
-- catalog; the authoritative provider-scope/typed-target applicability fact
-- must be present and append-only.
CREATE TABLE canonical_question_catalog_applicabilities (
    catalog_id text NOT NULL,
    question_version_id text NOT NULL,
    provider_scope_id text NOT NULL REFERENCES organization_service_provider_scopes(id),
    regulated_target_id text NOT NULL REFERENCES regulated_targets(id),
    status text NOT NULL CHECK (status IN ('ELIGIBLE', 'BLOCKED', 'UNRESOLVED')),
    reason text NOT NULL CHECK (btrim(reason) <> ''),
    actor_subject_id text NOT NULL REFERENCES identity_references(subject_id),
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (catalog_id, question_version_id, provider_scope_id, regulated_target_id),
    FOREIGN KEY (catalog_id, question_version_id)
        REFERENCES canonical_question_catalog_memberships(catalog_id, question_version_id)
);

CREATE INDEX canonical_question_catalog_applicability_scope_idx
    ON canonical_question_catalog_applicabilities (provider_scope_id, regulated_target_id, catalog_id, status);

CREATE TRIGGER canonical_question_catalog_applicabilities_append_only
BEFORE UPDATE OR DELETE ON canonical_question_catalog_applicabilities
FOR EACH ROW EXECUTE FUNCTION reject_immutable_row_change();

-- Coverage is many-to-many: a Lead Inspector may assign more than one team
-- member to the same immutable question version.  The question position is
-- a snapshot ordering value, not a uniqueness boundary.
ALTER TABLE canonical_audit_preparation_questions
    DROP CONSTRAINT IF EXISTS canonical_audit_preparation_questions_pkey,
    DROP CONSTRAINT IF EXISTS canonical_audit_preparation_questions_preparation_id_position_key,
    DROP CONSTRAINT IF EXISTS canonical_audit_preparation_questio_preparation_id_position_key;
ALTER TABLE canonical_audit_preparation_questions
    ADD PRIMARY KEY (preparation_id, question_version_id, subject_id);
