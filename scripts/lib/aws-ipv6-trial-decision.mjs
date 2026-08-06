#!/usr/bin/env node
import assert from "node:assert/strict";
import { lstatSync, readFileSync } from "node:fs";
import path from "node:path";

const DEFAULT_MAX_EVIDENCE_AGE_DAYS = 35;
const REQUIRED_SERVICES = ["cloudflared", "gateway", "web-demo"];
const REQUIRED_OWNERS = ["platform", "security", "dns", "cost", "rollbackDestroy"];
const ALLOWED_SECRET_REFERENCE_KEY = /(?:Env|Source|ParameterName|ParameterArn)$/;
const ISO_DATE = /^\d{4}-\d{2}-\d{2}$/u;
const ISO_DATE_TIME = /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d+)?Z$/u;

function add(errors, condition, message) {
  if (!condition) errors.push(message);
}

function requiredString(value, label, errors) {
  add(errors, typeof value === "string" && value.trim().length > 0, `missing-owner-input:${label}`);
}

function requiredBoolean(value, label, errors) {
  add(errors, typeof value === "boolean", `missing-owner-input:${label}`);
}

function requiredPositiveNumber(value, label, errors) {
  add(errors, typeof value === "number" && Number.isFinite(value) && value > 0, `invalid-number:${label}`);
}

function parseDate(value, label, errors) {
  if (typeof value !== "string" || !ISO_DATE_TIME.test(value)) {
    errors.push(`invalid-time:${label}`);
    return null;
  }
  const parsed = Date.parse(value);
  if (!Number.isFinite(parsed)) errors.push(`invalid-time:${label}`);
  return parsed;
}

function parseDateOnly(value, label, errors) {
  if (typeof value !== "string" || !ISO_DATE.test(value)) {
    errors.push(`invalid-date:${label}`);
    return null;
  }
  const parsed = Date.parse(`${value}T00:00:00Z`);
  if (!Number.isFinite(parsed)) errors.push(`invalid-date:${label}`);
  return parsed;
}

function walkForSecrets(value, keyPath, errors) {
  if (Array.isArray(value)) {
    value.forEach((entry, index) => walkForSecrets(entry, `${keyPath}[${index}]`, errors));
    return;
  }
  if (!value || typeof value !== "object") {
    if (typeof value === "string" && /(-----BEGIN|eyJ[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}\.|(?:^|\s)sk-[A-Za-z0-9]{16,})/u.test(value)) {
      errors.push(`secret-looking-value:${keyPath}`);
    }
    return;
  }
  for (const [key, child] of Object.entries(value)) {
    const childPath = keyPath ? `${keyPath}.${key}` : key;
    const normalizedKey = key.replaceAll("_", "").replaceAll("-", "").toLowerCase();
    const forbiddenSecretKey = new Set([
      "apitoken",
      "connectortoken",
      "token",
      "tokenvalue",
      "secret",
      "secretstring",
      "secretvalue",
      "password",
      "privatekey",
      "credential",
    ]).has(normalizedKey);
    if (forbiddenSecretKey && !ALLOWED_SECRET_REFERENCE_KEY.test(key)) {
      errors.push(`secret-value-field:${childPath}`);
    }
    walkForSecrets(child, childPath, errors);
  }
}

