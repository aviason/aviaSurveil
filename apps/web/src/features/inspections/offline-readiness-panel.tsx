import { useEffect, useMemo, useState } from "react";

import type {
  CheckoutInspectionPackageInput,
  CheckoutInspectionPackageOutput,
  InspectionPackage,
} from "../../backend/backend";
import { useApplicationRuntime, useBackendForRole } from "../../app/providers";
import {
  CURRENT_OFFLINE_VERSIONS,
  assessOfflineReadiness,
  createBrowserOfflineReadinessDependencies,
  describeLocalPackageLoss,
  classifyLocalDataState,
  getOrCreateDeviceInstanceId,
  readOfflineCheckoutSnapshot,
  writeOfflineCheckoutSnapshot,
  type OfflineCheckoutSnapshot,
  type OfflineReadinessDependencies,
  type OfflineReadinessResult,
} from "../../offline/storage-readiness";
import { createBrowserProfileAuthority, type ProfileAuthority } from "../../offline/profile-authority";

export interface OfflineReadinessPanelRuntime {
  checkout(input: CheckoutInspectionPackageInput): Promise<CheckoutInspectionPackageOutput>;
  dependencies: OfflineReadinessDependencies;
  getDeviceInstanceId(): Promise<string>;
  getProfileAuthority?(): Promise<ProfileAuthority>;
  readSnapshot(subjectId: string, packageId: string): Promise<OfflineCheckoutSnapshot | null>;
  writeSnapshot(snapshot: OfflineCheckoutSnapshot): Promise<void>;
  now(): Date;
}

interface OfflineReadinessPanelProps {
  inspectionPackage: InspectionPackage;
  subjectId: string;
  runtime?: OfflineReadinessPanelRuntime;
}

export function OfflineReadinessPanel(props: OfflineReadinessPanelProps) {
  if (props.runtime) {
    return <OfflineReadinessPanelCore {...props} runtime={props.runtime} />;
  }
  return (
    <ConnectedOfflineReadinessPanel
      inspectionPackage={props.inspectionPackage}
      subjectId={props.subjectId}
    />
  );
}

function ConnectedOfflineReadinessPanel({
  inspectionPackage,
  subjectId,
}: Omit<OfflineReadinessPanelProps, "runtime">) {
  const backend = useBackendForRole("inspector");
  const applicationRuntime = useApplicationRuntime();
  const runtime = useMemo<OfflineReadinessPanelRuntime>(
    () => ({
      checkout: (input) => backend.inspections.checkout(input),
      dependencies: createBrowserOfflineReadinessDependencies(),
      getDeviceInstanceId: getOrCreateDeviceInstanceId,
      getProfileAuthority: () => createBrowserProfileAuthority(subjectId),
      readSnapshot: readOfflineCheckoutSnapshot,
      writeSnapshot: writeOfflineCheckoutSnapshot,
      now: () =>
        applicationRuntime.now?.() ??
        (applicationRuntime.buildProfile === "demo"
          ? new Date("2026-06-15T09:00:00.000Z")
          : new Date()),
    }),
    [applicationRuntime.buildProfile, backend, subjectId],
  );
  return (
    <OfflineReadinessPanelCore
      inspectionPackage={inspectionPackage}
      subjectId={subjectId}
      runtime={runtime}
    />
  );
}

