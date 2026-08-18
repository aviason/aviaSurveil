import { describe, expect, it, vi } from "vitest";

import type { OfflineGrant } from "../backend/backend";
import {
  CURRENT_OFFLINE_VERSIONS,
  assessOfflineReadiness,
  classifyLocalDataState,
  describeLocalPackageLoss,
  type OfflineReadinessDependencies,
  type OfflineReadinessInput,
} from "./storage-readiness";

const now = new Date("2026-07-21T08:00:00.000Z");

const grant: OfflineGrant = {
  grantId: "GRANT-CANDIDATE-001",
  subjectId: "USR-INSPECTOR-ALICE",
  organizationId: "ORG-FLY-NAMIBIA",
  packageId: "PKG-CAB-2026-001",
  packageVersion: 1,
  packageDigest: "sha256:candidate-cabin-package-v1",
  allowedCommandTypes: [
    "UPSERT_CHECKLIST_RESPONSE",
    "CREATE_POTENTIAL_FINDING",
    "SUBMIT_CHECKLIST",
    "REGISTER_INSPECTION_ATTACHMENT",
  ],
  assignmentScope: { questionIds: ["CAB-EMEQ-PBE-001"] },
  deviceInstanceId: "DEVICE-CANDIDATE-001",
  profileKeyId: "sha256:" + "a".repeat(64),
  assignmentRevision: 3,
  issuedAt: "2026-07-21T07:59:00.000Z",
  expiresAt: "2026-07-22T08:00:00.000Z",
  leaseIssuedAt: "2026-07-21T07:59:00.000Z",
  leaseExpiresAt: "2026-07-22T08:00:00.000Z",
  protocolVersion: 1,
};

function input(overrides: Partial<OfflineReadinessInput> = {}): OfflineReadinessInput {
  return {
    userInitiated: true,
    managedPolicyApproved: true,
    storageProfileApproved: true,
    expectedSubjectId: "USR-INSPECTOR-ALICE",
    expectedOrganizationId: "ORG-FLY-NAMIBIA",
    expectedDeviceInstanceId: "DEVICE-CANDIDATE-001",
    packageDescriptor: {
      packageId: "PKG-CAB-2026-001",
      packageVersion: 1,
      packageDigest: "sha256:candidate-cabin-package-v1",
      schemaVersion: 1,
      protocolVersion: 1,
      expiresAt: "2026-07-24T08:00:00.000Z",
    },
    offlineGrant: grant,
    versions: CURRENT_OFFLINE_VERSIONS,
    requiredAppShellVersion: CURRENT_OFFLINE_VERSIONS.appShellVersion,
    packageByteEstimate: 2 * 1024 * 1024,
    attachmentByteEstimate: 25 * 1024 * 1024,
    minimumHeadroomBytes: 50 * 1024 * 1024,
    now,
    ...overrides,
  };
}

function dependencies(
  overrides: Partial<OfflineReadinessDependencies> = {},
): OfflineReadinessDependencies {
  return {
    isSecureContext: true,
    browserSupported: true,
    serviceWorkerReady: vi.fn().mockResolvedValue(true),
    indexedDbCanary: vi.fn().mockResolvedValue(true),
    opfsCanary: vi.fn().mockResolvedValue(true),
    restartCanary: vi.fn().mockResolvedValue(true),
    storagePersisted: vi.fn().mockResolvedValue(true),
    requestPersistence: vi.fn().mockResolvedValue(true),
    estimateStorage: vi.fn().mockResolvedValue({
      usage: 10 * 1024 * 1024,
      quota: 512 * 1024 * 1024,
    }),
    ...overrides,
  };
}

