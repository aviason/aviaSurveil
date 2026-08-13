#!/bin/sh
set -eu

script_directory=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
repository_root=$(CDPATH= cd -- "$script_directory/.." && pwd)

if [ "$#" -eq 0 ]; then
  exec "$script_directory/test-backup-profile.sh"
fi

. "$script_directory/lib/recovery-backup.sh"

usage() {
  echo "usage: $0 [--create] RECOVERY_POINT_ID [{full|diff|incr}]" >&2
  exit 64
}

create=false
if [ "${1:-}" = "--create" ]; then
  create=true
  shift
fi
[ "$#" -ge 1 ] && [ "$#" -le 2 ] || usage

recovery_point_id=$1
backup_type=${2:-incr}
validate_recovery_point_id "$recovery_point_id"
case "$backup_type" in
  full | diff | incr) ;;
  *) usage ;;
esac

if [ "$create" = true ]; then
  "$script_directory/backup-postgres.sh" \
    "$recovery_point_id" "$backup_type"
  "$script_directory/backup-identity-postgres.sh" \
    "$recovery_point_id" "$backup_type"
  "$script_directory/backup-objects.sh" "$recovery_point_id"
fi

acquire_recovery_lock "catalog-$recovery_point_id"
directory=$(prepare_component_directory "$recovery_point_id")
application_path="$directory/applicationDatabase.json"
identity_path="$directory/identityDatabase.json"
objects_path="$directory/applicationObjects.json"
configuration_path="$directory/configurationReferences.json"
catalog_directory="$recovery_catalog_directory/$recovery_point_id"
catalog_next="$catalog_directory/.catalog.json.next"
catalog_path="$catalog_directory/catalog.json"

for required_path in \
  "$application_path" \
  "$identity_path" \
  "$objects_path"; do
  if [ ! -s "$required_path" ]; then
    echo "partial recovery point refused: missing $(basename "$required_path")" >&2
    exit 1
  fi
done

jq -n \
  --arg recoveryPointId "$recovery_point_id" \
  --arg generatedAt "$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
  --arg applicationConfigSHA "$(
    shasum -a 256 "$repository_root/deploy/local/config/application.example.yaml" |
      awk '{print $1}'
  )" \
  --arg authSchemaSHA "$(
    shasum -a 256 "$repository_root/../../shared/auth/migrations/000008_local_preprod_authority_admin.up.sql" |
      awk '{print $1}'
  )" \
  --arg backupPolicySHA "$(
    shasum -a 256 "$repository_root/deploy/recovery/minio-backup-policy.json" |
      awk '{print $1}'
  )" \
  '{
    configurationReferences: {
      status: "verified locally",
      artifactStatus: "candidate-only",
      recoveryPointId: $recoveryPointId,
      generatedAt: $generatedAt,
      files: [
        {
          reference: "deploy/local/config/application.example.yaml",
          sha256: $applicationConfigSHA
        },
        {
          reference: "../../shared/auth/migrations/000008_local_preprod_authority_admin.up.sql",
          sha256: $authSchemaSHA
        },
        {
          reference: "deploy/recovery/minio-backup-policy.json",
          sha256: $backupPolicySHA
        }
      ],
      secretReferences: [
        "app_database_password",
        "preprod_auth_database_password",
        "backup_repository_cipher_passphrase",
        "preprod_oidc_client_secret",
        "session_encryption_key"
      ]
    }
  }' >"$configuration_path"
chmod 0600 "$configuration_path"

jq -s -e \
  --arg recoveryPointId "$recovery_point_id" \
  '
    .[0].applicationDatabase.status == "verified locally" and
    .[0].applicationDatabase.recoveryPointId == $recoveryPointId and
    (.[0].applicationDatabase.sha256 | test("^[a-f0-9]{64}$")) and
    .[1].identityDatabase.status == "verified locally" and
    .[1].identityDatabase.recoveryPointId == $recoveryPointId and
    (.[1].identityDatabase.sha256 | test("^[a-f0-9]{64}$")) and
    .[1].identityFingerprint.status == "verified locally" and
    .[1].identityFingerprint.recoveryPointId == $recoveryPointId and
    (.[1].identityFingerprint.sha256 | test("^[a-f0-9]{64}$")) and
    .[2].applicationObjects.status == "verified locally" and
    .[2].applicationObjects.recoveryPointId == $recoveryPointId and
    (.[2].applicationObjects.manifestSha256 | test("^[a-f0-9]{64}$")) and
    (.[2].applicationObjects.retentionUntil | fromdateiso8601) > now and
    .[3].configurationReferences.status == "verified locally" and
    .[3].configurationReferences.recoveryPointId == $recoveryPointId
  ' \
  "$application_path" \
  "$identity_path" \
  "$objects_path" \
  "$configuration_path" >/dev/null

retention_until=$(
  jq -er '.applicationObjects.retentionUntil' "$objects_path"
)
created_at=$(date -u +%Y-%m-%dT%H:%M:%SZ)
jq -s \
  --arg recoveryPointId "$recovery_point_id" \
  --arg createdAt "$created_at" \
  --arg retentionUntil "$retention_until" \
  '
    {
      schemaVersion: 1,
      status: "complete",
      artifactStatus: "candidate-only",
      failureDomain: "same-host-logically-isolated",
      productionHostLossCovered: false,
      recoveryPointId: $recoveryPointId,
      createdAt: $createdAt,
      retentionUntil: $retentionUntil
    } +
    .[0] + .[1] + .[2] + .[3]
  ' \
  "$application_path" \
  "$identity_path" \
  "$objects_path" \
  "$configuration_path" >"$catalog_next"

catalog_sha256=$(shasum -a 256 "$catalog_next" | awk '{print $1}')
catalog_with_hash="$catalog_next.hash"
jq --arg sha256 "$catalog_sha256" '. + {sha256: $sha256}' \
  "$catalog_next" >"$catalog_with_hash"
mv -f -- "$catalog_with_hash" "$catalog_next"
mv -f -- "$catalog_next" "$catalog_path"
chmod 0600 "$catalog_path"

compose_recovery exec --no-TTY \
  --env AVIA_ENVIRONMENT=local-candidate \
  --env AVIA_ENABLE_RECOVERY_BACKUP=true \
  recovery-toolbox \
  /usr/local/bin/object-backup \
  --publish-catalog \
  --recovery-point "$recovery_point_id" \
  <"$catalog_path" >"$catalog_directory/publish-result.json"
chmod 0600 "$catalog_directory/publish-result.json"

age_seconds=$(( $(date -u +%s) - $(date -u -j -f %Y-%m-%dT%H:%M:%SZ "$created_at" +%s 2>/dev/null || date -u +%s) ))
jq -cn \
  --arg name "backup.recovery_point.age" \
  --arg recoveryPointId "$recovery_point_id" \
  --argjson value "$age_seconds" \
  '{
    metric: $name,
    value: $value,
    unit: "s",
    attributes: {
      recoveryPointId: $recoveryPointId,
      outcome: "succeeded",
      artifactStatus: "candidate-only"
    }
  }' >>"$recovery_state_directory/recovery/metrics.jsonl"
chmod 0600 "$recovery_state_directory/recovery/metrics.jsonl"

record_backup_audit completed "$recovery_point_id" recoveryCatalog succeeded
echo "Recovery point $recovery_point_id catalog: verified locally"
echo "Recovery point SHA-256: $(jq -r '.sha256' "$catalog_path")"
echo "Failure domain: same-host-logically-isolated"
