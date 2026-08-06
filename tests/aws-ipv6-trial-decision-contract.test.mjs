import assert from "node:assert/strict";
import { chmodSync, existsSync, mkdtempSync, readFileSync, writeFileSync } from "node:fs";
import os from "node:os";
import path from "node:path";
import { spawnSync } from "node:child_process";
import test from "node:test";
import { fileURLToPath } from "node:url";

import { validateDecision } from "../scripts/lib/aws-ipv6-trial-decision.mjs";

const repositoryRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const docsPath = path.join(repositoryRoot, "docs/operations/AWS_IPV6_TRIAL_DECISIONS.md");
const checkerPath = path.join(repositoryRoot, "scripts/check-aws-ipv6-trial-decisions.sh");

function validDecision(now = Date.now()) {
  const iso = (offsetDays) => new Date(now + offsetDays * 86400000).toISOString();
  return {
    schemaVersion: 1,
    decisionId: "ipv6-trial-test-001",
    issuedAt: iso(-1),
    expiresAt: iso(7),
    aws: {
      accountId: "111122223333",
      operatorRoleArn: "arn:aws:iam::111122223333:role/AviaIpv6TrialPlanner",
      region: "eu-central-1",
      availabilityZone: "eu-central-1a",
    },
    cloudflare: {
      accountId: "a".repeat(32),
      zoneId: "b".repeat(32),
      hostname: "demo.example.invalid",
      apiTokenEnv: "CLOUDFLARE_API_TOKEN",
      connectorTokenParameterName: "/aviasurveil360/aws-ipv6-trial/cloudflare-connector",
      edgeIpVersion: 6,
    },
    access: {
      publicExposure: false,
      allowedIdentities: ["owner@example.invalid"],
      allowedDomains: [],
    },
    runtime: {
      profile: "demo",
      services: ["cloudflared", "gateway", "web-demo"],
      architecture: "linux/arm64",
      instanceType: "t4g.small",
      amiSsmParameterName: "/aws/service/ami-amazon-linux-latest/al2023-ami-kernel-default-arm64",
      imageDigests: {
        cloudflared: "cloudflare/cloudflared@sha256:" + "a".repeat(64),
        gateway: "111122223333.dkr.ecr.eu-central-1.on.aws/gateway@sha256:" + "b".repeat(64),
        "web-demo": "111122223333.dkr.ecr.eu-central-1.on.aws/web-demo@sha256:" + "c".repeat(64),
      },
    },
    network: {
      ipv6OnlyRuntime: true,
      publicIpv4: false,
      elasticIp: false,
      natGateway: false,
      loadBalancer: false,
      rds: false,
      interfaceVpcEndpoints: false,
      inboundSecurityGroupRules: false,
      ssh: false,
      amd64: false,
      emulation: false,
    },
    pricingEvidence: {
      checkedOn: new Date(now - 86400000).toISOString().slice(0, 10),
      eligibleRegions: ["eu-central-1"],
      trialHoursPerMonth: 750,
      offerExpiresOn: "2026-12-31",
      cpuCreditCaveat: true,
      onDemandPriceAfterExpiryUsdPerHour: 0.02,
      nonComputeMonthlyEstimateUsd: 7,
    },
    cost: {
      monthlyCeilingUsd: 25,
      oneRunCeilingUsd: 10,
      projectedMonthlyUsd: 12,
      projectedOneRunUsd: 5,
      nonComputeCeilingUsd: 12,
      alertRecipients: ["cost-owner@example.invalid"],
    },
    persistence: {
      rootVolumeGiB: 20,
      deleteRootVolumeOnTermination: true,
      snapshotRetained: false,
    },
    lifecycle: {
      trialExpiresAt: iso(5),
      retainOrDestroy: "destroy-after-trial",
    },
    owners: {
      platform: "platform@example.invalid",
      security: "security@example.invalid",
      dns: "dns@example.invalid",
      cost: "cost-owner@example.invalid",
      rollbackDestroy: "release@example.invalid",
    },
  };
}

function runChecker(filePath) {
  return spawnSync(checkerPath, [filePath], {
    cwd: repositoryRoot,
    encoding: "utf8",
    env: { ...process.env, NODE_BIN: process.execPath },
  });
}

