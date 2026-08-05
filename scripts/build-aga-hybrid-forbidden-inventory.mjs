#!/usr/bin/env node

import { createHash } from "node:crypto";
import { readFileSync, readdirSync, writeFileSync } from "node:fs";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const root = resolve(dirname(fileURLToPath(import.meta.url)), "..");
const migrationDirectory = resolve(root, "apps/api/migrations");
const composePath = resolve(root, "deploy/local/compose.yaml");
const overlayStorePath = resolve(root, "apps/api/internal/preproddata/agacandidatedemo/postgres_store.go");
const workspaceProvisionPath = resolve(root, "apps/api/internal/preproddata/agademoworkspace/postgres_provision.go");
const workspaceContractPath = resolve(root, "apps/api/internal/preproddata/agademoworkspace/contract.go");

const authTables = Object.freeze({
  identity_references: ["display_name"],
  user_profiles: ["display_name"],
  desired_membership_versions: ["observed_at", "drift_state"],
  desired_membership_sync: ["observed_provider_enabled", "observed_organization_id", "observed_roles", "observed_at", "drift_state"],
  session_references: ["authority_state", "revoked_at", "last_seen_at", "expires_at", "absolute_expires_at"],
  oidc_login_states: ["state_hash", "expires_at"],
  audit_events: ["entity_type", "entity_id", "action", "occurred_at", "details"],
});

const overlayTables = Object.freeze([
  "package_intents",
  "packages",
  "forms",
  "form_source_proposals",
  "source_reference_catalog",
  "questions",
  "question_source_proposals",
  "package_seals",
]);

const overlayViews = Object.freeze(["sealed_packages", "sealed_forms", "sealed_questions"]);

function digestFile(path) {
  return `sha256:${createHash("sha256").update(readFileSync(path)).digest("hex")}`;
}

function parseObjects(sql) {
  const tables = [...sql.matchAll(/CREATE\s+TABLE\s+(?:IF\s+NOT\s+EXISTS\s+)?(?:[A-Za-z0-9_]+\.)?([A-Za-z0-9_]+)/giu)].map((match) => match[1]);
  const explicitSequences = [...sql.matchAll(/CREATE\s+SEQUENCE\s+(?:IF\s+NOT\s+EXISTS\s+)?(?:[A-Za-z0-9_]+\.)?([A-Za-z0-9_]+)/giu)].map((match) => match[1]);
  // PostgreSQL creates an owned sequence for every identity column. Those
  // sequences do not appear as CREATE SEQUENCE statements in migrations, but
  // they are still mutable business objects and must be in the zero-delta
  // inventory. The default PostgreSQL name is <table>_<column>_seq.
  const identitySequences = [];
  for (const match of sql.matchAll(/CREATE\s+TABLE\s+(?:IF\s+NOT\s+EXISTS\s+)?(?:[A-Za-z0-9_]+\.)?([A-Za-z0-9_]+)\s*\(([^;]*?)\);/gisu)) {
    const [, table, columns] = match;
    for (const column of columns.matchAll(/(?:^|,)\s*([A-Za-z0-9_]+)\s+[^,\n]*?GENERATED\s+(?:ALWAYS|BY\s+DEFAULT)\s+AS\s+IDENTITY/giu)) {
      identitySequences.push(`${table}_${column[1]}_seq`);
    }
  }
  const sequences = [...new Set([...explicitSequences, ...identitySequences])];
  return { tables, sequences };
}

function migrationObjects() {
  const entries = [];
  const names = readdirSync(migrationDirectory).filter((name) => name.endsWith(".up.sql")).sort();
  for (const name of names) entries.push(parseObjects(readFileSync(resolve(migrationDirectory, name), "utf8")));
  return {
    tables: [...new Set(entries.flatMap((entry) => entry.tables))].sort(),
    sequences: [...new Set(entries.flatMap((entry) => entry.sequences))].sort(),
  };
}

function digestMigrationSources() {
  const names = readdirSync(migrationDirectory).filter((name) => name.endsWith(".up.sql")).sort();
  const sourceHash = createHash("sha256");
  for (const name of names) {
    sourceHash.update(name);
    sourceHash.update("\u0000");
    sourceHash.update(readFileSync(resolve(migrationDirectory, name)));
    sourceHash.update("\u0000");
  }
  return `sha256:${sourceHash.digest("hex")}`;
}

