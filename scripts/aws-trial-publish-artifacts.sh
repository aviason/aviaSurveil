#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/lib/aws-trial-gates.sh"

mode="${1:-preview}"
phase="${2:-artifact-publication}"
bundle="${3:-}"
aws_trial_require_phase "${phase}"
[[ "${phase}" == "artifact-publication" ]] ||
  aws_trial_fail "phase-boundary: publishing is artifact-publication only"

case "${mode}" in
  preview)
    # The preview prints the exact-authorization value without publishing.
    aws_trial_preview publish-artifacts "${phase}" "${bundle}"
    printf 'preview checks immutable sha256 digest, CycloneDX SBOM, and zero HIGH/CRITICAL scan\n'
    ;;
  execute)
    # check-aws-plan.sh binds every image digest and SBOM hash before publication.
    aws_trial_check_bundle "${bundle}"
    aws_trial_require_exact_authorization publish-artifacts "${bundle}"
    [[ -f "${AVIA_AWS_PUBLICATION_MANIFEST:-}" ]] ||
      aws_trial_fail "missing-sbom: publication manifest is required"
    command -v aws >/dev/null || aws_trial_fail "aws CLI is required"
    account_id="$(aws_trial_manifest_field "${bundle}" accountId)"
    region="$(aws_trial_manifest_field "${bundle}" region)"
    registry="${account_id}.dkr.ecr.${region}.amazonaws.com"
    while IFS='|' read -r repository digest archive sbom sbom_sha source_image; do
      [[ "${repository}" == "${registry}/"* ]] ||
        aws_trial_fail "image-scope: repository is outside reviewed account/region"
      [[ "${digest}" =~ ^sha256:[a-f0-9]{64}$ ]] ||
        aws_trial_fail "mutable-image: invalid digest for ${repository}"
      [[ "${sbom_sha}" =~ ^[a-f0-9]{64}$ ]] ||
        aws_trial_fail "missing-sbom: invalid SBOM hash"
      [[ -f "${archive}" && -f "${sbom}" ]] ||
        aws_trial_fail "missing-sbom: archive or SBOM is absent"
      actual_sbom_sha="$(shasum -a 256 "${sbom}" | awk '{print $1}')"
      [[ "${actual_sbom_sha}" == "${sbom_sha}" ]] ||
        aws_trial_fail "missing-sbom: SBOM hash changed"
      [[ -n "${source_image}" ]] ||
        aws_trial_fail "mutable-image: loaded source image is required"
      aws ecr get-login-password --region "${region}" |
        docker login --username AWS --password-stdin "${registry}"
      docker load --input "${archive}"
      publication_tag="verified-${digest#sha256:}"
      docker tag "${source_image}" "${repository}:${publication_tag}"
      docker push "${repository}:${publication_tag}" >/dev/null
      published_digest="$(aws ecr describe-images \
        --region "${region}" \
        --repository-name "${repository#*/}" \
        --image-ids "imageTag=${publication_tag}" \
        --query 'imageDetails[0].imageDigest' \
        --output text)"
      [[ "${published_digest}" == "${digest}" ]] ||
        aws_trial_fail "mutable-image: published digest does not match review"
    done <"${AVIA_AWS_PUBLICATION_MANIFEST}"
    ;;
  *)
    aws_trial_fail "usage: expected preview or execute"
    ;;
esac
