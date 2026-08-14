const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');

const root = path.resolve(__dirname, '..');

function read(relativePath) {
  return fs.readFileSync(path.join(root, relativePath), 'utf8');
}

function assertFile(relativePath) {
  assert.equal(
    fs.existsSync(path.join(root, relativePath)),
    true,
    `Missing required harness file: ${relativePath}`
  );
}

const requiredFiles = [
  'AGENTS.md',
  'ARCHITECTURE.md',
  'docs/PLANS.md',
  'docs/index.md',
  'docs/agent-harness/index.md',
  'docs/agent-harness/config.json',
  'docs/agent-harness/output-contract.md',
  'docs/agent-harness/registry.md',
  'docs/agent-harness/environment-contract.md',
  'docs/agent-harness/operating-loop.md',
  'docs/agent-harness/verification-matrix.md',
  'docs/agent-harness/coverage.md',
  'docs/agent-harness/certification.md',
  'docs/agent-harness/evidence/.gitkeep',
  'docs/agent-harness/entropy-cleanup-checklist.md',
  'docs/demo-evidence/stakeholder/PLANS2_4_STAKEHOLDER_DISPOSITION_2026-07-28.md',
  'docs/product-specs/index.md',
  'docs/demo-evidence/BUILD_SUMMARY.md',
  'tests/harness-docs-smoke.test.js'
];

requiredFiles.forEach(assertFile);

