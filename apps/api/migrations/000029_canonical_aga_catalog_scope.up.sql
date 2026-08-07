-- Canonical AGA catalog/scope foundation.  This migration deliberately adds
-- lineage, membership, review, scope, and preparation records only.  The
-- immutable question body/version authority remains question_versions.

CREATE TABLE canonical_question_catalogs (
    id text PRIMARY KEY,
    catalog_version text NOT NULL UNIQUE,
    usage_class text NOT NULL CHECK (usage_class IN ('GOVERNED_OPERATIONAL', 'PREPROD_EXERCISE')),
    profile_name text NOT NULL,
    profile_version text NOT NULL,
    status text NOT NULL CHECK (status IN ('IMPORTING', 'SEALED', 'RETIRED')),
    source_package_version text NOT NULL,
    source_package_json_sha256 text NOT NULL CHECK (btrim(source_package_json_sha256) <> ''),
    source_package_zip_sha256 text NOT NULL CHECK (btrim(source_package_zip_sha256) <> ''),
    root_digest text NOT NULL CHECK (btrim(root_digest) <> ''),
    question_count integer NOT NULL CHECK (question_count >= 0),
    form_count integer NOT NULL CHECK (form_count >= 0),
    created_by_subject_id text NOT NULL REFERENCES identity_references(subject_id),
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (id, usage_class)
);

CREATE TABLE canonical_question_catalog_import_runs (
    id text PRIMARY KEY,
    catalog_id text NOT NULL REFERENCES canonical_question_catalogs(id),
    operation_id text NOT NULL UNIQUE,
    idempotency_key text NOT NULL UNIQUE,
    package_zip_sha256 text NOT NULL,
    package_json_sha256 text NOT NULL,
    import_digest text NOT NULL,
    row_count integer NOT NULL CHECK (row_count >= 0),
    form_count integer NOT NULL CHECK (form_count >= 0),
    status text NOT NULL CHECK (status IN ('STARTED', 'SEALED', 'FAILED')),
    failure_reason text,
    actor_subject_id text NOT NULL REFERENCES identity_references(subject_id),
    audit_event_id text REFERENCES audit_events(event_id),
    expires_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now()
);

-- Form boundaries are first-class catalog lineage so zero-question forms are
-- still readable and cannot disappear when question memberships are queried.
CREATE TABLE canonical_question_catalog_forms (
    catalog_id text NOT NULL REFERENCES canonical_question_catalogs(id),
    form_code text NOT NULL CHECK (btrim(form_code) <> ''),
    form_digest text NOT NULL CHECK (btrim(form_digest) <> ''),
    archive_digest text,
    question_count integer NOT NULL CHECK (question_count >= 0),
    source_gap_state text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (catalog_id, form_code)
);

CREATE TRIGGER canonical_question_catalog_forms_append_only
BEFORE UPDATE OR DELETE ON canonical_question_catalog_forms
FOR EACH ROW EXECUTE FUNCTION reject_immutable_row_change();

CREATE TABLE canonical_question_catalog_memberships (
    catalog_id text NOT NULL REFERENCES canonical_question_catalogs(id),
    question_version_id text NOT NULL REFERENCES question_versions(id),
    usage_class text NOT NULL CHECK (usage_class IN ('GOVERNED_OPERATIONAL', 'PREPROD_EXERCISE')),
    form_code text NOT NULL CHECK (btrim(form_code) <> ''),
    proposal_id text NOT NULL CHECK (btrim(proposal_id) <> ''),
    ordinal integer NOT NULL CHECK (ordinal > 0),
    question_digest text NOT NULL CHECK (btrim(question_digest) <> ''),
    source_locator text,
    source_gap_state text NOT NULL,
    proposed_domain text,
    proposed_topic text,
    proposed_risk_band text,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (catalog_id, question_version_id),
    UNIQUE (catalog_id, form_code, proposal_id, ordinal),
    UNIQUE (catalog_id, question_version_id, usage_class)
);

CREATE TABLE canonical_question_catalog_membership_events (
    event_id text PRIMARY KEY,
    catalog_id text NOT NULL REFERENCES canonical_question_catalogs(id),
    question_version_id text NOT NULL REFERENCES question_versions(id),
    status text NOT NULL CHECK (status IN ('AVAILABLE', 'DEPRECATED')),
    reason text NOT NULL CHECK (btrim(reason) <> ''),
    actor_subject_id text NOT NULL REFERENCES identity_references(subject_id),
    audit_event_id text REFERENCES audit_events(event_id),
    occurred_at timestamptz NOT NULL DEFAULT now(),
    FOREIGN KEY (catalog_id, question_version_id)
        REFERENCES canonical_question_catalog_memberships(catalog_id, question_version_id)
);

