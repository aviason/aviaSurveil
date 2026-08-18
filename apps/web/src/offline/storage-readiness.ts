import type { InspectionPackage, OfflineGrant } from "../backend/backend";
import {
  getBrowserOfflineFieldDatabase,
  OFFLINE_FIELD_DATABASE_NAME,
  type FoundationRow,
} from "./db";
import { IndexedDbFieldRepository } from "./field-repository";
import { createBrowserProfileAuthority } from "./profile-authority";
import {
  classifyOfficialBrowser,
  DEFAULT_BROWSER_VERSION_POLICY,
  type BrowserVersionPolicy,
} from "./browser-policy";
import { CURRENT_FIELD_SCHEMA_VERSION } from "./schema-migrations";
import {
  CURRENT_OFFLINE_VERSIONS,
  type OfflineVersionVector,
} from "./offline-version-contract";

export { CURRENT_OFFLINE_VERSIONS } from "./offline-version-contract";

export type OfflineReadinessCode =
  | "ready"
  | "unsupported-browser"
  | "managed-policy-unapproved"
  | "ephemeral-or-unmanaged-storage"
  | "service-worker-unavailable"
  | "indexeddb-health-failed"
  | "opfs-health-failed"
  | "persistence-denied"
  | "quota-insufficient"
  | "offline-grant-invalid"
  | "app-version-incompatible"
  | "schema-version-incompatible"
  | "protocol-version-incompatible"
  | "runtime-canary-failed";

export type OfflineAdmissionState =
  | "OFFLINE_READY"
  | "ONLINE_ONLY"
  | "UNSUPPORTED"
  | "RECOVERY_REQUIRED";

export interface OfflineBrowserAdmission {
  official: boolean;
  state: "OFFICIAL" | "ONLINE_ONLY" | "UNSUPPORTED";
  reasonCode: string;
}

export type { OfflineVersionVector } from "./offline-version-contract";

export interface OfflinePackageDescriptor {
  packageId: string;
  packageVersion: number;
  packageDigest: string;
  schemaVersion: number;
  protocolVersion: number;
  expiresAt: string;
}

export interface OfflineReadinessInput {
  userInitiated: boolean;
  managedPolicyApproved: boolean;
  storageProfileApproved: boolean;
  expectedSubjectId: string;
  expectedOrganizationId: string;
  expectedDeviceInstanceId: string;
  packageDescriptor: OfflinePackageDescriptor;
  offlineGrant: OfflineGrant | null;
  versions: OfflineVersionVector;
  requiredAppShellVersion: number;
  packageByteEstimate: number;
  attachmentByteEstimate: number;
  minimumHeadroomBytes: number;
  now: Date;
}

export interface StorageEstimate {
  usage?: number;
  quota?: number;
}

export interface OfflineReadinessDependencies {
  isSecureContext: boolean;
  browserSupported: boolean;
  browserAdmission?: () => Promise<OfflineBrowserAdmission>;
  serviceWorkerReady(): Promise<boolean>;
  indexedDbCanary(): Promise<boolean>;
  opfsCanary(): Promise<boolean>;
  restartCanary(): Promise<boolean>;
  runtimeCanary?: () => Promise<boolean>;
  storagePersisted(): Promise<boolean>;
  requestPersistence(): Promise<boolean>;
  estimateStorage(): Promise<StorageEstimate>;
}

export interface OfflineReadinessResult {
  code: OfflineReadinessCode;
  ready: boolean;
  admissionState: OfflineAdmissionState;
  reasonCode: string;
  recoveryAction: string;
  capacityIsAdvisory: boolean;
  requiredBytes: number | null;
  availableBytes: number | null;
}

