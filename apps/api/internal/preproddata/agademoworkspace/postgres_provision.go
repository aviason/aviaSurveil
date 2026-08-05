package agademoworkspace

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/platform/database"
	"github.com/jackc/pgx/v5"
)

// WorkspaceSchemaDDL is the complete owned schema contract. It does not name
// a canonical application table or the accepted overlay schema. The loader
// and runtime are consequently unable to obtain a cross-schema write path.
const WorkspaceSchemaDDL = `
CREATE SCHEMA IF NOT EXISTS preprod_aga_demo_workspace;
REVOKE ALL ON SCHEMA preprod_aga_demo_workspace FROM PUBLIC;
CREATE TABLE IF NOT EXISTS preprod_aga_demo_workspace.generations (
  generation_id text PRIMARY KEY,
  state text NOT NULL CHECK (state IN ('ACTIVE','RESET')),
  classification_run_id text NOT NULL,
  classification_run_digest text NOT NULL,
  taxonomy_version text NOT NULL,
  taxonomy_digest text NOT NULL,
  fixture_digest text NOT NULL,
  revision integer NOT NULL CHECK (revision > 0),
  seal_digest text NOT NULL,
  created_at timestamptz NOT NULL,
  reset_from_generation_id text NULL
);
CREATE TABLE IF NOT EXISTS preprod_aga_demo_workspace.reset_tombstones (
  tombstone_id text PRIMARY KEY,
  from_generation_id text NOT NULL,
  to_generation_id text NOT NULL,
  expected_generation_id text NOT NULL,
  expected_generation_revision integer NOT NULL,
  expected_generation_seal_digest text NOT NULL,
  reason_code text NOT NULL,
  actor_subject_id text NOT NULL,
  created_at timestamptz NOT NULL,
  tombstone_digest text NOT NULL UNIQUE
);
CREATE TABLE IF NOT EXISTS preprod_aga_demo_workspace.taxonomy_versions (
  taxonomy_version text PRIMARY KEY,
  taxonomy_digest text NOT NULL UNIQUE,
  package_digest text NOT NULL,
  published_at timestamptz NOT NULL,
  sealed boolean NOT NULL
);
CREATE TABLE IF NOT EXISTS preprod_aga_demo_workspace.classification_runs (
  classification_run_id text PRIMARY KEY,
  state text NOT NULL CHECK (state = 'SEALED'),
  taxonomy_version text NOT NULL,
  taxonomy_digest text NOT NULL,
  input_digest text NOT NULL,
  aggregate_digest text NOT NULL,
  classification_run_digest text NOT NULL UNIQUE,
  candidate_record_count integer NOT NULL CHECK (candidate_record_count = 1310),
  challenge_record_count integer NOT NULL CHECK (challenge_record_count = 1310),
  item_count integer NOT NULL CHECK (item_count = 1310),
  payload jsonb NOT NULL,
  created_at timestamptz NOT NULL
);
CREATE TABLE IF NOT EXISTS preprod_aga_demo_workspace.classification_pass_records (
  classification_run_id text NOT NULL,
  pass_role text NOT NULL CHECK (pass_role IN ('CANDIDATE','CHALLENGE')),
  identity_key text NOT NULL,
  pass_run_id text NOT NULL,
  pass_result_digest text NOT NULL,
  payload jsonb NOT NULL,
  canonical_payload text NOT NULL,
  row_digest text NOT NULL,
  PRIMARY KEY (classification_run_id, pass_role, identity_key)
);
CREATE TABLE IF NOT EXISTS preprod_aga_demo_workspace.classification_items (
  classification_run_id text NOT NULL,
  identity_key text NOT NULL,
  payload jsonb NOT NULL,
  canonical_payload text NOT NULL,
  row_digest text NOT NULL,
  PRIMARY KEY (classification_run_id, identity_key)
);
CREATE TABLE IF NOT EXISTS preprod_aga_demo_workspace.classification_involvement_edges (
  classification_run_id text NOT NULL,
  identity_key text NOT NULL,
  edge_key text NOT NULL,
  payload jsonb NOT NULL,
  canonical_payload text NOT NULL,
  row_digest text NOT NULL,
  PRIMARY KEY (classification_run_id, identity_key, edge_key)
);
CREATE TABLE IF NOT EXISTS preprod_aga_demo_workspace.drafts (
  generation_id text NOT NULL,
  draft_id text NOT NULL,
  revision integer NOT NULL CHECK (revision > 0),
  content_digest text NOT NULL,
  state text NOT NULL,
  payload jsonb NOT NULL,
  created_at timestamptz NOT NULL,
  canonical_payload text NOT NULL,
  row_digest text NOT NULL,
  PRIMARY KEY (generation_id, draft_id, revision)
);
ALTER TABLE preprod_aga_demo_workspace.drafts
  ADD COLUMN IF NOT EXISTS created_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP;
CREATE TABLE IF NOT EXISTS preprod_aga_demo_workspace.draft_items (
  generation_id text NOT NULL,
  draft_id text NOT NULL,
  revision integer NOT NULL,
  question_key text NOT NULL,
  payload jsonb NOT NULL,
  canonical_payload text NOT NULL,
  row_digest text NOT NULL,
  PRIMARY KEY (generation_id, draft_id, revision, question_key)
);
CREATE TABLE IF NOT EXISTS preprod_aga_demo_workspace.question_versions (
  generation_id text NOT NULL,
  question_root_id text NOT NULL,
  question_version_id text NOT NULL,
  proposal_id text NOT NULL,
  root_sequence integer NOT NULL CHECK (root_sequence > 0),
  body_digest text NOT NULL,
  body text NOT NULL,
  parent_question_key jsonb NULL,
  actor_subject_id text NOT NULL,
  created_at timestamptz NOT NULL,
  reason_code text NOT NULL,
  current_leaf boolean NOT NULL,
  payload jsonb NOT NULL,
  canonical_payload text NOT NULL,
  row_digest text NOT NULL,
  PRIMARY KEY (generation_id, question_version_id),
  UNIQUE (generation_id, question_root_id, proposal_id),
  UNIQUE (generation_id, question_root_id, root_sequence, question_version_id)
);
CREATE TABLE IF NOT EXISTS preprod_aga_demo_workspace.manager_decisions (
  decision_id text PRIMARY KEY,
  generation_id text NOT NULL,
  draft_id text NOT NULL,
  draft_revision integer NOT NULL,
  question_key text NOT NULL,
  action text NOT NULL,
  disposition text NULL,
  reason_code text NOT NULL,
  actor_subject_id text NOT NULL,
  created_at timestamptz NOT NULL,
  payload jsonb NOT NULL
);
CREATE TABLE IF NOT EXISTS preprod_aga_demo_workspace.batch_previews (
  preview_id text PRIMARY KEY,
  generation_id text NOT NULL,
  draft_id text NOT NULL,
  draft_revision integer NOT NULL,
  preview_digest text NOT NULL UNIQUE,
  expires_at timestamptz NOT NULL,
  consumed_at timestamptz NULL,
  payload jsonb NOT NULL
);
CREATE TABLE IF NOT EXISTS preprod_aga_demo_workspace.authority_bindings (
  binding_id text PRIMARY KEY,
  generation_id text NOT NULL,
  subject_slot text NOT NULL,
  membership_slot text NOT NULL,
  organization_id text NOT NULL,
  department_id text NOT NULL,
  organizational_unit_id text NOT NULL,
  operation_roles jsonb NOT NULL,
  binding_digest text NOT NULL UNIQUE,
  active boolean NOT NULL,
  payload jsonb NOT NULL
);
CREATE TABLE IF NOT EXISTS preprod_aga_demo_workspace.provider_scopes (
  generation_id text NOT NULL,
  provider_scope_root_id text NOT NULL,
  provider_scope_id text NOT NULL,
  provider_scope_version integer NOT NULL,
  provider_type_id text NOT NULL,
  provider_type_code text NOT NULL,
  organization_id text NOT NULL,
  profile_digest text NOT NULL,
  payload jsonb NOT NULL,
  PRIMARY KEY (generation_id, provider_scope_id, provider_scope_version)
);
CREATE TABLE IF NOT EXISTS preprod_aga_demo_workspace.provider_targets (
  generation_id text NOT NULL,
  provider_scope_id text NOT NULL,
  provider_scope_version integer NOT NULL,
  target_id text NOT NULL,
  canonical_target_kind text NOT NULL,
  target_profile_code text NOT NULL,
  payload jsonb NOT NULL,
  PRIMARY KEY (generation_id, provider_scope_id, provider_scope_version, target_id)
);
CREATE TABLE IF NOT EXISTS preprod_aga_demo_workspace.readiness_snapshots (
  readiness_event_id text PRIMARY KEY,
  generation_id text NOT NULL,
  draft_id text NOT NULL,
  draft_revision integer NOT NULL,
  readiness_event_digest text NOT NULL UNIQUE,
  payload jsonb NOT NULL,
  created_at timestamptz NOT NULL
);
CREATE TABLE IF NOT EXISTS preprod_aga_demo_workspace.recommendation_snapshots (
  recommendation_id text PRIMARY KEY,
  generation_id text NOT NULL,
  draft_id text NOT NULL,
  draft_revision integer NOT NULL,
  recommendation_digest text NOT NULL UNIQUE,
  snapshot_digest text NOT NULL UNIQUE,
  payload jsonb NOT NULL,
  created_at timestamptz NOT NULL
);
CREATE TABLE IF NOT EXISTS preprod_aga_demo_workspace.lifecycle_streams (
  lifecycle_id text PRIMARY KEY,
  generation_id text NOT NULL,
  recommendation_id text NOT NULL,
  revision integer NOT NULL CHECK (revision > 0),
  digest text NOT NULL,
  state text NOT NULL,
  payload jsonb NOT NULL,
  created_at timestamptz NOT NULL
);
CREATE TABLE IF NOT EXISTS preprod_aga_demo_workspace.lifecycle_events (
  lifecycle_id text NOT NULL,
  sequence integer NOT NULL CHECK (sequence > 0),
  event_id text NOT NULL UNIQUE,
  operation_id text NOT NULL,
  command_key text NOT NULL,
  event_type text NOT NULL,
  payload jsonb NOT NULL,
  actor_subject_id text NOT NULL,
  created_at timestamptz NOT NULL,
  previous_digest text NOT NULL,
  event_digest text NOT NULL UNIQUE,
  PRIMARY KEY (lifecycle_id, sequence)
);
CREATE TABLE IF NOT EXISTS preprod_aga_demo_workspace.idempotency_responses (
  generation_id text NOT NULL,
  actor_subject_id text NOT NULL,
  operation_id text NOT NULL,
  idempotency_key text NOT NULL,
  command_hash text NOT NULL,
  authorization_scope_digest text NOT NULL,
  status_code integer NOT NULL,
  response jsonb NOT NULL,
  created_at timestamptz NOT NULL,
  PRIMARY KEY (generation_id, actor_subject_id, operation_id, idempotency_key)
);
CREATE TABLE IF NOT EXISTS preprod_aga_demo_workspace.workspace_seals (
  generation_id text PRIMARY KEY,
  classification_run_digest text NOT NULL,
  fixture_digest text NOT NULL,
  workspace_aggregate_digest text NOT NULL,
  seal_digest text NOT NULL UNIQUE,
  sealed_at timestamptz NOT NULL,
  loader_revoked boolean NOT NULL,
  fixture_payload jsonb NOT NULL DEFAULT '{}'::jsonb
);
ALTER TABLE preprod_aga_demo_workspace.workspace_seals
  ADD COLUMN IF NOT EXISTS fixture_payload jsonb NOT NULL DEFAULT '{}'::jsonb;
CREATE TABLE IF NOT EXISTS preprod_aga_demo_workspace.credential_revocation_receipts (
  generation_id text PRIMARY KEY,
  revoked_at timestamptz NOT NULL,
  receipt_digest text NOT NULL UNIQUE
);

CREATE OR REPLACE VIEW preprod_aga_demo_workspace.sealed_generations AS
SELECT generation_id, state, classification_run_id, classification_run_digest,
       taxonomy_version, taxonomy_digest, fixture_digest, revision, seal_digest,
       created_at, reset_from_generation_id
FROM preprod_aga_demo_workspace.generations
WHERE state = 'ACTIVE';
CREATE OR REPLACE VIEW preprod_aga_demo_workspace.sealed_classification_items AS
SELECT classification_run_id, identity_key, payload, canonical_payload, row_digest
FROM preprod_aga_demo_workspace.classification_items;
CREATE OR REPLACE VIEW preprod_aga_demo_workspace.sealed_drafts AS
SELECT generation_id, draft_id, revision, content_digest, state, payload,
       canonical_payload, row_digest
FROM preprod_aga_demo_workspace.drafts;
CREATE OR REPLACE VIEW preprod_aga_demo_workspace.sealed_authority_bindings AS
SELECT binding_id, generation_id, subject_slot, membership_slot, organization_id,
       department_id, organizational_unit_id, operation_roles, binding_digest,
       active, payload
FROM preprod_aga_demo_workspace.authority_bindings
WHERE active;
CREATE OR REPLACE VIEW preprod_aga_demo_workspace.sealed_provider_scopes AS
SELECT generation_id, provider_scope_root_id, provider_scope_id,
       provider_scope_version, provider_type_id, provider_type_code,
       organization_id, profile_digest, payload
FROM preprod_aga_demo_workspace.provider_scopes;
CREATE OR REPLACE VIEW preprod_aga_demo_workspace.sealed_recommendations AS
SELECT recommendation_id, generation_id, draft_id, draft_revision,
       recommendation_digest, snapshot_digest, payload, created_at
FROM preprod_aga_demo_workspace.recommendation_snapshots;
CREATE OR REPLACE VIEW preprod_aga_demo_workspace.sealed_lifecycle_projection AS
SELECT lifecycle_id, generation_id, recommendation_id, revision, digest, state,
       payload, created_at
FROM preprod_aga_demo_workspace.lifecycle_streams;

CREATE OR REPLACE FUNCTION preprod_aga_demo_workspace.reject_mutation() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
  IF TG_TABLE_NAME = 'generations' AND current_setting('preprod_aga_demo_workspace.allow_reset', true) = 'on' THEN
    RETURN NEW;
  END IF;
  RAISE EXCEPTION 'preprod AGA demo workspace rows are append-only';
END $$;
CREATE OR REPLACE FUNCTION preprod_aga_demo_workspace.reject_sealed_load() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
  IF EXISTS (SELECT 1 FROM preprod_aga_demo_workspace.workspace_seals) THEN
    RAISE EXCEPTION 'sealed preprod AGA demo workspace cannot accept loader rows';
  END IF;
  RETURN NEW;
END $$;

CREATE OR REPLACE FUNCTION preprod_aga_demo_workspace.workspace_load(input jsonb)
RETURNS jsonb
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = pg_catalog, preprod_aga_demo_workspace
AS $$
DECLARE
  row jsonb;
BEGIN
  IF jsonb_typeof(input) <> 'object' THEN
    RAISE EXCEPTION 'workspace load payload must be an object';
  END IF;
  IF EXISTS (SELECT 1 FROM preprod_aga_demo_workspace.workspace_seals) THEN
    RAISE EXCEPTION 'sealed preprod AGA demo workspace cannot be loaded twice';
  END IF;
  FOR row IN SELECT value FROM jsonb_array_elements(COALESCE(input->'generations', '[]'::jsonb)) LOOP
    INSERT INTO preprod_aga_demo_workspace.generations
    SELECT (jsonb_populate_record(NULL::preprod_aga_demo_workspace.generations, row)).*;
  END LOOP;
  FOR row IN SELECT value FROM jsonb_array_elements(COALESCE(input->'taxonomyVersions', '[]'::jsonb)) LOOP
    INSERT INTO preprod_aga_demo_workspace.taxonomy_versions
    SELECT (jsonb_populate_record(NULL::preprod_aga_demo_workspace.taxonomy_versions, row)).*;
  END LOOP;
  FOR row IN SELECT value FROM jsonb_array_elements(COALESCE(input->'classificationRuns', '[]'::jsonb)) LOOP
    INSERT INTO preprod_aga_demo_workspace.classification_runs
    SELECT (jsonb_populate_record(NULL::preprod_aga_demo_workspace.classification_runs, row)).*;
  END LOOP;
  FOR row IN SELECT value FROM jsonb_array_elements(COALESCE(input->'classificationPassRecords', '[]'::jsonb)) LOOP
    INSERT INTO preprod_aga_demo_workspace.classification_pass_records
    SELECT (jsonb_populate_record(NULL::preprod_aga_demo_workspace.classification_pass_records, row)).*;
  END LOOP;
  FOR row IN SELECT value FROM jsonb_array_elements(COALESCE(input->'classificationItems', '[]'::jsonb)) LOOP
    INSERT INTO preprod_aga_demo_workspace.classification_items
    SELECT (jsonb_populate_record(NULL::preprod_aga_demo_workspace.classification_items, row)).*;
  END LOOP;
  FOR row IN SELECT value FROM jsonb_array_elements(COALESCE(input->'drafts', '[]'::jsonb)) LOOP
    INSERT INTO preprod_aga_demo_workspace.drafts
    SELECT (jsonb_populate_record(NULL::preprod_aga_demo_workspace.drafts, row)).*;
  END LOOP;
  FOR row IN SELECT value FROM jsonb_array_elements(COALESCE(input->'authorityBindings', '[]'::jsonb)) LOOP
    INSERT INTO preprod_aga_demo_workspace.authority_bindings
    SELECT (jsonb_populate_record(NULL::preprod_aga_demo_workspace.authority_bindings, row)).*;
  END LOOP;
  FOR row IN SELECT value FROM jsonb_array_elements(COALESCE(input->'providerScopes', '[]'::jsonb)) LOOP
    INSERT INTO preprod_aga_demo_workspace.provider_scopes
    SELECT (jsonb_populate_record(NULL::preprod_aga_demo_workspace.provider_scopes, row)).*;
  END LOOP;
  FOR row IN SELECT value FROM jsonb_array_elements(COALESCE(input->'providerTargets', '[]'::jsonb)) LOOP
    INSERT INTO preprod_aga_demo_workspace.provider_targets
    SELECT (jsonb_populate_record(NULL::preprod_aga_demo_workspace.provider_targets, row)).*;
  END LOOP;
  FOR row IN SELECT value FROM jsonb_array_elements(COALESCE(input->'workspaceSeals', '[]'::jsonb)) LOOP
    INSERT INTO preprod_aga_demo_workspace.workspace_seals
    SELECT (jsonb_populate_record(NULL::preprod_aga_demo_workspace.workspace_seals, row)).*;
  END LOOP;
  RETURN jsonb_build_object('status', 'SEALED');
END $$;

CREATE OR REPLACE FUNCTION preprod_aga_demo_workspace.workspace_query(input jsonb)
RETURNS jsonb
LANGUAGE plpgsql
STABLE
SECURITY DEFINER
SET search_path = pg_catalog, preprod_aga_demo_workspace
AS $$
DECLARE
  operation text := COALESCE(input->>'operation', 'SNAPSHOT');
  selected_generation text;
BEGIN
  SELECT generation_id INTO selected_generation
  FROM preprod_aga_demo_workspace.generations
  WHERE state = 'ACTIVE'
  ORDER BY created_at DESC
  LIMIT 1;

  IF operation = 'RUNTIME_COMMAND_META' THEN
    RETURN jsonb_build_object(
      'generation', (SELECT jsonb_build_object(
        'generationId', generation_id, 'state', state,
        'classificationRunId', classification_run_id,
        'classificationRunDigest', classification_run_digest,
        'taxonomyVersion', taxonomy_version, 'taxonomyDigest', taxonomy_digest,
        'fixtureDigest', fixture_digest, 'revision', revision,
        'sealDigest', seal_digest)
        FROM preprod_aga_demo_workspace.generations
        WHERE generation_id = selected_generation),
      'draft', (SELECT jsonb_build_object(
        'draftId', draft_id, 'revision', revision, 'contentDigest', content_digest)
        FROM preprod_aga_demo_workspace.drafts
        WHERE generation_id = selected_generation ORDER BY revision DESC LIMIT 1));
  ELSIF operation = 'RUNTIME_COMMAND_STATE' THEN
    RETURN jsonb_build_object(
      'generation', (SELECT jsonb_build_object(
        'generationId', generation_id, 'state', state,
        'classificationRunId', classification_run_id,
        'classificationRunDigest', classification_run_digest,
        'taxonomyVersion', taxonomy_version, 'taxonomyDigest', taxonomy_digest,
        'fixtureDigest', fixture_digest, 'revision', revision,
        'sealDigest', seal_digest, 'createdAt', created_at,
        'resetFromGenerationId', reset_from_generation_id)
        FROM preprod_aga_demo_workspace.generations
        WHERE generation_id = selected_generation),
      'draft', (SELECT jsonb_build_object('draft', payload, 'createdAt', created_at)
        FROM preprod_aga_demo_workspace.drafts
        WHERE generation_id = selected_generation ORDER BY revision DESC LIMIT 1),
      'seal', (SELECT jsonb_build_object(
        'generationId', seal.generation_id, 'classificationRunDigest', seal.classification_run_digest,
        'fixtureDigest', seal.fixture_digest, 'workspaceAggregateDigest', seal.workspace_aggregate_digest,
        'sealDigest', seal.seal_digest, 'sealedAt', seal.sealed_at,
        'loaderRevoked', (seal.loader_revoked OR EXISTS (
          SELECT 1 FROM preprod_aga_demo_workspace.credential_revocation_receipts receipt
          WHERE receipt.generation_id = seal.generation_id)))
        FROM preprod_aga_demo_workspace.workspace_seals seal
        WHERE seal.generation_id = selected_generation));
  ELSIF operation = 'GET_DRAFT' THEN
    RETURN COALESCE((SELECT payload FROM preprod_aga_demo_workspace.drafts
      WHERE generation_id = COALESCE(input->>'generationId', selected_generation)
      ORDER BY revision DESC LIMIT 1), '{}'::jsonb);
  ELSIF operation = 'GET_DRAFT_BASE' THEN
    RETURN COALESCE((SELECT payload FROM preprod_aga_demo_workspace.drafts
      WHERE generation_id = COALESCE(input->>'generationId', selected_generation)
      ORDER BY revision ASC LIMIT 1), '{}'::jsonb);
  ELSIF operation = 'GET_RECOMMENDATION' THEN
    RETURN COALESCE((SELECT payload FROM preprod_aga_demo_workspace.recommendation_snapshots
      WHERE generation_id = input->>'generationId' AND recommendation_id = input->>'recommendationId'), 'null'::jsonb);
  ELSIF operation = 'GET_LIFECYCLE_EVENTS' THEN
    RETURN COALESCE((SELECT jsonb_agg(jsonb_build_object(
      'eventId', event_id, 'lifecycleId', lifecycle_id, 'sequence', sequence,
      'operationId', operation_id, 'commandKey', command_key, 'eventType', event_type,
      'payload', payload, 'actorSubjectId', actor_subject_id, 'createdAt', created_at,
      'previousDigest', previous_digest, 'eventDigest', event_digest) ORDER BY sequence)
      FROM preprod_aga_demo_workspace.lifecycle_events
      WHERE lifecycle_id = input->>'lifecycleId' AND payload->>'generationId' = input->>'generationId'), '[]'::jsonb);
  ELSIF operation = 'GET_IDEMPOTENCY' THEN
    RETURN COALESCE((SELECT jsonb_build_object(
      'generationId', generation_id, 'actorSubjectId', actor_subject_id,
      'operationId', operation_id, 'idempotencyKey', idempotency_key,
      'commandHash', command_hash, 'authorizationScopeDigest', authorization_scope_digest,
      'statusCode', status_code, 'response', response, 'createdAt', created_at)
      FROM preprod_aga_demo_workspace.idempotency_responses
      WHERE generation_id = input->>'generationId'
        AND actor_subject_id = input->>'actorSubjectId'
        AND operation_id = input->>'operationId'
        AND idempotency_key = input->>'idempotencyKey'), 'null'::jsonb);
  END IF;

  RETURN jsonb_build_object(
    'generation', (SELECT jsonb_build_object(
      'generationId', generation_id, 'state', state,
      'classificationRunId', classification_run_id,
      'classificationRunDigest', classification_run_digest,
      'taxonomyVersion', taxonomy_version, 'taxonomyDigest', taxonomy_digest,
      'fixtureDigest', fixture_digest, 'revision', revision,
      'sealDigest', seal_digest, 'createdAt', created_at,
      'resetFromGenerationId', reset_from_generation_id)
      FROM preprod_aga_demo_workspace.generations
      WHERE generation_id = selected_generation),
    'taxonomy', (SELECT jsonb_build_object(
      'taxonomyVersion', taxonomy_version, 'taxonomyDigest', taxonomy_digest,
      'packageDigest', package_digest, 'publishedAt', published_at, 'sealed', sealed)
      FROM preprod_aga_demo_workspace.taxonomy_versions ORDER BY published_at LIMIT 1),
    'run', (SELECT jsonb_build_object(
      'runId', classification_run_id, 'state', state,
      'taxonomyVersion', taxonomy_version, 'taxonomyDigest', taxonomy_digest,
      'inputDigest', input_digest, 'aggregateDigest', aggregate_digest,
      'classificationRunDigest', classification_run_digest, 'result', payload,
      'candidateRecordCount', candidate_record_count,
      'challengeRecordCount', challenge_record_count, 'itemCount', item_count,
      'createdAt', created_at)
      FROM preprod_aga_demo_workspace.classification_runs ORDER BY created_at LIMIT 1),
    'items', COALESCE((SELECT jsonb_agg(payload ORDER BY identity_key)
      FROM preprod_aga_demo_workspace.classification_items), '[]'::jsonb),
    'candidateRecords', COALESCE((SELECT jsonb_agg(payload ORDER BY identity_key)
      FROM preprod_aga_demo_workspace.classification_pass_records WHERE pass_role = 'CANDIDATE'), '[]'::jsonb),
    'challengeRecords', COALESCE((SELECT jsonb_agg(payload ORDER BY identity_key)
      FROM preprod_aga_demo_workspace.classification_pass_records WHERE pass_role = 'CHALLENGE'), '[]'::jsonb),
    'draft', (SELECT jsonb_build_object('draft', payload, 'createdAt', created_at)
      FROM preprod_aga_demo_workspace.drafts
      WHERE generation_id = selected_generation ORDER BY revision DESC LIMIT 1),
    'fixture', (SELECT fixture_payload FROM preprod_aga_demo_workspace.workspace_seals
      WHERE generation_id = selected_generation),
    'seal', (SELECT jsonb_build_object(
      'generationId', seal.generation_id, 'classificationRunDigest', seal.classification_run_digest,
      'fixtureDigest', seal.fixture_digest, 'workspaceAggregateDigest', seal.workspace_aggregate_digest,
      'sealDigest', seal.seal_digest, 'sealedAt', seal.sealed_at,
      'loaderRevoked', (seal.loader_revoked OR EXISTS (
        SELECT 1 FROM preprod_aga_demo_workspace.credential_revocation_receipts receipt
        WHERE receipt.generation_id = seal.generation_id
      )))
      FROM preprod_aga_demo_workspace.workspace_seals seal
      WHERE seal.generation_id = selected_generation)
  );
END $$;

CREATE OR REPLACE FUNCTION preprod_aga_demo_workspace.workspace_command(input jsonb)
RETURNS jsonb
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = pg_catalog, preprod_aga_demo_workspace
AS $$
BEGIN
  IF input->>'operation' = 'APPEND_DRAFT' THEN
    INSERT INTO preprod_aga_demo_workspace.drafts
      (generation_id,draft_id,revision,content_digest,state,payload,created_at,canonical_payload,row_digest)
    VALUES (input->>'generationId', input->>'draftId', (input->>'revision')::integer,
      input->>'contentDigest', input->>'state', input->'payload', (input->>'createdAt')::timestamptz,
      input->>'canonicalPayload', input->>'rowDigest');
    RETURN jsonb_build_object('status', 'APPENDED');
  ELSIF input->>'operation' = 'APPEND_RECOMMENDATION' THEN
    INSERT INTO preprod_aga_demo_workspace.recommendation_snapshots
      (recommendation_id,generation_id,draft_id,draft_revision,recommendation_digest,snapshot_digest,payload,created_at)
    VALUES (input->>'recommendationId', input->>'generationId', input->>'draftId', (input->>'draftRevision')::integer,
      input->>'recommendationDigest', input->>'snapshotDigest', input->'payload', (input->>'createdAt')::timestamptz);
    RETURN jsonb_build_object('status', 'APPENDED');
  ELSIF input->>'operation' = 'APPEND_IDEMPOTENCY' THEN
    INSERT INTO preprod_aga_demo_workspace.idempotency_responses
      (generation_id,actor_subject_id,operation_id,idempotency_key,command_hash,authorization_scope_digest,status_code,response,created_at)
    VALUES (input->>'generationId', input->>'actorSubjectId', input->>'operationId', input->>'idempotencyKey',
      input->>'commandHash', input->>'authorizationScopeDigest', (input->>'statusCode')::integer, input->'response', (input->>'createdAt')::timestamptz);
    RETURN jsonb_build_object('status', 'APPENDED');
  ELSIF input->>'operation' = 'APPEND_LIFECYCLE' THEN
    INSERT INTO preprod_aga_demo_workspace.lifecycle_streams
      (lifecycle_id,generation_id,recommendation_id,revision,digest,state,payload,created_at)
    SELECT input->>'lifecycleId', input->>'generationId', input->>'recommendationId', (input->>'revision')::integer,
      input->>'aggregateDigest', input->>'state', input->'payload', (input->>'createdAt')::timestamptz
    WHERE (input->>'sequence')::integer = 1;
    INSERT INTO preprod_aga_demo_workspace.lifecycle_events
      (lifecycle_id,sequence,event_id,operation_id,command_key,event_type,payload,actor_subject_id,created_at,previous_digest,event_digest)
    VALUES (input->>'lifecycleId', (input->>'sequence')::integer, input->>'eventId', input->>'operationId',
      input->>'commandKey', input->>'eventType', input->'payload', input->>'actorSubjectId',
      (input->>'createdAt')::timestamptz, input->>'previousDigest', input->>'eventDigest');
    RETURN jsonb_build_object('status', 'APPENDED');
  ELSIF input->>'operation' = 'APPEND_QUESTION_VERSION' THEN
    INSERT INTO preprod_aga_demo_workspace.question_versions
      (generation_id,question_root_id,question_version_id,proposal_id,root_sequence,body_digest,body,parent_question_key,actor_subject_id,created_at,reason_code,current_leaf,payload,canonical_payload,row_digest)
    VALUES (input->>'generationId', input->>'rootId', input->>'versionId', input->>'proposalId', (input->>'rootSequence')::integer,
      input->>'bodyDigest', input->>'body', input->'parentQuestionKey', input->>'actorSubjectId',
      (input->>'createdAt')::timestamptz, input->>'reasonCode', true, input->'payload', input->>'canonicalPayload', input->>'rowDigest');
    RETURN jsonb_build_object('status', 'APPENDED');
  END IF;
  RAISE EXCEPTION 'unknown workspace command operation';
END $$;

CREATE OR REPLACE FUNCTION preprod_aga_demo_workspace.workspace_reset(input jsonb)
RETURNS jsonb
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = pg_catalog, preprod_aga_demo_workspace
AS $$
DECLARE
  current_generation preprod_aga_demo_workspace.generations%ROWTYPE;
BEGIN
  SELECT * INTO current_generation
  FROM preprod_aga_demo_workspace.generations
  WHERE state = 'ACTIVE'
  ORDER BY created_at DESC
  LIMIT 1;
  IF current_generation.generation_id IS NULL
     OR current_generation.generation_id <> input->>'expectedGenerationId'
     OR current_generation.revision <> (input->>'expectedRevision')::integer
     OR current_generation.seal_digest <> input->>'expectedSealDigest' THEN
    RAISE EXCEPTION 'workspace reset compare-and-swap conflict';
  END IF;
  IF EXISTS (
  SELECT 1 FROM preprod_aga_demo_workspace.lifecycle_streams
    JOIN LATERAL (
      SELECT event.payload
      FROM preprod_aga_demo_workspace.lifecycle_events event
      WHERE event.lifecycle_id = lifecycle_streams.lifecycle_id
      ORDER BY event.sequence DESC
      LIMIT 1
    ) latest_event ON true
    WHERE lifecycle_streams.generation_id = input->>'expectedGenerationId'
      AND (
        latest_event.payload->>'state' <> 'COMPLETED'
        OR EXISTS (SELECT 1 FROM jsonb_array_elements(COALESCE(latest_event.payload->'findings', '[]'::jsonb)) finding WHERE finding->>'state' <> 'CLOSED')
        OR EXISTS (
          SELECT 1
          FROM (
            SELECT DISTINCT ON (value->>'rootId') value AS potential
            FROM jsonb_array_elements(COALESCE(latest_event.payload->'potentialFindings', '[]'::jsonb)) WITH ORDINALITY AS potential_rows(value, ordinal)
            ORDER BY value->>'rootId', ordinal DESC
          ) latest_potential
          WHERE latest_potential.potential->>'state' NOT IN ('DISMISSED', 'CONVERTED')
        )
        OR EXISTS (
          SELECT 1
          FROM (
            SELECT DISTINCT ON (value->>'findingId') value AS cap
            FROM jsonb_array_elements(COALESCE(latest_event.payload->'capRevisions', '[]'::jsonb)) WITH ORDINALITY AS cap_rows(value, ordinal)
            ORDER BY value->>'findingId', ordinal DESC
          ) latest_cap
          WHERE latest_cap.cap->>'state' IN ('SUBMITTED', 'PENDING_CAA_REVIEW', 'REJECTED', 'MORE_INFORMATION_REQUESTED')
        )
        OR EXISTS (
          SELECT 1
          FROM (
            SELECT DISTINCT ON (value->>'findingId') value AS evidence
            FROM jsonb_array_elements(COALESCE(latest_event.payload->'evidenceVersions', '[]'::jsonb)) WITH ORDINALITY AS evidence_rows(value, ordinal)
            ORDER BY value->>'findingId', ordinal DESC
          ) latest_evidence
          WHERE latest_evidence.evidence->>'reviewState' IN ('PENDING_CAA_REVIEW', 'PARTIALLY_ACCEPTED', 'REJECTED', 'MORE_INFORMATION_REQUESTED')
        )
      )
  ) THEN
    RAISE EXCEPTION 'workspace reset requires terminal lifecycle state';
  END IF;
  PERFORM set_config('preprod_aga_demo_workspace.allow_reset', 'on', true);
  UPDATE preprod_aga_demo_workspace.generations
    SET state = 'RESET'
    WHERE generation_id = input->>'expectedGenerationId';
  INSERT INTO preprod_aga_demo_workspace.reset_tombstones
    (tombstone_id,from_generation_id,to_generation_id,expected_generation_id,expected_generation_revision,expected_generation_seal_digest,reason_code,actor_subject_id,created_at,tombstone_digest)
  VALUES (input->>'tombstoneId', input->>'fromGenerationId', input->>'toGenerationId', input->>'expectedGenerationId',
    (input->>'expectedRevision')::integer, input->>'expectedSealDigest', input->>'reasonCode', input->>'actorSubjectId',
    (input->>'createdAt')::timestamptz, input->>'tombstoneDigest');
  INSERT INTO preprod_aga_demo_workspace.generations
    (generation_id,state,classification_run_id,classification_run_digest,taxonomy_version,taxonomy_digest,fixture_digest,revision,seal_digest,created_at,reset_from_generation_id)
  VALUES (input->>'toGenerationId', 'ACTIVE', input->>'classificationRunId', input->>'classificationRunDigest',
    input->>'taxonomyVersion', input->>'taxonomyDigest', input->>'fixtureDigest', 1, input->>'newGenerationSealDigest',
    (input->>'createdAt')::timestamptz, input->>'fromGenerationId');
  INSERT INTO preprod_aga_demo_workspace.drafts
    (generation_id,draft_id,revision,content_digest,state,payload,created_at,canonical_payload,row_digest)
  VALUES (input->>'toGenerationId', input->'draft'->>'draftId', 1, input->'draft'->>'contentDigest', 'WORKING',
    input->'draft', (input->>'createdAt')::timestamptz, input->>'draftCanonicalPayload', input->>'draftRowDigest');
  INSERT INTO preprod_aga_demo_workspace.workspace_seals
    (generation_id,classification_run_digest,fixture_digest,workspace_aggregate_digest,seal_digest,sealed_at,loader_revoked,fixture_payload)
  SELECT input->>'toGenerationId', classification_run_digest, fixture_digest, workspace_aggregate_digest,
    input->>'newWorkspaceSealDigest', (input->>'createdAt')::timestamptz, true, fixture_payload
  FROM preprod_aga_demo_workspace.workspace_seals
  WHERE generation_id = input->>'fromGenerationId';
  RETURN jsonb_build_object('status', 'RESET', 'generationId', input->>'toGenerationId');
END $$;
`

