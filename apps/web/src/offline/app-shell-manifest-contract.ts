import type { OfflineVersionVector } from "./offline-version-contract";

export const APP_SHELL_MANIFEST_SCHEMA_VERSION = 2;
export const APP_SHELL_CANONICALIZATION_VERSION = 1;
export const APP_SHELL_ACTIVATION_POLICY = "automatic-exact-vector" as const;
export const APP_SHELL_FINGERPRINT_PLACEHOLDER = "__AVIA_RELEASE_FINGERPRINT__";

export interface AppShellFileRecord {
  url: string;
  sha256: string;
  byteSize: number;
  contentType: string;
}

export interface AppShellPredecessorDescriptor {
  lockDigest: string | null;
  webImageReferenceDigest: string | null;
  platformManifestDigest: string | null;
  serviceWorkerURL: string;
  serviceWorkerSha256: string | null;
  appShellManifestSha256: string | null;
  releaseFingerprint: string | null;
  compatibility: OfflineVersionVector;
}

export interface AppShellWorkerDescriptor {
  url: string;
  sha256: string;
  templateSha256: string;
}

export interface AppShellManifest {
  schemaVersion: typeof APP_SHELL_MANIFEST_SCHEMA_VERSION;
  canonicalizationVersion: typeof APP_SHELL_CANONICALIZATION_VERSION;
  profile: "demo" | "http";
  releaseFingerprint: string;
  activationPolicy: typeof APP_SHELL_ACTIVATION_POLICY;
  compatibility: OfflineVersionVector;
  predecessor: AppShellPredecessorDescriptor | null;
  releaseDescriptor: AppShellPredecessorDescriptor;
  worker: AppShellWorkerDescriptor;
  files: AppShellFileRecord[];
}

function appendLengthPrefixed(parts: Uint8Array[], value: string): void {
  const bytes = new TextEncoder().encode(value);
  const length = new Uint8Array(4);
  new DataView(length.buffer).setUint32(0, bytes.byteLength, false);
  parts.push(length, bytes);
}

function appendInteger(parts: Uint8Array[], value: number): void {
  const bytes = new Uint8Array(8);
  new DataView(bytes.buffer).setBigInt64(0, BigInt(value), false);
  parts.push(bytes);
}

function appendNullableString(parts: Uint8Array[], value: string | null): void {
  appendLengthPrefixed(parts, value === null ? "null" : "string");
  if (value !== null) appendLengthPrefixed(parts, value);
}

function compareUnicodeCodePoints(left: string, right: string): number {
  const leftPoints = Array.from(left, (value) => value.codePointAt(0) ?? 0);
  const rightPoints = Array.from(right, (value) => value.codePointAt(0) ?? 0);
  const length = Math.min(leftPoints.length, rightPoints.length);
  for (let index = 0; index < length; index += 1) {
    if (leftPoints[index] !== rightPoints[index]) return leftPoints[index] - rightPoints[index];
  }
  return leftPoints.length - rightPoints.length;
}

export function canonicalAppShellManifestInput(manifest: Omit<AppShellManifest, "releaseFingerprint" | "worker"> & { worker: Pick<AppShellWorkerDescriptor, "templateSha256"> }): Uint8Array {
  const parts: Uint8Array[] = [];
  appendLengthPrefixed(parts, "AVIA_APP_SHELL_MANIFEST_V2");
  appendInteger(parts, manifest.schemaVersion);
  appendInteger(parts, manifest.canonicalizationVersion);
  appendLengthPrefixed(parts, manifest.profile);
  appendLengthPrefixed(parts, manifest.activationPolicy);
  appendInteger(parts, manifest.compatibility.appShellVersion);
  appendInteger(parts, manifest.compatibility.indexedDbSchemaVersion);
  appendInteger(parts, manifest.compatibility.packageSchemaVersion);
  appendInteger(parts, manifest.compatibility.syncProtocolVersion);
  const predecessor = manifest.predecessor;
  appendLengthPrefixed(parts, predecessor === null ? "null" : "descriptor");
  if (predecessor) {
    appendNullableString(parts, predecessor.lockDigest);
    appendNullableString(parts, predecessor.webImageReferenceDigest);
    appendNullableString(parts, predecessor.platformManifestDigest);
    appendLengthPrefixed(parts, predecessor.serviceWorkerURL);
    appendNullableString(parts, predecessor.serviceWorkerSha256);
    appendNullableString(parts, predecessor.appShellManifestSha256);
    appendNullableString(parts, predecessor.releaseFingerprint);
    appendInteger(parts, predecessor.compatibility.appShellVersion);
    appendInteger(parts, predecessor.compatibility.indexedDbSchemaVersion);
    appendInteger(parts, predecessor.compatibility.packageSchemaVersion);
    appendInteger(parts, predecessor.compatibility.syncProtocolVersion);
  }
  appendLengthPrefixed(parts, manifest.worker.templateSha256);
  const files = [...manifest.files].sort((left, right) => compareUnicodeCodePoints(left.url, right.url));
  appendInteger(parts, files.length);
  for (const file of files) {
    appendLengthPrefixed(parts, file.url);
    appendLengthPrefixed(parts, file.sha256);
    appendInteger(parts, file.byteSize);
    appendLengthPrefixed(parts, file.contentType.toLowerCase());
  }
  const total = parts.reduce((sum, part) => sum + part.byteLength, 0);
  const result = new Uint8Array(total);
  let offset = 0;
  for (const part of parts) {
    result.set(part, offset);
    offset += part.byteLength;
  }
  return result;
}

export function appShellDescriptorFromManifest(manifest: AppShellManifest): AppShellPredecessorDescriptor {
	return manifest.releaseDescriptor;
}
