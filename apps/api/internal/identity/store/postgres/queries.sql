-- name: GetSessionForAuthentication :one
SELECT id, subject_id, organization_id, roles, expires_at, absolute_expires_at, revoked_at,
       csrf_token_hash, provider_session_id, membership_id, membership_revision,
       authority_observed_at, authority_state
FROM session_references
WHERE session_token_hash = $1
FOR UPDATE;

-- name: GetIdentityReference :one
SELECT subject_id, issuer, display_name, created_at
FROM identity_references
WHERE subject_id = $1
  AND tombstoned_at IS NULL;

-- name: GetProfile :one
SELECT subject_id, display_name, organization_id, revision, created_at, updated_at
FROM user_profiles
WHERE subject_id = $1
  AND tombstoned_at IS NULL;

-- name: UpdateProfile :one
UPDATE user_profiles
SET display_name = $2,
    revision = revision + 1,
    updated_at = $3
WHERE subject_id = $1
  AND revision = $4
  AND tombstoned_at IS NULL
RETURNING subject_id, display_name, organization_id, revision, created_at, updated_at;

-- name: GetSettings :one
SELECT subject_id, notification_preferences, locale, timezone, revision, updated_at
FROM user_settings
WHERE subject_id = $1;

-- name: UpdateSettings :one
UPDATE user_settings
SET notification_preferences = $2,
    locale = $3,
    timezone = $4,
    revision = revision + 1,
    updated_at = $5
WHERE subject_id = $1
  AND revision = $6
RETURNING subject_id, notification_preferences, locale, timezone, revision, updated_at;
