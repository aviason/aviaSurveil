import { describe, expect, it } from "vitest";

import type { AppShellPredecessorDescriptor } from "./app-shell-manifest-contract";
import { CURRENT_OFFLINE_VERSIONS, type OfflineVersionVector } from "./offline-version-contract";
import {
  allClientsAcknowledgedSafeCheckpoint,
  canActivateAppShellCandidate,
  classifyAppShellRequest,
  isSafeCheckpointAck,
} from "../sw";

function descriptor(
  fingerprint: string,
  compatibility: OfflineVersionVector = CURRENT_OFFLINE_VERSIONS,
): AppShellPredecessorDescriptor {
  return {
    lockDigest: null,
    webImageReferenceDigest: null,
    platformManifestDigest: null,
    serviceWorkerURL: "/sw.js",
    serviceWorkerSha256: null,
    appShellManifestSha256: null,
    releaseFingerprint: fingerprint,
    compatibility,
  };
}

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

describe("Service Worker activation policy", () => {
  it("requires a quiescent, durable acknowledgement before worker takeover", () => {
    const ack = {
      clientId: "tab-a",
      fingerprint: "sha256:new",
      compatibility: CURRENT_OFFLINE_VERSIONS,
      dirtyFormCount: 0,
      active: { indexedDb: 0, opfs: 0, hashWorker: 0, sync: 0, mutation: 0 },
      durableWorkAcknowledged: true,
    };
    expect(isSafeCheckpointAck(ack, "sha256:new", CURRENT_OFFLINE_VERSIONS)).toBe(true);
    expect(isSafeCheckpointAck({ ...ack, active: { ...ack.active, sync: 1 } }, "sha256:new", CURRENT_OFFLINE_VERSIONS)).toBe(false);
    expect(isSafeCheckpointAck({ ...ack, durableWorkAcknowledged: false }, "sha256:new", CURRENT_OFFLINE_VERSIONS)).toBe(false);
    expect(allClientsAcknowledgedSafeCheckpoint([ack], "sha256:new", CURRENT_OFFLINE_VERSIONS)).toBe(true);
    expect(allClientsAcknowledgedSafeCheckpoint([
      ack,
      { ...ack, clientId: "tab-b", durableWorkAcknowledged: false },
    ], "sha256:new", CURRENT_OFFLINE_VERSIONS)).toBe(false);
  });

  it("activates a first install and an exact direct predecessor", () => {
    const previous = descriptor(`sha256:${"1".repeat(64)}`);
    expect(canActivateAppShellCandidate(
      { compatibility: CURRENT_OFFLINE_VERSIONS, predecessor: null },
      [],
    )).toBe(true);
    expect(canActivateAppShellCandidate(
      { compatibility: CURRENT_OFFLINE_VERSIONS, predecessor: previous },
      [{ compatibility: CURRENT_OFFLINE_VERSIONS, releaseDescriptor: previous }],
    )).toBe(true);
  });

  it("lets an exact-vector client skip missed app-shell releases", () => {
    const expectedPredecessor = descriptor(`sha256:${"2".repeat(64)}`);
    const olderCommittedRelease = descriptor(`sha256:${"1".repeat(64)}`);
    expect(canActivateAppShellCandidate(
      { compatibility: CURRENT_OFFLINE_VERSIONS, predecessor: expectedPredecessor },
      [{ compatibility: CURRENT_OFFLINE_VERSIONS, releaseDescriptor: olderCommittedRelease }],
    )).toBe(true);
  });

  it("activates the fingerprint-bound legacy v9 bridge without a committed v2 cache", () => {
    const legacy = {
      ...descriptor(`sha256:${"1".repeat(64)}`),
      serviceWorkerURL: "/sw.js?v=9",
    };
    expect(canActivateAppShellCandidate(
      { compatibility: CURRENT_OFFLINE_VERSIONS, predecessor: legacy },
      [],
    )).toBe(true);
  });

  it("activates an exact current predecessor candidate from the embedded legacy v9 bridge", () => {
    const currentPredecessor = descriptor(`sha256:${"2".repeat(64)}`);
    const legacy = {
      ...descriptor(`sha256:${"1".repeat(64)}`),
      serviceWorkerURL: "/sw.js?v=9",
    };
    expect(canActivateAppShellCandidate(
      { compatibility: CURRENT_OFFLINE_VERSIONS, predecessor: currentPredecessor },
      [],
      legacy,
    )).toBe(true);
  });

  it("rejects a skipped release when any offline compatibility dimension differs", () => {
    const expectedPredecessor = descriptor(`sha256:${"2".repeat(64)}`);
    const incompatible = {
      ...CURRENT_OFFLINE_VERSIONS,
      indexedDbSchemaVersion: CURRENT_OFFLINE_VERSIONS.indexedDbSchemaVersion - 1,
    };
    expect(canActivateAppShellCandidate(
      { compatibility: CURRENT_OFFLINE_VERSIONS, predecessor: expectedPredecessor },
      [{ compatibility: incompatible, releaseDescriptor: descriptor(`sha256:${"1".repeat(64)}`, incompatible) }],
    )).toBe(false);
  });

  it("rejects a legacy v9 bridge with an incompatible predecessor vector", () => {
    const incompatible = {
      ...CURRENT_OFFLINE_VERSIONS,
      packageSchemaVersion: CURRENT_OFFLINE_VERSIONS.packageSchemaVersion + 1,
    };
    const legacy = {
      ...descriptor(`sha256:${"1".repeat(64)}`, incompatible),
      serviceWorkerURL: "/sw.js?v=9",
    };
    expect(canActivateAppShellCandidate(
      { compatibility: CURRENT_OFFLINE_VERSIONS, predecessor: legacy },
      [],
    )).toBe(false);
  });

  it("rejects an incompatible embedded legacy bridge", () => {
    const currentPredecessor = descriptor(`sha256:${"2".repeat(64)}`);
    const incompatible = {
      ...CURRENT_OFFLINE_VERSIONS,
      syncProtocolVersion: CURRENT_OFFLINE_VERSIONS.syncProtocolVersion + 1,
    };
    const legacy = {
      ...descriptor(`sha256:${"1".repeat(64)}`, incompatible),
      serviceWorkerURL: "/sw.js?v=9",
    };
    expect(canActivateAppShellCandidate(
      { compatibility: CURRENT_OFFLINE_VERSIONS, predecessor: currentPredecessor },
      [],
      legacy,
    )).toBe(false);
  });
});
