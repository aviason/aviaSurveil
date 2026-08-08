import assert from "node:assert/strict";
import fs from "node:fs";
import path from "node:path";
import test from "node:test";
import { fileURLToPath } from "node:url";
import { assembleOpenApi } from "../../../scripts/bundle-openapi.mjs";

const repositoryRoot = path.resolve(
  path.dirname(fileURLToPath(import.meta.url)),
  "../../..",
);
const openApiPath = path.join(repositoryRoot, "api/openapi/aviasurveil360.yaml");
const vocabularyPath = path.join(
  repositoryRoot,
  "docs/product-specs/data-and-rules/PRODUCTION_CONTRACT_VOCABULARY.md",
);
const examplesDirectory = path.join(
  repositoryRoot,
  "api/openapi/examples/canonical",
);
const fullPlatformExamplesDirectory = path.join(
  repositoryRoot,
  "api/openapi/examples/full-platform",
);
const sourceDirectory = path.join(repositoryRoot, "api/openapi/source");
const sourceFragmentPaths = [
  "openapi.json",
  "paths/core.json",
  "paths/workflows.json",
  "paths/platform.json",
  "schemas/domain.json",
  "schemas/platform.json",
];

const frozenCapabilityOperationIds = {
  "assignments.list": "listAssignments",
  "inspections.getPackage": "getInspectionPackage",
  "inspections.checkout": "checkoutInspectionPackage",
  "inspections.upsertChecklistResponse": "upsertChecklistResponse",
  "inspections.submitChecklist": "submitChecklist",
  "inspections.reopenChecklist": "reopenChecklist",
  "potentialFindings.list": "listPotentialFindings",
  "potentialFindings.get": "getPotentialFinding",
  "potentialFindings.create": "createPotentialFinding",
  "potentialFindings.decide": "decidePotentialFinding",
  "findings.list": "listFindings",
  "findings.get": "getFinding",
  "findings.authorizedClose": "authorizedCloseFinding",
  "caps.listRevisions": "listCapRevisions",
  "caps.getRevision": "getCapRevision",
  "caps.submit": "submitCap",
  "caps.review": "reviewCap",
  "inspectionAttachments.beginUpload": "beginInspectionAttachmentUpload",
  "inspectionAttachments.completeUpload": "completeInspectionAttachmentUpload",
  "evidence.beginUpload": "beginEvidenceUpload",
  "evidence.completeUpload": "completeEvidenceUpload",
  "evidence.listVersions": "listEvidenceVersions",
  "evidence.review": "reviewEvidence",
  "reports.getVersion": "getReportVersion",
  "reports.decide": "decideReport",
  "dashboards.getManagerProjection": "getManagerDashboard",
  "organizations.list": "listOrganizations",
  "planning.list": "listPlanningItems",
  "planning.decide": "decidePlanningItem",
  "planningIntake.getDraft": "getPlanningIntakeDraft",
  "planningIntake.saveDraft": "savePlanningIntakeDraft",
  "planningIntake.submit": "submitPlanningIntake",
  "packageDrafts.get": "getInspectionPackageDraft",
  "packageDrafts.save": "saveInspectionPackageDraft",
  "configuration.listChecklistTemplateVersions": "listChecklistTemplateVersions",
  "configuration.getChecklistTemplateVersion": "getChecklistTemplateVersion",
  "configuration.listReminderRules": "listReminderRules",
  "auditTrail.list": "listAuditEvents",
  "sync.pushOperation": "pushFieldOperation",
  "sync.pull": "pullSyncChanges",
  "communications.list": "listCommunications",
  "communications.send": "sendCommunication",
  "calendar.list": "listCalendarItems",
  "calendar.openItem": "getCalendarItem",
  "profiles.getMine": "getMyProfile",
  "profiles.updateMine": "updateMyProfile",
  "teams.list": "listTeamMembers",
  "teams.openMember": "getTeamMember",
  "teams.listAuditTeams": "listAuditTeams",
  "teams.openAuditTeam": "getAuditTeam",
  "risk.getOverview": "getRiskOverview",
  "risk.getManagementProjection": "getRiskManagementProjection",
  "risk.openFinding": "getFinding",
  "documents.list": "listDocuments",
  "documents.open": "getDocument",
  "auditeeCoordination.list": "listAuditeeCoordination",
  "auditeeCoordination.respond": "respondAuditeeCoordination",
  "auditeeReports.listReleased": "listAuditeeReleasedReports",
  "auditeeReports.getReleased": "getAuditeeReleasedReport",
  "notifications.list": "listNotifications",
  "notifications.markRead": "markNotificationRead",
  "administration.getScreenProjection": "getAdministrationScreenProjection",
  "administration.listScreenProjections": "listAdministrationScreenProjections",
  "administration.invokeVisibleAction": "invokeAdministrationVisibleAction",
  "adminWorkspace.listRegulatoryReferences": "listRegulatoryReferences",
  "adminWorkspace.listTemplateMasters": "listTemplateMasters",
  "adminWorkspace.listQuestions": "listAdminQuestions",
  "adminWorkspace.createQuestion": "createAdminQuestion",
  "adminWorkspace.getTemplate": "getAdminTemplate",
  "adminWorkspace.createDraft": "createAdminTemplateDraft",
  "adminWorkspace.addDraftQuestion": "addAdminTemplateDraftQuestion",
  "adminWorkspace.moveDraftQuestion": "moveAdminTemplateDraftQuestion",
  "adminWorkspace.getInspectionPackage": "getAdminInspectionPackage",
  "adminWorkspace.listReportDefinitions": "listReportDefinitions",
  "adminWorkspace.listAccessDirectory": "listAccessDirectory",
  "adminWorkspace.listOrganizations": "listAdminOrganizations",
  "adminWorkspace.getOrganization": "getAdminOrganization",
  "adminWorkspace.listAuditEvents": "listAdminAuditEvents",
  "assistantDrafts.getGuidance": "getAssistantGuidance",
  "assistantDrafts.createDraft": "createAssistantDraft",
};

