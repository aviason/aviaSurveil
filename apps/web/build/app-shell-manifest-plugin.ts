import { createHash } from "node:crypto";
import { readdirSync, readFileSync, writeFileSync } from "node:fs";
import { resolve } from "node:path";
import type { Plugin, ResolvedConfig } from "vite";

import { CURRENT_OFFLINE_VERSIONS } from "../src/offline/offline-version-contract";
import {
  APP_SHELL_ACTIVATION_POLICY,
  APP_SHELL_CANONICALIZATION_VERSION,
  APP_SHELL_MANIFEST_SCHEMA_VERSION,
  canonicalAppShellManifestInput,
  type AppShellFileRecord,
  type AppShellManifest,
  type AppShellPredecessorDescriptor,
} from "../src/offline/app-shell-manifest-contract";

const RELEASE_FINGERPRINT_PLACEHOLDER = "__AVIA_RELEASE_FINGERPRINT__";
const LEGACY_PREDECESSOR_PLACEHOLDER = "__AVIA_LEGACY_PREDECESSOR_BASE64__";

function digest(bytes: Uint8Array): string {
  return `sha256:${createHash("sha256").update(bytes).digest("hex")}`;
}

function textBytes(value: string): Uint8Array {
  return new TextEncoder().encode(value);
}

function contentType(fileName: string): string {
  if (fileName === "index.html") return "text/html";
  if (fileName === "demo-build.json") return "application/json";
  const extension = fileName.split(".").pop()?.toLowerCase() ?? "";
  return ({
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
  } as Record<string, string>)[extension] ?? "application/octet-stream";
}

function compareCodePoints(left: string, right: string): number {
  const leftPoints = Array.from(left, (value) => value.codePointAt(0) ?? 0);
  const rightPoints = Array.from(right, (value) => value.codePointAt(0) ?? 0);
  for (let index = 0; index < Math.min(leftPoints.length, rightPoints.length); index += 1) {
    if (leftPoints[index] !== rightPoints[index]) return leftPoints[index] - rightPoints[index];
  }
  return leftPoints.length - rightPoints.length;
}

function parseDescriptor(raw: string | undefined, fallback: AppShellPredecessorDescriptor): AppShellPredecessorDescriptor {
  if (!raw?.trim()) return fallback;
  const value = JSON.parse(raw) as Partial<AppShellPredecessorDescriptor>;
  return {
    lockDigest: value.lockDigest ?? null,
    webImageReferenceDigest: value.webImageReferenceDigest ?? null,
    platformManifestDigest: value.platformManifestDigest ?? null,
    serviceWorkerURL: value.serviceWorkerURL ?? "/sw.js",
    serviceWorkerSha256: value.serviceWorkerSha256 ?? null,
    appShellManifestSha256: value.appShellManifestSha256 ?? null,
    releaseFingerprint: value.releaseFingerprint ?? null,
    compatibility: value.compatibility ?? CURRENT_OFFLINE_VERSIONS,
  };
}

function defaultDescriptor(): AppShellPredecessorDescriptor {
  return {
    lockDigest: null,
    webImageReferenceDigest: null,
    platformManifestDigest: null,
    serviceWorkerURL: "/sw.js",
    serviceWorkerSha256: null,
    appShellManifestSha256: null,
    releaseFingerprint: null,
    compatibility: CURRENT_OFFLINE_VERSIONS,
  };
}

function legacyPredecessorBase64(): string {
  const raw = process.env.AVIA_APP_SHELL_LEGACY_PREDECESSOR_JSON?.trim();
  if (!raw) return LEGACY_PREDECESSOR_PLACEHOLDER;
  return Buffer.from(JSON.stringify(JSON.parse(raw)), "utf8").toString("base64");
}

function inventory(root: string, prefix = ""): string[] {
  return readdirSync(resolve(root, prefix), { withFileTypes: true }).flatMap((entry) => {
    const relative = prefix ? `${prefix}/${entry.name}` : entry.name;
    return entry.isDirectory() ? inventory(root, relative) : [relative];
  });
}

