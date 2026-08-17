/// <reference lib="webworker" />

import { CURRENT_OFFLINE_VERSIONS, sameOfflineVersionVector, type OfflineVersionVector } from "./offline/offline-version-contract";
import {
  APP_SHELL_ACTIVATION_POLICY,
  APP_SHELL_CANONICALIZATION_VERSION,
  APP_SHELL_MANIFEST_SCHEMA_VERSION,
  appShellDescriptorFromManifest,
  canonicalAppShellManifestInput,
  type AppShellFileRecord,
  type AppShellManifest,
  type AppShellPredecessorDescriptor,
} from "./offline/app-shell-manifest-contract";
import {
  isContentHashedAssetPath,
  isNetworkOnlyPath,
  isRegisteredApplicationRoute,
} from "./offline/app-route-policy";

export type AppShellRequestPolicy =
  | "app-shell-navigation"
  | "versioned-static-asset"
  | "network-only";

export interface AppShellRequestDescriptor {
  url: string;
  method: string;
  mode: string;
}

const EXPECTED_RELEASE_FINGERPRINT = "__AVIA_RELEASE_FINGERPRINT__";
const LEGACY_V9_PREDECESSOR: AppShellPredecessorDescriptor | null = /*__AVIA_LEGACY_PREDECESSOR__*/ null;
const APP_SHELL_MANIFEST_URL = "/app-shell-assets.json";
const VERIFIED_MARKER = "/__avia_app_shell_verified__";
const MAX_MANIFEST_BYTES = 512 * 1024;
const MAX_FILE_COUNT = 256;
const MAX_FILE_BYTES = 10 * 1024 * 1024;
const MAX_TOTAL_BYTES = 50 * 1024 * 1024;
const serviceWorkerScope = globalThis as unknown as ServiceWorkerGlobalScope;
const committedClientCaches = new Map<string, string>();
let installedManifest: AppShellManifest | null = null;
let activeCacheName: string | null = null;
let activeManifest: AppShellManifest | null = null;

const contentTypeByExtension: Record<string, string> = {
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
};

export function classifyAppShellRequest(
  request: AppShellRequestDescriptor,
  origin: string,
): AppShellRequestPolicy {
  if (request.method !== "GET") return "network-only";
  let url: URL;
  try {
    url = new URL(request.url);
  } catch {
    return "network-only";
  }
  if (url.origin !== origin || url.search || url.hash || isNetworkOnlyPath(url.pathname)) {
    return "network-only";
  }
  if (request.mode === "navigate") {
    return isRegisteredApplicationRoute(url.pathname) ? "app-shell-navigation" : "network-only";
  }
  if (isContentHashedAssetPath(url.pathname)) return "versioned-static-asset";
  if (url.pathname === "/demo-build.json") return "versioned-static-asset";
  return "network-only";
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return Boolean(value) && typeof value === "object" && !Array.isArray(value);
}

function exactKeys(value: Record<string, unknown>, keys: readonly string[], label: string): void {
  const expected = new Set(keys);
  for (const key of Object.keys(value)) {
    if (!expected.has(key)) throw new Error(`${label} contains unknown field ${key}`);
  }
  for (const key of keys) {
    if (!(key in value)) throw new Error(`${label} is missing ${key}`);
  }
}

function digestShape(value: unknown, label: string): asserts value is string;
function digestShape(value: unknown, label: string, nullable: true): asserts value is string | null;
function digestShape(value: unknown, label: string, nullable = false): asserts value is string | null {
  if (nullable && value === null) return;
  if (typeof value !== "string" || !/^sha256:[0-9a-f]{64}$/.test(value)) {
    throw new Error(`${label} must be a lowercase SHA-256 digest`);
  }
}