function readRequiredJson(filePath) {
  assert.ok(fs.existsSync(filePath), `Required contract file is missing: ${filePath}`);
  return JSON.parse(fs.readFileSync(filePath, "utf8"));
}

function resolveSchema(document, schema) {
  if (!schema?.$ref) return schema;
  const prefix = "#/components/schemas/";
  assert.ok(schema.$ref.startsWith(prefix), `Unsupported schema reference: ${schema.$ref}`);
  const name = schema.$ref.slice(prefix.length);
  assert.ok(document.components.schemas[name], `Unknown schema reference: ${name}`);
  return document.components.schemas[name];
}

function validateValue(document, schemaInput, value, pointer = "$") {
  const schema = resolveSchema(document, schemaInput);
  if (schema.oneOf) {
    const matches = schema.oneOf.filter((candidate) => {
      try {
        validateValue(document, candidate, value, pointer);
        return true;
      } catch {
        return false;
      }
    });
    assert.equal(matches.length, 1, `${pointer} must match exactly one union member`);
    return;
  }

  if (schema.enum) {
    assert.ok(schema.enum.includes(value), `${pointer} is not an approved enum value: ${value}`);
  }

  const declaredTypes = Array.isArray(schema.type) ? schema.type : [schema.type];
  if (value === null && declaredTypes.includes("null")) return;
  if (schema.type === "null") {
    assert.equal(value, null, `${pointer} must be null`);
    return;
  }

  if (schema.type === "object") {
    assert.equal(typeof value, "object", `${pointer} must be an object`);
    assert.notEqual(value, null, `${pointer} must not be null`);
    assert.ok(!Array.isArray(value), `${pointer} must not be an array`);
    for (const key of schema.required ?? []) {
      assert.ok(Object.hasOwn(value, key), `${pointer}.${key} is required`);
    }
    if (schema.additionalProperties === false) {
      const allowed = new Set(Object.keys(schema.properties ?? {}));
      for (const key of Object.keys(value)) {
        assert.ok(allowed.has(key), `${pointer}.${key} is not allowed`);
      }
    }
    for (const [key, child] of Object.entries(schema.properties ?? {})) {
      if (Object.hasOwn(value, key)) validateValue(document, child, value[key], `${pointer}.${key}`);
    }
    return;
  }

  if (schema.type === "array") {
    assert.ok(Array.isArray(value), `${pointer} must be an array`);
    for (const [index, entry] of value.entries()) {
      validateValue(document, schema.items, entry, `${pointer}[${index}]`);
    }
    return;
  }

  if (schema.type === "string") assert.equal(typeof value, "string", `${pointer} must be a string`);
  if (schema.type === "integer") assert.ok(Number.isInteger(value), `${pointer} must be an integer`);
  if (schema.type === "number") assert.equal(typeof value, "number", `${pointer} must be a number`);
  if (schema.type === "boolean") assert.equal(typeof value, "boolean", `${pointer} must be a boolean`);
}

