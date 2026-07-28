import assert from "node:assert/strict";
import {
  chmodSync,
  existsSync,
  readFileSync,
  unlinkSync,
  writeFileSync,
} from "node:fs";
import path from "node:path";
import { spawnSync } from "node:child_process";
import test from "node:test";
import { fileURLToPath } from "node:url";

import {
  createAwsTrialBundle,
  wrapperNames,
} from "./helpers/aws-trial-fixture.mjs";

const repositoryRoot = path.resolve(
  path.dirname(fileURLToPath(import.meta.url)),
  "..",
);
const scriptsRoot = path.join(repositoryRoot, "scripts");
const checkerPath = path.join(scriptsRoot, "check-aws-plan.sh");

function readRequired(filePath) {
  assert.equal(
    existsSync(filePath),
    true,
    `${path.relative(repositoryRoot, filePath)} must exist`,
  );
  return readFileSync(filePath, "utf8");
}

function runChecker(bundle, environment = {}) {
  return spawnSync(checkerPath, ["offline", bundle.directory], {
    cwd: repositoryRoot,
    encoding: "utf8",
    env: {
      ...process.env,
      AVIA_AWS_CALLER_ARN: bundle.manifest.callerArn,
      NODE_BIN: process.execPath,
      OPA_BIN: process.env.OPA_BIN ??
        "/private/tmp/aviasurveil360-tools/bin/opa",
      ...environment,
    },
  });
}

