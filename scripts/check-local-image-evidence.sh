#!/bin/sh
set -eu

repository_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
profile=${1:-}
manifest_path=${AVIASURVEIL_IMAGE_EVIDENCE_PATH:-"$repository_root/.local/aviasurveil360/image-evidence.json"}
policy_tool="$repository_root/scripts/lib/local-image-policy.mjs"

case "$profile" in
  demo)
    required_images="gateway web-demo"
    ;;
  full)
    required_images="gateway web-http auth api worker migration"
    ;;
  recovery)
    required_images="postgres-recovery"
    ;;
  *)
    echo "usage: $0 demo|full|recovery" >&2
    exit 2
    ;;
esac

if [ ! -f "$manifest_path" ]; then
  echo "local image evidence manifest is missing; run the image build, SBOM, and scan gates first" >&2
  exit 1
fi

node "$policy_tool" validate-image-evidence "$manifest_path"

for image_name in $required_images; do
  image_tag=$(jq -er --arg name "$image_name" '.images[$name].tag' "$manifest_path")
  expected_digest=$(jq -er --arg name "$image_name" '.images[$name].digest' "$manifest_path")
  current_digest=$(docker image inspect --format '{{.Id}}' "$image_tag")
  if [ "$current_digest" != "$expected_digest" ]; then
    echo "image digest drift detected for $image_name; rebuild, regenerate the SBOM, and rescan before running $profile" >&2
    exit 1
  fi
done

echo "Local $profile profile image digests match the accepted SBOM and vulnerability evidence."