function assertClosedResponseSchema(
  document,
  schemaInput,
  pointer,
  seen = new Set(),
  requireContainer = true,
) {
  const schema = resolveSchema(document, schemaInput);
  if (schemaInput?.$ref) {
    if (seen.has(schemaInput.$ref)) return;
    seen.add(schemaInput.$ref);
  }
  for (const composition of ["oneOf", "anyOf", "allOf"]) {
    if (!schema[composition]) continue;
    for (const [index, member] of schema[composition].entries()) {
      assertClosedResponseSchema(
        document,
        member,
        `${pointer}.${composition}[${index}]`,
        seen,
        requireContainer,
      );
    }
    return;
  }
  const declaredTypes = Array.isArray(schema.type) ? schema.type : [schema.type];
  if (declaredTypes.includes("array")) {
    assertClosedResponseSchema(document, schema.items, `${pointer}.items`, seen, false);
    return;
  }
  if (declaredTypes.includes("object")) {
    // Distribution maps intentionally accept governed labels supplied by the
    // catalog (for example, "Cabin Safety"). The map keys are data, while
    // every value schema remains closed and typed.
    if (schema.additionalProperties && typeof schema.additionalProperties === "object") {
      assertClosedResponseSchema(document, schema.additionalProperties, `${pointer}.<value>`, seen, false);
    } else {
      assert.equal(schema.additionalProperties, false, `${pointer} must reject unknown fields`);
    }
    for (const pattern of Object.values(schema.patternProperties ?? {})) {
      assertClosedResponseSchema(document, pattern, `${pointer}.<pattern>`, seen, false);
    }
    for (const [name, property] of Object.entries(schema.properties ?? {})) {
      assertClosedResponseSchema(document, property, `${pointer}.${name}`, seen, false);
    }
    return;
  }
  assert.equal(
    requireContainer,
    false,
    `${pointer} must resolve to an object, array, or closed union`,
  );
}

test("the minimal OpenAPI contract and canonical vocabulary exist", () => {
  assert.ok(fs.existsSync(vocabularyPath), "Canonical English vocabulary is missing");
  const document = readRequiredJson(openApiPath);
  assert.equal(document.openapi, "3.1.0");
  assert.equal(document.info.title, "AviaSurveil360 API");
  assert.match(fs.readFileSync(vocabularyPath, "utf8"), /Canonical transport values/);
});

test("readiness exposes only closed named dependency states", () => {
  const document = readRequiredJson(openApiPath);
  const operation = document.paths["/health/ready"].get;
  for (const status of ["200", "503"]) {
    const response = operation.responses[status];
    const schema = response.content?.["application/json"]?.schema;
    assert.equal(
      schema?.$ref,
      "#/components/schemas/ReadinessResponse",
      `readiness ${status} must use the safe named report`,
    );
  }
  const report = document.components.schemas.ReadinessResponse;
  const dependency = document.components.schemas.ReadinessDependency;
  assert.equal(report.additionalProperties, false);
  assert.deepEqual(report.required, ["status", "dependencies"]);
  assert.deepEqual(report.properties.status.enum, [
    "ready",
    "degraded",
    "not_ready",
  ]);
  assert.equal(dependency.additionalProperties, false);
  assert.deepEqual(dependency.required, ["name", "required", "status"]);
  assert.equal(
    Object.hasOwn(dependency.properties, "error"),
    false,
    "readiness must not serialize downstream error details",
  );
});

test("every frozen Plan 1 backend capability maps to a bundled operation ID", () => {
  const document = readRequiredJson(openApiPath);
  const bundledOperationIds = new Set(
    Object.values(document.paths).flatMap((pathItem) =>
      Object.values(pathItem)
        .map((operation) => operation.operationId)
        .filter(Boolean),
    ),
  );
  const missing = Object.entries(frozenCapabilityOperationIds)
    .filter(([, operationId]) => !bundledOperationIds.has(operationId))
    .map(([capability, operationId]) => `${capability} -> ${operationId}`);

  assert.equal(
    Object.keys(frozenCapabilityOperationIds).length,
    80,
    "The frozen Plan 1 capability inventory must remain explicit",
  );
  assert.deepEqual(
    missing,
    [],
    `Missing full-platform operation IDs:\n${missing.join("\n")}`,
  );
});