const RECOVERY_ACTIONS: Record<Exclude<OfflineReadinessCode, "ready">, string> = {
  "unsupported-browser": "Use one of the six official Stable browser lanes over HTTPS, then retry.",
  "managed-policy-unapproved":
    "Confirm the owner-approved managed browser, device, and profile policy before checkout.",
  "ephemeral-or-unmanaged-storage":
    "Use the approved persistent profile, restart the browser to prove the canary survives, and retry.",
  "service-worker-unavailable": "Allow the Service Worker to install, then reload and retry.",
  "indexeddb-health-failed": "Resolve IndexedDB write/read/delete health before offline checkout.",
  "opfs-health-failed": "Resolve OPFS write/read/hash/delete health before offline checkout.",
  "persistence-denied": "Grant persistent storage from this explicit checkout flow or continue online-only.",
  "quota-insufficient": "Free local storage or reduce the package and planned attachment size.",
  "offline-grant-invalid": "Reconnect and request a fresh server-authorized offline grant.",
  "app-version-incompatible": "Update the AviaSurveil360 app shell before offline checkout.",
  "schema-version-incompatible": "Open online and migrate the local/package schema before checkout.",
  "protocol-version-incompatible": "Update the client and request a compatible offline grant.",
  "runtime-canary-failed": "Keep this browser online-only until the namespaced runtime canary passes after restart/readback.",
};

const REASON_CODES: Record<OfflineReadinessCode, string> = {
  ready: "OFFLINE_READY",
  "unsupported-browser": "UNSUPPORTED_BROWSER",
  "managed-policy-unapproved": "MANAGED_POLICY_UNAPPROVED",
  "ephemeral-or-unmanaged-storage": "EPHEMERAL_OR_UNMANAGED_STORAGE",
  "service-worker-unavailable": "SERVICE_WORKER_UNAVAILABLE",
  "indexeddb-health-failed": "INDEXEDDB_HEALTH_FAILED",
  "opfs-health-failed": "OPFS_HEALTH_FAILED",
  "persistence-denied": "PERSISTENCE_DENIED",
  "quota-insufficient": "QUOTA_INSUFFICIENT",
  "offline-grant-invalid": "OFFLINE_GRANT_INVALID",
  "app-version-incompatible": "APP_VERSION_INCOMPATIBLE",
  "schema-version-incompatible": "SCHEMA_VERSION_INCOMPATIBLE",
  "protocol-version-incompatible": "PROTOCOL_VERSION_INCOMPATIBLE",
  "runtime-canary-failed": "RUNTIME_CANARY_FAILED",
};

function failure(
  code: Exclude<OfflineReadinessCode, "ready">,
  capacity: { requiredBytes?: number; availableBytes?: number } = {},
  admissionState: OfflineAdmissionState = code === "unsupported-browser"
    ? "UNSUPPORTED"
    : code === "indexeddb-health-failed" || code === "opfs-health-failed" || code === "runtime-canary-failed"
      ? "RECOVERY_REQUIRED"
      : "ONLINE_ONLY",
  reasonCode = REASON_CODES[code],
): OfflineReadinessResult {
  return {
    code,
    ready: false,
    admissionState,
    reasonCode,
    recoveryAction: RECOVERY_ACTIONS[code],
    capacityIsAdvisory: true,
    requiredBytes: capacity.requiredBytes ?? null,
    availableBytes: capacity.availableBytes ?? null,
  };
}

async function healthy(check: () => Promise<boolean>): Promise<boolean> {
  try {
    return await check();
  } catch {
    return false;
  }
}

function isExactVersion(version: number, current: number): boolean {
  return Number.isSafeInteger(version) && Number.isSafeInteger(current) && version === current;
}

function validInstant(value: string): number | null {
  const parsed = Date.parse(value);
  return Number.isFinite(parsed) ? parsed : null;
}

const MAX_OFFLINE_LEASE_MS = 7 * 24 * 60 * 60_000;

