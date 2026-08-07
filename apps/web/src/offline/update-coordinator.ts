import { CURRENT_OFFLINE_VERSIONS, type OfflineVersionVector } from "./storage-readiness";

export const UPDATE_ACTIVATION_POLICY = {
  automaticSkipWaiting: false,
  automaticClientsClaim: false,
  deleteOldCachesOnActivate: false,
} as const;

export type MigrationPhase =
  | "none"
  | "before-expand"
  | "after-expand"
  | "after-copy"
  | "before-contract";

export interface ClientVersion extends OfflineVersionVector {
  clientId: string;
}

export interface UpdateSafetyInput {
  active: OfflineVersionVector;
  candidate: OfflineVersionVector;
  clients: ClientVersion[];
  localWork: {
    pendingOutboxCount: number;
    pendingAttachmentManifestCount: number;
    unsyncedPackageCount: number;
  };
  migration: {
    required: boolean;
    ownerLockAcquired: boolean;
    phase: MigrationPhase;
    failed: boolean;
  };
}

export type UpdateDecisionCode =
  | "ready-for-user-activation"
  | "deferred-incompatible-client"
  | "deferred-unsynced-work"
  | "deferred-migration-owner"
  | "paused-for-migration"
  | "read-only-recovery"
  | "rollback-shell-only";

export interface UpdateDecision {
  code: UpdateDecisionCode;
  allowEdits: boolean;
  autoActivate: false;
  preserveLocalData: true;
  deleteOldCaches: false;
  databaseDowngradeAllowed: false;
  reason: string;
}

function decision(
  code: UpdateDecisionCode,
  allowEdits: boolean,
  reason: string,
): UpdateDecision {
  return {
    code,
    allowEdits,
    autoActivate: false,
    preserveLocalData: true,
    deleteOldCaches: false,
    databaseDowngradeAllowed: false,
    reason,
  };
}

export function isNOrNMinusOneCompatible(version: number, current: number): boolean {
  return (
    Number.isSafeInteger(version) &&
    Number.isSafeInteger(current) &&
    version > 0 &&
    current > 0 &&
    (version === current || version === current - 1)
  );
}

function hasPendingLocalWork(input: UpdateSafetyInput): boolean {
  return (
    input.localWork.pendingOutboxCount > 0 ||
    input.localWork.pendingAttachmentManifestCount > 0 ||
    input.localWork.unsyncedPackageCount > 0
  );
}

function clientIsCompatible(client: ClientVersion, candidate: OfflineVersionVector): boolean {
  return (
    isNOrNMinusOneCompatible(client.appShellVersion, candidate.appShellVersion) &&
    isNOrNMinusOneCompatible(
      client.indexedDbSchemaVersion,
      candidate.indexedDbSchemaVersion,
    ) &&
    isNOrNMinusOneCompatible(client.packageSchemaVersion, candidate.packageSchemaVersion) &&
    isNOrNMinusOneCompatible(client.syncProtocolVersion, candidate.syncProtocolVersion)
  );
}

export function evaluateUpdateSafety(input: UpdateSafetyInput): UpdateDecision {
  if (input.migration.failed) {
    return decision(
      "read-only-recovery",
      false,
      `Migration failed at ${input.migration.phase}; preserve data and open read-only recovery.`,
    );
  }

  const databaseDowngradeRequested =
    input.candidate.indexedDbSchemaVersion < input.active.indexedDbSchemaVersion ||
    input.candidate.packageSchemaVersion < input.active.packageSchemaVersion ||
    input.candidate.syncProtocolVersion < input.active.syncProtocolVersion;
  if (databaseDowngradeRequested) {
    return decision(
      "read-only-recovery",
      false,
      "A shell rollback cannot downgrade IndexedDB, package, or protocol state.",
    );
  }

  if (input.clients.some((client) => !clientIsCompatible(client, input.candidate))) {
    return decision(
      "deferred-incompatible-client",
      true,
      "An open client is outside the explicit N/N-1 compatibility window.",
    );
  }
  if (hasPendingLocalWork(input)) {
    return decision(
      "deferred-unsynced-work",
      true,
      "Pending outbox, package, or attachment work blocks update activation.",
    );
  }
  if (input.migration.required && !input.migration.ownerLockAcquired) {
    return decision(
      "deferred-migration-owner",
      false,
      "One approved migration owner lock is required across tabs.",
    );
  }
  if (input.migration.required) {
    return decision(
      "paused-for-migration",
      false,
      `Edits remain paused during the ${input.migration.phase} migration phase.`,
    );
  }

  if (input.candidate.appShellVersion === input.active.appShellVersion - 1) {
    return decision(
      "rollback-shell-only",
      true,
      "The N-1 shell may be restored without a database downgrade.",
    );
  }
  return decision(
    "ready-for-user-activation",
    true,
    "The waiting shell is compatible; activation still requires an explicit coordinated action.",
  );
}

export interface UpdateOwnerLock {
  request<T>(name: string, callback: () => Promise<T>): Promise<T>;
}

export interface UpdateDecisionMessage {
  type: "update-decision";
  code: UpdateDecisionCode;
  allowEdits: boolean;
}

export class UpdateCoordinator {
  constructor(
    private readonly lock: UpdateOwnerLock,
    private readonly broadcast: (message: UpdateDecisionMessage) => void,
  ) {}

  async evaluate(input: UpdateSafetyInput): Promise<UpdateDecision> {
    return this.lock.request("aviasurveil360-offline-update-owner", async () => {
      const result = evaluateUpdateSafety(input);
      this.broadcast({ type: "update-decision", code: result.code, allowEdits: result.allowEdits });
      return result;
    });
  }
}

interface BrowserLockManager {
  request<T>(name: string, callback: () => Promise<T>): Promise<T>;
}

export function createBrowserUpdateCoordinator(): {
  coordinator: UpdateCoordinator | null;
  close(): void;
} {
  const browserNavigator = navigator as Navigator & { locks?: BrowserLockManager };
  if (!browserNavigator.locks || typeof BroadcastChannel === "undefined") {
    return { coordinator: null, close() {} };
  }
  const channel = new BroadcastChannel("aviasurveil360-offline-updates");
  return {
    coordinator: new UpdateCoordinator(browserNavigator.locks, (message) => channel.postMessage(message)),
    close: () => channel.close(),
  };
}

export async function registerAppShellServiceWorker(): Promise<ServiceWorkerRegistration | null> {
  if (!import.meta.env.PROD || !("serviceWorker" in navigator)) return null;
  const registration = await navigator.serviceWorker.register(
    `/sw.js?v=${CURRENT_OFFLINE_VERSIONS.appShellVersion}`,
    {
      scope: "/",
      type: "module",
      updateViaCache: "none",
    },
  );
  registration.addEventListener("updatefound", () => {
    const installing = registration.installing;
    installing?.addEventListener("statechange", () => {
      if (installing.state === "installed" && registration.waiting) {
        window.dispatchEvent(
          new CustomEvent("avia:app-shell-update-waiting", {
            detail: { automaticActivation: false },
          }),
        );
      }
    });
  });
  return registration;
}
