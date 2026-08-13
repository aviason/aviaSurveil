import { createHmac } from "node:crypto";

import { expect, test, type Page } from "@playwright/test";

import { REACT_ROUTE_CONTRACTS } from "../../src/app/route-contracts";

interface Session {
  organizationId: string | null;
  roles: string[];
  subjectId: string;
}

const baseUrl = requiredEnvironment("AVIA_E2E_BASE_URL");

function requiredEnvironment(name: string): string {
  const value = process.env[name]?.trim();
  if (!value) throw new Error(`${name} is required`);
  return value;
}

function decodeBase32(value: string): Buffer {
  const alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZ234567";
  let bits = "";
  for (const character of value.toUpperCase().replace(/[^A-Z2-7]/g, "")) {
    const index = alphabet.indexOf(character);
    if (index < 0) throw new Error("invalid MFA secret");
    bits += index.toString(2).padStart(5, "0");
  }
  const bytes: number[] = [];
  for (let index = 0; index + 8 <= bits.length; index += 8) {
    bytes.push(Number.parseInt(bits.slice(index, index + 8), 2));
  }
  return Buffer.from(bytes);
}

function totp(secret: string): string {
  const message = Buffer.alloc(8);
  message.writeBigUInt64BE(BigInt(Math.floor(Date.now() / 30_000)));
  const digest = createHmac("sha1", decodeBase32(secret)).update(message).digest();
  const offset = digest[digest.length - 1]! & 0x0f;
  const value = (
    ((digest[offset]! & 0x7f) << 24) |
    ((digest[offset + 1]! & 0xff) << 16) |
    ((digest[offset + 2]! & 0xff) << 8) |
    (digest[offset + 3]! & 0xff)
  ) % 1_000_000;
  return value.toString().padStart(6, "0");
}

async function loginWithOidcMfa(page: Page): Promise<Session> {
  await page.goto("/");
  await page
    .getByRole("button", { name: "Sign in with organization identity" })
    .click();
  await page
    .getByLabel(/username or email/i)
    .fill(requiredEnvironment("AVIA_AWS_SMOKE_USERNAME"));
  await page
    .locator('input[name="password"]')
    .fill(requiredEnvironment("AVIA_AWS_SMOKE_PASSWORD"));
  await page.getByRole("button", { name: /sign in/i }).click();
  const otp = page.locator('input[name="otp"], input[name="totp"]');
  await expect(otp).toBeVisible();
  await otp.fill(totp(requiredEnvironment("AVIA_AWS_SMOKE_TOTP_SECRET")));
  await page.getByRole("button", { name: /sign in/i }).click();
  await expect(page).toHaveURL(new RegExp(`^${escapeRegex(baseUrl)}`));
  const response = await page.request.get("/auth/session");
  expect(response.status()).toBe(200);
  return response.json() as Promise<Session>;
}

test("AWS trial serves HTTPS security headers and all 85 active authorized routes", async ({
  page,
}) => {
  test.setTimeout(900_000);
  expect(baseUrl.startsWith("https://")).toBe(true);
  const ready = await page.request.get("/health/ready");
  expect(ready.status()).toBe(200);

  const session = await loginWithOidcMfa(page);
  expect(session.subjectId).not.toBe("");
  expect(session.roles.length).toBeGreaterThan(0);
  expect(REACT_ROUTE_CONTRACTS).toHaveLength(85);

  const loaded = new Set<string>();
  for (const route of REACT_ROUTE_CONTRACTS) {
    const response = await page.goto(route.path, {
      waitUntil: "domcontentloaded",
    });
    expect(response?.status(), route.path).toBe(200);
    const headers = response?.headers() ?? {};
    expect(headers["strict-transport-security"]).toBeTruthy();
    expect(headers["x-content-type-options"]).toBe("nosniff");
    await expect(page.locator("main")).toHaveCount(1);
    loaded.add(route.id);
  }
  expect(loaded.size).toBe(85);
});

test("AWS trial exposes bounded canonical and operational smoke receipts", async ({
  page,
}) => {
  test.setTimeout(300_000);
  await loginWithOidcMfa(page);
  const endpoint = requiredEnvironment("AVIA_AWS_SMOKE_RECEIPT_PATH");
  const response = await page.request.get(endpoint);
  expect(response.status()).toBe(200);
  const receipt = await response.json() as {
    alerts: boolean;
    backup: boolean;
    canonicalMutation: boolean;
    evidenceScan: "not-run";
    emailProviderConfigured: boolean;
    pdf: boolean;
    telemetry: boolean;
  };
  expect(receipt).toEqual({
    alerts: true,
    backup: true,
    canonicalMutation: true,
    evidenceScan: "not-run",
    emailProviderConfigured: true,
    pdf: true,
    telemetry: true,
  });
});

function escapeRegex(value: string): string {
  return value.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
}
