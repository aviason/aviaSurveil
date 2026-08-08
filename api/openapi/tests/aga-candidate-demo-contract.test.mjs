import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { test } from "node:test";

test("normal API contract excludes the isolated AGA candidate demo routes", () => {
  const paths = JSON.parse(readFileSync("api/openapi/source/paths/platform.json", "utf8"));
  const prefix = "/v1/admin/governed-checklist/aga-candidate-demo";
  const actual = Object.keys(paths).filter((path) => path.startsWith(prefix));
  assert.deepEqual(actual, []);
});
