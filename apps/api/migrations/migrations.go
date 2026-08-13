package migrations

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/aviason/aviaSurveil/internal/platform/database"
)

const LatestVersion int64 = 46
const advisoryLockID int64 = 36020260721

//go:embed *.up.sql
var migrationFiles embed.FS

type migration struct {
	version int64
	name    string
	sql     string
}

func Apply(ctx context.Context, pool *database.Pool) error {
	connection, err := pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("acquire migration connection: %w", err)
	}
	defer connection.Release()
	if _, err := connection.Exec(ctx, "SELECT pg_advisory_lock($1)", advisoryLockID); err != nil {
		return fmt.Errorf("acquire migration lock: %w", err)
	}
	defer func() { _, _ = connection.Exec(context.Background(), "SELECT pg_advisory_unlock($1)", advisoryLockID) }()

	if _, err := connection.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version bigint PRIMARY KEY,
			name text NOT NULL,
			applied_at timestamptz NOT NULL DEFAULT now()
		)
	`); err != nil {
		return fmt.Errorf("create migration ledger: %w", err)
	}

	var current int64
	if err := connection.QueryRow(ctx, "SELECT COALESCE(MAX(version), 0) FROM schema_migrations").Scan(&current); err != nil {
		return fmt.Errorf("read migration ledger: %w", err)
	}
	available, err := load()
	if err != nil {
		return err
	}
	for _, candidate := range available {
		if candidate.version <= current {
			continue
		}
		transaction, err := connection.Begin(ctx)
		if err != nil {
			return fmt.Errorf("begin migration %s: %w", candidate.name, err)
		}
		if _, err := transaction.Exec(ctx, candidate.sql); err != nil {
			_ = transaction.Rollback(ctx)
			return fmt.Errorf("execute migration %s: %w", candidate.name, err)
		}
		if _, err := transaction.Exec(ctx, "INSERT INTO schema_migrations (version, name) VALUES ($1, $2)", candidate.version, candidate.name); err != nil {
			_ = transaction.Rollback(ctx)
			return fmt.Errorf("record migration %s: %w", candidate.name, err)
		}
		if err := transaction.Commit(ctx); err != nil {
			return fmt.Errorf("commit migration %s: %w", candidate.name, err)
		}
		current = candidate.version
	}
	if current != LatestVersion {
		return fmt.Errorf("migration version %d does not match embedded latest version %d", current, LatestVersion)
	}
	if err := RepairRegulatoryChecklistGovernance(ctx, pool); err != nil {
		return fmt.Errorf("apply version-%d forward repair: %w", LatestVersion, err)
	}
	return nil
}

func CurrentVersion(ctx context.Context, pool *database.Pool) (int64, error) {
	var version int64
	if err := pool.QueryRow(ctx, "SELECT COALESCE(MAX(version), 0) FROM schema_migrations").Scan(&version); err != nil {
		return 0, fmt.Errorf("read migration version: %w", err)
	}
	return version, nil
}

// RepairRegulatoryChecklistGovernance restores non-history derived objects
// idempotently without changing the migration ledger or deleting governed data.
func RepairRegulatoryChecklistGovernance(ctx context.Context, pool *database.Pool) error {
	const indexName = "organization_service_provider_scope_applicability_idx"
	const canonicalDefinition = "CREATE INDEX organization_service_provider_scope_applicability_idx ON public.organization_service_provider_scopes USING btree (organization_id, root_id, effective_from DESC, id DESC)"
	var definition string
	if err := pool.QueryRow(ctx, `
		SELECT COALESCE(pg_get_indexdef(to_regclass('public.organization_service_provider_scope_applicability_idx')), '')
	`).Scan(&definition); err != nil {
		return fmt.Errorf("read derived applicability index definition: %w", err)
	}
	if normalizeIndexDefinition(definition) != normalizeIndexDefinition(canonicalDefinition) {
		if _, err := pool.Exec(ctx, `DROP INDEX IF EXISTS organization_service_provider_scope_applicability_idx`); err != nil {
			return fmt.Errorf("drop incorrect derived applicability index: %w", err)
		}
		if _, err := pool.Exec(ctx, `CREATE INDEX organization_service_provider_scope_applicability_idx ON organization_service_provider_scopes (organization_id, root_id, effective_from DESC, id DESC)`); err != nil {
			return fmt.Errorf("restore derived applicability index: %w", err)
		}
	}
	if _, err := pool.Exec(ctx, task5ForwardRepairSQL); err != nil {
		return fmt.Errorf("restore complete Task 5 version-21 boundary: %w", err)
	}
	return nil
}

const task5ForwardRepairSQL = `
-- Migration 42 physically removes the obsolete fixed-ID package-draft store.
DROP TABLE IF EXISTS inspection_package_drafts;
-- Canonical package source identity is XOR, never a legacy template plus a
-- canonical scope at the same time. Keep this repair idempotent for databases
-- that already recorded migration 30 before the successor constraint was
-- tightened.
ALTER TABLE inspection_packages DROP CONSTRAINT IF EXISTS inspection_packages_source_identity_check;
ALTER TABLE inspection_packages ADD CONSTRAINT inspection_packages_source_identity_check
    CHECK ((checklist_template_version_id IS NOT NULL) <> (canonical_scope_snapshot_id IS NOT NULL));
ALTER TABLE audit_assignments ADD COLUMN IF NOT EXISTS released_scope_snapshot_id text REFERENCES canonical_audit_scope_snapshots(id);
ALTER TABLE audit_assignments ADD COLUMN IF NOT EXISTS planning_item_id text REFERENCES surveillance_plan_items(id);
ALTER TABLE canonical_audit_scope_snapshots ADD COLUMN IF NOT EXISTS planning_snapshot_digest text NOT NULL DEFAULT '';
ALTER TABLE canonical_audit_scope_snapshots ALTER COLUMN planning_snapshot_digest DROP DEFAULT;
DO $digest_constraint$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid='canonical_audit_scope_snapshots'::regclass
          AND conname='canonical_audit_scope_snapshot_planning_digest'
    ) THEN
            ALTER TABLE canonical_audit_scope_snapshots
            ADD CONSTRAINT canonical_audit_scope_snapshot_planning_digest
            CHECK (btrim(planning_snapshot_digest) = '' OR governed_sha256(planning_snapshot_digest));
    END IF;
END
$digest_constraint$;
ALTER TABLE canonical_audit_preparation_snapshots ADD COLUMN IF NOT EXISTS assignment_id text REFERENCES audit_assignments(id);
ALTER TABLE canonical_audit_preparation_snapshots ADD COLUMN IF NOT EXISTS confirmed_assignment_revision bigint;
CREATE TEMP TABLE IF NOT EXISTS canonical_preparation_confirmation_bindings (
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
  AND legacy.confirmed_assignment_revision IS NULL
ON CONFLICT (legacy_id) DO NOTHING;
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
JOIN canonical_audit_preparation_snapshots legacy ON legacy.id = binding.legacy_id
ON CONFLICT (id) DO NOTHING;
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
JOIN canonical_audit_preparation_questions questions ON questions.preparation_id = binding.legacy_id
ON CONFLICT DO NOTHING;
DO $binding_repair$
BEGIN
    IF EXISTS (
        SELECT 1 FROM canonical_audit_preparation_snapshots legacy
        WHERE legacy.status = 'CONFIRMED' AND legacy.confirmed_assignment_revision IS NULL
          AND NOT EXISTS (
              SELECT 1 FROM canonical_audit_preparation_snapshots successor
              WHERE successor.id = 'preparation:legacy-binding:' || legacy.id
                AND successor.confirmed_assignment_revision IS NOT NULL
          )
    ) THEN
        RAISE EXCEPTION 'cannot reconcile confirmed preparation without an exact audit assignment: forward repair is fail-closed';
    END IF;
END
$binding_repair$;
DO $repair$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid='canonical_audit_preparation_snapshots'::regclass AND conname='canonical_audit_preparation_confirmed_revision_shape') THEN
        ALTER TABLE canonical_audit_preparation_snapshots ADD CONSTRAINT canonical_audit_preparation_confirmed_revision_shape CHECK ((status='CONFIRMED' AND confirmed_assignment_revision IS NOT NULL AND confirmed_assignment_revision > 0) OR status <> 'CONFIRMED') NOT VALID;
    END IF;
