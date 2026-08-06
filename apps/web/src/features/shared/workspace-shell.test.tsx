// @vitest-environment jsdom
import "@testing-library/jest-dom/vitest";

import { cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { afterEach, describe, expect, it, vi } from "vitest";

import { AppProviders } from "../../app/providers";
import { SessionProvider, type SessionClient } from "../../auth/session-provider";
import { createMockBackendRuntime } from "../../mock/create-mock-backend";
import { WorkspaceShell } from "./workspace-shell";

afterEach(() => cleanup());

describe("WorkspaceShell", () => {
  it("uses the visible OIDC logout control to revoke the local session", async () => {
    const runtime = createMockBackendRuntime();
    const beforeSubjectChange = vi.fn().mockResolvedValue(undefined);
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
      <AppProviders runtime={{
        backend: runtime.backend,
        backendForRole: runtime.backendForRole,
        buildProfile: "http",
        environmentLabel: "Test",
        identityMode: "oidc-session",
        beforeSubjectChange,
      }}>
        <SessionProvider client={client} identityMode="oidc-session">
          <MemoryRouter>
            <WorkspaceShell roleLabel="CAA Inspector" routeLabel="My Assignments">
              <p>Authorized content</p>
            </WorkspaceShell>
          </MemoryRouter>
        </SessionProvider>
      </AppProviders>,
    );

    await waitFor(() => expect(screen.getByTestId("application-shell")).toHaveAttribute("data-active-role", "inspector"));
    await userEvent.click(screen.getByRole("button", { name: "Logout" }));

    await waitFor(() => expect(client.logout).toHaveBeenCalledTimes(1));
    expect(beforeSubjectChange).toHaveBeenCalledWith("LOGOUT");
  });
});
