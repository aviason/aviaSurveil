#!/usr/bin/env bash
set -euo pipefail

umask 077

repository_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
compose_file="$repository_root/deploy/local/compose.yaml"
operation="${1:-}"
configuration_file="${AVIA_AGA_DEMO_CONFIG_FILE:-}"
package_file="${AVIA_AGA_DEMO_PACKAGE_FILE:-}"
authorization_file="${AVIA_AGA_DEMO_AUTHORIZATION_FILE:-}"
base_evidence_file="${AVIA_AGA_DEMO_BASE_EVIDENCE_FILE:-}"
control_store_directory="${AVIA_AGA_DEMO_CONTROL_STORE_DIR:-}"

fail() { printf 'preprod-aga-candidate-demo-loader: %s\n' "$*" >&2; exit 1; }
[[ "$operation" == "prepare-aga-demo" || "$operation" == "verify-aga-demo-authorization" || "$operation" == "run-aga-demo" || "$operation" == "verify-aga-demo" || "$operation" == "cleanup-aga-demo" ]] || fail "usage: $0 prepare-aga-demo|verify-aga-demo-authorization|run-aga-demo|verify-aga-demo|cleanup-aga-demo"
for path in "$configuration_file" "$package_file" "$base_evidence_file" "$control_store_directory"; do
  [[ -n "$path" && "$path" = /* ]] || fail "absolute config, package, base-evidence, and control-store paths are required"
done
[[ -f "$configuration_file" && ! -L "$configuration_file" ]] || fail "configuration must be a regular file"
[[ -f "$package_file" && ! -L "$package_file" ]] || fail "package must be a regular file"
[[ -f "$base_evidence_file" && ! -L "$base_evidence_file" ]] || fail "base evidence must be a regular file"
[[ -d "$control_store_directory" && ! -L "$control_store_directory" ]] || fail "control store must be a directory"
chmod 600 "$configuration_file" "$base_evidence_file"
if [[ "$operation" == "verify-aga-demo-authorization" || "$operation" == "run-aga-demo" || "$operation" == "cleanup-aga-demo" ]]; then
  [[ -n "$authorization_file" && "$authorization_file" = /* && -f "$authorization_file" && ! -L "$authorization_file" ]] || fail "private authorization file is required"
  chmod 600 "$authorization_file"
fi

compose_args=(
  docker compose --project-name aviasurveil360-local-preprod --file "$compose_file"
  --profile local-preprod-loader --profile aga-candidate-demo run --rm
  --volume "$configuration_file:/run/config/aga-demo.json:ro"
  --volume "$package_file:/run/input/AGA_ALL_FORMS_SOURCE_RISK_DRAFT_2026-08-01.zip:ro"
  --volume "$base_evidence_file:/run/evidence/base-result.json:ro"
  --volume "$control_store_directory:/var/lib/aviasurveil360-preprod-control:rw"
)
if [[ -n "$authorization_file" ]]; then
  compose_args+=(--volume "$authorization_file:/run/secrets/aga_demo_authorization:ro")
fi
"${compose_args[@]}" preprod-aga-candidate-demo-loader "$operation" /run/config/aga-demo.json
