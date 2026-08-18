import Dexie, { type Table } from "dexie";

import type {
  AuthorizedSyncChange,
  ChecklistAnswer,
  FieldOperationBase,
  ChecklistResponseView,
  FieldSyncOperation,
  InspectionPackage,
  OfflineGrant,
  PushFieldOperationResult,
  SyncPullResponse,
  UploadPartReceipt,
  ResumableInspectionAttachmentUploadSession,
} from "../backend/backend";
import {
  getBrowserOfflineFieldDatabase,
  type AttachmentRecoveryState,
  type AttachmentManifestRow,
  type ChecklistResponseRow,
  type FieldAccessState,
  type FieldDatabaseOpenResult,
  type FoundationRow,
  type LocalCommitReceiptRow,
  type OfflineFieldDatabase,
  type OutboxRow,
  type PackageRow,
  type PotentialFindingDraftRow,
  type SyncStateRow,
} from "./db";
import {
  createOutboxRow,
  fieldOperationRequestDigest,
  isCausalDependencyState,
  isTerminalOutboxState,
  isUnsentOutboxState,
} from "./outbox";
import {
  canonicalJson,
  CURRENT_FIELD_PACKAGE_SCHEMA_VERSION,
  CURRENT_FIELD_PROTOCOL_VERSION,
  isFieldSchemaNOrNMinusOne,
  sha256Canonical,
} from "./schema-migrations";
import { CURRENT_OFFLINE_VERSIONS } from "./offline-version-contract";
import { createBrowserProfileAuthority, type ProfileAuthority } from "./profile-authority";

export type FieldTransactionBoundary =
  | "after-checklist-response-write"
  | "after-potential-finding-write"
  | "after-checklist-submission-write"
  | "before-pull-cursor-write"
  | "before-attachment-metadata-ready"
  | "after-attachment-metadata-ready"
  | "before-attachment-outbox-create"
  | "after-attachment-outbox-create"
  | "before-attachment-upload-start"
  | "after-attachment-upload-start"
  | "before-attachment-acknowledgement"
  | "after-attachment-acknowledgement";

export type FieldTransactionFault = (
  boundary: FieldTransactionBoundary,
) => void | Promise<void>;

export class FieldRepositoryError extends Error {
  constructor(
    readonly code: string,
    message: string,
    options?: ErrorOptions,
  ) {
    super(message, options);
    this.name = "FieldRepositoryError";
  }
}

export class FieldAtomicWriteError extends FieldRepositoryError {
  constructor(message: string, cause: unknown) {
    super("ATOMIC_WRITE_FAILED", message, { cause });
    this.name = "FieldAtomicWriteError";
  }
}

export class FieldPackageUnavailableError extends FieldRepositoryError {
  constructor(code: string, message: string) {
    super(code, message);
    this.name = "FieldPackageUnavailableError";
  }
}

export interface FieldCheckoutInput {
  inspectionPackage: InspectionPackage;
  offlineGrant: OfflineGrant;
  checkedOutAt: string;
}

export interface SaveLocalChecklistResponseInput {
  operationId: string;
  packageId: string;
  responseId: string;
  questionId: string;
  answer: ChecklistAnswer;
  comment: string;
}

export interface CreateLocalPotentialFindingInput {
  operationId: string;
  packageId: string;
  localId: string;
  questionId: string;
  checklistResponseId: string;
  title: string;
  description: string;
  requiredComment: string;
  inspectionAttachmentIds: string[];
}

export interface SubmitLocalChecklistInput {
  operationId: string;
  packageId: string;
}

export interface CreateAttachmentManifestInput {
  attachmentId: string;
  operationId: string;
  packageId: string;
  checklistResponseId: string;
  potentialFindingLocalId: string | null;
  fileName: string;
  mediaType: string;
  byteSize: number;
  temporaryOpfsPath: string;
  finalOpfsPath: string;
}

export interface CommitReadyAttachmentInput {
  attachmentId: string;
  observedByteSize: number;
  sha256: string;
}

export interface AcknowledgeAttachmentInput {
  attachmentId: string;
  authoritativeEntityId: string;
  acknowledgedAt: string;
}

export interface AcknowledgeUploadedAttachmentInput extends AcknowledgeAttachmentInput {
  uploadState?: "COMPLETED";
  objectVersion?: string;
  byteSize?: number;
  sha256?: string;
}

export interface FieldChecklistSubmission {
  auditId: string;
  checklistStatus: "SUBMITTED";
  checklistRevision: number;
  syncState: "PENDING";
  operationId: string;
}

export interface FieldPackageView {
  inspectionPackage: InspectionPackage;
  accessState: FieldAccessState;
  responses: ChecklistResponseRow[];
  potentialFindingDrafts: PotentialFindingDraftRow[];
  attachmentManifests: AttachmentManifestRow[];
  pendingOperationCount: number;
}

export interface ApplyFieldPullPageInput {
  packageId: string;
  grantId: string;
  expectedCursor: string | null;
  page: SyncPullResponse;
}

export interface FieldSubjectSnapshot {
  packages: PackageRow[];
  checklistResponses: ChecklistResponseRow[];
  potentialFindingDrafts: PotentialFindingDraftRow[];
  attachmentManifests: AttachmentManifestRow[];
  outbox: OutboxRow[];
  syncState: SyncStateRow[];
}

interface IndexedDbFieldRepositoryOptions {
  database?: OfflineFieldDatabase;
  subjectId: string;
  now?: () => Date;
  transactionFault?: FieldTransactionFault;
  profileAuthority?: ProfileAuthority | Promise<ProfileAuthority>;
}

const CLOCK_SKEW_MS = 5 * 60_000;
const MAX_OFFLINE_LEASE_MS = 7 * 24 * 60 * 60_000;
const MAX_INSPECTION_ATTACHMENT_BYTES = 25 * 1024 * 1024;
const INSPECTION_ATTACHMENT_MEDIA_TYPES = new Set([
  "application/pdf",
  "image/jpeg",
  "image/png",
] as const);

type InspectionAttachmentMediaType = Exclude<
  AttachmentManifestRow["declaredMediaType"],
  "application/octet-stream"
>;

function isInspectionAttachmentMediaType(value: string): value is InspectionAttachmentMediaType {
  return INSPECTION_ATTACHMENT_MEDIA_TYPES.has(value as InspectionAttachmentMediaType);
}

function parseInstant(value: string): number | null {
  const parsed = Date.parse(value);
  return Number.isFinite(parsed) ? parsed : null;
}

function clone<T>(value: T): T {
  return structuredClone(value);
}

function sortByKey<T>(values: T[], key: (value: T) => string): T[] {
  return values.sort((left, right) => key(left).localeCompare(key(right)));
}

export class IndexedDbFieldRepository {
  readonly database: OfflineFieldDatabase;
  readonly subjectId: string;
  private readonly now: () => Date;
  private readonly transactionFault?: FieldTransactionFault;
  private readonly profileAuthority?: Promise<ProfileAuthority>;

  constructor(options: IndexedDbFieldRepositoryOptions) {
    this.database = options.database ?? getBrowserOfflineFieldDatabase();
    this.subjectId = options.subjectId;
    this.now = options.now ?? (() => new Date());
    this.transactionFault = options.transactionFault;
    this.profileAuthority = options.profileAuthority
      ? Promise.resolve(options.profileAuthority)
      : undefined;
    if (!this.subjectId.trim()) {
      throw new FieldRepositoryError("SUBJECT_REQUIRED", "A bound session subject is required.");
    }
  }

  withClock(now: () => Date): IndexedDbFieldRepository {
    return new IndexedDbFieldRepository({
      database: this.database,
      subjectId: this.subjectId,
      now,
      transactionFault: this.transactionFault,
      profileAuthority: this.profileAuthority,
    });
  }

  async initialize(): Promise<FieldDatabaseOpenResult> {
    return this.database.openForFieldUse();
  }

  private async requireReadWrite(): Promise<void> {
    const result = await this.initialize();
    if (result.mode === "read-only-recovery") {
      throw new FieldRepositoryError(
        "READ_ONLY_RECOVERY",
        `Local schema recovery is read-only after ${result.failedPhase}.`,
      );
    }
  }

  private async fault(boundary: FieldTransactionBoundary): Promise<void> {
    await this.transactionFault?.(boundary);
  }

  private async atomic<T>(tables: Table[], action: () => Promise<T>): Promise<T> {
    await this.requireReadWrite();
    try {
      return await this.database.transaction("rw", tables, action);
    } catch (error) {
      if (error instanceof FieldRepositoryError) throw error;
      throw new FieldAtomicWriteError("The local entity and outbox transaction was aborted.", error);
    }
  }

  private validateCheckout(input: FieldCheckoutInput): void {
    const { inspectionPackage, offlineGrant } = input;
    const now = this.now().getTime();
    const issuedAt = parseInstant(offlineGrant.issuedAt);
    const grantExpiresAt = parseInstant(offlineGrant.expiresAt);
    const leaseIssuedAt = parseInstant(offlineGrant.leaseIssuedAt ?? "");
    const leaseExpiresAt = parseInstant(offlineGrant.leaseExpiresAt ?? "");
    const packageExpiresAt = parseInstant(inspectionPackage.expiresAt);
    if (!isFieldSchemaNOrNMinusOne(
      inspectionPackage.schemaVersion,
      CURRENT_FIELD_PACKAGE_SCHEMA_VERSION,
    )) {
      throw new FieldPackageUnavailableError(
        "PACKAGE_SCHEMA_INCOMPATIBLE",
        "The inspection package schema is outside the positive N/N-1 window.",
      );
    }
    if (
      inspectionPackage.protocolVersion !== CURRENT_FIELD_PROTOCOL_VERSION ||
      offlineGrant.protocolVersion !== CURRENT_FIELD_PROTOCOL_VERSION
    ) {
      throw new FieldPackageUnavailableError(
        "PROTOCOL_VERSION_INCOMPATIBLE",
        "The offline protocol version is incompatible.",
      );
    }
    if (
      offlineGrant.subjectId !== this.subjectId ||
      offlineGrant.organizationId !== inspectionPackage.organizationId ||
      offlineGrant.packageId !== inspectionPackage.id ||
      offlineGrant.packageVersion !== inspectionPackage.packageVersion ||
      offlineGrant.packageDigest !== inspectionPackage.packageDigest ||
      offlineGrant.deviceInstanceId.trim() === "" ||
      offlineGrant.assignmentScope.questionIds.length === 0
    ) {
      throw new FieldPackageUnavailableError(
        "OFFLINE_GRANT_SCOPE_INVALID",
        "The server-issued grant does not exactly match this subject and package.",
      );
    }
    if (
      this.profileAuthority &&
      (!offlineGrant.profileKeyId ||
        offlineGrant.assignmentRevision === undefined ||
        leaseIssuedAt === null ||
        leaseExpiresAt === null ||
        leaseExpiresAt <= leaseIssuedAt ||
        leaseExpiresAt - leaseIssuedAt > MAX_OFFLINE_LEASE_MS)
    ) {
      throw new FieldPackageUnavailableError(
        "OFFLINE_GRANT_AUTHORITY_INVALID",
        "The server-issued grant is missing a bounded profile authority lease.",
      );
    }
    const questions = new Map(inspectionPackage.questions.map((question) => [question.id, question]));
    if (
      offlineGrant.assignmentScope.questionIds.some((questionId) => {
        const question = questions.get(questionId);
        return !question || !question.assignedInspectorUserIds.includes(this.subjectId);
      })
    ) {
      throw new FieldPackageUnavailableError(
        "OFFLINE_GRANT_ASSIGNMENT_INVALID",
        "The grant assignment is not present in the immutable package.",
      );
    }
    if (
      issuedAt === null ||
      grantExpiresAt === null ||
      packageExpiresAt === null ||
      issuedAt > now + CLOCK_SKEW_MS ||
      grantExpiresAt <= now ||
      packageExpiresAt <= now ||
      (this.profileAuthority && (leaseIssuedAt === null || leaseExpiresAt === null || leaseExpiresAt <= now))
    ) {
      throw new FieldPackageUnavailableError(
        "OFFLINE_GRANT_EXPIRED",
        "The package or server-issued offline grant is not currently valid.",
      );
    }
  }

