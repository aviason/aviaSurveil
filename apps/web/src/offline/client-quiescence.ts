import type { OfflineVersionVector } from "./offline-version-contract";

export type QuiescenceCounter = "indexedDb" | "opfs" | "hashWorker" | "sync" | "mutation";

export interface ClientQuiescenceSnapshot {
  dirtyFormCount: number;
  active: Readonly<Record<QuiescenceCounter, number>>;
  currentFingerprint: string | null;
  persistedVector: OfflineVersionVector | null;
  pendingReloadFingerprint: string | null;
  acknowledgedReloadFingerprint: string | null;
}

const COUNTERS: readonly QuiescenceCounter[] = ["indexedDb", "opfs", "hashWorker", "sync", "mutation"];

export class ClientQuiescence {
  private readonly dirtyForms = new Set<string>();
  private readonly counters = new Map<QuiescenceCounter, number>(COUNTERS.map((counter) => [counter, 0]));
  private currentFingerprint: string | null = null;
  private persistedVector: OfflineVersionVector | null = null;
  private pendingReloadFingerprint: string | null = null;
  private acknowledgedReloadFingerprint: string | null = null;

  registerDirtyForm(formID: string): () => void {
    this.dirtyForms.add(formID);
    return () => this.clearDirtyForm(formID);
  }

  clearDirtyForm(formID: string): void {
    this.dirtyForms.delete(formID);
  }

  begin(counter: QuiescenceCounter): () => void {
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
  const onControllerChange = () => {
    sendReady();
    const pending = state.snapshot().pendingReloadFingerprint;
    if (pending && state.acknowledgeReload(pending)) window.location.reload();
  };
  const onWorkerMessage = (event: MessageEvent) => {
    if (event.data?.type !== "avia:app-shell-activation" || typeof event.data.fingerprint !== "string") return;
    if (event.data.legacyRetirement === true) return;
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
