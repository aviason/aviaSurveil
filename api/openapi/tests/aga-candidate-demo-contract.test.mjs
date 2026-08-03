import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { test } from "node:test";

test("AGA candidate demo contract exposes only the five read-only Admin routes", () => {
  const paths = JSON.parse(readFileSync("api/openapi/source/paths/platform.json", "utf8"));
  const prefix = "/v1/admin/governed-checklist/aga-candidate-demo";
  const expected = ["/capability", "/summary", "/forms", "/forms/{formCode}", "/questions"];
  const actual = Object.keys(paths).filter((path) => path.startsWith(prefix));
  assert.deepEqual(actual.sort(), expected.map((suffix) => prefix + suffix).sort());
  for (const path of actual) {
    assert.deepEqual(Object.keys(paths[path]), ["get"], `${path} must be read-only`);
  }
});