  async checkoutPackage(input: FieldCheckoutInput): Promise<void> {
    this.validateCheckout(input);
    const inspectionPackage = clone(input.inspectionPackage);
    const offlineGrant = clone(input.offlineGrant);
    const storageDigest = await sha256Canonical(inspectionPackage);
    await this.atomic(
      [
        this.database.packages,
        this.database.offlineGrants,
        this.database.checklistResponses,
        this.database.outbox,
        this.database.syncState,
      ],
      async () => {
        const key: [string, string] = [this.subjectId, inspectionPackage.id];
        const existing = await this.database.packages.get(key);
        if (
          existing &&
          (existing.packageVersion !== inspectionPackage.packageVersion ||
            existing.packageDigest !== inspectionPackage.packageDigest)
        ) {
          const active = await this.activePackageOutbox(inspectionPackage.id);
          if (active.length > 0) {
            throw new FieldRepositoryError(
              "PACKAGE_REPLACEMENT_REQUIRES_SYNC",
              "A changed package cannot replace pending local work.",
            );
          }
        }
        await this.database.packages.put({
          id: inspectionPackage.id,
          subjectId: this.subjectId,
          auditId: inspectionPackage.auditId,
          organizationId: inspectionPackage.organizationId,
          packageVersion: inspectionPackage.packageVersion,
          schemaVersion: inspectionPackage.schemaVersion,
          protocolVersion: inspectionPackage.protocolVersion,
          installedVector: CURRENT_OFFLINE_VERSIONS,
          packageDigest: inspectionPackage.packageDigest,
          storageDigest,
          checkedOutAt: input.checkedOutAt,
          expiresAt: inspectionPackage.expiresAt,
          grantId: offlineGrant.grantId,
          accessState: "AVAILABLE",
          unavailableReason: null,
          localChecklistStatus: existing?.localChecklistStatus ?? inspectionPackage.checklistStatus,
          localChecklistRevision:
            existing?.localChecklistRevision ?? inspectionPackage.checklistRevision,
          pendingSubmissionOperationId: existing?.pendingSubmissionOperationId ?? null,
          inspectionPackage,
        });
        await this.database.offlineGrants.put({
          grantId: offlineGrant.grantId,
          subjectId: this.subjectId,
          organizationId: offlineGrant.organizationId,
          packageId: offlineGrant.packageId,
          packageVersion: offlineGrant.packageVersion,
          packageDigest: offlineGrant.packageDigest,
          deviceInstanceId: offlineGrant.deviceInstanceId,
          profileKeyId: offlineGrant.profileKeyId,
          assignmentRevision: offlineGrant.assignmentRevision,
          issuedAt: offlineGrant.issuedAt,
          expiresAt: offlineGrant.expiresAt,
          leaseIssuedAt: offlineGrant.leaseIssuedAt,
          leaseExpiresAt: offlineGrant.leaseExpiresAt,
          protocolVersion: offlineGrant.protocolVersion,
          offlineGrant,
        });
        for (const question of inspectionPackage.questions) {
          if (!question.currentResponse) continue;
          const responseKey: [string, string] = [this.subjectId, question.currentResponse.id];
          if (await this.database.checklistResponses.get(responseKey)) continue;
          await this.database.checklistResponses.put({
            ...clone(question.currentResponse),
            subjectId: this.subjectId,
            packageId: inspectionPackage.id,
            auditId: inspectionPackage.auditId,
            questionId: question.id,
            syncState: "ACKNOWLEDGED",
            operationId: null,
            tombstoned: false,
          });
        }
        const existingSync = await this.database.syncState.get(key);
        await this.database.syncState.put({
          subjectId: this.subjectId,
          packageId: inspectionPackage.id,
          grantId: offlineGrant.grantId,
          projectionVersion: existingSync?.projectionVersion ?? 0,
          cursor: existingSync?.cursor ?? null,
          lastSuccessAt: existingSync?.lastSuccessAt ?? null,
          lastErrorCode: null,
        });
      },
    );
  }

  async resumePackageWithServerCheckout(input: FieldCheckoutInput): Promise<void> {
    return this.checkoutPackage(input);
  }

  private async quarantine(packageId: string, reason: string): Promise<never> {
    await this.database.packages.update([this.subjectId, packageId], {
      accessState: "QUARANTINED",
      unavailableReason: reason,
    });
    throw new FieldPackageUnavailableError(reason, "The local package is preserved in quarantine.");
  }

  private async lockUnavailable(packageId: string, reason: string): Promise<never> {
    await this.database.packages.update([this.subjectId, packageId], {
      accessState: "LOCKED",
      unavailableReason: reason,
    });
    throw new FieldPackageUnavailableError(reason, "The local package is locked and preserved.");
  }

  private accessError(row: PackageRow): never {
    const code = row.unavailableReason ?? (row.accessState === "LOCKED" ? "PACKAGE_LOCKED" : "PACKAGE_QUARANTINED");
    throw new FieldPackageUnavailableError(code, "The local package is unavailable but preserved.");
  }

  private async validateStoredPackage(row: PackageRow): Promise<void> {
    if (row.accessState !== "AVAILABLE") this.accessError(row);
    const now = this.now().getTime();
    const packageExpiry = parseInstant(row.expiresAt);
    const grant = await this.database.offlineGrants.get([this.subjectId, row.grantId]);
    if (!grant) return this.quarantine(row.id, "OFFLINE_GRANT_MISSING");
    const grantIssuedAt = parseInstant(grant.issuedAt);
    const grantExpiry = parseInstant(grant.expiresAt);
    if (
      packageExpiry === null ||
      grantIssuedAt === null ||
      grantExpiry === null ||
      grantIssuedAt > now + CLOCK_SKEW_MS ||
      packageExpiry <= now ||
      grantExpiry <= now
    ) {
      await this.lockUnavailable(row.id, "OFFLINE_GRANT_EXPIRED");
    }
    if (
      !isFieldSchemaNOrNMinusOne(row.schemaVersion, CURRENT_FIELD_PACKAGE_SCHEMA_VERSION) ||
      row.protocolVersion !== CURRENT_FIELD_PROTOCOL_VERSION ||
      grant.protocolVersion !== CURRENT_FIELD_PROTOCOL_VERSION
    ) {
      await this.quarantine(row.id, "PACKAGE_SCHEMA_INCOMPATIBLE");
    }
    if (
      grant.subjectId !== this.subjectId ||
      grant.organizationId !== row.organizationId ||
      grant.packageId !== row.id ||
      grant.packageVersion !== row.packageVersion ||
      grant.packageDigest !== row.packageDigest ||
      grant.deviceInstanceId.trim() === "" ||
      grant.offlineGrant.grantId !== grant.grantId ||
      grant.offlineGrant.subjectId !== grant.subjectId ||
      grant.offlineGrant.organizationId !== grant.organizationId ||
      grant.offlineGrant.packageId !== grant.packageId ||
      grant.offlineGrant.packageVersion !== grant.packageVersion ||
      grant.offlineGrant.packageDigest !== grant.packageDigest ||
      grant.offlineGrant.deviceInstanceId !== grant.deviceInstanceId ||
      grant.offlineGrant.issuedAt !== grant.issuedAt ||
      grant.offlineGrant.expiresAt !== grant.expiresAt ||
      grant.offlineGrant.protocolVersion !== grant.protocolVersion ||
      row.inspectionPackage.id !== row.id ||
      row.inspectionPackage.auditId !== row.auditId ||
      row.inspectionPackage.organizationId !== row.organizationId ||
      row.inspectionPackage.packageDigest !== row.packageDigest
    ) {
      await this.quarantine(row.id, "PACKAGE_SCOPE_CORRUPTED");
    }
    const questions = new Map(
      row.inspectionPackage.questions.map((question) => [question.id, question]),
    );
    if (
      grant.offlineGrant.assignmentScope.questionIds.length === 0 ||
      grant.offlineGrant.assignmentScope.questionIds.some((questionId) => {
        const question = questions.get(questionId);
        return !question || !question.assignedInspectorUserIds.includes(this.subjectId);
      })
    ) {
      await this.quarantine(row.id, "PACKAGE_ASSIGNMENT_CORRUPTED");
    }
    if ((await sha256Canonical(row.inspectionPackage)) !== row.storageDigest) {
      await this.quarantine(row.id, "PACKAGE_CONTENT_CORRUPTED");
    }
  }

  private async assertPackageAvailable(packageId: string): Promise<void> {
    await this.requireReadWrite();
    const row = await this.database.packages.get([this.subjectId, packageId]);
    if (!row) {
      throw new FieldPackageUnavailableError("PACKAGE_NOT_CHECKED_OUT", "Offline package is absent.");
    }
    await this.validateStoredPackage(row);
  }

  async loadPackage(packageId: string): Promise<FieldPackageView | null> {
    await this.requireReadWrite();
    const row = await this.database.packages.get([this.subjectId, packageId]);
    if (!row) return null;
    await this.validateStoredPackage(row);
    const [responses, potentialFindingDrafts, attachmentManifests, outbox] = await Promise.all([
      this.database.checklistResponses
        .where("[subjectId+packageId]")
        .equals([this.subjectId, packageId])
        .toArray(),
      this.database.potentialFindingDrafts
        .where("[subjectId+packageId]")
        .equals([this.subjectId, packageId])
        .toArray(),
      this.database.attachmentManifests
        .where("[subjectId+packageId]")
        .equals([this.subjectId, packageId])
        .toArray(),
      this.activePackageOutbox(packageId),
    ]);
    const responseByQuestion = new Map(
      responses.filter((response) => !response.tombstoned).map((response) => [response.questionId, response]),
    );
    const inspectionPackage = clone(row.inspectionPackage);
    inspectionPackage.checklistStatus = row.localChecklistStatus;
    inspectionPackage.checklistRevision = row.localChecklistRevision;
    inspectionPackage.questions = inspectionPackage.questions.map((question) => {
      const local = responseByQuestion.get(question.id);
      return {
        ...question,
        currentResponse: local
          ? {
              id: local.id,
              questionId: local.questionId,
              answer: local.answer,
              comment: local.comment,
              revision: local.revision,
              updatedAt: local.updatedAt,
            }
          : question.currentResponse,
      };
    });
    return {
      inspectionPackage,
      accessState: row.accessState,
      responses: clone(responses),
      potentialFindingDrafts: clone(potentialFindingDrafts.filter((draft) => !draft.tombstoned)),
      attachmentManifests: clone(attachmentManifests),
      pendingOperationCount: outbox.length,
    };
  }

