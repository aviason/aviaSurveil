#!/usr/bin/env bash
set -Eeuo pipefail

SCRIPT_DIRECTORY="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
WEB_ROOT="$(cd "${SCRIPT_DIRECTORY}/../apps/web" && pwd)"
WORK_ROOT="$(mktemp -d /private/tmp/aviasurveil360-app-shell-harness.XXXXXX)"
trap 'rm -rf "$WORK_ROOT"' EXIT INT TERM

build_profile() {
  local label="$1"
  local predecessor_json="${2:-}"
  rm -rf "${WEB_ROOT}/dist/demo"
  if [[ -n "$predecessor_json" ]]; then
    AVIA_BUILD_PROFILE=demo AVIA_APP_SHELL_PREDECESSOR_JSON="$predecessor_json" npm run build:demo --prefix "$WEB_ROOT" >/dev/null
  else
    AVIA_BUILD_PROFILE=demo npm run build:demo --prefix "$WEB_ROOT" >/dev/null
  fi
  node "${WEB_ROOT}/scripts/assert-app-shell-artifact.mjs" "${WEB_ROOT}/dist/demo" >/dev/null
  PYTHONDONTWRITEBYTECODE=1 python3 "${WEB_ROOT}/../../../../scripts/app_shell_artifact.py" verify-web-artifact "${WEB_ROOT}/dist/demo" >/dev/null
  cp -R "${WEB_ROOT}/dist/demo" "${WORK_ROOT}/${label}"
}

build_profile A
node -e 'const fs=require("node:fs"); const a=JSON.parse(fs.readFileSync(process.argv[1],"utf8")); fs.writeFileSync(process.argv[2], JSON.stringify(a.releaseDescriptor));' "${WORK_ROOT}/A/app-shell-assets.json" "${WORK_ROOT}/A-predecessor.json"
build_profile B "$(cat "${WORK_ROOT}/A-predecessor.json")"
node -e 'const fs=require("node:fs"); const b=JSON.parse(fs.readFileSync(process.argv[1],"utf8")); fs.writeFileSync(process.argv[2], JSON.stringify(b.releaseDescriptor));' "${WORK_ROOT}/B/app-shell-assets.json" "${WORK_ROOT}/B-predecessor.json"
build_profile C "$(cat "${WORK_ROOT}/B-predecessor.json")"

if cmp -s "${WORK_ROOT}/A/app-shell-assets.json" "${WORK_ROOT}/B/app-shell-assets.json"; then
  echo "blocked: predecessor-bound builds must not share a release fingerprint" >&2
  exit 1
fi
for label in A B C; do
  test -s "${WORK_ROOT}/${label}/sw.js"
  test -s "${WORK_ROOT}/${label}/assets/$(find "${WORK_ROOT}/${label}/assets" -maxdepth 1 -type f -name 'inspector-assignments-page-*.js' -print -quit | xargs -n1 basename)"
  rg -q 'force-window-client-navigation-v1' "${WORK_ROOT}/${label}/sw.js"
  rg -q '\.navigate\(' "${WORK_ROOT}/${label}/sw.js"
done
if rg -q 'committedClientCaches|committed\.map\(\(\{ cacheName \}\)' "${WEB_ROOT}/src/sw.ts"; then
  echo "blocked: legacy app-shell asset fallback remains in the Service Worker" >&2
  exit 1
fi
echo "verified locally: deterministic A/B/C app-shell artifacts, exact fingerprints, lazy-chunk presence, and independent validators"
