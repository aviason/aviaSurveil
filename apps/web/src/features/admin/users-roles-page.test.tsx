// @vitest-environment jsdom
import "@testing-library/jest-dom/vitest";

import {
  cleanup,
  render,
  screen,
  waitFor,
  within,
} from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { afterEach, describe, expect, it, vi } from "vitest";

import { AppProviders } from "../../app/providers";
import type {
  AdminAccessDirectoryEntryView,
  AdminWorkspaceBackend,
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
    expectedMembershipRevision: input.expectedMembershipRevision,
    resultingMembershipRevision: 0,
    membershipId: null,
    reason: input.reason,
    effectiveAt: input.effectiveAt ?? null,
    providerFailureClass: null,
    providerAcknowledgedAt: null,
    attemptCount: 0,
    requestedBySubjectId: "USR-ADMIN-ADA",
    outboxMessageId: "outbox-user-lifecycle-001",
    failureReason: null,
    createdAt: "2026-07-24T08:00:00Z",
    updatedAt: "2026-07-24T08:00:01Z",
  };
}

function directoryEntry(
  overrides: Partial<AdminAccessDirectoryEntryView> = {},
): AdminAccessDirectoryEntryView {
  return {
    subjectId: "kc-existing-001",
    displayName: "Existing Inspector",
    roles: ["inspector"],
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
    ...overrides,
  };
}

