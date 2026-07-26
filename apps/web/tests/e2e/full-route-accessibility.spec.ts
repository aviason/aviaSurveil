import { expect, test, type Locator, type Page } from "@playwright/test";

import {
  assertSurfaceSemantics,
  driveReactSurface,
  installDeterministicPageState,
  resolveReactSurfaceSemantics,
  VISUAL_SURFACES,
  VISUAL_VIEWPORTS,
} from "./support/legacy-parity-fixtures";

interface TargetSizeFailure {
  name: string;
  tag: string;
  width: number;
  height: number;
}

const formControlSelector = "input:not([type='hidden']):visible, select:visible, textarea:visible";
const targetSelector = [
  "a[href]:visible",
  "button:visible:not(:disabled)",
  "input:visible:not(:disabled)",
  "select:visible:not(:disabled)",
  "textarea:visible:not(:disabled)",
  "[role='button']:visible:not([aria-disabled='true'])",
  "[role='tab']:visible:not([aria-disabled='true'])",
].join(", ");

function pathname(page: Page): string {
  return new URL(page.url()).pathname;
}

async function visibleFormControlsWithoutNames(page: Page): Promise<string[]> {
  return page.locator(formControlSelector).evaluateAll((elements) => elements.flatMap((element, index) => {
    const control = element as HTMLInputElement | HTMLSelectElement | HTMLTextAreaElement;
    const labels = control.labels ? [...control.labels].map((label) => label.textContent ?? "").join(" ") : "";
    const name = (
      control.getAttribute("aria-label") ||
      labels ||
      control.getAttribute("title") ||
      control.getAttribute("placeholder") ||
      ""
    ).replace(/\s+/g, " ").trim();
    return name ? [] : [`${element.tagName.toLowerCase()}[${index}]`];
  }));
}

async function targetSizeFailures(page: Page, minimum: number): Promise<TargetSizeFailure[]> {
  return page.locator(targetSelector).evaluateAll((elements, requiredMinimum) => elements.flatMap((element) => {
    if (element.closest("[aria-hidden='true'], [inert]")) return [];
    const html = element as HTMLElement;
    const control = element as HTMLInputElement;
    const target = ["checkbox", "radio"].includes(control.type)
      ? control.labels?.[0] ?? element
      : element;
    const bounds = target.getBoundingClientRect();
    if (bounds.width >= requiredMinimum && bounds.height >= requiredMinimum) return [];
    const name = (
      html.getAttribute("aria-label") ||
      html.textContent ||
      html.getAttribute("title") ||
      html.getAttribute("placeholder") ||
      ""
    ).replace(/\s+/g, " ").trim();
    return [{
      name,
      tag: element.tagName,
      width: Number(bounds.width.toFixed(2)),
      height: Number(bounds.height.toFixed(2)),
    }];
  }), minimum);
}