END
$repair$;
-- Migration 30 introduced assignment ownership after some candidate databases
-- may already have preparation receipts. Those receipts are append-only and
-- must not be rewritten during a forward repair. Legacy rows therefore remain
-- readable with a NULL assignment_id; the NOT VALID check below fails closed
-- for every new canonical preparation until an explicit reconciliation is
-- performed outside this repair.
DO $repair$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid='canonical_audit_preparation_snapshots'::regclass AND conname='canonical_audit_preparation_assignment_required') THEN
        ALTER TABLE canonical_audit_preparation_snapshots ADD CONSTRAINT canonical_audit_preparation_assignment_required CHECK (assignment_id IS NOT NULL) NOT VALID;
    END IF;
    IF NOT EXISTS (SELECT 1 FROM canonical_audit_preparation_snapshots WHERE assignment_id IS NULL) THEN
        ALTER TABLE canonical_audit_preparation_snapshots VALIDATE CONSTRAINT canonical_audit_preparation_assignment_required;
    END IF;
END
$repair$;
CREATE UNIQUE INDEX IF NOT EXISTS canonical_audit_preparation_assignment_revision_idx ON canonical_audit_preparation_snapshots (assignment_id, revision);
ALTER TABLE canonical_audit_preparation_questions DROP CONSTRAINT IF EXISTS canonical_audit_preparation_questions_pkey;
ALTER TABLE canonical_audit_preparation_questions ADD PRIMARY KEY (preparation_id, question_version_id, subject_id);
ALTER TABLE canonical_audit_preparation_questions DROP CONSTRAINT IF EXISTS canonical_audit_preparation_questions_preparation_id_position_key;
CREATE UNIQUE INDEX IF NOT EXISTS canonical_audit_preparation_questions_position_subject_key ON canonical_audit_preparation_questions (preparation_id, position, subject_id);
DO $preparation_position_repair$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid='canonical_audit_scope_snapshot_questions'::regclass AND conname='canonical_scope_snapshot_question_position_key') THEN
        ALTER TABLE canonical_audit_scope_snapshot_questions
            ADD CONSTRAINT canonical_scope_snapshot_question_position_key
            UNIQUE (snapshot_id, question_version_id, position);
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid='canonical_audit_preparation_questions'::regclass AND conname='canonical_preparation_question_scope_position_fkey') THEN
        ALTER TABLE canonical_audit_preparation_questions
            ADD CONSTRAINT canonical_preparation_question_scope_position_fkey
            FOREIGN KEY (released_scope_snapshot_id, question_version_id, position)
            REFERENCES canonical_audit_scope_snapshot_questions (snapshot_id, question_version_id, position);
    END IF;
END
$preparation_position_repair$;
ALTER TABLE canonical_exercise_question_review_drafts ADD COLUMN IF NOT EXISTS scope_draft_id text REFERENCES canonical_audit_scope_drafts(id);
ALTER TABLE canonical_exercise_question_review_events ADD COLUMN IF NOT EXISTS scope_draft_id text REFERENCES canonical_audit_scope_drafts(id);
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
DO $exercise_scope_repair$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid='canonical_exercise_question_review_drafts'::regclass AND conname='canonical_exercise_question_review_drafts_scope_revision_key') THEN
        ALTER TABLE canonical_exercise_question_review_drafts ADD CONSTRAINT canonical_exercise_question_review_drafts_scope_revision_key UNIQUE (scope_draft_id, catalog_id, question_version_id, revision);
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid='canonical_exercise_question_review_drafts'::regclass AND conname='canonical_exercise_question_review_drafts_scope_required') THEN
        ALTER TABLE canonical_exercise_question_review_drafts ADD CONSTRAINT canonical_exercise_question_review_drafts_scope_required CHECK (scope_draft_id IS NOT NULL) NOT VALID;
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid='canonical_exercise_question_review_events'::regclass AND conname='canonical_exercise_question_review_events_scope_required') THEN
        ALTER TABLE canonical_exercise_question_review_events ADD CONSTRAINT canonical_exercise_question_review_events_scope_required CHECK (scope_draft_id IS NOT NULL) NOT VALID;
    END IF;
END
$exercise_scope_repair$;
ALTER TABLE canonical_exercise_question_review_drafts VALIDATE CONSTRAINT canonical_exercise_question_review_drafts_scope_required;
ALTER TABLE canonical_exercise_question_review_events VALIDATE CONSTRAINT canonical_exercise_question_review_events_scope_required;
CREATE INDEX IF NOT EXISTS canonical_exercise_question_review_drafts_scope_question_idx ON canonical_exercise_question_review_drafts (scope_draft_id, catalog_id, question_version_id, revision DESC);
CREATE INDEX IF NOT EXISTS canonical_exercise_question_review_events_scope_question_idx ON canonical_exercise_question_review_events (scope_draft_id, occurred_at DESC, event_id DESC);
ALTER TABLE canonical_question_catalogs ADD COLUMN IF NOT EXISTS governed_publication_decision_id text;
ALTER TABLE canonical_question_catalogs ADD COLUMN IF NOT EXISTS governed_candidate_draft_version_id text;
ALTER TABLE canonical_question_catalogs ADD COLUMN IF NOT EXISTS governed_candidate_revision bigint;
ALTER TABLE canonical_question_catalogs ADD COLUMN IF NOT EXISTS governed_candidate_content_digest text;
DO $repair$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid='canonical_question_catalogs'::regclass AND conname='canonical_question_catalog_publication_shape') THEN
        ALTER TABLE canonical_question_catalogs ADD CONSTRAINT canonical_question_catalog_publication_shape CHECK ((usage_class='PREPROD_EXERCISE' AND governed_publication_decision_id IS NULL AND governed_candidate_draft_version_id IS NULL AND governed_candidate_revision IS NULL AND governed_candidate_content_digest IS NULL) OR (usage_class='GOVERNED_OPERATIONAL' AND governed_publication_decision_id IS NOT NULL AND governed_candidate_draft_version_id IS NOT NULL AND governed_candidate_revision IS NOT NULL AND governed_candidate_content_digest IS NOT NULL));
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid='canonical_question_catalogs'::regclass AND conname='canonical_question_catalog_publication_fk') THEN
        ALTER TABLE canonical_question_catalogs ADD CONSTRAINT canonical_question_catalog_publication_fk FOREIGN KEY (governed_publication_decision_id) REFERENCES checklist_publication_decisions(id);
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid='canonical_question_catalogs'::regclass AND conname='canonical_question_catalog_candidate_fk') THEN
        ALTER TABLE canonical_question_catalogs ADD CONSTRAINT canonical_question_catalog_candidate_fk FOREIGN KEY (governed_candidate_draft_version_id, governed_candidate_revision, governed_candidate_content_digest) REFERENCES template_draft_versions(id, revision, candidate_content_digest);
    END IF;
END
$repair$;
CREATE OR REPLACE FUNCTION validate_canonical_governed_catalog_membership() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE catalog record; published_count integer;
BEGIN
    SELECT * INTO catalog FROM canonical_question_catalogs WHERE id=NEW.catalog_id;
    IF catalog.usage_class='GOVERNED_OPERATIONAL' THEN
        IF catalog.governed_publication_decision_id IS NULL OR catalog.governed_candidate_draft_version_id IS NULL OR catalog.governed_candidate_revision IS NULL OR catalog.governed_candidate_content_digest IS NULL THEN RAISE EXCEPTION 'governed catalog membership requires one exact publication decision and candidate digest'; END IF;
        SELECT count(*) INTO published_count FROM template_version_questions link JOIN checklist_template_versions template ON template.id=link.template_version_id WHERE link.question_version_id=NEW.question_version_id AND template.candidate_draft_version_id=catalog.governed_candidate_draft_version_id AND template.candidate_revision=catalog.governed_candidate_revision AND template.candidate_content_digest=catalog.governed_candidate_content_digest AND template.publication_decision_id=catalog.governed_publication_decision_id AND template.published_at IS NOT NULL;
        IF published_count<>1 THEN RAISE EXCEPTION 'governed catalog question must be present in the exact published candidate template'; END IF;
    END IF;
    RETURN NEW;
