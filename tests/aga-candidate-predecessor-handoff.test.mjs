import assert from "node:assert/strict";
import test from "node:test";

import { validateAGAPredecessorHandoff } from "../scripts/validate-aga-predecessor-handoff.mjs";

const databaseTarget = {
  environment: "local-preprod",
  databaseName: "aviasurveil360_local_preprod",
  databaseOwner: "aviasurveil360_preprod_loader",
  postgresSystemIdentifier: "7669215232100679702",
  postgresHost: "preprod-postgres",
  postgresPort: 5432,
  composeProject: "aviasurveil360-local-preprod",
};

test("AGA predecessor handoff requires an independent overlay control store", () => {
  const input = {
    receipt: {
      runId: "run-task7-connected-smoke",
      targetFingerprintDigest: `sha256:${"a".repeat(64)}`,
    },
    handoff: {
      schemaVersion: "preprod-retained-base-handoff/v1",
      baseResultFile: "/private/base-result.json",
      stateDirectory: "/private/state",
      controlStoreDirectory: "/private/state/control-store",
      targetFingerprintDigest: `sha256:${"a".repeat(64)}`,
      runId: "run-task7-connected-smoke",
      databaseTarget,
    },
    configuration: {
      target: { ...databaseTarget, overlaySchema: "preprod_aga_demo" },
    },
    baseResultFile: "/private/base-result.json",
    agaControlStoreDirectory: "/private/aga-control-store",
    stateDirectory: "/private/state",
  };

  assert.doesNotThrow(() => validateAGAPredecessorHandoff(input));
  assert.throws(
    () => validateAGAPredecessorHandoff({
      ...input,
      agaControlStoreDirectory: input.handoff.controlStoreDirectory,
    }),
    /independent/u,
  );
});
