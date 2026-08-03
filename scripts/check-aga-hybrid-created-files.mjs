import { existsSync, readFileSync } from "node:fs";
import { dirname, extname, posix, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const sliceOrder = Object.freeze([
  "gate0",
  "slice-b",
  "slice-c",
  "slice-d",
  "slice-e",
  "slice-f",
]);
const rootPrefixes = Object.freeze([
  "api/",
  "apps/",
  "deliverables/",
  "docs/",
  "scripts/",
  "tests/",
]);

class ControlledError extends Error {
  constructor(code) {
    super(code);
    this.code = code;
  }
}

function fail(code) {
  throw new ControlledError(code);
}

function parseArguments(argv) {
  const values = new Map();
  for (let index = 0; index < argv.length; index += 2) {
    const name = argv[index];
    const value = argv[index + 1];
    if (!name?.startsWith("--") || value === undefined || values.has(name)) {
      fail("ARGUMENTS_INVALID");
    }
    values.set(name, value);
  }
  if (
    values.size !== 2 ||
    !values.has("--inventory") ||
    !values.has("--through")
  ) {
    fail("ARGUMENTS_REQUIRED");
  }
  throughOwnerSlice(values.get("--through"));
  return values;
}

function ownerSlice(taskNumber) {
  if (taskNumber === 0) return "gate0";
  if (taskNumber <= 2) return "slice-b";
  if (taskNumber <= 4) return "slice-c";
  if (taskNumber <= 6) return "slice-d";
  if (taskNumber <= 8) return "slice-e";
  return "slice-f";
}

export function throughOwnerSlice(value) {
  if (sliceOrder.includes(value)) return value;
  const taskMatch = /^task([1-9]|10)$/u.exec(value);
  if (!taskMatch) fail("THROUGH_SLICE_INVALID");
  return ownerSlice(Number(taskMatch[1]));
}

function normalizePlannedPath(token, baseDirectory) {
  if (token.endsWith("/")) return { directory: token.slice(0, -1), path: null };
  if (rootPrefixes.some((prefix) => token.startsWith(prefix))) {
    return { directory: dirname(token), path: token };
  }
  if (!baseDirectory) fail("PLAN_CREATE_PATH_BASE_MISSING");
  return {
    directory: baseDirectory,
    path: posix.join(baseDirectory, token),
  };
}

export function plannedCreateEntries(plan) {
  const entries = [];
  const headings = ["Gate 0", ...Array.from({ length: 10 }, (_, index) => `Task ${index + 1}`)];
  for (const heading of headings) {
    const headingOffset = plan.indexOf(`### ${heading}`);
    if (headingOffset < 0) fail("PLAN_HEADING_MISSING");
    const filesOffset = plan.indexOf("**Files:**", headingOffset);
    if (filesOffset < 0) fail("PLAN_FILES_SECTION_MISSING");
    const candidateEnds = [
      plan.indexOf("\n**TDD", filesOffset),
      plan.indexOf("\n**Gate", filesOffset),
      plan.indexOf("\n**Commands", filesOffset),
      plan.indexOf("\n### ", filesOffset),
    ].filter((offset) => offset > filesOffset);
    const filesEnd = Math.min(...candidateEnds);
    const section = plan.slice(filesOffset + "**Files:**".length, filesEnd);
    const bullets = section.split(/\n(?=- )/u);
    const taskNumber = heading === "Gate 0" ? 0 : Number(heading.slice("Task ".length));
    for (const originalBullet of bullets) {
      const firstParagraph = originalBullet.split(/\n\n/u)[0].replace(/\n  /gu, " ");
      const createOffset = firstParagraph.search(/(?:^- Create\b|\band create\b)/u);
      if (createOffset < 0) continue;
      const bullet = firstParagraph.slice(createOffset).split(/;\s+modify\b/iu)[0];
      const tokens = [...bullet.matchAll(/`([^`]+)`/gu)].map((match) => match[1]);
      let baseDirectory = null;
      for (const token of tokens) {
        const normalized = normalizePlannedPath(token, baseDirectory);
        baseDirectory = normalized.directory;
        if (!normalized.path) continue;
        entries.push({
          path: normalized.path,
          ownerSlice: ownerSlice(taskNumber),
          ownerTask: taskNumber === 0 ? "gate0" : `task-${taskNumber}`,
        });
      }
    }
  }
  return entries;
}

function exactKeys(value, expected) {
  return (
    value &&
    typeof value === "object" &&
    !Array.isArray(value) &&
    Object.keys(value).sort().join("\u0000") === [...expected].sort().join("\u0000")
  );
}

function validateInventory(inventory, expectedEntries) {
  if (
    !exactKeys(inventory, ["schemaVersion", "planPath", "entries"]) ||
    inventory.schemaVersion !== "aga-hybrid-created-file-inventory/v1" ||
    inventory.planPath !==
      "docs/exec-plans/active/2026-08-03-aga-hybrid-classification-demo-lifecycle-plan.md" ||
    !Array.isArray(inventory.entries)
  ) {
    fail("INVENTORY_CONTRACT_MISMATCH");
  }
  const seen = new Set();
  for (const entry of inventory.entries) {
    if (
      !exactKeys(entry, ["path", "ownerSlice", "ownerTask"]) ||
      typeof entry.path !== "string" ||
      !/^[A-Za-z0-9._/-]+$/u.test(entry.path) ||
      entry.path.startsWith("/") ||
      entry.path.split("/").includes("..") ||
      !rootPrefixes.some((prefix) => entry.path.startsWith(prefix)) ||
      !sliceOrder.includes(entry.ownerSlice) ||
      !/^(?:gate0|task-(?:[1-9]|10))$/u.test(entry.ownerTask)
    ) {
      fail("INVENTORY_ENTRY_OUT_OF_SCOPE");
    }
    if (seen.has(entry.path)) fail("INVENTORY_ENTRY_DUPLICATE");
    seen.add(entry.path);
  }
  const expectedByPath = new Map(expectedEntries.map((entry) => [entry.path, entry]));
  if (expectedByPath.size !== expectedEntries.length) fail("PLAN_CREATE_PATH_DUPLICATE");
  if (inventory.entries.length !== expectedEntries.length) {
    fail("INVENTORY_CREATE_PATH_COUNT_MISMATCH");
  }
  for (const entry of inventory.entries) {
    const expected = expectedByPath.get(entry.path);
    if (
      !expected ||
      expected.ownerSlice !== entry.ownerSlice ||
      expected.ownerTask !== entry.ownerTask
    ) {
      fail("INVENTORY_CREATE_PATH_SET_MISMATCH");
    }
  }
}

function scanTextFile(path) {
  let text;
  try {
    text = readFileSync(path, "utf8");
  } catch {
    fail("DUE_FILE_UNREADABLE");
  }
  if (text.split(/\n/u).some((line) => /[ \t]+$/u.test(line))) {
    fail("DUE_FILE_TRAILING_WHITESPACE");
  }
  if (extname(path) === ".json") {
    try {
      JSON.parse(text);
    } catch {
      fail("DUE_JSON_INVALID");
    }
  }
  if (extname(path) === ".md" && ((text.match(/^```/gmu) ?? []).length % 2) !== 0) {
    fail("DUE_MARKDOWN_FENCE_UNBALANCED");
  }
}

