// @vitest-environment jsdom
import { describe, expect, it, vi } from "vitest";

import {
  createSessionClient,
  parseCsrfCookie,
  safeReturnTo,
  type SessionProjection,
} from "./session-client";

const inspectorSession: SessionProjection = {
  subjectId: "154ec5ac-6f97-4f55-916f-d2f142fc6211",
  displayName: "Local Inspector",
  organizationId: "CAA",
  roles: ["inspector"],
};

function jsonResponse(value: unknown, init: ResponseInit = {}): Response {
  return new Response(JSON.stringify(value), {
    status: 200,
    headers: { "content-type": "application/json" },
    ...init,
  });
}

describe("SessionClient", () => {
  it("loads a safe same-origin session projection without provider credentials", async () => {
    const fetchImplementation = vi.fn<typeof fetch>().mockResolvedValue(
      jsonResponse({
        ...inspectorSession,
        csrfToken: "server-must-not-return-this",
        providerTokens: { accessToken: "server-only" },
      }),
    );
    const client = createSessionClient({ fetchImplementation });

    await expect(client.get()).resolves.toEqual(inspectorSession);
    expect(fetchImplementation).toHaveBeenCalledWith("/auth/session", {
      credentials: "same-origin",
      headers: expect.any(Headers),
      signal: undefined,
    });
  });

  it("rejects unknown, empty, or unsupported roles without fabricating a normal-session role", async () => {
    const fetchImplementation = vi.fn<typeof fetch>().mockResolvedValue(
      jsonResponse({
        subjectId: "subject-with-bad-role",
        displayName: "Bad Role",
        organizationId: "CAA",
        roles: ["inspector", "superAdmin"],
      }),
    );
    const client = createSessionClient({ fetchImplementation });

    await expect(client.get()).rejects.toMatchObject({
      code: "SESSION_ROLE_UNSUPPORTED",
    });
  });

  it("keeps returnTo same-origin and restricted to registered React paths", () => {
    expect(safeReturnTo("/inspector/inspector-assignments")).toBe(
      "/inspector/inspector-assignments",
    );
    expect(safeReturnTo("/department-manager/aga-demo-workspace/inspection")).toBe(
      "/department-manager/aga-demo-workspace/inspection",
    );
    expect(safeReturnTo("https://evil.example/lead-inspector/lead-review")).toBe("/");
    expect(safeReturnTo("//evil.example/lead-inspector/lead-review")).toBe("/");
    expect(safeReturnTo("/legacy-only-screen")).toBe("/");
    expect(safeReturnTo("")).toBe("/");
  });

  it("starts login with an encoded safe returnTo value", () => {
    const assign = vi.fn();
    const client = createSessionClient({
      location: { assign } as unknown as Location,
    });

    client.login("/lead-inspector/lead-review");

    expect(assign).toHaveBeenCalledWith(
      "/auth/login?returnTo=%2Flead-inspector%2Flead-review",
    );
  });

  it("parses only the same-origin CSRF cookie", () => {
    Object.defineProperty(document, "cookie", {
      configurable: true,
      value: "theme=light; __Host-avia_csrf=csrf-cookie-value; other=value",
    });

    expect(parseCsrfCookie()).toBe("csrf-cookie-value");
  });

  it("parses the local HTTP CSRF cookie used by the disposable preprod demo", () => {
    expect(parseCsrfCookie("theme=light; avia_csrf=local-csrf; other=value")).toBe("local-csrf");
  });

  it("prefers the local token when a stale secure cookie is still present", () => {
    expect(parseCsrfCookie("__Host-avia_csrf=stale; avia_csrf=current", "http:")).toBe("current");
  });

  it("sends logout with the CSRF header and same-origin credentials", async () => {
    Object.defineProperty(document, "cookie", {
      configurable: true,
      value: "__Host-avia_csrf=logout-csrf",
    });
    const fetchImplementation = vi.fn<typeof fetch>().mockResolvedValue(
      new Response(null, { status: 204 }),
    );
    const client = createSessionClient({ fetchImplementation });

    await client.logout();

    const [, init] = fetchImplementation.mock.calls[0]!;
    expect(fetchImplementation.mock.calls[0]?.[0]).toBe("/auth/logout");
    expect(init?.method).toBe("POST");
    expect(init?.credentials).toBe("same-origin");
    expect(new Headers(init?.headers).get("x-csrf-token")).toBe("logout-csrf");
  });
});