END;
$$;
DROP TRIGGER IF EXISTS canonical_governed_catalog_membership_guard ON canonical_question_catalog_memberships;
CREATE TRIGGER canonical_governed_catalog_membership_guard BEFORE INSERT ON canonical_question_catalog_memberships FOR EACH ROW EXECUTE FUNCTION validate_canonical_governed_catalog_membership();
ALTER TABLE canonical_audit_scope_selection_operations ADD COLUMN IF NOT EXISTS expires_at timestamptz;
CREATE TABLE IF NOT EXISTS canonical_audit_scope_selection_preview_consumptions (
    preview_operation_id text PRIMARY KEY REFERENCES canonical_audit_scope_selection_operations(operation_id),
    commit_operation_id text NOT NULL UNIQUE,
    actor_subject_id text NOT NULL REFERENCES identity_references(subject_id),
    consumed_at timestamptz NOT NULL DEFAULT now()
);
DROP TRIGGER IF EXISTS canonical_audit_scope_selection_preview_consumptions_append_only ON canonical_audit_scope_selection_preview_consumptions;
CREATE TRIGGER canonical_audit_scope_selection_preview_consumptions_append_only BEFORE UPDATE OR DELETE ON canonical_audit_scope_selection_preview_consumptions FOR EACH ROW EXECUTE FUNCTION reject_immutable_row_change();

CREATE TABLE IF NOT EXISTS canonical_question_version_provenance (
    question_version_id text REFERENCES question_versions(id),
    usage_class text NOT NULL CHECK (usage_class IN ('GOVERNED_OPERATIONAL', 'PREPROD_EXERCISE')),
    catalog_id text NOT NULL REFERENCES canonical_question_catalogs(id),
    recorded_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (question_version_id, usage_class, catalog_id)
);
DO $repair$
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
    ) THEN
        RAISE EXCEPTION 'existing question version has conflicting canonical provenance';
    END IF;
END
$repair$;
INSERT INTO canonical_question_version_provenance
    (question_version_id, usage_class, catalog_id, recorded_at)
SELECT membership.question_version_id, membership.usage_class, membership.catalog_id, now()
FROM canonical_question_catalog_memberships membership
ON CONFLICT (question_version_id, usage_class, catalog_id) DO NOTHING;
DO $repair$
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
$repair$;
CREATE OR REPLACE FUNCTION reject_catalog_membership_provenance_mismatch() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    PERFORM pg_advisory_xact_lock(hashtextextended('qv-provenance:' || NEW.question_version_id, 0));
    IF EXISTS (SELECT 1 FROM canonical_question_version_provenance WHERE question_version_id = NEW.question_version_id AND usage_class <> NEW.usage_class)
       OR NOT EXISTS (SELECT 1 FROM canonical_question_version_provenance WHERE question_version_id = NEW.question_version_id AND usage_class = NEW.usage_class AND catalog_id = NEW.catalog_id) THEN
        RAISE EXCEPTION 'catalog membership must bind the exact immutable question provenance and usage class';
    END IF;
    RETURN NEW;
END;
$$;
DROP TRIGGER IF EXISTS canonical_catalog_membership_provenance_guard ON canonical_question_catalog_memberships;
CREATE TRIGGER canonical_catalog_membership_provenance_guard BEFORE INSERT ON canonical_question_catalog_memberships FOR EACH ROW EXECUTE FUNCTION reject_catalog_membership_provenance_mismatch();
CREATE OR REPLACE FUNCTION reject_exercise_question_version_reuse() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF EXISTS (SELECT 1 FROM canonical_question_version_provenance provenance WHERE provenance.question_version_id = ANY(NEW.question_version_ids) AND provenance.usage_class = 'PREPROD_EXERCISE') THEN
        RAISE EXCEPTION 'PREPROD_EXERCISE question versions cannot enter a governed candidate';
    END IF;
    RETURN NEW;
END;
$$;
DROP TRIGGER IF EXISTS template_draft_versions_exercise_provenance_guard ON template_draft_versions;
CREATE TRIGGER template_draft_versions_exercise_provenance_guard BEFORE INSERT OR UPDATE OF question_version_ids ON template_draft_versions FOR EACH ROW EXECUTE FUNCTION reject_exercise_question_version_reuse();
CREATE OR REPLACE FUNCTION reject_exercise_published_question_version() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF EXISTS (SELECT 1 FROM template_draft_versions draft JOIN canonical_question_version_provenance provenance ON provenance.question_version_id = ANY(draft.question_version_ids) WHERE draft.id = NEW.candidate_draft_version_id AND provenance.usage_class = 'PREPROD_EXERCISE') THEN
        RAISE EXCEPTION 'PREPROD_EXERCISE question versions cannot be published';
    END IF;
    RETURN NEW;
END;
$$;
DROP TRIGGER IF EXISTS checklist_publication_decisions_exercise_provenance_guard ON checklist_publication_decisions;
CREATE TRIGGER checklist_publication_decisions_exercise_provenance_guard BEFORE INSERT ON checklist_publication_decisions FOR EACH ROW EXECUTE FUNCTION reject_exercise_published_question_version();
CREATE OR REPLACE FUNCTION reject_exercise_template_question_link() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF EXISTS (SELECT 1 FROM checklist_template_versions template JOIN canonical_question_version_provenance provenance ON provenance.question_version_id = NEW.question_version_id WHERE template.id = NEW.template_version_id AND template.candidate_draft_version_id IS NOT NULL AND provenance.usage_class = 'PREPROD_EXERCISE') THEN
        RAISE EXCEPTION 'PREPROD_EXERCISE question versions cannot enter a governed published template';
    END IF;
    RETURN NEW;
END;
$$;
DROP TRIGGER IF EXISTS template_version_questions_exercise_provenance_guard ON template_version_questions;
CREATE TRIGGER template_version_questions_exercise_provenance_guard BEFORE INSERT ON template_version_questions FOR EACH ROW EXECUTE FUNCTION reject_exercise_template_question_link();
CREATE OR REPLACE FUNCTION reject_canonical_snapshot_usage_mismatch() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE snapshot_usage text; membership_usage text;
BEGIN
    SELECT usage_class INTO snapshot_usage FROM canonical_audit_scope_snapshots WHERE id = NEW.snapshot_id;
    SELECT usage_class INTO membership_usage FROM canonical_question_catalog_memberships WHERE catalog_id = NEW.catalog_id AND question_version_id = NEW.question_version_id;
    IF snapshot_usage IS NULL OR membership_usage IS NULL OR snapshot_usage <> membership_usage THEN
        RAISE EXCEPTION 'canonical scope snapshot usage class does not match question provenance';
    END IF;
    RETURN NEW;
END;
$$;
DROP TRIGGER IF EXISTS canonical_scope_snapshot_question_usage_guard ON canonical_audit_scope_snapshot_questions;
CREATE TRIGGER canonical_scope_snapshot_question_usage_guard BEFORE INSERT ON canonical_audit_scope_snapshot_questions FOR EACH ROW EXECUTE FUNCTION reject_canonical_snapshot_usage_mismatch();
DROP TRIGGER IF EXISTS canonical_question_version_provenance_append_only ON canonical_question_version_provenance;
CREATE TRIGGER canonical_question_version_provenance_append_only BEFORE UPDATE OR DELETE ON canonical_question_version_provenance FOR EACH ROW EXECUTE FUNCTION reject_immutable_row_change();

-- A question version may be reused by multiple catalogs in the same usage
-- class, but never crosses from exercise into governed operational content.
ALTER TABLE canonical_question_version_provenance DROP CONSTRAINT IF EXISTS canonical_question_version_provenance_pkey;
DO $provenance_shape$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid='canonical_question_version_provenance'::regclass
          AND conname='canonical_question_version_provenance_pkey'
    ) THEN
        ALTER TABLE canonical_question_version_provenance
            ADD PRIMARY KEY (question_version_id, usage_class, catalog_id);
    END IF;
END
$provenance_shape$;
CREATE OR REPLACE FUNCTION reject_catalog_membership_provenance_mismatch() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    PERFORM pg_advisory_xact_lock(hashtextextended('qv-provenance:' || NEW.question_version_id, 0));
    IF EXISTS (
        SELECT 1 FROM canonical_question_version_provenance provenance
        WHERE provenance.question_version_id = NEW.question_version_id
          AND provenance.usage_class <> NEW.usage_class
    ) THEN
        RAISE EXCEPTION 'catalog membership usage class conflicts with immutable question provenance';
    END IF;
    IF NOT EXISTS (
        SELECT 1 FROM canonical_question_version_provenance provenance
        WHERE provenance.question_version_id = NEW.question_version_id
          AND provenance.usage_class = NEW.usage_class
          AND provenance.catalog_id = NEW.catalog_id
    ) THEN
        RAISE EXCEPTION 'catalog membership must bind the exact immutable question provenance';
    END IF;
    RETURN NEW;
