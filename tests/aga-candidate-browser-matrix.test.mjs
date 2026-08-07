import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { test } from "node:test";

const privacySource = readFileSync(
  "apps/web/tests/e2e/aga-candidate-demo-privacy.http.spec.ts",
  "utf8",
);
const harnessSource = readFileSync(
  "scripts/test-aga-candidate-preprod-demo-connected.sh",
  "utf8",
);
const supportSource = readFileSync(
  "apps/web/tests/e2e/support/aga-candidate-demo.ts",
  "utf8",
);
const adminSource = readFileSync(
  "apps/web/tests/e2e/aga-candidate-demo-admin.http.spec.ts",
  "utf8",
);
const playwrightConfig = readFileSync("apps/web/playwright.config.ts", "utf8");
const viteConfig = readFileSync("apps/web/vite.config.ts", "utf8");
const composeSource = readFileSync("deploy/local/compose.yaml", "utf8");

function composeService(name) {
  const start = composeSource.indexOf(`\n  ${name}:\n`);
  assert.notEqual(start, -1, `${name} service is missing`);
  const tail = composeSource.slice(start + 1);
  const next = tail.slice(1).search(/^  [a-z0-9][a-z0-9-]*:\n/mu);
  return next === -1 ? tail : tail.slice(0, next + 1);
}

