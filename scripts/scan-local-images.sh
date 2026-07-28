#!/bin/sh
set -eu

repository_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
state_directory=${AVIASURVEIL_LOCAL_STATE_DIR:-"$repository_root/.local/aviasurveil360"}
manifest_path="$state_directory/image-evidence.json"
scan_directory="$state_directory/scans"
policy_path="$repository_root/deploy/local/vulnerability-policy.json"
policy_tool="$repository_root/scripts/lib/local-image-policy.mjs"
image_lock="$repository_root/deploy/local/image-lock.json"

if [ ! -f "$manifest_path" ]; then
  echo "local image evidence manifest is missing; run build-local-images.sh first" >&2
  exit 1
fi

node "$policy_tool" validate-vulnerability-policy "$policy_path"
mkdir -p "$scan_directory"
chmod 0700 "$scan_directory"
trivy_cache="$state_directory/trivy-cache"
mkdir -p "$trivy_cache"
chmod 0700 "$trivy_cache"
trivy_tool_image=$(jq -er '.images["trivy-tool"].reference' "$image_lock")
docker_host=${DOCKER_HOST:-$(docker context inspect --format '{{.Endpoints.docker.Host}}')}
case "$docker_host" in
  unix://*) docker_socket=${docker_host#unix://} ;;
  *)
    echo "containerized Trivy requires a local unix Docker socket" >&2
    exit 1
    ;;
esac
if [ ! -S "$docker_socket" ]; then
  echo "Docker socket is unavailable for containerized Trivy" >&2
  exit 1
fi
case "$(uname -s)" in
  Darwin) docker_socket_group=0 ;;
  Linux) docker_socket_group=$(stat -c '%g' "$docker_socket") ;;
  *)
    echo "containerized Trivy supports local Docker sockets on macOS and Linux" >&2
    exit 1
    ;;
esac
trivy_temp=$(mktemp -d "$state_directory/.trivy-tmp.XXXXXX")
chmod 0700 "$trivy_temp"
manifest_next=$(mktemp "$state_directory/.image-evidence-scan.XXXXXX")
cp "$manifest_path" "$manifest_next"
cleanup() {
  rm -f -- "$manifest_next" "$manifest_next.updated" "$scan_directory/.trivyignore"
  rm -rf -- "$trivy_temp"
}
trap cleanup EXIT HUP INT TERM

jq -r '.images | to_entries[] | [.key, .value.tag, .value.digest] | @tsv' \
  "$manifest_path" |
while IFS='	' read -r image_name image_tag image_digest; do
  current_digest=$(docker image inspect --format '{{.Id}}' "$image_tag")
  if [ "$current_digest" != "$image_digest" ]; then
    echo "image digest drift detected for $image_name" >&2
    exit 1
  fi
  locked_reference=$(
    jq -r --arg name "$image_name" \
      '.images[$name].reference // empty' \
      "$image_lock"
  )
  source_digest=
  case "$locked_reference" in
    *@sha256:????????????????????????????????????????????????????????????????)
      source_digest=${locked_reference##*@}
      ;;
  esac
  if [ -n "$source_digest" ]; then
    node "$policy_tool" exception-ids \
      "$policy_path" \
      "$image_name" \
      "$image_digest" \
      "$source_digest" \
      >"$scan_directory/.trivyignore"
  else
    node "$policy_tool" exception-ids \
      "$policy_path" \
      "$image_name" \
      "$image_digest" \
      >"$scan_directory/.trivyignore"
  fi
  output_path="$scan_directory/$image_name.json"
  docker run --rm \
    --read-only \
    --cap-drop ALL \
    --security-opt no-new-privileges \
    --user "$(id -u):$(id -g)" \
    --group-add "$docker_socket_group" \
    --env DOCKER_HOST=unix:///var/run/docker.sock \
    --env TMPDIR=/trivy-tmp \
    --volume "$docker_socket:/var/run/docker.sock:ro" \
    --volume "$scan_directory:/output" \
    --volume "$trivy_cache:/cache" \
    --volume "$trivy_temp:/trivy-tmp" \
    --tmpfs /tmp:rw,noexec,nosuid,nodev,mode=1777,size=2g \
    "$trivy_tool_image" \
    --cache-dir /cache \
    image \
    --scanners vuln \
    --severity HIGH,CRITICAL \
    --exit-code 1 \
    --ignorefile /output/.trivyignore \
    --format json \
    --output "/output/$image_name.json" \
    "$image_tag"
  output_hash=$(shasum -a 256 "$output_path" | awk '{print $1}')
  jq \
    --arg name "$image_name" \
    --arg digest "$image_digest" \
    --arg path "$output_path" \
    --arg sha256 "$output_hash" \
    '.scans[$name] = {
      digest: $digest,
      status: "passed",
      severities: ["HIGH", "CRITICAL"],
      path: $path,
      sha256: $sha256
    }' \
    "$manifest_next" >"$manifest_next.updated"
  mv -f -- "$manifest_next.updated" "$manifest_next"
done

node "$policy_tool" validate-image-evidence "$manifest_next"
mv -f -- "$manifest_next" "$manifest_path"
chmod 0600 "$manifest_path"
echo "All 9 local image digests passed the HIGH/CRITICAL vulnerability gate."
