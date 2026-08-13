#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/lib/aws-trial-gates.sh"

mode="${1:-preview}"
phase="${2:-data-runtime}"
bundle="${3:-}"
aws_trial_require_phase "${phase}"

case "${mode}" in
  preview)
    aws_trial_preview smoke "${phase}" "${bundle}"
    printf 'smoke scope: HTTPS, OIDC/MFA, 86 routes, canonical mutation, fail-closed scan, provider email, native Go PDF, telemetry, alerts, backup\n'
    ;;
  execute)
    # check-aws-plan.sh must pass before the exact-authorization token is accepted.
    aws_trial_check_bundle "${bundle}"
    aws_trial_require_exact_authorization smoke "${bundle}"
    [[ "${AVIA_E2E_BASE_URL:-}" == https://* ]] ||
      aws_trial_fail "public-exposure: AVIA_E2E_BASE_URL must use HTTPS"
    AVIA_E2E_PROFILE=aws-trial \
      npm --prefix "${aws_trial_repository_root}/apps/web" exec -- \
      playwright test \
      --config="${aws_trial_repository_root}/apps/web/playwright.config.ts" \
      e2e/aws-trial-smoke.spec.ts \
      --project=aws-trial
    ;;
  *)
    aws_trial_fail "usage: expected preview or execute"
    ;;
esac