test("each denied OIDC role has an independent fail-closed browser timeout", () => {
  assert.doesNotMatch(
    privacySource,
    /anonymous and every denied OIDC role receive one neutral no-store response/u,
  );
  assert.match(
    privacySource,
    /for \(const accountName of deniedAccounts\) \{\s*test\(/u,
  );
  assert.match(privacySource, /const neutralBody = '\{"error":"not found"\}'/u);
  assert.match(privacySource, /expect\(denied\.body\)\.toBe\(neutralBody\)/u);
  assert.match(privacySource, /test\.setTimeout\(120_000\)/u);
  assert.match(harnessSource, /tests=10 viewports=1440x900,1024x768,390x844/u);
});

test("OIDC form discovery uses stable provider controls and stops on the first mismatch", () => {
  assert.doesNotMatch(
    supportSource,
    /getByLabel\(\/username or email\/i\)|getByLabel\(\/password\/i\)/u,
  );
  assert.match(supportSource, /locator\("#username"\)/u);
  assert.match(supportSource, /locator\("#password"\)/u);
  assert.match(supportSource, /locator\("#kc-login"\)/u);
  assert.match(
    playwrightConfig,
    /maxFailures: profile === "preprod-aga-demo" \|\| profile === "preprod-aga-manager" \? 1 : 0/u,
  );
  assert.match(playwrightConfig, /actionTimeout: 30_000/u);
  assert.match(playwrightConfig, /navigationTimeout: 30_000/u);
});

test("OIDC login waits for the exact post-callback route before using the session", () => {
  assert.doesNotMatch(supportSource, /not\.toHaveURL\(\/\\\/identity\\\//u);
  assert.match(
    supportSource,
    /await expect\(page\)\.toHaveURL\(\(url\) => url\.pathname === returnTo\)/u,
  );
  assert.match(
    supportSource,
    /recordQualificationPhase\("oidc-callback-complete"\)/u,
  );
  assert.doesNotMatch(
    adminSource,
    /page\.goto\("\/admin\/checklist-builder"\)/u,
  );
  assert.match(supportSource, /recordQualificationPhase\("oidc-cookie-pair-present"\)/u);
  assert.match(privacySource, /recordQualificationPhase\("denied-session-cookie-sent"\)/u);
});

test("browser failure reports only a fixed phase marker and never the private log", () => {
  for (const phase of [
    "login-gate-visible",
    "provider-page-open",
    "provider-username-filled",
    "provider-password-filled",
    "provider-submit-complete",
    "oidc-provider-error",
    "oidc-provider-retained",
    "oidc-callback-invalid-state",
    "oidc-callback-invalid-token",
    "oidc-callback-stale-authority",
    "oidc-callback-session-failure",
    "oidc-callback-other-client-error",
    "oidc-callback-server-error",
    "oidc-callback-success-retained",
    "oidc-web-return-mismatch",
    "oidc-unexpected-location",
    "oidc-callback-complete",
    "anonymous-neutral-verified",
    "denied-login-complete",
    "denied-capability-requested",
    "denied-capability-received",
    "denied-neutral-verified",
    "denied-logout-complete",
    "denied-stale-session-verified",
    "mutation-login-complete",
    "mutation-methods-verified",
    "mutation-routes-verified",
    "mutation-logout-complete",
    "admin-login-complete",
    "logout-csrf-cookie-missing",
    "logout-csrf-rejected",
    "logout-session-missing",
    "logout-server-failure",
    "logout-unexpected-status",
    "admin-capability-requested",
    "admin-capability-verified",
    "admin-route-open",
    "admin-panel-visible",
    "admin-summary-visible",
    "admin-forms-read",
    "admin-question-slices-read",
    "admin-browser-storage-clear",
    "admin-viewports-verified",
    "admin-telemetry-silent",
    "admin-console-api-resource-error",
    "admin-console-auth-resource-error",
    "admin-console-vite-resource-error",
    "admin-console-asset-resource-error",
    "admin-console-other-web-resource-error",
    "admin-console-external-resource-error",
    "admin-console-unknown-resource-error",
    "admin-console-react-error",
    "admin-console-csp-error",
    "admin-console-other-error",
    "admin-console-clean",
    "admin-failed-requests-clean",
    "admin-runtime-clean",
    "admin-logout-request-complete",
    "admin-logout-session-revoked",
    "admin-logout-ui-cleared",
    "admin-logout-history-clean",
    "admin-logout-verified",
  ]) {
    assert.match(
      `${supportSource}\n${privacySource}\n${adminSource}`,
      new RegExp(`recordQualificationPhase\\("${phase}"\\)`, "u"),
    );
    assert.match(harnessSource, new RegExp(phase, "u"));
  }
  assert.match(harnessSource, /AVIA_AGA_BROWSER_PHASE_FILE=/u);
  assert.match(harnessSource, /isolated-browser qualification failed at safe phase=/u);
  assert.doesNotMatch(harnessSource, /cat .*playwright\.log/u);
});

test("browser failure classifies server session state without identity or token output", () => {
  assert.match(harnessSource, /safe_session_diagnostic\(\)/u);
  assert.match(
    harnessSource,
    /server session diagnostic=(?:active-valid|active-invalid|revocation-pending|denied|unclassified)/u,
  );
  assert.doesNotMatch(harnessSource, /SELECT (?:subject_id|session_token_hash|csrf_token_hash)/u);
  assert.match(supportSource, /AVIA_AGA_BROWSER_SESSION_DIGEST_FILE/u);
  assert.match(harnessSource, /browser session digest diagnostic=(?:match|mismatch|unclassified)/u);
  assert.match(harnessSource, /browser request digest diagnostic=(?:match|mismatch|unclassified)/u);
  assert.match(
    harnessSource,
    /API authentication diagnostic=(?:%s|unclassified)/u,
  );
  assert.doesNotMatch(harnessSource, /API authentication diagnostic=.*(?:subject|token|email)/u);
});

test("logout requires server revocation and inspects a controlled BFCache return", () => {
  assert.match(
    supportSource,
    /if \(result\.status === 204\) \{[\s\S]*?clearCookies\(\)/u,
  );
  assert.match(adminSource, /browserFetch\(page, `\$\{agaRoute\}\/capability`\)/u);
  assert.match(adminSource, /await page\.goto\("\/"\);\s*await page\.goBack\(\)/u);
  assert.doesNotMatch(adminSource, /await logout\(page\);\s*await page\.goBack\(\)/u);
});

test("the candidate browser run disables telemetry before any candidate read", () => {
  assert.match(
    harnessSource,
    /VITE_AVIA_DISABLE_BROWSER_TELEMETRY=1/u,
  );
  assert.match(adminSource, /const telemetryRequests: string\[\] = \[\]/u);
  assert.match(adminSource, /pathname\.startsWith\("\/otel\/v1\/"\)/u);
  assert.match(adminSource, /expect\(telemetryRequests\)\.toEqual\(\[\]\)/u);
});

test("the connected browser uses the built HTTP artifact through the local preview proxy", () => {
  assert.match(harnessSource, /npm run build:http/u);
  assert.match(harnessSource, /vite preview/u);
  assert.doesNotMatch(harnessSource, /npm run dev:http/u);
  assert.match(viteConfig, /preview:\s*\{[\s\S]*?proxy:/u);
});

test("only tagged browser-facing services join the dedicated non-internal edge", () => {
  assert.match(composeService("preprod-keycloak"), /- preprod-aga-demo-edge/u);
  assert.match(composeService("preprod-aga-demo-api"), /- preprod-aga-demo-edge/u);
  assert.doesNotMatch(
    composeService("preprod-aga-candidate-demo-loader"),
    /preprod-aga-demo-edge/u,
  );
  assert.match(composeSource, /^  preprod-aga-demo-edge:\n/mu);
});
