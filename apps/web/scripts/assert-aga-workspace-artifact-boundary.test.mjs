import assert from "node:assert/strict";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import test from "node:test";

import { assertAgaWorkspaceArtifact } from "./assert-aga-workspace-artifact-boundary.mjs";

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
  assert.doesNotThrow(() => assertAgaWorkspaceArtifact("demo", root));
});

test("rejects a demo artifact containing a supplemental workspace marker", () => {
  const root = fixture({
    "build-inputs.json": JSON.stringify({ profile: "demo", inputs: ["src/entry/demo.tsx"] }),
    "assets/demo.js": "\/v1\/preprod\/aga-demo-workspace\/classification\/query",
  });
  assert.throws(() => assertAgaWorkspaceArtifact("demo", root), /supplemental workspace/i);
});

test("requires the HTTP client and fixed supplemental route markers without embedded rows", () => {
  const root = fixture({
    "build-inputs.json": JSON.stringify({ profile: "http", inputs: ["src/entry/http.tsx", "src/backend/aga-demo-workspace.ts"] }),
    "index.html": "<html></html>",
    "assets/http.js": [
      "\/v1\/preprod\/aga-demo-workspace\/capability",
      "\/v1\/preprod\/aga-demo-workspace\/classification\/query",
      "AGADemoWorkspaceBackend",
      "classificationQuery",
    ].join("\n"),
    "assets/http.js.map": JSON.stringify({ sources: ["assets/http.js"], sourcesContent: [null] }),
  });
  assert.doesNotThrow(() => assertAgaWorkspaceArtifact("http", root));
});

test("rejects HTTP artifacts with embedded classification rows or source-map bodies", () => {
  const root = fixture({
    "build-inputs.json": JSON.stringify({ profile: "http", inputs: ["src/entry/http.tsx"] }),
    "assets/http.js": "\/v1\/preprod\/aga-demo-workspace\/capability \/v1\/preprod\/aga-demo-workspace\/classification\/query AGADemoWorkspaceBackend classificationQuery questionRef classificationRunId",
    "assets/http.js.map": JSON.stringify({ sources: ["src/backend/aga-demo-workspace.ts"], sourcesContent: ["const body = 'embedded';"] }),
  });
  assert.throws(() => assertAgaWorkspaceArtifact("http", root), /embed|source map/i);
});

test("rejects demo artifacts with supplemental source-map bodies", () => {
  const root = fixture({
    "build-inputs.json": JSON.stringify({ profile: "demo", inputs: ["src/entry/demo.tsx"] }),
    "index.html": "<html></html>",
    "assets/demo.js": "const demo = true;",
    "assets/demo.js.map": JSON.stringify({ sources: ["src/auth/session-client.ts"], sourcesContent: ["/v1/preprod/aga-demo-workspace/classification/query"] }),
  });
  assert.throws(() => assertAgaWorkspaceArtifact("demo", root), /supplemental|source map/i);
});
