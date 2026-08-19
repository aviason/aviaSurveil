export const APP_SHELL_UPDATE_PROTOCOL_VERSION = 2;

const LEGACY_QUIESCENT_RELOAD_PREFIX = "avia:legacy-quiescent-reload:";

export function legacyQuiescentReloadFingerprint(releaseFingerprint: string): string {
  return `${LEGACY_QUIESCENT_RELOAD_PREFIX}${releaseFingerprint}`;
}

export function releaseFingerprintFromActivationMessage(value: unknown): string | null {
  if (!value || typeof value !== "object") return null;
  const message = value as { fingerprint?: unknown; releaseFingerprint?: unknown };
  if (typeof message.releaseFingerprint === "string") return message.releaseFingerprint;
  if (typeof message.fingerprint !== "string") return null;
  return message.fingerprint.startsWith(LEGACY_QUIESCENT_RELOAD_PREFIX)
    ? message.fingerprint.slice(LEGACY_QUIESCENT_RELOAD_PREFIX.length)
    : message.fingerprint;
}