  private async requirePackage(packageId: string): Promise<{ packageRow: PackageRow; grant: OfflineGrant }> {
    const packageRow = await this.database.packages.get([this.subjectId, packageId]);
    if (!packageRow) {
      throw new FieldPackageUnavailableError("PACKAGE_NOT_CHECKED_OUT", "Offline package is absent.");
    }
    if (packageRow.accessState !== "AVAILABLE") this.accessError(packageRow);
    const grantRow = await this.database.offlineGrants.get([this.subjectId, packageRow.grantId]);
    if (!grantRow) {
      throw new FieldPackageUnavailableError("OFFLINE_GRANT_MISSING", "Offline grant is absent.");
    }
    return { packageRow, grant: grantRow.offlineGrant };
  }

  private requireCommand(grant: OfflineGrant, commandType: FieldSyncOperation["commandType"]): void {
    if (!grant.allowedCommandTypes.includes(commandType)) {
      throw new FieldRepositoryError("COMMAND_NOT_GRANTED", "The offline grant excludes this command.");
    }
  }

  private operationBase(input: {
    operationId: string;
    packageRow: PackageRow;
    grant: OfflineGrant;
    entityId: string;
    baseRevision: number | null;
    operationSequence: number;
    dependencies: string[];
  }) {
    return {
      operationId: input.operationId,
      protocolVersion: input.packageRow.protocolVersion,
      offlineGrantId: input.grant.grantId,
      packageId: input.packageRow.id,
      packageVersion: input.packageRow.packageVersion,
      packageRevision: Math.max(1, input.packageRow.inspectionPackage.checklistRevision),
      entityId: input.entityId,
      baseRevision: input.baseRevision,
      deviceInstanceId: input.grant.deviceInstanceId,
      actorSubject: this.subjectId,
      profileKeyId: input.grant.profileKeyId,
      operationSequence: input.operationSequence,
      dependencies: [...input.dependencies].sort(),
      clientOccurredAt: this.now().toISOString(),
    };
  }

  private async createOperation<TType extends FieldSyncOperation["commandType"], TPayload>(
    base: Omit<FieldOperationBase<TType, TPayload>, "payloadHash" | "requestHash" | "payload">,
    payload: TPayload,
  ): Promise<FieldOperationBase<TType, TPayload>> {
    const payloadHash = await Dexie.waitFor(sha256Canonical(payload));
    const candidate = {
      ...base,
      payloadHash,
      requestHash: "",
      payload,
    } as FieldOperationBase<TType, TPayload>;
    const requestHash = await Dexie.waitFor(fieldOperationRequestDigest(candidate as FieldSyncOperation));
    const authority = this.profileAuthority ? await Dexie.waitFor(this.profileAuthority) : null;
    if (authority) {
      if (authority.subjectId !== this.subjectId || authority.profileKeyId !== candidate.profileKeyId) {
        throw new FieldRepositoryError(
          "PROFILE_AUTHORITY_MISMATCH",
          "The persisted profile authority does not match the server-issued grant.",
        );
      }
    }
    const authorityProof = authority
      ? await Dexie.waitFor(authority.sign(new TextEncoder().encode(requestHash)))
      : undefined;
    return { ...candidate, requestHash, ...(authorityProof ? { authorityProof } : {}) };
  }

  private async existingOperation(operationId: string): Promise<OutboxRow | undefined> {
    return this.database.outbox.get([this.subjectId, operationId]);
  }

  private operationSequenceKey(packageId: string): string {
    return `operation-sequence:${this.subjectId}:${packageId}`;
  }

  private commitReceiptKey(operationId: string): string {
    return `commit-receipt:${this.subjectId}:${operationId}`;
  }

  private async nextOperationSequence(packageId: string): Promise<number> {
    const key = this.operationSequenceKey(packageId);
    const current = await this.database.foundation.get(key) as FoundationRow<{ nextSequence: number }> | undefined;
    const nextSequence = current?.value.nextSequence ?? 1;
    await this.database.foundation.put({ key, value: { nextSequence: nextSequence + 1 } });
    return nextSequence;
  }

  private async persistOutbox(input: Parameters<typeof createOutboxRow>[0]): Promise<void> {
    const operationSequence = input.operationSequence ?? input.operation.operationSequence;
    const entityHash = input.entityHash ?? input.operation.payloadHash ?? await Dexie.waitFor(sha256Canonical(input.operation.payload));
    const row = await Dexie.waitFor(createOutboxRow({ ...input, operationSequence, entityHash }));
    await this.database.outbox.put(row);
    const receipt: LocalCommitReceiptRow = {
      operationId: row.operationId,
      subjectId: this.subjectId,
      packageId: row.packageId,
      entityId: row.entityId,
      operationSequence: row.operationSequence,
      entityRevision: row.entityRevision,
      entityHash: row.entityHash,
      requestHash: row.requestDigest,
      committedAt: row.createdAt,
    };
    await this.database.foundation.put({ key: this.commitReceiptKey(row.operationId), value: receipt });
  }

  private async readBackLocalCommit(
    operationId: string,
    packageId: string,
    entityId: string,
  ): Promise<LocalCommitReceiptRow> {
    await this.requireReadWrite();
    return this.database.transaction(
      "r",
      [this.database.foundation, this.database.outbox],
      async () => {
        const receiptRow = await this.database.foundation.get(this.commitReceiptKey(operationId)) as
          FoundationRow<LocalCommitReceiptRow> | undefined;
        const outbox = await this.database.outbox.get([this.subjectId, operationId]);
        if (
          !receiptRow ||
          !outbox ||
          receiptRow.value.packageId !== packageId ||
          receiptRow.value.entityId !== entityId ||
          outbox.operationSequence !== receiptRow.value.operationSequence ||
          outbox.requestDigest !== receiptRow.value.requestHash
        ) {
          throw new FieldRepositoryError(
            "LOCAL_COMMIT_RECEIPT_MISSING",
            "The local commit receipt and outbox operation could not be read back together.",
          );
        }
        return clone(receiptRow.value);
      },
    );
  }

  private assertOperationReplay(existing: OutboxRow, input: {
    commandType: FieldSyncOperation["commandType"];
    packageId: string;
    entityId: string;
    payload: unknown;
  }): void {
    if (
      existing.commandType !== input.commandType ||
      existing.packageId !== input.packageId ||
      existing.entityId !== input.entityId ||
      canonicalJson(existing.operation.payload) !== canonicalJson(input.payload)
    ) {
      throw new FieldRepositoryError(
        "OPERATION_ID_REUSED",
        "An operation ID cannot be reused with a different command or payload.",
      );
    }
  }

  private async entityOutbox(entityId: string): Promise<OutboxRow[]> {
    return this.database.outbox
      .where("[subjectId+entityId]")
      .equals([this.subjectId, entityId])
      .toArray();
  }

  private async activePackageOutbox(packageId: string): Promise<OutboxRow[]> {
    const rows = await this.database.outbox
      .where("[subjectId+packageId]")
      .equals([this.subjectId, packageId])
      .toArray();
    return rows.filter((row) => isCausalDependencyState(row.state));
  }

  async saveChecklistResponse(input: SaveLocalChecklistResponseInput): Promise<ChecklistResponseRow> {
    const normalizedComment = input.comment.trim();
    await this.assertPackageAvailable(input.packageId);
    const saved: ChecklistResponseRow = await this.atomic(
      [
        this.database.packages,
        this.database.offlineGrants,
        this.database.checklistResponses,
        this.database.outbox,
        this.database.foundation,
      ],
      async () => {
        const { packageRow, grant } = await this.requirePackage(input.packageId);
        this.requireCommand(grant, "UPSERT_CHECKLIST_RESPONSE");
        if (packageRow.localChecklistStatus !== "IN_PROGRESS") {
          throw new FieldRepositoryError(
            "SUBMITTED_CHECKLIST_READ_ONLY",
            "A submitted checklist requires an online reasoned reopen.",
          );
        }
        const question = packageRow.inspectionPackage.questions.find(
          (candidate) => candidate.id === input.questionId,
        );
        if (!question) {
          throw new FieldRepositoryError("QUESTION_NOT_FOUND", "The exact package question is absent.");
        }
        if (
          !question.assignedInspectorUserIds.includes(this.subjectId) ||
          !grant.assignmentScope.questionIds.includes(question.id)
        ) {
          throw new FieldRepositoryError(
            "QUESTION_READ_ONLY",
            "Another Inspector's package question is read-only.",
          );
        }
        if (!question.allowedAnswers.includes(input.answer)) {
          throw new FieldRepositoryError("ANSWER_NOT_ALLOWED", "The immutable package excludes this answer.");
        }
        if (question.commentRequiredFor.includes(input.answer) && !normalizedComment) {
          throw new FieldRepositoryError("COMMENT_REQUIRED", "This answer requires an Inspector comment.");
        }
        const payload = {
          auditId: packageRow.auditId,
          questionId: question.id,
          answer: input.answer,
          comment: normalizedComment,
        };
        const duplicate = await this.existingOperation(input.operationId);
        if (duplicate) {
          this.assertOperationReplay(duplicate, {
            commandType: "UPSERT_CHECKLIST_RESPONSE",
            packageId: input.packageId,
            entityId: input.responseId,
            payload,
          });
          const existingResponse = await this.database.checklistResponses.get([
            this.subjectId,
            input.responseId,
          ]);
          if (!existingResponse) {
            throw new FieldRepositoryError("LOCAL_STATE_CORRUPTED", "The idempotent response is missing.");
          }
          return clone(existingResponse);
        }
        const existingResponse = await this.database.checklistResponses.get([
          this.subjectId,
          input.responseId,
        ]);
        if (existingResponse && existingResponse.questionId !== question.id) {
          throw new FieldRepositoryError(
            "RESPONSE_IDENTITY_CHANGED",
            "A response identity cannot move to another question.",
          );
        }
        const responseForQuestion = await this.database.checklistResponses
          .where("[subjectId+questionId]")
          .equals([this.subjectId, question.id])
          .filter((row) => row.packageId === packageRow.id && !row.tombstoned)
          .first();
        if (responseForQuestion && responseForQuestion.id !== input.responseId) {
          throw new FieldRepositoryError(
            "QUESTION_RESPONSE_IDENTITY_CHANGED",
            "A package question cannot acquire a second active response identity.",
          );
        }
        const activeEntityRows = (await this.entityOutbox(input.responseId)).filter(
          (row) => row.commandType === "UPSERT_CHECKLIST_RESPONSE" && !isTerminalOutboxState(row.state),
        );
        const inFlight = activeEntityRows.filter((row) => row.state === "IN_FLIGHT");
        const supersededOperationIds: string[] = [];
        for (const row of activeEntityRows.filter((candidate) => isUnsentOutboxState(candidate.state))) {
          row.state = "SUPERSEDED";
          row.supersededByOperationId = input.operationId;
          await this.database.outbox.put(row);
          supersededOperationIds.push(row.operationId);
        }
        for (const row of activeEntityRows.filter((candidate) => candidate.state === "CONFLICT")) {
          row.state = "SUPERSEDED";
          row.supersededByOperationId = input.operationId;
          await this.database.outbox.put(row);
          supersededOperationIds.push(row.operationId);
        }
        if (supersededOperationIds.length > 0) {
          const packageOperations = await this.database.outbox
            .where("[subjectId+packageId]")
            .equals([this.subjectId, packageRow.id])
            .toArray();
          for (const dependent of packageOperations) {
            if (!dependent.dependsOnOperationIds.some((candidate) => supersededOperationIds.includes(candidate))) {
              continue;
            }
            dependent.dependsOnOperationIds = [
              ...new Set([
                ...dependent.dependsOnOperationIds.filter(
                  (candidate) => !supersededOperationIds.includes(candidate),
                ),
                input.operationId,
              ]),
            ].sort();
            dependent.state = "BLOCKED_ON_DEPENDENCY";
            await this.database.outbox.put(dependent);
          }
        }
        const authoritativeRevision = existingResponse?.revision ?? question.currentResponse?.revision ?? 0;
        const baseRevision = inFlight.length > 0 || authoritativeRevision === 0
          ? null
          : authoritativeRevision;
        const dependencies = inFlight.map((row) => row.operationId).sort();
        const operationSequence = await this.nextOperationSequence(packageRow.id);
        const operation = await this.createOperation({
          ...this.operationBase({
            operationId: input.operationId,
            packageRow,
            grant,
            entityId: input.responseId,
            baseRevision,
            operationSequence,
            dependencies,
          }),
          commandType: "UPSERT_CHECKLIST_RESPONSE",
        }, payload);
        const response: ChecklistResponseRow = {
          id: input.responseId,
          subjectId: this.subjectId,
          packageId: packageRow.id,
          auditId: packageRow.auditId,
          questionId: question.id,
          answer: input.answer,
          comment: normalizedComment,
          revision: authoritativeRevision,
          syncState: "PENDING",
          updatedAt: operation.clientOccurredAt,
          operationId: operation.operationId,
          tombstoned: false,
        };
        await this.database.checklistResponses.put(response);
        await this.fault("after-checklist-response-write");
        await this.persistOutbox({
          subjectId: this.subjectId,
          operation,
          state: dependencies.length > 0 ? "BLOCKED_ON_DEPENDENCY" : "PENDING",
          createdAt: operation.clientOccurredAt,
          dependsOnOperationIds: dependencies,
        });
        return clone(response);
      },
    );
    await this.readBackLocalCommit(input.operationId, input.packageId, input.responseId);
    return saved;
  }