END;
$$;
DROP TRIGGER IF EXISTS canonical_catalog_membership_provenance_guard ON canonical_question_catalog_memberships;
CREATE TRIGGER canonical_catalog_membership_provenance_guard BEFORE INSERT ON canonical_question_catalog_memberships FOR EACH ROW EXECUTE FUNCTION reject_catalog_membership_provenance_mismatch();

CREATE TABLE IF NOT EXISTS canonical_governed_question_review_events (
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
    FOREIGN KEY (catalog_id, question_version_id) REFERENCES canonical_question_catalog_memberships(catalog_id, question_version_id),
    FOREIGN KEY (candidate_draft_version_id, candidate_revision, candidate_content_digest) REFERENCES template_draft_versions(id, revision, candidate_content_digest)
);
DROP TRIGGER IF EXISTS canonical_governed_question_review_events_append_only ON canonical_governed_question_review_events;
CREATE TRIGGER canonical_governed_question_review_events_append_only BEFORE UPDATE OR DELETE ON canonical_governed_question_review_events FOR EACH ROW EXECUTE FUNCTION reject_immutable_row_change();

CREATE OR REPLACE FUNCTION preserve_inspection_package_snapshot() RETURNS trigger LANGUAGE plpgsql AS $$
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
DROP TRIGGER IF EXISTS inspection_packages_snapshot_immutable ON inspection_packages;
CREATE TRIGGER inspection_packages_snapshot_immutable BEFORE UPDATE ON inspection_packages FOR EACH ROW EXECUTE FUNCTION preserve_inspection_package_snapshot();

CREATE TABLE IF NOT EXISTS canonical_question_catalog_applicabilities (
    catalog_id text NOT NULL,
    question_version_id text NOT NULL,
    provider_scope_id text NOT NULL REFERENCES organization_service_provider_scopes(id),
    regulated_target_id text NOT NULL REFERENCES regulated_targets(id),
    status text NOT NULL CHECK (status IN ('ELIGIBLE', 'BLOCKED', 'UNRESOLVED')),
    reason text NOT NULL CHECK (btrim(reason) <> ''),
    actor_subject_id text NOT NULL REFERENCES identity_references(subject_id),
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (catalog_id, question_version_id, provider_scope_id, regulated_target_id),
    FOREIGN KEY (catalog_id, question_version_id) REFERENCES canonical_question_catalog_memberships(catalog_id, question_version_id)
);
CREATE INDEX IF NOT EXISTS canonical_question_catalog_applicability_scope_idx
    ON canonical_question_catalog_applicabilities (provider_scope_id, regulated_target_id, catalog_id, status);
DROP TRIGGER IF EXISTS canonical_question_catalog_applicabilities_append_only ON canonical_question_catalog_applicabilities;
CREATE TRIGGER canonical_question_catalog_applicabilities_append_only BEFORE UPDATE OR DELETE ON canonical_question_catalog_applicabilities FOR EACH ROW EXECUTE FUNCTION reject_immutable_row_change();

CREATE TABLE IF NOT EXISTS regulatory_source_gap_facts (
	id text PRIMARY KEY,
	regulatory_source_version_id text NOT NULL REFERENCES regulatory_source_versions(id),
	gap_id text NOT NULL,
	reason text NOT NULL,
	ordinal integer NOT NULL,
	created_at timestamptz NOT NULL DEFAULT now()
);
ALTER TABLE regulatory_source_gap_facts ADD COLUMN IF NOT EXISTS regulatory_source_version_id text REFERENCES regulatory_source_versions(id);
ALTER TABLE regulatory_source_gap_facts ADD COLUMN IF NOT EXISTS gap_id text;
ALTER TABLE regulatory_source_gap_facts ADD COLUMN IF NOT EXISTS reason text;
ALTER TABLE regulatory_source_gap_facts ADD COLUMN IF NOT EXISTS ordinal integer;
ALTER TABLE regulatory_source_gap_facts ADD COLUMN IF NOT EXISTS created_at timestamptz DEFAULT now();
ALTER TABLE regulatory_source_gap_facts ALTER COLUMN regulatory_source_version_id SET NOT NULL;
ALTER TABLE regulatory_source_gap_facts ALTER COLUMN gap_id SET NOT NULL;
ALTER TABLE regulatory_source_gap_facts ALTER COLUMN reason SET NOT NULL;
ALTER TABLE regulatory_source_gap_facts ALTER COLUMN ordinal SET NOT NULL;
ALTER TABLE regulatory_source_gap_facts ALTER COLUMN created_at SET DEFAULT now();
ALTER TABLE regulatory_source_gap_facts ALTER COLUMN created_at SET NOT NULL;
ALTER TABLE regulatory_source_gap_facts DROP CONSTRAINT IF EXISTS regulatory_source_gap_facts_gap_id_check;
ALTER TABLE regulatory_source_gap_facts ADD CONSTRAINT regulatory_source_gap_facts_gap_id_check CHECK (btrim(gap_id) <> '');
ALTER TABLE regulatory_source_gap_facts DROP CONSTRAINT IF EXISTS regulatory_source_gap_facts_reason_check;
ALTER TABLE regulatory_source_gap_facts ADD CONSTRAINT regulatory_source_gap_facts_reason_check CHECK (btrim(reason) <> '');
ALTER TABLE regulatory_source_gap_facts DROP CONSTRAINT IF EXISTS regulatory_source_gap_facts_ordinal_check;
ALTER TABLE regulatory_source_gap_facts ADD CONSTRAINT regulatory_source_gap_facts_ordinal_check CHECK (ordinal >= 0);
DO $repair$
BEGIN
	IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid='regulatory_source_gap_facts'::regclass AND conname='regulatory_source_gap_facts_source_gap_unique') THEN
		ALTER TABLE regulatory_source_gap_facts ADD CONSTRAINT regulatory_source_gap_facts_source_gap_unique UNIQUE (regulatory_source_version_id, gap_id);
	END IF;
END
$repair$;

