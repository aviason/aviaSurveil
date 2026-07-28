#!/usr/bin/env bash

aws_trial_repository_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
aws_trial_control="${aws_trial_repository_root}/scripts/lib/aws-trial-control.mjs"
aws_trial_checker="${aws_trial_repository_root}/scripts/check-aws-plan.sh"

aws_trial_fail() {
  printf 'aws-trial: %s\n' "$*" >&2
  exit 64
}

aws_trial_node() {
  if [[ -n "${NODE_BIN:-}" ]]; then
    printf '%s\n' "${NODE_BIN}"
    return
  fi
  command -v node || aws_trial_fail "node is required"
}

aws_trial_require_phase() {
  case "${1:-}" in
    bootstrap|foundation-ecr|artifact-publication|data-runtime)
      ;;
    *)
      aws_trial_fail "phase-boundary: expected bootstrap, foundation-ecr, artifact-publication, or data-runtime"
      ;;
  esac
}

aws_trial_check_bundle() {
  command -v aws >/dev/null || aws_trial_fail "aws CLI is required"
  AVIA_AWS_CALLER_ARN="$(aws sts get-caller-identity --query Arn --output text)"
  export AVIA_AWS_CALLER_ARN
  "${aws_trial_checker}" offline "$1"
}

aws_trial_manifest_field() {
  local node_bin
  node_bin="$(aws_trial_node)"
  "${node_bin}" "${aws_trial_control}" field "$1" "$2"
}

aws_trial_expected_authorization() {
  local node_bin
  node_bin="$(aws_trial_node)"
  "${node_bin}" "${aws_trial_control}" authorization "$1" "$2"
}

aws_trial_require_exact_authorization() {
  local action="$1"
  local bundle="$2"
  local expected
  expected="$(aws_trial_expected_authorization "${action}" "${bundle}")"
  [[ "${AVIA_AWS_EXACT_AUTHORIZATION:-}" == "${expected}" ]] ||
    aws_trial_fail "exact-authorization: expected ${expected}"
}

aws_trial_preview() {
  local action="$1"
  local phase="$2"
  local bundle="${3:-}"
  printf 'aws-trial preview: action=%s phase=%s\n' "${action}" "${phase}"
  if [[ -n "${bundle}" && -f "${bundle}/manifest.json" ]]; then
    printf 'required authorization: %s\n' \
      "$(aws_trial_expected_authorization "${action}" "${bundle}")"
  else
    printf 'required authorization: protected bundle must be supplied before execute\n'
  fi
}
