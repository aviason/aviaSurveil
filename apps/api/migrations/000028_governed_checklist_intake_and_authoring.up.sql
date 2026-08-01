-- Version 28 is forward-only.  It adds the append-only intake and dual-
-- authoring ledger without changing the bytes or decisions of version 27.

CREATE TABLE governed_reviewed_source_sets (
    reviewed_source_set_id text PRIMARY KEY,
    root_id text NOT NULL,
    version bigint NOT NULL CHECK (version > 0),
    supersedes_set_id text,
    supersedes_digest text,
    schema_version text NOT NULL,
    canonical_digest text NOT NULL CHECK (governed_sha256(canonical_digest)),
    authority_evidence_id text NOT NULL,
    provenance text NOT NULL CHECK (provenance IN ('CANONICAL_TEST_FIXTURE','GOVERNANCE_DIRECTIVE')),
    created_by_subject_id text REFERENCES identity_references(subject_id),
    reason text NOT NULL CHECK (btrim(reason) <> ''),
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (root_id, version),
    UNIQUE (root_id, supersedes_set_id),
    UNIQUE (reviewed_source_set_id, version),
    UNIQUE (reviewed_source_set_id, version, canonical_digest),
    FOREIGN KEY (supersedes_set_id) REFERENCES governed_reviewed_source_sets(reviewed_source_set_id),
    CHECK ((supersedes_set_id IS NULL) = (supersedes_digest IS NULL))
);

CREATE TABLE governed_reviewed_source_set_links (
    reviewed_source_set_id text NOT NULL REFERENCES governed_reviewed_source_sets(reviewed_source_set_id),
    reviewed_source_set_version bigint NOT NULL,
    ordinal integer NOT NULL CHECK (ordinal > 0),
    source_id text NOT NULL,
    source_version_id text NOT NULL REFERENCES regulatory_source_versions(id),
    source_hash text NOT NULL CHECK (governed_sha256(source_hash)),
    source_class text NOT NULL,
    chain_role text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (reviewed_source_set_id, ordinal),
    UNIQUE (reviewed_source_set_id, source_version_id, source_hash, source_class, chain_role),
    FOREIGN KEY (reviewed_source_set_id, reviewed_source_set_version)
        REFERENCES governed_reviewed_source_sets(reviewed_source_set_id, version)
);

CREATE TABLE governed_checklist_functional_assignments (
    assignment_id text PRIMARY KEY,
    assignment_root_id text NOT NULL,
    assignment_version bigint NOT NULL CHECK (assignment_version > 0),
    supersedes_assignment_id text UNIQUE,
    subject_id text NOT NULL REFERENCES identity_references(subject_id),
    permission text NOT NULL CHECK (permission IN ('REGULATORY_SOURCE_OWNER','CHECKLIST_REVIEWER')),
    scope_kind text NOT NULL CHECK (scope_kind IN ('SOURCE_IDENTITY','REVIEWED_SOURCE_SET','CANDIDATE_ROOT')),
    source_version_id text REFERENCES regulatory_source_versions(id),
    reviewed_source_set_id text,
    reviewed_source_set_version bigint,
    reviewed_source_set_digest text,
    candidate_root_id text,
    department_id text REFERENCES caa_departments(id),
    organizational_unit_id text,
    effective_from timestamptz NOT NULL,
    effective_to timestamptz,
    status text NOT NULL CHECK (status IN ('ACTIVE','REVOKED','EXPIRED')),
    created_by_subject_id text REFERENCES identity_references(subject_id),
    reason text NOT NULL CHECK (btrim(reason) <> ''),
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (assignment_root_id, assignment_version),
    CHECK (effective_to IS NULL OR effective_to > effective_from),
    CHECK ((scope_kind = 'SOURCE_IDENTITY' AND source_version_id IS NOT NULL AND reviewed_source_set_id IS NULL AND candidate_root_id IS NULL)
        OR (scope_kind = 'REVIEWED_SOURCE_SET' AND source_version_id IS NULL AND reviewed_source_set_id IS NOT NULL AND reviewed_source_set_version IS NOT NULL AND reviewed_source_set_digest IS NOT NULL AND candidate_root_id IS NULL)
        OR (scope_kind = 'CANDIDATE_ROOT' AND source_version_id IS NULL AND reviewed_source_set_id IS NULL AND candidate_root_id IS NOT NULL)),
    FOREIGN KEY (supersedes_assignment_id) REFERENCES governed_checklist_functional_assignments(assignment_id),
    FOREIGN KEY (reviewed_source_set_id, reviewed_source_set_version, reviewed_source_set_digest)
        REFERENCES governed_reviewed_source_sets(reviewed_source_set_id, version, canonical_digest),
    FOREIGN KEY (department_id, organizational_unit_id)
        REFERENCES caa_organizational_units(department_id, id)
);
CREATE INDEX governed_checklist_functional_assignments_current_idx
    ON governed_checklist_functional_assignments (permission, scope_kind, subject_id, effective_from DESC, assignment_version DESC)
    WHERE status = 'ACTIVE';

