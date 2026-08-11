import assert from "node:assert/strict";
import { chmodSync, existsSync, mkdtempSync, readFileSync, writeFileSync } from "node:fs";
import os from "node:os";
import path from "node:path";
import { spawnSync } from "node:child_process";
import test from "node:test";
import { fileURLToPath } from "node:url";

import { validateDecision } from "../scripts/lib/aws-private-pilot-decision.mjs";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const checker = path.join(root, "scripts/check-aws-private-pilot-decisions.sh");

function validDecision(now = Date.now()) {
  const iso = (days) => new Date(now + days * 86400000).toISOString();
  const digest = (name, character) => `111122223333.dkr-ecr.eu-central-1.on.aws/${name}@sha256:${character.repeat(64)}`;
  return {
    schemaVersion: 1,
    decisionId: "private-pilot-test-001",
    issuedAt: iso(-1),
    expiresAt: iso(7),
    authorization: { scope: "local-preparation-only", remoteActionsAuthorized: false, productionReleaseAuthorized: false },
    aws: {
      accountId: "111122223333",
      operatorProfile: "avia",
      operatorRoleArn: "arn:aws:iam::111122223333:role/AviaPrivatePilotPlanner",
      region: "eu-central-1",
      workloadAvailabilityZone: "eu-central-1a",
      secondaryAvailabilityZone: "eu-central-1b",
    },
    cloudflare: {
      accountId: "fake-cloudflare-account",
      zoneId: "fake-cloudflare-zone",
      hostname: "pilot.example.invalid",
      tunnelName: "avia-private-pilot",
      connectorParameterArn: "arn:aws:ssm:eu-central-1:111122223333:parameter/avia/private-pilot/cloudflare-tunnel-token",
      apiTokenPermissions: ["account:cloudflare-tunnel:edit", "zone:zone:read", "zone:dns:edit"],
      dnsCutoverAuthorized: false,
      connectorTokenInTerraformState: false,
      connectorTokenRotationOwner: "security@example.invalid",
    },
    smtp: {
      provider: "fixture-relay",
      hostname: "smtp.example.invalid",
      port: 587,
      transport: "starttls",
      tlsServerName: "smtp.example.invalid",
      sender: "no-reply@example.invalid",
      credentialSecretArn: "arn:aws:secretsmanager:eu-central-1:111122223333:secret:private-pilot/smtp",
      hourlyQuota: 100,
      ipv6Required: true,
      dnsAAAAVerified: true,
      tlsIpv6PreflightVerified: true,
      ipv6Cidrs: ["2001:db8:100::25/128"],
      bounceOwner: "mail-owner@example.invalid",
      rotationOwner: "security@example.invalid",
      incidentContact: "oncall@example.invalid",
      ses: false,
      plaintextFallback: false,
    },
    runtime: {
      profile: "aws-private-pilot",
      architecture: "linux/arm64",
      instanceType: "t4g.small",
      services: ["gateway", "api", "worker", "keycloak"],
      forbiddenServices: ["postgres", "keycloak-postgres", "minio", "mailpit", "clamav", "backup-minio", "prometheus", "grafana", "loki", "tempo", "alertmanager", "fixture", "loader", "volume-init"],
      imageDigests: {
        cloudflared: digest("cloudflared", "0"), gateway: digest("gateway", "a"), api: digest("api", "c"), worker: digest("worker", "d"), keycloak: digest("keycloak", "1"),
      },
      composeSupervisor: "systemd", edgeConnector: "cloudflared-systemd-container", gatewayHostPorts: 1, reminderController: "worker",
      nonRoot: true, noNewPrivileges: true, dropAllCapabilities: true, dockerSocketMounted: false, amd64: false, emulation: false,
    },
    network: {
      workloadAvailabilityZones: 1, publicSubnets: 0, dualStackAppSubnets: 1, privateDatabaseSubnets: 2,
      natGateways: 0, internetGateways: 0, egressOnlyInternetGateways: 1, s3GatewayEndpoints: 1, interfaceEndpoints: 0,
      ec2PublicIpv4: false, ec2GlobalIpv6: true, runtimeSecurityGroupIngressRules: 0, databaseSecurityGroupIngressRules: 1, ipv4DefaultRoute: false,
      ipv6DefaultRoute: "egress-only-internet-gateway", origin: "cloudflare-tunnel-ipv6-loopback", gatewayBindAddress: "127.0.0.1",
      cloudflareTunnelEdgeIpVersion: 6, cloudflareTunnelPort: 7844,
      cloudflareTunnelIpv6Cidrs: ["2606:4700:a0::/48", "2606:4700:a8::/48"],
      directOriginInboundAllowed: false, alb: false, awsWaf: false,
    },
    database: {
      engine: "postgresql", instanceType: "db.t4g.micro", instances: 1, multiAz: false, publiclyAccessible: false,
      encrypted: true, logicalDatabases: ["aviasurveil360", "keycloak"], runtimeRoles: ["aviasurveil360_runtime", "keycloak_runtime"],
      bootstrapAuthorityRetained: false, pitrDays: 14,
    },
    objectStorage: {
      provider: "aws-s3-instance-profile", staticCredentials: false, publicAccess: false, versioning: true, encrypted: true,
      exactVersionIdentity: true, scanner: "guardduty-s3", cleanStatus: "NO_THREATS_FOUND", resultTagging: true,
      productionMinio: false, productionClamav: false,
    },
    recovery: {
      rpoMinutes: 15, rtoMinutes: 240, awsBackupRetentionDays: 35, cloudWatchLogRetentionDays: 30,
      restoreOwner: "recovery@example.invalid", coordinatedDatabaseRecoveryOwner: "recovery@example.invalid",
    },
    records: {
      retentionOwner: "records@example.invalid", legalHoldOwner: "legal@example.invalid", deletionAuthority: "records@example.invalid",
      automaticEvidenceDeletion: false, automaticAuditHistoryDeletion: false,
    },
    pricingEvidence: { checkedOn: new Date(now - 86400000).toISOString().slice(0, 10), t4gEligible: true, t4gOnDemandUsdPerHour: 0.02, estimatedMonthlyUsd: 125 },
    cost: { monthlyCeilingUsd: 200, oneRunCeilingUsd: 50, estimatedMonthlyUsd: 125, estimatedOneRunUsd: 20, alertRecipients: ["cost@example.invalid"] },
    release: {
      preprodDisposition: "accepted-for-fixture", releaseAuthority: "release@example.invalid", rollbackOwner: "rollback@example.invalid",
      rollbackPredecessorManifest: "sha256:" + "9".repeat(64), windowStart: iso(2), windowEnd: iso(3), retainOrDestroyDecision: "retain-until-expiry",
    },
    owners: {
      platform: "platform@example.invalid", security: "security@example.invalid", identity: "identity@example.invalid",
      recordsLegal: "records@example.invalid", recovery: "recovery@example.invalid", monitoringOnCall: "oncall@example.invalid",
      cost: "cost@example.invalid", release: "release@example.invalid", rollback: "rollback@example.invalid",
    },
    capacity: { minimumHostMemoryHeadroomPercent: 15, maximumDiskUsePercent: 70, sustainedSwapAllowed: false, failureDecision: "NO-GO for t4g.small" },
  };
}

