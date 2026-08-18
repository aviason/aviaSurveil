import "fake-indexeddb/auto";

import Dexie from "dexie";
import { afterEach, describe, expect, it } from "vitest";

import type { InspectionPackage, OfflineGrant } from "../../src/backend/backend";
import { OfflineFieldDatabase } from "../../src/offline/db";
import {
  FieldAtomicWriteError,
  IndexedDbFieldRepository,
  type FieldTransactionBoundary,
} from "../../src/offline/field-repository";
import {
  AttachmentTerminationError,
  InspectionAttachmentStore,
  type AttachmentFileWriter,
  type InspectionAttachmentFileSystem,
  type InspectionAttachmentStageBoundary,
} from "../../src/offline/opfs-inspection-attachment-store";
import { reconcileInspectionAttachments } from "../../src/offline/attachment-recovery";
import { sha256InspectionAttachment } from "../../src/offline/inspection-attachment-hash-worker";

const subjectId = "USR-INSPECTOR-AMINA";
const packageId = "PKG-CAB-2026-001";
const responseId = "RESP-CAB-EMEQ-PBE-001";
const questionId = "CAB-EMEQ-PBE-001";
const now = new Date("2026-07-21T08:00:00.000Z");
const databaseNames = new Set<string>();

function packageFixture(): InspectionPackage {
  return {
    id: packageId,
    auditId: "AUD-2026-001",
    organizationId: "ORG-FLY-NAMIBIA",
    organizationName: "Fly Namibia",
    title: "2026 Cabin Inspection - Fly Namibia",
    packageVersion: 1,
    schemaVersion: 1,
    protocolVersion: 1,
    templateVersionId: "TPL-CAB-001-V1",
    packageDigest: "sha256:candidate-cabin-package-v1",
    expiresAt: "2026-07-24T08:00:00.000Z",
    checklistStatus: "IN_PROGRESS",
    checklistRevision: 3,
    questions: [
      {
        id: questionId,
        sectionId: "Emergency equipment",
        prompt: "Protective breathing equipment serviceable and accessible?",
        regulatoryReference: "Configured cabin reference",
        expectedEvidence: "Serviceability record",
        allowedAnswers: ["COMPLIANT", "NON_COMPLIANT", "OBSERVATION", "NOT_CHECKED"],
        commentRequiredFor: ["NON_COMPLIANT", "OBSERVATION"],
        assignedInspectorUserIds: [subjectId],
        currentResponse: null,
      },
    ],
  };
}

function grantFixture(inspectionPackage = packageFixture()): OfflineGrant {
  return {
    grantId: "GRANT-CANDIDATE-001",
    subjectId,
    organizationId: inspectionPackage.organizationId,
    packageId: inspectionPackage.id,
    packageVersion: inspectionPackage.packageVersion,
    packageDigest: inspectionPackage.packageDigest,
    allowedCommandTypes: [
      "UPSERT_CHECKLIST_RESPONSE",
      "CREATE_POTENTIAL_FINDING",
      "SUBMIT_CHECKLIST",
      "REGISTER_INSPECTION_ATTACHMENT",
    ],
    assignmentScope: { questionIds: [questionId] },
    deviceInstanceId: "DEVICE-CANDIDATE-001",
    issuedAt: "2026-07-21T07:59:00.000Z",
    expiresAt: "2026-07-22T08:00:00.000Z",
    protocolVersion: 1,
  };
}

class MemoryAttachmentFileSystem implements InspectionAttachmentFileSystem {
  readonly files = new Map<string, Uint8Array>();
  readonly removed: string[] = [];
  quotaFailure = false;
  corruptReads = false;
  corruptFinalReads = false;

  async createWriter(path: string): Promise<AttachmentFileWriter> {
    if (this.quotaFailure) throw new DOMException("quota exhausted", "QuotaExceededError");
    this.files.set(path, new Uint8Array());
    return {
      write: async (chunk) => {
        if (this.quotaFailure) throw new DOMException("quota exhausted", "QuotaExceededError");
        const previous = this.files.get(path) ?? new Uint8Array();
        const next = new Uint8Array(previous.byteLength + chunk.byteLength);
        next.set(previous);
        next.set(chunk, previous.byteLength);
        this.files.set(path, next);
      },
      flush: async () => undefined,
    };
  }