CREATE TABLE checklist_import_batches (
    import_batch_id text PRIMARY KEY,
    operation_id text NOT NULL UNIQUE,
    idempotency_key text NOT NULL UNIQUE,
    expected_archive_sha256 text NOT NULL CHECK (governed_sha256(expected_archive_sha256)),
    observed_archive_sha256 text CHECK (observed_archive_sha256 IS NULL OR governed_sha256(observed_archive_sha256)),
    observed_archive_bytes bigint CHECK (observed_archive_bytes IS NULL OR observed_archive_bytes >= 0),
    status text NOT NULL CHECK (status IN ('RECEIVED','PROCESSING','INVENTORY_COMPLETE','INVENTORY_FAILED')),
    manifest_digest text CHECK (manifest_digest IS NULL OR governed_sha256(manifest_digest)),
    intake_safety_eligible boolean NOT NULL DEFAULT false,
    reason text NOT NULL CHECK (btrim(reason) <> ''),
    created_by_subject_id text NOT NULL REFERENCES identity_references(subject_id),
    created_at timestamptz NOT NULL DEFAULT now(),
    finalized_at timestamptz
);

CREATE TABLE checklist_import_files (
    import_file_id text PRIMARY KEY,
    import_batch_id text NOT NULL REFERENCES checklist_import_batches(import_batch_id),
    ordinal integer NOT NULL CHECK (ordinal > 0),
    normalized_path text NOT NULL CHECK (btrim(normalized_path) <> ''),
    original_path text NOT NULL,
    sha256 text NOT NULL CHECK (governed_sha256(sha256)),
    byte_count bigint NOT NULL CHECK (byte_count >= 0),
    media_type text NOT NULL,
    initial_identity_match_state text NOT NULL CHECK (initial_identity_match_state IN ('REGISTER_MATCHED','IDENTITY_REVIEW_REQUIRED','NOT_REGISTERED')),
    initial_candidate_import_state text NOT NULL CHECK (initial_candidate_import_state IN ('ELIGIBLE','REQUIRES_IDENTITY_RESOLUTION','INELIGIBLE')),
    register_form_code text,
    register_title text,
    visible_title text,
    terminal_manifest_digest text NOT NULL CHECK (governed_sha256(terminal_manifest_digest)),
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (import_batch_id, normalized_path),
    UNIQUE (import_batch_id, ordinal)
);

CREATE TABLE checklist_import_phase_receipts (
    receipt_id text PRIMARY KEY,
    import_batch_id text NOT NULL REFERENCES checklist_import_batches(import_batch_id),
    import_file_id text REFERENCES checklist_import_files(import_file_id),
    phase text NOT NULL CHECK (phase IN ('ARCHIVE_VALIDATE','OBJECT_FINALIZE','ARCHIVE_SCAN','ENTRY_VALIDATE','PDF_SCAN','PDF_PARSE','REGISTER_PARSE','IDENTITY_MATCH','SCRATCH_CLEANUP')),
    input_digest text NOT NULL CHECK (governed_sha256(input_digest)),
    policy_version text NOT NULL,
    result_digest text CHECK (result_digest IS NULL OR governed_sha256(result_digest)),
    outcome text NOT NULL CHECK (outcome IN ('SUCCEEDED','FAILED','ABANDONED_EXHAUSTED','NOT_RUN_DUE_TO_PREDECESSOR')),
    error_code text,
    payload jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(payload) = 'object'),
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (import_batch_id, import_file_id, phase)
);

