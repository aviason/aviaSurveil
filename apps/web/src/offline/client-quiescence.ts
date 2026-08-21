import type { OfflineVersionVector } from "./offline-version-contract";
import {
  APP_SHELL_UPDATE_PROTOCOL_VERSION,
  releaseFingerprintFromActivationMessage,
} from "./app-shell-update-protocol";

export type QuiescenceCounter = "indexedDb" | "opfs" | "hashWorker" | "sync" | "mutation";

export interface ClientQuiescenceSnapshot {
  dirtyFormCount: number;
  active: Readonly<Record<QuiescenceCounter, number>>;
  currentFingerprint: string | null;
  persistedVector: OfflineVersionVector | null;
  pendingReloadFingerprint: string | null;
  acknowledgedReloadFingerprint: string | null;
  frozenForSafeCheckpoint: boolean;
}

export interface SafeCheckpointAck {
  clientId: string;
  fingerprint: string;
  compatibility: OfflineVersionVector;
  dirtyFormCount: number;
  active: Readonly<Record<QuiescenceCounter, number>>;
  durableWorkAcknowledged: boolean;
}

const COUNTERS: readonly QuiescenceCounter[] = ["indexedDb", "opfs", "hashWorker", "sync", "mutation"];

export class ClientQuiescence {
  private readonly clientId: string;
  private readonly dirtyForms = new Set<string>();
  private readonly counters = new Map<QuiescenceCounter, number>(COUNTERS.map((counter) => [counter, 0]));
  private currentFingerprint: string | null = null;
  private persistedVector: OfflineVersionVector | null = null;
  private pendingReloadFingerprint: string | null = null;
  private acknowledgedReloadFingerprint: string | null = null;
  private frozenForSafeCheckpoint = false;
  private readonly quiescentListeners = new Set<() => void>();

  constructor(clientId: string = globalThis.crypto?.randomUUID?.() ?? `client-${Date.now()}`) {
    this.clientId = clientId;
  }

  registerDirtyForm(formID: string): () => void {
    if (this.frozenForSafeCheckpoint) {
      throw new Error("Client is frozen for a safe checkpoint.");
    }
    this.dirtyForms.add(formID);
    return () => this.clearDirtyForm(formID);
  }

  clearDirtyForm(formID: string): void {
    this.dirtyForms.delete(formID);
    this.notifyQuiescent();
  }

  begin(counter: QuiescenceCounter): () => void {
    if (this.frozenForSafeCheckpoint) {
      throw new Error("Client is frozen for a safe checkpoint.");
    }
    this.counters.set(counter, (this.counters.get(counter) ?? 0) + 1);
    let released = false;
    return () => {
      if (released) return;
      released = true;
      this.counters.set(counter, Math.max(0, (this.counters.get(counter) ?? 1) - 1));
      this.notifyQuiescent();
    };
  }

  onQuiescent(listener: () => void): () => void {
    this.quiescentListeners.add(listener);
    return () => this.quiescentListeners.delete(listener);
  }

  private notifyQuiescent(): void {
    if (!this.isQuiescent()) return;
    for (const listener of this.quiescentListeners) listener();
  }

  setPageState(fingerprint: string | null, vector: OfflineVersionVector | null): void {
    this.currentFingerprint = fingerprint;
    this.persistedVector = vector;
  }

  requestReload(fingerprint: string): void {
    this.pendingReloadFingerprint = fingerprint;
  }

  isQuiescent(): boolean {
    return this.dirtyForms.size === 0 && COUNTERS.every((counter) => (this.counters.get(counter) ?? 0) === 0);
  }

  freezeForSafeCheckpoint(): boolean {
    if (!this.isQuiescent()) return false;
    this.frozenForSafeCheckpoint = true;
    return true;
  }

  safeCheckpointAck(
    fingerprint: string,
    compatibility: OfflineVersionVector,
    durableWorkAcknowledged = true,
  ): SafeCheckpointAck | null {
    if (!this.frozenForSafeCheckpoint || !this.isQuiescent()) return null;
    const snapshot = this.snapshot();
    return {
      clientId: this.clientId,
      fingerprint,
      compatibility,
      dirtyFormCount: snapshot.dirtyFormCount,
      active: snapshot.active,
      durableWorkAcknowledged,
    };
  }

