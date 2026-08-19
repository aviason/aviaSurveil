import { expect, test } from "@playwright/test";
import { execFileSync } from "node:child_process";
import { createServer, type Server } from "node:http";
import { readFileSync } from "node:fs";
import { readFile, stat } from "node:fs/promises";
import { extname, resolve, sep } from "node:path";
import { fileURLToPath } from "node:url";

import { CURRENT_OFFLINE_VERSIONS } from "../../src/offline/offline-version-contract";
import type { AppShellPredecessorDescriptor } from "../../src/offline/app-shell-manifest-contract";

const legacyUpgradeBrowser = process.env.AVIA_LEGACY_UPDATE_BROWSER === "webkit"
  ? "webkit"
  : "chromium";
test.use({ browserName: legacyUpgradeBrowser });

const webRoot = resolve(fileURLToPath(new URL("../../", import.meta.url)));
const buildRoot = resolve(webRoot, "dist/demo");
const legacyPredecessor: AppShellPredecessorDescriptor = {
  lockDigest: `sha256:${"1".repeat(64)}`,
  webImageReferenceDigest: `sha256:${"2".repeat(64)}`,
  platformManifestDigest: `sha256:${"2".repeat(64)}`,
  serviceWorkerURL: "/sw.js?v=9",
  serviceWorkerSha256: `sha256:${"3".repeat(64)}`,
  appShellManifestSha256: `sha256:${"4".repeat(64)}`,
  releaseFingerprint: null,
  compatibility: CURRENT_OFFLINE_VERSIONS,
};
const currentPredecessor: AppShellPredecessorDescriptor = {
  ...legacyPredecessor,
  lockDigest: `sha256:${"5".repeat(64)}`,
  webImageReferenceDigest: `sha256:${"6".repeat(64)}`,
  platformManifestDigest: `sha256:${"6".repeat(64)}`,
  serviceWorkerURL: "/sw.js",
  serviceWorkerSha256: `sha256:${"7".repeat(64)}`,
  appShellManifestSha256: `sha256:${"8".repeat(64)}`,
  releaseFingerprint: `sha256:${"9".repeat(64)}`,
};

const contentTypes: Record<string, string> = {
  ".css": "text/css",
  ".html": "text/html",
  ".js": "text/javascript",
  ".json": "application/json",
  ".map": "application/json",
  ".svg": "image/svg+xml",
  ".png": "image/png",
  ".jpg": "image/jpeg",
  ".jpeg": "image/jpeg",
  ".webp": "image/webp",
  ".ttf": "font/ttf",
  ".woff": "font/woff",
  ".woff2": "font/woff2",
};

const legacyHTML = `<!doctype html>
<html lang="en">
  <head><meta charset="UTF-8"><title>Legacy AviaSurveil shell</title></head>
  <body><main><h1>Legacy shell</h1></main><script type="module" src="/assets/legacy-app.js"></script></body>
</html>`;

const legacyApp = `
globalThis.__legacyShellLoaded = true;
let registration;
let currentFingerprint = null;
let pendingReloadFingerprint = null;
let dirty = false;

function sendReady() {
  registration?.active?.postMessage({
    type: "avia:app-shell-client-ready",
    fingerprint: currentFingerprint,
  });
}

function reloadWhenQuiescent() {
  if (
    dirty ||
    !pendingReloadFingerprint ||
    !currentFingerprint ||
    pendingReloadFingerprint === currentFingerprint ||
    navigator.serviceWorker.controller !== registration?.active
  ) return;
  window.location.reload();
}

navigator.serviceWorker.addEventListener("controllerchange", () => {
  sendReady();
  reloadWhenQuiescent();
});
navigator.serviceWorker.addEventListener("message", (event) => {
  if (event.data?.type !== "avia:app-shell-activation" || typeof event.data.fingerprint !== "string") return;
  if (event.data.legacyRetirement === true) return;
  pendingReloadFingerprint = event.data.fingerprint;
  reloadWhenQuiescent();
});

globalThis.__setLegacyDirty = (value) => {
  dirty = value;
  if (!dirty) sendReady();
  reloadWhenQuiescent();
};

void navigator.serviceWorker.register("/sw.js", {
  scope: "/",
  type: "module",
  updateViaCache: "none",
}).then(async (nextRegistration) => {
  registration = nextRegistration;
  sendReady();
  const manifest = await fetch("/app-shell-assets.json", { cache: "no-store" }).then((response) => response.json()).catch(() => null);
  currentFingerprint = typeof manifest?.releaseFingerprint === "string" ? manifest.releaseFingerprint : "sha256:legacy";
  await nextRegistration.update();
});
`;

const legacyWorker = `
const CACHE_NAME = "aviasurveil360-app-shell-v9";
self.addEventListener("install", (event) => {
  event.waitUntil(caches.open(CACHE_NAME).then((cache) => cache.addAll(["/", "/assets/legacy-app.js"])));
  self.skipWaiting();
});
self.addEventListener("activate", (event) => event.waitUntil(self.clients.claim()));
self.addEventListener("fetch", (event) => {
  const url = new URL(event.request.url);
  if (event.request.mode === "navigate" && url.pathname !== "/app-shell-recovery.html") {
    event.respondWith(caches.open(CACHE_NAME).then((cache) => cache.match("/")));
  } else if (url.pathname === "/assets/legacy-app.js") {
    event.respondWith(caches.open(CACHE_NAME).then((cache) => cache.match(url.pathname)));
  }
});
`;

