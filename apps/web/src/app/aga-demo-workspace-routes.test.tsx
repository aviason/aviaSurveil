// @vitest-environment jsdom
import "@testing-library/jest-dom/vitest";

import { cleanup, render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { AppProviders } from "./providers";
import { AGA_DEMO_WORKSPACE_ROUTES, agaDemoWorkspaceRouteElements, agaDemoWorkspaceRouteElementsWithManagerPackage } from "./aga-demo-workspace-routes";
import type { AGADemoWorkspaceBackend } from "../backend/aga-demo-workspace";
import { createMockBackendRuntime } from "../mock/create-mock-backend";
import { ScenarioProvider } from "./scenario-context";
import { AppRouter } from "./router";

afterEach(() => cleanup());
beforeEach(() => cleanup());

function workspaceBackend(): AGADemoWorkspaceBackend {
  const generation = { generationId: "aga-ws-generation-0001", revision: 1, sealDigest: "sha256:0000000000000000000000000000000000000000000000000000000000000000" };
  const response = { operation: "SEARCH_ITEMS", generation, itemCount: 0, lifecycleAvailable: false };
  return {
    capability: vi.fn().mockResolvedValue({ available: true, projection: "DEPARTMENT_MANAGER_SCOPED", classificationEnabled: true, recommendationEnabled: true, lifecycleEnabled: true, resetEnabled: false }),
    classificationQuery: vi.fn().mockResolvedValue(response),
    classificationCommand: vi.fn(),
    recommendationCommand: vi.fn(),
    lifecycleQuery: vi.fn(),
    lifecycleCommand: vi.fn(),
    adminCommand: vi.fn(),
  };
}

describe("AGA demo workspace supplemental routes", () => {
  it("declares the five fixed role routes without changing the accepted route registry", () => {
    expect(AGA_DEMO_WORKSPACE_ROUTES.map(({ path }) => path)).toEqual([
      "/admin/aga-demo-workspace",
      "/department-manager/aga-demo-workspace",
      "/inspector/aga-demo-workspace",
      "/lead-inspector/aga-demo-workspace",
      "/auditee/aga-demo-workspace",
    ]);
    expect(agaDemoWorkspaceRouteElements).toHaveLength(5);
    expect(agaDemoWorkspaceRouteElementsWithManagerPackage).toHaveLength(13);
  });

  it("renders an authorized supplemental workspace route through the capability gate", async () => {
    const runtime = createMockBackendRuntime();
    const workspace = workspaceBackend();
    render(
      <AppProviders runtime={{ backend: { ...runtime.backend, agaDemoWorkspace: workspace }, buildProfile: "http", environmentLabel: "test", supplementalRouteElements: agaDemoWorkspaceRouteElements }}>
        <ScenarioProvider>
          <MemoryRouter initialEntries={["/department-manager/aga-demo-workspace"]}>
            <AppRouter />
          </MemoryRouter>
        </ScenarioProvider>
      </AppProviders>,
    );
    expect(await screen.findByTestId("aga-classification-workspace-page")).toBeInTheDocument();
  });

  it("renders the fixed Department Manager inspection-package route without a client identifier", async () => {
    const runtime = createMockBackendRuntime();
    const workspace = workspaceBackend();
    render(
      <AppProviders runtime={{ backend: { ...runtime.backend, agaDemoWorkspace: workspace }, buildProfile: "http", environmentLabel: "test", supplementalRouteElements: agaDemoWorkspaceRouteElementsWithManagerPackage }}>
        <ScenarioProvider>
          <MemoryRouter initialEntries={["/department-manager/aga-demo-workspace/inspection-package"]}>
            <AppRouter />
          </MemoryRouter>
        </ScenarioProvider>
      </AppProviders>,
    );
    expect(await screen.findByTestId("aga-demo-inspection-package-page")).toBeInTheDocument();
    expect(window.location.search).toBe("");
  });

  it("renders the explicit CAA Reviewer presentation route on the Lead-bound fixture session", async () => {
    const runtime = createMockBackendRuntime();
    const workspace = workspaceBackend();
    render(
      <AppProviders runtime={{ backend: { ...runtime.backend, agaDemoWorkspace: workspace }, buildProfile: "http", environmentLabel: "test", supplementalRouteElements: agaDemoWorkspaceRouteElementsWithManagerPackage }}>
        <ScenarioProvider>
          <MemoryRouter initialEntries={["/caa-reviewer/aga-demo-workspace/caps-evidence"]}>
            <AppRouter />
          </MemoryRouter>
        </ScenarioProvider>
      </AppProviders>,
    );
    expect(await screen.findByTestId("aga-demo-cap-evidence-page")).toBeInTheDocument();
  });

  it("does not mount supplemental elements when the runtime does not provide the HTTP extension", () => {
    const runtime = createMockBackendRuntime();
    render(
      <AppProviders runtime={{ backend: runtime.backend, buildProfile: "demo", environmentLabel: "test" }}>
        <MemoryRouter initialEntries={["/department-manager/aga-demo-workspace"]}><AppRouter /></MemoryRouter>
      </AppProviders>,
    );
    expect(screen.queryByTestId("aga-classification-workspace-page")).not.toBeInTheDocument();
  });

  it("mounts each fixed lifecycle suffix without putting an identifier in the route", async () => {
    const runtime = createMockBackendRuntime();
    const workspace = workspaceBackend();
    render(
      <AppProviders runtime={{ backend: { ...runtime.backend, agaDemoWorkspace: workspace }, buildProfile: "http", environmentLabel: "test", supplementalRouteElements: agaDemoWorkspaceRouteElements }}>
        <ScenarioProvider>
          <MemoryRouter initialEntries={["/lead-inspector/aga-demo-workspace/potential-findings"]}>
            <AppRouter />
          </MemoryRouter>
        </ScenarioProvider>
      </AppProviders>,
    );
    expect(await screen.findByTestId("aga-demo-potential-finding-page")).toBeInTheDocument();
    expect(window.location.search).toBe("");
  });

  it("limits the Auditee route to CAP and Evidence context", async () => {
    const runtime = createMockBackendRuntime();
    const workspace = workspaceBackend();
    render(
      <AppProviders runtime={{ backend: { ...runtime.backend, agaDemoWorkspace: workspace }, buildProfile: "http", environmentLabel: "test", supplementalRouteElements: agaDemoWorkspaceRouteElements }}>
        <ScenarioProvider>
          <MemoryRouter initialEntries={["/auditee/aga-demo-workspace"]}>
            <AppRouter />
          </MemoryRouter>
        </ScenarioProvider>
      </AppProviders>,
    );
    expect(await screen.findByTestId("aga-demo-cap-evidence-page")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "CAP and Evidence" })).toBeInTheDocument();
    expect(screen.queryByRole("link", { name: "Classification workspace" })).not.toBeInTheDocument();
    expect(screen.queryByRole("link", { name: "Inspection lifecycle" })).not.toBeInTheDocument();
    expect(screen.queryByRole("link", { name: "Potential Findings" })).not.toBeInTheDocument();
  });

  it("returns a stable local not-found view for an Auditee-restricted suffix without API calls", () => {
    const runtime = createMockBackendRuntime();
    const workspace = workspaceBackend();
    render(
      <AppProviders runtime={{ backend: { ...runtime.backend, agaDemoWorkspace: workspace }, buildProfile: "http", environmentLabel: "test", supplementalRouteElements: agaDemoWorkspaceRouteElements }}>
        <ScenarioProvider>
          <MemoryRouter initialEntries={["/auditee/aga-demo-workspace/potential-findings"]}>
            <AppRouter />
          </MemoryRouter>
        </ScenarioProvider>
      </AppProviders>,
    );
    expect(screen.getByTestId("aga-demo-workspace-not-found")).toBeInTheDocument();
    expect(workspace.capability).not.toHaveBeenCalled();
    expect(workspace.lifecycleQuery).not.toHaveBeenCalled();
  });

  it("returns a stable local not-found view for a deep unknown suffix without API calls", () => {
    const runtime = createMockBackendRuntime();
    const workspace = workspaceBackend();
    render(
      <AppProviders runtime={{ backend: { ...runtime.backend, agaDemoWorkspace: workspace }, buildProfile: "http", environmentLabel: "test", supplementalRouteElements: agaDemoWorkspaceRouteElementsWithManagerPackage }}>
        <ScenarioProvider>
          <MemoryRouter initialEntries={["/inspector/aga-demo-workspace/unknown/deep-link"]}>
            <AppRouter />
          </MemoryRouter>
        </ScenarioProvider>
      </AppProviders>,
    );
    expect(screen.getByTestId("aga-demo-workspace-not-found")).toBeInTheDocument();
    expect(workspace.capability).not.toHaveBeenCalled();
    expect(workspace.lifecycleQuery).not.toHaveBeenCalled();
  });
});
