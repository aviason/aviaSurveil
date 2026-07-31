package organizations

import (
	"context"
	"strings"
	"time"

	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/platform/database"
)

// ServiceProviderScope is the non-sensitive applicability record available to
// CAA-side selection services. It deliberately contains no department
// membership or review data and is never projected through Auditee workspaces.
type ServiceProviderScope struct {
	ID                    string
	ServiceProviderTypeID string
	AuthorizationID       string
	CertificateID         string
	PrimaryTargetID       string
}

// ListApplicableServiceProviderScopes is the single Task 2 selection seam for
// active organization scopes. Later checklist selection adds inspection-type,
// department, target, and qualifier matching without reinterpreting expiry.
func ListApplicableServiceProviderScopes(ctx context.Context, pool *database.Pool, organizationID string, at time.Time) ([]ServiceProviderScope, error) {
	rows, err := pool.Query(ctx, `
		SELECT id, service_provider_type_id, authorization_identifier,
		       COALESCE(certificate_identifier, ''), COALESCE(primary_target_id, '')
		FROM (
			SELECT DISTINCT ON (root_id) * FROM organization_service_provider_scopes
			WHERE organization_id = $1 AND effective_from <= $2::date
			ORDER BY root_id, effective_from DESC, id DESC
		) scope
		WHERE organization_id = $1
		  AND status = 'ACTIVE'
		  AND (effective_to IS NULL OR effective_to > $2::date)
		ORDER BY service_provider_type_id, authorization_identifier, effective_from, id
	`, strings.TrimSpace(organizationID), at.UTC())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	scopes := []ServiceProviderScope{}
	for rows.Next() {
		var scope ServiceProviderScope
		if err := rows.Scan(&scope.ID, &scope.ServiceProviderTypeID, &scope.AuthorizationID, &scope.CertificateID, &scope.PrimaryTargetID); err != nil {
			return nil, err
		}
		scopes = append(scopes, scope)
	}
	return scopes, rows.Err()
}
