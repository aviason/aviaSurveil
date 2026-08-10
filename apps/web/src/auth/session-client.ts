import {
  CANONICAL_AUDIT_PREPARATION_PATH,
  CANONICAL_INSPECTOR_AUDIT_PATH,
  CANONICAL_INSPECTOR_CHECKLIST_PATH,
  CANONICAL_QUESTION_REVIEW_PATH,
  REACT_ROUTE_CONTRACTS,
} from "../app/route-contracts";
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
const canonicalSessionPathPatterns = [
  CANONICAL_QUESTION_REVIEW_PATH,
  CANONICAL_AUDIT_PREPARATION_PATH,
  CANONICAL_INSPECTOR_AUDIT_PATH,
  CANONICAL_INSPECTOR_CHECKLIST_PATH,
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

export function parseCsrfCookie(cookieSource = document.cookie, protocol = window.location.protocol): string | null {
  // Local-preprod is served over plain HTTP and intentionally uses the
  // unprefixed cookie names. Prefer that cookie when a browser still carries a
  // stale secure cookie from an earlier run; otherwise the stale token would
  // be sent in the header while the API validates the local cookie.
  const prefixes = protocol === "http:"
    ? ["avia_csrf=", "__Host-avia_csrf="]
    : ["__Host-avia_csrf=", "avia_csrf="];
  const entries = cookieSource.split(";").map((entry) => entry.trim());
  for (const prefix of prefixes) {
    const match = entries.find((entry) => entry.startsWith(prefix));
    if (match) return decodeURIComponent(match.slice(prefix.length));
  }
  return null;
}

export function safeReturnTo(rawReturnTo: string, origin = window.location.origin): string {
  try {
    const parsed = new URL(rawReturnTo || "/", origin);
    if (parsed.origin !== origin) return "/";
    const pathSegments = parsed.pathname.split("/");
    const matchesCanonicalPath = canonicalSessionPathPatterns.some((pattern) => {
      const patternSegments = pattern.split("/");
      return patternSegments.length === pathSegments.length && patternSegments.every(
        (segment, index) => segment.startsWith(":")
          ? Boolean(pathSegments[index])
          : segment === pathSegments[index],
      );
    });
    const isRegisteredPath = sessionPaths.has(parsed.pathname) || matchesCanonicalPath;
    if (!isRegisteredPath) return "/";
    return `${parsed.pathname}${parsed.search}`;
  } catch {
    return "/";
  }
}

function asTrimmedString(value: unknown): string {
	return typeof value === "string" ? value.trim() : "";
}

export function safeProviderLogoutURL(
	rawLogoutURL: unknown,
	origin = window.location.origin,
): string {
	const value = asTrimmedString(rawLogoutURL);
	try {
		const parsed = new URL(value, origin);
		if (
			!value ||
			parsed.origin !== origin ||
			parsed.pathname !== "/auth/provider-logout" ||
			parsed.username ||
			parsed.password ||
			parsed.hash ||
			!parsed.searchParams.get("ticket") ||
			Array.from(parsed.searchParams.keys()).some((key) => key !== "ticket")
		) {
			throw new Error("unsafe provider logout URL");
		}
		return parsed.toString();
	} catch {
		throw new SessionClientError(
			"LOGOUT_RESPONSE_INVALID",
			"The identity provider logout response is invalid.",
		);
	}
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
		if (!response.ok) {
			throw new SessionClientError("LOGOUT_FAILED", "Logout could not be completed.");
		}
		let payload: unknown;
		try {
			payload = await response.json();
		} catch {
			throw new SessionClientError(
				"LOGOUT_RESPONSE_INVALID",
				"The identity provider logout response is invalid.",
			);
		}
		const logoutURL = safeProviderLogoutURL(
			(payload as { logoutUrl?: unknown } | null)?.logoutUrl,
			location.origin || window.location.origin,
		);
		location.assign(logoutURL);
	},
    csrfToken: () => parseCsrfCookie(),
  };
}
