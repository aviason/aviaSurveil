import assert from "node:assert/strict";
import fs from "node:fs";
import path from "node:path";
import test from "node:test";
import { fileURLToPath } from "node:url";
import Ajv2020 from "../apps/web/node_modules/@redocly/ajv/dist/2020.js";

const repositoryRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const catalogPath = path.join(
  repositoryRoot,
  "docs/regulatory-sources/catalogs/service-provider-catalog.v1.json",
);
const schemaPath = path.join(
  repositoryRoot,
  "docs/regulatory-sources/schemas/service-provider-catalog.schema.json",
);

const expectedProviders = [
  ["AIR_OPERATOR", "Air Operator (AOC Holder)", "Flight Operations; Cabin Safety; Operational Control; Crew Training; Dangerous Goods; SMS; Security; Manuals", "Flight Operations Inspectorate (FOI)"],
  ["AMO", "Approved Maintenance Organization (AMO)", "Maintenance Procedures; Personnel; Tooling; Facilities; Quality System; SMS; Records", "Airworthiness Inspectorate"],
  ["CAMO", "Continuing Airworthiness Management Organization (CAMO)", "Airworthiness Management; Maintenance Programme; Reliability Programme; Technical Records", "Airworthiness Inspectorate"],
  ["ATO", "Approved Training Organization (ATO)", "Training Programmes; Instructors; Examinations; Training Records; Simulators", "Personnel Licensing & Training Department"],
  ["FSTD", "Flight Simulation Training Device (FSTD)", "Simulator Qualification; Configuration; Maintenance; Records", "Personnel Licensing & Training Department"],
  ["AERODROME_OPERATOR", "Aerodrome Operator", "Runways; Taxiways; Aprons; Lighting; RFFS; Wildlife; Obstacle Control; Emergency Plan; SMS", "Aerodrome Inspectorate"],
  ["ANSP", "Air Navigation Service Provider (ANSP)", "ATS; ATC Procedures; Airspace Management; SMS; Contingency Plans", "Air Navigation Services Inspectorate"],
  ["CNS_PROVIDER", "Communication Service Provider (CNS)", "Communication Systems; Navigation Aids; Surveillance Systems; Maintenance", "CNS Inspectorate"],
  ["AIS_AIM_PROVIDER", "AIS/AIM Provider", "AIP; NOTAM; Charts; Data Quality; Digital AIM", "AIS/AIM Inspectorate"],
  ["MET_PROVIDER", "Meteorological Service Provider (MET)", "Aviation Weather Services; Forecasting; Observations; MET Reports", "Meteorological Oversight Unit"],
  ["SAR_ORGANIZATION", "Search and Rescue (SAR) Organization", "Rescue Coordination; SAR Plans; Exercises; Readiness", "SAR Oversight Unit"],
  ["GROUND_HANDLING", "Ground Handling Organization", "Passenger Handling; Ramp Operations; Load Control; Baggage; Aircraft Servicing; SMS", "Ground Operations / Flight Operations Inspectorate"],
  ["FUEL_PROVIDER", "Fuel Service Provider", "Fuel Storage; Fuel Quality; Refuelling Procedures; Equipment; Personnel", "Aerodrome Inspectorate"],
  ["CARGO_REGULATED_AGENT", "Cargo Terminal / Regulated Agent", "Cargo Acceptance; Dangerous Goods; Security Controls; Documentation", "Aviation Security (AVSEC) + Dangerous Goods Office"],
  ["AVSEC_PROVIDER", "Aviation Security Service Provider", "Passenger Screening; Access Control; Hold Baggage Screening; Staff Training", "AVSEC Inspectorate"],
  ["RPAS_UAS_OPERATOR", "RPAS/UAS Operator", "Flight Operations; Remote Pilot Competency; Maintenance; Operational Risk Assessment; C2 Link", "RPAS / Flight Operations Inspectorate"],
  ["DOA", "Aircraft Design Organization (DOA)", "Design Approval; Compliance Demonstration; Configuration Control", "Airworthiness Certification Department"],
  ["POA", "Production Organization (POA)", "Production System; Quality Assurance; Product Conformity", "Airworthiness Certification Department"],
  ["AEMC", "Aviation Medical Centre (AeMC)", "Medical Facilities; Equipment; Medical Records; Personnel", "Aeromedical Department"],
  ["AME", "Aviation Medical Examiner (AME)", "Medical Examinations; Certification Procedures; Record Keeping", "Aeromedical Department"],
];