CREATE TABLE IF NOT EXISTS governed_candidate_commands (
	id text PRIMARY KEY,
	command_kind text NOT NULL,
	operation_id text NOT NULL UNIQUE,
	idempotency_key text NOT NULL UNIQUE,
	semantic_payload_digest text NOT NULL,
	generation_run_id text NOT NULL REFERENCES regulatory_generation_runs(id),
	candidate_draft_version_id text REFERENCES template_draft_versions(id),
	candidate_revision bigint,
	candidate_content_digest text,
	actor_subject_id text NOT NULL REFERENCES identity_references(subject_id),
	reason text NOT NULL,
	audit_event_id text NOT NULL REFERENCES audit_events(event_id),
	created_at timestamptz NOT NULL DEFAULT now(),
	FOREIGN KEY (candidate_draft_version_id, candidate_revision, candidate_content_digest)
		REFERENCES template_draft_versions(id, revision, candidate_content_digest)
);
ALTER TABLE governed_candidate_commands ADD COLUMN IF NOT EXISTS command_kind text;
ALTER TABLE governed_candidate_commands ADD COLUMN IF NOT EXISTS operation_id text;
ALTER TABLE governed_candidate_commands ADD COLUMN IF NOT EXISTS idempotency_key text;
ALTER TABLE governed_candidate_commands ADD COLUMN IF NOT EXISTS semantic_payload_digest text;
ALTER TABLE governed_candidate_commands ADD COLUMN IF NOT EXISTS generation_run_id text REFERENCES regulatory_generation_runs(id);
ALTER TABLE governed_candidate_commands ADD COLUMN IF NOT EXISTS candidate_draft_version_id text REFERENCES template_draft_versions(id);
ALTER TABLE governed_candidate_commands ADD COLUMN IF NOT EXISTS candidate_revision bigint;
ALTER TABLE governed_candidate_commands ADD COLUMN IF NOT EXISTS candidate_content_digest text;
ALTER TABLE governed_candidate_commands ADD COLUMN IF NOT EXISTS actor_subject_id text REFERENCES identity_references(subject_id);
ALTER TABLE governed_candidate_commands ADD COLUMN IF NOT EXISTS reason text;
ALTER TABLE governed_candidate_commands ADD COLUMN IF NOT EXISTS audit_event_id text REFERENCES audit_events(event_id);
ALTER TABLE governed_candidate_commands ADD COLUMN IF NOT EXISTS created_at timestamptz DEFAULT now();
ALTER TABLE governed_candidate_commands ALTER COLUMN command_kind SET NOT NULL;
ALTER TABLE governed_candidate_commands ALTER COLUMN operation_id SET NOT NULL;
ALTER TABLE governed_candidate_commands ALTER COLUMN idempotency_key SET NOT NULL;
ALTER TABLE governed_candidate_commands ALTER COLUMN semantic_payload_digest SET NOT NULL;
ALTER TABLE governed_candidate_commands ALTER COLUMN generation_run_id SET NOT NULL;
ALTER TABLE governed_candidate_commands ALTER COLUMN candidate_draft_version_id DROP NOT NULL;
ALTER TABLE governed_candidate_commands ALTER COLUMN candidate_revision DROP NOT NULL;
ALTER TABLE governed_candidate_commands ALTER COLUMN candidate_content_digest DROP NOT NULL;
ALTER TABLE governed_candidate_commands ALTER COLUMN actor_subject_id SET NOT NULL;
ALTER TABLE governed_candidate_commands ALTER COLUMN reason SET NOT NULL;
ALTER TABLE governed_candidate_commands ALTER COLUMN audit_event_id SET NOT NULL;
ALTER TABLE governed_candidate_commands ALTER COLUMN created_at SET DEFAULT now();
ALTER TABLE governed_candidate_commands ALTER COLUMN created_at SET NOT NULL;
ALTER TABLE governed_candidate_commands DROP CONSTRAINT IF EXISTS governed_candidate_commands_command_kind_check;
ALTER TABLE governed_candidate_commands ADD CONSTRAINT governed_candidate_commands_command_kind_check CHECK (command_kind IN ('IMPORTED_GENERATION_RUN', 'FAILED_IMPORT', 'REVISION_CREATED', 'DEPARTMENT_REVIEW_SUBMITTED'));
ALTER TABLE governed_candidate_commands DROP CONSTRAINT IF EXISTS governed_candidate_commands_operation_id_check;
ALTER TABLE governed_candidate_commands ADD CONSTRAINT governed_candidate_commands_operation_id_check CHECK (btrim(operation_id) <> '');
ALTER TABLE governed_candidate_commands DROP CONSTRAINT IF EXISTS governed_candidate_commands_idempotency_key_check;
ALTER TABLE governed_candidate_commands ADD CONSTRAINT governed_candidate_commands_idempotency_key_check CHECK (btrim(idempotency_key) <> '');
ALTER TABLE governed_candidate_commands DROP CONSTRAINT IF EXISTS governed_candidate_commands_semantic_payload_digest_check;
ALTER TABLE governed_candidate_commands ADD CONSTRAINT governed_candidate_commands_semantic_payload_digest_check CHECK (governed_sha256(semantic_payload_digest));
ALTER TABLE governed_candidate_commands DROP CONSTRAINT IF EXISTS governed_candidate_commands_candidate_content_digest_check;
ALTER TABLE governed_candidate_commands ADD CONSTRAINT governed_candidate_commands_candidate_content_digest_check CHECK (candidate_content_digest IS NULL OR governed_sha256(candidate_content_digest));
ALTER TABLE governed_candidate_commands DROP CONSTRAINT IF EXISTS governed_candidate_commands_candidate_shape_check;
ALTER TABLE governed_candidate_commands ADD CONSTRAINT governed_candidate_commands_candidate_shape_check CHECK ((command_kind='FAILED_IMPORT' AND candidate_draft_version_id IS NULL AND candidate_revision IS NULL AND candidate_content_digest IS NULL) OR (command_kind<>'FAILED_IMPORT' AND candidate_draft_version_id IS NOT NULL AND candidate_revision IS NOT NULL AND candidate_content_digest IS NOT NULL));
ALTER TABLE governed_candidate_commands DROP CONSTRAINT IF EXISTS governed_candidate_commands_reason_check;
ALTER TABLE governed_candidate_commands ADD CONSTRAINT governed_candidate_commands_reason_check CHECK (btrim(reason) <> '');
DO $repair$
BEGIN
	IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid='governed_candidate_commands'::regclass AND contype='u' AND pg_get_constraintdef(oid)='UNIQUE (operation_id)') THEN
		ALTER TABLE governed_candidate_commands ADD CONSTRAINT governed_candidate_commands_operation_id_unique UNIQUE (operation_id);
	END IF;
	IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid='governed_candidate_commands'::regclass AND contype='u' AND pg_get_constraintdef(oid)='UNIQUE (idempotency_key)') THEN
		ALTER TABLE governed_candidate_commands ADD CONSTRAINT governed_candidate_commands_idempotency_key_unique UNIQUE (idempotency_key);
	END IF;
	IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid='governed_candidate_commands'::regclass AND contype='f' AND pg_get_constraintdef(oid) LIKE 'FOREIGN KEY (generation_run_id)%') THEN
		ALTER TABLE governed_candidate_commands ADD CONSTRAINT governed_candidate_commands_generation_run_fkey FOREIGN KEY (generation_run_id) REFERENCES regulatory_generation_runs(id);
	END IF;
	IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid='governed_candidate_commands'::regclass AND contype='f' AND pg_get_constraintdef(oid) LIKE 'FOREIGN KEY (candidate_draft_version_id, candidate_revision, candidate_content_digest)%') THEN
		ALTER TABLE governed_candidate_commands ADD CONSTRAINT governed_candidate_commands_candidate_identity_fkey FOREIGN KEY (candidate_draft_version_id, candidate_revision, candidate_content_digest) REFERENCES template_draft_versions(id, revision, candidate_content_digest);
	END IF;
END
$repair$;

DROP INDEX IF EXISTS regulatory_source_gap_facts_source_idx;
CREATE INDEX regulatory_source_gap_facts_source_idx ON regulatory_source_gap_facts (regulatory_source_version_id, ordinal, gap_id);
DROP INDEX IF EXISTS governed_candidate_commands_run_candidate_idx;
CREATE INDEX governed_candidate_commands_run_candidate_idx ON governed_candidate_commands (generation_run_id, candidate_draft_version_id, candidate_revision);
ALTER TABLE regulatory_generated_mapping_snapshots ADD COLUMN IF NOT EXISTS mapping_ordinal integer;
ALTER TABLE regulatory_generated_mapping_snapshots DISABLE TRIGGER regulatory_generated_mapping_snapshots_append_only;
ALTER TABLE regulatory_generated_mapping_snapshots
	DROP CONSTRAINT IF EXISTS regulatory_generated_mapping_snapshots_candidate_ordinal_unique;
DO $mapping_order_repair$
DECLARE
	candidate_record record;
	predecessor_max_ordinal integer;
	missing_ordinal_count integer;
	recomputed_digest text;
