import { execFileSync } from "node:child_process";
import assert from "node:assert/strict";
import { createHash } from "node:crypto";
import { readFileSync } from "node:fs";
import { join } from "node:path";
import test from "node:test";

const root = join(import.meta.dirname, "..");
const v1 = join(root, "deliverables", "AGA_ALL_FORMS_SOURCE_RISK_DRAFT_2026-08-01.zip");
const v2 = join(root, "deliverables", "AGA_ALL_FORMS_APPROVED_SOURCE_V2.zip");
const build = JSON.parse(readFileSync(join(root, "deliverables", "AGA_ALL_FORMS_APPROVED_SOURCE_V2.build.json"), "utf8"));

function sha256(bytes) {
  return `sha256:${createHash("sha256").update(bytes).digest("hex")}`;
}

function entry(path, name) {
  return execFileSync("unzip", ["-p", path, name], { maxBuffer: 64 * 1024 * 1024 });
}

test("approved AGA V2 is deterministic and preserves the historical V1 boundary", () => {
  const v1Bytes = readFileSync(v1);
  const v2Bytes = readFileSync(v2);
  assert.equal(sha256(v1Bytes), "sha256:30700a88aeb5b26514bf7eb76bef050deb08b96294db94117d185de5c9f163b2");
  assert.equal(sha256(v2Bytes), build.packageSha256);
  assert.notEqual(sha256(v2Bytes), sha256(v1Bytes));
  assert.equal(build.formCount, 52);
  assert.equal(build.questionBoundaryFormCount, 31);
  assert.equal(build.questionCount, 1310);
  assert.equal(build.packageVersion, "AGA_ALL_FORMS_APPROVED_SOURCE_V2");
});

test("V2 carries source classification and complete immutable ordered inventory", () => {
  const packageJson = JSON.parse(entry(v2, "AGA_ALL_FORMS_APPROVED_SOURCE_V2.json"));
  const sourceManifest = JSON.parse(entry(v2, "AGA_APPROVED_SOURCE_MANIFEST.json"));
  const manifest = entry(v2, "MANIFEST.sha256").toString("utf8");
  assert.equal(packageJson.status, "IMPORTED_APPROVED_SOURCE");
  assert.equal(packageJson.candidateOnly, false);
  assert.equal(packageJson.catalogUsageClass, "GOVERNED_OPERATIONAL");
  assert.equal(packageJson.catalogOrigin, "IMPORTED_APPROVED_SOURCE");
  assert.equal(packageJson.optionalEnrichmentPolicy.status, "OPTIONAL_NOT_APPROVED");
  assert.equal(sourceManifest.orderedQuestions.length, 1310);
  assert.equal(sourceManifest.forms.length, 52);
  assert.equal(sourceManifest.questionCount, 1310);
  assert.equal(sourceManifest.catalogRootDigest, build.catalogRootDigest);
  assert.match(manifest, /AGA_ALL_FORMS_APPROVED_SOURCE_V2\.json/);
  assert.match(manifest, /AGA_APPROVED_SOURCE_MANIFEST\.json/);
  assert.doesNotMatch(manifest, /password|secret|token/i);
});
