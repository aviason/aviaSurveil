import { CURRENT_OFFLINE_VERSIONS, type OfflineVersionVector } from "./offline-version-contract";

export const UPDATE_ACTIVATION_POLICY = {
  automaticSkipWaiting: true,
  automaticClientsClaim: true,
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
  | "ready-for-automatic-activation"
  | "deferred-incompatible-client"
  | "read-only-recovery"
  | "blocked-vector-change";

export interface UpdateDecision {
  code: UpdateDecisionCode;
  allowEdits: boolean;
  autoActivate: boolean;
  allowDocumentReload: boolean;
  preserveLocalData: true;
  deleteOldCaches: false;
  databaseDowngradeAllowed: false;
  reason: string;
}

function decision(
  code: UpdateDecisionCode,
  allowEdits: boolean,
  reason: string,
  allowDocumentReload = false,
): UpdateDecision {
  return {
    code,
    allowEdits,
    autoActivate: code === "ready-for-automatic-activation",
    allowDocumentReload,
    preserveLocalData: true,
    deleteOldCaches: false,
    databaseDowngradeAllowed: false,
    reason,
  };
}

export function isExactOfflineVersion(version: number, current: number): boolean {
  return (
    Number.isSafeInteger(version) &&
    Number.isSafeInteger(current) &&
    version > 0 &&
    current > 0 &&
    version === current
  );
}

function hasPendingLocalWork(input: UpdateSafetyInput): boolean {
  return (
    input.localWork.pendingOutboxCount > 0 ||
    input.localWork.pendingAttachmentManifestCount > 0 ||
    input.localWork.unsyncedPackageCount > 0
  );
}

function clientIsExact(client: ClientVersion, candidate: OfflineVersionVector): boolean {
  return (
    client.appShellVersion === candidate.appShellVersion &&
    client.indexedDbSchemaVersion === candidate.indexedDbSchemaVersion &&
    client.packageSchemaVersion === candidate.packageSchemaVersion &&
    client.syncProtocolVersion === candidate.syncProtocolVersion
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

  if (
    input.candidate.appShellVersion !== input.active.appShellVersion ||
    input.candidate.indexedDbSchemaVersion !== input.active.indexedDbSchemaVersion ||
    input.candidate.packageSchemaVersion !== input.active.packageSchemaVersion ||
    input.candidate.syncProtocolVersion !== input.active.syncProtocolVersion
  ) {
    return decision(
      "blocked-vector-change",
      false,
      "The candidate vector differs from the active vector; a separate migration/bootstrap-shell plan is required.",
    );
  }

  if (input.clients.some((client) => !clientIsExact(client, input.candidate))) {
    return decision(
      "deferred-incompatible-client",
      true,
      "An open client does not report the exact complete compatibility vector.",
    );
  }
  return decision(
    "ready-for-automatic-activation",
    true,
    hasPendingLocalWork(input)
      ? "The exact-vector worker may activate; document reload remains gated by client quiescence and durable local work."
      : "The exact-vector worker may activate; document reload remains gated by client quiescence.",
    !hasPendingLocalWork(input) && !input.migration.required,
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
    "/sw.js",
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
            detail: { automaticActivation: true },
          }),
        );
      }
    });
  });
  return registration;
}
