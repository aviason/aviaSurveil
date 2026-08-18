-- Prior-Audit recommendation receipts are append-only projections. They never
-- rewrite historical Audit, checklist, Finding, CAP, Evidence, report, or
-- published catalog rows.
ALTER TABLE canonical_question_catalog_ai_enrichments
    ADD COLUMN IF NOT EXISTS mandatory_control boolean NOT NULL DEFAULT false;

CREATE TABLE prior_audit_recommendation_evaluations (
    evaluation_id text PRIMARY KEY,
    organization_id text NOT NULL REFERENCES organizations(id),
    provider_scope_root_id text NOT NULL,
    provider_scope_id text NOT NULL REFERENCES organization_service_provider_scopes(id),
    regulated_target_id text NOT NULL REFERENCES regulated_targets(id),
    location text NOT NULL,
    audit_type text NOT NULL,
    catalog_version text NOT NULL,
    usage_class text NOT NULL CHECK (usage_class = 'GOVERNED_OPERATIONAL'),
    evaluation_as_of timestamptz NOT NULL,
    history_window_months integer NOT NULL CHECK (history_window_months BETWEEN 1 AND 120),
    comparable_audit_ids jsonb NOT NULL CHECK (jsonb_typeof(comparable_audit_ids) = 'array'),
    recommendation_snapshot_digest text NOT NULL CHECK (governed_sha256(recommendation_snapshot_digest)),
    created_by_subject_id text NOT NULL REFERENCES identity_references(subject_id),
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX prior_audit_recommendation_evaluations_scope_idx
    ON prior_audit_recommendation_evaluations (organization_id, provider_scope_id, regulated_target_id, audit_type, evaluation_as_of DESC);

CREATE TABLE canonical_audit_question_recommendation_snapshots (
    evaluation_id text NOT NULL REFERENCES prior_audit_recommendation_evaluations(evaluation_id),
    question_version_id text NOT NULL REFERENCES question_versions(id),
    recommendation_state text NOT NULL CHECK (recommendation_state IN ('SUGGESTED_NOW', 'MATCHING_OPTIONAL', 'RECENTLY_VERIFIED', 'OUTSIDE_FOCUS', 'UNCERTAIN_SIGNAL')),
    classification text NOT NULL CHECK (classification IN ('MANDATORY_CORE', 'FOCUSED_FULL', 'ROTATIONAL_SAMPLE', 'DEFER_ELIGIBLE')),
    included_by_default boolean NOT NULL,
    can_defer boolean NOT NULL,
    history_count integer NOT NULL CHECK (history_count >= 0),
    comparable_audit_count integer NOT NULL CHECK (comparable_audit_count >= 0),
    last_comparable_result text,
    last_comparable_audit_id text,
    last_verified_at timestamptz,
    recurrence_due_at timestamptz,
    signal_codes jsonb NOT NULL CHECK (jsonb_typeof(signal_codes) = 'array'),
    rationale text NOT NULL CHECK (btrim(rationale) <> ''),
    guardrails jsonb NOT NULL CHECK (jsonb_typeof(guardrails) = 'array'),
    snapshot_digest text NOT NULL CHECK (governed_sha256(snapshot_digest)),
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (evaluation_id, question_version_id)
);

CREATE TABLE canonical_audit_scope_recommendation_deviations (
    deviation_id text PRIMARY KEY,
    evaluation_id text NOT NULL REFERENCES prior_audit_recommendation_evaluations(evaluation_id),
    question_version_id text NOT NULL REFERENCES question_versions(id),
    action text NOT NULL CHECK (action = 'DEFER'),
    manager_reason text NOT NULL CHECK (btrim(manager_reason) <> ''),
    operation_id text NOT NULL UNIQUE,
    created_by_subject_id text NOT NULL REFERENCES identity_references(subject_id),
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (evaluation_id, question_version_id)
);

CREATE TABLE canonical_audit_scope_freezes (
    freeze_id text PRIMARY KEY,
    scope_draft_id text NOT NULL REFERENCES canonical_audit_scope_drafts(id),
    evaluation_id text NOT NULL REFERENCES prior_audit_recommendation_evaluations(evaluation_id),
    recommendation_snapshot_digest text NOT NULL CHECK (governed_sha256(recommendation_snapshot_digest)),
    deviation_digest text NOT NULL CHECK (governed_sha256(deviation_digest)),
    selection_digest text NOT NULL CHECK (btrim(selection_digest) <> ''),
    freeze_digest text NOT NULL CHECK (governed_sha256(freeze_digest)),
    selected_question_version_ids jsonb NOT NULL CHECK (jsonb_typeof(selected_question_version_ids) = 'array'),
    created_by_subject_id text NOT NULL REFERENCES identity_references(subject_id),
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (scope_draft_id, freeze_digest)
);

DROP TRIGGER IF EXISTS prior_audit_recommendation_evaluations_append_only ON prior_audit_recommendation_evaluations;
CREATE TRIGGER prior_audit_recommendation_evaluations_append_only
BEFORE UPDATE OR DELETE ON prior_audit_recommendation_evaluations
FOR EACH ROW EXECUTE FUNCTION reject_immutable_row_change();

DROP TRIGGER IF EXISTS canonical_audit_question_recommendation_snapshots_append_only ON canonical_audit_question_recommendation_snapshots;
CREATE TRIGGER canonical_audit_question_recommendation_snapshots_append_only
BEFORE UPDATE OR DELETE ON canonical_audit_question_recommendation_snapshots
FOR EACH ROW EXECUTE FUNCTION reject_immutable_row_change();

DROP TRIGGER IF EXISTS canonical_audit_scope_recommendation_deviations_append_only ON canonical_audit_scope_recommendation_deviations;
CREATE TRIGGER canonical_audit_scope_recommendation_deviations_append_only
BEFORE UPDATE OR DELETE ON canonical_audit_scope_recommendation_deviations
FOR EACH ROW EXECUTE FUNCTION reject_immutable_row_change();

DROP TRIGGER IF EXISTS canonical_audit_scope_freezes_append_only ON canonical_audit_scope_freezes;
CREATE TRIGGER canonical_audit_scope_freezes_append_only
BEFORE UPDATE OR DELETE ON canonical_audit_scope_freezes
FOR EACH ROW EXECUTE FUNCTION reject_immutable_row_change();
