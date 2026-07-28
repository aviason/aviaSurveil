#!/usr/bin/env bash
set -euo pipefail

repository_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
terragrunt_root="${repository_root}/infra/terragrunt"
components_root="${terragrunt_root}/environments/aws-trial/components"
mode="${1:-all}"

fail() {
  printf 'check-terragrunt: %s\n' "$*" >&2
  exit 64
}

require_fixture_inputs() {
  [[ -n "${AVIA_TG_INPUTS_FILE:-}" ]] ||
    fail "missing-owner-input: AVIA_TG_INPUTS_FILE"
  [[ -f "${AVIA_TG_INPUTS_FILE}" ]] ||
    fail "missing-owner-input: AVIA_TG_INPUTS_FILE does not resolve to a file"
  [[ -n "${AVIA_TG_PLAN_DIR:-}" ]] ||
    fail "missing-owner-input: AVIA_TG_PLAN_DIR"
  mkdir -p -m 0700 "${AVIA_TG_PLAN_DIR}"
}

require_fixture_inputs

case "${mode}" in
  hook-before)
    exit 0
    ;;
  hook-after)
    [[ -n "${2:-}" && -f "${2}" ]] ||
      fail "policy-hook: expected deterministic plan artifact"
    exit 0
    ;;
  all)
    ;;
  *)
    fail "unknown mode ${mode}"
    ;;
esac

command -v terraform >/dev/null || fail "terraform is required"
command -v terragrunt >/dev/null || fail "terragrunt is required"

export AWS_ACCESS_KEY_ID=fixture
export AWS_SECRET_ACCESS_KEY=fixture
export AWS_SESSION_TOKEN=fixture
export AWS_EC2_METADATA_DISABLED=true
export TF_CLI_ARGS_init=-reconfigure
export TG_NON_INTERACTIVE=true
export TG_NO_COLOR=true

terragrunt hcl fmt --check --working-dir "${terragrunt_root}"
terragrunt hcl validate --working-dir "${terragrunt_root}"
terragrunt dag graph --working-dir "${components_root}"
terragrunt run --all validate --parallelism 1 --working-dir "${components_root}"
terragrunt run --all plan --parallelism 1 --source-update --working-dir "${components_root}"

plan_count="$(find "${AVIA_TG_PLAN_DIR}" -maxdepth 1 -type f -name '*.tfplan' | wc -l | tr -d ' ')"
[[ "${plan_count}" == "12" ]] ||
  fail "policy-hook: expected 12 component plan artifacts, found ${plan_count}"
printf 'check-terragrunt: verified 12 non-deployable component plans\n'
