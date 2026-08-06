import assert from "node:assert/strict";
import { existsSync, readFileSync } from "node:fs";
import path from "node:path";
import { spawnSync } from "node:child_process";
import test from "node:test";
import { fileURLToPath } from "node:url";

const repositoryRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const envRoot = path.join(repositoryRoot, "infra/terragrunt/environments/aws-ipv6-trial");
const componentsRoot = path.join(envRoot, "components");

function read(filePath) {
  assert.equal(existsSync(filePath), true, `${path.relative(repositoryRoot, filePath)} must exist`);
  return readFileSync(filePath, "utf8");
}

test("aws-ipv6-trial Terragrunt environment is a separate five-component graph", () => {
  for (const file of ["root.hcl", "account.hcl", "environment.hcl", "region.hcl.example"]) read(path.join(envRoot, file));
  for (const component of ["network", "registry", "edge-runtime-auth", "budget", "compute"]) read(path.join(componentsRoot, component, "terragrunt.hcl"));
  const all = ["root.hcl", "account.hcl", "environment.hcl", "region.hcl.example", ...["network", "registry", "edge-runtime-auth", "budget", "compute"].map((name) => path.join("components", name, "terragrunt.hcl"))].map((file) => read(path.join(envRoot, file))).join("\n");
  assert.doesNotMatch(all, /aws-trial/u);
  assert.doesNotMatch(all, /aws_autoscaling_group|aws_nat_gateway|aws_eip|aws_lb|aws_db_instance|aws_vpc_endpoint|0\.0\.0\.0\/0|linux\/amd64/u);
  assert.match(all, /aws-ipv6-trial/u);
  const fixture = read(path.join(repositoryRoot, "infra/terragrunt/fixtures/aws-ipv6-trial-non-deployable.hcl"));
  assert.match(fixture, /fixture_mode\s*=\s*true/u);
  assert.doesNotMatch(fixture, /CLOUDFLARE_API_TOKEN|secretValue|tokenValue/u);
});

test("provider and state generation are explicit and remote-state lock remains an authorized boundary", () => {
  const root = read(path.join(envRoot, "root.hcl"));
  assert.match(root, /backend\s*=\s*local\.fixture_mode\s*\?\s*"local"\s*:\s*"s3"/u);
  assert.match(root, /encrypt\s*=\s*true/u);
  assert.match(root, /use_lockfile\s*=\s*true/u);
  assert.match(root, /AVIA_IPV6_TG_INPUTS_FILE/u);
  assert.match(root, /check-aws-ipv6-trial-terragrunt\.sh/u);
  assert.match(root, /check-aws-ipv6-trial-preflight\.sh/u);
  const preflight = read(path.join(repositoryRoot, "scripts/check-aws-ipv6-trial-preflight.sh"));
  assert.match(preflight, /fixture_mode/u);
  assert.match(preflight, /check-aws-ipv6-trial-decisions\.sh/u);
  assert.doesNotMatch(preflight, /\b(?:apply|destroy)\b/u);
  assert.match(read(path.join(repositoryRoot, "scripts/check-aws-ipv6-trial-terragrunt.sh")), /terraform show -json/u);
  assert.match(read(path.join(repositoryRoot, "scripts/check-aws-ipv6-trial-terragrunt.sh")), /aws-ipv6-trial-redaction\.mjs/u);
  const edge = read(path.join(componentsRoot, "edge-runtime-auth", "terragrunt.hcl"));
  assert.match(edge, /provider "cloudflare"/u);
  assert.match(read(path.join(componentsRoot, "registry", "terragrunt.hcl")), /toset\(\["gateway", "web-demo"\]\)/u);
  for (const component of ["network", "registry", "budget", "compute"]) assert.match(read(path.join(componentsRoot, component, "terragrunt.hcl")), /provider "aws"/u);
  assert.doesNotMatch(edge, /CLOUDFLARE_API_TOKEN\s*=/u);
});

test("dependency mocks are limited to offline validate/plan and never cross into mutable commands", () => {
  const componentFiles = ["network", "registry", "edge-runtime-auth", "budget", "compute"].map((name) => path.join(componentsRoot, name, "terragrunt.hcl"));
  const dependencies = componentFiles.map(read).join("\n");
  for (const match of dependencies.matchAll(/mock_outputs_allowed_terraform_commands\s*=\s*\[([^\]]+)\]/gu)) {
    assert.match(match[1], /"validate"/u);
    assert.match(match[1], /"plan"/u);
    assert.doesNotMatch(match[1], /"apply"|"destroy"/u);
  }
  assert.doesNotMatch(dependencies, /run\s+--all\s+(?:apply|destroy)/u);
});

test("offline Terragrunt checker fails closed when owner overlay is absent", () => {
  const checker = path.join(repositoryRoot, "scripts/check-aws-ipv6-trial-terragrunt.sh");
  const source = read(checker);
  assert.match(source, /missing-owner-input/u);
  assert.match(source, /AVIA_IPV6_TG_INPUTS_FILE/u);
  assert.match(source, /AVIA_TG_PLAN_DIR/u);
  assert.match(source, /artifact-permission/u);
  assert.doesNotMatch(source, /\b(?:apply|destroy)\b/u);
  const result = spawnSync(checker, [], { cwd: repositoryRoot, encoding: "utf8", env: { ...process.env, AVIA_IPV6_TG_INPUTS_FILE: "" } });
  assert.notEqual(result.status, 0);
  assert.match(result.stderr, /missing-owner-input/u);
});
