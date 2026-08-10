// @vitest-environment jsdom
import "@testing-library/jest-dom/vitest";

import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";

import { RouteLoadFailure, RouteLoadingState } from "./route-loader";

afterEach(cleanup);

describe("route load states", () => {
  it("renders a branded and announced loading state", () => {
    render(<RouteLoadingState />);
    expect(screen.getByRole("status")).toHaveTextContent("Loading workspace");
    expect(screen.getByText("Restoring the secured route and its current session.")).toBeVisible();
  });

  it("offers a working full-application reload after a route chunk failure", async () => {
    const reload = vi.fn();
    render(<RouteLoadFailure onReload={reload} />);
    expect(screen.getByRole("alert")).toHaveTextContent("This workspace route could not be loaded");
    await userEvent.click(screen.getByRole("button", { name: "Reload application" }));
    expect(reload).toHaveBeenCalledOnce();
  });
});
