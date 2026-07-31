import assert from 'node:assert/strict';
import { createHash } from 'node:crypto';
import { existsSync, readdirSync, readFileSync } from 'node:fs';
import path from 'node:path';
import test from 'node:test';

const root = path.resolve(import.meta.dirname, '..');
const registerPath = path.join(root, 'docs/product-specs/data-and-rules/aviacore-data-feed-coverage.json');
const registerGuidePath = path.join(root, 'docs/product-specs/data-and-rules/AVIACORE_DATA_FEED_COVERAGE.md');
const aviaCoreRoot = process.env.AVIACORE_ROOT || '/Users/marlonjd/Developer/monorepos/aviaCore';

function sha256(file) {
  return createHash('sha256').update(readFileSync(file)).digest('hex');
}

function sourceFiles(directory, predicate) {
  return readdirSync(directory).filter(predicate).sort().map((name) => path.join(directory, name));
}

function sourceFingerprint(relativePath) {
  return { path: relativePath, sha256: sha256(path.join(root, relativePath)) };
}

function createdRelations() {
  const relations = new Set();
  for (const file of sourceFiles(path.join(root, 'apps/api/migrations'), (name) => name.endsWith('.up.sql'))) {
    const sql = readFileSync(file, 'utf8');
    for (const match of sql.matchAll(/CREATE TABLE(?: IF NOT EXISTS)?\s+([a-z_]+)/g)) relations.add(match[1]);
  }
  return [...relations].sort();
}

function splitSqlItems(value) {
  const items = []; let start = 0; let depth = 0; let quote = false;
  for (let index = 0; index < value.length; index += 1) {
    const character = value[index];
    if (character === "'" && value[index - 1] !== '\\') quote = !quote;
    if (!quote && character === '(') depth += 1;
    if (!quote && character === ')') depth -= 1;
    if (!quote && depth === 0 && character === ',') { items.push(value.slice(start, index)); start = index + 1; }
  }
  items.push(value.slice(start));
  return items.map((item) => item.trim()).filter(Boolean);
}

function currentColumns() {
  const schema = new Map();
  for (const file of sourceFiles(path.join(root, 'apps/api/migrations'), (name) => name.endsWith('.up.sql'))) {
    const sql = readFileSync(file, 'utf8');
    for (const match of sql.matchAll(/CREATE TABLE(?: IF NOT EXISTS)?\s+([a-z_]+)\s*\(([\s\S]*?)\);/g)) {
      const columns = new Set();
      for (const item of splitSqlItems(match[2])) {
        const column = item.match(/^([a-z_][a-z0-9_]*)\s+/i)?.[1];
        if (column && !/^(primary|foreign|unique|check|constraint|exclude)$/i.test(column)) columns.add(column);
      }
      schema.set(match[1], columns);
    }
    for (const match of sql.matchAll(/ALTER TABLE\s+([a-z_]+)\s+([\s\S]*?);/g)) {
      const columns = schema.get(match[1]) || new Set();
      for (const item of splitSqlItems(match[2])) {
        const add = item.match(/^ADD COLUMN(?: IF NOT EXISTS)?\s+([a-z_]+)/i)?.[1];
        const drop = item.match(/^DROP COLUMN(?: IF EXISTS)?\s+([a-z_]+)/i)?.[1];
        const rename = item.match(/^RENAME COLUMN\s+([a-z_]+)\s+TO\s+([a-z_]+)/i);
        if (add) columns.add(add);
        if (drop) columns.delete(drop);
        if (rename) { columns.delete(rename[1]); columns.add(rename[2]); }
      }
      schema.set(match[1], columns);
    }
  }
  return [...schema.entries()].flatMap(([relation, columns]) => [...columns].sort().map((column) => `${relation}.${column}`)).sort();
}

function operationIds() {
  const ids = new Set();
  for (const file of sourceFiles(path.join(root, 'api/openapi/source/paths'), (name) => name.endsWith('.json'))) {
    const document = JSON.parse(readFileSync(file, 'utf8'));
    for (const operations of Object.values(document)) {
      for (const operation of Object.values(operations)) {
        if (operation && typeof operation === 'object' && typeof operation.operationId === 'string') ids.add(operation.operationId);
      }
    }
  }
  return [...ids].sort();
}

