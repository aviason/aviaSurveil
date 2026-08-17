import { describe, expect, it } from "vitest";

import { ClientQuiescence } from "./client-quiescence";

describe("client quiescence", () => {
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
