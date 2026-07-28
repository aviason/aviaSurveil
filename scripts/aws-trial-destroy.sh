#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/lib/aws-trial-gates.sh"

mode="${1:-preview}"
phase="${2:-data-runtime}"
bundle="${3:-}"
aws_trial_require_phase "${phase}"

case "${mode}" in
  preview)
    aws_trial_preview destroy "${phase}" "${bundle}"
    printf 'destroy requires a reviewed tagged-resource inventory and exact resource-manifest plan\n'
    ;;
  execute)
    # check-aws-plan.sh and exact-authorization bind this one phase only.
    aws_trial_check_bundle "${bundle}"
    aws_trial_require_exact_authorization destroy "${bundle}"
    [[ -f "${AVIA_AWS_TAGGED_RESOURCE_MANIFEST:-}" ]] ||
      aws_trial_fail "tagged-resource: exact resource-manifest is required"
    [[ -f "${AVIA_AWS_DESTROY_PLAN:-}" ]] ||
      aws_trial_fail "missing-artifact: reviewed destroy plan is required"
    [[ "${AVIA_AWS_RESIDUE_CHECK_COMMAND:-}" == "approved-tagged-residue-check" ]] ||
      aws_trial_fail "tagged-resource: residue check approval is required"
    terraform apply -input=false -auto-approve "${AVIA_AWS_DESTROY_PLAN}"
    printf 'destroy complete; run the approved tagged-resource residue check\n'
    ;;
  *)
    aws_trial_fail "usage: expected preview or execute"
    ;;
esac
