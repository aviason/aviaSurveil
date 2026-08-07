import Dexie, { type Table } from "dexie";

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
  type FieldMigrationFault,
  type FieldMigrationPhase,
} from "./schema-migrations";

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
  issuedAt: string;
  expiresAt: string;
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
  syncState: LocalRecordSyncState;
  plannedOperationId: string;
  operationId: string | null;
  authoritativeEntityId: string | null;
  quarantineReason: string | null;
  localBytesPresent: boolean;
  createdAt: string;
  updatedAt: string;
  uploadStartedAt: string | null;
  acknowledgedAt: string | null;
  purgeEligibleAt: string | null;
}

export interface OutboxRow {
  operationId: string;
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
  lastErrorCode: string | null;
  operation: FieldSyncOperation;
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
}

export type FieldDatabaseOpenResult =
  | { mode: "read-write"; version: number }
  | { mode: "read-only-recovery"; failedPhase: FieldMigrationPhase; error: string };

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
    super(options.name ?? OFFLINE_FIELD_DATABASE_NAME);
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

  async openForFieldUse(): Promise<FieldDatabaseOpenResult> {
    if (this.openResult) return this.openResult;
    try {
      this.failedMigrationPhase = "before-expand";
      this.migrationFault?.("before-expand");
      await this.open();
      this.openResult = { mode: "read-write", version: CURRENT_FIELD_SCHEMA_VERSION };
    } catch (error) {
      this.close();
      this.openResult = {
        mode: "read-only-recovery",
        failedPhase: this.failedMigrationPhase,
        error: error instanceof Error ? error.message : "IndexedDB migration failed",
      };
    }
    return this.openResult;
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
