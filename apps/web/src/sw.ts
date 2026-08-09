/// <reference lib="webworker" />

export type AppShellRequestPolicy =
  | "app-shell-navigation"
  | "versioned-static-asset"
  | "network-only";

export interface AppShellRequestDescriptor {
  url: string;
  method: string;
  mode: string;
}

const STATIC_CONFIG_PATHS = new Set([
  "/app-shell-assets.json",
  "/demo-build.json",
  "/http-config.json",
  "/index.html",
]);

export function classifyAppShellRequest(
  request: AppShellRequestDescriptor,
  origin: string,
): AppShellRequestPolicy {
  if (request.method !== "GET") return "network-only";
  const url = new URL(request.url);
  if (url.origin !== origin) return "network-only";
  if (
    /^\/(?:api|v1|auth|health|identity|__test|evidence-quarantine|evidence-clean|inspection-attachments|generated-documents)(?:\/|$)/.test(
      url.pathname,
    )
  ) {
    return "network-only";
  }
  if (request.mode === "navigate") return "app-shell-navigation";
  if (/^\/assets\/[A-Za-z0-9_.-]+\.(?:css|js|map|svg|png|jpg|jpeg|webp|ttf|woff2?)$/.test(url.pathname)) {
    return "versioned-static-asset";
  }
  if (STATIC_CONFIG_PATHS.has(url.pathname)) return "versioned-static-asset";
  return "network-only";
}

interface AppShellManifest {
  appShellVersion: number;
  assets: string[];
}

// Increment this whenever the app shell or its static asset graph changes so
// an older service-worker cache cannot keep serving a previous UI indefinitely.
const APP_SHELL_VERSION_MARKER = "AVIA_APP_SHELL_VERSION:000007";
const APP_SHELL_VERSION = Number(
  /^AVIA_APP_SHELL_VERSION:(\d{6})$/.exec(APP_SHELL_VERSION_MARKER)?.[1],
);
if (!Number.isSafeInteger(APP_SHELL_VERSION) || APP_SHELL_VERSION < 1) {
  throw new Error("App-shell build version marker is invalid");
}
const APP_SHELL_CACHE = `aviasurveil360-app-shell-v${APP_SHELL_VERSION}`;
const serviceWorkerScope = globalThis as unknown as ServiceWorkerGlobalScope;

async function installAppShell(): Promise<void> {
  const response = await fetch("/app-shell-assets.json", { cache: "no-store" });
  if (!response.ok) throw new Error(`App-shell manifest failed with status ${response.status}`);
  const manifest = (await response.json()) as AppShellManifest;
  if (manifest.appShellVersion !== APP_SHELL_VERSION || !Array.isArray(manifest.assets)) {
    throw new Error("App-shell manifest version is incompatible");
  }
  // Keep the root response as the canonical navigation shell. The local
  // server must not make navigations depend on a redirected /index.html
  // response: WebKit rejects a service-worker navigation response whose URL
  // was redirected, which breaks the OAuth callback return to "/".
  const assets = [...new Set(["/", "/app-shell-assets.json", ...manifest.assets])];
  const cache = await caches.open(APP_SHELL_CACHE);
  await cache.addAll(assets);
}

async function serveAppShellNavigation(request: Request): Promise<Response> {
  const cache = await caches.open(APP_SHELL_CACHE);
  return (await cache.match(request)) ?? (await cache.match("/")) ?? (await fetch(request));
}

async function serveVersionedAsset(request: Request): Promise<Response> {
  const cache = await caches.open(APP_SHELL_CACHE);
  return (await cache.match(request)) ?? (await fetch(request));
}

if (
  typeof serviceWorkerScope.addEventListener === "function" &&
  "registration" in serviceWorkerScope &&
  typeof caches !== "undefined"
) {
  serviceWorkerScope.addEventListener("install", (event: ExtendableEvent) => {
    event.waitUntil(installAppShell());
  });

  serviceWorkerScope.addEventListener("fetch", (event: FetchEvent) => {
    const policy = classifyAppShellRequest(event.request, serviceWorkerScope.location.origin);
    if (policy === "app-shell-navigation") {
      event.respondWith(serveAppShellNavigation(event.request));
    } else if (policy === "versioned-static-asset") {
      event.respondWith(serveVersionedAsset(event.request));
    }
  });
}
