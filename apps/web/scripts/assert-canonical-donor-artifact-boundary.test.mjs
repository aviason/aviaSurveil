import assert from "node:assert/strict";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import test from "node:test";

import { assertCanonicalDonorArtifactBoundary } from "./assert-canonical-donor-artifact-boundary.mjs";

function fixture(files) {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), "avia-aga-artifact-"));
  for (const [relativePath, contents] of Object.entries(files)) {
    const target = path.join(root, relativePath);
    fs.mkdirSync(path.dirname(target), { recursive: true });
    fs.writeFileSync(target, contents);
  }
  return root;
}

test("accepts the demo profile only when the supplemental workspace is absent", () => {
  const root = fixture({
    "build-inputs.json": JSON.stringify({ profile: "demo", inputs: ["src/entry/demo.tsx"] }),
    "index.html": "<html></html>",
    "assets/demo.js": "const demo = true;",
  });
  assert.doesNotThrow(() => assertCanonicalDonorArtifactBoundary("demo", root));
});

test("rejects a demo artifact containing a supplemental workspace marker", () => {
  const root = fixture({
    "build-inputs.json": JSON.stringify({ profile: "demo", inputs: ["src/entry/demo.tsx"] }),
    "assets/demo.js": "\/v1\/preprod\/aga-demo-workspace\/classification\/query",
  });
  assert.throws(() => assertCanonicalDonorArtifactBoundary("demo", root), /supplemental workspace/i);
});

test("accepts a donor-free HTTP artifact without embedded rows", () => {
  const root = fixture({
    "build-inputs.json": JSON.stringify({ profile: "http", inputs: ["src/entry/http.tsx"] }),
    "index.html": "<html></html>",
    "assets/http.js": "const canonical = true;",
    "assets/http.js.map": JSON.stringify({ sources: ["assets/http.js"], sourcesContent: [null] }),
  });
  assert.doesNotThrow(() => assertCanonicalDonorArtifactBoundary("http", root));
});

test("rejects HTTP artifacts with embedded donor rows or source-map bodies", () => {
  const root = fixture({
    "build-inputs.json": JSON.stringify({ profile: "http", inputs: ["src/entry/http.tsx"] }),
    "assets/http.js": "AGADemoWorkspaceBackend classificationQuery questionRef classificationRunId",
    "assets/http.js.map": JSON.stringify({ sources: ["src/backend/aga-demo-workspace.ts"], sourcesContent: ["const body = 'embedded';"] }),
  });
  assert.throws(() => assertCanonicalDonorArtifactBoundary("http", root), /removed donor|embed|source map/i);
});

test("rejects HTTP artifacts with the isolated AGA candidate-demo surface", () => {
  const root = fixture({
    "build-inputs.json": JSON.stringify({ profile: "http", inputs: ["src/entry/http.tsx"] }),
    "assets/http.js": "aga-candidate-demo AGACandidateDemo",
  });
  assert.throws(() => assertCanonicalDonorArtifactBoundary("http", root), /removed donor|candidate-demo/i);
});

test("rejects demo artifacts with supplemental source-map bodies", () => {
  const root = fixture({
    "build-inputs.json": JSON.stringify({ profile: "demo", inputs: ["src/entry/demo.tsx"] }),
    "index.html": "<html></html>",
    "assets/demo.js": "const demo = true;",
    "assets/demo.js.map": JSON.stringify({ sources: ["src/auth/session-client.ts"], sourcesContent: ["/v1/preprod/aga-demo-workspace/classification/query"] }),
  });
  assert.throws(() => assertCanonicalDonorArtifactBoundary("demo", root), /supplemental|source map/i);
});
