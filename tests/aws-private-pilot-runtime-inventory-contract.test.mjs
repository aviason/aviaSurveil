import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import path from "node:path";
import { spawnSync } from "node:child_process";
import test from "node:test";
import { fileURLToPath } from "node:url";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const readInventory = (name) => JSON.parse(readFileSync(path.join(root, "deploy/aws-private-pilot", name), "utf8"));

test("legacy and final target inventories remain separate immutable hash-bound records", () => {
  const legacy = readInventory("runtime-inventory-legacy.json");
  const target = readInventory("runtime-inventory-target.json");
  assert.equal(legacy.immutable, true);
  assert.equal(target.immutable, true);
  assert.notEqual(legacy.inventoryId, target.inventoryId);
  assert.equal(legacy.mode, "legacy");
  assert.equal(target.mode, "target");
  assert.match(legacy.canonicalAggregateSha256, /^sha256:[0-9a-f]{64}$/u);
  assert.match(target.canonicalAggregateSha256, /^sha256:[0-9a-f]{64}$/u);
  assert.notEqual(legacy.canonicalAggregateSha256, target.canonicalAggregateSha256);
  assert.deepEqual(target.longRunningComposeRoles, ["api", "gateway", "keycloak", "worker"]);
  assert.equal(target.releaseSubjects.length, 7);
  assert.deepEqual(target.ecrRepositories, ["application", "cloudflared", "database-bootstrap", "gateway", "keycloak"]);
  assert.equal(target.secretsManagerContainers.length, 8);
  assert.deepEqual(target.logGroups, ["api", "cloudflared", "gateway", "host", "keycloak", "vpc-flow", "worker"]);
  for (const layer of ["8.1", "8.2", "8.3", "8.4", "8.5"]) assert.ok(target.layerDeltas[layer], layer);
});

test("checked-in target inventory matches stdout generator without rewriting the legacy file", () => {
  const result = spawnSync(process.execPath, ["scripts/build-aws-private-pilot-runtime-inventory.mjs", "target", "--stdout"], {
    cwd: root,
    encoding: "utf8",
  });
  assert.equal(result.status, 0, result.stderr);
  assert.deepEqual(JSON.parse(result.stdout), readInventory("runtime-inventory-target.json"));
});
