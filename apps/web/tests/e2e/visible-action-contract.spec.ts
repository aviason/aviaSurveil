import { readFileSync } from "node:fs";
import { resolve } from "node:path";

import { expect, test, type Page } from "@playwright/test";

import type { Backend, BackendPrincipal } from "../../src/backend/backend";
import { createHttpBackend } from "../../src/backend/http-backend";
import { createCanonicalTestFetch } from "../../src/test-profile/http-test-boundary";
import {
  createCanonicalFinding,
  PRINCIPALS,
  submitAndAcceptCanonicalCap,
  submitCanonicalCap,
  submitEvidence,
} from "../contract/backend-contract";
import {
  assertSurfaceSemantics,
  driveReactSurface,
  installDeterministicPageState,
  resolveReactSurfaceSemantics,
  VISUAL_SURFACES,
  VISUAL_VIEWPORTS,
  type VisualSurfaceFixture,
} from "./support/legacy-parity-fixtures";

interface OwnershipRule {
  id: string;
  surfaceIds: string[];
  namePattern: string;
  boundary: string;
  durableEffect: string;
  evidence: string;
}

interface ActionEvidenceGroup {
  surfaceIds: string[];
  evidence: string;
}

interface ActionEvidence {
  surfaceId: string;
  scope: VisibleControl["scope"];
  viewports: string[];
  profiles?: string[];
  controlKey: string;
  boundary: string;
  durableEffect: string;
  evidence: string;
  assertion: string;
}

interface VisibleControl {
  controlKey: string;
  marker: string;
  scope: "route" | "shell" | "mobile-navigation";
  tag: string;
  role: string;
  name: string;
  disabled: boolean;
  readOnly: boolean;
  disabledReason: string;
  actionBoundary: string | null;
  href: string | null;
  download: string | null;
  type: string | null;
  ariaControls: string | null;
  ariaCurrent: string | null;
  ariaExpanded: string | null;
  ariaPressed: string | null;
  ariaSelected: string | null;
}

const repoRoot = resolve(process.cwd(), "../..");
const ledger = JSON.parse(
  readFileSync(resolve(repoRoot, "tests/parity/behavior-ledger.json"), "utf8"),
) as {
  actionEvidence?: ActionEvidence[];
  actionEvidenceGroups: ActionEvidenceGroup[];
  visibleActionOwnership: OwnershipRule[];
};

const controlSelector = [
  "button:visible",
  "a:visible",
  "input:visible",
  "select:visible",
  "textarea:visible",
  "[role='menuitem']:visible",
  "[role='tab']:visible",
  "[role='button']:visible",
].join(", ");

function normalize(value: string | null | undefined): string {
  return value?.replace(/\s+/g, " ").trim() ?? "";
}

type UnkeyedVisibleControl = Omit<VisibleControl, "controlKey">;

async function collectVisibleControls(page: Page, scope?: string): Promise<UnkeyedVisibleControl[]> {
  const selector = scope
    ? controlSelector.split(", ").map((control) => `${scope} ${control}`).join(", ")
    : controlSelector;
  return page.locator(selector).evaluateAll((elements) => elements.flatMap((element, index) => {
    if (element.closest("[aria-hidden='true'], [inert]")) return [];
    const html = element as HTMLElement;
    const formControl = element as HTMLInputElement | HTMLSelectElement | HTMLTextAreaElement;
    const labels = "labels" in formControl && formControl.labels
      ? [...formControl.labels].map((label) => label.textContent ?? "").join(" ")
      : "";
    const describedBy = html.getAttribute("aria-describedby")
      ?.split(/\s+/)
      .map((id) => document.getElementById(id)?.textContent ?? "")
      .join(" ") ?? "";
    const textClone = html.cloneNode(true) as HTMLElement;
    textClone.querySelectorAll("[aria-hidden='true']").forEach((node) => node.remove());
    const name = (
      html.getAttribute("aria-label") ||
      labels ||
      (element instanceof HTMLImageElement ? element.alt : "") ||
      (element instanceof HTMLInputElement && ["button", "submit", "reset"].includes(element.type) ? element.value : "") ||
      textClone.textContent ||
      html.getAttribute("title") ||
      html.getAttribute("placeholder") ||
      ""
    ).replace(/\s+/g, " ").trim();
    const disabled = "disabled" in formControl
      ? Boolean(formControl.disabled)
      : html.getAttribute("aria-disabled") === "true";
    const nearby = disabled
      ? (html.closest("label, span, td, .workbench-decision-panel__action")?.textContent ?? "")
      : "";
    const marker = `visible-action-${index}`;
    html.setAttribute("data-visible-action-marker", marker);
    return [{
      marker,
      scope: html.closest(".mobile-navigation__drawer")
        ? "mobile-navigation"
        : html.closest(".mobile-navigation__opener, .workspace-sidebar, .application-topbar, .auditee-demo-ribbon, .manager-root-topbar, .authority-root-topbar, .admin-root-topbar")
          ? "shell"
          : "route",
      tag: element.tagName,
      role: html.getAttribute("role") ?? "",
      name,
      disabled,
      readOnly: "readOnly" in formControl ? Boolean(formControl.readOnly) : false,
      disabledReason: (html.getAttribute("title") || describedBy || html.getAttribute("placeholder") || nearby).replace(/\s+/g, " ").trim(),
      actionBoundary: html.getAttribute("data-action-boundary"),
      href: element instanceof HTMLAnchorElement ? element.getAttribute("href") : null,
      download: element instanceof HTMLAnchorElement && element.hasAttribute("download")
        ? element.getAttribute("download") ?? ""
        : null,
      type: "type" in formControl ? formControl.type : null,
      ariaControls: html.getAttribute("aria-controls"),
      ariaCurrent: html.getAttribute("aria-current"),
      ariaExpanded: html.getAttribute("aria-expanded"),
      ariaPressed: html.getAttribute("aria-pressed"),
      ariaSelected: html.getAttribute("aria-selected"),
    }];
  }));
}