function readJson(filePath) {
  return JSON.parse(fs.readFileSync(filePath, "utf8"));
}

function validateCatalog(candidate) {
  const ajv = new Ajv2020({ allErrors: true, strict: false });
  const valid = ajv.validate(readJson(schemaPath), candidate);
  return { valid, errors: ajv.errors ?? [] };
}

function invalidCatalog(mutator) {
  const candidate = structuredClone(readJson(catalogPath));
  mutator(candidate);
  const result = validateCatalog(candidate);
  assert.equal(result.valid, false, JSON.stringify(result.errors));
}

test("catalog validation accepts exactly the supplied twenty provider identities and their source values", () => {
  const catalog = readJson(catalogPath);
  const result = validateCatalog(catalog);

  assert.equal(result.valid, true, JSON.stringify(result.errors));
  assert.equal(catalog.catalogVersion, "1.0.0");
  assert.deepEqual(catalog.targetKinds, [
    "ORGANIZATION",
    "PERSON",
    "FACILITY",
    "DEVICE",
    "SYSTEM",
    "ASSET",
    "LOCATION",
  ]);
  assert.equal(catalog.providers.length, 20);
  assert.deepEqual(
    catalog.providers.map((provider) => [
      provider.code,
      provider.label,
      provider.rawOversightTopics,
      provider.rawResponsibleCaaUnit,
    ]),
    expectedProviders,
  );

  for (const provider of catalog.providers) {
    assert.ok(provider.aliases.length > 0, `${provider.code} must retain an alias`);
    assert.ok(provider.targetKinds.length > 0, `${provider.code} must declare target kinds`);
    assert.ok(provider.topics.length > 0, `${provider.code} must declare individual topics`);
  }

  assert.deepEqual(catalog.providers.find((provider) => provider.code === "FSTD").targetKinds, ["DEVICE", "FACILITY"]);
  assert.deepEqual(catalog.providers.find((provider) => provider.code === "AME").targetKinds, ["PERSON"]);
  for (const code of ["AIS_AIM_PROVIDER", "GROUND_HANDLING", "CARGO_REGULATED_AGENT", "RPAS_UAS_OPERATOR"]) {
    const responsibility = catalog.providers.find((provider) => provider.code === code).responsibility;
    assert.equal(responsibility.normalizationStatus, "REVIEW_REQUIRED");
    assert.equal(Object.hasOwn(responsibility, "relationship"), false);
    assert.equal(Object.hasOwn(responsibility, "approvalRequired"), false);
  }
});

test("catalog schema rejects count, identity, source-value, target-kind, and ambiguous-ownership mutations", () => {
  invalidCatalog((catalog) => catalog.providers.pop());
  invalidCatalog((catalog) => catalog.providers.push(structuredClone(catalog.providers[0])));
  invalidCatalog((catalog) => { catalog.providers[1].code = "AIR_OPERATOR"; });
  invalidCatalog((catalog) => { catalog.providers[1].label = "Air Operator (AOC Holder)"; });
  invalidCatalog((catalog) => { catalog.providers.find((provider) => provider.code === "AME").code = "AEMC"; });
  invalidCatalog((catalog) => { catalog.providers.find((provider) => provider.code === "CNS_PROVIDER").code = "ANSP"; });
  invalidCatalog((catalog) => { catalog.providers[0].rawOversightTopics = "Flight Operations"; });
  invalidCatalog((catalog) => { catalog.providers[0].rawResponsibleCaaUnit = "Flight Operations Department"; });
  invalidCatalog((catalog) => { catalog.providers.find((provider) => provider.code === "FSTD").targetKinds = ["VEHICLE"]; });
  invalidCatalog((catalog) => { catalog.providers.find((provider) => provider.code === "FSTD").targetKinds = ["ORGANIZATION"]; });
  invalidCatalog((catalog) => {
    catalog.providers.find((provider) => provider.code === "AIR_OPERATOR").responsibility.normalizedParentApprovalDepartment = "Aeromedical Department";
  });
  invalidCatalog((catalog) => {
    catalog.providers.find((provider) => provider.code === "GROUND_HANDLING").responsibility = {
      normalizationStatus: "REVIEW_REQUIRED",
      relationship: "JOINT",
    };
  });
  invalidCatalog((catalog) => {
    catalog.providers.find((provider) => provider.code === "GROUND_HANDLING").responsibility = {
      normalizationStatus: "REVIEW_REQUIRED",
      normalizedParentApprovalDepartment: "Flight Operations Inspectorate (FOI)",
    };
  });
  invalidCatalog((catalog) => {
    catalog.providers.find((provider) => provider.code === "GROUND_HANDLING").responsibility = {
      normalizationStatus: "REVIEW_REQUIRED",
      approvalRequired: ["FLIGHT_OPERATIONS_INSPECTORATE"],
    };
  });
  invalidCatalog((catalog) => {
    catalog.providers.find((provider) => provider.code === "GROUND_HANDLING").responsibility = {
      normalizationStatus: "NORMALIZED",
      relationship: "PRIMARY",
      approvalRequired: ["FLIGHT_OPERATIONS"],
    };
  });
});

