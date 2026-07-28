#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/lib/aws-trial-gates.sh"

mode="${1:-preview}"
phase="${2:-data-runtime}"
bundle="${3:-}"
aws_trial_require_phase "${phase}"

case "${mode}" in
  preview)
    aws_trial_preview rollback "${phase}" "${bundle}"
    printf 'rollback switches compute to the reviewed previous digest without changing data stores\n'
    ;;
  execute)
    # check-aws-plan.sh protects the previous digest and exact-authorization boundary.
    aws_trial_check_bundle "${bundle}"
    aws_trial_require_exact_authorization rollback "${bundle}"
    [[ "${AVIA_AWS_PREVIOUS_IMAGE_DIGEST:-}" =~ ^sha256:[a-f0-9]{64}$ ]] ||
      aws_trial_fail "mutable-image: reviewed previous digest is required"
    [[ -f "${AVIA_AWS_ROLLBACK_PLAN:-}" ]] ||
      aws_trial_fail "missing-artifact: reviewed rollback plan is required"
    terraform apply -input=false -auto-approve "${AVIA_AWS_ROLLBACK_PLAN}"
    ;;
  *)
    aws_trial_fail "usage: expected preview or execute"
    ;;
esac
