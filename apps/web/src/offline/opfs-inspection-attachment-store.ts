import type { AttachmentManifestRow } from "./db";
import {
  FieldAtomicWriteError,
  FieldRepositoryError,
  type IndexedDbFieldRepository,
} from "./field-repository";
import {
  DedicatedInspectionAttachmentHasher,
  type InspectionAttachmentHasher,
} from "./inspection-attachment-hash-worker";

export type InspectionAttachmentStageBoundary =
  | "before-manifest-create"
  | "after-manifest-create"
  | "before-source-hash"
  | "after-source-hash"
  | "before-temporary-path-create"
  | "after-temporary-path-create"
  | "before-write-chunk"
  | "after-write-chunk"
  | "before-flush"
  | "after-flush"
  | "before-stored-hash"
  | "after-stored-hash"
  | "before-final-promotion"
  | "after-final-promotion"
  | "before-ready-commit"
  | "after-ready-commit";

export interface InspectionAttachmentStagePoint {
  boundary: InspectionAttachmentStageBoundary;
  chunkIndex?: number;
}

export type InspectionAttachmentStageFault = (
  point: InspectionAttachmentStagePoint,
) => void | Promise<void>;

export interface AttachmentFileWriter {
  write(chunk: Uint8Array): Promise<void>;
  flush(): Promise<void>;
}

export interface InspectionAttachmentFileSystem {
  createWriter(path: string): Promise<AttachmentFileWriter>;
  read(path: string): Promise<Uint8Array>;
  exists(path: string): Promise<boolean>;
  promote(temporaryPath: string, finalPath: string): Promise<void>;
  list(directoryPath: string): Promise<string[]>;
  remove(path: string): Promise<void>;
}

export interface StageInspectionAttachmentInput {
  attachmentId: string;
  operationId: string;
  packageId: string;
  checklistResponseId: string;
  potentialFindingLocalId: string | null;
  fileName: string;
  mediaType: string;
  bytes: Uint8Array;
}

interface InspectionAttachmentStoreOptions {
  repository: IndexedDbFieldRepository;
  fileSystem: InspectionAttachmentFileSystem;
  hasher: InspectionAttachmentHasher;
  now?: () => Date;
  chunkSize?: number;
  fault?: InspectionAttachmentStageFault;
}

export class AttachmentTerminationError extends Error {
  constructor(readonly boundary: string) {
    super(`Attachment staging terminated at ${boundary}.`);
    this.name = "AttachmentTerminationError";
  }
}

export class AttachmentStagingError extends Error {
  constructor(
    readonly code: string,
    message: string,
    options?: ErrorOptions,
  ) {
    super(message, options);
    this.name = "AttachmentStagingError";
  }
}

export function inspectionAttachmentDirectory(subjectId: string): string {
  return `aviasurveil360-inspection-attachments/${encodeURIComponent(subjectId)}`;
}

export function inspectionAttachmentPaths(subjectId: string, attachmentId: string) {
  const directory = inspectionAttachmentDirectory(subjectId);
  const safeId = encodeURIComponent(attachmentId);
  return {
    directory,
    temporaryPath: `${directory}/${safeId}.partial`,
    finalPath: `${directory}/${safeId}.bin`,
  };
}

function isQuotaError(error: unknown): boolean {
  return (
    error instanceof DOMException &&
    (error.name === "QuotaExceededError" || error.name === "NS_ERROR_DOM_QUOTA_REACHED")
  );
}

export class InspectionAttachmentStore {
  readonly repository: IndexedDbFieldRepository;
  readonly fileSystem: InspectionAttachmentFileSystem;
  readonly hasher: InspectionAttachmentHasher;
  private readonly now: () => Date;
  private readonly chunkSize: number;
  private readonly fault?: InspectionAttachmentStageFault;

  constructor(options: InspectionAttachmentStoreOptions) {
    this.repository = options.repository;
    this.fileSystem = options.fileSystem;
    this.hasher = options.hasher;
    this.now = options.now ?? (() => new Date());
    this.chunkSize = options.chunkSize ?? 256 * 1024;
    this.fault = options.fault;
    if (!Number.isSafeInteger(this.chunkSize) || this.chunkSize <= 0) {
      throw new AttachmentStagingError("ATTACHMENT_CHUNK_SIZE_INVALID", "Chunk size must be positive.");
    }
  }

  private async at(boundary: InspectionAttachmentStageBoundary, chunkIndex?: number): Promise<void> {
    await this.fault?.({ boundary, chunkIndex });
  }

