#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
mode="${1:-static}"

if [[ "$mode" == "bootstrap-plan-hook" ]]; then
  [[ "${AWS_PROFILE:-}" == "avia" ]] || {
    printf '%s\n' 'blocked: aws-operator-profile-must-be-avia; default and omitted AWS profiles are forbidden' >&2
    exit 76
  }
  [[ "${AWS_REGION:-}" == "eu-central-1" && "${AWS_DEFAULT_REGION:-}" == "eu-central-1" ]] || {
    printf '%s\n' 'blocked: aws-region-must-be-eu-central-1' >&2
    exit 76
  }
  case "${TG_TF_PATH:-}" in
    terraform | */terraform) ;;
    *)
      printf '%s\n' 'blocked: terraform-cli-must-be-selected-explicitly; OpenTofu default is not accepted for this plan' >&2
      exit 76
      ;;
  esac
  [[ "${AVIA_AWS_PRIVATE_PILOT_PLAN_AUTHORIZATION:-}" == "remote-state-bootstrap-provider-plan-no-apply" ]] || {
    printf '%s\n' 'blocked: exact bootstrap provider-plan authorization is required' >&2
    exit 77
  }
  printf '%s\n' 'authorized: remote-state bootstrap provider plan only; apply remains blocked'
  exit 0
fi

if [[ "$mode" == "remote-hook" ]]; then
  if [[ "${AWS_PROFILE:-}" != "avia" ]]; then
    printf '%s\n' 'blocked: aws-operator-profile-must-be-avia; default and omitted AWS profiles are forbidden' >&2
    exit 76
  fi
  printf '%s\n' 'blocked: remote-action-unauthorized; Task 7 requires a separate exact authorization bundle' >&2
  exit 77
fi

if [[ "$mode" != "static" ]]; then
  printf 'unsupported infrastructure check mode: %s\n' "$mode" >&2
  exit 64
fi

node --test "$repo_root/tests/aws-private-pilot-infrastructure-contract.test.mjs"
terraform -chdir="$repo_root/infra/terraform" fmt -check -recursive
terragrunt hcl fmt --check --working-dir "$repo_root/infra/terragrunt/environments/aws-private-pilot"
terragrunt hcl fmt --check --working-dir "$repo_root/infra/terragrunt/fixtures"

printf '%s\n' 'verified locally: private-pilot infrastructure source and mutation contracts; provider-backed plan not run'
