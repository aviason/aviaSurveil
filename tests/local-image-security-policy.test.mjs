import assert from "node:assert/strict";
import { existsSync, readFileSync } from "node:fs";
import { test } from "node:test";
import { fileURLToPath, pathToFileURL } from "node:url";
import path from "node:path";

const repositoryRoot = path.resolve(
  path.dirname(fileURLToPath(import.meta.url)),
  "..",
);
const imageLockPath = path.join(
  repositoryRoot,
  "deploy/local/image-lock.json",
);
const vulnerabilityPolicyPath = path.join(
  repositoryRoot,
  "deploy/local/vulnerability-policy.json",
);
const policyModulePath = path.join(
  repositoryRoot,
  "scripts/lib/local-image-policy.mjs",
);

function read(relativePath) {
  const absolutePath = path.join(repositoryRoot, relativePath);
  assert.ok(existsSync(absolutePath), `${relativePath} must exist`);
  return readFileSync(absolutePath, "utf8");
}

async function loadPolicyModule() {
  assert.ok(existsSync(policyModulePath), "local image policy module must exist");
  return import(pathToFileURL(policyModulePath));
}

function validEvidence() {
  const digest = `sha256:${"a".repeat(64)}`;
  return {
    images: {
      api: {
        tag: "aviasurveil360/api:local",
        digest,
      },
    },
    sboms: {
      api: {
        digest,
        format: "cyclonedx-json",
        path: "api.cdx.json",
      },
    },
    scans: {
      api: {
        digest,
        status: "passed",
        severities: ["HIGH", "CRITICAL"],
      },
    },
  };
}

test("all Task 2 build bases are immutable entries in image-lock.json", () => {
  const lock = JSON.parse(readFileSync(imageLockPath, "utf8"));
  for (const imageName of [
    "node-build",
    "go-build",
    "go-runtime",
  ]) {
    const reference = lock.images?.[imageName]?.reference;
    assert.equal(typeof reference, "string", `${imageName} must be locked`);
    assert.match(reference, /@sha256:[a-f0-9]{64}$/u);
  }
  assert.equal(
    lock.images?.["web-runtime"],
    undefined,
    "unused web runtime base must not remain in the reviewed image lock",
  );
});

test("SBOM and vulnerability tools run from reviewed digest-pinned containers", () => {
  const lock = JSON.parse(readFileSync(imageLockPath, "utf8"));
  for (const imageName of ["syft-tool", "trivy-tool"]) {
    const reference = lock.images?.[imageName]?.reference;
    assert.equal(typeof reference, "string", `${imageName} must be locked`);
    assert.match(reference, /@sha256:[a-f0-9]{64}$/u);
  }

  const sbomScript = read("scripts/generate-image-sboms.sh");
  const scanScript = read("scripts/scan-local-images.sh");
  assert.match(sbomScript, /\.images\["syft-tool"\]\.reference/u);
  assert.match(scanScript, /\.images\["trivy-tool"\]\.reference/u);
  for (const script of [sbomScript, scanScript]) {
    assert.match(script, /\bdocker run\b/u);
    assert.match(script, /--read-only/u);
    assert.match(script, /--security-opt no-new-privileges/u);
    assert.match(script, /--group-add "\$docker_socket_group"/u);
    assert.match(script, /--tmpfs \/tmp:.*mode=1777/u);
  }
  assert.match(sbomScript, /--tmpfs \/tmp:.*size=1g/u);
  assert.match(sbomScript, /Generated 8 digest-bound CycloneDX SBOMs/u);
  assert.match(scanScript, /--tmpfs \/tmp:.*size=2g/u);
  assert.match(
    scanScript,
    /trivy_temp=\$\(mktemp -d "\$state_directory\/\.trivy-tmp\.XXXXXX"\)/u,
  );
  assert.match(scanScript, /--volume "\$trivy_temp:\/trivy-tmp"/u);
  assert.match(scanScript, /--env TMPDIR=\/trivy-tmp/u);
  assert.match(scanScript, /rm -rf -- "\$trivy_temp"/u);
  assert.match(
    scanScript,
    /All 8 local image digests passed the HIGH\/CRITICAL vulnerability gate/u,
  );
  assert.doesNotMatch(sbomScript, /^\s*syft\s/mu);
  assert.doesNotMatch(scanScript, /^\s*trivy\s/mu);
});

