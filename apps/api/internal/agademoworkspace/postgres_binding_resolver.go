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
	candidates, err := resolver.bindingCandidates(ctx, principal)
	if err != nil {
		return preprod.AuthorityBinding{}, false, err
	}
	// A generic caller has not named an operation, so more than one matching
	// row is ambiguous.  Returning no binding is safer than manufacturing a
	// composite authority object.
	if len(candidates) != 1 {
		return preprod.AuthorityBinding{}, false, nil
	}
	return candidates[0], true, nil
}

func (resolver *PostgresBindingResolver) ResolveForOperation(ctx context.Context, principal identity.Principal, operation string) (preprod.AuthorityBinding, bool, error) {
	if resolver == nil || resolver.pool == nil || operation == "" {
		return preprod.AuthorityBinding{}, false, nil
	}
	candidates, err := resolver.bindingCandidates(ctx, principal)
	if err != nil {
		return preprod.AuthorityBinding{}, false, err
	}
	matching := make([]preprod.AuthorityBinding, 0, len(candidates))
	for _, binding := range candidates {
		if bindingAllowsWorkspaceOperation(binding, principal, operation) {
			matching = append(matching, binding)
		}
	}
	matching = preferFixtureMembershipBinding(principal, operation, matching)
	if len(matching) != 1 {
		return preprod.AuthorityBinding{}, false, nil
	}
	return matching[0], true, nil
}

// preferFixtureMembershipBinding resolves the deliberate shared-principal
// fixture boundary.  The lead-inspector subject is also used for the CAA
// reviewer projection, but ordinary lifecycle work belongs to the explicit
// lead-inspector membership.  ReviewCAP remains restricted by
// bindingHasWorkspaceRole to CAA_REVIEWER_MEMBERSHIP and therefore does not
// get silently widened by this preference.
//
// The helper only selects an unambiguous preferred row.  If the sealed
// projection contains zero or multiple preferred rows, the caller retains the
// complete candidate set and fails closed rather than guessing.
func preferFixtureMembershipBinding(principal identity.Principal, operation string, candidates []preprod.AuthorityBinding) []preprod.AuthorityBinding {
	preferredSlot := ""
	switch {
	case principal.HasRole(identity.RoleLeadInspector) && operation != OperationReviewCAP:
		preferredSlot = "LEAD_INSPECTOR_MEMBERSHIP"
	case principal.HasRole(identity.RoleInspector):
		preferredSlot = "INSPECTOR_MEMBERSHIP"
	}
	if preferredSlot == "" {
		return candidates
	}
	preferred := make([]preprod.AuthorityBinding, 0, len(candidates))
	for _, candidate := range candidates {
		if candidate.MembershipSlot == preferredSlot {
			preferred = append(preferred, candidate)
		}
	}
	if len(preferred) == 1 {
		return preferred
	}
	return candidates
}

func (resolver *PostgresBindingResolver) bindingCandidates(ctx context.Context, principal identity.Principal) ([]preprod.AuthorityBinding, error) {
	rows, err := resolver.pool.Query(ctx, `
		SELECT payload
		FROM preprod_aga_demo_workspace.sealed_authority_bindings
		WHERE active = true
		  AND payload->>'subjectId' = $1
		ORDER BY binding_id
	`, principal.SubjectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	candidates := make([]preprod.AuthorityBinding, 0)
	for rows.Next() {
		var payload []byte
		if err := rows.Scan(&payload); err != nil {
			return nil, err
		}
		var binding preprod.AuthorityBinding
		if err := json.Unmarshal(payload, &binding); err != nil {
			return nil, err
		}
		if validWorkspaceBinding(binding, principal) {
			candidates = append(candidates, binding)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	sort.SliceStable(candidates, func(left, right int) bool { return candidates[left].BindingID < candidates[right].BindingID })
	return candidates, nil
}
