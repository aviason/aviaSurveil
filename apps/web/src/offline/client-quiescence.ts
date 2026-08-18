import type { OfflineVersionVector } from "./offline-version-contract";

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
    };
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

export function bindBrowserQuiescence(registration: ServiceWorkerRegistration): BrowserQuiescenceBinding {
  const state = new ClientQuiescence();
  const page = document.documentElement.dataset;
  state.setPageState(page.appShellFingerprint ?? null, null);

  const sendReady = () => {
    registration.active?.postMessage({
      type: "avia:app-shell-client-ready",
      fingerprint: state.snapshot().currentFingerprint,
    });
  };
  const sendSafeCheckpointAck = (
    fingerprint: string,
    vector: OfflineVersionVector,
    target?: ServiceWorker | null,
  ) => {
    if (!state.freezeForSafeCheckpoint()) return;
    const ack = state.safeCheckpointAck(fingerprint, vector);
    if (!ack) return;
    (target ?? registration.waiting ?? registration.installing)?.postMessage({
      type: "avia:app-shell-safe-checkpoint-ack",
      ack,
    });
  };
  const onControllerChange = () => {
    sendReady();
    const pending = state.snapshot().pendingReloadFingerprint;
    if (pending && state.acknowledgeReload(pending)) {
      window.dispatchEvent(new CustomEvent("avia:app-shell-reload-required", { detail: { fingerprint: pending } }));
    }
  };
  const onWorkerMessage = (event: MessageEvent) => {
    if (event.data?.type === "avia:app-shell-update-available" && typeof event.data.fingerprint === "string") {
      const vector = event.data.compatibility as OfflineVersionVector;
      sendSafeCheckpointAck(event.data.fingerprint, vector, event.source as ServiceWorker | null);
      return;
    }
    if (event.data?.type !== "avia:app-shell-activation" || typeof event.data.fingerprint !== "string") return;
    state.requestReload(event.data.fingerprint);
    if (navigator.serviceWorker.controller === registration.active) onControllerChange();
  };
  navigator.serviceWorker.addEventListener("controllerchange", onControllerChange);
  navigator.serviceWorker.addEventListener("message", onWorkerMessage);
  sendReady();
  void fetch("/app-shell-assets.json", { cache: "no-store", credentials: "same-origin" })
    .then((response) => response.ok ? response.json() as Promise<{ releaseFingerprint?: unknown; compatibility?: OfflineVersionVector }> : null)
    .then((manifest) => {
      if (!manifest || typeof manifest.releaseFingerprint !== "string") return;
      state.setPageState(manifest.releaseFingerprint, manifest.compatibility ?? null);
    })
    .catch(() => undefined);
  return {
    state,
    dispose: () => {
      navigator.serviceWorker.removeEventListener("controllerchange", onControllerChange);
      navigator.serviceWorker.removeEventListener("message", onWorkerMessage);
    },
  };
}
