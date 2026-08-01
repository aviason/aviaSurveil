import crypto from "node:crypto";
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

export const GENERATOR_VERSION = "1.0.0";

const strictGovernedUnionNames = new Set([
  "GovernedRegulatoryTraceContent",
  "GovernedCandidateLineage",
  "GovernedCurrentDecision",
  "GovernedSourceReviewDetailView",
  "ChecklistImportExtractionDecisionSetView",
  "ChecklistImportIdentityResolutionState",
  "ExtractionDecisionInput",
  "CandidateLineageAction",
  "ExtractionDecisionAction",
]);

const scriptDirectory = path.dirname(fileURLToPath(import.meta.url));
const repositoryRoot = path.resolve(scriptDirectory, "..");
const outputRoot = process.env.AVIA_CONTRACT_OUTPUT_ROOT
  ? path.resolve(process.env.AVIA_CONTRACT_OUTPUT_ROOT)
  : repositoryRoot;
const specificationPath = process.env.AVIA_CONTRACT_SPECIFICATION
  ? path.resolve(process.env.AVIA_CONTRACT_SPECIFICATION)
  : path.join(repositoryRoot, "api/openapi/aviasurveil360.yaml");
const outputPath = path.join(outputRoot, "apps/api/internal/httpapi/generated/api.gen.go");
const specificationBytes = fs.readFileSync(specificationPath);
const specification = JSON.parse(specificationBytes);

function goName(value) {
  const chunks = String(value).split(/[^A-Za-z0-9]+/).filter(Boolean);
  const joined = chunks
    .map((chunk) => chunk.charAt(0).toUpperCase() + chunk.slice(1))
    .join("");
  const candidate = joined || "Value";
  return /^[0-9]/.test(candidate) ? `Value${candidate}` : candidate;
}

function resolveReference(reference) {
  const prefix = "#/components/";
  if (!reference?.startsWith(prefix)) {
    throw new Error(`Unsupported OpenAPI reference: ${reference}`);
  }
  return reference.slice(prefix.length).split("/").reduce((value, segment) => value[segment], specification.components);
}

function referenceName(reference) {
  return goName(reference.split("/").at(-1));
}

function schemaType(schema) {
  if (!schema) return "json.RawMessage";
  if (schema.$ref) return referenceName(schema.$ref);
  if (schema.oneOf) {
    const nonNull = schema.oneOf.filter((member) => member.type !== "null");
    if (nonNull.length === 1 && nonNull.length !== schema.oneOf.length) {
      const memberType = schemaType(nonNull[0]);
      return memberType.startsWith("*") ? memberType : `*${memberType}`;
    }
    if (schema.discriminator) {
      throw new Error("Discriminated unions must be named schemas: inline union encountered");
    }
    return "json.RawMessage";
  }

  const declaredTypes = Array.isArray(schema.type) ? schema.type : [schema.type];
  const nullable = declaredTypes.includes("null");
  const type = declaredTypes.find((candidate) => candidate !== "null");
  let result;
  switch (type) {
    case "string":
      result = "string";
      break;
    case "integer":
      result = "int64";
      break;
    case "number":
      result = "float64";
      break;
    case "boolean":
      result = "bool";
      break;
    case "array":
      result = `[]${schemaType(schema.items)}`;
      break;
    case "object":
      result = "map[string]any";
      break;
    case "null":
    case undefined:
      result = "json.RawMessage";
      break;
    default:
      throw new Error(`Unsupported schema type: ${type}`);
  }
  return nullable && !result.startsWith("*") ? `*${result}` : result;
}

function optionalType(type) {
  if (
    type.startsWith("*") ||
    type.startsWith("[]") ||
    type.startsWith("map[") ||
    type === "any" ||
    type === "json.RawMessage"
  ) {
    return type;
  }
  return `*${type}`;
}

function generateSchema(name, schema) {
  const typeName = goName(name);
  if (schema.enum) {
    const baseType = schemaType({ type: Array.isArray(schema.type) ? schema.type[0] : schema.type });
    const constants = schema.enum
      .map((value) => `\t${typeName}${goName(value)} ${typeName} = ${JSON.stringify(value)}`)
      .join("\n");
    return `type ${typeName} ${baseType}\n\nconst (\n${constants}\n)`;
  }
  if (schema.oneOf && schema.discriminator && strictGovernedUnionNames.has(name)) {
    const discriminator = schema.discriminator.propertyName;
    const variants = schema.oneOf.map((member) => {
      if (!member.$ref) throw new Error(`${name} discriminator members must be references`);
      return referenceName(member.$ref);
    });
    const fields = variants.map((variant) => `\t${variant} *${variant} \`json:"-"\``).join("\n");
    const unmarshalCases = variants.map((variant) => {
      const member = schema.oneOf.find((candidate) => referenceName(candidate.$ref) === variant);
      const memberSchema = resolveReference(member.$ref);
      const discriminatorSchema = memberSchema?.properties?.[discriminator];
      const literals = discriminatorSchema?.const === undefined
        ? discriminatorSchema?.enum
        : [discriminatorSchema.const];
      if (!Array.isArray(literals) || literals.length === 0) {
        throw new Error(`${name}.${variant} must declare ${discriminator} const or enum`);
      }
      return `\tcase ${literals.map((literal) => JSON.stringify(literal)).join(", ")}:\n\t\tvar value ${variant}\n\t\tif err := json.Unmarshal(data, &value); err != nil { return err }\n\t\t*receiver = ${typeName}{${variant}: &value}`;
    }).join("\n");
    const marshalCases = variants.map((variant) => `\tif value.${variant} != nil { return json.Marshal(value.${variant}) }`).join("\n");
    return `type ${typeName} struct {\n${fields}\n}\n\nfunc (receiver *${typeName}) UnmarshalJSON(data []byte) error {\n\tvar discriminatorValue struct {\n\t\tValue string \`json:"${discriminator}"\`\n\t}\n\tif err := json.Unmarshal(data, &discriminatorValue); err != nil { return err }\n\tswitch discriminatorValue.Value {\n${unmarshalCases}\n\tdefault:\n\t\treturn fmt.Errorf("${typeName}: unsupported ${discriminator} %q", discriminatorValue.Value)\n\t}\n\treturn nil\n}\n\nfunc (value ${typeName}) MarshalJSON() ([]byte, error) {\n${marshalCases}\n\treturn nil, fmt.Errorf("${typeName}: empty union")\n}`;
  }
  if (schema.oneOf) {
    return `type ${typeName} = json.RawMessage`;
  }
  const declaredTypes = Array.isArray(schema.type) ? schema.type : [schema.type];
  if (declaredTypes.includes("object") && schema.properties) {
    const required = new Set(schema.required ?? []);
    const fields = Object.entries(schema.properties).map(([propertyName, propertySchema]) => {
      let type = schemaType(propertySchema);
      if (!required.has(propertyName)) type = optionalType(type);
      const omitEmpty = required.has(propertyName) ? "" : ",omitempty";
      return `\t${goName(propertyName)} ${type} \`json:"${propertyName}${omitEmpty}"\``;
    });
    return `type ${typeName} struct {\n${fields.join("\n")}\n}`;
  }
  return `type ${typeName} = ${schemaType(schema)}`;
}