  canAcknowledgeReload(fingerprint: string): boolean {
    return this.pendingReloadFingerprint === fingerprint && this.acknowledgedReloadFingerprint !== fingerprint && this.currentFingerprint !== null && this.currentFingerprint !== fingerprint && this.isQuiescent();
  }

  acknowledgeReload(fingerprint: string): boolean {
    if (!this.canAcknowledgeReload(fingerprint)) return false;
    this.acknowledgedReloadFingerprint = fingerprint;
    return true;
  }

  snapshot(): ClientQuiescenceSnapshot {
    return {
      dirtyFormCount: this.dirtyForms.size,
      active: Object.fromEntries(COUNTERS.map((counter) => [counter, this.counters.get(counter) ?? 0])) as Record<QuiescenceCounter, number>,
      currentFingerprint: this.currentFingerprint,
      persistedVector: this.persistedVector,
      pendingReloadFingerprint: this.pendingReloadFingerprint,
      acknowledgedReloadFingerprint: this.acknowledgedReloadFingerprint,
      frozenForSafeCheckpoint: this.frozenForSafeCheckpoint,
    };
  }
}

export interface BrowserQuiescenceBinding {
  state: ClientQuiescence;
  dispose(): void;
}

export function appShellRecoveryURL(pathname: string, search: string, hash: string): string {
  const returnTo = `${pathname.startsWith("/") ? pathname : "/"}${search}${hash}`;
  return `/app-shell-recovery.html?returnTo=${encodeURIComponent(returnTo)}`;
}