function writePrivate(decision) {
  const directory = mkdtempSync(path.join(os.tmpdir(), `avia-private-pilot-${process.pid}-`));
  chmodSync(directory, 0o700);
  const file = path.join(directory, "decision.json");
  writeFileSync(file, JSON.stringify(decision));
  chmodSync(file, 0o600);
  return file;
}

test("committed contract is non-deployable and names the accepted boundary", () => {
  const decisions = readFileSync(path.join(root, "docs/operations/AWS_PRIVATE_PILOT_DECISIONS.md"), "utf8");
  const runtime = readFileSync(path.join(root, "docs/operations/AWS_PRIVATE_PILOT_RUNTIME.md"), "utf8");
  assert.equal(existsSync(checker), true);
  assert.match(decisions, /Cloudflare Tunnel/u);
  assert.match(decisions, /production-ready: not established/u);
  assert.match(runtime, /NO-GO for t4g\.small/u);
  assert.doesNotMatch(decisions, /\b\d{12}\b/u);
  assert.doesNotMatch(decisions, /-----BEGIN/u);
});

test("complete private fixture passes entirely offline", () => {
  const decision = validDecision();
  assert.deepEqual(validateDecision(decision), []);
  const result = spawnSync(checker, [writePrivate(decision)], { cwd: root, encoding: "utf8", env: { ...process.env, NODE_BIN: process.execPath } });
  assert.equal(result.status, 0, result.stderr);
  assert.match(result.stdout, /verified locally/u);
});