async function assertKeyboardReachability(page: Page, label: string): Promise<void> {
  const negativeTabIndexControls = await page.locator([
    "a[href][tabindex='-1']:visible",
    "button[tabindex='-1']:visible:not(:disabled)",
    "input[tabindex='-1']:visible:not(:disabled)",
    "select[tabindex='-1']:visible:not(:disabled)",
    "textarea[tabindex='-1']:visible:not(:disabled)",
  ].join(", ")).evaluateAll((elements) => elements.map((element) => (
    element.getAttribute("aria-label") || element.textContent || element.tagName
  )?.replace(/\s+/g, " ").trim()));
  expect(negativeTabIndexControls).toEqual([]);

  const targetCount = await page.locator("body *").evaluateAll((elements) => {
    const targets = elements.filter((element) => {
      if (!(element instanceof HTMLElement)) return false;
      if (element.closest("[aria-hidden='true'], [inert]")) return false;
      if (
        (element instanceof HTMLButtonElement ||
          element instanceof HTMLInputElement ||
          element instanceof HTMLSelectElement ||
          element instanceof HTMLTextAreaElement) &&
        element.disabled
      ) return false;
      if (element.getAttribute("aria-disabled") === "true") return false;
      if (element instanceof HTMLInputElement && element.type === "radio" && element.name) {
        const group = [...document.querySelectorAll<HTMLInputElement>("input[type='radio']")]
          .filter((radio) => radio.name === element.name && radio.form === element.form && !radio.disabled);
        const sequentialTarget = group.find((radio) => radio.checked) ?? group[0];
        if (element !== sequentialTarget) return false;
      }
      const style = getComputedStyle(element);
      const rect = element.getBoundingClientRect();
      return element.tabIndex >= 0 &&
        style.display !== "none" &&
        style.visibility !== "hidden" &&
        rect.width > 0 &&
        rect.height > 0;
    });
    targets.forEach((element, index) => element.setAttribute("data-keyboard-contract-index", String(index)));
    return targets.length;
  });
  expect(targetCount).toBeGreaterThan(0);
  const keyboardTargets = page.locator("[data-keyboard-contract-index]");
  const reached = new Set<string>();
  const firstTarget = keyboardTargets.first();
  await firstTarget.focus();
  expect(
    await firstTarget.evaluate((element) => {
      const style = getComputedStyle(element);
      const hasPaintedIndicator =
        (style.outlineStyle !== "none" && Number.parseFloat(style.outlineWidth) > 0) ||
        style.boxShadow !== "none";
      return hasPaintedIndicator;
    }),
    `${label} first target must expose a visible keyboard focus indicator`,
  ).toBe(true);
  const firstIndex = await firstTarget.getAttribute("data-keyboard-contract-index");
  expect(firstIndex).not.toBeNull();
  if (firstIndex) reached.add(firstIndex);
  const traversalFailures: string[] = [];
  for (let index = 1; reached.size < targetCount && index < targetCount + 64; index += 1) {
    await page.keyboard.press("Tab");
    const activeState = await page.evaluate(() => {
      const element = document.activeElement;
      if (!(element instanceof HTMLElement) || element === document.body || element === document.documentElement) {
        return null;
      }
      const style = getComputedStyle(element);
      return {
        contractIndex: element.getAttribute("data-keyboard-contract-index"),
        disabled: (
          element instanceof HTMLButtonElement ||
          element instanceof HTMLInputElement ||
          element instanceof HTMLSelectElement ||
          element instanceof HTMLTextAreaElement
        ) && element.disabled,
        description: JSON.stringify({
          tag: element.tagName,
          className: element.className,
          tabIndex: element.tabIndex,
          focusVisible: element.matches(":focus-visible"),
          outline: `${style.outlineWidth} ${style.outlineStyle} ${style.outlineColor}`,
          boxShadow: style.boxShadow,
          name: (element.getAttribute("aria-label") || element.textContent || "")
            .replace(/\s+/g, " ")
            .trim()
            .slice(0, 160),
        }),
        isBrowserFocusableScroller: (
          ["auto", "scroll"].includes(style.overflowX) && element.scrollWidth > element.clientWidth
        ) || (
          ["auto", "scroll"].includes(style.overflowY) && element.scrollHeight > element.clientHeight
        ),
        focusIndicatorVisible: (
          (style.outlineStyle !== "none" && Number.parseFloat(style.outlineWidth) > 0) ||
          style.boxShadow !== "none"
        ),
      };
    });
    if (!activeState) {
      traversalFailures.push(`${label} lost document focus before reaching every target`);
      break;
    }
    expect(activeState.disabled, `${label} must not Tab to a disabled control`).toBe(false);
    expect(
      activeState.focusIndicatorVisible,
      `${label} must expose a visible keyboard focus indicator on every Tab target: ${activeState.description}`,
    ).toBe(true);
    if (activeState.contractIndex) {
      reached.add(activeState.contractIndex);
      continue;
    }
    if (!activeState.isBrowserFocusableScroller) {
      traversalFailures.push(
        `${label} reached an unexpected Tab stop ${index + 1}: ${activeState.description}`,
      );
      break;
    }
  }
  const missingTargets = await keyboardTargets.evaluateAll((elements, reachedIndexes) => elements.flatMap((element) => {
    const index = element.getAttribute("data-keyboard-contract-index");
    if (index && reachedIndexes.includes(index)) return [];
    const html = element as HTMLElement;
    return [{
      index,
      tag: element.tagName,
      className: html.className,
      tabIndex: html.tabIndex,
      name: (element.getAttribute("aria-label") || element.textContent || "")
        .replace(/\s+/g, " ")
        .trim()
        .slice(0, 160),
    }];
  }), [...reached]);
  expect({ traversalFailures, missingTargets }, `${label} must reach every sequential focus target`).toEqual({
    traversalFailures: [],
    missingTargets: [],
  });
  await page.locator("[data-keyboard-contract-index]").evaluateAll((elements) => {
    elements.forEach((element) => element.removeAttribute("data-keyboard-contract-index"));
  });
}

