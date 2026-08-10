import assert from "node:assert/strict";
import { existsSync, readFileSync } from "node:fs";
import path from "node:path";
import { test } from "node:test";
import { fileURLToPath } from "node:url";

const repositoryRoot = path.resolve(
  path.dirname(fileURLToPath(import.meta.url)),
  "..",
);

function readRequired(relativePath) {
  const filePath = path.join(repositoryRoot, relativePath);
  assert.ok(existsSync(filePath), `${relativePath} must exist`);
  return readFileSync(filePath, "utf8");
}

function readJSON(relativePath) {
  return JSON.parse(readRequired(relativePath));
}

test("restore accepts only a named recovery point and unique isolated prefix", () => {
  const source = readRequired("scripts/restore-isolated-stack.sh");

  assert.match(source, /validate_recovery_point_id/);
  assert.match(source, /validate_isolated_prefix/);
  assert.match(source, /recoveryPointId/);
  assert.match(source, /isolatedPrefix/);
  assert.match(source, /AVIA_RESTORE_SOURCE_PROJECT/);
  assert.match(source, /AVIA_RESTORE_PROJECT/);
  assert.doesNotMatch(source, /docker (?:system|volume|network) prune/);
  assert.doesNotMatch(source, /rm\s+-rf\s+(?:\/|\$HOME|\.\.)/);
});

test("focused local backup/restore uses the explicit canonical test artifact", () => {
  const source = readRequired("scripts/test-local-recovery.sh");

  assert.match(source, /go -C .* build -tags canonicaltest -o .*\/api.*\.\/cmd\/api/u);
  assert.match(source, /AVIA_ENABLE_CANONICAL_TEST_PROFILE="true"/u);
  assert.match(source, /AVIA_CANONICAL_TEST_TOKEN=/u);
});

test("restore validates a complete checksum-bound recovery point before mutation", () => {
  const source = readRequired("scripts/restore-isolated-stack.sh");

  for (const requiredCheck of [
    /status.*complete/,
    /applicationDatabase/,
    /identityDatabase/,
    /identityFingerprint/,
    /applicationObjects/,
    /configurationReferences/,
    /catalog checksum mismatch/,
    /object manifest checksum mismatch/,
  ]) {
    assert.match(source, requiredCheck);
  }
  assert.match(source, /"--set=\$application_backup_label"/);
  assert.match(source, /"--set=\$identity_backup_label"/);
  assert.match(source, /--restore-manifest/);
  assert.match(source, /partial restore refused/);
});

test("restore uses project-scoped targets and never mounts active database or object volumes", () => {
  const source = readRequired("scripts/restore-isolated-stack.sh");

  assert.match(source, /\$\{restore_project\}_app-database/);
  assert.match(source, /\$\{restore_project\}_keycloak-database/);
  assert.match(source, /\$\{restore_project\}_object-store/);
  assert.doesNotMatch(source, /\$\{source_project\}_app-database/);
  assert.doesNotMatch(source, /\$\{source_project\}_keycloak-database/);
  assert.doesNotMatch(source, /\$\{source_project\}_object-store/);
  assert.match(source, /com\.docker\.compose\.project=\$\{restore_project\}/);
});

test("restore proves application identity object and browser state before success", () => {
  const source = readRequired("scripts/restore-isolated-stack.sh");

  assert.match(source, /application fingerprint mismatch/);
  assert.match(source, /identity fingerprint mismatch/);
  assert.match(source, /object fingerprint mismatch/);
  assert.match(source, /restored-platform/);
  assert.match(source, /browserSmoke/);
  assert.match(source, /directLoads.*86/);
  assert.match(source, /totpLogin.*true/);
  assert.match(source, /cleanupStatus/);
});

test("PostgreSQL recovery exports encrypted repository credentials for archive-get", () => {
  const source = readRequired("scripts/restore-isolated-stack.sh");

  assert.match(source, /PGBACKREST_REPO1_S3_KEY=/);
  assert.match(source, /PGBACKREST_REPO1_S3_KEY_SECRET=/);
  assert.match(source, /PGBACKREST_REPO1_CIPHER_PASS=/);
  assert.match(source, /exec postgres -c archive_mode=off/);
});

