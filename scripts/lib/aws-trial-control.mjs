#!/usr/bin/env node

import { createHash } from "node:crypto";
import {
  chmodSync,
  existsSync,
  readFileSync,
  readdirSync,
  statSync,
  writeFileSync,
} from "node:fs";
import path from "node:path";
import { spawnSync } from "node:child_process";

const phases = {
  "artifact-publication": new Set(["artifact-publication"]),
  "bootstrap": new Set(["remote-state"]),
  "data-runtime": new Set(["backup", "compute", "database"]),
  "foundation-ecr": new Set([
    "ecr",
    "identity-secrets",
    "load-balancer",
    "network",
    "object-storage",
    "observability",
    "security",
    "service-endpoints",
  ]),
};

const wrapperNames = [
  "aws-trial-plan.sh",
  "check-aws-plan.sh",
  "aws-trial-apply.sh",
  "aws-trial-publish-artifacts.sh",
  "aws-trial-smoke.sh",
  "aws-trial-rollback.sh",
  "aws-trial-destroy.sh",
];

class GateError extends Error {
  constructor(reason, detail) {
    super(detail);
    this.reason = reason;
  }
}

function fail(reason, detail) {
  throw new GateError(reason, detail);
}

function readJson(filePath, reason) {
  if (!existsSync(filePath)) fail(reason, `${filePath} does not exist`);
  try {
    return JSON.parse(readFileSync(filePath, "utf8"));
  } catch {
    fail(reason, `${filePath} is not valid JSON`);
  }
}

function requireString(value, reason, detail) {
  if (typeof value !== "string" || value.trim() === "") fail(reason, detail);
  return value;
}

function requireSha(value, reason, detail) {
  if (typeof value !== "string" || !/^[a-f0-9]{64}$/.test(value)) {
    fail(reason, detail);
  }
  return value;
}

function requireProtected(filePath) {
  if (!existsSync(filePath)) fail("missing-artifact", `${filePath} does not exist`);
  if ((statSync(filePath).mode & 0o077) !== 0) {
    fail("artifact-permission", `${filePath} must have mode 0600 or stricter`);
  }
}

function resolveInside(bundleDirectory, relativePath, reason) {
  requireString(relativePath, reason, `${reason} path is required`);
  const resolved = path.resolve(bundleDirectory, relativePath);
  const prefix = `${path.resolve(bundleDirectory)}${path.sep}`;
  if (!resolved.startsWith(prefix)) fail(reason, "artifact path escapes bundle");
  return resolved;
}

function sha256File(filePath) {
  return createHash("sha256").update(readFileSync(filePath)).digest("hex");
}