// WorkspaceAppendOnlyTriggerDDL installs the mutation barrier on every
// append-only family. Reset tombstones and idempotency responses are included;
// a response is inserted once and replayed, never updated in place.
func WorkspaceAppendOnlyTriggerDDL() string {
	const relations = `generations reset_tombstones taxonomy_versions classification_runs classification_pass_records classification_items classification_involvement_edges drafts draft_items question_versions manager_decisions batch_previews authority_bindings provider_scopes provider_targets readiness_snapshots recommendation_snapshots lifecycle_streams lifecycle_events idempotency_responses workspace_seals credential_revocation_receipts`
	var statements []string
	for _, relation := range strings.Fields(relations) {
		statements = append(statements, fmt.Sprintf("DROP TRIGGER IF EXISTS %s_append_only ON %s.%s; CREATE TRIGGER %s_append_only BEFORE UPDATE OR DELETE ON %s.%s FOR EACH ROW EXECUTE FUNCTION %s.reject_mutation();", relation, WorkspaceSchemaName, relation, relation, WorkspaceSchemaName, relation, WorkspaceSchemaName))
	}
	return strings.Join(statements, "\n")
}

func WorkspaceLoaderBarrierDDL() string {
	return fmt.Sprintf(`
DROP TRIGGER IF EXISTS classification_pass_records_after_seal ON %s.classification_pass_records;
CREATE TRIGGER classification_pass_records_after_seal BEFORE INSERT ON %s.classification_pass_records FOR EACH ROW EXECUTE FUNCTION %s.reject_sealed_load();
DROP TRIGGER IF EXISTS classification_items_after_seal ON %s.classification_items;
CREATE TRIGGER classification_items_after_seal BEFORE INSERT ON %s.classification_items FOR EACH ROW EXECUTE FUNCTION %s.reject_sealed_load();
`, WorkspaceSchemaName, WorkspaceSchemaName, WorkspaceSchemaName,
		WorkspaceSchemaName, WorkspaceSchemaName, WorkspaceSchemaName)
}

