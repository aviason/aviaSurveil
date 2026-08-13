import assert from "node:assert/strict";
import { existsSync, readFileSync } from "node:fs";
import { test } from "node:test";

const read = (file) => readFileSync(file, "utf8");

test("canonical preprod supports an exact task-owned HTTPS qualification instance", () => {
  const start = read("scripts/start-canonical-preprod.sh");
  const status = read("scripts/status-canonical-preprod.sh");
  const stop = read("scripts/stop-canonical-preprod.sh");

  for (const source of [start, status, stop]) {
    assert.match(source, /AVIA_CANONICAL_PREPROD_PROJECT/u);
    assert.match(source, /AVIA_PREPROD_HTTPS_PORT/u);
  }
  assert.match(start, /AVIA_PREPROD_HTTPS_PORT must be a user-space TCP port/u);
  assert.doesNotMatch(start, /currently requires AVIA_PREPROD_HTTPS_PORT=8445/u);
  assert.match(status, /metadata\.project !== project/u);
  assert.match(stop, /metadata\.project !== project/u);
  assert.match(stop, /docker (?:ps|volume ls|network ls)[\s\S]*com\.docker\.compose\.project=\$project_name/u);
});

test("Task 8 fault/restart runner owns the complete disposable lifecycle", () => {
  const scriptPath = "scripts/test-canonical-preprod-fault-restart.sh";
  assert.equal(existsSync(scriptPath), true, `${scriptPath} must exist`);
  const script = read(scriptPath);
  const lifecycle = read("apps/web/tests/e2e/canonical-quick-tunnel-lifecycle.spec.ts");
  const panels = read("apps/web/tests/e2e/canonical-quick-tunnel-panels.spec.ts");
  const playwright = read("apps/web/playwright.config.ts");
  const makefile = read("Makefile");

  assert.match(script, /set -euo pipefail/u);
  assert.match(script, /mktemp -d \/private\/tmp\/aviasurveil360-canonical-task8-fault\.XXXXXX/u);
  assert.match(script, /aviasurveil360-task-canonical-preprod-fault-/u);
  assert.match(script, /AVIA_CANONICAL_PREPROD_STATE_DIR/u);
  assert.match(script, /AVIA_CANONICAL_PREPROD_PROJECT/u);
  assert.match(script, /AVIA_PREPROD_HTTPS_PORT/u);
  assert.match(script, /start-canonical-preprod\.sh/u);
  assert.match(script, /canonical-quick-tunnel-lifecycle\.spec\.ts/u);
  assert.match(script, /canonical-quick-tunnel-panels\.spec\.ts/u);
  assert.match(script, /AVIA_E2E_IGNORE_HTTPS_ERRORS=1/u);

  for (const required of ["preprod-postgres", "preprod-auth", "preprod-minio", "preprod-clamav"]) {
    assert.match(script, new RegExp(required, "u"));
  }
  for (const optional of ["preprod-gotenberg", "preprod-mailpit"]) {
    assert.match(script, new RegExp(optional, "u"));
  }
  assert.match(script, /not_ready/u);
  assert.match(script, /degraded/u);
  assert.match(script, /RestartCount/u);
  assert.match(script, /fingerprint_database/u);
  assert.match(script, /wait_for_database_quiescence/u);
  assert.match(script, /fingerprint changed across cold restart/u);
  assert.match(script, /lease_owner IS NOT NULL/u);
  assert.match(script, /outbox_messages/u);
  assert.doesNotMatch(script, /wait_for_outbox_quiescence/u);
  assert.doesNotMatch(script, /processed_at/u);
  assert.match(script, /aga-demo-workspace/u);
  assert.match(script, /aga-candidate-demo/u);
  assert.match(script, /401\|404/u);
  assert.match(lifecycle, /\/api\/v1\/preprod\/aga-demo-workspace\/classification\/query/u);
  assert.match(lifecycle, /donorWorkspace\.status\)\.toBe\(404\)/u);
  assert.match(playwright, /--ignore-certificate-errors/u);
  assert.match(panels, /hasRegistration/u);
  assert.match(panels, /state: "activated"/u);
  assert.doesNotMatch(panels, /await navigator\.serviceWorker\.ready;/u);
  assert.match(script, /task-owned Compose residue remains/u);
  assert.match(script, /docker compose[\s\S]*down --volumes --remove-orphans/u);
  assert.match(script, /FullPlatformRoleOrganizationAndPrivacyDenials/u);
  assert.match(script, /AdHocPlanningWithholdsAuditeeNoticeAfterMaterialization/u);
  assert.match(script, /LostAcknowledgementReplaysOneCanonicalMutationAndTransitionEnvelope/u);
  assert.match(script, /FullFindingLifecycleAuthority/u);

  assert.match(makefile, /^preprod-test-fault-restart:/mu);
  assert.match(makefile, /test-canonical-preprod-fault-restart\.sh/u);
});
