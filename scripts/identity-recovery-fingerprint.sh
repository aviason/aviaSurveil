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

password=$(tr -d '\r\n' </run/secrets/preprod_auth_database_password)
export PGPASSWORD="$password"
unset password

identity_json=$(
  psql \
    --host preprod-auth-postgres \
    --username auth_preprod \
    --dbname auth_local_preprod \
    --tuples-only \
    --no-align \
    --set ON_ERROR_STOP=1 <<'SQL'
SELECT jsonb_build_object(
  'accounts', COALESCE((
    SELECT jsonb_agg(
      jsonb_build_object(
        'subjectId', subject_id,
        'state', state,
        'emailVerified', email_verified,
        'authRevision', auth_revision,
        'failedLoginCount', failed_login_count,
        'lockedUntil', locked_until
      ) ORDER BY subject_id
    ) FROM auth_identity.accounts
  ), '[]'::jsonb),
  'identifiers', COALESCE((
    SELECT jsonb_agg(
      jsonb_build_object(
        'subjectId', subject_id,
        'type', identifier_type,
        'normalizedValue', normalized_value,
        'verifiedAt', verified_at
      ) ORDER BY subject_id, identifier_type
    ) FROM auth_identity.identifiers
  ), '[]'::jsonb),
  'authorities', COALESCE((
    SELECT jsonb_agg(
      jsonb_build_object(
        'subjectId', subject_id,
        'membershipId', membership_id,
        'organizationId', organization_id,
        'role', role,
        'state', state,
        'membershipRevision', membership_revision
      ) ORDER BY subject_id
    ) FROM auth_identity.application_authorities
  ), '[]'::jsonb),
  'mfaFactors', COALESCE((
    SELECT jsonb_agg(
      jsonb_build_object(
        'subjectId', subject_id,
        'enabled', enabled,
        'lastUsedCounter', last_used_counter,
        'recoveryFailures', recovery_failures,
        'secretFingerprint', md5(secret_ciphertext::text)
      ) ORDER BY subject_id
    ) FROM auth_identity.mfa_factors
  ), '[]'::jsonb),
  'sessions', COALESCE((
    SELECT jsonb_agg(
      jsonb_build_object(
        'sessionId', session_id,
        'subjectId', subject_id,
        'state', state,
        'authRevision', auth_revision,
        'issuedAt', issued_at,
        'revokedAt', revoked_at
      ) ORDER BY session_id
    ) FROM auth_identity.provider_sessions
  ), '[]'::jsonb),
  'adminReceipts', COALESCE((
    SELECT jsonb_agg(
      jsonb_build_object(
        'operationId', operation_id,
        'operationKind', operation_kind,
        'state', state,
        'responseStatus', response_status
      ) ORDER BY operation_id
    ) FROM auth_identity.provider_admin_operation_receipts
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
      accounts: ($state.accounts | length),
      identifiers: ($state.identifiers | length),
      authorities: ($state.authorities | length),
      mfaFactors: ($state.mfaFactors | length),
      sessions: ($state.sessions | length),
      adminReceipts: ($state.adminReceipts | length),
      state: $state
    }
  }'