CREATE INDEX canonical_question_catalog_membership_search_idx
    ON canonical_question_catalog_memberships (catalog_id, form_code, proposed_domain, proposed_topic, ordinal);

CREATE TABLE canonical_exercise_question_review_drafts (
    id text PRIMARY KEY,
    catalog_id text NOT NULL REFERENCES canonical_question_catalogs(id),
    question_version_id text NOT NULL REFERENCES question_versions(id),
    usage_class text NOT NULL CHECK (usage_class = 'PREPROD_EXERCISE'),
    revision bigint NOT NULL CHECK (revision > 0),
    disposition text NOT NULL CHECK (disposition IN ('RETAIN', 'INCLUDE', 'EXCLUDE', 'DEFER')),
    reviewed_domain text,
    reviewed_topic text,
    controlled_reason text NOT NULL CHECK (btrim(controlled_reason) <> ''),
    actor_subject_id text NOT NULL REFERENCES identity_references(subject_id),
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (catalog_id, question_version_id, revision),
    FOREIGN KEY (catalog_id, question_version_id, usage_class)
        REFERENCES canonical_question_catalog_memberships(catalog_id, question_version_id, usage_class)
);

CREATE TABLE canonical_exercise_question_review_events (
    event_id text PRIMARY KEY,
    draft_id text NOT NULL REFERENCES canonical_exercise_question_review_drafts(id),
    action text NOT NULL CHECK (action IN ('RETAIN', 'INCLUDE', 'EXCLUDE', 'DEFER', 'DOMAIN_RECLASSIFIED', 'TOPIC_RECLASSIFIED')),
    payload jsonb NOT NULL CHECK (jsonb_typeof(payload) = 'object'),
    actor_subject_id text NOT NULL REFERENCES identity_references(subject_id),
    audit_event_id text REFERENCES audit_events(event_id),
    occurred_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE canonical_audit_scope_drafts (
    id text PRIMARY KEY,
    planning_intake_draft_id text NOT NULL REFERENCES planning_intake_drafts(id),
    organization_id text NOT NULL REFERENCES organizations(id),
    provider_scope_id text NOT NULL REFERENCES organization_service_provider_scopes(id),
    regulated_target_id text NOT NULL REFERENCES regulated_targets(id),
    audit_type text NOT NULL,
    catalog_id text NOT NULL REFERENCES canonical_question_catalogs(id),
    usage_class text NOT NULL CHECK (usage_class IN ('GOVERNED_OPERATIONAL', 'PREPROD_EXERCISE')),
    revision bigint NOT NULL DEFAULT 1 CHECK (revision > 0),
    status text NOT NULL CHECK (status IN ('DRAFT', 'SUBMITTED', 'RELEASED', 'ABANDONED')),
    selected_question_count integer NOT NULL DEFAULT 0 CHECK (selected_question_count >= 0),
    selection_digest text NOT NULL DEFAULT '',
    requested_budget numeric(14, 2) NOT NULL DEFAULT 0 CHECK (requested_budget >= 0),
    notice_policy text NOT NULL,
    created_by_subject_id text NOT NULL REFERENCES identity_references(subject_id),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (id, revision),
    FOREIGN KEY (catalog_id, usage_class)
        REFERENCES canonical_question_catalogs(id, usage_class)
);

CREATE TABLE canonical_audit_scope_selection_operations (
    id text PRIMARY KEY,
    scope_draft_id text NOT NULL REFERENCES canonical_audit_scope_drafts(id),
    operation_id text NOT NULL UNIQUE,
    idempotency_key text NOT NULL UNIQUE,
    operation_kind text NOT NULL CHECK (operation_kind IN ('PREVIEW', 'ADD', 'REMOVE', 'REPLACE', 'SUBMIT')),
    expected_digest text NOT NULL,
    result_digest text NOT NULL,
    affected_question_version_ids jsonb NOT NULL CHECK (jsonb_typeof(affected_question_version_ids) = 'array'),
    filter_payload jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(filter_payload) = 'object'),
    actor_subject_id text NOT NULL REFERENCES identity_references(subject_id),
    audit_event_id text REFERENCES audit_events(event_id),
    created_at timestamptz NOT NULL DEFAULT now()
);

-- The JSON array on the operation is a receipt for clients. This relation is
-- the durable authority for selected identities and keeps every question FK
-- bound to the exact catalog membership without mutating a draft row.
CREATE TABLE canonical_audit_scope_selection_questions (
    operation_id text NOT NULL REFERENCES canonical_audit_scope_selection_operations(id),
    catalog_id text NOT NULL,
    question_version_id text NOT NULL REFERENCES question_versions(id),
    position integer NOT NULL CHECK (position >= 0),
    selection_digest text NOT NULL CHECK (btrim(selection_digest) <> ''),
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (operation_id, question_version_id),
    UNIQUE (operation_id, position),
    FOREIGN KEY (catalog_id, question_version_id)
        REFERENCES canonical_question_catalog_memberships(catalog_id, question_version_id)
);

