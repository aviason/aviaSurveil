export interface SupportDiagnosticsInput {
  versions: {
    appShellVersion: number;
    indexedDbSchemaVersion: number;
    packageSchemaVersion: number;
    syncProtocolVersion: number;
  };
  browser: { family: string; version: string; capabilityFingerprint: string };
  originId: string;
  storage: { persisted: boolean; quotaBytes: number; usageBytes: number };
  counts: { packages: number; pendingOperations: number; attachments: Record<string, number> };
  sync: { status: string; errorCode: string; lastReceiptId: string | null };
  recovery: { migrationState: string; updateState: string; quarantineCount: number; integrity: string };
  forbidden?: Record<string, unknown>;
}

export interface SupportDiagnostics {
  schemaVersion: 1;
  evidenceLabel: "candidate-only";
  versions: SupportDiagnosticsInput["versions"];
  browser: SupportDiagnosticsInput["browser"];
  originId: string;
  storage: SupportDiagnosticsInput["storage"];
  counts: SupportDiagnosticsInput["counts"];
  sync: { status: string; errorCode: string; lastReceiptId: string | null };
  recovery: SupportDiagnosticsInput["recovery"];
}

function safeCode(value: string, fallback: string): string {
  return /^[A-Za-z0-9_.:-]{1,128}$/u.test(value) ? value : fallback;
}

export function buildSupportDiagnostics(input: SupportDiagnosticsInput): SupportDiagnostics {
  return {
    schemaVersion: 1,
    evidenceLabel: "candidate-only",
    versions: { ...input.versions },
    browser: {
      family: safeCode(input.browser.family, "UNKNOWN_BROWSER"),
      version: safeCode(input.browser.version, "UNKNOWN_VERSION"),
      capabilityFingerprint: safeCode(input.browser.capabilityFingerprint, "UNKNOWN_CAPABILITIES"),
    },
    originId: safeCode(input.originId, "ORIGIN_UNCONFIGURED"),
    storage: {
      persisted: input.storage.persisted,
      quotaBytes: Math.max(0, Math.trunc(input.storage.quotaBytes)),
      usageBytes: Math.max(0, Math.trunc(input.storage.usageBytes)),
    },
    counts: {
      packages: Math.max(0, Math.trunc(input.counts.packages)),
      pendingOperations: Math.max(0, Math.trunc(input.counts.pendingOperations)),
      attachments: Object.fromEntries(
        Object.entries(input.counts.attachments)
          .filter(([key, value]) => /^[A-Z_]+$/u.test(key) && Number.isFinite(value))
          .map(([key, value]) => [key, Math.max(0, Math.trunc(value))]),
      ),
    },
    sync: {
      status: safeCode(input.sync.status, "UNKNOWN_SYNC_STATUS"),
      errorCode: safeCode(input.sync.errorCode, "UNKNOWN_SYNC_ERROR"),
      lastReceiptId: input.sync.lastReceiptId ? safeCode(input.sync.lastReceiptId, "REDACTED") : null,
    },
    recovery: {
      migrationState: safeCode(input.recovery.migrationState, "UNKNOWN_MIGRATION_STATE"),
      updateState: safeCode(input.recovery.updateState, "UNKNOWN_UPDATE_STATE"),
      quarantineCount: Math.max(0, Math.trunc(input.recovery.quarantineCount)),
      integrity: safeCode(input.recovery.integrity, "UNKNOWN_INTEGRITY"),
    },
  };
}
