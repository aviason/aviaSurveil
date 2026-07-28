#!/bin/sh
set -eu

repository_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
state_directory=${AVIASURVEIL_LOCAL_STATE_DIR:-"$repository_root/.local/aviasurveil360"}
manifest_path="$state_directory/image-evidence.json"
sbom_directory="$state_directory/sboms"
image_lock="$repository_root/deploy/local/image-lock.json"

if [ ! -f "$manifest_path" ]; then
  echo "local image evidence manifest is missing; run build-local-images.sh first" >&2
  exit 1
fi

mkdir -p "$sbom_directory"
chmod 0700 "$sbom_directory"
syft_tool_image=$(jq -er '.images["syft-tool"].reference' "$image_lock")
docker_host=${DOCKER_HOST:-$(docker context inspect --format '{{.Endpoints.docker.Host}}')}
case "$docker_host" in
  unix://*) docker_socket=${docker_host#unix://} ;;
  *)
    echo "containerized Syft requires a local unix Docker socket" >&2
    exit 1
    ;;
esac
if [ ! -S "$docker_socket" ]; then
  echo "Docker socket is unavailable for containerized Syft" >&2
  exit 1
fi
case "$(uname -s)" in
  Darwin) docker_socket_group=0 ;;
  Linux) docker_socket_group=$(stat -c '%g' "$docker_socket") ;;
  *)
    echo "containerized Syft supports local Docker sockets on macOS and Linux" >&2
    exit 1
    ;;
esac
manifest_next=$(mktemp "$state_directory/.image-evidence-sbom.XXXXXX")
cp "$manifest_path" "$manifest_next"
cleanup() {
  rm -f -- "$manifest_next" "$manifest_next.updated"
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
  output_path="$sbom_directory/$image_name.cdx.json"
  docker run --rm \
    --read-only \
    --cap-drop ALL \
    --security-opt no-new-privileges \
    --network none \
    --user "$(id -u):$(id -g)" \
    --group-add "$docker_socket_group" \
    --env DOCKER_HOST=unix:///var/run/docker.sock \
    --env SYFT_CHECK_FOR_APP_UPDATE=false \
    --env SYFT_CACHE_DIR=/tmp/syft-cache \
    --volume "$docker_socket:/var/run/docker.sock:ro" \
    --volume "$sbom_directory:/output" \
    --tmpfs /tmp:rw,noexec,nosuid,nodev,mode=1777,size=1g \
    "$syft_tool_image" \
    "docker:$image_tag" \
    --output "cyclonedx-json=/output/$image_name.cdx.json"
  output_hash=$(shasum -a 256 "$output_path" | awk '{print $1}')
  jq \
    --arg name "$image_name" \
    --arg digest "$image_digest" \
    --arg path "$output_path" \
    --arg sha256 "$output_hash" \
    '.sboms[$name] = {
      digest: $digest,
      format: "cyclonedx-json",
      path: $path,
      sha256: $sha256
    }' \
    "$manifest_next" >"$manifest_next.updated"
  mv -f -- "$manifest_next.updated" "$manifest_next"
done

mv -f -- "$manifest_next" "$manifest_path"
chmod 0600 "$manifest_path"
echo "Generated 9 digest-bound CycloneDX SBOMs."
