#!/usr/bin/env bash
set -euo pipefail

umask 077
set -o noclobber

repository_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
evidence_root_relative="docs/demo-evidence/preprod-data"
evidence_root="$repository_root/$evidence_root_relative"
mode="${1:-}"
profile_name="${2:-}"

fail() {
  printf 'preprod-data-profile: %s\n' "$*" >&2
  exit 1
}

require_profile() {
  case "$1" in
    smoke | acceptance | realistic | stress)
      ;;
    *)
      fail "profile must be smoke|acceptance|realistic|stress"
      ;;
  esac
}

describe_profile() {
  local selected_profile="$1"
  command -v node >/dev/null 2>&1 || fail "node is required"
  AVIA_TASK8_SPECIFICATION_PATH="$repository_root/docs/product-specs/data-and-rules/PREPROD_IDENTITY_AND_DATA_PROFILE.md" \
  AVIA_TASK8_PROFILE="$selected_profile" \
  AVIA_TASK8_EVIDENCE_ROOT="$evidence_root_relative" \
    node --input-type=module <<'NODE'
import { readFileSync } from "node:fs";

const beginMarker = "<!-- PREPROD_IDENTITY_DATA_CONTRACT:BEGIN -->";
const endMarker = "<!-- PREPROD_IDENTITY_DATA_CONTRACT:END -->";
const source = readFileSync(process.env.AVIA_TASK8_SPECIFICATION_PATH, "utf8");
const begin = source.indexOf(beginMarker);
const end = source.indexOf(endMarker);
if (begin < 0 || end <= begin) {
  throw new Error("machine-readable profile contract markers are missing");
}
const fenced = source.slice(begin + beginMarker.length, end).trim();
const match = fenced.match(/^```json\s*\n([\s\S]+)\n```$/u);
if (!match) {
  throw new Error("machine-readable profile contract is malformed");
}
const contract = JSON.parse(match[1]);
const fullVolumeProfile = contract.dataProfiles?.profiles?.find(
  ({ name }) => name === process.env.AVIA_TASK8_PROFILE,
);
if (!fullVolumeProfile || fullVolumeProfile.version !== "1.0.0") {
  throw new Error("exact frozen profile was not found");
}
const revision = fullVolumeProfile.localQualification;
let profile = fullVolumeProfile;
let fullVolumeEndurance;
if (revision) {
  const [sourceName, sourceVersion] = revision.sourceProfile.split("@");
  const source = contract.dataProfiles.profiles.find(
    (candidate) =>
      candidate.name === sourceName && candidate.version === sourceVersion,
  );
  if (!source) {
    throw new Error(`qualification source ${revision.sourceProfile} is missing`);
  }
  const preserved = new Set(revision.preservedCatalogCountFamilies);
  const expectedCounts = Object.fromEntries(
    Object.entries(source.expectedCounts).map(([family, count]) => [
      family,
      preserved.has(family) ? count : count * revision.scaleMultiplier,
    ]),
  );
  const exactDistributions = Object.fromEntries(
    Object.entries(source.exactDistributions).map(([family, distribution]) => [
      family,
      family === "organizations"
        ? revision.organizationDistribution
        : Object.fromEntries(
          Object.entries(distribution).map(([state, count]) => [
            state,
            preserved.has(family)
              ? count
              : count * revision.scaleMultiplier,
          ]),
        ),
    ]),
  );
  profile = {
    ...fullVolumeProfile,
    version: revision.version,
    status: revision.status,
    resourceEnvelope: revision.resourceEnvelope,
    expectedCounts,
    exactDistributions,
  };
  fullVolumeEndurance = {
    profile: fullVolumeProfile.name,
    version: fullVolumeProfile.version,
    purpose: "release-readiness-evidence",
    status: fullVolumeProfile.runtimeStatus,
  };
}
process.stdout.write(`${JSON.stringify({
  schemaVersion: "preprod-profile-qualification-contract/v1",
  profile,
  fullVolumeEndurance,
  evidenceRoot: process.env.AVIA_TASK8_EVIDENCE_ROOT,
  operations: [
    "LOAD_EMPTY_TARGET",
    "RESUME_RUN",
    "DROP_RECREATE_TARGET",
  ],
  cleanupMode: "whole-disposable-namespace",
  selectiveDeletionAllowed: false,
  syntheticOnly: true,
})}\n`);
NODE
}

if [[ "$mode" == "--describe" ]]; then
  [[ "$#" -eq 2 ]] || fail "usage: $0 --describe PROFILE"
  require_profile "$profile_name"
  describe_profile "$profile_name"
  exit 0
fi

[[ "$#" -eq 1 ]] ||
  fail "usage: $0 smoke|acceptance|realistic|stress"
profile_name="$mode"
require_profile "$profile_name"

command -v docker >/dev/null 2>&1 || fail "docker is required"
command -v node >/dev/null 2>&1 || fail "node is required"
command -v openssl >/dev/null 2>&1 || fail "openssl is required"
command -v git >/dev/null 2>&1 || fail "git is required"

contract_json="$(describe_profile "$profile_name")"
resource_fields=(
  durationSeconds \
  memoryMiB \
  cpuCores \
  diskMiB \
  objectBytes \
  cleanupSeconds
)
if [[ "$profile_name" == "realistic" || "$profile_name" == "stress" ]]; then
  resource_fields+=(qualificationSeconds)
fi
for field in "${resource_fields[@]}"; do
  AVIA_TASK8_CONTRACT_JSON="$contract_json" \
  AVIA_TASK8_FIELD="$field" \
    node --input-type=module <<'NODE'
const contract = JSON.parse(process.env.AVIA_TASK8_CONTRACT_JSON);
const value = contract.profile.resourceEnvelope[process.env.AVIA_TASK8_FIELD];
if (!Number.isSafeInteger(value) || value <= 0) {
  throw new Error(`invalid resource envelope ${process.env.AVIA_TASK8_FIELD}`);
}
NODE
done

profile_version="$(
  AVIA_TASK8_CONTRACT_JSON="$contract_json" \
    node --input-type=module <<'NODE'
const contract = JSON.parse(process.env.AVIA_TASK8_CONTRACT_JSON);
if (!/^\d+[.]\d+[.]\d+$/u.test(contract.profile.version)) {
  throw new Error("invalid profile version");
}
process.stdout.write(contract.profile.version);
NODE
)"

mkdir -p "$evidence_root"
run_id="run-task8-${profile_name}-$(date -u +%Y%m%d-%H%M%S)-$$"
evidence_directory="$evidence_root/$run_id"
if [[ -e "$evidence_directory" ]]; then
  fail "evidence run directory already exists"
fi
mkdir "$evidence_directory"

printf '%s\n' "$contract_json" >"$evidence_directory/contract.json"
chmod 0600 "$evidence_directory/contract.json"

export AVIA_TASK8_PROFILE_QUALIFICATION="true"
export AVIA_TASK8_RUN_ID="$run_id"
export AVIA_TASK8_EVIDENCE_DIRECTORY="$evidence_directory"
export AVIA_TASK8_PROFILE_VERSION="$profile_version"
export AVIA_PREPROD_PROFILE_QUALIFICATION="true"
export AVIA_PREPROD_PROFILE="aga-preprod@1.0.0"
export AVIA_PREPROD_IDENTITY_NAMESPACE="canonical-aga-preprod-exercise-v1"

"$repository_root/scripts/test-preprod-connected-scenarios.sh" "$profile_name"