function vector(value: unknown, label: string): asserts value is OfflineVersionVector {
  if (!isRecord(value)) throw new Error(`${label} must be an object`);
  exactKeys(value, ["appShellVersion", "indexedDbSchemaVersion", "packageSchemaVersion", "syncProtocolVersion"], label);
  for (const key of Object.keys(CURRENT_OFFLINE_VERSIONS)) {
    const numberValue = value[key];
    if (!Number.isSafeInteger(numberValue) || (numberValue as number) < 1) throw new Error(`${label}.${key} must be a positive integer`);
  }
}

function descriptor(value: unknown, label: string, allowLegacyWorkerURL = false): AppShellPredecessorDescriptor {
  if (!isRecord(value)) throw new Error(`${label} must be an object`);
  exactKeys(value, ["lockDigest", "webImageReferenceDigest", "platformManifestDigest", "serviceWorkerURL", "serviceWorkerSha256", "appShellManifestSha256", "releaseFingerprint", "compatibility"], label);
  digestShape(value.lockDigest, `${label}.lockDigest`, true);
  digestShape(value.webImageReferenceDigest, `${label}.webImageReferenceDigest`, true);
  digestShape(value.platformManifestDigest, `${label}.platformManifestDigest`, true);
  if (typeof value.serviceWorkerURL !== "string" || (value.serviceWorkerURL !== "/sw.js" && !(allowLegacyWorkerURL && value.serviceWorkerURL === "/sw.js?v=9"))) throw new Error(`${label}.serviceWorkerURL is not an allowed stable worker URL`);
  digestShape(value.serviceWorkerSha256, `${label}.serviceWorkerSha256`, true);
  digestShape(value.appShellManifestSha256, `${label}.appShellManifestSha256`, true);
  digestShape(value.releaseFingerprint, `${label}.releaseFingerprint`, true);
  vector(value.compatibility, `${label}.compatibility`);
  return {
    lockDigest: value.lockDigest as string | null,
    webImageReferenceDigest: value.webImageReferenceDigest as string | null,
    platformManifestDigest: value.platformManifestDigest as string | null,
    serviceWorkerURL: value.serviceWorkerURL,
    serviceWorkerSha256: value.serviceWorkerSha256 as string | null,
    appShellManifestSha256: value.appShellManifestSha256 as string | null,
    releaseFingerprint: value.releaseFingerprint as string | null,
    compatibility: value.compatibility as OfflineVersionVector,
  };
}

function contentTypeForURL(url: string): string {
  if (url === "/" || url === "/index.html") return "text/html";
  if (url === "/demo-build.json") return "application/json";
  const extension = url.split(".").pop()?.toLowerCase() ?? "";
  return contentTypeByExtension[extension] ?? "";
}

function validateFileRecord(value: unknown, index: number, profile: AppShellManifest["profile"]): AppShellFileRecord {
  const label = `files[${index}]`;
  if (!isRecord(value)) throw new Error(`${label} must be an object`);
  exactKeys(value, ["url", "sha256", "byteSize", "contentType"], label);
  if (typeof value.url !== "string" || value.url !== new URL(value.url, serviceWorkerScope.location.origin).pathname || value.url.includes("?") || value.url.includes("#") || value.url.includes("\\") || /%2f|%5c|\/\.\.?(?:\/|$)/i.test(value.url)) {
    throw new Error(`${label}.url is not a canonical same-origin path`);
  }
  const allowed = value.url === "/" || value.url === "/index.html" || (profile === "demo" && value.url === "/demo-build.json") || isContentHashedAssetPath(value.url);
  if (!allowed) throw new Error(`${label}.url is not an allowed app-shell file`);
  digestShape(value.sha256, `${label}.sha256`);
  if (!Number.isSafeInteger(value.byteSize) || (value.byteSize as number) < 1 || (value.byteSize as number) > MAX_FILE_BYTES) throw new Error(`${label}.byteSize is outside the bounded range`);
  if (typeof value.contentType !== "string" || value.contentType.toLowerCase() !== contentTypeForURL(value.url)) throw new Error(`${label}.contentType does not match its URL`);
  return { url: value.url, sha256: value.sha256, byteSize: value.byteSize as number, contentType: value.contentType.toLowerCase() };
}