test("missing owner overlay reports exact missing-owner-input", () => {
  const result = spawnSync(checker, [path.join(os.tmpdir(), "missing-avia-private-pilot.json")], { cwd: root, encoding: "utf8" });
  assert.notEqual(result.status, 0);
  assert.match(result.stderr, /missing-owner-input/u);
});

test("unsafe and contradictory fixtures fail closed", async (t) => {
  const cases = {
    "remote-authority": [(d) => { d.authorization.remoteActionsAuthorized = true; }, /remote-authority-forbidden/u],
    "default-aws-profile": [(d) => { d.aws.operatorProfile = "default"; }, /aws-operator-profile-must-be-avia/u],
    "malformed-region": [(d) => { d.aws.region = "["; }, /aws.region/u],
    "cross-region-zone": [(d) => { d.aws.workloadAvailabilityZone = "eu-west-1a"; }, /availability-zone-region-mismatch/u],
    "cross-account-connector-parameter": [(d) => { d.cloudflare.connectorParameterArn = d.cloudflare.connectorParameterArn.replace("111122223333", "444455556666"); }, /connector-parameter-owner-mismatch/u],
    "external-runtime-image": [(d) => { d.runtime.imageDigests.api = `registry.example.invalid/api@sha256:${"a".repeat(64)}`; }, /runtime-image-must-use-target-ecr/u],
    "second-host": [(d) => { d.network.workloadAvailabilityZones = 2; }, /single-az/u],
    alb: [(d) => { d.network.alb = true; }, /alb-forbidden/u],
    nat: [(d) => { d.network.natGateways = 1; }, /nat-gateway-forbidden/u],
    "public-subnet": [(d) => { d.network.publicSubnets = 2; }, /public-subnets-forbidden/u],
    "ipv4-default-route": [(d) => { d.network.ipv4DefaultRoute = true; }, /ipv4-default-route-forbidden/u],
    ingress: [(d) => { d.network.runtimeSecurityGroupIngressRules = 1; }, /runtime-security-group-ingress-forbidden/u],
    "database-ingress-creep": [(d) => { d.network.databaseSecurityGroupIngressRules = 2; }, /database-security-group-ingress-must-be-bounded/u],
    "wrong-edge-ip-version": [(d) => { d.network.cloudflareTunnelEdgeIpVersion = 4; }, /edge-ip-version/u],
    "smtp-no-aaaa": [(d) => { d.smtp.dnsAAAAVerified = false; }, /smtp-aaaa/u],
    "multi-az-rds": [(d) => { d.database.multiAz = true; }, /multi-az-rds/u],
    amd64: [(d) => { d.runtime.amd64 = true; }, /amd64/u],
    minio: [(d) => { d.objectStorage.productionMinio = true; }, /fallback/u],
    "static-key": [(d) => { d.objectStorage.accessKey = "AKIA" + "A".repeat(16); }, /secret/u],
    "missing-scan": [(d) => { d.objectStorage.scanner = "none"; }, /managed-scan/u],
    "public-rds": [(d) => { d.database.publiclyAccessible = true; }, /public-rds/u],
    "interface-sprawl": [(d) => { d.network.interfaceEndpoints = 5; }, /interface-endpoint/u],
    waf: [(d) => { d.network.awsWaf = true; }, /aws-waf/u],
    ses: [(d) => { d.smtp.ses = true; }, /ses/u],
    "plaintext-smtp": [(d) => { d.smtp.transport = "plaintext"; }, /encrypted-transport/u],
    "connector-token-in-state": [(d) => { d.cloudflare.connectorTokenInTerraformState = true; }, /terraform-state-forbidden/u],
    "premature-dns-cutover": [(d) => { d.cloudflare.dnsCutoverAuthorized = true; }, /dns-cutover-requires-separate-authorization/u],
    "over-budget": [(d) => { d.cost.estimatedMonthlyUsd = 999; }, /over-budget/u],
    "expired-pricing": [(d) => { d.pricingEvidence.checkedOn = "2020-01-01"; }, /expired-t4g/u],
  };
  for (const [name, [mutate, expected]] of Object.entries(cases)) {
    await t.test(name, () => {
      const decision = validDecision();
      mutate(decision);
      const errors = validateDecision(decision);
      assert.match(errors.join("\n"), expected);
    });
  }
});

test("world-readable decision files fail before they can be used", () => {
  const file = writePrivate(validDecision());
  chmodSync(file, 0o644);
  const result = spawnSync(checker, [file], { cwd: root, encoding: "utf8" });
  assert.notEqual(result.status, 0);
  assert.match(result.stderr, /artifact-permission/u);
});