function assignControlKeys(controls: UnkeyedVisibleControl[]): VisibleControl[] {
  const occurrences = new Map<string, number>();
  return controls.map((control) => {
    const stableHref = control.href?.startsWith("data:")
      ? control.href.slice(0, control.href.indexOf(","))
      : control.href;
    const baseKey = [
      control.scope,
      control.tag,
      control.role || control.type || "control",
      normalize(control.name),
      stableHref ?? control.ariaControls ?? "",
    ].join("|");
    const occurrence = (occurrences.get(baseKey) ?? 0) + 1;
    occurrences.set(baseKey, occurrence);
    return {
      ...control,
      controlKey: `${baseKey}|${occurrence}`,
    };
  });
}

function ownershipFor(surface: VisualSurfaceFixture, control: VisibleControl): string | null {
  if (control.actionBoundary) return `declared-${control.actionBoundary}`;
  if (control.tag === "A" && control.download !== null) return "verified-browser-artifact";
  if (control.tag === "A" && control.href && control.href !== "#") return "verified-navigation";
  if (
    control.disabled &&
    normalize(control.disabledReason) &&
    normalize(control.disabledReason) !== normalize(control.name)
  ) return "explicit-disabled-reason";
  if (control.disabled) return null;
  if (["INPUT", "SELECT", "TEXTAREA"].includes(control.tag)) return "verified-form-behavior";
  if (control.tag === "BUTTON" && ["submit", "reset"].includes(control.type ?? "")) return "verified-form-behavior";
  if (control.role === "tab" && control.ariaSelected !== null) return "verified-tab-state";
  if (control.ariaCurrent !== null || control.ariaExpanded !== null || control.ariaPressed !== null || control.ariaSelected !== null) return "verified-visible-state";
  if (control.ariaControls) return "verified-controlled-state";
  const declared = ledger.visibleActionOwnership.find((rule) =>
    (rule.surfaceIds.includes("*") || rule.surfaceIds.includes(surface.id)) &&
    new RegExp(rule.namePattern, "i").test(control.name),
  );
  return declared ? `${declared.boundary}:${declared.id}` : null;
}

function evidenceFor(
  surface: VisualSurfaceFixture,
  viewport: string,
  control: VisibleControl,
  ownership: string | null,
  profile: string,
): ActionEvidence | null {
  if (!ownership) return null;
  return (ledger.actionEvidence ?? []).find((entry) =>
    (entry.surfaceId === surface.id || (entry.surfaceId === "*" && control.scope !== "route")) &&
    entry.scope === control.scope &&
    entry.viewports.includes(viewport) &&
    (!entry.profiles || entry.profiles.includes(profile)) &&
    entry.controlKey === control.controlKey
  ) ?? null;
}

function actionEvidenceKey(entry: Pick<ActionEvidence, "surfaceId" | "scope" | "controlKey">): string {
  return `${entry.surfaceId}|${entry.scope}|${entry.controlKey}`;
}

async function durablePageState(page: Page): Promise<string> {
  return page.evaluate(() => {
    const routeRootElement = document.querySelector(
      ".role-select-page, .workspace-content > :last-child",
    ) as HTMLElement | null;
    const formState = [...(routeRootElement?.querySelectorAll("input, select, textarea") ?? [])].map((element) => {
      const control = element as HTMLInputElement | HTMLSelectElement | HTMLTextAreaElement;
      return {
        tag: control.tagName,
        type: control.getAttribute("type") ?? "",
        name: control.getAttribute("name") ?? control.getAttribute("aria-label") ?? control.getAttribute("placeholder") ?? "",
        value: control.value,
        checked: control instanceof HTMLInputElement ? control.checked : null,
        selectedIndex: control instanceof HTMLSelectElement ? control.selectedIndex : null,
      };
    });
    const routeRoot = routeRootElement?.cloneNode(true) as HTMLElement | undefined;
    routeRoot?.querySelectorAll([
      "[role='status']:not([data-durable-outcome])",
      ".toast",
      "[data-testid$='-status']",
      "[data-testid='response-status']",
    ].join(", ")).forEach((element) => element.remove());
    return JSON.stringify({
      path: `${window.location.pathname}${window.location.search}`,
      localStorage: Object.fromEntries(
        Object.keys(window.localStorage).sort().map((key) => [key, window.localStorage.getItem(key)]),
      ),
      sessionStorage: Object.fromEntries(
        Object.keys(window.sessionStorage).sort().map((key) => [key, window.sessionStorage.getItem(key)]),
      ),
      activeElement: document.activeElement instanceof HTMLElement
        ? `${document.activeElement.tagName}:${document.activeElement.id}:${document.activeElement.getAttribute("aria-label") ?? ""}`
        : "",
      scrollY: window.scrollY,
      formState,
      routeHtml: routeRoot?.outerHTML.replace(/\s+/g, " ").trim() ?? "",
    });
  });
}

function isNativeFormControl(control: VisibleControl): boolean {
  return ["INPUT", "SELECT", "TEXTAREA"].includes(control.tag) ||
    (control.tag === "BUTTON" && ["submit", "reset"].includes(control.type ?? ""));
}

function hasAccessibleState(control: VisibleControl): boolean {
  return control.ariaCurrent !== null ||
    control.ariaExpanded !== null ||
    control.ariaPressed !== null ||
    control.ariaSelected !== null;
}

