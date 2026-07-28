-- name: ListRegulatoryReferenceVersions :many
SELECT id, reference_id, version, title, status, effective_date, snapshot, created_at
FROM regulatory_reference_versions
WHERE (sqlc.arg(status_filter)::text = '' OR status = sqlc.arg(status_filter)::text)
ORDER BY effective_date DESC, reference_id, version DESC
LIMIT sqlc.arg(result_limit);

-- name: CreateRegulatoryReferenceVersion :one
INSERT INTO regulatory_reference_versions (
    id, reference_id, version, title, status, effective_date, snapshot
) VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id, reference_id, version, title, status, effective_date, snapshot, created_at;

-- name: ListQuestionVersions :many
SELECT id, question_id, version, prompt, configured_reference,
       expected_evidence, created_by_subject_id, created_at
FROM question_versions
ORDER BY question_id, version DESC
LIMIT $1;

-- name: CreateQuestionVersion :one
INSERT INTO question_versions (
    id, question_id, version, prompt, configured_reference,
    expected_evidence, created_by_subject_id
) VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id, question_id, version, prompt, configured_reference,
          expected_evidence, created_by_subject_id, created_at;

-- name: ListTemplateMasters :many
SELECT id, title, owner_role, published_template_version_id, revision,
       created_at, updated_at
FROM template_masters
WHERE tombstoned_at IS NULL
ORDER BY title, id
LIMIT $1;

-- name: GetTemplateMaster :one
SELECT id, title, owner_role, published_template_version_id, revision,
       created_at, updated_at
FROM template_masters
WHERE id = $1 AND tombstoned_at IS NULL;

-- name: CreateTemplateMaster :one
INSERT INTO template_masters (
    id, title, owner_role, published_template_version_id, revision
) VALUES ($1, $2, $3, $4, 1)
RETURNING id, title, owner_role, published_template_version_id, revision,
          created_at, updated_at;

-- name: AddTemplateVersionQuestion :one
INSERT INTO template_version_questions (
    template_version_id, question_version_id, position
) VALUES ($1, $2, $3)
RETURNING template_version_id, question_version_id, position, created_at;

-- name: ListReportDefinitionVersions :many
SELECT id, definition_id, version, title, description, definition,
       created_by_subject_id, created_at
FROM report_definition_versions
ORDER BY definition_id, version DESC
LIMIT $1;

-- name: CreateReportDefinitionVersion :one
INSERT INTO report_definition_versions (
    id, definition_id, version, title, description, definition,
    created_by_subject_id
) VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id, definition_id, version, title, description, definition,
          created_by_subject_id, created_at;

-- name: ListUserLifecycleRequests :many
SELECT id, subject_id, requested_action, requested_roles,
       requested_organization_id, requested_email, requested_display_name,
       status, idempotency_key,
       requested_by_subject_id, outbox_message_id, failure_reason,
       created_at, updated_at
FROM user_lifecycle_requests
WHERE (sqlc.arg(status_filter)::text = '' OR status = sqlc.arg(status_filter)::text)
ORDER BY created_at, id
LIMIT sqlc.arg(result_limit);

-- name: GetUserLifecycleRequestByIdempotencyKey :one
SELECT id, subject_id, requested_action, requested_roles,
       requested_organization_id, requested_email, requested_display_name,
       status, idempotency_key,
       requested_by_subject_id, outbox_message_id, failure_reason,
       created_at, updated_at
FROM user_lifecycle_requests
WHERE idempotency_key = $1;

-- name: CreateUserLifecycleRequest :one
INSERT INTO user_lifecycle_requests (
    id, subject_id, requested_action, requested_roles,
    requested_organization_id, requested_email, requested_display_name,
    status, idempotency_key, requested_by_subject_id, outbox_message_id
) VALUES ($1, $2, $3, $4, $5, $6, $7, 'PENDING', $8, $9, $10)
RETURNING id, subject_id, requested_action, requested_roles,
          requested_organization_id, requested_email, requested_display_name,
          status, idempotency_key,
          requested_by_subject_id, outbox_message_id, failure_reason,
          created_at, updated_at;

-- name: UpdateUserLifecycleRequest :one
UPDATE user_lifecycle_requests
SET status = $2,
    failure_reason = $3,
    updated_at = $4
WHERE id = $1
RETURNING id, subject_id, requested_action, requested_roles,
          requested_organization_id, requested_email, requested_display_name,
          status, idempotency_key,
          requested_by_subject_id, outbox_message_id, failure_reason,
          created_at, updated_at;

-- name: ListAccessDirectoryLocalStates :many
WITH latest_membership AS (
    SELECT DISTINCT ON (subject_id)
        membership_id, subject_id, revision, membership_state,
        organization_id, roles, drift_state
    FROM desired_membership_versions
    ORDER BY subject_id, revision DESC
),
latest_lifecycle AS (
    SELECT DISTINCT ON (subject_id)
        id, subject_id, requested_action, status
    FROM user_lifecycle_requests
    WHERE subject_id IS NOT NULL
    ORDER BY subject_id, updated_at DESC, id DESC
),
last_session AS (
    SELECT subject_id, MAX(last_seen_at) AS last_seen_at
    FROM session_references
    GROUP BY subject_id
)
SELECT identity.subject_id,
       (profile.subject_id IS NOT NULL)::boolean AS profile_linked,
       COALESCE(membership.membership_id, '') AS membership_id,
       COALESCE(membership.revision, 0)::bigint AS membership_revision,
       COALESCE(membership.membership_state, 'ABSENT') AS membership_state,
       COALESCE(membership.organization_id, '') AS organization_id,
       COALESCE(membership.roles, ARRAY[]::text[])::text[] AS roles,
       COALESCE(membership.drift_state, 'UNTRACKED') AS stored_drift,
       COALESCE(lifecycle.requested_action, '') AS lifecycle_action,
       COALESCE(lifecycle.status, '') AS lifecycle_status,
       COALESCE(invitation_delivery.status, '') AS invitation_delivery_status,
       last_session.last_seen_at::timestamptz AS last_successful_session
FROM identity_references identity
LEFT JOIN user_profiles profile
  ON profile.subject_id = identity.subject_id
LEFT JOIN latest_membership membership
  ON membership.subject_id = identity.subject_id
LEFT JOIN latest_lifecycle lifecycle
  ON lifecycle.subject_id = identity.subject_id
LEFT JOIN LATERAL (
    SELECT job.status
    FROM notification_records record
    JOIN notification_delivery_jobs job
      ON job.notification_id = record.id
     AND job.recipient_subject_id = identity.subject_id
     AND job.channel = 'EMAIL'
    WHERE record.recipient_subject_id = identity.subject_id
      AND record.related_entity_type = 'USER_LIFECYCLE_INVITATION'
      AND record.related_entity_id = lifecycle.id
      AND record.tombstoned_at IS NULL
    ORDER BY job.created_at DESC, job.id DESC
    LIMIT 1
) invitation_delivery ON TRUE
LEFT JOIN last_session
  ON last_session.subject_id = identity.subject_id
WHERE identity.subject_id = ANY(sqlc.arg(subject_ids)::text[])
  AND identity.tombstoned_at IS NULL
ORDER BY identity.subject_id;