describe("assessOfflineReadiness", () => {
  it.each([
    ["unsupported-browser", {}, { isSecureContext: false }],
    ["managed-policy-unapproved", { managedPolicyApproved: false }, {}],
    ["ephemeral-or-unmanaged-storage", { storageProfileApproved: false }, {}],
    ["service-worker-unavailable", {}, { serviceWorkerReady: vi.fn().mockResolvedValue(false) }],
    ["indexeddb-health-failed", {}, { indexedDbCanary: vi.fn().mockResolvedValue(false) }],
    ["opfs-health-failed", {}, { opfsCanary: vi.fn().mockResolvedValue(false) }],
    [
      "persistence-denied",
      {},
      {
        storagePersisted: vi.fn().mockResolvedValue(false),
        requestPersistence: vi.fn().mockResolvedValue(false),
      },
    ],
    [
      "quota-insufficient",
      {},
      { estimateStorage: vi.fn().mockResolvedValue({ usage: 90, quota: 100 }) },
    ],
    [
      "offline-grant-invalid",
      { offlineGrant: { ...grant, expiresAt: "2026-07-21T07:59:59.000Z" } },
      {},
    ],
    [
      "offline-grant-invalid",
      { offlineGrant: { ...grant, leaseExpiresAt: "2026-07-29T08:00:00.000Z" } },
      {},
    ],
    [
      "app-version-incompatible",
      { requiredAppShellVersion: CURRENT_OFFLINE_VERSIONS.appShellVersion + 2 },
      {},
    ],
    [
      "schema-version-incompatible",
      {
        packageDescriptor: {
          ...input().packageDescriptor,
          schemaVersion: CURRENT_OFFLINE_VERSIONS.packageSchemaVersion + 2,
        },
      },
      {},
    ],
    [
      "protocol-version-incompatible",
      {
        packageDescriptor: {
          ...input().packageDescriptor,
          protocolVersion: CURRENT_OFFLINE_VERSIONS.syncProtocolVersion + 2,
        },
        offlineGrant: {
          ...grant,
          protocolVersion: CURRENT_OFFLINE_VERSIONS.syncProtocolVersion + 2,
        },
      },
      {},
    ],
  ] as const)("returns %s without claiming readiness", async (code, inputOverride, dependencyOverride) => {
    const result = await assessOfflineReadiness(
      input(inputOverride as Partial<OfflineReadinessInput>),
      dependencies(dependencyOverride as Partial<OfflineReadinessDependencies>),
    );

    expect(result.code).toBe(code);
    expect(result.ready).toBe(false);
    expect(result.recoveryAction.length).toBeGreaterThan(10);
  });

  it("returns ready only after all checks and describes capacity as advisory", async () => {
    const result = await assessOfflineReadiness(input(), dependencies());

    expect(result).toMatchObject({
      code: "ready",
      ready: true,
      capacityIsAdvisory: true,
    });
    expect(result.requiredBytes).toBe(77 * 1024 * 1024);
    expect(result.availableBytes).toBe(502 * 1024 * 1024);
  });

  it("requests persistence only from the user-initiated checkout flow", async () => {
    const requestPersistence = vi.fn().mockResolvedValue(true);
    const result = await assessOfflineReadiness(
      input(),
      dependencies({
        storagePersisted: vi.fn().mockResolvedValue(false),
        requestPersistence,
      }),
    );

    expect(result.code).toBe("ready");
    expect(requestPersistence).toHaveBeenCalledOnce();

    const passiveRequest = vi.fn().mockResolvedValue(true);
    const passive = await assessOfflineReadiness(
      input({ userInitiated: false }),
      dependencies({
        storagePersisted: vi.fn().mockResolvedValue(false),
        requestPersistence: passiveRequest,
      }),
    );
    expect(passive.code).toBe("persistence-denied");
    expect(passiveRequest).not.toHaveBeenCalled();
  });

  it("requires a canary observed after a browser restart", async () => {
    const result = await assessOfflineReadiness(
      input(),
      dependencies({ restartCanary: vi.fn().mockResolvedValue(false) }),
    );

    expect(result.code).toBe("ephemeral-or-unmanaged-storage");
    expect(result.recoveryAction).toMatch(/restart.*browser/i);
  });

  it("runs the final runtime canary after capability and quota gates", async () => {
    const events: string[] = [];
    const result = await assessOfflineReadiness(
      input(),
      dependencies({
        browserAdmission: vi.fn().mockImplementation(async () => {
          events.push("browser");
          return { official: true, state: "OFFICIAL", reasonCode: "OFFICIAL_BROWSER" };
        }),
        serviceWorkerReady: vi.fn().mockImplementation(async () => {
          events.push("service-worker");
          return true;
        }),
        indexedDbCanary: vi.fn().mockImplementation(async () => {
          events.push("indexeddb");
          return true;
        }),
        opfsCanary: vi.fn().mockImplementation(async () => {
          events.push("opfs");
          return true;
        }),
        estimateStorage: vi.fn().mockImplementation(async () => {
          events.push("quota");
          return { usage: 10 * 1024 * 1024, quota: 512 * 1024 * 1024 };
        }),
        runtimeCanary: vi.fn().mockImplementation(async () => {
          events.push("runtime");
          return true;
        }),
      }),
    );

    expect(result).toMatchObject({ code: "ready", ready: true, admissionState: "OFFLINE_READY" });
    expect(events).toEqual(["browser", "service-worker", "indexeddb", "opfs", "quota", "runtime"]);
  });

  it("fails closed when the runtime canary cannot prove restart/readback", async () => {
    const result = await assessOfflineReadiness(
      input(),
      dependencies({ runtimeCanary: vi.fn().mockResolvedValue(false) }),
    );

    expect(result).toMatchObject({
      code: "runtime-canary-failed",
      ready: false,
      admissionState: "RECOVERY_REQUIRED",
    });
  });

  it("fails a grant whose subject, device, package, digest, or scope is not exact", async () => {
    for (const invalidGrant of [
      { ...grant, subjectId: "USR-OTHER" },
      { ...grant, deviceInstanceId: "DEVICE-OTHER" },
      { ...grant, packageId: "PKG-OTHER" },
      { ...grant, packageDigest: "sha256:other" },
      { ...grant, assignmentScope: { questionIds: [] } },
    ]) {
      const result = await assessOfflineReadiness(
        input({ offlineGrant: invalidGrant }),
        dependencies(),
      );
      expect(result.code).toBe("offline-grant-invalid");
    }
  });

  it("states the irrecoverable boundary after explicit site-data deletion", () => {
    expect(classifyLocalDataState({ outstandingCheckout: true, localPackagePresent: false })).toBe(
      "LOCAL_DATA_CLEARED",
    );
    expect(describeLocalPackageLoss({ outstandingCheckout: true, localPackagePresent: false })).toBe(
      "Local package missing. Unsynced single-device work cannot be recovered after site data is cleared.",
    );
    expect(describeLocalPackageLoss({ outstandingCheckout: false, localPackagePresent: false })).toBeNull();
  });
});