  async createPotentialFindingDraft(
    input: CreateLocalPotentialFindingInput,
  ): Promise<PotentialFindingDraftRow> {
    const title = input.title.trim();
    const description = input.description.trim();
    const requiredComment = input.requiredComment.trim();
    await this.assertPackageAvailable(input.packageId);
    const saved = await this.atomic(
      [
        this.database.packages,
        this.database.offlineGrants,
        this.database.checklistResponses,
        this.database.potentialFindingDrafts,
        this.database.outbox,
        this.database.foundation,
      ],
      async () => {
        const { packageRow, grant } = await this.requirePackage(input.packageId);
        this.requireCommand(grant, "CREATE_POTENTIAL_FINDING");
        if (packageRow.localChecklistStatus !== "IN_PROGRESS") {
          throw new FieldRepositoryError(
            "SUBMITTED_CHECKLIST_READ_ONLY",
            "Potential Findings cannot be drafted after submission.",
          );
        }
        const question = packageRow.inspectionPackage.questions.find(
          (candidate) => candidate.id === input.questionId,
        );
        if (
          !question ||
          !question.assignedInspectorUserIds.includes(this.subjectId) ||
          !grant.assignmentScope.questionIds.includes(input.questionId)
        ) {
          throw new FieldRepositoryError(
            "QUESTION_READ_ONLY",
            "Potential Finding authority requires the assigned Inspector.",
          );
        }
        const response = await this.database.checklistResponses.get([
          this.subjectId,
          input.checklistResponseId,
        ]);
        if (!response || response.packageId !== packageRow.id || response.questionId !== question.id) {
          throw new FieldRepositoryError(
            "CHECKLIST_RESPONSE_REQUIRED",
            "The exact local checklist response is required.",
          );
        }
        if (response.answer !== "NON_COMPLIANT" && response.answer !== "OBSERVATION") {
          throw new FieldRepositoryError(
            "POTENTIAL_FINDING_ANSWER_INVALID",
            "Only Non-Compliant or Observation may create a Potential Finding.",
          );
        }
        if (!title || !description || !requiredComment) {
          throw new FieldRepositoryError(
            "POTENTIAL_FINDING_FIELDS_REQUIRED",
            "Title, description, and required Inspector comment are mandatory.",
          );
        }
        const payload = {
          auditId: packageRow.auditId,
          questionId: question.id,
          checklistResponseId: response.id,
          expectedChecklistResponseRevision: response.revision || null,
          title,
          description,
          requiredComment,
          inspectionAttachmentIds: [...input.inspectionAttachmentIds],
        };
        const duplicate = await this.existingOperation(input.operationId);
        if (duplicate) {
          this.assertOperationReplay(duplicate, {
            commandType: "CREATE_POTENTIAL_FINDING",
            packageId: packageRow.id,
            entityId: input.localId,
            payload,
          });
          const existingDraft = await this.database.potentialFindingDrafts.get([
            this.subjectId,
            input.localId,
          ]);
          if (!existingDraft) {
            throw new FieldRepositoryError("LOCAL_STATE_CORRUPTED", "The idempotent draft is missing.");
          }
          return clone(existingDraft);
        }
        const activeDrafts = await this.database.potentialFindingDrafts
          .where("[subjectId+questionId]")
          .equals([this.subjectId, question.id])
          .filter(
            (draft) =>
              draft.packageId === packageRow.id &&
              !draft.tombstoned &&
              draft.syncState !== "REJECTED",
          )
          .toArray();
        if (activeDrafts.length > 0) {
          throw new FieldRepositoryError(
            "POTENTIAL_FINDING_ALREADY_EXISTS",
            "An active Potential Finding already exists for this response.",
          );
        }
        const dependencies = (await this.entityOutbox(response.id))
          .filter((row) => isCausalDependencyState(row.state))
          .map((row) => row.operationId)
          .sort();
        const operationSequence = await this.nextOperationSequence(packageRow.id);
        const operation = await this.createOperation({
          ...this.operationBase({
            operationId: input.operationId,
            packageRow,
            grant,
            entityId: input.localId,
            baseRevision: null,
            operationSequence,
            dependencies,
          }),
          commandType: "CREATE_POTENTIAL_FINDING",
        }, payload);
        const draft: PotentialFindingDraftRow = {
          id: input.localId,
          subjectId: this.subjectId,
          packageId: packageRow.id,
          auditId: packageRow.auditId,
          questionId: question.id,
          checklistResponseId: response.id,
          organizationId: packageRow.organizationId,
          title,
          description,
          requiredComment,
          inspectionAttachmentIds: [...input.inspectionAttachmentIds],
          baseRevision: payload.expectedChecklistResponseRevision,
          status: "PENDING_LEAD_REVIEW",
          syncState: "PENDING",
          updatedAt: operation.clientOccurredAt,
          operationId: operation.operationId,
          authoritativeEntityId: null,
          tombstoned: false,
        };
        await this.database.potentialFindingDrafts.put(draft);
        await this.fault("after-potential-finding-write");
        await this.persistOutbox({
          subjectId: this.subjectId,
          operation,
          state: dependencies.length > 0 ? "BLOCKED_ON_DEPENDENCY" : "PENDING",
          createdAt: operation.clientOccurredAt,
          dependsOnOperationIds: dependencies,
        });
        return clone(draft);
      },
    );
    await this.readBackLocalCommit(input.operationId, input.packageId, input.localId);
    return saved;
  }

  async submitChecklist(input: SubmitLocalChecklistInput): Promise<FieldChecklistSubmission> {
    await this.assertPackageAvailable(input.packageId);
    const saved: FieldChecklistSubmission = await this.atomic(
      [
        this.database.packages,
        this.database.offlineGrants,
        this.database.checklistResponses,
        this.database.outbox,
        this.database.foundation,
      ],
      async () => {
        const { packageRow, grant } = await this.requirePackage(input.packageId);
        this.requireCommand(grant, "SUBMIT_CHECKLIST");
        const payload = { auditId: packageRow.auditId };
        const duplicate = await this.existingOperation(input.operationId);
        if (duplicate) {
          this.assertOperationReplay(duplicate, {
            commandType: "SUBMIT_CHECKLIST",
            packageId: packageRow.id,
            entityId: packageRow.auditId,
            payload,
          });
          return {
            auditId: packageRow.auditId,
            checklistStatus: "SUBMITTED",
            checklistRevision: packageRow.localChecklistRevision,
            syncState: "PENDING",
            operationId: input.operationId,
          };
        }
        if (packageRow.localChecklistStatus !== "IN_PROGRESS") {
          throw new FieldRepositoryError(
            "SUBMITTED_CHECKLIST_READ_ONLY",
            "Only an in-progress checklist can be submitted offline.",
          );
        }
        const assigned = packageRow.inspectionPackage.questions.some((question) =>
          question.assignedInspectorUserIds.includes(this.subjectId),
        );
        const responses = await this.database.checklistResponses
          .where("[subjectId+packageId]")
          .equals([this.subjectId, packageRow.id])
          .toArray();
        if (!assigned || responses.length === 0) {
          throw new FieldRepositoryError(
            "CHECKLIST_RESPONSE_REQUIRED",
            "An assigned Inspector response is required before submission.",
          );
        }
        const dependencies = (await this.activePackageOutbox(packageRow.id))
          .map((row) => row.operationId)
          .sort();
        const operationSequence = await this.nextOperationSequence(packageRow.id);
        const operation = await this.createOperation({
          ...this.operationBase({
            operationId: input.operationId,
            packageRow,
            grant,
            entityId: packageRow.auditId,
            baseRevision: packageRow.localChecklistRevision,
            operationSequence,
            dependencies,
          }),
          commandType: "SUBMIT_CHECKLIST",
        }, payload);
        packageRow.localChecklistStatus = "SUBMITTED";
        packageRow.pendingSubmissionOperationId = input.operationId;
        await this.database.packages.put(packageRow);
        await this.fault("after-checklist-submission-write");
        await this.persistOutbox({
          subjectId: this.subjectId,
          operation,
          state: dependencies.length > 0 ? "BLOCKED_ON_DEPENDENCY" : "PENDING",
          createdAt: operation.clientOccurredAt,
          dependsOnOperationIds: dependencies,
        });
        return {
          auditId: packageRow.auditId,
          checklistStatus: "SUBMITTED",
          checklistRevision: packageRow.localChecklistRevision,
          syncState: "PENDING",
          operationId: input.operationId,
        };
      },
    );
    await this.readBackLocalCommit(input.operationId, input.packageId, saved.auditId);
    return saved;
  }

  private async requireAttachment(attachmentId: string): Promise<AttachmentManifestRow> {
    const manifest = await this.database.attachmentManifests.get([
      this.subjectId,
      attachmentId,
    ]);
    if (!manifest) {
      throw new FieldRepositoryError(
        "ATTACHMENT_MANIFEST_NOT_FOUND",
        "Inspection Attachment manifest is absent.",
      );
    }
    return manifest;
  }

