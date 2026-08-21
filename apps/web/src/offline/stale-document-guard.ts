import { staleDocumentFingerprint } from "./update-coordinator";
import { appShellRecoveryURL } from "./client-quiescence";

interface StaleDocumentGuardEnvironment {
  entryURL: string;
  loadManifest(): Promise<unknown>;
  replace(url: string): void;
  pathname: string;
  search: string;
  hash: string;
}

export async function detectStaleDocument(
  environment: Pick<StaleDocumentGuardEnvironment, "entryURL" | "loadManifest">,
): Promise<string | null> {
  return staleDocumentFingerprint(await environment.loadManifest(), environment.entryURL);
}

export function installStaleDocumentGuard(
  environment: StaleDocumentGuardEnvironment = browserEnvironment(),
): void {
  if (!import.meta.env.PROD || !environment.entryURL) return;
  void detectStaleDocument(environment)
    .then((fingerprint) => {
      if (!fingerprint) return;
      environment.replace(appShellRecoveryURL(environment.pathname, environment.search, environment.hash));
    })
    .catch(() => {
      // A transient manifest failure must not block the current document.
    });
}

function browserEnvironment(): StaleDocumentGuardEnvironment {
  const entry = [...document.querySelectorAll<HTMLScriptElement>("script[type=module][src]")].find((candidate) => {
    const pathname = new URL(candidate.src, window.location.href).pathname;
    return candidate.dataset.aviaAppEntry === "true" || /\/assets\/app-[A-Za-z0-9_-]+\.js$/.test(pathname);
  });
  return {
    entryURL: entry?.src ?? "",
    loadManifest: async () => {
      const response = await fetch("/app-shell-assets.json", {
        cache: "no-store",
        credentials: "same-origin",
        headers: { accept: "application/json" },
      });
      if (!response.ok) throw new Error(`App-shell manifest probe failed with ${response.status}.`);
      return response.json();
    },
    replace: (url) => window.location.replace(url),
    pathname: window.location.pathname,
    search: window.location.search,
    hash: window.location.hash,
  };
}