function grantIsValid(input: OfflineReadinessInput): boolean {
  const { offlineGrant: grant, packageDescriptor: descriptor } = input;
  if (!grant) return false;
  const now = input.now.getTime();
  const issuedAt = validInstant(grant.issuedAt);
  const grantExpiresAt = validInstant(grant.expiresAt);
  const leaseIssuedAt = validInstant(grant.leaseIssuedAt ?? "");
  const leaseExpiresAt = validInstant(grant.leaseExpiresAt ?? "");
  const packageExpiresAt = validInstant(descriptor.expiresAt);
  if (
    issuedAt === null ||
    grantExpiresAt === null ||
    leaseIssuedAt === null ||
    leaseExpiresAt === null ||
    packageExpiresAt === null
  ) return false;
  if (
    issuedAt > now + 5 * 60_000 ||
    grantExpiresAt <= now ||
    packageExpiresAt <= now ||
    leaseIssuedAt > now + 5 * 60_000 ||
    leaseExpiresAt <= now ||
    leaseExpiresAt <= leaseIssuedAt ||
    leaseExpiresAt - leaseIssuedAt > MAX_OFFLINE_LEASE_MS
  ) return false;
  return (
    grant.subjectId === input.expectedSubjectId &&
    grant.organizationId === input.expectedOrganizationId &&
    grant.deviceInstanceId === input.expectedDeviceInstanceId &&
    /^sha256:[0-9a-f]{64}$/u.test(grant.profileKeyId ?? "") &&
    Number.isSafeInteger(grant.assignmentRevision) &&
    (grant.assignmentRevision ?? 0) > 0 &&
    grant.packageId === descriptor.packageId &&
    grant.packageVersion === descriptor.packageVersion &&
    grant.packageDigest === descriptor.packageDigest &&
    grant.protocolVersion === descriptor.protocolVersion &&
    grant.allowedCommandTypes.length > 0 &&
    grant.assignmentScope.questionIds.length > 0
  );
}

export async function assessOfflineReadiness(
  input: OfflineReadinessInput,
  dependencies: OfflineReadinessDependencies,
): Promise<OfflineReadinessResult> {
  const browserAdmission = dependencies.browserAdmission
    ? await (async () => {
        try {
          return await dependencies.browserAdmission?.();
        } catch {
          return {
            official: false,
            state: "UNSUPPORTED" as const,
            reasonCode: "BROWSER_ADMISSION_FAILED",
          };
        }
      })()
    : {
        official: dependencies.browserSupported,
        state: dependencies.browserSupported ? ("OFFICIAL" as const) : ("UNSUPPORTED" as const),
        reasonCode: dependencies.browserSupported ? "OFFICIAL_BROWSER" : "UNSUPPORTED_BROWSER",
      };
  if (!browserAdmission?.official) {
    return failure(
      "unsupported-browser",
      {},
      "UNSUPPORTED",
      browserAdmission?.reasonCode ?? "UNSUPPORTED_BROWSER",
    );
  }
  if (!dependencies.isSecureContext) {
    return failure("unsupported-browser", {}, "UNSUPPORTED", "SECURE_CONTEXT_REQUIRED");
  }
  if (!input.managedPolicyApproved) return failure("managed-policy-unapproved");
  if (!input.storageProfileApproved) return failure("ephemeral-or-unmanaged-storage");
  if (!(await healthy(dependencies.serviceWorkerReady))) {
    return failure("service-worker-unavailable");
  }
  if (!(await healthy(dependencies.indexedDbCanary))) {
    return failure("indexeddb-health-failed");
  }
  if (!(await healthy(dependencies.opfsCanary))) return failure("opfs-health-failed");

  let persisted = await healthy(dependencies.storagePersisted);
  if (!persisted && input.userInitiated) persisted = await healthy(dependencies.requestPersistence);
  if (!persisted) return failure("persistence-denied");

  let estimate: StorageEstimate;
  try {
    estimate = await dependencies.estimateStorage();
  } catch {
    return failure("quota-insufficient");
  }
  const requiredBytes =
    input.packageByteEstimate + input.attachmentByteEstimate + input.minimumHeadroomBytes;
  const usage = estimate.usage;
  const quota = estimate.quota;
  if (
    usage === undefined ||
    quota === undefined ||
    !Number.isFinite(usage) ||
    !Number.isFinite(quota) ||
    quota - usage < requiredBytes
  ) {
    return failure("quota-insufficient", {
      requiredBytes,
      availableBytes: usage === undefined || quota === undefined ? 0 : Math.max(0, quota - usage),
    });
  }

  if (!(await healthy(dependencies.restartCanary))) {
    return failure("ephemeral-or-unmanaged-storage", { requiredBytes, availableBytes: quota - usage });
  }
  if (dependencies.runtimeCanary && !(await healthy(dependencies.runtimeCanary))) {
    return failure("runtime-canary-failed", { requiredBytes, availableBytes: quota - usage });
  }
  if (!grantIsValid(input)) {
    return failure("offline-grant-invalid", { requiredBytes, availableBytes: quota - usage });
  }
  if (!isExactVersion(input.versions.appShellVersion, input.requiredAppShellVersion)) {
    return failure("app-version-incompatible", { requiredBytes, availableBytes: quota - usage });
  }
  if (!isExactVersion(input.packageDescriptor.schemaVersion, input.versions.packageSchemaVersion)) {
    return failure("schema-version-incompatible", { requiredBytes, availableBytes: quota - usage });
  }
  if (
    input.offlineGrant?.protocolVersion !== input.packageDescriptor.protocolVersion ||
    !isExactVersion(
      input.packageDescriptor.protocolVersion,
      input.versions.syncProtocolVersion,
    )
  ) {
    return failure("protocol-version-incompatible", { requiredBytes, availableBytes: quota - usage });
  }

  return {
    code: "ready",
    ready: true,
    admissionState: "OFFLINE_READY",
    reasonCode: REASON_CODES.ready,
    recoveryAction: "Offline package checkout may continue.",
    capacityIsAdvisory: true,
    requiredBytes,
    availableBytes: quota - usage,
  };
}