test("deterministic source fragments reproduce the bundled OpenAPI artifact", () => {
  const missingFragments = sourceFragmentPaths.filter(
    (relativePath) => !fs.existsSync(path.join(sourceDirectory, relativePath)),
  );
  assert.deepEqual(
    missingFragments,
    [],
    `Missing deterministic OpenAPI source fragments:\n${missingFragments.join("\n")}`,
  );
  for (const relativePath of sourceFragmentPaths.slice(1)) {
    assert.ok(
      Object.keys(readRequiredJson(path.join(sourceDirectory, relativePath))).length > 0,
      `${relativePath} must own part of the modular contract`,
    );
  }

  assert.deepEqual(assembleOpenApi(sourceDirectory), readRequiredJson(openApiPath));
});

test("full-platform examples validate and preserve Auditee-safe closed projections", () => {
  const document = readRequiredJson(openApiPath);
  assert.ok(
    fs.existsSync(fullPlatformExamplesDirectory),
    "Full-platform example directory is missing",
  );
  const files = fs
    .readdirSync(fullPlatformExamplesDirectory)
    .filter((file) => file.endsWith(".json"));
  assert.ok(files.length > 0, "At least one full-platform JSON example is required");

  for (const file of files) {
    const envelope = readRequiredJson(path.join(fullPlatformExamplesDirectory, file));
    assert.equal(typeof envelope.schema, "string", `${file} must declare a schema`);
    assert.ok(Object.hasOwn(envelope, "value"), `${file} must declare a value`);
    const schema = document.components.schemas[envelope.schema];
    assert.ok(schema, `${file} references missing schema ${envelope.schema}`);
    assert.equal(
      resolveSchema(document, schema).additionalProperties,
      false,
      `${envelope.schema} must be closed`,
    );
    validateValue(document, schema, envelope.value);
    if (envelope.schema.startsWith("Auditee")) {
      assert.doesNotMatch(
        JSON.stringify(envelope.value),
        /internalCaaNote|internalRisk|inspectorWorkload|enforcementDeliberation/i,
      );
    }
  }
});

test("every full-platform operation declares role security and every mutation declares command guards", () => {
  const document = readRequiredJson(openApiPath);
  assert.ok(
    document.components.securitySchemes?.oidc,
    "The full-platform contract must declare OIDC role and organization security",
  );

  const mutationMethods = new Set(["post", "put", "patch", "delete"]);
  for (const [route, pathItem] of Object.entries(document.paths)) {
    for (const [method, operation] of Object.entries(pathItem)) {
      if (route.startsWith("/health/")) continue;
      assert.ok(operation.security?.length, `${operation.operationId} must declare role security`);
      if (!mutationMethods.has(method)) continue;
      const parameterRefs = new Set(
        (operation.parameters ?? []).map((parameter) => parameter.$ref),
      );
      if (operation["x-operation-kind"] === "query") {
        assert.ok(
          parameterRefs.has("#/components/parameters/CsrfToken"),
          `${operation.operationId} must declare CSRF`,
        );
        assert.ok(
          !parameterRefs.has("#/components/parameters/IdempotencyKey"),
          `${operation.operationId} query must not declare Idempotency-Key`,
        );
        assert.ok(
          !parameterRefs.has("#/components/parameters/ExpectedRevision"),
          `${operation.operationId} query must not declare expected revision`,
        );
        continue;
      }
      assert.ok(
        parameterRefs.has("#/components/parameters/IdempotencyKey"),
        `${operation.operationId} must declare Idempotency-Key`,
      );
      assert.ok(
        parameterRefs.has("#/components/parameters/CsrfToken"),
        `${operation.operationId} must declare CSRF`,
      );
      assert.ok(
        parameterRefs.has("#/components/parameters/ExpectedRevision"),
        `${operation.operationId} must declare expected revision`,
      );
      for (const status of ["400", "401", "403", "409", "412", "422"]) {
        if (operation["x-neutral-denial"] === true && (status === "401" || status === "403")) continue;
        const response = operation.responses?.[status];
        const problemReference =
          response?.$ref ??
          response?.content?.["application/problem+json"]?.schema?.$ref;
        assert.ok(
          ["#/components/responses/Problem", "#/components/schemas/Problem"].includes(
            problemReference,
          ),
          `${operation.operationId} must declare typed ${status} problem response`,
        );
      }
    }
  }
});

