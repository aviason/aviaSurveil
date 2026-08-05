import { readFileSync } from "node:fs";
import { resolve } from "node:path";
import { pathToFileURL } from "node:url";

const targetKeys = [
  "environment",
  "databaseName",
  "databaseOwner",
  "postgresSystemIdentifier",
  "postgresHost",
  "postgresPort",
  "composeProject",
];

export function validateAGAPredecessorHandoff({
  receipt,
  handoff,
  configuration,
  baseResultFile,
  agaControlStoreDirectory,
  stateDirectory,
}) {
  if (
    handoff?.schemaVersion !== "preprod-retained-base-handoff/v1" ||
    resolve(handoff.baseResultFile ?? "") !== resolve(baseResultFile) ||
    resolve(handoff.stateDirectory ?? "") !== resolve(stateDirectory) ||
    handoff.targetFingerprintDigest !== receipt?.targetFingerprintDigest ||
    handoff.runId !== receipt?.runId ||
    !handoff.databaseTarget ||
    !configuration?.target ||
    (configuration.overlaySchema ?? configuration.target.overlaySchema) !==
      "preprod_aga_demo" ||
    targetKeys.some(
      (key) => handoff.databaseTarget[key] !== configuration.target[key],
    )
  ) {
    throw new Error(
      "configuration database target differs from retained base receipt",
    );
  }
  if (
    resolve(handoff.controlStoreDirectory ?? "") ===
    resolve(agaControlStoreDirectory)
  ) {
    throw new Error(
      "AGA overlay requires an independent control store from its predecessor",
    );
  }
  return true;
}

function runFromEnvironment(environment) {
  const required = [
    "AVIA_AGA_BASE_RESULT_FILE",
    "AVIA_AGA_HANDOFF_FILE",
    "AVIA_AGA_CONFIG_FILE",
    "AVIA_AGA_CONTROL_STORE_DIRECTORY",
    "AVIA_AGA_STATE_DIRECTORY",
  ];
  if (required.some((name) => !environment[name])) {
    throw new Error("AGA predecessor handoff validator input is incomplete");
  }
  validateAGAPredecessorHandoff({
    receipt: JSON.parse(readFileSync(environment.AVIA_AGA_BASE_RESULT_FILE)),
    handoff: JSON.parse(readFileSync(environment.AVIA_AGA_HANDOFF_FILE)),
    configuration: JSON.parse(readFileSync(environment.AVIA_AGA_CONFIG_FILE)),
    baseResultFile: environment.AVIA_AGA_BASE_RESULT_FILE,
    agaControlStoreDirectory: environment.AVIA_AGA_CONTROL_STORE_DIRECTORY,
    stateDirectory: environment.AVIA_AGA_STATE_DIRECTORY,
  });
}

if (
  process.argv[1] &&
  import.meta.url === pathToFileURL(resolve(process.argv[1])).href
) {
  runFromEnvironment(process.env);
}