  async read(path: string): Promise<Uint8Array> {
    const value = this.files.get(path);
    if (!value) throw new DOMException("missing", "NotFoundError");
    const result = value.slice();
    if ((this.corruptReads || (this.corruptFinalReads && path.endsWith(".bin"))) && result.byteLength > 0) result[0] ^= 0xff;
    return result;
  }

  async exists(path: string): Promise<boolean> {
    return this.files.has(path);
  }

  async promote(temporaryPath: string, finalPath: string): Promise<void> {
    const bytes = this.files.get(temporaryPath);
    if (!bytes) throw new DOMException("missing", "NotFoundError");
    this.files.set(finalPath, bytes.slice());
  }

  async list(directoryPath: string): Promise<string[]> {
    return [...this.files.keys()].filter((path) => path.startsWith(`${directoryPath}/`)).sort();
  }

  async remove(path: string): Promise<void> {
    this.removed.push(path);
    this.files.delete(path);
  }
}

function createDatabase(): OfflineFieldDatabase {
  const name = `aviasurveil360-attachment-test-${crypto.randomUUID()}`;
  databaseNames.add(name);
  return new OfflineFieldDatabase({ name });
}

async function setup(
  input: { transactionFault?: (boundary: FieldTransactionBoundary) => void; fileSystem?: MemoryAttachmentFileSystem } = {},
) {
  const database = createDatabase();
  const setupRepository = new IndexedDbFieldRepository({ database, subjectId, now: () => now });
  const inspectionPackage = packageFixture();
  await setupRepository.checkoutPackage({
    inspectionPackage,
    offlineGrant: grantFixture(inspectionPackage),
    checkedOutAt: now.toISOString(),
  });
  await setupRepository.saveChecklistResponse({
    operationId: "OP-RESPONSE-ATTACHMENT",
    packageId,
    responseId,
    questionId,
    answer: "NON_COMPLIANT",
    comment: "PBE record unavailable.",
  });
  const repository = new IndexedDbFieldRepository({
    database,
    subjectId,
    now: () => now,
    transactionFault: input.transactionFault,
  });
  const fileSystem = input.fileSystem ?? new MemoryAttachmentFileSystem();
  const store = new InspectionAttachmentStore({
    repository,
    fileSystem,
    hasher: { sha256: sha256InspectionAttachment },
    now: () => now,
    chunkSize: 3,
  });
  return { database, repository, fileSystem, store };
}

async function acknowledgeAttachmentDependency(database: OfflineFieldDatabase): Promise<void> {
  await database.transaction("rw", database.outbox, database.attachmentManifests, async () => {
    const responseOperation = await database.outbox.get([subjectId, "OP-RESPONSE-ATTACHMENT"]);
    if (!responseOperation) throw new Error("response operation fixture missing");
    await database.outbox.put({ ...responseOperation, state: "ACKNOWLEDGED" });

    const attachmentOperation = await database.outbox.get([subjectId, stageInput.operationId]);
    if (!attachmentOperation) throw new Error("attachment operation fixture missing");
    await database.outbox.put({
      ...attachmentOperation,
      state: "ACKNOWLEDGED",
      dependsOnOperationIds: [],
    });
    const manifest = await database.attachmentManifests.get([subjectId, stageInput.attachmentId]);
    if (!manifest) throw new Error("attachment manifest fixture missing");
    await database.attachmentManifests.put({
      ...manifest,
      authoritativeEntityId: "ATT-SERVER-001",
      syncState: "PENDING",
    });
  });
}

const stageInput = {
  attachmentId: "ATT-LOCAL-001",
  operationId: "OP-ATTACHMENT-001",
  packageId,
  checklistResponseId: responseId,
  potentialFindingLocalId: null,
  fileName: "pbe-serviceability.pdf",
  mediaType: "application/pdf",
  bytes: new TextEncoder().encode("candidate inspection attachment"),
};

