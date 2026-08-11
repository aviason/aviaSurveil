#!/usr/bin/env node
import { lstatSync, readFileSync } from "node:fs";
import path from "node:path";

const DAY_MS = 24 * 60 * 60 * 1000;
const REQUIRED_COMPOSE_SERVICES = [
  "gateway",
  "api",
  "worker",
  "keycloak",
];
const REQUIRED_RUNTIME_SUBJECTS = ["cloudflared", ...REQUIRED_COMPOSE_SERVICES];
const REQUIRED_OWNERS = [
  "platform",
  "security",
  "identity",
  "recordsLegal",
  "recovery",
  "monitoringOnCall",
  "cost",
  "release",
  "rollback",
];
const FORBIDDEN_SERVICES = [
  "postgres",
  "keycloak-postgres",
  "minio",
  "mailpit",
  "clamav",
  "backup-minio",
  "prometheus",
  "grafana",
  "loki",
  "tempo",
  "alertmanager",
  "fixture",
  "loader",
  "volume-init",
];
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
  add(errors, typeof value === "number" && Number.isFinite(value) && value > 0, `missing-owner-input:${label}`);
}

function parseDateTime(value, label, errors) {
  if (typeof value !== "string" || !ISO_DATE_TIME.test(value)) {
    errors.push(`invalid-time:${label}`);
    return null;
  }
  const parsed = Date.parse(value);
  if (!Number.isFinite(parsed)) errors.push(`invalid-time:${label}`);
  return parsed;
}

function parseDate(value, label, errors) {
  if (typeof value !== "string" || !ISO_DATE.test(value)) {
    errors.push(`invalid-date:${label}`);
    return null;
  }
  const parsed = Date.parse(`${value}T00:00:00Z`);
  if (!Number.isFinite(parsed)) errors.push(`invalid-date:${label}`);
  return parsed;
}

function exactArray(value, expected, label, errors) {
  add(errors, Array.isArray(value) && JSON.stringify(value) === JSON.stringify(expected), label);
}

function arnBelongsTo(value, { service, region, accountId, resourcePrefix }) {
  if (typeof value !== "string") return false;
  const parts = value.split(":");
  return parts.length >= 6
    && parts[0] === "arn"
    && /^aws(?:-[a-z0-9]+)*$/u.test(parts[1])
    && parts[2] === service
    && parts[3] === region
    && parts[4] === accountId
    && parts.slice(5).join(":").startsWith(resourcePrefix);
}

function availabilityZoneBelongsToRegion(zone, region) {
  return typeof zone === "string"
    && typeof region === "string"
    && zone.startsWith(region)
    && /^[a-z]$/u.test(zone.slice(region.length));
}

function validIpv6Cidrs(value) {
  return Array.isArray(value)
    && value.length > 0
    && value.every((cidr) => typeof cidr === "string" && cidr.includes(":") && cidr !== "::/0");
}

function scanForSecretValues(value, keyPath, errors) {
  if (Array.isArray(value)) {
    value.forEach((entry, index) => scanForSecretValues(entry, `${keyPath}[${index}]`, errors));
    return;
  }
  if (!value || typeof value !== "object") return;
  for (const [key, child] of Object.entries(value)) {
    const childPath = keyPath ? `${keyPath}.${key}` : key;
    const normalized = key.replaceAll(/[_-]/gu, "").toLowerCase();
    const secretField = /(?:password|secret|token|privatekey|accesskey|credential)$/u.test(normalized);
    const referenceField = /(?:arn|name|ref|reference|source|env)$/u.test(normalized);
    if (secretField && !referenceField) errors.push(`secret-value-field:${childPath}`);
    if (typeof child === "string" && /(?:-----BEGIN|AKIA[0-9A-Z]{16}|ASIA[0-9A-Z]{16}|sk-[A-Za-z0-9_-]{16,})/u.test(child)) {
      errors.push(`secret-looking-value:${childPath}`);
    }
    scanForSecretValues(child, childPath, errors);
  }
}

