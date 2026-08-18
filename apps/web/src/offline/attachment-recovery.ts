import type { AttachmentManifestRow } from "./db";
import type { IndexedDbFieldRepository } from "./field-repository";
import type { InspectionAttachmentHasher } from "./inspection-attachment-hash-worker";
import {
  inspectionAttachmentDirectory,
  type InspectionAttachmentFileSystem,
} from "./opfs-inspection-attachment-store";

export interface AttachmentRecoveryBlockingItem {
  attachmentId: string;
  code: "REFERENCED_BYTES_MISSING";
  message: string;
}

export interface AttachmentRecoveryReport {
  blocking: AttachmentRecoveryBlockingItem[];
  recoveredAttachmentIds: string[];
  quarantinedAttachmentIds: string[];
  quarantinedUnknownPaths: string[];
  deletedPaths: string[];
}

interface ReconcileInspectionAttachmentsInput {
  repository: IndexedDbFieldRepository;
  fileSystem: InspectionAttachmentFileSystem;
  hasher: InspectionAttachmentHasher;
}

interface PathObservation {
  path: string;
  byteSize: number;
  sha256: string;
  exact: boolean;
}

function isFinalState(manifest: AttachmentManifestRow): boolean {
  return (
    manifest.stagingState === "ready" ||
    manifest.stagingState === "uploading" ||
    manifest.stagingState === "acknowledged" ||
    manifest.stagingState === "purge_eligible"
  );
}

async function observePath(
  path: string,
  fileSystem: InspectionAttachmentFileSystem,
  hasher: InspectionAttachmentHasher,
): Promise<{ path: string; byteSize: number; sha256: string; exact: boolean } | null> {
  if (!(await fileSystem.exists(path))) return null;
  const bytes = await fileSystem.read(path);
  return {
    path,
    byteSize: bytes.byteLength,
    sha256: await hasher.sha256(bytes),
    exact: false,
  };
}

function isExact(
  observation: PathObservation | null,
  declaredByteSize: number,
  expectedSha256: string,
): boolean {
  return observation !== null &&
    observation.byteSize === declaredByteSize &&
    observation.sha256 === expectedSha256;
}