  async createAttachmentManifest(
    input: CreateAttachmentManifestInput,
  ): Promise<AttachmentManifestRow> {
    await this.assertPackageAvailable(input.packageId);
    const fileName = input.fileName.trim();
    if (
      !input.attachmentId.trim() ||
      !input.operationId.trim() ||
      !fileName ||
      fileName.length > 255 ||
      /[\\/]/.test(fileName)
    ) {
      throw new FieldRepositoryError(
        "ATTACHMENT_IDENTITY_INVALID",
        "Attachment identity and filename must be safe non-empty values.",
      );
    }
    if (!isInspectionAttachmentMediaType(input.mediaType)) {
      throw new FieldRepositoryError(
        "ATTACHMENT_MEDIA_TYPE_NOT_ALLOWED",
        "Only PDF, JPEG, and PNG Inspection Attachments are allowed.",
      );
    }
    const mediaType: InspectionAttachmentMediaType = input.mediaType;
    if (
      !Number.isSafeInteger(input.byteSize) ||
      input.byteSize <= 0 ||
      input.byteSize > MAX_INSPECTION_ATTACHMENT_BYTES
    ) {
      throw new FieldRepositoryError(
        "ATTACHMENT_SIZE_NOT_ALLOWED",
        "Inspection Attachment size must be between 1 byte and 25 MB.",
      );
    }
    if (!input.temporaryOpfsPath || !input.finalOpfsPath) {
      throw new FieldRepositoryError(
        "ATTACHMENT_PATH_REQUIRED",
        "Temporary and final OPFS paths are required before staging.",
      );
    }

    return this.atomic(
      [
        this.database.packages,
        this.database.offlineGrants,
        this.database.checklistResponses,
        this.database.potentialFindingDrafts,
        this.database.attachmentManifests,
        this.database.outbox,
      ],
      async () => {
        const { packageRow, grant } = await this.requirePackage(input.packageId);
        this.requireCommand(grant, "REGISTER_INSPECTION_ATTACHMENT");
        const existing = await this.database.attachmentManifests.get([
          this.subjectId,
          input.attachmentId,
        ]);
        if (existing) {
          if (
            existing.packageId === input.packageId &&
            existing.checklistResponseId === input.checklistResponseId &&
            existing.potentialFindingLocalId === input.potentialFindingLocalId &&
            existing.fileName === fileName &&
            existing.declaredMediaType === mediaType &&
            existing.declaredByteSize === input.byteSize &&
            existing.plannedOperationId === input.operationId &&
            existing.temporaryOpfsPath === input.temporaryOpfsPath &&
            existing.finalOpfsPath === input.finalOpfsPath
          ) {
            return clone(existing);
          }
          throw new FieldRepositoryError(
            "ATTACHMENT_ID_REUSED",
            "An attachment ID cannot be reused with different metadata.",
          );
        }
        if (await this.existingOperation(input.operationId)) {
          throw new FieldRepositoryError(
            "OPERATION_ID_REUSED",
            "The planned attachment operation ID already exists.",
          );
        }
        const response = await this.database.checklistResponses.get([
          this.subjectId,
          input.checklistResponseId,
        ]);
        if (!response || response.packageId !== packageRow.id || response.tombstoned) {
          throw new FieldRepositoryError(
            "ATTACHMENT_RESPONSE_REQUIRED",
            "The exact non-tombstoned checklist response is required.",
          );
        }
        const question = packageRow.inspectionPackage.questions.find(
          (candidate) => candidate.id === response.questionId,
        );
        if (
          !question ||
          !question.assignedInspectorUserIds.includes(this.subjectId) ||
          !grant.assignmentScope.questionIds.includes(question.id)
        ) {
          throw new FieldRepositoryError(
            "QUESTION_READ_ONLY",
            "Inspection Attachment authority requires the assigned Inspector.",
          );
        }
        if (input.potentialFindingLocalId) {
          const potential = await this.database.potentialFindingDrafts.get([
            this.subjectId,
            input.potentialFindingLocalId,
          ]);
          if (
            !potential ||
            potential.packageId !== packageRow.id ||
            potential.checklistResponseId !== response.id ||
            potential.tombstoned
          ) {
            throw new FieldRepositoryError(
              "ATTACHMENT_POTENTIAL_FINDING_SCOPE_INVALID",
              "The Potential Finding link is outside the exact response.",
            );
          }
        }
        const duplicateFileName = await this.database.attachmentManifests
          .where("[subjectId+packageId]")
          .equals([this.subjectId, packageRow.id])
          .filter((manifest) => manifest.fileName.toLowerCase() === fileName.toLowerCase())
          .first();
        if (duplicateFileName) {
          throw new FieldRepositoryError(
            "ATTACHMENT_FILENAME_DUPLICATE",
            "An active Inspection Attachment already uses this filename in the package.",
          );
        }
        const createdAt = this.now().toISOString();
        const manifest: AttachmentManifestRow = {
          attachmentId: input.attachmentId,
          subjectId: this.subjectId,
          packageId: packageRow.id,
          auditId: packageRow.auditId,
          checklistResponseId: response.id,
          potentialFindingLocalId: input.potentialFindingLocalId,
          fileName,
          declaredMediaType: mediaType,
          declaredByteSize: input.byteSize,
          observedByteSize: null,
          expectedSha256: null,
          sha256: null,
          temporaryOpfsPath: input.temporaryOpfsPath,
          finalOpfsPath: input.finalOpfsPath,
          stagingState: "manifest_created",
          recoveryState: "MANIFEST_COMMITTED",
          syncState: "PENDING",
          plannedOperationId: input.operationId,
          operationId: null,
          authoritativeEntityId: null,
          quarantineReason: null,
          localBytesPresent: false,
          createdAt,
          updatedAt: createdAt,
          uploadStartedAt: null,
          acknowledgedAt: null,
          purgeEligibleAt: null,
        };
        await this.database.attachmentManifests.put(manifest);
        return clone(manifest);
      },
    );
  }

  async recordAttachmentExpectedDigest(
    attachmentId: string,
    expectedSha256: string,
  ): Promise<AttachmentManifestRow> {
    if (!/^sha256:[a-f0-9]{64}$/.test(expectedSha256)) {
      throw new FieldRepositoryError("ATTACHMENT_DIGEST_INVALID", "A SHA-256 digest is required.");
    }
    return this.atomic([this.database.attachmentManifests], async () => {
      const manifest = await this.requireAttachment(attachmentId);
      if (manifest.stagingState !== "manifest_created") {
        throw new FieldRepositoryError(
          "ATTACHMENT_STATE_INVALID",
          "The source digest is recorded only after manifest creation.",
        );
      }
      manifest.expectedSha256 = expectedSha256;
      manifest.updatedAt = this.now().toISOString();
      await this.database.attachmentManifests.put(manifest);
      return clone(manifest);
    });
  }

  async markAttachmentWriting(attachmentId: string): Promise<AttachmentManifestRow> {
    return this.atomic([this.database.attachmentManifests], async () => {
      const manifest = await this.requireAttachment(attachmentId);
      if (manifest.stagingState !== "manifest_created" || !manifest.expectedSha256) {
        throw new FieldRepositoryError(
          "ATTACHMENT_STATE_INVALID",
          "Attachment writing requires a manifest and source digest.",
        );
      }
      manifest.stagingState = "writing";
      manifest.recoveryState = "WRITING_TEMP";
      manifest.updatedAt = this.now().toISOString();
      await this.database.attachmentManifests.put(manifest);
      return clone(manifest);
    });
  }

  async markAttachmentBytesPresent(
    attachmentId: string,
    present: boolean,
  ): Promise<AttachmentManifestRow> {
    return this.atomic([this.database.attachmentManifests], async () => {
      const manifest = await this.requireAttachment(attachmentId);
      manifest.localBytesPresent = present;
      manifest.updatedAt = this.now().toISOString();
      await this.database.attachmentManifests.put(manifest);
      return clone(manifest);
    });
  }

  async markAttachmentRecovery(
    attachmentId: string,
    input: { state: "recovery_required" | "quarantined"; reason: string; localBytesPresent: boolean },
  ): Promise<AttachmentManifestRow> {
    return this.atomic([this.database.attachmentManifests], async () => {
      const manifest = await this.requireAttachment(attachmentId);
      manifest.stagingState = input.state;
      manifest.recoveryState = input.state === "quarantined" ? "QUARANTINED" : "RECOVERY_REQUIRED";
      manifest.quarantineReason = input.reason;
      manifest.localBytesPresent = input.localBytesPresent;
      manifest.updatedAt = this.now().toISOString();
      await this.database.attachmentManifests.put(manifest);
      return clone(manifest);
    });
  }

  async markAttachmentRecoveryPhase(
    attachmentId: string,
    recoveryState: Exclude<AttachmentRecoveryState, "LOCAL_READY" | "RECOVERY_REQUIRED" | "QUARANTINED">,
  ): Promise<AttachmentManifestRow> {
    return this.atomic([this.database.attachmentManifests], async () => {
      const manifest = await this.requireAttachment(attachmentId);
      manifest.recoveryState = recoveryState;
      manifest.updatedAt = this.now().toISOString();
      await this.database.attachmentManifests.put(manifest);
      return clone(manifest);
    });
  }

  async commitReadyAttachment(input: CommitReadyAttachmentInput): Promise<AttachmentManifestRow> {
    const existing = await this.getAttachmentManifest(input.attachmentId);
    if (!existing) {
      throw new FieldRepositoryError("ATTACHMENT_MANIFEST_NOT_FOUND", "Manifest is absent.");
    }
    await this.assertPackageAvailable(existing.packageId);
    if (!Number.isSafeInteger(input.observedByteSize) || input.observedByteSize <= 0) {
      throw new FieldRepositoryError("ATTACHMENT_SIZE_MISMATCH", "Observed byte size is invalid.");
    }
    if (!/^sha256:[a-f0-9]{64}$/.test(input.sha256)) {
      throw new FieldRepositoryError("ATTACHMENT_DIGEST_INVALID", "Observed SHA-256 is invalid.");
    }
    const saved = await this.atomic(
      [
        this.database.packages,
        this.database.offlineGrants,
        this.database.checklistResponses,
        this.database.potentialFindingDrafts,
        this.database.attachmentManifests,
        this.database.outbox,
        this.database.foundation,
      ],
      async () => {
        const manifest = await this.requireAttachment(input.attachmentId);
        const { packageRow, grant } = await this.requirePackage(manifest.packageId);
        this.requireCommand(grant, "REGISTER_INSPECTION_ATTACHMENT");
        if (
          manifest.declaredByteSize !== input.observedByteSize ||
          manifest.expectedSha256 !== input.sha256
        ) {
          throw new FieldRepositoryError(
            "ATTACHMENT_INTEGRITY_MISMATCH",
            "Observed attachment bytes do not match the manifest source.",
          );
        }
        const existingOperation = await this.existingOperation(manifest.plannedOperationId);
        if (existingOperation) {
          if (
            manifest.stagingState === "ready" &&
            manifest.operationId === manifest.plannedOperationId &&
            manifest.sha256 === input.sha256
          ) {
            return clone(manifest);
          }
          throw new FieldRepositoryError(
            "OPERATION_ID_REUSED",
            "Attachment operation exists without matching ready metadata.",
          );
        }
        const potential = manifest.potentialFindingLocalId
          ? await this.database.potentialFindingDrafts.get([
              this.subjectId,
              manifest.potentialFindingLocalId,
            ])
          : undefined;
        const payload = {
          auditId: packageRow.auditId,
          checklistResponseId: manifest.checklistResponseId,
          potentialFindingOperationId: potential?.operationId ?? null,
          fileName: manifest.fileName,
          mediaType: manifest.declaredMediaType,
          byteSize: input.observedByteSize,
          sha256: input.sha256,
        };
        const dependencyRows = await this.activePackageOutbox(packageRow.id);
        const dependencies = dependencyRows
          .filter(
            (row) =>
              row.operationId !== manifest.plannedOperationId &&
              (row.entityId === manifest.checklistResponseId ||
                row.operationId === potential?.operationId),
          )
          .map((row) => row.operationId)
          .sort();
        const operationSequence = await this.nextOperationSequence(packageRow.id);
        const operation = await this.createOperation({
          ...this.operationBase({
            operationId: manifest.plannedOperationId,
            packageRow,
            grant,
            entityId: manifest.attachmentId,
            baseRevision: null,
            operationSequence,
            dependencies,
          }),
          commandType: "REGISTER_INSPECTION_ATTACHMENT",
        }, payload);
        await this.fault("before-attachment-metadata-ready");
        manifest.observedByteSize = input.observedByteSize;
        manifest.sha256 = input.sha256;
        manifest.stagingState = "ready";
        manifest.recoveryState = "LOCAL_READY";
        manifest.syncState = "PENDING";
        manifest.operationId = operation.operationId;
        manifest.quarantineReason = null;
        manifest.localBytesPresent = true;
        manifest.updatedAt = operation.clientOccurredAt;
        await this.database.attachmentManifests.put(manifest);
        await this.fault("after-attachment-metadata-ready");
        await this.fault("before-attachment-outbox-create");
        await this.persistOutbox({
          subjectId: this.subjectId,
          operation,
          state: dependencies.length > 0 ? "BLOCKED_ON_DEPENDENCY" : "PENDING",
          createdAt: operation.clientOccurredAt,
          dependsOnOperationIds: dependencies,
        });
        await this.fault("after-attachment-outbox-create");
        return clone(manifest);
      },
    );
    await this.readBackLocalCommit(
      saved.operationId ?? saved.plannedOperationId,
      saved.packageId,
      input.attachmentId,
    );
    return saved;
  }

