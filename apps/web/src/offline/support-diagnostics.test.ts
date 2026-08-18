import { describe, expect, it } from "vitest";

import { buildSupportDiagnostics } from "./support-diagnostics";

describe("privacy-safe support diagnostics", () => {
  it("projects only bounded operational fields and excludes sensitive local content", () => {
    const result = buildSupportDiagnostics({
      versions: { appShellVersion: 9, indexedDbSchemaVersion: 2, packageSchemaVersion: 1, syncProtocolVersion: 1 },
      browser: { family: "windows-chrome", version: "stable-n", capabilityFingerprint: "cap-1" },
      originId: "ORIGIN-DEMO-ONLY",
      storage: { persisted: true, quotaBytes: 100, usageBytes: 20 },
      counts: { packages: 1, pendingOperations: 2, attachments: { LOCAL_READY: 1 } },
      sync: { status: "synchronized", errorCode: "NONE", lastReceiptId: "receipt-1" },
      recovery: { migrationState: "OPEN", updateState: "SAFE", quarantineCount: 0, integrity: "PASS" },
      forbidden: {
        token: "secret-token",
        answer: "NON_COMPLIANT",
        note: "private note",
        filename: "secret.pdf",
        path: "/Users/private/secret.pdf",
        uploadUrl: "https://signed.example/secret",
        serverError: "database password leaked",
      },
    });

    const encoded = JSON.stringify(result);
    expect(result).toMatchObject({ originId: "ORIGIN-DEMO-ONLY", sync: { lastReceiptId: "receipt-1" } });
    expect(encoded).not.toContain("secret-token");
    expect(encoded).not.toContain("NON_COMPLIANT");
    expect(encoded).not.toContain("private note");
    expect(encoded).not.toContain("secret.pdf");
    expect(encoded).not.toContain("signed.example");
    expect(encoded).not.toContain("database password");
  });
});