export function describeLocalPackageLoss(input: {
  outstandingCheckout: boolean;
  localPackagePresent: boolean;
}): string | null {
  if (classifyLocalDataState(input) !== "LOCAL_DATA_CLEARED") return null;
  return "Local package missing. Unsynced single-device work cannot be recovered after site data is cleared.";
}

export function classifyLocalDataState(input: {
  outstandingCheckout: boolean;
  localPackagePresent: boolean;
}): "LOCAL_DATA_PRESENT" | "LOCAL_DATA_CLEARED" | "NO_OUTSTANDING_CHECKOUT" {
  if (!input.outstandingCheckout) return "NO_OUTSTANDING_CHECKOUT";
  return input.localPackagePresent ? "LOCAL_DATA_PRESENT" : "LOCAL_DATA_CLEARED";
}

const FOUNDATION_DATABASE_NAME = OFFLINE_FIELD_DATABASE_NAME;
const FOUNDATION_STORE_NAME = "foundation";
const FOUNDATION_DATABASE_VERSION = CURRENT_OFFLINE_VERSIONS.indexedDbSchemaVersion;

function createBootId(): string {
  return globalThis.crypto?.randomUUID?.() ?? `boot-${Date.now()}`;
}

// Keep the canary identity in module memory. The lifecycle workspace must not
// write identifiers to Web Storage, and a new module instance still provides
// the restart distinction required by the IndexedDB canary.
const CURRENT_BOOT_ID = createBootId();

async function openFoundationDatabase() {
  const database = getBrowserOfflineFieldDatabase();
  const result = await database.openForFieldUse();
  if (result.mode === "read-only-recovery") {
    throw new Error(`IndexedDB field migration failed at ${result.failedPhase}`);
  }
  return database;
}

