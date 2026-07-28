import { createHash } from "node:crypto";
import {
  chmodSync,
  mkdtempSync,
  readFileSync,
  writeFileSync,
} from "node:fs";
import os from "node:os";
import path from "node:path";

export const wrapperNames = [
  "aws-trial-plan.sh",
  "check-aws-plan.sh",
  "aws-trial-apply.sh",
  "aws-trial-publish-artifacts.sh",
  "aws-trial-smoke.sh",
  "aws-trial-rollback.sh",
  "aws-trial-destroy.sh",
];

const requiredTags = {
  CostCenter: "trial-001",
  DataClassification: "restricted",
  Environment: "aws-trial",
  ManagedBy: "terraform",
  Owner: "platform-operations",
};

export function safeTerraformPlan() {
  return {
    format_version: "1.2",
    terraform_version: "1.15.8",
    resource_changes: [
      change("aws_db_instance.application", "aws_db_instance", {
        deletion_protection: true,
        publicly_accessible: false,
        storage_encrypted: true,
        tags: requiredTags,
      }),
      change("aws_s3_bucket.application", "aws_s3_bucket", {
        tags: requiredTags,
      }),
      change(
        "aws_s3_bucket_public_access_block.application",
        "aws_s3_bucket_public_access_block",
        {
          block_public_acls: true,
          block_public_policy: true,
          ignore_public_acls: true,
          restrict_public_buckets: true,
        },
      ),
      change(
        "aws_s3_bucket_server_side_encryption_configuration.application",
        "aws_s3_bucket_server_side_encryption_configuration",
        {
          rule: [{
            apply_server_side_encryption_by_default: [{
              kms_master_key_id:
                "arn:aws:kms:eu-central-1:111122223333:key/11111111-2222-3333-4444-555555555555",
              sse_algorithm: "aws:kms",
            }],
          }],
        },
      ),
      change("aws_lb_listener.https", "aws_lb_listener", {
        port: 443,
        protocol: "HTTPS",
        ssl_policy: "ELBSecurityPolicy-TLS13-1-2-2021-06",
        tags: requiredTags,
      }),
      change(
        "aws_vpc_security_group_ingress_rule.alb_https",
        "aws_vpc_security_group_ingress_rule",
        {
          cidr_ipv4: "0.0.0.0/0",
          from_port: 443,
          ip_protocol: "tcp",
          to_port: 443,
          tags: requiredTags,
        },
      ),
      change(
        "aws_vpc_security_group_ingress_rule.runtime_from_alb",
        "aws_vpc_security_group_ingress_rule",
        {
          from_port: 8443,
          ip_protocol: "tcp",
          referenced_security_group_id: "sg-00000000000000001",
          to_port: 8443,
          tags: requiredTags,
        },
      ),
      change("aws_iam_policy.runtime", "aws_iam_policy", {
        policy: JSON.stringify({
          Statement: [{
            Action: [
              "ecr:GetAuthorizationToken",
              "secretsmanager:GetSecretValue",
            ],
            Effect: "Allow",
            Resource: [
              "arn:aws:ecr:eu-central-1:111122223333:repository/avia-trial",
              "arn:aws:secretsmanager:eu-central-1:111122223333:secret:avia-trial/*",
            ],
          }],
          Version: "2012-10-17",
        }),
        tags: requiredTags,
      }),
      change("aws_launch_template.runtime", "aws_launch_template", {
        block_device_mappings: [{
          ebs: [{
            encrypted: true,
            kms_key_id:
              "arn:aws:kms:eu-central-1:111122223333:key/11111111-2222-3333-4444-555555555555",
          }],
        }],
        metadata_options: [{
          http_endpoint: "enabled",
          http_tokens: "required",
        }],
        tags: {
          ...requiredTags,
          ImageDigest:
            "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
        },
      }),
    ],
  };
}