function mutatingOperationIds() {
  const ids = new Set();
  for (const file of sourceFiles(path.join(root, 'api/openapi/source/paths'), (name) => name.endsWith('.json'))) {
    const document = JSON.parse(readFileSync(file, 'utf8'));
    for (const operations of Object.values(document)) {
      for (const [method, operation] of Object.entries(operations)) {
        if (['post', 'put', 'patch', 'delete'].includes(method) && operation && typeof operation === 'object' && typeof operation.operationId === 'string') ids.add(operation.operationId);
      }
    }
  }
  return ids;
}

function aviaCoreEventTypes() {
  const eventCatalog = path.join(aviaCoreRoot, 'contracts/aviasurveil-production/v1/event-catalog.json');
  assert.ok(existsSync(eventCatalog), `AviaCore event catalog must exist at ${eventCatalog}`);
  return JSON.parse(readFileSync(eventCatalog, 'utf8')).events.map((event) => event.event_type).sort();
}

function aviaCoreV3EventTypes() {
  const eventCatalog = path.join(root, 'integrations/aviacore/contracts/contracts/aviasurveil-production/v3/event-catalog.json');
  assert.ok(existsSync(eventCatalog), `locked v3 event catalog must exist at ${eventCatalog}`);
  return JSON.parse(readFileSync(eventCatalog, 'utf8')).events.map((event) => event.event_type).sort();
}

function literalDomainFacts(field) {
  const values = new Set();
  for (const directory of ['apps/api/internal/application', 'apps/api/internal/checklistgovernance']) {
    for (const file of sourceFiles(path.join(root, directory), (name) => name.endsWith('.go') && !name.endsWith('_test.go'))) {
      const source = readFileSync(file, 'utf8');
      for (const match of source.matchAll(new RegExp(`${field}\\s*:\\s*"([^\"]+)"`, 'g'))) values.add(match[1]);
    }
  }
  return [...values].sort();
}

function requiredPolicyFields(policy) {
  return [
    'id', 'disposition', 'source_authority', 'contract_family', 'grain', 'entity_references',
    'tenant_binding', 'organization_binding', 'actor_organization_binding', 'visibility_purpose_scope',
    'effective_time', 'known_time', 'producer_revision_sequence', 'null_semantics',
    'correction_supersession', 'pii_class', 'minimization_rationale', 'retention_legal_hold_deletion',
    'expected_volume', 'consumer', 'lineage', 'feature_label_eligibility', 'field_policy',
  ];
}

test('coverage register exists before a producer contract or outbox can be implemented', () => {
  assert.ok(existsSync(registerPath), 'missing aviacore source-to-contract coverage register');
  assert.ok(existsSync(registerGuidePath), 'missing aviacore source-to-contract coverage guide');
  assert.match(readFileSync(registerGuidePath, 'utf8'), /aviacore-data-feed-coverage\.json/);
});

test('coverage register is a closed, fingerprint-bound source inventory', () => {
  const register = JSON.parse(readFileSync(registerPath, 'utf8'));
  assert.equal(register.schema_version, 'aviacore_data_feed_coverage.v1');
  assert.equal(register.claim_boundary.training_allowed, false);
  assert.equal(register.claim_boundary.production_ml_readiness, 'NOT_READY');
  assert.deepEqual(register.allowed_dispositions, [
    'approved_event_field', 'approved_snapshot_contract', 'approved_reference_data_contract',
    'approved_data_product_derivation', 'contract_extension_candidate', 'operational_dq_only',
    'sensitive_restricted', 'forbidden',
  ]);

  const expectedMigrations = sourceFiles(path.join(root, 'apps/api/migrations'), (name) => name.endsWith('.up.sql'))
    .map((file) => sourceFingerprint(path.relative(root, file))).sort((a, b) => a.path.localeCompare(b.path));
  assert.deepEqual(register.source_fingerprints.migrations, expectedMigrations,
    'adding, removing, or editing a migration requires an explicit coverage-register decision');

  const expectedOpenApi = [
    'api/openapi/source/openapi.json',
    ...sourceFiles(path.join(root, 'api/openapi/source/paths'), (name) => name.endsWith('.json')).map((file) => path.relative(root, file)),
    ...sourceFiles(path.join(root, 'api/openapi/source/schemas'), (name) => name.endsWith('.json')).map((file) => path.relative(root, file)),
  ].map(sourceFingerprint).sort((a, b) => a.path.localeCompare(b.path));
  assert.deepEqual(register.source_fingerprints.openapi, expectedOpenApi,
    'OpenAPI mutation/schema drift requires an explicit coverage-register decision');

  const expectedCommands = [
    ...sourceFiles(path.join(root, 'apps/api/internal/application'), (name) => name.endsWith('.go') && !name.endsWith('_test.go')),
    ...sourceFiles(path.join(root, 'apps/api/internal/checklistgovernance'), (name) => name.endsWith('.go') && !name.endsWith('_test.go')),
    path.join(root, 'apps/api/internal/preproddata/profiles/profiles.go'),
  ].map((file) => sourceFingerprint(path.relative(root, file))).sort((a, b) => a.path.localeCompare(b.path));
  assert.deepEqual(register.source_fingerprints.authoritative_commands, expectedCommands,
    'command, transition, or profile-manifest drift requires an explicit coverage-register decision');
});

