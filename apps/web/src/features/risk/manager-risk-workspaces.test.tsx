// @vitest-environment jsdom
import "@testing-library/jest-dom/vitest";

import { cleanup, render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { afterEach, describe, expect, it } from "vitest";

import { AppProviders } from "../../app/providers";
import { AppRouter } from "../../app/router";
import { ScenarioProvider } from "../../app/scenario-context";
import type { DemoBackend } from "../../backend/backend";
import { createMockBackendRuntime } from "../../mock/create-mock-backend";
import { seedVisualRuntimeForPath } from "../../mock/seed-visual-runtime";

type MockRuntime = ReturnType<typeof createMockBackendRuntime>;

afterEach(cleanup);

function renderManagerRoute(path: string, runtime: MockRuntime = createMockBackendRuntime()) {
  render(
    <AppProviders runtime={{
      backend: runtime.backend,
      backendForRole: runtime.backendForRole,
      buildProfile: "demo",
      environmentLabel: "test",
      identityMode: "demo-role-switch",
      subjectId: "USR-MANAGER-NORA",
    }}>
      <ScenarioProvider>
        <MemoryRouter initialEntries={[path]}><AppRouter /></MemoryRouter>
      </ScenarioProvider>
    </AppProviders>,
  );
  return runtime;
}

describe("Department Manager intelligence workspaces", () => {
  it.each([
    ["/department-manager/risk-dashboard", "manager-risk-dashboard-page", "Risk Dashboard"],
    ["/department-manager/safety-intelligence", "manager-safety-intelligence-page", "Safety Intelligence Dashboard"],
    ["/department-manager/organizations/ORG-FLY-NAMIBIA/risk-profile", "organization-risk-profile-page", "Organization Risk Profile — Fly Namibia"],
    ["/department-manager/ssp-nasp", "manager-ssp-nasp-page", "SSP/NASP Management Dashboard"],
    ["/department-manager/usoap-readiness", "manager-usoap-readiness-page", "USOAP Readiness Workspace"],
    ["/department-manager/cap-effectiveness", "manager-cap-effectiveness-page", "CAP Effectiveness"],
  ])("direct-loads %s as the real advisory workspace", async (path, testId, heading) => {
    renderManagerRoute(path);

    const page = await screen.findByTestId(testId);
    expect(within(page).getByRole("heading", { level: 1, name: heading })).toBeVisible();
    expect(screen.getByTestId("application-shell")).toHaveAttribute("data-active-role", "manager");
    expect(screen.queryByTestId("route-pending-implementation")).toBeNull();
    expect(page).toHaveTextContent(/indicator|monitoring|readiness|effectiveness/i);
    expect(page).toHaveTextContent(/not (a legal decision|an official|automatic)/i);
    expect(page).not.toHaveTextContent(/automatically (suspend|revoke|close|certify)/i);
    const navigation = screen.getByRole("navigation", { name: "Primary role navigation" });
    expect(within(navigation).getByRole("link", { name: "Risk Dashboard" })).toHaveAttribute(
      "href",
      "/department-manager/risk-dashboard",
    );
    expect(within(navigation).getAllByRole("link").filter((link) => link.hasAttribute("aria-current"))).toHaveLength(1);
  });

  it("filters management signals and preserves exact Organization navigation", async () => {
    const user = userEvent.setup();
    renderManagerRoute("/department-manager/safety-intelligence");
    const page = await screen.findByTestId("manager-safety-intelligence-page");

    expect(await within(page).findByText("Fly Namibia")).toBeVisible();
    expect(within(page).getByRole("link", { name: "Open risk profile for ORG-FLY-NAMIBIA" })).toHaveAttribute(
      "href",
      "/department-manager/organizations/ORG-FLY-NAMIBIA/risk-profile",
    );
    expect(within(page).getByRole("button", { name: "Risk profile unavailable for ORG-SKYCARGO" })).toHaveAttribute(
      "title",
      "Organization ORG-SKYCARGO has no declared Department Manager risk-profile route in Plan 1.",
    );
    expect(within(page).queryByRole("link", { name: "Open risk profile for ORG-SKYCARGO" })).toBeNull();
    await user.selectOptions(within(page).getByLabelText("Signal filter"), "overdue");
    expect(within(page).getByTestId("active-signal-filter")).toHaveTextContent("overdue");
  });

  it("restores the complete typed Finding-risk command hierarchy with the configured heatmap immediately after filters", async () => {
    renderManagerRoute("/department-manager/risk-dashboard");
    const page = await screen.findByTestId("manager-risk-dashboard-page");

    const filters = within(page).getByRole("region", { name: "Risk filters" });
    const matrix = within(page).getByRole("region", { name: "Risk Exposure Matrix" });
    const indicatorRegion = within(page).getByRole("region", { name: "Risk indicators" });
    expect(matrix.querySelectorAll("[data-matrix-cell]")).toHaveLength(25);
    expect(matrix.querySelectorAll(".is-low, .is-medium, .is-high, .is-critical")).toHaveLength(25);
    expect(filters.compareDocumentPosition(matrix) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
    expect(matrix.compareDocumentPosition(indicatorRegion) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
    for (const label of ["Date Range", "Department", "Inspection", "Risk Level"]) {
      expect(within(filters).getByLabelText(label)).toBeEnabled();
    }
    expect(within(filters).getByRole("button", { name: "Reset risk filters unavailable" })).toBeDisabled();
    expect(within(filters).getByRole("button", { name: "Reset risk filters unavailable" })).toHaveAttribute(
      "title",
      "Risk filters are already at their defaults.",
    );
    expect(within(filters).getByRole("link", { name: "Export CSV" })).toHaveAttribute(
      "download",
      "AviaSurveil360_Department_Risk_Summary.csv",
    );

    const indicators = within(page).getByRole("region", { name: "Risk indicators" });
    for (const label of ["Total Findings", "High", "Medium", "Low", "Very Low", "Overdue CAPs"]) {
      expect(within(indicators).getByText(label)).toBeVisible();
    }
    expect(within(indicators).getByText("Total Findings").closest("article")).toHaveTextContent("1");
    expect(within(indicators).getByText("Medium").closest("article")).toHaveTextContent("1");

    for (const heading of [
      "Findings by Risk",
      "Risk Trend",
      "Risk Exposure Matrix",
      "Top Risky Areas",
      "Department Risk Distribution",
      "Overdue CAPs by Risk",
      "Recent Finding records",
    ]) {
      expect(within(page).getByRole("heading", { name: heading })).toBeVisible();
    }
    expect(within(page).getByTestId("risk-exposure-matrix")).toHaveTextContent(
      "Finding placement follows the accepted demo severity and Due State mapping",
    );
    expect(page).toHaveTextContent("FND-SKYCARGO-2026-099");
    expect(page).not.toHaveTextContent("Oversight Health indicator");
    expect(page).not.toHaveTextContent(/\b78\b|\b62\b/);
  });

  it("applies all four Finding-risk filters, exports the active projection, shows no-match, and resets", async () => {
    const user = userEvent.setup();
    renderManagerRoute("/department-manager/risk-dashboard");
    const page = await screen.findByTestId("manager-risk-dashboard-page");
    const filters = within(page).getByRole("region", { name: "Risk filters" });

    await user.selectOptions(within(filters).getByLabelText("Date Range"), "last-30");
    expect(within(filters).getByRole("button", { name: "Reset filters" })).toBeEnabled();
    await user.selectOptions(within(filters).getByLabelText("Department"), "unavailable");
    await user.selectOptions(within(filters).getByLabelText("Inspection"), "AUD-2026-099");
    await user.selectOptions(within(filters).getByLabelText("Risk Level"), "HIGH");
    expect(within(filters).getByTestId("active-risk-filters")).toHaveTextContent(
      "last-30 · unavailable · AUD-2026-099 · HIGH",
    );
    expect(within(page).getByText("No Finding records match the active risk filters.")).toBeVisible();

    await user.click(within(filters).getByRole("link", { name: "Export CSV" }));
    expect(await within(filters).findByRole("status")).toHaveTextContent("CSV prepared for 0 Finding records");
    expect(within(filters).getByRole("link", { name: "Export CSV" }).getAttribute("href")).toMatch(/^data:text\/csv/);

    await user.click(within(filters).getByRole("button", { name: "Reset filters" }));
    expect(within(filters).getByTestId("active-risk-filters")).toHaveTextContent("all · all · all · all");
    expect(within(page).getByText("FND-SKYCARGO-2026-099")).toBeVisible();
    expect(within(filters).getByRole("button", { name: "Reset risk filters unavailable" })).toBeDisabled();
  });

  it("keeps Organization risk identity and Finding actions exact", async () => {
    renderManagerRoute("/department-manager/organizations/ORG-FLY-NAMIBIA/risk-profile");
    const page = await screen.findByTestId("organization-risk-profile-page");

    expect(page).toHaveTextContent("ORG-FLY-NAMIBIA");
    expect(within(page).getByRole("link", { name: "Open Fly Namibia organization record" })).toHaveAttribute(
      "href",
      "/department-manager/organizations/ORG-FLY-NAMIBIA",
    );
    expect(within(page).getByRole("link", { name: "Open Findings Review for ORG-FLY-NAMIBIA" })).toHaveAttribute(
      "href",
      "/department-manager/findings-review?organizationId=ORG-FLY-NAMIBIA",
    );
    expect(await within(page).findByText("74")).toBeVisible();
    expect(page).toHaveTextContent("Needs Attention");
    expect(page).toHaveTextContent("Configured demo scenario");
    expect(page).not.toHaveTextContent("Unavailable");
  });

  it("projects CAP revision and closure eligibility without inventing monitoring evidence", async () => {
    renderManagerRoute("/department-manager/cap-effectiveness");
    const page = await screen.findByTestId("manager-cap-effectiveness-page");

    expect(page).toHaveTextContent("FND-SKYCARGO-2026-099");
    expect(page).toHaveTextContent("CAP-CAR-2026-099");
    expect(page).toHaveTextContent("CAP-CAR-2026-099-R2");
    expect(page).toHaveTextContent("revision 2");
    expect(page).toHaveTextContent(
      "Finding FND-SKYCARGO-2026-099 is OPEN; effectiveness requires a CLOSED Finding with a closure or verification basis.",
    );
    expect(page).not.toHaveTextContent("Needs follow-up");
    expect(page).not.toHaveTextContent("Monitoring indicator");
    expect(within(page).getByRole("link", { name: "Open FND-SKYCARGO-2026-099" })).toHaveAttribute(
      "href",
      "/department-manager/findings-review?organizationId=ORG-SKYCARGO&findingId=FND-SKYCARGO-2026-099",
    );
  });

  it("moves a real closed Finding/CAP revision only to pending post-closure verification", async () => {
    const runtime = createMockBackendRuntime();
    const manager = runtime.backendForRole("manager") as DemoBackend;
    const finding = await manager.findings.get({ findingId: "FND-SKYCARGO-2026-099" });
    await manager.findings.authorizedClose({
      findingId: finding.id,
      expectedFindingRevision: finding.revision,
      operationId: "CLOSE-FND-SKYCARGO-2026-099-FOR-EFFECTIVENESS",
      reason: "Authorized closure recorded before separate effectiveness verification.",
    });

    const projection = await manager.risk.getManagementProjection({});
    expect(projection.capEffectiveness).toContainEqual(expect.objectContaining({
      findingId: "FND-SKYCARGO-2026-099",
      capId: "CAP-CAR-2026-099",
      capRevisionId: "CAP-CAR-2026-099-R2",
      capRevision: 2,
      closureBasis: "AUTHORIZED",
      state: "PENDING_POST_CLOSURE_VERIFICATION",
      reason: "Finding FND-SKYCARGO-2026-099 closed with AUTHORIZED; no typed post-closure effectiveness verification record is available.",
    }));
  });

  it("preserves verified closure and exact CAP revision identity in effectiveness eligibility", async () => {
    const runtime = createMockBackendRuntime();
    await seedVisualRuntimeForPath(runtime, "/department-manager/evidence/FND-CAB-2026-001");
    const lead = runtime.backendForRole("leadInspector");
    const finding = await lead.findings.get({ findingId: "FND-CAB-2026-001" });
    const latestEvidence = (await lead.evidence.listVersions({ findingId: finding.id })).at(-1);
    if (!latestEvidence) throw new Error("Expected the latest typed Evidence version.");
    await lead.evidence.review({
      operationId: "VERIFY-FND-CAB-2026-001-FOR-EFFECTIVENESS",
      evidenceVersionId: latestEvidence.id,
      expectedEvidenceVersionRevision: latestEvidence.revision,
      findingId: finding.id,
      expectedFindingRevision: finding.revision,
      decision: "CLOSE",
      commentToAuditee: "Latest Evidence accepted for Finding closure.",
      internalCaaNote: "Post-closure effectiveness remains a separate verification record.",
    });

    const projection = await runtime.backendForRole("manager").risk.getManagementProjection({});
    expect(projection.capEffectiveness).toContainEqual(expect.objectContaining({
      findingId: "FND-CAB-2026-001",
      capId: "CAP-CAB-2026-001",
      capRevisionId: "CAP-CAB-2026-001-R1",
      capRevision: 1,
      closureBasis: "EVIDENCE_VERIFIED",
      state: "PENDING_POST_CLOSURE_VERIFICATION",
      reason: "Finding FND-CAB-2026-001 closed with EVIDENCE_VERIFIED; no typed post-closure effectiveness verification record is available.",
    }));
  });

  it.each([1440, 1024, 390])("keeps advisory hierarchy and controls ordered at %ipx", async (width) => {
    Object.defineProperty(window, "innerWidth", { configurable: true, value: width });
    renderManagerRoute("/department-manager/risk-dashboard");
    const page = await screen.findByTestId("manager-risk-dashboard-page");
    const filters = within(page).getByRole("region", { name: "Risk filters" });
    const indicators = within(page).getByRole("region", { name: "Risk indicators" });
    const attention = within(page).getByRole("region", { name: "Management attention" });
    expect(filters.compareDocumentPosition(indicators) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
    expect(indicators.compareDocumentPosition(attention) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
    expect(within(page).getByLabelText("Risk Level")).toBeEnabled();
  });
});