test("AWS trial wrappers, runbook, and smoke specification exist", () => {
  for (const wrapperName of wrapperNames) {
    const source = readRequired(path.join(scriptsRoot, wrapperName));
    assert.match(source, /^#!\/usr\/bin\/env bash/);
    assert.match(source, /set -euo pipefail/);
  }
  readRequired(
    path.join(repositoryRoot, "docs", "operations", "AWS_TRIAL_RUNBOOK.md"),
  );
  readRequired(
    path.join(repositoryRoot, "apps", "web", "tests", "e2e", "aws-trial-smoke.spec.ts"),
  );
});

test("offline checker accepts a protected, internally bound fixture bundle", () => {
  const bundle = createAwsTrialBundle(repositoryRoot);
  const result = runChecker(bundle);
  assert.equal(result.status, 0, result.stderr);
  assert.match(result.stdout, /verified protected foundation-ecr plan bundle/);
});

test("offline checker fails closed for every ownership and artifact mutation", async (t) => {
  const mutations = {
    "account-mismatch": () => createAwsTrialBundle(repositoryRoot, {
      mutateManifest: (manifest) => {
        manifest.accountId = "999900001111";
      },
    }),
    "capacity-unbounded": () => createAwsTrialBundle(repositoryRoot, {
      mutateDecision: (decision) => {
        decision.capacity.max = 101;
      },
    }),
    "change-window": () => createAwsTrialBundle(repositoryRoot, {
      mutateDecision: (decision) => {
        decision.changeWindow.startsAt =
          new Date(Date.now() + 3_600_000).toISOString();
        decision.changeWindow.endsAt =
          new Date(Date.now() + 7_200_000).toISOString();
      },
    }),
    "certificate-mismatch": () => createAwsTrialBundle(repositoryRoot, {
      mutateDecision: (decision) => {
        decision.certificateArn =
          "arn:aws:acm:eu-west-1:999900001111:certificate/11111111-2222-3333-4444-555555555555";
      },
    }),
    "cost-unbounded": () => createAwsTrialBundle(repositoryRoot, {
      mutateDecision: (decision) => {
        decision.budgetCeilingUsd = 0;
      },
    }),
    "data-residency": () => createAwsTrialBundle(repositoryRoot, {
      mutateDecision: (decision) => {
        decision.dataResidencyApproved = false;
      },
    }),
    "image-scope": () => createAwsTrialBundle(repositoryRoot, {
      mutateManifest: (manifest) => {
        manifest.images[0].reference =
          "999900001111.dkr.ecr.eu-west-1.amazonaws.com/avia-trial@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa";
      },
    }),
    "image-binding": () => createAwsTrialBundle(repositoryRoot, {
      mutateManifest: (manifest) => {
        manifest.images[0].reference =
          "111122223333.dkr.ecr.eu-central-1.amazonaws.com/avia-trial@sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc";
      },
    }),
    "missing-sbom": () => createAwsTrialBundle(repositoryRoot, {
      mutateManifest: (manifest) => {
        manifest.images[0].sbomSha256 = "";
      },
    }),
    "mutable-image": () => createAwsTrialBundle(repositoryRoot, {
      mutateManifest: (manifest) => {
        manifest.images[0].reference =
          "111122223333.dkr.ecr.eu-central-1.amazonaws.com/avia-trial:latest";
      },
    }),
    "phase-boundary": () => createAwsTrialBundle(repositoryRoot, {
      mutateManifest: (manifest) => {
        manifest.commands[0].unit = "database";
      },
    }),
    "region-mismatch": () => createAwsTrialBundle(repositoryRoot, {
      mutateManifest: (manifest) => {
        manifest.region = "eu-west-1";
      },
    }),
    "stale-plan": () => createAwsTrialBundle(repositoryRoot, {
      mutateManifest: (manifest) => {
        manifest.expiresAt = "2020-01-01T00:00:00.000Z";
      },
    }),
    "unscanned-image": () => createAwsTrialBundle(repositoryRoot, {
      mutateManifest: (manifest) => {
        manifest.images[0].trivyHigh = 1;
      },
    }),
    "unredacted-plan": () => createAwsTrialBundle(repositoryRoot, {
      mutatePlan: (plan) => {
        plan.resource_changes[0].change.after.password =
          "plaintext-must-not-enter-plan-evidence";
      },
    }),
    "wrapper-hash": () => createAwsTrialBundle(repositoryRoot, {
      mutateManifest: (manifest) => {
        manifest.wrappers["aws-trial-apply.sh"] = "0".repeat(64);
      },
    }),
  };

  for (const [reason, build] of Object.entries(mutations)) {
    await t.test(reason, () => {
      const result = runChecker(build());
      assert.notEqual(result.status, 0);
      assert.match(result.stderr, new RegExp(reason));
    });
  }
});

test("offline checker rejects missing, changed, or world-readable artifacts", async (t) => {
  await t.test("missing decision", () => {
    const bundle = createAwsTrialBundle(repositoryRoot);
    unlinkSync(bundle.decisionPath);
    const result = runChecker(bundle);
    assert.notEqual(result.status, 0);
    assert.match(result.stderr, /missing-decision/);
  });

  await t.test("changed plan", () => {
    const bundle = createAwsTrialBundle(repositoryRoot);
    writeFileSync(bundle.planBinaryPath, "changed\n");
    const result = runChecker(bundle);
    assert.notEqual(result.status, 0);
    assert.match(result.stderr, /plan-hash/);
  });

  await t.test("world-readable plan", () => {
    const bundle = createAwsTrialBundle(repositoryRoot);
    chmodSync(bundle.planJsonPath, 0o644);
    const result = runChecker(bundle);
    assert.notEqual(result.status, 0);
    assert.match(result.stderr, /artifact-permission/);
  });

  await t.test("wrong caller", () => {
    const bundle = createAwsTrialBundle(repositoryRoot);
    const result = runChecker(bundle, {
      AVIA_AWS_CALLER_ARN:
        "arn:aws:iam::111122223333:role/UnexpectedCaller",
    });
    assert.notEqual(result.status, 0);
    assert.match(result.stderr, /caller-mismatch/);
  });
});

test("command wrappers preview exact phases and gate every mutable execution", () => {
  const sources = Object.fromEntries(
    wrapperNames.map((name) => [
      name,
      readRequired(path.join(scriptsRoot, name)),
    ]),
  );
  const combined = Object.values(sources).join("\n");

  for (const phase of [
    "bootstrap",
    "foundation-ecr",
    "artifact-publication",
    "data-runtime",
  ]) {
    assert.match(combined, new RegExp(phase));
  }
  for (const source of Object.values(sources)) {
    assert.match(source, /\bpreview\b/);
  }
  for (const name of wrapperNames.filter(
    (candidate) => !["check-aws-plan.sh", "aws-trial-plan.sh"].includes(candidate),
  )) {
    assert.match(sources[name], /exact-authorization/);
    assert.match(sources[name], /check-aws-plan\.sh/);
  }

  assert.doesNotMatch(combined, /\brun(?:-all|\s+--all)\s+(?:apply|destroy)\b/);
  assert.doesNotMatch(combined, /aws\s+\S+\s+(?:delete|terminate)[^\n]*--all\b/);
  assert.match(sources["aws-trial-plan.sh"], /AVIA_TG_PLAN_DIR/);
  assert.doesNotMatch(
    sources["aws-trial-plan.sh"],
    /terragrunt\s+plan[\s\S]{0,300}--\s+-out=/,
  );
  assert.match(sources["aws-trial-publish-artifacts.sh"], /sha256|digest/i);
  assert.match(sources["aws-trial-publish-artifacts.sh"], /sbom/i);
  assert.match(sources["aws-trial-publish-artifacts.sh"], /shasum.*sbom/i);
  assert.match(sources["aws-trial-smoke.sh"], /86/);
  assert.match(sources["aws-trial-smoke.sh"], /MFA|OIDC/);
  assert.match(sources["aws-trial-smoke.sh"], /AVIA_E2E_PROFILE=aws-trial/);
  assert.match(sources["aws-trial-smoke.sh"], /--project=aws-trial/);
  const playwrightConfig = readRequired(
    path.join(repositoryRoot, "apps", "web", "playwright.config.ts"),
  );
  assert.match(playwrightConfig, /e2eProfile === "aws-trial"/);
  assert.match(playwrightConfig, /name: "aws-trial"/);
  assert.match(playwrightConfig, /aws-trial-smoke\.spec\.ts/);
  assert.match(sources["aws-trial-rollback.sh"], /previous.*digest/is);
  assert.doesNotMatch(
    sources["aws-trial-rollback.sh"],
    /(?:dropdb|DROP DATABASE|delete-db-instance)/i,
  );
  assert.match(sources["aws-trial-destroy.sh"], /tagged-resource|resource-manifest/);
});