const brokenWaitingWorker = `
const CACHE_NAME = "aviasurveil360-app-shell-broken-waiting";
self.addEventListener("install", (event) => {
  event.waitUntil(caches.open(CACHE_NAME).then((cache) => cache.put(
    "/broken-waiting-marker",
    new Response("waiting"),
  )));
});
`;

class LegacyUpgradeServer {
  private server: Server | null = null;
  private phase: "legacy" | "broken-waiting" | "current" = "legacy";

  constructor(private readonly port: number) {}

  get origin(): string {
    return `http://127.0.0.1:${this.port}`;
  }

  promoteBrokenWaitingWorker(): void {
    this.phase = "broken-waiting";
  }

  promoteCurrent(): void {
    this.phase = "current";
  }

  async start(): Promise<void> {
    this.server = createServer(async (request, response) => {
      try {
        const requestURL = new URL(request.url ?? "/", this.origin);
        if (this.phase !== "current") {
          const body = requestURL.pathname === "/sw.js"
            ? this.phase === "broken-waiting" ? brokenWaitingWorker : legacyWorker
            : requestURL.pathname === "/assets/legacy-app.js"
              ? legacyApp
              : legacyHTML;
          response.setHeader(
            "Content-Type",
            requestURL.pathname.endsWith(".js") ? "text/javascript" : "text/html",
          );
          response.setHeader("Cache-Control", "no-store, no-transform");
          response.setHeader("Service-Worker-Allowed", "/");
          response.writeHead(200).end(body);
          return;
        }

        const relativePath = decodeURIComponent(requestURL.pathname).replace(/^\/+/, "");
        let filePath = resolve(buildRoot, relativePath || "index.html");
        if (filePath !== buildRoot && !filePath.startsWith(`${buildRoot}${sep}`)) {
          response.writeHead(403).end("Forbidden");
          return;
        }
        try {
          if (!(await stat(filePath)).isFile()) throw new Error("not a file");
        } catch {
          filePath = resolve(buildRoot, "index.html");
        }
        const body = await readFile(filePath);
        response.setHeader("Content-Type", contentTypes[extname(filePath)] ?? "application/octet-stream");
        response.setHeader(
          "Cache-Control",
          requestURL.pathname === "/" ||
          requestURL.pathname === "/index.html" ||
          requestURL.pathname === "/app-shell-recovery.html" ||
          requestURL.pathname === "/sw.js" ||
          requestURL.pathname === "/app-shell-assets.json"
            ? "no-store, no-transform"
            : "public, max-age=31536000, immutable",
        );
        response.setHeader("Service-Worker-Allowed", "/");
        response.writeHead(200).end(body);
      } catch (error) {
        response.writeHead(500).end(error instanceof Error ? error.message : "server failure");
      }
    });
    await new Promise<void>((resolveStart, reject) => {
      this.server?.once("error", reject);
      this.server?.listen(this.port, "127.0.0.1", resolveStart);
    });
  }

  async stop(): Promise<void> {
    const active = this.server;
    this.server = null;
    if (!active) return;
    active.closeAllConnections();
    await new Promise<void>((resolveStop, reject) => {
      active.close((error) => error ? reject(error) : resolveStop());
    });
  }
}

test.beforeAll(() => {
  execFileSync("npm", ["exec", "--", "vite", "build"], {
    cwd: webRoot,
    env: {
      ...process.env,
      AVIA_BUILD_PROFILE: "demo",
      AVIA_APP_SHELL_PREDECESSOR_JSON: JSON.stringify(currentPredecessor),
      AVIA_APP_SHELL_LEGACY_PREDECESSOR_JSON: JSON.stringify(legacyPredecessor),
      AVIA_APP_SHELL_RELEASE_DESCRIPTOR_JSON: "",
    },
    stdio: "pipe",
  });
  const manifest = JSON.parse(readFileSync(resolve(buildRoot, "app-shell-assets.json"), "utf8")) as {
    predecessor?: { serviceWorkerURL?: string };
  };
  expect(manifest.predecessor?.serviceWorkerURL).toBe("/sw.js");
});

