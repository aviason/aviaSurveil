// @vitest-environment jsdom
import "@testing-library/jest-dom/vitest";

import { cleanup, render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { afterEach, describe, expect, it, vi } from "vitest";

import { AppProviders } from "../../app/providers";
import { AppRouter } from "../../app/router";
import { SessionClientError, type SessionClient } from "../../auth/session-client";
import { SessionProvider } from "../../auth/session-provider";
import { createMockBackendRuntime } from "../../mock/create-mock-backend";
import { planningItemLabel, recordReference } from "../shared/record-presentation";
import { FinanceReviewPage } from "./planning-workspaces";

afterEach(() => {
  cleanup();
  vi.restoreAllMocks();
});

type MockRuntime = ReturnType<typeof createMockBackendRuntime>;
const PLAN_ID = "PLAN-2026-CAB-001";
const PLAN_TITLE = "2026 Cabin Surveillance — Fly Namibia";
const PLAN_LABEL = planningItemLabel(PLAN_TITLE, PLAN_ID);
const PLAN_REFERENCE = recordReference("Plan", PLAN_ID);

function renderPage(runtime: MockRuntime, identityMode: "demo-role-switch" | "oidc-session" = "demo-role-switch") {
  return render(
    <AppProviders
      runtime={{
        backend: runtime.backend,
        backendForRole: runtime.backendForRole,
        buildProfile: identityMode === "oidc-session" ? "http" : "demo",
        environmentLabel: "test",
        identityMode,
        subjectId: "USR-FINANCE-DERYA",
      }}
    >
      <MemoryRouter initialEntries={["/finance/finance-review"]}>
        <FinanceReviewPage />
      </MemoryRouter>
    </AppProviders>,
  );
}

describe("FinanceReviewPage", () => {
  it("direct-loads the source-faithful Finance queue, dossier, approval rail, and authority boundary", async () => {
    const runtime = createMockBackendRuntime();
    const list = vi.spyOn(runtime.backendForRole("finance").planning, "list");
    renderPage(runtime);

    expect(await screen.findByRole("heading", { name: "Finance Review" })).toBeVisible();
    expect(list).toHaveBeenCalledWith({ limit: 100 });
    expect(screen.getByText("Budget approval before GM Review")).toBeVisible();
    const summary = screen.getByRole("region", { name: "Finance review summary" });
    expect(within(summary).getAllByRole("article")).toHaveLength(3);
    expect(within(summary).getByText("Approval path")).toBeVisible();

    const queue = screen.getByRole("table", { name: "Finance Review Queue" });
    for (const column of ["Plan", "Department", "Requested", "Current Owner", "Status", "Action"]) {
      expect(within(queue).getByRole("columnheader", { name: column })).toBeVisible();
    }
    expect(within(queue).getByText(PLAN_REFERENCE)).toBeVisible();
    await userEvent.setup().click(within(queue).getByRole("button", { name: `Review ${PLAN_LABEL}` }));
    const approvalFlow = screen.getByRole("list", { name: "Finance approval flow" });
    expect(approvalFlow).toBeVisible();
    expect(within(approvalFlow).getAllByRole("listitem").map((stage) => [
      stage.getAttribute("data-planning-stage"),
      stage.querySelector("b")?.textContent,
    ])).toEqual([
      ["DEPARTMENT_MANAGER", "Department Manager"],
      ["FINANCE_REVIEW", "Finance Review"],
      ["GM_REVIEW", "GM Review"],
      ["EXECUTIVE_DIRECTOR_REVIEW", "Executive Director"],
      ["GM_RELEASE", "GM Release"],
    ]);
    expect(approvalFlow.querySelectorAll('[aria-current="step"]')).toHaveLength(1);
    expect(approvalFlow.querySelector('[data-planning-stage="FINANCE_REVIEW"]')).toHaveAttribute("aria-current", "step");
    expect(screen.getByTestId("planning-status")).toHaveTextContent("FINANCE_REVIEW");
    expect(screen.getByTestId("planning-owner")).toHaveTextContent("Finance Review");
    expect(screen.getByText("Revision 1")).toBeVisible();
    expect(within(queue).getByRole("button", { name: `${PLAN_LABEL} is already selected` })).toBeDisabled();
    expect(within(queue).getByRole("button", { name: `${PLAN_LABEL} is already selected` })).toHaveAttribute(
      "title",
      "This Planning item is already open in the Finance dossier.",
    );
    expect(screen.getByRole("button", { name: "Approve Budget" })).toBeEnabled();
    expect(screen.getByRole("button", { name: "Return for Revision" })).toBeEnabled();
    expect(document.body).not.toHaveTextContent(/sign plan|release plan|close Finding/i);
  });

  it("requires a return reason and exposes a stale planning revision without overwriting it", async () => {
    const runtime = createMockBackendRuntime();
    renderPage(runtime);
    const user = userEvent.setup();

    await user.click(await screen.findByRole("button", { name: `Review ${PLAN_LABEL}` }));
    await user.click(await screen.findByRole("button", { name: "Return for Revision" }));
    await user.click(screen.getByRole("button", { name: "Confirm Finance Decision" }));
    expect(screen.getByRole("alert")).toHaveTextContent(/decision reason is required/i);

    const current = (await runtime.backendForRole("finance").planning.list({ limit: 20 })).items[0]!;
    await runtime.backendForRole("finance").planning.decide({
      operationId: "OP-FINANCE-EXTERNAL-ADVANCE",
      planningItemId: current.id,
      expectedPlanningRevision: current.revision,
      decision: "APPROVE_BUDGET",
      reason: "External concurrent budget approval.",
    });

    await user.type(screen.getByLabelText("Finance decision reason"), "Return the budget for correction.");
    await user.click(screen.getByRole("button", { name: "Confirm Finance Decision" }));
    expect(await screen.findByRole("alert")).toHaveTextContent(/revision|stale/i);
    expect(screen.getByTestId("planning-status")).toHaveTextContent("FINANCE_REVIEW");
  });

  it("advances only the Finance-owned item and bounds the General Manager handoff", async () => {
    const runtime = createMockBackendRuntime();
    renderPage(runtime);
    const user = userEvent.setup();

    await user.click(await screen.findByRole("button", { name: `Review ${PLAN_LABEL}` }));
    await user.click(await screen.findByRole("button", { name: "Approve Budget" }));
    await user.type(screen.getByLabelText("Finance decision reason"), "Budget and resources reviewed.");
    await user.click(screen.getByRole("button", { name: "Confirm Finance Decision" }));

    expect(await screen.findByTestId("planning-status")).toHaveTextContent("GM_REVIEW");
    expect(screen.getByTestId("planning-owner")).toHaveTextContent("General Manager");
    expect(screen.getByText("GM REVIEW")).toHaveClass("authority-badge");
    expect(screen.getByRole("button", { name: "Continue as General Manager" })).toBeEnabled();
    expect(screen.getByRole("list", { name: "Finance approval flow" }).querySelector('[data-planning-stage="GM_REVIEW"]')).toHaveAttribute(
      "aria-current",
      "step",
    );

    cleanup();
    renderPage(runtime, "oidc-session");
    await screen.findByRole("table", { name: "Finance Review Queue" });
    await user.selectOptions(screen.getByLabelText("Finance status"), "all");
    await user.click(await screen.findByRole("button", { name: `Review ${PLAN_LABEL}` }));
    expect(await screen.findByRole("button", { name: "Continue as General Manager" })).toBeDisabled();
    expect(screen.getByText(/session does not include General Manager authority/i)).toBeVisible();
  });

  it("fails a direct unauthenticated route before mounting the Finance data read", async () => {
    const runtime = createMockBackendRuntime();
    const list = vi.spyOn(runtime.backendForRole("finance").planning, "list");
    const sessionClient: SessionClient = {
      get: vi.fn().mockRejectedValue(new SessionClientError("UNAUTHENTICATED", "Authentication required")),
      login: vi.fn(),
      logout: vi.fn(),
      csrfToken: vi.fn(() => null),
    };
    render(
      <AppProviders runtime={{
        backend: runtime.backend,
        backendForRole: runtime.backendForRole,
        buildProfile: "http",
        environmentLabel: "test",
        identityMode: "oidc-session",
        subjectId: "anonymous",
      }}>
        <SessionProvider client={sessionClient} identityMode="oidc-session" initialRole="finance">
          <MemoryRouter initialEntries={["/finance/finance-review"]}><AppRouter /></MemoryRouter>
        </SessionProvider>
      </AppProviders>,
    );

    expect(await screen.findByTestId("route-unauthenticated")).toBeVisible();
    expect(list).not.toHaveBeenCalled();
  });
});
