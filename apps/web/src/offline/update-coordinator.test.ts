import { describe, expect, it, vi } from "vitest";

import {
  APP_SHELL_UPDATE_POLL_INTERVAL_MS,
  UPDATE_ACTIVATION_POLICY,
  UpdateCoordinator,
  evaluateUnresponsiveClientFence,
  evaluateUpdateSafety,
  installAppShellUpdateMonitor,
  isExactOfflineVersion,
  type AppShellUpdateMonitorEnvironment,
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

  it("allows only a safe checkpoint for an exact-vector worker", () => {
    expect(evaluateUpdateSafety(input())).toMatchObject({
      code: "ready-for-safe-checkpoint",
      allowEdits: true,
      autoActivate: false,
      allowDocumentReload: false,
      preserveLocalData: true,
      deleteOldCaches: false,
      databaseDowngradeAllowed: false,
    });
    expect(UPDATE_ACTIVATION_POLICY).toEqual({
      automaticSkipWaiting: true,
      automaticClientsClaim: true,
      deleteOldCachesOnActivate: false,
    });
  });

  it("waits for zero quiescence counters before activation", () => {
    const result = evaluateUpdateSafety(
      input({
        quiescence: {
          dirtyFormCount: 0,
          active: { indexedDb: 1, opfs: 0, hashWorker: 0, sync: 0, mutation: 0 },
        },
      }),
    );
    expect(result).toMatchObject({
      code: "waiting-for-safe-checkpoint",
      allowEdits: true,
      autoActivate: false,
      allowDocumentReload: false,
      deleteOldCaches: false,
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
    expect(result.code).toBe("waiting-for-safe-checkpoint");
    expect(result.autoActivate).toBe(false);
    expect(result.allowDocumentReload).toBe(false);
    expect(result.preserveLocalData).toBe(true);
    expect(result.deleteOldCaches).toBe(false);
  });

  it("requires one migration owner before a safe checkpoint", () => {
    expect(evaluateUpdateSafety(input({ migration: { required: true, ownerLockAcquired: false, phase: "before-expand", failed: false } }))).toMatchObject({
      code: "waiting-for-safe-checkpoint",
      autoActivate: false,
      allowDocumentReload: false,
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
    ).toMatchObject({ code: "ready-for-safe-checkpoint", allowDocumentReload: false });
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
      expect.objectContaining({ type: "update-decision", code: "ready-for-safe-checkpoint" }),
    );
  });

  it("keeps an unresponsive client fenced until safe acknowledgement or exit", () => {
    expect(evaluateUnresponsiveClientFence({
      ackTimedOut: true,
      clientExited: false,
      securityCritical: false,
      oldVectorDeadlineReached: false,
      serverMinimumWriteVectorCommitted: false,
      responsiveClientsFrozenAndAcked: false,
      resumedClientSafeAck: false,
    })).toMatchObject({
      state: "ACK_TIMEOUT",
      activationMayProceed: false,
      mutationsAllowed: true,
    });

    expect(evaluateUnresponsiveClientFence({
      ackTimedOut: true,
      clientExited: false,
      securityCritical: true,
      oldVectorDeadlineReached: true,
      serverMinimumWriteVectorCommitted: true,
      responsiveClientsFrozenAndAcked: true,
      resumedClientSafeAck: false,
    })).toMatchObject({
      state: "UNRESPONSIVE_CLIENT_FENCED_READ_ONLY_PENDING_RESUME",
      activationMayProceed: false,
      mutationsAllowed: false,
    });

    expect(evaluateUnresponsiveClientFence({
      ackTimedOut: true,
      clientExited: true,
      securityCritical: true,
      oldVectorDeadlineReached: true,
      serverMinimumWriteVectorCommitted: true,
      responsiveClientsFrozenAndAcked: true,
      resumedClientSafeAck: false,
    })).toMatchObject({ state: "CLIENT_EXITED", activationMayProceed: true });
  });
});

function updateMonitorHarness() {
  const eventTarget = new EventTarget();
  const documentTarget = new EventTarget() as EventTarget & { visibilityState: string };
  let visibilityState = "visible";
  let online = true;
  let intervalCallback: (() => void) | null = null;
  const intervalHandle = { id: "app-shell-update-poll" };
  const clearInterval = vi.fn();
  const reportFailure = vi.fn();
  Object.defineProperty(documentTarget, "visibilityState", {
    get: () => visibilityState,
  });
  const environment: AppShellUpdateMonitorEnvironment = {
    eventTarget,
    documentTarget,
    isOnline: () => online,
    setInterval(callback, intervalMs) {
      expect(intervalMs).toBe(APP_SHELL_UPDATE_POLL_INTERVAL_MS);
      intervalCallback = callback;
      return intervalHandle;
    },
    clearInterval,
    reportFailure,
  };
  return {
    environment,
    eventTarget,
    documentTarget,
    intervalHandle,
    clearInterval,
    reportFailure,
    poll: () => intervalCallback?.(),
    setOnline: (value: boolean) => {
      online = value;
    },
    setVisibility: (value: string) => {
      visibilityState = value;
    },
  };
}

async function settleUpdateMonitor(): Promise<void> {
  await Promise.resolve();
  await Promise.resolve();
}

describe("app-shell update monitor", () => {
  it("checks on startup, polling, foreground, online recovery, and page restoration", async () => {
    const harness = updateMonitorHarness();
    const update = vi.fn().mockResolvedValue(undefined);
    const monitor = installAppShellUpdateMonitor(
      { update } as unknown as ServiceWorkerRegistration,
      harness.environment,
    );

    await monitor.checkNow();
    expect(update).toHaveBeenCalledTimes(1);

    harness.poll();
    await settleUpdateMonitor();
    expect(update).toHaveBeenCalledTimes(2);

    harness.documentTarget.dispatchEvent(new Event("visibilitychange"));
    await settleUpdateMonitor();
    expect(update).toHaveBeenCalledTimes(3);

    harness.eventTarget.dispatchEvent(new Event("online"));
    await settleUpdateMonitor();
    expect(update).toHaveBeenCalledTimes(4);

    harness.eventTarget.dispatchEvent(new Event("pageshow"));
    await settleUpdateMonitor();
    expect(update).toHaveBeenCalledTimes(5);
    harness.eventTarget.dispatchEvent(new Event("focus"));
    await settleUpdateMonitor();
    expect(update).toHaveBeenCalledTimes(6);

    monitor.close();
    expect(harness.clearInterval).toHaveBeenCalledWith(harness.intervalHandle);
    harness.poll();
    harness.eventTarget.dispatchEvent(new Event("online"));
    harness.eventTarget.dispatchEvent(new Event("pageshow"));
    harness.eventTarget.dispatchEvent(new Event("focus"));
    harness.documentTarget.dispatchEvent(new Event("visibilitychange"));
    await settleUpdateMonitor();
    expect(update).toHaveBeenCalledTimes(6);
  });

  it("pauses while hidden or offline and coalesces concurrent checks", async () => {
    const harness = updateMonitorHarness();
    harness.setOnline(false);
    const finishUpdates: Array<() => void> = [];
    const update = vi.fn(() => new Promise<void>((resolve) => {
      finishUpdates.push(resolve);
    }));
    const monitor = installAppShellUpdateMonitor(
      { update } as unknown as ServiceWorkerRegistration,
      harness.environment,
    );

    expect(update).not.toHaveBeenCalled();
    harness.poll();
    harness.eventTarget.dispatchEvent(new Event("online"));
    expect(update).not.toHaveBeenCalled();

    harness.setOnline(true);
    harness.setVisibility("hidden");
    harness.eventTarget.dispatchEvent(new Event("online"));
    expect(update).not.toHaveBeenCalled();

    harness.setVisibility("visible");
    harness.documentTarget.dispatchEvent(new Event("visibilitychange"));
    harness.poll();
    harness.eventTarget.dispatchEvent(new Event("pageshow"));
    harness.eventTarget.dispatchEvent(new Event("focus"));
    expect(update).toHaveBeenCalledTimes(1);

    finishUpdates.shift()?.();
    await monitor.checkNow();
    await settleUpdateMonitor();
    harness.poll();
    expect(update).toHaveBeenCalledTimes(2);
    finishUpdates.shift()?.();
    monitor.close();
  });

  it("reports a failed check and retries on the next trigger", async () => {
    const harness = updateMonitorHarness();
    const failure = new Error("temporary update failure");
    const update = vi.fn()
      .mockRejectedValueOnce(failure)
      .mockResolvedValue(undefined);
    const monitor = installAppShellUpdateMonitor(
      { update } as unknown as ServiceWorkerRegistration,
      harness.environment,
    );

    await monitor.checkNow();
    expect(harness.reportFailure).toHaveBeenCalledWith(failure);
    await settleUpdateMonitor();

    harness.eventTarget.dispatchEvent(new Event("online"));
    await settleUpdateMonitor();
    expect(update).toHaveBeenCalledTimes(2);
    monitor.close();
  });
});
