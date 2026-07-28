#!/bin/sh
set -eu

repository_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
state_directory=${AVIASURVEIL_LOCAL_STATE_DIR:-"$repository_root/.local/aviasurveil360"}
manifest_path="$state_directory/image-evidence.json"
image_lock="$repository_root/deploy/local/image-lock.json"

mkdir -p "$state_directory"
chmod 0700 "$state_directory"

node_build_image=$(jq -er '.images["node-build"].reference' "$image_lock")
go_build_image=$(jq -er '.images["go-build"].reference' "$image_lock")
go_runtime_image=$(jq -er '.images["go-runtime"].reference' "$image_lock")
keycloak_image=$(jq -er '.images.keycloak.reference' "$image_lock")
postgres_image=$(jq -er '.images.postgres.reference' "$image_lock")
caddy_version=$(jq -er '.components.caddy.version' "$image_lock")
grpc_version=$(jq -er '.components.grpc.version' "$image_lock")
source_revision=$(git -C "$repository_root" rev-parse HEAD)
source_date_epoch=$(git -C "$repository_root" show -s --format=%ct HEAD)
source_dirty=false
if [ -n "$(git -C "$repository_root" status --porcelain --untracked-files=all)" ]; then
  source_dirty=true
fi
source_state_sha256=$(
  (
    git -C "$repository_root" diff --binary HEAD
    git -C "$repository_root" ls-files --others --exclude-standard |
      LC_ALL=C sort |
      while IFS= read -r untracked_path; do
        shasum -a 256 "$repository_root/$untracked_path"
      done
  ) |
    shasum -a 256 |
    awk '{print $1}'
)

build_image() {
  image_name=$1
  image_tag=$2
  dockerfile=$3
  target=$4
  shift 4
  docker build \
    --file "$repository_root/$dockerfile" \
    --target "$target" \
    --tag "$image_tag" \
    --build-arg "SOURCE_DATE_EPOCH=$source_date_epoch" \
    --label "org.opencontainers.image.revision=$source_revision" \
    --label "org.opencontainers.image.source=local://aviasurveil360" \
    --label "io.aviasurveil360.source-dirty=$source_dirty" \
    --label "io.aviasurveil360.source-state-sha256=$source_state_sha256" \
    "$@" \
    "$repository_root" >&2
  image_digest=$(docker image inspect --format '{{.Id}}' "$image_tag")
  case "$image_digest" in
    sha256:????????????????????????????????????????????????????????????????) ;;
    *)
      echo "built image $image_name has no sha256 content digest" >&2
      exit 1
      ;;
  esac
  printf '%s|%s|%s\n' "$image_name" "$image_tag" "$image_digest"
}

records_file=$(mktemp "$state_directory/.image-records.XXXXXX")
manifest_next=$(mktemp "$state_directory/.image-evidence.XXXXXX")
cleanup() {
  rm -f -- "$records_file" "$manifest_next"
}
trap cleanup EXIT HUP INT TERM

build_image gateway aviasurveil360/gateway:local deploy/local/gateway/Dockerfile gateway \
  --build-arg "GO_BUILD_IMAGE=$go_build_image" \
  --build-arg "CADDY_VERSION=$caddy_version" \
  --build-arg "GRPC_VERSION=$grpc_version" >>"$records_file"
build_image web-demo aviasurveil360/web-demo:local apps/web/Dockerfile demo \
  --build-arg "NODE_BUILD_IMAGE=$node_build_image" \
  --build-arg "GO_BUILD_IMAGE=$go_build_image" >>"$records_file"
build_image web-http aviasurveil360/web-http:local apps/web/Dockerfile http \
  --build-arg "NODE_BUILD_IMAGE=$node_build_image" \
  --build-arg "GO_BUILD_IMAGE=$go_build_image" >>"$records_file"
build_image keycloak aviasurveil360/keycloak:local deploy/local/keycloak/Dockerfile keycloak \
  --build-arg "KEYCLOAK_IMAGE=$keycloak_image" \
  --build-arg "NODE_BUILD_IMAGE=$node_build_image" >>"$records_file"
build_image postgres-recovery aviasurveil360/postgres-recovery:local deploy/recovery/Dockerfile postgres-recovery \
  --build-arg "GO_BUILD_IMAGE=$go_build_image" \
  --build-arg "POSTGRES_IMAGE=$postgres_image" >>"$records_file"

for specification in \
  "api|aviasurveil360/api:local|api" \
  "worker|aviasurveil360/worker:local|worker" \
  "scheduler|aviasurveil360/scheduler:local|scheduler" \
  "migration|aviasurveil360/migration:local|migration"
do
  image_name=${specification%%|*}
  remainder=${specification#*|}
  image_tag=${remainder%%|*}
  target=${remainder##*|}
  build_image "$image_name" "$image_tag" apps/api/Dockerfile "$target" \
    --build-arg "GO_BUILD_IMAGE=$go_build_image" \
    --build-arg "GO_RUNTIME_IMAGE=$go_runtime_image" >>"$records_file"
done

jq -n \
  --arg revision "$source_revision" \
  --argjson sourceDateEpoch "$source_date_epoch" \
  --argjson sourceDirty "$source_dirty" \
  --arg sourceStateSha256 "$source_state_sha256" \
  '{
    schemaVersion: 1,
    sourceRevision: $revision,
    sourceDateEpoch: $sourceDateEpoch,
    sourceDirty: $sourceDirty,
    sourceStateSha256: $sourceStateSha256,
    images: {},
    sboms: {},
    scans: {}
  }' >"$manifest_next"

while IFS='|' read -r image_name image_tag image_digest; do
  updated_manifest="$manifest_next.updated"
  jq \
    --arg name "$image_name" \
    --arg tag "$image_tag" \
    --arg digest "$image_digest" \
    '.images[$name] = {tag: $tag, digest: $digest}' \
    "$manifest_next" >"$updated_manifest"
  mv -f -- "$updated_manifest" "$manifest_next"
done <"$records_file"

mv -f -- "$manifest_next" "$manifest_path"
chmod 0600 "$manifest_path"
echo "Built 9 local runtime images and recorded digest-bound evidence metadata."
