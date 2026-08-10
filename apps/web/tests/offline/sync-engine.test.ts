import { describe, expect, it, vi } from "vitest";

import type {
  Backend,
  FieldSyncOperation,
  PushFieldOperationResult,
  SyncPullResponse,
} from "../../src/backend/backend";
import type { AttachmentManifestRow, OutboxRow, SyncStateRow } from "../../src/offline/db";
import {
  createBrowserSyncBroadcast,
  ForegroundSyncEngine,
  installForegroundSyncTriggers,
  type FieldSyncRepository,
  type SyncOwnerLock,
} from "../../src/offline/sync-engine";

const now = new Date("2026-07-21T12:00:00.000Z");
const packageId = "PKG-CAB-2026-001";
const grantId = "GRANT-CAB-2026-001";

function operation(
  operationId: string,
  commandType: FieldSyncOperation["commandType"] = "UPSERT_CHECKLIST_RESPONSE",
): FieldSyncOperation {
  const base = {
    operationId,
    protocolVersion: 1,
    offlineGrantId: grantId,
    packageId,
    packageVersion: 1,
    entityId: commandType === "SUBMIT_CHECKLIST" ? "AUD-2026-001" : "RESP-CAB-001",
    commandType,
    baseRevision: 1,
    deviceInstanceId: "DEVICE-CAB-001",
    clientOccurredAt: now.toISOString(),
  };
  if (commandType === "UPSERT_CHECKLIST_RESPONSE") {
    return {
      ...base,
      commandType,
      payload: {
        auditId: "AUD-2026-001",
        questionId: "CAB-EMEQ-PBE-001",
        answer: "NON_COMPLIANT",
        comment: "PBE record unavailable.",
      },
    };
  }
  if (commandType === "CREATE_POTENTIAL_FINDING") {
    return {
      ...base,
      entityId: "PF-LOCAL-001",
      commandType,
      baseRevision: null,
      payload: {
        auditId: "AUD-2026-001",
        questionId: "CAB-EMEQ-PBE-001",
        checklistResponseId: "RESP-CAB-001",
        expectedChecklistResponseRevision: 2,
        title: "PBE gap",
        description: "PBE status was not confirmed.",
        requiredComment: "Provide the serviceability record.",
        inspectionAttachmentIds: [],
      },
    };
  }
  if (commandType === "SUBMIT_CHECKLIST") {
    return { ...base, commandType, payload: { auditId: "AUD-2026-001" } };
  }
  return {
    ...base,
    entityId: "ATT-LOCAL-001",
    commandType,
    baseRevision: null,
    payload: {
      auditId: "AUD-2026-001",
      checklistResponseId: "RESP-CAB-001",
      potentialFindingOperationId: "OP-PF-001",
      fileName: "pbe.pdf",
      mediaType: "application/pdf",
      byteSize: 4,
      sha256: `sha256:${"a".repeat(64)}`,
    },
  };
}

function outboxRow(input: FieldSyncOperation, state: OutboxRow["state"] = "PENDING"): OutboxRow {
  return {
    operationId: input.operationId,
    subjectId: "USR-INSPECTOR-AMINA",
    packageId,
    commandType: input.commandType,
    entityId: input.entityId,
    baseRevision: input.baseRevision,
    state,
    createdAt: input.clientOccurredAt,
    attemptCount: 0,
    nextAttemptAt: input.clientOccurredAt,
    dependsOnOperationIds: [],
    supersededByOperationId: null,
    requestDigest: "sha256:test",
    lastErrorCode: null,
    operation: input,
  };
}

function accepted(
  input: FieldSyncOperation,
  authoritativeRevision = 2,
  authoritativeEntityId = input.entityId,
): PushFieldOperationResult {
  return {
    operationId: input.operationId,
    status: "accepted",
    authoritativeEntityId,
    authoritativeRevision,
    errorCode: null,
    conflict: null,
    acknowledgedAt: now.toISOString(),
  };
}

