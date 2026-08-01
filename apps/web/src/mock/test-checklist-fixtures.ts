import type { DemoBackend } from "../backend/backend";

type MockChecklistRuntime = {
  backendForRole(role: "leadInspector"): DemoBackend;
};

export interface CompleteMockChecklistOptions {
  excludeQuestionIds?: readonly string[];
}

/**
 * Completes the canonical mock package for tests that need to exercise a
 * later lifecycle stage. The Lead Inspector is used only as a fixture actor
 * so every server-assigned question can receive an exact response without
 * changing Inspector assignment boundaries.
 */
export async function completeMockChecklist(
  runtime: MockChecklistRuntime,
  packageId: string,
  options: CompleteMockChecklistOptions = {},
): Promise<void> {
  const lead = runtime.backendForRole("leadInspector");
  const packageView = await lead.inspections.getPackage({ packageId });
  const excluded = new Set(options.excludeQuestionIds ?? []);

  for (const question of packageView.questions) {
    if (excluded.has(question.id)) continue;
    if (question.currentResponse) continue;
    await lead.inspections.upsertChecklistResponse({
      operationId: `OP-FIXTURE-COMPLETE-${question.id}`,
      responseId: `RESP-${question.id}`,
      auditId: packageView.auditId,
      questionId: question.id,
      expectedResponseRevision: null,
      answer: "COMPLIANT",
      comment: "",
    });
  }
}