test("every JSON response resolves to a closed schema", () => {
  const document = readRequiredJson(openApiPath);
  for (const [route, pathItem] of Object.entries(document.paths)) {
    for (const [method, operation] of Object.entries(pathItem)) {
      for (const [status, responseInput] of Object.entries(operation.responses ?? {})) {
        const response = responseInput.$ref
          ? document.components.responses[responseInput.$ref.split("/").at(-1)]
          : responseInput;
        for (const [mediaType, media] of Object.entries(response.content ?? {})) {
          if (!mediaType.endsWith("json")) continue;
          assertClosedResponseSchema(
            document,
            media.schema,
            `${method.toUpperCase()} ${route} ${status} ${mediaType}`,
          );
        }
      }
    }
  }
});

test("full-platform schemas preserve exact frozen backend transport shapes", () => {
  const document = readRequiredJson(openApiPath);
  const schemas = document.components.schemas;

  assert.equal(
    schemas.SavePlanningIntakeDraftInput.properties.values.$ref,
    "#/components/schemas/PlanningIntakeDraftValues",
  );
  assert.ok(!schemas.PlanningIntakeDraftValues.required.includes("id"));
  assert.ok(!schemas.PlanningIntakeDraftValues.required.includes("revision"));

  const inspectionTeamRequired = new Set(schemas.InspectionTeamAuditView.required);
  for (const field of ["assignments", "documents", "history"]) {
    assert.ok(inspectionTeamRequired.has(field), `InspectionTeamAuditView must require ${field}`);
  }

  const managementProjection = schemas.RiskManagementProjectionView;
  const riskFinding = managementProjection.properties.findings.items;
  for (const field of ["inspectionId", "inspectionTitle", "department", "issuedAt"]) {
    assert.ok(riskFinding.required.includes(field), `Risk finding projection must require ${field}`);
  }
  const capEffectiveness = managementProjection.properties.capEffectiveness.items;
  for (const field of [
    "closureBasis",
    "capId",
    "capRevisionId",
    "capRevision",
    "capStatus",
  ]) {
    assert.ok(
      capEffectiveness.required.includes(field),
      `CAP effectiveness projection must require ${field}`,
    );
  }

  assert.equal(
    schemas.AdministrationScreenProjection.properties.visibleActions.items.$ref,
    "#/components/schemas/VisibleScreenAction",
  );
  assert.equal(
    schemas.VisibleActionResult.properties.effect.$ref,
    "#/components/schemas/VisibleActionEffect",
  );
  assert.equal(schemas.VisibleActionEffect.discriminator.propertyName, "type");
});

test("full-platform operation role metadata matches the frozen capability matrix", () => {
  const document = readRequiredJson(openApiPath);
  const byOperationId = new Map(
    Object.values(document.paths).flatMap((pathItem) =>
      Object.values(pathItem).map((operation) => [operation.operationId, operation]),
    ),
  );
  const expectedRoles = {
    listCommunications: ["inspector", "leadInspector", "manager", "auditee"],
    sendCommunication: ["inspector", "leadInspector", "manager", "auditee"],
    listCalendarItems: ["inspector", "leadInspector", "manager", "auditee"],
    getCalendarItem: ["inspector", "leadInspector", "manager", "auditee"],
    listDocuments: ["inspector", "leadInspector", "manager", "auditee", "admin"],
    getDocument: ["inspector", "leadInspector", "manager", "auditee", "admin"],
    getRiskOverview: ["manager"],
    getRiskManagementProjection: ["manager"],
    getAdministrationScreenProjection: [
      "inspector",
      "leadInspector",
      "manager",
      "finance",
      "gm",
      "executiveDirector",
      "auditee",
      "admin",
    ],
    listAdministrationScreenProjections: [
      "inspector",
      "leadInspector",
      "manager",
      "finance",
      "gm",
      "executiveDirector",
      "auditee",
      "admin",
    ],
    invokeAdministrationVisibleAction: [
      "inspector",
      "leadInspector",
      "manager",
      "finance",
      "gm",
      "executiveDirector",
      "auditee",
      "admin",
    ],
    listRegulatoryReferences: ["admin"],
    createAdminQuestion: ["admin"],
    getAssistantGuidance: ["inspector", "leadInspector"],
    createAssistantDraft: ["inspector", "leadInspector"],
    listAuditeeCoordination: ["auditee"],
    respondAuditeeCoordination: ["auditee"],
    listAuditeeReleasedReports: ["auditee"],
    getAuditeeReleasedReport: ["auditee"],
  };

  for (const [operationId, roles] of Object.entries(expectedRoles)) {
    assert.deepEqual(
      byOperationId.get(operationId)?.["x-authorized-roles"],
      roles,
      `${operationId} role metadata must match the frozen capability matrix`,
    );
  }
});
test("every canonical JSON example validates against its declared schema", () => {
  const document = readRequiredJson(openApiPath);
  assert.ok(fs.existsSync(examplesDirectory), "Canonical example directory is missing");
  const files = fs.readdirSync(examplesDirectory).filter((file) => file.endsWith(".json"));
  assert.ok(files.length > 0, "At least one canonical JSON example is required");
  for (const file of files) {
    const envelope = readRequiredJson(path.join(examplesDirectory, file));
    assert.equal(typeof envelope.schema, "string", `${file} must declare a schema`);
    assert.ok(Object.hasOwn(envelope, "value"), `${file} must declare a value`);
    validateValue(document, { $ref: `#/components/schemas/${envelope.schema}` }, envelope.value);
  }
});