function parseManifest(value: unknown): AppShellManifest {
  if (!isRecord(value)) throw new Error("app-shell manifest must be an object");
  exactKeys(value, ["schemaVersion", "canonicalizationVersion", "profile", "releaseFingerprint", "activationPolicy", "compatibility", "predecessor", "releaseDescriptor", "worker", "files"], "manifest");
  if (value.schemaVersion !== APP_SHELL_MANIFEST_SCHEMA_VERSION || value.canonicalizationVersion !== APP_SHELL_CANONICALIZATION_VERSION) throw new Error("unsupported app-shell manifest schema");
  if (value.profile !== "demo" && value.profile !== "http") throw new Error("unsupported app-shell profile");
  digestShape(value.releaseFingerprint, "manifest.releaseFingerprint");
  if (value.activationPolicy !== APP_SHELL_ACTIVATION_POLICY) throw new Error("unsupported app-shell activation policy");
  vector(value.compatibility, "manifest.compatibility");
  const predecessor = value.predecessor === null ? null : descriptor(value.predecessor, "manifest.predecessor", true);
  const releaseDescriptor = descriptor(value.releaseDescriptor, "manifest.releaseDescriptor");
  if (!isRecord(value.worker)) throw new Error("manifest.worker must be an object");
  exactKeys(value.worker, ["url", "sha256", "templateSha256"], "manifest.worker");
  if (value.worker.url !== "/sw.js") throw new Error("manifest.worker.url must be /sw.js");
  digestShape(value.worker.sha256, "manifest.worker.sha256");
  digestShape(value.worker.templateSha256, "manifest.worker.templateSha256");
  if (!Array.isArray(value.files) || value.files.length === 0 || value.files.length > MAX_FILE_COUNT) throw new Error("manifest.files is outside the bounded range");
  const files = value.files.map((file, index) => validateFileRecord(file, index, value.profile as AppShellManifest["profile"]));
  const sorted = [...files].sort((left, right) => left.url < right.url ? -1 : left.url > right.url ? 1 : 0);
  if (files.some((file, index) => file.url !== sorted[index]?.url) || new Set(files.map((file) => file.url)).size !== files.length) throw new Error("manifest.files must be unique and sorted");
  if (!files.some((file) => file.url === "/") || !files.some((file) => file.url === "/index.html")) throw new Error("manifest must contain both / and /index.html");
  if (value.profile === "demo" && !files.some((file) => file.url === "/demo-build.json")) throw new Error("demo manifest must contain /demo-build.json");
  return {
    schemaVersion: APP_SHELL_MANIFEST_SCHEMA_VERSION,
    canonicalizationVersion: APP_SHELL_CANONICALIZATION_VERSION,
    profile: value.profile,
    releaseFingerprint: value.releaseFingerprint,
    activationPolicy: APP_SHELL_ACTIVATION_POLICY,
    compatibility: value.compatibility as OfflineVersionVector,
    predecessor,
    releaseDescriptor,
    worker: { url: value.worker.url, sha256: value.worker.sha256, templateSha256: value.worker.templateSha256 },
    files,
  };
}

async function sha256(bytes: ArrayBuffer | Uint8Array): Promise<string> {
  const input = bytes instanceof Uint8Array ? bytes.slice().buffer as ArrayBuffer : bytes;
  const digest = await crypto.subtle.digest("SHA-256", input);
  return `sha256:${Array.from(new Uint8Array(digest), (byte) => byte.toString(16).padStart(2, "0")).join("")}`;
}

async function verifyManifestFingerprint(manifest: AppShellManifest): Promise<void> {
  const input = canonicalAppShellManifestInput({ ...manifest, worker: { templateSha256: manifest.worker.templateSha256 } });
  if (await sha256(input) !== manifest.releaseFingerprint) throw new Error("app-shell manifest fingerprint mismatch");
}

