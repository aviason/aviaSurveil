package agademoworkspace

import (
	"context"
	"encoding/json"
	"fmt"

	aga "github.com/MarlonJD/aviaSurveil360/apps/api/internal/agaapplicability"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/platform/database"
	preprod "github.com/MarlonJD/aviaSurveil360/apps/api/internal/preproddata/agademoworkspace"
)

// NewPostgresRecommendationScopeResolver reads the immutable sealed scope
// projection. The browser supplies exact selectors, while this resolver
// returns the complete provider fact, including the effective interval and
// qualifier maps that are bound into the readiness/profile digest.
func NewPostgresRecommendationScopeResolver(pool *database.Pool) RecommendationScopeResolver {
	return func(ctx context.Context, workspace preprod.LoadedWorkspace, request aga.RecommendationRequest) ([]aga.ProviderScopeFact, error) {
		if pool == nil || workspace.Generation.GenerationID == "" {
			return nil, fmt.Errorf("workspace scope resolver requires a PostgreSQL pool and generation")
		}
		rows, err := pool.Query(ctx, `
			SELECT payload
			FROM preprod_aga_demo_workspace.sealed_provider_scopes
			WHERE generation_id = $1
			  AND organization_id = $2
			  AND provider_scope_root_id = $3
			  AND provider_scope_id = $4
			  AND provider_scope_version = $5
		`, workspace.Generation.GenerationID, request.OrganizationID, request.ProviderScopeRootID, request.ProviderScopeID, request.ProviderScopeVersion)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		facts := make([]aga.ProviderScopeFact, 0, 1)
		for rows.Next() {
			var payload []byte
			if err := rows.Scan(&payload); err != nil {
				return nil, err
			}
			var scope preprod.ProviderScope
			if err := json.Unmarshal(payload, &scope); err != nil {
				return nil, fmt.Errorf("decode sealed provider scope: %w", err)
			}
			fact := aga.ProviderScopeFact{
				GenerationID: scope.GenerationID, ProfileDigest: scope.ProfileDigest,
				OrganizationID: scope.OrganizationID, ProviderScopeRootID: scope.ProviderScopeRootID,
				ProviderScopeID: scope.ProviderScopeID, ProviderScopeVersion: scope.ProviderScopeVersion,
				ProviderTypeID: scope.ProviderTypeID, ProviderTypeCode: scope.ProviderTypeCode,
				Status: scope.Status, EffectiveFrom: scope.EffectiveFrom, EffectiveTo: scope.EffectiveTo,
				DepartmentID: scope.DepartmentID, OrganizationalUnitID: scope.OrganizationalUnitID,
				OperationQualifiers: append([]aga.Qualifier(nil), scope.OperationQualifiers...),
				ActivityQualifiers:  append([]aga.Qualifier(nil), scope.ActivityQualifiers...),
			}
			for _, target := range scope.Targets {
				fact.Targets = append(fact.Targets, aga.TypedTarget{ID: target.TargetID, Kind: target.CanonicalKind, ProfileCode: target.ProfileCode})
			}
			facts = append(facts, fact)
		}
		if err := rows.Err(); err != nil {
			return nil, err
		}
		return facts, nil
	}
}

// NewFixtureLifecycleBindingResolver pins lifecycle actors to the exported
// synthetic fixture. It never resolves canonical assignment or membership
// authority; membership versions are used as the binding revision pin.
func NewFixtureLifecycleBindingResolver() LifecycleBindingResolver {
	return func(_ context.Context, workspace preprod.LoadedWorkspace, recommendation aga.Recommendation, kind string) ([]LifecycleBindingFact, error) {
		type selection struct {
			slot         string
			organization string
		}
		wanted := map[string]selection{
			"INSPECTOR": {slot: "INSPECTOR", organization: recommendation.OrganizationID},
			"LEAD":      {slot: "LEAD_INSPECTOR", organization: recommendation.OrganizationID},
			"AUDITEE":   {slot: "AUDITEE_MATCHING", organization: recommendation.OrganizationID},
		}
		choice, ok := wanted[kind]
		if !ok {
			return nil, fmt.Errorf("unknown lifecycle binding kind %q", kind)
		}
		accounts := make(map[string]preprod.FixtureAccount, len(workspace.Fixture.Accounts))
		for _, account := range workspace.Fixture.Accounts {
			accounts[account.Slot] = account
		}
		facts := make([]LifecycleBindingFact, 0, 1)
		for _, binding := range workspace.Fixture.Bindings {
			if binding.SubjectSlot != choice.slot || binding.OrganizationID != choice.organization || !binding.Active {
				continue
			}
			account, found := accounts[binding.SubjectSlot]
			if !found || account.SubjectID == "" || account.OrganizationID == "" {
				continue
			}
			revision := account.MembershipVersion
			if revision < 1 {
				revision = 1
			}
			facts = append(facts, LifecycleBindingFact{
				BindingID: binding.BindingID, BindingRevision: revision, SubjectID: account.SubjectID,
				MembershipSlot: binding.MembershipSlot, OrganizationID: binding.OrganizationID,
				SourceOrganizationID: account.OrganizationID,
				DepartmentID:         binding.DepartmentID, OrganizationalUnitID: binding.OrganizationalUnitID,
				ProviderScopeID: recommendation.ProviderScopeID, Active: true,
			})
		}
		return facts, nil
	}
}
