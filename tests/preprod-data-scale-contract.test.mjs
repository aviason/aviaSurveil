import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import path from "node:path";
import { spawnSync } from "node:child_process";
import test from "node:test";
import { fileURLToPath } from "node:url";

const repositoryRoot = path.resolve(
  path.dirname(fileURLToPath(import.meta.url)),
  "..",
);
const profileRunner = path.join(
  repositoryRoot,
  "scripts",
  "test-preprod-data-profile.sh",
);
const specificationPath = path.join(
  repositoryRoot,
  "docs",
  "product-specs",
  "data-and-rules",
  "PREPROD_IDENTITY_AND_DATA_PROFILE.md",
);
const contractBegin = "<!-- PREPROD_IDENTITY_DATA_CONTRACT:BEGIN -->";
const contractEnd = "<!-- PREPROD_IDENTITY_DATA_CONTRACT:END -->";

function readContract() {
  const source = readFileSync(specificationPath, "utf8");
  const begin = source.indexOf(contractBegin);
  const end = source.indexOf(contractEnd);
  assert.ok(begin >= 0 && end > begin, "profile contract markers must exist");
  const fenced = source.slice(begin + contractBegin.length, end).trim();
  const match = fenced.match(/^```json\s*\n([\s\S]+)\n```$/u);
  assert.ok(match, "profile contract must contain one fenced JSON object");
  return JSON.parse(match[1]);
}

function materializeQualificationProfile(contract, name) {
  const profiles = contract.dataProfiles.profiles;
  const fullVolume = profiles.find((profile) => profile.name === name);
  assert.ok(fullVolume, `missing ${name} profile`);
  const revision = fullVolume.localQualification;
  if (!revision) return fullVolume;

  const [sourceName, sourceVersion] = revision.sourceProfile.split("@");
  const source = profiles.find(
    (profile) =>
      profile.name === sourceName && profile.version === sourceVersion,
  );
  assert.ok(source, `missing qualification source ${revision.sourceProfile}`);
  const preserved = new Set(revision.preservedCatalogCountFamilies);
  const expectedCounts = Object.fromEntries(
    Object.entries(source.expectedCounts).map(([family, count]) => [
      family,
      preserved.has(family) ? count : count * revision.scaleMultiplier,
    ]),
  );
  const exactDistributions = Object.fromEntries(
    Object.entries(source.exactDistributions).map(([family, distribution]) => [
      family,
      family === "organizations"
        ? revision.organizationDistribution
        : Object.fromEntries(
          Object.entries(distribution).map(([state, count]) => [
            state,
            preserved.has(family)
              ? count
              : count * revision.scaleMultiplier,
          ]),
        ),
    ]),
  );
  return {
    ...fullVolume,
    version: revision.version,
    status: revision.status,
    resourceEnvelope: revision.resourceEnvelope,
    expectedCounts,
    exactDistributions,
  };
}

function serviceBlock(compose, name) {
  return compose.match(
    new RegExp(
      `^  ${name}:\\n(?<service>(?:^    .*\\n|^\\n)*)`,
      "mu",
    ),
  )?.groups?.service ?? "";
}

test("profile runner exposes the exact four frozen qualification contracts", () => {
  const contract = readContract();
  const expectedProfiles = [
    "smoke",
    "acceptance",
    "realistic",
    "stress",
  ].map((name) => materializeQualificationProfile(contract, name));
  assert.deepEqual(
    expectedProfiles.map(({ name }) => name),
    ["smoke", "acceptance", "realistic", "stress"],
  );

  for (const expected of expectedProfiles) {
    const result = spawnSync(profileRunner, ["--describe", expected.name], {
      cwd: repositoryRoot,
      encoding: "utf8",
    });
    assert.equal(
      result.status,
      0,
      `${expected.name} describe failed:\n${result.stderr}`,
    );
    assert.equal(result.stderr, "");
    const actual = JSON.parse(result.stdout);
    assert.equal(
      actual.schemaVersion,
      "preprod-profile-qualification-contract/v1",
    );
    assert.equal(actual.profile.name, expected.name);
    assert.equal(actual.profile.version, expected.version);
    assert.deepEqual(actual.profile.catalogs, expected.catalogs);
    assert.deepEqual(actual.profile.resourceEnvelope, expected.resourceEnvelope);
    assert.deepEqual(actual.profile.expectedCounts, expected.expectedCounts);
    assert.deepEqual(
      actual.profile.exactDistributions,
      expected.exactDistributions,
    );
    assert.equal(
      actual.evidenceRoot,
      "docs/demo-evidence/preprod-data",
    );
    assert.deepEqual(actual.operations, [
      "LOAD_EMPTY_TARGET",
      "RESUME_RUN",
      "DROP_RECREATE_TARGET",
    ]);
    assert.equal(actual.cleanupMode, "whole-disposable-namespace");
    assert.equal(actual.selectiveDeletionAllowed, false);
    assert.equal(actual.syntheticOnly, true);
    if (expected.name === "realistic" || expected.name === "stress") {
      assert.equal(
        actual.profile.resourceEnvelope.qualificationSeconds,
        expected.name === "realistic" ? 900 : 1800,
      );
      assert.equal(actual.fullVolumeEndurance.profile, expected.name);
      assert.equal(actual.fullVolumeEndurance.version, "1.0.0");
      assert.equal(
        actual.fullVolumeEndurance.purpose,
        "release-readiness-evidence",
      );
      assert.equal(actual.fullVolumeEndurance.status, "not run");
    }
  }
});

