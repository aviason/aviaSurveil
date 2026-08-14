// @vitest-environment jsdom
import "@testing-library/jest-dom/vitest";

import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";

import { LoginPage } from "./login-page";

afterEach(() => cleanup());

describe("LoginPage", () => {
  it("presents server-authorized access without local candidate labels", () => {
    const { container } = render(<LoginPage onLogin={() => undefined} />);

    expect(screen.getByRole("heading", { name: "Sign in to AviaSurveil360" })).toBeInTheDocument();
    expect(screen.getByText("Server-authorized access")).toBeInTheDocument();
    expect(container).not.toHaveTextContent(/candidate|local-preprod/i);
  });

  it("starts the organization identity redirect and exposes a supplied failure", async () => {
    const user = userEvent.setup();
    const onLogin = vi.fn();
    render(<LoginPage message="Identity service unavailable" onLogin={onLogin} />);

    expect(screen.getByRole("alert")).toHaveTextContent("Identity service unavailable");
    await user.click(screen.getByRole("button", { name: /Sign in with organization identity/i }));
    expect(onLogin).toHaveBeenCalledOnce();
  });
});