test("product authority keeps Department Manager technical review and publication separate without a technical-expert role", () => {
  const authorityDocuments = [
    "docs/product-specs/data-and-rules/CONCEPTUAL_DATA_MODEL.md",
    "docs/product-specs/data-and-rules/STATUS_PERMISSION_SECURITY.md",
    "docs/product-specs/workflows/AUDIT_CHECKLIST_WORKFLOW.md",
    "docs/product-specs/modules/AUDIT_PLANNING.md",
    "docs/product-specs/workflows/SURVEILLANCE_PLANNING_WORKFLOW.md",
    "docs/product-specs/screen-specs/DEPARTMENT_MANAGER_WORKSPACES.md",
    "docs/regulatory-sources/README.md",
  ];
  for (const relativePath of authorityDocuments) {
    const content = fs.readFileSync(path.join(repositoryRoot, relativePath), "utf8");
    assert.doesNotMatch(content, /technical[_ -]?expert/i, `${relativePath} must not introduce a technical-expert role`);
  }

  const conceptualModel = fs.readFileSync(
    path.join(repositoryRoot, "docs/product-specs/data-and-rules/CONCEPTUAL_DATA_MODEL.md"),
    "utf8",
  );
  const permissionRules = fs.readFileSync(
    path.join(repositoryRoot, "docs/product-specs/data-and-rules/STATUS_PERMISSION_SECURITY.md"),
    "utf8",
  );
  assert.match(conceptualModel, /TECHNICAL_REVIEW_REQUIRED/);
  assert.match(conceptualModel, /EXPERT_REVIEW_REQUIRED[\s\S]{0,240}legacy/i);
  assert.match(conceptualModel, /several provider scopes/i);
  assert.match(conceptualModel, /ORGANIZATION, PERSON, FACILITY, DEVICE, SYSTEM, ASSET, and LOCATION/);
  assert.match(permissionRules, /Start or import a checklist generation run/);
  assert.match(permissionRules, /Technically approve/);
  assert.match(permissionRules, /Publish a technically approved checklist version/);
  assert.match(permissionRules, /separate recorded action/i);

  const allowedRoles = ["inspector", "leadInspector", "manager", "finance", "gm", "executiveDirector", "auditee", "admin"];
  const openApi = readJson(path.join(repositoryRoot, "api/openapi/source/openapi.json"));
  assert.deepEqual(openApi.components.schemas.Role.enum, allowedRoles);

  const roleAndActionFiles = [
    "apps/api/internal/identity/principal.go",
    "apps/api/internal/testprofile/regulatory_pilot.go",
    "apps/web/src/backend/backend.ts",
    "apps/web/src/mock/create-mock-backend.ts",
    "apps/web/src/mock/seed-data.ts",
    "apps/web/src/app/route-contracts.ts",
    "apps/web/src/ui/role-navigation.tsx",
  ];
  for (const relativePath of roleAndActionFiles) {
    const content = fs.readFileSync(path.join(repositoryRoot, relativePath), "utf8");
    assert.doesNotMatch(content, /technical[_ -]?expert/i, `${relativePath} must not introduce a technical-expert role or visible action`);
  }
});