export function bindBrowserQuiescence(registration: ServiceWorkerRegistration): BrowserQuiescenceBinding {
  const state = new ClientQuiescence();
  const clientAssetURL = import.meta.url;
  let pendingReloadFingerprint: string | null = null;
  let pendingCandidate: {
    fingerprint: string;
    vector: OfflineVersionVector;
    target: ServiceWorker | null;
  } | null = null;
  let acknowledgedCandidateFingerprint: string | null = null;
  let reportedCandidateFingerprint: string | null = null;
  let reloadStarted = false;
  let staleDocumentFingerprint: string | null = null;
  let recoveryStarted = false;
  let watchedInstallingWorker: ServiceWorker | null = null;

  const requestWaitingCandidate = () => {
    registration.waiting?.postMessage({ type: "avia:app-shell-update-check" });
  };

  const sendReady = () => {
    registration.active?.postMessage({
      type: "avia:app-shell-client-ready",
      fingerprint: state.snapshot().currentFingerprint,
      protocolVersion: APP_SHELL_UPDATE_PROTOCOL_VERSION,
      clientAssetURL,
    });
    requestWaitingCandidate();
  };
  const notifyUpdateCandidate = (fingerprint: string) => {
    if (reportedCandidateFingerprint === fingerprint) return;
    reportedCandidateFingerprint = fingerprint;
    window.dispatchEvent(new CustomEvent("avia:app-shell-update-candidate", {
      detail: { fingerprint },
    }));
  };
  const reloadWhenQuiescent = () => {
    if (
      reloadStarted ||
      !pendingReloadFingerprint ||
      navigator.serviceWorker.controller !== registration.active ||
      !state.freezeForSafeCheckpoint() ||
      !state.acknowledgeReload(pendingReloadFingerprint)
    ) {
      return;
    }
    reloadStarted = true;
    window.dispatchEvent(new CustomEvent("avia:app-shell-reload-required", {
      detail: { fingerprint: pendingReloadFingerprint, automatic: true },
    }));
    window.location.reload();
  };
  const recoverStaleDocumentWhenQuiescent = () => {
    if (
      recoveryStarted ||
      !staleDocumentFingerprint ||
      !state.freezeForSafeCheckpoint()
    ) {
      return;
    }
    recoveryStarted = true;
    window.location.replace(appShellRecoveryURL(
      window.location.pathname,
      window.location.search,
      window.location.hash,
    ));
  };
  const acknowledgePendingCandidate = () => {
    if (
      !pendingCandidate ||
      acknowledgedCandidateFingerprint === pendingCandidate.fingerprint ||
      !state.freezeForSafeCheckpoint()
    ) {
      return;
    }
    const ack = state.safeCheckpointAck(pendingCandidate.fingerprint, pendingCandidate.vector);
    if (!ack) return;
    (pendingCandidate.target ?? registration.waiting ?? registration.installing)?.postMessage({
      type: "avia:app-shell-safe-checkpoint-ack",
      ack,
    });
    acknowledgedCandidateFingerprint = pendingCandidate.fingerprint;
  };
  const sendSafeCheckpointAck = (
    fingerprint: string,
    vector: OfflineVersionVector,
    target?: ServiceWorker | null,
  ) => {
    pendingCandidate = { fingerprint, vector, target: target ?? null };
    acknowledgePendingCandidate();
  };
  const onControllerChange = () => {
    sendReady();
    reloadWhenQuiescent();
  };
  const onWorkerMessage = (event: MessageEvent) => {
    if (event.data?.type === "avia:app-shell-update-available" && typeof event.data.fingerprint === "string") {
      const vector = event.data.compatibility as OfflineVersionVector;
      notifyUpdateCandidate(event.data.fingerprint);
      sendSafeCheckpointAck(event.data.fingerprint, vector, event.source as ServiceWorker | null);
      return;
    }
    if (event.data?.type === "avia:app-shell-worker-ready") {
      const loadedFingerprint = typeof event.data.fingerprint === "string" ? event.data.fingerprint : null;
      const activeFingerprint = typeof event.data.activeFingerprint === "string" ? event.data.activeFingerprint : null;
      if (loadedFingerprint) state.setPageState(loadedFingerprint, event.data.compatibility ?? null);
      if (loadedFingerprint && activeFingerprint && loadedFingerprint !== activeFingerprint) {
        pendingReloadFingerprint = activeFingerprint;
        notifyUpdateCandidate(activeFingerprint);
        state.requestReload(activeFingerprint);
        reloadWhenQuiescent();
      }
      return;
    }
    if (event.data?.type !== "avia:app-shell-activation") return;
    const releaseFingerprint = releaseFingerprintFromActivationMessage(event.data);
    if (!releaseFingerprint) return;
    pendingReloadFingerprint = releaseFingerprint;
    notifyUpdateCandidate(releaseFingerprint);
    state.requestReload(releaseFingerprint);
    if (navigator.serviceWorker.controller === registration.active) onControllerChange();
  };
  const onPageShow = () => sendReady();
  const onVisibilityChange = () => {
    if (document.visibilityState === "visible") sendReady();
  };
  const onStaleDocument = (event: Event) => {
    const fingerprint = (event as CustomEvent<{ fingerprint?: unknown }>).detail?.fingerprint;
    if (typeof fingerprint !== "string") return;
    staleDocumentFingerprint = fingerprint;
    notifyUpdateCandidate(fingerprint);
    recoverStaleDocumentWhenQuiescent();
  };
  const onInstallingStateChange = () => {
    if (watchedInstallingWorker?.state === "installed") requestWaitingCandidate();
  };
  const watchInstallingWorker = () => {
    watchedInstallingWorker?.removeEventListener("statechange", onInstallingStateChange);
    watchedInstallingWorker = registration.installing;
    watchedInstallingWorker?.addEventListener("statechange", onInstallingStateChange);
    onInstallingStateChange();
  };
  const stopQuiescentListener = state.onQuiescent(() => {
    recoverStaleDocumentWhenQuiescent();
    if (recoveryStarted) return;
    acknowledgePendingCandidate();
    reloadWhenQuiescent();
  });
  navigator.serviceWorker.addEventListener("controllerchange", onControllerChange);
  navigator.serviceWorker.addEventListener("message", onWorkerMessage);
  window.addEventListener("pageshow", onPageShow);
  window.addEventListener("avia:app-shell-stale-document", onStaleDocument);
  document.addEventListener("visibilitychange", onVisibilityChange);
  registration.addEventListener("updatefound", watchInstallingWorker);
  watchInstallingWorker();
  sendReady();
  return {
    state,
    dispose: () => {
      stopQuiescentListener();
      navigator.serviceWorker.removeEventListener("controllerchange", onControllerChange);
      navigator.serviceWorker.removeEventListener("message", onWorkerMessage);
      window.removeEventListener("pageshow", onPageShow);
      window.removeEventListener("avia:app-shell-stale-document", onStaleDocument);
      document.removeEventListener("visibilitychange", onVisibilityChange);
      registration.removeEventListener("updatefound", watchInstallingWorker);
      watchedInstallingWorker?.removeEventListener("statechange", onInstallingStateChange);
    },
  };
}
