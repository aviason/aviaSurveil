import assert from "node:assert/strict";
import { existsSync, readFileSync, writeFileSync } from "node:fs";
import path from "node:path";
import { spawnSync } from "node:child_process";
import test from "node:test";
import { fileURLToPath } from "node:url";

import { safeTerraformPlan } from "./helpers/aws-trial-fixture.mjs";

const repositoryRoot = path.resolve(
  path.dirname(fileURLToPath(import.meta.url)),
  "..",
);
const policyPath = path.join(
  repositoryRoot,
  "infra",
  "policies",
  "aws-plan.rego",
);
const opaBin = process.env.OPA_BIN ??
  "/private/tmp/aviasurveil360-tools/bin/opa";

function readRequired(filePath) {
  assert.equal(
    existsSync(filePath),
    true,
    `${path.relative(repositoryRoot, filePath)} must exist`,
  );
  return readFileSync(filePath, "utf8");
}

function evaluate(plan) {
  const result = spawnSync(
    opaBin,
    [
      "eval",
      "--format=json",
      "--data",
      policyPath,
      "--stdin-input",
      "data.aviasurveil360.aws_plan.deny",
    ],
    {
      cwd: repositoryRoot,
      encoding: "utf8",
      input: JSON.stringify(plan),
    },
  );
  assert.equal(result.status, 0, result.stderr);
  const response = JSON.parse(result.stdout);
  return response.result[0]?.expressions[0]?.value ?? [];
}

test("AWS plan policy and decision authority files exist", () => {
  const policy = readRequired(policyPath);
  assert.match(policy, /package aviasurveil360\.aws_plan/);
  assert.match(policy, /\bdeny\b/);
  readRequired(
    path.join(repositoryRoot, "docs", "operations", "AWS_TRIAL_DECISIONS.md"),
  );
  readRequired(
    path.join(repositoryRoot, "docs", "operations", "AWS_TRIAL_RUNBOOK.md"),
  );
});

test("safe AWS fixture plan has no policy denials", () => {
  assert.deepEqual(evaluate(safeTerraformPlan()), []);
});

test("AWS policy denies every unsafe plan mutation", async (t) => {
  const mutations = {
    "destroyable database": (plan) => {
      resource(plan, "aws_db_instance.application").change.after
        .deletion_protection = false;
    },
    "HTTP load balancer": (plan) => {
      resource(plan, "aws_lb_listener.https").change.after.protocol = "HTTP";
      resource(plan, "aws_lb_listener.https").change.after.port = 80;
    },
    "missing required owner tag": (plan) => {
      delete resource(plan, "aws_db_instance.application").change.after.tags.Owner;
    },
    "mutable image identity": (plan) => {
      delete resource(plan, "aws_launch_template.runtime").change.after.tags
        .ImageDigest;
    },
    "public application ingress": (plan) => {
      resource(
        plan,
        "aws_vpc_security_group_ingress_rule.runtime_from_alb",
      ).change.after.cidr_ipv4 = "0.0.0.0/0";
      delete resource(
        plan,
        "aws_vpc_security_group_ingress_rule.runtime_from_alb",
      ).change.after.referenced_security_group_id;
    },
    "public database": (plan) => {
      resource(plan, "aws_db_instance.application").change.after
        .publicly_accessible = true;
    },
    "public object storage": (plan) => {
      resource(
        plan,
        "aws_s3_bucket_public_access_block.application",
      ).change.after.block_public_policy = false;
    },
    "unencrypted database": (plan) => {
      resource(plan, "aws_db_instance.application").change.after
        .storage_encrypted = false;
    },
    "unencrypted object storage": (plan) => {
      resource(
        plan,
        "aws_s3_bucket_server_side_encryption_configuration.application",
      ).change.after.rule[0].apply_server_side_encryption_by_default[0]
        .sse_algorithm = "AES256";
    },
    "wildcard IAM action": (plan) => {
      resource(plan, "aws_iam_policy.runtime").change.after.policy =
        JSON.stringify({
          Statement: [{ Action: "*", Effect: "Allow", Resource: "*" }],
          Version: "2012-10-17",
        });
    },
  };

  for (const [name, mutate] of Object.entries(mutations)) {
    await t.test(name, () => {
      const plan = safeTerraformPlan();
      mutate(plan);
      assert.notDeepEqual(evaluate(plan), []);
    });
  }
});

test("policy source contains no fixture-specific allow switch", () => {
  const policy = readRequired(policyPath);
  assert.doesNotMatch(policy, /fixture_mode|skip_policy|allow_unsafe/i);
});

function resource(plan, address) {
  const match = plan.resource_changes.find(
    (candidate) => candidate.address === address,
  );
  assert.ok(match, `${address} fixture resource must exist`);
  return match;
}
