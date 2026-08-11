#!/usr/bin/env node

import { createHash } from "node:crypto";
import { existsSync, readFileSync, writeFileSync } from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const mode = process.argv[2] ?? "target";
if (!new Set(["legacy", "target"]).has(mode)) {
  throw new Error("usage: build-aws-private-pilot-runtime-inventory.mjs legacy|target");
}

const sha256 = (bytes) => `sha256:${createHash("sha256").update(bytes).digest("hex")}`;
const sourcePaths = mode === "legacy"
  ? [
      "deploy/aws-private-pilot/compose.yaml",
      "deploy/aws-private-pilot/gateway/Caddyfile",
      "deploy/aws-private-pilot/image-lock.json",
      "deploy/aws-private-pilot/release-manifest.schema.json",
      "deploy/aws-private-pilot/runtime/app-entrypoint.sh",
      "deploy/aws-private-pilot/runtime/ipv6-preflight.sh",
      "deploy/aws-private-pilot/runtime/supervisor.sh",
      "deploy/aws-private-pilot/systemd/aviasurveil360-private-pilot.service",
      "infra/terraform/modules/aws-private-pilot/storage.tf",
      "infra/terraform/modules/aws-private-pilot/network.tf",
      "infra/terraform/modules/aws-private-pilot/operations.tf",
      "apps/api/Dockerfile",
      "apps/web/Dockerfile",
      "apps/api/internal/documents/gotenberg_renderer.go",
    ]
  : [
      "deploy/aws-private-pilot/compose.yaml",
      "deploy/aws-private-pilot/gateway/Caddyfile",
      "deploy/aws-private-pilot/image-lock.json",
      "deploy/aws-private-pilot/release-manifest.schema.json",
      "deploy/aws-private-pilot/runtime/app-entrypoint.sh",
      "deploy/aws-private-pilot/runtime/ipv6-preflight.sh",
      "deploy/aws-private-pilot/runtime/supervisor.sh",
      "deploy/aws-private-pilot/systemd/aviasurveil360-private-pilot.service",
      "infra/terraform/modules/aws-private-pilot/storage.tf",
      "infra/terraform/modules/aws-private-pilot/network.tf",
      "infra/terraform/modules/aws-private-pilot/operations.tf",
      "apps/api/Dockerfile",
      "deploy/aws-private-pilot/gateway/Dockerfile",
      "apps/web/src/sw.ts",
      "apps/web/vite.config.ts",
      "apps/web/package.json",
      "apps/web/package-lock.json",
      "apps/api/internal/documents/native_renderer.go",
      "apps/api/internal/documents/fonts/SOURCE.txt",
    ];

const files = sourcePaths.filter((relativePath) => existsSync(path.join(root, relativePath))).map((relativePath) => ({
  path: relativePath,
  sha256: sha256(readFileSync(path.join(root, relativePath))),
}));
const aggregateInput = JSON.stringify(files);