test("Task 1 decision docs and checker exist without committed owner values", () => {
  assert.equal(existsSync(docsPath), true);
  assert.equal(existsSync(checkerPath), true);
  const docs = readFileSync(docsPath, "utf8");
  assert.match(docs, /schemaVersion: 1/u);
  assert.match(docs, /TUNNEL_EDGE_IP_VERSION=6/u);
  assert.match(docs, /production-ready: not established/u);
  assert.doesNotMatch(docs, /\b\d{12}\b/u);
  assert.doesNotMatch(docs, /-----BEGIN/u);
  assert.match(readFileSync(checkerPath, "utf8"), /missing-owner-input/u);
});

test("valid owner overlay passes offline and emits local evidence", () => {
  const directory = mkdtempSync(path.join(os.tmpdir(), "avia-ipv6-decision-test-" + process.pid + "-"));
  const filePath = path.join(directory, "decision.json");
  writeFileSync(filePath, JSON.stringify(validDecision(), null, 2));
  chmodSync(filePath, 0o600);
  assert.deepEqual(validateDecision(JSON.parse(readFileSync(filePath, "utf8"))), []);
  const result = runChecker(filePath);
  assert.equal(result.status, 0, result.stderr);
  assert.match(result.stdout, /verified locally/u);
});

test("missing, stale, contradictory, secret-bearing, and over-budget overlays fail closed", async (t) => {
  await t.test("missing file", () => {
    const result = runChecker(path.join(os.tmpdir(), "does-not-exist-avia-ipv6-decision.json"));
    assert.notEqual(result.status, 0);
    assert.match(result.stderr, /missing-owner-input/u);
  });

  const cases = {
    stale: { mutate: (decision) => { decision.expiresAt = "2020-01-01T00:00:00.000Z"; }, error: "stale" },
    contradictory: { mutate: (decision) => { decision.runtime.profile = "full"; }, error: "runtime-profile" },
    "over-budget": { mutate: (decision) => { decision.cost.projectedMonthlyUsd = 999; }, error: "over-budget" },
    "secret-bearing": { mutate: (decision) => { decision.cloudflare.apiToken = "sk-live-secret"; }, error: "secret" },
    "wrong-edge": { mutate: (decision) => { decision.cloudflare.edgeIpVersion = 4; }, error: "edge-ip-version" },
    "amd64": { mutate: (decision) => { decision.network.amd64 = true; }, error: "amd64" },
  };

  for (const [name, scenario] of Object.entries(cases)) {
    await t.test(name, () => {
      const directory = mkdtempSync(path.join(os.tmpdir(), `avia-ipv6-decision-${name}-${process.pid}-`));
      const filePath = path.join(directory, "decision.json");
      const decision = validDecision();
      scenario.mutate(decision);
      writeFileSync(filePath, JSON.stringify(decision));
      chmodSync(filePath, 0o600);
      const result = runChecker(filePath);
      assert.notEqual(result.status, 0);
      assert.match(result.stderr, new RegExp(scenario.error, "u"));
    });
  }
});

test("world-readable owner overlays are rejected before validation", () => {
  const directory = mkdtempSync(path.join(os.tmpdir(), `avia-ipv6-decision-mode-${process.pid}-`));
  const filePath = path.join(directory, "decision.json");
  writeFileSync(filePath, JSON.stringify(validDecision()));
  chmodSync(filePath, 0o644);
  const result = runChecker(filePath);
  assert.notEqual(result.status, 0);
  assert.match(result.stderr, /artifact-permission/u);
});

test("world-readable owner overlay directories are rejected", () => {
  const directory = mkdtempSync(path.join(os.tmpdir(), `avia-ipv6-decision-directory-${process.pid}-`));
  const filePath = path.join(directory, "decision.json");
  writeFileSync(filePath, JSON.stringify(validDecision()));
  chmodSync(filePath, 0o600);
  chmodSync(directory, 0o755);
  const result = runChecker(filePath);
  assert.notEqual(result.status, 0);
  assert.match(result.stderr, /decision-directory/u);
});
