import { CURRENT_OFFLINE_VERSIONS, type OfflineVersionVector } from "../offline/offline-version-contract";

export type BuildArtifactLane = "demo" | "http" | "production-offline";

export interface BuildArtifactContract {
  schemaVersion: 1;
  lane: BuildArtifactLane;
  buildProfile: "demo" | "http";
  artifactIdentity: string;
  cacheNamespace: string;
  acceptanceLane: string;
  backend: "mock" | "http";
  status: "candidate-only";
  offlineVersionVector: OfflineVersionVector;
}

const CONTRACTS: Readonly<Record<BuildArtifactLane, Omit<BuildArtifactContract, "offlineVersionVector">>> = {
  demo: {
    schemaVersion: 1,
    lane: "demo",
    buildProfile: "demo",
    artifactIdentity: "aviasurveil360-demo-online-mock",
    cacheNamespace: "aviasurveil360-demo-app-shell",
    acceptanceLane: "demo-online-mock",
    backend: "mock",
    status: "candidate-only",
  },
  http: {
    schemaVersion: 1,
    lane: "http",
    buildProfile: "http",
    artifactIdentity: "aviasurveil360-http-candidate",
    cacheNamespace: "aviasurveil360-http-app-shell",
    acceptanceLane: "http-candidate",
    backend: "http",
    status: "candidate-only",
  },
  "production-offline": {
    schemaVersion: 1,
    lane: "production-offline",
    buildProfile: "http",
    artifactIdentity: "aviasurveil360-production-offline-browser-tab",
    cacheNamespace: "aviasurveil360-production-offline-app-shell",
    acceptanceLane: "production-offline-browser-matrix",
    backend: "http",
    status: "candidate-only",
  },
};

export function buildArtifactContract(lane: BuildArtifactLane): BuildArtifactContract {
  return {
    ...CONTRACTS[lane],
    offlineVersionVector: { ...CURRENT_OFFLINE_VERSIONS },
  };
}

export function resolveBuildArtifactLane(
  value: string | undefined,
  buildProfile: "demo" | "http",
): BuildArtifactLane {
  if (value === undefined || value === "") return buildProfile;
  if (value === "demo" || value === "http" || value === "production-offline") {
    const contract = buildArtifactContract(value);
    if (contract.buildProfile !== buildProfile) {
      throw new Error(`AVIA_BUILD_LANE '${value}' requires AVIA_BUILD_PROFILE '${contract.buildProfile}'`);
    }
    return value;
  }
  throw new Error("AVIA_BUILD_LANE must be exactly 'demo', 'http', or 'production-offline'");
}
