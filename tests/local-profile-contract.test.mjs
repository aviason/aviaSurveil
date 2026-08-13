import assert from "node:assert/strict";
import { existsSync, readFileSync } from "node:fs";
import test from "node:test";

const demoScriptPath = "scripts/test-local-demo-profile.sh";
const imageEvidenceCheckPath = "scripts/check-local-image-evidence.sh";
const playwrightConfigPath = "apps/web/playwright.config.ts";
const canonicalFaultScriptPath = "scripts/test-canonical-preprod-fault-restart.sh";
const canonicalLifecycleSpecPath = "apps/web/tests/e2e/canonical-quick-tunnel-lifecycle.spec.ts";
const canonicalPanelsSpecPath = "apps/web/tests/e2e/canonical-quick-tunnel-panels.spec.ts";

function readRequired(path) {
  assert.equal(existsSync(path), true, `${path} must exist`);
  return readFileSync(path, "utf8");
}

test("demo profile cleanup handles partial startup and exact task-owned resources", () => {
  const source = readRequired(demoScriptPath);
  assert.match(source, /mktemp -d/);
  assert.match(source, /AVIA_LOCAL_PROJECT=.*(?:date|RANDOM|\$\$)/);
  assert.match(source, /AVIASURVEIL_LOCAL_STATE_DIR/);
  assert.match(source, /trap cleanup EXIT/);
  assert.match(source, /STACK_STARTED="true"[\s\S]*local-stack\.sh"\s+up\s+demo/);
  assert.match(source, /force_remove_task_owned_residue/);
  assert.match(source, /com\.docker\.compose\.project=\$\{AVIA_LOCAL_PROJECT\}/);
  assert.equal(source.includes('--filter "name=^${AVIA_LOCAL_PROJECT}_"'), true);
  assert.equal(source.includes('[[ "${resource_id}" == "${AVIA_LOCAL_PROJECT}_"* ]]'), true);
  assert.doesNotMatch(source, /docker (?:system|container|volume|network) prune/);
});

test("demo profile runs one explicit Playwright project from exact scanned images", () => {
  const demo = readRequired(demoScriptPath);
  const checker = readRequired(imageEvidenceCheckPath);
  assert.match(demo, /--project=local-demo/);
  assert.match(demo, /--forbid-only/);
  assert.match(demo, /skipped/);
  assert.match(demo, /summary\.json/);
  assert.match(demo, /check-local-image-evidence\.sh"\s+demo/);
  assert.doesNotMatch(demo, /compose build/);
  assert.match(checker, /image-evidence\.json/);
  assert.match(checker, /validate-image-evidence/);
  assert.match(checker, /docker image inspect/);
  assert.match(checker, /image digest drift detected/i);
});

test("focused HTTP profile preserves an explicitly supplied canonical test token", () => {
  const source = readRequired("scripts/test-http-profile.sh");
  assert.equal(
    source.includes('AVIA_CANONICAL_TEST_TOKEN="${AVIA_CANONICAL_TEST_TOKEN:-$(openssl rand -hex 32)}"'),
    true,
  );
});

test("full HTTP profile shards the heavy AGA race package", () => {
  const source = readRequired("scripts/test-http-profile.sh");
  assert.doesNotMatch(source, /test -race -p 1 -count=1 \.\/\.\.\./u);
  assert.match(source, /internal\/agaapplicability/u);
  assert.match(
    source,
    /test-agaapplicability-race-shards\.sh/u,
  );
});

test("governed checklist HTTP mode runs its canonical focused transport contract", () => {
  const source = readRequired("scripts/test-http-profile.sh");
  assert.match(
    source,
    /FOCUSED_E2E}" == "governed-checklist"[\s\S]*test:contract:http --[\s\S]{0,40}--run src\/backend\/governed-checklist-http-parity\.test\.ts[\s\S]*TestTask9SyntheticPublicationAndBlockedRealPilotHaveSeparatePersistedEffects/,
  );
});

test("the obsolete fixed-workspace local-full runner is retired in favor of the canonical profile", () => {
  assert.equal(existsSync("scripts/test-local-full-profile.sh"), false);
  assert.equal(existsSync("apps/web/tests/e2e/local-full-platform.spec.ts"), false);
  const config = readRequired(playwrightConfigPath);
  assert.doesNotMatch(config, /e2eProfile === "local-full"|name: "local-full"|local-full-platform/u);
  assert.match(config, /e2eProfile === "canonical-quick-tunnel"/u);
  assert.match(config, /name: "canonical-quick-tunnel"/u);
  assert.match(config, /canonical-quick-tunnel-panels\.spec\.ts/u);
  assert.match(config, /canonical-quick-tunnel-lifecycle\.spec\.ts/u);
  assert.match(config, /profile !== "canonical-quick-tunnel"/u);
});

test("canonical connected proof covers exact roles, lifecycle, native adapters, and cleanup", () => {
  const fault = readRequired(canonicalFaultScriptPath);
  const lifecycle = readRequired(canonicalLifecycleSpecPath);
  const panels = readRequired(canonicalPanelsSpecPath);
  for (const service of ["preprod-postgres", "preprod-auth", "preprod-minio"]) {
    assert.match(fault, new RegExp(service));
  }
  assert.doesNotMatch(fault, /assert_mailpit_delivery/u);
  assert.match(fault, /assert_no_project_residue/u);
  assert.match(fault, /canonical-quick-tunnel-lifecycle\.spec\.ts/u);
  assert.match(fault, /canonical-quick-tunnel-panels\.spec\.ts/u);
  assert.match(lifecycle, /1_310/u);
  assert.match(lifecycle, /\/api\/v1\/evidence\/uploads/u);
  assert.match(lifecycle, /scanState[\s\S]*CLEAN/u);
  assert.match(lifecycle, /renderStatus[\s\S]*SUCCEEDED/u);
  assert.match(lifecycle, /aga-demo-workspace[\s\S]*404/u);
  assert.doesNotMatch(lifecycle, /\/api\/v1\/audit-workspaces|packageDraftId/u);
  for (const role of [
    "admin",
    "auditee",
    "executiveDirector",
    "finance",
    "gm",
    "inspector",
    "leadInspector",
    "manager",
  ]) {
    assert.match(panels, new RegExp(`role: ["']${role}["']`));
  }
});
