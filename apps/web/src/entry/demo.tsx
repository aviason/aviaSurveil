import { bootstrap } from "../app/bootstrap";
import { DEMO_PRINCIPALS, createMockBackendPersistentRuntime } from "../mock/create-mock-backend";
import { seedVisualRuntimeForPath } from "../mock/seed-visual-runtime";

const mockRuntime = createMockBackendPersistentRuntime(window.localStorage);

declare global {
  interface Window {
    __aviaMaterializeSyntheticGovernedPackageForTest?: () => unknown;
  }
}

// The test-only seam is available only in the memory-mock artifact. It is not
// part of the HTTP client or production API surface.
window.__aviaMaterializeSyntheticGovernedPackageForTest = () =>
  mockRuntime.materializeSyntheticGovernedPackageForTest();

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
