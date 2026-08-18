// @vitest-environment jsdom
import "@testing-library/jest-dom/vitest";

import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";

import type { InspectionPackage, OfflineGrant } from "../../backend/backend";
import { CURRENT_OFFLINE_VERSIONS } from "../../offline/storage-readiness";
import {
  OfflineReadinessPanel,
  type OfflineReadinessPanelRuntime,
} from "./offline-readiness-panel";
import type { ProfileAuthority } from "../../offline/profile-authority";

const packageView: InspectionPackage = {
  id: "PKG-CAB-2026-001",
  auditId: "AUD-2026-001",
  organizationId: "ORG-FLY-NAMIBIA",
  organizationName: "Fly Namibia",
  title: "Cabin Inspection",
  packageVersion: 1,
  schemaVersion: 1,
  protocolVersion: 1,
  templateVersionId: "TPL-CAB-001-V1",
  packageDigest: "sha256:candidate-cabin-package-v1",
  expiresAt: "2026-07-24T08:00:00.000Z",
  checklistStatus: "IN_PROGRESS",
  checklistRevision: 1,
  questions: [],
};

const grant: OfflineGrant = {
  grantId: "GRANT-CANDIDATE-001",
  subjectId: "USR-INSPECTOR-AMINA",
  organizationId: packageView.organizationId,
  packageId: packageView.id,
  packageVersion: packageView.packageVersion,
  packageDigest: packageView.packageDigest,
  allowedCommandTypes: ["UPSERT_CHECKLIST_RESPONSE"],
  assignmentScope: { questionIds: ["CAB-EMEQ-PBE-001"] },
  deviceInstanceId: "DEVICE-CANDIDATE-001",
  profileKeyId: "sha256:" + "a".repeat(64),
  assignmentRevision: 1,
  issuedAt: "2026-07-21T07:59:00.000Z",
  expiresAt: "2026-07-22T08:00:00.000Z",
  leaseIssuedAt: "2026-07-21T07:59:00.000Z",
  leaseExpiresAt: "2026-07-22T08:00:00.000Z",
  protocolVersion: 1,
};

const profileAuthority: ProfileAuthority = {
  subjectId: grant.subjectId,
  profileKeyId: grant.profileKeyId!,
  publicJwk: { kty: "EC", crv: "P-256", x: "x", y: "y" },
  sign: vi.fn().mockResolvedValue("proof"),
  verify: vi.fn().mockResolvedValue(true),
};

afterEach(cleanup);

function runtime(overrides: Partial<OfflineReadinessPanelRuntime> = {}): OfflineReadinessPanelRuntime {
  return {
    checkout: vi.fn().mockResolvedValue({ inspectionPackage: packageView, offlineGrant: grant }),
    dependencies: {
      isSecureContext: true,
      browserSupported: true,
      serviceWorkerReady: vi.fn().mockResolvedValue(true),
      indexedDbCanary: vi.fn().mockResolvedValue(true),
      opfsCanary: vi.fn().mockResolvedValue(true),
      restartCanary: vi.fn().mockResolvedValue(true),
      storagePersisted: vi.fn().mockResolvedValue(true),
      requestPersistence: vi.fn().mockResolvedValue(true),
      estimateStorage: vi.fn().mockResolvedValue({ usage: 0, quota: 512 * 1024 * 1024 }),
    },
    getDeviceInstanceId: vi.fn().mockResolvedValue(grant.deviceInstanceId),
    getProfileAuthority: vi.fn().mockResolvedValue(profileAuthority),
    readSnapshot: vi.fn().mockResolvedValue(null),
    writeSnapshot: vi.fn().mockResolvedValue(undefined),
    now: () => new Date("2026-07-21T08:00:00.000Z"),
    ...overrides,
  };
}

describe("OfflineReadinessPanel", () => {
  it("blocks without managed-policy attestation and preserves online use", async () => {
    const testRuntime = runtime();
    render(
      <OfflineReadinessPanel
        inspectionPackage={packageView}
        subjectId="USR-INSPECTOR-AMINA"
        runtime={testRuntime}
      />,
    );

    await userEvent.click(screen.getByRole("button", { name: "Check out for offline use" }));

    expect(await screen.findByText(/managed browser, device, and profile policy/i)).toBeVisible();
    expect(testRuntime.checkout).not.toHaveBeenCalled();
    expect(screen.getByText(/Online inspection remains available/i)).toBeVisible();
    expect(screen.getByText(/If the server reports an outstanding checkout/i)).toBeVisible();
  });

  it("stores the exact subject-scoped package and grant only after every readiness check", async () => {
    const testRuntime = runtime();
    render(
      <OfflineReadinessPanel
        inspectionPackage={packageView}
        subjectId="USR-INSPECTOR-AMINA"
        runtime={testRuntime}
      />,
    );

    await userEvent.click(screen.getByLabelText(/official browser\/version lane/i));
    await userEvent.click(screen.getByLabelText(/encrypted managed profile/i));
    await userEvent.click(screen.getByRole("button", { name: "Check out for offline use" }));

    expect(await screen.findByText("Ready for official offline checkout")).toBeVisible();
    expect(testRuntime.checkout).toHaveBeenCalledWith({
      operationId: "OP-OFFLINE-CHECKOUT-PKG-CAB-2026-001-DEVICE-CANDIDATE-001",
      packageId: packageView.id,
      expectedPackageVersion: 1,
      deviceInstanceId: "DEVICE-CANDIDATE-001",
      profileKeyId: grant.profileKeyId,
      profilePublicJwk: profileAuthority.publicJwk,
    });
    expect(testRuntime.writeSnapshot).toHaveBeenCalledWith({
      subjectId: "USR-INSPECTOR-AMINA",
      inspectionPackage: packageView,
      offlineGrant: grant,
      checkedOutAt: "2026-07-21T08:00:00.000Z",
      versions: CURRENT_OFFLINE_VERSIONS,
    });
    expect(screen.getByText(/capacity estimate is advisory/i)).toBeVisible();
  });

  it("renders an existing exact-subject snapshot and the site-data loss boundary", async () => {
    const testRuntime = runtime({
      readSnapshot: vi.fn().mockResolvedValue({
        subjectId: "USR-INSPECTOR-AMINA",
        inspectionPackage: packageView,
        offlineGrant: grant,
        checkedOutAt: "2026-07-21T08:00:00.000Z",
        versions: CURRENT_OFFLINE_VERSIONS,
      }),
    });
    render(
      <OfflineReadinessPanel
        inspectionPackage={packageView}
        subjectId="USR-INSPECTOR-AMINA"
        runtime={testRuntime}
      />,
    );

    expect(await screen.findByText(/PKG-CAB-2026-001 is available in this managed profile/i)).toBeVisible();
    expect(screen.getByText(/Unsynced single-device work cannot be recovered/i)).toBeVisible();
  });
});
