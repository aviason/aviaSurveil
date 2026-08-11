import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import path from "node:path";
import test from "node:test";
import { fileURLToPath } from "node:url";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const migration = readFileSync(path.join(root, "apps/api/migrations/000044_document_render_leases_and_dispositions.up.sql"), "utf8");
const service = readFileSync(path.join(root, "apps/api/internal/documents/service.go"), "utf8");

test("legacy native-PDF migration is append-only and does not claim active legacy jobs", () => {
  assert.match(migration, /'SUPERSEDED_GOTENBERG'/u);
  assert.match(migration, /job\.status IN \('PENDING', 'FAILED'\)/u);
  assert.match(migration, /job\.status = 'RUNNING'.*job\.lease_expires_at IS NOT NULL/su);
  assert.doesNotMatch(migration, /UPDATE\s+outbox_messages/iu);
  assert.doesNotMatch(migration, /UPDATE\s+document_render_jobs/iu);
  assert.match(service, /AND job\.input_snapshot \? 'source'/u);
  assert.match(service, /SUPERSEDED_GOTENBERG/u);
});