function isExecutableRouteControl(control: VisibleControl): boolean {
  return control.scope === "route" &&
    !control.disabled &&
    (
      (control.tag === "A" && control.download !== null) ||
      isNativeFormControl(control) ||
      control.tag === "BUTTON" ||
      control.role === "button" ||
      control.role === "tab"
    );
}

function expectedExecutableAssertion(control: VisibleControl): string {
  if (isNativeFormControl(control)) return "assertNativeFormControlOutcome";
  if (hasAccessibleState(control)) return "assertAccessibleStateOutcome";
  if (control.ariaControls) return "assertControlledSurfaceOutcome";
  if (control.tag === "A" && control.download !== null) return "suggestedFilename";
  return "assertDurableControlOutcome";
}

async function assertNativeFormControlOutcome(
  page: Page,
  surface: VisualSurfaceFixture,
  current: VisibleControl,
  control: ReturnType<Page["locator"]>,
): Promise<void> {
  const before = await durablePageState(page);
  let expectsStateChange = true;
  if (current.readOnly) {
    const beforeValue = await control.inputValue();
    await expect(control).toHaveAttribute("readonly", "");
    await control.focus();
    await page.keyboard.type(" Task 11 mutation");
    await expect(control).toHaveValue(beforeValue);
    return;
  }
  if (current.tag === "SELECT") {
    const selection = await control.evaluate((element) => {
      const select = element as HTMLSelectElement;
      const targetIndex = [...select.options].findIndex((option, index) =>
        !option.disabled && index !== select.selectedIndex
      );
      return {
        beforeValue: select.value,
        optionCount: select.options.length,
        targetIndex,
        targetValue: targetIndex >= 0 ? select.options[targetIndex].value : select.value,
      };
    });
    if (selection.targetIndex < 0) {
      expect(selection.optionCount, `${surface.id}/${current.controlKey} exposes one exact fixed option`).toBe(1);
      await control.focus();
      await page.keyboard.press("Space");
      await page.keyboard.press("Escape");
      await expect(control).toHaveValue(selection.beforeValue);
      return;
    }
    await control.selectOption({ index: selection.targetIndex });
    await expect(control).toHaveValue(selection.targetValue);
    expect(selection.targetValue).not.toBe(selection.beforeValue);
  } else if (current.tag === "TEXTAREA") {
    const nextValue = `Task 11 verified ${surface.id}`;
    await control.fill(nextValue);
    await expect(control).toHaveValue(nextValue);
  } else {
    const inputType = (current.type ?? "text").toLowerCase();
    if (inputType === "file") {
      const fileName = `${surface.id}-task-11.txt`;
      await control.setInputFiles({
        name: fileName,
        mimeType: "text/plain",
        buffer: Buffer.from("AviaSurveil360 Task 11 control verification"),
      });
      expect(await control.evaluate((element) => (element as HTMLInputElement).files?.[0]?.name)).toBe(fileName);
    } else if (inputType === "checkbox") {
      const wasChecked = await control.isChecked();
      if (wasChecked) await control.uncheck();
      else await control.check();
      expect(await control.isChecked()).toBe(!wasChecked);
    } else if (inputType === "radio") {
      const wasChecked = await control.isChecked();
      await control.check();
      expect(await control.isChecked()).toBe(true);
      expectsStateChange = !wasChecked;
    } else if (["button", "submit", "reset"].includes(inputType)) {
      await control.focus();
      await page.keyboard.press("Enter");
    } else {
      const nextValue = await control.evaluate((element, id) => {
        const input = element as HTMLInputElement;
        switch (input.type) {
          case "date": return "2026-07-30";
          case "datetime-local": return "2026-07-30T12:30";
          case "email": return `${id}@example.test`;
          case "month": return "2026-07";
          case "number":
          case "range": {
            const parsedMinimum = Number(input.min);
            const parsedMaximum = Number(input.max);
            const minimum = input.min !== "" && Number.isFinite(parsedMinimum) ? parsedMinimum : 1;
            const maximum = input.max !== "" && Number.isFinite(parsedMaximum) ? parsedMaximum : minimum + 10;
            const current = Number.isFinite(input.valueAsNumber) ? input.valueAsNumber : minimum;
            return String(current < maximum ? current + 1 : Math.max(minimum, current - 1));
          }
          case "time": return "12:30";
          case "url": return "https://example.test/task-11";
          case "week": return "2026-W31";
          case "color": return "#1c7a84";
          default: return `Task 11 verified ${id}`;
        }
      }, surface.id.replace(/[^a-z0-9]/gi, ""));
      await control.fill(nextValue);
      await expect(control).toHaveValue(nextValue);
    }
  }
  if (!expectsStateChange) return;
  await expect.poll(
    () => durablePageState(page),
    {
      message: `${surface.id}/${current.controlKey} must expose its exact live form-state postcondition`,
      timeout: 2_500,
    },
  ).not.toBe(before);
}

async function assertAccessibleStateOutcome(
  page: Page,
  surface: VisualSurfaceFixture,
  current: VisibleControl,
  control: ReturnType<Page["locator"]>,
): Promise<void> {
  const before = {
    expanded: current.ariaExpanded,
    pressed: current.ariaPressed,
    selected: current.ariaSelected,
  };
  await control.click();
  if (before.expanded !== null) {
    await expect(control, `${surface.id}/${current.controlKey} toggles aria-expanded`).not.toHaveAttribute(
      "aria-expanded",
      before.expanded,
    );
  }
  if (before.pressed !== null) {
    await expect(control, `${surface.id}/${current.controlKey} exposes its exact aria-pressed state`).toHaveAttribute(
      "aria-pressed",
      "true",
    );
  }
  if (before.selected !== null) {
    await expect(control, `${surface.id}/${current.controlKey} activates its exact selected state`).toHaveAttribute(
      "aria-selected",
      "true",
    );
  }
  if (current.ariaCurrent !== null) {
    await expect(control, `${surface.id}/${current.controlKey} preserves its exact current state`).toHaveAttribute(
      "aria-current",
      current.ariaCurrent,
    );
  }
  if (current.ariaControls) {
    await expect(page.locator(`[id="${current.ariaControls}"]`)).toBeVisible();
  }
}