type WorkspacePasswords struct {
	Exporter string
	Loader   string
	Reader   string
	Command  string
}

func (passwords WorkspacePasswords) Validate() error {
	if strings.TrimSpace(passwords.Exporter) == "" || strings.TrimSpace(passwords.Loader) == "" || strings.TrimSpace(passwords.Reader) == "" || strings.TrimSpace(passwords.Command) == "" {
		return fmt.Errorf("%w: workspace passwords", ErrWorkspaceContract)
	}
	return nil
}

// ProvisionWorkspaceSchema is a bootstrap-only operation. It is intentionally
// not called by the tagged API, and it never accepts the normal application
// pool as a runtime dependency.
func ProvisionWorkspaceSchema(ctx context.Context, pool *database.Pool, passwords WorkspacePasswords) error {
	if pool == nil {
		return fmt.Errorf("workspace bootstrap PostgreSQL pool is required")
	}
	if err := passwords.Validate(); err != nil {
		return err
	}
	if err := ValidateWorkspaceRoleMatrix(WorkspaceRoleMatrix()); err != nil {
		return err
	}
	return database.WithinTransaction(ctx, pool, func(ctx context.Context, tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, WorkspaceRoleDDL()); err != nil {
			return fmt.Errorf("create workspace roles: %w", err)
		}
		for _, value := range []struct{ role, password string }{
			{WorkspaceExporterRole, passwords.Exporter}, {WorkspaceLoaderRole, passwords.Loader},
			{WorkspaceReaderRole, passwords.Reader}, {WorkspaceCommandRole, passwords.Command},
		} {
			password := strings.ReplaceAll(value.password, "'", "''")
			if _, err := tx.Exec(ctx, "ALTER ROLE "+value.role+" PASSWORD '"+password+"'"); err != nil {
				return fmt.Errorf("set workspace role password: %w", err)
			}
		}
		if _, err := tx.Exec(ctx, WorkspaceSchemaDDL); err != nil {
			return fmt.Errorf("create workspace schema: %w", err)
		}
		if _, err := tx.Exec(ctx, WorkspaceAppendOnlyTriggerDDL()); err != nil {
			return fmt.Errorf("install workspace append-only triggers: %w", err)
		}
		if _, err := tx.Exec(ctx, WorkspaceLoaderBarrierDDL()); err != nil {
			return fmt.Errorf("install workspace loader barriers: %w", err)
		}
		if _, err := tx.Exec(ctx, WorkspaceRuntimeGrantDDL()); err != nil {
			return fmt.Errorf("grant workspace runtime permissions: %w", err)
		}
		return nil
	})
}