test("Auditee projections are closed and structurally omit internal CAA data", () => {
  const document = readRequiredJson(openApiPath);
  const schemas = document.components.schemas;
  const auditeeSchemaNames = Object.keys(schemas).filter((name) => name.startsWith("Auditee"));
  assert.ok(auditeeSchemaNames.length > 0, "At least one Auditee-specific schema is required");
  for (const name of auditeeSchemaNames) {
    const source = JSON.stringify(schemas[name]);
    assert.equal(schemas[name].additionalProperties, false, `${name} must reject unknown fields`);
    assert.doesNotMatch(source, /internalCaaNote|internalRisk|inspectorWorkload|enforcementDeliberation/i);
  }
});

test("sync request and response payloads use closed discriminated unions", () => {
  const document = readRequiredJson(openApiPath);
  const schemas = document.components.schemas;
  assert.ok(schemas.FieldSyncOperation.oneOf?.length >= 4, "FieldSyncOperation must be a typed union");
  assert.equal(schemas.FieldSyncOperation.discriminator?.propertyName, "commandType");
  assert.ok(schemas.AuthorizedSyncChange.oneOf?.length >= 4, "AuthorizedSyncChange must be a typed union");
  assert.equal(schemas.AuthorizedSyncChange.discriminator?.propertyName, "kind");
  assert.equal(schemas.AuthorizedConflictDescriptor.additionalProperties, false);
});

test("first-production route families have versioned paths, closed schemas, and canonical examples", () => {
  const document = readRequiredJson(openApiPath);
  for (const route of [
    "/v1/organizations",
    "/v1/planning/items",
    "/v1/planning/items/{id}/decisions",
    "/v1/configuration/checklist-template-versions",
    "/v1/configuration/checklist-template-versions/{templateVersionId}",
    "/v1/configuration/reminder-rules",
    "/v1/audit-events",
  ]) {
    assert.ok(document.paths[route], `Missing first-production path: ${route}`);
  }
  for (const schemaName of [
    "OrganizationSummary",
    "PlanningItemView",
    "PlanningDecisionInput",
    "ChecklistTemplateVersionView",
    "ChecklistTemplateQuestionView",
    "ChecklistTemplateVersionDetailView",
    "ReminderRuleView",
    "AuditEventView",
  ]) {
    assert.equal(
      document.components.schemas[schemaName]?.additionalProperties,
      false,
      `${schemaName} must be a closed schema`,
    );
  }
  const files = fs.readdirSync(examplesDirectory);
  for (const example of [
    "organization-list.json",
    "planning-item.json",
    "checklist-template-version.json",
    "checklist-template-version-detail.json",
    "reminder-rules.json",
    "audit-events.json",
  ]) {
    assert.ok(files.includes(example), `Missing canonical route-family example: ${example}`);
  }
});