const legacy = {
  schemaVersion: 1,
  inventoryId: "aws-private-pilot-runtime-legacy-2026-08-11",
  profile: "aws-private-pilot",
  mode,
  immutable: true,
  architecture: "linux/arm64",
  longRunningComposeRoles: ["api", "data-feed-worker", "gateway", "gotenberg", "keycloak", "scheduler", "web", "worker"],
  boundedJobs: ["database-bootstrap", "keycloak-bootstrap", "migration"],
  systemdOwnedRoles: ["cloudflared", "private-pilot-compose"],
  executableCommands: ["api", "data-feed-backfill", "data-feed-reconcile", "data-feed-replay", "data-feed-worker", "migrate", "scheduler", "worker"],
  dockerTargets: ["api", "data-feed-backfill", "data-feed-reconcile", "data-feed-replay", "data-feed-worker", "gotenberg", "migration", "scheduler", "web", "worker"],
  composeRoles: ["api", "data-feed-worker", "gateway", "gotenberg", "keycloak", "scheduler", "web", "worker", "database-bootstrap", "keycloak-bootstrap", "migration"],
  supervisorRoles: ["api", "data-feed-worker", "gateway", "gotenberg", "keycloak", "scheduler", "web", "worker"],
  healthChecks: ["api", "data-feed-worker", "gateway", "gotenberg", "keycloak", "scheduler", "web", "worker"],
  releaseSubjects: ["api", "cloudflared", "data-feed-worker", "database-bootstrap", "gateway", "gotenberg-chromium", "keycloak", "migration", "scheduler", "web", "worker"],
  imageLocks: ["api", "cloudflared", "data-feed-worker", "database-bootstrap", "gateway", "gotenberg-chromium", "keycloak", "migration", "scheduler", "web", "worker"],
  ecrRepositories: ["application", "cloudflared", "database-bootstrap", "gateway", "gotenberg", "keycloak", "web"],
  secretsManagerContainers: ["app-database-password", "app-migration-password", "app-smtp-password", "data-feed-client-private-key", "data-feed-payload-key", "keycloak-database-password", "keycloak-service-client-secret", "keycloak-smtp-password", "oidc-client-secret", "session-encryption-key"],
  separateSecretIdentities: ["database-bootstrap-rds-managed", "cloudflare-tunnel-token-ssm"],
  iamGrants: ["runtime-data-feed-egress", "runtime-ecr", "runtime-secrets", "runtime-s3", "runtime-logs"],
  ipv6EgressDestinations: ["cloudflare", "smtp", "data-feed", "rds", "ecr", "ssm"],
  preflightDestinations: ["cloudflare", "smtp", "data-feed", "rds", "ecr", "ssm"],
  logGroups: ["api", "cloudflared", "gateway", "gotenberg", "host", "keycloak", "scheduler", "vpc-flow", "web", "worker"],
  costInputs: ["api", "cloudflared", "gateway", "gotenberg", "keycloak", "scheduler", "web", "worker"],
  sourceFiles: files,
  layerDeltas: {
    "8.1": { owner: "data-feed-runtime", removes: ["data-feed-worker", "data-feed-backfill", "data-feed-replay", "data-feed-reconcile", "AVIA_DATA_FEED_*", "data-feed-secrets", "data-feed-egress"] },
    "8.2": { owner: "reminder-controller", removes: ["scheduler", "reminder-scheduler", "scheduler-health-marker"] },
    "8.3": { owner: "gateway-static-artifact", removes: ["web", "web-server", "AVIA_WEB_IMAGE", "web-ecr", "web-proxy-hop"] },
    "8.4": { owner: "native-pdf", removes: ["gotenberg", "gotenberg-chromium", "Chromium", "GotenbergRenderer", "gotenberg-network"] },
    "8.5": { owner: "target-reconciliation", reduces: ["long-running-role-count", "release-subject-count", "ecr-repository-count", "runtime-secret-count", "log-group-count", "retired-surface"] },
  },
  generatedAt: "2026-08-11T00:00:00Z",
};

const target = {
  ...legacy,
  inventoryId: "aws-private-pilot-runtime-final-target-2026-08-11",
  mode,
  longRunningComposeRoles: ["api", "gateway", "keycloak", "worker"],
  executableCommands: ["api", "database-bootstrap", "keycloak-bootstrap", "migrate", "worker"],
  dockerTargets: ["api", "database-bootstrap", "gateway", "keycloak", "migration", "worker"],
  composeRoles: ["api", "gateway", "keycloak", "worker", "database-bootstrap", "keycloak-bootstrap", "migration"],
  supervisorRoles: ["api", "gateway", "keycloak", "worker"],
  healthChecks: ["api", "gateway", "keycloak", "worker"],
  releaseSubjects: ["api", "cloudflared", "database-bootstrap", "gateway", "keycloak", "migration", "worker"],
  imageLocks: ["api", "cloudflared", "database-bootstrap", "gateway", "keycloak", "migration", "worker"],
  ecrRepositories: ["application", "cloudflared", "database-bootstrap", "gateway", "keycloak"],
  secretsManagerContainers: ["app-database-password", "app-migration-password", "app-smtp-password", "keycloak-database-password", "keycloak-service-client-secret", "keycloak-smtp-password", "oidc-client-secret", "session-encryption-key"],
  iamGrants: ["runtime-ecr", "runtime-secrets", "runtime-s3", "runtime-logs"],
  ipv6EgressDestinations: ["cloudflare", "smtp", "rds", "ecr", "ssm"],
  preflightDestinations: ["cloudflare", "smtp", "rds", "ecr", "ssm"],
  logGroups: ["api", "cloudflared", "gateway", "host", "keycloak", "vpc-flow", "worker"],
  costInputs: ["api", "cloudflared", "gateway", "keycloak", "worker"],
};

const inventory = mode === "legacy" ? legacy : target;
inventory.sourceFiles = [...inventory.sourceFiles].sort((a, b) => a.path.localeCompare(b.path));
inventory.canonicalAggregateSha256 = sha256(JSON.stringify({ ...inventory, canonicalAggregateSha256: undefined }));
const destination = path.join(root, "deploy/aws-private-pilot", `runtime-inventory-${mode}.json`);
if (process.argv.includes("--stdout")) {
  process.stdout.write(`${JSON.stringify(inventory, null, 2)}\n`);
} else {
  writeFileSync(destination, `${JSON.stringify(inventory, null, 2)}\n`, { mode: 0o644 });
  console.log(`${destination}: ${inventory.canonicalAggregateSha256}`);
}
