import assert from 'node:assert/strict';
import { execFileSync } from 'node:child_process';
import { createHash } from 'node:crypto';
import { existsSync, readdirSync, readFileSync, statSync } from 'node:fs';
import { join, relative } from 'node:path';
import test from 'node:test';

const root = new URL('..', import.meta.url).pathname;
const mirror = join(root, 'integrations/aviacore/contracts');
const lockPath = join(root, 'integrations/aviacore/contract-lock.json');
const sourceRoot = process.env.AVIACORE_ROOT ?? '/Users/marlonjd/Developer/monorepos/aviaCore';

function files(rootPath, base = rootPath) {
  return readdirSync(rootPath, { withFileTypes: true }).flatMap((entry) => {
    const path = join(rootPath, entry.name);
    return entry.isDirectory() ? files(path, base) : [relative(base, path)];
  }).sort();
}

function sha256(path) {
  return createHash('sha256').update(readFileSync(path)).digest('hex');
}

test('Task 3B mirrors exactly the approved v3 producer contract inventory', () => {
  assert.equal(existsSync(lockPath), true, 'Task 3B must record a contract lock');
  const lock = JSON.parse(readFileSync(lockPath, 'utf8'));
  assert.equal(lock.contract_version, '3.0.0');
  assert.equal(lock.behavioral_contract_digest, 'd87ec3649ff0f3b5f3871e90496eac2b1177dbec9f26fea72ced825d0beff121');
  assert.equal(lock.authorization_digest, '201cbed1f998b60506293efdae81c060b29d3d6e30696257785b4ec02be92c0e');
  assert.equal(lock.authorization_id, 'OWNER-DIRECTIVE-2026-07-30-AVS-V3-RECEIPT-BINDING-01');
  assert.equal(lock.source_root, sourceRoot);
  assert.match(lock.source_head, /^[0-9a-f]{40}$/);
  assert.equal(lock.update_mode, 'explicit-authorized-task-3b-only');
  assert.ok(Array.isArray(lock.artifacts) && lock.artifacts.length > 100);
  assert.deepEqual(lock.artifacts.map((item) => item.path).sort(), files(mirror));
  for (const artifact of lock.artifacts) {
    assert.equal(sha256(join(mirror, artifact.path)), artifact.sha256, artifact.path);
  }
});

test('Task 3B contract checker is read-only by default and rejects source/local drift', () => {
  const script = join(root, 'scripts/check-aviacore-contracts.sh');
  assert.equal(existsSync(script), true);
  execFileSync(script, [], { cwd: root, env: { ...process.env, AVIACORE_ROOT: sourceRoot }, stdio: 'pipe' });
  assert.throws(
    () => execFileSync(script, ['--update'], { cwd: root, env: { ...process.env, AVIACORE_ROOT: sourceRoot }, stdio: 'pipe' }),
    /explicit Task 3B update authorization/,
  );
});

test('Task 3B retains the complete executable negative branch matrix and generated Go validator', () => {
  const matrix = JSON.parse(readFileSync(join(mirror, 'contracts/aviasurveil-production/v3/negative-branch-matrix.json'), 'utf8'));
  assert.deepEqual(matrix.required_categories, [
    'unknown_event_type', 'missing_required_field', 'extra_field', 'payload_digest_mismatch',
    'tenant_identity_mismatch', 'timestamp_rule', 'privacy_forbidden_field',
    'correction_supersession_reference', 'bootstrap_resume', 'version_compatibility_overlap',
  ]);
  assert.ok(matrix.cases.length >= 11);
  assert.equal(existsSync(join(root, 'apps/api/internal/aviacorecontract/v3/validator.go')), true);
  assert.ok(statSync(join(root, 'apps/api/internal/aviacorecontract/v3/validator_test.go')).isFile());
  execFileSync('node', [join(root, 'scripts/generate-aviacore-contract-types.mjs'), '--check'], { cwd: root, stdio: 'pipe' });
});
