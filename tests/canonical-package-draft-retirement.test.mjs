import assert from "node:assert/strict";
import { existsSync, readFileSync } from "node:fs";
import test from "node:test";

const read = (file) => readFileSync(file, "utf8");

test("the fixed-ID Inspection Package Draft surface is absent from the normal product", () => {
  const forbidden = /inspection-package-drafts|InspectionPackageDraft|packageDrafts|PKG-AUD-2026-001-CABIN/u;
  for (const file of [
    "api/openapi/source/paths/workflows.json",
    "api/openapi/source/schemas/domain.json",
    "apps/api/internal/httpapi/canonical_api.go",
    "apps/api/internal/httpapi/task4_api.go",
    "apps/web/src/backend/backend-contracts.ts",
    "apps/web/src/backend/backend.ts",
    "apps/web/src/backend/http-backend.ts",
    "apps/web/src/backend/transport-mappers.ts",
    "apps/web/src/app/route-contracts.ts",
    "apps/web/src/app/screen-component-registry.tsx",
  ]) {
    assert.doesNotMatch(read(file), forbidden, `${file} still exposes the retired fixed-ID package draft`);
  }
  assert.equal(
    existsSync("apps/web/src/features/inspections/inspection-package-builder-page.tsx"),
    false,
    "the retired Department Manager package builder page must be physically removed",
  );
  assert.equal(
    existsSync("apps/api/internal/inspections/draft_service.go"),
    false,
    "the retired mutable package draft service must be physically removed",
  );
});

test("migration 42 removes the obsolete mutable package draft table", () => {
  const migration = read("apps/api/migrations/000042_remove_inspection_package_drafts.up.sql");
  assert.match(migration, /DROP TABLE inspection_package_drafts/u);
  assert.doesNotMatch(read("apps/api/internal/configuration/workspace.go"), /inspection_package_drafts/u);
  assert.doesNotMatch(read("apps/api/internal/testprofile/canonical.go"), /inspection_package_drafts/u);
});