CREATE TABLE canonical_audit_scope_draft_questions (
    scope_draft_id text NOT NULL,
    revision bigint NOT NULL CHECK (revision > 0),
    catalog_id text NOT NULL,
    question_version_id text NOT NULL REFERENCES question_versions(id),
    position integer NOT NULL CHECK (position >= 0),
    selection_digest text NOT NULL CHECK (btrim(selection_digest) <> ''),
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (scope_draft_id, revision, question_version_id),
    UNIQUE (scope_draft_id, revision, position),
    FOREIGN KEY (scope_draft_id, revision)
        REFERENCES canonical_audit_scope_drafts(id, revision),
    FOREIGN KEY (catalog_id, question_version_id)
        REFERENCES canonical_question_catalog_memberships(catalog_id, question_version_id)
);

CREATE TABLE canonical_audit_scope_snapshots (
    id text PRIMARY KEY,
    scope_draft_id text NOT NULL REFERENCES canonical_audit_scope_drafts(id),
    revision bigint NOT NULL CHECK (revision > 0),
    stage text NOT NULL CHECK (stage IN ('SUBMITTED', 'RELEASED')),
    catalog_id text NOT NULL REFERENCES canonical_question_catalogs(id),
    usage_class text NOT NULL CHECK (usage_class IN ('GOVERNED_OPERATIONAL', 'PREPROD_EXERCISE')),
    selection_digest text NOT NULL CHECK (btrim(selection_digest) <> ''),
    selected_question_count integer NOT NULL CHECK (selected_question_count > 0),
    snapshot jsonb NOT NULL CHECK (jsonb_typeof(snapshot) = 'object'),
    created_by_subject_id text NOT NULL REFERENCES identity_references(subject_id),
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (scope_draft_id, stage, revision),
    UNIQUE (id, catalog_id),
    FOREIGN KEY (catalog_id, usage_class)
        REFERENCES canonical_question_catalogs(id, usage_class)
);

CREATE TABLE canonical_audit_scope_snapshot_questions (
    snapshot_id text NOT NULL REFERENCES canonical_audit_scope_snapshots(id),
    catalog_id text NOT NULL,
    question_version_id text NOT NULL REFERENCES question_versions(id),
    position integer NOT NULL CHECK (position >= 0),
    PRIMARY KEY (snapshot_id, question_version_id),
    UNIQUE (snapshot_id, position),
    FOREIGN KEY (snapshot_id, catalog_id)
        REFERENCES canonical_audit_scope_snapshots(id, catalog_id),
    FOREIGN KEY (catalog_id, question_version_id)
        REFERENCES canonical_question_catalog_memberships(catalog_id, question_version_id)
);

CREATE TABLE canonical_audit_preparation_snapshots (
    id text PRIMARY KEY,
    released_scope_snapshot_id text NOT NULL REFERENCES canonical_audit_scope_snapshots(id),
    lead_subject_id text NOT NULL REFERENCES identity_references(subject_id),
    revision bigint NOT NULL CHECK (revision > 0),
    status text NOT NULL CHECK (status IN ('DRAFT', 'CONFIRMED', 'MATERIALIZED')),
    preparation_digest text NOT NULL CHECK (btrim(preparation_digest) <> ''),
    confirmed_by_subject_id text REFERENCES identity_references(subject_id),
    confirmed_at timestamptz,
    snapshot jsonb NOT NULL CHECK (jsonb_typeof(snapshot) = 'object'),
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (released_scope_snapshot_id, revision),
    UNIQUE (id, released_scope_snapshot_id)
);

CREATE TABLE canonical_audit_preparation_questions (
    preparation_id text NOT NULL REFERENCES canonical_audit_preparation_snapshots(id),
    released_scope_snapshot_id text NOT NULL,
    question_version_id text NOT NULL REFERENCES question_versions(id),
    subject_id text NOT NULL REFERENCES identity_references(subject_id),
    position integer NOT NULL CHECK (position >= 0),
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (preparation_id, question_version_id),
    UNIQUE (preparation_id, position),
    FOREIGN KEY (preparation_id, released_scope_snapshot_id)
        REFERENCES canonical_audit_preparation_snapshots(id, released_scope_snapshot_id),
    FOREIGN KEY (released_scope_snapshot_id, question_version_id)
        REFERENCES canonical_audit_scope_snapshot_questions(snapshot_id, question_version_id)
);

CREATE OR REPLACE FUNCTION validate_canonical_catalog_membership_usage() RETURNS trigger
LANGUAGE plpgsql AS $$
DECLARE catalog_usage text;
BEGIN
    SELECT usage_class INTO catalog_usage FROM canonical_question_catalogs WHERE id = NEW.catalog_id;
    IF catalog_usage IS NULL OR catalog_usage <> NEW.usage_class THEN
        RAISE EXCEPTION 'catalog membership usage class does not match catalog';
    END IF;
    RETURN NEW;