function validateDecision(decision, phase, now = Date.now()) {
  if (decision.schemaVersion !== 1) {
    fail("decision-schema", "schemaVersion must be 1");
  }
  if (!/^\d{12}$/.test(decision.accountId ?? "")) {
    fail("account-mismatch", "decision accountId must be 12 digits");
  }
  if (!/^(?:af|ap|ca|eu|il|me|mx|sa|us)-[a-z0-9-]+-\d$/.test(
    decision.region ?? "",
  )) {
    fail("region-mismatch", "decision region is invalid");
  }
  if (decision.dataResidencyApproved !== true) {
    fail("data-residency", "data residency approval is required");
  }
  if (!Array.isArray(decision.approvedPhases) ||
      !decision.approvedPhases.includes(phase)) {
    fail("phase-boundary", `${phase} is not approved for planning`);
  }
  if (!phases[phase]) fail("phase-boundary", `${phase} is not a known phase`);
  if (!Number.isFinite(decision.budgetCeilingUsd) ||
      decision.budgetCeilingUsd <= 0 ||
      decision.budgetCeilingUsd > 10_000) {
    fail("cost-unbounded", "budget ceiling must be within (0, 10000]");
  }
  const capacity = decision.capacity ?? {};
  if (![capacity.min, capacity.desired, capacity.max].every(Number.isInteger) ||
      capacity.min < 1 ||
      capacity.desired < capacity.min ||
      capacity.max < capacity.desired ||
      capacity.max > 20) {
    fail("capacity-unbounded", "capacity must satisfy 1 <= min <= desired <= max <= 20");
  }
  if (!Number.isInteger(decision.backupRetentionDays) ||
      decision.backupRetentionDays < 1 ||
      decision.backupRetentionDays > 35) {
    fail("missing-decision", "backup retention must be within 1-35 days");
  }
  for (const owner of ["platform", "records", "release", "security"]) {
    requireString(
      decision.ownerContacts?.[owner],
      "missing-decision",
      `${owner} owner contact is required`,
    );
  }
  if (!["destroy-after-trial", "retain-approved"].includes(
    decision.destroyDecision,
  )) {
    fail("missing-decision", "destroy or retention decision is required");
  }
  const startsAt = Date.parse(decision.changeWindow?.startsAt ?? "");
  const endsAt = Date.parse(decision.changeWindow?.endsAt ?? "");
  if (!Number.isFinite(startsAt) || !Number.isFinite(endsAt) ||
      startsAt >= endsAt || now < startsAt || now > endsAt) {
    fail("change-window", "a current bounded change window is required");
  }
  requireString(decision.domain, "missing-decision", "domain is required");
  if (phase !== "bootstrap") {
    const certificateArn = requireString(
      decision.certificateArn,
      "certificate-mismatch",
      "certificate ARN is required",
    );
    const certificatePattern = new RegExp(
      `^arn:aws(?:-us-gov|-cn)?:acm:${decision.region}:${decision.accountId}:certificate/[A-Za-z0-9-]+$`,
    );
    if (!certificatePattern.test(certificateArn)) {
      fail(
        "certificate-mismatch",
        "certificate ARN must match the reviewed account and region",
      );
    }
  }
}

function normalizePlans(manifest) {
  if (Array.isArray(manifest.plans)) return manifest.plans;
  if (manifest.planBinary && manifest.planJson) {
    return [{
      binary: manifest.planBinary,
      json: manifest.planJson,
      unit: manifest.commands?.[0]?.unit,
    }];
  }
  fail("missing-artifact", "manifest must bind at least one plan");
}

function rejectSensitivePlanValue(value, location = "plan") {
  if (Array.isArray(value)) {
    value.forEach((entry, index) =>
      rejectSensitivePlanValue(entry, `${location}[${index}]`));
    return;
  }
  if (value === null || typeof value !== "object") return;

  for (const [key, entry] of Object.entries(value)) {
    const normalized = key.toLowerCase();
    const sensitiveKey = normalized === "password" ||
      normalized.endsWith("_password") ||
      [
        "access_key",
        "authorization_header",
        "private_key",
        "secret_string",
        "secret_value",
        "session_cookie",
        "token_value",
      ].includes(normalized);
    if (sensitiveKey && typeof entry === "string" && entry !== "") {
      fail("unredacted-plan", `${location}.${key} contains a sensitive value`);
    }
    if (typeof entry === "string" &&
        /-----BEGIN (?:RSA |EC |OPENSSH )?PRIVATE KEY-----/.test(entry)) {
      fail("unredacted-plan", `${location}.${key} contains a private key`);
    }
    rejectSensitivePlanValue(entry, `${location}.${key}`);
  }
}