BEGIN
	FOR candidate_record IN
		WITH RECURSIVE candidate_chain AS (
			SELECT candidate.id,candidate.candidate_root_id,
			       candidate.supersedes_candidate_id,candidate.generation_run_id,
			       candidate.template_id,candidate.revision,
			       candidate.candidate_content_digest,0 AS depth
			FROM template_draft_versions candidate
			WHERE candidate.generation_run_id IS NOT NULL
			  AND candidate.supersedes_candidate_id IS NULL
			UNION ALL
			SELECT successor.id,successor.candidate_root_id,
			       successor.supersedes_candidate_id,successor.generation_run_id,
			       successor.template_id,successor.revision,
			       successor.candidate_content_digest,chain.depth+1
			FROM template_draft_versions successor
			JOIN candidate_chain chain
			  ON successor.supersedes_candidate_id=chain.id
		)
		SELECT * FROM candidate_chain
		ORDER BY candidate_root_id,depth,revision,id
	LOOP
		IF candidate_record.supersedes_candidate_id IS NULL THEN
			UPDATE regulatory_generated_mapping_snapshots mapping
			SET mapping_ordinal=ordered.ordinality-1
			FROM regulatory_generation_runs run
			CROSS JOIN LATERAL
				jsonb_array_elements(run.output_artifact->'complianceMappings')
				WITH ORDINALITY AS ordered(snapshot,ordinality)
			WHERE run.id=candidate_record.generation_run_id
			  AND mapping.candidate_draft_version_id=candidate_record.id
			  AND ordered.snapshot->>'mappingId'=mapping.mapping_id;
		ELSE
			UPDATE regulatory_generated_mapping_snapshots mapping
			SET mapping_ordinal=predecessor.mapping_ordinal
			FROM regulatory_generated_mapping_snapshots predecessor
			WHERE mapping.candidate_draft_version_id=candidate_record.id
			  AND predecessor.candidate_draft_version_id=
			      candidate_record.supersedes_candidate_id
			  AND predecessor.mapping_id=mapping.mapping_id;

			SELECT COALESCE(MAX(mapping_ordinal),-1)
			INTO predecessor_max_ordinal
			FROM regulatory_generated_mapping_snapshots
			WHERE candidate_draft_version_id=
			      candidate_record.supersedes_candidate_id;

			WITH first_ordered_reference AS (
				SELECT DISTINCT ON (mapping.mapping_id)
				       mapping.mapping_id,
				       ordered_question.ordinality AS question_ordinal,
				       mapping_reference.ordinality AS reference_ordinal
				FROM template_draft_versions candidate
				CROSS JOIN LATERAL
					unnest(candidate.question_version_ids) WITH ORDINALITY
					AS ordered_question(question_version_id,ordinality)
				JOIN question_versions question
				  ON question.id=ordered_question.question_version_id
				JOIN regulatory_generated_question_snapshots question_snapshot
				  ON question_snapshot.candidate_draft_version_id=candidate.id
				 AND question_snapshot.question_id=question.question_id
				CROSS JOIN LATERAL
					jsonb_array_elements_text(
						question_snapshot.snapshot->'mappingIds'
					) WITH ORDINALITY
					AS mapping_reference(mapping_id,ordinality)
				JOIN regulatory_generated_mapping_snapshots mapping
				  ON mapping.candidate_draft_version_id=candidate.id
				 AND mapping.mapping_id=mapping_reference.mapping_id
				WHERE candidate.id=candidate_record.id
				  AND NOT EXISTS (
					SELECT 1
					FROM regulatory_generated_mapping_snapshots predecessor
					WHERE predecessor.candidate_draft_version_id=
					      candidate_record.supersedes_candidate_id
					  AND predecessor.mapping_id=mapping.mapping_id
				  )
				ORDER BY mapping.mapping_id,ordered_question.ordinality,
				         mapping_reference.ordinality
			),
			ordered_new_mapping AS (
				SELECT mapping_id,
				       row_number() OVER (
						ORDER BY question_ordinal,reference_ordinal,mapping_id
				       )-1 AS append_ordinal
				FROM first_ordered_reference
			)
			UPDATE regulatory_generated_mapping_snapshots mapping
			SET mapping_ordinal=
			    predecessor_max_ordinal+1+ordered.append_ordinal
			FROM ordered_new_mapping ordered
			WHERE mapping.candidate_draft_version_id=candidate_record.id
			  AND mapping.mapping_id=ordered.mapping_id;
		END IF;

		SELECT COUNT(*) INTO missing_ordinal_count
		FROM regulatory_generated_mapping_snapshots
		WHERE candidate_draft_version_id=candidate_record.id
		  AND mapping_ordinal IS NULL;
		IF missing_ordinal_count<>0 THEN
			RAISE EXCEPTION
				'cannot recover exact mapping order for governed candidate %',
				candidate_record.id;
		END IF;

		IF candidate_record.supersedes_candidate_id IS NULL THEN
			SELECT run.output_digest INTO recomputed_digest
			FROM regulatory_generation_runs run
			WHERE run.id=candidate_record.generation_run_id;
		ELSE
			SELECT governed_jsonb_sha256(
				jsonb_build_object(
					'complianceMappings',
					(
						SELECT jsonb_agg(
							mapping.snapshot ORDER BY mapping.mapping_ordinal
						)
						FROM regulatory_generated_mapping_snapshots mapping
						WHERE mapping.candidate_draft_version_id=candidate_record.id
					),
					'inspectionChecklist',
					jsonb_build_object(
						'checklistId',candidate_record.template_id,
						'questions',
						(
							SELECT jsonb_agg(
								question_snapshot.snapshot
								ORDER BY ordered_question.ordinality
							)
							FROM template_draft_versions candidate
							CROSS JOIN LATERAL
								unnest(candidate.question_version_ids) WITH ORDINALITY
								AS ordered_question(question_version_id,ordinality)
							JOIN question_versions question
							  ON question.id=ordered_question.question_version_id
							JOIN regulatory_generated_question_snapshots question_snapshot
							  ON question_snapshot.candidate_draft_version_id=candidate.id
							 AND question_snapshot.question_id=question.question_id
							WHERE candidate.id=candidate_record.id
						)
					)
				)
			) INTO recomputed_digest;
		END IF;
		IF recomputed_digest IS DISTINCT FROM
		   candidate_record.candidate_content_digest THEN
			RAISE EXCEPTION
				'repaired governed candidate % digest % does not match stored digest %',
				candidate_record.id,recomputed_digest,
				candidate_record.candidate_content_digest;
		END IF;
	END LOOP;
END
$mapping_order_repair$;
ALTER TABLE regulatory_generated_mapping_snapshots ENABLE TRIGGER regulatory_generated_mapping_snapshots_append_only;
ALTER TABLE regulatory_generated_mapping_snapshots ALTER COLUMN mapping_ordinal SET NOT NULL;
ALTER TABLE regulatory_generated_mapping_snapshots DROP CONSTRAINT IF EXISTS regulatory_generated_mapping_snapshots_mapping_ordinal_check;
ALTER TABLE regulatory_generated_mapping_snapshots ADD CONSTRAINT regulatory_generated_mapping_snapshots_mapping_ordinal_check CHECK (mapping_ordinal >= 0);
DO $repair$
BEGIN
	IF NOT EXISTS (
		SELECT 1 FROM pg_constraint
		WHERE conrelid='regulatory_generated_mapping_snapshots'::regclass
		  AND contype='u'
		  AND pg_get_constraintdef(oid)='UNIQUE (candidate_draft_version_id, mapping_ordinal)'
	) THEN
		ALTER TABLE regulatory_generated_mapping_snapshots
			ADD CONSTRAINT regulatory_generated_mapping_snapshots_candidate_ordinal_unique
			UNIQUE (candidate_draft_version_id, mapping_ordinal);
	END IF;
END
$repair$;
ALTER TABLE department_review_decisions ADD COLUMN IF NOT EXISTS candidate_root_id text;
ALTER TABLE department_review_decisions DISABLE TRIGGER department_review_decisions_append_only;
UPDATE department_review_decisions decision
SET candidate_root_id=candidate.candidate_root_id
FROM template_draft_versions candidate
WHERE candidate.id=decision.candidate_draft_version_id
  AND decision.candidate_root_id IS NULL;
ALTER TABLE department_review_decisions ENABLE TRIGGER department_review_decisions_append_only;
ALTER TABLE department_review_decisions ALTER COLUMN candidate_root_id SET NOT NULL;
ALTER TABLE checklist_publication_decisions ADD COLUMN IF NOT EXISTS candidate_root_id text;
ALTER TABLE checklist_publication_decisions DISABLE TRIGGER checklist_publication_decisions_append_only;
UPDATE checklist_publication_decisions decision
SET candidate_root_id=candidate.candidate_root_id
FROM template_draft_versions candidate
WHERE candidate.id=decision.candidate_draft_version_id
  AND decision.candidate_root_id IS NULL;
ALTER TABLE checklist_publication_decisions ENABLE TRIGGER checklist_publication_decisions_append_only;
ALTER TABLE checklist_publication_decisions ALTER COLUMN candidate_root_id SET NOT NULL;
ALTER TABLE department_review_decisions ALTER COLUMN decided_at DROP DEFAULT;
ALTER TABLE department_review_decisions
	DROP CONSTRAINT IF EXISTS department_review_decisions_reason_check;
ALTER TABLE department_review_decisions
	ADD CONSTRAINT department_review_decisions_reason_check
	CHECK (btrim(reason) <> '');
DO $repair$
BEGIN
	IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid='department_review_decisions'::regclass AND contype='f' AND pg_get_constraintdef(oid) LIKE 'FOREIGN KEY (candidate_root_id)%') THEN
		ALTER TABLE department_review_decisions ADD CONSTRAINT department_review_decisions_candidate_root_fkey FOREIGN KEY (candidate_root_id) REFERENCES template_draft_versions(id);
	END IF;
	IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid='checklist_publication_decisions'::regclass AND contype='f' AND pg_get_constraintdef(oid) LIKE 'FOREIGN KEY (candidate_root_id)%') THEN
		ALTER TABLE checklist_publication_decisions ADD CONSTRAINT checklist_publication_decisions_candidate_root_fkey FOREIGN KEY (candidate_root_id) REFERENCES template_draft_versions(id);
	END IF;
