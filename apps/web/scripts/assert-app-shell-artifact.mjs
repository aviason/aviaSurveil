#!/usr/bin/env node
import assert from "node:assert/strict";
import { createHash } from "node:crypto";
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const CURRENT_VECTOR = Object.freeze({
  appShellVersion: 9,
  indexedDbSchemaVersion: 2,
  packageSchemaVersion: 1,
  syncProtocolVersion: 1,
});
export const BRAND_ASSET_BASENAMES = Object.freeze([
  "aviasurveil360-mark",
  "airspace-texture",
  "DMSans-Variable",
  "air-traffic-control",
  "buildings",
  "arrow-right",
  "wallet",
  "seal-check",
  "gear",
  "globe-hemisphere-west",
  "compass",
  "bank",
]);
const CONTENT_TYPES = Object.freeze({
  html: "text/html",
  json: "application/json",
  css: "text/css",
  js: "text/javascript",
  map: "application/json",
  svg: "image/svg+xml",
  png: "image/png",
  jpg: "image/jpeg",
  jpeg: "image/jpeg",
  webp: "image/webp",
  ttf: "font/ttf",
  woff: "font/woff",
  woff2: "font/woff2",
});

function sha256(bytes) {
  return `sha256:${createHash("sha256").update(bytes).digest("hex")}`;
}

function appendString(parts, value) {
  const bytes = Buffer.from(value, "utf8");
  const prefix = Buffer.alloc(4);
  prefix.writeUInt32BE(bytes.length, 0);
  parts.push(prefix, bytes);
}

function appendInt(parts, value) {
  const buffer = Buffer.alloc(8);
  buffer.writeBigInt64BE(BigInt(value), 0);
  parts.push(buffer);
}

function appendNullable(parts, value) {
  appendString(parts, value === null ? "null" : "string");
  if (value !== null) appendString(parts, value);
}

function compareCodePoints(left, right) {
  const a = Array.from(left, (value) => value.codePointAt(0));
  const b = Array.from(right, (value) => value.codePointAt(0));
  for (let index = 0; index < Math.min(a.length, b.length); index += 1) {
    if (a[index] !== b[index]) return a[index] - b[index];
  }
  return a.length - b.length;
}

function canonicalBytes(manifest) {
  const parts = [];
  appendString(parts, "AVIA_APP_SHELL_MANIFEST_V2");
  appendInt(parts, manifest.schemaVersion);
  appendInt(parts, manifest.canonicalizationVersion);
  appendString(parts, manifest.profile);
  appendString(parts, manifest.activationPolicy);
  for (const key of Object.keys(CURRENT_VECTOR)) appendInt(parts, manifest.compatibility[key]);
  appendString(parts, manifest.predecessor === null ? "null" : "descriptor");
  if (manifest.predecessor) {
    for (const key of ["lockDigest", "webImageReferenceDigest", "platformManifestDigest"]) appendNullable(parts, manifest.predecessor[key]);
    appendString(parts, manifest.predecessor.serviceWorkerURL);
    for (const key of ["serviceWorkerSha256", "appShellManifestSha256", "releaseFingerprint"]) appendNullable(parts, manifest.predecessor[key]);
    for (const key of Object.keys(CURRENT_VECTOR)) appendInt(parts, manifest.predecessor.compatibility[key]);
  }
  appendString(parts, manifest.worker.templateSha256);
  const files = [...manifest.files].sort((left, right) => compareCodePoints(left.url, right.url));
  appendInt(parts, files.length);
  for (const file of files) {
    appendString(parts, file.url);
    appendString(parts, file.sha256);
    appendInt(parts, file.byteSize);
    appendString(parts, file.contentType.toLowerCase());
  }
  return Buffer.concat(parts);
}

function inventory(directory, prefix = "") {
  return fs.readdirSync(directory, { withFileTypes: true }).flatMap((entry) => {
    const relative = path.posix.join(prefix, entry.name);
    return entry.isDirectory() ? inventory(path.join(directory, entry.name), relative) : [relative];
  });
}

function expectedContentType(url) {
  if (url === "/" || url === "/index.html") return CONTENT_TYPES.html;
  if (url === "/demo-build.json") return CONTENT_TYPES.json;
  const extension = url.split(".").pop()?.toLowerCase();
  return CONTENT_TYPES[extension];
}

function assertDescriptor(value, label, allowLegacyWorkerURL = false) {
  assert.equal(typeof value, "object", `${label} must be an object`);
  assert.ok(value === null || value.serviceWorkerURL === "/sw.js" || (allowLegacyWorkerURL && value.serviceWorkerURL === "/sw.js?v=9"), `${label}.serviceWorkerURL is invalid`);
}