function validateBundle(bundleDirectory, repositoryRoot, callerArn) {
  const absoluteBundle = path.resolve(bundleDirectory);
  if (!existsSync(absoluteBundle)) {
    fail("missing-artifact", `${absoluteBundle} does not exist`);
  }
  if ((statSync(absoluteBundle).mode & 0o077) !== 0) {
    fail("artifact-permission", `${absoluteBundle} must have mode 0700 or stricter`);
  }

  const decisionPath = path.join(absoluteBundle, "decision.json");
  const manifestPath = path.join(absoluteBundle, "manifest.json");
  if (!existsSync(decisionPath)) {
    fail("missing-decision", `${decisionPath} does not exist`);
  }
  requireProtected(decisionPath);
  requireProtected(manifestPath);
  const decision = readJson(decisionPath, "missing-decision");
  const manifest = readJson(manifestPath, "missing-manifest");
  const now = Date.now();

  if (manifest.schemaVersion !== 1) {
    fail("manifest-schema", "schemaVersion must be 1");
  }
  validateDecision(decision, manifest.phase, now);
  if (manifest.accountId !== decision.accountId) {
    fail("account-mismatch", "manifest and decision accountId differ");
  }
  if (manifest.region !== decision.region) {
    fail("region-mismatch", "manifest and decision region differ");
  }
  if (manifest.callerArn !== callerArn) {
    fail("caller-mismatch", "current caller does not match reviewed caller");
  }
  if (!manifest.callerArn.includes(`::${manifest.accountId}:`)) {
    fail("account-mismatch", "caller ARN does not belong to reviewed account");
  }
  if (manifest.decisionSha256 !== sha256File(decisionPath)) {
    fail("decision-hash", "decision file changed after plan creation");
  }
  const expiresAt = Date.parse(manifest.expiresAt ?? "");
  const cleanupAt = Date.parse(manifest.cleanupAt ?? "");
  if (!Number.isFinite(expiresAt) || expiresAt <= now) {
    fail("stale-plan", "plan bundle has expired");
  }
  if (!Number.isFinite(cleanupAt) || cleanupAt < expiresAt) {
    fail("stale-plan", "cleanup record must outlive plan expiry");
  }
  if (!["ephemeral-0600", "age-encrypted"].includes(manifest.retention)) {
    fail("artifact-permission", "retention must be ephemeral-0600 or age-encrypted");
  }
  if (!Number.isFinite(manifest.costEstimateUsd) ||
      manifest.costEstimateUsd < 0 ||
      manifest.costEstimateUsd > decision.budgetCeilingUsd) {
    fail("cost-unbounded", "cost estimate exceeds reviewed ceiling");
  }
  if (JSON.stringify(manifest.capacity) !== JSON.stringify(decision.capacity)) {
    fail("capacity-unbounded", "manifest capacity differs from decision");
  }
  if (manifest.policyDenials !== 0) {
    fail("policy-denials", "manifest records policy denials");
  }

  const allowedUnits = phases[manifest.phase];
  if (!Array.isArray(manifest.commands) || manifest.commands.length === 0) {
    fail("phase-boundary", "phase command manifest is required");
  }
  for (const command of manifest.commands) {
    if (command.kind !== "plan" ||
        !["terraform", "terragrunt"].includes(command.executable) ||
        !allowedUnits.has(command.unit) ||
        command.arguments?.some((argument) => /apply|destroy/.test(argument))) {
      fail("phase-boundary", "command escapes the reviewed plan phase");
    }
  }

  const plans = normalizePlans(manifest).map((plan) => {
    if (!allowedUnits.has(plan.unit)) {
      fail("phase-boundary", `${plan.unit} is outside ${manifest.phase}`);
    }
    const binaryPath = resolveInside(
      absoluteBundle,
      plan.binary?.path,
      "missing-artifact",
    );
    const jsonPath = resolveInside(
      absoluteBundle,
      plan.json?.path,
      "missing-artifact",
    );
    requireProtected(binaryPath);
    requireProtected(jsonPath);
    if (sha256File(binaryPath) !== plan.binary.sha256 ||
        sha256File(jsonPath) !== plan.json.sha256) {
      fail("plan-hash", `${plan.unit} plan artifact changed`);
    }
    const planDocument = readJson(jsonPath, "plan-json");
    rejectSensitivePlanValue(planDocument);
    return { binaryPath, jsonPath, planDocument, unit: plan.unit };
  });

  for (const wrapperName of wrapperNames) {
    const expected = requireSha(
      manifest.wrappers?.[wrapperName],
      "wrapper-hash",
      `${wrapperName} hash is missing`,
    );
    const actual = sha256File(path.join(repositoryRoot, "scripts", wrapperName));
    if (actual !== expected) {
      fail("wrapper-hash", `${wrapperName} changed after plan creation`);
    }
  }
  requireSha(manifest.lockSha256, "lock-hash", "provider lock hash is required");
  requireString(
    manifest.terraformVersion,
    "tool-version",
    "Terraform version is required",
  );
  requireString(
    manifest.terragruntVersion,
    "tool-version",
    "Terragrunt version is required",
  );

  if (!Array.isArray(manifest.images) || manifest.images.length === 0) {
    fail("mutable-image", "at least one digest-bound image is required");
  }
  for (const image of manifest.images) {
    const expectedRepositoryPrefix =
      `${manifest.accountId}.dkr.ecr.${manifest.region}.amazonaws.com/`;
    if (!image.reference?.startsWith(expectedRepositoryPrefix)) {
      fail("image-scope", "image reference must match reviewed account and region");
    }
    if (!/@sha256:[a-f0-9]{64}$/.test(image.reference ?? "")) {
      fail("mutable-image", "image reference must end with an immutable digest");
    }
    requireSha(
      image.sbomSha256,
      "missing-sbom",
      "CycloneDX SBOM hash is required",
    );
    requireString(image.sbomPath, "missing-sbom", "CycloneDX SBOM path is required");
    if (image.trivyCritical !== 0 || image.trivyHigh !== 0 ||
        !Number.isFinite(Date.parse(image.scannedAt ?? ""))) {
      fail("unscanned-image", "image must have a current zero HIGH/CRITICAL scan");
    }
  }
  const reviewedImageDigests = new Set(
    manifest.images.map((image) => image.reference.split("@sha256:").at(-1)),
  );
  for (const plan of plans) {
    for (const resource of plan.planDocument.resource_changes ?? []) {
      if (resource.type !== "aws_launch_template") continue;
      const digest = resource.change?.after?.tags?.ImageDigest;
      if (typeof digest === "string" && !reviewedImageDigests.has(digest)) {
        fail(
          "image-binding",
          `${resource.address} digest lacks matching SBOM and scan evidence`,
        );
      }
    }
  }

  return { decision, manifest, plans };
}