async function readFoundationValue<T>(key: string): Promise<T | null> {
  const database = await openFoundationDatabase();
  return ((await database.foundation.get(key)) as FoundationRow<T> | undefined)?.value ?? null;
}

async function writeFoundationValue<T>(key: string, value: T): Promise<void> {
  const database = await openFoundationDatabase();
  await database.foundation.put({ key, value: structuredClone(value) });
}

async function deleteFoundationValue(key: string): Promise<void> {
  const database = await openFoundationDatabase();
  await database.foundation.delete(key);
}

async function runIndexedDbCanary(): Promise<boolean> {
  const key = `health-canary:${CURRENT_BOOT_ID}`;
  const expected = `indexeddb:${CURRENT_BOOT_ID}`;
  await writeFoundationValue(key, expected);
  const observed = await readFoundationValue<string>(key);
  await deleteFoundationValue(key);
  return observed === expected;
}

function bytesToHex(value: ArrayBuffer): string {
  return Array.from(new Uint8Array(value), (byte) => byte.toString(16).padStart(2, "0")).join("");
}

async function runOpfsCanary(): Promise<boolean> {
  if (!("storage" in navigator) || typeof navigator.storage.getDirectory !== "function") return false;
  const root = await navigator.storage.getDirectory();
  const directoryName = "aviasurveil360-readiness-canary";
  const fileName = `${CURRENT_BOOT_ID}.bin`;
  const directory = await root.getDirectoryHandle(directoryName, { create: true });
  const expected = new TextEncoder().encode(`AviaSurveil360:${CURRENT_BOOT_ID}`);
  try {
    const handle = await directory.getFileHandle(fileName, { create: true });
    const writable = await handle.createWritable();
    await writable.write(expected);
    await writable.close();
    const observed = await (await handle.getFile()).arrayBuffer();
    const [expectedHash, observedHash] = await Promise.all([
      crypto.subtle.digest("SHA-256", expected),
      crypto.subtle.digest("SHA-256", observed),
    ]);
    return bytesToHex(expectedHash) === bytesToHex(observedHash);
  } finally {
    await directory.removeEntry(fileName).catch(() => undefined);
    await root.removeEntry(directoryName, { recursive: true }).catch(() => undefined);
  }
}

async function verifyRestartCanary(): Promise<boolean> {
  const key = "restart-canary";
  const existing = await readFoundationValue<{ bootId: string; verified: boolean }>(key);
  if (existing && existing.bootId !== CURRENT_BOOT_ID) {
    await writeFoundationValue(key, { bootId: CURRENT_BOOT_ID, verified: true });
    return true;
  }
  if (existing?.verified) return true;
  await writeFoundationValue(key, { bootId: CURRENT_BOOT_ID, verified: false });
  return false;
}

function configuredBrowserVersionPolicy(): BrowserVersionPolicy {
  const raw = (import.meta as ImportMeta & { env?: { VITE_AVIA_BROWSER_VERSION_POLICY_JSON?: string } }).env
    ?.VITE_AVIA_BROWSER_VERSION_POLICY_JSON;
  if (!raw) return DEFAULT_BROWSER_VERSION_POLICY;
  try {
    const value = JSON.parse(raw) as Partial<BrowserVersionPolicy>;
    return {
      safariStableMajor: value.safariStableMajor ?? null,
      chromeStableMajor: value.chromeStableMajor ?? null,
      minimumOsVersionByFamily: value.minimumOsVersionByFamily ?? {},
    };
  } catch {
    return DEFAULT_BROWSER_VERSION_POLICY;
  }
}

function currentBrowserAdmission(): OfflineBrowserAdmission {
  const classification = classifyOfficialBrowser(navigator.userAgent, {
    policy: configuredBrowserVersionPolicy(),
    maxTouchPoints: navigator.maxTouchPoints,
  });
  return {
    official: classification.official,
    state: classification.official ? "OFFICIAL" : "UNSUPPORTED",
    reasonCode: classification.reasonCode,
  };
}