afterEach(async () => {
  await Promise.all([...databaseNames].map((name) => Dexie.delete(name)));
  databaseNames.clear();
});

describe("manifest-first Inspection Attachment staging", () => {
  it("leaves no metadata or bytes when terminated before manifest creation", async () => {
    const { repository, fileSystem } = await setup();
    const store = new InspectionAttachmentStore({
      repository,
      fileSystem,
      hasher: { sha256: sha256InspectionAttachment },
      now: () => now,
      fault: ({ boundary }) => {
        if (boundary === "before-manifest-create") {
          throw new AttachmentTerminationError(boundary);
        }
      },
    });

    await expect(store.stage(stageInput)).rejects.toBeInstanceOf(AttachmentTerminationError);
    expect(await repository.getAttachmentManifest(stageInput.attachmentId)).toBeNull();
    expect(fileSystem.files.size).toBe(0);
  });

  it("commits a ready manifest and typed registration outbox only after verified bytes", async () => {
    const { repository, fileSystem, store } = await setup();

    const manifest = await store.stage(stageInput);

    expect(manifest).toMatchObject({
      attachmentId: stageInput.attachmentId,
      fileName: stageInput.fileName,
      stagingState: "ready",
      recoveryState: "LOCAL_READY",
      syncState: "PENDING",
      observedByteSize: stageInput.bytes.byteLength,
      localBytesPresent: true,
    });
    expect(manifest.sha256).toMatch(/^sha256:[a-f0-9]{64}$/);
    expect(await fileSystem.exists(manifest.finalOpfsPath!)).toBe(true);
    expect(await fileSystem.exists(manifest.temporaryOpfsPath!)).toBe(true);
    expect(await repository.listOutbox(packageId)).toEqual(
      expect.arrayContaining([
        expect.objectContaining({
          operationId: stageInput.operationId,
          commandType: "REGISTER_INSPECTION_ATTACHMENT",
          state: "BLOCKED_ON_DEPENDENCY",
          dependsOnOperationIds: ["OP-RESPONSE-ATTACHMENT"],
          operation: expect.objectContaining({
            payload: expect.objectContaining({
              fileName: stageInput.fileName,
              mediaType: stageInput.mediaType,
              byteSize: stageInput.bytes.byteLength,
              sha256: manifest.sha256,
            }),
          }),
        }),
      ]),
    );
  });

  it("orders registration after the exact response and Potential Finding operations", async () => {
    const { repository, store } = await setup();
    await repository.createPotentialFindingDraft({
      operationId: "OP-PF-ATTACHMENT",
      packageId,
      localId: "PF-LOCAL-ATTACHMENT",
      questionId,
      checklistResponseId: responseId,
      title: "PBE serviceability not confirmed",
      description: "The configured check could not confirm PBE serviceability.",
      requiredComment: "PBE record unavailable.",
      inspectionAttachmentIds: [],
    });

    await store.stage({
      ...stageInput,
      potentialFindingLocalId: "PF-LOCAL-ATTACHMENT",
    });

    expect(await repository.listOutbox(packageId)).toEqual(
      expect.arrayContaining([
        expect.objectContaining({
          operationId: stageInput.operationId,
          state: "BLOCKED_ON_DEPENDENCY",
          dependsOnOperationIds: ["OP-PF-ATTACHMENT", "OP-RESPONSE-ATTACHMENT"],
          operation: expect.objectContaining({
            payload: expect.objectContaining({
              potentialFindingOperationId: "OP-PF-ATTACHMENT",
            }),
          }),
        }),
      ]),
    );
  });

  it("creates the manifest before any OPFS path can exist", async () => {
    const { repository, fileSystem } = await setup();
    const store = new InspectionAttachmentStore({
      repository,
      fileSystem,
      hasher: { sha256: sha256InspectionAttachment },
      now: () => now,
      fault: ({ boundary }) => {
        if (boundary === "after-manifest-create") {
          expect(fileSystem.files.size).toBe(0);
          throw new AttachmentTerminationError(boundary);
        }
      },
    });

    await expect(store.stage(stageInput)).rejects.toBeInstanceOf(AttachmentTerminationError);
    expect(await repository.getAttachmentManifest(stageInput.attachmentId)).toMatchObject({
      stagingState: "manifest_created",
      localBytesPresent: false,
    });
  });

  it.each([
    "before-source-hash",
    "after-source-hash",
    "before-temporary-path-create",
    "after-temporary-path-create",
    "before-write-chunk",
    "after-write-chunk",
    "before-flush",
    "after-flush",
    "before-stored-hash",
    "after-stored-hash",
    "before-final-promotion",
    "after-final-promotion",
    "before-ready-commit",
    "after-ready-commit",
  ] as InspectionAttachmentStageBoundary[])(
    "preserves recoverable metadata or bytes after termination at %s",
    async (boundary) => {
      const { repository, fileSystem } = await setup();
      let fired = false;
      const store = new InspectionAttachmentStore({
        repository,
        fileSystem,
        hasher: { sha256: sha256InspectionAttachment },
        chunkSize: 3,
        fault: (point) => {
          if (!fired && point.boundary === boundary) {
            fired = true;
            throw new AttachmentTerminationError(boundary);
          }
        },
      });

      await expect(store.stage(stageInput)).rejects.toBeInstanceOf(AttachmentTerminationError);
      expect(fired).toBe(true);
      const beforePaths = [...fileSystem.files.keys()].sort();
      const report = await reconcileInspectionAttachments({
        repository,
        fileSystem,
        hasher: { sha256: sha256InspectionAttachment },
      });
      const afterPaths = [...fileSystem.files.keys()].sort();
      expect(afterPaths.length).toBeGreaterThanOrEqual(beforePaths.length === 0 ? 0 : 1);
      expect(fileSystem.removed).toEqual([]);
      expect(report.deletedPaths).toEqual([]);
      expect(await repository.getAttachmentManifest(stageInput.attachmentId)).toBeTruthy();
    },
  );

  it("exposes a termination injection point before and after every chunk write", async () => {
    const { repository, fileSystem } = await setup();
    const points: Array<{ boundary: InspectionAttachmentStageBoundary; chunkIndex?: number }> = [];
    const store = new InspectionAttachmentStore({
      repository,
      fileSystem,
      hasher: { sha256: sha256InspectionAttachment },
      chunkSize: 3,
      fault: (point) => {
        points.push(point);
      },
    });

    await store.stage(stageInput);
    const expectedChunkIndexes = Array.from(
      { length: Math.ceil(stageInput.bytes.byteLength / 3) },
      (_, index) => index,
    );
    expect(
      points.filter((point) => point.boundary === "before-write-chunk").map((point) => point.chunkIndex),
    ).toEqual(expectedChunkIndexes);
    expect(
      points.filter((point) => point.boundary === "after-write-chunk").map((point) => point.chunkIndex),
    ).toEqual(expectedChunkIndexes);
  });

  it.each([
    "before-attachment-metadata-ready",
    "after-attachment-metadata-ready",
    "before-attachment-outbox-create",
    "after-attachment-outbox-create",
  ] as FieldTransactionBoundary[])(
    "never splits ready metadata from its outbox at %s",
    async (boundary) => {
      const { repository, store } = await setup({
        transactionFault: (candidate) => {
          if (candidate === boundary) throw new DOMException("quota exhausted", "QuotaExceededError");
        },
      });

      await expect(store.stage(stageInput)).rejects.toBeInstanceOf(FieldAtomicWriteError);
      expect(await repository.listOutbox(packageId)).not.toEqual(
        expect.arrayContaining([expect.objectContaining({ operationId: stageInput.operationId })]),
      );
      expect(await repository.getAttachmentManifest(stageInput.attachmentId)).not.toMatchObject({
        stagingState: "ready",
      });
    },
  );

  it("quarantines hash mismatch without deleting the temporary bytes", async () => {
    const fileSystem = new MemoryAttachmentFileSystem();
    fileSystem.corruptReads = true;
    const { repository, store } = await setup({ fileSystem });

    await expect(store.stage(stageInput)).rejects.toMatchObject({ code: "ATTACHMENT_HASH_MISMATCH" });
    const manifest = await repository.getAttachmentManifest(stageInput.attachmentId);
    expect(manifest).toMatchObject({
      stagingState: "quarantined",
      quarantineReason: "ATTACHMENT_HASH_MISMATCH",
      localBytesPresent: true,
    });
    expect(await fileSystem.exists(manifest!.temporaryOpfsPath!)).toBe(true);
    expect(fileSystem.removed).toEqual([]);
  });

  it("does not mark local-ready when the promoted final-path readback is corrupt", async () => {
    const fileSystem = new MemoryAttachmentFileSystem();
    const { repository, store } = await setup({ fileSystem });
    fileSystem.corruptFinalReads = true;

    await expect(store.stage(stageInput)).rejects.toMatchObject({
      code: "ATTACHMENT_FINAL_READBACK_MISMATCH",
    });
    expect(await repository.getAttachmentManifest(stageInput.attachmentId)).toMatchObject({
      recoveryState: "QUARANTINED",
      stagingState: "quarantined",
    });
  });

  it("quarantines a mismatching temp object when a valid final object is also present", async () => {
    const { repository, fileSystem, store } = await setup();
    const manifest = await store.stage(stageInput);
    fileSystem.files.set(manifest.temporaryOpfsPath!, new Uint8Array([0xff, 0xee]));

    const report = await reconcileInspectionAttachments({
      repository,
      fileSystem,
      hasher: { sha256: sha256InspectionAttachment },
    });

    expect(report.quarantinedAttachmentIds).toEqual([stageInput.attachmentId]);
    expect(await repository.getAttachmentManifest(stageInput.attachmentId)).toMatchObject({
      recoveryState: "QUARANTINED",
      quarantineReason: "TEMP_HASH_MISMATCH",
    });
    expect(await fileSystem.exists(manifest.finalOpfsPath!)).toBe(true);
    expect(await fileSystem.exists(manifest.temporaryOpfsPath!)).toBe(true);
  });

  it("keeps a partial temp object in restartable recovery instead of quarantining it", async () => {
    const { repository, fileSystem } = await setup();
    const store = new InspectionAttachmentStore({
      repository,
      fileSystem,
      hasher: { sha256: sha256InspectionAttachment },
      chunkSize: 3,
      fault: ({ boundary, chunkIndex }) => {
        if (boundary === "after-write-chunk" && chunkIndex === 0) {
          throw new AttachmentTerminationError(boundary);
        }
      },
    });

    await expect(store.stage(stageInput)).rejects.toBeInstanceOf(AttachmentTerminationError);
    const report = await reconcileInspectionAttachments({
      repository,
      fileSystem,
      hasher: { sha256: sha256InspectionAttachment },
    });

    expect(report.quarantinedAttachmentIds).toEqual([]);
    expect(await repository.getAttachmentManifest(stageInput.attachmentId)).toMatchObject({
      recoveryState: "RECOVERY_REQUIRED",
      stagingState: "recovery_required",
    });
    expect(await fileSystem.exists((await repository.getAttachmentManifest(stageInput.attachmentId))!.temporaryOpfsPath!)).toBe(true);
  });

  it("preserves a recovery manifest when OPFS reports quota exhaustion", async () => {
    const fileSystem = new MemoryAttachmentFileSystem();
    fileSystem.quotaFailure = true;
    const { repository, store } = await setup({ fileSystem });

    await expect(store.stage(stageInput)).rejects.toMatchObject({ code: "ATTACHMENT_QUOTA_EXCEEDED" });
    expect(await repository.getAttachmentManifest(stageInput.attachmentId)).toMatchObject({
      stagingState: "recovery_required",
      quarantineReason: "ATTACHMENT_QUOTA_EXCEEDED",
    });
    expect(await repository.listOutbox(packageId)).not.toEqual(
      expect.arrayContaining([expect.objectContaining({ operationId: stageInput.operationId })]),
    );
  });

  it("rejects a duplicate active filename without replacing the first bytes", async () => {
    const { fileSystem, store } = await setup();
    const first = await store.stage(stageInput);
    const firstBytes = await fileSystem.read(first.finalOpfsPath!);

    await expect(
      store.stage({
        ...stageInput,
        attachmentId: "ATT-LOCAL-002",
        operationId: "OP-ATTACHMENT-002",
      }),
    ).rejects.toMatchObject({ code: "ATTACHMENT_FILENAME_DUPLICATE" });
    expect(await fileSystem.read(first.finalOpfsPath!)).toEqual(firstBytes);
  });
});

