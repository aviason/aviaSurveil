import { isRegisteredApplicationRoute } from "../offline/app-route-policy";
import "../styles/app-shell-recovery.css";

interface RecoveryManifest {
  releaseFingerprint: string;
}

const statusElement = document.getElementById("app-shell-recovery-status");
const retryButton = document.getElementById("app-shell-recovery-retry") as HTMLButtonElement | null;

function setStatus(message: string): void {
  if (statusElement) statusElement.textContent = message;
}

function returnToPath(): string {
  const raw = new URLSearchParams(window.location.search).get("returnTo") ?? "/";
  try {
    const target = new URL(raw, window.location.origin);
    if (target.origin !== window.location.origin || !isRegisteredApplicationRoute(target.pathname)) return "/";
    return `${target.pathname}${target.search}${target.hash}`;
  } catch {
    return "/";
  }
}

async function installedWorker(registration: ServiceWorkerRegistration): Promise<ServiceWorker | null> {
  if (registration.waiting) return registration.waiting;
  const installing = registration.installing;
  if (!installing) return null;
  if (installing.state === "installed") return installing;
  return new Promise((resolve) => {
    const onStateChange = () => {
      if (installing.state === "installed") {
        installing.removeEventListener("statechange", onStateChange);
        resolve(installing);
      } else if (installing.state === "activated" || installing.state === "redundant") {
        installing.removeEventListener("statechange", onStateChange);
        resolve(null);
      }
    };
    installing.addEventListener("statechange", onStateChange);
  });
}

async function activateWaitingWorker(
  registration: ServiceWorkerRegistration,
  worker: ServiceWorker,
  fingerprint: string,
): Promise<void> {
  if (worker.state === "activated" || registration.active === worker) return;
  await new Promise<void>((resolve, reject) => {
    const timeout = window.setTimeout(() => {
      navigator.serviceWorker.removeEventListener("controllerchange", onControllerChange);
      reject(new Error("The verified update did not activate in time."));
    }, 15_000);
    const onControllerChange = () => {
      window.clearTimeout(timeout);
      navigator.serviceWorker.removeEventListener("controllerchange", onControllerChange);
      resolve();
    };
    navigator.serviceWorker.addEventListener("controllerchange", onControllerChange);
    worker.postMessage({
      type: "avia:app-shell-recovery-activate",
      fingerprint,
    });
  });
}

async function recover(): Promise<void> {
  if (!("serviceWorker" in navigator)) {
    window.location.replace(returnToPath());
    return;
  }
  setStatus("Checking the stable application worker…");
  const registration = await navigator.serviceWorker.register("/sw.js", {
    scope: "/",
    type: "module",
    updateViaCache: "none",
  });
  await registration.update();
  const manifestResponse = await fetch("/app-shell-assets.json", {
    cache: "no-store",
    credentials: "same-origin",
  });
  if (!manifestResponse.ok) throw new Error("The release manifest is unavailable.");
  const manifest = await manifestResponse.json() as RecoveryManifest;
  if (!/^sha256:[0-9a-f]{64}$/.test(manifest.releaseFingerprint)) {
    throw new Error("The release manifest identity is invalid.");
  }
  const waiting = await installedWorker(registration);
  if (waiting) {
    setStatus("Activating the verified release without clearing local work…");
    await activateWaitingWorker(registration, waiting, manifest.releaseFingerprint);
  }
  setStatus("Update complete. Opening your workspace…");
  window.location.replace(returnToPath());
}

retryButton?.addEventListener("click", () => window.location.reload());

void recover().catch(() => {
  setStatus("The update could not be completed automatically. Please retry.");
  if (retryButton) retryButton.hidden = false;
});
