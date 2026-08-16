import test from "node:test";
import assert from "node:assert/strict";
import { buildArtifact, validateArtifact } from "./build-aga-ai-checklist-recommendations.mjs";

test("offline AI recommendation artifact binds all approved question versions", () => {
  const artifact = validateArtifact(buildArtifact());
  assert.equal(artifact.itemCount, 1310);
  assert.equal(new Set(artifact.items.map((item) => item.questionVersionId)).size, 1310);
  assert.equal(artifact.sourceCatalog.catalogRootDigest, "sha256:972f06005ba7befecb480d477334ea8cee542555d39d2604607f082deeee6e48");
  assert.equal(artifact.recommendationPolicy.runtimeModelCalls, false);
  assert.ok(artifact.items.some((item) => item.riskTier === "HIGH"));
  assert.ok(artifact.items.some((item) => item.riskTier === "LOW"));
  assert.ok(artifact.items.some((item) => item.riskTier === "UNKNOWN"));
});

test("offline AI recommendation artifact rejects drift", () => {
  const artifact = buildArtifact();
  artifact.items[0].questionVersionId = "qv:drift";
  assert.throws(() => validateArtifact(artifact), /artifact digest mismatch/);
});
