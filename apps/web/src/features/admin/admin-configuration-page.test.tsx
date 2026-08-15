// @vitest-environment jsdom
import "@testing-library/jest-dom/vitest";

import { cleanup, render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { afterEach, describe, expect, it, vi } from "vitest";

import { AppProviders } from "../../app/providers";
import { AppRouter } from "../../app/router";
import type { SessionClient } from "../../auth/session-client";
import { SessionProvider } from "../../auth/session-provider";
import { createMockBackendRuntime } from "../../mock/create-mock-backend";
import { AdminConfigurationPage } from "./admin-configuration-page";

afterEach(() => {
  cleanup();
  vi.restoreAllMocks();
});

type MockRuntime = ReturnType<typeof createMockBackendRuntime>;

function renderPage(runtime: MockRuntime) {
  return render(
    <AppProviders runtime={{
      backend: runtime.backend,
      backendForRole: runtime.backendForRole,
      buildProfile: "demo",
      environmentLabel: "test",
      identityMode: "demo-role-switch",
      subjectId: "USR-ADMIN-ADA",
    }}>
      <MemoryRouter initialEntries={["/admin/templates"]}>
        <AdminConfigurationPage />
      </MemoryRouter>
    </AppProviders>,
  );
}

describe("AdminConfigurationPage", () => {
  it("lists published versions and direct-loads the exact immutable Backend detail", async () => {
    const runtime = createMockBackendRuntime();
    const admin = runtime.backendForRole("admin");
    const list = vi.spyOn(admin.configuration, "listChecklistTemplateVersions");
    const get = vi.spyOn(admin.configuration, "getChecklistTemplateVersion");
    renderPage(runtime);

    expect(await screen.findByRole("heading", { name: "Checklist Templates" })).toBeVisible();
    expect(list).toHaveBeenCalledWith({ limit: 100 });
    await userEvent.setup().click(screen.getByRole("button", { name: "Preview CTV-CABIN-1" }));
    expect(get).toHaveBeenCalledWith({ templateVersionId: "CTV-CABIN-1" });

    const summary = await screen.findByRole("region", { name: "Published template summary" });
    expect(within(summary).getByText("CTV-CABIN-1")).toBeVisible();
    expect(within(summary).getByTestId("template-version")).toHaveTextContent("Version 1");
    expect(within(summary).getByText(/2026-06-15/)).toBeVisible();
    expect(within(summary).getByText("Published")).toBeVisible();

    const register = screen.getByRole("table", { name: "Published checklist questions" });
    expect(within(register).getAllByRole("row")).toHaveLength(7);
    expect(within(register).getByText("Are galley restraints and stowage areas serviceable and secure?")).toBeVisible();
    expect(within(register).getByText("Configured Cabin Inspection reference — GALLEY")).toBeVisible();
    expect(within(register).getAllByText("Inspector observation and required exception comment").length).toBeGreaterThan(0);
    expect(within(register).getAllByText(/Allowed answers: Compliant · Non-Compliant · Observation · Not Applicable · Not Checked/).length).toBeGreaterThan(0);
    expect(within(register).getAllByText("Comment required for Non-Compliant · Observation").length).toBeGreaterThan(0);
  });

  it("returns to the declared Template List route without maintaining a hidden substitute register", async () => {
    const runtime = createMockBackendRuntime();
    renderPage(runtime);

    expect(await screen.findByRole("heading", { name: "Checklist Templates" })).toBeVisible();
    expect(screen.getByRole("button", { name: "Preview CTV-CABIN-1" })).toBeVisible();
    expect(screen.queryByRole("link", { name: "Back to templates" })).toBeNull();
    expect(screen.queryByRole("button", { name: /edit|publish|delete|add user|add role|configure/i })).toBeNull();
    expect(document.body).not.toHaveTextContent(/draft version|save changes/i);
  });

  it("fails a non-Admin direct URL before any configuration fetch", async () => {
    const runtime = createMockBackendRuntime();
    const list = vi.spyOn(runtime.backendForRole("admin").configuration, "listChecklistTemplateVersions");
    const sessionClient: SessionClient = {
      get: vi.fn().mockResolvedValue({
        subjectId: "USR-MANAGER-NORA",
        displayName: "Nora Manager",
        organizationId: "CAA",
        roles: ["manager"],
      }),
      login: vi.fn(),
      logout: vi.fn(),
      csrfToken: vi.fn(() => "csrf"),
    };

    render(
      <AppProviders runtime={{
        backend: runtime.backend,
        backendForRole: runtime.backendForRole,
        buildProfile: "http",
        environmentLabel: "test",
        identityMode: "oidc-session",
        subjectId: "USR-MANAGER-NORA",
      }}>
        <SessionProvider client={sessionClient} identityMode="oidc-session" initialRole="manager">
          <MemoryRouter initialEntries={["/admin/templates"]}><AppRouter /></MemoryRouter>
        </SessionProvider>
      </AppProviders>,
    );

    expect(await screen.findByTestId("route-forbidden")).toBeVisible();
    expect(list).not.toHaveBeenCalled();
  });

  it("renders an explicit empty state without requesting a missing detail", async () => {
    const runtime = createMockBackendRuntime();
    const admin = runtime.backendForRole("admin");
    vi.spyOn(admin.configuration, "listChecklistTemplateVersions").mockResolvedValue({ items: [], nextCursor: null });
    const get = vi.spyOn(admin.configuration, "getChecklistTemplateVersion");
    renderPage(runtime);

    expect(await screen.findByRole("heading", { name: "Checklist Templates" })).toBeVisible();
    expect(screen.getByText("No published checklist template versions are available.")).toBeVisible();
    expect(get).not.toHaveBeenCalled();
  });

  it("fails closed with a visible error when a listed detail is malformed or unavailable", async () => {
    const runtime = createMockBackendRuntime();
    vi.spyOn(runtime.backendForRole("admin").configuration, "getChecklistTemplateVersion")
      .mockRejectedValue(new Error("Checklist template detail is malformed."));
    renderPage(runtime);

    await userEvent.setup().click(await screen.findByRole("button", { name: "Preview CTV-CABIN-1" }));
    expect(await screen.findByRole("alert")).toHaveTextContent("Checklist template detail is malformed.");
    expect(screen.queryByRole("table", { name: "Published checklist questions" })).toBeNull();
  });
});
