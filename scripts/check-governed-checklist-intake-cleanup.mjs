#!/usr/bin/env node

import { existsSync, readdirSync } from "node:fs";
import { join } from "node:path";

const roots = [
  process.env.AVIA_INTAKE_RUNTIME_DIR,
  "/private/tmp/aviasurveil360-aga-intake",
].filter(Boolean);
const leftovers = [];
for (const root of roots) {
  if (!existsSync(root)) continue;
  for (const entry of readdirSync(root)) leftovers.push(join(root, entry));
}
if (leftovers.length) {
  console.error(`governed-checklist-intake cleanup failed: ${leftovers.join(", ")}`);
  process.exit(1);
}
console.log("governed-checklist-intake cleanup: ok");