CREATE TABLE checklist_register_entries (
    register_entry_id text PRIMARY KEY,
    import_batch_id text NOT NULL REFERENCES checklist_import_batches(import_batch_id),
    register_file_id text NOT NULL REFERENCES checklist_import_files(import_file_id),
    page integer NOT NULL CHECK (page > 0),
    row_number integer NOT NULL CHECK (row_number > 0),
    ordinal integer NOT NULL CHECK (ordinal > 0),
    form_code text NOT NULL CHECK (btrim(form_code) <> ''),
    title_text text NOT NULL,
    version_text text,
    status_text text,
    matched_import_file_id text REFERENCES checklist_import_files(import_file_id),
    match_state text NOT NULL CHECK (match_state IN ('MATCHED','UNMATCHED','DUPLICATE','EXTRA')),
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (import_batch_id, ordinal),
    UNIQUE (import_batch_id, form_code)
);

CREATE TABLE checklist_import_identity_resolutions (
    resolution_id text PRIMARY KEY,
    resolution_root_id text NOT NULL,
    resolution_revision bigint NOT NULL CHECK (resolution_revision > 0),
    supersedes_resolution_id text UNIQUE,
    import_file_id text NOT NULL REFERENCES checklist_import_files(import_file_id),
    expected_prior_leaf_id text,
    expected_prior_digest text,
    expected_file_sha256 text NOT NULL CHECK (governed_sha256(expected_file_sha256)),
    expected_manifest_digest text NOT NULL CHECK (governed_sha256(expected_manifest_digest)),
    selected_identity_source text NOT NULL CHECK (selected_identity_source IN ('REGISTER','VISIBLE','PDF_METADATA','HUMAN_TRANSCRIPTION')),
    selected_identity_value text NOT NULL CHECK (btrim(selected_identity_value) <> ''),
    transcription_reason text,
    transcription_receipt_id text,
    competing_values jsonb NOT NULL DEFAULT '[]'::jsonb CHECK (jsonb_typeof(competing_values) = 'array'),
    actor_subject_id text NOT NULL REFERENCES identity_references(subject_id),
    actor_membership_id text,
    reason text NOT NULL CHECK (btrim(reason) <> ''),
    operation_id text NOT NULL UNIQUE,
    idempotency_key text NOT NULL UNIQUE,
    semantic_payload_digest text NOT NULL CHECK (governed_sha256(semantic_payload_digest)),
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (resolution_root_id, resolution_revision),
    FOREIGN KEY (supersedes_resolution_id) REFERENCES checklist_import_identity_resolutions(resolution_id),
    CHECK ((supersedes_resolution_id IS NULL AND expected_prior_leaf_id IS NULL AND expected_prior_digest IS NULL)
        OR (supersedes_resolution_id IS NOT NULL AND expected_prior_leaf_id IS NOT NULL AND expected_prior_digest IS NOT NULL)),
    CHECK ((selected_identity_source = 'HUMAN_TRANSCRIPTION' AND transcription_reason IS NOT NULL AND transcription_receipt_id IS NOT NULL)
        OR (selected_identity_source <> 'HUMAN_TRANSCRIPTION' AND transcription_reason IS NULL AND transcription_receipt_id IS NULL))
);
CREATE INDEX checklist_import_identity_resolutions_current_idx
    ON checklist_import_identity_resolutions (import_file_id, resolution_revision DESC, created_at DESC);

