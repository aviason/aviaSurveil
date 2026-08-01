import assert from "node:assert/strict";
import { existsSync, readFileSync } from "node:fs";
import test from "node:test";

const demoScriptPath = "scripts/test-local-demo-profile.sh";
const fullScriptPath = "scripts/test-local-full-profile.sh";
const imageEvidenceCheckPath = "scripts/check-local-image-evidence.sh";
const playwrightConfigPath = "apps/web/playwright.config.ts";
const fullSpecPath = "apps/web/tests/e2e/local-full-platform.spec.ts";
const identitySetupPath = "apps/api/tests/identitysetup/prepare_test.go";
const behaviorLedgerPath = "tests/parity/behavior-ledger.json";

function readRequired(path) {
  assert.equal(existsSync(path), true, `${path} must exist`);
  return readFileSync(path, "utf8");
}

test("clean profile scripts use unique task-owned projects, state, and cleanup traps", () => {
  for (const path of [demoScriptPath, fullScriptPath]) {
    const source = readRequired(path);
    assert.match(source, /mktemp -d/);
    assert.match(source, /AVIA_LOCAL_PROJECT=.*(?:date|RANDOM|\$\$)/);
    assert.match(source, /AVIASURVEIL_LOCAL_STATE_DIR/);
    assert.match(source, /trap cleanup EXIT/);
    assert.match(source, /local-stack\.sh"\s+down/);
    assert.match(source, /com\.docker\.compose\.project/);
    assert.match(source, /task-owned residue/i);
  }
});

test("full profile cleanup force-removes only exact task-owned labeled residue", () => {
  const source = readRequired(fullScriptPath);
  assert.match(source, /force_remove_task_owned_residue/);
  assert.match(source, /docker ps --all --quiet[\s\S]*com\.docker\.compose\.project=\$\{AVIA_LOCAL_PROJECT\}/);
  assert.match(source, /docker volume ls --quiet[\s\S]*com\.docker\.compose\.project=\$\{AVIA_LOCAL_PROJECT\}/);
  assert.match(source, /docker network ls --quiet[\s\S]*com\.docker\.compose\.project=\$\{AVIA_LOCAL_PROJECT\}/);
  assert.match(source, /docker rm --force/);
  assert.match(source, /docker volume rm/);
  assert.match(source, /docker network rm/);
  assert.doesNotMatch(source, /docker (?:system|container|volume|network) prune/);
});

test("demo profile cleanup handles partial startup and exact task-owned networks", () => {
  const source = readRequired(demoScriptPath);
  assert.match(source, /STACK_STARTED="true"[\s\S]*local-stack\.sh"\s+up\s+demo/);
  assert.match(source, /force_remove_task_owned_residue/);
  assert.match(source, /docker ps --all --quiet[\s\S]*com\.docker\.compose\.project=\$\{AVIA_LOCAL_PROJECT\}/);
  assert.match(source, /docker volume ls --quiet[\s\S]*com\.docker\.compose\.project=\$\{AVIA_LOCAL_PROJECT\}/);
  assert.match(source, /docker network ls --quiet[\s\S]*com\.docker\.compose\.project=\$\{AVIA_LOCAL_PROJECT\}/);
  assert.match(source, /docker rm --force/);
  assert.match(source, /docker volume rm/);
  assert.match(source, /docker network rm/);
  assert.doesNotMatch(source, /docker (?:system|container|volume|network) prune/);
});

test("clean profile cleanup catches unlabeled resources only by exact validated project prefix", () => {
  for (const path of [demoScriptPath, fullScriptPath]) {
    const source = readRequired(path);
    assert.equal(
      source.includes('--filter "name=^${AVIA_LOCAL_PROJECT}_"'),
      true,
      `${path} must query the exact Compose project-name prefix`,
    );
    assert.equal(
      source.includes('[[ "${resource_id}" == "${AVIA_LOCAL_PROJECT}_"* ]]'),
      true,
      `${path} must validate every name-fallback resource before removal`,
    );
    assert.match(source, /task-owned residue: unlabeled volumes remain/);
    assert.match(source, /task-owned residue: unlabeled networks remain/);
  }
});

test("clean profile scripts run an explicit Playwright project and reject skipped tests", () => {
  const demo = readRequired(demoScriptPath);
  const full = readRequired(fullScriptPath);
  assert.match(demo, /--project=local-demo/);
  assert.match(full, /--project=local-full/);
  for (const source of [demo, full]) {
    assert.match(source, /--forbid-only/);
    assert.match(source, /skipped/);
    assert.match(source, /summary\.json/);
    assert.match(source, /application\/json/);
  }
});

test("clean profiles run only the exact SBOM-scanned image digests", () => {
  const checker = readRequired(imageEvidenceCheckPath);
  assert.match(checker, /image-evidence\.json/);
  assert.match(checker, /validate-image-evidence/);
  assert.match(checker, /docker image inspect/);
  assert.match(checker, /image digest drift detected/i);

  for (const [path, profile] of [
    [demoScriptPath, "demo"],
    [fullScriptPath, "full"],
  ]) {
    const source = readRequired(path);
    assert.match(
      source,
      new RegExp(`check-local-image-evidence\\.sh"\\s+${profile}`),
    );
    assert.doesNotMatch(source, /compose build/);
  }
});

test("full profile script cannot enable test authority or publish internal services", () => {
  const source = readRequired(fullScriptPath);
  for (const forbidden of [
    "AVIA_ENABLE_CANONICAL_SEED",
    "AVIA_ENABLE_CANONICAL_TEST_PROFILE",
    "AVIA_CANONICAL_TEST_TOKEN",
    "deterministic-test",
    "fixture-init",
    "compose.test.yaml",
    "__test/reset",
  ]) {
    assert.equal(source.includes(forbidden), false, `${forbidden} is forbidden in full mode`);
  }
  assert.doesNotMatch(source, /(?:5432|9000|9001|1025|3000):(?:5432|9000|9001|1025|3000)/);
  assert.match(source, /local-stack\.sh"\s+up\s+full/);
});

test("focused HTTP profile preserves an explicitly supplied canonical test token", () => {
  const source = readRequired("scripts/test-http-profile.sh");
  assert.equal(
    source.includes('AVIA_CANONICAL_TEST_TOKEN="${AVIA_CANONICAL_TEST_TOKEN:-$(openssl rand -hex 32)}"'),
    true,
  );
});

test("full profile binds one exact Admin membership before normal OIDC login", () => {
  const script = readRequired(fullScriptPath);
  const setup = readRequired(identitySetupPath);
  const setupBuild = script.indexOf(
    "TestTask4PrepareOIDCHarnessApplicationAdministrator",
  );
  const browserProof = script.indexOf("npx playwright test");

  assert.match(script, /go[\s\S]*test[\s\S]*-c[\s\S]*-tags canonicaltest/);
  assert.match(
    script,
    /TestTask4PrepareOIDCHarnessApplicationAdministrator/u,
  );
  assert.match(
    script,
    /compose run[\s\S]*--rm[\s\S]*--no-deps[\s\S]*--volume[\s\S]*identitysetup\.test:ro/u,
  );
  assert.doesNotMatch(script, /compose cp/u);
  assert.ok(
    setupBuild >= 0 && browserProof > setupBuild,
    "authoritative Admin membership setup must precede browser login",
  );
  assert.match(setup, /RequestLifecycle/u);
  assert.match(setup, /ProcessNext/u);
  assert.doesNotMatch(
    script,
    /for role in inspector leadInspector manager gm finance executiveDirector admin/u,
  );
  assert.doesNotMatch(
    script,
    /--request POST[\s\S]{0,320}\/admin\/realms\/aviasurveil360\/users(?:["\s\\]|$)/u,
  );
  assert.doesNotMatch(
    script,
    /--request POST[\s\S]{0,320}role-mappings\/realm/u,
  );
});

test("full profile uses eight distinct exact-role sessions", () => {
  const source = readRequired(fullSpecPath);
  for (const role of [
    "inspector",
    "leadInspector",
    "manager",
    "finance",
    "gm",
    "executiveDirector",
    "auditee",
    "admin",
  ]) {
    assert.match(
      source,
      new RegExp(`["']${role}["']`),
      `full profile must retain a distinct ${role} authority`,
    );
  }
  assert.match(source, /rolePages/u);
  assert.match(source, /route\.requiredRole/u);
  assert.doesNotMatch(
    source,
    /route\.requiredRole === "auditee" \? auditeePage : adminPage/u,
  );
});

test("Playwright registers isolated local demo and full projects without a Vite server", () => {
  const source = readRequired(playwrightConfigPath);
  assert.match(source, /e2eProfile === "local-demo"/);
  assert.match(source, /e2eProfile === "local-full"/);
  assert.match(source, /name: "local-demo"/);
  assert.match(source, /name: "local-full"/);
  assert.match(source, /local-full-platform\.spec\.ts/);
  assert.match(source, /profile !== "local-demo"/);
  assert.match(source, /profile !== "local-full"/);
});

test("full profile browser proof uses only normal OIDC and application boundaries", () => {
  const source = readRequired(fullSpecPath);
  for (const forbidden of [
    "../../src/mock/",
    "seed-data",
    "createCanonicalTestFetch",
    "AVIA_CANONICAL_TEST_TOKEN",
    "deterministic-test",
    "fixture-init",
    "__test/reset",
  ]) {
    assert.equal(source.includes(forbidden), false, `${forbidden} is forbidden in the full proof`);
  }
  assert.match(source, /CONFIGURE_TOTP|totp/i);
  assert.match(source, /toHaveLength\(86\)/);
  assert.match(source, /toHaveLength\(10\)/);
  assert.match(source, /\/__test\/.*404|404.*\/__test\//s);
  assert.match(source, /ClamAV|scan-clean/i);
  assert.match(source, /Mailpit|SMTP/i);
  assert.match(source, /Gotenberg|PDF/i);
  assert.match(source, /MinIO|object/i);
});

test("full profile proves normal SMTP delivery through the private Mailpit API", () => {
  const script = readRequired(fullScriptPath);
  const spec = readRequired(fullSpecPath);
  assert.match(spec, /\/api\/v1\/communications/);
  assert.match(spec, /emailDeliveryStatus[\s\S]*DELIVERED/);
  assert.match(script, /http:\/\/mailpit:8025\/api\/v1\/messages/);
  assert.match(script, /Mailpit[\s\S]*Plan 3 full-profile SMTP delivery/);
});

test("behavior ledger records the clean full profile proof", () => {
  const ledger = JSON.parse(readRequired(behaviorLedgerPath));
  const entry = ledger.entries.find(({ id }) => id === "local-production-like-full-profile");
  assert.ok(entry, "local-production-like-full-profile ledger entry is required");
  assert.equal(entry.expectedStatus, "CLEAN_FULL_PROFILE_VERIFIED_LOCALLY");
  assert.ok(entry.reactTest.includes("apps/web/tests/e2e/local-full-platform.spec.ts"));
  assert.ok(entry.reactTest.includes("scripts/test-local-full-profile.sh"));
});