END;
$$;

CREATE TRIGGER canonical_catalog_membership_usage_guard
BEFORE INSERT ON canonical_question_catalog_memberships
FOR EACH ROW EXECUTE FUNCTION validate_canonical_catalog_membership_usage();

CREATE OR REPLACE FUNCTION reject_canonical_exercise_governed_reference() RETURNS trigger
LANGUAGE plpgsql AS $$
DECLARE catalog_usage text;
BEGIN
    SELECT usage_class INTO catalog_usage FROM canonical_question_catalogs WHERE id = NEW.catalog_id;
    IF NEW.usage_class <> catalog_usage THEN
        RAISE EXCEPTION 'scope usage class does not match catalog';
    END IF;
    IF NEW.usage_class = 'PREPROD_EXERCISE' AND NEW.stage IN ('SUBMITTED', 'RELEASED') AND NOT EXISTS (
        SELECT 1 FROM canonical_question_catalogs c
        WHERE c.id = NEW.catalog_id AND c.profile_name = 'aga-preprod'
    ) THEN
        RAISE EXCEPTION 'exercise scope snapshots require the dedicated aga-preprod profile';
    END IF;
    RETURN NEW;
END;
$$;

CREATE TRIGGER canonical_scope_snapshot_usage_guard
BEFORE INSERT ON canonical_audit_scope_snapshots
FOR EACH ROW EXECUTE FUNCTION reject_canonical_exercise_governed_reference();

CREATE TRIGGER canonical_question_catalogs_append_only
BEFORE UPDATE OR DELETE ON canonical_question_catalogs
FOR EACH ROW EXECUTE FUNCTION reject_immutable_row_change();
CREATE TRIGGER canonical_question_catalog_import_runs_append_only
BEFORE UPDATE OR DELETE ON canonical_question_catalog_import_runs
FOR EACH ROW EXECUTE FUNCTION reject_immutable_row_change();
CREATE TRIGGER canonical_question_catalog_memberships_append_only
BEFORE UPDATE OR DELETE ON canonical_question_catalog_memberships
FOR EACH ROW EXECUTE FUNCTION reject_immutable_row_change();
CREATE TRIGGER canonical_question_catalog_membership_events_append_only
BEFORE UPDATE OR DELETE ON canonical_question_catalog_membership_events
FOR EACH ROW EXECUTE FUNCTION reject_immutable_row_change();
CREATE TRIGGER canonical_exercise_question_review_drafts_append_only
BEFORE UPDATE OR DELETE ON canonical_exercise_question_review_drafts
FOR EACH ROW EXECUTE FUNCTION reject_immutable_row_change();
CREATE TRIGGER canonical_exercise_question_review_events_append_only
BEFORE UPDATE OR DELETE ON canonical_exercise_question_review_events
FOR EACH ROW EXECUTE FUNCTION reject_immutable_row_change();
CREATE TRIGGER canonical_audit_scope_selection_operations_append_only
BEFORE UPDATE OR DELETE ON canonical_audit_scope_selection_operations
FOR EACH ROW EXECUTE FUNCTION reject_immutable_row_change();
CREATE TRIGGER canonical_audit_scope_selection_questions_append_only
BEFORE UPDATE OR DELETE ON canonical_audit_scope_selection_questions
FOR EACH ROW EXECUTE FUNCTION reject_immutable_row_change();
CREATE TRIGGER canonical_audit_scope_draft_questions_append_only
BEFORE UPDATE OR DELETE ON canonical_audit_scope_draft_questions
FOR EACH ROW EXECUTE FUNCTION reject_immutable_row_change();
CREATE TRIGGER canonical_audit_scope_snapshots_append_only
BEFORE UPDATE OR DELETE ON canonical_audit_scope_snapshots
FOR EACH ROW EXECUTE FUNCTION reject_immutable_row_change();
CREATE TRIGGER canonical_audit_scope_snapshot_questions_append_only
BEFORE UPDATE OR DELETE ON canonical_audit_scope_snapshot_questions
FOR EACH ROW EXECUTE FUNCTION reject_immutable_row_change();
CREATE TRIGGER canonical_audit_preparation_snapshots_append_only
BEFORE UPDATE OR DELETE ON canonical_audit_preparation_snapshots
FOR EACH ROW EXECUTE FUNCTION reject_immutable_row_change();
CREATE TRIGGER canonical_audit_preparation_questions_append_only
BEFORE UPDATE OR DELETE ON canonical_audit_preparation_questions
FOR EACH ROW EXECUTE FUNCTION reject_immutable_row_change();
