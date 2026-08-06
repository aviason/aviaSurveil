import assert from "node:assert/strict";
import test from "node:test";

import { validateRedactedPlan } from "../scripts/lib/aws-ipv6-trial-redaction.mjs";

test("IPv6 trial plan redaction accepts references and rejects sensitive values", () => {
  assert.deepEqual(validateRedactedPlan({
    connectorParameterName: "/aviasurveil360/trial/connector",
    apiTokenEnv: "CLOUDFLARE_API_TOKEN",
    sensitiveValue: null,
    planned: { value: "arn:aws:ssm:eu-central-1:111122223333:parameter/trial" },
  }), []);
  assert.deepEqual(validateRedactedPlan({ tokenValue: "secret-token" }), ["unredacted-plan:plan.tokenValue"]);
  assert.deepEqual(validateRedactedPlan({ nested: { privateKey: "-----BEGIN PRIVATE KEY-----" } }), ["unredacted-plan:plan.nested.privateKey", "secret-looking-value:plan.nested.privateKey"]);
});
