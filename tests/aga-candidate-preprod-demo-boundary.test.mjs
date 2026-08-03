import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { test } from "node:test";

const read = (file) => readFileSync(file, "utf8");
const routePrefix = "/v1/admin/governed-checklist/aga-candidate-demo";

test("AGA candidate-demo transport is Admin-only, read-only, and label-bounded", () => {
  const paths = JSON.parse(read("api/openapi/source/paths/platform.json"));
  const candidatePaths = Object.entries(paths).filter(([path]) => path.startsWith(routePrefix));
  assert.equal(candidatePaths.length, 5);
  for (const [path, operations] of candidatePaths) {
    assert.deepEqual(Object.keys(operations), ["get"], `${path} must not expose a command or export`);
  }
  const routeSource = read("apps/api/internal/httpapi/aga_candidate_demo_api.go");
  assert.match(routeSource, /actor, ok := agaDemoActor\(request\)/u);
  assert.match(routeSource, /StatusNotFound/u);
  assert.match(routeSource, /\{"error":"not found"\}/u);
  assert.match(routeSource, /Cache-Control", "private, no-store"/u);
  assert.match(routeSource, /Vary", "Cookie"/u);
  assert.match(routeSource, /Content-Length", strconv\.Itoa\(len\(body\)\)/u);
  for (const prohibited of ["POST", "PUT", "DELETE", "PATCH", "export", "publish", "attest", "assign", "approve", "Finding", "Audit"]) {
    assert.doesNotMatch(routeSource, new RegExp(`(?:Method|Handle)${prohibited}`, "u"));
  }
});

test("candidate reader and UI cannot reach loaders, real-domain commands, storage, or telemetry", () => {
  const reader = read("apps/api/internal/agacandidatedemo/postgres_reader_preproddemo.go");
  const service = read("apps/api/internal/agacandidatedemo/service.go");
  const projection = read("apps/api/internal/preproddata/agacandidatedemo/store.go");
  const ui = read("apps/web/src/features/admin/aga-candidate-demo-panel.tsx");
  const adapter = read("apps/web/src/backend/http-backend.ts");
  const loader = read("apps/api/cmd/preprod-aga-candidate-demo-loader/main.go");

  assert.match(reader, /sealed_packages/u);
  assert.match(reader, /sealed_forms/u);
  assert.match(reader, /sealed_questions/u);
  assert.match(reader, /sourceResolutionRequirements/u);
  assert.match(projection, /packageProjectionRow/u);
  assert.match(projection, /SourceResolutionRequirements/u);
  assert.doesNotMatch(reader, /INSERT|UPDATE|DELETE|CREATE|DROP/iu);
  assert.doesNotMatch(service, /checklistintake|preproddata/u);
  assert.match(service, /OrganizationID == "CAA"/u);
  assert.match(service, /RoleAdmin/u);
  for (const prohibited of ["localStorage", "sessionStorage", "indexedDB", "caches", "serviceWorker", "telemetry", "fetch(", "createFinding", "createAudit", "capability.publish", "capability.approve", "capability.attest", "capability.assign"]) {
    assert.doesNotMatch(ui, new RegExp(prohibited.replace(/[()]/g, "\\$&"), "iu"));
  }
  assert.match(adapter, /cache: "no-store", suppressTelemetry: true/u);
  for (const prohibited of ["internal/identity", "internal/objectstore", "internal/datafeed", "internal/checklistintake", "provider", "mailpit", "minio"]) {
    assert.doesNotMatch(loader, new RegExp(prohibited, "iu"));
  }
});

test("normal artifacts and the tagged API keep their distinct least-privilege dependencies", () => {
  const normal = read("apps/api/cmd/api/profile_normal.go");
  const tagged = read("apps/api/cmd/api/profile_preproddemo.go");
  const compose = read("deploy/local/compose.yaml");
  assert.doesNotMatch(normal, /agacandidatedemo|AVIA_AGA_DEMO_DATABASE_URL/u);
  assert.match(tagged, /AGADemoDatabaseURL/u);
  assert.match(tagged, /NewPostgresReader/u);
  assert.match(tagged, /skipMigrations: true/u);
  assert.match(tagged, /agaDemoOnly: true/u);
  assert.match(compose, /AVIA_DATABASE_USER:\s*preprod_normal_api/u);
  assert.match(compose, /preprod_aga_demo_reader_database_password/u);
  assert.match(compose, /preprod_aga_demo_writer_database_password/u);
});

test("connected qualification rebuilds every tagged executable before use", () => {
  const harness = read("scripts/test-aga-candidate-preprod-demo-connected.sh");
  const build = harness.indexOf(
    "compose_command build preprod-aga-candidate-demo-loader preprod-aga-demo-role-provisioner preprod-aga-demo-oidc-fixture",
  );
  const provision = harness.indexOf(
    "compose_command run --rm preprod-aga-demo-role-provisioner",
  );
  assert.notEqual(build, -1, "tagged executables must be rebuilt from the current tree");
  assert.ok(build < provision, "tagged rebuild must precede role provisioning");
  assert.match(
    harness,
    /docker run --rm --network none --entrypoint sha256sum[\s\S]*aviasurveil360\/preprod-aga-candidate-demo-loader:local[\s\S]*\/app\/preprod-aga-candidate-demo-loader/u,
  );
  for (const contractFile of [
    "apps/api/internal/preproddata/agacandidatedemo/contract.go",
    "api/openapi/source/paths/platform.json",
    "api/openapi/source/schemas/platform.json",
  ]) {
    assert.match(harness, new RegExp(contractFile.replaceAll("/", "\\/"), "u"));
  }
});
