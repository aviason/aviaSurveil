import { readFileSync } from "node:fs";
import { resolve } from "node:path";

import { expect, test, type Page } from "@playwright/test";

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
): ActionEvidence | null {
  if (!ownership) return null;
  return (ledger.actionEvidence ?? []).find((entry) =>
    (entry.surfaceId === surface.id || (entry.surfaceId === "*" && control.scope !== "route")) &&
    entry.scope === control.scope &&
    entry.viewports.includes(viewport) &&
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
  const beforeTarget = await target.evaluateAll((elements) =>
    elements.map((element) => element.outerHTML.replace(/\s+/g, " ").trim()).join("\n")
  );
  const beforeTop = await target.first().evaluate((element) => element.getBoundingClientRect().top);
  await control.click();
  await expect(target, `${surface.id}/${current.controlKey} exposes its declared controlled surface`).toBeVisible();
  if (await target.evaluate((element) =>
    document.activeElement === element || element.contains(document.activeElement)
  )) return;
  await expect.poll(
    async () => {
      const afterTarget = await target.evaluateAll((elements) =>
        elements.map((element) => element.outerHTML.replace(/\s+/g, " ").trim()).join("\n")
      );
      const afterTop = await target.first().evaluate((element) => element.getBoundingClientRect().top);
      return afterTarget !== beforeTarget || Math.abs(afterTop - beforeTop) > 1;
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

expect(VISUAL_SURFACES).toHaveLength(86);
expect(VISUAL_VIEWPORTS).toHaveLength(3);
expect(VISUAL_SURFACES.length * VISUAL_VIEWPORTS.length).toBe(258);

for (const viewport of VISUAL_VIEWPORTS) {
  test(`inventories all 86 visible-action surfaces at ${viewport.id}`, async ({ page }, testInfo) => {
    test.setTimeout(300_000);
    await page.setViewportSize(viewport);
    if (testInfo.project.name === "http") await resetHttpProfile(page);
    const consoleIssues: string[] = [];
    page.on("console", (message) => {
      if (message.type() === "error") consoleIssues.push(`console: ${message.text()}`);
    });
    page.on("pageerror", (error) => consoleIssues.push(`pageerror: ${error.message}`));
    let actionInventories = 0;
    const seenEvidenceKeys = new Set<string>();
    for (const surface of VISUAL_SURFACES) {
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
        const evidence = evidenceFor(surface, viewport.id, control, ownership);
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
              evidence: evidenceFor(surface, viewport.id, control, ownership),
            };
          }),
          duplicateActions,
          consoleErrors: consoleIssues,
        }, null, 2),
        contentType: "application/json",
      });

      expect.soft(unnamed, `${surface.id}/${viewport.id} has unnamed controls`).toEqual([]);
      expect.soft(unowned, `${surface.id}/${viewport.id} has unowned or unexplained controls`).toEqual([]);
      expect.soft(missingEvidence, `${surface.id}/${viewport.id} has actions without outcome-test evidence`).toEqual([]);
      expect.soft(duplicateActions, `${surface.id}/${viewport.id} exposes duplicate route actions`).toEqual([]);
      expect.soft(consoleIssues, `${surface.id}/${viewport.id} has zero console errors`).toEqual([]);
    }
    expect(actionInventories).toBe(86);
    const expectedEvidenceKeys = (ledger.actionEvidence ?? [])
      .filter((entry) => entry.viewports.includes(viewport.id))
      .map(actionEvidenceKey)
      .sort();
    expect([...seenEvidenceKeys].sort(), `${viewport.id} action evidence rejects stale or missing entries`).toEqual(
      expectedEvidenceKeys,
    );
  });
}

test("executes every active route command with a durable outcome", async ({ page }) => {
  test.setTimeout(900_000);
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
      executableAssertions.has(entry.assertion)
    )
    .map(actionEvidenceKey)
    .sort();
  const verifiedEvidenceKeys = new Set<string>();

  for (const viewport of executionViewports) {
    await page.setViewportSize(viewport);
    for (const surface of VISUAL_SURFACES) {
      await installDeterministicPageState(page);
      await driveReactSurface(page, surface);
      const controls = assignControlKeys(await collectVisibleControls(page));
      const routeCommands = controls.filter(isExecutableRouteControl);
      for (const control of routeCommands) {
        const ownership = ownershipFor(surface, control);
        const evidence = evidenceFor(surface, viewport.id, control, ownership);
        expect(evidence, `${surface.id}/${viewport.id}/${control.controlKey} has exact executable evidence`).not.toBeNull();
        const evidenceKey = actionEvidenceKey(evidence!);
        if (verifiedEvidenceKeys.has(evidenceKey)) continue;
        await test.step(`${surface.id}/${viewport.id}: ${control.name}`, async () => {
          expect(evidence!.assertion).toBe(expectedExecutableAssertion(control));
          await assertDurableControlOutcome(page, surface, control);
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