function candidateCacheName(manifest: AppShellManifest): string {
  return `aviasurveil360-app-shell-${manifest.releaseFingerprint.slice("sha256:".length)}`;
}

async function fetchExact(url: string, expectedType: string, expectedSize?: number, expectedDigest?: string): Promise<Response> {
  const absolute = new URL(url, serviceWorkerScope.location.origin).href;
  const response = await fetch(absolute, { cache: "no-store", credentials: "same-origin", redirect: "error" });
  if (response.status !== 200 || response.type !== "basic" || response.url !== absolute || response.redirected) throw new Error(`app-shell response failed validation: ${url}`);
  const actualType = (response.headers.get("content-type") ?? "").split(";", 1)[0].trim().toLowerCase();
  if (actualType !== expectedType.toLowerCase()) throw new Error(`app-shell content type mismatch: ${url}`);
  if (expectedSize === undefined && expectedDigest === undefined) return response;
  const bytes = await response.clone().arrayBuffer();
  if (expectedSize !== undefined && bytes.byteLength !== expectedSize) throw new Error(`app-shell byte size mismatch: ${url}`);
  if (expectedDigest !== undefined && await sha256(bytes) !== expectedDigest) throw new Error(`app-shell digest mismatch: ${url}`);
  return response;
}

async function installAppShell(): Promise<AppShellManifest> {
  const manifestAbsolute = new URL(APP_SHELL_MANIFEST_URL, serviceWorkerScope.location.origin).href;
  const manifestResponse = await fetch(manifestAbsolute, { cache: "no-store", credentials: "same-origin", redirect: "error" });
  if (manifestResponse.status !== 200 || manifestResponse.type !== "basic" || manifestResponse.url !== manifestAbsolute || manifestResponse.redirected) throw new Error("app-shell manifest response failed validation");
  const manifestType = (manifestResponse.headers.get("content-type") ?? "").split(";", 1)[0].trim().toLowerCase();
  if (manifestType !== "application/json") throw new Error("app-shell manifest must be JSON");
  const manifestBytes = await manifestResponse.arrayBuffer();
  if (manifestBytes.byteLength > MAX_MANIFEST_BYTES) throw new Error("app-shell manifest is too large");
  let manifest: AppShellManifest;
  try {
    manifest = parseManifest(JSON.parse(new TextDecoder("utf-8", { fatal: true }).decode(manifestBytes)));
  } catch (error) {
    throw new Error(`invalid app-shell manifest: ${error instanceof Error ? error.message : "parse failure"}`);
  }
  await verifyManifestFingerprint(manifest);
  if (manifest.releaseFingerprint !== EXPECTED_RELEASE_FINGERPRINT) throw new Error("app-shell worker/manifest fingerprint mismatch");
  if (!sameOfflineVersionVector(manifest.compatibility, CURRENT_OFFLINE_VERSIONS)) throw new Error("app-shell compatibility vector is not exact");
  const totalBytes = manifest.files.reduce((sum, file) => sum + file.byteSize, 0);
  if (totalBytes > MAX_TOTAL_BYTES) throw new Error("app-shell file map is too large");
  const cacheName = candidateCacheName(manifest);
  const cache = await caches.open(cacheName);
  try {
    for (const file of manifest.files) {
      const response = await fetchExact(file.url, file.contentType, file.byteSize, file.sha256);
      await cache.put(new URL(file.url, serviceWorkerScope.location.origin).href, response);
    }
    await cache.put(new URL(VERIFIED_MARKER, serviceWorkerScope.location.origin).href, new Response(JSON.stringify(manifest), { headers: { "content-type": "application/json" } }));
    installedManifest = manifest;
    return manifest;
  } catch (error) {
    await caches.delete(cacheName);
    throw error;
  }
}

