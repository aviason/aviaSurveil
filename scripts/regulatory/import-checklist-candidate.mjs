#!/usr/bin/env node
import fs from "node:fs";
import path from "node:path";
import { spawnSync } from "node:child_process";
import { validateCandidateBundle } from "./checklist-generation-contracts.mjs";

const [candidatePath, ...rest] = process.argv.slice(2);
const requestedIdentity = rest.length === 2 && rest[0] === "--request" ? rest[1] : null;
if (!candidatePath || (rest.length !== 0 && !requestedIdentity)) {
  console.error("usage: node scripts/regulatory/import-checklist-candidate.mjs path/to/candidate.json [--request GENREQ-OPS-AOC-0001]");
  process.exitCode = 2;
} else {
  try {
    const candidate = JSON.parse(fs.readFileSync(path.resolve(candidatePath), "utf8"));
    validateCandidateBundle(candidate);
    if (requestedIdentity === "GENREQ-OPS-AOC-0001") {
      throw new Error("import blocked: the real OPS/AOC pilot has unresolved source authority and cannot create a candidate, decision, publication, or Audit side effect");
    }
    if (!process.env.AVIA_REGULATORY_DATABASE_URL) {
      throw new Error("import blocked: AVIA_REGULATORY_DATABASE_URL is required for the explicit local test-only transactional persistence seam");
    }
    const database = new URL(process.env.AVIA_REGULATORY_DATABASE_URL);
    if (!process.env.AVIA_REGULATORY_TEST_MODE || !["localhost", "127.0.0.1", "::1"].includes(database.hostname) || !/(?:task|test)/iu.test(database.pathname)) {
      throw new Error("import blocked: only explicit loopback test-only PostgreSQL profiles are permitted");
    }
    const result = spawnSync("go", ["run", "./cmd/regulatory-import", "--candidate", path.resolve(candidatePath), "--database-url", process.env.AVIA_REGULATORY_DATABASE_URL, "--synthetic-test-profile"], { cwd: path.resolve("apps/api"), encoding: "utf8" });
    if (result.status !== 0) throw new Error(result.stderr.trim() || "local database importer failed");
    process.stdout.write(result.stdout);
  } catch (error) {
    console.error(`candidate import failed: ${error.message}`);
    process.exitCode = 1;
  }
}
