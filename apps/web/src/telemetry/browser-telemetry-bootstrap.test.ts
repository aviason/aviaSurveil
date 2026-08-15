// @vitest-environment jsdom

import { afterEach, describe, expect, it, vi } from "vitest";

import type { BrowserTelemetry } from "./browser-telemetry";
import {
  classifyWebVitalEntry,
  currentBrowserRouteID,
  installBrowserTelemetry,
  observeBrowserNavigation,
} from "./browser-telemetry-bootstrap";

function telemetryStub(): BrowserTelemetry {
  return {
    recordNavigation: vi.fn(),
    recordWebVital: vi.fn(),
    recordAPIOutcome: vi.fn(),
    recordHandledError: vi.fn(),
    requestHeaders: vi.fn(() => ({})),
    flush: vi.fn(async () => ({ delivered: true, count: 0 })),
    shutdown: vi.fn(async () => undefined),
  };
}

function performanceEntry(
  entry: Partial<PerformanceEntry> & Record<string, unknown>,
): PerformanceEntry {
  return {
    name: "test",
    entryType: "test",
    startTime: 0,
    duration: 0,
    toJSON: () => ({}),
    ...entry,
  };
}

describe("browser navigation telemetry", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
    window.history.replaceState({}, "", "/");
  });

  it("keeps demo telemetry local when no collector is present", async () => {
    const send = vi.fn(async () => ({ ok: true }) as Response);
    vi.stubGlobal("fetch", send);

    const telemetry = installBrowserTelemetry("demo", "test");
    telemetry.recordAPIOutcome("read", "succeeded", "dashboard");
    await expect(telemetry.flush()).resolves.toEqual({
      delivered: true,
      count: 2,
    });

    expect(send).not.toHaveBeenCalled();
    window.dispatchEvent(new Event("pagehide"));
  });

  it("emits no records or network payloads when HTTP telemetry is explicitly disabled", async () => {
    const send = vi.fn(async () => ({ ok: true }) as Response);
    vi.stubGlobal("fetch", send);

    const telemetry = installBrowserTelemetry("http", "test", { disabled: true });
    telemetry.recordNavigation("admin-checklist-builder", "load");
    telemetry.recordAPIOutcome("read", "succeeded", "admin-checklist-builder");

    await expect(telemetry.flush()).resolves.toEqual({ delivered: true, count: 0 });
    expect(telemetry.requestHeaders()).toEqual({});
    expect(send).not.toHaveBeenCalled();
    window.dispatchEvent(new Event("pagehide"));
  });

  it("classifies LCP, CLS, and interaction latency without labeling first input as INP", () => {
    expect(classifyWebVitalEntry(performanceEntry({
      entryType: "largest-contentful-paint",
      startTime: 1800,
    }))).toEqual({ name: "LCP", value: 1800 });
    expect(classifyWebVitalEntry(performanceEntry({
      entryType: "layout-shift",
      value: 0.12,
      hadRecentInput: false,
    }))).toEqual({ name: "CLS", value: 0.12 });
    expect(classifyWebVitalEntry(performanceEntry({
      entryType: "layout-shift",
      value: 0.4,
      hadRecentInput: true,
    }))).toBeNull();
    expect(classifyWebVitalEntry(performanceEntry({
      entryType: "event",
      duration: 240,
      interactionId: 7,
    }))).toEqual({ name: "INP", value: 240 });
    expect(classifyWebVitalEntry(performanceEntry({
      entryType: "first-input",
      startTime: 20,
      duration: 10,
    }))).toBeNull();
  });

  it("records bounded route IDs for push, replace, and popstate navigation", () => {
    const telemetry = telemetryStub();
    const stop = observeBrowserNavigation(telemetry);

    window.history.pushState({}, "", "/inspector/findings");
    window.history.replaceState(
      {},
      "",
      "/inspector/findings/finding-42?source=private",
    );
    window.dispatchEvent(new PopStateEvent("popstate"));

    expect(telemetry.recordNavigation).toHaveBeenNthCalledWith(
      1,
      "inspector-findings",
      "push",
    );
    expect(telemetry.recordNavigation).toHaveBeenNthCalledWith(
      2,
      "finding-detail",
      "replace",
    );
    expect(telemetry.recordNavigation).toHaveBeenNthCalledWith(
      3,
      "finding-detail",
      "popstate",
    );
    expect(currentBrowserRouteID()).toBe("finding-detail");

    stop();
    window.history.pushState({}, "", "/");
    expect(telemetry.recordNavigation).toHaveBeenCalledTimes(3);
  });
});
