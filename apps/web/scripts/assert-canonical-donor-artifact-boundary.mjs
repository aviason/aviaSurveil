#!/usr/bin/env node
import assert from "node:assert/strict";
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const SUPPLEMENTAL_ROUTE = "/v1/preprod/aga-demo-workspace/";
const SUPPLEMENTAL_UI_ROUTE = "/admin/aga-demo-workspace";

function inventory(directory, prefix = "") {
  return fs.readdirSync(directory, { withFileTypes: true }).flatMap((entry) => {
    const relativePath = path.posix.join(prefix, entry.name);
    return entry.isDirectory()
      ? inventory(path.join(directory, entry.name), relativePath)
      : [relativePath];
  });
}

function resolveArtifactRoot(suppliedPath, profile) {
  const suppliedRoot = path.resolve(suppliedPath);
  const profileRoot = path.join(suppliedRoot, profile);
  return fs.existsSync(profileRoot) ? profileRoot : suppliedRoot;
}

function readTextFiles(root, files) {
  return files
    // Vite's internal manifest records every source graph entry for tooling;
    // it is not loaded by the browser and may mention HTTP-only route names
    // even when the emitted demo chunks contain no supplemental code.
    .filter((relativePath) => relativePath !== ".vite/manifest.json")
    .filter((relativePath) => /\.(?:js|mjs|html|json|map|css)$/i.test(relativePath))
    .map((relativePath) => ({
      relativePath,
      contents: fs.readFileSync(path.join(root, relativePath), "utf8"),
    }));
}

function assertNoSourceMapBodies(root, files) {
  for (const relativePath of files.filter((file) => file.endsWith(".map"))) {
    let parsed;
    try {
      parsed = JSON.parse(fs.readFileSync(path.join(root, relativePath), "utf8"));
    } catch (error) {
      throw new Error(`Artifact source map is not valid JSON: ${relativePath}: ${error.message}`);
    }
    const bodies = Array.isArray(parsed.sourcesContent) ? parsed.sourcesContent : [];
    assert.ok(
      bodies.every((body) => body === null || body === undefined || body === ""),
      `Artifact contains a body-bearing source map: ${relativePath}`,
    );
  }
}

export function assertCanonicalDonorArtifactBoundary(profile, suppliedPath = "dist") {
  assert.ok(profile === "demo" || profile === "http", `Unknown artifact profile: ${profile}`);
  const artifactRoot = resolveArtifactRoot(suppliedPath, profile);
  assert.ok(fs.existsSync(artifactRoot), `Artifact directory is missing: ${artifactRoot}`);

  const files = inventory(artifactRoot);
  const textFiles = readTextFiles(artifactRoot, files);
  const source = textFiles.map(({ relativePath, contents }) => `${relativePath}\n${contents}`).join("\n");
  const buildInputsPath = path.join(artifactRoot, "build-inputs.json");
  assert.ok(fs.existsSync(buildInputsPath), "Artifact input inventory is missing: build-inputs.json");
  const buildInputs = JSON.parse(fs.readFileSync(buildInputsPath, "utf8"));
  assert.equal(buildInputs.profile, profile, `Artifact input inventory must identify ${profile}`);

  if (profile === "demo") {
    const forbidden = [
      new RegExp(SUPPLEMENTAL_ROUTE.replaceAll("/", "\\/"), "i"),
      new RegExp(SUPPLEMENTAL_UI_ROUTE.replaceAll("/", "\\/"), "i"),
      /AGADemoWorkspaceBackend/i,
      /aga-demo-workspace/i,
      /classificationQuery/i,
      /expectedGenerationId/i,
      /aga-ws-[a-z0-9-]{8,}/i,
      /AGA_ALL_FORMS_SOURCE_RISK_DRAFT_V1/i,
      /FSS-AGA-FORM-\d{3}/i,
    ];
    for (const pattern of forbidden) {
      assert.doesNotMatch(source, pattern, `Demo artifact contains supplemental workspace or candidate data: ${pattern}`);
    }
    for (const input of buildInputs.inputs ?? []) {
      assert.doesNotMatch(input, /aga-demo-workspace|src\/entry\/http|src\/backend\/aga-demo-workspace/i, `Demo artifact imports supplemental workspace input: ${input}`);
    }
    assertNoSourceMapBodies(artifactRoot, files);
    return { profile, files: files.length, inputs: buildInputs.inputs?.length ?? 0 };
  }

  for (const marker of [
    SUPPLEMENTAL_ROUTE,
    `${SUPPLEMENTAL_ROUTE}capability`,
    `${SUPPLEMENTAL_ROUTE}classification/query`,
    // TypeScript interface names are erased by the production build. The
    // request method is the stable emitted client marker instead.
    "classificationQuery",
    "aga-candidate-demo",
    "AGACandidateDemo",
    "agaCandidateDemo",
  ]) {
    assert.doesNotMatch(source, new RegExp(marker.replaceAll("/", "\\/")), `HTTP artifact still contains removed donor marker: ${marker}`);
  }
  for (const input of buildInputs.inputs ?? []) {
    assert.doesNotMatch(input, /aga-demo-workspace|aga-candidate-demo|src\/backend\/(?:aga-demo-workspace|aga-candidate-demo)/i, `HTTP artifact imports removed donor input: ${input}`);
  }
  assert.doesNotMatch(source, /FSS-AGA-FORM-\d{3}|AGA_ALL_FORMS_SOURCE_RISK_DRAFT_V1|questionRef/i, "HTTP artifact embeds candidate/classification rows");
  assert.doesNotMatch(source, /AGADemoWorkspaceCapability[^\n]{0,240}available[^\n]{0,80}true/i, "HTTP artifact statically succeeds the workspace capability");
  assertNoSourceMapBodies(artifactRoot, files);
  return { profile, files: files.length, inputs: buildInputs.inputs?.length ?? 0 };
}

const invokedPath = process.argv[1] ? path.resolve(process.argv[1]) : null;
if (invokedPath === fileURLToPath(import.meta.url)) {
  const profile = process.argv[2] === "--profile" ? process.argv[3] : "http";
  const artifactIndex = process.argv.indexOf("--artifact");
  const artifact = artifactIndex >= 0 ? process.argv[artifactIndex + 1] : process.argv[2] ?? "dist";
  const result = assertCanonicalDonorArtifactBoundary(profile, artifact);
  console.log(`canonical-donor-artifact-scan: ok (${result.profile}, ${result.files} files, ${result.inputs} inputs)`);
}
