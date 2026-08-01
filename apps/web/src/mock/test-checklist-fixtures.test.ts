import { describe, expect, it } from "vitest";

import { createMockBackendRuntime } from "./create-mock-backend";
import { completeMockChecklist } from "./test-checklist-fixtures";

describe("mock checklist fixtures", () => {
  it("completes every exact package question before submission", async () => {
    const runtime = createMockBackendRuntime();
    const inspector = runtime.backendForRole("inspector");
    const packageView = await inspector.inspections.getPackage({ packageId: "PKG-CAB-2026-001" });

    await inspector.inspections.upsertChecklistResponse({
      operationId: "OP-FIXTURE-PBE-RESPONSE",
      responseId: "RESP-CAB-EMEQ-PBE-001",
      auditId: packageView.auditId,
      questionId: "CAB-EMEQ-PBE-001",
      expectedResponseRevision: null,
      answer: "NON_COMPLIANT",
      comment: "PBE serviceability and accessibility could not be confirmed.",
    });

    await completeMockChecklist(runtime, packageView.id);

    const submitted = await runtime.backendForRole("inspector").inspections.submitChecklist({
      operationId: "OP-FIXTURE-SUBMIT",
      auditId: packageView.auditId,
      expectedChecklistRevision: packageView.checklistRevision,
    });

    expect(submitted.checklistStatus).toBe("SUBMITTED");
    expect(submitted.checklistRevision).toBe(packageView.checklistRevision + 1);
  });
});