  async getAttachmentManifest(attachmentId: string): Promise<AttachmentManifestRow | null> {
    await this.requireReadWrite();
    return clone(
      (await this.database.attachmentManifests.get([this.subjectId, attachmentId])) ?? null,
    );
  }

  async listAttachmentManifests(): Promise<AttachmentManifestRow[]> {
    await this.requireReadWrite();
    return sortByKey(
      clone(await this.database.attachmentManifests.where("subjectId").equals(this.subjectId).toArray()),
      (manifest) => manifest.attachmentId,
    );
  }

  async beginAttachmentUpload(attachmentId: string): Promise<AttachmentManifestRow> {
    const existing = await this.getAttachmentManifest(attachmentId);
    if (!existing) {
      throw new FieldRepositoryError("ATTACHMENT_MANIFEST_NOT_FOUND", "Manifest is absent.");
    }
    await this.assertPackageAvailable(existing.packageId);
    return this.atomic([this.database.attachmentManifests, this.database.outbox], async () => {
      const manifest = await this.requireAttachment(attachmentId);
      const operation = manifest.operationId
        ? await this.database.outbox.get([this.subjectId, manifest.operationId])
        : undefined;
      const resumableSessionExists = manifest.stagingState === "uploading" && Boolean(manifest.uploadSessionId);
      if (
        (!resumableSessionExists && manifest.stagingState !== "ready") ||
        !manifest.authoritativeEntityId ||
        !operation ||
        operation.state !== "ACKNOWLEDGED"
      ) {
        throw new FieldRepositoryError(
          "ATTACHMENT_UPLOAD_NOT_READY",
          "Only a server-registered ready attachment may start byte upload.",
        );
      }
      if (resumableSessionExists) return clone(manifest);
      await this.fault("before-attachment-upload-start");
      manifest.stagingState = "uploading";
      manifest.uploadState = "OPEN";
      manifest.uploadStartedAt = this.now().toISOString();
      manifest.updatedAt = manifest.uploadStartedAt;
      await this.database.attachmentManifests.put(manifest);
      await this.fault("after-attachment-upload-start");
      return clone(manifest);
    });
  }

  async acknowledgeAttachment(input: AcknowledgeAttachmentInput): Promise<AttachmentManifestRow> {
    return this.atomic([this.database.attachmentManifests, this.database.outbox], async () => {
      const manifest = await this.requireAttachment(input.attachmentId);
      const operation = manifest.operationId
        ? await this.database.outbox.get([this.subjectId, manifest.operationId])
        : undefined;
      if (
        manifest.stagingState !== "uploading" ||
        !operation ||
        operation.state !== "ACKNOWLEDGED" ||
        !input.authoritativeEntityId.trim() ||
        parseInstant(input.acknowledgedAt) === null
      ) {
        throw new FieldRepositoryError(
          "ATTACHMENT_ACKNOWLEDGEMENT_INVALID",
          "Acknowledgement must target the exact in-flight attachment operation.",
        );
      }
      await this.fault("before-attachment-acknowledgement");
      manifest.stagingState = "acknowledged";
      manifest.syncState = "ACKNOWLEDGED";
      manifest.authoritativeEntityId = input.authoritativeEntityId;
      manifest.acknowledgedAt = input.acknowledgedAt;
      manifest.updatedAt = input.acknowledgedAt;
      await this.database.attachmentManifests.put(manifest);
      await this.fault("after-attachment-acknowledgement");
      return clone(manifest);
    });
  }

  async markAttachmentPurgeEligible(_attachmentId: string): Promise<never> {
    throw new FieldRepositoryError(
      "ATTACHMENT_PURGE_POLICY_NOT_APPROVED",
      "No owner-approved local attachment purge policy exists in this candidate.",
    );
  }

  async quarantineUnknownAttachmentPath(input: {
    path: string;
    byteSize: number;
    sha256: string;
  }): Promise<AttachmentManifestRow> {
    const pathDigest = await sha256Canonical({ subjectId: this.subjectId, path: input.path });
    const attachmentId = `UNKNOWN-${pathDigest.slice("sha256:".length, "sha256:".length + 24)}`;
    return this.atomic([this.database.attachmentManifests], async () => {
      const existingByPath = await this.database.attachmentManifests
        .where("subjectId")
        .equals(this.subjectId)
        .filter(
          (manifest) =>
            manifest.temporaryOpfsPath === input.path || manifest.finalOpfsPath === input.path,
        )
        .first();
      if (existingByPath) return clone(existingByPath);
      const timestamp = this.now().toISOString();
      const manifest: AttachmentManifestRow = {
        attachmentId,
        subjectId: this.subjectId,
        packageId: "__unknown__",
        auditId: "__unknown__",
        checklistResponseId: "__unknown__",
        potentialFindingLocalId: null,
        fileName: input.path.split("/").at(-1) ?? "unknown-opfs-bytes",
        declaredMediaType: "application/octet-stream",
        declaredByteSize: input.byteSize,
        observedByteSize: input.byteSize,
        expectedSha256: input.sha256,
        sha256: input.sha256,
        temporaryOpfsPath: null,
        finalOpfsPath: input.path,
        stagingState: "quarantined",
        recoveryState: "QUARANTINED",
        syncState: "CONFLICT",
        plannedOperationId: `NO-OP-${attachmentId}`,
        operationId: null,
        authoritativeEntityId: null,
        quarantineReason: "UNKNOWN_OPFS_BYTES",
        localBytesPresent: true,
        createdAt: timestamp,
        updatedAt: timestamp,
        uploadStartedAt: null,
        acknowledgedAt: null,
        purgeEligibleAt: null,
      };
      await this.database.attachmentManifests.put(manifest);
      return clone(manifest);
    });
  }

  async markOperationInFlight(operationId: string): Promise<OutboxRow> {
    return this.atomic([this.database.outbox], async () => {
      const row = await this.database.outbox.get([this.subjectId, operationId]);
      if (!row) throw new FieldRepositoryError("OPERATION_NOT_FOUND", "Outbox operation is absent.");
      if (row.state !== "PENDING") {
        throw new FieldRepositoryError(
          "OPERATION_NOT_DELIVERABLE",
          "Only a dependency-free pending operation may enter flight.",
        );
      }
      row.state = "IN_FLIGHT";
      row.attemptCount += 1;
      row.nextAttemptAt = new Date(this.now().getTime() + 30_000).toISOString();
      await this.database.outbox.put(row);
      return clone(row);
    });
  }

  async recoverInterruptedOperations(packageId: string): Promise<void> {
    await this.atomic([this.database.outbox], async () => {
      const rows = await this.database.outbox
        .where("[subjectId+packageId]")
        .equals([this.subjectId, packageId])
        .toArray();
      for (const row of rows) {
        if (row.state !== "IN_FLIGHT") continue;
        row.state = "FAILED_RETRYABLE";
        row.lastErrorCode = "INTERRUPTED_IN_FLIGHT";
        row.nextAttemptAt = this.now().toISOString();
        await this.database.outbox.put(row);
      }
    });
  }

  async releaseRetryableOperations(packageId: string, force: boolean): Promise<void> {
    const now = this.now().getTime();
    await this.atomic([this.database.outbox], async () => {
      const rows = await this.database.outbox
        .where("[subjectId+packageId]")
        .equals([this.subjectId, packageId])
        .toArray();
      for (const row of rows) {
        if (
          row.state !== "FAILED_RETRYABLE" ||
          (!force && (parseInstant(row.nextAttemptAt) ?? Number.POSITIVE_INFINITY) > now)
        ) {
          continue;
        }
        row.state = row.dependsOnOperationIds.length > 0 ? "BLOCKED_ON_DEPENDENCY" : "PENDING";
        await this.database.outbox.put(row);
      }
    });
  }

  private async markEntityPushState(
    row: OutboxRow,
    syncState: "CONFLICT" | "REJECTED",
    result: PushFieldOperationResult,
  ): Promise<void> {
    if (row.commandType === "UPSERT_CHECKLIST_RESPONSE") {
      const response = await this.database.checklistResponses.get([this.subjectId, row.entityId]);
      if (response) {
        response.syncState = syncState;
        if (syncState === "CONFLICT" && result.conflict?.authoritativeRevision != null) {
          response.revision = result.conflict?.authoritativeRevision ?? response.revision;
        }
        await this.database.checklistResponses.put(response);
      }
      return;
    }
    if (row.commandType === "CREATE_POTENTIAL_FINDING") {
      const draft = await this.database.potentialFindingDrafts.get([this.subjectId, row.entityId]);
      if (draft) {
        draft.syncState = syncState;
        if (syncState === "CONFLICT" && result.conflict?.authoritativeRevision != null) {
          draft.baseRevision = result.conflict?.authoritativeRevision ?? draft.baseRevision;
        }
        await this.database.potentialFindingDrafts.put(draft);
      }
      return;
    }
    if (row.commandType === "REGISTER_INSPECTION_ATTACHMENT") {
      const manifest = await this.database.attachmentManifests.get([this.subjectId, row.entityId]);
      if (manifest) {
        manifest.syncState = syncState;
        await this.database.attachmentManifests.put(manifest);
      }
    }
  }