async function runRuntimeCanary(): Promise<boolean> {
  return (await runIndexedDbCanary()) && (await runOpfsCanary()) && (await verifyRestartCanary());
}

async function serviceWorkerIsReady(): Promise<boolean> {
  if (!("serviceWorker" in navigator)) return false;
  return Promise.race([
    navigator.serviceWorker.ready.then(() => true),
    new Promise<boolean>((resolve) => setTimeout(() => resolve(false), 5_000)),
  ]);
}

export function createBrowserOfflineReadinessDependencies(): OfflineReadinessDependencies {
  const admission = currentBrowserAdmission();
  return {
    isSecureContext: globalThis.isSecureContext,
    browserSupported: admission.official,
    browserAdmission: async () => currentBrowserAdmission(),
    serviceWorkerReady: serviceWorkerIsReady,
    indexedDbCanary: runIndexedDbCanary,
    opfsCanary: runOpfsCanary,
    restartCanary: verifyRestartCanary,
    runtimeCanary: runRuntimeCanary,
    storagePersisted: async () => navigator.storage.persisted(),
    requestPersistence: async () => navigator.storage.persist(),
    estimateStorage: async () => navigator.storage.estimate(),
  };
}

export async function getOrCreateDeviceInstanceId(): Promise<string> {
  const key = "device-instance-id";
  const existing = await readFoundationValue<string>(key);
  if (existing) return existing;
  const deviceId = `DEVICE-${crypto.randomUUID()}`;
  await writeFoundationValue(key, deviceId);
  return deviceId;
}

export interface OfflineCheckoutSnapshot {
  subjectId: string;
  inspectionPackage: InspectionPackage;
  offlineGrant: OfflineGrant;
  checkedOutAt: string;
  versions: OfflineVersionVector;
}

function checkoutSnapshotKey(subjectId: string, packageId: string): string {
  return `checkout:${subjectId}:${packageId}`;
}

export async function writeOfflineCheckoutSnapshot(snapshot: OfflineCheckoutSnapshot): Promise<void> {
  const repository = new IndexedDbFieldRepository({
    subjectId: snapshot.subjectId,
    now: () => new Date(snapshot.checkedOutAt),
    profileAuthority: createBrowserProfileAuthority(snapshot.subjectId),
  });
  await repository.checkoutPackage({
    inspectionPackage: snapshot.inspectionPackage,
    offlineGrant: snapshot.offlineGrant,
    checkedOutAt: snapshot.checkedOutAt,
  });
  await writeFoundationValue(
    checkoutSnapshotKey(snapshot.subjectId, snapshot.inspectionPackage.id),
    structuredClone(snapshot),
  );
}

export async function readOfflineCheckoutSnapshot(
  subjectId: string,
  packageId: string,
): Promise<OfflineCheckoutSnapshot | null> {
  const database = await openFoundationDatabase();
  const packageRow = await database.packages.get([subjectId, packageId]);
  const grantRow = packageRow
    ? await database.offlineGrants.get([subjectId, packageRow.grantId])
    : undefined;
  if (packageRow?.accessState === "AVAILABLE" && grantRow && packageRow.installedVector) {
    return {
      subjectId,
      inspectionPackage: structuredClone(packageRow.inspectionPackage),
      offlineGrant: structuredClone(grantRow.offlineGrant),
      checkedOutAt: packageRow.checkedOutAt,
      versions: structuredClone(packageRow.installedVector),
    };
  }
  if (packageRow) return null;
  return readFoundationValue<OfflineCheckoutSnapshot>(checkoutSnapshotKey(subjectId, packageId));
}

export const OFFLINE_FOUNDATION_STORAGE = {
  databaseName: FOUNDATION_DATABASE_NAME,
  storeName: FOUNDATION_STORE_NAME,
  databaseVersion: FOUNDATION_DATABASE_VERSION,
} as const;