function evaluatePolicy(plans, opaBin, policyPath) {
  for (const plan of plans) {
    const result = spawnSync(
      opaBin,
      [
        "eval",
        "--format=json",
        "--data",
        policyPath,
        "--input",
        plan.jsonPath,
        "data.aviasurveil360.aws_plan.deny",
      ],
      { encoding: "utf8" },
    );
    if (result.status !== 0) {
      fail("policy-tool", result.stderr.trim() || "OPA evaluation failed");
    }
    const response = JSON.parse(result.stdout);
    const denials = response.result?.[0]?.expressions?.[0]?.value ?? [];
    if (!Array.isArray(denials) || denials.length !== 0) {
      fail("policy-denials", `${plan.unit} has ${denials.length} denial(s)`);
    }
  }
}

function manifestField(bundleDirectory, field) {
  const manifest = readJson(
    path.join(path.resolve(bundleDirectory), "manifest.json"),
    "missing-manifest",
  );
  const value = field.split(".").reduce(
    (current, key) => current?.[key],
    manifest,
  );
  if (value === undefined || value === null || typeof value === "object") {
    fail("manifest-field", `${field} is not a scalar manifest field`);
  }
  process.stdout.write(String(value));
}

function authorization(action, bundleDirectory) {
  const manifest = readJson(
    path.join(path.resolve(bundleDirectory), "manifest.json"),
    "missing-manifest",
  );
  const plans = normalizePlans(manifest);
  const aggregate = createHash("sha256");
  for (const plan of [...plans].sort((a, b) => a.unit.localeCompare(b.unit))) {
    aggregate.update(`${plan.unit}:${plan.binary.sha256}\n`);
  }
  process.stdout.write(
    `exact-authorization:${action}:${manifest.phase}:${manifest.accountId}:${manifest.region}:${aggregate.digest("hex")}`,
  );
}

function listPlans(bundleDirectory) {
  const absoluteBundle = path.resolve(bundleDirectory);
  const manifest = readJson(
    path.join(absoluteBundle, "manifest.json"),
    "missing-manifest",
  );
  for (const plan of normalizePlans(manifest)) {
    const binaryPath = resolveInside(
      absoluteBundle,
      plan.binary?.path,
      "missing-artifact",
    );
    process.stdout.write(`${plan.unit}|${binaryPath}\n`);
  }
}

function planAuthorization(phase, decisionPath) {
  const decision = readJson(path.resolve(decisionPath), "missing-decision");
  validateDecision(decision, phase);
  process.stdout.write(
    `exact-authorization:plan:${phase}:${decision.accountId}:${decision.region}:${sha256File(path.resolve(decisionPath))}`,
  );
}

