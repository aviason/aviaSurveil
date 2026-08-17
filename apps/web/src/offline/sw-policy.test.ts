import { describe, expect, it } from "vitest";

import { classifyAppShellRequest } from "../sw";

describe("Service Worker request policy", () => {
  it.each([
    ["https://candidate.test/", "navigate", "app-shell-navigation"],
    ["https://candidate.test/inspector/audits/AUD-2026-001", "navigate", "app-shell-navigation"],
    ["https://candidate.test/assets/index-abcd1234.js", "no-cors", "versioned-static-asset"],
    ["https://candidate.test/assets/aviasurveil360-mark-abcd1234.png", "no-cors", "versioned-static-asset"],
    ["https://candidate.test/assets/air-traffic-control-abcd1234.svg", "no-cors", "versioned-static-asset"],
    ["https://candidate.test/assets/DMSans-Variable-abcd1234.ttf", "cors", "versioned-static-asset"],
    ["https://candidate.test/http-config.json", "cors", "network-only"],
  ] as const)("classifies %s as %s", (url, mode, expected) => {
    expect(
      classifyAppShellRequest(
        { url, method: "GET", mode },
        "https://candidate.test",
      ),
    ).toBe(expected);
  });

  it.each([
    ["https://candidate.test/v1/findings", "cors"],
    ["https://candidate.test/auth/session", "cors"],
    ["https://candidate.test/health/ready", "cors"],
    ["https://candidate.test/__test/reset", "cors"],
    ["https://candidate.test/reports/RPT-001.pdf", "cors"],
    ["https://other.test/assets/index-abcd1234.js", "no-cors"],
    ["https://candidate.test/auth/login?returnTo=%2Fadmin", "navigate"],
    ["https://candidate.test/auth/callback?state=opaque&code=opaque", "navigate"],
    [
      "https://candidate.test/identity/realms/aviasurveil360-local-preprod/protocol/openid-connect/auth?client_id=web",
      "navigate",
    ],
    [
      "https://candidate.test/identity/realms/aviasurveil360-local-preprod/.well-known/openid-configuration",
      "cors",
    ],
    ["https://candidate.test/api/v1/admin", "navigate"],
    ["https://candidate.test/v1/findings", "navigate"],
    ["https://candidate.test/health/ready", "navigate"],
    ["https://candidate.test/evidence-clean/object-version-1", "navigate"],
    ["https://candidate.test/evidence-quarantine/object-version-1", "navigate"],
    ["https://candidate.test/inspection-attachments/object-version-1", "navigate"],
    ["https://candidate.test/generated-documents/final-report.pdf", "navigate"],
    ["https://candidate.test/operations/dashboard", "navigate"],
    ["https://candidate.test/otel/v1/traces", "navigate"],
    ["https://candidate.test/private/object", "navigate"],
  ] as const)("never caches business, API, auth, health, test, or cross-origin request %s", (url, mode) => {
    expect(
      classifyAppShellRequest(
        { url, method: "GET", mode },
        "https://candidate.test",
      ),
    ).toBe("network-only");
  });

  it("never caches a mutation even when its path resembles a static asset", () => {
    expect(
      classifyAppShellRequest(
        {
          url: "https://candidate.test/assets/index-abcd1234.js",
          method: "POST",
          mode: "cors",
        },
        "https://candidate.test",
      ),
    ).toBe("network-only");
  });
});
