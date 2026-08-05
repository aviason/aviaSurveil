// @vitest-environment jsdom
import "@testing-library/jest-dom/vitest";

import { cleanup, render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { afterEach, describe, expect, it, vi } from "vitest";

import { AppProviders } from "../app/providers";
import type { AGADemoWorkspaceBackend } from "../backend/aga-demo-workspace";
import { createMockBackendRuntime } from "../mock/create-mock-backend";
import { AGADemoWorkspaceGuard } from "./aga-demo-workspace-guard";

afterEach(() => cleanup());

function renderGuard(capability: AGADemoWorkspaceBackend["capability"]) {
  const runtime = createMockBackendRuntime();
  const workspace = {
    capability,
  } as AGADemoWorkspaceBackend;
  render(
    <AppProviders runtime={{ backend: { ...runtime.backend, agaDemoWorkspace: workspace }, buildProfile: "http", environmentLabel: "test" }}>
      <MemoryRouter>
        <AGADemoWorkspaceGuard requiredRole="manager">
          {(value) => <div data-testid="authorized-workspace">{value.projection}</div>}
        </AGADemoWorkspaceGuard>
      </MemoryRouter>
    </AppProviders>,
  );
}

describe("AGA demo workspace capability guard", () => {
  it("renders the authorized child only after the server capability succeeds", async () => {
    const capability = vi.fn().mockResolvedValue({
      available: true,
      projection: "DEPARTMENT_MANAGER_SCOPED",
      classificationEnabled: true,
      recommendationEnabled: true,
      lifecycleEnabled: false,
      resetEnabled: false,
    });
    renderGuard(capability);
    expect(await screen.findByTestId("authorized-workspace")).toHaveTextContent("DEPARTMENT_MANAGER_SCOPED");
    expect(capability).toHaveBeenCalledTimes(1);
  });

  it("keeps a neutral unavailable result when authorization or capability fails", async () => {
    const capability = vi.fn().mockResolvedValue({
      available: false,
      projection: "",
      classificationEnabled: false,
      recommendationEnabled: false,
      lifecycleEnabled: false,
      resetEnabled: false,
    });
    renderGuard(capability);
    expect(await screen.findByTestId("aga-workspace-capability-unavailable")).toHaveTextContent(/not available/i);
    expect(screen.queryByTestId("authorized-workspace")).not.toBeInTheDocument();
  });

  it("purges a BFCache-restored capability result before rendering the child", async () => {
    const capability = vi.fn().mockResolvedValue({
      available: true,
      projection: "DEPARTMENT_MANAGER_SCOPED",
      classificationEnabled: true,
      recommendationEnabled: true,
      lifecycleEnabled: false,
      resetEnabled: false,
    });
    renderGuard(capability);
    expect(await screen.findByTestId("authorized-workspace")).toBeInTheDocument();
    window.dispatchEvent(new PageTransitionEvent("pageshow", { persisted: true }));
    expect(await screen.findByRole("alert")).toHaveTextContent(/restored page was cleared/i);
    expect(screen.queryByTestId("authorized-workspace")).not.toBeInTheDocument();
  });
});