CREATE TABLE checklist_import_extraction_review_packets (
    packet_id text PRIMARY KEY,
    import_batch_id text NOT NULL REFERENCES checklist_import_batches(import_batch_id),
    import_file_id text NOT NULL REFERENCES checklist_import_files(import_file_id),
    terminal_manifest_digest text NOT NULL CHECK (governed_sha256(terminal_manifest_digest)),
    parser_receipt_id text NOT NULL REFERENCES checklist_import_phase_receipts(receipt_id),
    parser_output_digest text NOT NULL CHECK (governed_sha256(parser_output_digest)),
    parser_output_bytes bigint NOT NULL CHECK (parser_output_bytes >= 0 AND parser_output_bytes <= 26214400),
    generator_policy_version text NOT NULL,
    outcome text NOT NULL CHECK (outcome IN ('READY','FAILED')),
    proposal_count integer NOT NULL CHECK (proposal_count >= 0 AND proposal_count <= 2000),
    packet_digest text NOT NULL CHECK (governed_sha256(packet_digest)),
    failure_code text,
    created_by_subject_id text NOT NULL REFERENCES identity_references(subject_id),
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (import_file_id, terminal_manifest_digest, parser_receipt_id, generator_policy_version),
    CHECK ((outcome = 'READY' AND proposal_count > 0 AND failure_code IS NULL) OR (outcome = 'FAILED' AND proposal_count = 0 AND failure_code IS NOT NULL))
);

CREATE TABLE checklist_import_extraction_review_proposals (
    proposal_id text PRIMARY KEY,
    packet_id text NOT NULL REFERENCES checklist_import_extraction_review_packets(packet_id),
    proposal_ordinal integer NOT NULL CHECK (proposal_ordinal > 0),
    original_text text NOT NULL CHECK (length(original_text) > 0 AND length(original_text) <= 65536),
    text_digest text NOT NULL CHECK (governed_sha256(text_digest)),
    page integer NOT NULL CHECK (page > 0),
    section text,
    row_locator text,
    region_locator text,
    text_span jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(text_span) = 'object'),
    parser_provenance jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(parser_provenance) = 'object'),
    proposed_boundary_kind text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (packet_id, proposal_ordinal)
);

CREATE TABLE checklist_import_extraction_decision_sets (
    decision_set_id text PRIMARY KEY,
    decision_set_root_id text NOT NULL,
    revision bigint NOT NULL CHECK (revision > 0),
    supersedes_decision_set_id text UNIQUE,
    packet_id text NOT NULL REFERENCES checklist_import_extraction_review_packets(packet_id),
    import_file_id text NOT NULL REFERENCES checklist_import_files(import_file_id),
    terminal_manifest_digest text NOT NULL CHECK (governed_sha256(terminal_manifest_digest)),
    parser_output_digest text NOT NULL CHECK (governed_sha256(parser_output_digest)),
    expected_prior_leaf_id text,
    expected_prior_digest text,
    actor_subject_id text NOT NULL REFERENCES identity_references(subject_id),
    reason text NOT NULL CHECK (btrim(reason) <> ''),
    operation_id text NOT NULL UNIQUE,
    idempotency_key text NOT NULL UNIQUE,
    semantic_payload_digest text NOT NULL CHECK (governed_sha256(semantic_payload_digest)),
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (decision_set_root_id, revision),
    FOREIGN KEY (supersedes_decision_set_id) REFERENCES checklist_import_extraction_decision_sets(decision_set_id),
    CHECK ((supersedes_decision_set_id IS NULL AND expected_prior_leaf_id IS NULL AND expected_prior_digest IS NULL)
        OR (supersedes_decision_set_id IS NOT NULL AND expected_prior_leaf_id IS NOT NULL AND expected_prior_digest IS NOT NULL))
);

CREATE TABLE checklist_import_extraction_decisions (
    decision_id text PRIMARY KEY,
    decision_set_id text NOT NULL REFERENCES checklist_import_extraction_decision_sets(decision_set_id),
    decision_ordinal integer NOT NULL CHECK (decision_ordinal > 0),
    decision_kind text NOT NULL CHECK (decision_kind IN ('ACCEPT','SPLIT','MERGE','TRANSCRIBE','EXCLUDE')),
    consumed_proposal_ids text[] NOT NULL CHECK (cardinality(consumed_proposal_ids) > 0),
    consumed_proposal_digests text[] NOT NULL,
    output_seed_ids text[] NOT NULL DEFAULT '{}',
    output_payload jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(output_payload) = 'object'),
    reason text NOT NULL CHECK (btrim(reason) <> ''),
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (decision_set_id, decision_ordinal)
);

