#!/usr/bin/env bash
set -euo pipefail

repository_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
mode="${1:-preview}"
bundle="${2:-}"

fail() {
  printf 'check-aws-plan: %s\n' "$*" >&2
  exit 64
}

case "${mode}" in
  preview)
    printf 'check-aws-plan preview: validates a protected bundle without contacting AWS\n'
    exit 0
    ;;
  offline)
    ;;
  *)
    fail "usage: expected preview or offline"
    ;;
esac

[[ -n "${bundle}" ]] || fail "missing-artifact: bundle path is required"
node_bin="${NODE_BIN:-$(command -v node || true)}"
opa_bin="${OPA_BIN:-$(command -v opa || true)}"
[[ -n "${node_bin}" ]] || fail "tool-version: node is required"
[[ -n "${opa_bin}" ]] || fail "policy-tool: opa is required"
[[ -n "${AVIA_AWS_CALLER_ARN:-}" ]] ||
  fail "caller-mismatch: AVIA_AWS_CALLER_ARN is required"

"${node_bin}" \
  "${repository_root}/scripts/lib/aws-trial-control.mjs" \
  validate \
  "${bundle}" \
  "${repository_root}" \
  "${opa_bin}" \
  "${repository_root}/infra/policies/aws-plan.rego" \
  "${AVIA_AWS_CALLER_ARN}"
