package agademoworkspace

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/identity"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/platform/database"
	preprod "github.com/MarlonJD/aviaSurveil360/apps/api/internal/preproddata/agademoworkspace"
)

// PostgresBindingResolver reads only the sealed workspace authority projection.
// It never consults canonical membership, assignment, or provider tables.
type PostgresBindingResolver struct {
	pool *database.Pool
}

func NewPostgresBindingResolver(pool *database.Pool) (*PostgresBindingResolver, error) {
	if pool == nil {
		return nil, fmt.Errorf("workspace binding resolver PostgreSQL pool is required")
	}
	return &PostgresBindingResolver{pool: pool}, nil
}

func (resolver *PostgresBindingResolver) Resolve(ctx context.Context, principal identity.Principal) (preprod.AuthorityBinding, bool, error) {
	if resolver == nil || resolver.pool == nil || principal.SubjectID == "" || principal.OrganizationID == "" {
		return preprod.AuthorityBinding{}, false, nil
	}
	rows, err := resolver.pool.Query(ctx, `
		SELECT payload
		FROM preprod_aga_demo_workspace.sealed_authority_bindings
		WHERE active = true
		  AND payload->>'subjectId' = $1
		  AND organization_id = $2
		ORDER BY binding_id
	`, principal.SubjectID, principal.OrganizationID)
	if err != nil {
		return preprod.AuthorityBinding{}, false, err
	}
	defer rows.Close()

	var resolved preprod.AuthorityBinding
	seenRoles := map[string]struct{}{}
	found := false
	for rows.Next() {
		var payload []byte
		if err := rows.Scan(&payload); err != nil {
			return preprod.AuthorityBinding{}, false, err
		}
		var binding preprod.AuthorityBinding
		if err := json.Unmarshal(payload, &binding); err != nil {
			return preprod.AuthorityBinding{}, false, err
		}
		if !found {
			resolved = binding
			found = true
		}
		for _, role := range binding.OperationRoles {
			if _, ok := seenRoles[role]; ok {
				continue
			}
			seenRoles[role] = struct{}{}
			resolved.OperationRoles = append(resolved.OperationRoles, role)
		}
	}
	if err := rows.Err(); err != nil {
		return preprod.AuthorityBinding{}, false, err
	}
	if !found {
		return preprod.AuthorityBinding{}, false, nil
	}
	sort.Strings(resolved.OperationRoles)
	return resolved, true, nil
}