function validateAcceptedTopology(decision, errors) {
  const runtime = decision.runtime;
  add(errors, runtime && typeof runtime === "object", "missing-owner-input:runtime");
  if (runtime && typeof runtime === "object") {
    add(errors, runtime.profile === "aws-private-pilot", "runtime-profile-must-be-aws-private-pilot");
    add(errors, runtime.architecture === "linux/arm64", "runtime-architecture-must-be-linux-arm64");
    add(errors, runtime.instanceType === "t4g.small", "instance-type-must-be-t4g-small");
    exactArray(runtime.services, REQUIRED_COMPOSE_SERVICES, "runtime-services-mismatch", errors);
    add(errors, runtime.edgeConnector === "cloudflared-systemd-container", "edge-connector-must-be-systemd-cloudflared");
    add(errors, runtime.composeSupervisor === "systemd", "compose-supervisor-must-be-systemd");
    add(errors, runtime.gatewayHostPorts === 1, "gateway-must-be-only-host-port");
    add(errors, runtime.reminderController === "worker", "reminder-controller-must-be-worker-owned");
    add(errors, runtime.nonRoot === true, "runtime-must-be-non-root");
    add(errors, runtime.noNewPrivileges === true, "runtime-no-new-privileges-required");
    add(errors, runtime.dropAllCapabilities === true, "runtime-capability-drop-required");
    add(errors, runtime.dockerSocketMounted === false, "docker-socket-forbidden");
    add(errors, runtime.amd64 === false && runtime.emulation === false, "amd64-or-emulation-forbidden");
    const digests = runtime.imageDigests;
    add(errors, digests && typeof digests === "object", "missing-owner-input:runtime.imageDigests");
    if (digests && typeof digests === "object") {
      add(errors, Object.keys(digests).length === REQUIRED_RUNTIME_SUBJECTS.length, "runtime-image-subjects-mismatch");
      const ecrPrefix = `${decision.aws?.accountId}.dkr-ecr.${decision.aws?.region}.on.aws/`;
      for (const service of REQUIRED_RUNTIME_SUBJECTS) {
        add(errors, /^.+@sha256:[0-9a-f]{64}$/u.test(digests[service] ?? ""), `missing-owner-input:runtime.imageDigests.${service}`);
        add(errors, String(digests[service] ?? "").startsWith(ecrPrefix), `runtime-image-must-use-target-ecr:${service}`);
      }
    }
    const listed = Array.isArray(runtime.forbiddenServices) ? runtime.forbiddenServices : [];
    for (const service of FORBIDDEN_SERVICES) add(errors, listed.includes(service), `missing-runtime-prohibition:${service}`);
  }

  const network = decision.network;
  add(errors, network && typeof network === "object", "missing-owner-input:network");
  if (network && typeof network === "object") {
    add(errors, network.workloadAvailabilityZones === 1, "workload-must-be-single-az");
    add(errors, network.publicSubnets === 0, "public-subnets-forbidden");
    add(errors, network.dualStackAppSubnets === 1, "exactly-one-dual-stack-app-subnet-required");
    add(errors, network.privateDatabaseSubnets === 2, "rds-requires-two-private-subnets");
    add(errors, network.natGateways === 0, "nat-gateway-forbidden");
    add(errors, network.internetGateways === 0, "internet-gateway-forbidden");
    add(errors, network.egressOnlyInternetGateways === 1, "exactly-one-egress-only-internet-gateway-required");
    add(errors, network.s3GatewayEndpoints === 1, "exactly-one-s3-gateway-endpoint-required");
    add(errors, network.interfaceEndpoints === 0, "interface-endpoint-fleet-forbidden");
    add(errors, network.ec2PublicIpv4 === false, "public-ec2-forbidden");
    add(errors, network.ec2GlobalIpv6 === true, "global-ipv6-required");
    add(errors, network.runtimeSecurityGroupIngressRules === 0, "runtime-security-group-ingress-forbidden");
    add(errors, network.databaseSecurityGroupIngressRules === 1, "database-security-group-ingress-must-be-bounded");
    add(errors, network.ipv4DefaultRoute === false, "ipv4-default-route-forbidden");
    add(errors, network.ipv6DefaultRoute === "egress-only-internet-gateway", "ipv6-egress-route-mismatch");
    add(errors, network.origin === "cloudflare-tunnel-ipv6-loopback", "origin-path-mismatch");
    add(errors, network.gatewayBindAddress === "127.0.0.1", "gateway-must-bind-loopback");
    add(errors, network.cloudflareTunnelEdgeIpVersion === 6, "cloudflare-edge-ip-version-must-be-six");
    add(errors, network.cloudflareTunnelPort === 7844, "cloudflare-tunnel-port-must-be-7844");
    add(errors, validIpv6Cidrs(network.cloudflareTunnelIpv6Cidrs), "missing-owner-input:network.cloudflareTunnelIpv6Cidrs");
    add(errors, network.directOriginInboundAllowed === false, "direct-origin-access-forbidden");
    add(errors, network.alb === false, "alb-forbidden");
    add(errors, network.awsWaf === false, "aws-waf-forbidden");
  }

  const database = decision.database;
  add(errors, database && typeof database === "object", "missing-owner-input:database");
  if (database && typeof database === "object") {
    add(errors, database.engine === "postgresql", "database-engine-must-be-postgresql");
    add(errors, database.instanceType === "db.t4g.micro", "database-instance-must-be-db-t4g-micro");
    add(errors, database.instances === 1, "exactly-one-database-instance-required");
    add(errors, database.multiAz === false, "multi-az-rds-forbidden");
    add(errors, database.publiclyAccessible === false, "public-rds-forbidden");
    add(errors, database.encrypted === true, "database-encryption-required");
    exactArray(database.logicalDatabases, ["aviasurveil360", "keycloak"], "logical-database-contract-mismatch", errors);
    exactArray(database.runtimeRoles, ["aviasurveil360_runtime", "keycloak_runtime"], "database-role-contract-mismatch", errors);
    add(errors, database.bootstrapAuthorityRetained === false, "database-bootstrap-authority-must-be-removed");
    add(errors, database.pitrDays === 14, "database-pitr-must-be-14-days");
  }

  const storage = decision.objectStorage;
  add(errors, storage && typeof storage === "object", "missing-owner-input:objectStorage");
  if (storage && typeof storage === "object") {
    add(errors, storage.provider === "aws-s3-instance-profile", "object-store-must-use-instance-profile");
    add(errors, storage.staticCredentials === false, "static-s3-credentials-forbidden");
    add(errors, storage.publicAccess === false, "public-s3-forbidden");
    add(errors, storage.versioning === true, "s3-versioning-required");
    add(errors, storage.encrypted === true, "s3-encryption-required");
    add(errors, storage.exactVersionIdentity === true, "exact-version-identity-required");
    add(errors, storage.scanner === "guardduty-s3", "managed-scan-gate-required");
    add(errors, storage.cleanStatus === "NO_THREATS_FOUND", "clean-status-mismatch");
    add(errors, storage.resultTagging === true, "guardduty-result-tagging-required");
    add(errors, storage.productionMinio === false && storage.productionClamav === false, "local-object-or-scanner-fallback-forbidden");
  }
}

