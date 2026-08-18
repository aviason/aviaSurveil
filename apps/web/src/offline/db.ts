import Dexie, { type ChromeTransactionDurability, type Table } from "dexie";

import type {
  ChecklistAnswer,
  FieldCommandType,
  FieldSyncOperation,
  InspectionPackage,
  OfflineGrant,
  PotentialFindingStatus,
} from "../backend/backend";
import {
  CURRENT_FIELD_SCHEMA_VERSION,
  migrateReleasedFoundationToFieldSchema,
  sha256Canonical,
  type FieldMigrationFault,
  type FieldMigrationPhase,
} from "./schema-migrations";
import type { OfflineVersionVector } from "./offline-version-contract";

export type { FieldMigrationPhase } from "./schema-migrations";

export const OFFLINE_FIELD_DATABASE_NAME = "aviasurveil360-offline-foundation";

export type FieldAccessState = "AVAILABLE" | "LOCKED" | "QUARANTINED";
export type AttachmentStagingState =
  | "manifest_created"
  | "writing"
  | "ready"
  | "uploading"
  | "acknowledged"
  | "purge_eligible"
  | "recovery_required"
  | "quarantined";
export type AttachmentRecoveryState =
  | "MANIFEST_COMMITTED"
  | "WRITING_TEMP"
  | "FLUSHED"
  | "HASH_VERIFIED"
  | "PROMOTED"
  | "LOCAL_READY"
  | "RECOVERY_REQUIRED"
  | "QUARANTINED";
export type AttachmentUploadState =
  | "OPEN"
  | "UPLOADING"
  | "PARTIALLY_COMMITTED"
  | "COMPLETING"
  | "COMPLETED"
  | "EXPIRED"
  | "ABORTED"
  | "QUARANTINED";
export type LocalRecordSyncState =
  | "ACKNOWLEDGED"
  | "PENDING"
  | "CONFLICT"
  | "REJECTED"
  | "TOMBSTONED";
export type FieldOutboxState =
  | "PENDING"
  | "BLOCKED_ON_DEPENDENCY"
  | "IN_FLIGHT"
  | "ACKNOWLEDGED"
  | "SUPERSEDED"
  | "CONFLICT"
  | "REJECTED"
  | "QUARANTINED"
  | "FAILED_RETRYABLE";

export interface FoundationRow<T = unknown> {
  key: string;
  value: T;
}

export interface PackageRow {
  id: string;
  subjectId: string;
  auditId: string;
  organizationId: string;
  packageVersion: number;
  schemaVersion: number;
  protocolVersion: number;
  installedVector: OfflineVersionVector | null;
  packageDigest: string;
  storageDigest: string;
  checkedOutAt: string;
  expiresAt: string;
  grantId: string;
  accessState: FieldAccessState;
  unavailableReason: string | null;
  localChecklistStatus: "NOT_STARTED" | "IN_PROGRESS" | "SUBMITTED";
  localChecklistRevision: number;
  pendingSubmissionOperationId: string | null;
  inspectionPackage: InspectionPackage;
}

export interface OfflineGrantRow {
  grantId: string;
  subjectId: string;
  organizationId: string;
  packageId: string;
  packageVersion: number;
  packageDigest: string;
  deviceInstanceId: string;
  profileKeyId?: string;
  assignmentRevision?: number;
  issuedAt: string;
  expiresAt: string;
  leaseIssuedAt?: string;
  leaseExpiresAt?: string;
  protocolVersion: number;
  offlineGrant: OfflineGrant;
}

export interface ChecklistResponseRow {
  id: string;
  subjectId: string;
  packageId: string;
  auditId: string;
  questionId: string;
  answer: ChecklistAnswer;
  comment: string;
  revision: number;
  syncState: LocalRecordSyncState;
  updatedAt: string;
  operationId: string | null;
  tombstoned: boolean;
}

export interface PotentialFindingDraftRow {
  id: string;
  subjectId: string;
  packageId: string;
  auditId: string;
  questionId: string;
  checklistResponseId: string;
  organizationId: string;
  title: string;
  description: string;
  requiredComment: string;
  inspectionAttachmentIds: string[];
  baseRevision: number | null;
  status: PotentialFindingStatus;
  syncState: LocalRecordSyncState;
  updatedAt: string;
  operationId: string | null;
  authoritativeEntityId: string | null;
  tombstoned: boolean;
}