export async function reconcileInspectionAttachments(
  input: ReconcileInspectionAttachmentsInput,
): Promise<AttachmentRecoveryReport> {
  const report: AttachmentRecoveryReport = {
    blocking: [],
    recoveredAttachmentIds: [],
    quarantinedAttachmentIds: [],
    quarantinedUnknownPaths: [],
    deletedPaths: [],
  };
  const manifests = await input.repository.listAttachmentManifests();
  const referencedPaths = new Set<string>();
  for (const manifest of manifests) {
    if (manifest.temporaryOpfsPath) referencedPaths.add(manifest.temporaryOpfsPath);
    if (manifest.finalOpfsPath) referencedPaths.add(manifest.finalOpfsPath);
  }

  for (const manifest of manifests) {
    if (manifest.packageId === "__unknown__" || manifest.stagingState === "quarantined") continue;
    const final = manifest.finalOpfsPath
      ? await observePath(manifest.finalOpfsPath, input.fileSystem, input.hasher)
      : null;
    const temporary = manifest.temporaryOpfsPath
      ? await observePath(manifest.temporaryOpfsPath, input.fileSystem, input.hasher)
      : null;
    if (!final && !temporary) {
      await input.repository.markAttachmentRecovery(manifest.attachmentId, {
        state: "recovery_required",
        reason: "REFERENCED_BYTES_MISSING",
        localBytesPresent: false,
      });
      report.blocking.push({
        attachmentId: manifest.attachmentId,
        code: "REFERENCED_BYTES_MISSING",
        message: "Referenced Inspection Attachment bytes are missing; recovery is required.",
      });
      continue;
    }
    const expectedSha256 = manifest.expectedSha256 ?? manifest.sha256;
    if (!expectedSha256) {
      await input.repository.markAttachmentRecovery(manifest.attachmentId, {
        state: "recovery_required",
        reason: "ATTACHMENT_SOURCE_DIGEST_MISSING",
        localBytesPresent: Boolean(final || temporary),
      });
      continue;
    }

    if (final && temporary) {
      if (isExact(final, manifest.declaredByteSize, expectedSha256) && isExact(temporary, manifest.declaredByteSize, expectedSha256)) {
        if (!isFinalState(manifest)) {
          await input.repository.commitReadyAttachment({
            attachmentId: manifest.attachmentId,
            observedByteSize: final.byteSize,
            sha256: final.sha256,
          });
          report.recoveredAttachmentIds.push(manifest.attachmentId);
        }
        continue;
      }
      const reason = isExact(final, manifest.declaredByteSize, expectedSha256)
        ? "TEMP_HASH_MISMATCH"
        : isExact(temporary, manifest.declaredByteSize, expectedSha256)
          ? "FINAL_HASH_MISMATCH"
          : "ATTACHMENT_HASH_MISMATCH";
      await input.repository.markAttachmentRecovery(manifest.attachmentId, {
        state: "quarantined",
        reason,
        localBytesPresent: true,
      });
      report.quarantinedAttachmentIds.push(manifest.attachmentId);
      continue;
    }

    if (final) {
      if (!isExact(final, manifest.declaredByteSize, expectedSha256)) {
        await input.repository.markAttachmentRecovery(manifest.attachmentId, {
          state: "quarantined",
          reason: final.byteSize !== manifest.declaredByteSize ? "ATTACHMENT_SIZE_MISMATCH" : "ATTACHMENT_HASH_MISMATCH",
          localBytesPresent: true,
        });
        report.quarantinedAttachmentIds.push(manifest.attachmentId);
        continue;
      }
      if (!isFinalState(manifest)) {
        await input.repository.commitReadyAttachment({
          attachmentId: manifest.attachmentId,
          observedByteSize: final.byteSize,
          sha256: final.sha256,
        });
        report.recoveredAttachmentIds.push(manifest.attachmentId);
      }
      continue;
    }

    if (!temporary) continue;
    if (isExact(temporary, manifest.declaredByteSize, expectedSha256)) {
      if (!manifest.temporaryOpfsPath || !manifest.finalOpfsPath) {
        await input.repository.markAttachmentRecovery(manifest.attachmentId, {
          state: "quarantined",
          reason: "ATTACHMENT_PATH_MISSING",
          localBytesPresent: true,
        });
        report.quarantinedAttachmentIds.push(manifest.attachmentId);
        continue;
      }
      await input.repository.markAttachmentRecoveryPhase(manifest.attachmentId, "HASH_VERIFIED");
      await input.fileSystem.promote(manifest.temporaryOpfsPath, manifest.finalOpfsPath);
      const promotedFinal = await observePath(manifest.finalOpfsPath, input.fileSystem, input.hasher);
      if (!isExact(promotedFinal, manifest.declaredByteSize, expectedSha256)) {
        await input.repository.markAttachmentRecovery(manifest.attachmentId, {
          state: "quarantined",
          reason: "ATTACHMENT_FINAL_READBACK_MISMATCH",
          localBytesPresent: true,
        });
        report.quarantinedAttachmentIds.push(manifest.attachmentId);
        continue;
      }
      await input.repository.markAttachmentRecoveryPhase(manifest.attachmentId, "PROMOTED");
      await input.repository.commitReadyAttachment({
        attachmentId: manifest.attachmentId,
        observedByteSize: temporary.byteSize,
        sha256: temporary.sha256,
      });
      report.recoveredAttachmentIds.push(manifest.attachmentId);
      continue;
    }

    if (temporary.byteSize < manifest.declaredByteSize) {
      await input.repository.markAttachmentRecovery(manifest.attachmentId, {
        state: "recovery_required",
        reason: "ATTACHMENT_PARTIAL_TEMP",
        localBytesPresent: true,
      });
      continue;
    }
    await input.repository.markAttachmentRecovery(manifest.attachmentId, {
      state: "quarantined",
      reason: temporary.byteSize !== manifest.declaredByteSize ? "ATTACHMENT_SIZE_MISMATCH" : "ATTACHMENT_HASH_MISMATCH",
      localBytesPresent: true,
    });
    report.quarantinedAttachmentIds.push(manifest.attachmentId);
  }

  const directory = inspectionAttachmentDirectory(input.repository.subjectId);
  const paths = await input.fileSystem.list(directory);
  for (const path of paths) {
    if (referencedPaths.has(path)) continue;
    const bytes = await input.fileSystem.read(path);
    const sha256 = await input.hasher.sha256(bytes);
    await input.repository.quarantineUnknownAttachmentPath({
      path,
      byteSize: bytes.byteLength,
      sha256,
    });
    report.quarantinedUnknownPaths.push(path);
  }
  report.blocking.sort((left, right) => left.attachmentId.localeCompare(right.attachmentId));
  report.recoveredAttachmentIds.sort();
  report.quarantinedAttachmentIds.sort();
  report.quarantinedUnknownPaths.sort();
  return report;
}
