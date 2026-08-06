#!/usr/bin/env bash
set -euo pipefail

repository_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
state_directory=${AVIASURVEIL_LOCAL_STATE_DIR:-"$repository_root/.local/aviasurveil360/aws-ipv6-trial"}
manifest_path="$state_directory/image-evidence.json"
platform=""
cloudflared_image=${AVIA_TRIAL_CLOUDFLARED_IMAGE:-}

usage() {
  cat >&2 <<'EOF'
usage: build-aws-ipv6-trial-images.sh --platform linux/arm64 --cloudflared-image repository@sha256:<digest>

The cloudflared image must already exist in the local Docker cache. This
script never pulls, publishes, or mutates an AWS/Cloudflare resource.
EOF
}

while (($#)); do
  case "$1" in
    --platform)
      [[ $# -ge 2 ]] || { usage; exit 64; }
      platform=$2
      shift 2
      ;;
    --cloudflared-image)
      [[ $# -ge 2 ]] || { usage; exit 64; }
      cloudflared_image=$2
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "unknown argument: $1" >&2
      usage
      exit 64
      ;;
  esac
done

[[ "$platform" == "linux/arm64" ]] || { echo "platform-contract: only linux/arm64 is accepted" >&2; exit 64; }
case "$(uname -m)" in
  arm64|aarch64) ;;
  *) echo "host-architecture: ARM64 build host required; use a native ARM64 builder, not emulation" >&2; exit 64 ;;
esac
[[ "${DOCKER_DEFAULT_PLATFORM:-}" != "linux/amd64" ]] || { echo "emulation-contract: DOCKER_DEFAULT_PLATFORM=linux/amd64 is forbidden" >&2; exit 64; }
command -v docker >/dev/null 2>&1 || { echo "not-run: docker is unavailable" >&2; exit 78; }
command -v jq >/dev/null 2>&1 || { echo "not-run: jq is unavailable" >&2; exit 78; }
command -v trivy >/dev/null 2>&1 || { echo "not-run: trivy is required for the HIGH/CRITICAL image gate" >&2; exit 78; }

[[ "$cloudflared_image" =~ ^[^:[:space:]]+@sha256:[0-9a-f]{64}$ ]] || {
  echo "image-contract: cloudflared must be an immutable @sha256 subject" >&2
  exit 64
}

mkdir -p "$state_directory"
chmod 0700 "$state_directory"
image_lock="$repository_root/deploy/local/image-lock.json"
node_build_image=$(jq -er '.images["node-build"].reference' "$image_lock")
go_build_image=$(jq -er '.images["go-build"].reference' "$image_lock")
syft_tool_image=$(jq -er '.images["syft-tool"].reference' "$image_lock")
source_revision=$(git -C "$repository_root" rev-parse HEAD)
source_state_sha256=$(
  (
    git -C "$repository_root" diff --binary HEAD
    git -C "$repository_root" ls-files --others --exclude-standard |
      LC_ALL=C sort |
      while IFS= read -r untracked_path; do
        shasum -a 256 "$repository_root/$untracked_path"
      done
  ) | shasum -a 256 | awk '{print $1}'
)
metadata_files=()

build_image() {
  local name=$1 tag=$2 dockerfile=$3 target=$4 digest architecture metadata_file
  metadata_file=$(mktemp "$state_directory/.build-metadata.XXXXXX")
  metadata_files+=("$metadata_file")
  docker buildx build \
    --platform "$platform" \
    --provenance=mode=max \
    --sbom="generator=$syft_tool_image" \
    --metadata-file "$metadata_file" \
    --load \
    --file "$repository_root/$dockerfile" \
    --target "$target" \
    --tag "$tag" \
    --build-arg "GO_BUILD_IMAGE=$go_build_image" \
    --build-arg "NODE_BUILD_IMAGE=$node_build_image" \
    --label "org.opencontainers.image.revision=$source_revision" \
    --label "io.aviasurveil360.source-state-sha256=$source_state_sha256" \
    "$repository_root" >&2
  architecture=$(docker image inspect --format '{{.Architecture}}' "$tag")
  [[ "$architecture" == "arm64" ]] || { echo "image-contract: $name architecture is $architecture" >&2; exit 65; }
  digest=$(jq -er '."containerimage.digest" | select(test("^sha256:[0-9a-f]{64}$"))' "$metadata_file")
  [[ "$digest" =~ ^sha256:[0-9a-f]{64}$ ]] || { echo "image-contract: $name has no immutable content digest" >&2; exit 65; }
  rm -f -- "$metadata_file"
  trivy image --quiet --severity HIGH,CRITICAL --exit-code 1 "$tag" >&2
  printf '%s\t%s\t%s\t%s\n' "$name" "$tag" "$digest" "$architecture"
}

cloudflared_architecture=$(docker image inspect --format '{{.Architecture}}' "$cloudflared_image" 2>/dev/null || true)
[[ "$cloudflared_architecture" == "arm64" ]] || { echo "image-contract: cloudflared must already be cached as arm64" >&2; exit 65; }
trivy image --quiet --severity HIGH,CRITICAL --exit-code 1 "$cloudflared_image" >&2

records_file=$(mktemp "$state_directory/.image-records.XXXXXX")
manifest_next=$(mktemp "$state_directory/.image-evidence.XXXXXX")
cleanup() { rm -f -- "$records_file" "$manifest_next" "${metadata_files[@]}"; }
trap cleanup EXIT HUP INT TERM

build_image gateway "aviasurveil360/aws-ipv6-trial-gateway:build" deploy/aws-ipv6-trial/gateway/Dockerfile gateway >>"$records_file"
build_image web-demo "aviasurveil360/aws-ipv6-trial-web-demo:build" apps/web/Dockerfile demo >>"$records_file"

jq -n \
  --arg platform "$platform" \
  --arg revision "$source_revision" \
  --arg sourceStateSha256 "$source_state_sha256" \
  --arg cloudflared "$cloudflared_image" \
  '{schemaVersion: 1, profile: "aws-ipv6-trial", platform: $platform, sourceRevision: $revision, sourceStateSha256: $sourceStateSha256, images: {cloudflared: {reference: $cloudflared, platform: "linux/arm64", sbom: "trivy-image-and-buildx-attestation", vulnerabilityScan: "HIGH_CRITICAL_PASS"}}, prohibited: {amd64: false, emulation: false, mutableTags: false}}' >"$manifest_next"

while IFS=$'\t' read -r name tag digest architecture; do
  updated="$manifest_next.updated"
  jq --arg name "$name" --arg tag "$tag" --arg digest "$digest" --arg architecture "$architecture" \
    '.images[$name] = {reference: ($tag + "@" + $digest), platform: $architecture, sbom: "buildx-attestation", vulnerabilityScan: "HIGH_CRITICAL_PASS"}' \
    "$manifest_next" >"$updated"
  mv -f -- "$updated" "$manifest_next"
done <"$records_file"

mv -f -- "$manifest_next" "$manifest_path"
chmod 0600 "$manifest_path"
echo "verified locally: ARM64 trial image evidence written to $manifest_path"