function parseQualifiedObjects(sql, keyword) {
  return [...sql.matchAll(new RegExp(`${keyword}\\s+(?:IF\\s+NOT\\s+EXISTS\\s+)?(?:([A-Za-z0-9_]+)\\.)?([A-Za-z0-9_]+)`, "giu"))]
    .map((match) => match[1] ? `${match[1]}.${match[2]}` : match[2]);
}

function composeContract() {
  const compose = readFileSync(composePath, "utf8");
  const requiredServices = [
    "preprod-aga-demo-workspace-role-provisioner",
    "preprod-aga-demo-workspace-fixture-exporter",
    "preprod-aga-demo-workspace-loader",
    "preprod-aga-demo-api",
    "preprod-postgres",
  ];
  const requiredSecrets = [
    "preprod_aga_demo_workspace_fixture_exporter_database_password",
    "preprod_aga_demo_workspace_loader_database_password",
    "preprod_aga_demo_workspace_reader_database_password",
    "preprod_aga_demo_workspace_command_database_password",
    "preprod_aga_demo_session_encryption_key",
  ];
  for (const value of [...requiredServices, ...requiredSecrets]) if (!compose.includes(value)) throw new Error(`compose contract missing ${value}`);
  return { services: requiredServices, secrets: requiredSecrets };
}

function extractRoles(contract) {
  return [...contract.matchAll(/Workspace(?:Owner|Exporter|Loader|Reader|Command)Role\s*=\s*"([^"]+)"/gu)].map((match) => match[1]).sort();
}

function buildInventory() {
  const migrations = migrationObjects();
  const overlaySQL = readFileSync(overlayStorePath, "utf8");
  const workspaceSQL = readFileSync(workspaceProvisionPath, "utf8");
  const workspaceContract = readFileSync(workspaceContractPath, "utf8");
  const workspaceTables = parseQualifiedObjects(workspaceSQL, "CREATE TABLE");
  const workspaceFunctions = parseQualifiedObjects(workspaceSQL, "CREATE(?: OR REPLACE)? FUNCTION");
  const allPublicTables = migrations.tables;
  const authTableNames = Object.keys(authTables).sort();
  const forbiddenTables = allPublicTables.filter((name) => !authTableNames.includes(name)).sort();
  const authObjects = authTableNames.map((table) => `public.${table}`).sort();
  const forbiddenObjects = forbiddenTables.map((table) => `public.${table}`).sort();
  const forbiddenSequences = migrations.sequences.map((sequence) => `public.${sequence}`).sort();
  const overlayObjects = [
    ...overlayTables.map((table) => `preprod_aga_demo.${table}`),
    ...overlayViews.map((view) => `preprod_aga_demo.${view}`),
  ].sort();
  const workspaceObjects = workspaceTables.filter((object) => object.startsWith("preprod_aga_demo_workspace.")).sort();
  const workspaceGrants = [
    "GRANT USAGE ON SCHEMA preprod_aga_demo_workspace TO preprod_aga_demo_workspace_reader, preprod_aga_demo_workspace_command, preprod_aga_demo_workspace_loader",
    "GRANT SELECT ON sealed_generations, sealed_classification_items, sealed_drafts, sealed_authority_bindings, sealed_provider_scopes, sealed_recommendations, sealed_lifecycle_projection TO preprod_aga_demo_workspace_reader",
    "GRANT EXECUTE ON workspace_query(jsonb), workspace_command(jsonb), workspace_reset(jsonb) TO preprod_aga_demo_workspace_command",
    "REVOKE ALL ON ALL TABLES IN SCHEMA preprod_aga_demo_workspace FROM PUBLIC, preprod_aga_demo_workspace_fixture_exporter, preprod_aga_demo_workspace_loader, preprod_aga_demo_workspace_reader",
    "REVOKE ALL ON ALL FUNCTIONS IN SCHEMA preprod_aga_demo_workspace FROM PUBLIC, preprod_aga_demo_workspace_fixture_exporter, preprod_aga_demo_workspace_loader",
  ];
  return {
    schemaVersion: "aga-hybrid-forbidden-object-inventory/v1",
    planPath: "docs/exec-plans/active/2026-08-03-aga-hybrid-classification-demo-lifecycle-plan.md",
    sourceDigests: {
      migrations: digestMigrationSources(),
      overlayContract: digestFile(overlayStorePath),
      workspaceContract: digestFile(workspaceContractPath),
      workspaceProvision: digestFile(workspaceProvisionPath),
      compose: digestFile(composePath),
    },
    classes: {
      FORBIDDEN_BUSINESS: {
        objects: [...forbiddenObjects, ...forbiddenSequences],
        reason: "Canonical business tables, sequences, and non-workspace records remain zero-delta after predecessor setup.",
      },
      AUTH_CONTROL_PLANE: {
        objects: authObjects,
        columns: Object.fromEntries(authTableNames.map((table) => [`public.${table}`, authTables[table]])),
        allowedShapes: ["AUTHENTICATION_OBSERVATION", "SESSION_REFRESH", "LOGOUT", "OIDC_LOGIN_STATE"],
      },
      SEALED_OVERLAY_ALLOWED: {
        schema: "preprod_aga_demo",
        objects: overlayObjects,
        roles: ["preprod_aga_demo_owner", "preprod_aga_demo_writer", "preprod_aga_demo_reader"],
        postSealMutation: "forbidden",
      },
      WORKSPACE_ALLOWED: {
        schema: "preprod_aga_demo_workspace",
        objects: workspaceObjects,
        functions: workspaceFunctions.sort(),
        roles: extractRoles(workspaceContract),
        grants: workspaceGrants,
        postSealMutation: "append-only-command-store-only",
      },
    },
    compose: composeContract(),
    coverage: {
      migrationTableCount: allPublicTables.length,
      migrationSequenceCount: migrations.sequences.length,
      authControlTableCount: authObjects.length,
      forbiddenBusinessObjectCount: forbiddenObjects.length + forbiddenSequences.length,
      overlayObjectCount: overlayObjects.length,
      workspaceObjectCount: workspaceObjects.length,
    },
  };
}

