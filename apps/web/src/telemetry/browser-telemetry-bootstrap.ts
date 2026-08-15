import { REACT_ROUTE_CONTRACTS } from "../app/route-contracts";
import {
  activateBrowserTelemetry,
  createBrowserTelemetry,
  type BrowserTelemetry,
} from "./browser-telemetry";

interface BrowserTelemetryInstallOptions {
  disabled?: boolean;
}

function disabledBrowserTelemetry(): BrowserTelemetry {
  return {
    recordNavigation: () => undefined,
    recordWebVital: () => undefined,
    recordAPIOutcome: () => undefined,
    recordHandledError: () => undefined,
    requestHeaders: () => ({}),
    flush: async () => ({ delivered: true, count: 0 }),
    shutdown: async () => undefined,
  };
}

export function installBrowserTelemetry(
  buildProfile: "demo" | "http",
  serviceVersion = "runtime",
  options: BrowserTelemetryInstallOptions = {},
): BrowserTelemetry {
  if (options.disabled) {
    const telemetry = disabledBrowserTelemetry();
    const deactivate = activateBrowserTelemetry(telemetry, currentBrowserRouteID);
    window.addEventListener("pagehide", deactivate, { once: true });
    return telemetry;
  }
  const telemetry = createBrowserTelemetry({
    buildProfile,
    serviceVersion,
    transport:
      buildProfile === "demo"
        ? {
            send: async () => undefined,
          }
        : undefined,
    autoFlush: true,
  });
  const deactivate = activateBrowserTelemetry(telemetry, currentBrowserRouteID);
  const stopNavigation = observeBrowserNavigation(telemetry);
  telemetry.recordNavigation(currentBrowserRouteID(), "load");
  window.addEventListener(
    "pagehide",
    () => {
      stopNavigation();
      deactivate();
      void telemetry.shutdown();
    },
    { once: true },
  );

  if ("PerformanceObserver" in window) {
    for (const entryType of ["largest-contentful-paint", "layout-shift", "event"]) {
      try {
        const observer = new PerformanceObserver((list) => {
          for (const entry of list.getEntries()) {
            const vital = classifyWebVitalEntry(entry);
            if (!vital) continue;
            telemetry.recordWebVital(
              vital.name,
              vital.value,
              webVitalRating(vital.name, vital.value),
              currentBrowserRouteID(),
            );
          }
        });
        observer.observe({
          type: entryType,
          buffered: true,
          ...(entryType === "event" ? { durationThreshold: 40 } : {}),
        });
      } catch {
        // Unsupported performance entries never block application startup.
      }
    }
  }
  return telemetry;
}

export function currentBrowserRouteID(): string {
  const path = window.location.pathname;
  const route = REACT_ROUTE_CONTRACTS.find((contract) => {
    const templateSegments = contract.path.split("/").filter(Boolean);
    const pathSegments = path.split("/").filter(Boolean);
    return templateSegments.length === pathSegments.length && templateSegments.every((segment, index) => segment.startsWith(":") || segment === pathSegments[index]);
  });
  return route?.id ?? "unknown";
}

export function observeBrowserNavigation(
  telemetry: BrowserTelemetry,
): () => void {
  const originalPushState = window.history.pushState;
  const originalReplaceState = window.history.replaceState;
  const pushState: History["pushState"] = function (...arguments_) {
    originalPushState.apply(window.history, arguments_);
    telemetry.recordNavigation(currentBrowserRouteID(), "push");
  };
  const replaceState: History["replaceState"] = function (...arguments_) {
    originalReplaceState.apply(window.history, arguments_);
    telemetry.recordNavigation(currentBrowserRouteID(), "replace");
  };
  const onPopState = () => {
    telemetry.recordNavigation(currentBrowserRouteID(), "popstate");
  };
  window.history.pushState = pushState;
  window.history.replaceState = replaceState;
  window.addEventListener("popstate", onPopState);

  return () => {
    if (window.history.pushState === pushState) {
      window.history.pushState = originalPushState;
    }
    if (window.history.replaceState === replaceState) {
      window.history.replaceState = originalReplaceState;
    }
    window.removeEventListener("popstate", onPopState);
  };
}

export function classifyWebVitalEntry(
  entry: PerformanceEntry,
): { name: "LCP" | "CLS" | "INP"; value: number } | null {
  if (entry.entryType === "largest-contentful-paint") {
    return { name: "LCP", value: entry.startTime };
  }
  if (entry.entryType === "layout-shift") {
    const layoutShift = entry as PerformanceEntry & {
      value?: number;
      hadRecentInput?: boolean;
    };
    if (layoutShift.hadRecentInput) return null;
    return { name: "CLS", value: layoutShift.value ?? 0 };
  }
  if (entry.entryType === "event") {
    const eventTiming = entry as PerformanceEntry & { interactionId?: number };
    if (!eventTiming.interactionId) return null;
    return { name: "INP", value: entry.duration };
  }
  return null;
}

function webVitalRating(name: string, value: number): string {
  const thresholds: Record<string, readonly [number, number]> = {
    LCP: [2500, 4000],
    CLS: [0.1, 0.25],
    INP: [200, 500],
  };
  const [good, poor] = thresholds[name] ?? [0, 0];
  if (value <= good) return "good";
  if (value <= poor) return "needs-improvement";
  return "poor";
}