test("the gateway rebuilds pinned Caddy with patched Go dependencies into scratch", () => {
  const lock = JSON.parse(readFileSync(imageLockPath, "utf8"));
  assert.match(lock.components?.caddy?.version ?? "", /^v\d+\.\d+\.\d+$/u);
  assert.match(lock.components?.grpc?.version ?? "", /^v\d+\.\d+\.\d+$/u);

  const dockerfile = read("deploy/local/gateway/Dockerfile");
  assert.match(dockerfile, /^FROM \$\{GO_BUILD_IMAGE\} AS gateway-build$/mu);
  assert.match(dockerfile, /github\.com\/caddyserver\/caddy\/v2/u);
  assert.match(dockerfile, /google\.golang\.org\/grpc/u);
  assert.match(dockerfile, /^FROM scratch AS gateway$/mu);
  assert.match(dockerfile, /\/healthcheck/u);
  assert.match(dockerfile, /--chown=1000:1000 \/runtime\/data \/data/u);
  assert.match(dockerfile, /--chown=1000:1000 \/runtime\/config \/config/u);
  assert.doesNotMatch(dockerfile, /\bapk\b/u);
});

test("the Docker build context excludes local secrets and host build state", () => {
  const dockerIgnore = read(".dockerignore");
  assert.match(dockerIgnore, /^\.local\/?$/mu);
  assert.match(dockerIgnore, /^\*\*\/node_modules\/?$/mu);
  assert.match(dockerIgnore, /^\*\*\/dist\/?$/mu);
  assert.match(dockerIgnore, /^\.git\/?$/mu);
});

test("Dockerfile base defaults resolve only through the reviewed image lock", () => {
  const lock = JSON.parse(readFileSync(imageLockPath, "utf8"));
  const lockedReferences = new Set(
    Object.values(lock.images).map((entry) => entry.reference),
  );
  for (const relativePath of [
    "apps/web/Dockerfile",
    "apps/api/Dockerfile",
    "deploy/local/gateway/Dockerfile",
    "deploy/recovery/Dockerfile",
  ]) {
    const dockerfile = read(relativePath);
    const defaults = [
      ...dockerfile.matchAll(/^ARG [A-Z0-9_]+=(\S+@sha256:[a-f0-9]{64})$/gmu),
    ].map((match) => match[1]);
    assert.ok(defaults.length > 0, `${relativePath} must pin its base ARGs`);
    for (const reference of defaults) {
      assert.ok(
        lockedReferences.has(reference),
        `${relativePath} uses an image absent from image-lock.json`,
      );
    }
  }
});

test("image build evidence identifies dirty repository input without exposing content", () => {
  const buildScript = read("scripts/build-local-images.sh");
  assert.match(buildScript, /\bsource_dirty\b/u);
  assert.match(buildScript, /\bsource_state_sha256\b/u);
  assert.match(buildScript, /git[^\n]* status --porcelain/u);
  assert.match(buildScript, /git[^\n]* diff --binary HEAD/u);
  assert.match(buildScript, /org\.opencontainers\.image\.revision/u);
  assert.match(buildScript, /io\.aviasurveil360\.source-dirty/u);
  assert.match(buildScript, /io\.aviasurveil360\.source-state-sha256/u);
  assert.doesNotMatch(buildScript, /(?:cat|printf)[^\n]*source_diff/u);
});

test("image evidence rejects a missing CycloneDX SBOM", async () => {
  const { validateImageEvidence } = await loadPolicyModule();
  const evidence = validEvidence();
  delete evidence.sboms.api;
  assert.ok(
    validateImageEvidence(evidence).some(
      (violation) => violation.code === "MISSING_SBOM",
    ),
  );
});

test("image evidence rejects a scan for any digest other than the built digest", async () => {
  const { validateImageEvidence } = await loadPolicyModule();
  const evidence = validEvidence();
  evidence.scans.api.digest = `sha256:${"b".repeat(64)}`;
  assert.ok(
    validateImageEvidence(evidence).some(
      (violation) => violation.code === "UNSCANNED_DIGEST",
    ),
  );
});

