-- Approved-source catalog successor boundary.
--
-- The old disposable exercise/catalog-authority model is removed at the
-- schema boundary. Existing exercise rows are retained only as retired,
-- non-selectable lineage so migration does not silently turn them into live
-- operational content. New operational catalogs have one explicit source
-- origin and do not require an internal technical-approval/publication row.

ALTER TABLE canonical_question_catalogs
    ADD COLUMN IF NOT EXISTS source_origin text NOT NULL DEFAULT 'INTERNAL_GENERATED_CANDIDATE',
    ADD COLUMN IF NOT EXISTS source_manifest_sha256 text NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS catalog_root_digest text NOT NULL DEFAULT '';

UPDATE canonical_question_catalogs
SET catalog_root_digest = root_digest
WHERE catalog_root_digest = '';

ALTER TABLE canonical_question_catalog_import_runs
    ADD COLUMN IF NOT EXISTS source_origin text NOT NULL DEFAULT 'INTERNAL_GENERATED_CANDIDATE',
    ADD COLUMN IF NOT EXISTS source_manifest_sha256 text NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS catalog_root_digest text NOT NULL DEFAULT '';

ALTER TABLE canonical_question_catalog_memberships
    ADD COLUMN IF NOT EXISTS source_origin text NOT NULL DEFAULT 'INTERNAL_GENERATED_CANDIDATE';

ALTER TABLE canonical_question_version_provenance
    ADD COLUMN IF NOT EXISTS source_origin text NOT NULL DEFAULT 'INTERNAL_GENERATED_CANDIDATE';

ALTER TABLE canonical_audit_scope_drafts
    ADD COLUMN IF NOT EXISTS catalog_root_digest text NOT NULL DEFAULT '';

ALTER TABLE canonical_audit_scope_snapshots
    ADD COLUMN IF NOT EXISTS catalog_root_digest text NOT NULL DEFAULT '';

UPDATE canonical_audit_scope_drafts scope
SET catalog_root_digest = catalog.catalog_root_digest
FROM canonical_question_catalogs catalog
WHERE catalog.id = scope.catalog_id
  AND scope.catalog_root_digest = '';

UPDATE canonical_audit_scope_snapshots snapshot
SET catalog_root_digest = catalog.catalog_root_digest
FROM canonical_question_catalogs catalog
WHERE catalog.id = snapshot.catalog_id
  AND snapshot.catalog_root_digest = '';

-- Child rows carry the same retired governed usage value before the composite
-- foreign keys are narrowed to the single live usage class.
UPDATE canonical_question_catalog_memberships
SET usage_class = 'GOVERNED_OPERATIONAL',
    source_origin = 'INTERNAL_GENERATED_CANDIDATE'
WHERE usage_class = 'PREPROD_EXERCISE';
UPDATE canonical_question_version_provenance
SET usage_class = 'GOVERNED_OPERATIONAL',
    source_origin = 'INTERNAL_GENERATED_CANDIDATE'
WHERE usage_class = 'PREPROD_EXERCISE';
UPDATE canonical_audit_scope_drafts
SET usage_class = 'GOVERNED_OPERATIONAL'
WHERE usage_class = 'PREPROD_EXERCISE';
UPDATE canonical_audit_scope_snapshots
SET usage_class = 'GOVERNED_OPERATIONAL'
WHERE usage_class = 'PREPROD_EXERCISE';
UPDATE canonical_question_catalogs
SET usage_class = 'GOVERNED_OPERATIONAL',
    status = 'RETIRED',
    source_origin = 'INTERNAL_GENERATED_CANDIDATE'
WHERE usage_class = 'PREPROD_EXERCISE';

ALTER TABLE canonical_question_catalogs
    DROP CONSTRAINT IF EXISTS canonical_question_catalogs_usage_class_check,
    DROP CONSTRAINT IF EXISTS canonical_question_catalog_publication_shape,
    DROP CONSTRAINT IF EXISTS canonical_question_catalog_publication_fk,
    DROP CONSTRAINT IF EXISTS canonical_question_catalog_candidate_fk;