export interface AttachmentManifestRow {
  attachmentId: string;
  subjectId: string;
  packageId: string;
  auditId: string;
  checklistResponseId: string;
  potentialFindingLocalId: string | null;
  fileName: string;
  declaredMediaType: "application/pdf" | "image/jpeg" | "image/png" | "application/octet-stream";
  declaredByteSize: number;
  observedByteSize: number | null;
  expectedSha256: string | null;
  sha256: string | null;
  temporaryOpfsPath: string | null;
  finalOpfsPath: string | null;
  stagingState: AttachmentStagingState;
  recoveryState?: AttachmentRecoveryState;
  syncState: LocalRecordSyncState;
  plannedOperationId: string;
  operationId: string | null;
  authoritativeEntityId: string | null;
  quarantineReason: string | null;
  localBytesPresent: boolean;
  createdAt: string;
  updatedAt: string;
  uploadStartedAt: string | null;
  uploadState?: AttachmentUploadState;
  uploadSessionId?: string | null;
  uploadSessionEpoch?: number | null;
  uploadPartSize?: number | null;
  uploadReceivedParts?: number[];
  uploadAcknowledgedOffsets?: number[];
  uploadPartHashes?: Record<string, string>;
  uploadPartObjectVersions?: Record<string, string>;
  uploadWholeFileSha256?: string | null;
  uploadExpiresAt?: string | null;
  uploadObjectVersion?: string | null;
  uploadBeginOperationId?: string | null;
  uploadCompleteOperationId?: string | null;
  acknowledgedAt: string | null;
  purgeEligibleAt: string | null;
}

export interface OutboxRow {
  operationId: string;
  operationSequence: number;
  subjectId: string;
  packageId: string;
  commandType: FieldCommandType;
  entityId: string;
  baseRevision: number | null;
  state: FieldOutboxState;
  createdAt: string;
  attemptCount: number;
  nextAttemptAt: string;
  dependsOnOperationIds: string[];
  supersededByOperationId: string | null;
  requestDigest: string;
  entityRevision: number;
  entityHash: string;
  commitReceiptKey: string;
  lastErrorCode: string | null;
  operation: FieldSyncOperation;
}

export interface LocalCommitReceiptRow {
  operationId: string;
  subjectId: string;
  packageId: string;
  entityId: string;
  operationSequence: number;
  entityRevision: number;
  entityHash: string;
  requestHash: string;
  committedAt: string;
}

export interface SyncStateRow {
  subjectId: string;
  packageId: string;
  grantId: string;
  projectionVersion: number;
  cursor: string | null;
  lastSuccessAt: string | null;
  lastErrorCode: string | null;
}

export interface OfflineFieldDatabaseOptions {
  name?: string;
  migrationFault?: FieldMigrationFault;
  durability?: ChromeTransactionDurability;
}

export type FieldDatabaseOpenResult =
  | { mode: "read-write"; version: number }
  | { mode: "read-only-recovery"; failedPhase: FieldMigrationPhase; error: string };

export type FieldDatabaseLifecycleState =
  | "CLOSED"
  | "OPENING"
  | "OPEN"
  | "BLOCKED"
  | "VERSION_CHANGE"
  | "READ_ONLY_RECOVERY";

const FIELD_STORES = {
  foundation: "&key",
  packages: "[subjectId+id],subjectId,[subjectId+auditId],[subjectId+accessState]",
  offlineGrants: "[subjectId+grantId],subjectId,[subjectId+packageId]",
  checklistResponses:
    "[subjectId+id],subjectId,[subjectId+packageId],[subjectId+questionId],[subjectId+syncState]",
  potentialFindingDrafts:
    "[subjectId+id],subjectId,[subjectId+packageId],[subjectId+questionId],[subjectId+syncState]",
  attachmentManifests:
    "[subjectId+attachmentId],subjectId,[subjectId+packageId],[subjectId+syncState]",
  outbox:
    "[subjectId+operationId],subjectId,[subjectId+packageId],[subjectId+entityId],[subjectId+state],createdAt",
  syncState: "[subjectId+packageId],subjectId,[subjectId+grantId]",
} as const;

export class OfflineFieldDatabase extends Dexie {
  readonly requestedDurability: ChromeTransactionDurability;
  private databaseLifecycleState: FieldDatabaseLifecycleState = "CLOSED";
  foundation!: Table<FoundationRow, string>;
  packages!: Table<PackageRow, [string, string]>;
  offlineGrants!: Table<OfflineGrantRow, [string, string]>;
  checklistResponses!: Table<ChecklistResponseRow, [string, string]>;
  potentialFindingDrafts!: Table<PotentialFindingDraftRow, [string, string]>;
  attachmentManifests!: Table<AttachmentManifestRow, [string, string]>;
  outbox!: Table<OutboxRow, [string, string]>;
  syncState!: Table<SyncStateRow, [string, string]>;

  private readonly migrationFault?: FieldMigrationFault;
  private failedMigrationPhase: FieldMigrationPhase = "before-expand";
  private openResult: FieldDatabaseOpenResult | null = null;

  constructor(options: OfflineFieldDatabaseOptions = {}) {
    const durability = options.durability ?? "strict";
    super(options.name ?? OFFLINE_FIELD_DATABASE_NAME, { chromeTransactionDurability: durability });
    this.requestedDurability = durability;
    this.on("blocked", () => {
      this.databaseLifecycleState = "BLOCKED";
    });
    this.on("versionchange", () => {
      this.databaseLifecycleState = "VERSION_CHANGE";
      this.close({ disableAutoOpen: true });
    });
    this.on("close", () => {
      if (this.databaseLifecycleState !== "READ_ONLY_RECOVERY") {
        this.databaseLifecycleState = "CLOSED";
      }
    });
    this.migrationFault = options.migrationFault;
    this.version(1).stores({ foundation: "&key" });
    this.version(CURRENT_FIELD_SCHEMA_VERSION)
      .stores(FIELD_STORES)
      .upgrade(async (transaction) => {
        await migrateReleasedFoundationToFieldSchema(transaction, (phase) => {
          this.failedMigrationPhase = phase;
          this.migrationFault?.(phase);
        });
      });
  }