test("configuration template detail exposes exact immutable question contract", () => {
  const document = readRequiredJson(openApiPath);
  assert.ok(
    document.paths["/v1/configuration/checklist-template-versions/{templateVersionId}"]?.get,
    "Checklist template version detail must support direct GET",
  );

  const question = document.components.schemas.ChecklistTemplateQuestionView;
  assert.equal(question?.additionalProperties, false, "ChecklistTemplateQuestionView must be closed");
  assert.deepEqual(question?.required, [
    "id",
    "sectionId",
    "prompt",
    "regulatoryReference",
    "expectedEvidence",
    "allowedAnswers",
    "commentRequiredFor",
  ]);
  assert.doesNotMatch(
    JSON.stringify(question),
    /assignedInspectorUserIds|currentResponse|draft|secret|userAdministration/i,
  );

  const detail = document.components.schemas.ChecklistTemplateVersionDetailView;
  assert.equal(detail?.additionalProperties, false, "ChecklistTemplateVersionDetailView must be closed");
  assert.ok(detail?.required?.includes("questions"), "Checklist template detail must require questions");
  assert.equal(
    detail?.properties?.questions?.items?.$ref,
    "#/components/schemas/ChecklistTemplateQuestionView",
  );

  const example = readRequiredJson(path.join(examplesDirectory, "checklist-template-version-detail.json"));
  assert.equal(example.schema, "ChecklistTemplateVersionDetailView");
  assert.equal(example.value.id, "CTV-CABIN-1");
  assert.equal(example.value.questions.length, 6);
  assert.deepEqual(example.value.questions[0].allowedAnswers, [
    "COMPLIANT",
    "NON_COMPLIANT",
    "OBSERVATION",
    "NOT_APPLICABLE",
    "NOT_CHECKED",
  ]);
  assert.deepEqual(example.value.questions[0].commentRequiredFor, [
    "NON_COMPLIANT",
    "OBSERVATION",
  ]);
  assert.doesNotMatch(JSON.stringify(example.value), /assignedInspectorUserIds|currentResponse/i);
});

test("lifecycle read projections expose Potential Finding and role-shaped CAP revision contracts", () => {
  const document = readRequiredJson(openApiPath);
  for (const route of [
    "/v1/potential-findings",
    "/v1/potential-findings/{potentialFindingId}",
    "/v1/findings/{findingId}/cap-revisions",
    "/v1/cap-revisions/{capRevisionId}",
  ]) {
    assert.ok(document.paths[route], `Missing lifecycle read path: ${route}`);
  }

  assert.ok(
    document.paths["/v1/potential-findings"].get,
    "Potential Findings must have a read list operation",
  );
  assert.ok(
    document.paths["/v1/potential-findings/{potentialFindingId}"].get,
    "Potential Findings must have a direct get operation",
  );
  assert.ok(
    document.paths["/v1/findings/{findingId}/cap-revisions"].get,
    "CAP revisions must be listable from a Finding",
  );
  assert.ok(
    document.paths["/v1/cap-revisions/{capRevisionId}"].get,
    "CAP revisions must support direct get",
  );

  for (const schemaName of [
    "ListPotentialFindingsOutput",
    "CapRevisionSubmission",
    "CaaCapRevisionView",
    "AuditeeCapRevisionView",
    "CapRevisionView",
    "ListCapRevisionsOutput",
  ]) {
    assert.equal(
      document.components.schemas[schemaName]?.additionalProperties,
      false,
      `${schemaName} must be closed`,
    );
  }

  const capUnion = document.components.schemas.CapRevisionView;
  assert.equal(capUnion.discriminator?.propertyName, "audience");
  assert.equal(
    JSON.stringify(document.components.schemas.AuditeeCapRevisionView).includes("internalCaaNote"),
    false,
    "Auditee CAP revision schema must structurally omit Internal CAA Note",
  );

  const files = fs.readdirSync(examplesDirectory);
  for (const example of [
    "potential-findings-response.json",
    "cap-revision-caa.json",
    "cap-revision-auditee.json",
  ]) {
    assert.ok(files.includes(example), `Missing canonical lifecycle read example: ${example}`);
  }

  const caaExample = readRequiredJson(path.join(examplesDirectory, "cap-revision-caa.json"));
  const auditeeExample = readRequiredJson(path.join(examplesDirectory, "cap-revision-auditee.json"));
  assert.match(JSON.stringify(caaExample.value), /internalCaaNote/);
  assert.doesNotMatch(JSON.stringify(auditeeExample.value), /internalCaaNote/);
});
