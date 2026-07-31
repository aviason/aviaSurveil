#!/usr/bin/env node
import { requestDigest, validateRealOPSAOCRequest } from "./checklist-generation-contracts.mjs";

const args = process.argv.slice(2);
if (args.length !== 2 || args[0] !== "--request" || args[1] !== "GENREQ-OPS-AOC-0001") {
  console.error("usage: node scripts/regulatory/prepare-checklist-generation.mjs --request GENREQ-OPS-AOC-0001");
  process.exitCode = 2;
} else {
  const request = {
    schemaVersion: "1.0.0",
    requestId: "GENREQ-OPS-AOC-0001",
    organizationId: "ORG-FLY-NAMIBIA",
    serviceProviderScopeFactIds: ["SCOPE-OPS-AOC-SOURCE-BOUND"],
    serviceProviderTypes: ["AIR_OPERATOR"],
    providerCatalogVersion: "1.0.0",
    inspectionType: "RAMP_INSPECTION",
    target: { targetId: "TARGET-OPS-AOC-SOURCE-BOUND", kind: "ORGANIZATION" },
    sourceSnapshots: [{ sourceSnapshotId: "NCAA-CC-ANNEX6-PARTI-A610-SUPPLIED-2026-07-28", sourceHash: "sha256:13fe82d1767320443f91ed61cf7d3b4bba0ea24f217fad45bbd9cae5fc682af2", clauseIds: ["NCAA-CC-A610-4.2.2.2"], clauseLocators: ["Annex 6 Part I 4.2.2.2"] }],
    secondaryCrosswalkPartition: { partitionId: "CC-OPS-TRAIN-1", stableRowIds: ["CC:NAMB:ANNEX6:4.2.2.2"] },
    unresolvedSourceGaps: [
      { gapId: "CONTROLLED_PROCEDURE", reason: "The controlled NCAA Operations surveillance/ramp-inspection procedure has not been supplied." },
      { gapId: "PART_140_AUTHORITY", reason: "Current Part 140 authority and supersession require source-owner confirmation." },
      { gapId: "PART_127_APPLICABILITY", reason: "Exact Part 127 operation/configuration applicability requires Department Manager confirmation." }
    ],
    generationPolicyVersion: "regulatory-checklist-v1",
    providerId: "imported-result-only",
    providerVersion: "1.0.0",
    requestedOutputs: ["COMPLIANCE_MAPPING", "INSPECTION_CHECKLIST"]
  };
  request.canonicalInputDigest = requestDigest(request);
	validateRealOPSAOCRequest(request);
  process.stdout.write(`${JSON.stringify(request, null, 2)}\n`);
}