CREATE TABLE checklist_import_object_intents (
    intent_id text PRIMARY KEY,
    import_batch_id text NOT NULL REFERENCES checklist_import_batches(import_batch_id),
    import_file_id text REFERENCES checklist_import_files(import_file_id),
    purpose text NOT NULL CHECK (purpose IN ('ARCHIVE_QUARANTINE','PARSER_OUTPUT')),
    object_key text NOT NULL UNIQUE,
    expected_sha256 text NOT NULL CHECK (governed_sha256(expected_sha256)),
    expected_bytes bigint NOT NULL CHECK (expected_bytes >= 0),
    state text NOT NULL CHECK (state IN ('PENDING','VERIFIED','FAILED')),
    object_version text,
    observed_sha256 text CHECK (observed_sha256 IS NULL OR governed_sha256(observed_sha256)),
    observed_bytes bigint CHECK (observed_bytes IS NULL OR observed_bytes >= 0),
    expires_at timestamptz NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    CHECK ((state = 'VERIFIED' AND object_version IS NOT NULL AND observed_sha256 = expected_sha256 AND observed_bytes = expected_bytes) OR state <> 'VERIFIED')
);

CREATE TABLE checklist_import_attempts (
    attempt_id text PRIMARY KEY,
    attempt_root_id text NOT NULL,
    predecessor_attempt_id text REFERENCES checklist_import_attempts(attempt_id),
    ordinal integer NOT NULL CHECK (ordinal > 0),
    phase text NOT NULL CHECK (phase IN ('ARCHIVE_VALIDATE','OBJECT_FINALIZE','ARCHIVE_SCAN','ENTRY_VALIDATE','PDF_SCAN','PDF_PARSE','REGISTER_PARSE','IDENTITY_MATCH','SCRATCH_CLEANUP')),
    import_batch_id text NOT NULL REFERENCES checklist_import_batches(import_batch_id),
    import_file_id text REFERENCES checklist_import_files(import_file_id),
    input_digest text NOT NULL CHECK (governed_sha256(input_digest)),
    policy_version text NOT NULL,
    lease_owner text,
    lease_expires_at timestamptz,
    fencing_token bigint NOT NULL DEFAULT 1 CHECK (fencing_token > 0),
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (attempt_root_id, ordinal)
);

