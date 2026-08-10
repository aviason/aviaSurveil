import { afterEach, describe, expect, it, vi } from "vitest";

import { ROUTE_LOAD_TIMEOUT_MS, loadRouteWithTimeout } from "./route-loader";

describe("route chunk loading", () => {
  afterEach(() => {
    vi.useRealTimers();
  });

  it("rejects a route chunk that never settles instead of leaving the app on an infinite loading screen", async () => {
    vi.useFakeTimers();
    const pending = loadRouteWithTimeout(
      () => new Promise<{ default: () => null }>(() => undefined),
    );
    const rejection = expect(pending).rejects.toThrow("The requested workspace route could not be loaded");

    await vi.advanceTimersByTimeAsync(ROUTE_LOAD_TIMEOUT_MS);

    await rejection;
  });

  it("preserves the original route load failure", async () => {
    const failure = new Error("chunk request failed");
    await expect(loadRouteWithTimeout(async () => { throw failure; })).rejects.toBe(failure);
  });
});