function filesFromDirectory(root: string, profile: "demo" | "http"): AppShellFileRecord[] {
  const records = new Map<string, AppShellFileRecord>();
  for (const relative of inventory(root)) {
    if (relative === "sw.js" || relative === "app-shell-assets.json" || relative === "build-inputs.json" || relative.startsWith(".vite/")) continue;
    if (relative !== "index.html" && relative !== "demo-build.json" && !relative.startsWith("assets/")) continue;
    if (relative.startsWith("assets/") && !/\.(?:css|js|map|svg|png|jpg|jpeg|webp|ttf|woff|woff2)$/.test(relative)) continue;
    if (profile === "http" && relative === "demo-build.json") continue;
    const fileBytes = new Uint8Array(readFileSync(resolve(root, relative)));
    const url = `/${relative}`;
    const record: AppShellFileRecord = { url, sha256: digest(fileBytes), byteSize: fileBytes.byteLength, contentType: contentType(relative) };
    records.set(url, record);
    if (relative === "index.html") records.set("/", { ...record, url: "/" });
  }
  return [...records.values()].sort((left, right) => compareCodePoints(left.url, right.url));
}

export function createAppShellManifestPlugin(profile: "demo" | "http"): Plugin {
  let outputRoot = "";
  return {
    name: "aviasurveil360-app-shell-manifest",
    configResolved(config: ResolvedConfig) {
      outputRoot = resolve(config.root, config.build.outDir);
    },
    writeBundle() {
      if (!outputRoot) throw new Error("app-shell output root is unavailable");
      const workerPath = resolve(outputRoot, "sw.js");
      let workerTemplate = readFileSync(workerPath, "utf8");
      const legacyPlaceholderCount = workerTemplate.split(LEGACY_PREDECESSOR_PLACEHOLDER).length - 1;
      if (legacyPlaceholderCount !== 1) throw new Error(`app-shell worker must contain exactly one legacy predecessor placeholder, found ${legacyPlaceholderCount}`);
      workerTemplate = workerTemplate.replace(LEGACY_PREDECESSOR_PLACEHOLDER, legacyPredecessorBase64());
      const placeholderCount = workerTemplate.split(RELEASE_FINGERPRINT_PLACEHOLDER).length - 1;
      if (placeholderCount !== 1) throw new Error(`app-shell worker must contain exactly one fingerprint placeholder, found ${placeholderCount}`);
      const workerTemplateSha256 = digest(textBytes(workerTemplate));
      const predecessorRaw = process.env.AVIA_APP_SHELL_PREDECESSOR_JSON?.trim();
      const predecessor = predecessorRaw ? (JSON.parse(predecessorRaw) as AppShellPredecessorDescriptor) : null;
      const releaseDescriptor = parseDescriptor(process.env.AVIA_APP_SHELL_RELEASE_DESCRIPTOR_JSON, defaultDescriptor());
      const manifestBase = {
        schemaVersion: APP_SHELL_MANIFEST_SCHEMA_VERSION,
        canonicalizationVersion: APP_SHELL_CANONICALIZATION_VERSION,
        profile,
        activationPolicy: APP_SHELL_ACTIVATION_POLICY,
        compatibility: CURRENT_OFFLINE_VERSIONS,
        predecessor,
        releaseDescriptor,
        worker: { url: "/sw.js", sha256: "sha256:" + "0".repeat(64), templateSha256: workerTemplateSha256 },
        files: filesFromDirectory(outputRoot, profile),
      } satisfies Omit<AppShellManifest, "releaseFingerprint">;
      const fingerprint = digest(canonicalAppShellManifestInput(manifestBase));
      const finalWorker = workerTemplate.replace(RELEASE_FINGERPRINT_PLACEHOLDER, fingerprint);
      writeFileSync(workerPath, finalWorker);
      const workerSha256 = digest(textBytes(finalWorker));
      const manifest: AppShellManifest = {
        ...manifestBase,
        releaseFingerprint: fingerprint,
        releaseDescriptor: {
          ...releaseDescriptor,
          serviceWorkerURL: "/sw.js",
          serviceWorkerSha256: workerSha256,
          releaseFingerprint: fingerprint,
          compatibility: CURRENT_OFFLINE_VERSIONS,
        },
        worker: { url: "/sw.js", sha256: workerSha256, templateSha256: workerTemplateSha256 },
      };
      writeFileSync(resolve(outputRoot, "app-shell-assets.json"), `${JSON.stringify(manifest, null, 2)}\n`);
    },
  };
}