test("a normal root visit automatically upgrades a legacy client without a waiting candidate", async ({ context, page }) => {
  test.setTimeout(90_000);
  const server = new LegacyUpgradeServer(4190);
  const consoleErrors: string[] = [];
  try {
    await server.start();
    page.on("console", (message) => {
      if (message.type() === "error") consoleErrors.push(message.text());
    });
    page.on("pageerror", (error) => consoleErrors.push(error.message));
    await page.goto(server.origin);
    await page.evaluate(async () => navigator.serviceWorker.ready);
    await page.reload();
    await expect(page.getByRole("heading", { name: "Legacy shell" })).toBeVisible();
    await page.evaluate(() => localStorage.setItem("legacy-root-upgrade-sentinel", "preserved"));

    server.promoteCurrent();
    const secondPage = await context.newPage();
    secondPage.on("console", (message) => {
      if (message.type() === "error") consoleErrors.push(message.text());
    });
    secondPage.on("pageerror", (error) => consoleErrors.push(error.message));
    await secondPage.goto(server.origin);

    await expect(secondPage.getByRole("heading", { name: "AviaSurveil360" })).toBeVisible({
      timeout: 30_000,
    });
    await expect(page.getByRole("heading", { name: "AviaSurveil360" })).toBeVisible({
      timeout: 30_000,
    });
    expect(await secondPage.evaluate(() => localStorage.getItem("legacy-root-upgrade-sentinel"))).toBe(
      "preserved",
    );
    expect(consoleErrors).toEqual([]);
  } finally {
    await server.stop().catch(() => undefined);
  }
});

test("legacy clients do not block a repaired exact-vector successor", async ({ context, page }) => {
  test.setTimeout(90_000);
  const server = new LegacyUpgradeServer(4189);
  const consoleErrors: string[] = [];
  try {
    await server.start();
    context.on("page", (openedPage) => {
      openedPage.on("console", (message) => {
        if (message.type() === "error") consoleErrors.push(message.text());
      });
      openedPage.on("pageerror", (error) => consoleErrors.push(error.message));
    });
    page.on("console", (message) => {
      if (message.type() === "error") consoleErrors.push(message.text());
    });
    page.on("pageerror", (error) => consoleErrors.push(error.message));
    await page.goto(server.origin);
    await page.evaluate(async () => navigator.serviceWorker.ready);
    await page.reload();
    await expect(page.getByRole("heading", { name: "Legacy shell" })).toBeVisible();
    await page.evaluate(() => {
      localStorage.setItem("legacy-upgrade-sentinel", "preserved");
      (globalThis as typeof globalThis & { __setLegacyDirty(value: boolean): void }).__setLegacyDirty(true);
    });

    server.promoteBrokenWaitingWorker();
    await page.evaluate(async () => (await navigator.serviceWorker.getRegistration())?.update());
    await expect.poll(() => page.evaluate(async () => Boolean(
      (await navigator.serviceWorker.getRegistration())?.waiting,
    ))).toBe(true);
    await expect(page.getByRole("heading", { name: "Legacy shell" })).toBeVisible();

    server.promoteCurrent();
    await page.evaluate(async () => (await navigator.serviceWorker.getRegistration())?.update());
    const secondPage = await context.newPage();
    await secondPage.goto(`${server.origin}/app-shell-recovery.html?returnTo=%2F`);
    await expect(secondPage.getByRole("heading", { name: "Updating AviaSurveil360" })).toBeVisible();

    await expect(secondPage.getByRole("heading", { name: "AviaSurveil360" })).toBeVisible({
      timeout: 30_000,
    });
    await expect(page.getByRole("heading", { name: "Legacy shell" })).toBeVisible();
    await expect.poll(() => secondPage.evaluate(async () => {
      const registration = await navigator.serviceWorker.getRegistration();
      return registration?.active?.scriptURL ?? null;
    })).toBe(`${server.origin}/sw.js`);

    expect(await secondPage.evaluate(() => ({
      sentinel: localStorage.getItem("legacy-upgrade-sentinel"),
      scripts: Array.from(document.scripts, (script) => script.src),
    }))).toMatchObject({
      sentinel: "preserved",
      scripts: expect.not.arrayContaining([`${server.origin}/assets/legacy-app.js`]),
    });

    expect(await page.evaluate(() => localStorage.getItem("legacy-upgrade-sentinel"))).toBe(
      "preserved",
    );
    await page.evaluate(() => {
      (globalThis as typeof globalThis & { __setLegacyDirty(value: boolean): void }).__setLegacyDirty(false);
    });
    await expect(page.getByRole("heading", { name: "AviaSurveil360" })).toBeVisible({
      timeout: 30_000,
    });
    await expect(secondPage).toHaveURL(`${server.origin}/`);
    await expect.poll(() => secondPage.evaluate(async () => caches.keys())).not.toContain(
      "aviasurveil360-app-shell-v9",
    );
    await expect.poll(() => secondPage.evaluate(async () => caches.keys())).not.toContain(
      "aviasurveil360-app-shell-broken-waiting",
    );

    if (legacyUpgradeBrowser === "chromium") {
      const cdp = await context.newCDPSession(secondPage);
      await cdp.send("ServiceWorker.enable");
      await cdp.send("ServiceWorker.stopAllWorkers");
    }
    await server.stop();
    await secondPage.reload({ waitUntil: "domcontentloaded" });
    await expect(secondPage.getByRole("heading", { name: "AviaSurveil360" })).toBeVisible();
    expect(await secondPage.evaluate(() => localStorage.getItem("legacy-upgrade-sentinel"))).toBe(
      "preserved",
    );
    expect(consoleErrors).toEqual([]);
  } finally {
    await server.stop().catch(() => undefined);
  }
});