test('every persisted relation has exactly one detailed disposition policy', () => {
  const register = JSON.parse(readFileSync(registerPath, 'utf8'));
  const policies = register.relation_dispositions;
  const relations = createdRelations();
  const coveredRelations = policies.flatMap((policy) => policy.relations).sort();
  assert.deepEqual(coveredRelations, relations,
    'every created relation must have one and only one source-to-contract disposition');
  assert.equal(new Set(coveredRelations).size, relations.length, 'duplicate relation dispositions are forbidden');
  for (const policy of policies) {
    const resolved = { ...register.policy_defaults, ...policy };
    assert.ok(Array.isArray(policy.relations) && policy.relations.length > 0, 'a disposition policy must name its relations');
    assert.ok(register.allowed_dispositions.includes(resolved.disposition), `${policy.id} has an unknown disposition`);
    for (const field of requiredPolicyFields(resolved)) {
      assert.notEqual(resolved[field], undefined, `${policy.id} lacks required ${field}`);
    }
    assert.ok(Array.isArray(resolved.entity_references));
    assert.ok(Object.keys(resolved.field_policy).length > 0, `${policy.id} requires an explicit field policy`);
    assert.match(resolved.minimization_rationale, /\S/);
    assert.match(resolved.retention_legal_hold_deletion, /\S/);
  }
});

test('every CREATE and ALTER surviving column has one explicit, immutable disposition', () => {
  const register = JSON.parse(readFileSync(registerPath, 'utf8'));
  const expected = currentColumns();
  const actual = register.column_dispositions.map((item) => `${item.relation}.${item.column}`).sort();
  assert.deepEqual(actual, expected, 'every final migration column must have one explicit disposition');
  assert.equal(new Set(actual).size, expected.length, 'duplicate column dispositions are forbidden');
  for (const item of register.column_dispositions) {
    assert.ok(register.allowed_dispositions.includes(item.disposition), `${item.relation}.${item.column} has unknown disposition`);
    assert.ok(item.field_class === 'closed_event_candidate' || item.field_class === 'extension_candidate' || item.field_class === 'operational' || item.field_class === 'sensitive_restricted' || item.field_class === 'forbidden_inline');
    if (item.disposition === 'approved_event_field') assert.ok(register.v1_event_relation_allowlist.includes(item.relation), `${item.relation}.${item.column} cannot claim v1 event coverage`);
    if (['responsible_person', 'expected_evidence', 'finding_basis'].includes(item.column)) assert.notEqual(item.field_class, 'closed_event_candidate', `${item.relation}.${item.column} is unbounded or identity-bearing text and cannot be a closed event field`);
  }
});