async function committedManifests(): Promise<Array<{ cacheName: string; manifest: AppShellManifest }>> {
  const result: Array<{ cacheName: string; manifest: AppShellManifest }> = [];
  for (const cacheName of await caches.keys()) {
    if (!cacheName.startsWith("aviasurveil360-app-shell-")) continue;
    const response = await (await caches.open(cacheName)).match(new URL(VERIFIED_MARKER, serviceWorkerScope.location.origin).href);
    if (!response) continue;
    try {
      result.push({ cacheName, manifest: parseManifest(await response.json()) });
    } catch {
      // Untrusted or partial cache entries are ignored and are never selected.
    }
  }
  return result;
}

function sameDescriptor(left: AppShellPredecessorDescriptor, right: AppShellPredecessorDescriptor): boolean {
  return JSON.stringify(left) === JSON.stringify(right);
}

async function canActivate(manifest: AppShellManifest): Promise<boolean> {
  if (!sameOfflineVersionVector(manifest.compatibility, CURRENT_OFFLINE_VERSIONS)) return false;
  const committed = (await committedManifests()).filter(({ cacheName }) => cacheName !== candidateCacheName(manifest));
  const predecessor = manifest.predecessor;
  if (predecessor === null) return committed.length === 0;
  if (LEGACY_V9_PREDECESSOR && sameDescriptor(predecessor, LEGACY_V9_PREDECESSOR)) return true;
  return committed.some(({ manifest: active }) => sameDescriptor(predecessor, appShellDescriptorFromManifest(active)));
}

async function serveAppShellNavigation(request: Request): Promise<Response> {
  const cacheName = activeCacheName ?? (await committedManifests())[0]?.cacheName;
  if (!cacheName) return fetch(request, { cache: "no-store" });
  const cache = await caches.open(cacheName);
  return (await cache.match(request)) ?? (await cache.match(new URL("/", serviceWorkerScope.location.origin).href)) ?? (await fetch(request, { cache: "no-store" }));
}

async function serveVersionedAsset(request: Request, clientId: string): Promise<Response> {
  const committed = await committedManifests();
  const preferred = committedClientCaches.get(clientId);
  const order = [preferred, activeCacheName, ...committed.map(({ cacheName }) => cacheName)].filter((name, index, all): name is string => Boolean(name) && all.indexOf(name) === index);
  for (const cacheName of order) {
    const match = await (await caches.open(cacheName)).match(request);
    if (match) return match;
  }
  return fetch(request, { cache: "no-store" });
}

if (typeof serviceWorkerScope.addEventListener === "function" && "registration" in serviceWorkerScope && typeof caches !== "undefined") {
  serviceWorkerScope.addEventListener("install", (event: ExtendableEvent) => {
    event.waitUntil((async () => {
      const manifest = await installAppShell();
      if (await canActivate(manifest)) serviceWorkerScope.skipWaiting();
    })());
  });

  serviceWorkerScope.addEventListener("activate", (event: ExtendableEvent) => {
    event.waitUntil((async () => {
      if (!installedManifest || !(await canActivate(installedManifest))) throw new Error("app-shell predecessor/vector activation gate failed");
      activeCacheName = candidateCacheName(installedManifest);
      activeManifest = installedManifest;
      await serviceWorkerScope.clients.claim();
    })());
  });

  serviceWorkerScope.addEventListener("message", (event: ExtendableMessageEvent) => {
    if (event.data?.type === "avia:app-shell-client-ready") {
      const source = event.source as Client | null;
      if (source?.id && typeof event.data.fingerprint === "string") committedClientCaches.set(source.id, `aviasurveil360-app-shell-${event.data.fingerprint.replace(/^sha256:/, "")}`);
      if (source && activeManifest) source.postMessage({ type: "avia:app-shell-activation", fingerprint: activeManifest.releaseFingerprint });
    }
  });

  serviceWorkerScope.addEventListener("fetch", (event: FetchEvent) => {
    const policy = classifyAppShellRequest(event.request, serviceWorkerScope.location.origin);
    if (policy === "app-shell-navigation") event.respondWith(serveAppShellNavigation(event.request));
    if (policy === "versioned-static-asset") event.respondWith(serveVersionedAsset(event.request, event.clientId));
  });
}