function operationParameters(pathItem, operation) {
  return [...(pathItem.parameters ?? []), ...(operation.parameters ?? [])].map((parameter) =>
    parameter.$ref ? resolveReference(parameter.$ref) : parameter,
  );
}

function successBodyType(operation) {
  const success = Object.entries(operation.responses ?? {})
    .filter(([status]) => /^2\d\d$/.test(status))
    .sort(([left], [right]) => left.localeCompare(right))[0]?.[1];
  const response = success?.$ref ? resolveReference(success.$ref) : success;
  const schema = response?.content?.["application/json"]?.schema;
  return schemaType(schema);
}

const operations = [];
for (const [routePath, pathItem] of Object.entries(specification.paths)) {
  for (const [method, operation] of Object.entries(pathItem)) {
    if (!operation?.operationId) continue;
    operations.push({ routePath, method: method.toUpperCase(), operation, pathItem });
  }
}

const schemaBlocks = Object.entries(specification.components?.schemas ?? {}).map(([name, schema]) =>
  generateSchema(name, schema),
);
const operationBlocks = operations.map(({ operation, pathItem }) => {
  const name = goName(operation.operationId);
  const requestName = `Operation${name}Request`;
  const responseName = `Operation${name}Response`;
  const parameters = operationParameters(pathItem, operation).map((parameter) => {
    let type = schemaType(parameter.schema);
    if (!parameter.required) type = optionalType(type);
    return `\t${goName(parameter.name)} ${type} \`${parameter.in}:"${parameter.name}"\``;
  });
  const bodySchema = operation.requestBody?.content?.["application/json"]?.schema;
  const multipartSchema = operation.requestBody?.content?.["multipart/form-data"]?.schema;
  const bodyField = bodySchema
    ? [`\tBody ${schemaType(bodySchema)} \`json:"body"\``]
    : multipartSchema
      ? [
          `\tArchive io.Reader \`multipart:"archive"\``,
          `\tReceipt ${schemaType(multipartSchema.properties?.receipt)} \`json:"receipt"\``,
        ]
      : [];
  return `// operationId: ${operation.operationId}\n` +
    `type ${requestName} struct {\n\tHeaders http.Header \`json:"-"\`\n${[...parameters, ...bodyField].join("\n")}\n}\n\n` +
    `type ${responseName} struct {\n\tStatusCode int\n\tHeaders http.Header\n\tBody ${successBodyType(operation)}\n}`;
});
const interfaceMethods = operations
  .map(({ operation }) => {
    const name = goName(operation.operationId);
    return `\t${name}(context.Context, Operation${name}Request) (Operation${name}Response, error)`;
  })
  .join("\n");
const routes = operations
  .map(
    ({ method, routePath, operation }) =>
      `\t{Method: ${JSON.stringify(method)}, Path: ${JSON.stringify(routePath)}, OperationID: ${JSON.stringify(operation.operationId)}},`,
  )
  .join("\n");
const specificationHash = crypto.createHash("sha256").update(specificationBytes).digest("hex");

const output = `// Code generated by scripts/generate-go-contracts.mjs v${GENERATOR_VERSION}; DO NOT EDIT.
// OpenAPI-SHA256: ${specificationHash}

package generated

import (
\t"context"
\t"encoding/json"
\t"fmt"
\t"io"
\t"net/http"
)

${schemaBlocks.join("\n\n")}

${operationBlocks.join("\n\n")}

type StrictServerInterface interface {
${interfaceMethods}
}

type OperationRoute struct {
\tMethod string
\tPath string
\tOperationID string
}

var OperationRoutes = []OperationRoute{
${routes}
}
`;

fs.mkdirSync(path.dirname(outputPath), { recursive: true });
fs.writeFileSync(outputPath, output);
console.log(`generated ${path.relative(repositoryRoot, outputPath)}`);
