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

	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/platform/database"
)

const LatestVersion int64 = 28
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
