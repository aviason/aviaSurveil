import { describe, expect, it } from "vitest";

import {
  buildArtifactContract,
  type BuildArtifactLane,
} from "./build-artifact-contract";

describe("build artifact separation", () => {
  it("assigns distinct identities and cache namespaces to demo and production-offline lanes", () => {
    const demo = buildArtifactContract("demo");
    const offline = buildArtifactContract("production-offline");

    expect(demo).toMatchObject({
      artifactIdentity: "aviasurveil360-demo-online-mock",
      cacheNamespace: "aviasurveil360-demo-app-shell",
      acceptanceLane: "demo-online-mock",
      backend: "mock",
      status: "candidate-only",
    });
    expect(offline).toMatchObject({
      artifactIdentity: "aviasurveil360-production-offline-browser-tab",
      cacheNamespace: "aviasurveil360-production-offline-app-shell",
      acceptanceLane: "production-offline-browser-matrix",
      backend: "http",
      status: "candidate-only",
    });
    expect(offline.artifactIdentity).not.toBe(demo.artifactIdentity);
    expect(offline.cacheNamespace).not.toBe(demo.cacheNamespace);
  });

  it("never marks a locally built artifact production-ready", () => {
    for (const profile of ["demo", "http", "production-offline"] as BuildArtifactLane[]) {
      expect(buildArtifactContract(profile).status).toBe("candidate-only");
      expect(buildArtifactContract(profile)).not.toHaveProperty("productionReady", true);
    }
  });
});