const docsIndex = read('docs/index.md');
[
  'agent-harness/index.md',
  'exec-plans/index.md',
  'exec-plans/tech-debt-tracker.md',
  'product-specs/index.md',
  'demo-handoff/',
  'demo-evidence/BUILD_SUMMARY.md'
].forEach((target) => {
  assert.match(
    docsIndex,
    new RegExp(target.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')),
    `docs/index.md must link ${target}`
  );
});

[
  'docs/00_RESEARCH_AND_POSITIONING',
  'docs/01_PRODUCT_PLAN',
  'docs/02_UX_PLAN',
  'docs/03_WORKFLOWS',
  'docs/04_MODULES',
  'docs/05_SCREEN_SPECS',
  'docs/06_DATA_AND_RULES',
  'docs/07_ANALYTICS',
  'docs/08_DEMO_AND_BUILD_HANDOFF',
  'docs/09_SCENARIOS',
  'docs/10_REFERENCES',
  'docs/DEMO_BUILD_SUMMARY.md',
  'docs/DEMO_BUILD_SUMMARY.turkce.md'
].forEach((relativePath) => {
  assert.equal(
    fs.existsSync(path.join(root, relativePath)),
    false,
    `Legacy docs path should not exist: ${relativePath}`
  );
});

const planIndex = read('docs/exec-plans/index.md');
[
  'active/2026-06-29-agent-harness-readiness-completion-plan.md',
  'active/2026-06-29-aviasurveil-harness-engineering-adaptation-plan.md',
  'tech-debt-tracker.md'
].forEach((target) => {
  assert.match(
    planIndex,
    new RegExp(target.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')),
    `docs/exec-plans/index.md must link ${target}`
  );
});

const harnessIndex = read('docs/agent-harness/index.md');
[
  '../../AGENTS.md',
  '../../ARCHITECTURE.md',
  '../PLANS.md',
  '../product-specs/index.md',
  'output-contract.md',
  'config.json',
  'registry.md',
  'environment-contract.md',
  'operating-loop.md',
  'verification-matrix.md',
  'coverage.md',
  'certification.md',
  'entropy-cleanup-checklist.md',
  '../exec-plans/index.md',
  '../exec-plans/tech-debt-tracker.md',
  '../demo-evidence/BUILD_SUMMARY.md'
].forEach((target) => {
  assert.match(
    harnessIndex,
    new RegExp(target.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')),
    `docs/agent-harness/index.md must link ${target}`
  );
});

const agents = read('AGENTS.md');
[
  '](ARCHITECTURE.md)',
  '](docs/PLANS.md)',
  '](docs/exec-plans/index.md)',
  '](docs/agent-harness/registry.md)',
  '](docs/agent-harness/environment-contract.md)',
  '](docs/agent-harness/operating-loop.md)',
  '](docs/agent-harness/coverage.md)',
  '](docs/agent-harness/certification.md)',
  '](docs/agent-harness/verification-matrix.md)'
].forEach((route) => {
  assert.match(
    agents,
    new RegExp(route.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')),
    `AGENTS.md must expose a Markdown route containing ${route}`
  );
});

function collectFiles(directory) {
  return fs.readdirSync(directory, { withFileTypes: true }).flatMap((entry) => {
    const fullPath = path.join(directory, entry.name);
    if (entry.name === '.git' || entry.name === 'node_modules' || entry.name === 'dist') {
      return [];
    }
    return entry.isDirectory() ? collectFiles(fullPath) : [fullPath];
  });
}

const turkishCompanions = collectFiles(root)
  .filter((file) => file.endsWith('.turkce.md'))
  .map((file) => path.relative(root, file));

assert.deepEqual(
  turkishCompanions,
  [],
  `English-only documentation policy violated by: ${turkishCompanions.join(', ')}`
);

const outputContract = read('docs/agent-harness/output-contract.md');
[
  'verified locally',
  'blocked',
  'not run',
  'production-readiness not claimed'
].forEach((label) => {
  assert.match(
    outputContract,
    new RegExp(label.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')),
    `output-contract.md must define status label: ${label}`
  );
});

const harnessDocs = [
  'docs/agent-harness/index.md',
  'docs/agent-harness/environment-contract.md',
  'docs/agent-harness/operating-loop.md',
  'docs/agent-harness/output-contract.md',
  'docs/agent-harness/registry.md',
  'docs/agent-harness/verification-matrix.md',
  'docs/agent-harness/coverage.md',
  'docs/agent-harness/certification.md',
  'docs/agent-harness/entropy-cleanup-checklist.md'
];

const harnessConfig = JSON.parse(read('docs/agent-harness/config.json'));
assert.deepEqual(harnessConfig, {
  schema_version: 1,
  authorities: {
    instructions: 'AGENTS.md',
    architecture: 'ARCHITECTURE.md',
    planning: 'docs/PLANS.md',
    registry: 'docs/agent-harness/registry.md',
    environment: 'docs/agent-harness/environment-contract.md',
    verification: 'docs/agent-harness/verification-matrix.md',
    coverage: 'docs/agent-harness/coverage.md',
    certification: 'docs/agent-harness/certification.md'
  }
});

const coverage = read('docs/agent-harness/coverage.md');
const canonicalCoverageRows = [
  'Humans set intent; agents execute within authority',
  'Break large goals into reusable design, code, review, test, and verification steps',
  'Agents can self-review and respond to feedback',
  'Application behavior is directly readable',
  'Logs, metrics, and traces are queryable when relevant',
  'Repository knowledge is the durable record',
  'Repository tools and authorized work context are directly invocable',
  'Dependencies and abstractions remain agent-legible',
  '`AGENTS.md` is a concise map, not an encyclopedia',
  'Plans are versioned living artifacts',
  'Architecture and critical taste boundaries are mechanical',
  'Local autonomy exists inside enforced central boundaries',
  'Verification proves working behavior, not only code changes',
  'Failures and review judgment feed back into the harness',
  'Entropy and technical debt are continuously controlled',
  'Autonomy increases only after test, review, recovery, and escalation loops exist',
  'Merge throughput policy matches project risk',
  'Release, deployment, and production actions require repository-local authority',
  'Repository-specific OpenAI examples are treated as options, not universal mandates',
  'Zero human-authored code as an operating constraint',
  'Reported repository size, pull-request throughput, elapsed-time speedup, and long agent-run duration as targets',
  'Local and cloud agent review loops continue until reviewers are satisfied while human review is optional',
  'Per-worktree application isolation',
  'Per-worktree observability stack',
  'Chrome DevTools Protocol for UI control',
  'Victoria Logs, Metrics, and Traces with LogQL/PromQL/TraceQL',
  "OpenAI's fixed layered domain architecture",
  'Reimplementing upstream dependency behavior locally',
  'Minimally blocking merge gates and short-lived pull requests',
  'Scheduled Codex documentation gardening and quality-scoring agents open targeted repair pull requests',
  'Automated merge and agent-authored release tooling'
];
assert.equal(canonicalCoverageRows.length, 31, 'coverage test must retain all 31 canonical rows');
canonicalCoverageRows.forEach((capability) => assert.match(coverage, new RegExp(capability.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')), `coverage must retain ${capability}`));

const certification = read('docs/agent-harness/certification.md');
[
  'Data',
  'DEBT-001',
  'fixtures',
  'source commit',
  'caller-owned external HMAC key'
].forEach((boundary) => assert.match(certification, new RegExp(boundary.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')), `certification must state ${boundary} boundary`));

const provenance = read('docs/demo-evidence/stakeholder/PLANS2_4_STAKEHOLDER_DISPOSITION_2026-07-28.md');
[
  '3f392ecbe4eafee644347e3f0fc0067e54096279',
  '391d0b4a4e2caaf5befa7db7bfc03dcf0d718dd0',
  '68f4ce8ca90385c921e1284f6343e2f584cace92'
].forEach((identity) => assert.match(provenance, new RegExp(identity), `provenance must preserve ${identity}`));

const registry = read('docs/agent-harness/registry.md');
[
  'Go/PostgreSQL',
  '../../shared/auth/',
  'legacy frontend-only boundary'
].forEach((phrase) => assert.match(registry, new RegExp(phrase.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')), `registry must route ${phrase}`));

const forbiddenClaims = [
  'production-ready',
  'real authentication is implemented',
  'real upload is implemented',
  'real AI service is implemented'
];

harnessDocs.forEach((relativePath) => {
  const content = read(relativePath);
  forbiddenClaims.forEach((claim) => {
    assert.doesNotMatch(
      content,
      new RegExp(claim.replace(/[.*+?^${}()|[\]\\]/g, '\\$&'), 'i'),
      `${relativePath} contains forbidden readiness claim: ${claim}`
    );
  });
});

console.log('harness-docs-smoke: ok');