async function pageHeaderTopbarOverlaps(page: Page): Promise<string[]> {
  const topbar = page.locator(".application-topbar");
  if (await topbar.count() === 0 || !await topbar.isVisible()) return [];
  const topbarBox = await topbar.boundingBox();
  if (!topbarBox) return [];
  return page.locator(".workbench-page-header a:visible, .workbench-page-header button:visible")
    .evaluateAll((elements, bar) => elements.flatMap((element) => {
      const rect = element.getBoundingClientRect();
      const overlaps = rect.left < bar.x + bar.width &&
        rect.right > bar.x &&
        rect.top < bar.y + bar.height &&
        rect.bottom > bar.y;
      if (!overlaps) return [];
      const name = element.getAttribute("aria-label") || element.textContent || element.tagName;
      return [name.replace(/\s+/g, " ").trim()];
    }), topbarBox);
}

async function primaryNavigation(page: Page): Promise<Locator> {
  const navigation = page.getByRole("navigation", { name: "Primary role navigation" });
  await expect(navigation).toHaveCount(1);
  await expect(navigation.locator("[aria-current='page']")).toHaveCount(1);
  return navigation;
}

expect(VISUAL_SURFACES).toHaveLength(86);
expect(VISUAL_VIEWPORTS).toHaveLength(3);
expect(VISUAL_SURFACES.length * VISUAL_VIEWPORTS.length).toBe(258);

