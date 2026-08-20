import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const repositoryRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const defaultSourceDirectory = path.join(repositoryRoot, "api/openapi/source");
const defaultOutputPath = path.join(repositoryRoot, "api/openapi/aviasurveil360.yaml");
const mutationMethods = new Set(["post", "put", "patch", "delete"]);
const problemStatuses = ["400", "401", "403", "409", "412", "422"];
const allRoles = [
  "inspector",
  "leadInspector",
  "manager",
  "finance",
  "gm",
  "executiveDirector",
  "auditee",
  "admin",
];

function readJson(filePath) {
  return JSON.parse(fs.readFileSync(filePath, "utf8"));
}

function mergeUnique(label, ...objects) {
  const merged = {};
  for (const object of objects) {
    for (const [key, value] of Object.entries(object ?? {})) {
      if (Object.hasOwn(merged, key)) {
        throw new Error(`Duplicate ${label} key in OpenAPI source fragments: ${key}`);
      }
      merged[key] = value;
    }
  }
  return merged;
}

function authorizedRoles(route) {
  if (route.startsWith("/v1/auditee/")) return ["auditee"];
  if (route.startsWith("/v1/admin/")) return ["admin"];
  if (
    route === "/v1/profile" ||
    route.startsWith("/v1/notifications") ||
    route.startsWith("/v1/administration/")
  ) {
    return allRoles;
  }
  if (route === "/v1/communications") {
    return ["inspector", "leadInspector", "manager", "auditee"];
  }
  if (route.startsWith("/v1/calendar-items")) {
    return ["inspector", "leadInspector", "manager", "auditee"];
  }
  if (route.startsWith("/v1/documents")) {
    return ["inspector", "leadInspector", "manager", "auditee", "admin"];
  }
  if (route.startsWith("/v1/risk/")) return ["manager"];
  if (route.startsWith("/v1/assistant/")) return ["inspector", "leadInspector"];
  if (
    route.startsWith("/v1/team-members") ||
    route.startsWith("/v1/audit-teams")
  ) {
    return ["manager", "admin"];
  }
  if (route.startsWith("/v1/planning/intake-drafts")) {
    return ["manager"];
  }
  if (route.includes("/audit-package-setup") || route.includes("/audit-package-finalizations")) {
    return ["manager"];
  }
  return allRoles.filter((role) => role !== "auditee");
}

function addParameter(operation, reference) {
  operation.parameters ??= [];
  if (!operation.parameters.some((parameter) => parameter.$ref === reference)) {
    operation.parameters.push({ $ref: reference });
  }
}

function applyOperationContract(document) {
  for (const [route, pathItem] of Object.entries(document.paths)) {
    for (const [method, operation] of Object.entries(pathItem)) {
      if (route.startsWith("/health/")) continue;

      const operationKind = operation["x-operation-kind"];
      const neutralDenial = operation["x-neutral-denial"] === true;

      operation.security ??= [{ oidc: [] }];
      operation["x-authorized-roles"] ??= authorizedRoles(route);
      operation["x-organization-scope"] ??=
        route.startsWith("/v1/auditee/") ? "principal-organization-only" : "authorized-projection";

      for (const [status, response] of Object.entries(operation.responses ?? {})) {
        if (!/^2\d\d$/.test(status) || response.$ref) continue;
        response.headers ??= {};
        response.headers.ETag ??= { $ref: "#/components/headers/ETag" };
      }

      if (!mutationMethods.has(method)) {
        if (neutralDenial) {
          delete operation.responses["401"];
          delete operation.responses["403"];
          operation.responses["404"] ??= { $ref: "#/components/responses/Problem" };
        } else {
          operation.responses["401"] ??= { $ref: "#/components/responses/Problem" };
          operation.responses["403"] ??= { $ref: "#/components/responses/Problem" };
        }
        continue;
      }

      // Query-shaped POSTs still carry CSRF, but they are not commands: no
      // idempotency or If-Match header is advertised for them.
      if (operationKind === "query") {
        addParameter(operation, "#/components/parameters/CsrfToken");
        if (neutralDenial) {
          delete operation.responses["401"];
          delete operation.responses["403"];
          operation.responses["404"] ??= { $ref: "#/components/responses/Problem" };
        }
        continue;
      }

      addParameter(operation, "#/components/parameters/IdempotencyKey");
      addParameter(operation, "#/components/parameters/CsrfToken");
      addParameter(operation, "#/components/parameters/ExpectedRevision");
      for (const status of problemStatuses) {
        if (neutralDenial && (status === "401" || status === "403")) continue;
        operation.responses[status] ??= { $ref: "#/components/responses/Problem" };
      }
      if (neutralDenial) {
        delete operation.responses["401"];
        delete operation.responses["403"];
        operation.responses["404"] ??= { $ref: "#/components/responses/Problem" };
      }
    }
  }
}