export function assertAppShellArtifact(suppliedPath) {
  const artifactRoot = path.resolve(suppliedPath);
  assert.ok(fs.existsSync(artifactRoot), `App-shell artifact directory is missing: ${artifactRoot}`);
  const files = inventory(artifactRoot);
  for (const required of ["index.html", "sw.js", "app-shell-assets.json"]) assert.ok(files.includes(required), `App-shell artifact is missing ${required}`);
  const manifest = JSON.parse(fs.readFileSync(path.join(artifactRoot, "app-shell-assets.json"), "utf8"));
  assert.equal(manifest.schemaVersion, 2);
  assert.equal(manifest.canonicalizationVersion, 1);
  assert.ok(["demo", "http"].includes(manifest.profile));
  assert.equal(manifest.activationPolicy, "automatic-exact-vector");
  assert.deepEqual(manifest.compatibility, CURRENT_VECTOR);
  assert.match(manifest.releaseFingerprint, /^sha256:[0-9a-f]{64}$/);
  assert.ok(Array.isArray(manifest.files) && manifest.files.length > 0);
  assert.ok(manifest.predecessor === null || typeof manifest.predecessor === "object");
  if (manifest.predecessor !== null) assertDescriptor(manifest.predecessor, "manifest.predecessor", true);
  assertDescriptor(manifest.releaseDescriptor, "manifest.releaseDescriptor");
  assert.equal(manifest.releaseDescriptor.serviceWorkerURL, "/sw.js");
  assert.equal(manifest.worker.url, "/sw.js");
  assert.match(manifest.worker.sha256, /^sha256:[0-9a-f]{64}$/);
  assert.match(manifest.worker.templateSha256, /^sha256:[0-9a-f]{64}$/);

  const seen = new Set();
  let total = 0;
  for (const record of manifest.files) {
    assert.ok(!seen.has(record.url), `duplicate app-shell file: ${record.url}`);
    seen.add(record.url);
    assert.ok(record.url === "/" || record.url === "/index.html" || (manifest.profile === "demo" && record.url === "/demo-build.json") || /^\/assets\/[A-Za-z0-9_.-]+-[A-Za-z0-9_-]{6,}\.(?:css|js|map|svg|png|jpg|jpeg|webp|ttf|woff|woff2)$/.test(record.url), `invalid app-shell file URL: ${record.url}`);
    assert.match(record.sha256, /^sha256:[0-9a-f]{64}$/);
    assert.equal(typeof record.byteSize, "number");
    assert.equal(record.contentType, expectedContentType(record.url));
    const relative = record.url === "/" ? "index.html" : record.url.slice(1);
    const bytes = fs.readFileSync(path.join(artifactRoot, relative));
    assert.equal(bytes.byteLength, record.byteSize, `byte size mismatch: ${record.url}`);
    assert.equal(sha256(bytes), record.sha256, `digest mismatch: ${record.url}`);
    total += bytes.byteLength;
  }
  assert.ok(seen.has("/") && seen.has("/index.html"));
  if (manifest.profile === "demo") assert.ok(seen.has("/demo-build.json"));
  assert.ok(total <= 50 * 1024 * 1024);

  const workerBytes = fs.readFileSync(path.join(artifactRoot, "sw.js"));
  const worker = workerBytes.toString("utf8");
  assert.equal(sha256(workerBytes), manifest.worker.sha256);
  const workerTemplate = worker.replace(manifest.releaseFingerprint, "__AVIA_RELEASE_FINGERPRINT__");
  assert.equal(sha256(Buffer.from(workerTemplate)), manifest.worker.templateSha256);
  assert.equal(sha256(canonicalBytes(manifest)), manifest.releaseFingerprint);
  assert.ok(worker.includes(manifest.releaseFingerprint));
  assert.match(worker, /skipWaiting/);
  assert.match(worker, /clients\.claim/);
  assert.doesNotMatch(worker, /indexedDB\.deleteDatabase/);
  assert.match(worker, /force-window-client-navigation-v1/);
  assert.match(worker, /\.navigate\(/);
  assert.doesNotMatch(worker, /await\s+[\w$.]+\.navigate\(/);
  assert.doesNotMatch(worker, /cache\.addAll/);
  return { files: files.length, assets: manifest.files.length, profile: manifest.profile, releaseFingerprint: manifest.releaseFingerprint };
}

const invokedPath = process.argv[1] ? path.resolve(process.argv[1]) : null;
if (invokedPath === fileURLToPath(import.meta.url)) {
  const result = assertAppShellArtifact(process.argv[2] ?? "dist/demo");
  console.log(`app-shell-artifact-scan: ok (${result.profile}, ${result.files} files, ${result.assets} app-shell records, ${result.releaseFingerprint})`);
}
