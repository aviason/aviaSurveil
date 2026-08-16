import assert from "node:assert/strict";
import { execFileSync } from "node:child_process";
import { readFileSync } from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";
import test from "node:test";

import { validateComposePolicy } from "../scripts/lib/local-compose-policy.mjs";

const repositoryRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const composePath = path.join(repositoryRoot, "deploy/local/compose.yaml");
const lock = JSON.parse(readFileSync(path.join(repositoryRoot, "deploy/local/image-lock.json"), "utf8"));
const policy = JSON.parse(readFileSync(path.join(repositoryRoot, "deploy/local/compose-policy.json"), "utf8"));

for (const profile of Object.keys(policy.profileServices)) {
  test(`local ${profile} compose profile has an exact locked topology`, () => {
    const rendered = execFileSync("docker", [
      "compose", "--file", composePath, "--profile", profile, "config", "--format", "json",
    ], { cwd: repositoryRoot, encoding: "utf8" });
    const compose = JSON.parse(rendered);
    assert.deepEqual(Object.keys(compose.services).sort(), [...policy.profileServices[profile]].sort());
    assert.deepEqual(validateComposePolicy({ compose, lock, policy, profile }), []);
  });
}

test("runtime database permission reconciliation has its migration credential", () => {
  const rendered = execFileSync("docker", [
    "compose", "--file", composePath, "--profile", "full", "config", "--format", "json",
  ], { cwd: repositoryRoot, encoding: "utf8" });
  const compose = JSON.parse(rendered);
  const permissions = compose.services["runtime-database-permissions"];
  assert.equal(
    permissions.environment.AVIA_DATABASE_MIGRATION_PASSWORD_FILE,
    "/run/secrets/surveil_migration_database_password",
  );
  assert.ok(
    permissions.secrets.some((secret) =>
      (typeof secret === "string" ? secret : secret.source) === "surveil_migration_database_password"
    ),
  );
});
