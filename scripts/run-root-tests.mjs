#!/usr/bin/env node

import { readdirSync, statSync } from "node:fs";
import { dirname, join, relative, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { spawnSync } from "node:child_process";

const repositoryRoot = resolve(dirname(fileURLToPath(import.meta.url)), "..");
const testsRoot = join(repositoryRoot, "tests");

function discoverTests(directory) {
  const discovered = [];
  for (const entry of readdirSync(directory, { withFileTypes: true })) {
    const path = join(directory, entry.name);
    if (entry.isDirectory()) {
      discovered.push(...discoverTests(path));
      continue;
    }
    if (entry.isFile() && /\.test\.(?:js|mjs)$/.test(entry.name)) {
      discovered.push(path);
    }
  }
  return discovered;
}

const testFiles = discoverTests(testsRoot).sort();
if (testFiles.length === 0) {
  console.error(`root-test-discovery: no test files found under ${relative(repositoryRoot, testsRoot)}`);
  process.exit(1);
}

console.log(`root-test-discovery: ${testFiles.length} files`);
for (const testFile of testFiles) {
  console.log(`  ${relative(repositoryRoot, testFile)}`);
}

// Run the discovered suite in a stable order. Several contract tests invoke
// repository generators and inspect task-owned fixtures; allowing Node's
// default parallel workers to overlap those subprocesses makes the root gate
// nondeterministic and can mask the first actionable failure.
const result = spawnSync(process.execPath, ["--test", "--test-concurrency=1", ...testFiles], {
  cwd: repositoryRoot,
  stdio: "inherit",
});

if (result.error) {
  console.error(`root-test-discovery: unable to run Node test runner: ${result.error.message}`);
  process.exit(1);
}

process.exit(result.status ?? 1);