export function validateDecision(decision, { now = Date.now(), maxEvidenceAgeDays = DEFAULT_MAX_EVIDENCE_AGE_DAYS } = {}) {
  const errors = [];
  add(errors, decision && typeof decision === "object" && !Array.isArray(decision), "invalid-json:root-object");
  if (!decision || typeof decision !== "object" || Array.isArray(decision)) return errors;

  add(errors, decision.schemaVersion === 1, "unsupported-schema-version");
  requiredString(decision.decisionId, "decisionId", errors);
  const issuedAt = parseDate(decision.issuedAt, "issuedAt", errors);
  const expiresAt = parseDate(decision.expiresAt, "expiresAt", errors);
  if (issuedAt !== null) add(errors, issuedAt <= now, "future-decision:issuedAt");
  if (expiresAt !== null) add(errors, expiresAt > now, "stale-decision:expiresAt");
  if (issuedAt !== null && expiresAt !== null) add(errors, issuedAt < expiresAt, "contradictory-window");

  const aws = decision.aws;
  add(errors, aws && typeof aws === "object", "missing-owner-input:aws");
  if (aws && typeof aws === "object") {
    add(errors, /^\d{12}$/u.test(aws.accountId ?? ""), "invalid-account-id");
    add(errors, /^arn:aws[a-z-]*:iam::\d{12}:role\/[A-Za-z0-9+=,.@_-]+$/u.test(aws.operatorRoleArn ?? ""), "invalid-operator-role");
    add(errors, !aws.operatorRoleArn || aws.operatorRoleArn.includes(`::${aws.accountId}:`), "operator-role-account-mismatch");
    requiredString(aws.region, "aws.region", errors);
    requiredString(aws.availabilityZone, "aws.availabilityZone", errors);
    add(errors, /^(?:af|ap|ca|eu|il|me|mx|sa|us)-[a-z0-9-]+-\d$/u.test(aws.region ?? ""), "invalid-region");
    add(errors, /^[a-z]{2,3}(?:-[a-z0-9]+)+[a-z]$/u.test(aws.availabilityZone ?? ""), "invalid-availability-zone");
  }

  const cloudflare = decision.cloudflare;
  add(errors, cloudflare && typeof cloudflare === "object", "missing-owner-input:cloudflare");
  if (cloudflare && typeof cloudflare === "object") {
    add(errors, /^[0-9a-f]{32}$/u.test(cloudflare.accountId ?? ""), "invalid-cloudflare-account-id");
    add(errors, /^[0-9a-f]{32}$/u.test(cloudflare.zoneId ?? ""), "invalid-cloudflare-zone-id");
    requiredString(cloudflare.hostname, "cloudflare.hostname", errors);
    add(errors, cloudflare.apiTokenEnv === "CLOUDFLARE_API_TOKEN", "invalid-api-token-source");
    requiredString(cloudflare.connectorTokenParameterName, "cloudflare.connectorTokenParameterName", errors);
    add(errors, /^\/[A-Za-z0-9._\/-]+$/u.test(cloudflare.connectorTokenParameterName ?? ""), "invalid-connector-parameter-name");
    add(errors, cloudflare.edgeIpVersion === 6 || cloudflare.edgeIpVersion === "6", "cloudflare-edge-ip-version-must-be-6");
  }

  const access = decision.access;
  add(errors, access && typeof access === "object", "missing-owner-input:access");
  if (access && typeof access === "object") {
    requiredBoolean(access.publicExposure, "access.publicExposure", errors);
    const identities = Array.isArray(access.allowedIdentities) ? access.allowedIdentities : [];
    const domains = Array.isArray(access.allowedDomains) ? access.allowedDomains : [];
    add(errors, access.publicExposure ? identities.length === 0 && domains.length === 0 : identities.length + domains.length > 0, "contradictory-access-audience");
    add(errors, identities.every((item) => typeof item === "string" && /^[^@\s]+@[^@\s]+$/u.test(item)), "invalid-access-identity");
    add(errors, domains.every((item) => typeof item === "string" && /^[a-z0-9][a-z0-9.-]+[a-z0-9]$/u.test(item)), "invalid-access-domain");
  }

  const runtime = decision.runtime;
  add(errors, runtime && typeof runtime === "object", "missing-owner-input:runtime");
  if (runtime && typeof runtime === "object") {
    add(errors, runtime.profile === "demo", "runtime-profile-must-be-demo");
    add(errors, JSON.stringify(runtime.services) === JSON.stringify(REQUIRED_SERVICES), "runtime-services-must-be-milestone-1");
    add(errors, runtime.architecture === "linux/arm64", "runtime-architecture-must-be-linux-arm64");
    add(errors, runtime.instanceType === "t4g.small", "instance-type-must-be-t4g-small");
    add(errors, /^\/aws\/service\/ami-amazon-linux-latest\/al2023-ami-kernel-default-arm64$/u.test(runtime.amiSsmParameterName ?? ""), "runtime-ami-must-be-aws-arm64-al2023-ssm-parameter");
    const digests = runtime.imageDigests;
    add(errors, digests && typeof digests === "object", "missing-owner-input:runtime.imageDigests");
    if (digests && typeof digests === "object") {
      for (const service of REQUIRED_SERVICES) {
        add(errors, /^.+@sha256:[0-9a-f]{64}$/u.test(digests[service] ?? ""), `invalid-image-digest:${service}`);
      }
    }
  }

  const network = decision.network;
  add(errors, network && typeof network === "object", "missing-owner-input:network");
  if (network && typeof network === "object") {
    for (const key of ["ipv6OnlyRuntime", "publicIpv4", "elasticIp", "natGateway", "loadBalancer", "rds", "interfaceVpcEndpoints", "inboundSecurityGroupRules", "ssh", "amd64", "emulation"]) requiredBoolean(network[key], `network.${key}`, errors);
    add(errors, network.ipv6OnlyRuntime === true, "ipv6-only-runtime-required");
    for (const forbidden of ["publicIpv4", "elasticIp", "natGateway", "loadBalancer", "rds", "interfaceVpcEndpoints", "inboundSecurityGroupRules", "ssh", "amd64", "emulation"]) add(errors, network[forbidden] === false, `forbidden-network-feature:${forbidden}`);
  }

  const pricing = decision.pricingEvidence;
  add(errors, pricing && typeof pricing === "object", "missing-owner-input:pricingEvidence");
  if (pricing && typeof pricing === "object") {
    const checkedOn = parseDateOnly(pricing.checkedOn, "pricingEvidence.checkedOn", errors);
    if (checkedOn !== null) {
      add(errors, checkedOn <= now, "pricing-evidence-from-future");
      add(errors, now - checkedOn <= maxEvidenceAgeDays * 24 * 60 * 60 * 1000, "stale-pricing-evidence");
    }
    add(errors, Array.isArray(pricing.eligibleRegions) && pricing.eligibleRegions.length > 0, "missing-owner-input:pricingEvidence.eligibleRegions");
    add(errors, Array.isArray(pricing.eligibleRegions) && pricing.eligibleRegions.includes(aws?.region), "pricing-region-not-eligible");
    add(errors, pricing.trialHoursPerMonth === 750, "t4g-trial-hours-must-be-750");
    add(errors, pricing.offerExpiresOn === "2026-12-31", "t4g-trial-expiry-must-be-2026-12-31");
    add(errors, pricing.cpuCreditCaveat === true, "cpu-credit-caveat-required");
    requiredPositiveNumber(pricing.onDemandPriceAfterExpiryUsdPerHour, "pricingEvidence.onDemandPriceAfterExpiryUsdPerHour", errors);
    requiredPositiveNumber(pricing.nonComputeMonthlyEstimateUsd, "pricingEvidence.nonComputeMonthlyEstimateUsd", errors);
  }

  const cost = decision.cost;
  add(errors, cost && typeof cost === "object", "missing-owner-input:cost");
  if (cost && typeof cost === "object") {
    for (const key of ["monthlyCeilingUsd", "oneRunCeilingUsd", "projectedMonthlyUsd", "projectedOneRunUsd", "nonComputeCeilingUsd"]) requiredPositiveNumber(cost[key], `cost.${key}`, errors);
    add(errors, cost.monthlyCeilingUsd <= 500, "cost-ceiling-too-high:monthly");
    add(errors, cost.oneRunCeilingUsd <= 250, "cost-ceiling-too-high:one-run");
    add(errors, cost.nonComputeCeilingUsd <= cost.monthlyCeilingUsd, "cost-ceiling-too-high:non-compute");
    add(errors, cost.projectedMonthlyUsd <= cost.monthlyCeilingUsd, "over-budget:monthly");
    add(errors, cost.projectedOneRunUsd <= cost.oneRunCeilingUsd, "over-budget:one-run");
    add(errors, cost.nonComputeCeilingUsd >= (pricing?.nonComputeMonthlyEstimateUsd ?? Number.POSITIVE_INFINITY), "over-budget:non-compute");
    add(errors, Array.isArray(cost.alertRecipients) && cost.alertRecipients.length > 0, "missing-owner-input:cost.alertRecipients");
    add(errors, Array.isArray(cost.alertRecipients) && cost.alertRecipients.every((recipient) => typeof recipient === "string" && /^[^@\s]+@[^@\s]+\.[^@\s]+$/u.test(recipient)), "invalid-alert-recipient");
  }

  const persistence = decision.persistence;
  add(errors, persistence && typeof persistence === "object", "missing-owner-input:persistence");
  if (persistence && typeof persistence === "object") {
    add(errors, Number.isInteger(persistence.rootVolumeGiB) && persistence.rootVolumeGiB >= 8 && persistence.rootVolumeGiB <= 64, "invalid-root-volume");
    requiredBoolean(persistence.deleteRootVolumeOnTermination, "persistence.deleteRootVolumeOnTermination", errors);
    requiredBoolean(persistence.snapshotRetained, "persistence.snapshotRetained", errors);
  }

  const lifecycle = decision.lifecycle;
  add(errors, lifecycle && typeof lifecycle === "object", "missing-owner-input:lifecycle");
  if (lifecycle && typeof lifecycle === "object") {
    const trialExpiresAt = parseDate(lifecycle.trialExpiresAt, "lifecycle.trialExpiresAt", errors);
    if (trialExpiresAt !== null) {
      add(errors, trialExpiresAt > now, "stale-lifecycle:trialExpiresAt");
      if (expiresAt !== null) add(errors, trialExpiresAt <= expiresAt, "contradictory-lifecycle-window");
    }
    add(errors, ["destroy-after-trial", "retain-with-owner-expiry"].includes(lifecycle.retainOrDestroy), "invalid-retain-destroy-decision");
  }

  const owners = decision.owners;
  add(errors, owners && typeof owners === "object", "missing-owner-input:owners");
  if (owners && typeof owners === "object") for (const owner of REQUIRED_OWNERS) requiredString(owners[owner], `owners.${owner}`, errors);

  walkForSecrets(decision, "", errors);
  return [...new Set(errors)];
}

