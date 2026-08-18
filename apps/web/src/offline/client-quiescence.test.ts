import { describe, expect, it } from "vitest";

import { ClientQuiescence } from "./client-quiescence";

describe("client quiescence", () => {
  it("freezes a quiescent client and emits a safe-checkpoint acknowledgement", () => {
    const state = new ClientQuiescence("client-a");
    state.setPageState("sha256:old", { appShellVersion: 9, indexedDbSchemaVersion: 2, packageSchemaVersion: 1, syncProtocolVersion: 1 });

    expect(state.freezeForSafeCheckpoint()).toBe(true);
    expect(state.snapshot().frozenForSafeCheckpoint).toBe(true);
    expect(state.safeCheckpointAck("sha256:new", { appShellVersion: 9, indexedDbSchemaVersion: 2, packageSchemaVersion: 1, syncProtocolVersion: 1 })).toMatchObject({
      clientId: "client-a",
      fingerprint: "sha256:new",
      dirtyFormCount: 0,
      active: { indexedDb: 0, opfs: 0, hashWorker: 0, sync: 0, mutation: 0 },
      durableWorkAcknowledged: true,
    });
    expect(() => state.begin("mutation")).toThrow(/safe checkpoint/i);
  });

  it("does not freeze a client with active work", () => {
    const state = new ClientQuiescence("client-b");
    const release = state.begin("sync");
    expect(state.freezeForSafeCheckpoint()).toBe(false);
    expect(state.snapshot().frozenForSafeCheckpoint).toBe(false);
    release();
  });

  it("does not acknowledge a reload while forms or durable work are active", () => {
    const state = new ClientQuiescence();
    state.setPageState("sha256:old", { appShellVersion: 9, indexedDbSchemaVersion: 2, packageSchemaVersion: 1, syncProtocolVersion: 1 });
    state.requestReload("sha256:new");
    const clearForm = state.registerDirtyForm("checklist");
    const releaseDatabase = state.begin("indexedDb");
    expect(state.canAcknowledgeReload("sha256:new")).toBe(false);
    clearForm();
    expect(state.canAcknowledgeReload("sha256:new")).toBe(false);
    releaseDatabase();
    expect(state.canAcknowledgeReload("sha256:new")).toBe(true);
    expect(state.acknowledgeReload("sha256:new")).toBe(true);
    expect(state.snapshot().acknowledgedReloadFingerprint).toBe("sha256:new");
  });

  it("requires a different known fingerprint and is idempotent", () => {
    const state = new ClientQuiescence();
    state.setPageState("sha256:same", null);
    state.requestReload("sha256:same");
    expect(state.acknowledgeReload("sha256:same")).toBe(false);
    state.requestReload("sha256:next");
    expect(state.acknowledgeReload("sha256:next")).toBe(true);
    expect(state.acknowledgeReload("sha256:next")).toBe(false);
  });
});
