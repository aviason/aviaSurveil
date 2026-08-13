-- This retained N-1 fixture represents a real pre-30 database with one
-- immutable submitted scope snapshot. Migration 30 must repair its digest
-- before installing the non-empty append-only contract.
-- avia-include: migrations/000001_foundation.up.sql
-- avia-include: migrations/000002_workflow_foundation.up.sql
-- avia-include: migrations/000003_authority_foundation.up.sql
-- avia-include: migrations/000004_evidence_upload_foundation.up.sql
-- avia-include: migrations/000005_sync_foundation.up.sql
-- avia-include: migrations/000006_first_production_routes.up.sql
-- avia-include: migrations/000007_full_workflow_projection.up.sql
-- avia-include: migrations/000008_communications_documents.up.sql
-- avia-include: migrations/000009_notifications_risk_admin.up.sql
-- avia-include: migrations/000010_identity_settings.up.sql
-- avia-include: migrations/000011_preliminary_report_versions.up.sql
-- avia-include: migrations/000012_provider_user_lifecycle.up.sql
-- avia-include: migrations/000013_authority_organization.up.sql
-- avia-include: migrations/000014_malware_scan_metadata.up.sql
-- avia-include: migrations/000015_document_render_provenance.up.sql
-- avia-include: migrations/000016_notification_email_delivery.up.sql
-- avia-include: migrations/000017_outbox_trace_context.up.sql
-- avia-include: migrations/000018_desired_membership_versions.up.sql
-- avia-include: migrations/000019_identity_lifecycle_delivery.up.sql
-- avia-include: migrations/000020_session_membership_authority.up.sql
-- avia-include: migrations/000021_regulatory_checklist_governance.up.sql
-- avia-include: migrations/000022_aviasurveil_data_feed_outbox.up.sql
-- avia-include: migrations/000023_aviasurveil_data_feed_publisher.up.sql
-- avia-include: migrations/000024_aviasurveil_data_feed_replay_runs.up.sql
-- avia-include: migrations/000025_regulatory_source_currentness_activation.up.sql
-- avia-include: migrations/000026_regulatory_source_currentness_no_reactivation.up.sql
-- avia-include: migrations/000027_regulatory_source_impact_multi_candidate_links.up.sql
-- avia-include: migrations/000028_governed_checklist_intake_and_authoring.up.sql
-- avia-include: migrations/000029_canonical_aga_catalog_scope.up.sql

CREATE TABLE schema_migrations (
    version bigint PRIMARY KEY,
    name text NOT NULL,
    applied_at timestamptz NOT NULL DEFAULT now()
);

INSERT INTO schema_migrations (version, name) VALUES
    (1, '000001_foundation.up.sql'),
    (2, '000002_workflow_foundation.up.sql');

INSERT INTO schema_migrations (version, name)
SELECT version, format('fixture-%s', version)
FROM generate_series(3, 29) AS versions(version)
ON CONFLICT (version) DO NOTHING;

INSERT INTO identity_references (subject_id, issuer, display_name)
VALUES ('fixture-n1-manager', 'fixture', 'N-1 synthetic manager');

INSERT INTO organizations (id, legal_name, organization_type, status)
VALUES ('fixture-n1-org', 'N-1 Synthetic Operator', 'AIR_OPERATOR', 'ACTIVE');

INSERT INTO regulated_targets (id, target_kind, organization_id)
VALUES ('fixture-n1-target', 'ORGANIZATION', 'fixture-n1-org');

INSERT INTO organization_service_provider_scopes (
    id, organization_id, service_provider_type_id, authorization_identifier,
    status, effective_from, primary_target_id
)
VALUES (
    'fixture-n1-scope', 'fixture-n1-org', 'AIR_OPERATOR', 'FIXTURE-N1-AOC',
    'ACTIVE', DATE '2026-01-01', 'fixture-n1-target'
);

INSERT INTO question_versions (
    id, question_id, version, prompt, configured_reference, expected_evidence,
    created_by_subject_id
)
VALUES (
    'fixture-n1-question-v1', 'fixture-n1-question', 1,
    'Synthetic N-1 question body.', 'Fixture reference', 'Fixture evidence',
    'fixture-n1-manager'
);

INSERT INTO canonical_question_catalogs (
    id, catalog_version, usage_class, profile_name, profile_version, status,
    source_package_version, source_package_json_sha256, source_package_zip_sha256,
    root_digest, question_count, form_count, created_by_subject_id
)
VALUES (
    'fixture-n1-catalog', 'fixture-n1@1.0.0', 'PREPROD_EXERCISE', 'aga-preprod',
    '1.0.0', 'SEALED', 'fixture', 'sha256:fixture-json', 'sha256:fixture-zip',
    'sha256:fixture-root', 1, 1, 'fixture-n1-manager'
);

INSERT INTO canonical_question_catalog_forms (
    catalog_id, form_code, form_digest, question_count, source_gap_state
)
VALUES ('fixture-n1-catalog', 'FIXTURE', 'sha256:fixture-form', 1, 'NONE');

INSERT INTO canonical_question_catalog_memberships (
    catalog_id, question_version_id, usage_class, form_code, proposal_id,
    ordinal, question_digest, source_gap_state
)
VALUES (
    'fixture-n1-catalog', 'fixture-n1-question-v1', 'PREPROD_EXERCISE',
    'FIXTURE', 'FIXTURE-1', 1, 'sha256:fixture-question', 'NONE'
);

INSERT INTO planning_intake_drafts (
    id, organization_id, values, created_by_subject_id
)
VALUES (
    'fixture-n1-draft', 'fixture-n1-org', '{"fixture":true}'::jsonb,
    'fixture-n1-manager'
);

INSERT INTO canonical_audit_scope_drafts (
    id, planning_intake_draft_id, organization_id, provider_scope_id,
    regulated_target_id, audit_type, catalog_id, usage_class, status,
    selected_question_count, selection_digest, notice_policy, created_by_subject_id
)
VALUES (
    'fixture-n1-scope-draft', 'fixture-n1-draft', 'fixture-n1-org',
    'fixture-n1-scope', 'fixture-n1-target', 'CONTINUED_SURVEILLANCE',
    'fixture-n1-catalog', 'PREPROD_EXERCISE', 'SUBMITTED', 1,
    'sha256:fixture-selection', 'ADVANCE', 'fixture-n1-manager'
);

INSERT INTO canonical_audit_scope_snapshots (
    id, scope_draft_id, revision, stage, catalog_id, usage_class,
    selection_digest, selected_question_count, snapshot, created_by_subject_id
)
VALUES (
    'fixture-n1-submitted-snapshot', 'fixture-n1-scope-draft', 1, 'SUBMITTED',
    'fixture-n1-catalog', 'PREPROD_EXERCISE', 'sha256:fixture-selection', 1,
    '{"fixture":"populated-n-1"}'::jsonb, 'fixture-n1-manager'
);

INSERT INTO canonical_audit_scope_snapshot_questions (
    snapshot_id, catalog_id, question_version_id, position
)
VALUES ('fixture-n1-submitted-snapshot', 'fixture-n1-catalog', 'fixture-n1-question-v1', 0);
