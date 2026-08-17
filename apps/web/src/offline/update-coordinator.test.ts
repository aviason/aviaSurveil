import { describe, expect, it, vi } from "vitest";

import {
  UPDATE_ACTIVATION_POLICY,
  UpdateCoordinator,
  evaluateUpdateSafety,
  isExactOfflineVersion,
  type UpdateSafetyInput,
} from "./update-coordinator";

function input(overrides: Partial<UpdateSafetyInput> = {}): UpdateSafetyInput {
  return {
    active: {
      appShellVersion: 2,
      indexedDbSchemaVersion: 2,
      packageSchemaVersion: 2,
      syncProtocolVersion: 2,
    },
    candidate: {
      appShellVersion: 2,
      indexedDbSchemaVersion: 2,
      packageSchemaVersion: 2,
      syncProtocolVersion: 2,
    },
    clients: [
      {
        clientId: "tab-a",
        appShellVersion: 2,
        indexedDbSchemaVersion: 2,
        packageSchemaVersion: 2,
        syncProtocolVersion: 2,
      },
      {
        clientId: "tab-b",
        appShellVersion: 2,
        indexedDbSchemaVersion: 2,
        packageSchemaVersion: 2,
        syncProtocolVersion: 2,
      },
    ],
    localWork: {
      pendingOutboxCount: 0,
      pendingAttachmentManifestCount: 0,
      unsyncedPackageCount: 0,
    },
    migration: {
      required: false,
      ownerLockAcquired: true,
      phase: "none",
      failed: false,
    },
    ...overrides,
  };
}

describe("update safety", () => {
  it("accepts only the exact complete vector", () => {
    expect(isExactOfflineVersion(3, 3)).toBe(true);
    expect(isExactOfflineVersion(2, 3)).toBe(false);
    expect(isExactOfflineVersion(1, 3)).toBe(false);
    expect(isExactOfflineVersion(4, 3)).toBe(false);
    expect(isExactOfflineVersion(0, 1)).toBe(false);
    expect(isExactOfflineVersion(-1, 1)).toBe(false);
  });

  it("allows automatic exact-vector worker activation", () => {
    expect(evaluateUpdateSafety(input())).toMatchObject({
      code: "ready-for-automatic-activation",
      allowEdits: true,
      autoActivate: true,
      allowDocumentReload: true,
      preserveLocalData: true,
      deleteOldCaches: true,
      databaseDowngradeAllowed: false,
    });
    expect(UPDATE_ACTIVATION_POLICY).toEqual({
      automaticSkipWaiting: true,
      automaticClientsClaim: true,
      deleteOldCachesOnActivate: true,
    });
  });

  it("defers an update while any client reports a different vector", () => {
    const result = evaluateUpdateSafety(
      input({
        clients: [
          {
            clientId: "old-tab",
            appShellVersion: 1,
            indexedDbSchemaVersion: 2,
            packageSchemaVersion: 2,
            syncProtocolVersion: 2,
          },
        ],
      }),
    );
    expect(result.code).toBe("deferred-incompatible-client");
    expect(result.allowEdits).toBe(true);
  });

  it.each([
    { pendingOutboxCount: 1, pendingAttachmentManifestCount: 0, unsyncedPackageCount: 0 },
    { pendingOutboxCount: 0, pendingAttachmentManifestCount: 1, unsyncedPackageCount: 0 },
    { pendingOutboxCount: 0, pendingAttachmentManifestCount: 0, unsyncedPackageCount: 1 },
  ])("activates without reloading over pending local work: %j", (localWork) => {
    const result = evaluateUpdateSafety(input({ localWork }));
    expect(result.code).toBe("ready-for-automatic-activation");
    expect(result.autoActivate).toBe(true);
    expect(result.allowDocumentReload).toBe(true);
    expect(result.preserveLocalData).toBe(true);
    expect(result.deleteOldCaches).toBe(true);
  });

  it("requires one migration owner and pauses edits during an incompatible migration", () => {
    expect(evaluateUpdateSafety(input({ migration: { required: true, ownerLockAcquired: false, phase: "before-expand", failed: false } }))).toMatchObject({
      code: "ready-for-automatic-activation",
      autoActivate: true,
      allowDocumentReload: true,
    });

    expect(
      evaluateUpdateSafety(
        input({
          migration: {
            required: true,
            ownerLockAcquired: true,
            phase: "after-expand",
            failed: false,
          },
        }),
      ),
    ).toMatchObject({ code: "ready-for-automatic-activation", allowDocumentReload: true });
  });

  it.each(["before-expand", "after-expand", "after-copy", "before-contract"] as const)(
    "opens read-only recovery after termination at %s",
    (phase) => {
      expect(
        evaluateUpdateSafety(
          input({ migration: { required: true, ownerLockAcquired: true, phase, failed: true } }),
        ),
      ).toMatchObject({
        code: "read-only-recovery",
        allowEdits: false,
        preserveLocalData: true,
        deleteOldCaches: false,
        databaseDowngradeAllowed: false,
      });
    },
  );

  it("blocks a vector-changing shell rollback", () => {
    const result = evaluateUpdateSafety(
      input({
        active: {
          appShellVersion: 3,
          indexedDbSchemaVersion: 3,
          packageSchemaVersion: 3,
          syncProtocolVersion: 3,
        },
        candidate: {
          appShellVersion: 2,
          indexedDbSchemaVersion: 3,
          packageSchemaVersion: 3,
          syncProtocolVersion: 3,
        },
        clients: [],
      }),
    );
    expect(result.code).toBe("blocked-vector-change");
    expect(result.databaseDowngradeAllowed).toBe(false);
  });

  it("serializes update evaluation through the approved owner lock and broadcasts the decision", async () => {
    let active = 0;
    let maximumActive = 0;
    const lock = {
      request: async <T,>(_name: string, callback: () => Promise<T>) => {
        while (active > 0) await new Promise((resolve) => setTimeout(resolve, 1));
        active += 1;
        maximumActive = Math.max(maximumActive, active);
        try {
          return await callback();
        } finally {
          active -= 1;
        }
      },
    };
    const broadcast = vi.fn();
    const coordinator = new UpdateCoordinator(lock, broadcast);

    await Promise.all([coordinator.evaluate(input()), coordinator.evaluate(input())]);

    expect(maximumActive).toBe(1);
    expect(broadcast).toHaveBeenCalledTimes(2);
    expect(broadcast).toHaveBeenLastCalledWith(
      expect.objectContaining({ type: "update-decision", code: "ready-for-automatic-activation" }),
    );
  });
});