function exact(value, expected) {
  return JSON.stringify(value) === JSON.stringify(expected);
}

export function validateInventory(inventory, expected = buildInventory()) {
  if (!inventory || inventory.schemaVersion !== expected.schemaVersion || inventory.planPath !== expected.planPath) throw new Error("ERR_AGA_HYBRID_INVENTORY_HEADER");
  if (!exact(inventory.sourceDigests, expected.sourceDigests)) throw new Error("ERR_AGA_HYBRID_INVENTORY_SOURCES");
  if (!exact(inventory.classes, expected.classes)) throw new Error("ERR_AGA_HYBRID_INVENTORY_CLASSES");
  if (!exact(inventory.compose, expected.compose)) throw new Error("ERR_AGA_HYBRID_INVENTORY_COMPOSE");
  if (!exact(inventory.coverage, expected.coverage)) throw new Error("ERR_AGA_HYBRID_INVENTORY_COVERAGE");
  const allObjects = Object.values(inventory.classes).flatMap((entry) => entry.objects ?? []);
  if (new Set(allObjects).size !== allObjects.length) throw new Error("ERR_AGA_HYBRID_INVENTORY_OVERLAP");
  return true;
}

function parseArgs(argv) {
  const values = new Map();
  for (let index = 0; index < argv.length; index += 2) {
    const key = argv[index];
    const value = argv[index + 1];
    if (!key?.startsWith("--") || value === undefined || values.has(key)) throw new Error("ERR_AGA_HYBRID_INVENTORY_ARGUMENTS");
    values.set(key.slice(2), value);
  }
  return values;
}

if (process.argv[1] && fileURLToPath(import.meta.url) === resolve(process.argv[1])) {
  const values = parseArgs(process.argv.slice(2));
  try {
    const expected = buildInventory();
    const outputPath = resolve(root, values.get("output") ?? "tests/fixtures/aga-hybrid-forbidden-object-inventory.v1.json");
    if (values.has("check")) {
      const actual = JSON.parse(readFileSync(resolve(root, values.get("check")), "utf8"));
      validateInventory(actual, expected);
      process.stdout.write(`aga-hybrid-forbidden-inventory: ok objects=${actual.coverage.migrationTableCount + actual.coverage.migrationSequenceCount}\n`);
    } else if (values.has("write")) {
      writeFileSync(outputPath, `${JSON.stringify(expected, null, 2)}\n`, { encoding: "utf8", mode: 0o644 });
      process.stdout.write(`aga-hybrid-forbidden-inventory: wrote ${outputPath}\n`);
    } else {
      process.stdout.write(`${JSON.stringify(expected, null, 2)}\n`);
    }
  } catch (error) {
    process.stderr.write(`${error instanceof Error ? error.message : "ERR_AGA_HYBRID_INVENTORY_UNEXPECTED"}\n`);
    process.exitCode = 1;
  }
}