test("restore carries the generated Keycloak realm config into the isolated state", () => {
  const source = readRequired("scripts/restore-isolated-stack.sh");

  assert.match(source, /source_state\}\/keycloak\/realm\.json/);
  assert.match(source, /restore_state\}\/keycloak/);
  assert.match(source, /cp .*realm\.json.*restore_state\}\/keycloak\/realm\.json/);
  assert.match(source, /chmod 0600 .*restore_state\}\/keycloak\/realm\.json/);
});

test("drill catalog covers the five bounded local failure scenarios", () => {
  const scenarios = readJSON("deploy/recovery/drill-scenarios.json");

  assert.equal(scenarios.schemaVersion, 1);
  assert.equal(scenarios.artifactStatus, "candidate-only");
  assert.equal(scenarios.productionCertification, false);
  assert.deepEqual(
    scenarios.scenarios.map(({ id }) => id),
    [
      "database-loss",
      "primary-object-loss",
      "latest-backup-corruption-fallback",
      "worker-backlog",
      "lost-application-node",
    ],
  );
  for (const scenario of scenarios.scenarios) {
    assert.equal(scenario.activeStackMutationAllowed, false);
    assert.ok(scenario.expectedResult);
    assert.ok(scenario.recoveryAction);
  }
});

test("drill runs two complete restores and enforces local RPO RTO and cleanup", () => {
  const source = readRequired("scripts/test-rpo-rto-drill.sh");

  assert.match(source, /drill-1/);
  assert.match(source, /drill-2/);
  assert.match(source, /databaseRpoSeconds/);
  assert.match(source, /objectRpoSeconds/);
  assert.match(source, /rtoSeconds/);
  assert.match(source, /database RPO exceeded 900 seconds/);
  assert.match(source, /object RPO exceeded 900 seconds/);
  assert.match(source, /RTO exceeded 3600 seconds/);
  assert.match(source, /zero isolated residue/);
  assert.match(source, /latest-backup-corruption-fallback/);
});

test("drill seeds and proves one real notification delivery backlog item", () => {
  const drill = readRequired("scripts/test-rpo-rto-drill.sh");
  const restore = readRequired("scripts/restore-isolated-stack.sh");

  assert.match(drill, /notification\.email_requested/);
  assert.match(drill, /INSERT INTO notification_delivery_jobs/);
  assert.match(drill, /initial worker backlog was not exactly one pending item/);
  assert.doesNotMatch(drill, /:'subject_id'/);
  assert.match(restore, /plan4-restore-backlog-outbox/);
  assert.match(restore, /job\.status = 'DELIVERED'/);
  assert.match(restore, /outbox\.delivered_at IS NOT NULL/);
  assert.match(restore, /restored worker backlog did not recover/);
});

test("restored browser smoke verifies retained MFA scope and all 86 routes without skips", () => {
  const source = readRequired(
    "apps/web/tests/e2e/restored-platform-smoke.spec.ts",
  );
  const config = readRequired("apps/web/playwright.config.ts");

  assert.match(source, /AVIA_RESTORED_TOTP_SECRET/);
  assert.match(source, /AVIA_RESTORED_EXPECTED_ORGANIZATION_ID/);
  assert.match(source, /AVIA_RESTORED_EXPECTED_ROLES/);
  assert.match(source, /REACT_ROUTE_CONTRACTS/);
  assert.match(source, /toHaveLength\(86\)/);
  assert.match(source, /directLoads:\s*86/);
  assert.match(source, /totpLogin:\s*true/);
  assert.doesNotMatch(source, /test\.skip|\.skip\(/);
  assert.match(config, /restored-platform/);
});

test("restore and disaster recovery runbooks preserve the local candidate boundary", () => {
  for (const relativePath of [
    "docs/operations/runbooks/RESTORE.md",
    "docs/operations/runbooks/DISASTER_RECOVERY.md",
  ]) {
    const source = readRequired(relativePath);
    for (const heading of [
      "## Preconditions",
      "## Safety Boundary",
      "## Procedure",
      "## Expected Evidence",
      "## Cleanup",
      "## Escalation",
    ]) {
      assert.match(source, new RegExp(`^${heading}$`, "m"));
    }
    assert.match(source, /verified locally/);
    assert.match(source, /candidate-only/);
    assert.match(source, /not production-ready/);
    assert.doesNotMatch(source, /docker (?:system|volume|network) prune/);
  }
});
