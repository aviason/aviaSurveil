import { CURRENT_OFFLINE_VERSIONS, type OfflineVersionVector } from "./offline-version-contract";
import type { QuiescenceCounter } from "./client-quiescence";

export const APP_SHELL_UPDATE_POLL_INTERVAL_MS = 60_000;

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
  responsive?: boolean;
}

export interface UpdateQuiescenceInput {
  dirtyFormCount: number;
  active: Partial<Record<QuiescenceCounter, number>>;
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
  quiescence?: UpdateQuiescenceInput;
  migration: {
    required: boolean;
    ownerLockAcquired: boolean;
    phase: MigrationPhase;
    failed: boolean;
  };
}

export type UpdateDecisionCode =
  | "ready-for-safe-checkpoint"
  | "waiting-for-safe-checkpoint"
  | "deferred-incompatible-client"
  | "read-only-recovery"
  | "blocked-vector-change";

export interface UpdateDecision {
  code: UpdateDecisionCode;
  allowEdits: boolean;
  autoActivate: boolean;
  allowDocumentReload: boolean;
  preserveLocalData: true;
  deleteOldCaches: boolean;
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
    autoActivate: false,
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
  const activeCounters = Object.values(input.quiescence?.active ?? {}).some(
    (count) => (count ?? 0) > 0,
  );
  return (
    (input.quiescence?.dirtyFormCount ?? 0) > 0 ||
    activeCounters ||
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

  if (hasPendingLocalWork(input) || input.migration.required && !input.migration.ownerLockAcquired) {
    return decision(
      "waiting-for-safe-checkpoint",
      true,
      hasPendingLocalWork(input)
        ? "Mutation, form, storage, hashing, upload, or sync work is active; wait for a safe checkpoint before activation."
        : "A migration owner is not acquired; keep the candidate waiting and preserve local work.",
    );
  }

  if (input.clients.some((client) => !clientIsExact(client, input.candidate) || client.responsive === false)) {
    return decision(
      "deferred-incompatible-client",
      true,
      "An open client does not report the exact complete compatibility vector.",
    );
  }
  return decision(
    "ready-for-safe-checkpoint",
    true,
    "All observed clients report the exact vector and local work is quiescent; obtain safe-checkpoint ACKs before activation.",
  );
}

export type UnresponsiveClientFenceState =
  | "WAITING_FOR_ACK"
  | "ACK_TIMEOUT"
  | "SERVER_MINIMUM_WRITE_VECTOR_COMMITTED"
  | "RESPONSIVE_CLIENTS_FROZEN_AND_ACKED"
  | "UNRESPONSIVE_CLIENT_FENCED_READ_ONLY_PENDING_RESUME"
  | "SECURITY_UPDATE_ENFORCED_SHELL_PENDING"
  | "SAFE_CHECKPOINT_ACKED"
  | "CLIENT_EXITED";

export interface UnresponsiveClientFenceInput {
  ackTimedOut: boolean;
  clientExited: boolean;
  securityCritical: boolean;
  oldVectorDeadlineReached: boolean;
  serverMinimumWriteVectorCommitted: boolean;
  responsiveClientsFrozenAndAcked: boolean;
  resumedClientSafeAck: boolean;
}

export interface UnresponsiveClientFenceDecision {
  state: UnresponsiveClientFenceState;
  activationMayProceed: boolean;
  mutationsAllowed: boolean;
}

export function evaluateUnresponsiveClientFence(
  input: UnresponsiveClientFenceInput,
): UnresponsiveClientFenceDecision {
  if (!input.ackTimedOut) {
    return { state: "WAITING_FOR_ACK", activationMayProceed: false, mutationsAllowed: true };
  }
  if (input.clientExited) {
    return { state: "CLIENT_EXITED", activationMayProceed: true, mutationsAllowed: false };
  }
  if (input.resumedClientSafeAck) {
    return { state: "SAFE_CHECKPOINT_ACKED", activationMayProceed: true, mutationsAllowed: false };
  }
  if (
    (input.securityCritical || input.oldVectorDeadlineReached) &&
    input.serverMinimumWriteVectorCommitted &&
    input.responsiveClientsFrozenAndAcked
  ) {
    return {
      state: "UNRESPONSIVE_CLIENT_FENCED_READ_ONLY_PENDING_RESUME",
      activationMayProceed: false,
      mutationsAllowed: false,
    };
  }
  return { state: "ACK_TIMEOUT", activationMayProceed: false, mutationsAllowed: true };
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

export interface AppShellUpdateMonitorEnvironment {
  eventTarget: EventTarget;
  documentTarget: EventTarget & { visibilityState: string };
  isOnline(): boolean;
  currentAssetURL?: string;
  loadNetworkManifest?(): Promise<unknown>;
  loadCurrentAssetDigest?(): Promise<string | null>;
  reportStaleDocument?(fingerprint: string): void;
  setInterval(callback: () => void, intervalMs: number): unknown;
  clearInterval(handle: unknown): void;
  reportFailure(error: unknown): void;
}

export interface AppShellUpdateMonitor {
  checkNow(): Promise<void>;
  close(): void;
}

function browserUpdateMonitorEnvironment(): AppShellUpdateMonitorEnvironment {
  return {
    eventTarget: window,
    documentTarget: document,
    isOnline: () => navigator.onLine,
    currentAssetURL: import.meta.url,
    loadNetworkManifest: async () => {
      const manifestURL = new URL("/app-shell-assets.json", window.location.href);
      manifestURL.searchParams.set("avia_shell_probe", `${Date.now()}-${Math.random().toString(36).slice(2)}`);
      const response = await fetch(manifestURL, {
        cache: "no-store",
        headers: { accept: "application/json" },
      });
      if (!response.ok) throw new Error(`App-shell manifest probe failed with ${response.status}.`);
      return response.json();
    },
    loadCurrentAssetDigest: async () => {
      const assetURL = new URL(import.meta.url, window.location.href);
      assetURL.searchParams.set("avia_shell_probe", `${Date.now()}-${Math.random().toString(36).slice(2)}`);
      const response = await fetch(assetURL, {
        cache: "no-store",
        credentials: "same-origin",
        headers: { accept: "text/javascript" },
      });
      if (!response.ok) throw new Error(`App-shell asset probe failed with ${response.status}.`);
      const bytes = await response.arrayBuffer();
      const hash = await crypto.subtle.digest("SHA-256", bytes);
      return `sha256:${Array.from(new Uint8Array(hash), (byte) => byte.toString(16).padStart(2, "0")).join("")}`;
    },
    reportStaleDocument: (fingerprint) => {
      window.dispatchEvent(new CustomEvent("avia:app-shell-stale-document", {
        detail: { fingerprint },
      }));
    },
    setInterval: (callback, intervalMs) => window.setInterval(callback, intervalMs),
    clearInterval: (handle) => window.clearInterval(handle as number),
    reportFailure: () => {
      window.dispatchEvent(new CustomEvent("avia:app-shell-update-check-failed"));
    },
  };
}

export function staleDocumentFingerprint(
  manifest: unknown,
  currentAssetURL: string,
  currentAssetDigest?: string | null,
): string | null {
  if (!manifest || typeof manifest !== "object") return null;
  const candidate = manifest as { releaseFingerprint?: unknown; files?: unknown };
  if (
    typeof candidate.releaseFingerprint !== "string" ||
    !/^sha256:[0-9a-f]{64}$/.test(candidate.releaseFingerprint) ||
    !Array.isArray(candidate.files)
  ) {
    return null;
  }
  let currentPath: string;
  try {
    currentPath = new URL(currentAssetURL).pathname;
  } catch {
    return null;
  }
  const record = candidate.files.find((file) => (
    Boolean(file) &&
    typeof file === "object" &&
    (file as { url?: unknown }).url === currentPath
  )) as { sha256?: unknown } | undefined;
  if (!record) return candidate.releaseFingerprint;
  if (typeof currentAssetDigest === "string" && record.sha256 !== currentAssetDigest) return candidate.releaseFingerprint;
  return null;
}

export function installAppShellUpdateMonitor(
  registration: ServiceWorkerRegistration,
  environment: AppShellUpdateMonitorEnvironment = browserUpdateMonitorEnvironment(),
): AppShellUpdateMonitor {
  let closed = false;
  let inFlight: Promise<void> | null = null;

  const checkNow = (): Promise<void> => {
    if (
      closed ||
      !environment.isOnline() ||
      environment.documentTarget.visibilityState !== "visible"
    ) {
      return Promise.resolve();
    }
    if (inFlight) return inFlight;

    const request = (async () => {
      try {
        await registration.update();
        if (
          environment.currentAssetURL &&
          environment.loadNetworkManifest &&
          environment.reportStaleDocument
        ) {
          const fingerprint = staleDocumentFingerprint(
            await environment.loadNetworkManifest(),
            environment.currentAssetURL,
            environment.loadCurrentAssetDigest ? await environment.loadCurrentAssetDigest() : null,
          );
          if (fingerprint) environment.reportStaleDocument(fingerprint);
        }
      } catch (error) {
        environment.reportFailure(error);
      }
    })();
    inFlight = request;
    void request.finally(() => {
      if (inFlight === request) inFlight = null;
    });
    return request;
  };

  const checkWhenVisible = () => {
    if (environment.documentTarget.visibilityState === "visible") void checkNow();
  };
  const checkWhenFocused = () => void checkNow();
  const checkWhenOnline = () => void checkNow();
  const checkWhenShown = () => void checkNow();
  const poll = () => void checkNow();

  environment.eventTarget.addEventListener("online", checkWhenOnline);
  environment.eventTarget.addEventListener("pageshow", checkWhenShown);
  environment.eventTarget.addEventListener("focus", checkWhenFocused);
  environment.documentTarget.addEventListener("visibilitychange", checkWhenVisible);
  const intervalHandle = environment.setInterval(poll, APP_SHELL_UPDATE_POLL_INTERVAL_MS);
  void checkNow();

  return {
    checkNow,
    close() {
      if (closed) return;
      closed = true;
      environment.clearInterval(intervalHandle);
      environment.eventTarget.removeEventListener("online", checkWhenOnline);
      environment.eventTarget.removeEventListener("pageshow", checkWhenShown);
      environment.eventTarget.removeEventListener("focus", checkWhenFocused);
      environment.documentTarget.removeEventListener("visibilitychange", checkWhenVisible);
    },
  };
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