CREATE TABLE checklist_import_attempt_events (
    event_id text PRIMARY KEY,
    attempt_id text NOT NULL UNIQUE REFERENCES checklist_import_attempts(attempt_id),
    state text NOT NULL CHECK (state IN ('SUCCEEDED','FAILED','ABANDONED')),
    result_digest text CHECK (result_digest IS NULL OR governed_sha256(result_digest)),
    error_code text,
    fencing_token bigint NOT NULL,
    completed_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE existing_checklist_candidates (
    existing_candidate_id text PRIMARY KEY,
    candidate_root_id text NOT NULL,
    revision bigint NOT NULL CHECK (revision > 0),
    supersedes_existing_candidate_id text UNIQUE,
    content_digest text NOT NULL CHECK (governed_sha256(content_digest)),
    import_batch_id text NOT NULL REFERENCES checklist_import_batches(import_batch_id),
    import_file_id text NOT NULL REFERENCES checklist_import_files(import_file_id),
    extraction_packet_id text NOT NULL REFERENCES checklist_import_extraction_review_packets(packet_id),
    extraction_decision_set_id text NOT NULL REFERENCES checklist_import_extraction_decision_sets(decision_set_id),
    identity_basis text NOT NULL CHECK (identity_basis IN ('REGISTER_MATCHED','ADMIN_RESOLUTION')),
    resolution_id text REFERENCES checklist_import_identity_resolutions(resolution_id),
    origin text NOT NULL CHECK (origin = 'EXISTING_CHECKLIST_CANDIDATE'),
    schema_version text NOT NULL,
    source_file_sha256 text NOT NULL CHECK (governed_sha256(source_file_sha256)),
    form_code text NOT NULL,
    title text,
    question_count integer NOT NULL CHECK (question_count >= 0),
    created_by_subject_id text NOT NULL REFERENCES identity_references(subject_id),
    reason text NOT NULL CHECK (btrim(reason) <> ''),
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (candidate_root_id, revision),
    FOREIGN KEY (supersedes_existing_candidate_id) REFERENCES existing_checklist_candidates(existing_candidate_id),
    CHECK ((identity_basis = 'REGISTER_MATCHED' AND resolution_id IS NULL) OR (identity_basis = 'ADMIN_RESOLUTION' AND resolution_id IS NOT NULL))
);

CREATE TABLE existing_checklist_candidate_questions (
    existing_candidate_id text NOT NULL REFERENCES existing_checklist_candidates(existing_candidate_id),
    question_id text NOT NULL,
    ordinal integer NOT NULL CHECK (ordinal > 0),
    wording text NOT NULL,
    source_locators jsonb NOT NULL DEFAULT '[]'::jsonb CHECK (jsonb_typeof(source_locators) = 'array'),
    operational_intent text,
    expected_evidence text,
    result_history text,
    applicability text,
    scope_classification text,
    provenance_state text NOT NULL CHECK (provenance_state IN ('SUPPLIED_UNVERIFIED','NOT_SUPPLIED','UNREADABLE','HUMAN_TRANSCRIBED_WITH_RECEIPT')),
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (existing_candidate_id, question_id),
    UNIQUE (existing_candidate_id, ordinal)
);

CREATE TABLE governed_candidate_source_binding_sets (
    binding_set_id text PRIMARY KEY,
    candidate_root_id text NOT NULL,
    candidate_id text NOT NULL,
    candidate_revision bigint NOT NULL,
    candidate_content_digest text NOT NULL CHECK (governed_sha256(candidate_content_digest)),
    reviewed_source_set_id text,
    reviewed_source_set_version bigint,
    reviewed_source_set_digest text,
    source_chain_digest text NOT NULL CHECK (governed_sha256(source_chain_digest)),
    created_at timestamptz NOT NULL DEFAULT now(),
    FOREIGN KEY (reviewed_source_set_id, reviewed_source_set_version, reviewed_source_set_digest)
        REFERENCES governed_reviewed_source_sets(reviewed_source_set_id, version, canonical_digest),
    UNIQUE (candidate_id, candidate_revision, candidate_content_digest)
);

CREATE TABLE governed_candidate_source_binding_links (
    binding_set_id text NOT NULL REFERENCES governed_candidate_source_binding_sets(binding_set_id),
    ordinal integer NOT NULL CHECK (ordinal > 0),
    source_version_id text NOT NULL REFERENCES regulatory_source_versions(id),
    source_hash text NOT NULL CHECK (governed_sha256(source_hash)),
    source_class text NOT NULL,
    chain_role text NOT NULL,
    currentness_event_id text REFERENCES regulatory_source_currentness_events(event_id),
    source_authority_attestation_id text,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (binding_set_id, ordinal),
    UNIQUE (binding_set_id, source_version_id, source_hash, source_class, chain_role)
);

CREATE TABLE regulatory_source_authority_attestations (
    decision_id text PRIMARY KEY,
    decision_root_id text NOT NULL,
    supersedes_decision_id text UNIQUE,
    outcome text NOT NULL CHECK (outcome IN ('ACCEPT','RETURN')),
    source_id text NOT NULL,
    source_version_id text NOT NULL REFERENCES regulatory_source_versions(id),
    source_hash text NOT NULL CHECK (governed_sha256(source_hash)),
    source_class text NOT NULL,
    chain_role text NOT NULL,
    currentness_event_id text REFERENCES regulatory_source_currentness_events(event_id),
    assignment_id text NOT NULL REFERENCES governed_checklist_functional_assignments(assignment_id),
    actor_subject_id text NOT NULL REFERENCES identity_references(subject_id),
    actor_membership_id text NOT NULL,
    reason text NOT NULL CHECK (btrim(reason) <> ''),
    operation_id text NOT NULL UNIQUE,
    idempotency_key text NOT NULL UNIQUE,
    decision_subject_digest text NOT NULL CHECK (governed_sha256(decision_subject_digest)),
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (decision_root_id, decision_subject_digest),
    FOREIGN KEY (supersedes_decision_id) REFERENCES regulatory_source_authority_attestations(decision_id)
);

CREATE TABLE governed_candidate_source_chain_links (
    binding_set_id text NOT NULL REFERENCES governed_candidate_source_binding_sets(binding_set_id),
    question_id text NOT NULL,
    ordinal integer NOT NULL CHECK (ordinal > 0),
    chain_role text NOT NULL CHECK (chain_role IN ('REGULATORY_AUTHORITY','NATIONAL_REQUIREMENT','CONTROLLED_CAA_PROCEDURE','DERIVED_CONTEXT')),
    source_id text NOT NULL,
    source_version_id text NOT NULL REFERENCES regulatory_source_versions(id),
    source_hash text NOT NULL CHECK (governed_sha256(source_hash)),
    locator jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(locator) = 'object'),
    currentness_event_id text REFERENCES regulatory_source_currentness_events(event_id),
    source_authority_attestation_id text REFERENCES regulatory_source_authority_attestations(decision_id),
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (binding_set_id, question_id, ordinal),
    UNIQUE (binding_set_id, question_id, source_version_id, source_hash, chain_role)
);