  private async acknowledgeEntity(
    row: OutboxRow,
    result: PushFieldOperationResult,
  ): Promise<void> {
    const authoritativeRevision = result.authoritativeRevision;
    const authoritativeEntityId = result.authoritativeEntityId;
    if (authoritativeRevision === null || authoritativeEntityId === null) {
      throw new FieldRepositoryError(
        "ACKNOWLEDGEMENT_INCOMPLETE",
        "An applied field operation requires an authoritative entity and revision.",
      );
    }
    if (row.commandType === "UPSERT_CHECKLIST_RESPONSE") {
      const response = await this.database.checklistResponses.get([this.subjectId, row.entityId]);
      if (response) {
        response.revision = authoritativeRevision;
        if (response.operationId === row.operationId) {
          response.operationId = null;
          response.syncState = "ACKNOWLEDGED";
          response.updatedAt = result.acknowledgedAt;
        }
        await this.database.checklistResponses.put(response);
      }
      return;
    }
    if (row.commandType === "CREATE_POTENTIAL_FINDING") {
      const draft = await this.database.potentialFindingDrafts.get([this.subjectId, row.entityId]);
      if (draft) {
        draft.authoritativeEntityId = authoritativeEntityId;
        draft.baseRevision = authoritativeRevision;
        if (draft.operationId === row.operationId) {
          draft.operationId = null;
          draft.syncState = "ACKNOWLEDGED";
          draft.updatedAt = result.acknowledgedAt;
        }
        await this.database.potentialFindingDrafts.put(draft);
      }
      return;
    }
    if (row.commandType === "SUBMIT_CHECKLIST") {
      const packageRow = await this.database.packages.get([this.subjectId, row.packageId]);
      if (packageRow && packageRow.pendingSubmissionOperationId === row.operationId) {
        packageRow.localChecklistStatus = "SUBMITTED";
        packageRow.localChecklistRevision = authoritativeRevision;
        packageRow.pendingSubmissionOperationId = null;
        await this.database.packages.put(packageRow);
      }
      return;
    }
    const manifest = await this.database.attachmentManifests.get([this.subjectId, row.entityId]);
    if (manifest) {
      manifest.authoritativeEntityId = authoritativeEntityId;
      manifest.syncState = "PENDING";
      manifest.updatedAt = result.acknowledgedAt;
      await this.database.attachmentManifests.put(manifest);
    }
  }

  private async unblockAcknowledgedDependents(
    operationId: string,
    result: PushFieldOperationResult,
  ): Promise<void> {
    const rows = await this.database.outbox
      .where("[subjectId+packageId]")
      .equals([this.subjectId, (await this.database.outbox.get([this.subjectId, operationId]))!.packageId])
      .toArray();
    for (const dependent of rows) {
      if (!dependent.dependsOnOperationIds.includes(operationId)) continue;
      dependent.dependsOnOperationIds = dependent.dependsOnOperationIds.filter(
        (candidate) => candidate !== operationId,
      );
      if (result.authoritativeRevision !== null) {
        if (dependent.commandType === "UPSERT_CHECKLIST_RESPONSE") {
          dependent.baseRevision = result.authoritativeRevision;
          dependent.operation.baseRevision = result.authoritativeRevision;
        } else if (dependent.operation.commandType === "CREATE_POTENTIAL_FINDING") {
          dependent.operation.payload.expectedChecklistResponseRevision =
            result.authoritativeRevision;
        }
      }
      dependent.requestDigest = await Dexie.waitFor(
        fieldOperationRequestDigest(dependent.operation),
      );
      if (dependent.dependsOnOperationIds.length === 0) dependent.state = "PENDING";
      await this.database.outbox.put(dependent);
    }
  }

  private async quarantineCausalDependents(operationId: string): Promise<void> {
    const poisoned = new Set([operationId]);
    const rows = await this.database.outbox
      .where("[subjectId+packageId]")
      .equals([this.subjectId, (await this.database.outbox.get([this.subjectId, operationId]))?.packageId ?? ""])
      .toArray();
    let changed = true;
    while (changed) {
      changed = false;
      for (const row of rows) {
        if (
          isTerminalOutboxState(row.state) ||
          !row.dependsOnOperationIds.some((dependency) => poisoned.has(dependency))
        ) continue;
        row.state = "QUARANTINED";
        row.lastErrorCode = "DEPENDENCY_POISONED";
        await this.database.outbox.put(row);
        poisoned.add(row.operationId);
        changed = true;
      }
    }
  }

  async applyPushResult(
    operationId: string,
    result: PushFieldOperationResult,
  ): Promise<void> {
    await this.atomic(
      [
        this.database.packages,
        this.database.checklistResponses,
        this.database.potentialFindingDrafts,
        this.database.attachmentManifests,
        this.database.outbox,
      ],
      async () => {
        const row = await this.database.outbox.get([this.subjectId, operationId]);
        if (!row || row.state !== "IN_FLIGHT" || result.operationId !== operationId) {
          throw new FieldRepositoryError(
            "PUSH_RESULT_SCOPE_MISMATCH",
            "The push result must target the exact in-flight operation.",
          );
        }
        row.lastErrorCode = result.errorCode ?? result.conflict?.code ?? null;
        if (result.status === "accepted" || result.status === "already_applied") {
          row.state = "ACKNOWLEDGED";
          row.lastErrorCode = null;
          await this.database.outbox.put(row);
          await this.acknowledgeEntity(row, result);
          await this.unblockAcknowledgedDependents(operationId, result);
          return;
        }
        if (result.status === "retryable") {
          row.state = "FAILED_RETRYABLE";
          const backoffSeconds = Math.min(300, 2 ** Math.min(row.attemptCount, 8));
          row.nextAttemptAt = new Date(this.now().getTime() + backoffSeconds * 1_000).toISOString();
          await this.database.outbox.put(row);
          return;
        }
        row.state = result.status === "conflict" ? "CONFLICT" : "REJECTED";
        await this.database.outbox.put(row);
        await this.markEntityPushState(
          row,
          result.status === "conflict" ? "CONFLICT" : "REJECTED",
          result,
        );
        await this.quarantineCausalDependents(operationId);
        if (result.conflict?.code === "PACKAGE_REVOKED") {
          const packageRow = await this.database.packages.get([this.subjectId, row.packageId]);
          if (packageRow) {
            packageRow.accessState = "QUARANTINED";
            packageRow.unavailableReason = "PACKAGE_REVOKED";
            await this.database.packages.put(packageRow);
          }
        }
      },
    );
  }

  async listRegisteredAttachmentsReadyForUpload(packageId: string): Promise<AttachmentManifestRow[]> {
    await this.requireReadWrite();
    const manifests = await this.database.attachmentManifests
      .where("[subjectId+packageId]")
      .equals([this.subjectId, packageId])
      .toArray();
    const ready: AttachmentManifestRow[] = [];
    for (const manifest of manifests) {
      if (
        (manifest.stagingState !== "ready" && manifest.stagingState !== "uploading") ||
        !manifest.authoritativeEntityId ||
        !manifest.localBytesPresent ||
        !manifest.finalOpfsPath ||
        !manifest.operationId
      ) {
        continue;
      }
      const operation = await this.database.outbox.get([this.subjectId, manifest.operationId]);
      if (operation?.state === "ACKNOWLEDGED") ready.push(manifest);
    }
    return sortByKey(clone(ready), (manifest) => manifest.createdAt);
  }

  async beginRegisteredAttachmentUpload(attachmentId: string): Promise<AttachmentManifestRow> {
    return this.beginAttachmentUpload(attachmentId);
  }

  async recordAttachmentUploadSession(
    attachmentId: string,
    session: ResumableInspectionAttachmentUploadSession & { beginOperationId: string },
  ): Promise<AttachmentManifestRow> {
    return this.atomic([this.database.attachmentManifests], async () => {
      const manifest = await this.requireAttachment(attachmentId);
      if (manifest.stagingState !== "uploading" || !manifest.authoritativeEntityId) {
        throw new FieldRepositoryError("ATTACHMENT_UPLOAD_STATE_INVALID", "The attachment is not in an uploadable state.");
      }
      const sessionChanged = manifest.uploadSessionId !== session.uploadId || manifest.uploadSessionEpoch !== session.sessionEpoch;
      manifest.uploadSessionId = session.uploadId;
      manifest.uploadSessionEpoch = session.sessionEpoch;
      manifest.uploadPartSize = session.partSize;
      manifest.uploadReceivedParts = [...session.receivedParts].sort((left, right) => left - right);
      manifest.uploadAcknowledgedOffsets = [...session.acknowledgedOffsets];
      manifest.uploadPartHashes = { ...session.partHashes };
      if (sessionChanged) manifest.uploadPartObjectVersions = {};
      manifest.uploadWholeFileSha256 = session.wholeFileSha256;
      manifest.uploadExpiresAt = session.expiresAt;
      manifest.uploadBeginOperationId = session.beginOperationId;
      if (sessionChanged || !manifest.uploadCompleteOperationId) {
        manifest.uploadCompleteOperationId = `OP-ATTACHMENT-COMPLETE-${manifest.attachmentId}-EPOCH-${session.sessionEpoch}`;
      }
      manifest.uploadState = session.receivedParts.length > 0 ? "PARTIALLY_COMMITTED" : "OPEN";
      manifest.updatedAt = this.now().toISOString();
      await this.database.attachmentManifests.put(manifest);
      return clone(manifest);
    });
  }

  async recordAttachmentUploadPart(
    attachmentId: string,
    receipt: UploadPartReceipt,
  ): Promise<AttachmentManifestRow> {
    return this.atomic([this.database.attachmentManifests], async () => {
      const manifest = await this.requireAttachment(attachmentId);
      if (manifest.stagingState !== "uploading" || !manifest.uploadSessionId) {
        throw new FieldRepositoryError("ATTACHMENT_UPLOAD_STATE_INVALID", "The resumable upload session is absent.");
      }
      const received = new Set(manifest.uploadReceivedParts ?? []);
      received.add(receipt.partNumber);
      manifest.uploadReceivedParts = [...received].sort((left, right) => left - right);
      manifest.uploadAcknowledgedOffsets = [
        ...(manifest.uploadAcknowledgedOffsets ?? []),
        receipt.acknowledgedOffset,
      ].filter((value, index, values) => values.indexOf(value) === index).sort((left, right) => left - right);
      manifest.uploadPartHashes = { ...(manifest.uploadPartHashes ?? {}), [String(receipt.partNumber)]: receipt.sha256 };
      manifest.uploadPartObjectVersions = {
        ...(manifest.uploadPartObjectVersions ?? {}),
        [String(receipt.partNumber)]: receipt.objectVersion,
      };
      manifest.uploadState = "PARTIALLY_COMMITTED";
      manifest.updatedAt = this.now().toISOString();
      await this.database.attachmentManifests.put(manifest);
      return clone(manifest);
    });
  }

  async markAttachmentUploadCompleting(attachmentId: string): Promise<void> {
    await this.atomic([this.database.attachmentManifests], async () => {
      const manifest = await this.requireAttachment(attachmentId);
      if (manifest.stagingState !== "uploading" || !manifest.uploadSessionId) {
        throw new FieldRepositoryError("ATTACHMENT_UPLOAD_STATE_INVALID", "The resumable upload session is absent.");
      }
      manifest.uploadState = "COMPLETING";
      manifest.updatedAt = this.now().toISOString();
      await this.database.attachmentManifests.put(manifest);
    });
  }

  async markAttachmentUploadRetryable(attachmentId: string, _errorCode: string): Promise<void> {
    await this.atomic([this.database.attachmentManifests], async () => {
      const manifest = await this.requireAttachment(attachmentId);
      if (manifest.stagingState !== "uploading") {
        throw new FieldRepositoryError(
          "ATTACHMENT_UPLOAD_STATE_INVALID",
          "Only an uploading attachment may return to retry-ready state.",
        );
      }
      manifest.stagingState = "ready";
      manifest.syncState = "PENDING";
      manifest.updatedAt = this.now().toISOString();
      await this.database.attachmentManifests.put(manifest);
    });
  }

