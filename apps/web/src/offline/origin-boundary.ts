export type OriginBoundaryResult =
  | { ok: true; origin: string }
  | { ok: false; code: "ORIGIN_UNCONFIGURED" | "ORIGIN_INVALID" | "ORIGIN_MISMATCH" };

function parseOrigin(value: string): string | null {
  try {
    const parsed = new URL(value);
    if (!parsed.protocol || !parsed.hostname || parsed.username || parsed.password || parsed.pathname !== "/" || parsed.search || parsed.hash) {
      return null;
    }
    return parsed.origin;
  } catch {
    return null;
  }
}

export function assertExactOrigin(actual: string, expected: string): OriginBoundaryResult {
  if (!expected.trim()) return { ok: false, code: "ORIGIN_UNCONFIGURED" };
  const expectedOrigin = parseOrigin(expected);
  const actualOrigin = parseOrigin(actual);
  if (!expectedOrigin || !actualOrigin) return { ok: false, code: "ORIGIN_INVALID" };
  return actualOrigin === expectedOrigin
    ? { ok: true, origin: actualOrigin }
    : { ok: false, code: "ORIGIN_MISMATCH" };
}