function renderHttpPage(
  overrides: Partial<AdminWorkspaceBackend> = {},
) {
  const mockRuntime = createMockBackendRuntime();
  const demoAdmin = mockRuntime.backendForRole("admin");
  const httpBackend = {
    ...demoAdmin,
    mode: "http",
    adminWorkspace: {
      ...demoAdmin.adminWorkspace,
      ...overrides,
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

  return { httpBackend, user: userEvent.setup() };
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
        reason: "Approved existing lifecycle request.",
        expectedMembershipRevision: 0,
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
    await user.type(
      screen.getByLabelText("Provisioning reason"),
      "Approved new Inspector membership.",
    );
    await user.type(screen.getByLabelText("Provisioning organization"), "CAA");
    await user.selectOptions(screen.getByLabelText("Provisioning role"), "inspector");
    await user.click(screen.getByRole("button", { name: "Review provisioning" }));

    const provisionDialog = screen.getByRole("dialog", {
      name: "Confirm Provision for new.inspector@example.test",
    });
    expect(requestUserLifecycle).not.toHaveBeenCalled();
    await user.click(within(provisionDialog).getByRole("button", {
      name: "Confirm Provision",
    }));

    expect(requestUserLifecycle).toHaveBeenCalledWith(
      expect.objectContaining({
        action: "PROVISION",
        roles: ["inspector"],
        organizationId: "CAA",
        email: "new.inspector@example.test",
        displayName: "New Inspector",
        reason: "Approved new Inspector membership.",
        expectedMembershipRevision: 0,
      }),
    );
    expect(await screen.findByText(/Provisioning status: PENDING/)).toBeVisible();
    expect(screen.getByText(/TOTP MFA is optional and self-enrolled/i)).toBeVisible();

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
    const deactivateDialog = screen.getByRole("dialog", {
      name: "Confirm Deactivate for kc-existing-001",
    });
    expect(within(deactivateDialog).getByRole("button", {
      name: "Confirm Deactivate",
    })).toBeDisabled();
    await user.type(
      within(deactivateDialog).getByLabelText("Action reason"),
      "Approved retained deactivation.",
    );
    await user.click(within(deactivateDialog).getByRole("button", {
      name: "Confirm Deactivate",
    }));
    expect(requestUserLifecycle).toHaveBeenLastCalledWith(
      expect.objectContaining({
        subjectId: "kc-existing-001",
        action: "DEACTIVATE",
        roles: ["inspector"],
        organizationId: "CAA",
        reason: "Approved retained deactivation.",
        expectedMembershipRevision: 1,
      }),
    );

    await user.click(
      screen.getByRole("button", {
        name: "Force logout kc-existing-001",
      }),
    );
    const logoutDialog = screen.getByRole("dialog", {
      name: "Confirm Force logout for kc-existing-001",
    });
    await user.type(
      within(logoutDialog).getByLabelText("Action reason"),
      "Approved forced session revocation.",
    );
    await user.click(within(logoutDialog).getByRole("button", {
      name: "Confirm Force logout",
    }));
    expect(requestUserLifecycle).toHaveBeenLastCalledWith(
      expect.objectContaining({
        subjectId: "kc-existing-001",
        action: "FORCE_LOGOUT",
        roles: ["inspector"],
        organizationId: "CAA",
        reason: "Approved forced session revocation.",
        expectedMembershipRevision: 1,
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
      screen.getByLabelText("Provisioning reason"),
      "Approved scope validation proof.",
    );
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
      screen.getByRole("button", { name: "Review provisioning" }),
    );

    expect(
      await screen.findByText(
        "CAA roles require the exact CAA organization.",
      ),
    ).toBeVisible();
    expect(requestUserLifecycle).not.toHaveBeenCalled();
  });

  it("shows exact provider, membership, and session facts and explains unavailable actions", async () => {
    const mockRuntime = createMockBackendRuntime();
    const demoAdmin = mockRuntime.backendForRole("admin");
    const httpBackend = {
      ...demoAdmin,
      mode: "http",
      adminWorkspace: {
        ...demoAdmin.adminWorkspace,
        listAccessDirectory: vi.fn(async () => ({
          items: [directoryEntry({
            subjectId: "kc-deactivated-001",
            displayName: "Retained Inspector",
            accountStatus: "disabled",
            membershipState: "deactivated",
            membershipRevision: 4,
            membershipDrift: "in-sync",
            lastSuccessfulSessionAt: null,
          })],
          nextCursor: null,
        })),
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

    expect(await screen.findByText("Retained Inspector")).toBeVisible();
    expect(screen.getByText("Provider account")).toBeVisible();
    expect(screen.getByText("Application profile")).toBeVisible();
    expect(screen.getByText("Required actions")).toBeVisible();
    expect(screen.getByText("Desired membership")).toBeVisible();
    expect(screen.getByText("Authority alignment")).toBeVisible();
    expect(screen.getByText("Last successful session")).toBeVisible();
    expect(screen.getByText("Provider observed")).toBeVisible();
    expect(screen.getByText("No successful application session recorded")).toBeVisible();

    expect(screen.getByRole("button", {
      name: "Reactivate kc-deactivated-001",
    })).toBeEnabled();
    expect(screen.getByRole("button", {
      name: "Deactivate kc-deactivated-001 unavailable: Membership is already deactivated.",
    })).toBeDisabled();
    expect(screen.getByRole("button", {
      name: "Suspend kc-deactivated-001 unavailable: Deactivated membership must be reactivated before suspension.",
    })).toBeDisabled();
    expect(screen.getByRole("button", {
      name: "Resend invitation kc-deactivated-001 unavailable: Invitation resend is available only for invited memberships.",
    })).toBeDisabled();
  });

  it("requires a reasoned confirmation and sends every server-supported lifecycle action", async () => {
    const active = directoryEntry();
    const invited = directoryEntry({
      subjectId: "kc-invited-auditee",
      displayName: "Invited Auditee",
      roles: ["auditee"],
      organizationId: "ORG-FLY-NAMIBIA",
      mfaEnrolled: false,
      mfaState: "unenrolled",
      invitationState: "delivered",
      membershipId: "membership-invited-auditee",
      membershipState: "invited",
      membershipRevision: 2,
      lastSuccessfulSessionAt: null,
    });
    const suspended = directoryEntry({
      subjectId: "kc-suspended-inspector",
      displayName: "Suspended Inspector",
      accountStatus: "disabled",
      membershipId: "membership-suspended-inspector",
      membershipState: "suspended",
      membershipRevision: 3,
      lastSuccessfulSessionAt: null,
    });
    const requestUserLifecycle = vi.fn(
      async (input: RequestUserLifecycleInput) => lifecycle(input),
    );
    const { user } = renderHttpPage({
      listAccessDirectory: vi.fn(async () => ({
        items: [active, invited, suspended],
        nextCursor: null,
      })),
      requestUserLifecycle,
    });

    expect(await screen.findByText("Existing Inspector")).toBeVisible();

    async function confirmAction(
      label: string,
      subjectId: string,
      reason: string,
      configure?: (dialog: HTMLElement) => Promise<void>,
    ) {
      const opener = screen.getByRole("button", {
        name: `${label} ${subjectId}`,
      });
      await user.click(opener);
      const dialog = screen.getByRole("dialog", {
        name: `Confirm ${label} for ${subjectId}`,
      });
      const confirm = within(dialog).getByRole("button", {
        name: `Confirm ${label}`,
      });
      expect(confirm).toBeDisabled();
      if (configure) await configure(dialog);
      await user.type(within(dialog).getByLabelText("Action reason"), reason);
      await user.click(confirm);
      await waitFor(() => expect(requestUserLifecycle).toHaveBeenCalled());
    }

    const updateOpener = screen.getByRole("button", {
      name: "Update role kc-existing-001",
    });
    await user.click(updateOpener);
    const cancelled = screen.getByRole("dialog", {
      name: "Confirm Update role for kc-existing-001",
    });
    await user.click(within(cancelled).getByRole("button", { name: "Cancel" }));
    expect(updateOpener).toHaveFocus();

    await confirmAction(
      "Update role",
      "kc-existing-001",
      "Approved move to Manager authority.",
      async (dialog) => {
        await user.selectOptions(
          within(dialog).getByLabelText("New role"),
          "manager",
        );
      },
    );
    expect(requestUserLifecycle).toHaveBeenLastCalledWith(
      expect.objectContaining({
        subjectId: "kc-existing-001",
        action: "UPDATE_ROLES",
        roles: ["manager"],
        organizationId: "CAA",
        reason: "Approved move to Manager authority.",
        expectedMembershipRevision: 1,
      }),
    );

    await confirmAction(
      "Suspend",
      "kc-existing-001",
      "Approved temporary authority suspension.",
    );
    expect(requestUserLifecycle).toHaveBeenLastCalledWith(
      expect.objectContaining({
        subjectId: "kc-existing-001",
        action: "SUSPEND",
        roles: ["inspector"],
        organizationId: "CAA",
        reason: "Approved temporary authority suspension.",
        expectedMembershipRevision: 1,
      }),
    );

    await confirmAction(
      "Deactivate",
      "kc-existing-001",
      "Approved retained deactivation.",
    );
    expect(requestUserLifecycle).toHaveBeenLastCalledWith(
      expect.objectContaining({
        subjectId: "kc-existing-001",
        action: "DEACTIVATE",
        expectedMembershipRevision: 1,
      }),
    );

    await confirmAction(
      "Reset password",
      "kc-existing-001",
      "Approved account recovery.",
    );
    expect(requestUserLifecycle).toHaveBeenLastCalledWith(
      expect.objectContaining({
        action: "RESET_PASSWORD",
        reason: "Approved account recovery.",
      }),
    );

    await confirmAction(
      "Reset MFA",
      "kc-existing-001",
      "Approved lost-authenticator reset.",
    );
    expect(requestUserLifecycle).toHaveBeenLastCalledWith(
      expect.objectContaining({
        action: "RESET_MFA",
        reason: "Approved lost-authenticator reset.",
      }),
    );

    await confirmAction(
      "Force logout",
      "kc-existing-001",
      "Approved immediate session revocation.",
    );
    expect(requestUserLifecycle).toHaveBeenLastCalledWith(
      expect.objectContaining({
        action: "FORCE_LOGOUT",
        reason: "Approved immediate session revocation.",
      }),
    );

    await confirmAction(
      "Transfer organization",
      "kc-invited-auditee",
      "Approved future auditee transfer.",
      async (dialog) => {
        await user.type(
          within(dialog).getByLabelText("Destination organization"),
          "ORG-SKYCARGO",
        );
        await user.type(
          within(dialog).getByLabelText("Effective time"),
          "2026-08-01T09:30",
        );
      },
    );
    expect(requestUserLifecycle).toHaveBeenLastCalledWith(
      expect.objectContaining({
        subjectId: "kc-invited-auditee",
        action: "TRANSFER_ORGANIZATION",
        roles: ["auditee"],
        organizationId: "ORG-SKYCARGO",
        effectiveAt: expect.any(String),
        reason: "Approved future auditee transfer.",
        expectedMembershipRevision: 2,
      }),
    );

    await confirmAction(
      "Resend invitation",
      "kc-invited-auditee",
      "Approved bounded invitation resend.",
    );
    expect(requestUserLifecycle).toHaveBeenLastCalledWith(
      expect.objectContaining({
        subjectId: "kc-invited-auditee",
        action: "RESEND_INVITATION",
        reason: "Approved bounded invitation resend.",
        expectedMembershipRevision: 2,
      }),
    );

    await confirmAction(
      "Reactivate",
      "kc-suspended-inspector",
      "Approved retained membership reactivation.",
    );
    expect(requestUserLifecycle).toHaveBeenLastCalledWith(
      expect.objectContaining({
        subjectId: "kc-suspended-inspector",
        action: "REACTIVATE",
        reason: "Approved retained membership reactivation.",
        expectedMembershipRevision: 3,
      }),
    );
    expect(requestUserLifecycle).toHaveBeenCalledTimes(9);
  });

  it("keeps loading, empty, unavailable, stale, and retry directory states explicit", async () => {
    const neverLoads = new Promise<{
      items: AdminAccessDirectoryEntryView[];
      nextCursor: null;
    }>(() => undefined);
    renderHttpPage({
      listAccessDirectory: vi.fn(() => neverLoads),
    });
    expect(screen.getByRole("status")).toHaveTextContent(
      "Loading user directory",
    );
    cleanup();

    renderHttpPage({
      listAccessDirectory: vi.fn(async () => ({
        items: [],
        nextCursor: null,
      })),
    });
    expect(await screen.findByText(
      "No accounts match the current filters.",
    )).toBeVisible();
    cleanup();

    const listAccessDirectory = vi.fn()
      .mockRejectedValueOnce(new Error("Keycloak directory unavailable."))
      .mockResolvedValueOnce({
        items: [directoryEntry()],
        nextCursor: null,
      })
      .mockRejectedValueOnce(new Error("Provider observation timed out."));
    const { user } = renderHttpPage({ listAccessDirectory });
    const unavailable = await screen.findByRole("alert");
    expect(unavailable).toHaveTextContent("Keycloak directory unavailable.");
    await user.click(screen.getByRole("button", {
      name: "Retry user directory",
    }));
    expect(await screen.findByText("Existing Inspector")).toBeVisible();

    await user.click(screen.getByRole("button", {
      name: "Refresh user directory",
    }));
    expect(await screen.findByRole("status")).toHaveTextContent(
      "Showing the last successful directory because Provider observation timed out.",
    );
    expect(screen.getByText("Existing Inspector")).toBeVisible();
    expect(screen.getByRole("button", {
      name: "Retry user directory",
    })).toBeEnabled();
  });

  it("makes retryable lifecycle failure, success reconciliation, and command errors stable", async () => {
    const requestUserLifecycle = vi.fn()
      .mockImplementationOnce(async (input: RequestUserLifecycleInput) => ({
        ...lifecycle(input, "FAILED_RETRYABLE"),
        providerFailureClass: "RETRYABLE" as const,
        attemptCount: 1,
        failureReason: "Keycloak write timed out.",
      }))
      .mockRejectedValueOnce(new Error("Lifecycle command was rejected."));
    const getUserLifecycleRequest = vi.fn(
      async () => lifecycle({
        idempotencyKey: "deactivate:kc-existing-001",
        subjectId: "kc-existing-001",
        action: "DEACTIVATE",
        roles: ["inspector"],
        organizationId: "CAA",
        reason: "Approved retained deactivation.",
        expectedMembershipRevision: 1,
      }, "SUCCEEDED"),
    );
    const listAccessDirectory = vi.fn(async () => ({
      items: [directoryEntry()],
      nextCursor: null,
    }));
    const { user } = renderHttpPage({
      listAccessDirectory,
      requestUserLifecycle,
      getUserLifecycleRequest,
    });

    expect(await screen.findByText("Existing Inspector")).toBeVisible();
    await user.click(screen.getByRole("button", {
      name: "Deactivate kc-existing-001",
    }));
    let dialog = screen.getByRole("dialog", {
      name: "Confirm Deactivate for kc-existing-001",
    });
    await user.type(
      within(dialog).getByLabelText("Action reason"),
      "Approved retained deactivation.",
    );
    await user.click(within(dialog).getByRole("button", {
      name: "Confirm Deactivate",
    }));

    const retryable = await screen.findByRole("status", {
      name: "Lifecycle request status",
    });
    expect(retryable).toHaveTextContent("Failed — retry available");
    expect(retryable).toHaveTextContent("Keycloak write timed out.");
    await user.click(within(retryable).getByRole("button", {
      name: "Retry lifecycle status",
    }));
    expect(getUserLifecycleRequest).toHaveBeenCalledWith({
      requestId: "user-lifecycle-001",
    });
    expect(await screen.findByText("Succeeded")).toBeVisible();
    await waitFor(() => expect(listAccessDirectory).toHaveBeenCalledTimes(2));

    await user.click(screen.getByRole("button", {
      name: "Force logout kc-existing-001",
    }));
    dialog = screen.getByRole("dialog", {
      name: "Confirm Force logout for kc-existing-001",
    });
    await user.type(
      within(dialog).getByLabelText("Action reason"),
      "Approved immediate session revocation.",
    );
    await user.click(within(dialog).getByRole("button", {
      name: "Confirm Force logout",
    }));

    const commandError = await screen.findByRole("alert");
    expect(commandError).toHaveTextContent("Lifecycle command was rejected.");
    expect(commandError).toHaveFocus();
    expect(screen.getByRole("button", {
      name: "Force logout kc-existing-001",
    })).toBeEnabled();
  });

});
