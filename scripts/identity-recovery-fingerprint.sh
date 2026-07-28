#!/bin/sh
set -eu

if [ "${AVIA_ENVIRONMENT:-}" != "local-candidate" ] ||
  [ "${AVIA_ENABLE_RECOVERY_BACKUP:-}" != "true" ]; then
  echo "local-candidate recovery authorization is required" >&2
  exit 1
fi
if [ -z "${AVIA_RECOVERY_POINT_ID:-}" ]; then
  echo "AVIA_RECOVERY_POINT_ID is required" >&2
  exit 1
fi

password=$(tr -d '\r\n' </run/secrets/keycloak_database_password)
export PGPASSWORD="$password"
unset password

identity_json=$(
  psql \
    --host keycloak-postgres \
    --username keycloak \
    --dbname keycloak \
    --tuples-only \
    --no-align \
    --set ON_ERROR_STOP=1 <<'SQL'
SELECT jsonb_build_object(
  'users', COALESCE((
    SELECT jsonb_agg(
      jsonb_build_object(
        'id', id,
        'realmId', realm_id,
        'username', username,
        'enabled', enabled,
        'emailVerified', email_verified
      )
      ORDER BY id
    )
    FROM user_entity
  ), '[]'::jsonb),
  'roles', COALESCE((
    SELECT jsonb_agg(
      jsonb_build_object(
        'id', id,
        'realmId', realm_id,
        'client', client_realm_constraint,
        'name', name
      )
      ORDER BY id
    )
    FROM keycloak_role
  ), '[]'::jsonb),
  'roleMappings', COALESCE((
    SELECT jsonb_agg(
      jsonb_build_object('userId', user_id, 'roleId', role_id)
      ORDER BY user_id, role_id
    )
    FROM user_role_mapping
  ), '[]'::jsonb),
  'credentials', COALESCE((
    SELECT jsonb_agg(
      jsonb_build_object(
        'id', id,
        'userId', user_id,
        'type', type,
        'priority', priority,
        'createdDate', created_date,
        'secretFingerprint', md5(COALESCE(secret_data, '')),
        'credentialFingerprint', md5(COALESCE(credential_data, ''))
      )
      ORDER BY user_id, type, id
    )
    FROM credential
  ), '[]'::jsonb),
  'totpEnrollmentCount', (
    SELECT COUNT(*) FROM credential WHERE lower(type) = 'otp'
  ),
  'requiredActions', COALESCE((
    SELECT jsonb_agg(
      jsonb_build_object('userId', user_id, 'action', required_action)
      ORDER BY user_id, required_action
    )
    FROM user_required_action
  ), '[]'::jsonb),
  'provisionedUsers', COALESCE((
    SELECT jsonb_agg(
      jsonb_build_object('id', id, 'username', username)
      ORDER BY id
    )
    FROM user_entity
    WHERE service_account_client_link IS NULL
  ), '[]'::jsonb)
)::text;
SQL
)
unset PGPASSWORD

canonical_json=$(printf '%s\n' "$identity_json" | jq -cS .)
sha256=$(printf '%s' "$canonical_json" | sha256sum | awk '{print $1}')

jq -cn \
  --arg recoveryPointId "$AVIA_RECOVERY_POINT_ID" \
  --arg generatedAt "$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
  --arg sha256 "$sha256" \
  --argjson state "$canonical_json" \
  '{
    identityFingerprint: {
      status: "verified locally",
      artifactStatus: "candidate-only",
      recoveryPointId: $recoveryPointId,
      generatedAt: $generatedAt,
      sha256: $sha256,
      users: ($state.users | length),
      roles: ($state.roles | length),
      roleMappings: ($state.roleMappings | length),
      credentials: ($state.credentials | length),
      totpEnrollmentCount: $state.totpEnrollmentCount,
      requiredActions: ($state.requiredActions | length),
      provisionedUsers: ($state.provisionedUsers | length),
      state: $state
    }
  }'