func RevokeWorkspaceOneShotLogins(ctx context.Context, pool *database.Pool) error {
	if pool == nil {
		return fmt.Errorf("workspace bootstrap PostgreSQL pool is required")
	}
	return database.WithinTransaction(ctx, pool, func(ctx context.Context, tx pgx.Tx) error {
		_, err := tx.Exec(ctx, "ALTER ROLE "+WorkspaceExporterRole+" NOLOGIN; ALTER ROLE "+WorkspaceLoaderRole+" NOLOGIN;")
		if err != nil {
			return fmt.Errorf("revoke workspace one-shot logins: %w", err)
		}
		var generationID string
		err = tx.QueryRow(ctx, "SELECT generation_id FROM "+WorkspaceSchemaName+".workspace_seals ORDER BY sealed_at DESC LIMIT 1").Scan(&generationID)
		if err == pgx.ErrNoRows {
			return nil
		}
		if err != nil {
			return fmt.Errorf("read sealed workspace for revocation receipt: %w", err)
		}
		revokedAt := time.Now().UTC()
		digestBytes := sha256.Sum256([]byte("AGA-DEMO-WORKSPACE-CREDENTIAL-REVOCATION-V1\n" + generationID + "\n" + revokedAt.Format(time.RFC3339Nano)))
		receiptDigest := "sha256:" + hex.EncodeToString(digestBytes[:])
		_, err = tx.Exec(ctx, "INSERT INTO "+WorkspaceSchemaName+".credential_revocation_receipts (generation_id, revoked_at, receipt_digest) VALUES ($1, $2, $3) ON CONFLICT (generation_id) DO NOTHING", generationID, revokedAt, receiptDigest)
		if err != nil {
			return fmt.Errorf("record workspace credential revocation: %w", err)
		}
		return nil
	})
}

func WorkspaceSchemaObjectNames() []string {
	return strings.Fields("generations reset_tombstones taxonomy_versions classification_runs classification_pass_records classification_items classification_involvement_edges drafts draft_items question_versions manager_decisions batch_previews authority_bindings provider_scopes provider_targets readiness_snapshots recommendation_snapshots lifecycle_streams lifecycle_events idempotency_responses workspace_seals credential_revocation_receipts")
}
