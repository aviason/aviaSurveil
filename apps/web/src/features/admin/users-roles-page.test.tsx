// @vitest-environment jsdom
import "@testing-library/jest-dom/vitest";

import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { afterEach, describe, expect, it, vi } from "vitest";

import { AppProviders } from "../../app/providers";
import type {
  Backend,
  RequestUserLifecycleInput,
  UserLifecycleRequestView,
} from "../../backend/backend";
import { createMockBackendRuntime } from "../../mock/create-mock-backend";
import { UsersRolesPage } from "./users-roles-page";

afterEach(() => cleanup());

function lifecycle(
  input: RequestUserLifecycleInput,
  status: UserLifecycleRequestView["status"] = "PENDING",
): UserLifecycleRequestView {
  return {
    id: "user-lifecycle-001",
    subjectId: input.subjectId ?? null,
    action: input.action,
    roles: [...input.roles],
    organizationId: input.organizationId,
    email: input.email ?? null,
    displayName: input.displayName ?? null,
    status,
    idempotencyKey: input.idempotencyKey,
    requestedBySubjectId: "USR-ADMIN-ADA",
    outboxMessageId: "outbox-user-lifecycle-001",
    failureReason: null,
    createdAt: "2026-07-24T08:00:00Z",
    updatedAt: "2026-07-24T08:00:01Z",
  };
}

describe("UsersRolesPage production-like identity controls", () => {
  it("provisions with approved role/org data, presents reconciliation, and requests deactivation", async () => {
    const mockRuntime = createMockBackendRuntime();
    const demoAdmin = mockRuntime.backendForRole("admin");
    const requestUserLifecycle = vi.fn(
      async (input: RequestUserLifecycleInput) => lifecycle(input),
    );
    const getUserLifecycleRequest = vi.fn(async () => ({
      ...lifecycle({
        idempotencyKey: "provision:new.inspector@example.test",
        action: "PROVISION",
        roles: ["inspector"],
        organizationId: "CAA",
        email: "new.inspector@example.test",
        displayName: "New Inspector",
      }, "SUCCEEDED"),
      subjectId: "kc-subject-001",
    }));
    const httpBackend = {
      ...demoAdmin,
      mode: "http",
      adminWorkspace: {
        ...demoAdmin.adminWorkspace,
        listAccessDirectory: vi.fn(async () => ({
          items: [{
            subjectId: "kc-existing-001",
            displayName: "Existing Inspector",
            roles: ["inspector" as const],
            organizationId: "CAA",
            email: "existing.inspector@example.test",
            mfaEnrolled: true,
            mfaState: "enrolled",
            requiredActions: [],
            invitationState: "required-actions-complete",
            accountStatus: "enabled",
            applicationProfileState: "linked",
            membershipId: "membership-existing-001",
            membershipState: "active",
            membershipRevision: 1,
            membershipDrift: "in-sync",
            lastSuccessfulSessionAt: "2026-07-21T12:00:00Z",
            providerObservedAt: "2026-07-21T12:00:00Z",
          }],
          nextCursor: null,
        })),
        requestUserLifecycle,
        getUserLifecycleRequest,
      },
    } satisfies Backend;

    render(
      <AppProviders runtime={{
        backend: httpBackend,
        buildProfile: "http",
        environmentLabel: "Local production-like",
        identityMode: "oidc-session",
        subjectId: "USR-ADMIN-ADA",
      }}>
        <MemoryRouter>
          <UsersRolesPage />
        </MemoryRouter>
      </AppProviders>,
    );

    const user = userEvent.setup();
    expect(await screen.findByText("Existing Inspector")).toBeVisible();
    await user.click(screen.getByRole("button", { name: "Create user" }));
    await user.type(screen.getByLabelText("Provisioning email"), "new.inspector@example.test");
    await user.type(screen.getByLabelText("Provisioning display name"), "New Inspector");
    await user.type(screen.getByLabelText("Provisioning organization"), "CAA");
    await user.selectOptions(screen.getByLabelText("Provisioning role"), "inspector");
    await user.click(screen.getByRole("button", { name: "Submit provisioning" }));

    expect(requestUserLifecycle).toHaveBeenCalledWith(
      expect.objectContaining({
        action: "PROVISION",
        roles: ["inspector"],
        organizationId: "CAA",
        email: "new.inspector@example.test",
        displayName: "New Inspector",
      }),
    );
    expect(await screen.findByText(/Provisioning status: PENDING/)).toBeVisible();
    expect(screen.getByText(/first login requires TOTP MFA enrollment/i)).toBeVisible();

    await user.click(screen.getByRole("button", { name: "Refresh provisioning status" }));
    expect(getUserLifecycleRequest).toHaveBeenCalledWith({
      requestId: "user-lifecycle-001",
    });
    expect(await screen.findByText(/Provisioning status: SUCCEEDED/)).toBeVisible();

    const deactivate = screen.getByRole("button", {
      name: "Deactivate kc-existing-001",
    });
    expect(deactivate).toHaveAttribute(
      "aria-controls",
      "user-lifecycle-status",
    );
    await user.click(deactivate);
    expect(requestUserLifecycle).toHaveBeenLastCalledWith(
      expect.objectContaining({
        subjectId: "kc-existing-001",
        action: "SUSPEND",
        roles: ["inspector"],
        organizationId: "CAA",
      }),
    );
  });

  it("rejects an incompatible CAA role and organization before transport", async () => {
    const mockRuntime = createMockBackendRuntime();
    const demoAdmin = mockRuntime.backendForRole("admin");
    const requestUserLifecycle = vi.fn(
      async (input: RequestUserLifecycleInput) => lifecycle(input),
    );
    const httpBackend = {
      ...demoAdmin,
      mode: "http",
      adminWorkspace: {
        ...demoAdmin.adminWorkspace,
        listAccessDirectory: vi.fn(async () => ({
          items: [],
          nextCursor: null,
        })),
        requestUserLifecycle,
      },
    } satisfies Backend;

    render(
      <AppProviders runtime={{
        backend: httpBackend,
        buildProfile: "http",
        environmentLabel: "Local production-like",
        identityMode: "oidc-session",
        subjectId: "USR-ADMIN-ADA",
      }}>
        <MemoryRouter>
          <UsersRolesPage />
        </MemoryRouter>
      </AppProviders>,
    );

    const user = userEvent.setup();
    await user.click(screen.getByRole("button", { name: "Create user" }));
    await user.type(
      screen.getByLabelText("Provisioning email"),
      "wrong-scope@example.test",
    );
    await user.type(
      screen.getByLabelText("Provisioning display name"),
      "Wrong Scope",
    );
    await user.type(
      screen.getByLabelText("Provisioning organization"),
      "ORG-FLY-NAMIBIA",
    );
    await user.click(
      screen.getByRole("button", { name: "Submit provisioning" }),
    );

    expect(
      await screen.findByText(
        "CAA roles require the exact CAA organization.",
      ),
    ).toBeVisible();
    expect(requestUserLifecycle).not.toHaveBeenCalled();
  });
});
