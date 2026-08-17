import type { InspectionPackage, OfflineGrant } from "../backend/backend";
import {
  getBrowserOfflineFieldDatabase,
  OFFLINE_FIELD_DATABASE_NAME,
  type FoundationRow,
} from "./db";
import { IndexedDbFieldRepository } from "./field-repository";
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
  | "protocol-version-incompatible";

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
  serviceWorkerReady(): Promise<boolean>;
  indexedDbCanary(): Promise<boolean>;
  opfsCanary(): Promise<boolean>;
  restartCanary(): Promise<boolean>;
  storagePersisted(): Promise<boolean>;
  requestPersistence(): Promise<boolean>;
  estimateStorage(): Promise<StorageEstimate>;
}

export interface OfflineReadinessResult {
  code: OfflineReadinessCode;
  ready: boolean;
  recoveryAction: string;
  capacityIsAdvisory: boolean;
  requiredBytes: number | null;
  availableBytes: number | null;
}

const RECOVERY_ACTIONS: Record<Exclude<OfflineReadinessCode, "ready">, string> = {
  "unsupported-browser": "Use current managed Chrome over HTTPS or localhost, then retry.",
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
};

function failure(
  code: Exclude<OfflineReadinessCode, "ready">,
  capacity: { requiredBytes?: number; availableBytes?: number } = {},
): OfflineReadinessResult {
  return {
    code,
    ready: false,
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

function grantIsValid(input: OfflineReadinessInput): boolean {
  const { offlineGrant: grant, packageDescriptor: descriptor } = input;
  if (!grant) return false;
  const now = input.now.getTime();
  const issuedAt = validInstant(grant.issuedAt);
  const grantExpiresAt = validInstant(grant.expiresAt);
  const packageExpiresAt = validInstant(descriptor.expiresAt);
  if (issuedAt === null || grantExpiresAt === null || packageExpiresAt === null) return false;
  if (issuedAt > now + 5 * 60_000 || grantExpiresAt <= now || packageExpiresAt <= now) return false;
  return (
    grant.subjectId === input.expectedSubjectId &&
    grant.organizationId === input.expectedOrganizationId &&
    grant.deviceInstanceId === input.expectedDeviceInstanceId &&
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
  if (!dependencies.isSecureContext || !dependencies.browserSupported) {
    return failure("unsupported-browser");
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
  if (!input.outstandingCheckout || input.localPackagePresent) return null;
  return "Local package missing. Unsynced single-device work cannot be recovered after site data is cleared.";
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

function currentManagedChromeIsSupported(): boolean {
  const userAgent = navigator.userAgent;
  return /(?:Chrome|Chromium)\//.test(userAgent) && !/(?:Edg|OPR)\//.test(userAgent);
}

async function serviceWorkerIsReady(): Promise<boolean> {
  if (!("serviceWorker" in navigator)) return false;
  return Promise.race([
    navigator.serviceWorker.ready.then(() => true),
    new Promise<boolean>((resolve) => setTimeout(() => resolve(false), 5_000)),
  ]);
}

export function createBrowserOfflineReadinessDependencies(): OfflineReadinessDependencies {
  return {
    isSecureContext: globalThis.isSecureContext,
    browserSupported: currentManagedChromeIsSupported(),
    serviceWorkerReady: serviceWorkerIsReady,
    indexedDbCanary: runIndexedDbCanary,
    opfsCanary: runOpfsCanary,
    restartCanary: verifyRestartCanary,
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
