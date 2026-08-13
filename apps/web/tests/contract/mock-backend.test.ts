import { describe, expect, it } from "vitest";

import {
  backendContract,
  FIXED_NOW,
  PRINCIPALS,
  type BackendContractFixture,
  type BackendContractHarness,
} from "./backend-contract";
import type { DemoBackend } from "../../src/backend/backend";
import { createMockBackend } from "../../src/mock/create-mock-backend";
import { MemoryMockStore } from "../../src/mock/memory-mock-store";

backendContract(async (fixture: BackendContractFixture = "canonical"): Promise<BackendContractHarness> => {
  const store = MemoryMockStore.createCanonical({ clock: () => FIXED_NOW });
  if (fixture === "coordination") {
    store.execute("TEST-FIXTURE-COORDINATION", {}, (state) => {
      const packageView = state.packages["PKG-CAB-2026-001"];
      const assignment = state.assignments.find((candidate) => candidate.auditId === "AUD-2026-001");
      if (!packageView || !assignment) {
        throw new Error("Canonical coordination fixture is unavailable.");
      }
      packageView.checklistStatus = "NOT_STARTED";
      assignment.status = "AWAITING_AUDITEE_CONFIRMATION";
      assignment.nextAction = "Start inspection when ready";
      return null;
    });
  }
  store.execute("TEST-FIXTURE-FINAL-REPORT-DM-REVIEW", {}, (state) => {
    const report = state.reportVersions["RPT-CAB-2026-001-V1"];
    if (!report) throw new Error("Canonical Final Report fixture is unavailable.");
    report.status = "DEPARTMENT_REVIEW";
    report.revision = 1;
    report.issuedAt = null;
    return report;
  });
  return {
    backendFor(principal) {
      return createMockBackend({ store, principal });
    },
  };
});

describe("mock-only 85-screen capability boundary", () => {
  it("exposes immutable composed demo capabilities without activating HTTP behavior", async () => {
    const store = MemoryMockStore.createCanonical({ clock: () => FIXED_NOW });
    const backend: DemoBackend = createMockBackend({ store, principal: PRINCIPALS.inspector });

    expect(backend.mode).toBe("mock");
    expect(backend).toHaveProperty("communications");
    expect(backend).toHaveProperty("calendar");
    expect(backend).toHaveProperty("profiles");
    expect(backend).toHaveProperty("teams");
    expect(backend).toHaveProperty("risk");
    expect(backend).toHaveProperty("documents");
    expect(backend).toHaveProperty("notifications");
    expect(backend).toHaveProperty("administration");
    expect(backend).toHaveProperty("assistantDrafts");
    expect(backend).toHaveProperty("auditeeCoordination");
    expect(backend).toHaveProperty("auditeeReports");
    expect(backend).toHaveProperty("planningIntake");
    expect(backend.planningIntake).toHaveProperty("saveDraft");
    expect(backend.planningIntake).toHaveProperty("submit");
    expect(backend.assistantDrafts).toHaveProperty("createDraft");
    expect(backend.assistantDrafts).not.toHaveProperty("create");
    expect(backend.auditeeCoordination).toHaveProperty("list");
    expect(backend.auditeeCoordination).toHaveProperty("respond");
    expect(backend.auditeeReports).toHaveProperty("listReleased");
    expect(backend.auditeeReports).toHaveProperty("getReleased");

    const profile = await backend.profiles.getMine({});
    profile.displayName = "mutated caller copy";
    expect((await backend.profiles.getMine({})).displayName).toBe("Amina Inspector");
  });
});