export function assembleOpenApi(sourceDirectory = defaultSourceDirectory) {
  const source = readJson(path.join(sourceDirectory, "openapi.json"));
  const paths = mergeUnique(
    "path",
    readJson(path.join(sourceDirectory, "paths/core.json")),
    source.paths,
    readJson(path.join(sourceDirectory, "paths/workflows.json")),
    readJson(path.join(sourceDirectory, "paths/platform.json")),
  );
  const schemas = mergeUnique(
    "schema",
    source.components.schemas,
    readJson(path.join(sourceDirectory, "schemas/domain.json")),
    readJson(path.join(sourceDirectory, "schemas/platform.json")),
    readJson(path.join(sourceDirectory, "schemas/prior-audit-recommendations.json")),
  );
  const catalogEntry = schemas.CanonicalQuestionCatalogEntry;
  schemas.CanonicalQuestionCatalogEntry = {
    ...catalogEntry,
    required: [...new Set([...(catalogEntry.required ?? []), "recommendation"])],
    properties: {
      ...catalogEntry.properties,
      recommendation: { $ref: "#/components/schemas/CanonicalQuestionRecommendation" },
    },
  };
  const document = {
    ...source,
    paths,
    components: {
      ...source.components,
      parameters: {
        ...source.components.parameters,
        Cursor: {
          name: "cursor",
          in: "query",
          schema: { type: "string" },
        },
        Limit: {
          name: "limit",
          in: "query",
          schema: { type: "integer", minimum: 1, maximum: 100, default: 50 },
        },
        IdempotencyKey: {
          name: "Idempotency-Key",
          in: "header",
          required: true,
          schema: { type: "string", minLength: 1 },
        },
        CsrfToken: {
          name: "X-CSRF-Token",
          in: "header",
          required: true,
          schema: { type: "string", minLength: 1 },
        },
        ExpectedRevision: {
          name: "If-Match",
          in: "header",
          required: true,
          description: "Expected entity revision encoded as a strong ETag.",
          schema: { type: "string", pattern: "^\\\"rev-[0-9]+\\\"$" },
        },
      },
      headers: {
        ...source.components.headers,
        ETag: {
          description: "Strong entity or projection revision.",
          required: true,
          schema: { type: "string", pattern: "^\\\"rev-[0-9]+\\\"$" },
        },
      },
      securitySchemes: {
        ...source.components.securitySchemes,
        oidc: {
          type: "openIdConnect",
          openIdConnectUrl: "/.well-known/openid-configuration",
          description:
            "OIDC principal. Authorization evaluates role and organization_id claims server-side.",
        },
      },
      schemas,
    },
  };
  const catalogListOperation = document.paths["/v1/question-catalogs/{catalogVersion}/questions"]?.get;
  if (catalogListOperation && !catalogListOperation.parameters.some((parameter) => parameter.name === "includedByDefault")) {
    catalogListOperation.parameters.push({
      name: "includedByDefault",
      in: "query",
      description: "When true, return the server-evaluated default recommendation set. This is independent of recommendationState.",
      schema: { type: "boolean" },
    });
  }
  applyOperationContract(document);
  return document;
}

export function serializeOpenApi(document) {
  return `${JSON.stringify(document, null, 2)}\n`;
}

if (process.argv[1] === fileURLToPath(import.meta.url)) {
  const outputPath = process.argv[2] ? path.resolve(process.argv[2]) : defaultOutputPath;
  fs.writeFileSync(outputPath, serializeOpenApi(assembleOpenApi()));
  console.log(`openapi-bundle: wrote ${path.relative(repositoryRoot, outputPath)}`);
}