describe("attachment recovery and no-delete policy", () => {
  it("reports referenced missing bytes as blocking and keeps the manifest", async () => {
    const { repository, fileSystem, store } = await setup();
    const manifest = await store.stage(stageInput);
    fileSystem.files.delete(manifest.finalOpfsPath!);
    fileSystem.files.delete(manifest.temporaryOpfsPath!);

    const report = await reconcileInspectionAttachments({
      repository,
      fileSystem,
      hasher: { sha256: sha256InspectionAttachment },
    });

    expect(report.blocking).toEqual([
      expect.objectContaining({ attachmentId: stageInput.attachmentId, code: "REFERENCED_BYTES_MISSING" }),
    ]);
    expect(await repository.getAttachmentManifest(stageInput.attachmentId)).toMatchObject({
      stagingState: "recovery_required",
    });
    expect(report.deletedPaths).toEqual([]);
  });

  it("creates quarantine metadata for unknown bytes and never auto-deletes them", async () => {
    const { repository, fileSystem } = await setup();
    const directory = `aviasurveil360-inspection-attachments/${encodeURIComponent(subjectId)}`;
    const unknownPath = `${directory}/unknown.partial`;
    fileSystem.files.set(unknownPath, new Uint8Array([1, 2, 3]));

    const report = await reconcileInspectionAttachments({
      repository,
      fileSystem,
      hasher: { sha256: sha256InspectionAttachment },
    });

    expect(report.quarantinedUnknownPaths).toEqual([unknownPath]);
    expect(await fileSystem.exists(unknownPath)).toBe(true);
    expect(fileSystem.removed).toEqual([]);
    expect(await repository.listAttachmentManifests()).toEqual(
      expect.arrayContaining([
        expect.objectContaining({
          stagingState: "quarantined",
          quarantineReason: "UNKNOWN_OPFS_BYTES",
          finalOpfsPath: unknownPath,
        }),
      ]),
    );
  });

  it("preserves ready attachment bytes and outbox state across database restart", async () => {
    const { database, fileSystem, store } = await setup();
    const manifest = await store.stage(stageInput);
    database.close();

    const reopenedDatabase = new OfflineFieldDatabase({ name: database.name });
    const reopened = new IndexedDbFieldRepository({
      database: reopenedDatabase,
      subjectId,
      now: () => now,
    });
    const report = await reconcileInspectionAttachments({
      repository: reopened,
      fileSystem,
      hasher: { sha256: sha256InspectionAttachment },
    });

    expect(report.blocking).toEqual([]);
    expect(await reopened.getAttachmentManifest(manifest.attachmentId)).toMatchObject({
      stagingState: "ready",
      syncState: "PENDING",
    });
    expect(await reopened.listOutbox(packageId)).toEqual(
      expect.arrayContaining([expect.objectContaining({ operationId: stageInput.operationId })]),
    );
    expect(await fileSystem.exists(manifest.finalOpfsPath!)).toBe(true);
  });

  it("restores verified temporary bytes to the final path for an already-ready manifest", async () => {
    const { repository, fileSystem, store } = await setup();
    const manifest = await store.stage(stageInput);
    const bytes = await fileSystem.read(manifest.finalOpfsPath!);
    fileSystem.files.delete(manifest.finalOpfsPath!);
    fileSystem.files.set(manifest.temporaryOpfsPath!, bytes);

    const report = await reconcileInspectionAttachments({
      repository,
      fileSystem,
      hasher: { sha256: sha256InspectionAttachment },
    });

    expect(report.recoveredAttachmentIds).toEqual([stageInput.attachmentId]);
    expect(await fileSystem.exists(manifest.finalOpfsPath!)).toBe(true);
    expect(await fileSystem.exists(manifest.temporaryOpfsPath!)).toBe(true);
    expect(await repository.listOutbox(packageId)).toEqual(
      expect.arrayContaining([expect.objectContaining({ operationId: stageInput.operationId })]),
    );
  });

  it("quarantines a recovery promotion when the fresh final-path readback fails", async () => {
    const { repository, fileSystem, store } = await setup();
    const manifest = await store.stage(stageInput);
    const bytes = await fileSystem.read(manifest.finalOpfsPath!);
    fileSystem.files.delete(manifest.finalOpfsPath!);
    fileSystem.files.set(manifest.temporaryOpfsPath!, bytes);
    fileSystem.corruptFinalReads = true;

    const report = await reconcileInspectionAttachments({
      repository,
      fileSystem,
      hasher: { sha256: sha256InspectionAttachment },
    });

    expect(report.quarantinedAttachmentIds).toEqual([stageInput.attachmentId]);
    expect(await repository.getAttachmentManifest(stageInput.attachmentId)).toMatchObject({
      recoveryState: "QUARANTINED",
      quarantineReason: "ATTACHMENT_FINAL_READBACK_MISMATCH",
    });
  });

  it.each([
    "before-attachment-upload-start",
    "after-attachment-upload-start",
    "before-attachment-acknowledgement",
    "after-attachment-acknowledgement",
  ] as FieldTransactionBoundary[])("keeps upload lifecycle atomic at %s", async (boundary) => {
    const { database, fileSystem, store } = await setup();
    await store.stage(stageInput);
    await acknowledgeAttachmentDependency(database);
    const failing = new IndexedDbFieldRepository({
      database,
      subjectId,
      now: () => now,
      transactionFault: (candidate) => {
        if (candidate === boundary) throw new Error(`terminated at ${boundary}`);
      },
    });

    if (boundary.includes("upload-start")) {
      await expect(failing.beginAttachmentUpload(stageInput.attachmentId)).rejects.toBeInstanceOf(
        FieldAtomicWriteError,
      );
      expect(await failing.getAttachmentManifest(stageInput.attachmentId)).toMatchObject({
        stagingState: "ready",
      });
    } else {
      await store.beginUpload(stageInput.attachmentId);
      await expect(
        failing.acknowledgeAttachment({
          attachmentId: stageInput.attachmentId,
          authoritativeEntityId: "ATT-SERVER-001",
          acknowledgedAt: now.toISOString(),
        }),
      ).rejects.toBeInstanceOf(FieldAtomicWriteError);
      expect(await failing.getAttachmentManifest(stageInput.attachmentId)).toMatchObject({
        stagingState: "uploading",
      });
    }
    expect(fileSystem.removed).toEqual([]);
  });

  it("keeps acknowledged bytes and rejects purge without an approved owner policy", async () => {
    const { database, repository, fileSystem, store } = await setup();
    const ready = await store.stage(stageInput);
    await acknowledgeAttachmentDependency(database);
    await store.beginUpload(stageInput.attachmentId);
    const acknowledged = await store.acknowledge({
      attachmentId: stageInput.attachmentId,
      authoritativeEntityId: "ATT-SERVER-001",
    });

    expect(acknowledged).toMatchObject({
      stagingState: "acknowledged",
      syncState: "ACKNOWLEDGED",
      authoritativeEntityId: "ATT-SERVER-001",
    });
    expect(await fileSystem.exists(ready.finalOpfsPath!)).toBe(true);
    await expect(repository.markAttachmentPurgeEligible(stageInput.attachmentId)).rejects.toMatchObject({
      code: "ATTACHMENT_PURGE_POLICY_NOT_APPROVED",
    });
    await expect(store.purge(stageInput.attachmentId)).rejects.toMatchObject({
      code: "ATTACHMENT_PURGE_NOT_ELIGIBLE",
    });
    expect(await fileSystem.exists(ready.finalOpfsPath!)).toBe(true);
    expect(fileSystem.removed).toEqual([]);
  });
});
