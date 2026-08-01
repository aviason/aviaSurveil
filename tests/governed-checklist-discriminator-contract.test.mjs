import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

test("entry_path, not generation-run nullability, is the governed discriminator", async () => {
  const source = await readFile("api/openapi/source/schemas/platform.json", "utf8");
  assert.match(source, /entryPath/);
  const dual = await readFile("apps/api/internal/regulatory/dual_authoring.go", "utf8");
  assert.match(dual, /GenerationRunID\s+string/);
  assert.match(dual, /GenerationRunID\s+string/);
  assert.doesNotMatch(dual, /GenerationRunID\s*:/);
  assert.doesNotMatch(dual, /generation_run_id\s+IS\s+(?:NOT\s+)?NULL/i);
  assert.match(dual, /SourceAuthorityAttestationID/);
  assert.match(dual, /SourceAuthorityAttestationResolver/);
  assert.doesNotMatch(dual, /AuthorityAccepted\s+bool/);
  const intake = await readFile("apps/api/internal/checklistintake/service.go", "utf8");
  assert.doesNotMatch(intake, /GenerationRunID|generation_run_id/);
  assert.match(intake, /PhaseArchiveValidate/);
  assert.doesNotMatch(intake, /ImportBatchInventoryComplete/);
});
