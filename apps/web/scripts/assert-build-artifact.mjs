#!/usr/bin/env node
import assert from "node:assert/strict";
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const CURRENT_VECTOR = Object.freeze({
  appShellVersion: 9,
  indexedDbSchemaVersion: 2,
  packageSchemaVersion: 1,
  syncProtocolVersion: 1,
});

const EXPECTED_BY_DIRECTORY = Object.freeze({
  demo: "demo",
  http: "http",
  "production-offline": "production-offline",
});

export function assertBuildArtifact(suppliedPath = "dist") {
  const suppliedRoot = path.resolve(suppliedPath);
  const artifactRoot = fs.existsSync(path.join(suppliedRoot, "build-artifact.json"))
    ? suppliedRoot
    : path.join(suppliedRoot, path.basename(suppliedRoot));
  assert.ok(fs.existsSync(artifactRoot), `Build artifact directory is missing: ${artifactRoot}`);
  const descriptorPath = path.join(artifactRoot, "build-artifact.json");
  assert.ok(fs.existsSync(descriptorPath), `Build artifact descriptor is missing: ${descriptorPath}`);
  const descriptor = JSON.parse(fs.readFileSync(descriptorPath, "utf8"));
  assert.deepEqual(Object.keys(descriptor).sort(), [
    "acceptanceLane",
    "artifactIdentity",
    "backend",
    "buildProfile",
    "cacheNamespace",
    "lane",
    "offlineVersionVector",
    "schemaVersion",
    "status",
  ]);
  assert.equal(descriptor.schemaVersion, 1);
  assert.deepEqual(descriptor.offlineVersionVector, CURRENT_VECTOR);
  assert.equal(descriptor.status, "candidate-only");
  assert.ok(!Object.hasOwn(descriptor, "productionReady"));
  assert.match(descriptor.artifactIdentity, /^aviasurveil360-[a-z0-9-]+$/);
  assert.match(descriptor.cacheNamespace, /^aviasurveil360-[a-z0-9-]+-app-shell$/);
  const expectedLane = EXPECTED_BY_DIRECTORY[path.basename(artifactRoot)];
  if (expectedLane) assert.equal(descriptor.lane, expectedLane);
  if (descriptor.lane === "demo") {
    assert.equal(descriptor.buildProfile, "demo");
    assert.equal(descriptor.backend, "mock");
    assert.equal(descriptor.acceptanceLane, "demo-online-mock");
  } else {
    assert.equal(descriptor.buildProfile, "http");
    assert.equal(descriptor.backend, "http");
    assert.doesNotMatch(descriptor.artifactIdentity, /demo|mock/i);
  }
  const indexSource = fs.readFileSync(path.join(artifactRoot, "index.html"), "utf8");
  assert.match(indexSource, new RegExp(`content="${descriptor.lane}"`));
  assert.doesNotMatch(indexSource, /__AVIA_BUILD_LANE__/);
  return descriptor;
}

const invokedPath = process.argv[1] ? path.resolve(process.argv[1]) : null;
if (invokedPath === fileURLToPath(import.meta.url)) {
  const descriptor = assertBuildArtifact(process.argv[2] ?? "dist");
  console.log(`build-artifact-scan: ok (${descriptor.lane}, ${descriptor.artifactIdentity})`);
}