function run(argv) {
  const argumentsMap = parseArguments(argv);
  const inventoryPath = argumentsMap.get("--inventory");
  let inventory;
  try {
    inventory = JSON.parse(readFileSync(inventoryPath, "utf8"));
  } catch {
    fail("INVENTORY_UNAVAILABLE");
  }
  let plan;
  try {
    plan = readFileSync(inventory.planPath, "utf8");
  } catch {
    fail("PLAN_UNAVAILABLE");
  }
  const expectedEntries = plannedCreateEntries(plan);
  validateInventory(inventory, expectedEntries);
  const through = argumentsMap.get("--through");
  const throughIndex = sliceOrder.indexOf(throughOwnerSlice(through));
  const due = inventory.entries.filter(
    (entry) => sliceOrder.indexOf(entry.ownerSlice) <= throughIndex,
  );
  for (const entry of due) {
    if (!existsSync(resolve(entry.path))) fail("DUE_FILE_MISSING");
    scanTextFile(entry.path);
  }
  process.stdout.write(
    `aga-hybrid-created-files: ok through=${through} due=${due.length} planned=${inventory.entries.length}\n`,
  );
}

if (
  process.argv[1] &&
  fileURLToPath(import.meta.url) === resolve(process.argv[1])
) {
  try {
    run(process.argv.slice(2));
  } catch (error) {
    const code = error instanceof ControlledError ? error.code : "UNEXPECTED";
    process.stderr.write(`ERR_AGA_HYBRID_CREATED_FILES code=${code}\n`);
    process.exitCode = 1;
  }
}
