#!/usr/bin/env node
import { readFileSync, statSync } from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

import { releaseManifestDigest, validateReleaseManifest } from "./lib/aws-private-pilot-release.mjs";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const [action, manifestArgument, ...flags] = process.argv.slice(2);
const supported = new Set(["validate", "render", "build", "export", "bootstrap", "migrate", "start", "health", "drain", "rollback", "retain", "destroy"]);

if (!supported.has(action) || !manifestArgument || flags.some((flag) => flag !== "--execute")) {
  process.stderr.write("usage: scripts/aws-private-pilot-release.mjs <validate|render|build|export|bootstrap|migrate|start|health|drain|rollback|retain|destroy> <private-manifest.json> [--execute]\n");
  process.exit(64);
}

const manifestPath = path.resolve(manifestArgument);
let raw;
let manifest;
try {
  const mode = statSync(manifestPath).mode & 0o777;
  if ((mode & 0o077) !== 0) throw new Error("release manifest must not be group/world accessible");
  raw = readFileSync(manifestPath);
  manifest = JSON.parse(raw);
} catch (error) {
  process.stderr.write(`invalid-private-release-manifest: ${error.message}\n`);
  process.exit(66);
}

const errors = validateReleaseManifest(manifest, { root });
if (errors.length > 0) {
  process.stderr.write(`release-contract-failed: ${errors.join(",")}\n`);
  process.exit(65);
}

const manifestSha256 = releaseManifestDigest(raw);
if (action === "validate") {
  process.stdout.write(`${JSON.stringify({ status: "verified locally", artifactStatus: "candidate-only", releasePending: true, productionReady: false, manifestSha256 })}\n`);
  process.exit(0);
}

if (flags.includes("--execute")) {
  process.stderr.write("blocked: Task 7 exact authorization is required; no remote, deployment, migration, traffic, rollback, retain, or destroy action was executed\n");
  process.exit(77);
}

if (action === "rollback" && manifest.rollback.nMinusOneCompatible !== true) {
  process.stderr.write("binary-rollback-forbidden: use roll-forward-or-coordinated-restore\n");
  process.exit(78);
}

const compose = "deploy/aws-private-pilot/compose.yaml";
const plans = {
  render: [
    "verify root-owned runtime environment and secret-file references",
    `docker compose --project-name aviasurveil360-private-pilot --file ${compose} config`,
    "compare rendered image subjects with the resolved image lock",
  ],
  build: [
    "build application subjects with docker buildx --platform linux/arm64",
    "record OCI index and config digests; reject amd64 or emulation",
    "generate SBOM, provenance, and vulnerability evidence before resolving the image lock",
  ],
  export: [
    "export digest-bound linux/arm64 OCI subjects and the signed release bundle",
    "verify every exported digest against this manifest before transfer",
  ],
  bootstrap: [
    `docker compose --project-name aviasurveil360-private-pilot --file ${compose} --profile bootstrap run --rm database-bootstrap`,
    `docker compose --project-name aviasurveil360-private-pilot --file ${compose} --profile bootstrap run --rm keycloak-bootstrap`,
    "remove bootstrap credential files before normal runtime",
  ],
  migrate: [
    `docker compose --project-name aviasurveil360-private-pilot --file ${compose} --profile bootstrap run --rm -e AVIA_DATABASE_BOOTSTRAP_MODE=migration-enable database-bootstrap`,
    `docker compose --project-name aviasurveil360-private-pilot --file ${compose} --profile migration run --rm migration`,
    `docker compose --project-name aviasurveil360-private-pilot --file ${compose} --profile bootstrap run --rm -e AVIA_DATABASE_BOOTSTRAP_MODE=lockdown database-bootstrap`,
    "run lockdown after every success or failure path, then verify migration version and N/N-1 compatibility before traffic",
  ],
  start: [
    "systemctl start aviasurveil360-private-pilot.service",
    "systemctl start aviasurveil360-private-pilot-tunnel.service aviasurveil360-private-pilot-tunnel-health.timer",
    "require application health and four Cloudflare Tunnel edge connections before Cloudflare traffic",
  ],
  health: [
    "/opt/aviasurveil360/private-pilot/deploy/aws-private-pilot/runtime/supervisor.sh health",
    "/opt/aviasurveil360/private-pilot/deploy/aws-private-pilot/runtime/supervisor.sh tunnel-health",
    "record service and tunnel health, restarts, cgroup CPU/memory/PIDs, disk, connection pools, and swap",
  ],
  drain: [
    "systemctl stop aviasurveil360-private-pilot-tunnel-health.timer aviasurveil360-private-pilot-tunnel.service",
    "/opt/aviasurveil360/private-pilot/deploy/aws-private-pilot/runtime/supervisor.sh drain",
    "preserve leased/outbox work for idempotent crash recovery before stopping the remaining services",
  ],
  rollback: [
    `install predecessor manifest ${manifest.rollback.predecessorManifestSha256} only after digest and N/N-1 verification`,
    "drain, switch exact image/config subjects, start, and prove health without database downgrade",
  ],
  retain: [
    `record retain decision for ${manifest.targets.terraformAddresses.join(",")}`,
    "make no resource mutation and preserve deletion protection, versions, backups, and evidence",
  ],
  destroy: [
    `create a reviewed destroy-only plan scoped to ${manifest.targets.terraformAddresses.join(",")}`,
    "require separate backup/export, legal/records, recovery, cost, and residue authorization before apply",
  ],
};

process.stdout.write(`${JSON.stringify({
  mode: "dry-run",
  action,
  manifestSha256,
  target: {
    awsProfile: manifest.targets.awsProfile,
    accountId: manifest.targets.awsAccountId,
    region: manifest.targets.region,
    hostname: manifest.targets.hostname,
    instanceId: manifest.targets.instanceId,
  },
  steps: plans[action],
  externalEvidence: "not run",
  artifactStatus: "candidate-only",
  release: "release pending",
  productionReady: "not established",
}, null, 2)}\n`);
