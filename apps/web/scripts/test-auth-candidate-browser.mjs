import { chromium } from "playwright";
import { createServer } from "node:http";
import { readFile } from "node:fs/promises";

function required(name) {
  const value = process.env[name]?.trim();
  if (!value) throw new Error(`${name} is required`);
  return value;
}

const baseURL = required("AVIA_AUTH_CANDIDATE_BASE_URL").replace(/\/$/, "");
const callbackPort = Number(required("AVIA_AUTH_CANDIDATE_CALLBACK_PORT"));
const headed = process.env.AVIA_AUTH_CANDIDATE_BROWSER_HEADED === "1";
const clientSecret = (await readFile(required("AVIA_AUTH_CANDIDATE_CLIENT_SECRET_FILE"), "utf8")).trim();
const accountPassword = (await readFile(required("AVIA_AUTH_CANDIDATE_BROWSER_PASSWORD_FILE"), "utf8")).trim();
const mfaCode = (await readFile(required("AVIA_AUTH_CANDIDATE_MFA_CODE_FILE"), "utf8")).trim();
const resetPassword = (await readFile(required("AVIA_AUTH_CANDIDATE_BROWSER_RESET_PASSWORD_FILE"), "utf8")).trim();
const passwordResetURL = (await readFile(required("AVIA_AUTH_CANDIDATE_PASSWORD_RESET_URL_FILE"), "utf8")).trim();
const mfaResetURL = (await readFile(required("AVIA_AUTH_CANDIDATE_MFA_RESET_URL_FILE"), "utf8")).trim();
const callbackURL = `http://127.0.0.1:${callbackPort}/callback`;
const logoutURL = `http://127.0.0.1:${callbackPort}/logout`;
let callbackRequestURL = "";
let callbackParameters = new URLSearchParams();
function stage(name) {
  process.stdout.write(`auth-candidate browser stage: ${name}\n`);
}