  private async preserveFailure(
    attachmentId: string,
    error: unknown,
  ): Promise<never> {
    if (error instanceof AttachmentTerminationError) throw error;
    if (
      error instanceof AttachmentStagingError &&
      (error.code === "ATTACHMENT_HASH_MISMATCH" ||
        error.code === "ATTACHMENT_SIZE_MISMATCH" ||
        error.code === "ATTACHMENT_FINAL_READBACK_MISMATCH")
    ) {
      throw error;
    }
    const manifest = await this.repository.getAttachmentManifest(attachmentId).catch(() => null);
    if (!manifest) throw error;
    const localBytesPresent = Boolean(
      (manifest.temporaryOpfsPath && (await this.fileSystem.exists(manifest.temporaryOpfsPath))) ||
      (manifest.finalOpfsPath && (await this.fileSystem.exists(manifest.finalOpfsPath))),
    );
    const code = isQuotaError(error)
      ? "ATTACHMENT_QUOTA_EXCEEDED"
      : error instanceof FieldAtomicWriteError
        ? "ATTACHMENT_READY_TRANSACTION_ABORTED"
        : "ATTACHMENT_STAGING_INTERRUPTED";
    await this.repository.markAttachmentRecovery(attachmentId, {
      state: "recovery_required",
      reason: code,
      localBytesPresent,
    });
    if (error instanceof FieldAtomicWriteError || error instanceof FieldRepositoryError) throw error;
    throw new AttachmentStagingError(code, "Inspection Attachment staging was preserved for recovery.", {
      cause: error,
    });
  }

  async stage(input: StageInspectionAttachmentInput): Promise<AttachmentManifestRow> {
    const bytes = Uint8Array.from(input.bytes);
    const paths = inspectionAttachmentPaths(this.repository.subjectId, input.attachmentId);
    try {
      await this.at("before-manifest-create");
      await this.repository.createAttachmentManifest({
        attachmentId: input.attachmentId,
        operationId: input.operationId,
        packageId: input.packageId,
        checklistResponseId: input.checklistResponseId,
        potentialFindingLocalId: input.potentialFindingLocalId,
        fileName: input.fileName,
        mediaType: input.mediaType,
        byteSize: bytes.byteLength,
        temporaryOpfsPath: paths.temporaryPath,
        finalOpfsPath: paths.finalPath,
      });
      await this.at("after-manifest-create");

      await this.at("before-source-hash");
      const expectedSha256 = await this.hasher.sha256(bytes);
      await this.repository.recordAttachmentExpectedDigest(input.attachmentId, expectedSha256);
      await this.at("after-source-hash");
      await this.repository.markAttachmentWriting(input.attachmentId);

      await this.at("before-temporary-path-create");
      const writer = await this.fileSystem.createWriter(paths.temporaryPath);
      await this.repository.markAttachmentBytesPresent(input.attachmentId, true);
      await this.at("after-temporary-path-create");

      for (let offset = 0, chunkIndex = 0; offset < bytes.byteLength; offset += this.chunkSize) {
        const chunk = bytes.slice(offset, Math.min(bytes.byteLength, offset + this.chunkSize));
        await this.at("before-write-chunk", chunkIndex);
        await writer.write(chunk);
        await this.at("after-write-chunk", chunkIndex);
        chunkIndex += 1;
      }
      await this.at("before-flush");
      await writer.flush();
      await this.at("after-flush");
      await this.repository.markAttachmentRecoveryPhase(input.attachmentId, "FLUSHED");

      const storedBytes = await this.fileSystem.read(paths.temporaryPath);
      await this.at("before-stored-hash");
      const storedSha256 = await this.hasher.sha256(storedBytes);
      await this.at("after-stored-hash");
      if (storedBytes.byteLength !== bytes.byteLength) {
        await this.repository.markAttachmentRecovery(input.attachmentId, {
          state: "quarantined",
          reason: "ATTACHMENT_SIZE_MISMATCH",
          localBytesPresent: true,
        });
        throw new AttachmentStagingError(
          "ATTACHMENT_SIZE_MISMATCH",
          "Stored attachment byte count differs from the manifest.",
        );
      }
      if (storedSha256 !== expectedSha256) {
        await this.repository.markAttachmentRecovery(input.attachmentId, {
          state: "quarantined",
          reason: "ATTACHMENT_HASH_MISMATCH",
          localBytesPresent: true,
        });
        throw new AttachmentStagingError(
          "ATTACHMENT_HASH_MISMATCH",
          "Stored attachment hash differs from the selected file.",
        );
      }
      await this.repository.markAttachmentRecoveryPhase(input.attachmentId, "HASH_VERIFIED");

      await this.at("before-final-promotion");
      await this.fileSystem.promote(paths.temporaryPath, paths.finalPath);
      await this.at("after-final-promotion");
      const finalBytes = await this.fileSystem.read(paths.finalPath);
      const finalSha256 = await this.hasher.sha256(finalBytes);
      if (finalBytes.byteLength !== bytes.byteLength || finalSha256 !== expectedSha256) {
        await this.repository.markAttachmentRecovery(input.attachmentId, {
          state: "quarantined",
          reason: "ATTACHMENT_FINAL_READBACK_MISMATCH",
          localBytesPresent: true,
        });
        throw new AttachmentStagingError(
          "ATTACHMENT_FINAL_READBACK_MISMATCH",
          "Promoted final attachment bytes failed the fresh size/hash readback.",
        );
      }
      await this.repository.markAttachmentRecoveryPhase(input.attachmentId, "PROMOTED");
      await this.at("before-ready-commit");
      const ready = await this.repository.commitReadyAttachment({
        attachmentId: input.attachmentId,
        observedByteSize: storedBytes.byteLength,
        sha256: storedSha256,
      });
      await this.at("after-ready-commit");
      return ready;
    } catch (error) {
      return this.preserveFailure(input.attachmentId, error);
    }
  }

