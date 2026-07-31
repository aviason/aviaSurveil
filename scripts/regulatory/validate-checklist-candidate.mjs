#!/usr/bin/env node
import fs from "node:fs";
import path from "node:path";
import { validateCandidateBundle } from "./checklist-generation-contracts.mjs";

const input = process.argv[2];
if (!input || process.argv.length !== 3) {
  console.error("usage: node scripts/regulatory/validate-checklist-candidate.mjs path/to/candidate.json");
  process.exitCode = 2;
} else {
  try {
    const candidate = JSON.parse(fs.readFileSync(path.resolve(input), "utf8"));
    validateCandidateBundle(candidate);
    process.stdout.write(`validated ${candidate.candidateBundleId} ${candidate.inputDigest} ${candidate.outputDigest}\n`);
  } catch (error) {
    console.error(`candidate validation failed: ${error.message}`);
    process.exitCode = 1;
  }
}
