#!/usr/bin/env bash
set -euo pipefail

repository_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
compose_file="$repository_root/deploy/aws-ipv6-trial/compose.yaml"
project=${AVIA_TRIAL_PROJECT:-"aviasurveil360-aws-ipv6-trial-${PPID}"}
evidence_root=${AVIA_TRIAL_EVIDENCE_ROOT:-"$repository_root/.local/aviasurveil360/aws-ipv6-trial/runtime"}
run_mode=false

if [[ "${1:-}" == "--run" ]]; then
  run_mode=true
elif [[ "${1:-}" == "--contract-only" || -z "${1:-}" ]]; then
  echo "not run: ARM64 runtime capacity/browser observation requires --run, local digest-bound images, and a root-owned 0600 tunnel token file" >&2
  exit 78
else
  echo "usage: test-aws-ipv6-trial-runtime.sh [--run|--contract-only]" >&2
  exit 64
fi

case "$(uname -m)" in
  arm64|aarch64) ;;
  *) echo "host-architecture: native ARM64 host required; emulation is forbidden" >&2; exit 64 ;;
esac
command -v docker >/dev/null 2>&1 || { echo "not-run: docker is unavailable" >&2; exit 78; }
command -v jq >/dev/null 2>&1 || { echo "not-run: jq is unavailable" >&2; exit 78; }

for variable in AVIA_TRIAL_CLOUDFLARED_IMAGE AVIA_TRIAL_GATEWAY_IMAGE AVIA_TRIAL_WEB_DEMO_IMAGE AVIA_TRIAL_TUNNEL_TOKEN_FILE; do
  [[ -n "${!variable:-}" ]] || { echo "missing-runtime-input:$variable" >&2; exit 64; }
done
for variable in AVIA_TRIAL_CLOUDFLARED_IMAGE AVIA_TRIAL_GATEWAY_IMAGE AVIA_TRIAL_WEB_DEMO_IMAGE; do
  [[ "${!variable}" =~ ^[^:[:space:]]+@sha256:[0-9a-f]{64}$ ]] || { echo "image-contract:$variable must be @sha256-bound" >&2; exit 64; }
done
[[ -f "$AVIA_TRIAL_TUNNEL_TOKEN_FILE" ]] || { echo "missing-runtime-input:tunnel-token-file" >&2; exit 64; }
token_mode=$(stat -f '%Lp' "$AVIA_TRIAL_TUNNEL_TOKEN_FILE" 2>/dev/null || stat -c '%a' "$AVIA_TRIAL_TUNNEL_TOKEN_FILE")
[[ "$token_mode" == "600" || "$token_mode" == "400" ]] || { echo "secret-permission: tunnel token must be 0600 or 0400" >&2; exit 65; }

mkdir -p "$evidence_root"
chmod 0700 "$evidence_root"
compose=(docker compose --project-name "$project" --file "$compose_file")
cleanup() {
  set +e
  "${compose[@]}" down --volumes --remove-orphans --timeout 15 >/dev/null 2>&1
  residue=$(docker ps --all --filter "label=com.docker.compose.project=$project" --format '{{.ID}}'; docker volume ls --filter "label=com.docker.compose.project=$project" --format '{{.Name}}'; docker network ls --filter "label=com.docker.compose.project=$project" --format '{{.Name}}')
  if [[ -n "$residue" ]]; then
    echo "cleanup-failure: task-owned runtime residue remains" >&2
    printf '%s\n' "$residue" >&2
    exit 70
  fi
}
trap cleanup EXIT HUP INT TERM

export AVIA_TRIAL_PROJECT="$project"
"${compose[@]}" config --quiet
services_json=$("${compose[@]}" config --format json)
[[ "$(jq -r '.services | keys | join(",")' <<<"$services_json")" == "cloudflared,gateway,web-demo" ]] || { echo "runtime-contract: unexpected service set" >&2; exit 65; }
[[ "$(jq '[.services[].ports // []] | add | length' <<<"$services_json")" == "0" ]] || { echo "runtime-contract: host-published application port" >&2; exit 65; }

if [[ "$run_mode" == true ]]; then
  "${compose[@]}" up --detach --wait
  "${compose[@]}" exec --no-TTY gateway /healthcheck
  "${compose[@]}" exec --no-TTY web-demo /web-server healthcheck
  "${compose[@]}" exec --no-TTY cloudflared cloudflared version >/dev/null
  docker stats --no-stream --format '{{.Name}}\t{{.MemUsage}}\t{{.CPUPerc}}\t{{.PIDs}}' >"$evidence_root/stats.tsv"
  "${compose[@]}" ps --all >"$evidence_root/compose-ps.txt"
  cat >"$evidence_root/capacity.json" <<'EOF'
{
  "status": "candidate-only",
  "shortRun": "verified locally",
  "thirtyMinuteBrowserLoop": "not run",
  "t4gSmallGoNoGo": "not established",
  "headroomThreshold": "20 percent",
  "swapDependency": "not accepted"
}
EOF
  chmod 0600 "$evidence_root/capacity.json" "$evidence_root/stats.tsv" "$evidence_root/compose-ps.txt"
  echo "verified locally: ARM64 three-service runtime started; 30-minute capacity/browser gate remains not run"
fi
