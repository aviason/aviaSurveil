#!/usr/bin/env bash
set -euo pipefail

repository_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
environment_root="$repository_root/infra/terragrunt/environments/aws-ipv6-trial"
input_file=${AVIA_IPV6_TG_INPUTS_FILE:-}
mode=${1:-offline}
plan_directory=${AVIA_TG_PLAN_DIR:-}

if [[ "$mode" != "contract" && -z "$input_file" ]]; then
  echo "missing-owner-input: set AVIA_IPV6_TG_INPUTS_FILE to an untracked owner overlay" >&2
  exit 64
fi
if [[ -n "$input_file" && ! -f "$input_file" ]]; then
  echo "missing-owner-input: overlay does not exist: $input_file" >&2
  exit 64
fi

required=(
  "$environment_root/root.hcl"
  "$environment_root/account.hcl"
  "$environment_root/environment.hcl"
  "$environment_root/region.hcl.example"
  "$environment_root/components/network/terragrunt.hcl"
  "$environment_root/components/registry/terragrunt.hcl"
  "$environment_root/components/edge-runtime-auth/terragrunt.hcl"
  "$environment_root/components/budget/terragrunt.hcl"
  "$environment_root/components/compute/terragrunt.hcl"
)
for file in "${required[@]}"; do
  [[ -f "$file" ]] || { echo "layout-contract: missing $file" >&2; exit 65; }
done

if [[ "$mode" == "hook-before" || "$mode" == "hook-after" ]]; then
  [[ -n "$plan_directory" ]] || { echo "missing-owner-input: AVIA_TG_PLAN_DIR is required for protected plan artifacts" >&2; exit 64; }
  mkdir -p -m 0700 "$plan_directory"
  plan_mode=$(stat -f '%Lp' "$plan_directory" 2>/dev/null || stat -c '%a' "$plan_directory")
  [[ "$plan_mode" == "700" ]] || { echo "artifact-permission: AVIA_TG_PLAN_DIR must be private" >&2; exit 65; }
fi

if rg --quiet --glob '*.hcl' 'aws-trial|aws_autoscaling_group|aws_nat_gateway|aws_eip|aws_lb|aws_db_instance|aws_vpc_endpoint|0\.0\.0\.0/0|linux/amd64|qemu|rosetta' "$environment_root"; then
  echo "layout-contract: prohibited aws-trial topology or architecture marker" >&2
  exit 65
fi

if [[ "$mode" == "hook-after" ]]; then
  plan_file=${2:-}
  [[ -n "$plan_file" && -f "$plan_file" ]] || { echo "missing-plan-artifact: protected local plan path required" >&2; exit 66; }
  command -v terraform >/dev/null 2>&1 || { echo "not-run: terraform is required for local plan redaction" >&2; exit 78; }
  plan_json=$(terraform show -json "$plan_file") || { echo "plan-redaction: terraform show failed" >&2; exit 65; }
  if ! printf '%s' "$plan_json" | node "$repository_root/scripts/lib/aws-ipv6-trial-redaction.mjs"; then
    echo "plan-redaction: sensitive plan material detected" >&2
    exit 65
  fi
fi

echo "verified locally: aws-ipv6-trial Terragrunt layout and offline boundary"