END
$repair$;
CREATE OR REPLACE FUNCTION validate_governed_decision_actor() RETURNS trigger LANGUAGE plpgsql AS $guard$
DECLARE membership record; effective_membership record; department_status text; unit_status text;
BEGIN
	IF NEW.candidate_root_id IS NULL THEN
		SELECT candidate_root_id INTO NEW.candidate_root_id
		FROM template_draft_versions
		WHERE id=NEW.candidate_draft_version_id
		  AND revision=NEW.candidate_revision
		  AND candidate_content_digest=NEW.candidate_content_digest;
	END IF;
	SELECT * INTO membership
	FROM caa_department_memberships
	WHERE id=NEW.actor_department_membership_id;
	SELECT * INTO effective_membership
	FROM caa_department_memberships fact
	WHERE fact.root_id=membership.root_id
	  AND fact.effective_from<=NEW.decided_at::date
	ORDER BY fact.effective_from DESC,fact.id DESC
	LIMIT 1;
	SELECT status INTO department_status
	FROM caa_department_status_facts fact
	WHERE fact.department_id=NEW.actor_department_id
	  AND fact.effective_from<=NEW.decided_at::date
	ORDER BY fact.effective_from DESC,fact.id DESC
	LIMIT 1;
	SELECT status INTO unit_status
	FROM caa_organizational_unit_status_facts fact
	WHERE fact.organizational_unit_id=NEW.actor_organizational_unit_id
	  AND fact.effective_from<=NEW.decided_at::date
	ORDER BY fact.effective_from DESC,fact.id DESC
	LIMIT 1;
	IF membership.id IS NULL
	   OR effective_membership.id IS DISTINCT FROM membership.id
	   OR membership.subject_id IS DISTINCT FROM NEW.actor_subject_id
	   OR membership.department_id IS DISTINCT FROM NEW.actor_department_id
	   OR membership.organizational_unit_id IS DISTINCT FROM NEW.actor_organizational_unit_id
	   OR membership.membership_role IS DISTINCT FROM 'DEPARTMENT_MANAGER'
	   OR membership.status IS DISTINCT FROM 'ACTIVE'
	   OR department_status IS DISTINCT FROM 'ACTIVE'
	   OR unit_status IS DISTINCT FROM 'ACTIVE'
	   OR membership.effective_from>NEW.decided_at::date
	   OR (membership.effective_to IS NOT NULL AND membership.effective_to<=NEW.decided_at::date)
	THEN
		RAISE EXCEPTION 'decision actor has no current matching Department Manager assignment';
	END IF;
	RETURN NEW;
END;
$guard$;
CREATE OR REPLACE FUNCTION validate_governed_publication_approval() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM candidate_required_owner_assignments WHERE candidate_draft_version_id = NEW.candidate_draft_version_id AND candidate_revision = NEW.candidate_revision AND candidate_content_digest = NEW.candidate_content_digest AND approval_required) OR EXISTS (
        SELECT 1 FROM candidate_required_owner_assignments owner WHERE owner.candidate_draft_version_id = NEW.candidate_draft_version_id AND owner.candidate_revision = NEW.candidate_revision AND owner.candidate_content_digest = NEW.candidate_content_digest AND owner.approval_required AND NOT EXISTS (
            SELECT 1 FROM department_review_decisions review WHERE review.candidate_draft_version_id = owner.candidate_draft_version_id AND review.candidate_revision = owner.candidate_revision AND review.candidate_content_digest = owner.candidate_content_digest AND review.decision = 'TECHNICALLY_APPROVED' AND review.actor_department_id = owner.department_id AND review.actor_organizational_unit_id = owner.organizational_unit_id
        )
    ) THEN RAISE EXCEPTION 'publication requires all required technical approvals for the exact candidate digest'; END IF;
    RETURN NEW;
END;
$$;
DROP TRIGGER IF EXISTS department_review_decisions_actor_guard ON department_review_decisions;
CREATE TRIGGER department_review_decisions_actor_guard BEFORE INSERT ON department_review_decisions FOR EACH ROW EXECUTE FUNCTION validate_governed_decision_actor();
DROP TRIGGER IF EXISTS checklist_publication_decisions_actor_guard ON checklist_publication_decisions;
CREATE TRIGGER checklist_publication_decisions_actor_guard BEFORE INSERT ON checklist_publication_decisions FOR EACH ROW EXECUTE FUNCTION validate_governed_decision_actor();
DROP TRIGGER IF EXISTS checklist_publication_decisions_approval_guard ON checklist_publication_decisions;
CREATE TRIGGER checklist_publication_decisions_approval_guard BEFORE INSERT ON checklist_publication_decisions FOR EACH ROW EXECUTE FUNCTION validate_governed_publication_approval();
DROP INDEX IF EXISTS candidate_required_owner_assignments_review_queue_idx;
CREATE INDEX candidate_required_owner_assignments_review_queue_idx
	ON candidate_required_owner_assignments
	(department_id, organizational_unit_id, candidate_draft_version_id, candidate_revision, candidate_content_digest)
	WHERE approval_required;
DROP INDEX IF EXISTS department_review_decisions_candidate_idx;
CREATE INDEX department_review_decisions_candidate_idx
	ON department_review_decisions
	(candidate_draft_version_id, candidate_revision, candidate_content_digest, decided_at, id);
DROP INDEX IF EXISTS department_review_decisions_exact_owner_approval_idx;
CREATE UNIQUE INDEX department_review_decisions_exact_owner_approval_idx
	ON department_review_decisions
	(candidate_draft_version_id, candidate_revision, candidate_content_digest,
	 actor_department_id, actor_organizational_unit_id)
	WHERE decision='TECHNICALLY_APPROVED';
DROP INDEX IF EXISTS checklist_publication_decisions_candidate_unique_idx;
CREATE UNIQUE INDEX checklist_publication_decisions_candidate_unique_idx
	ON checklist_publication_decisions
	(candidate_draft_version_id, candidate_revision, candidate_content_digest);
DROP INDEX IF EXISTS checklist_template_versions_governed_candidate_unique_idx;
CREATE UNIQUE INDEX checklist_template_versions_governed_candidate_unique_idx
	ON checklist_template_versions
	(candidate_draft_version_id, candidate_revision, candidate_content_digest)
	WHERE candidate_draft_version_id IS NOT NULL;
DROP INDEX IF EXISTS template_draft_versions_governed_review_queue_idx;
CREATE INDEX template_draft_versions_governed_review_queue_idx
	ON template_draft_versions (status, id)
	WHERE generation_run_id IS NOT NULL;

ALTER TABLE template_draft_versions DROP CONSTRAINT IF EXISTS template_draft_versions_status_check;
ALTER TABLE template_draft_versions ADD CONSTRAINT template_draft_versions_status_check
	CHECK (status IN ('DRAFT','GENERATED_DRAFT','DEPARTMENT_REVIEW','RETURNED','REJECTED','TECHNICALLY_APPROVED','PUBLISHED'));

CREATE OR REPLACE FUNCTION validate_governed_generated_candidate() RETURNS trigger LANGUAGE plpgsql AS $guard$
DECLARE parent record;
BEGIN
    IF NEW.generation_run_id IS NULL THEN RETURN NEW; END IF;
    IF cardinality(NEW.question_version_ids) = 0 OR array_position(NEW.question_version_ids, '') IS NOT NULL OR EXISTS (SELECT 1 FROM unnest(NEW.question_version_ids) question_version_id WHERE NOT EXISTS (SELECT 1 FROM question_versions question WHERE question.id = question_version_id)) THEN RAISE EXCEPTION 'generated candidate requires nonempty immutable question-version identities'; END IF;
    IF NOT EXISTS (SELECT 1 FROM regulatory_generation_runs run WHERE run.id = NEW.generation_run_id AND run.status = 'GENERATED' AND run.output_artifact IS NOT NULL) OR NOT EXISTS (SELECT 1 FROM regulatory_generation_run_scope_facts WHERE generation_run_id = NEW.generation_run_id) OR NOT EXISTS (SELECT 1 FROM regulatory_generation_run_source_snapshots WHERE generation_run_id = NEW.generation_run_id) THEN RAISE EXCEPTION 'generated candidate requires complete exact generation lineage'; END IF;
    IF NEW.supersedes_candidate_id IS NULL THEN
        IF NEW.candidate_root_id <> NEW.id OR NOT EXISTS (SELECT 1 FROM regulatory_generation_runs run WHERE run.id = NEW.generation_run_id AND run.output_digest = NEW.candidate_content_digest) THEN RAISE EXCEPTION 'generated candidate root must pin its exact generated output digest'; END IF;
    ELSE
        SELECT * INTO parent FROM template_draft_versions WHERE id = NEW.supersedes_candidate_id;
        IF parent.id IS NULL OR parent.generation_run_id IS NULL OR parent.template_id <> NEW.template_id OR parent.candidate_root_id <> NEW.candidate_root_id OR parent.generation_run_id <> NEW.generation_run_id OR NEW.version <= parent.version OR NEW.revision <= parent.revision THEN RAISE EXCEPTION 'generated candidate successor must form one increasing immutable revision chain'; END IF;
    END IF;
    RETURN NEW;
