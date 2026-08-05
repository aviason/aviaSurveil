import { REACT_ROUTE_CONTRACTS } from "../app/route-contracts";
import type { Role } from "../backend/backend";

export type IdentityMode =
  | "demo-role-switch"
  | "canonical-test-role-switch"
  | "oidc-session";

export interface SessionProjection {
  subjectId: string;
  displayName: string;
  organizationId: string;
  roles: Role[];
}

export type SessionState =
  | { status: "loading" }
  | { status: "unauthenticated" }
  | { status: "authenticated"; session: SessionProjection; activeRole: Role }
  | { status: "unavailable"; message: string }
  | { status: "expired" };

export interface SessionClient {
  get(signal?: AbortSignal): Promise<SessionProjection>;
  login(returnTo: string): void;
  logout(): Promise<void>;
  csrfToken(): string | null;
}

export class SessionClientError extends Error {
  constructor(
    readonly code: string,
    message: string,
  ) {
    super(message);
    this.name = "SessionClientError";
  }
}

const sessionPaths = new Set(REACT_ROUTE_CONTRACTS.map((contract) => contract.path));
const supplementalWorkspacePaths = [
  "/admin/aga-demo-workspace",
  "/department-manager/aga-demo-workspace",
  "/inspector/aga-demo-workspace",
  "/lead-inspector/aga-demo-workspace",
  "/auditee/aga-demo-workspace",
] as const;
const supportedRoles = new Set<Role>([
  "inspector",
  "leadInspector",
  "manager",
  "finance",
  "gm",
  "executiveDirector",
  "auditee",
  "admin",
]);

export function parseCsrfCookie(cookieSource = document.cookie): string | null {
  const match = cookieSource
    .split(";")
    .map((entry) => entry.trim())
    .find((entry) => entry.startsWith("__Host-avia_csrf="));
  if (!match) return null;
  return decodeURIComponent(match.slice("__Host-avia_csrf=".length));
}

export function safeReturnTo(rawReturnTo: string, origin = window.location.origin): string {
  try {
    const parsed = new URL(rawReturnTo || "/", origin);
    if (parsed.origin !== origin) return "/";
    const isRegisteredPath = sessionPaths.has(parsed.pathname) || supplementalWorkspacePaths.some(
      (path) => parsed.pathname === path || parsed.pathname.startsWith(`${path}/`),
    );
    if (!isRegisteredPath) return "/";
    return `${parsed.pathname}${parsed.search}`;
  } catch {
    return "/";
  }
}

function asTrimmedString(value: unknown): string {
  return typeof value === "string" ? value.trim() : "";
}

function parseSessionProjection(value: unknown): SessionProjection {
  if (!value || typeof value !== "object") {
    throw new SessionClientError("SESSION_PROJECTION_INVALID", "Session projection is invalid.");
  }
  const candidate = value as Record<string, unknown>;
  const roles = Array.isArray(candidate.roles) ? candidate.roles : [];
  if (roles.length === 0 || roles.some((role) => !supportedRoles.has(role as Role))) {
    throw new SessionClientError(
      "SESSION_ROLE_UNSUPPORTED",
      "The authenticated session does not contain a supported AviaSurveil360 role.",
    );
  }
  const projection: SessionProjection = {
    subjectId: asTrimmedString(candidate.subjectId),
    displayName: asTrimmedString(candidate.displayName),
    organizationId: asTrimmedString(candidate.organizationId),
    roles: roles.map((role) => role as Role),
  };
  if (!projection.subjectId || !projection.displayName || !projection.organizationId) {
    throw new SessionClientError("SESSION_PROJECTION_INVALID", "Session projection is incomplete.");
  }
  return projection;
}

export function createSessionClient({
  fetchImplementation = fetch,
  location = window.location,
}: {
  fetchImplementation?: typeof fetch;
  location?: Location;
} = {}): SessionClient {
  return {
    async get(signal) {
      const response = await fetchImplementation("/auth/session", {
        credentials: "same-origin",
        headers: new Headers({ Accept: "application/json" }),
        signal,
      });
      if (response.status === 401) {
        throw new SessionClientError("UNAUTHENTICATED", "Authentication is required.");
      }
      if (!response.ok) {
        throw new SessionClientError("SESSION_UNAVAILABLE", "Session projection is unavailable.");
      }
      return parseSessionProjection(await response.json());
    },
    login(returnTo) {
      location.assign(`/auth/login?returnTo=${encodeURIComponent(safeReturnTo(returnTo))}`);
    },
    async logout() {
      const headers = new Headers({ Accept: "application/json" });
      const token = parseCsrfCookie();
      if (token) headers.set("x-csrf-token", token);
      const response = await fetchImplementation("/auth/logout", {
        method: "POST",
        credentials: "same-origin",
        headers,
      });
      if (response.status === 401) return;
      if (!response.ok && response.status !== 204) {
        throw new SessionClientError("LOGOUT_FAILED", "Logout could not be completed.");
      }
    },
    csrfToken: () => parseCsrfCookie(),
  };
}
