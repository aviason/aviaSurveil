-- name: GetOrganization :one
SELECT id, legal_name, organization_type, status, revision, created_at, updated_at
FROM organizations
WHERE id = $1
  AND tombstoned_at IS NULL;

-- name: ListOrganizations :many
SELECT id, legal_name, organization_type, status, revision, created_at, updated_at
FROM organizations
WHERE tombstoned_at IS NULL
ORDER BY legal_name, id
LIMIT $1 OFFSET $2;

-- name: ListOrganizationRegistry :many
SELECT organization.id, organization.legal_name, organization.organization_type,
       organization.status, organization.revision,
       COUNT(DISTINCT finding.id) FILTER (WHERE finding.status <> 'CLOSED')::bigint AS open_finding_count,
       COALESCE(to_char(MAX(inspection.due_date) FILTER (WHERE inspection.status IN ('COMPLETED', 'CLOSED')), 'YYYY-MM-DD'), '')::text AS last_audit_date,
       COALESCE(
           to_char(MIN(plan_item.scheduled_date), 'YYYY-MM-DD'),
           to_char(MIN(inspection.due_date) FILTER (WHERE inspection.status NOT IN ('COMPLETED', 'CANCELLED')), 'YYYY-MM-DD'),
           ''
       )::text AS next_audit_date
FROM organizations organization
LEFT JOIN findings finding ON finding.organization_id = organization.id
LEFT JOIN inspections inspection ON inspection.organization_id = organization.id
LEFT JOIN surveillance_plan_items plan_item ON plan_item.organization_id = organization.id
WHERE organization.organization_type <> 'AUTHORITY'
  AND organization.id <> 'ORG-SYNTHETIC-AOC'
  AND organization.tombstoned_at IS NULL
  AND (sqlc.arg(organization_scope)::text = '' OR organization.id = sqlc.arg(organization_scope))
GROUP BY organization.id
ORDER BY organization.legal_name, organization.id
LIMIT sqlc.arg(result_limit);