export function validateDecision(decision, { now = Date.now(), maxPricingAgeDays = 30 } = {}) {
  const errors = [];
  add(errors, decision && typeof decision === "object" && !Array.isArray(decision), "invalid-json:root-object");
  if (!decision || typeof decision !== "object" || Array.isArray(decision)) return errors;
  add(errors, decision.schemaVersion === 1, "unsupported-schema-version");
  requiredString(decision.decisionId, "decisionId", errors);
  const issuedAt = parseDateTime(decision.issuedAt, "issuedAt", errors);
  const expiresAt = parseDateTime(decision.expiresAt, "expiresAt", errors);
  if (issuedAt !== null) add(errors, issuedAt <= now, "future-decision:issuedAt");
  if (expiresAt !== null) add(errors, expiresAt > now, "stale-decision:expiresAt");
  if (issuedAt !== null && expiresAt !== null) add(errors, issuedAt < expiresAt, "contradictory-decision-window");

  const authorization = decision.authorization;
  add(errors, authorization && typeof authorization === "object", "missing-owner-input:authorization");
  if (authorization && typeof authorization === "object") {
    add(errors, authorization.scope === "local-preparation-only", "authorization-scope-must-remain-local");
    add(errors, authorization.remoteActionsAuthorized === false, "remote-authority-forbidden");
    add(errors, authorization.productionReleaseAuthorized === false, "release-authority-not-granted");
  }

  const aws = decision.aws;
  add(errors, aws && typeof aws === "object", "missing-owner-input:aws");
  if (aws && typeof aws === "object") {
    add(errors, /^\d{12}$/u.test(aws.accountId ?? ""), "missing-owner-input:aws.accountId");
    add(errors, aws.operatorProfile === "avia", "aws-operator-profile-must-be-avia");
    add(errors, /^[a-z]{2}(?:-gov)?-[a-z]+-\d$/u.test(aws.region ?? ""), "missing-owner-input:aws.region");
    requiredString(aws.workloadAvailabilityZone, "aws.workloadAvailabilityZone", errors);
    requiredString(aws.secondaryAvailabilityZone, "aws.secondaryAvailabilityZone", errors);
    add(errors, availabilityZoneBelongsToRegion(aws.workloadAvailabilityZone, aws.region), "workload-availability-zone-region-mismatch");
    add(errors, availabilityZoneBelongsToRegion(aws.secondaryAvailabilityZone, aws.region), "secondary-availability-zone-region-mismatch");
    add(errors, aws.workloadAvailabilityZone !== aws.secondaryAvailabilityZone, "availability-zones-must-differ");
    requiredString(aws.operatorRoleArn, "aws.operatorRoleArn", errors);
    add(errors, !aws.operatorRoleArn || aws.operatorRoleArn.includes(`::${aws.accountId}:`), "operator-role-account-mismatch");
  }

  const edge = decision.cloudflare;
  add(errors, edge && typeof edge === "object", "missing-owner-input:cloudflare");
  if (edge && typeof edge === "object") {
    requiredString(edge.accountId, "cloudflare.accountId", errors);
    requiredString(edge.zoneId, "cloudflare.zoneId", errors);
    requiredString(edge.hostname, "cloudflare.hostname", errors);
    requiredString(edge.tunnelName, "cloudflare.tunnelName", errors);
    requiredString(edge.connectorParameterArn, "cloudflare.connectorParameterArn", errors);
    add(errors, arnBelongsTo(edge.connectorParameterArn, { service: "ssm", region: aws?.region, accountId: aws?.accountId, resourcePrefix: "parameter/" }), "connector-parameter-owner-mismatch");
    exactArray(edge.apiTokenPermissions, ["account:cloudflare-tunnel:edit", "zone:zone:read", "zone:dns:edit"], "cloudflare-api-token-permissions-mismatch", errors);
    add(errors, edge.dnsCutoverAuthorized === false, "cloudflare-dns-cutover-requires-separate-authorization");
    add(errors, edge.connectorTokenInTerraformState === false, "connector-token-in-terraform-state-forbidden");
    requiredString(edge.connectorTokenRotationOwner, "cloudflare.connectorTokenRotationOwner", errors);
  }

  const smtp = decision.smtp;
  add(errors, smtp && typeof smtp === "object", "missing-owner-input:smtp");
  if (smtp && typeof smtp === "object") {
    requiredString(smtp.provider, "smtp.provider", errors);
    requiredString(smtp.hostname, "smtp.hostname", errors);
    requiredPositiveNumber(smtp.port, "smtp.port", errors);
    add(errors, ["implicit-tls", "starttls"].includes(smtp.transport), "smtp-encrypted-transport-required");
    requiredString(smtp.tlsServerName, "smtp.tlsServerName", errors);
    requiredString(smtp.sender, "smtp.sender", errors);
    requiredString(smtp.credentialSecretArn, "smtp.credentialSecretArn", errors);
    add(errors, arnBelongsTo(smtp.credentialSecretArn, { service: "secretsmanager", region: aws?.region, accountId: aws?.accountId, resourcePrefix: "secret:" }), "smtp-secret-owner-mismatch");
    requiredString(smtp.bounceOwner, "smtp.bounceOwner", errors);
    requiredString(smtp.rotationOwner, "smtp.rotationOwner", errors);
    requiredString(smtp.incidentContact, "smtp.incidentContact", errors);
    requiredPositiveNumber(smtp.hourlyQuota, "smtp.hourlyQuota", errors);
    add(errors, smtp.ipv6Required === true, "smtp-ipv6-required");
    add(errors, smtp.dnsAAAAVerified === true, "smtp-aaaa-verification-required");
    add(errors, smtp.tlsIpv6PreflightVerified === true, "smtp-ipv6-tls-preflight-required");
    add(errors, validIpv6Cidrs(smtp.ipv6Cidrs), "missing-owner-input:smtp.ipv6Cidrs");
    add(errors, smtp.ses === false && smtp.plaintextFallback === false, "smtp-public-plaintext-or-ses-forbidden");
  }

  validateAcceptedTopology(decision, errors);

  const recovery = decision.recovery;
  add(errors, recovery && typeof recovery === "object", "missing-owner-input:recovery");
  if (recovery && typeof recovery === "object") {
    add(errors, recovery.rpoMinutes <= 15 && recovery.rpoMinutes > 0, "recovery-rpo-must-be-at-most-15-minutes");
    add(errors, recovery.rtoMinutes <= 240 && recovery.rtoMinutes > 0, "recovery-rto-must-be-at-most-240-minutes");
    add(errors, recovery.awsBackupRetentionDays === 35, "aws-backup-retention-must-be-35-days");
    add(errors, recovery.cloudWatchLogRetentionDays === 30, "cloudwatch-retention-must-be-30-days");
    requiredString(recovery.restoreOwner, "recovery.restoreOwner", errors);
    requiredString(recovery.coordinatedDatabaseRecoveryOwner, "recovery.coordinatedDatabaseRecoveryOwner", errors);
  }

  const records = decision.records;
  add(errors, records && typeof records === "object", "missing-owner-input:records");
  if (records && typeof records === "object") {
    requiredString(records.retentionOwner, "records.retentionOwner", errors);
    requiredString(records.legalHoldOwner, "records.legalHoldOwner", errors);
    requiredString(records.deletionAuthority, "records.deletionAuthority", errors);
    add(errors, records.automaticEvidenceDeletion === false, "automatic-evidence-deletion-forbidden");
    add(errors, records.automaticAuditHistoryDeletion === false, "automatic-audit-history-deletion-forbidden");
  }

  const pricing = decision.pricingEvidence;
  add(errors, pricing && typeof pricing === "object", "missing-owner-input:pricingEvidence");
  if (pricing && typeof pricing === "object") {
    const checkedOn = parseDate(pricing.checkedOn, "pricingEvidence.checkedOn", errors);
    if (checkedOn !== null) {
      add(errors, checkedOn <= now, "pricing-evidence-from-future");
      add(errors, now - checkedOn <= maxPricingAgeDays * DAY_MS, "expired-t4g-pricing-evidence");
    }
    add(errors, pricing.t4gEligible === true, "missing-owner-input:pricingEvidence.t4gEligible");
    requiredPositiveNumber(pricing.t4gOnDemandUsdPerHour, "pricingEvidence.t4gOnDemandUsdPerHour", errors);
    requiredPositiveNumber(pricing.estimatedMonthlyUsd, "pricingEvidence.estimatedMonthlyUsd", errors);
  }

  const cost = decision.cost;
  add(errors, cost && typeof cost === "object", "missing-owner-input:cost");
  if (cost && typeof cost === "object") {
    requiredPositiveNumber(cost.monthlyCeilingUsd, "cost.monthlyCeilingUsd", errors);
    requiredPositiveNumber(cost.oneRunCeilingUsd, "cost.oneRunCeilingUsd", errors);
    add(errors, cost.estimatedMonthlyUsd <= cost.monthlyCeilingUsd, "over-budget:monthly");
    add(errors, cost.estimatedOneRunUsd <= cost.oneRunCeilingUsd, "over-budget:one-run");
    add(errors, Array.isArray(cost.alertRecipients) && cost.alertRecipients.length > 0, "missing-owner-input:cost.alertRecipients");
  }

  const release = decision.release;
  add(errors, release && typeof release === "object", "missing-owner-input:release");
  if (release && typeof release === "object") {
    requiredString(release.preprodDisposition, "release.preprodDisposition", errors);
    requiredString(release.releaseAuthority, "release.releaseAuthority", errors);
    requiredString(release.rollbackOwner, "release.rollbackOwner", errors);
    requiredString(release.rollbackPredecessorManifest, "release.rollbackPredecessorManifest", errors);
    requiredString(release.windowStart, "release.windowStart", errors);
    requiredString(release.windowEnd, "release.windowEnd", errors);
    requiredString(release.retainOrDestroyDecision, "release.retainOrDestroyDecision", errors);
  }

  const owners = decision.owners;
  add(errors, owners && typeof owners === "object", "missing-owner-input:owners");
  if (owners && typeof owners === "object") {
    for (const owner of REQUIRED_OWNERS) requiredString(owners[owner], `owners.${owner}`, errors);
  }

  const capacity = decision.capacity;
  add(errors, capacity && typeof capacity === "object", "missing-owner-input:capacity");
  if (capacity && typeof capacity === "object") {
    add(errors, capacity.minimumHostMemoryHeadroomPercent >= 15, "capacity-headroom-must-be-at-least-15-percent");
    add(errors, capacity.maximumDiskUsePercent <= 70, "capacity-disk-use-must-be-at-most-70-percent");
    add(errors, capacity.sustainedSwapAllowed === false, "sustained-swap-forbidden");
    add(errors, capacity.failureDecision === "NO-GO for t4g.small", "capacity-failure-must-be-literal-no-go");
  }

  scanForSecretValues(decision, "", errors);
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
    const parent = lstatSync(path.dirname(filePath));
    if (!parent.isDirectory() || (parent.mode & 0o077) !== 0) errors.push("artifact-permission:decision-directory-must-be-0700-or-stricter");
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

if (process.argv[1] && path.resolve(process.argv[1]) === path.resolve(new URL(import.meta.url).pathname)) {
  const filePath = process.argv[2];
  if (!filePath) {
    console.error("usage: aws-private-pilot-decision.mjs <decision.json>");
    process.exitCode = 64;
  } else {
    const result = validateDecisionFile(filePath);
    if (result.errors.length > 0) {
      result.errors.forEach((error) => console.error(error));
      process.exitCode = 65;
    } else {
      console.log("verified locally: aws-private-pilot decision contract");
    }
  }
}
