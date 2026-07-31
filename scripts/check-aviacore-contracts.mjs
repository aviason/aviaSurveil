#!/usr/bin/env node
import { execFileSync } from 'node:child_process';
import { createHash } from 'node:crypto';
import { cpSync, existsSync, mkdirSync, readdirSync, readFileSync, rmSync, statSync, writeFileSync } from 'node:fs';
import { dirname, join, relative, resolve } from 'node:path';
import process from 'node:process';

const root = resolve(dirname(new URL(import.meta.url).pathname), '..');
const mirror = join(root, 'integrations/aviacore/contracts');
const lockPath = join(root, 'integrations/aviacore/contract-lock.json');
const sourceRoot = resolve(process.env.AVIACORE_ROOT ?? '/Users/marlonjd/Developer/monorepos/aviaCore');
const selected = [
  'contracts/aviasurveil-production/v3',
  'schemas/events/aviasurveil/v3',
  'api/openapi/aviasurveil-ingestion-v3.openapi.json',
];

function canonical(value) { return JSON.stringify(value, Object.keys(value).sort()); }
function sha256(value) { return createHash('sha256').update(value).digest('hex'); }
function treeFiles(rootPath, base = rootPath) {
  return readdirSync(rootPath, { withFileTypes: true }).flatMap((entry) => {
    const full = join(rootPath, entry.name);
    return entry.isDirectory() ? treeFiles(full, base) : [relative(base, full)];
  }).sort();
}
function sourceFiles() {
  return selected.flatMap((item) => {
    const full = join(sourceRoot, item);
    if (!existsSync(full)) throw new Error(`approved AviaCore source is missing: ${full}`);
    return statSync(full).isDirectory() ? treeFiles(full).map((path) => join(item, path)) : [item];
  }).sort();
}
function artifacts(base, paths) {
  return paths.map((path) => {
    const bytes = readFileSync(join(base, path));
    return { path, sha256: sha256(bytes), bytes: bytes.length };
  });
}
function contractRootDigest(items) {
  return sha256(JSON.stringify(items.map(({ path, sha256: digest, bytes }) => ({ path, sha256: digest, bytes }))));
}
function sourceHead() {
  return execFileSync('git', ['-C', sourceRoot, 'rev-parse', 'HEAD'], { encoding: 'utf8' }).trim();
}
function readJSON(path) { return JSON.parse(readFileSync(path, 'utf8')); }
function update() {
  if (process.env.AVIACORE_CONTRACT_UPDATE !== 'authorized-task-3b') {
    throw new Error('explicit Task 3B update authorization is required');
  }
  rmSync(mirror, { recursive: true, force: true });
  mkdirSync(mirror, { recursive: true });
  for (const item of selected) {
    const source = join(sourceRoot, item);
    const target = join(mirror, item);
    mkdirSync(dirname(target), { recursive: true });
    cpSync(source, target, { recursive: true });
  }
  const items = artifacts(sourceRoot, sourceFiles());
  const identity = readJSON(join(sourceRoot, 'contracts/aviasurveil-production/v3/behavioral-contract-identity.json'));
  const authorization = readJSON(join(sourceRoot, 'contracts/aviasurveil-production/v3/authorization-envelope.json'));
  const contractSet = readJSON(join(sourceRoot, 'contracts/aviasurveil-production/v3/contract-set.json'));
  const lock = {
    schema_version: 'aviasurveil_producer_contract_lock.v1',
    update_mode: 'explicit-authorized-task-3b-only',
    source_root: sourceRoot,
    source_head: sourceHead(),
    source_path: 'contracts/aviasurveil-production/v3',
    contract_version: contractSet.contract_version,
    behavioral_contract_digest: identity.behavioral_contract_digest,
    authorization_digest: authorization.authorization_digest,
    authorization_id: authorization.authorization_id,
    authorization_status: authorization.status,
    authorization_currentness: 'Task 3B mirror only; not Phase 2.3 or producer runtime authorization',
    aggregate_root_digest: contractRootDigest(items),
    artifacts: items,
  };
  mkdirSync(dirname(lockPath), { recursive: true });
  writeFileSync(lockPath, `${JSON.stringify(lock, null, 2)}\n`);
}
function check() {
  if (!existsSync(lockPath) || !existsSync(mirror)) throw new Error('Task 3B mirror or contract lock is missing');
  const lock = readJSON(lockPath);
  if (lock.update_mode !== 'explicit-authorized-task-3b-only') throw new Error('contract lock has an invalid update mode');
  if (lock.source_root !== sourceRoot) throw new Error('contract lock source root does not match AVIACORE_ROOT');
  if (lock.source_head !== sourceHead()) throw new Error('contract lock source commit drifted');
  const expectedPaths = sourceFiles();
  const localPaths = treeFiles(mirror);
  if (JSON.stringify(localPaths) !== JSON.stringify(expectedPaths)) throw new Error('contract mirror has missing or extra artifacts');
  const source = artifacts(sourceRoot, expectedPaths);
  const local = artifacts(mirror, localPaths);
  if (JSON.stringify(source) !== JSON.stringify(local) || JSON.stringify(lock.artifacts) !== JSON.stringify(local)) throw new Error('contract mirror or lock SHA-256 inventory drifted');
  if (lock.aggregate_root_digest !== contractRootDigest(local)) throw new Error('contract lock aggregate root digest drifted');
  const identity = readJSON(join(mirror, 'contracts/aviasurveil-production/v3/behavioral-contract-identity.json'));
  const authorization = readJSON(join(mirror, 'contracts/aviasurveil-production/v3/authorization-envelope.json'));
  if (identity.behavioral_contract_digest !== lock.behavioral_contract_digest || authorization.authorization_digest !== lock.authorization_digest || authorization.status !== 'approved') throw new Error('contract lock authorization or behavioral identity is stale, revoked, or mismatched');

  execFileSync('node', [join(root, 'scripts/generate-aviacore-contract-types.mjs'), '--check'], { stdio: 'pipe' });
}

try {
  const args = process.argv.slice(2);
  if (args.length === 1 && args[0] === '--update') update();
  else if (args.length === 0) check();
  else throw new Error('usage: check-aviacore-contracts.sh [--update]');
  process.stdout.write('aviacore-contract-lock: ok\n');
} catch (error) {
  process.stderr.write(`aviacore-contract-lock: ${error.message}\n`);
  process.exitCode = 1;
}