async function assertControlledSurfaceOutcome(
  page: Page,
  surface: VisualSurfaceFixture,
  current: VisibleControl,
  control: ReturnType<Page["locator"]>,
): Promise<void> {
  const target = page.locator(`[id="${current.ariaControls}"]`);
  const targetExisted = await target.count() > 0;
  const beforeTarget = targetExisted
    ? await target.evaluateAll((elements) =>
        elements.map((element) => element.outerHTML.replace(/\s+/g, " ").trim()).join("\n")
      )
    : "";
  const beforeTop = targetExisted
    ? await target.first().evaluate((element) => element.getBoundingClientRect().top)
    : null;
  await control.click();
  await expect(target, `${surface.id}/${current.controlKey} exposes its declared controlled surface`).toBeVisible();
  if (!targetExisted) return;
  if (await target.evaluate((element) =>
    document.activeElement === element || element.contains(document.activeElement)
  )) return;
  await expect.poll(
    async () => {
      const afterTarget = await target.evaluateAll((elements) =>
        elements.map((element) => element.outerHTML.replace(/\s+/g, " ").trim()).join("\n")
      );
      const afterTop = await target.first().evaluate((element) => element.getBoundingClientRect().top);
      return afterTarget !== beforeTarget || Math.abs(afterTop - beforeTop!) > 1;
    },
    {
      message: `${surface.id}/${current.controlKey} must update its exact controlled surface`,
      timeout: 2_500,
    },
  ).toBe(true);
}

