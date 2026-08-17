import Dexie, { type Transaction } from "dexie";

import type {
  ChecklistResponseRow,
  FoundationRow,
  OfflineGrantRow,
  PackageRow,
  SyncStateRow,
} from "./db";
import type { InspectionPackage, OfflineGrant } from "../backend/backend";
import {
  CURRENT_OFFLINE_VERSIONS,
  type OfflineVersionVector,
} from "./offline-version-contract";

export const RELEASED_FIELD_SCHEMA_VERSION = 1;
export const CURRENT_FIELD_SCHEMA_VERSION = CURRENT_OFFLINE_VERSIONS.indexedDbSchemaVersion;
export const CURRENT_FIELD_PACKAGE_SCHEMA_VERSION = CURRENT_OFFLINE_VERSIONS.packageSchemaVersion;
export const CURRENT_FIELD_PROTOCOL_VERSION = CURRENT_OFFLINE_VERSIONS.syncProtocolVersion;

export type FieldMigrationPhase =
  | "before-expand"
  | "after-expand"
  | "after-copy"
  | "before-contract";

export type FieldMigrationFault = (phase: FieldMigrationPhase) => void;

export function isFieldSchemaNOrNMinusOne(version: number, current: number): boolean {
  return (
    Number.isSafeInteger(version) &&
    Number.isSafeInteger(current) &&
    version > 0 &&
    current > 0 &&
    (version === current || version === current - 1)
  );
}

function canonicalize(value: unknown): unknown {
  if (Array.isArray(value)) return value.map(canonicalize);
  if (value && typeof value === "object") {
    return Object.fromEntries(
      Object.entries(value as Record<string, unknown>)
        .sort(([left], [right]) => left.localeCompare(right))
        .map(([key, child]) => [key, canonicalize(child)]),
    );
  }
  return value;
}

export function canonicalJson(value: unknown): string {
  return JSON.stringify(canonicalize(value));
}

export async function sha256Canonical(value: unknown): Promise<string> {
  const bytes = new TextEncoder().encode(canonicalJson(value));
  const digest = await crypto.subtle.digest("SHA-256", bytes);
  return `sha256:${Array.from(new Uint8Array(digest), (byte) =>
    byte.toString(16).padStart(2, "0"),
  ).join("")}`;
}

interface LegacyOfflineCheckoutSnapshot {
  subjectId: string;
  inspectionPackage: InspectionPackage;
  offlineGrant: OfflineGrant;
  checkedOutAt: string;
  versions: {
    appShellVersion: number;
    indexedDbSchemaVersion: number;
    packageSchemaVersion: number;
    syncProtocolVersion: number;
  };
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return Boolean(value) && typeof value === "object" && !Array.isArray(value);
}

function isLegacySnapshot(value: unknown): value is LegacyOfflineCheckoutSnapshot {
  if (!isRecord(value) || !isRecord(value.inspectionPackage) || !isRecord(value.offlineGrant)) {
    return false;
  }
  return (
    typeof value.subjectId === "string" &&
    typeof value.checkedOutAt === "string" &&
    typeof value.inspectionPackage.id === "string" &&
    typeof value.inspectionPackage.auditId === "string" &&
    typeof value.inspectionPackage.organizationId === "string" &&
    typeof value.offlineGrant.grantId === "string" &&
    value.offlineGrant.subjectId === value.subjectId &&
    value.offlineGrant.packageId === value.inspectionPackage.id
  );
}

function recordedVector(value: unknown): OfflineVersionVector | null {
  if (!isRecord(value) || !isRecord(value.versions)) return null;
  const candidate = value.versions;
  const values = [candidate.appShellVersion, candidate.indexedDbSchemaVersion, candidate.packageSchemaVersion, candidate.syncProtocolVersion];
  return values.every((item) => Number.isSafeInteger(item) && (item as number) > 0) ? {
    appShellVersion: candidate.appShellVersion as number,
    indexedDbSchemaVersion: candidate.indexedDbSchemaVersion as number,
    packageSchemaVersion: candidate.packageSchemaVersion as number,
    syncProtocolVersion: candidate.syncProtocolVersion as number,
  } : null;
}