  get lifecycleState(): FieldDatabaseLifecycleState {
    return this.databaseLifecycleState;
  }

  async openForFieldUse(): Promise<FieldDatabaseOpenResult> {
    if (this.openResult) return this.openResult;
    this.databaseLifecycleState = "OPENING";
    try {
      this.failedMigrationPhase = "before-expand";
      this.migrationFault?.("before-expand");
      await this.open();
      await this.forwardMigrateLocalMutationReceipts();
      this.openResult = { mode: "read-write", version: CURRENT_FIELD_SCHEMA_VERSION };
      this.databaseLifecycleState = "OPEN";
    } catch (error) {
      this.close();
      this.databaseLifecycleState = "READ_ONLY_RECOVERY";
      this.openResult = {
        mode: "read-only-recovery",
        failedPhase: this.failedMigrationPhase,
        error: error instanceof Error ? error.message : "IndexedDB migration failed",
      };
    }
    return this.openResult;
  }

  private async forwardMigrateLocalMutationReceipts(): Promise<void> {
    await this.transaction(
      "rw",
      [this.foundation, this.outbox],
      async () => {
        const rows = await this.outbox.toArray();
        const byPackage = new Map<string, OutboxRow[]>();
        for (const row of rows) {
          const packageRows = byPackage.get(row.packageId) ?? [];
          packageRows.push(row);
          byPackage.set(row.packageId, packageRows);
        }
        for (const [packageId, packageRows] of byPackage) {
          packageRows.sort((left, right) =>
            left.createdAt.localeCompare(right.createdAt) || left.operationId.localeCompare(right.operationId),
          );
          const sequenceKey = `operation-sequence:${packageRows[0]?.subjectId ?? ""}:${packageId}`;
          const storedSequence = await this.foundation.get(sequenceKey) as FoundationRow<{ nextSequence: number }> | undefined;
          let nextSequence = Math.max(
            storedSequence?.value.nextSequence ?? 1,
            ...packageRows.map((row) => Number.isSafeInteger(row.operationSequence) ? row.operationSequence + 1 : 1),
          );
          for (const row of packageRows) {
            const receiptKey = row.commitReceiptKey || `commit-receipt:${row.subjectId}:${row.operationId}`;
            const existingReceipt = await this.foundation.get(receiptKey) as FoundationRow<LocalCommitReceiptRow> | undefined;
            const complete =
              Number.isSafeInteger(row.operationSequence) && row.operationSequence > 0 &&
              Number.isSafeInteger(row.entityRevision) && typeof row.entityHash === "string" &&
              /^sha256:[0-9a-f]{64}$/.test(row.entityHash) && Boolean(existingReceipt);
            if (complete) continue;
            const operationSequence = Number.isSafeInteger(row.operationSequence) && row.operationSequence > 0
              ? row.operationSequence
              : nextSequence++;
            const entityHash = /^sha256:[0-9a-f]{64}$/.test(row.entityHash ?? "")
              ? row.entityHash
              : await Dexie.waitFor(sha256Canonical(row.operation.payload));
            row.operationSequence = operationSequence;
            row.entityRevision = row.entityRevision ?? row.baseRevision ?? 0;
            row.entityHash = entityHash;
            row.commitReceiptKey = receiptKey;
            await this.outbox.put(row);
            await this.foundation.put({
              key: receiptKey,
              value: {
                operationId: row.operationId,
                subjectId: row.subjectId,
                packageId: row.packageId,
                entityId: row.entityId,
                operationSequence,
                entityRevision: row.entityRevision,
                entityHash,
                requestHash: row.requestDigest,
                committedAt: row.createdAt,
              } satisfies LocalCommitReceiptRow,
            });
          }
          await this.foundation.put({ key: sequenceKey, value: { nextSequence } });
        }
      },
    );
  }

  async readFoundationRecoveryRecord<T>(key: string): Promise<FoundationRow<T> | null> {
    return new Promise((resolve, reject) => {
      const request = indexedDB.open(this.name);
      request.onerror = () => reject(request.error ?? new Error("Recovery database open failed"));
      request.onsuccess = () => {
        const database = request.result;
        if (!database.objectStoreNames.contains("foundation")) {
          database.close();
          resolve(null);
          return;
        }
        const transaction = database.transaction("foundation", "readonly");
        const read = transaction.objectStore("foundation").get(key);
        read.onerror = () => reject(read.error ?? new Error("Recovery record read failed"));
        read.onsuccess = () => resolve((read.result as FoundationRow<T> | undefined) ?? null);
        transaction.oncomplete = () => database.close();
      };
    });
  }
}

let browserDatabase: OfflineFieldDatabase | null = null;

export function getBrowserOfflineFieldDatabase(): OfflineFieldDatabase {
  browserDatabase ??= new OfflineFieldDatabase();
  return browserDatabase;
}
