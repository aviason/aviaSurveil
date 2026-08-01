import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

const specificationPaths = {
  builder: "docs/product-specs/modules/CHECKLIST_BUILDER_AND_RUNNER.md",
  planning: "docs/product-specs/modules/AUDIT_PLANNING.md",
  admin: "docs/product-specs/modules/ADMIN_CONFIGURATION.md",
  model: "docs/product-specs/data-and-rules/CONCEPTUAL_DATA_MODEL.md",
  security: "docs/product-specs/data-and-rules/STATUS_PERMISSION_SECURITY.md",
  screens: "docs/product-specs/screen-specs/SCREEN_INVENTORY_AND_FORMS.md",
  manager: "docs/product-specs/screen-specs/DEPARTMENT_MANAGER_WORKSPACES.md",
};

async function readSpecifications() {
  return Object.fromEntries(await Promise.all(
    Object.entries(specificationPaths).map(async ([name, path]) => [name, await readFile(path, "utf8")]),
  ));
}

test("Gate 0 freezes the governed intake lifecycle and source-authority separation", async () => {
  const specs = await readSpecifications();
  const combined = Object.values(specs).join("\n");

  for (const phrase of [
    "EXISTING_CHECKLIST_CANDIDATE",
    "REGULATORY_TRACE",
    "HYBRID_RECONCILED",
    "SOURCE_MAPPING_REQUIRED",
    "REGULATORY_SOURCE_OWNER",
    "CHECKLIST_REVIEWER",
    "REVIEWED_SOURCE_SET",
    "OFFICIAL_CHECKLIST_SOURCE_CHAIN_V1",
    "AGA_ZIP_PDF_V1",
    "source-authority acceptance",
    "candidate mapping attestation",
    "technical approval and publication are separate",
  ]) {
    assert.match(combined, new RegExp(phrase.replace(/[.*+?^${}()|[\]\\]/g, "\\$&"), "i"));
  }

  assert.match(specs.model, /Existing checklist material is non-authoritative candidate input/i);
  assert.match(specs.model, /every required[\s\S]*separate[\s\S]*append-only source-authority acceptance/i);
  assert.match(specs.model, /server derives immutable required-owner facts/i);
  assert.match(specs.builder, /SOURCE_MAPPING_REQUIRED[\s\S]*cannot be published/i);
  assert.match(specs.planning, /computed[\s\S]*Audit-package eligibility[\s\S]*separate/i);
});

test("Gate 0 documents bounded AGA evidence and blocked functional-assignment provisioning", async () => {
  const specs = await readSpecifications();
  const combined = Object.values(specs).join("\n");

  for (const phrase of [
    "Maximum archive bytes: 50 MiB",
    "Maximum PDF entries: 128",
    "Maximum per-file and whole-archive expansion ratio: 20:1",
    "dd819cfa6a670760e0cfceed94496e2e466dc53bac13e6fd792b1128314d6e32",
    "29ed8384693b615926fc42a0ca4654be2ea9a36b0946f217975571ca0ad9564f",
    "495aa7b0a1edca1ac5e874e6a63f50b47c6d207aa264cc390970a7db1acdc6e3",
    "functional-assignment provisioning is blocked",
    "synthetic fixtures only",
  ]) {
    assert.match(combined, new RegExp(phrase.replace(/[.*+?^${}()|[\]\\]/g, "\\$&"), "i"));
  }

  assert.match(specs.admin, /no Admin grant\/revoke[\s\S]*route/i);
  assert.match(specs.security, /missing or ambiguous ownership fails closed/i);
  assert.match(specs.screens, /archive[\s\S]*never[\s\S]*exposes raw extracted text/i);
});