  async beginUpload(attachmentId: string): Promise<AttachmentManifestRow> {
    return this.repository.beginAttachmentUpload(attachmentId);
  }

  async acknowledge(input: {
    attachmentId: string;
    authoritativeEntityId: string;
  }): Promise<AttachmentManifestRow> {
    return this.repository.acknowledgeAttachment({
      ...input,
      acknowledgedAt: this.now().toISOString(),
    });
  }

  async purge(attachmentId: string): Promise<never> {
    const manifest = await this.repository.getAttachmentManifest(attachmentId);
    if (!manifest || manifest.stagingState !== "purge_eligible" || !manifest.authoritativeEntityId) {
      throw new AttachmentStagingError(
        "ATTACHMENT_PURGE_NOT_ELIGIBLE",
        "Explicit purge requires an acknowledged, owner-approved purge-eligible manifest.",
      );
    }
    throw new AttachmentStagingError(
      "ATTACHMENT_PURGE_POLICY_NOT_IMPLEMENTED",
      "Local byte purge remains disabled in this candidate.",
    );
  }
}

interface DirectoryEntriesHandle extends FileSystemDirectoryHandle {
  entries(): AsyncIterableIterator<[string, FileSystemHandle]>;
}

export class BrowserInspectionAttachmentFileSystem implements InspectionAttachmentFileSystem {
  private async root(): Promise<FileSystemDirectoryHandle> {
    if (!("storage" in navigator) || typeof navigator.storage.getDirectory !== "function") {
      throw new AttachmentStagingError("OPFS_UNAVAILABLE", "OPFS is unavailable in this browser.");
    }
    return navigator.storage.getDirectory();
  }

  private async directory(path: string, create: boolean): Promise<FileSystemDirectoryHandle> {
    let directory = await this.root();
    for (const segment of path.split("/").filter(Boolean)) {
      directory = await directory.getDirectoryHandle(segment, { create });
    }
    return directory;
  }

  private splitFilePath(path: string): { directoryPath: string; fileName: string } {
    const segments = path.split("/").filter(Boolean);
    const fileName = segments.pop();
    if (!fileName) throw new AttachmentStagingError("OPFS_PATH_INVALID", "OPFS file path is invalid.");
    return { directoryPath: segments.join("/"), fileName };
  }

  private async file(path: string, create: boolean): Promise<FileSystemFileHandle> {
    const { directoryPath, fileName } = this.splitFilePath(path);
    const directory = await this.directory(directoryPath, create);
    return directory.getFileHandle(fileName, { create });
  }

  async createWriter(path: string): Promise<AttachmentFileWriter> {
    const handle = await this.file(path, true);
    const writable = await handle.createWritable({ keepExistingData: false });
    return {
      write: async (chunk) => {
        await writable.write(Uint8Array.from(chunk));
      },
      flush: async () => {
        await writable.close();
      },
    };
  }

  async read(path: string): Promise<Uint8Array> {
    const handle = await this.file(path, false);
    return new Uint8Array(await (await handle.getFile()).arrayBuffer());
  }

  async exists(path: string): Promise<boolean> {
    try {
      await this.file(path, false);
      return true;
    } catch (error) {
      if (error instanceof DOMException && error.name === "NotFoundError") return false;
      throw error;
    }
  }

  async promote(temporaryPath: string, finalPath: string): Promise<void> {
    const bytes = await this.read(temporaryPath);
    const writer = await this.createWriter(finalPath);
    await writer.write(bytes);
    await writer.flush();
  }

  async list(directoryPath: string): Promise<string[]> {
    try {
      const directory = (await this.directory(directoryPath, false)) as DirectoryEntriesHandle;
      const paths: string[] = [];
      for await (const [name, handle] of directory.entries()) {
        if (handle.kind === "file") paths.push(`${directoryPath}/${name}`);
      }
      return paths.sort();
    } catch (error) {
      if (error instanceof DOMException && error.name === "NotFoundError") return [];
      throw error;
    }
  }

  async remove(path: string): Promise<void> {
    const { directoryPath, fileName } = this.splitFilePath(path);
    const directory = await this.directory(directoryPath, false);
    await directory.removeEntry(fileName);
  }
}

export function createBrowserInspectionAttachmentStore(
  repository: IndexedDbFieldRepository,
): InspectionAttachmentStore {
  return new InspectionAttachmentStore({
    repository,
    fileSystem: new BrowserInspectionAttachmentFileSystem(),
    hasher: new DedicatedInspectionAttachmentHasher(),
  });
}