async function assertDurableControlOutcome(
  page: Page,
  surface: VisualSurfaceFixture,
  expected: VisibleControl,
): Promise<void> {
  await driveReactSurface(page, surface);
  const controls = assignControlKeys(await collectVisibleControls(page));
  const current = controls.find((control) => control.controlKey === expected.controlKey);
  expect(current, `${surface.id} retains ${expected.controlKey}`).toBeDefined();
  const control = page.locator(`[data-visible-action-marker="${current!.marker}"]`);
  await expect(control).toBeVisible();

  if (current!.disabled) {
    expect(normalize(current!.disabledReason)).not.toBe("");
    return;
  }
  if (current!.tag === "A") {
    if (current!.download !== null) {
      const downloadPromise = page.waitForEvent("download", { timeout: 2_500 });
      await control.focus();
      await page.keyboard.press("Enter");
      const download = await downloadPromise;
      expect(await download.suggestedFilename()).not.toBe("");
      return;
    }
    expect(current!.href, `${surface.id}/${current!.controlKey} declares an exact destination`).toMatch(/^\//);
    return;
  }
  if (isNativeFormControl(current!)) {
    await assertNativeFormControlOutcome(page, surface, current!, control);
    return;
  }
  if (hasAccessibleState(current!)) {
    await assertAccessibleStateOutcome(page, surface, current!, control);
    return;
  }
  if (current!.ariaControls) {
    await assertControlledSurfaceOutcome(page, surface, current!, control);
    return;
  }

  const expectsDownload = /download|export/i.test(current!.name);
  const downloadPromise = expectsDownload
    ? page.waitForEvent("download", { timeout: 2_500 }).catch(() => null)
    : Promise.resolve(null);
  await control.focus();
  const before = await durablePageState(page);
  await page.keyboard.press("Enter");
  const download = await downloadPromise;
  if (download) {
    expect(await download.suggestedFilename()).not.toBe("");
    return;
  }
  await page.waitForLoadState("domcontentloaded").catch(() => undefined);
  await page.waitForTimeout(100);
  await expect.poll(
    () => durablePageState(page),
    {
      message: `${surface.id}/${current!.controlKey} must change durable state, not only a toast/status message`,
      timeout: 2_500,
    },
  ).not.toBe(before);
}

async function resetHttpProfile(page: Page): Promise<void> {
  const apiURL = process.env.AVIA_HTTP_API_URL ?? "http://127.0.0.1:58081";
  const token = process.env.AVIA_CANONICAL_TEST_TOKEN ?? "";
  const response = await page.request.post(`${apiURL}/__test/reset`, {
    headers: { "x-avia-test-token": token },
  });
  expect(response.ok()).toBe(true);
}

function liveHttpBackendFor(principal: BackendPrincipal): Backend {
  const apiURL = process.env.AVIA_HTTP_API_URL ?? "http://127.0.0.1:58081";
  const token = process.env.AVIA_CANONICAL_TEST_TOKEN ?? "";
  const subjectId =
    principal.subjectId === "USR-INSPECTOR-AMINA"
      ? "154ec5ac-6f97-4f55-916f-d2f142fc6211"
      : principal.subjectId;
  return createHttpBackend(
    { apiBaseUrl: apiURL, environmentLabel: "HTTP direct-load fixture" },
    { fetchImplementation: createCanonicalTestFetch(subjectId, token) },
  );
}

async function prepareHttpFindingFixture(): Promise<void> {
  const harness = { backendFor: liveHttpBackendFor };
  const finding = await createCanonicalFinding(harness);
  await submitCanonicalCap(harness, finding);
}

async function prepareHttpLeadReviewFixture(): Promise<void> {
  const inspector = liveHttpBackendFor(PRINCIPALS.inspector);
  const packageView = await inspector.inspections.getPackage({
    packageId: "PKG-CAB-2026-001",
  });
  const response = await inspector.inspections.upsertChecklistResponse({
    operationId: "OP-HTTP-LEAD-QUEUE-RESPONSE",
    responseId: "RESP-CAB-EMEQ-PBE-001",
    auditId: "AUD-2026-001",
    questionId: "CAB-EMEQ-PBE-001",
    expectedResponseRevision: null,
    answer: "NON_COMPLIANT",
    comment: "PBE serviceability and accessibility could not be confirmed.",
  });
  await inspector.potentialFindings.create({
    operationId: "OP-HTTP-LEAD-QUEUE-POTENTIAL",
    auditId: "AUD-2026-001",
    questionId: "CAB-EMEQ-PBE-001",
    checklistResponseId: response.id,
    expectedChecklistResponseRevision: response.revision,
    title: "PBE serviceability and accessibility not confirmed",
    description: "The configured cabin check could not confirm PBE serviceability.",
    requiredComment: response.comment,
    inspectionAttachmentIds: [],
  });
  await inspector.inspections.submitChecklist({
    operationId: "OP-HTTP-LEAD-QUEUE-CHECKLIST",
    auditId: packageView.auditId,
    expectedChecklistRevision: packageView.checklistRevision,
  });
}

async function prepareHttpFindingOnlyFixture(): Promise<void> {
  await createCanonicalFinding({ backendFor: liveHttpBackendFor });
}

async function prepareHttpCapReviewFixture(): Promise<void> {
  const harness = { backendFor: liveHttpBackendFor };
  const finding = await createCanonicalFinding(harness);
  const auditee = liveHttpBackendFor(PRINCIPALS.auditee);
  const lead = liveHttpBackendFor(PRINCIPALS.leadInspector);
  const first = await auditee.caps.submit({
    operationId: "OP-HTTP-CAP-R1",
    findingId: finding.id,
    expectedFindingRevision: finding.revision,
    rootCause: "Initial root cause retained for immutable history.",
    correctiveAction: "Initial corrective action.",
    preventiveAction: "Initial preventive action.",
    responsiblePerson: "Fly Namibia Cabin Safety Manager",
    targetCompletionDate: "2026-07-15",
    commentToCaa: "Initial CAP submitted for CAA review.",
  });
  const moreInformation = await lead.caps.review({
    operationId: "OP-HTTP-CAP-R1-REVIEW",
    capRevisionId: first.capRevisionId,
    expectedCapRevision: first.capRevision,
    findingId: finding.id,
    expectedFindingRevision: first.findingRevision,
    decision: "REQUEST_MORE_INFORMATION",
    commentToAuditee: "Clarify how PBE position records will be sampled.",
    internalCaaNote: "Internal CAA note for revision 1.",
  });
  await auditee.caps.submit({
    operationId: "OP-HTTP-CAP-R2",
    findingId: finding.id,
    expectedFindingRevision: moreInformation.findingRevision,
    rootCause: "Revised root cause with record reconciliation.",
    correctiveAction: "Replace affected PBE and update the cabin defect record.",
    preventiveAction: "Add supervisor review and monthly sampling.",
    responsiblePerson: "Fly Namibia Cabin Safety Manager",
    targetCompletionDate: "2026-07-20",
    commentToCaa: "Revised CAP submitted for CAA review.",
  });
}

async function prepareHttpEvidenceReviewFixture(): Promise<void> {
  const harness = { backendFor: liveHttpBackendFor };
  const finding = await createCanonicalFinding(harness);
  await submitAndAcceptCanonicalCap(harness, finding);
  const firstVersion = await submitEvidence(
    harness,
    "HTTP-V1",
    "Fly_Namibia_PBE_Serviceability_Record_CAB-2026-001.pdf",
  );
  const lead = liveHttpBackendFor(PRINCIPALS.leadInspector);
  const current = await lead.findings.get({ findingId: finding.id });
  await lead.evidence.review({
    operationId: "OP-HTTP-EVIDENCE-V1-REVIEW",
    evidenceVersionId: firstVersion.id,
    expectedEvidenceVersionRevision: firstVersion.revision,
    findingId: finding.id,
    expectedFindingRevision: current.revision,
    decision: "PARTIALLY_CLOSE",
    commentToAuditee: "Serviceability accepted; provide cabin position confirmation.",
    internalCaaNote: "Version 1 does not verify accessibility.",
  });
  await submitEvidence(
    harness,
    "HTTP-V2",
    "Fly_Namibia_PBE_Position_Confirmation_CAB-2026-001.pdf",
  );
}

async function prepareHttpMessageFixture(): Promise<void> {
  await liveHttpBackendFor(PRINCIPALS.inspector).communications.send({
    expectedRevision: null,
    idempotencyKey: "MSG-HTTP-AUDITEE-1",
    organizationId: "ORG-FLY-NAMIBIA",
    subject: "Inspection coordination update",
    body: "The proposed inspection date is ready for your confirmation.",
    audience: "AUDITEE",
  });
}

async function advanceHttpPlanningToGeneralManager(): Promise<void> {
  const finance = liveHttpBackendFor(PRINCIPALS.finance);
  const plans = await finance.planning.list({ limit: 20 });
  const plan = plans.items.find((item) => item.id === "PLAN-2026-CAB-001") ?? plans.items[0];
  if (!plan) throw new Error("HTTP fixture requires the canonical Planning item.");
  if (plan.status === "FINANCE_REVIEW") {
    await finance.planning.decide({
      operationId: `OP-HTTP-FINANCE-${plan.id}-${plan.revision}`,
      planningItemId: plan.id,
      expectedPlanningRevision: plan.revision,
      decision: "APPROVE_BUDGET",
      reason: "Finance approved the exact HTTP Planning revision.",
    });
  }
}

async function advanceHttpPlanningToExecutiveDirector(): Promise<void> {
  await advanceHttpPlanningToGeneralManager();
  const gm = liveHttpBackendFor(PRINCIPALS.gm);
  const plans = await gm.planning.list({ limit: 20 });
  const plan = plans.items.find((item) => item.id === "PLAN-2026-CAB-001") ?? plans.items[0];
  if (!plan) throw new Error("HTTP fixture requires the canonical Planning item.");
  if (plan.status === "GM_REVIEW") {
    await gm.planning.decide({
      operationId: `OP-HTTP-GM-${plan.id}-${plan.revision}`,
      planningItemId: plan.id,
      expectedPlanningRevision: plan.revision,
      decision: "FORWARD_FOR_FINAL_APPROVAL",
      reason: "General Manager forwarded the exact HTTP Planning revision.",
    });
  }
}

async function advanceHttpPreliminaryReportToGeneralManager(): Promise<void> {
  const manager = liveHttpBackendFor(PRINCIPALS.manager);
  const report = await manager.reports.getVersion({
    reportVersionId: "PR-2026-018-V1",
  });
  if (report.status === "DEPARTMENT_REVIEW") {
    await manager.reports.decide({
      operationId: `OP-HTTP-MANAGER-${report.reportVersionId}-${report.revision}`,
      reportVersionId: report.reportVersionId,
      expectedReportVersionRevision: report.revision,
      decision: "FORWARD",
      reason: "Department Manager forwarded the exact HTTP Preliminary Report version.",
    });
  }
}

async function prepareHttpProfileFixture(): Promise<void> {
  const profiles = liveHttpBackendFor(PRINCIPALS.inspector).profiles;
  const profile = await profiles.getMine({});
  await profiles.updateMine({
    expectedRevision: profile.revision,
    idempotencyKey: "PROFILE-HTTP-AYLIN",
    displayName: "Aylin Sezer",
  });
}

async function lockHttpAuditeeReportFixtures(): Promise<void> {
  const manager = liveHttpBackendFor(PRINCIPALS.manager);
  const gm = liveHttpBackendFor(PRINCIPALS.gm);
  const executive = liveHttpBackendFor(PRINCIPALS.executiveDirector);
  const preliminary = await manager.reports.getVersion({
    reportVersionId: "PR-2026-018-V1",
  });
  const atGeneralManager =
    preliminary.status === "DEPARTMENT_REVIEW"
      ? await manager.reports.decide({
          operationId: `OP-HTTP-DIRECT-MANAGER-${preliminary.revision}`,
          reportVersionId: preliminary.reportVersionId,
          expectedReportVersionRevision: preliminary.revision,
          decision: "FORWARD",
          reason: "Prepare the exact Preliminary Report for HTTP direct-load verification.",
        })
      : preliminary;
  const atExecutive =
    atGeneralManager.status === "GM_REVIEW"
      ? await gm.reports.decide({
          operationId: `OP-HTTP-DIRECT-GM-${atGeneralManager.revision}`,
          reportVersionId: atGeneralManager.reportVersionId,
          expectedReportVersionRevision: atGeneralManager.revision,
          decision: "FORWARD",
          reason: "Forward the exact Preliminary Report for HTTP direct-load verification.",
        })
      : atGeneralManager;
  if (atExecutive.status === "EXECUTIVE_DIRECTOR_REVIEW") {
    await executive.reports.decide({
      operationId: `OP-HTTP-DIRECT-EXEC-PRELIMINARY-${atExecutive.revision}`,
      reportVersionId: atExecutive.reportVersionId,
      expectedReportVersionRevision: atExecutive.revision,
      decision: "ISSUE_AND_LOCK",
      reason: "Issue the exact Preliminary Report for Auditee direct-load verification.",
    });
  }
  const finalReport = await manager.reports.getVersion({
    reportVersionId: "RPT-CAB-2026-001-V1",
  });
  const finalAtGeneralManager =
    finalReport.status === "DEPARTMENT_REVIEW"
      ? await manager.reports.decide({
          operationId: `OP-HTTP-DIRECT-MANAGER-FINAL-${finalReport.revision}`,
          reportVersionId: finalReport.reportVersionId,
          expectedReportVersionRevision: finalReport.revision,
          decision: "FORWARD",
          reason: "Prepare the exact Final Report for HTTP direct-load verification.",
        })
      : finalReport;
  const finalAtExecutive =
    finalAtGeneralManager.status === "GM_REVIEW"
      ? await gm.reports.decide({
          operationId: `OP-HTTP-DIRECT-GM-FINAL-${finalAtGeneralManager.revision}`,
          reportVersionId: finalAtGeneralManager.reportVersionId,
          expectedReportVersionRevision: finalAtGeneralManager.revision,
          decision: "FORWARD",
          reason: "Forward the exact Final Report for HTTP direct-load verification.",
        })
      : finalAtGeneralManager;
  if (finalAtExecutive.status === "EXECUTIVE_DIRECTOR_REVIEW") {
    await executive.reports.decide({
      operationId: `OP-HTTP-DIRECT-EXEC-FINAL-${finalAtExecutive.revision}`,
      reportVersionId: finalAtExecutive.reportVersionId,
      expectedReportVersionRevision: finalAtExecutive.revision,
      decision: "ISSUE_AND_LOCK",
      reason: "Issue the exact Final Report for Auditee direct-load verification.",
    });
  }
}

async function prepareHttpSurfaceFixture(
  page: Page,
  surface: VisualSurfaceFixture,
): Promise<void> {
  await resetHttpProfile(page);
  const pathname = surface.reactPath;
  if ([
    "/auditee/preliminary-reports",
    "/auditee/final-reports",
    "/auditee/reports/RPT-CAB-2026-001",
    "/auditee/documents",
  ].includes(pathname)) {
    if (pathname === "/auditee/documents") await prepareHttpEvidenceReviewFixture();
    await lockHttpAuditeeReportFixtures();
    return;
  }
  if (pathname === "/auditee/messages") {
    await prepareHttpMessageFixture();
    return;
  }
  if (pathname === "/general-manager/planning") {
    await advanceHttpPlanningToGeneralManager();
    return;
  }
  if (pathname === "/general-manager/report-approvals") {
    await advanceHttpPreliminaryReportToGeneralManager();
    return;
  }
  if (pathname === "/executive-director/planning") {
    await advanceHttpPlanningToExecutiveDirector();
    return;
  }
  if (pathname === "/inspector/profile") {
    await prepareHttpProfileFixture();
    return;
  }
  if (pathname === "/lead-inspector/lead-review") {
    await prepareHttpLeadReviewFixture();
    return;
  }
  if (pathname === "/lead-inspector/preliminary-reports/PR-2026-018") {
    await prepareHttpFindingOnlyFixture();
    return;
  }
  if ([
    "/inspector/findings",
    "/inspector/findings/FND-CAB-2026-001",
    "/inspector/closure-reports/CR-CAB-2026-001",
    "/inspector/assistant",
  ].includes(pathname)) {
    await prepareHttpFindingFixture();
    return;
  }
  if (pathname === "/lead-inspector/cap-review/FND-CAB-2026-001") {
    await prepareHttpCapReviewFixture();
    return;
  }
  if ([
    "/department-manager/findings-review",
    "/department-manager/cap-monitoring",
    "/department-manager/evidence/FND-CAB-2026-001",
    "/department-manager/findings/FND-CAB-2026-001/closure-review",
    "/department-manager/organizations/ORG-FLY-NAMIBIA",
  ].includes(pathname)) {
    await prepareHttpEvidenceReviewFixture();
    return;
  }
  if (pathname === "/auditee/service-provider-cap") {
    await prepareHttpFindingOnlyFixture();
  }
}

expect(VISUAL_SURFACES).toHaveLength(86);
expect(VISUAL_VIEWPORTS).toHaveLength(3);
expect(VISUAL_SURFACES.length * VISUAL_VIEWPORTS.length).toBe(258);

const focusedViewport = process.env.AVIA_VISIBLE_ACTION_VIEWPORT;
const inventoryViewports = focusedViewport
  ? VISUAL_VIEWPORTS.filter((viewport) => viewport.id === focusedViewport)
  : VISUAL_VIEWPORTS;
if (focusedViewport && inventoryViewports.length === 0) {
  throw new Error(`Unknown AVIA_VISIBLE_ACTION_VIEWPORT: ${focusedViewport}`);
}

for (const viewport of inventoryViewports) {
  test(`inventories all 86 visible-action surfaces at ${viewport.id}`, async ({ page }, testInfo) => {
    test.skip(
      process.env.AVIA_VISIBLE_ACTION_EXECUTION_ONLY === "1",
      "Focused command diagnosis excludes surface inventories.",
    );
    test.setTimeout(300_000);
    await page.setViewportSize(viewport);
    const consoleIssues: string[] = [];
    page.on("console", (message) => {
      if (message.type() === "error") consoleIssues.push(`console: ${message.text()}`);
    });
    page.on("pageerror", (error) => consoleIssues.push(`pageerror: ${error.message}`));
    let actionInventories = 0;
    const seenEvidenceKeys = new Set<string>();
    for (const surface of VISUAL_SURFACES) {
      if (testInfo.project.name === "http") await prepareHttpSurfaceFixture(page, surface);
      consoleIssues.length = 0;
      await installDeterministicPageState(page);
      await driveReactSurface(page, surface);
      actionInventories += 1;

      await assertSurfaceSemantics(page, resolveReactSurfaceSemantics(surface));
      const pageControls = await collectVisibleControls(page);
      let navigationControls: UnkeyedVisibleControl[] = [];
      if (surface.id === "role-select") {
        await expect(page.getByRole("navigation", { name: "Primary role navigation" })).toHaveCount(0);
      } else if (viewport.width <= 900) {
        await expect(page.locator(".workspace-sidebar")).toHaveAttribute("aria-hidden", "true");
        const opener = page.getByRole("button", { name: "Open navigation" });
        await expect(opener).toHaveAttribute("aria-expanded", "false");
        await opener.click();
        await expect(opener).toHaveAttribute("aria-expanded", "true");
        await expect(page.getByRole("navigation", { name: "Primary role navigation" })).toHaveCount(1);
        navigationControls = await collectVisibleControls(page, ".mobile-navigation__drawer");
        await page.keyboard.press("Escape");
        await expect(opener).toBeFocused();
      } else {
        await expect(page.getByRole("navigation", { name: "Primary role navigation" })).toHaveCount(1);
        await expect(page.locator(".workspace-sidebar")).not.toHaveAttribute("aria-hidden", "true");
      }

      const controls = assignControlKeys([...pageControls, ...navigationControls]);
      const unnamed = controls.filter((control) => !normalize(control.name));
      const unowned = controls.map((control) => ({
        ...control,
        ownership: ownershipFor(surface, control),
      })).filter(({ ownership }) => ownership === null);
      const missingEvidence = controls.map((control) => {
        const ownership = ownershipFor(surface, control);
        const evidence = evidenceFor(
          surface,
          viewport.id,
          control,
          ownership,
          testInfo.project.name,
        );
        if (evidence) seenEvidenceKeys.add(actionEvidenceKey(evidence));
        return {
          ...control,
          ownership,
          evidence,
        };
      }).filter(({ ownership, evidence }) => ownership !== null && evidence === null);
      const duplicateActions = viewport.width <= 1024
        ? [...controls.reduce((groups, control) => {
            if (control.disabled || !["A", "BUTTON"].includes(control.tag)) return groups;
            const key = `${control.tag}:${control.name}:${control.href ?? control.ariaControls ?? ""}`;
            groups.set(key, (groups.get(key) ?? 0) + 1);
            return groups;
          }, new Map<string, number>())].filter(([, count]) => count > 1)
        : [];

      await testInfo.attach(`visible-actions-${surface.id}-${viewport.id}`, {
        body: JSON.stringify({
          surfaceId: surface.id,
          viewport: viewport.id,
          controls: controls.map((control) => {
            const ownership = ownershipFor(surface, control);
            return {
              ...control,
              ownership,
              evidence: evidenceFor(
                surface,
                viewport.id,
                control,
                ownership,
                testInfo.project.name,
              ),
            };
          }),
          duplicateActions,
          consoleErrors: consoleIssues,
        }, null, 2),
        contentType: "application/json",
      });

      expect.soft(unnamed, `${surface.id}/${viewport.id} has unnamed controls`).toEqual([]);
      expect.soft(unowned, `${surface.id}/${viewport.id} has unowned or unexplained controls`).toEqual([]);
      expect.soft(
        missingEvidence.map(({ controlKey }) => controlKey),
        `${surface.id}/${viewport.id} has actions without outcome-test evidence`,
      ).toEqual([]);
      expect.soft(duplicateActions, `${surface.id}/${viewport.id} exposes duplicate route actions`).toEqual([]);
      expect.soft(consoleIssues, `${surface.id}/${viewport.id} has zero console errors`).toEqual([]);
    }
    expect(actionInventories).toBe(86);
    const expectedEvidenceKeys = (ledger.actionEvidence ?? [])
      .filter((entry) =>
        entry.viewports.includes(viewport.id) &&
        (!entry.profiles || entry.profiles.includes(testInfo.project.name))
      )
      .map(actionEvidenceKey)
      .sort();
    expect([...seenEvidenceKeys].sort(), `${viewport.id} action evidence rejects stale or missing entries`).toEqual(
      expectedEvidenceKeys,
    );
  });
}

test("executes every active route command with a durable outcome", async ({ page }, testInfo) => {
  test.skip(
    process.env.AVIA_VISIBLE_ACTION_INVENTORY_ONLY === "1",
    "Focused inventory diagnosis excludes command execution.",
  );
  test.setTimeout(1_800_000);
  let verifiedCommands = 0;
  const executionViewports = [VISUAL_VIEWPORTS[0], VISUAL_VIEWPORTS[2]];
  const executableAssertions = new Set([
    "assertNativeFormControlOutcome",
    "assertAccessibleStateOutcome",
    "assertControlledSurfaceOutcome",
    "assertDurableControlOutcome",
    "suggestedFilename",
  ]);
  const expectedEvidenceKeys = (ledger.actionEvidence ?? [])
    .filter((entry) =>
      entry.scope === "route" &&
      (!entry.profiles || entry.profiles.includes(testInfo.project.name)) &&
      executableAssertions.has(entry.assertion)
    )
    .map(actionEvidenceKey)
    .sort();
  const verifiedEvidenceKeys = new Set<string>();

  for (const viewport of executionViewports) {
    await page.setViewportSize(viewport);
    for (const surface of VISUAL_SURFACES) {
      if (testInfo.project.name === "http") await prepareHttpSurfaceFixture(page, surface);
      await installDeterministicPageState(page);
      await driveReactSurface(page, surface);
      const controls = assignControlKeys(await collectVisibleControls(page));
      const routeCommands = controls.filter(isExecutableRouteControl);
      let httpBackendStateDirty = false;
      for (const control of routeCommands) {
        const ownership = ownershipFor(surface, control);
        const evidence = evidenceFor(
          surface,
          viewport.id,
          control,
          ownership,
          testInfo.project.name,
        );
        expect(evidence, `${surface.id}/${viewport.id}/${control.controlKey} has exact executable evidence`).not.toBeNull();
        const evidenceKey = actionEvidenceKey(evidence!);
        if (verifiedEvidenceKeys.has(evidenceKey)) continue;
        await test.step(`${surface.id}/${viewport.id}: ${control.name}`, async () => {
          if (testInfo.project.name === "http" && httpBackendStateDirty) {
            await prepareHttpSurfaceFixture(page, surface);
            await installDeterministicPageState(page);
            httpBackendStateDirty = false;
          }
          expect(evidence!.assertion).toBe(expectedExecutableAssertion(control));
          await assertDurableControlOutcome(page, surface, control);
          if (
            testInfo.project.name === "http" &&
            (
              /mutation|workflow/i.test(evidence!.boundary) ||
              /^Deactivate /.test(control.name)
            )
          ) {
            httpBackendStateDirty = true;
          }
          verifiedEvidenceKeys.add(evidenceKey);
        });
        verifiedCommands += 1;
      }
    }
  }

  expect([...verifiedEvidenceKeys].sort(), "every executable ledger record is activated exactly once").toEqual(
    expectedEvidenceKeys,
  );
  expect(verifiedCommands).toBe(expectedEvidenceKeys.length);
});
