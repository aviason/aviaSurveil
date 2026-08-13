import { readFile } from "node:fs/promises";

function required(name) {
  const value = process.env[name]?.trim();
  if (!value) throw new Error(`${name} is required`);
  return value;
}

function boundedCount(name, fallback, maximum) {
  const value = Number(process.env[name] ?? fallback);
  if (!Number.isInteger(value) || value < 1 || value > maximum) throw new Error(`${name} must be an integer from 1 to ${maximum}`);
  return value;
}

async function request(url, options = {}) {
  const response = await fetch(url, { ...options, redirect: "manual", signal: AbortSignal.timeout(20_000) });
  return response;
}

const baseURL = required("AVIA_AUTH_CANDIDATE_BASE_URL").replace(/\/$/, "");
const clientSecret = (await readFile(required("AVIA_AUTH_CANDIDATE_CLIENT_SECRET_FILE"), "utf8")).trim();
const accountPassword = (await readFile(required("AVIA_AUTH_CANDIDATE_LOAD_PASSWORD_FILE"), "utf8")).trim();
const loginCount = boundedCount("AVIA_AUTH_CANDIDATE_LOAD_LOGIN_COUNT", 2, 2);
const rejectedCount = boundedCount("AVIA_AUTH_CANDIDATE_LOAD_REJECTED_COUNT", 2, 2);
const recoveryCount = boundedCount("AVIA_AUTH_CANDIDATE_LOAD_RECOVERY_COUNT", 4, 8);
const callbackURL = "http://127.0.0.1:18082/callback";

const discoveryResponse = await request(`${baseURL}/.well-known/openid-configuration`);
if (!discoveryResponse.ok) throw new Error(`discovery status ${discoveryResponse.status}`);
const discovery = await discoveryResponse.json();
if (discovery.scopes_supported?.includes("offline_access")) throw new Error("discovery advertises disabled offline_access scope");
if (discovery.grant_types_supported?.includes("refresh_token")) throw new Error("discovery advertises disabled refresh-token grant");

async function authorize(index, expectedSuccess) {
  const verifier = `load-verifier-${index.toString().padStart(2, "0")}-012345678901234567890123456789`;
  const challenge = Buffer.from(await crypto.subtle.digest("SHA-256", new TextEncoder().encode(verifier))).toString("base64url");
  const authorizeURL = new URL(`${baseURL}/authorize`);
  const state = `load-state-${index}`;
  authorizeURL.search = new URLSearchParams({ client_id: "as360-local-candidate-web", response_type: "code", response_mode: "query", scope: "openid profile email", redirect_uri: callbackURL, state, nonce: `load-nonce-${index}`, code_challenge: challenge, code_challenge_method: "S256" }).toString();
  const authorizeResponse = await request(authorizeURL);
  if (authorizeResponse.status !== 302) throw new Error(`authorize ${index} status ${authorizeResponse.status}`);
  const loginURL = new URL(requiredLocation(authorizeResponse, `authorize ${index}`), baseURL);
  const requestID = loginURL.searchParams.get("id");
  if (loginURL.pathname !== "/login" || !requestID) throw new Error(`authorize ${index} did not enter provider login`);
  const identifier = expectedSuccess ? "load-candidate@example.invalid" : `missing-load-${index}@example.invalid`;
  const loginResponse = await request(loginURL, { method: "POST", headers: { "Content-Type": "application/x-www-form-urlencoded", "User-Agent": `auth-load-${index}` }, body: new URLSearchParams({ id: requestID, identifier, password: expectedSuccess ? accountPassword : `${accountPassword}x` }) });
  if (!expectedSuccess) {
    if (loginResponse.status !== 200 || !(await loginResponse.text()).includes("invalid credentials")) throw new Error(`rejected login ${index} was not denied`);
    return;
  }
  if (loginResponse.status !== 302) {
    const body = await loginResponse.text();
    const reason = body.includes("invalid credentials") ? "invalid-credentials" : body.includes("too many login attempts") ? "rate-limited" : "unexpected-response";
    throw new Error(`login ${index} status ${loginResponse.status} (${reason})`);
  }
  const providerCallback = new URL(requiredLocation(loginResponse, `login ${index}`), baseURL);
  if (providerCallback.pathname !== "/authorize/callback") throw new Error(`login ${index} did not enter provider callback`);
  const callbackResponse = await request(providerCallback);
  if (callbackResponse.status !== 302) throw new Error(`provider callback ${index} status ${callbackResponse.status}`);
  const clientCallback = new URL(requiredLocation(callbackResponse, `provider callback ${index}`));
  const code = clientCallback.searchParams.get("code");
  if (clientCallback.origin !== new URL(callbackURL).origin || !code || clientCallback.searchParams.get("state") !== state) throw new Error(`provider callback ${index} was invalid`);
  const tokenResponse = await request(discovery.token_endpoint, { method: "POST", headers: { "Content-Type": "application/x-www-form-urlencoded", Authorization: `Basic ${Buffer.from(`as360-local-candidate-web:${clientSecret}`).toString("base64")}` }, body: new URLSearchParams({ grant_type: "authorization_code", code, redirect_uri: callbackURL, code_verifier: verifier }) });
  if (!tokenResponse.ok) throw new Error(`token ${index} status ${tokenResponse.status}`);
  const token = await tokenResponse.json();
  if (!token.access_token || !token.id_token || token.refresh_token) throw new Error(`token ${index} violated the no-refresh-token contract`);
}

function requiredLocation(response, stage) {
  const location = response.headers.get("location");
  if (!location) throw new Error(`${stage} did not return a redirect`);
  return location;
}

async function recovery(index) {
  const response = await request(`${baseURL}/recover/password`, { method: "POST", headers: { "Content-Type": "application/x-www-form-urlencoded" }, body: new URLSearchParams({ email: "load-candidate@example.invalid" }) });
  if (response.status !== 202 || (await response.text()) !== "If the account is eligible, recovery instructions will be sent.") throw new Error(`recovery ${index} was not enumeration-safe acceptance`);
}

async function probe(index) {
  const [ready, configuration] = await Promise.all([request(`${baseURL}/health/ready`), request(`${baseURL}/.well-known/openid-configuration`)]);
  if (!ready.ok || !configuration.ok) throw new Error(`probe ${index} failed (${ready.status}/${configuration.status})`);
}

const started = performance.now();
await Promise.all([
  ...Array.from({ length: recoveryCount }, (_, index) => recovery(index)),
  ...Array.from({ length: loginCount }, (_, index) => probe(index)),
  ...Array.from({ length: loginCount }, (_, index) => authorize(index, true)),
]);
await Promise.all(Array.from({ length: rejectedCount }, (_, index) => authorize(loginCount + index, false)));
const elapsedMilliseconds = Math.round(performance.now() - started);
process.stdout.write(`auth-candidate ARM64 bounded mixed-load: verified locally (${loginCount} successful logins at the configured Argon2id capacity, ${rejectedCount} rejected Argon2id logins, ${recoveryCount} recovery requests, ${loginCount * 2} readiness/discovery probes, ${elapsedMilliseconds}ms)\n`);