CREATE TABLE governed_required_owner_resolution_facts (
    resolution_id text PRIMARY KEY,
    candidate_id text NOT NULL,
    candidate_revision bigint NOT NULL,
    candidate_content_digest text NOT NULL CHECK (governed_sha256(candidate_content_digest)),
    provider_scope_id text NOT NULL,
    target_id text NOT NULL,
    inspection_type text NOT NULL,
    input_digest text NOT NULL CHECK (governed_sha256(input_digest)),
    outcome text NOT NULL CHECK (outcome IN ('RESOLVED','REVIEW_REQUIRED')),
    owners jsonb NOT NULL DEFAULT '[]'::jsonb CHECK (jsonb_typeof(owners) = 'array'),
    blocker_digest text CHECK (blocker_digest IS NULL OR governed_sha256(blocker_digest)),
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (candidate_id, candidate_revision, candidate_content_digest)
);

CREATE TABLE governed_checklist_review_comments (
    comment_id text PRIMARY KEY,
    candidate_id text NOT NULL,
    candidate_revision bigint NOT NULL,
    candidate_content_digest text NOT NULL CHECK (governed_sha256(candidate_content_digest)),
    question_id text,
    field_path text,
    visibility text NOT NULL CHECK (visibility = 'INTERNAL_CAA'),
    recommendation text CHECK (recommendation IN ('RECOMMEND','RETURN_RECOMMENDED','NO_RECOMMENDATION')),
    body text NOT NULL CHECK (btrim(body) <> ''),
    actor_subject_id text NOT NULL REFERENCES identity_references(subject_id),
    actor_membership_id text,
    operation_id text NOT NULL UNIQUE,
    idempotency_key text NOT NULL UNIQUE,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE governed_reviewer_recommendation_dispositions (
    disposition_id text PRIMARY KEY,
    comment_id text NOT NULL REFERENCES governed_checklist_review_comments(comment_id),
    candidate_id text NOT NULL,
    candidate_revision bigint NOT NULL,
    candidate_content_digest text NOT NULL CHECK (governed_sha256(candidate_content_digest)),
    disposition text NOT NULL CHECK (disposition IN ('ACKNOWLEDGED','ADOPTED_AS_RETURN','NOT_ADOPTED_WITH_REASON')),
    reason text NOT NULL CHECK (btrim(reason) <> ''),
    actor_subject_id text NOT NULL REFERENCES identity_references(subject_id),
    operation_id text NOT NULL UNIQUE,
    idempotency_key text NOT NULL UNIQUE,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE governed_source_mapping_attestations (
    decision_id text PRIMARY KEY,
    decision_root_id text NOT NULL,
    supersedes_decision_id text UNIQUE,
    outcome text NOT NULL CHECK (outcome IN ('ACCEPT','RETURN')),
    candidate_id text NOT NULL,
    candidate_revision bigint NOT NULL,
    candidate_content_digest text NOT NULL CHECK (governed_sha256(candidate_content_digest)),
    binding_set_id text REFERENCES governed_candidate_source_binding_sets(binding_set_id),
    complete_chain_digest text CHECK (complete_chain_digest IS NULL OR governed_sha256(complete_chain_digest)),
    incomplete_proposal_digest text CHECK (incomplete_proposal_digest IS NULL OR governed_sha256(incomplete_proposal_digest)),
    assignment_id text NOT NULL REFERENCES governed_checklist_functional_assignments(assignment_id),
    actor_subject_id text NOT NULL REFERENCES identity_references(subject_id),
    actor_membership_id text NOT NULL,
    reason text NOT NULL CHECK (btrim(reason) <> ''),
    operation_id text NOT NULL UNIQUE,
    idempotency_key text NOT NULL UNIQUE,
    decision_subject_digest text NOT NULL CHECK (governed_sha256(decision_subject_digest)),
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (decision_root_id, decision_subject_digest),
    FOREIGN KEY (supersedes_decision_id) REFERENCES governed_source_mapping_attestations(decision_id),
    CHECK ((outcome = 'ACCEPT' AND complete_chain_digest IS NOT NULL AND incomplete_proposal_digest IS NULL)
        OR (outcome = 'RETURN' AND ((complete_chain_digest IS NOT NULL AND incomplete_proposal_digest IS NULL) OR (complete_chain_digest IS NULL AND incomplete_proposal_digest IS NOT NULL))))
);

DO $$
BEGIN
    IF to_regclass('public.template_draft_versions') IS NOT NULL THEN
        ALTER TABLE template_draft_versions ADD COLUMN IF NOT EXISTS entry_path text;
        ALTER TABLE template_draft_versions ADD COLUMN IF NOT EXISTS lineage_kind text;
        ALTER TABLE template_draft_versions ADD COLUMN IF NOT EXISTS owner_resolution_digest text;
        ALTER TABLE template_draft_versions ADD COLUMN IF NOT EXISTS blocker_digest text;
        ALTER TABLE template_draft_versions ADD COLUMN IF NOT EXISTS existing_candidate_id text;
        ALTER TABLE template_draft_versions ADD COLUMN IF NOT EXISTS governed_source_binding_set_id text;
        ALTER TABLE template_draft_versions ADD COLUMN IF NOT EXISTS legacy_authority_state text;
        ALTER TABLE template_draft_versions ADD COLUMN IF NOT EXISTS creation_basis text;
        UPDATE template_draft_versions SET entry_path = 'GENERATION_RUN', lineage_kind = 'PRE_V28_UNATTESTED'
        WHERE generation_run_id IS NOT NULL AND entry_path IS NULL;
        CREATE INDEX IF NOT EXISTS template_draft_versions_entry_path_queue_idx
            ON template_draft_versions (entry_path, status, id);
    END IF;
END $$;

CREATE OR REPLACE FUNCTION governed_v28_append_only_guard() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION '% records are append-only', TG_TABLE_NAME USING ERRCODE = '55000';
END;
$$;

DO $$
DECLARE table_name text;
BEGIN
    FOREACH table_name IN ARRAY ARRAY[
        'governed_reviewed_source_sets','governed_reviewed_source_set_links',
        'governed_checklist_functional_assignments','checklist_import_batches',
        'checklist_import_files','checklist_import_phase_receipts','checklist_register_entries',
        'checklist_import_identity_resolutions','checklist_import_extraction_review_packets',
        'checklist_import_extraction_review_proposals','checklist_import_extraction_decision_sets',
        'checklist_import_extraction_decisions','checklist_import_object_intents',
        'checklist_import_attempts','checklist_import_attempt_events','existing_checklist_candidates',
        'existing_checklist_candidate_questions','governed_candidate_source_binding_sets',
        'governed_candidate_source_binding_links','regulatory_source_authority_attestations',
        'governed_candidate_source_chain_links','governed_required_owner_resolution_facts',
        'governed_checklist_review_comments','governed_reviewer_recommendation_dispositions',
        'governed_source_mapping_attestations'
    ] LOOP
        EXECUTE format('DROP TRIGGER IF EXISTS %I ON %I', table_name || '_append_only', table_name);
        EXECUTE format('CREATE TRIGGER %I BEFORE UPDATE OR DELETE ON %I FOR EACH ROW EXECUTE FUNCTION governed_v28_append_only_guard()', table_name || '_append_only', table_name);
    END LOOP;
END $$;