test("image evidence rejects unresolved HIGH or CRITICAL findings", async () => {
  const { validateImageEvidence } = await loadPolicyModule();
  const evidence = validEvidence();
  evidence.scans.api.status = "blocked";
  assert.ok(
    validateImageEvidence(evidence).some(
      (violation) => violation.code === "UNAPPROVED_FINDINGS",
    ),
  );
});

test("vulnerability exceptions require owner, expiry, rationale, and tracker", async () => {
  assert.ok(
    existsSync(vulnerabilityPolicyPath),
    "vulnerability policy must exist",
  );
  const {
    exceptionIDsForImage,
    validateVulnerabilityPolicy,
  } = await loadPolicyModule();
  const policy = JSON.parse(readFileSync(vulnerabilityPolicyPath, "utf8"));
  assert.deepEqual(
    validateVulnerabilityPolicy(policy, { today: "2026-07-24" }),
    [],
  );
  assert.deepEqual(policy.exceptions, []);
  const invalid = structuredClone(policy);
  invalid.exceptions = [
    {
      image: "api",
      digest: `sha256:${"a".repeat(64)}`,
      vulnerabilityId: "CVE-2099-0001",
      owner: "",
      expiresOn: "2026-07-23",
      rationale: "short",
      tracker: "",
    },
  ];
  const codes = new Set(
    validateVulnerabilityPolicy(invalid, { today: "2026-07-24" }).map(
      (violation) => violation.code,
    ),
  );
  for (const code of [
    "MISSING_EXCEPTION_OWNER",
    "EXPIRED_EXCEPTION",
    "MISSING_EXCEPTION_RATIONALE",
    "MISSING_EXCEPTION_TRACKER",
  ]) {
    assert.ok(codes.has(code), `expected ${code}`);
  }

  const builtDigest = `sha256:${"b".repeat(64)}`;
  const sourceDigest = `sha256:${"c".repeat(64)}`;
  const sourceBound = structuredClone(policy);
  sourceBound.exceptions = [
    {
      image: "auth",
      digest: sourceDigest,
      vulnerabilityId: "CVE-2099-0002",
      owner: "Local Platform Security",
      expiresOn: "2026-08-08",
      rationale: "Immutable upstream source image is explicitly accepted.",
      tracker: "docs/exec-plans/tech-debt-tracker.md#td-test",
    },
  ];
  assert.deepEqual(
    exceptionIDsForImage(
      sourceBound,
      "auth",
      builtDigest,
      {
        additionalDigests: [sourceDigest],
        today: "2026-07-24",
      },
    ),
    ["CVE-2099-0002"],
  );
  assert.deepEqual(
    exceptionIDsForImage(
      sourceBound,
      "auth",
      builtDigest,
      {
        additionalDigests: [`sha256:${"d".repeat(64)}`],
        today: "2026-07-24",
      },
    ),
    [],
    "an exception must not float to another source digest",
  );

  const scanScript = read("scripts/scan-local-images.sh");
  assert.match(scanScript, /\.images\[\$name\]\.reference/u);
  assert.match(scanScript, /source_digest/u);
});

test("SBOM and scan scripts are digest-bound and fail closed", () => {
  const buildScript = read("scripts/build-local-images.sh");
  const sbomScript = read("scripts/generate-image-sboms.sh");
  const scanScript = read("scripts/scan-local-images.sh");
  assert.match(buildScript, /docker image inspect/u);
  assert.match(buildScript, /sha256:/u);
  assert.match(sbomScript, /syft-tool/u);
  assert.match(sbomScript, /cyclonedx-json/u);
  assert.match(sbomScript, /digest/u);
  assert.match(scanScript, /trivy-tool/u);
  assert.match(scanScript, /HIGH,CRITICAL/u);
  assert.match(scanScript, /--exit-code[=\s]+1/u);
  assert.match(scanScript, /local-image-policy\.mjs/u);
  assert.doesNotMatch(scanScript, /\|\|\s*true/u);
});