async function assertAccessibleForm(page, heading, labels = []) {
  if (await page.locator("html").getAttribute("lang") !== "en") throw new Error(`${heading} form is missing its document language`);
  if (await page.getByRole("main").count() !== 1) throw new Error(`${heading} form does not expose exactly one main landmark`);
  if (await page.getByRole("heading", { name: heading, level: 1 }).count() !== 1) throw new Error(`${heading} form does not expose its level-one heading`);
  if (await page.locator("form").count() !== 1) throw new Error(`${heading} form does not expose exactly one form`);
  if (await page.getByRole("button", { name: "Continue" }).count() !== 1) throw new Error(`${heading} form does not expose its submit button`);
  for (const label of labels) {
    const control = page.getByLabel(label, { exact: true });
    if (await control.count() !== 1) throw new Error(`${heading} form does not expose a single ${label} label/control pair`);
    if (await control.getAttribute("required") === null && await control.getAttribute("type") !== "checkbox") throw new Error(`${heading} form control ${label} is not required`);
  }
}
const callback = createServer(async (request, response) => {
  callbackRequestURL = `http://127.0.0.1:${callbackPort}${request.url ?? ""}`;
  callbackParameters = new URL(callbackRequestURL).searchParams;
  if (request.method === "POST") {
    const chunks = [];
    for await (const chunk of request) chunks.push(chunk);
    for (const [key, value] of new URLSearchParams(Buffer.concat(chunks).toString("utf8"))) callbackParameters.set(key, value);
  }
  response.writeHead(200, { "Content-Type": "text/plain; charset=utf-8", "Cache-Control": "no-store" });
  response.end("isolated callback received");
});
await new Promise((resolve, reject) => callback.once("error", reject).listen(callbackPort, "127.0.0.1", resolve));
for (const recoveryURL of [passwordResetURL, mfaResetURL]) {
  if (new URL(recoveryURL).origin !== baseURL) throw new Error("recovery URL does not belong to the isolated candidate");
}
let browser;
try {
  browser = await chromium.launch({ headless: !headed });
  const context = await browser.newContext({ viewport: { width: 1280, height: 800 } });
  const page = await context.newPage();
  page.setDefaultTimeout(10_000);
  const observedRoutes = [];
  let providerCallbackURL = "";
  page.on("response", (response) => {
    const endpoint = new URL(response.url());
    if (endpoint.origin === baseURL) {
      const location = response.headers().location;
      if (response.request().method() === "POST" && endpoint.pathname === "/mfa" && location) providerCallbackURL = new URL(location, baseURL).toString();
      const redirect = location ? ` -> ${new URL(location, baseURL).origin}${new URL(location, baseURL).pathname}` : "";
      observedRoutes.push(`${response.request().method()} ${endpoint.pathname} ${response.status()}${redirect}`);
    }
  });
  page.on("requestfailed", (request) => {
    const endpoint = new URL(request.url());
    const failure = request.failure()?.errorText ?? "unknown";
    observedRoutes.push(`${request.method()} ${endpoint.origin}${endpoint.pathname} failed:${failure}`);
  });
  stage("discovery");
  const discovery = await page.request.get(`${baseURL}/.well-known/openid-configuration`);
  if (!discovery.ok()) throw new Error(`discovery status ${discovery.status()}`);
  const configuration = await discovery.json();
  const verifier = "browser-isolated-verifier-012345678901234567890123456789";
  const challenge = Buffer.from(await crypto.subtle.digest("SHA-256", new TextEncoder().encode(verifier))).toString("base64url");
  const authorize = new URL(`${baseURL}/authorize`);
  authorize.search = new URLSearchParams({ client_id: "as360-local-candidate-web", response_type: "code", response_mode: "form_post", scope: "openid profile email offline_access", redirect_uri: callbackURL, state: "browser-state", nonce: "browser-nonce", code_challenge: challenge, code_challenge_method: "S256" }).toString();
  stage("authorize");
  await page.goto(authorize.toString(), { waitUntil: "domcontentloaded" });
  await page.getByRole("heading", { name: "Sign in" }).waitFor();
  await assertAccessibleForm(page, "Sign in", ["identifier", "Password"]);
  await page.locator('input[name="identifier"]').fill("browser-candidate@example.invalid");
  await page.locator('input[name="password"]').fill(accountPassword);
  await page.getByRole("button", { name: "Continue" }).click();
  stage("mfa");
  await page.getByRole("heading", { name: "Verify MFA" }).waitFor();
  await assertAccessibleForm(page, "Verify MFA", ["code", "Use recovery code"]);
  await page.locator('input[name="code"]').fill(mfaCode);
  await page.getByRole("button", { name: "Continue" }).click();
  stage("callback");
  try {
    await page.getByText("isolated callback received").waitFor();
  } catch (error) {
    const callbackPath = new URL(page.url()).pathname;
    const pageText = await page.locator("body").innerText().catch(() => "unavailable");
    let callbackProbe = "unavailable";
    if (providerCallbackURL) {
      const response = await page.request.get(providerCallbackURL, { maxRedirects: 0 });
      const location = response.headers()["location"];
      callbackProbe = `${response.status()}${location ? ` -> ${new URL(location, baseURL).origin}${new URL(location, baseURL).pathname}` : ""}`;
    }
    throw new Error(`OIDC callback page was not reached (path=${callbackPath}, body=${pageText.slice(0, 300)}, callbackProbe=${callbackProbe}, routes=${observedRoutes.slice(-8).join(" | ")})`, { cause: error });
  }
  const authorizationCode = callbackParameters.get("code");
  if (!authorizationCode || callbackParameters.get("state") !== "browser-state") throw new Error("OIDC callback did not contain the expected code and state");
  const token = await page.request.post(configuration.token_endpoint, { form: { grant_type: "authorization_code", code: authorizationCode, redirect_uri: callbackURL, code_verifier: verifier }, headers: { Authorization: `Basic ${Buffer.from(`as360-local-candidate-web:${clientSecret}`).toString("base64")}` } });
  if (!token.ok()) throw new Error(`token status ${token.status()}`);
  const tokenPayload = await token.json();
  if (!tokenPayload.access_token || !tokenPayload.id_token || !tokenPayload.refresh_token) throw new Error("token response was incomplete");
  stage("recovery");
  await page.goto(`${baseURL}/recover/password`, { waitUntil: "domcontentloaded" });
  await page.getByRole("heading", { name: "Recover access" }).waitFor();
  await assertAccessibleForm(page, "Recover access", ["email"]);
  await page.locator('input[name="email"]').fill("browser-candidate@example.invalid");
  await page.getByRole("button", { name: "Continue" }).click();
  await page.getByText("If the account is eligible, recovery instructions will be sent.").waitFor();
  stage("password-reset");
  await page.goto(passwordResetURL, { waitUntil: "domcontentloaded" });
  await page.getByRole("heading", { name: "Set a new password" }).waitFor();
  await assertAccessibleForm(page, "Set a new password", ["password"]);
  await page.locator('input[name="password"]').fill(resetPassword);
  await page.getByRole("button", { name: "Continue" }).click();
  await page.getByRole("heading", { name: "Sign in" }).waitFor();
  stage("mfa-reset");
  await page.goto(mfaResetURL, { waitUntil: "domcontentloaded" });
  await page.getByRole("heading", { name: "Reset MFA" }).waitFor();
  await assertAccessibleForm(page, "Reset MFA");
  await page.getByRole("button", { name: "Continue" }).click();
  await page.getByRole("heading", { name: "Sign in" }).waitFor();
  stage("post-reset-authorize");
  const postResetAuthorize = new URL(authorize);
  postResetAuthorize.searchParams.set("state", "browser-state-after-reset");
  postResetAuthorize.searchParams.set("nonce", "browser-nonce-after-reset");
  await page.goto(postResetAuthorize.toString(), { waitUntil: "domcontentloaded" });
  await page.getByRole("heading", { name: "Sign in" }).waitFor();
  await assertAccessibleForm(page, "Sign in", ["identifier", "Password"]);
  await page.locator('input[name="identifier"]').fill("browser-candidate@example.invalid");
  await page.locator('input[name="password"]').fill(resetPassword);
  await page.getByRole("button", { name: "Continue" }).click();
  await page.getByText("isolated callback received").waitFor();
  if (!callbackParameters.get("code") || callbackParameters.get("state") !== "browser-state-after-reset") throw new Error("post-reset authorization did not complete without MFA");
  stage("logout");
  await page.goto(`${baseURL}/logout?${new URLSearchParams({ client_id: "as360-local-candidate-web", post_logout_redirect_uri: logoutURL })}`, { waitUntil: "domcontentloaded" });
  try {
    await page.getByText("isolated callback received").waitFor();
  } catch (error) {
    const logoutPath = new URL(page.url()).pathname;
    const pageText = await page.locator("body").innerText().catch(() => "unavailable");
    throw new Error(`logout callback was not reached (path=${logoutPath}, body=${pageText.slice(0, 300)}, routes=${observedRoutes.slice(-8).join(" | ")})`, { cause: error });
  }
  process.stdout.write("auth-candidate browser qualification: verified locally\n");
} finally {
  await browser?.close();
  await new Promise((resolve) => callback.close(resolve));
}
