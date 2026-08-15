// @vitest-environment jsdom
import "@testing-library/jest-dom/vitest";

import { cleanup, render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { afterEach, describe, expect, it } from "vitest";

import { AppProviders } from "../../app/providers";
import { AppRouter } from "../../app/router";
import { ScenarioProvider } from "../../app/scenario-context";
import { createMockBackendRuntime } from "../../mock/create-mock-backend";
import { seedVisualRuntimeForPath } from "../../mock/seed-visual-runtime";

afterEach(cleanup);

type MockRuntime = ReturnType<typeof createMockBackendRuntime>;

function renderRoute(path: string, runtime: MockRuntime = createMockBackendRuntime()) {
  render(
    <AppProviders runtime={{
      backend: runtime.backend,
      backendForRole: runtime.backendForRole,
      buildProfile: "demo",
      environmentLabel: "test",
      identityMode: "demo-role-switch",
      subjectId: "USR-INSPECTOR-AMINA",
    }}>
      <ScenarioProvider>
        <MemoryRouter initialEntries={[path]}>
          <AppRouter />
        </MemoryRouter>
      </ScenarioProvider>
    </AppProviders>,
  );
  return runtime;
}

describe("Inspector secondary routes", () => {
  it("loads the server-owned Finding queue without implicit selection or fixture actions", async () => {
    const runtime = createMockBackendRuntime();
    renderRoute("/inspector/findings", runtime);

    const page = await screen.findByTestId("inspector-findings-page");
    expect(within(page).getByRole("heading", { name: "Findings" })).toBeVisible();
    expect(within(page).getByText(/^\d+ findings$/)).toBeVisible();
    expect(within(page).queryByRole("article", { name: /Selected Finding/ })).toBeNull();
    expect(page).not.toHaveTextContent(/Accept CAP|Return for Revision/);
  });

  it("binds the selected Finding dossier to the query-string identity and keeps sections read-only", async () => {
    const runtime = createMockBackendRuntime();
    await seedVisualRuntimeForPath(runtime, "/inspector/findings");
    const expected = (await runtime.backendForRole("inspector").findings.list({ limit: 50 })).items[0];
    if (!expected) throw new Error("Expected a server-owned Finding for the selection contract.");
    renderRoute(`/inspector/findings?findingId=${encodeURIComponent(expected.id)}`, runtime);

    const page = await screen.findByTestId("inspector-findings-page");
    const selected = await within(page).findByRole("article", { name: `Selected Finding ${expected.findingNumber}` });
    expect(selected).toHaveAttribute("data-finding-id", expected.id);
    expect(within(selected).getByRole("tablist", { name: "Finding dossier sections" })).toBeVisible();
    expect(within(selected).getByRole("button", { name: "Accept CAP unavailable" })).toBeDisabled();
    const user = userEvent.setup();
    await user.click(within(selected).getByRole("tab", { name: "Details" }));
    expect(within(selected).getByRole("region", { name: `Finding details for ${expected.findingNumber}` })).toBeVisible();
  });

  it("filters the exact queue and resets through server-owned controls", async () => {
    const runtime = createMockBackendRuntime();
    await seedVisualRuntimeForPath(runtime, "/inspector/findings");
    renderRoute("/inspector/findings", runtime);
    const page = await screen.findByTestId("inspector-findings-page");
    const user = userEvent.setup();
    const allCount = await within(page).findByText(/\d+ findings/);
    const initial = Number(allCount.textContent?.match(/\d+/)?.[0] ?? 0);

    await user.click(within(page).getByRole("button", { name: /CAP Submitted/ }));
    expect(within(page).getByRole("button", { name: /CAP Submitted/ })).toHaveAttribute("aria-pressed", "true");
    expect(within(page).getByRole("button", { name: "Reset" })).toBeEnabled();
    await user.click(within(page).getByRole("button", { name: "Reset" }));
    expect(within(page).getByText(`${initial} findings`)).toBeVisible();
    expect(within(page).getByRole("button", { name: "Reset Finding filters unavailable" })).toBeDisabled();
  });

  it("keeps message, reports, and assistant surfaces connected to their declared server boundaries", async () => {
    const routes = [
      ["/inspector/messages", "inspector-messages-page", "Messages from the CAA"],
      ["/inspector/reports", "inspector-reports-page", "Reports"],
      ["/inspector/assistant", "inspector-assistant-page", "AI Inspector Assistant Panel"],
    ] as const;
    for (const [path, testId, heading] of routes) {
      const runtime = createMockBackendRuntime();
      await seedVisualRuntimeForPath(runtime, path);
      renderRoute(path, runtime);
      const page = await screen.findByTestId(testId);
      expect(within(page).getByRole("heading", { name: heading })).toBeVisible();
      expect(page).not.toHaveTextContent(/localStorage|sessionStorage|candidate-intake|Final authorized demo approval/i);
      cleanup();
    }
  });
});