ALTER TABLE canonical_question_catalogs
    ADD CONSTRAINT canonical_question_catalogs_usage_class_check
        CHECK (usage_class = 'GOVERNED_OPERATIONAL'),
    ADD CONSTRAINT canonical_question_catalog_source_origin_check
        CHECK (source_origin IN ('IMPORTED_APPROVED_SOURCE', 'INTERNAL_GENERATED_CANDIDATE')),
    ADD CONSTRAINT canonical_question_catalog_root_digest_check
        CHECK (btrim(catalog_root_digest) <> '');

ALTER TABLE canonical_question_catalog_import_runs
    ADD CONSTRAINT canonical_question_catalog_import_source_origin_check
        CHECK (source_origin IN ('IMPORTED_APPROVED_SOURCE', 'INTERNAL_GENERATED_CANDIDATE'));

ALTER TABLE canonical_question_catalog_memberships
    DROP CONSTRAINT IF EXISTS canonical_question_catalog_memberships_usage_class_check,
    ADD CONSTRAINT canonical_question_catalog_memberships_usage_class_check
        CHECK (usage_class = 'GOVERNED_OPERATIONAL'),
    ADD CONSTRAINT canonical_question_catalog_memberships_source_origin_check
        CHECK (source_origin IN ('IMPORTED_APPROVED_SOURCE', 'INTERNAL_GENERATED_CANDIDATE'));

ALTER TABLE canonical_question_version_provenance
    DROP CONSTRAINT IF EXISTS canonical_question_version_provenance_usage_class_check,
    ADD CONSTRAINT canonical_question_version_provenance_usage_class_check
        CHECK (usage_class = 'GOVERNED_OPERATIONAL'),
    ADD CONSTRAINT canonical_question_version_provenance_source_origin_check
        CHECK (source_origin IN ('IMPORTED_APPROVED_SOURCE', 'INTERNAL_GENERATED_CANDIDATE'));

ALTER TABLE canonical_audit_scope_drafts
    DROP CONSTRAINT IF EXISTS canonical_audit_scope_drafts_usage_class_check,
    ADD CONSTRAINT canonical_audit_scope_drafts_usage_class_check
        CHECK (usage_class = 'GOVERNED_OPERATIONAL');
ALTER TABLE canonical_audit_scope_snapshots
    DROP CONSTRAINT IF EXISTS canonical_audit_scope_snapshots_usage_class_check,
    ADD CONSTRAINT canonical_audit_scope_snapshots_usage_class_check
        CHECK (usage_class = 'GOVERNED_OPERATIONAL');

ALTER TABLE canonical_audit_scope_drafts
    ADD CONSTRAINT canonical_audit_scope_draft_root_digest_check
        CHECK (catalog_root_digest = '' OR btrim(catalog_root_digest) <> '');
ALTER TABLE canonical_audit_scope_snapshots
    ADD CONSTRAINT canonical_audit_scope_snapshot_root_digest_check
        CHECK (catalog_root_digest = '' OR btrim(catalog_root_digest) <> '');

-- The governed membership trigger used to require a published candidate
-- template. Approved Aviation source is already operationally classified and
-- must be insertable without that second workflow.
DROP TRIGGER IF EXISTS canonical_governed_catalog_membership_guard ON canonical_question_catalog_memberships;
DROP FUNCTION IF EXISTS validate_canonical_governed_catalog_membership();

-- Remove the obsolete exercise review store and its trigger functions after
-- all historical references have been made non-live above.
DROP TRIGGER IF EXISTS template_draft_versions_exercise_provenance_guard ON template_draft_versions;
DROP TRIGGER IF EXISTS checklist_publication_decisions_exercise_provenance_guard ON checklist_publication_decisions;
DROP TRIGGER IF EXISTS template_version_questions_exercise_provenance_guard ON template_version_questions;
DROP TRIGGER IF EXISTS canonical_scope_snapshot_question_usage_guard ON canonical_audit_scope_snapshot_questions;
DROP FUNCTION IF EXISTS reject_exercise_question_version_reuse();
DROP FUNCTION IF EXISTS reject_exercise_published_question_version();
DROP FUNCTION IF EXISTS reject_exercise_template_question_link();
DROP FUNCTION IF EXISTS reject_canonical_snapshot_usage_mismatch();
DROP TABLE IF EXISTS canonical_exercise_question_review_events CASCADE;
DROP TABLE IF EXISTS canonical_exercise_question_review_drafts CASCADE;