function createManifest(
  phase,
  decisionPath,
  bundleDirectory,
  repositoryRoot,
  callerArn,
  costEstimate,
  imageEvidencePath,
) {
  const absoluteBundle = path.resolve(bundleDirectory);
  const decision = readJson(path.resolve(decisionPath), "missing-decision");
  validateDecision(decision, phase);
  const images = readJson(path.resolve(imageEvidencePath), "unscanned-image");
  const allowedUnits = phases[phase];
  const plans = readdirSync(absoluteBundle)
    .filter((name) => name.endsWith(".tfplan"))
    .map((name) => {
      const stem = name.slice(0, -".tfplan".length);
      const unit = stem
        .replace(/^bootstrap__/, "")
        .replace(/^environments__aws-trial__components__/, "");
      if (!allowedUnits.has(unit)) {
        fail("phase-boundary", `${unit} is outside ${phase}`);
      }
      const binaryPath = path.join(absoluteBundle, name);
      const jsonPath = path.join(absoluteBundle, `${stem}.json`);
      requireProtected(binaryPath);
      requireProtected(jsonPath);
      return {
        binary: { path: name, sha256: sha256File(binaryPath) },
        json: { path: `${stem}.json`, sha256: sha256File(jsonPath) },
        unit,
      };
    })
    .sort((a, b) => a.unit.localeCompare(b.unit));
  if (plans.length === 0) fail("missing-artifact", "phase produced no plans");

  const now = Date.now();
  const manifest = {
    accountId: decision.accountId,
    artifactMode: "aws-read-only-plan",
    callerArn,
    capacity: decision.capacity,
    cleanupAt: new Date(now + 86_400_000).toISOString(),
    commands: plans.map(({ unit }) => ({
      arguments: ["plan"],
      executable: "terragrunt",
      kind: "plan",
      unit,
    })),
    costEstimateUsd: Number(costEstimate),
    createdAt: new Date(now).toISOString(),
    decisionSha256: sha256File(path.resolve(decisionPath)),
    expiresAt: new Date(
      Math.min(now + 3_600_000, Date.parse(decision.changeWindow.endsAt)),
    ).toISOString(),
    images,
    lockSha256: process.env.AVIA_AWS_LOCK_SHA256,
    phase,
    plans,
    policyDenials: 0,
    region: decision.region,
    retention: "ephemeral-0600",
    schemaVersion: 1,
    terraformVersion: process.env.AVIA_TERRAFORM_VERSION,
    terragruntVersion: process.env.AVIA_TERRAGRUNT_VERSION,
    wrappers: Object.fromEntries(
      wrapperNames.map((name) => [
        name,
        sha256File(path.join(repositoryRoot, "scripts", name)),
      ]),
    ),
  };
  const manifestPath = path.join(absoluteBundle, "manifest.json");
  writeFileSync(manifestPath, `${JSON.stringify(manifest, null, 2)}\n`, {
    mode: 0o600,
  });
  chmodSync(manifestPath, 0o600);
}

async function main() {
  const [command, ...args] = process.argv.slice(2);
  if (command === "validate") {
    const [bundle, repository, opaBin, policy, caller] = args;
    const result = validateBundle(bundle, repository, caller);
    evaluatePolicy(result.plans, opaBin, policy);
    process.stdout.write(
      `check-aws-plan: verified protected ${result.manifest.phase} plan bundle\n`,
    );
    return;
  }
  if (command === "field") {
    manifestField(args[0], args[1]);
    return;
  }
  if (command === "authorization") {
    authorization(args[0], args[1]);
    return;
  }
  if (command === "plans") {
    listPlans(args[0]);
    return;
  }
  if (command === "plan-authorization") {
    planAuthorization(args[0], args[1]);
    return;
  }
  if (command === "create-manifest") {
    createManifest(...args);
    return;
  }
  fail("usage", `unknown command ${command ?? ""}`);
}

main().catch((error) => {
  if (error instanceof GateError) {
    process.stderr.write(`check-aws-plan: ${error.reason}: ${error.message}\n`);
    process.exitCode = 64;
    return;
  }
  process.stderr.write(`check-aws-plan: internal-error: ${error.message}\n`);
  process.exitCode = 70;
});