test("profile runner rejects an unknown profile without touching Docker", () => {
  const result = spawnSync(
    profileRunner,
    ["--describe", "reduced-stress"],
    {
      cwd: repositoryRoot,
      encoding: "utf8",
      env: {
        ...process.env,
        PATH: "/usr/bin:/bin",
      },
    },
  );
  assert.notEqual(result.status, 0);
  assert.match(result.stderr, /smoke\|acceptance\|realistic\|stress/u);
});

test("profile runner is fail-closed and creates only bounded evidence", () => {
  const source = readFileSync(profileRunner, "utf8");
  assert.match(source, /^set -euo pipefail$/mu);
  assert.match(source, /^umask 077$/mu);
  assert.match(source, /docs\/demo-evidence\/preprod-data/u);
  assert.match(source, /LOAD_EMPTY_TARGET/u);
  assert.match(source, /RESUME_RUN/u);
  assert.match(source, /DROP_RECREATE_TARGET/u);
  assert.match(source, /durationSeconds/u);
  assert.match(source, /memoryMiB/u);
  assert.match(source, /cpuCores/u);
  assert.match(source, /diskMiB/u);
  assert.match(source, /objectBytes/u);
  assert.match(source, /cleanupSeconds/u);
  assert.match(source, /qualificationSeconds/u);
  assert.match(source, /AVIA_TASK8_PROFILE_VERSION/u);
  assert.match(
    source,
    /export AVIA_PREPROD_PROFILE_QUALIFICATION="true"/u,
  );
  assert.match(source, /O_EXCL|noclobber|already exists/u);
  assert.doesNotMatch(source, /aws\b|kubectl\b|terraform\b/iu);
  assert.doesNotMatch(source, /DELETE\s+FROM|--force|compose\s+down\s+-v/u);
});

test("qualification uses an isolated normal API probe and loader-only interruption", () => {
  const compose = readFileSync(
    path.join(repositoryRoot, "deploy", "local", "compose.yaml"),
    "utf8",
  );
  const connectedRunner = readFileSync(
    path.join(
      repositoryRoot,
      "scripts",
      "test-preprod-connected-scenarios.sh",
    ),
    "utf8",
  );
  const loaderRunner = readFileSync(
    path.join(repositoryRoot, "scripts", "load-preprod-data.sh"),
    "utf8",
  );
  const api = serviceBlock(compose, "preprod-api");
  const loader = serviceBlock(compose, "preprod-data-loader");
  assert.notEqual(api, "", "preprod-api service must exist");
  assert.match(api, /profiles:\s*\[local-preprod-loader\]/u);
  assert.match(api, /target:\s*api/u);
  assert.match(api, /AVIA_ENVIRONMENT:\s*development/u);
  assert.match(api, /AVIA_DATABASE_NAME:\s*aviasurveil360_local_preprod/u);
  assert.match(api, /preprod-app-database/u);
  assert.doesNotMatch(api, /ports:|canonical|testprofile|AVIA_TEST_/iu);
  assert.match(api, /entrypoint:\s*\[\/bin\/sh,\s*-c\]/u);
  assert.match(api, /exec \/app\/api/u);
  assert.match(api, /preprod_app_database_password/u);
  assert.doesNotMatch(
    api,
    /preprod_(?:oidc|session|keycloak_service|minio_api)/u,
  );
  assert.match(loader, /AVIA_PREPROD_PROFILE_QUALIFICATION/u);
  assert.match(
    loader,
    /AVIA_PREPROD_QUALIFICATION_INTERRUPT_AFTER_COMMANDS/u,
  );
  assert.match(
    loaderRunner,
    /AVIA_PREPROD_PROFILE_QUALIFICATION[\s\S]+sleep 2; \/app\/preprod-data-loader "\$@"; status=\$\?; sleep 2; exit "\$status"/u,
  );
  assert.match(connectedRunner, /resourceSampleCount:\s*lines\.length/u);
  assert.match(
    connectedRunner,
    /qualification_started_epoch[\s\S]+qualification_seconds/u,
  );
  assert.match(
    connectedRunner,
    /qualification-duration-exceeded/u,
  );
  assert.match(
    connectedRunner,
    /profileVersion:\s*process\.env\.AVIA_TASK8_PROFILE_VERSION/u,
  );
  assert.match(
    connectedRunner,
    /metrics\.withinEnvelope = envelopeViolations\.length === 0/u,
  );
  assert.match(
    connectedRunner,
    /peak-loader-memory-non-positive[\s\S]+peak-loader-cpu-non-positive/u,
  );
});
