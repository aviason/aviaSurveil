import { bootstrap } from "../app/bootstrap";
import { DEMO_PRINCIPALS, createMockBackendPersistentRuntime } from "../mock/create-mock-backend";
import { seedVisualRuntimeForPath } from "../mock/seed-visual-runtime";
import { completeMockChecklist } from "../mock/test-checklist-fixtures";

const mockRuntime = createMockBackendPersistentRuntime(window.localStorage);

declare global {
  interface Window {
    __aviaMaterializeSyntheticGovernedPackageForTest?: () => unknown;
    __aviaCompleteCanonicalChecklistForTest?: () => Promise<unknown>;
  }
}

// The test-only seam is available only in the memory-mock artifact. It is not
// part of the HTTP client or production API surface.
window.__aviaMaterializeSyntheticGovernedPackageForTest = () =>
  mockRuntime.materializeSyntheticGovernedPackageForTest();

// Test-only fixture seam: complete every canonical question except the PBE
// question driven by the UI, so the lifecycle exercises the server's
// fail-closed submit rule without changing Inspector assignment boundaries.
window.__aviaCompleteCanonicalChecklistForTest = async () => {
  const inspector = mockRuntime.backendForRole("inspector");
  const packageView = await inspector.inspections.getPackage({ packageId: "PKG-CAB-2026-001" });
  await completeMockChecklist(mockRuntime, packageView.id, {
    excludeQuestionIds: ["CAB-EMEQ-PBE-001"],
  });
  const completed = await inspector.inspections.getPackage({ packageId: packageView.id });
  return completed.questions.map((question) => ({
    id: question.id,
    assignedInspectorUserIds: question.assignedInspectorUserIds,
    currentResponse: question.currentResponse?.answer ?? null,
  }));
};

async function startDemo(): Promise<void> {
  if (
    import.meta.env.VITE_AVIA_VISUAL_FIXTURES === "1" ||
    window.sessionStorage.getItem("avia-route-matrix-fixtures") === "1"
  ) {
    await seedVisualRuntimeForPath(mockRuntime, window.location.pathname);
  }

  bootstrap({
    backend: mockRuntime.backend,
    backendForRole: mockRuntime.backendForRole,
    buildProfile: "demo",
    environmentLabel: "Deterministic memory mock",
    identityMode: "demo-role-switch",
    subjectId: DEMO_PRINCIPALS.inspector.subjectId,
  });
}

void startDemo();
