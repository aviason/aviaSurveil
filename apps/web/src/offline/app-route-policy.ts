export const NETWORK_ONLY_ROUTE_PREFIXES = Object.freeze([
  "/api/",
  "/v1/",
  "/auth/",
  "/identity/",
  "/health/",
  "/operations/",
  "/otel/",
  "/__test/",
  "/evidence-quarantine/",
  "/evidence-clean/",
  "/inspection-attachments/",
  "/generated-documents/",
  "/object-store/",
  "/private/",
]);

// This is deliberately a positive registry. New React routes must be added
// here and to the route-contract tests before a worker can serve them offline.
export const APPLICATION_ROUTE_PREFIXES = Object.freeze([
  "/admin",
  "/auditee",
  "/department-manager",
  "/executive-director",
  "/finance",
  "/general-manager",
  "/inspector",
  "/lead-inspector",
  "/manager",
]);

export function isNetworkOnlyPath(pathname: string): boolean {
  return pathname === "/http-config.json" || pathname === "/app-shell-assets.json" || pathname === "/app-shell-recovery.html" || NETWORK_ONLY_ROUTE_PREFIXES.some((prefix) => pathname === prefix.slice(0, -1) || pathname.startsWith(prefix));
}

export function normalizeApplicationPath(pathname: string): string | null {
  if (!pathname.startsWith("/") || pathname.includes("\\") || /%2f|%5c/i.test(pathname)) return null;
  let decoded: string;
  try {
    decoded = decodeURIComponent(pathname);
  } catch {
    return null;
  }
  if (decoded !== pathname || decoded.includes("\\") || decoded.split("/").some((part) => part === "." || part === "..")) return null;
  const normalized = pathname.replace(/\/+/g, "/");
  if (normalized !== pathname || normalized.length > 1 && normalized.endsWith("/")) return null;
  return normalized;
}

export function isRegisteredApplicationRoute(pathname: string): boolean {
  const normalized = normalizeApplicationPath(pathname);
  if (normalized === null || isNetworkOnlyPath(normalized)) return false;
  if (normalized === "/") return true;
  return APPLICATION_ROUTE_PREFIXES.some((prefix) => normalized === prefix || normalized.startsWith(`${prefix}/`));
}

export function isContentHashedAssetPath(pathname: string): boolean {
  return /^\/assets\/[A-Za-z0-9_.-]+-[A-Za-z0-9_-]{6,}\.(?:css|js|map|svg|png|jpg|jpeg|webp|ttf|woff|woff2)$/.test(pathname);
}
