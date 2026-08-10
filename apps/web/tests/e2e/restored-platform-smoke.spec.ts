import { createHmac } from "node:crypto";
import { writeFileSync } from "node:fs";

import { expect, test, type Page } from "@playwright/test";

import { REACT_ROUTE_CONTRACTS } from "../../src/app/route-contracts";

interface ApplicationSession {
  organizationId: string | null;
  roles: string[];
  subjectId: string;
}

const applicationOrigin = requiredEnvironment("AVIA_E2E_BASE_URL");

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
    if (index < 0) throw new Error("Keycloak returned an invalid TOTP secret");
    bits += index.toString(2).padStart(5, "0");
  }
  const bytes: number[] = [];
  for (let index = 0; index + 8 <= bits.length; index += 8) {
    bytes.push(Number.parseInt(bits.slice(index, index + 8), 2));
  }
  return Buffer.from(bytes);
}

function totp(secret: string, now = Date.now()): string {
  const counter = BigInt(Math.floor(now / 30_000));
  const message = Buffer.alloc(8);
  message.writeBigUInt64BE(counter);
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

async function avoidTotpBoundary(page: Page): Promise<void> {
  const remainingSeconds = 30 - (Math.floor(Date.now() / 1000) % 30);
  if (remainingSeconds < 4) {
    await page.waitForTimeout((remainingSeconds + 1) * 1000);
  }
}

async function beginLogin(page: Page): Promise<void> {
  await page.goto("/");
  await expect(
    page.getByRole("heading", { name: /Sign in to AviaSurveil360/i }),
  ).toBeVisible();
  await page
    .getByRole("button", { name: "Sign in with organization identity" })
    .click();
  await page
    .getByLabel(/username or email/i)
    .fill(requiredEnvironment("AVIA_RESTORED_USERNAME"));
  await page
    .locator('input[name="password"]')
    .fill(requiredEnvironment("AVIA_RESTORED_PASSWORD"));
  await page.getByRole("button", { name: /sign in/i }).click();
}

async function applicationSession(page: Page): Promise<ApplicationSession> {
  return expect.poll(async () => {
    if (new URL(page.url()).origin !== applicationOrigin) return null;
    return page.evaluate(async () => {
      const response = await fetch("/auth/session", {
        credentials: "same-origin",
      });
      if (!response.ok) return null;
      return response.json() as Promise<ApplicationSession>;
    });
  }, { timeout: 60_000 }).not.toBeNull().then(async () =>
    page.evaluate(async () => {
      const response = await fetch("/auth/session", {
        credentials: "same-origin",
      });
      return response.json() as Promise<ApplicationSession>;
    })
  );
}

async function enrollSourceTotp(page: Page): Promise<string> {
  await beginLogin(page);
  await expect(page.locator("#kc-totp-settings")).toBeVisible();
  await page.locator("#mode-manual").click();
  const secret = (await page.locator("#kc-totp-secret-key").innerText())
    .replace(/\s+/g, "");
  expect(secret).not.toBe("");
  await avoidTotpBoundary(page);
  await page.locator('input[name="totp"]').fill(totp(secret));
  const label = page.locator('input[name="userLabel"]');
  if (await label.isVisible()) await label.fill("Plan 4 restore drill");
  await page.locator('button[type="submit"], input[type="submit"]').click();
  await applicationSession(page);
  return secret;
}

async function loginWithRestoredTotp(page: Page): Promise<ApplicationSession> {
  await beginLogin(page);
  const otpField = page.locator('input[name="otp"], input[name="totp"]');
  await expect(otpField).toBeVisible();
  await avoidTotpBoundary(page);
  await otpField.fill(totp(requiredEnvironment("AVIA_RESTORED_TOTP_SECRET")));
  await page.getByRole("button", { name: /sign in/i }).click();
  return applicationSession(page);
}

test("restored platform retains MFA scope and serves all 85 active React routes", async ({
  page,
}) => {
  test.setTimeout(900_000);
  const mode = requiredEnvironment("AVIA_RESTORED_MODE");
  const expectedOrganization = requiredEnvironment(
    "AVIA_RESTORED_EXPECTED_ORGANIZATION_ID",
  );
  const expectedRoles = requiredEnvironment("AVIA_RESTORED_EXPECTED_ROLES")
    .split(",")
    .map((role) => role.trim())
    .filter(Boolean)
    .sort();

  if (mode === "prepare") {
    const secret = await enrollSourceTotp(page);
    const session = await applicationSession(page);
    expect(session.organizationId).toBe(expectedOrganization);
    expect([...session.roles].sort()).toEqual(expectedRoles);
    writeFileSync(requiredEnvironment("AVIA_RESTORED_TOTP_SECRET_FILE"), secret, {
      encoding: "utf8",
      mode: 0o600,
    });
    await test.info().attach("restored-platform-prepare", {
      body: JSON.stringify({
        identity: "production-mode Keycloak CONFIGURE_TOTP",
        organizationId: session.organizationId,
        roles: [...session.roles].sort(),
      }, null, 2),
      contentType: "application/json",
    });
    return;
  }
  expect(mode).toBe("verify");

  const session = await loginWithRestoredTotp(page);
  expect(session.organizationId).toBe(expectedOrganization);
  expect([...session.roles].sort()).toEqual(expectedRoles);
  expect(REACT_ROUTE_CONTRACTS).toHaveLength(85);
  const loaded = new Set<string>();
  for (const route of REACT_ROUTE_CONTRACTS) {
    const response = await page.goto(route.path, {
      waitUntil: "domcontentloaded",
    });
    expect(response?.status(), `direct load ${route.path}`).toBe(200);
    await expect(page.locator("main")).toHaveCount(1);
    expect(new URL(page.url()).pathname).toBe(route.path);
    loaded.add(route.id);
  }
  expect(loaded.size).toBe(85);

  await test.info().attach("restored-platform-summary", {
    body: JSON.stringify({
      status: "verified locally",
      artifactStatus: "candidate-only",
      directLoads: 85,
      totpLogin: true,
      organizationId: session.organizationId,
      roles: [...session.roles].sort(),
    }, null, 2),
    contentType: "application/json",
  });
});