class FakeRepository implements FieldSyncRepository {
  rows = new Map<string, OutboxRow>();
  applied: PushFieldOperationResult[] = [];
  pullPages: SyncPullResponse[] = [];
  recovered = 0;
  released: boolean[] = [];
  manifests: AttachmentManifestRow[] = [];
  uploadErrors: string[] = [];
  uploaded: string[] = [];
  syncState: SyncStateRow = {
    subjectId: "USR-INSPECTOR-AMINA",
    packageId,
    grantId,
    projectionVersion: 1,
    cursor: null,
    lastSuccessAt: null,
    lastErrorCode: null,
  };

  constructor(rows: OutboxRow[] = []) {
    for (const row of rows) this.rows.set(row.operationId, structuredClone(row));
  }

  async recoverInterruptedOperations(): Promise<void> {
    this.recovered += 1;
  }

  async releaseRetryableOperations(_packageId: string, force: boolean): Promise<void> {
    this.released.push(force);
    for (const row of this.rows.values()) {
      if (row.state === "FAILED_RETRYABLE" && force) row.state = "PENDING";
    }
  }

  async listDeliverableOperations(): Promise<OutboxRow[]> {
    return [...this.rows.values()].filter((row) => row.state === "PENDING");
  }

  async markOperationInFlight(operationId: string): Promise<OutboxRow> {
    const row = this.rows.get(operationId)!;
    row.state = "IN_FLIGHT";
    row.attemptCount += 1;
    return structuredClone(row);
  }

  async applyPushResult(
    operationId: string,
    result: PushFieldOperationResult,
  ): Promise<void> {
    this.applied.push(structuredClone(result));
    const row = this.rows.get(operationId)!;
    row.lastErrorCode = result.errorCode;
    row.state = result.status === "accepted" || result.status === "already_applied"
      ? "ACKNOWLEDGED"
      : result.status === "retryable"
        ? "FAILED_RETRYABLE"
        : result.status === "conflict"
          ? "CONFLICT"
          : "REJECTED";
    for (const dependent of this.rows.values()) {
      if (!dependent.dependsOnOperationIds.includes(operationId) || row.state !== "ACKNOWLEDGED") continue;
      dependent.dependsOnOperationIds = dependent.dependsOnOperationIds.filter(
        (candidate) => candidate !== operationId,
      );
      if (dependent.dependsOnOperationIds.length === 0) dependent.state = "PENDING";
    }
  }

  async getSyncState(): Promise<SyncStateRow | null> {
    return structuredClone(this.syncState);
  }

  async applyPullPage(input: { page: SyncPullResponse }): Promise<void> {
    this.pullPages.push(structuredClone(input.page));
    this.syncState.cursor = input.page.nextCursor;
    this.syncState.projectionVersion = input.page.projectionVersion;
    this.syncState.lastErrorCode = input.page.resnapshotRequired ? "RESNAPSHOT_REQUIRED" : null;
  }

  async listRegisteredAttachmentsReadyForUpload(): Promise<AttachmentManifestRow[]> {
    return structuredClone(this.manifests.filter((manifest) => manifest.stagingState === "ready"));
  }

  async beginRegisteredAttachmentUpload(attachmentId: string): Promise<AttachmentManifestRow> {
    const manifest = this.manifests.find((candidate) => candidate.attachmentId === attachmentId)!;
    manifest.stagingState = "uploading";
    return structuredClone(manifest);
  }

  async markAttachmentUploadRetryable(attachmentId: string, errorCode: string): Promise<void> {
    const manifest = this.manifests.find((candidate) => candidate.attachmentId === attachmentId)!;
    manifest.stagingState = "ready";
    this.uploadErrors.push(errorCode);
  }

  async acknowledgeUploadedAttachment(input: {
    attachmentId: string;
    authoritativeEntityId: string;
    acknowledgedAt: string;
  }): Promise<void> {
    const manifest = this.manifests.find((candidate) => candidate.attachmentId === input.attachmentId)!;
    manifest.stagingState = "acknowledged";
    manifest.syncState = "ACKNOWLEDGED";
    this.uploaded.push(input.attachmentId);
  }
}