export function validateDecisionFile(filePath, options = {}) {
  const errors = [];
  let stat;
  try {
    stat = lstatSync(filePath);
  } catch {
    return { errors: [`missing-owner-input:file:${filePath}`] };
  }
  if (!stat.isFile()) errors.push("invalid-decision-file:not-regular-file");
  if ((stat.mode & 0o077) !== 0) errors.push("artifact-permission:decision-file-must-be-0600-or-stricter");
  try {
    const directoryStat = lstatSync(path.dirname(filePath));
    if (!directoryStat.isDirectory() || (directoryStat.mode & 0o077) !== 0) errors.push("artifact-permission:decision-directory-must-be-0700-or-stricter");
  } catch {
    errors.push("artifact-permission:decision-directory-unreadable");
  }
  let decision;
  try {
    decision = JSON.parse(readFileSync(filePath, "utf8"));
  } catch (error) {
    errors.push(`invalid-json:${error instanceof Error ? error.message : String(error)}`);
    return { errors };
  }
  errors.push(...validateDecision(decision, options));
  return { errors, decision };
}

function usage() {
  console.error("usage: aws-ipv6-trial-decision.mjs <decision.json> [--max-evidence-age-days N]");
}

if (process.argv[1] && path.resolve(process.argv[1]) === path.resolve(new URL(import.meta.url).pathname)) {
  const filePath = process.argv[2];
  if (!filePath) {
    usage();
    process.exitCode = 64;
  } else {
    const ageIndex = process.argv.indexOf("--max-evidence-age-days");
    const maxEvidenceAgeDays = ageIndex >= 0 ? Number(process.argv[ageIndex + 1]) : DEFAULT_MAX_EVIDENCE_AGE_DAYS;
    const result = validateDecisionFile(filePath, { maxEvidenceAgeDays });
    if (result.errors.length > 0) {
      for (const error of result.errors) console.error(error);
      process.exitCode = 65;
    } else {
      console.log("verified locally: aws-ipv6-trial decision contract");
    }
  }
}