export async function migrateReleasedFoundationToFieldSchema(
  transaction: Transaction,
  recordPhase: (phase: FieldMigrationPhase) => void,
): Promise<void> {
  recordPhase("after-expand");
  const foundation = transaction.table<FoundationRow, string>("foundation");
  const packages = transaction.table<PackageRow, [string, string]>("packages");
  const grants = transaction.table<OfflineGrantRow, [string, string]>("offlineGrants");
  const responses = transaction.table<ChecklistResponseRow, [string, string]>(
    "checklistResponses",
  );
  const syncStates = transaction.table<SyncStateRow, [string, string]>("syncState");
  const legacyRows = await foundation.toArray();

  for (const row of legacyRows) {
    if (!row.key.startsWith("checkout:") || !isLegacySnapshot(row.value)) continue;
    const snapshot = row.value;
    const inspectionPackage = structuredClone(snapshot.inspectionPackage);
    const offlineGrant = structuredClone(snapshot.offlineGrant);
    const compatible =
      inspectionPackage.schemaVersion === CURRENT_FIELD_PACKAGE_SCHEMA_VERSION &&
      inspectionPackage.protocolVersion === CURRENT_FIELD_PROTOCOL_VERSION &&
      offlineGrant.protocolVersion === CURRENT_FIELD_PROTOCOL_VERSION;
    const installedVector = recordedVector(snapshot);
    const packageRow: PackageRow = {
      id: inspectionPackage.id,
      subjectId: snapshot.subjectId,
      auditId: inspectionPackage.auditId,
      organizationId: inspectionPackage.organizationId,
      packageVersion: inspectionPackage.packageVersion,
      schemaVersion: inspectionPackage.schemaVersion,
      protocolVersion: inspectionPackage.protocolVersion,
      installedVector,
      packageDigest: inspectionPackage.packageDigest,
      storageDigest: await Dexie.waitFor(sha256Canonical(inspectionPackage)),
      checkedOutAt: snapshot.checkedOutAt,
      expiresAt: inspectionPackage.expiresAt,
      grantId: offlineGrant.grantId,
      accessState: compatible && installedVector ? "AVAILABLE" : "QUARANTINED",
      unavailableReason: compatible && installedVector ? null : installedVector ? "PACKAGE_SCHEMA_INCOMPATIBLE" : "INSTALLED_VERSION_VECTOR_MISSING",
      localChecklistStatus: inspectionPackage.checklistStatus,
      localChecklistRevision: inspectionPackage.checklistRevision,
      pendingSubmissionOperationId: null,
      inspectionPackage,
    };
    const grantRow: OfflineGrantRow = {
      grantId: offlineGrant.grantId,
      subjectId: snapshot.subjectId,
      organizationId: offlineGrant.organizationId,
      packageId: offlineGrant.packageId,
      packageVersion: offlineGrant.packageVersion,
      packageDigest: offlineGrant.packageDigest,
      deviceInstanceId: offlineGrant.deviceInstanceId,
      issuedAt: offlineGrant.issuedAt,
      expiresAt: offlineGrant.expiresAt,
      protocolVersion: offlineGrant.protocolVersion,
      offlineGrant,
    };
    await packages.put(packageRow);
    await grants.put(grantRow);
    for (const question of inspectionPackage.questions) {
      if (!question.currentResponse) continue;
      await responses.put({
        ...structuredClone(question.currentResponse),
        subjectId: snapshot.subjectId,
        packageId: inspectionPackage.id,
        auditId: inspectionPackage.auditId,
        questionId: question.id,
        syncState: "ACKNOWLEDGED",
        operationId: null,
        tombstoned: false,
      });
    }
    await syncStates.put({
      subjectId: snapshot.subjectId,
      packageId: inspectionPackage.id,
      grantId: offlineGrant.grantId,
      projectionVersion: 0,
      cursor: null,
      lastSuccessAt: null,
      lastErrorCode: null,
    });
  }
  recordPhase("after-copy");
  recordPhase("before-contract");
}