function backend(input: {
  push?: (operation: FieldSyncOperation) => Promise<PushFieldOperationResult>;
  pull?: () => Promise<SyncPullResponse>;
  beginUpload?: Backend["inspectionAttachments"]["beginUpload"];
  completeUpload?: Backend["inspectionAttachments"]["completeUpload"];
} = {}): Backend {
  return {
    mode: "http",
    assignments: {} as Backend["assignments"],
    inspections: {} as Backend["inspections"],
    potentialFindings: {} as Backend["potentialFindings"],
    findings: {} as Backend["findings"],
    caps: {} as Backend["caps"],
    evidence: {} as Backend["evidence"],
    reports: {} as Backend["reports"],
    dashboards: {} as Backend["dashboards"],
    organizations: {} as Backend["organizations"],
    planning: {} as Backend["planning"],
    configuration: {} as Backend["configuration"],
    auditTrail: {} as Backend["auditTrail"],
    communications: {} as Backend["communications"],
    calendar: {} as Backend["calendar"],
    profiles: {} as Backend["profiles"],
    teams: {} as Backend["teams"],
    risk: {} as Backend["risk"],
    documents: {} as Backend["documents"],
    notifications: {} as Backend["notifications"],
    administration: {} as Backend["administration"],
    adminWorkspace: {} as Backend["adminWorkspace"],
    governedChecklistReview: {} as Backend["governedChecklistReview"],
    governedChecklistIntake: {} as Backend["governedChecklistIntake"],
    assistantDrafts: {} as Backend["assistantDrafts"],
    planningIntake: {} as Backend["planningIntake"],
    auditeeCoordination: {} as Backend["auditeeCoordination"],
    auditeeReports: {} as Backend["auditeeReports"],
    inspectionAttachments: {
      beginUpload: input.beginUpload ?? (async (request) => ({
        uploadId: `UPLOAD-${request.inspectionAttachmentId}`,
        stagingObjectKey: "private/staging/key",
        uploadUrl: "https://upload.invalid/object",
        requiredHeaders: { "Content-Type": request.declaredMediaType },
        expiresAt: new Date(now.getTime() + 60_000).toISOString(),
        maximumByteSize: 25 * 1024 * 1024,
      })),
      completeUpload: input.completeUpload ?? (async (request) => ({
        inspectionAttachmentId: request.uploadId.replace("UPLOAD-", ""),
        uploadState: "UPLOADED",
        scanState: "PENDING",
      })),
    },
    sync: {
      pushOperation: async ({ operation: value }) =>
        input.push?.(value) ?? accepted(value),
      pull: async () =>
        input.pull?.() ?? {
          changes: [], nextCursor: "sync_cursor_1", hasMore: false,
          resnapshotRequired: false, projectionVersion: 1,
        },
    },
  };
}

const acquiredLock: SyncOwnerLock = {
  request: async (_name, callback) => ({ acquired: true, value: await callback() }),
};