export function createAwsTrialBundle(
  repositoryRoot,
  {
    mutateDecision,
    mutateManifest,
    mutatePlan,
  } = {},
) {
  const directory = mkdtempSync(
    path.join(os.tmpdir(), "aviasurveil360-aws-plan-"),
  );
  chmodSync(directory, 0o700);

  const now = Date.now();
  const decision = {
    accountId: "111122223333",
    approvalId: "fixture-non-deployable",
    approvedPhases: ["foundation-ecr"],
    backupRetentionDays: 7,
    budgetCeilingUsd: 250,
    capacity: {
      desired: 1,
      max: 2,
      min: 1,
    },
    certificateArn:
      "arn:aws:acm:eu-central-1:111122223333:certificate/11111111-2222-3333-4444-555555555555",
    changeWindow: {
      endsAt: new Date(now + 3_600_000).toISOString(),
      startsAt: new Date(now - 60_000).toISOString(),
    },
    dataResidencyApproved: true,
    destroyDecision: "destroy-after-trial",
    domain: "trial.invalid",
    ownerContacts: {
      platform: "platform@example.invalid",
      records: "records@example.invalid",
      release: "release@example.invalid",
      security: "security@example.invalid",
    },
    region: "eu-central-1",
    schemaVersion: 1,
  };
  mutateDecision?.(decision);

  const plan = safeTerraformPlan();
  mutatePlan?.(plan);

  const decisionPath = path.join(directory, "decision.json");
  const planBinaryPath = path.join(directory, "plan.tfplan");
  const planJsonPath = path.join(directory, "plan.json");
  writeProtectedJson(decisionPath, decision);
  writeProtectedJson(planJsonPath, plan);
  writeFileSync(planBinaryPath, "non-deployable fixture plan\n", {
    encoding: "utf8",
    mode: 0o600,
  });

  const manifest = {
    accountId: decision.accountId,
    artifactMode: "offline-fixture",
    callerArn:
      "arn:aws:iam::111122223333:role/AviaTrialReadOnlyPlanner",
    capacity: structuredClone(decision.capacity),
    cleanupAt: new Date(now + 7_200_000).toISOString(),
    commands: [{
      arguments: ["plan"],
      executable: "terragrunt",
      kind: "plan",
      unit: "network",
    }],
    costEstimateUsd: 42,
    createdAt: new Date(now).toISOString(),
    decisionSha256: sha256File(decisionPath),
    expiresAt: new Date(now + 3_600_000).toISOString(),
    images: [{
      reference:
        "111122223333.dkr.ecr.eu-central-1.amazonaws.com/avia-trial@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
      sbomPath: "runtime.cdx.json",
      sbomSha256:
        "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
      scannedAt: new Date(now).toISOString(),
      trivyCritical: 0,
      trivyHigh: 0,
    }],
    lockSha256:
      "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
    phase: "foundation-ecr",
    planBinary: {
      path: "plan.tfplan",
      sha256: sha256File(planBinaryPath),
    },
    planJson: {
      path: "plan.json",
      sha256: sha256File(planJsonPath),
    },
    policyDenials: 0,
    region: decision.region,
    retention: "ephemeral-0600",
    schemaVersion: 1,
    terraformVersion: "1.15.8",
    terragruntVersion: "1.0.4",
    wrappers: Object.fromEntries(
      wrapperNames.map((name) => [
        name,
        sha256File(path.join(repositoryRoot, "scripts", name)),
      ]),
    ),
  };
  mutateManifest?.(manifest);

  const manifestPath = path.join(directory, "manifest.json");
  writeProtectedJson(manifestPath, manifest);
  return {
    decision,
    decisionPath,
    directory,
    manifest,
    manifestPath,
    plan,
    planBinaryPath,
    planJsonPath,
  };
}

export function sha256File(filePath) {
  return createHash("sha256").update(readFileSync(filePath)).digest("hex");
}

function change(address, type, after) {
  return {
    address,
    change: {
      actions: ["create"],
      after,
      after_unknown: {},
      before: null,
    },
    mode: "managed",
    name: address.split(".").at(-1),
    provider_name: "registry.terraform.io/hashicorp/aws",
    type,
  };
}

function writeProtectedJson(filePath, value) {
  writeFileSync(filePath, `${JSON.stringify(value, null, 2)}\n`, {
    encoding: "utf8",
    mode: 0o600,
  });
}