  async acknowledgeUploadedAttachment(input: AcknowledgeUploadedAttachmentInput): Promise<void> {
    await this.atomic([this.database.attachmentManifests, this.database.outbox], async () => {
      const manifest = await this.requireAttachment(input.attachmentId);
      const operation = manifest.operationId
        ? await this.database.outbox.get([this.subjectId, manifest.operationId])
        : undefined;
      if (
        manifest.stagingState !== "uploading" ||
        !operation ||
        operation.state !== "ACKNOWLEDGED" ||
        !input.authoritativeEntityId.trim() ||
        parseInstant(input.acknowledgedAt) === null
      ) {
        throw new FieldRepositoryError("ATTACHMENT_ACKNOWLEDGEMENT_INVALID", "Acknowledgement must target the exact in-flight attachment operation.");
      }
      manifest.stagingState = "acknowledged";
      manifest.uploadState = "COMPLETED";
      manifest.syncState = "ACKNOWLEDGED";
      manifest.authoritativeEntityId = input.authoritativeEntityId;
      manifest.acknowledgedAt = input.acknowledgedAt;
      manifest.uploadObjectVersion = input.objectVersion ?? manifest.uploadObjectVersion ?? null;
      manifest.observedByteSize = input.byteSize ?? manifest.observedByteSize;
      manifest.sha256 = input.sha256 ?? manifest.sha256;
      manifest.updatedAt = input.acknowledgedAt;
      await this.database.attachmentManifests.put(manifest);
    });
  }

  async listOutbox(
    packageId: string,
    options: { includeTerminal?: boolean } = {},
  ): Promise<OutboxRow[]> {
    await this.requireReadWrite();
    const rows = await this.database.outbox
      .where("[subjectId+packageId]")
      .equals([this.subjectId, packageId])
      .toArray();
    return sortByKey(
      clone(options.includeTerminal ? rows : rows.filter((row) => !isTerminalOutboxState(row.state))),
      (row) => `${row.createdAt}:${row.operationId}`,
    );
  }

  async listDeliverableOperations(packageId: string): Promise<OutboxRow[]> {
    return (await this.listOutbox(packageId)).filter((row) => row.state === "PENDING");
  }

  async getChecklistResponse(
    packageId: string,
    responseId: string,
  ): Promise<ChecklistResponseRow | null> {
    await this.requireReadWrite();
    const row = await this.database.checklistResponses.get([this.subjectId, responseId]);
    return row?.packageId === packageId ? clone(row) : null;
  }

  async getSyncState(packageId: string): Promise<SyncStateRow | null> {
    await this.requireReadWrite();
    return clone((await this.database.syncState.get([this.subjectId, packageId])) ?? null);
  }

  async getOfflineGrant(packageId: string): Promise<OfflineGrant | null> {
    await this.requireReadWrite();
    const row = await this.database.offlineGrants
      .where("[subjectId+packageId]")
      .equals([this.subjectId, packageId])
      .first();
    return row ? clone(row.offlineGrant) : null;
  }

  private async applyAuthorizedChange(packageRow: PackageRow, change: AuthorizedSyncChange): Promise<void> {
    if (change.kind === "checklist_response") {
      const question = packageRow.inspectionPackage.questions.find(
        (candidate) => candidate.id === change.value.questionId,
      );
      if (!question) {
        throw new FieldRepositoryError(
          "PULL_CHANGE_OUT_OF_SCOPE",
          "A pulled checklist response is outside the exact package.",
        );
      }
      const existing = await this.database.checklistResponses
        .where("[subjectId+questionId]")
        .equals([this.subjectId, question.id])
        .filter((row) => row.packageId === packageRow.id && !row.tombstoned)
        .first();
      const hasLocalWork = existing?.operationId
        ? (await this.existingOperation(existing.operationId))?.state !== "ACKNOWLEDGED"
        : false;
      await this.database.checklistResponses.put({
        ...(hasLocalWork && existing ? existing : clone(change.value)),
        id: existing?.id ?? change.value.id,
        subjectId: this.subjectId,
        packageId: packageRow.id,
        auditId: packageRow.auditId,
        questionId: question.id,
        revision: change.value.revision,
        syncState: hasLocalWork ? "PENDING" : "ACKNOWLEDGED",
        updatedAt: hasLocalWork && existing ? existing.updatedAt : change.value.updatedAt,
        operationId: hasLocalWork && existing ? existing.operationId : null,
        tombstoned: false,
      });
      return;
    }
    if (change.kind === "potential_finding") {
      if (change.value.auditId !== packageRow.auditId) {
        throw new FieldRepositoryError(
          "PULL_CHANGE_OUT_OF_SCOPE",
          "A pulled Potential Finding is outside the exact Audit.",
        );
      }
      const existing = await this.database.potentialFindingDrafts
        .filter(
          (row) =>
            row.subjectId === this.subjectId &&
            row.packageId === packageRow.id &&
            (row.id === change.value.id || row.authoritativeEntityId === change.value.id),
        )
        .first();
      await this.database.potentialFindingDrafts.put({
        id: existing?.id ?? change.value.id,
        subjectId: this.subjectId,
        packageId: packageRow.id,
        auditId: packageRow.auditId,
        questionId: change.value.questionId,
        checklistResponseId: existing?.checklistResponseId ?? "",
        organizationId: packageRow.organizationId,
        title: change.value.title,
        description: change.value.description,
        requiredComment: existing?.requiredComment ?? "",
        inspectionAttachmentIds: existing?.inspectionAttachmentIds ?? [],
        baseRevision: change.value.revision,
        status: change.value.status,
        syncState: existing?.syncState === "PENDING" ? "PENDING" : "ACKNOWLEDGED",
        updatedAt: existing?.updatedAt ?? this.now().toISOString(),
        operationId: existing?.syncState === "PENDING" ? existing.operationId : null,
        authoritativeEntityId: change.value.id,
        tombstoned: false,
      });
      return;
    }
    if (change.kind === "package_revoked") {
      if (change.packageId !== packageRow.id) {
        throw new FieldRepositoryError("PULL_CHANGE_OUT_OF_SCOPE", "Package revocation is out of scope.");
      }
      packageRow.accessState = "QUARANTINED";
      packageRow.unavailableReason = "PACKAGE_REVOKED";
      await this.database.packages.put(packageRow);
      return;
    }
    if (change.entityType === "checklist_response") {
      const response = await this.database.checklistResponses.get([
        this.subjectId,
        change.entityId,
      ]);
      if (response?.packageId === packageRow.id) {
        response.tombstoned = true;
        response.syncState = response.operationId ? "PENDING" : "TOMBSTONED";
        response.revision = Math.max(response.revision, change.revision);
        await this.database.checklistResponses.put(response);
      }
      return;
    }
    const draft = await this.database.potentialFindingDrafts.get([
      this.subjectId,
      change.entityId,
    ]);
    if (draft?.packageId === packageRow.id) {
      draft.tombstoned = true;
      draft.syncState = draft.operationId ? "PENDING" : "TOMBSTONED";
      draft.baseRevision = Math.max(draft.baseRevision ?? 0, change.revision);
      await this.database.potentialFindingDrafts.put(draft);
    }
  }

  async applyPullPage(input: ApplyFieldPullPageInput): Promise<void> {
    await this.assertPackageAvailable(input.packageId);
    await this.atomic(
      [
        this.database.packages,
        this.database.offlineGrants,
        this.database.checklistResponses,
        this.database.potentialFindingDrafts,
        this.database.outbox,
        this.database.syncState,
      ],
      async () => {
        const { packageRow } = await this.requirePackage(input.packageId);
        if (packageRow.grantId !== input.grantId) {
          throw new FieldRepositoryError("PULL_GRANT_SCOPE_MISMATCH", "Pull grant is not current.");
        }
        const state = await this.database.syncState.get([this.subjectId, input.packageId]);
        if (!state || state.grantId !== input.grantId || state.cursor !== input.expectedCursor) {
          throw new FieldRepositoryError("PULL_CURSOR_SCOPE_MISMATCH", "Pull cursor is not current.");
        }
        if (input.page.resnapshotRequired) {
          packageRow.accessState = "LOCKED";
          packageRow.unavailableReason = "RESNAPSHOT_REQUIRED";
          await this.database.packages.put(packageRow);
          await this.database.syncState.put({
            ...state,
            projectionVersion: input.page.projectionVersion,
            lastErrorCode: "RESNAPSHOT_REQUIRED",
          });
          return;
        }
        for (const change of input.page.changes) {
          await this.applyAuthorizedChange(packageRow, change);
        }
        await this.fault("before-pull-cursor-write");
        await this.database.syncState.put({
          ...state,
          projectionVersion: input.page.projectionVersion,
          cursor: input.page.nextCursor,
          lastSuccessAt: this.now().toISOString(),
          lastErrorCode: input.page.resnapshotRequired ? "RESNAPSHOT_REQUIRED" : null,
        });
      },
    );
  }

  async lockSubject(reason: "LOGOUT" | "USER_SWITCH" | string): Promise<void> {
    await this.atomic([this.database.packages], async () => {
      const rows = await this.database.packages.where("subjectId").equals(this.subjectId).toArray();
      for (const row of rows) {
        row.accessState = "LOCKED";
        row.unavailableReason = reason === "LOGOUT" || reason === "USER_SWITCH"
          ? "PACKAGE_LOCKED"
          : reason;
        await this.database.packages.put(row);
      }
    });
  }

  async exportSubjectSnapshot(
    options: { includeLocked?: boolean } = {},
  ): Promise<FieldSubjectSnapshot> {
    await this.requireReadWrite();
    const [packages, checklistResponses, potentialFindingDrafts, attachmentManifests, outbox, syncState] =
      await Promise.all([
        this.database.packages.where("subjectId").equals(this.subjectId).toArray(),
        this.database.checklistResponses.where("subjectId").equals(this.subjectId).toArray(),
        this.database.potentialFindingDrafts.where("subjectId").equals(this.subjectId).toArray(),
        this.database.attachmentManifests.where("subjectId").equals(this.subjectId).toArray(),
        this.database.outbox.where("subjectId").equals(this.subjectId).toArray(),
        this.database.syncState.where("subjectId").equals(this.subjectId).toArray(),
      ]);
    return {
      packages: sortByKey(
        clone(options.includeLocked ? packages : packages.filter((row) => row.accessState === "AVAILABLE")),
        (row) => row.id,
      ),
      checklistResponses: sortByKey(clone(checklistResponses), (row) => row.id),
      potentialFindingDrafts: sortByKey(clone(potentialFindingDrafts), (row) => row.id),
      attachmentManifests: sortByKey(clone(attachmentManifests), (row) => row.attachmentId),
      outbox: sortByKey(clone(outbox), (row) => row.operationId),
      syncState: sortByKey(clone(syncState), (row) => row.packageId),
    };
  }
}

export function createBrowserFieldRepository(
  subjectId: string,
  now?: () => Date,
  profileAuthority?: ProfileAuthority | Promise<ProfileAuthority>,
): IndexedDbFieldRepository {
  return new IndexedDbFieldRepository({
    subjectId,
    now,
    profileAuthority: profileAuthority ?? createBrowserProfileAuthority(subjectId),
  });
}

export async function outboxPayloadIsImmutable(
  before: OutboxRow,
  after: OutboxRow,
): Promise<boolean> {
  return (
    before.requestDigest === after.requestDigest &&
    before.requestDigest === (await fieldOperationRequestDigest(before.operation)) &&
    canonicalJson(before.operation) === canonicalJson(after.operation)
  );
}

export function toChecklistResponseView(row: ChecklistResponseRow): ChecklistResponseView {
  return {
    id: row.id,
    questionId: row.questionId,
    answer: row.answer,
    comment: row.comment,
    revision: row.revision,
    updatedAt: row.updatedAt,
  };
}