test('the approved v1/v3 event scopes, their gaps, and the authoritative mutation register are explicit', () => {
	const register = JSON.parse(readFileSync(registerPath, 'utf8'));
	assert.deepEqual(register.approved_v1_event_types.slice().sort(), aviaCoreEventTypes());
	assert.deepEqual(register.approved_v3_event_types.slice().sort(), aviaCoreV3EventTypes());
  assert.deepEqual(register.openapi_mutations.slice().sort(), operationIds());
  assert.deepEqual(register.authoritative_domain_facts.audit_actions.slice().sort(), literalDomainFacts('Action'));
  assert.deepEqual(register.authoritative_domain_facts.outbox_topics.slice().sort(), literalDomainFacts('OutboxTopic'));
  const mappings = register.authoritative_fact_mappings;
  const sourceRelations = new Set(createdRelations());
  for (const key of ['openapi_operations', 'audit_actions', 'outbox_topics']) {
    assert.ok(Array.isArray(mappings[key]), `missing ${key} mappings`);
    const actual = mappings[key].map((item) => item.id).sort();
    const expected = key === 'openapi_operations' ? operationIds() : register.authoritative_domain_facts[key].slice().sort();
    assert.deepEqual(actual, expected, `${key} require one exact source-to-contract mapping`);
    assert.equal(new Set(actual).size, actual.length, `${key} mappings cannot duplicate an authority fact`);
    for (const item of mappings[key]) {
      assert.ok(sourceRelations.has(item.target_relation), `${item.id} maps to an unknown relation`);
      assert.ok(register.allowed_dispositions.includes(item.disposition), `${item.id} has unknown disposition`);
      assert.match(item.contract_target, /\S/);
		if (item.disposition === 'approved_event_field') {

			if (key === 'openapi_operations') assert.ok(mutatingOperationIds().has(item.id), `${item.id} is a read-only operation and cannot claim an emitted event`);
			const v3EventTypes = item.v3_event_types ?? [];
			assert.ok(Array.isArray(v3EventTypes), `${item.id} v3 event types must be an array`);
			assert.ok((item.v1_event_type !== null) !== (v3EventTypes.length > 0), `${item.id} must bind exactly one event-contract family`);
			if (item.v1_event_type !== null) {
				assert.ok(register.v1_event_relation_allowlist.includes(item.target_relation), `${item.id} maps outside current v1 event scope`);
				assert.ok(register.approved_v1_event_types.includes(item.v1_event_type), `${item.id} requires a compatible current v1 event type`);
			} else {
				assert.deepEqual(v3EventTypes.slice().sort(), [...new Set(v3EventTypes)].sort(), `${item.id} v3 event types cannot duplicate`);
				assert.ok(v3EventTypes.every((eventType) => register.approved_v3_event_types.includes(eventType)), `${item.id} requires compatible locked v3 event types`);
			}
		} else {
			assert.equal(item.v1_event_type, null, `${item.id} cannot borrow a v1 event type for an extension or restricted fact`);
			assert.deepEqual(item.v3_event_types ?? [], [], `${item.id} cannot borrow a v3 event type for an extension or restricted fact`);
		}
    }
  }
  assert.ok(register.authoritative_mutation_families.length > 0);
  assert.ok(register.event_scope_gaps.some((gap) => gap.family === 'identity_membership'));
  assert.ok(register.event_scope_gaps.some((gap) => gap.family === 'planning_approvals'));
  assert.ok(register.event_scope_gaps.some((gap) => gap.family === 'documents_communications_notifications'));
  assert.ok(register.event_scope_gaps.every((gap) => gap.disposition === 'contract_extension_candidate' || gap.disposition === 'sensitive_restricted' || gap.disposition === 'operational_dq_only'));
});

test('field and profile reconciliation proofs fail closed on forbidden inline data or unowned derivations', () => {
  const register = JSON.parse(readFileSync(registerPath, 'utf8'));
  const forbidden = register.privacy_policy.forbidden_inline_field_patterns;
  assert.deepEqual(forbidden, ['free_text', 'evidence_bytes', 'filename', 'internal_caa_note', 'investigation_note', 'person_name', 'contact_value', 'credential']);
  assert.equal(register.data_product_rule.transport_substitute_forbidden, true);
  assert.equal(register.data_product_rule.requires_deterministic_field_lineage, true);
  assert.equal(register.completeness_proofs.static_persisted_source_inventory, 'fingerprint-bound');
  assert.equal(register.completeness_proofs.command_transition_to_contract_coverage, 'authoritative-mutation-register');
  assert.equal(register.completeness_proofs.profile_manifest_to_event_ack_reconciliation, 'profile-bound-no-unexecuted-branch-claim');
  assert.deepEqual(register.tenant_isolation_test_shapes, [
    'two_platform_tenants_with_colliding_local_business_ids',
    'one_caa_tenant_with_multiple_inspected_auditee_organizations',
  ]);
});
