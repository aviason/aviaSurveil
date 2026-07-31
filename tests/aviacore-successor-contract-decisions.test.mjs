import assert from 'node:assert/strict';
import { existsSync, readFileSync } from 'node:fs';
import path from 'node:path';
import test from 'node:test';

const root = path.resolve(import.meta.dirname, '..');
const decisionsPath = path.join(root, 'docs/product-specs/data-and-rules/aviacore-successor-contract-decisions.json');
const aviaCoreRoot = process.env.AVIACORE_ROOT || '/Users/marlonjd/Developer/monorepos/aviaCore';
const coverage = JSON.parse(readFileSync(path.join(root, 'docs/product-specs/data-and-rules/aviacore-data-feed-coverage.json'), 'utf8'));

test('Task 2 successor decisions exist before any v3 cut or producer code generation', () => {
  assert.ok(existsSync(decisionsPath), 'missing owner-approved successor contract decision package');
  assert.ok(existsSync(path.join(root, 'docs/product-specs/data-and-rules/AVIACORE_SUCCESSOR_CONTRACT_DECISIONS.md')));
});

test('every Task 1 extension candidate and predecessor contradiction has one owner-approved disposition', () => {
  const decisions = JSON.parse(readFileSync(decisionsPath, 'utf8'));
  assert.equal(decisions.schema_version, 'aviacore_successor_contract_decisions.v1');
  assert.equal(decisions.owner.name, 'Burak Karahan');
  assert.equal(decisions.owner.role, 'owner');
  const candidates = coverage.relation_dispositions.filter((item) => item.disposition === 'contract_extension_candidate').map((item) => item.id).sort();
  assert.deepEqual(decisions.extension_decisions.map((item) => item.coverage_policy_id).sort(), candidates);
  for (const item of decisions.extension_decisions) {
    for (const field of ['purpose', 'authority', 'grain', 'privacy_class', 'retention_class', 'correction_rule', 'owner', 'successor_disposition']) assert.match(item[field], /\S/);
    assert.notEqual(item.successor_disposition, 'approved_data_product_derivation', 'a product cannot substitute for source transport');
  }
  assert.deepEqual(decisions.predecessor_contradictions.map((item) => item.id).sort(), ['correction_supersession_scope', 'hash_projection_registry', 'negative_branch_vectors', 'tombstone_replay_suppression']);
});

test('v3 path, zero overlap, event-api backfill, immutable retention, and non-training boundaries are closed', () => {
  const decisions = JSON.parse(readFileSync(decisionsPath, 'utf8'));
  assert.equal(decisions.successor.root, 'contracts/aviasurveil-production/v3/');
  assert.equal(decisions.successor.existing_predecessor_mutation, 'forbidden');
  assert.equal(decisions.compatibility.overlap, 'none_not_deployed');
  assert.equal(decisions.compatibility.rollback, 'forward_fix_only');
  assert.equal(decisions.bootstrap.mode, 'historical_event_api_backfill_from_source_consistent_cut');
  assert.equal(decisions.bootstrap.snapshot_delivery, 'disabled');
  assert.equal(decisions.retention.canonical_retention, 'indefinite_immutable');
  assert.equal(decisions.retention.tombstone, 'replay_and_publication_suppression_only');
  assert.equal(decisions.retention.tombstone_after_legal_hold_release_days, 30);
  assert.equal(decisions.ml.training_allowed, false);
  assert.equal(decisions.ml.production_ml_readiness, 'NOT_READY');
});

test('owner-approved v3 extension event families have closed minimised field contracts', () => {
  const decisions = JSON.parse(readFileSync(decisionsPath, 'utf8'));
  assert.deepEqual(Object.keys(decisions.extension_event_contracts).sort(), [
    'assignment_package',
    'configuration_reference_versions',
    'governed_checklist_generation',
    'organization_provider_scope',
    'planning_reminders',
    'potential_findings_report_decisions',
  ]);
  assert.deepEqual(decisions.common_envelope.required_fields, [
    'event_id', 'event_type', 'occurred_at', 'effective_at', 'known_at',
    'source_system', 'tenant_id', 'owning_organization_id',
    'actor_organization_id', 'visibility_purpose_code', 'correlation_id',
    'causation_id', 'aggregate_type', 'aggregate_id', 'aggregate_revision',
    'payload_sha256',
  ]);
  assert.equal(decisions.common_envelope.additional_properties, false);
  for (const family of Object.values(decisions.extension_event_contracts)) {
    assert.ok(family.event_names.length > 0);
    assert.equal(family.additional_properties, false);
    assert.ok(family.required_fields.length > 0);
    assert.ok(family.forbidden_fields.length > 0);
    assert.match(family.entity_ref, /tenant_id/);
    assert.match(family.grain, /one immutable/i);
  }
  assert.deepEqual(
    decisions.extension_event_contracts.planning_reminders.event_names,
    ['surveillance_plan.versioned', 'surveillance_plan.approval_recorded', 'surveillance_plan.status_changed', 'reminder.dispatch_recorded'],
  );
  assert.equal(decisions.question_snapshot_contract.max_prompt_bytes, 4096);
  assert.equal(decisions.question_snapshot_contract.additional_properties, false);
  assert.deepEqual(decisions.question_snapshot_contract.required_fields, [
    'question_id', 'ordinal', 'prompt', 'mapping_reference_ids',
    'citation_reference_ids', 'verification_method_code',
    'expected_evidence_type_codes', 'allowed_answer_codes', 'mandatory',
    'safety_critical',
  ]);
});

test('AviaCore predecessor roots are read-only inputs and Task 3 remains separately authorized', () => {
  const decisions = JSON.parse(readFileSync(decisionsPath, 'utf8'));
  for (const relativePath of decisions.predecessor_read_only_paths) assert.ok(existsSync(path.join(aviaCoreRoot, relativePath)), `missing predecessor input ${relativePath}`);
  assert.equal(decisions.execution_gates.aviacore_v3_cut, 'separately_authorized_task_3a');
  assert.equal(decisions.execution_gates.producer_mirror_codegen, 'separately_authorized_task_3b');
  assert.equal(decisions.execution_gates.aviacore_phase_2_3, 'separately_authorized_after_3a_and_3b');
});
