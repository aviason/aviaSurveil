#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/lib/aws-trial-gates.sh"

mode="${1:-preview}"
phase="${2:-}"
decision_path="${3:-}"
bundle="${4:-}"

aws_trial_require_phase "${phase}"

case "${mode}" in
  preview)
    printf 'aws-trial plan preview: phase=%s\n' "${phase}"
    printf 'phase order: bootstrap -> foundation-ecr -> artifact-publication -> data-runtime\n'
    printf 'exact-authorization is required before execute; no AWS command was run\n'
    ;;
  execute)
    [[ -f "${decision_path}" ]] ||
      aws_trial_fail "missing-decision: decision JSON is required"
    [[ -n "${bundle}" ]] ||
      aws_trial_fail "missing-artifact: bundle directory is required"
    node_bin="$(aws_trial_node)"
    expected="$("${node_bin}" "${aws_trial_control}" plan-authorization "${phase}" "${decision_path}")"
    [[ "${AVIA_AWS_EXACT_AUTHORIZATION:-}" == "${expected}" ]] ||
      aws_trial_fail "exact-authorization: expected ${expected}"
    command -v aws >/dev/null || aws_trial_fail "aws CLI is required"
    command -v terraform >/dev/null || aws_trial_fail "terraform is required"
    command -v terragrunt >/dev/null || aws_trial_fail "terragrunt is required"
    [[ -n "${AVIA_AWS_COST_ESTIMATE_USD:-}" ]] ||
      aws_trial_fail "cost-unbounded: AVIA_AWS_COST_ESTIMATE_USD is required"
    [[ -f "${AVIA_AWS_IMAGE_EVIDENCE:-}" ]] ||
      aws_trial_fail "unscanned-image: AVIA_AWS_IMAGE_EVIDENCE is required"
    [[ -n "${AVIA_AWS_LOCK_SHA256:-}" ]] ||
      aws_trial_fail "lock-hash: AVIA_AWS_LOCK_SHA256 is required"
    [[ -f "${AVIA_TG_INPUTS_FILE:-}" ]] ||
      aws_trial_fail "missing-decision: AVIA_TG_INPUTS_FILE is required"
    grep -Eq 'fixture_mode[[:space:]]*=[[:space:]]*false' \
      "${AVIA_TG_INPUTS_FILE}" ||
      aws_trial_fail "phase-boundary: a non-fixture Terragrunt overlay is required"

    umask 077
    mkdir -p -m 0700 "${bundle}"
    export AVIA_TG_PLAN_DIR="${bundle}"
    cp "${decision_path}" "${bundle}/decision.json"
    chmod 0600 "${bundle}/decision.json"

    caller_arn="$(aws sts get-caller-identity --query Arn --output text)"
    case "${phase}" in
      bootstrap)
        units=(remote-state)
        ;;
      foundation-ecr)
        units=(identity-secrets network ecr object-storage observability security service-endpoints load-balancer)
        ;;
      artifact-publication)
        units=(artifact-publication)
        ;;
      data-runtime)
        units=(database backup compute)
        ;;
    esac

    for unit in "${units[@]}"; do
      if [[ "${unit}" == "remote-state" ]]; then
        unit_directory="${aws_trial_repository_root}/infra/terragrunt/bootstrap/remote-state"
      else
        unit_directory="${aws_trial_repository_root}/infra/terragrunt/environments/aws-trial/components/${unit}"
      fi
      if [[ "${unit}" == "remote-state" ]]; then
        plan_stem="bootstrap__remote-state"
      else
        plan_stem="environments__aws-trial__components__${unit}"
      fi
      plan_path="${bundle}/${plan_stem}.tfplan"
      json_path="${bundle}/${plan_stem}.json"
      [[ ! -e "${plan_path}" && ! -e "${json_path}" ]] ||
        aws_trial_fail "stale-plan: bundle already contains ${unit} artifacts"
      terragrunt plan \
        --working-dir "${unit_directory}"
      terragrunt show \
        --working-dir "${unit_directory}" \
        -- -json "${plan_path}" >"${json_path}"
      chmod 0600 "${plan_path}" "${json_path}"
    done

    terraform_version="$(terraform version -json | "${node_bin}" -e 'let s="";process.stdin.on("data",d=>s+=d).on("end",()=>process.stdout.write(JSON.parse(s).terraform_version))')"
    terragrunt_version="$(terragrunt --version | awk '{print $3}')"
    export AVIA_TERRAFORM_VERSION="${terraform_version}"
    export AVIA_TERRAGRUNT_VERSION="${terragrunt_version}"
    "${node_bin}" "${aws_trial_control}" create-manifest \
      "${phase}" \
      "${bundle}/decision.json" \
      "${bundle}" \
      "${aws_trial_repository_root}" \
      "${caller_arn}" \
      "${AVIA_AWS_COST_ESTIMATE_USD}" \
      "${AVIA_AWS_IMAGE_EVIDENCE}"
    export AVIA_AWS_CALLER_ARN="${caller_arn}"
    aws_trial_check_bundle "${bundle}"
    printf 'aws-trial plan: stop for protected plan review before any apply\n'
    ;;
  *)
    aws_trial_fail "usage: expected preview or execute"
    ;;
esac