describe("ForegroundSyncEngine", () => {
  it("delivers one frozen operation at a time and causally unblocks its dependent", async () => {
    const response = outboxRow(operation("OP-RESPONSE-001"));
    const potential = outboxRow(operation("OP-PF-001", "CREATE_POTENTIAL_FINDING"), "BLOCKED_ON_DEPENDENCY");
    potential.dependsOnOperationIds = [response.operationId];
    const repository = new FakeRepository([response, potential]);
    const pushed: string[] = [];
    const activeBackend = backend({
      push: async (value) => {
        pushed.push(value.operationId);
        return accepted(
          value,
          value.commandType === "CREATE_POTENTIAL_FINDING" ? 1 : 2,
          value.commandType === "CREATE_POTENTIAL_FINDING" ? "PF-SERVER-001" : value.entityId,
        );
      },
    });
    const engine = new ForegroundSyncEngine({
      backend: activeBackend,
      repository,
      lock: acquiredLock,
      now: () => now,
    });

    const result = await engine.run({ packageId, offlineGrantId: grantId }, "startup");

    expect(pushed).toEqual(["OP-RESPONSE-001", "OP-PF-001"]);
    expect(repository.applied.map((entry) => entry.authoritativeEntityId)).toEqual([
      "RESP-CAB-001",
      "PF-SERVER-001",
    ]);
    expect(repository.pullPages).toHaveLength(1);
    expect(result).toMatchObject({ acquired: true, pushed: 2, pulled: 0, status: "synchronized" });
  });

  it("preserves a lost-ack operation as retryable and replays it only on an eligible trigger", async () => {
    const row = outboxRow(operation("OP-LOST-ACK"));
    const repository = new FakeRepository([row]);
    let attempt = 0;
    const push = vi.fn(async (value: FieldSyncOperation) => {
      attempt += 1;
      if (attempt === 1) throw new TypeError("network ended after server commit");
      return accepted(value);
    });
    const engine = new ForegroundSyncEngine({
      backend: backend({ push }), repository, lock: acquiredLock, now: () => now,
    });

    const failed = await engine.run({ packageId, offlineGrantId: grantId }, "startup");
    expect(failed.status).toBe("retryable");
    expect(repository.rows.get(row.operationId)?.state).toBe("FAILED_RETRYABLE");

    const retried = await engine.run({ packageId, offlineGrantId: grantId }, "manual");
    expect(retried.status).toBe("synchronized");
    expect(push).toHaveBeenCalledTimes(2);
    expect(repository.rows.get(row.operationId)?.state).toBe("ACKNOWLEDGED");
  });

  it.each(["conflict", "forbidden", "invalid"] as const)(
    "does not automatically retry a terminal %s result",
    async (status) => {
      const row = outboxRow(operation(`OP-${status.toUpperCase()}`));
      const repository = new FakeRepository([row]);
      const push = vi.fn(async (value: FieldSyncOperation): Promise<PushFieldOperationResult> => ({
        operationId: value.operationId,
        status,
        authoritativeEntityId: null,
        authoritativeRevision: null,
        errorCode: status === "conflict" ? null : "VALIDATION_FAILED",
        conflict: status === "conflict"
          ? {
              code: "STALE_REVISION",
              entityId: value.entityId,
              authoritativeRevision: 9,
              authoritativeStatus: "NON_COMPLIANT",
              changedAt: now.toISOString(),
            }
          : null,
        acknowledgedAt: now.toISOString(),
      }));
      const engine = new ForegroundSyncEngine({
        backend: backend({ push }), repository, lock: acquiredLock, now: () => now,
      });

      const first = await engine.run({ packageId, offlineGrantId: grantId }, "manual");
      const second = await engine.run({ packageId, offlineGrantId: grantId }, "manual");

      expect(first.status).toBe(status);
      expect(second.pushed).toBe(0);
      expect(push).toHaveBeenCalledTimes(1);
      expect(repository.rows.get(row.operationId)?.state).toBe(
        status === "conflict" ? "CONFLICT" : "REJECTED",
      );
    },
  );

  it("lets one tab own a package cycle while the second tab observes contention", async () => {
    let held = false;
    const lock: SyncOwnerLock = {
      request: async (_name, callback) => {
        if (held) return { acquired: false };
        held = true;
        try {
          await new Promise((resolve) => setTimeout(resolve, 10));
          return { acquired: true, value: await callback() };
        } finally {
          held = false;
        }
      },
    };
    const repository = new FakeRepository();
    const engine = new ForegroundSyncEngine({ backend: backend(), repository, lock, now: () => now });

    const [owner, observer] = await Promise.all([
      engine.run({ packageId, offlineGrantId: grantId }, "foreground"),
      engine.run({ packageId, offlineGrantId: grantId }, "foreground"),
    ]);

    expect([owner.acquired, observer.acquired].sort()).toEqual([false, true]);
    expect(repository.pullPages).toHaveLength(1);
  });

  it("broadcasts the owning tab's package status to another foreground tab", () => {
    class FakeBroadcastChannel extends EventTarget {
      static instances: FakeBroadcastChannel[] = [];
      readonly name: string;
      private closed = false;

      constructor(name: string) {
        super();
        this.name = name;
        FakeBroadcastChannel.instances.push(this);
      }

      postMessage(message: unknown): void {
        for (const instance of FakeBroadcastChannel.instances) {
          if (instance === this || instance.closed || instance.name !== this.name) continue;
          const event = new Event("message") as MessageEvent;
          Object.defineProperty(event, "data", { value: structuredClone(message) });
          instance.dispatchEvent(event);
        }
      }

      close(): void {
        this.closed = true;
      }
    }
    vi.stubGlobal("BroadcastChannel", FakeBroadcastChannel);
    try {
      const owner = createBrowserSyncBroadcast();
      const observer = createBrowserSyncBroadcast();
      const receive = vi.fn();
      const unsubscribe = observer?.subscribe(receive);
      const report = {
        acquired: true,
        trigger: "foreground" as const,
        status: "synchronized" as const,
        pushed: 2,
        pulled: 1,
        uploaded: 1,
        conflict: null,
        errorCode: null,
      };

      owner?.broadcast({ type: "field-sync-status", packageId, report });

      expect(receive).toHaveBeenCalledWith({ type: "field-sync-status", packageId, report });
      unsubscribe?.();
      owner?.close();
      observer?.close();
    } finally {
      vi.unstubAllGlobals();
    }
  });

  it("retries an expired attachment URL without deleting local bytes", async () => {
    const attachmentOperation = operation("OP-ATTACHMENT-001", "REGISTER_INSPECTION_ATTACHMENT");
    const repository = new FakeRepository([outboxRow(attachmentOperation, "ACKNOWLEDGED")]);
    repository.manifests = [{
      attachmentId: "ATT-LOCAL-001",
      subjectId: "USR-INSPECTOR-AMINA",
      packageId,
      auditId: "AUD-2026-001",
      checklistResponseId: "RESP-CAB-001",
      potentialFindingLocalId: "PF-LOCAL-001",
      fileName: "pbe.pdf",
      declaredMediaType: "application/pdf",
      declaredByteSize: 4,
      observedByteSize: 4,
      expectedSha256: `sha256:${"a".repeat(64)}`,
      sha256: `sha256:${"a".repeat(64)}`,
      temporaryOpfsPath: null,
      finalOpfsPath: "attachments/ATT-LOCAL-001.bin",
      stagingState: "ready",
      syncState: "PENDING",
      plannedOperationId: attachmentOperation.operationId,
      operationId: attachmentOperation.operationId,
      authoritativeEntityId: "ATT-SERVER-001",
      quarantineReason: null,
      localBytesPresent: true,
      createdAt: now.toISOString(),
      updatedAt: now.toISOString(),
      uploadStartedAt: null,
      acknowledgedAt: null,
      purgeEligibleAt: null,
    }];
    let uploadAttempt = 0;
    const uploadBytes = vi.fn(async () => {
      uploadAttempt += 1;
      if (uploadAttempt === 1) throw new Error("expired upload URL");
    });
    const engine = new ForegroundSyncEngine({
      backend: backend(), repository, lock: acquiredLock, now: () => now,
      readAttachmentBytes: async () => new Uint8Array([1, 2, 3, 4]),
      uploadBytes,
      operationId: (prefix) => `${prefix}-${uploadAttempt + 1}`,
    });

    const first = await engine.run({ packageId, offlineGrantId: grantId }, "online");
    const second = await engine.run({ packageId, offlineGrantId: grantId }, "manual");

    expect(first.status).toBe("retryable");
    expect(second.status).toBe("synchronized");
    expect(uploadBytes).toHaveBeenCalledTimes(2);
    expect(repository.uploaded).toEqual(["ATT-LOCAL-001"]);
    expect(repository.manifests[0]?.localBytesPresent).toBe(true);
  });

  it("installs startup, foreground, online, manual and app-open triggers without Background Sync", () => {
    const run = vi.fn(async () => undefined);
    const eventTarget = new EventTarget();
    const documentTarget = new EventTarget() as EventTarget & { visibilityState: string };
    Object.defineProperty(documentTarget, "visibilityState", { value: "visible" });
    const register = vi.fn();
    const serviceWorker = { ready: Promise.resolve({ sync: { register } }) };

    const triggers = installForegroundSyncTriggers({
      eventTarget,
      documentTarget,
      run,
      serviceWorker,
    });
    eventTarget.dispatchEvent(new Event("online"));
    eventTarget.dispatchEvent(new Event("pageshow"));
    documentTarget.dispatchEvent(new Event("visibilitychange"));
    eventTarget.dispatchEvent(new CustomEvent("avia:manual-sync"));
    triggers.close();

    expect(run).toHaveBeenCalledWith("startup");
    expect(run).toHaveBeenCalledWith("online");
    expect(run).toHaveBeenCalledWith("app-open");
    expect(run).toHaveBeenCalledWith("foreground");
    expect(run).toHaveBeenCalledWith("manual");
    expect(register).not.toHaveBeenCalled();
  });
});