function OfflineReadinessPanelCore({
  inspectionPackage,
  subjectId,
  runtime: activeRuntime,
}: Required<OfflineReadinessPanelProps>) {
  const [managedPolicyApproved, setManagedPolicyApproved] = useState(false);
  const [storageProfileApproved, setStorageProfileApproved] = useState(false);
  const [result, setResult] = useState<OfflineReadinessResult | null>(null);
  const [snapshot, setSnapshot] = useState<OfflineCheckoutSnapshot | null>(null);
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    let active = true;
    void activeRuntime
      .readSnapshot(subjectId, inspectionPackage.id)
      .then((value) => {
        if (active && value?.subjectId === subjectId) setSnapshot(value);
      })
      .catch(() => undefined);
    return () => {
      active = false;
    };
  }, [activeRuntime, inspectionPackage.id, subjectId]);

  const buildInput = (
    deviceInstanceId: string,
    checkout: CheckoutInspectionPackageOutput | null,
  ) => ({
    userInitiated: true,
    managedPolicyApproved,
    storageProfileApproved,
    expectedSubjectId: subjectId,
    expectedOrganizationId: inspectionPackage.organizationId,
    expectedDeviceInstanceId: deviceInstanceId,
    packageDescriptor: {
      packageId: inspectionPackage.id,
      packageVersion: inspectionPackage.packageVersion,
      packageDigest: inspectionPackage.packageDigest,
      schemaVersion: inspectionPackage.schemaVersion,
      protocolVersion: inspectionPackage.protocolVersion,
      expiresAt: inspectionPackage.expiresAt,
    },
    offlineGrant: checkout?.offlineGrant ?? null,
    versions: CURRENT_OFFLINE_VERSIONS,
    requiredAppShellVersion: CURRENT_OFFLINE_VERSIONS.appShellVersion,
    packageByteEstimate: new TextEncoder().encode(JSON.stringify(inspectionPackage)).byteLength,
    attachmentByteEstimate: 25 * 1024 * 1024,
    minimumHeadroomBytes: 50 * 1024 * 1024,
    now: activeRuntime.now(),
  });

  const checkOut = async () => {
    setBusy(true);
    setResult(null);
    try {
      const deviceInstanceId = await activeRuntime.getDeviceInstanceId();
      const profileAuthority = await activeRuntime.getProfileAuthority?.();
      const preflight = await assessOfflineReadiness(
        buildInput(deviceInstanceId, null),
        activeRuntime.dependencies,
      );
      if (preflight.code !== "offline-grant-invalid") {
        setResult(preflight);
        return;
      }
      const checkout = await activeRuntime.checkout({
        operationId: `OP-OFFLINE-CHECKOUT-${inspectionPackage.id}-${deviceInstanceId}`,
        packageId: inspectionPackage.id,
        expectedPackageVersion: inspectionPackage.packageVersion,
        deviceInstanceId,
        ...(profileAuthority
          ? { profileKeyId: profileAuthority.profileKeyId, profilePublicJwk: profileAuthority.publicJwk }
          : {}),
      });
      const finalResult = await assessOfflineReadiness(
        buildInput(deviceInstanceId, checkout),
        activeRuntime.dependencies,
      );
      setResult(finalResult);
      if (!finalResult.ready) return;
      const nextSnapshot: OfflineCheckoutSnapshot = {
        subjectId,
        inspectionPackage: checkout.inspectionPackage,
        offlineGrant: checkout.offlineGrant,
        checkedOutAt: activeRuntime.now().toISOString(),
        versions: CURRENT_OFFLINE_VERSIONS,
      };
      await activeRuntime.writeSnapshot(nextSnapshot);
      setSnapshot(nextSnapshot);
    } catch (error) {
      setResult({
        code: "offline-grant-invalid",
        ready: false,
        admissionState: "ONLINE_ONLY",
        reasonCode: "OFFLINE_GRANT_INVALID",
        recoveryAction:
          error instanceof Error
            ? `Reconnect before checkout: ${error.message}`
            : "Reconnect before requesting an offline grant.",
        capacityIsAdvisory: true,
        requiredBytes: null,
        availableBytes: null,
      });
    } finally {
      setBusy(false);
    }
  };

  const lossBoundary = describeLocalPackageLoss({
    outstandingCheckout: true,
    localPackagePresent: snapshot !== null,
  });

  return (
    <section className="surface-card offline-readiness" data-testid="offline-readiness-panel">
      <div className="card-heading">
        <div>
          <p className="eyebrow">Managed-browser field mode</p>
          <h2>Offline readiness</h2>
        </div>
        <span className="status-pill">Online use preserved</span>
      </div>
      <p>
        This explicit gate checks the app shell, local storage health, persistence, advisory
        capacity, restart survival, compatibility, and the server-issued grant. It does not detect
        private browsing or reserve disk space.
      </p>
      <div className="offline-readiness__attestations">
        <label>
          <input
            type="checkbox"
            checked={managedPolicyApproved}
            onChange={(event) => setManagedPolicyApproved(event.target.checked)}
          />
          I confirm the owner-approved official browser/version lane and managed profile policy for this device.
        </label>
        <label>
          <input
            type="checkbox"
            checked={storageProfileApproved}
            onChange={(event) => setStorageProfileApproved(event.target.checked)}
          />
          I confirm an encrypted managed profile with clear-on-exit disabled.
        </label>
      </div>
      <button className="primary-button" type="button" disabled={busy} onClick={() => void checkOut()}>
        {busy ? "Checking offline readiness…" : "Check out for offline use"}
      </button>
      <p>Online inspection remains available when offline checkout is blocked.</p>
      {result ? (
        <div
          className={result.ready ? "offline-readiness__result is-ready" : "offline-readiness__result"}
          role="status"
          data-readiness-code={result.code}
          data-admission-state={result.admissionState}
        >
          <strong>{result.ready ? "Ready for official offline checkout" : result.code}</strong>
          <span>{result.recoveryAction}</span>
          {result.requiredBytes !== null && result.availableBytes !== null ? (
            <span>
              Capacity estimate is advisory: {Math.ceil(result.requiredBytes / 1024 / 1024)} MB
              required with headroom; {Math.floor(result.availableBytes / 1024 / 1024)} MB reported
              available.
            </span>
          ) : null}
        </div>
      ) : null}
      {snapshot ? (
        <p className="offline-readiness__snapshot" data-testid="offline-package-status">
          {snapshot.inspectionPackage.id} is available in this managed profile for subject {subjectId}.
        </p>
      ) : null}
      <p className="offline-readiness__warning">
        <span data-local-data-state={classifyLocalDataState({
          outstandingCheckout: true,
          localPackagePresent: snapshot !== null,
        })}>
        {lossBoundary
          ? `If the server reports an outstanding checkout: ${lossBoundary}`
          : "Explicit site-data clearing removes this browser working copy. Unsynced single-device work cannot be recovered; canonical server records remain separate."}
        </span>
      </p>
    </section>
  );
}
