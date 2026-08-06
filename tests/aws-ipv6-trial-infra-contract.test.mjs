import assert from "node:assert/strict";
import { existsSync, readFileSync } from "node:fs";
import path from "node:path";
import { spawnSync } from "node:child_process";
import test from "node:test";
import { fileURLToPath } from "node:url";

const repositoryRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const terraformRoot = path.join(repositoryRoot, "infra/terraform/modules");
const policyPath = path.join(repositoryRoot, "infra/policies/aws-ipv6-trial.rego");

function read(relativePath) {
  const filePath = path.join(repositoryRoot, relativePath);
  assert.equal(existsSync(filePath), true, `${relativePath} must exist`);
  return readFileSync(filePath, "utf8");
}

test("focused IPv6 trial Terraform modules are separate from the ALB/RDS topology", () => {
  for (const module of ["ipv6-trial-network", "arm64-single-node", "cloudflare-edge-runtime-auth", "trial-budget"]) {
    assert.equal(existsSync(path.join(terraformRoot, module, "main.tf")), true);
    assert.equal(existsSync(path.join(terraformRoot, module, "versions.tf")), true);
  }
  const network = read("infra/terraform/modules/ipv6-trial-network/main.tf");
  assert.match(network, /assign_generated_ipv6_cidr_block\s*=\s*true/u);
  assert.match(network, /ipv6_native\s*=\s*true/u);
  assert.match(network, /ipv6_cidr_block\s*=\s*"::\/0"/u);
  assert.doesNotMatch(network, /resource\s+"aws_vpc_security_group_ingress_rule"/u);
  assert.doesNotMatch(network, /cidr_block\s*=\s*"0\.0\.0\.0\/0"/u);
  assert.doesNotMatch(network, /aws_nat_gateway|aws_eip|aws_lb|aws_db_instance|aws_vpc_endpoint/u);
  assert.equal(existsSync(path.join(repositoryRoot, "infra/terraform/tests/aws-ipv6-trial.tftest.hcl")), true);
});

test("ARM64 single-node module is exactly t4g.small, IPv6-only, IMDSv2/IPv6, SSM-only, and digest-bound", () => {
  const compute = read("infra/terraform/modules/arm64-single-node/main.tf");
  assert.match(compute, /instance_type\s*==\s*"t4g\.small"/u);
  assert.match(compute, /resource\s+"aws_instance"\s+"runtime"/u);
  assert.match(compute, /ipv6_address_count\s*=\s*1/u);
  assert.match(compute, /associate_public_ip_address\s*=\s*false/u);
  assert.match(compute, /http_tokens\s*=\s*"required"/u);
  assert.match(compute, /http_protocol_ipv6\s*=\s*"enabled"/u);
  assert.match(compute, /volume_type\s*=\s*"gp3"/u);
  assert.match(compute, /encrypted\s*=\s*true/u);
  assert.doesNotMatch(compute, /key_name\s*=/u);
  assert.match(compute, /ecr\.\$\{var\.region\}\.api\.aws/u);
  assert.match(compute, /dkr-ecr/u);
  assert.match(compute, /\.on\.aws/u);
  assert.match(compute, /ssm get-parameter[\s\S]*with-decryption/u);
  assert.match(compute, /TUNNEL_EDGE_IP_VERSION=6/u);
  assert.match(compute, /cloudflared_image/u);
  assert.match(compute, /disable --now sshd/u);
  assert.match(compute, /SessionManagerControlPlane/u);
  assert.doesNotMatch(compute, /SECRET_VALUE|PASSWORD=[^$]|-----BEGIN/u);
});

test("Cloudflare module binds tunnel, DNS, Access, and SecureString token in one state boundary", () => {
  const edge = read("infra/terraform/modules/cloudflare-edge-runtime-auth/main.tf");
  assert.match(edge, /cloudflare_zero_trust_tunnel_cloudflared/u);
  assert.match(edge, /cloudflare_zero_trust_tunnel_cloudflared_token/u);
  assert.match(edge, /cloudflare_zero_trust_tunnel_cloudflared_config/u);
  assert.match(edge, /cloudflare_dns_record/u);
  assert.match(edge, /cloudflare_zero_trust_access_application/u);
  assert.match(edge, /cloudflare_zero_trust_access_policy/u);
  assert.match(edge, /resource\s+"aws_ssm_parameter"\s+"connector"/u);
  assert.match(edge, /type\s*=\s*"SecureString"/u);
  assert.match(edge, /value\s*=\s*data\.cloudflare_zero_trust_tunnel_cloudflared_token/u);
  assert.doesNotMatch(edge, /no_tls_verify|noTLSVerify/u);
  assert.doesNotMatch(edge, /cloudflare_api_token\s*=/u);
});

test("budget module has bounded estimates, expiry, and alert thresholds", () => {
  const budget = read("infra/terraform/modules/trial-budget/main.tf");
  assert.match(budget, /aws_budgets_budget/u);
  assert.match(budget, /threshold\s*=\s*80/u);
  assert.match(budget, /threshold\s*=\s*100/u);
  assert.match(budget, /trial_expiry/u);
  assert.match(budget, /estimated_monthly_usd\s*<=\s*var\.monthly_ceiling_usd/u);
  assert.match(budget, /estimated_one_run_usd\s*<=\s*var\.one_run_ceiling_usd/u);
});

test("IPv6 trial OPA policy denies the prohibited cost and security mutations", () => {
  const policy = read("infra/policies/aws-ipv6-trial.rego");
  assert.match(policy, /package aviasurveil360\.aws_ipv6_trial/u);
  for (const marker of ["aws_nat_gateway", "aws_eip", "aws_lb", "aws_db_instance", "aws_ecr_repository", "IMMUTABLE", "t4g.small", "ipv6_native", "associate_public_ip_address", "aws_vpc_security_group_ingress_rule", "aws_budgets_budget", "SecureString"]) {
    assert.match(policy, new RegExp(marker.replaceAll(".", "\\."), "u"), `${marker} policy marker missing`);
  }
  assert.doesNotMatch(policy, /allow_unsafe|fixture_mode|skip_policy/u);
  const opa = process.env.OPA_BIN ?? "/private/tmp/aviasurveil360-tools/bin/opa";
  if (!existsSync(opa)) {
    console.log("not run: OPA binary is unavailable; policy source contract verified locally");
    return;
  }
  const fixture = {
    resource_changes: [
      {
        address: "aws_instance.runtime",
        mode: "managed",
        type: "aws_instance",
        change: {
          actions: ["create"],
          after: {
            instance_type: "t4g.small",
            associate_public_ip_address: false,
            ipv6_address_count: 1,
            metadata_options: [{ http_tokens: "required", http_protocol_ipv6: "enabled" }],
            root_block_device: [{ encrypted: true, volume_type: "gp3" }],
            tags: { Environment: "fixture", Owner: "platform", CostCenter: "trial", DataClassification: "restricted", ManagedBy: "terraform" },
          },
        },
      },
    ],
  };
  const result = spawnSync(opa, ["eval", "--format=json", "--data", policyPath, "--stdin-input", "data.aviasurveil360.aws_ipv6_trial.deny"], { cwd: repositoryRoot, encoding: "utf8", input: JSON.stringify(fixture) });
  assert.equal(result.status, 0, result.stderr);
  const response = JSON.parse(result.stdout);
  assert.deepEqual(response.result?.[0]?.expressions?.[0]?.value ?? [], []);
});