for (const viewport of VISUAL_VIEWPORTS) {
  test(`checks all 86 responsive routes at ${viewport.id}`, async ({ page }, testInfo) => {
    test.setTimeout(300_000);
    await page.setViewportSize(viewport);
    const consoleIssues: string[] = [];
    page.on("console", (message) => {
      if (message.type() === "error") consoleIssues.push(`console: ${message.text()}`);
    });
    page.on("pageerror", (error) => consoleIssues.push(`pageerror: ${error.message}`));
    let responsiveRouteChecks = 0;

    for (const surface of VISUAL_SURFACES) {
      consoleIssues.length = 0;
      await installDeterministicPageState(page);
      await driveReactSurface(page, surface);
      responsiveRouteChecks += 1;

      await assertSurfaceSemantics(page, resolveReactSurfaceSemantics(surface));
      expect.soft(pathname(page), `${surface.id}/${viewport.id} preserves its exact deep link`).toBe(surface.reactPath);
      await expect.soft(page.locator("main"), `${surface.id}/${viewport.id} exposes one main landmark`).toHaveCount(1);
      expect.soft(
        await page.getByRole("heading").count(),
        `${surface.id}/${viewport.id} exposes a heading`,
      ).toBeGreaterThan(0);
      await expect.soft(
        page.locator("[role='combobox']:not(select), [role='listbox']:not(select)"),
        `${surface.id}/${viewport.id} does not emulate a dropdown`,
      ).toHaveCount(0);
      expect.soft(
        await visibleFormControlsWithoutNames(page),
        `${surface.id}/${viewport.id} gives every native form control an accessible name`,
      ).toEqual([]);
      await assertKeyboardReachability(page, `${surface.id}/${viewport.id}`);
      expect.soft(
        await pageHeaderTopbarOverlaps(page),
        `${surface.id}/${viewport.id} keeps page-header actions clear of the application topbar`,
      ).toEqual([]);

      const minimumTargetDimension = viewport.id === "mobile" ? 44 : 24;
      expect.soft(
        await targetSizeFailures(page, minimumTargetDimension),
        `${surface.id}/${viewport.id} meets the ${minimumTargetDimension}px target-size boundary`,
      ).toEqual([]);
      const overflow = await page.evaluate(() => ({
        clientWidth: document.documentElement.clientWidth,
        scrollWidth: document.documentElement.scrollWidth,
      }));
      expect.soft(
        overflow.scrollWidth,
        `${surface.id}/${viewport.id} has no document overflow`,
      ).toBeLessThanOrEqual(overflow.clientWidth + 1);

      if (surface.id === "role-select") {
        await expect.soft(
          page.getByRole("navigation"),
          `${surface.id}/${viewport.id} has no navigation landmark`,
        ).toHaveCount(0);
      } else if (viewport.id === "mobile") {
        const opener = page.getByRole("button", { name: "Open navigation" });
        await expect(opener).toHaveAttribute("aria-expanded", "false");
        await expect(page.locator(".workspace-sidebar")).toHaveAttribute("aria-hidden", "true");
        await expect.soft(
          page.getByRole("navigation"),
          `${surface.id}/${viewport.id} has no accessible off-canvas navigation`,
        ).toHaveCount(0);
        await opener.click();
        const navigation = await primaryNavigation(page);
        await expect.soft(
          page.getByRole("navigation"),
          `${surface.id}/${viewport.id} exposes exactly one navigation landmark`,
        ).toHaveCount(1);
        await page.keyboard.press("Escape");
        await expect(opener).toBeFocused();
        await expect(opener).toHaveAttribute("aria-expanded", "false");

        await opener.click();
        const currentPath = pathname(page);
        const targetPath = await navigation.locator("a[href]").evaluateAll((links, activePath) => {
          const next = links.find((link) => new URL((link as HTMLAnchorElement).href).pathname !== activePath) ??
            links[0];
          return next ? new URL((next as HTMLAnchorElement).href).pathname : null;
        }, currentPath);
        expect(targetPath).not.toBeNull();
        await navigation.locator(`a[href="${targetPath}"]`).first().click();
        await expect(page.getByRole("dialog", { name: "Primary navigation" })).toHaveCount(0);
        expect(pathname(page)).toBe(targetPath);
      } else {
        await expect(page.locator(".workspace-sidebar")).not.toHaveAttribute("aria-hidden", "true");
        await primaryNavigation(page);
        await expect.soft(
          page.getByRole("navigation"),
          `${surface.id}/${viewport.id} exposes exactly one navigation landmark`,
        ).toHaveCount(1);
      }

      expect.soft(consoleIssues, `${surface.id}/${viewport.id} has zero console errors`).toEqual([]);
      await testInfo.attach(`responsive-route-${surface.id}-${viewport.id}`, {
        body: JSON.stringify({
          surfaceId: surface.id,
          viewport: viewport.id,
          path: surface.reactPath,
          consoleErrors: consoleIssues,
        }, null, 2),
        contentType: "application/json",
      });
    }

    expect(responsiveRouteChecks).toBe(86);
  });
}

test("traps mobile navigation focus and returns it after Escape", async ({ page }) => {
  await page.setViewportSize({ width: 390, height: 844 });
  await page.goto("/inspector/inspector-assignments");
  const opener = page.getByRole("button", { name: "Open navigation" });
  await opener.click();

  const dialog = page.getByRole("dialog", { name: "Primary navigation" });
  await expect(dialog).toHaveAttribute("aria-modal", "true");
  const close = dialog.getByRole("button", { name: "Close navigation" });
  await expect(close).toBeFocused();
  await page.keyboard.press("Shift+Tab");
  await expect(dialog.locator("a[href]").last()).toBeFocused();
  await page.keyboard.press("Escape");
  await expect(opener).toBeFocused();
  await expect(dialog).toHaveCount(0);
});

test("traps report-preview focus and returns it after Escape", async ({ page }) => {
  await page.goto("/department-manager/reports/RPT-CAB-2026-001-V1");
  const opener = page.getByRole("button", { name: "Review Full Report" });
  await expect(opener).toBeVisible();
  await opener.click();

  const dialog = page.getByRole("dialog", { name: "Immutable report preview" });
  const close = dialog.getByRole("button", { name: "Close report preview" });
  await expect(close).toBeFocused();
  await page.keyboard.press("Tab");
  await expect(close).toBeFocused();
  await page.keyboard.press("Escape");
  await expect(opener).toBeFocused();
  await expect(dialog).toHaveCount(0);
});
