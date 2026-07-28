import assert from "node:assert/strict";
import { existsSync, readFileSync, readdirSync } from "node:fs";
import path from "node:path";
import test from "node:test";
import { fileURLToPath } from "node:url";

const repositoryRoot = path.resolve(
  path.dirname(fileURLToPath(import.meta.url)),
  "..",
);
const terragruntRoot = path.join(repositoryRoot, "infra", "terragrunt");
const environmentRoot = path.join(
  terragruntRoot,
  "environments",
  "aws-trial",
);
const componentsRoot = path.join(environmentRoot, "components");

const componentNames = [
  "network",
  "identity-secrets",
  "ecr",
  "object-storage",
  "observability",
  "security",
  "service-endpoints",
  "load-balancer",
  "artifact-publication",
  "database",
  "compute",
  "backup",
];

function readRequired(filePath) {
  assert.equal(
    existsSync(filePath),
    true,
    `${path.relative(repositoryRoot, filePath)} must exist`,
  );
  return readFileSync(filePath, "utf8");
}

function walkFiles(root, suffix) {
  if (!existsSync(root)) return [];
  return readdirSync(root, { withFileTypes: true }).flatMap((entry) => {
    const entryPath = path.join(root, entry.name);
    if (entry.isDirectory()) return walkFiles(entryPath, suffix);
    return entry.name.endsWith(suffix) ? [entryPath] : [];
  });
}

test("Terragrunt layout declares bootstrap, catalog, environment, and every phase unit", () => {
  readRequired(path.join(terragruntRoot, "terragrunt.hcl"));
  readRequired(path.join(terragruntRoot, "root.hcl"));
  readRequired(path.join(terragruntRoot, "catalog", "components.hcl"));
  readRequired(path.join(environmentRoot, "account.hcl"));
  readRequired(path.join(environmentRoot, "environment.hcl"));
  readRequired(path.join(environmentRoot, "region.hcl.example"));
  readRequired(
    path.join(terragruntRoot, "bootstrap", "remote-state", "terragrunt.hcl"),
  );
  readRequired(
    path.join(
      terragruntRoot,
      "bootstrap",
      "REMOTE_STATE_MIGRATION.md",
    ),
  );
  for (const component of componentNames) {
    readRequired(path.join(componentsRoot, component, "terragrunt.hcl"));
  }
});

test("Terragrunt composes only pinned or repository-owned Terraform modules", () => {
  const files = walkFiles(terragruntRoot, ".hcl");
  assert.ok(files.length >= componentNames.length + 6);
  for (const filePath of files) {
    const source = readRequired(filePath);
    assert.doesNotMatch(
      source,
      /^\s*resource\s+"/m,
      `${path.relative(repositoryRoot, filePath)} must not own resources`,
    );
    for (const [, moduleSource] of source.matchAll(
      /^\s*source\s*=\s*"([^"]+)"/gm,
    )) {
      if (/^[a-z0-9-]+\/[a-z0-9-]+$/.test(moduleSource)) continue;
      if (/^(?:git::|https?:)/.test(moduleSource)) {
        assert.match(moduleSource, /\?ref=[a-f0-9]{40}$/);
      } else {
        assert.match(moduleSource, /infra\/terraform\//);
      }
    }
  }
});

test("generated provider and backend require explicit identity and protected native-lock state", () => {
  const root = readRequired(path.join(terragruntRoot, "root.hcl"));
  assert.match(root, /generate\s+"provider"/);
  assert.match(root, /remote_state\s*\{/);
  assert.match(root, /backend\s*=\s*local\.fixture_mode\s*\?\s*"local"\s*:\s*"s3"/);
  assert.match(root, /config\s*=\s*local\.fixture_mode\s*\?/);
  assert.match(root, /encrypt\s*=\s*true/);
  assert.match(root, /use_lockfile\s*=\s*true/);
  assert.match(root, /kms_key_id/);
  assert.match(root, /state_bucket_name/);
  assert.match(root, /path_relative_to_include\(\)/);

  const environmentSources = [
    readRequired(path.join(environmentRoot, "account.hcl")),
    readRequired(path.join(environmentRoot, "environment.hcl")),
  ].join("\n");
  assert.doesNotMatch(environmentSources, /\b\d{12}\b/);
  assert.doesNotMatch(environmentSources, /\b(?:us|eu|ap|sa|ca|me|af)-[a-z]+-\d\b/);
  assert.match(environmentSources, /get_env\("[A-Z0-9_]+",\s*""\)/);
});

test("dependency mocks are plan-only and cannot survive apply", () => {
  const componentFiles = walkFiles(componentsRoot, "terragrunt.hcl");
  const dependencyBlocks = componentFiles
    .map((filePath) => readRequired(filePath))
    .join("\n");
  assert.match(dependencyBlocks, /mock_outputs\s*=/);
  for (const match of dependencyBlocks.matchAll(
    /mock_outputs_allowed_terraform_commands\s*=\s*\[([^\]]+)\]/g,
  )) {
    assert.match(match[1], /"validate"/);
    assert.match(match[1], /"plan"/);
    assert.doesNotMatch(match[1], /"apply"|"destroy"/);
  }
  assert.doesNotMatch(
    dependencyBlocks,
    /mock_outputs_allowed_terraform_commands\s*=\s*\[[^\]]*(?:"apply"|"destroy")/,
  );
});

test("plan orchestration has preflight and policy hooks plus deterministic artifact names", () => {
  const root = readRequired(path.join(terragruntRoot, "root.hcl"));
  assert.match(root, /before_hook\s+"preflight"/);
  assert.match(root, /after_hook\s+"policy"/);
  assert.match(root, /commands\s*=\s*\["plan"\]/);
  assert.match(root, /AVIA_TG_PLAN_DIR/);
  assert.match(root, /path_relative_to_include\(\)/);
  assert.match(root, /check-terragrunt\.sh/);
});

test("catalog fixes bootstrap-to-runtime phase order without broad destroy", () => {
  const catalog = readRequired(
    path.join(terragruntRoot, "catalog", "components.hcl"),
  );
  for (const phase of [
    "bootstrap",
    "foundation",
    "artifact-publication",
    "data-runtime",
  ]) {
    assert.match(catalog, new RegExp(`phase\\s*=\\s*"${phase}"`));
  }
  for (const component of componentNames) {
    assert.match(catalog, new RegExp(`\\b${component.replaceAll("-", "_")}\\s*=`));
  }

  const allSources = walkFiles(terragruntRoot, ".hcl")
    .map((filePath) => readRequired(filePath))
    .join("\n");
  assert.doesNotMatch(allSources, /\brun\s+--all\s+destroy\b/);
  assert.doesNotMatch(allSources, /\brun-all\s+destroy\b/);
});

test("Terragrunt checker fails closed and never performs apply or destroy", () => {
  const checker = readRequired(
    path.join(repositoryRoot, "scripts", "check-terragrunt.sh"),
  );
  assert.match(checker, /missing-owner-input/);
  assert.match(checker, /AVIA_TG_INPUTS_FILE/);
  assert.match(checker, /terragrunt\s+hcl\s+fmt\s+--check/);
  assert.match(checker, /terragrunt\s+run\s+--all\s+validate/);
  assert.match(checker, /terragrunt\s+run\s+--all\s+plan/);
  assert.doesNotMatch(checker, /\b(?:apply|destroy)\b/);
});
