import { staleDocumentFingerprint } from "./update-coordinator";
import { appShellRecoveryURL } from "./client-quiescence";

interface StaleDocumentGuardEnvironment {
  entryURL: string;
  loadManifest(): Promise<unknown>;
  loadEntryDigest?(): Promise<string | null>;
  replace(url: string): void;
  pathname: string;
  search: string;
  hash: string;
}

export async function detectStaleDocument(
  environment: Pick<StaleDocumentGuardEnvironment, "entryURL" | "loadManifest" | "loadEntryDigest">,
): Promise<string | null> {
  const manifest = await environment.loadManifest();
  const digest = environment.loadEntryDigest ? await environment.loadEntryDigest() : null;
  return staleDocumentFingerprint(manifest, environment.entryURL, digest);
}

export function installStaleDocumentGuard(
  environment?: StaleDocumentGuardEnvironment,
): Promise<boolean> {
  if (!import.meta.env.PROD) return Promise.resolve(false);
  const target = environment ?? browserEnvironment();
  if (!target.entryURL) return Promise.resolve(false);
  return detectStaleDocument(target)
    .then((fingerprint) => {
      if (!fingerprint) return false;
      target.replace(appShellRecoveryURL(target.pathname, target.search, target.hash));
      return true;
    })
    .catch(() => {
      // A transient manifest failure must not block the current document.
      return false;
    });
}

let staleDocumentGuardReady: Promise<boolean> = Promise.resolve(false);

export function waitForStaleDocumentGuard(): Promise<boolean> {
  return staleDocumentGuardReady;
}

function browserEnvironment(): StaleDocumentGuardEnvironment {
  const entry = [...document.querySelectorAll<HTMLScriptElement>("script[type=module][src]")].find((candidate) => {
    const pathname = new URL(candidate.src, window.location.href).pathname;
    return candidate.dataset.aviaAppEntry === "true" || /\/assets\/app-[A-Za-z0-9_-]+\.js$/.test(pathname);
  });
  return {
    entryURL: entry?.src ?? "",
    loadManifest: async () => {
      const manifestURL = new URL("/app-shell-assets.json", window.location.href);
      manifestURL.searchParams.set("avia_shell_probe", `${Date.now()}-${Math.random().toString(36).slice(2)}`);
      const response = await fetch(manifestURL, {
        cache: "no-store",
        credentials: "same-origin",
        headers: { accept: "application/json" },
      });
      if (!response.ok) throw new Error(`App-shell manifest probe failed with ${response.status}.`);
      return response.json();
    },
    loadEntryDigest: async () => {
      const assetURL = new URL(entry?.src ?? "", window.location.href);
      assetURL.searchParams.set("avia_shell_probe", `${Date.now()}-${Math.random().toString(36).slice(2)}`);
      const response = await fetch(assetURL, {
        cache: "no-store",
        credentials: "same-origin",
        headers: { accept: "text/javascript" },
      });
      if (!response.ok) throw new Error(`App-shell asset probe failed with ${response.status}.`);
      const bytes = await response.arrayBuffer();
      const hash = await crypto.subtle.digest("SHA-256", bytes);
      return `sha256:${Array.from(new Uint8Array(hash), (byte) => byte.toString(16).padStart(2, "0")).join("")}`;
    },
    replace: (url) => window.location.replace(url),
    pathname: window.location.pathname,
    search: window.location.search,
    hash: window.location.hash,
  };
}

staleDocumentGuardReady = installStaleDocumentGuard();