END;
$guard$;
DROP TRIGGER IF EXISTS template_draft_versions_generated_lineage_guard ON template_draft_versions;
CREATE TRIGGER template_draft_versions_generated_lineage_guard BEFORE INSERT ON template_draft_versions FOR EACH ROW EXECUTE FUNCTION validate_governed_generated_candidate();

CREATE OR REPLACE FUNCTION validate_governed_generation_crosswalk_partition() RETURNS trigger LANGUAGE plpgsql AS $guard$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM regulatory_evaluation_partition_rows row JOIN regulatory_evaluation_partitions partition ON partition.id=row.partition_id WHERE row.partition_id=NEW.evaluation_partition_id AND row.state_compliance_crosswalk_row_id=NEW.state_compliance_crosswalk_row_id AND row.stable_row_identity=NEW.stable_row_identity AND partition.partition_kind='GENERATION_INPUT') THEN
        RAISE EXCEPTION 'generation run crosswalk lineage requires one exact generation-input partition row';
    END IF;
    RETURN NEW;
END;
$guard$;
DROP TRIGGER IF EXISTS regulatory_generation_run_crosswalk_partition_rows_guard ON regulatory_generation_run_crosswalk_partition_rows;
CREATE TRIGGER regulatory_generation_run_crosswalk_partition_rows_guard BEFORE INSERT ON regulatory_generation_run_crosswalk_partition_rows FOR EACH ROW EXECUTE FUNCTION validate_governed_generation_crosswalk_partition();

CREATE OR REPLACE FUNCTION governed_generated_candidate_immutable_guard() RETURNS trigger LANGUAGE plpgsql AS $guard$
BEGIN
	IF OLD.generation_run_id IS NULL THEN RETURN NEW; END IF;
	IF (OLD.status, NEW.status) IN (
	       ('GENERATED_DRAFT', 'DEPARTMENT_REVIEW'),
	       ('DEPARTMENT_REVIEW', 'RETURNED'),
	       ('DEPARTMENT_REVIEW', 'REJECTED'),
	       ('DEPARTMENT_REVIEW', 'TECHNICALLY_APPROVED'),
	       ('TECHNICALLY_APPROVED', 'PUBLISHED')
	   )
	   AND NEW.id = OLD.id AND NEW.template_id = OLD.template_id AND NEW.version = OLD.version
	   AND NEW.owner_role = OLD.owner_role AND NEW.creator_subject_id = OLD.creator_subject_id
	   AND NEW.change_reason = OLD.change_reason AND NEW.question_version_ids = OLD.question_version_ids
	   AND NEW.revision = OLD.revision AND NEW.generation_run_id = OLD.generation_run_id
	   AND NEW.candidate_content_digest = OLD.candidate_content_digest
	   AND NEW.candidate_schema_version = OLD.candidate_schema_version
	   AND NEW.candidate_root_id = OLD.candidate_root_id
	   AND NEW.supersedes_candidate_id IS NOT DISTINCT FROM OLD.supersedes_candidate_id
	THEN RETURN NEW; END IF;
	RAISE EXCEPTION 'generated candidate revisions are immutable except governed status transitions';
END;
$guard$;
DROP TRIGGER IF EXISTS template_draft_versions_generated_immutable ON template_draft_versions;
CREATE TRIGGER template_draft_versions_generated_immutable BEFORE UPDATE OR DELETE ON template_draft_versions FOR EACH ROW EXECUTE FUNCTION governed_generated_candidate_immutable_guard();
DROP TRIGGER IF EXISTS governed_candidate_commands_append_only ON governed_candidate_commands;
CREATE TRIGGER governed_candidate_commands_append_only BEFORE UPDATE OR DELETE ON governed_candidate_commands FOR EACH ROW EXECUTE FUNCTION governed_append_only_guard();
DROP TRIGGER IF EXISTS regulatory_source_gap_facts_append_only ON regulatory_source_gap_facts;
CREATE TRIGGER regulatory_source_gap_facts_append_only BEFORE UPDATE OR DELETE ON regulatory_source_gap_facts FOR EACH ROW EXECUTE FUNCTION governed_append_only_guard();

CREATE TABLE IF NOT EXISTS canonical_audit_preparation_edit_previews (
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
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX IF NOT EXISTS canonical_preparation_edit_previews_operation_key
    ON canonical_audit_preparation_edit_previews (assignment_id, edit_kind, operation_id);
CREATE UNIQUE INDEX IF NOT EXISTS canonical_preparation_edit_previews_idempotency_key
    ON canonical_audit_preparation_edit_previews (assignment_id, edit_kind, idempotency_key);
CREATE TABLE IF NOT EXISTS canonical_audit_preparation_edit_preview_consumptions (
    preview_id text PRIMARY KEY REFERENCES canonical_audit_preparation_edit_previews(id),
    consumed_by_subject_id text NOT NULL REFERENCES identity_references(subject_id),
    consumed_at timestamptz NOT NULL DEFAULT now()
);
DROP TRIGGER IF EXISTS canonical_preparation_edit_previews_append_only ON canonical_audit_preparation_edit_previews;
CREATE TRIGGER canonical_preparation_edit_previews_append_only BEFORE UPDATE OR DELETE ON canonical_audit_preparation_edit_previews FOR EACH ROW EXECUTE FUNCTION reject_immutable_row_change();
DROP TRIGGER IF EXISTS canonical_preparation_edit_preview_consumptions_append_only ON canonical_audit_preparation_edit_preview_consumptions;
CREATE TRIGGER canonical_preparation_edit_preview_consumptions_append_only BEFORE UPDATE OR DELETE ON canonical_audit_preparation_edit_preview_consumptions FOR EACH ROW EXECUTE FUNCTION reject_immutable_row_change();
CREATE INDEX IF NOT EXISTS canonical_audit_preparation_edit_previews_expiry_idx
    ON canonical_audit_preparation_edit_previews (assignment_id, expires_at);
`

func normalizeIndexDefinition(definition string) string {
	return strings.Join(strings.Fields(definition), " ")
}

func load() ([]migration, error) {
	entries, err := fs.ReadDir(migrationFiles, ".")
	if err != nil {
		return nil, fmt.Errorf("read embedded migrations: %w", err)
	}
	loaded := make([]migration, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".up.sql") {
			continue
		}
		prefix, _, ok := strings.Cut(filepath.Base(entry.Name()), "_")
		if !ok {
			return nil, fmt.Errorf("migration %s has no numeric prefix", entry.Name())
		}
		version, err := strconv.ParseInt(prefix, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("parse migration version %s: %w", entry.Name(), err)
		}
		contents, err := migrationFiles.ReadFile(entry.Name())
		if err != nil {
			return nil, fmt.Errorf("read migration %s: %w", entry.Name(), err)
		}
		loaded = append(loaded, migration{version: version, name: entry.Name(), sql: string(contents)})
	}
	sort.Slice(loaded, func(left, right int) bool { return loaded[left].version < loaded[right].version })
	if len(loaded) == 0 || loaded[len(loaded)-1].version != LatestVersion {
		return nil, fmt.Errorf("embedded migration set does not end at version %d", LatestVersion)
	}
	for index, candidate := range loaded {
		expected := int64(index + 1)
		if candidate.version != expected {
			return nil, fmt.Errorf("migration sequence has version %d, expected %d", candidate.version, expected)
		}
	}
	return loaded, nil
}
