import type { Plugin } from "vite";

import {
  buildArtifactContract,
  type BuildArtifactLane,
} from "../src/app/build-artifact-contract";

export function createBuildArtifactPlugin(
  lane: BuildArtifactLane,
  buildProfile: "demo" | "http",
): Plugin {
  return {
    name: "aviasurveil360-build-artifact-contract",
    generateBundle() {
      const contract = buildArtifactContract(lane);
      if (contract.buildProfile !== buildProfile) {
        throw new Error(`Build artifact lane ${lane} cannot use ${buildProfile} source`);
      }
      this.emitFile({
        type: "asset",
        fileName: "build-artifact.json",
        source: `${JSON.stringify(contract, null, 2)}\n`,
      });
    },
  };
}
