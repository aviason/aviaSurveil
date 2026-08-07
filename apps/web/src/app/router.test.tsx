// @vitest-environment jsdom
import "@testing-library/jest-dom/vitest";

import { cleanup, render, screen, waitFor } from "@testing-library/react";
import { StrictMode } from "react";
import { MemoryRouter } from "react-router-dom";
import { afterEach, describe, expect, it, vi } from "vitest";

import { AppProviders } from "./providers";
import { agaDemoWorkspaceRouteElements } from "./aga-demo-workspace-routes";
import { ScenarioProvider } from "./scenario-context";
import { AppRouter, createRoleEntryPath, ROLE_ENTRIES } from "./router";
import { SessionProvider, type SessionClient } from "../auth/session-provider";
import type { AGADemoWorkspaceBackend } from "../backend/aga-demo-workspace";
import { createHttpBackend } from "../backend/http-backend";
import { createMockBackendRuntime } from "../mock/create-mock-backend";
import { seedVisualRuntimeForPath } from "../mock/seed-visual-runtime";

afterEach(() => cleanup());

describe("authorized role-entry inventory", () => {
  it("matches the eight verified legacy entry routes in display order", () => {
    expect(ROLE_ENTRIES.map(({ role, route }) => `${role}:${route}`)).toEqual([
      "inspector:inspector-assignments",
      "leadInspector:lead-review",
      "manager:dashboard",
      "gm:gm-dashboard",
      "finance:finance-review",
      "executiveDirector:executive-dashboard",
      "auditee:service-provider-cap",
      "admin:templates",
    ]);
  });

  it("creates stable URL paths without importing legacy globals", () => {
    expect(createRoleEntryPath("leadInspector")).toBe("/lead-inspector/lead-review");
    expect(createRoleEntryPath("auditee")).toBe("/auditee/service-provider-cap");
  });

  it("renders the actionable Finance workspace instead of a route-name placeholder", async () => {
    const runtime = createMockBackendRuntime();
    render(
      <AppProviders
        runtime={{
          backend: runtime.backend,
          backendForRole: runtime.backendForRole,
          buildProfile: "demo",
          environmentLabel: "Test",
        }}
      >
        <MemoryRouter initialEntries={["/finance/finance-review"]}>
          <AppRouter />
        </MemoryRouter>
      </AppProviders>,
    );
    expect(await screen.findByRole("heading", { name: "Finance Review" })).toBeInTheDocument();
    expect(
      await screen.findByRole("heading", { name: "2026 Cabin Surveillance — Fly Namibia" }),
    ).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Approve Budget" })).toBeInTheDocument();
    expect(screen.queryByText(/candidate React entry route/i)).not.toBeInTheDocument();
  });

  it("renders the presentational role-selection shell on the root route", () => {
    const runtime = createMockBackendRuntime();
    render(
      <AppProviders
        runtime={{
          backend: runtime.backend,
          backendForRole: runtime.backendForRole,
          buildProfile: "demo",
          environmentLabel: "Test",
        }}
      >
        <MemoryRouter initialEntries={["/"]}>
          <AppRouter />
        </MemoryRouter>
      </AppProviders>,
    );

    expect(screen.getByTestId("role-select-panel")).toBeInTheDocument();
    expect(screen.getAllByTestId("role-card-icon")).toHaveLength(8);
  });

  it("lands an authenticated local-preprod Inspector on the canonical assignments surface", async () => {
    const runtime = createMockBackendRuntime();
    const workspace: AGADemoWorkspaceBackend = {
      capability: vi.fn().mockResolvedValue({ available: true, projection: "INSPECTOR_ASSIGNED", classificationEnabled: false, recommendationEnabled: false, lifecycleEnabled: true, resetEnabled: false }),
      classificationQuery: vi.fn(),
      classificationCommand: vi.fn(),
      recommendationCommand: vi.fn(),
      lifecycleQuery: vi.fn(),
      lifecycleCommand: vi.fn(),
      adminCommand: vi.fn(),
    };
    const assignmentsList = vi.spyOn(runtime.backend.assignments, "list");
    const client: SessionClient = {
      get: vi.fn().mockResolvedValue({
        subjectId: "154ec5ac-6f97-4f55-916f-d2f142fc6211",
        displayName: "Local Inspector",
        organizationId: "CAA",
        roles: ["inspector"],
      }),
      login: vi.fn(),
      logout: vi.fn().mockResolvedValue(undefined),
      csrfToken: vi.fn(() => "csrf"),
    };
    render(
      <StrictMode>
        <AppProviders
          runtime={{
            backend: { ...runtime.backend, agaDemoWorkspace: workspace },
            backendForRole: runtime.backendForRole,
            buildProfile: "http",
            environmentLabel: "Test",
            identityMode: "oidc-session",
            beforeSubjectChange: vi.fn().mockResolvedValue(undefined),
            supplementalRouteElements: agaDemoWorkspaceRouteElements,
            agaDemoWorkspaceSurfaceEnabled: true,
          }}
        >
          <SessionProvider client={client} identityMode="oidc-session">
            <ScenarioProvider>
              <MemoryRouter initialEntries={["/"]}>
                <AppRouter />
              </MemoryRouter>
            </ScenarioProvider>
          </SessionProvider>
        </AppProviders>
      </StrictMode>,
    );

    await waitFor(() => expect(assignmentsList).toHaveBeenCalled());
    expect(client.logout).not.toHaveBeenCalled();
    expect(workspace.capability).not.toHaveBeenCalled();
  });

  it("does not fall back to the removed AGA donor from the canonical Inspector route", async () => {
    const runtime = createMockBackendRuntime();
    const workspace: AGADemoWorkspaceBackend = {
      capability: vi.fn().mockResolvedValue({ available: true, projection: "INSPECTOR_ASSIGNED", classificationEnabled: false, recommendationEnabled: false, lifecycleEnabled: true, resetEnabled: false }),
      classificationQuery: vi.fn(),
      classificationCommand: vi.fn(),
      recommendationCommand: vi.fn(),
      lifecycleQuery: vi.fn().mockResolvedValue({ operation: "GET_CURRENT_INSPECTION", lifecycleAvailable: false }),
      lifecycleCommand: vi.fn(),
      adminCommand: vi.fn(),
    };
    const assignmentsList = vi.spyOn(runtime.backend.assignments, "list");
    render(
      <AppProviders
        runtime={{
          backend: { ...runtime.backend, agaDemoWorkspace: workspace },
          backendForRole: runtime.backendForRole,
          buildProfile: "http",
          environmentLabel: "Test",
          supplementalRouteElements: agaDemoWorkspaceRouteElements,
          agaDemoWorkspaceSurfaceEnabled: true,
        }}
      >
        <ScenarioProvider>
          <MemoryRouter initialEntries={["/inspector/inspector-assignments"]}>
            <AppRouter />
          </MemoryRouter>
        </ScenarioProvider>
      </AppProviders>,
    );

    expect(await screen.findByRole("heading", { name: "My Assignments" })).toBeInTheDocument();
    expect(assignmentsList).toHaveBeenCalled();
    expect(workspace.capability).not.toHaveBeenCalled();
  });

  it("redirects an undeclared path to role selection without rendering a placeholder", async () => {
    const runtime = createMockBackendRuntime();
    render(
      <AppProviders runtime={{ backend: runtime.backend, backendForRole: runtime.backendForRole, buildProfile: "demo", environmentLabel: "Test" }}>
        <MemoryRouter initialEntries={["/scope-leak"]}><AppRouter /></MemoryRouter>
      </AppProviders>,
    );

    expect(await screen.findByTestId("role-select-panel")).toBeInTheDocument();
    expect(screen.queryByText(/placeholder|coming soon|candidate React entry route/i)).not.toBeInTheDocument();
  });

  it("direct-loads a formerly blocked Finding route in the HTTP profile", async () => {
    const runtime = createMockBackendRuntime();
    render(
      <AppProviders runtime={{ backend: runtime.backend, backendForRole: runtime.backendForRole, buildProfile: "http", environmentLabel: "Test" }}>
        <ScenarioProvider><MemoryRouter initialEntries={["/inspector/findings"]}><AppRouter /></MemoryRouter></ScenarioProvider>
      </AppProviders>,
    );

    expect(await screen.findByRole("heading", { name: "Findings" })).toBeInTheDocument();
    expect(screen.queryByRole("alert", { name: "Unavailable HTTP capability" })).not.toBeInTheDocument();
  });

  it("direct-loads the Admin inspection-package route in the HTTP profile", async () => {
    const fetchImplementation = vi.fn<typeof fetch>().mockResolvedValue(
      new Response(JSON.stringify({
        id: "PKG-CAB-2026-001",
        auditId: "AUD-2026-001",
        organizationId: "ORG-FLY-NAMIBIA",
        organizationName: "Fly Namibia",
        questionIds: ["CAB-EMEQ-PBE-001"],
        configuredReferences: ["Configured EM EQ / PBE"],
        expectedEvidence: ["PBE record"],
        riskFocus: ["Emergency equipment serviceability"],
      }), {
        headers: { "content-type": "application/json" },
      }),
    );
    const backend = createHttpBackend(
      { apiBaseUrl: "/", environmentLabel: "Test" },
      { fetchImplementation },
    );
    expect(backend.adminWorkspace).toBeDefined();

    render(
      <AppProviders runtime={{ backend, backendForRole: () => backend, buildProfile: "http", environmentLabel: "Test" }}>
        <MemoryRouter initialEntries={["/admin/inspection-package-builder"]}><AppRouter /></MemoryRouter>
      </AppProviders>,
    );

    expect(await screen.findByRole("heading", { name: "Inspection Package Builder" })).toBeInTheDocument();
    expect(await screen.findByText("PKG-CAB-2026-001")).toBeInTheDocument();
    expect(screen.queryByRole("alert", { name: "Unavailable HTTP capability" })).not.toBeInTheDocument();
    expect(fetchImplementation).toHaveBeenCalledWith(
      "/v1/admin/inspection-packages/PKG-CAB-2026-001",
      expect.objectContaining({ credentials: "same-origin" }),
    );
  });

  it("direct-loads the Admin organization-detail route in the HTTP profile", async () => {
    const fetchImplementation = vi.fn<typeof fetch>().mockResolvedValue(
      new Response(JSON.stringify({
        id: "ORG-FLY-NAMIBIA",
        legalName: "Fly Namibia",
        organizationType: "OPERATOR",
        status: "ACTIVE",
        scope: "CAA oversight",
        detailAvailable: true,
        disabledReason: null,
      }), {
        headers: { "content-type": "application/json" },
      }),
    );
    const backend = createHttpBackend(
      { apiBaseUrl: "/", environmentLabel: "Test" },
      { fetchImplementation },
    );

    render(
      <AppProviders runtime={{ backend, backendForRole: () => backend, buildProfile: "http", environmentLabel: "Test" }}>
        <MemoryRouter initialEntries={["/admin/organization-master-data/ORG-FLY-NAMIBIA"]}><AppRouter /></MemoryRouter>
      </AppProviders>,
    );

    expect(await screen.findByRole("heading", { name: "Organization Detail" })).toBeInTheDocument();
    expect(await screen.findAllByText("Fly Namibia")).toHaveLength(2);
    expect(screen.queryByRole("alert", { name: "Unavailable HTTP capability" })).not.toBeInTheDocument();
    expect(fetchImplementation).toHaveBeenCalledWith(
      "/v1/admin/organizations/ORG-FLY-NAMIBIA",
      expect.objectContaining({ credentials: "same-origin" }),
    );
  });

  it("direct-loads ui-audit-044 through the Department Manager shell and backend", async () => {
    const runtime = createMockBackendRuntime();
    await seedVisualRuntimeForPath(runtime, "/department-manager/evidence/FND-CAB-2026-001");
    const managerListVersions = vi.spyOn(runtime.backendForRole("manager").evidence, "listVersions");
    render(
      <AppProviders runtime={{ backend: runtime.backend, backendForRole: runtime.backendForRole, buildProfile: "demo", environmentLabel: "Test", subjectId: "USR-MANAGER-NORA" }}>
        <ScenarioProvider><MemoryRouter initialEntries={["/department-manager/evidence/FND-CAB-2026-001"]}><AppRouter /></MemoryRouter></ScenarioProvider>
      </AppProviders>,
    );

    expect(await screen.findByTestId("evidence-review-target")).toBeVisible();
    expect(screen.getByTestId("application-shell")).toHaveAttribute("data-active-role", "manager");
    expect(screen.getByRole("link", { name: "Findings Review" })).toHaveAttribute("aria-current", "page");
    expect(screen.queryByRole("link", { name: "Evidence Review" })).not.toBeInTheDocument();
    expect(document.querySelector(".evidence-root-page")).not.toHaveTextContent("Lead Inspector workspace");
    expect(managerListVersions).toHaveBeenCalledWith({ findingId: "FND-CAB-2026-001" });
  });

  it("keeps a Lead subject direct-load of ui-audit-044 on the Manager route authority", async () => {
    const runtime = createMockBackendRuntime();
    await seedVisualRuntimeForPath(runtime, "/department-manager/evidence/FND-CAB-2026-001");
    const managerListVersions = vi.spyOn(runtime.backendForRole("manager").evidence, "listVersions");
    const leadListVersions = vi.spyOn(runtime.backendForRole("leadInspector").evidence, "listVersions");
    render(
      <AppProviders runtime={{ backend: runtime.backend, backendForRole: runtime.backendForRole, buildProfile: "demo", environmentLabel: "Test", subjectId: "USR-LEAD-CANER" }}>
        <ScenarioProvider><MemoryRouter initialEntries={["/department-manager/evidence/FND-CAB-2026-001"]}><AppRouter /></MemoryRouter></ScenarioProvider>
      </AppProviders>,
    );

    expect(await screen.findByTestId("evidence-review-target")).toBeVisible();
    expect(screen.getByTestId("application-shell")).toHaveAttribute("data-active-role", "manager");
    expect(managerListVersions).toHaveBeenCalledWith({ findingId: "FND-CAB-2026-001" });
    expect(leadListVersions).not.toHaveBeenCalled();
  });
});
