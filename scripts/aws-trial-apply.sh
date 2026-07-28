#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/lib/aws-trial-gates.sh"

mode="${1:-preview}"
phase="${2:-}"
bundle="${3:-}"
aws_trial_require_phase "${phase}"

case "${mode}" in
  preview)
    # The preview prints the exact-authorization value without executing it.
    aws_trial_preview apply "${phase}" "${bundle}"
    ;;
  execute)
    # check-aws-plan.sh verifies caller, plan, decision, wrapper, image, and policy hashes.
    aws_trial_check_bundle "${bundle}"
    aws_trial_require_exact_authorization apply "${bundle}"
    [[ "$(aws_trial_manifest_field "${bundle}" phase)" == "${phase}" ]] ||
      aws_trial_fail "phase-boundary: bundle phase mismatch"
    node_bin="$(aws_trial_node)"
    while IFS='|' read -r unit plan_path; do
      printf 'applying reviewed unit %s\n' "${unit}"
      if [[ "${unit}" == "remote-state" ]]; then
        unit_directory="${aws_trial_repository_root}/infra/terragrunt/bootstrap/remote-state"
      else
        unit_directory="${aws_trial_repository_root}/infra/terragrunt/environments/aws-trial/components/${unit}"
      fi
      terragrunt apply \
        --working-dir "${unit_directory}" \
        -- -input=false -auto-approve "${plan_path}"
    done < <("${node_bin}" "${aws_trial_control}" plans "${bundle}")
    ;;
  *)
    aws_trial_fail "usage: expected preview or execute"
    ;;
esac
