#!/usr/bin/env node

import { readFileSync } from "node:fs";
import { resolve } from "node:path";
import { fileURLToPath } from "node:url";

const root = resolve(new URL("..", import.meta.url).pathname);
const inventoryPath = resolve(root, "tests/fixtures/aga-hybrid-forbidden-object-inventory.v1.json");
const inventory = JSON.parse(readFileSync(inventoryPath, "utf8"));

function fail(message) {
  throw new Error(`ERR_AGA_HYBRID_SNAPSHOT_QUERY ${message}`);
}

function quoteIdentifier(value) {
  if (!/^[A-Za-z0-9_]+$/u.test(value)) fail("IDENTIFIER");
  return `"${value}"`;
}

function quoteLiteral(value) {
  if (typeof value !== "string" || /[\u0000]/u.test(value)) fail("LITERAL");
  return `'${value.replaceAll("'", "''")}'`;
}

function splitObject(value) {
  const [schema, name] = String(value).split(".");
  if (!schema || !name) fail("OBJECT");
  return { schema, name };
}

function relationPair({ schema, name }) {
  return `(${quoteLiteral(schema)}, ${quoteLiteral(name)})`;
}

function rowAggregate({ schema, name }, columns = null) {
  const relation = `${quoteIdentifier(schema)}.${quoteIdentifier(name)}`;
  const expression = columns
    ? `jsonb_build_object(${columns.map((column) => `${quoteLiteral(column)}, r.${quoteIdentifier(column)}`).join(", ")})`
    : "to_jsonb(r)";
  return `(SELECT COALESCE(jsonb_agg(${expression} ORDER BY ${expression}::text), '[]'::jsonb) FROM ${relation} AS r)`;
}

function grants(objects) {
  const pairs = objects.map(relationPair);
  const values = pairs.length ? `VALUES ${pairs.join(", ")}` : "VALUES ('', '')";
  return `(SELECT COALESCE(jsonb_agg(to_jsonb(g) ORDER BY to_jsonb(g)::text), '[]'::jsonb)
    FROM (
      SELECT n.nspname AS schema, NULL::text AS name, 'schema'::text AS object_kind,
             CASE WHEN x.grantee = 0 THEN 'PUBLIC' ELSE pg_get_userbyid(x.grantee) END AS grantee,
             pg_get_userbyid(x.grantor) AS grantor, x.privilege_type, x.is_grantable
      FROM pg_namespace AS n
      CROSS JOIN LATERAL aclexplode(COALESCE(n.nspacl, acldefault('n', n.nspowner))) AS x
      WHERE n.nspname IN (${[...new Set(objects.map(({ schema }) => quoteLiteral(schema)))].join(", ") || "''"})
      UNION ALL
      SELECT n.nspname, c.relname, c.relkind::text,
             CASE WHEN x.grantee = 0 THEN 'PUBLIC' ELSE pg_get_userbyid(x.grantee) END,
             pg_get_userbyid(x.grantor), x.privilege_type, x.is_grantable
      FROM pg_class AS c
      JOIN pg_namespace AS n ON n.oid = c.relnamespace
      CROSS JOIN LATERAL aclexplode(COALESCE(c.relacl, acldefault(c.relkind, c.relowner))) AS x
      WHERE (n.nspname, c.relname) IN (${values})
    ) AS g)`;
}

function sequenceSnapshot(sequences) {
  const pairs = sequences.map((value) => relationPair(splitObject(value)));
  if (!pairs.length) return "'[]'::jsonb";
  return `(SELECT COALESCE(jsonb_agg(to_jsonb(s) ORDER BY to_jsonb(s)::text), '[]'::jsonb)
    FROM (
      SELECT schemaname AS schema, sequencename AS name, last_value AS last_value,
             start_value, min_value, max_value, increment_by, cycle, cache_size
      FROM pg_sequences
      WHERE (schemaname, sequencename) IN (${pairs.join(", ")})
    ) AS s)`;
}

function objectMap(objects, columns = null) {
  if (!objects.length) return "'{}'::jsonb";
  const entries = objects.map((object) => {
    const parsed = splitObject(object);
    return `(${quoteLiteral(object)}, ${rowAggregate(parsed, columns)})`;
  }).join(", ");
  return `(SELECT COALESCE(jsonb_object_agg(entry.object_name, entry.rows ORDER BY entry.object_name), '{}'::jsonb) FROM (VALUES ${entries}) AS entry(object_name, rows))`;
}

function parseArgs(argv) {
  if (argv.length !== 2 || argv[0] !== "--kind") fail("ARGUMENTS");
  if (!["forbidden", "overlay", "auth"].includes(argv[1])) fail("KIND");
  return argv[1];
}

try {
  const kind = parseArgs(process.argv.slice(2));
  const classes = inventory.classes;
  const forbiddenTables = (classes.FORBIDDEN_BUSINESS.objects ?? []).filter((object) => object.startsWith("public.") && !/(?:_seq|sequence)$/u.test(object));
  const forbiddenSequences = (classes.FORBIDDEN_BUSINESS.objects ?? []).filter((object) => object.startsWith("public.") && /(?:_seq|sequence)$/u.test(object));
  const overlayObjects = classes.SEALED_OVERLAY_ALLOWED.objects ?? [];
  const authObjects = Object.keys(classes.AUTH_CONTROL_PLANE.columns ?? {});
  const authColumns = classes.AUTH_CONTROL_PLANE.columns ?? {};
  let tables;
  let sequences;
  let schemaVersion;
  if (kind === "forbidden") {
    tables = forbiddenTables;
    sequences = forbiddenSequences;
    schemaVersion = "aga-hybrid-forbidden-snapshot/v2";
  } else if (kind === "overlay") {
    tables = overlayObjects;
    sequences = [];
    schemaVersion = "aga-hybrid-overlay-snapshot/v2";
  } else {
    tables = authObjects;
    sequences = [];
    schemaVersion = "aga-hybrid-auth-control-snapshot/v1";
  }
  const tableColumns = kind === "auth" ? undefined : null;
  const tableExpression = kind === "auth"
    ? objectMap(tables, null)
    : objectMap(tables, tableColumns);
  const authExactColumns = kind === "auth"
    ? `jsonb_build_object(${tables.map((object) => `${quoteLiteral(object)}, ${rowAggregate(splitObject(object), authColumns[object])}`).join(", ")})`
    : "'{}'::jsonb";
  const sql = `SELECT jsonb_build_object(
    'schemaVersion', ${quoteLiteral(schemaVersion)},
    'tables', ${tableExpression},
    'authControlColumns', ${authExactColumns},
    'sequences', ${sequenceSnapshot(sequences)},
    'grants', ${grants([...tables, ...sequences].map(splitObject))},
    'sealRows', ${kind === "overlay" ? rowAggregate({ schema: "preprod_aga_demo", name: "package_seals" }) : "'[]'::jsonb"}
  )::text;`;
  process.stdout.write(`${sql}\n`);
} catch (error) {
  process.stderr.write(`${error instanceof Error ? error.message : "ERR_AGA_HYBRID_SNAPSHOT_QUERY UNEXPECTED"}\n`);
  process.exitCode = 1;
}
