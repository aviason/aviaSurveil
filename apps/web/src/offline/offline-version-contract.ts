export interface OfflineVersionVector {
  appShellVersion: number;
  indexedDbSchemaVersion: number;
  packageSchemaVersion: number;
  syncProtocolVersion: number;
}

// This module is intentionally dependency-free. It is imported by browser
// code, the Service Worker, and the Node-side Vite build without pulling in
// Dexie or any other runtime-only dependency.
export const CURRENT_OFFLINE_VERSIONS: Readonly<OfflineVersionVector> = {
  appShellVersion: 9,
  indexedDbSchemaVersion: 2,
  packageSchemaVersion: 1,
  syncProtocolVersion: 1,
};

export function sameOfflineVersionVector(
  left: OfflineVersionVector,
  right: OfflineVersionVector,
): boolean {
  return (
    left.appShellVersion === right.appShellVersion &&
    left.indexedDbSchemaVersion === right.indexedDbSchemaVersion &&
    left.packageSchemaVersion === right.packageSchemaVersion &&
    left.syncProtocolVersion === right.syncProtocolVersion
  );
}
